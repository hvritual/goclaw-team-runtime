package harness

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/smallnest/goclaw/governance"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	service, err := NewService(Config{
		Enabled:      true,
		Root:         t.TempDir(),
		ProjectID:    "test-project",
		TraceEnabled: true,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	return service
}

func TestServiceBootstrapsSeedHarness(t *testing.T) {
	service := newTestService(t)
	active, err := service.ActiveState()
	if err != nil {
		t.Fatalf("ActiveState: %v", err)
	}
	if active.Version != "v0.1.0" {
		t.Fatalf("unexpected active version: %s", active.Version)
	}
	version, projectID, instructions, components, err := service.ActiveInstructions()
	if err != nil {
		t.Fatalf("ActiveInstructions: %v", err)
	}
	if version != active.Version || projectID != "test-project" {
		t.Fatalf("unexpected harness identity: %s %s", version, projectID)
	}
	if instructions == "" || len(components) != 1 {
		t.Fatalf("seed harness instructions were not loaded")
	}
}

func TestExperimentValidateApprovePromoteRollback(t *testing.T) {
	service := newTestService(t)
	exp, err := service.CreateExperiment("", "v0.2.0", ChangeManifest{
		TargetComponents: []string{"components/instructions.md"},
		RootCause:        "missing completion evidence",
		ChangeSummary:    "require evidence before completion",
	})
	if err != nil {
		t.Fatalf("CreateExperiment: %v", err)
	}
	report, err := service.ValidateExperiment(context.Background(), exp.ID, false)
	if err != nil {
		t.Fatalf("ValidateExperiment: %v", err)
	}
	if !report.Accepted {
		t.Fatalf("seed report should pass: %v", report.Rejection)
	}
	exp, err = service.ApproveExperiment(exp.ID, "human", "validation evidence accepted")
	if err != nil {
		t.Fatalf("ApproveExperiment: %v", err)
	}
	if exp.Status != ExperimentHumanApproved {
		t.Fatalf("unexpected approval status: %s", exp.Status)
	}
	active, err := service.PromoteExperiment(exp.ID, "human")
	if err != nil {
		t.Fatalf("PromoteExperiment: %v", err)
	}
	if active.Version != "v0.2.0" || active.PreviousVersion != "v0.1.0" {
		t.Fatalf("unexpected promoted state: %+v", active)
	}
	rolledBack, err := service.Rollback("human")
	if err != nil {
		t.Fatalf("Rollback: %v", err)
	}
	if rolledBack.Version != "v0.1.0" {
		t.Fatalf("unexpected rollback state: %+v", rolledBack)
	}
}

