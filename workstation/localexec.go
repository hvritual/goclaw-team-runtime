package workstation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const maxLocalCommandOutputBytes = 16 * 1024 * 1024
const codexRunnerPermissionProfile = "goclaw-runner"

type LocalExecConfig struct {
	RunnerID               string            `mapstructure:"runner_id" json:"runner_id" yaml:"runner_id"`
	ExecutionProfile       ExecutionProfile  `mapstructure:"execution_profile" json:"execution_profile,omitempty" yaml:"execution_profile,omitempty"`
	DeviceKeyPath          string            `mapstructure:"device_key_path" json:"device_key_path" yaml:"device_key_path"`
	WorkRoot               string            `mapstructure:"work_root" json:"work_root" yaml:"work_root"`
	RepositoryPaths        map[string]string `mapstructure:"repository_paths" json:"repository_paths" yaml:"repository_paths"`
	CodexCommand           string            `mapstructure:"codex_command" json:"codex_command" yaml:"codex_command"`
	CodexModel             string            `mapstructure:"codex_model" json:"codex_model" yaml:"codex_model"`
	TimeoutSeconds         int               `mapstructure:"timeout_seconds" json:"timeout_seconds" yaml:"timeout_seconds"`
	AllowedEnvironment     []string          `mapstructure:"allowed_environment" json:"allowed_environment,omitempty" yaml:"allowed_environment,omitempty"`
	VerificationSandbox    []string          `mapstructure:"verification_sandbox" json:"verification_sandbox,omitempty" yaml:"verification_sandbox,omitempty"`
	UnsafeHostVerification bool              `mapstructure:"unsafe_host_verification" json:"unsafe_host_verification,omitempty" yaml:"unsafe_host_verification,omitempty"`
}

// LocalExecutor has no control-plane network client. Callers claim a task,
// execute it locally using the workstation's Codex OAuth login, then submit the
// returned signed bundle through their chosen transport.
type LocalExecutor struct {
	cfg             LocalExecConfig
	deviceKey       []byte
	codexHome       string
	gitCommand      string
	runtime         RunnerRuntime
	runtimeMetadata map[string]string
}

func NewLocalExecutor(cfg LocalExecConfig) (*LocalExecutor, error) {
	runtimeInfo := CurrentRunnerRuntime()
	profile, err := NormalizeExecutionProfile(string(cfg.ExecutionProfile))
	if err != nil {
		return nil, err
	}
	cfg.ExecutionProfile = profile
	if err := ValidateRunnerExecutionProfile(runtimeInfo, profile); err != nil {
		return nil, err
	}
	cfg.RunnerID = strings.TrimSpace(cfg.RunnerID)
	if err := validateID(cfg.RunnerID); err != nil {
		return nil, err
	}
	if err := validateDeviceKeyFile(cfg.DeviceKeyPath); err != nil {
		return nil, err
	}
	deviceKey, err := os.ReadFile(cfg.DeviceKeyPath)
	if err != nil {
		return nil, err
	}
	var gitCommand string
	if profile == ExecutionProfileCodexDelegated {
		gitCommand, err = resolveConfiguredCommand("git")
	} else {
		gitCommand, err = findRunnerCommand("git")
	}
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(cfg.WorkRoot) == "" {
		return nil, errors.New("work_root is required")
	}
	workRoot, err := filepath.Abs(cfg.WorkRoot)
	if err != nil {
		return nil, err
	}
	if err := ensureDir(workRoot, 0o700); err != nil {
		return nil, err
	}
	if err := validateRunnerLocalPath(runtimeInfo, workRoot); err != nil {
		return nil, fmt.Errorf("work_root: %w", err)
	}
	if err := validateTrustedPathChain(workRoot, true); err != nil {
		return nil, fmt.Errorf("work_root ownership: %w", err)
	}
	cfg.WorkRoot = workRoot
	if len(cfg.RepositoryPaths) == 0 {
		return nil, errors.New("repository_paths requires at least one repository")
	}
	repositories := make(map[string]string, len(cfg.RepositoryPaths))
	for id, path := range cfg.RepositoryPaths {
		id = strings.TrimSpace(id)
		if err := validateID(id); err != nil {
			return nil, fmt.Errorf("repository id: %w", err)
		}
		absolute, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		info, err := os.Stat(absolute)
		if err != nil {
			return nil, fmt.Errorf("repository %s: %w", id, err)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("repository %s path is not a directory", id)
		}
		if err := validateRunnerLocalPath(runtimeInfo, absolute); err != nil {
			return nil, fmt.Errorf("repository %s: %w", id, err)
		}
		if err := validateTrustedPathChain(absolute, true); err != nil {
			return nil, fmt.Errorf("repository %s ownership: %w", id, err)
		}
		auditCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		err = auditRepositoryGitConfiguration(auditCtx, gitCommand, absolute)
		cancel()
		if err != nil {
			return nil, fmt.Errorf("repository %s: %w", id, err)
		}
		repositories[id] = absolute
	}
	cfg.RepositoryPaths = repositories
	if strings.TrimSpace(cfg.CodexCommand) == "" {
		cfg.CodexCommand = "codex"
	}
	if cfg.TimeoutSeconds <= 0 {
		cfg.TimeoutSeconds = 3600
	}
	cfg.AllowedEnvironment = normalizeStrings(cfg.AllowedEnvironment)
	for index := range cfg.VerificationSandbox {
		cfg.VerificationSandbox[index] = strings.TrimSpace(
			cfg.VerificationSandbox[index],
		)
		if cfg.VerificationSandbox[index] == "" {
			return nil, errors.New(
				"verification_sandbox entries must not be empty",
			)
		}
	}
	if len(cfg.VerificationSandbox) > 0 && cfg.UnsafeHostVerification {
		return nil, errors.New(
			"verification_sandbox and unsafe_host_verification are mutually exclusive",
		)
	}
	if profile == ExecutionProfileCodexDelegated &&
		(len(cfg.VerificationSandbox) > 0 || cfg.UnsafeHostVerification) {
		return nil, errors.New(
			"codex-delegated profile cannot be combined with strict verification isolation flags",
		)
	}
	if len(cfg.VerificationSandbox) > 0 {
		sandboxExecutable := cfg.VerificationSandbox[0]
		if err := validateSandboxExecutable(sandboxExecutable); err != nil {
			return nil, err
		}
	}
	if profile == ExecutionProfileStrict &&
		len(cfg.VerificationSandbox) == 0 && !cfg.UnsafeHostVerification {
		return nil, errors.New(
			"verification_sandbox is required; use the bundled bwrap wrapper or explicitly opt into unsafe_host_verification only inside an already isolated VM/container",
		)
	}
	if resolved, resolveErr := resolveConfiguredCommand(
		cfg.CodexCommand,
	); resolveErr == nil {
		cfg.CodexCommand = resolved
	} else if filepath.IsAbs(cfg.CodexCommand) ||
		strings.ContainsRune(cfg.CodexCommand, filepath.Separator) {
		return nil, fmt.Errorf("codex_command: %w", resolveErr)
	}
	runtimeMetadata := RunnerRegistrationMetadataForProfile(profile)
	if len(cfg.VerificationSandbox) > 0 {
		runtimeMetadata, err = RunnerRegistrationMetadataForSandbox(
			cfg.VerificationSandbox[0],
		)
		if err != nil {
			return nil, err
		}
		runtimeMetadata["execution_profile"] = string(profile)
		runtimeMetadata["directory_boundary"] = "goclaw-worktree-v1"
		runtimeMetadata["network_isolation"] = "required"
		runtimeMetadata["security_posture"] = "strict"
	} else if profile == ExecutionProfileStrict {
		runtimeMetadata["isolation_backend"] = "external-vm"
	}
	codexHome, err := resolveLocalCodexHome()
	if err != nil {
		return nil, err
	}
	if err := validateRunnerLocalPath(runtimeInfo, codexHome); err != nil {
		return nil, fmt.Errorf("CODEX_HOME: %w", err)
	}
	if info, statErr := os.Stat(codexHome); statErr == nil && info.IsDir() {
		if err := validateTrustedPathChain(codexHome, true); err != nil {
			return nil, fmt.Errorf("CODEX_HOME ownership: %w", err)
		}
	}
	return &LocalExecutor{
		cfg:             cfg,
		deviceKey:       append([]byte(nil), deviceKey...),
		codexHome:       codexHome,
		gitCommand:      gitCommand,
		runtime:         runtimeInfo,
		runtimeMetadata: cloneMetadata(runtimeMetadata),
	}, nil
}

