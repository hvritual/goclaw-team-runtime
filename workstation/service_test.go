package workstation

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *fakeClock) Time() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Add(value time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(value)
}

func newTestService(t *testing.T, maxAttempts int) (*Service, *fakeClock) {
	t.Helper()
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.Root = t.TempDir()
	cfg.LeaseDurationSeconds = 10
	cfg.RunnerOfflineSeconds = 15
	cfg.DefaultMaxAttempts = maxAttempts
	service, err := NewService(cfg)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	clock := &fakeClock{now: time.Date(2026, 7, 25, 10, 0, 0, 0, time.UTC)}
	service.now = clock.Time
	return service, clock
}

func testDeviceKey(seed byte) []byte {
	return bytes.Repeat([]byte{seed}, deviceKeyBytes)
}

func registerTestRunner(
	t *testing.T,
	service *Service,
	id string,
	key []byte,
	projects, capabilities []string,
) Runner {
	t.Helper()
	runner, err := service.RegisterRunner(RegisterRunnerRequest{
		ID: id, Name: id, OwnerUserID: "user-" + id,
		Projects: projects, Capabilities: capabilities,
	}, key)
	if err != nil {
		t.Fatalf("RegisterRunner(%s): %v", id, err)
	}
	return runner
}

func testEnqueueRequest(projectID, key, prompt string) EnqueueRequest {
	return EnqueueRequest{
		ProjectID:            projectID,
		IdempotencyKey:       key,
		RequiredCapabilities: []string{"codex", "go"},
		ExecutionPack: ExecutionPack{
			TaskRevision:  1,
			ProjectID:     projectID,
			CorrelationID: "corr-1",
			IssueIDs:      []string{"BUG-42"},
			SpecHash:      "spec-sha",
			WorkItemIDs:   []string{"WI-1"},
			RepositoryID:  "iot-platform",
			BaseRef:       "main",
			BaseCommit:    "0123456789abcdef",
			Prompt:        prompt,
			Verification: []CommandSpec{{
				Name: "go test", Argv: []string{"go", "test", "./..."},
			}},
			HarnessVersion:    "org-go-v1",
			PolicyPackVersion: "iot-v2",
			PolicyBundleHash:  fmt.Sprintf("%064x", 7),
		},
	}
}

func mustEnqueue(t *testing.T, service *Service, projectID, key string) Task {
	t.Helper()
	task, err := service.Enqueue(testEnqueueRequest(projectID, key, "implement the frozen task"))
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	return task
}

func mustClaim(t *testing.T, service *Service, runnerID, projectID, key string) ClaimResult {
	t.Helper()
	claim, err := service.Claim(ClaimRequest{
		RunnerID: runnerID, ProjectID: projectID, IdempotencyKey: key,
	})
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	return claim
}

func signedEvidence(
	t *testing.T,
	task Task,
	lease Lease,
	key []byte,
	outcome string,
	now time.Time,
) EvidenceBundle {
	t.Helper()
	bundle, err := SignEvidenceBundle(EvidenceBundle{
		TaskID:              task.ID,
		ProjectID:           task.ProjectID,
		ExecutionPackSHA256: task.ExecutionPackSHA256,
		RunnerID:            lease.RunnerID,
		LeaseID:             lease.ID,
		Attempt:             lease.Attempt,
		Outcome:             outcome,
		StartedAt:           now.Add(-time.Minute),
		FinishedAt:          now,
		BaseCommit:          task.ExecutionPack.BaseCommit,
		HeadCommit:          task.ExecutionPack.BaseCommit,
		Branch:              "goclaw/" + task.ID,
		ChangedFiles:        []string{"internal/device/service.go"},
		DiffPatch:           "diff --git a/internal/device/service.go b/internal/device/service.go\n",
		Checks: []EvidenceCheck{
			{Name: "runner-setup", Passed: true},
			{Name: "codex-exec", Passed: true},
			{Name: "go test", Passed: true, DurationMS: 1200},
			{Name: "scope-policy", Passed: true},
			{Name: "no-automatic-commit", Passed: true},
		},
		Artifacts: []EvidenceArtifact{{
			Name: "test-log", SHA256: fmt.Sprintf("%064x", 99), SizeBytes: 1024,
		}},
		TraceIDs: []string{"trace-1"},
	}, key)
	if err != nil {
		t.Fatalf("SignEvidenceBundle: %v", err)
	}
	bundle.DiffSHA256 = sha256Bytes([]byte(bundle.DiffPatch))
	bundle, err = SignEvidenceBundle(bundle, key)
	if err != nil {
		t.Fatalf("resign evidence bundle: %v", err)
	}
	return bundle
}

