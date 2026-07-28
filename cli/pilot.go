package cli

import (
	"archive/tar"
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/smallnest/goclaw/config"
	dev "github.com/smallnest/goclaw/orchestratorlite"
	"github.com/smallnest/goclaw/session"
	"github.com/smallnest/goclaw/workstation"
	"github.com/spf13/cobra"
)

const (
	pilotBackupSchema      = "goclaw.pilot-backup/v1"
	pilotAttestationSchema = "goclaw.credential-attestation/v1"
	pilotRequiredMembers   = 3
)

type pilotCheck struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

type pilotCheckReport struct {
	SchemaVersion string       `json:"schema_version"`
	ProjectID     string       `json:"project_id"`
	Ready         bool         `json:"ready"`
	CheckedAt     time.Time    `json:"checked_at"`
	Checks        []pilotCheck `json:"checks"`
}

type pilotCredentialAttestation struct {
	SchemaVersion string    `json:"schema_version"`
	IssueID       string    `json:"issue_id"`
	Status        string    `json:"status"`
	AttestedBy    string    `json:"attested_by"`
	AttestedAt    time.Time `json:"attested_at"`
	EvidenceRef   string    `json:"evidence_ref,omitempty"`
}

type pilotBackupManifest struct {
	SchemaVersion string              `json:"schema_version"`
	CreatedAt     time.Time           `json:"created_at"`
	Release       string              `json:"release"`
	Files         []pilotBackupFile   `json:"files"`
	Sources       []pilotBackupSource `json:"sources"`
	Repositories  []pilotBackupRepo   `json:"repositories,omitempty"`
	ManifestHash  string              `json:"manifest_sha256,omitempty"`
}

type pilotBackupFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
	Mode   uint32 `json:"mode"`
}

type pilotBackupSource struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}

type pilotBackupRepo struct {
	Name       string `json:"name"`
	HeadCommit string `json:"head_commit"`
	BundlePath string `json:"bundle_path"`
}

type pilotSource struct {
	Name string
	Path string
}

func init() {
	rootCmd.AddCommand(newPilotCommand())
}

func newPilotCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "pilot",
		Short: "Validate and protect a controlled three-person pilot",
	}
	command.AddCommand(
		newPilotCheckCommand(),
		newPilotBackupCommand(),
		newPilotVerifyBackupCommand(),
		newPilotRestoreCommand(),
	)
	return command
}

func newPilotCheckCommand() *cobra.Command {
	var projectID, attestationPath, registryPath, backupPath, backupIdentity string
	var outputJSON bool
	command := &cobra.Command{
		Use:   "check",
		Short: "Fail closed unless the three-person pilot is ready",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load("")
			if err != nil {
				return err
			}
			report := runPilotCheck(
				cmd.Context(),
				cfg,
				projectID,
				attestationPath,
				registryPath,
				backupPath,
				backupIdentity,
			)
			if outputJSON {
				if err := printTeamValue(report); err != nil {
					return err
				}
			} else {
				fmt.Printf("Three-person pilot: %s\n", map[bool]string{
					true: "READY", false: "BLOCKED",
				}[report.Ready])
				for _, check := range report.Checks {
					fmt.Printf("  %-7s %-28s %s\n", strings.ToUpper(check.Status), check.ID, check.Detail)
				}
			}
			if !report.Ready {
				return errors.New("pilot preflight is blocked")
			}
			return nil
		},
	}
	command.Flags().StringVar(&projectID, "project", "", "Pilot project id")
	command.Flags().StringVar(
		&attestationPath,
		"credential-attestation",
		"",
		"0600 JSON attestation for the historical credential finding",
	)
	command.Flags().StringVar(
		&registryPath,
		"wave-registry",
		"docs/waves/wave-registry.json",
		"Approved Wave registry",
	)
	command.Flags().StringVar(
		&backupPath,
		"backup",
		"",
		"Encrypted pilot backup produced by `goclaw pilot backup`",
	)
	command.Flags().StringVar(
		&backupIdentity,
		"backup-age-identity",
		"",
		"age identity used to decrypt and semantically verify the recovery point",
	)
	command.Flags().BoolVar(&outputJSON, "json", false, "Print machine-readable JSON")
	_ = command.MarkFlagRequired("project")
	return command
}