func (e *LocalExecutor) Config() LocalExecConfig {
	result := e.cfg
	result.RepositoryPaths = cloneMetadata(e.cfg.RepositoryPaths)
	result.AllowedEnvironment = append([]string(nil), e.cfg.AllowedEnvironment...)
	result.VerificationSandbox = append(
		[]string(nil),
		e.cfg.VerificationSandbox...,
	)
	return result
}

// ExecuteClaim never commits, pushes, merges, or removes its worktree. A
// non-nil error means the signed bundle has Outcome "failed" and should be
// submitted with Service.Fail; a nil error is suitable for Service.Complete.
func (e *LocalExecutor) ExecuteClaim(ctx context.Context, claim ClaimResult) (EvidenceBundle, error) {
	started := time.Now().UTC()
	task := claim.Task
	lease := claim.Lease
	bundle := EvidenceBundle{
		TaskID:              task.ID,
		ProjectID:           task.ProjectID,
		ExecutionPackSHA256: task.ExecutionPackSHA256,
		RunnerID:            e.cfg.RunnerID,
		LeaseID:             lease.ID,
		Attempt:             lease.Attempt,
		Outcome:             "failed",
		StartedAt:           started,
		BaseCommit:          task.ExecutionPack.BaseCommit,
		Branch:              task.ExecutionPack.Branch,
		Metadata: map[string]string{
			"correlation_id":     task.ExecutionPack.CorrelationID,
			"harness_version":    task.ExecutionPack.HarnessVersion,
			"policy_bundle_hash": task.ExecutionPack.PolicyBundleHash,
			"runtime_contract":   e.runtime.Contract,
			"execution_profile":  string(e.cfg.ExecutionProfile),
			"runner_goos":        e.runtime.OS,
			"runner_goarch":      e.runtime.Arch,
			"host_profile":       e.runtime.Substrate,
			"isolation_backend":  e.runtimeMetadata["isolation_backend"],
			"sandbox_sha256":     e.runtimeMetadata["sandbox_sha256"],
		},
		TraceIDs: []string{"local:" + lease.ID},
	}
	var failures []string
	failSetup := func(message string) (EvidenceBundle, error) {
		bundle.Checks = append(bundle.Checks, EvidenceCheck{
			Name: "runner-setup", Passed: false, Details: message,
		})
		return e.finishEvidence(bundle, []string{message})
	}
	if err := validateID(task.ID); err != nil {
		return failSetup(err.Error())
	}
	if err := validateID(lease.ID); err != nil {
		return failSetup(err.Error())
	}
	if task.Status != TaskLeased || task.Lease == nil {
		return failSetup("claimed task is not in leased status")
	}
	if lease.RunnerID != e.cfg.RunnerID || task.Lease.RunnerID != e.cfg.RunnerID {
		return failSetup("claim does not belong to this runner")
	}
	if task.Lease.ID != lease.ID || task.Attempt != lease.Attempt {
		return failSetup("claim lease or attempt does not match task projection")
	}
	packHash, err := HashExecutionPack(task.ExecutionPack)
	if err != nil {
		return failSetup(err.Error())
	}
	if packHash != task.ExecutionPackSHA256 {
		return failSetup("execution pack SHA-256 mismatch")
	}
	taskProfile, err := NormalizeExecutionProfile(
		string(task.ExecutionPack.ExecutionProfile),
	)
	if err != nil {
		return failSetup(err.Error())
	}
	if taskProfile != e.cfg.ExecutionProfile {
		return failSetup("execution pack profile does not match runner profile")
	}
	repository, ok := e.cfg.RepositoryPaths[task.ExecutionPack.RepositoryID]
	if !ok {
		return failSetup("repository is not registered on this workstation")
	}
	if err := auditRepositoryGitConfiguration(
		ctx,
		e.gitCommand,
		repository,
	); err != nil {
		return failSetup("repository Git safety audit: " + err.Error())
	}
	base, err := e.git(ctx, repository, "rev-parse", "--verify", task.ExecutionPack.BaseCommit+"^{commit}")
	if err != nil {
		return failSetup("resolve frozen base commit: " + commandFailure(err, base))
	}
	if strings.TrimSpace(base.Stdout) != task.ExecutionPack.BaseCommit {
		return failSetup("frozen base commit resolved to a different SHA")
	}
	if err := auditGitAttributesAtCommit(
		ctx,
		e.gitCommand,
		repository,
		task.ExecutionPack.BaseCommit,
	); err != nil {
		return failSetup(err.Error())
	}

	worktreeRel := filepath.Join(
		task.ID,
		fmt.Sprintf("r%d-a%d", task.ExecutionPack.TaskRevision, lease.Attempt),
	)
	worktree, err := safeJoin(e.cfg.WorkRoot, worktreeRel)
	if err != nil {
		return failSetup(err.Error())
	}
	if _, err := os.Stat(worktree); err == nil {
		return failSetup("revision worktree already exists; refusing stale reuse")
	} else if !errors.Is(err, os.ErrNotExist) {
		return failSetup(err.Error())
	}
	if err := ensureDir(filepath.Dir(worktree), 0o700); err != nil {
		return failSetup(err.Error())
	}
	branch := localBranch(task.ExecutionPack.Branch, task.ID, task.ExecutionPack.TaskRevision, lease.Attempt)
	checkBranch, err := e.git(ctx, repository, "check-ref-format", "--branch", branch)
	if err != nil {
		return failSetup("invalid task branch: " + commandFailure(err, checkBranch))
	}
	addWorktree, err := e.git(
		ctx,
		repository,
		"worktree",
		"add",
		"-b",
		branch,
		worktree,
		task.ExecutionPack.BaseCommit,
	)
	if err != nil {
		return failSetup("create revision worktree: " + commandFailure(err, addWorktree))
	}
	bundle.Branch = branch
	bundle.Metadata["worktree_relative"] = filepath.ToSlash(worktreeRel)
	bundle.Checks = append(bundle.Checks, EvidenceCheck{
		Name: "runner-setup", Passed: true, Details: "created revision-isolated worktree",
	})

	artifactRoot, err := safeJoin(
		e.cfg.WorkRoot,
		filepath.Join(".evidence", task.ID, lease.ID),
	)
	if err != nil {
		return e.finishEvidence(bundle, []string{err.Error()})
	}
	if err := ensureDir(artifactRoot, 0o700); err != nil {
		return e.finishEvidence(bundle, []string{err.Error()})
	}
	verificationHome := filepath.Join(artifactRoot, "verify-home")
	if err := ensureDir(verificationHome, 0o700); err != nil {
		return e.finishEvidence(bundle, []string{err.Error()})
	}
	codexResult, codexErr := e.runCodex(
		ctx,
		worktree,
		artifactRoot,
		task.ExecutionPack,
	)
	codexCheck := EvidenceCheck{
		Name:       "codex-exec",
		Passed:     codexErr == nil,
		ExitCode:   codexResult.ExitCode,
		DurationMS: codexResult.DurationMS,
		Details:    truncateLocalOutput(codexResult.Stderr, 64*1024),
	}
	if codexResult.TimedOut {
		codexCheck.Details = "codex execution timed out"
	}
	bundle.Checks = append(bundle.Checks, codexCheck)
	if codexErr != nil {
		failures = append(failures, "codex execution: "+commandFailure(codexErr, codexResult))
	}
	if codexResult.Stdout != "" {
		path := filepath.Join(artifactRoot, "codex-events.jsonl")
		if err := writeBytesAtomic(path, []byte(codexResult.Stdout), 0o600); err != nil {
			failures = append(failures, "write Codex trace: "+err.Error())
		} else {
			bundle.Artifacts = append(bundle.Artifacts, EvidenceArtifact{
				Name:      "codex-events",
				URI:       localEvidenceURI(e.cfg.RunnerID, task.ID, lease.ID, "codex-events"),
				SHA256:    sha256Bytes([]byte(codexResult.Stdout)),
				SizeBytes: int64(len(codexResult.Stdout)),
			})
		}
	}
	if codexResult.Stderr != "" {
		path := filepath.Join(artifactRoot, "codex-stderr.txt")
		if err := writeBytesAtomic(path, []byte(codexResult.Stderr), 0o600); err != nil {
			failures = append(failures, "write Codex stderr: "+err.Error())
		} else {
			bundle.Artifacts = append(bundle.Artifacts, EvidenceArtifact{
				Name:      "codex-stderr",
				URI:       localEvidenceURI(e.cfg.RunnerID, task.ID, lease.ID, "codex-stderr"),
				SHA256:    sha256Bytes([]byte(codexResult.Stderr)),
				SizeBytes: int64(len(codexResult.Stderr)),
			})
		}
	}

	for _, spec := range task.ExecutionPack.Verification {
		check := EvidenceCheck{Name: valueOr(spec.Name, strings.Join(spec.Argv, " "))}
		if len(spec.Argv) == 0 || strings.TrimSpace(spec.Argv[0]) == "" {
			check.Details = "verification argv is empty"
			failures = append(failures, check.Name+": "+check.Details)
			bundle.Checks = append(bundle.Checks, check)
			continue
		}
		result, runErr := e.verificationCommand(
			ctx,
			worktree,
			verificationHome,
			spec.Argv[0],
			spec.Argv[1:]...,
		)
		check.ExitCode = result.ExitCode
		check.DurationMS = result.DurationMS
		check.Passed = runErr == nil && result.ExitCode == 0
		check.Details = truncateLocalOutput(
			strings.TrimSpace(result.Stdout+"\n"+result.Stderr),
			64*1024,
		)
		if !check.Passed {
			failures = append(failures, "verification "+check.Name+": "+commandFailure(runErr, result))
		}
		bundle.Checks = append(bundle.Checks, check)
	}

	if err := auditRepositoryGitConfiguration(
		ctx,
		e.gitCommand,
		repository,
	); err != nil {
		failures = append(failures, "post-execution Git safety audit: "+err.Error())
		bundle.Checks = append(bundle.Checks, EvidenceCheck{
			Name: "git-safety", Passed: false, Details: err.Error(),
		})
	} else {
		bundle.Checks = append(bundle.Checks, EvidenceCheck{
			Name: "git-safety", Passed: true,
			Details: "hooks, filters, fsmonitor, includes, credentials, and external drivers remain disabled",
		})
	}

	// Verification commands are part of the frozen execution contract, but
	// they may format or generate files. Collect the final diff only after all
	// checks so the signed patch and scope decision describe the exact
	// worktree that was verified.
	changedFiles, diff, collectErr := e.collectChanges(ctx, worktree, task.ExecutionPack.BaseCommit)
	if collectErr != nil {
		failures = append(failures, "collect changes: "+collectErr.Error())
	} else {
		bundle.ChangedFiles = changedFiles
		if len(diff) > MaxEvidenceDiffPatchBytes {
			failures = append(
				failures,
				fmt.Sprintf("diff patch exceeds %d bytes", MaxEvidenceDiffPatchBytes),
			)
		} else if diff != "" {
			bundle.DiffPatch = diff
			bundle.DiffSHA256 = sha256Bytes([]byte(diff))
		}
	}

	policyFailures := localScopeViolations(worktree, changedFiles, task.ExecutionPack)
	policyPassed := collectErr == nil && len(policyFailures) == 0 && len(diff) <= MaxEvidenceDiffPatchBytes
	bundle.Checks = append(bundle.Checks, EvidenceCheck{
		Name:    "scope-policy",
		Passed:  policyPassed,
		Details: strings.Join(policyFailures, "; "),
	})
	failures = append(failures, policyFailures...)

	head, headErr := e.git(ctx, worktree, "rev-parse", "HEAD")
	if headErr != nil {
		failures = append(failures, "read worktree head: "+commandFailure(headErr, head))
	} else {
		bundle.HeadCommit = strings.TrimSpace(head.Stdout)
		if bundle.HeadCommit != task.ExecutionPack.BaseCommit {
			failures = append(failures, "Codex created a commit; automatic commit/push/merge is forbidden")
			bundle.Checks = append(bundle.Checks, EvidenceCheck{
				Name: "no-automatic-commit", Passed: false, Details: bundle.HeadCommit,
			})
		} else {
			bundle.Checks = append(bundle.Checks, EvidenceCheck{
				Name: "no-automatic-commit", Passed: true,
			})
		}
	}
	return e.finishEvidence(bundle, failures)
}