func TestEnqueueRegistrationAuthorizationAndIdempotency(t *testing.T) {
	service, _ := newTestService(t, 3)
	key := testDeviceKey('a')
	runner := registerTestRunner(t, service, "runner-alpha", key, []string{"alpha"}, []string{"go", "codex"})
	if runner.KeyID == "" {
		t.Fatal("runner key id is empty")
	}
	if runner.OwnerUserID != "user-runner-alpha" {
		t.Fatalf("runner owner = %q", runner.OwnerUserID)
	}
	persistedRunner, err := service.GetRunner("runner-alpha")
	if err != nil {
		t.Fatalf("GetRunner: %v", err)
	}
	if persistedRunner.OwnerUserID != runner.OwnerUserID {
		t.Fatalf("persisted owner = %q, want %q", persistedRunner.OwnerUserID, runner.OwnerUserID)
	}
	if _, err := service.RegisterRunner(RegisterRunnerRequest{
		ID: "runner-alpha", Name: "runner-alpha", OwnerUserID: "user-runner-alpha",
		Projects:     []string{"alpha"},
		Capabilities: []string{"codex", "go"},
	}, key); err != nil {
		t.Fatalf("idempotent registration: %v", err)
	}

	first := mustEnqueue(t, service, "alpha", "enqueue-1")
	second, err := service.Enqueue(testEnqueueRequest("alpha", "enqueue-1", "implement the frozen task"))
	if err != nil {
		t.Fatalf("idempotent Enqueue: %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("idempotent enqueue returned %s, want %s", second.ID, first.ID)
	}
	changed := testEnqueueRequest("alpha", "enqueue-1", "different request")
	if _, err := service.Enqueue(changed); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("changed idempotent enqueue error = %v", err)
	}

	mustEnqueue(t, service, "beta", "enqueue-beta")
	if _, err := service.Claim(ClaimRequest{
		RunnerID: "runner-alpha", ProjectID: "beta", IdempotencyKey: "claim-beta",
	}); !errors.Is(err, ErrNoTaskAvailable) {
		t.Fatalf("unauthorized project claim error = %v", err)
	}

	claim := mustClaim(t, service, "runner-alpha", "alpha", "claim-alpha")
	retry := mustClaim(t, service, "runner-alpha", "alpha", "claim-alpha")
	if claim.Task.ID != retry.Task.ID || claim.Lease.ID != retry.Lease.ID {
		t.Fatalf("claim idempotency mismatch: %#v %#v", claim, retry)
	}
	if _, err := service.Claim(ClaimRequest{
		RunnerID: "runner-alpha", ProjectID: "beta", IdempotencyKey: "claim-alpha",
	}); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("claim key reuse error = %v", err)
	}
}

func TestClaimHonorsExecutionPackAssignee(t *testing.T) {
	service, _ := newTestService(t, 3)
	alice := registerTestRunner(
		t,
		service,
		"runner-alice",
		testDeviceKey('a'),
		[]string{"alpha"},
		[]string{"codex", "go"},
	)
	bob := registerTestRunner(
		t,
		service,
		"runner-bob",
		testDeviceKey('b'),
		[]string{"alpha"},
		[]string{"codex", "go"},
	)
	assigned := testEnqueueRequest("alpha", "enqueue-assigned", "assigned task")
	assigned.ExecutionPack.Metadata = map[string]string{
		"assignee_id": alice.OwnerUserID,
	}
	assignedTask, err := service.Enqueue(assigned)
	if err != nil {
		t.Fatalf("enqueue assigned task: %v", err)
	}
	if _, err := service.Claim(ClaimRequest{
		RunnerID:       bob.ID,
		ProjectID:      "alpha",
		IdempotencyKey: "claim-bob-assigned",
	}); !errors.Is(err, ErrNoTaskAvailable) {
		t.Fatalf("bob claim assigned task error = %v, want ErrNoTaskAvailable", err)
	}
	aliceClaim := mustClaim(t, service, alice.ID, "alpha", "claim-alice-assigned")
	if aliceClaim.Task.ID != assignedTask.ID {
		t.Fatalf("alice claimed %s, want assigned task %s", aliceClaim.Task.ID, assignedTask.ID)
	}
}