func runPilotCheck(
	ctx context.Context,
	cfg *config.Config,
	projectID, attestationPath, registryPath, backupPath, backupIdentity string,
) pilotCheckReport {
	report := pilotCheckReport{
		SchemaVersion: "goclaw.pilot-check/v1",
		ProjectID:     strings.TrimSpace(projectID),
		Ready:         true,
		CheckedAt:     time.Now().UTC(),
	}
	add := func(id string, passed bool, detail string) {
		status := "passed"
		if !passed {
			status = "blocked"
			report.Ready = false
		}
		report.Checks = append(report.Checks, pilotCheck{ID: id, Status: status, Detail: detail})
	}
	if report.ProjectID == "" {
		add("project", false, "project id is required")
		return report
	}
	_, _, conversationErr := session.ProjectConversationKey(
		report.ProjectID,
		session.DefaultProjectTopic,
	)
	add(
		"project.conversation-boundary",
		conversationErr == nil,
		errorOr(
			conversationErr,
			"project id is valid for the versioned shared-conversation key",
		),
	)

	membersValue, membersErr := callGatewayRPCContext(ctx, cfg, "team.members", map[string]interface{}{
		"project_id": report.ProjectID,
	})
	members := valueMaps(membersValue)
	add(
		"members.exactly-three",
		membersErr == nil && len(members) == pilotRequiredMembers,
		checkCountDetail(len(members), pilotRequiredMembers, membersErr),
	)
	activeMembers := make(map[string]bool)
	for _, member := range members {
		id := stringValue(member["id"])
		if id != "" && stringValue(member["status"]) == "active" {
			activeMembers[id] = true
		}
	}
	add(
		"members.active",
		membersErr == nil && len(activeMembers) == pilotRequiredMembers,
		fmt.Sprintf("%d/%d active", len(activeMembers), pilotRequiredMembers),
	)

	runnersValue, runnersErr := callGatewayRPCContext(ctx, cfg, "runner.list", map[string]interface{}{
		"project_id": report.ProjectID,
	})
	runners := valueMaps(runnersValue)
	add(
		"runners.exactly-three",
		runnersErr == nil && len(runners) == pilotRequiredMembers,
		checkCountDetail(len(runners), pilotRequiredMembers, runnersErr),
	)
	owners := make(map[string]bool)
	platformOK := runnersErr == nil && len(runners) == pilotRequiredMembers
	sandboxHashes := make(map[string]bool)
	onlineCount := 0
	for _, runner := range runners {
		owner := stringValue(runner["member_id"])
		if owner != "" {
			owners[owner] = true
		}
		switch stringValue(runner["status"]) {
		case "online", "busy":
			onlineCount++
		}
		metadata := stringMapValue(runner["metadata"])
		sandboxSHA256, validPlatform := validatePilotRunnerPlatform(
			metadata,
			runner["capabilities"],
		)
		if !validPlatform {
			platformOK = false
		} else {
			sandboxHashes[sandboxSHA256] = true
		}
	}
	if len(sandboxHashes) != 1 {
		platformOK = false
	}
	ownersOK := len(owners) == pilotRequiredMembers
	for owner := range owners {
		if !activeMembers[owner] {
			ownersOK = false
		}
	}
	add("runners.distinct-owners", ownersOK, fmt.Sprintf("%d distinct active owners", len(owners)))
	add(
		"runners.online",
		runnersErr == nil && onlineCount == pilotRequiredMembers,
		fmt.Sprintf("%d/%d online or busy", onlineCount, pilotRequiredMembers),
	)
	add(
		"runners.linux-substrate",
		platformOK,
		"all runners must attest the same reviewed Linux bwrap wrapper, a released architecture, and an approved host profile",
	)

	policyValue, policyErr := callGatewayRPCContext(ctx, cfg, "policy.status", map[string]interface{}{
		"project_id": report.ProjectID,
	})
	policy, _ := policyValue.(map[string]interface{})
	add(
		"policy.compliant",
		policyErr == nil && boolValue(policy["compliant"]),
		errorOr(policyErr, "effective project policy is compliant"),
	)

	waveID, waveErr := activePilotWave(registryPath)
	add(
		"wave.active",
		waveErr == nil && waveID == "PILOT-W00",
		errorOr(waveErr, "PILOT-W00 is the unique active Wave"),
	)

	attestationErr := validatePilotAttestation(attestationPath)
	add(
		"credential.attested",
		attestationErr == nil,
		errorOr(attestationErr, "credential owner closure is present"),
	)
	_, backupErr := verifyPilotBackup(ctx, backupPath, backupIdentity, "")
	add(
		"backup.verified",
		backupErr == nil,
		errorOr(backupErr, "encrypted recovery point decrypted and passed semantic verification"),
	)
	return report
}

func allowedPilotHostProfile(value string) bool {
	switch value {
	case "native-linux", "wsl2", "lima":
		return true
	default:
		return false
	}
}

func validatePilotRunnerPlatform(
	metadata map[string]string,
	capabilities interface{},
) (string, bool) {
	if metadata["runner_goos"] != "linux" ||
		metadata["isolation_backend"] != "bwrap" ||
		!allowedPilotHostProfile(metadata["host_profile"]) ||
		!containsStringValue(capabilities, "goclaw-runtime-linux-v1") {
		return "", false
	}
	switch metadata["runner_goarch"] {
	case "amd64", "arm64":
	default:
		return "", false
	}
	sandboxSHA256 := strings.ToLower(strings.TrimSpace(metadata["sandbox_sha256"]))
	decoded, err := hex.DecodeString(sandboxSHA256)
	if err != nil || len(decoded) != sha256.Size {
		return "", false
	}
	return sandboxSHA256, true
}

func checkCountDetail(got, want int, err error) string {
	if err != nil {
		return err.Error()
	}
	return fmt.Sprintf("%d found; want exactly %d", got, want)
}

func errorOr(err error, success string) string {
	if err != nil {
		return err.Error()
	}
	return success
}