func (e *LocalExecutor) finishEvidence(
	bundle EvidenceBundle,
	failures []string,
) (EvidenceBundle, error) {
	bundle.FinishedAt = time.Now().UTC()
	if len(failures) == 0 {
		bundle.Outcome = "completed"
	} else {
		bundle.Outcome = "failed"
		bundle.Metadata["failure_count"] = fmt.Sprintf("%d", len(failures))
		bundle.Metadata["failure_summary"] = truncateLocalOutput(strings.Join(failures, "; "), 16*1024)
	}
	signed, err := SignEvidenceBundle(bundle, e.deviceKey)
	if err != nil {
		return EvidenceBundle{}, err
	}
	if len(failures) > 0 {
		return signed, fmt.Errorf("local execution failed: %s", strings.Join(failures, "; "))
	}
	return signed, nil
}

func (e *LocalExecutor) runCodex(
	ctx context.Context,
	worktree string,
	runtimeRoot string,
	pack ExecutionPack,
) (localCommandResult, error) {
	packJSON, err := json.MarshalIndent(pack, "", "  ")
	if err != nil {
		return localCommandResult{}, err
	}
	prompt := strings.Join([]string{
		"You are the local GoClaw workstation Codex Hand.",
		"Implement only the immutable ExecutionPack below in the current worktree.",
		"Do not commit, push, merge, modify Git configuration, or leave the worktree.",
		"Respect allowed_paths, denied_paths, the frozen base commit, and repository instructions.",
		"Run focused checks while working; the workstation will independently run frozen verification argv.",
		"\nEXECUTION_PACK:\n" + string(packJSON),
	}, "\n")
	runtimeHome := filepath.Join(runtimeRoot, "codex-runtime-home")
	cacheHome := filepath.Join(runtimeHome, ".cache")
	configHome := filepath.Join(runtimeHome, ".config")
	runtimeDir := filepath.Join(runtimeHome, ".runtime")
	tempDir := filepath.Join(runtimeHome, ".tmp")
	for _, path := range []string{
		runtimeHome,
		cacheHome,
		configHome,
		runtimeDir,
		tempDir,
	} {
		if err := ensureDir(path, 0o700); err != nil {
			return localCommandResult{ExitCode: -1}, err
		}
	}
	codexEnvironment := []string{
		"HOME=" + runtimeHome,
		"CODEX_HOME=" + e.codexHome,
		"XDG_CACHE_HOME=" + cacheHome,
		"XDG_CONFIG_HOME=" + configHome,
		"XDG_RUNTIME_DIR=" + runtimeDir,
		"TMPDIR=" + tempDir,
		"TMP=" + tempDir,
		"TEMP=" + tempDir,
		"PATH=" + e.executionPath(),
		"GIT_TERMINAL_PROMPT=0",
	}
	if err := e.verifyCodexCredentialIsolation(
		ctx,
		worktree,
		codexEnvironment,
	); err != nil {
		return localCommandResult{ExitCode: -1}, err
	}
	args := []string{"--ask-for-approval", "never", "--strict-config"}
	args = append(args, codexPermissionProfileArgs(e.codexHome)...)
	args = append(args, "exec", "--ignore-user-config", "--ephemeral", "--json")
	if strings.TrimSpace(e.cfg.CodexModel) != "" && e.cfg.CodexModel != "default" {
		args = append(args, "--model", e.cfg.CodexModel)
	}
	args = append(args, "-")
	return e.command(
		ctx,
		worktree,
		prompt,
		codexEnvironment,
		e.cfg.CodexCommand,
		args...,
	)
}