func TestClaimAllowsUnassignedExecutionPack(t *testing.T) {
	service, _ := newTestService(t, 3)
	runner := registerTestRunner(
		t,
		service,
		"runner-any",
		testDeviceKey('u'),
		[]string{"alpha"},
		[]string{"codex", "go"},
	)
	unassigned := mustEnqueue(t, service, "alpha", "enqueue-unassigned")
	claim := mustClaim(t, service, runner.ID, "alpha", "claim-unassigned")
	if claim.Task.ID != unassigned.ID {
		t.Fatalf("runner claimed %s, want unassigned task %s", claim.Task.ID, unassigned.ID)
	}
}

func TestCompleteRequiresFrozenPassingEvidenceChecks(t *testing.T) {
	service, clock := newTestService(t, 3)
	key := testDeviceKey('c')
	runner := registerTestRunner(
		t,
		service,
		"runner-contract",
		key,
		[]string{"alpha"},
		[]string{"codex", "go"},
	)
	task := mustEnqueue(t, service, "alpha", "enqueue-contract")
	claim := mustClaim(t, service, runner.ID, "alpha", "claim-contract")
	valid := signedEvidence(
		t,
		claim.Task,
		claim.Lease,
		key,
		"completed",
		clock.Time(),
	)

	missingChecks := valid
	missingChecks.Checks = nil
	missingChecks, err := SignEvidenceBundle(missingChecks, key)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Complete(CompleteRequest{
		RunnerID: runner.ID, TaskID: task.ID, LeaseID: claim.Lease.ID,
		IdempotencyKey: "complete-missing-checks", Evidence: missingChecks,
	}); !errors.Is(err, ErrInvalidEvidence) {
		t.Fatalf("missing checks completion error = %v", err)
	}

	failedCheck := valid
	for index := range failedCheck.Checks {
		if failedCheck.Checks[index].Name == "go test" {
			failedCheck.Checks[index].Passed = false
			failedCheck.Checks[index].ExitCode = 1
		}
	}
	failedCheck, err = SignEvidenceBundle(failedCheck, key)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Complete(CompleteRequest{
		RunnerID: runner.ID, TaskID: task.ID, LeaseID: claim.Lease.ID,
		IdempotencyKey: "complete-failed-check", Evidence: failedCheck,
	}); !errors.Is(err, ErrInvalidEvidence) {
		t.Fatalf("failed check completion error = %v", err)
	}
}

