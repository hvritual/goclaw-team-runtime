package cli

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/smallnest/goclaw/config"
	"github.com/smallnest/goclaw/teamcontrol"
	"github.com/smallnest/goclaw/workstation"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newRunnerCommand())
}

func newRunnerCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "runner",
		Short: "Provision and run a project-scoped local Codex workstation",
	}
	command.AddCommand(
		newRunnerDoctorCommand(),
		newRunnerRegisterCommand(),
		newRunnerUpdateCommand(),
		newRunnerRotateKeyCommand(),
		newRunnerWorkCommand(),
		newRunnerListCommand(),
		newRunnerCancelCommand(),
		newRunnerEvidenceCommand(),
		newRunnerPatchCommand(),
		newRunnerReleaseCommand(),
	)
	return command
}

func newRunnerUpdateCommand() *cobra.Command {
	var id, name string
	var projects, capabilities []string
	var disable, enable bool
	command := &cobra.Command{
		Use:   "update",
		Short: "Update or disable an idle workstation runner",
		RunE: func(cmd *cobra.Command, args []string) error {
			params := map[string]any{"runner_id": id}
			if cmd.Flags().Changed("name") {
				params["name"] = name
			}
			if cmd.Flags().Changed("project") {
				params["projects"] = projects
			}
			if cmd.Flags().Changed("capability") {
				params["capabilities"] = capabilities
			}
			if disable {
				params["disabled"] = true
			}
			if enable {
				params["disabled"] = false
			}
			if len(params) == 1 {
				return errors.New("at least one update flag is required")
			}
			result, err := callRunnerRPCWithRetry("runner.update", params)
			if err != nil {
				return err
			}
			return printTeamValue(result)
		},
	}
	command.Flags().StringVar(&id, "id", "", "Registered workstation id")
	command.Flags().StringVar(&name, "name", "", "New workstation display name")
	command.Flags().StringSliceVar(&projects, "project", nil, "Replace authorized project ids")
	command.Flags().StringSliceVar(&capabilities, "capability", nil, "Replace runner capabilities")
	command.Flags().BoolVar(&disable, "disable", false, "Disable claims after the current runner is idle")
	command.Flags().BoolVar(&enable, "enable", false, "Re-enable a disabled runner")
	command.MarkFlagsMutuallyExclusive("disable", "enable")
	_ = command.MarkFlagRequired("id")
	return command
}

func newRunnerCancelCommand() *cobra.Command {
	var reason, idempotencyKey string
	command := &cobra.Command{
		Use:   "cancel TASK_ID",
		Short: "Cancel queued or failed work before revising its development task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(idempotencyKey) == "" {
				idempotencyKey = runnerOperationKey("cancel-" + args[0])
			}
			result, err := callRunnerRPCWithRetry("runner.cancel", map[string]any{
				"task_id":         args[0],
				"reason":          reason,
				"idempotency_key": idempotencyKey,
			})
			if err != nil {
				return err
			}
			return printTeamValue(result)
		},
	}
	command.Flags().StringVar(
		&reason,
		"reason",
		"",
		"Why this queued execution is being cancelled",
	)
	command.Flags().StringVar(
		&idempotencyKey,
		"idempotency-key",
		"",
		"Stable retry key; generated when omitted",
	)
	_ = command.MarkFlagRequired("reason")
	return command
}