func activePilotWave(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("wave registry path is required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var registry struct {
		ActiveWave string `json:"active_wave"`
		Waves      []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"waves"`
	}
	if err := json.Unmarshal(data, &registry); err != nil {
		return "", err
	}
	activeCount := 0
	for _, wave := range registry.Waves {
		if wave.Status == "active" {
			activeCount++
			if wave.ID != registry.ActiveWave {
				return "", errors.New("registry active pointer disagrees with active Wave")
			}
		}
	}
	if activeCount != 1 || strings.TrimSpace(registry.ActiveWave) == "" {
		return "", fmt.Errorf("registry has %d active Waves; want exactly one", activeCount)
	}
	return registry.ActiveWave, nil
}

func validatePilotAttestation(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("credential attestation is required")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("credential attestation must be a regular file")
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		return errors.New("credential attestation must not be accessible by group or others")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var value pilotCredentialAttestation
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if value.SchemaVersion != pilotAttestationSchema || value.IssueID != "FE-ISSUE-007" {
		return errors.New("credential attestation schema or issue_id is invalid")
	}
	switch value.Status {
	case "revoked", "rotated", "never_valid":
	default:
		return errors.New("credential attestation status must be revoked, rotated, or never_valid")
	}
	if strings.TrimSpace(value.AttestedBy) == "" || value.AttestedAt.IsZero() ||
		value.AttestedAt.After(time.Now().UTC().Add(5*time.Minute)) {
		return errors.New("credential attestation identity or timestamp is invalid")
	}
	return nil
}

func validateEncryptedBackupPath(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("an encrypted pilot backup is required")
	}
	if !strings.HasSuffix(strings.ToLower(path), ".age") {
		return errors.New("pilot backup must be an age-encrypted .age file")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() == 0 {
		return errors.New("pilot backup must be a non-empty regular file")
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		return errors.New("pilot backup must have mode 0600")
	}
	return nil
}

func newPilotBackupCommand() *cobra.Command {
	var configPath, output, recipient, attestationPath string
	var repositories []string
	command := &cobra.Command{
		Use:   "backup",
		Short: "Create an encrypted, maintenance-locked cold backup",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}
			manifest, err := createPilotBackup(
				cmd.Context(),
				cfg,
				configPath,
				output,
				recipient,
				attestationPath,
				repositories,
			)
			if err != nil {
				return err
			}
			return printTeamValue(map[string]interface{}{
				"archive":         output,
				"manifest_sha256": manifest.ManifestHash,
				"files":           len(manifest.Files),
				"repositories":    len(manifest.Repositories),
			})
		},
	}
	command.Flags().StringVar(&configPath, "config", "", "Explicit config.json path")
	command.Flags().StringVar(&output, "output", "", "New .tar.age archive")
	command.Flags().StringVar(&recipient, "age-recipient", "", "age recipient (age1… or ssh public key)")
	command.Flags().StringVar(
		&attestationPath,
		"credential-attestation",
		"",
		"Validated credential-owner attestation to preserve in the recovery point",
	)
	command.Flags().StringSliceVar(
		&repositories,
		"repo",
		nil,
		"Repository NAME=/absolute/path to include as a Git bundle",
	)
	_ = command.MarkFlagRequired("config")
	_ = command.MarkFlagRequired("output")
	_ = command.MarkFlagRequired("age-recipient")
	_ = command.MarkFlagRequired("credential-attestation")
	return command
}

func createPilotBackup(
	ctx context.Context,
	cfg *config.Config,
	configPath, output, recipient, attestationPath string,
	repositories []string,
) (pilotBackupManifest, error) {
	var empty pilotBackupManifest
	if err := validatePilotAttestation(attestationPath); err != nil {
		return empty, fmt.Errorf("credential attestation: %w", err)
	}
	absoluteOutput, err := filepath.Abs(output)
	if err != nil {
		return empty, err
	}
	if !strings.HasSuffix(strings.ToLower(absoluteOutput), ".age") {
		return empty, errors.New("--output must end in .age")
	}
	if _, err := os.Lstat(absoluteOutput); err == nil {
		return empty, errors.New("backup output already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return empty, err
	}
	lockPath, err := acquirePilotMaintenanceLock()
	if err != nil {
		return empty, err
	}
	defer os.Remove(lockPath)
	if running := localGatewayRunning(cfg); running {
		return empty, errors.New("Gateway is running; stop it before a cold backup")
	}
	if _, err := exec.LookPath("age"); err != nil {
		return empty, errors.New("age executable is required for encrypted pilot backups")
	}
	stage, err := os.MkdirTemp("", "goclaw-pilot-backup-*")
	if err != nil {
		return empty, err
	}
	defer os.RemoveAll(stage)
	if err := os.Chmod(stage, 0o700); err != nil {
		return empty, err
	}
	sources, err := pilotBackupSources(cfg, configPath, attestationPath)
	if err != nil {
		return empty, err
	}
	for _, source := range sources {
		destination := filepath.Join(stage, "data", source.Name)
		if err := copyPilotSource(source.Path, destination); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return empty, fmt.Errorf("backup %s: %w", source.Name, err)
		}
	}
	repos, err := createPilotGitBundles(ctx, stage, repositories)
	if err != nil {
		return empty, err
	}
	manifest, err := buildPilotManifest(stage, sources, repos)
	if err != nil {
		return empty, err
	}
	if err := writePilotManifest(stage, &manifest); err != nil {
		return empty, err
	}
	tarPath := filepath.Join(stage, "pilot-backup.tar")
	if err := writePilotTar(stage, tarPath); err != nil {
		return empty, err
	}
	age := exec.CommandContext(ctx, "age", "-r", recipient, "-o", absoluteOutput, tarPath)
	age.Env = append(os.Environ(), "LC_ALL=C")
	outputBytes, err := age.CombinedOutput()
	if err != nil {
		_ = os.Remove(absoluteOutput)
		return empty, fmt.Errorf("age encryption failed: %w: %s", err, strings.TrimSpace(string(outputBytes)))
	}
	if err := os.Chmod(absoluteOutput, 0o600); err != nil {
		_ = os.Remove(absoluteOutput)
		return empty, err
	}
	return manifest, nil
}

func newPilotVerifyBackupCommand() *cobra.Command {
	var archive, identity string
	command := &cobra.Command{
		Use:   "verify-backup",
		Short: "Decrypt and verify a pilot backup without restoring it",
		RunE: func(cmd *cobra.Command, args []string) error {
			manifest, err := verifyPilotBackup(cmd.Context(), archive, identity, "")
			if err != nil {
				return err
			}
			return printTeamValue(map[string]interface{}{
				"archive":         archive,
				"manifest_sha256": manifest.ManifestHash,
				"files":           len(manifest.Files),
				"status":          "verified",
			})
		},
	}
	command.Flags().StringVar(&archive, "archive", "", "Encrypted .tar.age backup")
	command.Flags().StringVar(&identity, "age-identity", "", "age identity file")
	_ = command.MarkFlagRequired("archive")
	_ = command.MarkFlagRequired("age-identity")
	return command
}

func newPilotRestoreCommand() *cobra.Command {
	var archive, identity, target string
	command := &cobra.Command{
		Use:   "restore",
		Short: "Verify and restore a cold backup into a new directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			absolute, err := filepath.Abs(target)
			if err != nil {
				return err
			}
			if err := requireNewRestoreTarget(absolute); err != nil {
				return err
			}
			manifest, err := verifyPilotBackup(cmd.Context(), archive, identity, absolute)
			if err != nil {
				// requireNewRestoreTarget established that this exact target was
				// absent or empty before extraction. Remove any partial restore so
				// a failed verification cannot be mistaken for recoverable state.
				_ = os.RemoveAll(absolute)
				return err
			}
			return printTeamValue(map[string]interface{}{
				"restored_to":     absolute,
				"manifest_sha256": manifest.ManifestHash,
				"next":            "point a copy of config.json at this new root, start Gateway, then run `goclaw pilot check`",
			})
		},
	}
	command.Flags().StringVar(&archive, "archive", "", "Encrypted .tar.age backup")
	command.Flags().StringVar(&identity, "age-identity", "", "age identity file")
	command.Flags().StringVar(&target, "target", "", "New empty restore directory")
	_ = command.MarkFlagRequired("archive")
	_ = command.MarkFlagRequired("age-identity")
	_ = command.MarkFlagRequired("target")
	return command
}

func verifyPilotBackup(
	ctx context.Context,
	archive, identity, restoreTarget string,
) (pilotBackupManifest, error) {
	var empty pilotBackupManifest
	if err := validateEncryptedBackupPath(archive); err != nil {
		return empty, err
	}
	if err := validateAgeIdentityPath(identity); err != nil {
		return empty, err
	}
	if _, err := exec.LookPath("age"); err != nil {
		return empty, errors.New("age executable is required")
	}
	temp, err := os.MkdirTemp("", "goclaw-pilot-verify-*")
	if err != nil {
		return empty, err
	}
	defer os.RemoveAll(temp)
	tarPath := filepath.Join(temp, "backup.tar")
	command := exec.CommandContext(ctx, "age", "-d", "-i", identity, "-o", tarPath, archive)
	command.Env = append(os.Environ(), "LC_ALL=C")
	if output, err := command.CombinedOutput(); err != nil {
		return empty, fmt.Errorf("age decryption failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	extractRoot := filepath.Join(temp, "extract")
	if restoreTarget != "" {
		extractRoot = restoreTarget
	}
	if err := os.MkdirAll(extractRoot, 0o700); err != nil {
		return empty, err
	}
	if err := extractPilotTar(tarPath, extractRoot); err != nil {
		return empty, err
	}
	manifest, err := readAndVerifyPilotManifest(extractRoot)
	if err != nil {
		if restoreTarget != "" {
			_ = os.RemoveAll(restoreTarget)
		}
		return empty, err
	}
	return manifest, nil
}

func validateAgeIdentityPath(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("age identity file is required")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("age identity file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return errors.New("age identity must be a regular file")
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		return errors.New("age identity must not be accessible by group or others")
	}
	return nil
}

func acquirePilotMaintenanceLock() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	root := filepath.Join(home, ".goclaw")
	if err := os.MkdirAll(root, 0o700); err != nil {
		return "", err
	}
	path := filepath.Join(root, "pilot-maintenance.lock")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return "", errors.New("pilot maintenance lock already exists")
		}
		return "", err
	}
	_, writeErr := fmt.Fprintf(file, "%d %s\n", os.Getpid(), time.Now().UTC().Format(time.RFC3339))
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		_ = os.Remove(path)
		return "", errors.Join(writeErr, closeErr)
	}
	return path, nil
}

func localGatewayRunning(cfg *config.Config) bool {
	port := cfg.Gateway.Port
	if port == 0 {
		port = config.GetGatewayHTTPPort(cfg)
	}
	host := strings.TrimSpace(cfg.Gateway.Host)
	switch host {
	case "", "0.0.0.0", "::", "[::]":
		host = "127.0.0.1"
	}
	connection, err := net.DialTimeout(
		"tcp",
		net.JoinHostPort(strings.Trim(host, "[]"), fmt.Sprintf("%d", port)),
		750*time.Millisecond,
	)
	if err != nil {
		return false
	}
	_ = connection.Close()
	return true
}

func pilotBackupSources(
	cfg *config.Config,
	configPath, attestationPath string,
) ([]pilotSource, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	goclawRoot := filepath.Join(home, ".goclaw")
	defaultRoot := func(value, name string) string {
		if strings.TrimSpace(value) != "" {
			return value
		}
		return filepath.Join(goclawRoot, name)
	}
	catalogPath := cfg.Memory.Catalog.DatabasePath
	if strings.TrimSpace(catalogPath) == "" {
		catalogPath = filepath.Join(goclawRoot, "memory", "catalog.db")
	}
	workspacePath, err := config.GetWorkspacePath(cfg)
	if err != nil {
		return nil, err
	}
	sources := []pilotSource{
		{Name: "config", Path: configPath},
		{Name: "credential-attestation", Path: attestationPath},
		{Name: "teamcontrol", Path: defaultRoot(cfg.TeamControl.Root, "teamcontrol")},
		{Name: "workstation", Path: defaultRoot(cfg.Workstation.Root, "workstation")},
		{Name: "development", Path: defaultRoot(cfg.Development.Root, "development")},
		{Name: "harness", Path: defaultRoot(cfg.Harness.Root, "harness")},
		{Name: "ouroboros", Path: defaultRoot(cfg.Ouroboros.Root, "ouroboros")},
		{Name: "sessions", Path: filepath.Join(goclawRoot, "sessions")},
		{Name: "memory-catalog", Path: catalogPath},
		{Name: "workspace", Path: workspacePath},
	}
	if root := strings.TrimSpace(cfg.Harness.KnowledgeRoot); root != "" {
		sources = append(sources, pilotSource{Name: "knowledge", Path: root})
	} else if root := strings.TrimSpace(cfg.Harness.VaultPath); root != "" {
		sources = append(sources, pilotSource{Name: "knowledge", Path: root})
	}
	return sources, nil
}

func copyPilotSource(source, destination string) error {
	info, err := os.Lstat(source)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return errors.New("symbolic links are forbidden in pilot backups")
	}
	if info.Mode().IsRegular() {
		if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
			return err
		}
		return copyPilotFile(source, destination, info.Mode().Perm())
	}
	if !info.IsDir() {
		return errors.New("backup source must be a regular file or directory")
	}
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := destination
		if relative != "." {
			target = filepath.Join(destination, relative)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symbolic link is forbidden: %s", relative)
		}
		if entry.IsDir() {
			return os.MkdirAll(target, 0o700)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("special file is forbidden: %s", relative)
		}
		return copyPilotFile(path, target, info.Mode().Perm())
	})
}

func copyPilotFile(source, destination string, mode os.FileMode) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode&0o700)
	if err != nil {
		return err
	}
	ok := false
	defer func() {
		_ = output.Close()
		if !ok {
			_ = os.Remove(destination)
		}
	}()
	if _, err := io.Copy(output, input); err != nil {
		return err
	}
	if err := output.Sync(); err != nil {
		return err
	}
	if err := output.Close(); err != nil {
		return err
	}
	ok = true
	return nil
}

func createPilotGitBundles(
	ctx context.Context,
	stage string,
	values []string,
) ([]pilotBackupRepo, error) {
	mappings := make(map[string]string)
	for _, value := range values {
		name, path, found := strings.Cut(value, "=")
		name, path = strings.TrimSpace(name), strings.TrimSpace(path)
		if !found || name == "" || path == "" || strings.ContainsAny(name, `/\`) {
			return nil, fmt.Errorf("invalid --repo %q; expected NAME=/absolute/path", value)
		}
		if !filepath.IsAbs(path) {
			return nil, fmt.Errorf("repository %s path must be absolute", name)
		}
		if _, exists := mappings[name]; exists {
			return nil, fmt.Errorf("duplicate repository name %q", name)
		}
		mappings[name] = path
	}
	names := make([]string, 0, len(mappings))
	for name := range mappings {
		names = append(names, name)
	}
	sort.Strings(names)
	result := make([]pilotBackupRepo, 0, len(names))
	for _, name := range names {
		path := mappings[name]
		if err := rejectDirtyRepository(ctx, path); err != nil {
			return nil, fmt.Errorf("repository %s: %w", name, err)
		}
		head, err := pilotGitOutput(ctx, path, "rev-parse", "HEAD")
		if err != nil {
			return nil, err
		}
		bundleRelative := filepath.ToSlash(filepath.Join("repositories", name+".bundle"))
		bundlePath := filepath.Join(stage, filepath.FromSlash(bundleRelative))
		if err := os.MkdirAll(filepath.Dir(bundlePath), 0o700); err != nil {
			return nil, err
		}
		if _, err := pilotGitOutput(ctx, path, "bundle", "create", bundlePath, "--all"); err != nil {
			return nil, fmt.Errorf("create Git bundle %s: %w", name, err)
		}
		result = append(result, pilotBackupRepo{
			Name: name, HeadCommit: strings.TrimSpace(head), BundlePath: bundleRelative,
		})
	}
	return result, nil
}

func rejectDirtyRepository(ctx context.Context, path string) error {
	output, err := pilotGitOutput(ctx, path, "status", "--porcelain=v1", "--untracked-files=all")
	if err != nil {
		return err
	}
	if strings.TrimSpace(output) != "" {
		return errors.New("repository is dirty; commit or preserve work through signed Evidence before backup")
	}
	return nil
}

func pilotGitOutput(ctx context.Context, repository string, args ...string) (string, error) {
	base := []string{
		"-c", "core.hooksPath=/dev/null",
		"-c", "core.fsmonitor=false",
		"-c", "diff.external=",
		"-C", repository,
	}
	command := exec.CommandContext(ctx, "git", append(base, args...)...)
	command.Env = []string{
		"PATH=/usr/bin:/bin",
		"LC_ALL=C",
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_TERMINAL_PROMPT=0",
	}
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", args[0], err, strings.TrimSpace(string(output)))
	}
	return string(output), nil
}

func buildPilotManifest(
	stage string,
	sources []pilotSource,
	repositories []pilotBackupRepo,
) (pilotBackupManifest, error) {
	manifest := pilotBackupManifest{
		SchemaVersion: pilotBackupSchema,
		CreatedAt:     time.Now().UTC(),
		Release:       "0.8.0-pilot.1",
		Repositories:  repositories,
	}
	for _, source := range sources {
		if _, err := os.Stat(filepath.Join(stage, "data", source.Name)); err == nil {
			manifest.Sources = append(manifest.Sources, pilotBackupSource{
				Name: source.Name, Kind: "filesystem",
			})
		}
	}
	err := filepath.WalkDir(stage, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || entry.Name() == "manifest.json" ||
			entry.Name() == "pilot-backup.tar" {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("manifest member is not regular: %s", path)
		}
		relative, err := filepath.Rel(stage, path)
		if err != nil {
			return err
		}
		digest, err := fileSHA256(path)
		if err != nil {
			return err
		}
		manifest.Files = append(manifest.Files, pilotBackupFile{
			Path:   filepath.ToSlash(relative),
			SHA256: digest,
			Size:   info.Size(),
			Mode:   uint32(info.Mode().Perm() & 0o700),
		})
		return nil
	})
	sort.Slice(manifest.Files, func(i, j int) bool {
		return manifest.Files[i].Path < manifest.Files[j].Path
	})
	return manifest, err
}

func writePilotManifest(stage string, manifest *pilotBackupManifest) error {
	hashInput := *manifest
	hashInput.ManifestHash = ""
	data, err := json.Marshal(hashInput)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(data)
	manifest.ManifestHash = hex.EncodeToString(digest[:])
	output, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	output = append(output, '\n')
	return os.WriteFile(filepath.Join(stage, "manifest.json"), output, 0o600)
}

func writePilotTar(stage, output string) error {
	file, err := os.OpenFile(output, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	writer := tar.NewWriter(file)
	ok := false
	defer func() {
		_ = writer.Close()
		_ = file.Close()
		if !ok {
			_ = os.Remove(output)
		}
	}()
	var paths []string
	if err := filepath.WalkDir(stage, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == output {
			return nil
		}
		paths = append(paths, path)
		return nil
	}); err != nil {
		return err
	}
	sort.Strings(paths)
	for _, path := range paths {
		if path == stage {
			continue
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || (!info.IsDir() && !info.Mode().IsRegular()) {
			return fmt.Errorf("unsupported archive member: %s", path)
		}
		relative, err := filepath.Rel(stage, path)
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(relative)
		header.Uid, header.Gid = 0, 0
		header.Uname, header.Gname = "", ""
		header.ModTime = time.Unix(0, 0).UTC()
		header.AccessTime, header.ChangeTime = time.Time{}, time.Time{}
		if info.IsDir() {
			header.Mode = 0o700
		} else {
			header.Mode = int64(info.Mode().Perm() & 0o700)
		}
		if err := writer.WriteHeader(header); err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			input, err := os.Open(path)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(writer, input)
			closeErr := input.Close()
			if copyErr != nil || closeErr != nil {
				return errors.Join(copyErr, closeErr)
			}
		}
	}
	if err := writer.Close(); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	ok = true
	return nil
}

func extractPilotTar(path, target string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	reader := tar.NewReader(file)
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		clean := filepath.Clean(filepath.FromSlash(header.Name))
		if clean == "." || filepath.IsAbs(clean) ||
			clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return fmt.Errorf("unsafe archive path %q", header.Name)
		}
		destination := filepath.Join(target, clean)
		relative, err := filepath.Rel(target, destination)
		if err != nil || relative == ".." ||
			strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return fmt.Errorf("archive member escapes target: %q", header.Name)
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(destination, 0o700); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
				return err
			}
			output, err := os.OpenFile(
				destination,
				os.O_CREATE|os.O_EXCL|os.O_WRONLY,
				os.FileMode(header.Mode)&0o700,
			)
			if err != nil {
				return err
			}
			_, copyErr := io.CopyN(output, reader, header.Size)
			closeErr := output.Close()
			if copyErr != nil || closeErr != nil {
				return errors.Join(copyErr, closeErr)
			}
		default:
			return fmt.Errorf("archive member type %d is forbidden", header.Typeflag)
		}
	}
}

func readAndVerifyPilotManifest(root string) (pilotBackupManifest, error) {
	var manifest pilotBackupManifest
	data, err := os.ReadFile(filepath.Join(root, "manifest.json"))
	if err != nil {
		return manifest, err
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return manifest, err
	}
	if manifest.SchemaVersion != pilotBackupSchema {
		return manifest, errors.New("unsupported pilot backup schema")
	}
	hashInput := manifest
	hashInput.ManifestHash = ""
	canonical, err := json.Marshal(hashInput)
	if err != nil {
		return manifest, err
	}
	digest := sha256.Sum256(canonical)
	if hex.EncodeToString(digest[:]) != manifest.ManifestHash {
		return manifest, errors.New("pilot backup manifest hash mismatch")
	}
	allowed := make(map[string]pilotBackupFile, len(manifest.Files))
	for _, entry := range manifest.Files {
		clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(entry.Path)))
		if clean != entry.Path || clean == "." || strings.HasPrefix(clean, "../") {
			return manifest, fmt.Errorf("unsafe manifest path %q", entry.Path)
		}
		if _, duplicate := allowed[entry.Path]; duplicate {
			return manifest, fmt.Errorf("duplicate manifest path %q", entry.Path)
		}
		allowed[entry.Path] = entry
		path := filepath.Join(root, filepath.FromSlash(entry.Path))
		info, err := os.Lstat(path)
		if err != nil {
			return manifest, err
		}
		if !info.Mode().IsRegular() || info.Size() != entry.Size ||
			uint32(info.Mode().Perm()&0o700) != entry.Mode {
			return manifest, fmt.Errorf("backup member size/type mismatch: %s", entry.Path)
		}
		actual, err := fileSHA256(path)
		if err != nil {
			return manifest, err
		}
		if actual != entry.SHA256 {
			return manifest, fmt.Errorf("backup member digest mismatch: %s", entry.Path)
		}
	}
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() {
			return walkErr
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if relative == "manifest.json" {
			return nil
		}
		if _, ok := allowed[relative]; !ok {
			return fmt.Errorf("unmanifested backup member: %s", relative)
		}
		return nil
	})
	if err == nil {
		err = validatePilotBackupSemantics(root, manifest)
	}
	return manifest, err
}

func validatePilotBackupSemantics(
	root string,
	manifest pilotBackupManifest,
) error {
	if err := validatePilotBackupSources(root, manifest); err != nil {
		return err
	}
	if err := validateTeamControlBackup(filepath.Join(
		root,
		"data",
		"teamcontrol",
	)); err != nil {
		return err
	}
	if err := validateWorkstationBackup(filepath.Join(
		root,
		"data",
		"workstation",
	)); err != nil {
		return err
	}
	if err := validateDevelopmentEventChains(filepath.Join(
		root,
		"data",
		"development",
		"tasks",
	)); err != nil {
		return err
	}
	for _, repository := range manifest.Repositories {
		path := filepath.Join(root, filepath.FromSlash(repository.BundlePath))
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		output, err := pilotGitCommand(
			ctx,
			"bundle",
			"list-heads",
			path,
		)
		cancel()
		if err != nil {
			return fmt.Errorf("verify Git bundle %s: %w", repository.Name, err)
		}
		if !strings.Contains(output, repository.HeadCommit) {
			return fmt.Errorf(
				"Git bundle %s does not contain frozen HEAD %s",
				repository.Name,
				repository.HeadCommit,
			)
		}
	}
	return nil
}

func validatePilotBackupSources(
	root string,
	manifest pilotBackupManifest,
) error {
	present := make(map[string]bool, len(manifest.Sources))
	for _, source := range manifest.Sources {
		name := strings.TrimSpace(source.Name)
		if name == "" || source.Kind != "filesystem" || present[name] {
			return fmt.Errorf("invalid or duplicate backup source %q", source.Name)
		}
		present[name] = true
		if _, err := os.Lstat(filepath.Join(root, "data", name)); err != nil {
			return fmt.Errorf("backup source %s is missing from the archive: %w", name, err)
		}
	}
	required := []string{
		"config",
		"credential-attestation",
		"teamcontrol",
		"workstation",
		"development",
		"sessions",
		"workspace",
	}
	configPath := filepath.Join(root, "data", "config")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read recovered config: %w", err)
	}
	var recovered struct {
		Harness struct {
			Enabled       bool   `json:"enabled"`
			KnowledgeRoot string `json:"knowledge_root"`
			VaultPath     string `json:"vault_path"`
		} `json:"harness"`
		Ouroboros struct {
			Enabled bool `json:"enabled"`
		} `json:"ouroboros"`
		Memory struct {
			Catalog struct {
				Enabled *bool `json:"enabled"`
			} `json:"catalog"`
		} `json:"memory"`
	}
	if err := json.Unmarshal(data, &recovered); err != nil {
		return fmt.Errorf("decode recovered config: %w", err)
	}
	if recovered.Harness.Enabled {
		required = append(required, "harness")
		if strings.TrimSpace(recovered.Harness.KnowledgeRoot) != "" ||
			strings.TrimSpace(recovered.Harness.VaultPath) != "" {
			required = append(required, "knowledge")
		}
	}
	if recovered.Ouroboros.Enabled {
		required = append(required, "ouroboros")
	}
	if recovered.Memory.Catalog.Enabled == nil ||
		*recovered.Memory.Catalog.Enabled {
		required = append(required, "memory-catalog")
	}
	for _, name := range required {
		if !present[name] {
			return fmt.Errorf("backup is missing required source %s", name)
		}
	}
	if err := validatePilotAttestation(filepath.Join(
		root,
		"data",
		"credential-attestation",
	)); err != nil {
		return fmt.Errorf("recovered credential attestation: %w", err)
	}
	return nil
}

func validateTeamControlBackup(path string) error {
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	statePath := path
	if info.IsDir() {
		statePath = filepath.Join(path, "teamcontrol.json")
	}
	data, err := os.ReadFile(statePath)
	if errors.Is(err, os.ErrNotExist) {
		return errors.New("teamcontrol backup is missing teamcontrol.json")
	}
	if err != nil {
		return err
	}
	var header struct {
		SchemaVersion int `json:"schema_version"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return fmt.Errorf("decode teamcontrol backup: %w", err)
	}
	if header.SchemaVersion != 1 {
		return fmt.Errorf(
			"unsupported teamcontrol backup schema %d",
			header.SchemaVersion,
		)
	}
	return nil
}