func TestRotateRunnerDeviceKeyRequiresNoActiveLease(t *testing.T) {
	service, _ := newTestService(t, 3)
	oldKey := testDeviceKey('o')
	newKey := testDeviceKey('n')
	runner := registerTestRunner(
		t,
		service,
		"runner-rotate",
		oldKey,
		[]string{"alpha"},
		[]string{"codex", "go"},
	)
	mustEnqueue(t, service, "alpha", "enqueue-rotate")
	claim := mustClaim(t, service, runner.ID, "alpha", "claim-rotate")
	if _, err := service.RotateRunnerDeviceKey(runner.ID, newKey); !errors.Is(err, ErrConflict) {
		t.Fatalf("rotate with active lease error = %v", err)
	}
	evidence := signedEvidence(
		t,
		claim.Task,
		claim.Lease,
		oldKey,
		"completed",
		time.Now().UTC(),
	)
	if _, err := service.Complete(CompleteRequest{
		RunnerID: runner.ID, TaskID: claim.Task.ID, LeaseID: claim.Lease.ID,
		IdempotencyKey: "complete-rotate", Evidence: evidence,
	}); err != nil {
		t.Fatal(err)
	}
	rotated, err := service.RotateRunnerDeviceKey(runner.ID, newKey)
	if err != nil {
		t.Fatal(err)
	}
	newKeyID, err := DeviceKeyID(newKey)
	if err != nil {
		t.Fatal(err)
	}
	if rotated.KeyID != newKeyID || rotated.KeyID == runner.KeyID {
		t.Fatalf("rotated key id = %q, old = %q", rotated.KeyID, runner.KeyID)
	}
	stored, err := service.loadCredentialUnlocked(runner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if string(stored) != string(newKey) {
		t.Fatal("rotated credential does not match new key")
	}
	historical, err := service.GetEvidenceBundle(claim.Task.ID)
	if err != nil {
		t.Fatalf("read evidence signed before rotation: %v", err)
	}
	if historical.KeyID == rotated.KeyID {
		t.Fatal("historical evidence unexpectedly changed signing key")
	}
	if err := service.VerifyEvidence(runner.ID, historical); err != nil {
		t.Fatalf("verify evidence signed before rotation: %v", err)
	}
	idempotent, err := service.RotateRunnerDeviceKey(runner.ID, newKey)
	if err != nil {
		t.Fatal(err)
	}
	if idempotent.KeyID != rotated.KeyID {
		t.Fatal("idempotent rotation changed key id")
	}
}

func TestHeartbeatExtendsLeaseAndExpiryRecovery(t *testing.T) {
	service, clock := newTestService(t, 2)
	registerTestRunner(t, service, "runner-alpha", testDeviceKey('b'), []string{"alpha"}, []string{"go", "codex"})
	mustEnqueue(t, service, "alpha", "enqueue-expire")
	claim := mustClaim(t, service, "runner-alpha", "alpha", "claim-1")
	originalExpiry := claim.Lease.ExpiresAt

	clock.Add(6 * time.Second)
	heartbeat, err := service.Heartbeat(HeartbeatRequest{
		RunnerID: "runner-alpha", TaskID: claim.Task.ID, LeaseID: claim.Lease.ID,
		IdempotencyKey: "heartbeat-1",
	})
	if err != nil {
		t.Fatalf("Heartbeat: %v", err)
	}
	if !heartbeat.Lease.ExpiresAt.After(originalExpiry) {
		t.Fatalf("heartbeat expiry %s did not extend %s", heartbeat.Lease.ExpiresAt, originalExpiry)
	}
	retry, err := service.Heartbeat(HeartbeatRequest{
		RunnerID: "runner-alpha", TaskID: claim.Task.ID, LeaseID: claim.Lease.ID,
		IdempotencyKey: "heartbeat-1",
	})
	if err != nil {
		t.Fatalf("idempotent Heartbeat: %v", err)
	}
	if !retry.Lease.ExpiresAt.Equal(heartbeat.Lease.ExpiresAt) {
		t.Fatal("idempotent heartbeat extended lease twice")
	}

	clock.Add(11 * time.Second)
	report, err := service.RecoverExpiredLeases()
	if err != nil {
		t.Fatalf("RecoverExpiredLeases: %v", err)
	}
	if len(report.RequeuedTaskIDs) != 1 || report.RequeuedTaskIDs[0] != claim.Task.ID {
		t.Fatalf("unexpected first recovery: %#v", report)
	}
	if _, err := service.Claim(ClaimRequest{
		RunnerID:       "runner-alpha",
		ProjectID:      "alpha",
		IdempotencyKey: "claim-1",
	}); !errors.Is(err, ErrConflict) {
		t.Fatalf("expired claim receipt replay error = %v", err)
	}
	second := mustClaim(t, service, "runner-alpha", "alpha", "claim-2")
	clock.Add(16 * time.Second)
	report, err = service.RecoverExpiredLeases()
	if err != nil {
		t.Fatalf("second RecoverExpiredLeases: %v", err)
	}
	if len(report.FailedTaskIDs) != 1 || report.FailedTaskIDs[0] != second.Task.ID {
		t.Fatalf("unexpected exhausted recovery: %#v", report)
	}
	task, err := service.GetTask(second.Task.ID)
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if task.Status != TaskFailed || task.Attempt != 2 {
		t.Fatalf("expired task = status %s attempt %d", task.Status, task.Attempt)
	}
	runner, err := service.GetRunner("runner-alpha")
	if err != nil {
		t.Fatalf("GetRunner: %v", err)
	}
	if runner.Status != RunnerOffline {
		t.Fatalf("runner status = %s, want offline", runner.Status)
	}
}

func TestCompleteVerifiesEvidenceAndPersistsWithoutDeviceKey(t *testing.T) {
	service, clock := newTestService(t, 3)
	key := testDeviceKey('c')
	registerTestRunner(t, service, "runner-alpha", key, []string{"alpha"}, []string{"go", "codex"})
	task := mustEnqueue(t, service, "alpha", "enqueue-complete")
	claim := mustClaim(t, service, "runner-alpha", "alpha", "claim-complete")
	bundle := signedEvidence(t, claim.Task, claim.Lease, key, "completed", clock.Time())
	tampered := bundle
	tampered.FinishedAt = tampered.FinishedAt.Add(time.Millisecond)
	if _, err := service.Complete(CompleteRequest{
		RunnerID: "runner-alpha", TaskID: task.ID, LeaseID: claim.Lease.ID,
		IdempotencyKey: "complete-tampered", Evidence: tampered,
	}); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("tampered evidence error = %v", err)
	}
	mismatchedPatch := bundle
	mismatchedPatch.DiffPatch += "tampered\n"
	mismatchedPatch, err := SignEvidenceBundle(mismatchedPatch, key)
	if err != nil {
		t.Fatalf("sign mismatched patch: %v", err)
	}
	if _, err := service.Complete(CompleteRequest{
		RunnerID: "runner-alpha", TaskID: task.ID, LeaseID: claim.Lease.ID,
		IdempotencyKey: "complete-mismatched-patch", Evidence: mismatchedPatch,
	}); !errors.Is(err, ErrInvalidEvidence) {
		t.Fatalf("mismatched patch digest error = %v", err)
	}
	oversized := bundle
	oversized.DiffPatch = strings.Repeat("x", MaxEvidenceDiffPatchBytes+1)
	oversized.DiffSHA256 = sha256Bytes([]byte(oversized.DiffPatch))
	oversized, err = SignEvidenceBundle(oversized, key)
	if err != nil {
		t.Fatalf("sign oversized patch: %v", err)
	}
	if _, err := service.Complete(CompleteRequest{
		RunnerID: "runner-alpha", TaskID: task.ID, LeaseID: claim.Lease.ID,
		IdempotencyKey: "complete-oversized-patch", Evidence: oversized,
	}); !errors.Is(err, ErrInvalidEvidence) {
		t.Fatalf("oversized patch error = %v", err)
	}

	request := CompleteRequest{
		RunnerID: "runner-alpha", TaskID: task.ID, LeaseID: claim.Lease.ID,
		IdempotencyKey: "complete-1", Summary: "done", Evidence: bundle,
	}
	completed, err := service.Complete(request)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if completed.Status != TaskCompleted || completed.Result == nil {
		t.Fatalf("completed task = %#v", completed)
	}
	retry, err := service.Complete(request)
	if err != nil {
		t.Fatalf("idempotent Complete: %v", err)
	}
	if retry.Result.Evidence.BundleSHA256 != bundle.BundleSHA256 {
		t.Fatal("idempotent complete changed result")
	}
	if err := service.VerifyEvidence("runner-alpha", bundle); err != nil {
		t.Fatalf("VerifyEvidence: %v", err)
	}
	stored, err := service.GetEvidenceBundle(task.ID)
	if err != nil {
		t.Fatalf("GetEvidenceBundle: %v", err)
	}
	if stored.BundleSHA256 != bundle.BundleSHA256 {
		t.Fatalf("stored evidence digest = %s", stored.BundleSHA256)
	}

	taskPath := filepath.Join(service.Config().Root, "tasks", task.ID+".json")
	taskJSON, err := os.ReadFile(taskPath)
	if err != nil {
		t.Fatalf("read task JSON: %v", err)
	}
	if bytes.Contains(taskJSON, key) {
		t.Fatal("raw device key leaked into task JSON")
	}
	var decoded Task
	if err := json.Unmarshal(taskJSON, &decoded); err != nil {
		t.Fatalf("task JSON is not valid: %v", err)
	}
	credentialPath := filepath.Join(service.Config().Root, "credentials", "runner-alpha.key")
	info, err := os.Stat(credentialPath)
	if err != nil {
		t.Fatalf("stat credential: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("credential mode = %o, want 600", info.Mode().Perm())
	}

	restarted, err := NewService(service.Config())
	if err != nil {
		t.Fatalf("restart NewService: %v", err)
	}
	reloaded, err := restarted.GetTask(task.ID)
	if err != nil {
		t.Fatalf("restarted GetTask: %v", err)
	}
	if reloaded.Status != TaskCompleted || reloaded.Result.Evidence.BundleSHA256 != bundle.BundleSHA256 {
		t.Fatalf("reloaded task = %#v", reloaded)
	}
}

func TestRunnerExecutionProfileCannotChangeDuringLease(t *testing.T) {
	service, _ := newTestService(t, 2)
	registerTestRunner(
		t,
		service,
		"runner-profile",
		testDeviceKey('u'),
		[]string{"alpha"},
		[]string{"codex", "go"},
	)
	mustEnqueue(t, service, "alpha", "enqueue-profile-lock")
	claim := mustClaim(
		t,
		service,
		"runner-profile",
		"alpha",
		"claim-profile-lock",
	)
	disabled := true
	if _, err := service.UpdateRunner(
		"runner-profile",
		UpdateRunnerRequest{Disabled: &disabled},
	); !errors.Is(err, ErrConflict) {
		t.Fatalf("active runner profile update error = %v", err)
	}
	renamed, err := service.UpdateRunner(
		"runner-profile",
		UpdateRunnerRequest{Name: "Renamed while busy"},
	)
	if err != nil {
		t.Fatalf("cosmetic runner update: %v", err)
	}
	if renamed.Name != "Renamed while busy" {
		t.Fatalf("runner name = %q", renamed.Name)
	}
	if _, err := service.Fail(FailRequest{
		RunnerID:       "runner-profile",
		TaskID:         claim.Task.ID,
		LeaseID:        claim.Lease.ID,
		IdempotencyKey: "fail-profile-lock",
		Error:          "fixture failure",
	}); err != nil {
		t.Fatalf("Fail: %v", err)
	}
	disabledRunner, err := service.UpdateRunner(
		"runner-profile",
		UpdateRunnerRequest{Disabled: &disabled},
	)
	if err != nil {
		t.Fatalf("disable idle runner: %v", err)
	}
	if disabledRunner.Status != RunnerDisabled {
		t.Fatalf("runner status = %s", disabledRunner.Status)
	}
}

func TestFailAndRequeueAreIdempotent(t *testing.T) {
	service, _ := newTestService(t, 2)
	registerTestRunner(t, service, "runner-alpha", testDeviceKey('d'), []string{"alpha"}, []string{"go", "codex"})
	task := mustEnqueue(t, service, "alpha", "enqueue-fail")
	claim := mustClaim(t, service, "runner-alpha", "alpha", "claim-fail")
	fail := FailRequest{
		RunnerID: "runner-alpha", TaskID: task.ID, LeaseID: claim.Lease.ID,
		IdempotencyKey: "fail-1", Error: "tests failed",
	}
	failed, err := service.Fail(fail)
	if err != nil {
		t.Fatalf("Fail: %v", err)
	}
	if failed.Status != TaskFailed {
		t.Fatalf("failed status = %s", failed.Status)
	}
	if _, err := service.Fail(fail); err != nil {
		t.Fatalf("idempotent Fail: %v", err)
	}
	changed := fail
	changed.Error = "different failure"
	if _, err := service.Fail(changed); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("changed fail key error = %v", err)
	}

	requeue := RequeueRequest{
		TaskID: task.ID, Actor: "operator", Reason: "approved retry", IdempotencyKey: "requeue-1",
	}
	queued, err := service.Requeue(requeue)
	if err != nil {
		t.Fatalf("Requeue: %v", err)
	}
	if queued.Status != TaskQueued {
		t.Fatalf("requeued status = %s", queued.Status)
	}
	if _, err := service.Requeue(requeue); err != nil {
		t.Fatalf("idempotent Requeue: %v", err)
	}
	second := mustClaim(t, service, "runner-alpha", "alpha", "claim-after-requeue")
	if second.Task.Attempt != 2 {
		t.Fatalf("second claim attempt = %d", second.Task.Attempt)
	}
}