func codexPermissionProfileArgs(codexHome string) []string {
	profile := strconv.Quote(codexRunnerPermissionProfile)
	deniedPath := strconv.Quote(filepath.Clean(codexHome))
	return []string{
		"-c", "default_permissions=" + profile,
		"-c", "permissions." + codexRunnerPermissionProfile +
			".extends=" + strconv.Quote(":workspace"),
		"-c", "permissions." + codexRunnerPermissionProfile +
			".filesystem." + deniedPath + "=" + strconv.Quote("deny"),
		"-c", "permissions." + codexRunnerPermissionProfile +
			".network.enabled=false",
	}
}

func (e *LocalExecutor) verifyCodexCredentialIsolation(
	ctx context.Context,
	worktree string,
	environment []string,
) error {
	args := []string{"--strict-config"}
	args = append(args, codexPermissionProfileArgs(e.codexHome)...)
	sandboxOS := "linux"
	if e.cfg.ExecutionProfile == ExecutionProfileCodexDelegated {
		sandboxOS = codexSandboxTarget(
			valueOr(e.runtime.OS, CurrentRunnerRuntime().OS),
		)
	}
	canaryCommand, canaryArgs := codexCredentialCanaryCommand(
		sandboxOS,
		e.codexHome,
	)
	args = append(
		args,
		"sandbox", sandboxOS,
		"--permissions-profile", codexRunnerPermissionProfile,
		"--",
		canaryCommand,
	)
	args = append(args, canaryArgs...)
	result, err := e.command(
		ctx,
		worktree,
		"",
		environment,
		e.cfg.CodexCommand,
		args...,
	)
	if err != nil {
		return fmt.Errorf(
			"Codex credential isolation canary failed closed: %s",
			commandFailure(err, result),
		)
	}
	return nil
}