func newRunnerRegisterCommand() *cobra.Command {
	var id, name, keyFile, verificationSandbox, executionProfile string
	var projects, capabilities []string
	var reuseKey bool
	command := &cobra.Command{
		Use:   "register",
		Short: "Register this workstation and save its device key once",
		RunE: func(cmd *cobra.Command, args []string) error {
			profile, err := workstation.NormalizeExecutionProfile(executionProfile)
			if err != nil {
				return err
			}
			metadata := workstation.RunnerRegistrationMetadataForProfile(profile)
			metadata["current_version"] = Version
			metadata["current_release_id"] = ""
			metadata["target_version"] = ""
			metadata["target_release_id"] = ""
			metadata["release_protocol"] = workstation.RunnerReleaseProtocol
			metadata["rollout_state"] = "unmanaged"
			if strings.TrimSpace(verificationSandbox) != "" {
				if profile != workstation.ExecutionProfileStrict {
					return errors.New(
						"--verification-sandbox requires --execution-profile strict",
					)
				}
				metadata, err = workstation.RunnerRegistrationMetadataForSandbox(
					verificationSandbox,
				)
				if err != nil {
					return fmt.Errorf(
						"verification sandbox metadata: %w",
						err,
					)
				}
				metadata["execution_profile"] = string(profile)
				metadata["directory_boundary"] = "goclaw-worktree-v1"
				metadata["network_isolation"] = "required"
				metadata["security_posture"] = "strict"
			}
			capabilities = workstation.RunnerRegistrationCapabilitiesForProfile(
				capabilities,
				profile,
			)
			versionCapability, err := workstation.RunnerVersionCapability(Version)
			if err != nil {
				return err
			}
			capabilities = append(
				capabilities,
				versionCapability,
			)
			var key []byte
			if reuseKey {
				key, err = os.ReadFile(keyFile)
				if err != nil {
					return fmt.Errorf("read existing device key: %w", err)
				}
				if _, err := workstation.DeviceKeyID(key); err != nil {
					return fmt.Errorf("validate existing device key: %w", err)
				}
			} else {
				key, err = workstation.GenerateDeviceKey()
				if err != nil {
					return err
				}
				if _, err := stageBinarySecretFile(keyFile, key); err != nil {
					return err
				}
			}
			result, err := callRunnerRPCWithRetry("runner.register", map[string]any{
				"id":           id,
				"name":         name,
				"projects":     projects,
				"capabilities": capabilities,
				"metadata":     metadata,
				"device_key":   base64.RawURLEncoding.EncodeToString(key),
			})
			if err != nil {
				return fmt.Errorf(
					"register runner (device key retained at %s; retry with --reuse-key): %w",
					keyFile,
					err,
				)
			}
			return printTeamValue(map[string]any{
				"runner":          result,
				"device_key_file": keyFile,
				"next":            "goclaw runner work --id " + id + " --key-file " + keyFile + " --work-root /local/goclaw-worktrees --repo REPOSITORY_ID=/local/checkout",
			})
		},
	}
	command.Flags().StringVar(&id, "id", "", "Stable workstation id")
	command.Flags().StringVar(&name, "name", "", "Workstation display name")
	command.Flags().StringVar(
		&executionProfile,
		"execution-profile",
		string(workstation.ExecutionProfileStrict),
		"Execution profile: strict or codex-delegated",
	)
	command.Flags().StringVar(&keyFile, "key-file", "", "New 0600 file for the raw device key")
	command.Flags().BoolVar(&reuseKey, "reuse-key", false, "Reuse an existing key after an ambiguous registration failure")
	command.Flags().StringSliceVar(&projects, "project", nil, "Authorized project id; repeat as needed")
	command.Flags().StringSliceVar(&capabilities, "capability", []string{"codex"}, "Runner capability")
	command.Flags().StringVar(
		&verificationSandbox,
		"verification-sandbox",
		"",
		"Reviewed absolute bwrap wrapper recorded in runner metadata",
	)
	_ = command.MarkFlagRequired("id")
	_ = command.MarkFlagRequired("name")
	_ = command.MarkFlagRequired("key-file")
	_ = command.MarkFlagRequired("project")
	return command
}

func newRunnerRotateKeyCommand() *cobra.Command {
	var id, keyFile string
	var reuseKey bool
	command := &cobra.Command{
		Use:   "rotate-key",
		Short: "Rotate an idle workstation's evidence-signing device key",
		RunE: func(cmd *cobra.Command, args []string) error {
			var key []byte
			var err error
			if reuseKey {
				key, err = os.ReadFile(keyFile)
				if err != nil {
					return err
				}
				if _, err := workstation.DeviceKeyID(key); err != nil {
					return err
				}
			} else {
				key, err = workstation.GenerateDeviceKey()
				if err != nil {
					return err
				}
				if _, err := stageBinarySecretFile(keyFile, key); err != nil {
					return err
				}
			}
			result, err := callRunnerRPCWithRetry("runner.key.rotate", map[string]any{
				"runner_id":  id,
				"device_key": base64.RawURLEncoding.EncodeToString(key),
			})
			if err != nil {
				return fmt.Errorf(
					"rotate runner key (candidate retained at %s for safe retry): %w",
					keyFile,
					err,
				)
			}
			return printTeamValue(map[string]any{
				"runner":              result,
				"new_device_key_file": keyFile,
				"next":                "stop the old runner and update GOCLAW_RUNNER_KEY_FILE before restarting",
			})
		},
	}
	command.Flags().StringVar(&id, "id", "", "Registered workstation id")
	command.Flags().StringVar(&keyFile, "new-key-file", "", "New 0600 file for the rotated raw device key")
	command.Flags().BoolVar(&reuseKey, "reuse-key", false, "Reuse the candidate key after an ambiguous rotation failure")
	_ = command.MarkFlagRequired("id")
	_ = command.MarkFlagRequired("new-key-file")
	return command
}

