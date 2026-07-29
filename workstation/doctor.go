package workstation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

var runnerSafePath = defaultRunnerCommandPath()

func defaultRunnerCommandPath() string {
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		return os.Getenv("PATH")
	}
	return "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
}

type DoctorStatus string

const (
	DoctorPass DoctorStatus = "pass"
	DoctorWarn DoctorStatus = "warn"
	DoctorFail DoctorStatus = "fail"
)

type RunnerDoctorConfig struct {
	ExecutionProfile       ExecutionProfile
	DeviceKeyPath          string
	WorkRoot               string
	RepositoryPaths        map[string]string
	CodexCommand           string
	VerificationSandbox    []string
	UnsafeHostVerification bool
}

type RunnerDoctorCheck struct {
	ID      string       `json:"id"`
	Status  DoctorStatus `json:"status"`
	Summary string       `json:"summary"`
	Detail  string       `json:"detail,omitempty"`
}

type RunnerDoctorReport struct {
	SchemaVersion int                 `json:"schema_version"`
	GeneratedAt   time.Time           `json:"generated_at"`
	Ready         bool                `json:"ready"`
	Runtime       RunnerRuntime       `json:"runtime"`
	Capabilities  []string            `json:"capabilities"`
	Metadata      map[string]string   `json:"metadata"`
	Checks        []RunnerDoctorCheck `json:"checks"`
}

func (r RunnerDoctorReport) MarshalJSONIndent() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