func (e *LocalExecutor) git(
	ctx context.Context,
	repository string,
	args ...string,
) (localCommandResult, error) {
	hardened := []string{
		"--no-pager",
		"-c", "core.hooksPath=/dev/null",
		"-c", "core.fsmonitor=false",
		"-c", "credential.helper=",
		"-c", "protocol.file.allow=never",
		"-c", "submodule.recurse=false",
		"-C", repository,
	}
	return e.commandWithAllowedEnvironment(
		ctx,
		repository,
		"",
		nil,
		[]string{
			"HOME=/nonexistent",
			"PATH=" + e.executionPath(),
			"GIT_CONFIG_NOSYSTEM=1",
			"GIT_CONFIG_GLOBAL=/dev/null",
			"GIT_ATTR_NOSYSTEM=1",
			"GIT_TERMINAL_PROMPT=0",
			"GIT_ASKPASS=/bin/false",
			"SSH_ASKPASS=/bin/false",
		},
		e.gitCommand,
		append(hardened, args...)...,
	)
}

func (e *LocalExecutor) command(
	ctx context.Context,
	directory, stdin string,
	extraEnvironment []string,
	name string,
	args ...string,
) (localCommandResult, error) {
	return e.commandWithAllowedEnvironment(
		ctx,
		directory,
		stdin,
		e.cfg.AllowedEnvironment,
		extraEnvironment,
		name,
		args...,
	)
}