func newRunnerListCommand() *cobra.Command {
	var projectID string
	command := &cobra.Command{
		Use:   "list",
		Short: "List workstations visible in a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := callRunnerRPCWithRetry("runner.list", map[string]any{
				"project_id": projectID,
			})
			if err != nil {
				return err
			}
			return printTeamValue(result)
		},
	}
	command.Flags().StringVar(&projectID, "project", "", "Project id")
	_ = command.MarkFlagRequired("project")
	return command
}

func newRunnerEvidenceCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "evidence TASK_ID",
		Short: "Read the verified signed evidence for a workstation task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := callRunnerRPCWithRetry("runner.evidence", map[string]any{
				"task_id": args[0],
			})
			if err != nil {
				return err
			}
			return printTeamValue(result)
		},
	}
}

func newRunnerPatchCommand() *cobra.Command {
	var output string
	command := &cobra.Command{
		Use:   "patch TASK_ID",
		Short: "Download and verify a recoverable patch from signed evidence",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := callRunnerRPCWithRetry("runner.evidence", map[string]any{
				"task_id": args[0],
			})
			if err != nil {
				return err
			}
			var evidence workstation.EvidenceBundle
			if err := remarshal(result, &evidence); err != nil {
				return err
			}
			if evidence.DiffPatch == "" || evidence.DiffSHA256 == "" {
				return errors.New("signed evidence contains no recoverable patch")
			}
			digest := sha256.Sum256([]byte(evidence.DiffPatch))
			actual := hex.EncodeToString(digest[:])
			if actual != evidence.DiffSHA256 {
				return fmt.Errorf(
					"patch SHA-256 mismatch: got %s want %s",
					actual,
					evidence.DiffSHA256,
				)
			}
			absolute, err := filepath.Abs(output)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(absolute), 0o755); err != nil {
				return err
			}
			file, err := os.OpenFile(
				absolute,
				os.O_CREATE|os.O_EXCL|os.O_WRONLY,
				0o644,
			)
			if err != nil {
				return err
			}
			if _, err := file.WriteString(evidence.DiffPatch); err != nil {
				_ = file.Close()
				_ = os.Remove(absolute)
				return err
			}
			if err := file.Close(); err != nil {
				_ = os.Remove(absolute)
				return err
			}
			return printTeamValue(map[string]any{
				"task_id":       args[0],
				"patch_file":    absolute,
				"diff_sha256":   actual,
				"bundle_sha256": evidence.BundleSHA256,
			})
		},
	}
	command.Flags().StringVar(&output, "output", "", "New output patch file")
	_ = command.MarkFlagRequired("output")
	return command
}