func validateWorkstationBackup(root string) error {
	if _, err := os.Stat(root); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	for _, required := range []string{"tasks", "runners", "credentials", "evidence"} {
		info, err := os.Stat(filepath.Join(root, required))
		if err != nil || !info.IsDir() {
			return fmt.Errorf("workstation backup is missing %s directory", required)
		}
	}
	entries, err := os.ReadDir(filepath.Join(root, "tasks"))
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		var task workstation.Task
		data, err := os.ReadFile(filepath.Join(root, "tasks", entry.Name()))
		if err != nil {
			return err
		}
		if err := json.Unmarshal(data, &task); err != nil {
			return fmt.Errorf("decode workstation task %s: %w", entry.Name(), err)
		}
		var reference *workstation.EvidenceReference
		var runnerID string
		if task.Result != nil {
			reference = &task.Result.Evidence
			runnerID = task.Result.RunnerID
		} else if task.LastFailure != nil && task.LastFailure.Evidence != nil {
			reference = task.LastFailure.Evidence
			runnerID = task.LastFailure.RunnerID
		}
		if reference == nil {
			continue
		}
		evidencePath, err := safePilotJoin(
			filepath.Join(root, "evidence"),
			reference.Path,
		)
		if err != nil {
			return err
		}
		var bundle workstation.EvidenceBundle
		evidenceData, err := os.ReadFile(evidencePath)
		if err != nil {
			return fmt.Errorf("read evidence for task %s: %w", task.ID, err)
		}
		if err := json.Unmarshal(evidenceData, &bundle); err != nil {
			return fmt.Errorf("decode evidence for task %s: %w", task.ID, err)
		}
		if bundle.BundleSHA256 != reference.BundleSHA256 ||
			bundle.Signature != reference.Signature ||
			bundle.RunnerID != runnerID {
			return fmt.Errorf("evidence reference mismatch for task %s", task.ID)
		}
		key, err := restoredRunnerKey(root, runnerID, bundle.KeyID)
		if err != nil {
			return fmt.Errorf("resolve evidence key for task %s: %w", task.ID, err)
		}
		if err := workstation.VerifyEvidenceBundle(bundle, key); err != nil {
			return fmt.Errorf("verify evidence for task %s: %w", task.ID, err)
		}
	}
	return nil
}

