package workstation

import (
	"errors"
	"fmt"
	"strings"
)

type ExecutionProfile string

const (
	ExecutionProfileStrict         ExecutionProfile = "strict"
	ExecutionProfileCodexDelegated ExecutionProfile = "codex-delegated"

	RunnerStrictProfileCapability  = "goclaw-execution-profile:strict"
	RunnerCodexDelegatedCapability = "goclaw-execution-profile:codex-delegated"
)

func NormalizeExecutionProfile(value string) (ExecutionProfile, error) {
	switch ExecutionProfile(strings.ToLower(strings.TrimSpace(value))) {
	case "", ExecutionProfileStrict:
		return ExecutionProfileStrict, nil
	case ExecutionProfileCodexDelegated:
		return ExecutionProfileCodexDelegated, nil
	default:
		return "", errors.New("unsupported runner execution profile")
	}
}

func ExecutionProfileCapability(profile ExecutionProfile) string {
	if profile == ExecutionProfileCodexDelegated {
		return RunnerCodexDelegatedCapability
	}
	return RunnerStrictProfileCapability
}

func RunnerVersionCapability(version string) (string, error) {
	version = strings.TrimSpace(version)
	if version == "" || len(version) > 100 {
		return "", errors.New(
			"runner version is required and must not exceed 100 bytes",
		)
	}
	return "goclaw-version-sha256:" + sha256Bytes([]byte(version)), nil
}

func RunnerReleaseCapability(releaseID string) (string, error) {
	releaseID = strings.TrimSpace(releaseID)
	if err := validateID(releaseID); err != nil {
		return "", err
	}
	return "goclaw-release:" + releaseID, nil
}

func ValidateRunnerExecutionProfile(
	info RunnerRuntime,
	profile ExecutionProfile,
) error {
	switch profile {
	case ExecutionProfileStrict:
		return ValidateRunnerExecutionRuntime(info)
	case ExecutionProfileCodexDelegated:
		switch info.OS {
		case "linux", "windows", "darwin":
		default:
			return fmt.Errorf(
				"codex-delegated execution is unsupported on %s",
				valueOr(info.OS, "this host"),
			)
		}
		switch info.Arch {
		case "amd64", "arm64":
			return nil
		default:
			return fmt.Errorf(
				"codex-delegated execution is unsupported on architecture %s",
				valueOr(info.Arch, "unknown"),
			)
		}
	default:
		return errors.New("unsupported runner execution profile")
	}
}

func RunnerRegistrationCapabilitiesForProfile(
	configured []string,
	profile ExecutionProfile,
) []string {
	info := CurrentRunnerRuntime()
	result := RunnerRegistrationCapabilities(configured)
	if ValidateRunnerExecutionProfile(info, profile) == nil {
		result = append(result, ExecutionProfileCapability(profile))
	}
	return normalizeCapabilities(result)
}

func RunnerRegistrationMetadataForProfile(
	profile ExecutionProfile,
) map[string]string {
	metadata := RunnerRegistrationMetadata()
	metadata["execution_profile"] = string(profile)
	metadata["directory_boundary"] = "goclaw-worktree-v1"
	if profile == ExecutionProfileCodexDelegated {
		metadata["isolation_backend"] = "codex-delegated"
		metadata["network_isolation"] = "not-provided"
		metadata["security_posture"] = "degraded-explicit"
	} else {
		metadata["network_isolation"] = "required"
		metadata["security_posture"] = "strict"
	}
	return metadata
}

func runnerSupportsExecutionProfile(
	runner Runner,
	requested ExecutionProfile,
) bool {
	profile, err := NormalizeExecutionProfile(string(requested))
	if err != nil {
		return false
	}
	if containsString(
		runner.Capabilities,
		ExecutionProfileCapability(profile),
		true,
	) {
		return true
	}
	// Pre-RN runners did not advertise the strict profile explicitly. Preserve
	// compatibility only for the old Linux runtime contract; delegated is
	// always opt-in and never inferred.
	if profile != ExecutionProfileStrict {
		return false
	}
	return runner.Metadata["execution_profile"] !=
		string(ExecutionProfileCodexDelegated)
}