func newRunnerWorkCommand() *cobra.Command {
	var id, keyFile, workRoot, projectID, codexCommand, codexModel, executionProfile string
	var repositoryMappings []string
	var allowedEnvironment []string
	var verificationSandbox []string
	var timeoutSeconds int
	var heartbeat, poll time.Duration
	var once, unsafeHostVerification bool
	command := &cobra.Command{
		Use:   "work",
		Short: "Claim queued work and execute it with this computer's Codex login",
		RunE: func(cmd *cobra.Command, args []string) error {
			profile, err := workstation.NormalizeExecutionProfile(executionProfile)
			if err != nil {
				return err
			}
			repositories, err := parseRepositoryMappings(repositoryMappings)
			if err != nil {
				return err
			}
			doctor := workstation.RunRunnerDoctor(
				cmd.Context(),
				workstation.RunnerDoctorConfig{
					ExecutionProfile:       profile,
					DeviceKeyPath:          keyFile,
					WorkRoot:               workRoot,
					RepositoryPaths:        repositories,
					CodexCommand:           codexCommand,
					VerificationSandbox:    verificationSandbox,
					UnsafeHostVerification: unsafeHostVerification,
				},
			)
			if !doctor.Ready {
				return errors.New(
					"runner preflight failed; run `goclaw runner doctor` with the same local flags",
				)
			}
			releaseProcessLock, err := workstation.AcquireRunnerProcessLock(
				workRoot,
			)
			if err != nil {
				return err
			}
			defer func() { _ = releaseProcessLock() }()
			executor, err := workstation.NewLocalExecutor(workstation.LocalExecConfig{
				RunnerID:               id,
				ExecutionProfile:       profile,
				DeviceKeyPath:          keyFile,
				WorkRoot:               workRoot,
				RepositoryPaths:        repositories,
				CodexCommand:           codexCommand,
				CodexModel:             codexModel,
				TimeoutSeconds:         timeoutSeconds,
				AllowedEnvironment:     allowedEnvironment,
				VerificationSandbox:    verificationSandbox,
				UnsafeHostVerification: unsafeHostVerification,
			})
			if err != nil {
				return err
			}
			releaseManager, err := workstation.NewReleaseManager(
				workstation.ReleaseManagerConfig{WorkRoot: workRoot},
			)
			if err != nil {
				return err
			}
			lifecycle, err := releaseManager.LifecycleProjection(
				Version,
				profile,
			)
			if err != nil {
				return err
			}
			if heartbeat < 5*time.Second {
				return errors.New("--heartbeat must be at least 5s")
			}
			if poll < time.Second {
				return errors.New("--poll must be at least 1s")
			}
			return runWorkstationLoop(
				cmd.Context(),
				executor,
				lifecycle,
				projectID,
				heartbeat,
				poll,
				once,
			)
		},
	}
	command.Flags().StringVar(&id, "id", "", "Registered workstation id")
	command.Flags().StringVar(&keyFile, "key-file", "", "Raw device key file created by runner register")
	command.Flags().StringVar(&workRoot, "work-root", "", "Local root for isolated task worktrees and evidence")
	command.Flags().StringSliceVar(&repositoryMappings, "repo", nil, "Repository mapping ID=/absolute/local/path")
	command.Flags().StringVar(&projectID, "project", "", "Project queue to claim from")
	command.Flags().StringVar(
		&executionProfile,
		"execution-profile",
		string(workstation.ExecutionProfileStrict),
		"Execution profile: strict or codex-delegated",
	)
	command.Flags().StringVar(&codexCommand, "codex-command", "codex", "Local Codex CLI command")
	command.Flags().StringVar(&codexModel, "codex-model", "default", "Codex subscription model; default uses the local account default")
	command.Flags().StringSliceVar(&allowedEnvironment, "allow-env", nil, "Explicitly pass a non-GoClaw sensitive environment variable to Codex; frozen verification remains isolated")
	command.Flags().StringSliceVar(
		&verificationSandbox,
		"verification-sandbox",
		nil,
		"Verification wrapper argv prefix; it receives WORKTREE HOME -- COMMAND ARG...",
	)
	command.Flags().BoolVar(
		&unsafeHostVerification,
		"unsafe-host-verification",
		false,
		"Run frozen verification directly on the host (only inside an already isolated disposable VM/container)",
	)
	command.Flags().IntVar(&timeoutSeconds, "timeout", 21600, "Maximum seconds for Codex and each verification command")
	command.Flags().DurationVar(&heartbeat, "heartbeat", 30*time.Second, "Task lease heartbeat interval")
	command.Flags().DurationVar(&poll, "poll", 5*time.Second, "Idle queue polling interval")
	command.Flags().BoolVar(&once, "once", false, "Process at most one task, or return immediately if idle")
	command.MarkFlagsMutuallyExclusive(
		"verification-sandbox",
		"unsafe-host-verification",
	)
	_ = command.MarkFlagRequired("id")
	_ = command.MarkFlagRequired("key-file")
	_ = command.MarkFlagRequired("work-root")
	_ = command.MarkFlagRequired("repo")
	_ = command.MarkFlagRequired("project")
	return command
}

func newRunnerReleaseCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "release",
		Short: "Stage, select, confirm, or roll back a local runner release",
	}
	command.AddCommand(
		newRunnerReleaseStageCommand(),
		newRunnerReleaseStageFromControlCommand(),
		newRunnerReleaseStatusCommand(),
		newRunnerReleaseActivateCommand(),
		newRunnerReleaseConfirmCommand(),
		newRunnerReleaseRollbackCommand(),
		newRunnerReleasePathCommand(),
	)
	return command
}

