package workstation

import (
	"testing"
	"time"
)

func TestExecutionProfileNormalizationAndRuntimeMatrix(t *testing.T) {
	for input, expected := range map[string]ExecutionProfile{
		"":                ExecutionProfileStrict,
		" strict ":        ExecutionProfileStrict,
		"CODEX-DELEGATED": ExecutionProfileCodexDelegated,
	} {
		got, err := NormalizeExecutionProfile(input)
		if err != nil || got != expected {
			t.Fatalf("NormalizeExecutionProfile(%q) = %q, %v", input, got, err)
		}
	}
	if _, err := NormalizeExecutionProfile("unsafe-host"); err == nil {
		t.Fatal("unknown execution profile passed")
	}

	for _, info := range []RunnerRuntime{
		{OS: "linux", Arch: "amd64"},
		{OS: "windows", Arch: "amd64"},
		{OS: "darwin", Arch: "arm64"},
	} {
		if err := ValidateRunnerExecutionProfile(
			info,
			ExecutionProfileCodexDelegated,
		); err != nil {
			t.Fatalf("delegated %s/%s rejected: %v", info.OS, info.Arch, err)
		}
	}
	if err := ValidateRunnerExecutionProfile(
		RunnerRuntime{OS: "windows", Arch: "amd64"},
		ExecutionProfileStrict,
	); err == nil {
		t.Fatal("strict native Windows unexpectedly passed")
	}
	if err := ValidateRunnerExecutionProfile(
		RunnerRuntime{OS: "linux", Arch: "386"},
		ExecutionProfileCodexDelegated,
	); err == nil {
		t.Fatal("unsupported delegated architecture unexpectedly passed")
	}
}

func TestRunnerProfileCapabilityNeverInfersDelegated(t *testing.T) {
	legacy := Runner{Capabilities: []string{"codex", "go"}}
	if !runnerSupportsExecutionProfile(legacy, ExecutionProfileStrict) {
		t.Fatal("legacy runner lost strict compatibility")
	}
	if runnerSupportsExecutionProfile(
		legacy,
		ExecutionProfileCodexDelegated,
	) {
		t.Fatal("legacy runner inferred delegated capability")
	}
	delegated := Runner{
		Capabilities: []string{RunnerCodexDelegatedCapability},
		Metadata: map[string]string{
			"execution_profile": string(ExecutionProfileCodexDelegated),
		},
	}
	if !runnerSupportsExecutionProfile(
		delegated,
		ExecutionProfileCodexDelegated,
	) {
		t.Fatal("declared delegated runner rejected")
	}
	if runnerSupportsExecutionProfile(delegated, ExecutionProfileStrict) {
		t.Fatal("delegated-only runner claimed strict work")
	}
}

func TestCodexSandboxTargetUsesCodexPlatformNames(t *testing.T) {
	for input, expected := range map[string]string{
		"linux":   "linux",
		"windows": "windows",
		"darwin":  "macos",
	} {
		if got := codexSandboxTarget(input); got != expected {
			t.Fatalf("codexSandboxTarget(%q) = %q, want %q", input, got, expected)
		}
	}
	command, args := codexCredentialCanaryCommand(
		"windows",
		`C:\Users\alice\.codex`,
	)
	if command != "powershell.exe" ||
		args[len(args)-1] != `C:\Users\alice\.codex` {
		t.Fatalf("Windows canary = %q %#v", command, args)
	}
}