func (e *LocalExecutor) verificationCommand(
	ctx context.Context,
	directory string,
	verificationHome string,
	name string,
	args ...string,
) (localCommandResult, error) {
	cacheHome := filepath.Join(verificationHome, ".cache")
	configHome := filepath.Join(verificationHome, ".config")
	runtimeHome := filepath.Join(verificationHome, ".runtime")
	codexHome := filepath.Join(verificationHome, ".codex")
	tempDir := filepath.Join(verificationHome, ".tmp")
	for _, path := range []string{
		cacheHome,
		configHome,
		runtimeHome,
		codexHome,
		tempDir,
	} {
		if err := ensureDir(path, 0o700); err != nil {
			return localCommandResult{ExitCode: -1}, err
		}
	}
	commandName := name
	commandArgs := append([]string(nil), args...)
	if len(e.cfg.VerificationSandbox) > 0 {
		commandName = e.cfg.VerificationSandbox[0]
		commandArgs = append(
			append(
				append(
					[]string(nil),
					e.cfg.VerificationSandbox[1:]...,
				),
				directory,
				verificationHome,
				"--",
				name,
			),
			args...,
		)
	}
	return e.commandWithAllowedEnvironment(
		ctx,
		directory,
		"",
		nil,
		[]string{
			"HOME=" + verificationHome,
			"CODEX_HOME=" + codexHome,
			"XDG_CACHE_HOME=" + cacheHome,
			"XDG_CONFIG_HOME=" + configHome,
			"XDG_RUNTIME_DIR=" + runtimeHome,
			"TMPDIR=" + tempDir,
			"TMP=" + tempDir,
			"TEMP=" + tempDir,
			"PATH=" + e.executionPath(),
			"GIT_TERMINAL_PROMPT=0",
		},
		commandName,
		commandArgs...,
	)
}

func (e *LocalExecutor) commandWithAllowedEnvironment(
	ctx context.Context,
	directory, stdin string,
	allowedSensitive []string,
	extraEnvironment []string,
	name string,
	args ...string,
) (localCommandResult, error) {
	commandCtx, cancel := context.WithTimeout(
		ctx,
		time.Duration(e.cfg.TimeoutSeconds)*time.Second,
	)
	defer cancel()
	started := time.Now()
	resolvedName, err := e.resolveCommand(directory, name)
	if err != nil {
		return localCommandResult{ExitCode: -1}, err
	}
	command := exec.CommandContext(commandCtx, resolvedName, args...)
	prepareLocalCommand(command)
	command.Dir = directory
	command.Env = localEnvironmentOverrides(
		sanitizedLocalEnvironment(os.Environ(), allowedSensitive),
		extraEnvironment,
	)
	if stdin != "" {
		command.Stdin = strings.NewReader(stdin)
	}
	stdout := &localLimitedBuffer{limit: maxLocalCommandOutputBytes}
	stderr := &localLimitedBuffer{limit: maxLocalCommandOutputBytes}
	command.Stdout = stdout
	command.Stderr = stderr
	err = command.Run()
	result := localCommandResult{
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
		ExitCode:   0,
		DurationMS: time.Since(started).Milliseconds(),
		TimedOut:   errors.Is(commandCtx.Err(), context.DeadlineExceeded),
		Truncated:  stdout.truncated || stderr.truncated,
	}
	if err == nil {
		return result, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.ExitCode = exitErr.ExitCode()
	} else {
		result.ExitCode = -1
	}
	if result.TimedOut {
		return result, context.DeadlineExceeded
	}
	return result, err
}

func (e *LocalExecutor) executionPath() string {
	if e.cfg.ExecutionProfile == ExecutionProfileCodexDelegated {
		return os.Getenv("PATH")
	}
	return runnerSafePath
}

func (e *LocalExecutor) resolveCommand(directory, name string) (string, error) {
	name = strings.TrimSpace(name)
	if e.cfg.ExecutionProfile != ExecutionProfileCodexDelegated ||
		filepath.IsAbs(name) ||
		strings.ContainsRune(name, filepath.Separator) {
		return resolveLocalCommand(directory, name)
	}
	return resolveConfiguredCommand(name)
}

func codexCredentialCanaryCommand(
	targetOS, codexHome string,
) (string, []string) {
	if targetOS == "windows" {
		return "powershell.exe", []string{
			"-NoLogo",
			"-NoProfile",
			"-NonInteractive",
			"-Command",
			`$ErrorActionPreference='Stop'; try { Get-ChildItem -LiteralPath $args[0] -Force | Out-Null; exit 97 } catch { exit 0 }`,
			codexHome,
		}
	}
	return "/bin/sh", []string{
		"-c",
		`if command ls -la -- "$1" >/dev/null 2>&1; then exit 97; fi
if command head -c 1 -- "$1/auth.json" >/dev/null 2>&1; then exit 98; fi
exit 0`,
		"goclaw-codex-home-canary",
		codexHome,
	}
}

func codexSandboxTarget(goos string) string {
	if strings.EqualFold(strings.TrimSpace(goos), "darwin") {
		return "macos"
	}
	return strings.ToLower(strings.TrimSpace(goos))
}

func localEnvironmentOverrides(
	environment []string,
	overrides []string,
) []string {
	names := make(map[string]struct{}, len(overrides))
	for _, entry := range overrides {
		name, _, found := strings.Cut(entry, "=")
		if found {
			names[strings.ToUpper(strings.TrimSpace(name))] = struct{}{}
		}
	}
	result := make([]string, 0, len(environment)+len(overrides))
	for _, entry := range environment {
		name, _, found := strings.Cut(entry, "=")
		if !found {
			continue
		}
		if _, overridden := names[strings.ToUpper(strings.TrimSpace(name))]; overridden {
			continue
		}
		result = append(result, entry)
	}
	return append(result, overrides...)
}