func newRunnerReleaseStageFromControlCommand() *cobra.Command {
	var workRoot, projectID, artifact string
	command := &cobra.Command{
		Use:   "stage-from-control RELEASE_ID",
		Short: "Stage a local artifact against an approved Team Control release",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			value, err := callRunnerRPCWithRetry(
				"runner.release.get",
				map[string]any{
					"project_id":        projectID,
					"runner_release_id": args[0],
				},
			)
			if err != nil {
				return err
			}
			data, err := json.Marshal(value)
			if err != nil {
				return err
			}
			var release teamcontrol.RunnerRelease
			if err := json.Unmarshal(data, &release); err != nil {
				return fmt.Errorf("decode Team Control runner release: %w", err)
			}
			if release.ID != args[0] || release.ProjectID != projectID {
				return errors.New("Team Control runner release identity mismatch")
			}
			if release.Status != teamcontrol.RegistryApproved {
				return errors.New("Team Control runner release is not approved")
			}
			if release.SizeBytes <= 0 {
				return errors.New(
					"legacy runner release has no size_bytes and cannot be staged",
				)
			}
			manager, err := workstation.NewReleaseManager(
				workstation.ReleaseManagerConfig{WorkRoot: workRoot},
			)
			if err != nil {
				return err
			}
			record, err := manager.StageLocal(workstation.LocalReleaseInput{
				ReleaseID:  release.ID,
				Version:    release.Version,
				OS:         release.OS,
				Arch:       release.Arch,
				Protocol:   release.MinProtocol,
				SourcePath: artifact,
				SHA256:     release.SHA256,
				SizeBytes:  release.SizeBytes,
			})
			if err != nil {
				return err
			}
			return printTeamValue(map[string]any{
				"release": record,
				"source":  "team-control",
			})
		},
	}
	command.Flags().StringVar(&workRoot, "work-root", "", "Runner work root")
	command.Flags().StringVar(&projectID, "project", "", "Team Control project id")
	command.Flags().StringVar(
		&artifact,
		"artifact",
		"",
		"Absolute operator-provided local artifact; registry URI is not fetched",
	)
	for _, name := range []string{"work-root", "project", "artifact"} {
		_ = command.MarkFlagRequired(name)
	}
	return command
}

func newRunnerReleaseStageCommand() *cobra.Command {
	var workRoot, id, version, targetOS, arch, protocol, artifact, checksum string
	var sizeBytes int64
	command := &cobra.Command{
		Use:   "stage",
		Short: "Verify and stage an operator-provided local release artifact",
		RunE: func(cmd *cobra.Command, args []string) error {
			manager, err := workstation.NewReleaseManager(
				workstation.ReleaseManagerConfig{
					WorkRoot: workRoot,
					OS:       targetOS,
					Arch:     arch,
					Protocol: protocol,
				},
			)
			if err != nil {
				return err
			}
			record, err := manager.StageLocal(workstation.LocalReleaseInput{
				ReleaseID:  id,
				Version:    version,
				OS:         targetOS,
				Arch:       arch,
				Protocol:   protocol,
				SourcePath: artifact,
				SHA256:     checksum,
				SizeBytes:  sizeBytes,
			})
			if err != nil {
				return err
			}
			return printTeamValue(record)
		},
	}
	command.Flags().StringVar(&workRoot, "work-root", "", "Runner work root")
	command.Flags().StringVar(&id, "id", "", "Immutable Team Control release id")
	command.Flags().StringVar(&version, "version", "", "Release version")
	command.Flags().StringVar(&targetOS, "os", "", "Release target operating system")
	command.Flags().StringVar(&arch, "arch", "", "Release target architecture")
	command.Flags().StringVar(
		&protocol,
		"protocol",
		workstation.RunnerReleaseProtocol,
		"Required runner protocol contract",
	)
	command.Flags().StringVar(
		&artifact,
		"artifact",
		"",
		"Absolute local artifact path; URLs are not accepted",
	)
	command.Flags().StringVar(&checksum, "sha256", "", "Expected SHA-256")
	command.Flags().Int64Var(&sizeBytes, "size", 0, "Expected artifact size in bytes")
	for _, name := range []string{
		"work-root", "id", "version", "os", "arch", "artifact", "sha256", "size",
	} {
		_ = command.MarkFlagRequired(name)
	}
	return command
}