func TestCancelQueuedTaskIsIdempotentAndRejectsActiveLease(t *testing.T) {
	service, _ := newTestService(t, 2)
	registerTestRunner(
		t,
		service,
		"runner-cancel",
		testDeviceKey('x'),
		[]string{"alpha"},
		[]string{"go", "codex"},
	)
	queued := mustEnqueue(t, service, "alpha", "enqueue-cancel")
	request := CancelRequest{
		TaskID:         queued.ID,
		Actor:          "project-owner",
		Reason:         "superseded by a reviewed revision",
		IdempotencyKey: "cancel-queued",
	}
	cancelled, err := service.Cancel(request)
	if err != nil {
		t.Fatal(err)
	}
	if cancelled.Status != TaskCancelled ||
		cancelled.CompletedAt == nil ||
		cancelled.Lease != nil {
		t.Fatalf("cancelled task = %+v", cancelled)
	}
	if _, err := service.Cancel(request); err != nil {
		t.Fatalf("idempotent Cancel: %v", err)
	}
	changed := request
	changed.Reason = "different reason"
	if _, err := service.Cancel(changed); !errors.Is(
		err,
		ErrIdempotencyConflict,
	) {
		t.Fatalf("changed cancel key error = %v", err)
	}

	active := mustEnqueue(t, service, "alpha", "enqueue-active-cancel")
	claim := mustClaim(
		t,
		service,
		"runner-cancel",
		"alpha",
		"claim-active-cancel",
	)
	if claim.Task.ID != active.ID {
		t.Fatalf("claimed %s, want %s", claim.Task.ID, active.ID)
	}
	if _, err := service.Cancel(CancelRequest{
		TaskID:         active.ID,
		Actor:          "project-owner",
		Reason:         "must not interrupt an active executor",
		IdempotencyKey: "cancel-active",
	}); !errors.Is(err, ErrConflict) {
		t.Fatalf("active lease cancel error = %v", err)
	}
}