func RunRunnerDoctor(
	ctx context.Context,
	cfg RunnerDoctorConfig,
) RunnerDoctorReport {
	runtimeInfo := CurrentRunnerRuntime()
	profile, profileErr := NormalizeExecutionProfile(
		string(cfg.ExecutionProfile),
	)
	metadata := RunnerRegistrationMetadataForProfile(profile)
	if len(cfg.VerificationSandbox) > 0 {
		if resolved, err := RunnerRegistrationMetadataForSandbox(
			cfg.VerificationSandbox[0],
		); err == nil {
			metadata = resolved
			metadata["execution_profile"] = string(profile)
			metadata["directory_boundary"] = "goclaw-worktree-v1"
			metadata["network_isolation"] = "required"
			metadata["security_posture"] = "strict"
		}
	} else if cfg.UnsafeHostVerification {
		metadata["isolation_backend"] = "external-vm"
	}
	report := RunnerDoctorReport{
		SchemaVersion: 1,
		GeneratedAt:   time.Now().UTC(),
		Ready:         true,
		Runtime:       runtimeInfo,
		Capabilities: RunnerRegistrationCapabilitiesForProfile(
			[]string{"codex"},
			profile,
		),
		Metadata: metadata,
	}
	add := func(check RunnerDoctorCheck) {
		report.Checks = append(report.Checks, check)
		if check.Status == DoctorFail {
			report.Ready = false
		}
	}

	if profileErr != nil {
		add(RunnerDoctorCheck{
			ID: "execution-profile", Status: DoctorFail,
			Summary: "Runner execution profile is unsupported",
			Detail:  profileErr.Error(),
		})
	} else {
		add(RunnerDoctorCheck{
			ID: "execution-profile", Status: DoctorPass,
			Summary: "Runner execution profile is explicit",
			Detail:  string(profile),
		})
	}

	if err := ValidateRunnerExecutionProfile(runtimeInfo, profile); err != nil {
		add(RunnerDoctorCheck{
			ID: "runtime", Status: DoctorFail,
			Summary: "Execution substrate does not satisfy the selected profile",
			Detail:  err.Error(),
		})
	} else {
		add(RunnerDoctorCheck{
			ID: "runtime", Status: DoctorPass,
			Summary: fmt.Sprintf(
				"%s/%s on %s satisfies %s",
				runtimeInfo.OS,
				runtimeInfo.Arch,
				runtimeInfo.Substrate,
				string(profile),
			),
		})
	}

	var gitCommand string
	var err error
	if profile == ExecutionProfileCodexDelegated {
		gitCommand, err = resolveConfiguredCommand("git")
	} else {
		gitCommand, err = findRunnerCommand("git")
	}
	if err != nil {
		add(RunnerDoctorCheck{
			ID: "git", Status: DoctorFail,
			Summary: "Git is unavailable on the fixed runner PATH",
			Detail:  err.Error(),
		})
	} else if output, runErr := doctorCommand(
		ctx,
		"",
		gitCommand,
		"--version",
	); runErr != nil {
		add(RunnerDoctorCheck{
			ID: "git", Status: DoctorFail,
			Summary: "Git version probe failed",
			Detail:  sanitizedDoctorOutput(output, runErr),
		})
	} else {
		add(RunnerDoctorCheck{
			ID: "git", Status: DoctorPass,
			Summary: strings.TrimSpace(output),
			Detail:  gitCommand,
		})
	}

	codexCommand := strings.TrimSpace(cfg.CodexCommand)
	if codexCommand == "" {
		codexCommand = "codex"
	}
	resolvedCodex, err := resolveConfiguredCommand(codexCommand)
	if err != nil {
		add(RunnerDoctorCheck{
			ID: "codex", Status: DoctorFail,
			Summary: "Codex CLI is unavailable",
			Detail:  err.Error(),
		})
	} else if output, runErr := doctorCommand(
		ctx,
		"",
		resolvedCodex,
		"--version",
	); runErr != nil {
		add(RunnerDoctorCheck{
			ID: "codex", Status: DoctorFail,
			Summary: "Codex version probe failed",
			Detail:  sanitizedDoctorOutput(output, runErr),
		})
	} else {
		add(RunnerDoctorCheck{
			ID: "codex", Status: DoctorPass,
			Summary: strings.TrimSpace(output),
			Detail:  resolvedCodex,
		})
	}

	codexHome, err := resolveLocalCodexHome()
	codexHomeReady := false
	if err != nil {
		add(RunnerDoctorCheck{
			ID: "codex-home", Status: DoctorFail,
			Summary: "Codex OAuth home is invalid",
			Detail:  err.Error(),
		})
	} else if info, statErr := os.Stat(codexHome); statErr != nil || !info.IsDir() {
		detail := "not a directory"
		if statErr != nil {
			detail = statErr.Error()
		}
		add(RunnerDoctorCheck{
			ID: "codex-home", Status: DoctorFail,
			Summary: "Codex OAuth home is unavailable",
			Detail:  codexHome + ": " + detail,
		})
	} else if pathErr := validateRunnerLocalPath(runtimeInfo, codexHome); pathErr != nil {
		add(RunnerDoctorCheck{
			ID: "codex-home", Status: DoctorFail,
			Summary: "Codex OAuth home crosses the guest boundary",
			Detail:  pathErr.Error(),
		})
	} else if ownerErr := validateTrustedPathChain(
		codexHome,
		true,
	); ownerErr != nil {
		add(RunnerDoctorCheck{
			ID: "codex-home", Status: DoctorFail,
			Summary: "Codex OAuth home has an unsafe owner or parent",
			Detail:  ownerErr.Error(),
		})
	} else {
		codexHomeReady = true
		add(RunnerDoctorCheck{
			ID: "codex-home", Status: DoctorPass,
			Summary: "Codex OAuth stays on this runner",
			Detail:  codexHome,
		})
	}
	if resolvedCodex != "" && codexHomeReady {
		output, loginErr := doctorCommandWithEnvironment(
			ctx,
			"",
			[]string{"CODEX_HOME=" + codexHome},
			resolvedCodex,
			"login",
			"status",
		)
		if loginErr != nil {
			add(RunnerDoctorCheck{
				ID: "codex-login", Status: DoctorFail,
				Summary: "Codex subscription login is unavailable",
				Detail:  sanitizedDoctorOutput(output, loginErr),
			})
		} else {
			add(RunnerDoctorCheck{
				ID: "codex-login", Status: DoctorPass,
				Summary: "Codex subscription login status passed",
				Detail:  strings.TrimSpace(truncateLocalOutput(output, 1024)),
			})
		}
	}

	if err := validateDeviceKeyFile(cfg.DeviceKeyPath); err != nil {
		add(RunnerDoctorCheck{
			ID: "device-key", Status: DoctorFail,
			Summary: "Runner device key is unavailable or overexposed",
			Detail:  err.Error(),
		})
	} else if pathErr := validateRunnerLocalPath(runtimeInfo, cfg.DeviceKeyPath); pathErr != nil {
		add(RunnerDoctorCheck{
			ID: "device-key", Status: DoctorFail,
			Summary: "Runner device key crosses the guest boundary",
			Detail:  pathErr.Error(),
		})
	} else {
		add(RunnerDoctorCheck{
			ID: "device-key", Status: DoctorPass,
			Summary: "Runner device key is a protected local file",
			Detail:  filepath.Clean(cfg.DeviceKeyPath),
		})
	}

	if err := doctorWorkRoot(runtimeInfo, cfg.WorkRoot); err != nil {
		add(RunnerDoctorCheck{
			ID: "work-root", Status: DoctorFail,
			Summary: "Runner work root is unsafe",
			Detail:  err.Error(),
		})
	} else {
		add(RunnerDoctorCheck{
			ID: "work-root", Status: DoctorPass,
			Summary: "Runner work root passed the directory boundary audit",
			Detail:  filepath.Clean(cfg.WorkRoot),
		})
	}

	repositoryIDs := make([]string, 0, len(cfg.RepositoryPaths))
	for id := range cfg.RepositoryPaths {
		repositoryIDs = append(repositoryIDs, id)
	}
	sort.Strings(repositoryIDs)
	if len(repositoryIDs) == 0 {
		add(RunnerDoctorCheck{
			ID: "repositories", Status: DoctorFail,
			Summary: "At least one repository mapping is required",
		})
	} else {
		for _, id := range repositoryIDs {
			path := cfg.RepositoryPaths[id]
			checkID := "repository:" + id
			if err := doctorRepository(
				ctx,
				runtimeInfo,
				gitCommand,
				id,
				path,
			); err != nil {
				add(RunnerDoctorCheck{
					ID: checkID, Status: DoctorFail,
					Summary: "Repository failed the local safety audit",
					Detail:  err.Error(),
				})
			} else {
				add(RunnerDoctorCheck{
					ID: checkID, Status: DoctorPass,
					Summary: "Repository is local and Git configuration is inert",
					Detail:  filepath.Clean(path),
				})
			}
		}
	}

	if profile == ExecutionProfileCodexDelegated &&
		(len(cfg.VerificationSandbox) > 0 || cfg.UnsafeHostVerification) {
		add(RunnerDoctorCheck{
			ID: "verification-isolation", Status: DoctorFail,
			Summary: "Delegated profile cannot use strict isolation flags",
		})
	} else if profile == ExecutionProfileCodexDelegated {
		add(RunnerDoctorCheck{
			ID: "verification-isolation", Status: DoctorWarn,
			Summary: "OS-level process and network isolation is not provided",
			Detail: "GoClaw enforces the worktree/diff boundary; Codex named " +
				"permissions enforce tool access. Use strict for untrusted code.",
		})
	} else if len(cfg.VerificationSandbox) > 0 &&
		cfg.UnsafeHostVerification {
		add(RunnerDoctorCheck{
			ID: "verification-isolation", Status: DoctorFail,
			Summary: "Verification isolation modes are mutually exclusive",
		})
	} else if len(cfg.VerificationSandbox) > 0 {
		wrapper := cfg.VerificationSandbox[0]
		if err := validateSandboxExecutable(wrapper); err != nil {
			add(RunnerDoctorCheck{
				ID: "verification-isolation", Status: DoctorFail,
				Summary: "Verification wrapper is unsafe",
				Detail:  err.Error(),
			})
		} else if output, runErr := doctorCommand(
			ctx,
			"",
			wrapper,
			"--goclaw-doctor",
		); runErr != nil {
			add(RunnerDoctorCheck{
				ID: "verification-isolation", Status: DoctorFail,
				Summary: "Verification wrapper capability probe failed",
				Detail:  sanitizedDoctorOutput(output, runErr),
			})
		} else if !strings.Contains(output, "goclaw-verifier/linux-bwrap-v1") {
			add(RunnerDoctorCheck{
				ID: "verification-isolation", Status: DoctorFail,
				Summary: "Verification wrapper does not implement the pilot contract",
				Detail:  strings.TrimSpace(output),
			})
		} else {
			add(RunnerDoctorCheck{
				ID: "verification-isolation", Status: DoctorPass,
				Summary: "bubblewrap network and namespace probe passed",
				Detail:  wrapper,
			})
		}
	} else if cfg.UnsafeHostVerification {
		add(RunnerDoctorCheck{
			ID: "verification-isolation", Status: DoctorWarn,
			Summary: "Host verification is explicitly enabled",
			Detail: "Use only inside a disposable, externally isolated Linux VM; " +
				"the runner cannot prove that outer boundary.",
		})
	} else {
		add(RunnerDoctorCheck{
			ID: "verification-isolation", Status: DoctorFail,
			Summary: "Verification sandbox is required",
		})
	}

	return report
}