func newRunnerReleaseStatusCommand() *cobra.Command {
	var workRoot string
	command := &cobra.Command{
		Use:   "status",
		Short: "Show the local atomic release selection",
		RunE: func(cmd *cobra.Command, args []string) error {
			manager, err := workstation.NewReleaseManager(
				workstation.ReleaseManagerConfig{WorkRoot: workRoot},
			)
			if err != nil {
				return err
			}
			state, err := manager.Status()
			if err != nil {
				return err
			}
			return printTeamValue(state)
		},
	}
	command.Flags().StringVar(&workRoot, "work-root", "", "Runner work root")
	_ = command.MarkFlagRequired("work-root")
	return command
}

func newRunnerReleaseActivateCommand() *cobra.Command {
	var workRoot string
	command := &cobra.Command{
		Use:   "activate RELEASE_ID",
		Short: "Atomically select a staged release while the runner is stopped",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			manager, err := workstation.NewReleaseManager(
				workstation.ReleaseManagerConfig{WorkRoot: workRoot},
			)
			if err != nil {
				return err
			}
			state, err := manager.Activate(args[0])
			if err != nil {
				return err
			}
			return printTeamValue(state)
		},
	}
	command.Flags().StringVar(&workRoot, "work-root", "", "Runner work root")
	_ = command.MarkFlagRequired("work-root")
	return command
}

func newRunnerReleaseConfirmCommand() *cobra.Command {
	var workRoot string
	command := &cobra.Command{
		Use:   "confirm RELEASE_ID",
		Short: "Confirm health of the selected release after a controlled smoke test",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			manager, err := workstation.NewReleaseManager(
				workstation.ReleaseManagerConfig{WorkRoot: workRoot},
			)
			if err != nil {
				return err
			}
			state, err := manager.Confirm(args[0])
			if err != nil {
				return err
			}
			return printTeamValue(state)
		},
	}
	command.Flags().StringVar(&workRoot, "work-root", "", "Runner work root")
	_ = command.MarkFlagRequired("work-root")
	return command
}

func newRunnerReleaseRollbackCommand() *cobra.Command {
	var workRoot string
	command := &cobra.Command{
		Use:   "rollback",
		Short: "Atomically select the previous verified local release",
		RunE: func(cmd *cobra.Command, args []string) error {
			manager, err := workstation.NewReleaseManager(
				workstation.ReleaseManagerConfig{WorkRoot: workRoot},
			)
			if err != nil {
				return err
			}
			state, err := manager.Rollback()
			if err != nil {
				return err
			}
			return printTeamValue(state)
		},
	}
	command.Flags().StringVar(&workRoot, "work-root", "", "Runner work root")
	_ = command.MarkFlagRequired("work-root")
	return command
}

func newRunnerReleasePathCommand() *cobra.Command {
	var workRoot string
	command := &cobra.Command{
		Use:   "path",
		Short: "Print the reverified selected runner binary path",
		RunE: func(cmd *cobra.Command, args []string) error {
			manager, err := workstation.NewReleaseManager(
				workstation.ReleaseManagerConfig{WorkRoot: workRoot},
			)
			if err != nil {
				return err
			}
			path, err := manager.CurrentBinary()
			if err != nil {
				return err
			}
			return printTeamValue(map[string]any{"binary_path": path})
		},
	}
	command.Flags().StringVar(&workRoot, "work-root", "", "Runner work root")
	_ = command.MarkFlagRequired("work-root")
	return command
}