func restoredRunnerKey(root, runnerID, keyID string) ([]byte, error) {
	active := filepath.Join(root, "credentials", runnerID+".key")
	if key, err := os.ReadFile(active); err == nil {
		if actual, idErr := workstation.DeviceKeyID(key); idErr == nil &&
			actual == keyID {
			return key, nil
		}
	}
	archive, err := safePilotJoin(
		filepath.Join(root, "credentials", "archive", runnerID),
		keyID+".key",
	)
	if err != nil {
		return nil, err
	}
	key, err := os.ReadFile(archive)
	if err != nil {
		return nil, err
	}
	actual, err := workstation.DeviceKeyID(key)
	if err != nil {
		return nil, err
	}
	if actual != keyID {
		return nil, errors.New("archived device key digest mismatch")
	}
	return key, nil
}

func validateDevelopmentEventChains(tasksRoot string) error {
	entries, err := os.ReadDir(tasksRoot)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			return fmt.Errorf("unexpected development task member %s", entry.Name())
		}
		path := filepath.Join(tasksRoot, entry.Name(), "events.jsonl")
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("open development events %s: %w", entry.Name(), err)
		}
		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
		var sequence int64 = 1
		var previous string
		for scanner.Scan() {
			var event dev.SessionEvent
			if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
				_ = file.Close()
				return fmt.Errorf("decode development event %s: %w", entry.Name(), err)
			}
			if event.Sequence != sequence || event.PreviousHash != previous ||
				event.TaskID != entry.Name() {
				_ = file.Close()
				return fmt.Errorf("development event chain identity mismatch for %s", entry.Name())
			}
			hashable := event
			hashable.Hash = ""
			encoded, err := json.Marshal(hashable)
			if err != nil {
				_ = file.Close()
				return err
			}
			digest := sha256.Sum256(encoded)
			if hex.EncodeToString(digest[:]) != event.Hash {
				_ = file.Close()
				return fmt.Errorf("development event hash mismatch for %s", entry.Name())
			}
			previous = event.Hash
			sequence++
		}
		scanErr := scanner.Err()
		closeErr := file.Close()
		if scanErr != nil || closeErr != nil {
			return errors.Join(scanErr, closeErr)
		}
		if sequence == 1 {
			return fmt.Errorf("development event chain is empty for %s", entry.Name())
		}
	}
	return nil
}