func (e *LocalExecutor) collectChanges(
	ctx context.Context,
	worktree, baseCommit string,
) ([]string, string, error) {
	tracked, err := e.git(ctx, worktree, "diff", "--name-only", baseCommit, "--")
	if err != nil {
		return nil, "", fmt.Errorf("list tracked changes: %s", commandFailure(err, tracked))
	}
	untracked, err := e.git(ctx, worktree, "ls-files", "--others", "--exclude-standard")
	if err != nil {
		return nil, "", fmt.Errorf("list untracked changes: %s", commandFailure(err, untracked))
	}
	seen := make(map[string]struct{})
	var changed []string
	var untrackedPaths []string
	for _, content := range []struct {
		value     string
		untracked bool
	}{
		{tracked.Stdout, false},
		{untracked.Stdout, true},
	} {
		for _, line := range strings.Split(content.value, "\n") {
			line = filepath.ToSlash(strings.TrimSpace(line))
			if line == "" {
				continue
			}
			if _, found := seen[line]; !found {
				seen[line] = struct{}{}
				changed = append(changed, line)
			}
			if content.untracked {
				untrackedPaths = append(untrackedPaths, line)
			}
		}
	}
	sort.Strings(changed)
	if len(untrackedPaths) > 0 {
		args := append([]string{"add", "-N", "--"}, untrackedPaths...)
		result, err := e.git(ctx, worktree, args...)
		if err != nil {
			return changed, "", fmt.Errorf("mark untracked changes for diff: %s", commandFailure(err, result))
		}
	}
	diff, err := e.git(ctx, worktree, "diff", "--binary", "--no-ext-diff", baseCommit, "--")
	if err != nil {
		return changed, "", fmt.Errorf("collect diff: %s", commandFailure(err, diff))
	}
	return changed, diff.Stdout, nil
}

func localScopeViolations(worktree string, changed []string, pack ExecutionPack) []string {
	var violations []string
	for _, path := range changed {
		clean := filepath.ToSlash(filepath.Clean(path))
		if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || filepath.IsAbs(path) {
			violations = append(violations, "invalid changed path: "+path)
			continue
		}
		if localMatchesAnyPath(clean, pack.DeniedPaths) {
			violations = append(violations, "denied path changed: "+clean)
		}
		if len(pack.AllowedPaths) > 0 && !localMatchesAnyPath(clean, pack.AllowedPaths) {
			violations = append(violations, "path outside approved scope: "+clean)
		}
		if err := localValidateSymlinkBoundary(worktree, clean); err != nil {
			violations = append(violations, err.Error())
		}
	}
	sort.Strings(violations)
	return violations
}

func localMatchesAnyPath(path string, patterns []string) bool {
	path = filepath.ToSlash(path)
	for _, pattern := range patterns {
		pattern = filepath.ToSlash(strings.TrimSpace(pattern))
		if pattern == "" {
			continue
		}
		if matched, _ := filepath.Match(pattern, path); matched {
			return true
		}
		if strings.HasSuffix(pattern, "/**") {
			prefix := strings.TrimSuffix(pattern, "**")
			if strings.HasPrefix(path, prefix) {
				return true
			}
		}
		if !strings.ContainsAny(pattern, "*?[") {
			prefix := strings.TrimSuffix(pattern, "/")
			if path == prefix || strings.HasPrefix(path, prefix+"/") {
				return true
			}
		}
	}
	return false
}

func localValidateSymlinkBoundary(worktree, relative string) error {
	root, err := filepath.EvalSymlinks(worktree)
	if err != nil {
		return fmt.Errorf("resolve worktree boundary: %w", err)
	}
	current := worktree
	for _, part := range strings.Split(filepath.FromSlash(relative), string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			break
		}
		if err != nil {
			return fmt.Errorf("inspect changed path %s: %w", relative, err)
		}
		if info.Mode()&os.ModeSymlink == 0 {
			continue
		}
		resolved, err := filepath.EvalSymlinks(current)
		if err != nil {
			return fmt.Errorf("resolve changed symlink %s: %w", relative, err)
		}
		if resolved != root && !strings.HasPrefix(resolved, root+string(filepath.Separator)) {
			return fmt.Errorf("changed path crosses worktree through symlink: %s", relative)
		}
	}
	return nil
}

func localBranch(prefix, taskID string, revision, attempt int) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		prefix = "goclaw/" + strings.TrimPrefix(taskID, "wstask-")
	}
	prefix = strings.TrimSuffix(prefix, "/")
	return fmt.Sprintf("%s/r%d-a%d", prefix, revision, attempt)
}

func localEvidenceURI(runnerID, taskID, leaseID, name string) string {
	return fmt.Sprintf(
		"workstation://%s/tasks/%s/leases/%s/%s",
		runnerID,
		taskID,
		leaseID,
		name,
	)
}