func newRunnerDoctorCommand() *cobra.Command {
	var keyFile, workRoot, codexCommand, executionProfile string
	var repositoryMappings, verificationSandbox []string
	var unsafeHostVerification, jsonOutput bool
	command := &cobra.Command{
		Use:   "doctor",
		Short: "Fail-closed preflight for a Linux, WSL2, or Lima pilot runner",
		RunE: func(cmd *cobra.Command, args []string) error {
			profile, err := workstation.NormalizeExecutionProfile(executionProfile)
			if err != nil {
				return err
			}
			repositories, err := parseRepositoryMappings(repositoryMappings)
			if err != nil {
				return err
			}
			report := workstation.RunRunnerDoctor(
				cmd.Context(),
				workstation.RunnerDoctorConfig{
					ExecutionProfile:       profile,
					DeviceKeyPath:          keyFile,
					WorkRoot:               workRoot,
					RepositoryPaths:        repositories,
					CodexCommand:           codexCommand,
					VerificationSandbox:    verificationSandbox,
					UnsafeHostVerification: unsafeHostVerification,
				},
			)
			if err := printRunnerDoctor(
				cmd.OutOrStdout(),
				report,
				jsonOutput,
			); err != nil {
				return err
			}
			if !report.Ready {
				return errors.New("runner doctor found blocking checks")
			}
			return nil
		},
	}
	command.Flags().StringVar(
		&keyFile,
		"key-file",
		"",
		"Raw device key file created by runner register",
	)
	command.Flags().StringVar(
		&workRoot,
		"work-root",
		"",
		"Local root for isolated task worktrees and evidence",
	)
	command.Flags().StringSliceVar(
		&repositoryMappings,
		"repo",
		nil,
		"Repository mapping ID=/absolute/local/path",
	)
	command.Flags().StringVar(
		&codexCommand,
		"codex-command",
		"codex",
		"Local Codex CLI command",
	)
	command.Flags().StringVar(
		&executionProfile,
		"execution-profile",
		string(workstation.ExecutionProfileStrict),
		"Execution profile: strict or codex-delegated",
	)
	command.Flags().StringSliceVar(
		&verificationSandbox,
		"verification-sandbox",
		nil,
		"Verification wrapper argv prefix",
	)
	command.Flags().BoolVar(
		&unsafeHostVerification,
		"unsafe-host-verification",
		false,
		"Accept externally isolated VM verification with an explicit warning",
	)
	command.Flags().BoolVar(
		&jsonOutput,
		"json",
		false,
		"Emit the machine-readable doctor report",
	)
	command.MarkFlagsMutuallyExclusive(
		"verification-sandbox",
		"unsafe-host-verification",
	)
	_ = command.MarkFlagRequired("key-file")
	_ = command.MarkFlagRequired("work-root")
	_ = command.MarkFlagRequired("repo")
	return command
}

func printRunnerDoctor(
	writer io.Writer,
	report workstation.RunnerDoctorReport,
	jsonOutput bool,
) error {
	if jsonOutput {
		data, err := report.MarshalJSONIndent()
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(writer, string(data))
		return err
	}
	state := "READY"
	if !report.Ready {
		state = "BLOCKED"
	}
	if _, err := fmt.Fprintf(
		writer,
		"Runner doctor: %s (%s/%s, %s)\n",
		state,
		report.Runtime.OS,
		report.Runtime.Arch,
		report.Runtime.Substrate,
	); err != nil {
		return err
	}
	for _, check := range report.Checks {
		if _, err := fmt.Fprintf(
			writer,
			"[%s] %s: %s\n",
			strings.ToUpper(string(check.Status)),
			check.ID,
			check.Summary,
		); err != nil {
			return err
		}
		if strings.TrimSpace(check.Detail) != "" {
			if _, err := fmt.Fprintf(
				writer,
				"       %s\n",
				check.Detail,
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func runWorkstationLoop(
	ctx context.Context,
	executor *workstation.LocalExecutor,
	lifecycle workstation.RunnerLifecycleProjection,
	projectID string,
	heartbeatInterval, pollInterval time.Duration,
	once bool,
) error {
	runnerID := executor.Config().RunnerID
	for {
		if _, err := callRunnerRPCWithRetry("runner.ping", map[string]any{
			"runner_id":          runnerID,
			"current_version":    lifecycle.CurrentVersion,
			"current_release_id": lifecycle.CurrentReleaseID,
			"release_protocol":   lifecycle.ReleaseProtocol,
			"execution_profile":  lifecycle.ExecutionProfile,
		}); err != nil {
			return fmt.Errorf("runner presence heartbeat: %w", err)
		}
		claimKey := runnerOperationKey(runnerID + "-claim")
		result, err := callRunnerRPCWithRetry("runner.claim", map[string]any{
			"runner_id":       runnerID,
			"project_id":      projectID,
			"idempotency_key": claimKey,
		})
		if err != nil {
			if isNoWorkError(err) {
				if once {
					return printTeamValue(map[string]any{
						"runner_id": runnerID,
						"status":    "idle",
					})
				}
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(pollInterval):
					continue
				}
			}
			return fmt.Errorf("claim task: %w", err)
		}
		var claim workstation.ClaimResult
		if err := remarshal(result, &claim); err != nil {
			return fmt.Errorf("decode runner claim: %w", err)
		}
		executionContext, cancelExecution := context.WithCancel(ctx)
		heartbeatDone := make(chan error, 1)
		go runLeaseHeartbeat(
			executionContext,
			cancelExecution,
			claim,
			heartbeatInterval,
			heartbeatDone,
		)
		evidence, executionErr := executor.ExecuteClaim(executionContext, claim)
		cancelExecution()
		heartbeatErr := <-heartbeatDone
		if heartbeatErr != nil && executionErr == nil {
			executionErr = fmt.Errorf("lease heartbeat failed: %w", heartbeatErr)
		}
		if executionErr != nil {
			params := map[string]any{
				"runner_id":       runnerID,
				"task_id":         claim.Task.ID,
				"lease_id":        claim.Lease.ID,
				"idempotency_key": runnerOperationKey(runnerID + "-fail"),
				"error":           executionErr.Error(),
			}
			if evidence.Signature != "" {
				params["evidence"] = evidence
			}
			failed, err := callRunnerRPCWithRetry("runner.fail", params)
			if err != nil {
				return fmt.Errorf(
					"submit failure for task %s after local error %v: %w",
					claim.Task.ID,
					executionErr,
					err,
				)
			}
			if err := printTeamValue(map[string]any{
				"task":        failed,
				"local_error": executionErr.Error(),
			}); err != nil {
				return err
			}
			if once {
				return executionErr
			}
			continue
		}
		completed, err := callRunnerRPCWithRetry("runner.complete", map[string]any{
			"runner_id":       runnerID,
			"task_id":         claim.Task.ID,
			"lease_id":        claim.Lease.ID,
			"idempotency_key": runnerOperationKey(runnerID + "-complete"),
			"summary":         "local Codex execution and frozen verification passed",
			"evidence":        evidence,
		})
		if err != nil {
			return fmt.Errorf("submit completed task %s: %w", claim.Task.ID, err)
		}
		if err := printTeamValue(completed); err != nil {
			return err
		}
		if once {
			return nil
		}
	}
}

func runLeaseHeartbeat(
	ctx context.Context,
	cancelExecution context.CancelFunc,
	claim workstation.ClaimResult,
	interval time.Duration,
	done chan<- error,
) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			done <- nil
			return
		case <-ticker.C:
			_, err := callRunnerRPCWithRetry("runner.heartbeat", map[string]any{
				"runner_id":       claim.Lease.RunnerID,
				"task_id":         claim.Task.ID,
				"lease_id":        claim.Lease.ID,
				"idempotency_key": runnerOperationKey(claim.Lease.RunnerID + "-heartbeat"),
			})
			if err != nil {
				cancelExecution()
				done <- err
				return
			}
		}
	}
}