func doctorWorkRoot(info RunnerRuntime, path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("work_root is required")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if err := validateRunnerLocalPath(info, absolute); err != nil {
		return err
	}
	current := absolute
	for {
		stat, statErr := os.Stat(current)
		if statErr == nil {
			if !stat.IsDir() {
				return fmt.Errorf("%s is not a directory", current)
			}
			if err := validateTrustedPathChain(current, true); err != nil {
				return err
			}
			if stat.Mode().Perm()&0o200 == 0 {
				return fmt.Errorf("%s is not writable by its owner", current)
			}
			return nil
		}
		if !errors.Is(statErr, os.ErrNotExist) {
			return statErr
		}
		parent := filepath.Dir(current)
		if parent == current {
			return fmt.Errorf("no existing parent for %s", absolute)
		}
		current = parent
	}
}

func doctorRepository(
	ctx context.Context,
	info RunnerRuntime,
	gitCommand, id, path string,
) error {
	if err := validateID(id); err != nil {
		return fmt.Errorf("repository id: %w", err)
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	stat, err := os.Stat(absolute)
	if err != nil {
		return err
	}
	if !stat.IsDir() {
		return fmt.Errorf("%s is not a directory", absolute)
	}
	if err := validateTrustedPathChain(absolute, true); err != nil {
		return err
	}
	if err := validateRunnerLocalPath(info, absolute); err != nil {
		return err
	}
	if strings.TrimSpace(gitCommand) == "" {
		return errors.New("Git is unavailable")
	}
	if err := auditRepositoryGitConfiguration(ctx, gitCommand, absolute); err != nil {
		return err
	}
	result, runErr := rawGitCommand(
		ctx,
		gitCommand,
		absolute,
		"rev-parse",
		"--is-inside-work-tree",
	)
	if runErr != nil || strings.TrimSpace(result.Stdout) != "true" {
		return fmt.Errorf(
			"not a Git worktree: %s",
			commandFailure(runErr, result),
		)
	}
	return nil
}

func validateDeviceKeyFile(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("device_key_path is required")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("device_key_path must be a regular file")
	}
	if err := validateCurrentUserOwner(path, info); err != nil {
		return err
	}
	if err := validatePrivateFilePermissions(path, info); err != nil {
		return err
	}
	if err := validateTrustedPathChain(filepath.Dir(path), false); err != nil {
		return fmt.Errorf("device_key_path parent: %w", err)
	}
	key, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	_, err = DeviceKeyID(key)
	return err
}