func safePilotJoin(root, relative string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(relative))
	if clean == "." || filepath.IsAbs(clean) ||
		clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("unsafe relative path %q", relative)
	}
	path := filepath.Join(root, clean)
	check, err := filepath.Rel(root, path)
	if err != nil || check == ".." ||
		strings.HasPrefix(check, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes backup root: %q", relative)
	}
	return path, nil
}

func pilotGitCommand(ctx context.Context, args ...string) (string, error) {
	command := exec.CommandContext(ctx, "git", args...)
	command.Env = []string{
		"PATH=/usr/bin:/bin",
		"LC_ALL=C",
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_TERMINAL_PROMPT=0",
	}
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", args[0], err, strings.TrimSpace(string(output)))
	}
	return string(output), nil
}

func requireNewRestoreTarget(path string) error {
	info, err := os.Lstat(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return nil
	case err != nil:
		return err
	case !info.IsDir():
		return errors.New("restore target exists and is not a directory")
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	if len(entries) != 0 {
		return errors.New("restore target must be new or empty")
	}
	return nil
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func valueMaps(value interface{}) []map[string]interface{} {
	items, ok := value.([]interface{})
	if !ok {
		if typed, ok := value.([]map[string]interface{}); ok {
			return typed
		}
		return nil
	}
	result := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		if object, ok := item.(map[string]interface{}); ok {
			result = append(result, object)
		}
	}
	return result
}

func stringValue(value interface{}) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func stringMapValue(value interface{}) map[string]string {
	result := make(map[string]string)
	switch typed := value.(type) {
	case map[string]interface{}:
		for key, item := range typed {
			if text, ok := item.(string); ok {
				result[key] = strings.TrimSpace(text)
			}
		}
	case map[string]string:
		for key, item := range typed {
			result[key] = strings.TrimSpace(item)
		}
	}
	return result
}

func containsStringValue(value interface{}, expected string) bool {
	switch typed := value.(type) {
	case []interface{}:
		for _, item := range typed {
			if stringValue(item) == expected {
				return true
			}
		}
	case []string:
		for _, item := range typed {
			if strings.TrimSpace(item) == expected {
				return true
			}
		}
	}
	return false
}

func boolValue(value interface{}) bool {
	result, _ := value.(bool)
	return result
}