func TestConcurrentClaimsLeaseTaskExactlyOnce(t *testing.T) {
	service, _ := newTestService(t, 3)
	for index := 0; index < 8; index++ {
		id := fmt.Sprintf("runner-%d", index)
		registerTestRunner(t, service, id, testDeviceKey(byte(index+1)), []string{"alpha"}, []string{"go", "codex"})
	}
	task := mustEnqueue(t, service, "alpha", "enqueue-race")
	var wg sync.WaitGroup
	var mu sync.Mutex
	successes := make([]ClaimResult, 0)
	for index := 0; index < 8; index++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			claim, err := service.Claim(ClaimRequest{
				RunnerID:  fmt.Sprintf("runner-%d", index),
				ProjectID: "alpha", IdempotencyKey: fmt.Sprintf("claim-%d", index),
			})
			if err == nil {
				mu.Lock()
				successes = append(successes, claim)
				mu.Unlock()
				return
			}
			if !errors.Is(err, ErrNoTaskAvailable) {
				t.Errorf("Claim(%d): %v", index, err)
			}
		}(index)
	}
	wg.Wait()
	if len(successes) != 1 {
		t.Fatalf("successful claims = %d, want 1", len(successes))
	}
	if successes[0].Task.ID != task.ID {
		t.Fatalf("claimed task = %s, want %s", successes[0].Task.ID, task.ID)
	}
	stored, err := service.GetTask(task.ID)
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if stored.Status != TaskLeased || stored.Attempt != 1 || stored.Lease == nil {
		t.Fatalf("stored task after race = %#v", stored)
	}
}