func validateSandboxExecutable(path string) error {
	path = strings.TrimSpace(path)
	if !filepath.IsAbs(path) {
		return errors.New(
			"verification_sandbox executable must be an absolute path",
		)
	}
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("verification_sandbox executable: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
		return errors.New(
			"verification_sandbox executable must be a regular executable file",
		)
	}
	if info.Mode().Perm()&0o022 != 0 {
		return errors.New(
			"verification_sandbox executable must not be writable by group or others",
		)
	}
	if err := validateRootOwner(path, info); err != nil {
		return err
	}
	if err := validateTrustedPathChain(filepath.Dir(path), false); err != nil {
		return fmt.Errorf("verification_sandbox parent: %w", err)
	}
	return nil
}

func resolveConfiguredCommand(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("command is required")
	}
	if filepath.IsAbs(name) {
		return validateResolvedCommand(name)
	}
	if strings.ContainsRune(name, filepath.Separator) {
		return "", errors.New("configured command must be absolute or on PATH")
	}
	path, err := exec.LookPath(name)
	if err != nil {
		return "", err
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return validateResolvedCommand(absolute)
}

func findRunnerCommand(name string) (string, error) {
	for _, directory := range filepath.SplitList(runnerSafePath) {
		candidate := filepath.Join(directory, name)
		if result, err := validateResolvedCommand(candidate); err == nil {
			return result, nil
		}
	}
	return "", fmt.Errorf("%s not found on fixed PATH %s", name, runnerSafePath)
}

func validateResolvedCommand(path string) (string, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if err := validateExecutableFile(resolved, info); err != nil {
		return "", err
	}
	return filepath.Clean(resolved), nil
}

func doctorCommand(
	ctx context.Context,
	directory, name string,
	args ...string,
) (string, error) {
	return doctorCommandWithEnvironment(ctx, directory, nil, name, args...)
}

func doctorCommandWithEnvironment(
	ctx context.Context,
	directory string,
	extraEnvironment []string,
	name string,
	args ...string,
) (string, error) {
	probeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	command := exec.CommandContext(probeCtx, name, args...)
	prepareLocalCommand(command)
	command.Dir = directory
	command.Env = append([]string{
		"HOME=/nonexistent",
		"PATH=" + runnerSafePath,
		"LANG=C.UTF-8",
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_ATTR_NOSYSTEM=1",
		"GIT_TERMINAL_PROMPT=0",
	}, extraEnvironment...)
	output, err := command.CombinedOutput()
	if errors.Is(probeCtx.Err(), context.DeadlineExceeded) {
		return string(output), context.DeadlineExceeded
	}
	return string(output), err
}

func sanitizedDoctorOutput(output string, err error) string {
	value := strings.TrimSpace(truncateLocalOutput(output, 8*1024))
	if value == "" && err != nil {
		value = err.Error()
	}
	return value
}