func TestHarnessPromotionRequiresSeparateHuman(t *testing.T) {
	service := newTestService(t)
	service.SetGovernancePolicy(governance.Config{
		Enabled:                           true,
		RequireRationale:                  true,
		MinRationaleRunes:                 1,
		HarnessApprovalQuorum:             1,
		ForbidHarnessPromoterFromApproval: true,
	})
	exp, err := service.CreateExperiment("", "v0.2.0", ChangeManifest{
		TargetComponents: []string{"components/instructions.md"},
		RootCause:        "missing completion evidence",
		ChangeSummary:    "require evidence before completion",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ValidateExperiment(context.Background(), exp.ID, false); err != nil {
		t.Fatal(err)
	}
	exp, err = service.ApproveExperimentWithReview(exp.ID, governance.Review{
		ReviewerID: "approver",
		Role:       governance.RoleHarnessApprove,
		Rationale:  "validated",
		Source:     "test",
		CreatedAt:  time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if exp.Status != ExperimentHumanApproved {
		t.Fatalf("expected approved candidate, got %s", exp.Status)
	}
	_, err = service.PromoteExperimentWithReview(exp.ID, governance.Review{
		ReviewerID: "approver",
		Role:       governance.RoleHarnessPromote,
		Rationale:  "promote",
		Source:     "test",
		CreatedAt:  time.Now().UTC(),
	})
	if err == nil || !strings.Contains(err.Error(), "also approved") {
		t.Fatalf("same-person promotion must be rejected, got %v", err)
	}
	active, err := service.PromoteExperimentWithReview(exp.ID, governance.Review{
		ReviewerID: "promoter",
		Role:       governance.RoleHarnessPromote,
		Rationale:  "promote",
		Source:     "test",
		CreatedAt:  time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if active.ActivatedBy != "promoter" {
		t.Fatalf("unexpected promoter: %#v", active)
	}
	policy := service.GovernancePolicy()
	policy.RequireAuthenticatedReviewers = true
	service.SetGovernancePolicy(policy)
	if _, err := service.Rollback("untrusted"); err == nil {
		t.Fatal("authenticated governance must reject the actor-only rollback wrapper")
	}
	rolledBack, err := service.RollbackWithReview(governance.Review{
		ReviewerID:      "rollback-operator",
		Role:            governance.RoleHarnessRollback,
		Rationale:       "A post-promotion regression requires immediate rollback.",
		Counterargument: "Rollback temporarily gives up the intended Harness improvement.",
		Source:          "test",
		Authenticated:   true,
		CreatedAt:       time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if rolledBack.Version != "v0.1.0" || rolledBack.Decision == nil {
		t.Fatalf("rollback decision was not persisted: %#v", rolledBack)
	}
}

func TestGoldenFailureRejectsCandidate(t *testing.T) {
	service := newTestService(t)
	fixturePath := filepath.Join(service.evalsDir(), "fixtures", "golden-knowledge-authority.json")
	trace := Trace{
		SchemaVersion: SchemaVersion,
		ID:            "bad-fixture",
		ProjectID:     "wrong-project",
		Output:        "I changed the ADR directly.",
	}
	if err := writeJSONAtomic(fixturePath, trace, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	exp, err := service.CreateExperiment("", "v0.2.0", ChangeManifest{
		TargetComponents: []string{"components/instructions.md"},
		RootCause:        "unsafe knowledge mutation",
		ChangeSummary:    "unsafe candidate",
	})
	if err != nil {
		t.Fatalf("CreateExperiment: %v", err)
	}
	report, err := service.ValidateExperiment(context.Background(), exp.ID, false)
	if err != nil {
		t.Fatalf("ValidateExperiment: %v", err)
	}
	if report.Accepted {
		t.Fatalf("critical golden failure must reject candidate")
	}
	if report.Scores[SplitGolden].Critical == 0 {
		t.Fatalf("critical golden failure was not counted")
	}
}

func TestExperimentRejectsProtectedTarget(t *testing.T) {
	service := newTestService(t)
	_, err := service.CreateExperiment("", "v0.2.0", ChangeManifest{
		TargetComponents: []string{"evals/golden/case.yaml"},
		RootCause:        "try to change the judge",
		ChangeSummary:    "unsafe grader mutation",
	})
	if err == nil || !strings.Contains(err.Error(), "protected path") {
		t.Fatalf("expected protected target rejection, got %v", err)
	}
}

func TestValidationRejectsBaselineRegression(t *testing.T) {
	service := newTestService(t)
	workDir := t.TempDir()
	exitCode := 0
	evalCase := EvalCase{
		SchemaVersion: SchemaVersion,
		ID:            "candidate-regression",
		Description:   "Candidate instructions must preserve the baseline marker.",
		Split:         SplitHoldout,
		Runner: RunnerSpec{
			Command:    []string{"sh", "-c", `if grep -q REGRESSION "$GOCLAW_HARNESS_CANDIDATE/components/instructions.md"; then echo broken; else echo stable; fi`},
			WorkingDir: workDir,
		},
		Expected: ExpectedBehavior{
			ExitCode:       &exitCode,
			OutputContains: []string{"stable"},
		},
	}
	if err := writeYAMLAtomic(filepath.Join(service.evalsDir(), "cases", evalCase.ID+".yaml"), evalCase); err != nil {
		t.Fatalf("write eval: %v", err)
	}
	exp, err := service.CreateExperiment("", "v0.2.0", ChangeManifest{
		TargetComponents: []string{"components/instructions.md"},
		RootCause:        "candidate edit",
		ChangeSummary:    "candidate causes a regression",
	})
	if err != nil {
		t.Fatalf("CreateExperiment: %v", err)
	}
	instructions := filepath.Join(exp.CandidatePath, "components", "instructions.md")
	file, err := os.OpenFile(instructions, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString("\nREGRESSION\n"); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	report, err := service.ValidateExperiment(context.Background(), exp.ID, true)
	if err != nil {
		t.Fatalf("ValidateExperiment: %v", err)
	}
	if report.Accepted || len(report.Regressions) != 1 {
		t.Fatalf("expected one baseline regression, got accepted=%v regressions=%+v", report.Accepted, report.Regressions)
	}
}

func TestTracePersistenceAndFeedback(t *testing.T) {
	service := newTestService(t)
	start := time.Now().UTC()
	trace := Trace{
		ID:         "trace-1",
		ProjectID:  "test-project",
		TopicID:    "topic",
		Status:     "completed",
		Input:      "hello",
		Output:     "world",
		StartedAt:  start,
		FinishedAt: start.Add(time.Second),
	}
	if err := service.RecordTrace(trace); err != nil {
		t.Fatalf("RecordTrace: %v", err)
	}
	traces, err := service.ListTraces("test-project", 10)
	if err != nil {
		t.Fatalf("ListTraces: %v", err)
	}
	if len(traces) != 1 || traces[0].ID != "trace-1" {
		t.Fatalf("unexpected traces: %+v", traces)
	}
	if err := service.AddHumanFeedback("trace-1", HumanFeedback{Rating: "corrected"}); err != nil {
		t.Fatalf("AddHumanFeedback: %v", err)
	}
	if _, err := os.Stat(filepath.Join(service.tracesDir(), "feedback.jsonl")); err != nil {
		t.Fatalf("feedback file missing: %v", err)
	}
}

func TestResolveProjectRoutePrecedence(t *testing.T) {
	service, err := NewService(Config{
		Root:      t.TempDir(),
		ProjectID: "default",
		Routes: map[string]string{
			"feishu":                "channel-project",
			"feishu:chat-1":         "chat-project",
			"feishu:account:chat-1": "specific-project",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := service.ResolveProject("feishu", "account", "chat-1"); got != "specific-project" {
		t.Fatalf("unexpected specific route: %s", got)
	}
	if got := service.ResolveProject("feishu", "other", "chat-1"); got != "chat-project" {
		t.Fatalf("unexpected chat route: %s", got)
	}
	if got := service.ResolveProject("feishu", "other", "chat-2"); got != "channel-project" {
		t.Fatalf("unexpected channel route: %s", got)
	}
}