func TestRunnerCannotHoldMultipleActiveLeases(t *testing.T) {
	service, _ := newTestService(t, 3)
	registerTestRunner(
		t,
		service,
		"runner-single",
		testDeviceKey('s'),
		[]string{"alpha"},
		[]string{"go", "codex"},
	)
	first := mustEnqueue(t, service, "alpha", "enqueue-single-1")
	second := mustEnqueue(t, service, "alpha", "enqueue-single-2")
	claim := mustClaim(t, service, "runner-single", "alpha", "claim-single-1")
	unclaimedID := first.ID
	if claim.Task.ID == first.ID {
		unclaimedID = second.ID
	} else if claim.Task.ID != second.ID {
		t.Fatalf("claim returned unknown task %s", claim.Task.ID)
	}
	_, err := service.Claim(ClaimRequest{
		RunnerID:       "runner-single",
		ProjectID:      "alpha",
		IdempotencyKey: "claim-single-2",
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("second active lease error = %v", err)
	}
	stored, err := service.GetTask(unclaimedID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != TaskQueued {
		t.Fatalf("second task status = %s, want queued", stored.Status)
	}
}

func TestThreePilotOwnersClaimOnlyTheirAssignedTasksConcurrently(t *testing.T) {
	service, _ := newTestService(t, 3)
	runnerIDs := []string{"pilot-alice", "pilot-bob", "pilot-carol"}
	taskForOwner := make(map[string]string, len(runnerIDs))
	for index, runnerID := range runnerIDs {
		registerTestRunner(
			t,
			service,
			runnerID,
			testDeviceKey(byte(index+20)),
			[]string{"pilot-project"},
			[]string{"codex", "go", RunnerLinuxCapability},
		)
		request := testEnqueueRequest(
			"pilot-project",
			"pilot-enqueue-"+runnerID,
			"execute assigned pilot task",
		)
		request.ExecutionPack.Metadata = map[string]string{
			"assignee_id": "user-" + runnerID,
		}
		task, err := service.Enqueue(request)
		if err != nil {
			t.Fatalf("Enqueue(%s): %v", runnerID, err)
		}
		taskForOwner["user-"+runnerID] = task.ID
	}

	var wait sync.WaitGroup
	var mutex sync.Mutex
	claims := make(map[string]ClaimResult, len(runnerIDs))
	for _, runnerID := range runnerIDs {
		runnerID := runnerID
		wait.Add(1)
		go func() {
			defer wait.Done()
			claim, err := service.Claim(ClaimRequest{
				RunnerID:       runnerID,
				ProjectID:      "pilot-project",
				IdempotencyKey: "pilot-claim-" + runnerID,
			})
			if err != nil {
				t.Errorf("Claim(%s): %v", runnerID, err)
				return
			}
			mutex.Lock()
			claims[runnerID] = claim
			mutex.Unlock()
		}()
	}
	wait.Wait()
	if len(claims) != len(runnerIDs) {
		t.Fatalf("successful claims = %d, want %d", len(claims), len(runnerIDs))
	}
	for _, runnerID := range runnerIDs {
		claim := claims[runnerID]
		owner := "user-" + runnerID
		if claim.Task.ID != taskForOwner[owner] {
			t.Fatalf(
				"%s claimed %s, want assigned task %s",
				runnerID,
				claim.Task.ID,
				taskForOwner[owner],
			)
		}
		if claim.Task.ExecutionPack.Metadata["assignee_id"] != owner {
			t.Fatalf("%s received foreign assignee: %#v", runnerID, claim.Task)
		}
		if _, err := service.Claim(ClaimRequest{
			RunnerID:       runnerID,
			ProjectID:      "pilot-project",
			IdempotencyKey: "pilot-second-" + runnerID,
		}); !errors.Is(err, ErrConflict) {
			t.Fatalf("%s second active claim error = %v", runnerID, err)
		}
	}
}