func TestDelegatedTaskRequiresExplicitRunnerCapability(t *testing.T) {
	service, _ := newTestService(t, 2)
	registerTestRunner(
		t,
		service,
		"strict-runner",
		testDeviceKey('s'),
		[]string{"alpha"},
		[]string{"codex", "go"},
	)
	registerTestRunner(
		t,
		service,
		"delegated-runner",
		testDeviceKey('d'),
		[]string{"alpha"},
		[]string{"codex", "go", RunnerCodexDelegatedCapability},
	)
	request := testEnqueueRequest(
		"alpha",
		"delegated-enqueue",
		"run delegated task",
	)
	request.ExecutionPack.ExecutionProfile = ExecutionProfileCodexDelegated
	if _, err := service.Enqueue(request); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Claim(ClaimRequest{
		RunnerID:       "strict-runner",
		ProjectID:      "alpha",
		IdempotencyKey: "strict-claim",
	}); err != ErrNoTaskAvailable {
		t.Fatalf("strict-only runner claim error = %v", err)
	}
	if _, err := service.Claim(ClaimRequest{
		RunnerID:       "delegated-runner",
		ProjectID:      "alpha",
		IdempotencyKey: "delegated-claim",
	}); err != nil {
		t.Fatalf("delegated runner claim: %v", err)
	}
}

func TestClaimPriorityAgingPreventsStarvation(t *testing.T) {
	service, clock := newTestService(t, 2)
	registerTestRunner(
		t,
		service,
		"fair-runner",
		testDeviceKey('f'),
		[]string{"alpha"},
		[]string{"codex", "go"},
	)
	old := testEnqueueRequest("alpha", "old-low", "old low priority")
	old.Priority = minTaskPriority
	oldTask, err := service.Enqueue(old)
	if err != nil {
		t.Fatal(err)
	}
	clock.Add((maxTaskPriority - minTaskPriority + 1) * claimPriorityAgingGap)
	fresh := testEnqueueRequest("alpha", "fresh-high", "fresh high priority")
	fresh.Priority = maxTaskPriority
	if _, err := service.Enqueue(fresh); err != nil {
		t.Fatal(err)
	}
	claim := mustClaim(
		t,
		service,
		"fair-runner",
		"alpha",
		"fair-aging-claim",
	)
	if claim.Task.ID != oldTask.ID {
		t.Fatalf("claimed %s, want aged task %s", claim.Task.ID, oldTask.ID)
	}

	outOfRange := testEnqueueRequest("alpha", "bad-priority", "bad")
	outOfRange.Priority = maxTaskPriority + 1
	if _, err := service.Enqueue(outOfRange); err == nil {
		t.Fatal("out-of-range priority unexpectedly accepted")
	}

	if claimWaitingAge(
		Task{CreatedAt: clock.Time().Add(time.Minute)},
		clock.Time(),
	) != 0 {
		t.Fatal("future task acquired waiting age")
	}
}

func TestHeartbeatProjectsRunnerLifecycleWithoutSecrets(t *testing.T) {
	service, _ := newTestService(t, 2)
	runner := registerTestRunner(
		t,
		service,
		"lifecycle-runner",
		testDeviceKey('l'),
		[]string{"alpha"},
		[]string{"codex", RunnerStrictProfileCapability},
	)
	updated, err := service.HeartbeatRunnerLifecycle(
		runner.ID,
		RunnerLifecycleProjection{
			CurrentVersion:   "0.9.0",
			CurrentReleaseID: "release-090",
			ReleaseProtocol:  RunnerReleaseProtocol,
			ExecutionProfile: ExecutionProfileStrict,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Metadata["current_version"] != "0.9.0" ||
		updated.Metadata["current_release_id"] != "release-090" ||
		updated.Metadata["rollout_state"] != "reported" {
		t.Fatalf("lifecycle metadata = %#v", updated.Metadata)
	}
	versionCapability, _ := RunnerVersionCapability("0.9.0")
	releaseCapability, _ := RunnerReleaseCapability("release-090")
	if !containsString(updated.Capabilities, versionCapability, true) ||
		!containsString(updated.Capabilities, releaseCapability, true) {
		t.Fatalf("lifecycle capabilities = %#v", updated.Capabilities)
	}
	if _, err := service.HeartbeatRunnerLifecycle(
		runner.ID,
		RunnerLifecycleProjection{
			CurrentVersion:   "0.9.1",
			ReleaseProtocol:  "future",
			ExecutionProfile: ExecutionProfileStrict,
		},
	); err == nil {
		t.Fatal("incompatible release protocol passed")
	}
}