func callRunnerRPCWithRetry(method string, params map[string]any) (any, error) {
	cfg, err := config.Load("")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(os.Getenv("GOCLAW_USER_TOKEN")) == "" {
		return nil, errors.New("GOCLAW_USER_TOKEN is required for runner operations")
	}
	var last error
	for attempt := 0; attempt < 3; attempt++ {
		result, err := callGatewayRPC(cfg, method, params)
		if err == nil {
			return result, nil
		}
		last = err
		if isNoWorkError(err) {
			break
		}
		if attempt < 2 {
			time.Sleep(time.Duration(attempt+1) * 500 * time.Millisecond)
		}
	}
	return nil, last
}

func runnerOperationKey(prefix string) string {
	random, err := generateAccessToken()
	if err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return prefix + "-" + random
}

func isNoWorkError(err error) bool {
	return err != nil && strings.Contains(
		strings.ToLower(err.Error()),
		"no compatible queued task",
	)
}

func parseRepositoryMappings(values []string) (map[string]string, error) {
	result := make(map[string]string, len(values))
	for _, value := range values {
		id, path, found := strings.Cut(value, "=")
		id = strings.TrimSpace(id)
		path = strings.TrimSpace(path)
		if !found || id == "" || path == "" {
			return nil, fmt.Errorf("invalid --repo %q; expected ID=/absolute/path", value)
		}
		if _, exists := result[id]; exists {
			return nil, fmt.Errorf("duplicate repository id %q", id)
		}
		result[id] = path
	}
	if len(result) == 0 {
		return nil, errors.New("at least one --repo mapping is required")
	}
	return result, nil
}

func stageBinarySecretFile(path string, value []byte) (func(), error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("secret file is required")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(absolute), 0o700); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(absolute, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("create secret file: %w", err)
	}
	remove := func() { _ = os.Remove(absolute) }
	if _, err := file.Write(value); err != nil {
		_ = file.Close()
		remove()
		return nil, err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		remove()
		return nil, err
	}
	if err := file.Close(); err != nil {
		remove()
		return nil, err
	}
	return remove, nil
}

func remarshal(value, target any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}