func sanitizedLocalEnvironment(
	environment []string,
	allowedSensitive []string,
) []string {
	allowed := make(map[string]struct{}, len(allowedSensitive))
	for _, name := range allowedSensitive {
		allowed[strings.ToUpper(strings.TrimSpace(name))] = struct{}{}
	}
	result := make([]string, 0, len(environment))
	for _, entry := range environment {
		name, _, found := strings.Cut(entry, "=")
		if !found {
			continue
		}
		upper := strings.ToUpper(strings.TrimSpace(name))
		if upper == "" ||
			protectedControlPlaneEnvironment(upper) ||
			forbiddenHostCapabilityEnvironment(upper) {
			continue
		}
		_, explicitlyAllowed := allowed[upper]
		if !safeLocalEnvironmentName(upper) && !explicitlyAllowed {
			continue
		}
		if sensitiveEnvironmentName(upper) && !explicitlyAllowed {
			continue
		}
		result = append(result, entry)
	}
	return result
}

func safeLocalEnvironmentName(name string) bool {
	switch name {
	case "PATH", "PATHEXT", "SYSTEMROOT", "WINDIR", "COMSPEC",
		"TMPDIR", "TMP", "TEMP", "LANG", "LANGUAGE", "TERM",
		"COLORTERM", "TZ", "SSL_CERT_FILE", "SSL_CERT_DIR",
		"REQUESTS_CA_BUNDLE", "CURL_CA_BUNDLE", "HTTP_PROXY",
		"HTTPS_PROXY", "ALL_PROXY", "NO_PROXY", "GOPATH",
		"GOMODCACHE", "GOCACHE", "GOROOT", "GOENV", "GOPROXY",
		"GONOSUMDB", "GOPRIVATE":
		return true
	default:
		return strings.HasPrefix(name, "LC_")
	}
}

func forbiddenHostCapabilityEnvironment(name string) bool {
	switch name {
	case "SSH_AUTH_SOCK", "SSH_AGENT_PID", "GIT_ASKPASS",
		"SSH_ASKPASS", "GIT_SSH", "GIT_SSH_COMMAND", "DOCKER_HOST",
		"DOCKER_CONTEXT", "KUBECONFIG", "GOOGLE_APPLICATION_CREDENTIALS",
		"AWS_CONFIG_FILE", "AWS_SHARED_CREDENTIALS_FILE",
		"AZURE_CONFIG_DIR", "CLOUDSDK_CONFIG", "KRB5CCNAME",
		"LD_PRELOAD", "LD_LIBRARY_PATH", "NODE_OPTIONS", "BUN_OPTIONS",
		"BASH_ENV", "ENV", "SHELLOPTS", "PS4", "PYTHONPATH",
		"PYTHONHOME", "PYTHONSTARTUP", "RUBYOPT", "PERL5OPT",
		"JAVA_TOOL_OPTIONS", "_JAVA_OPTIONS", "JDK_JAVA_OPTIONS",
		"RUSTC_WRAPPER", "GIT_CONFIG_COUNT":
		return true
	default:
		return strings.Contains(name, "DOCKER_SOCKET") ||
			strings.Contains(name, "CONTAINER_HOST") ||
			strings.HasPrefix(name, "VAULT_") ||
			strings.HasPrefix(name, "DYLD_") ||
			strings.HasPrefix(name, "GIT_CONFIG_KEY_") ||
			strings.HasPrefix(name, "GIT_CONFIG_VALUE_")
	}
}

func resolveLocalCodexHome() (string, error) {
	value := strings.TrimSpace(os.Getenv("CODEX_HOME"))
	if value == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve Codex OAuth home: %w", err)
		}
		value = filepath.Join(home, ".codex")
	}
	if !filepath.IsAbs(value) {
		return "", errors.New("CODEX_HOME must be absolute")
	}
	return filepath.Clean(value), nil
}

func protectedControlPlaneEnvironment(name string) bool {
	return name == "GOCLAW_USER_TOKEN" ||
		name == "GOCLAW_GATEWAY_TOKEN" ||
		name == "GOCLAW_REVIEWER_TOKEN" ||
		name == "CODEX_ACCESS_TOKEN" ||
		name == "CODEX_REFRESH_TOKEN" ||
		strings.HasPrefix(name, "GOSKILLS_GATEWAY_") ||
		strings.HasPrefix(name, "GOCLAW_RUNNER_")
}

func sensitiveEnvironmentName(name string) bool {
	for _, marker := range []string{
		"TOKEN",
		"SECRET",
		"PASSWORD",
		"PASSWD",
		"CREDENTIAL",
		"PRIVATE_KEY",
		"API_KEY",
		"AUTH",
	} {
		if strings.Contains(name, marker) {
			return true
		}
	}
	return false
}

func commandFailure(err error, result localCommandResult) string {
	if err == nil {
		return strings.TrimSpace(result.Stderr)
	}
	detail := strings.TrimSpace(result.Stderr)
	if detail == "" {
		detail = strings.TrimSpace(result.Stdout)
	}
	if detail == "" {
		return err.Error()
	}
	return err.Error() + ": " + truncateLocalOutput(detail, 16*1024)
}

func truncateLocalOutput(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return value[:limit] + "\n[truncated]"
}

type localCommandResult struct {
	Stdout     string
	Stderr     string
	ExitCode   int
	DurationMS int64
	TimedOut   bool
	Truncated  bool
}

type localLimitedBuffer struct {
	buffer    bytes.Buffer
	limit     int
	truncated bool
}

func (b *localLimitedBuffer) Write(data []byte) (int, error) {
	original := len(data)
	remaining := b.limit - b.buffer.Len()
	if remaining <= 0 {
		b.truncated = true
		return original, nil
	}
	if len(data) > remaining {
		data = data[:remaining]
		b.truncated = true
	}
	_, _ = b.buffer.Write(data)
	return original, nil
}

func (b *localLimitedBuffer) String() string {
	return b.buffer.String()
}
