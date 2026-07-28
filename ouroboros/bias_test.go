package ouroboros

import (
	"testing"
	"time"

	"github.com/smallnest/goclaw/governance"
)

func TestResolveEvaluationRecordsOutcomeOnlyAfterHumanDisposition(t *testing.T) {
	service, err := NewService(Config{Root: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	evaluation := Evaluation{
		ID:        "evaluation-disputed",
		SessionID: "session-disputed",
		SeedHash:  "missing-seed-is-allowed-for-disposition",
		TaskID:    "task-disputed",
		Mechanical: EvaluationStage{
			Name: "mechanical", Passed: true, Score: 1, Reviewer: "go-core",
		},
		Semantic: EvaluationStage{
			Name: "semantic", Passed: true, Score: 0.9, Reviewer: "semantic",
		},
		Consensus: EvaluationStage{
			Name: "consensus", Passed: false, Score: 0.8, Reviewer: "go-core-majority",
			Summary: "same-model correlation requires human disposition",
		},
		HumanDecisionRequired: true,
		CreatedAt:             time.Now().UTC(),
	}
	session := Session{
		SchemaVersion:  SchemaVersion,
		ID:             evaluation.SessionID,
		ProjectID:      "project-disputed",
		Status:         StatusBlocked,
		Evaluations:    []Evaluation{evaluation},
		BlockedReasons: []string{"latest evaluation contains correlated or disputed judgments requiring human disposition"},
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	if err := service.appendEventUnlocked(session, "test.session_created", "test", session); err != nil {
		t.Fatal(err)
	}
	review := governance.Review{
		ReviewerID:      "evidence-adjudicator",
		Role:            governance.RoleEvaluationResolve,
		Rationale:       "The blinded artifacts independently support the acceptance criteria.",
		Counterargument: "The judges share a provider and may still carry correlated blind spots.",
		EvidenceRefs:    []string{"artifact:diff", "artifact:test-log"},
		Source:          "test",
		Authenticated:   true,
		CreatedAt:       time.Now().UTC(),
	}
	resolved, err := service.ResolveEvaluation(session.ID, evaluation.ID, true, review)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Status != StatusEvaluated || len(resolved.BlockedReasons) != 0 {
		t.Fatalf("expected disputed evaluation to recover, got status=%s reasons=%v",
			resolved.Status, resolved.BlockedReasons)
	}
	if len(resolved.Outcomes) != 1 {
		t.Fatalf("expected exactly one outcome after disposition, got %d", len(resolved.Outcomes))
	}
	if outcome := resolved.Outcomes[0]; !outcome.Passed || outcome.EvaluationID != evaluation.ID {
		t.Fatalf("unexpected disposition outcome: %#v", outcome)
	}
	got := resolved.Evaluations[0]
	if !got.Passed || got.HumanDecisionRequired || got.HumanDisposition == nil ||
		!got.HumanDisposition.Accepted {
		t.Fatalf("evaluation disposition was not persisted: %#v", got)
	}
	if _, err := service.ResolveEvaluation(session.ID, evaluation.ID, true, review); err == nil {
		t.Fatal("a resolved evaluation must not be adjudicated twice")
	}
}

func TestResolveEvaluationCannotOverrideMechanicalFailure(t *testing.T) {
	service, err := NewService(Config{Root: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	evaluation := Evaluation{
		ID:                    "evaluation-mechanical-failure",
		SessionID:             "session-mechanical-failure",
		Mechanical:            EvaluationStage{Name: "mechanical", Passed: false},
		Semantic:              EvaluationStage{Name: "semantic", Passed: true},
		Consensus:             EvaluationStage{Name: "consensus", Passed: false},
		HumanDecisionRequired: true,
		CreatedAt:             time.Now().UTC(),
	}
	session := Session{
		SchemaVersion: SchemaVersion,
		ID:            evaluation.SessionID,
		ProjectID:     "project-failure",
		Status:        StatusEvaluated,
		Evaluations:   []Evaluation{evaluation},
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
	if err := service.appendEventUnlocked(session, "test.session_created", "test", session); err != nil {
		t.Fatal(err)
	}
	_, err = service.ResolveEvaluation(session.ID, evaluation.ID, true, governance.Review{
		ReviewerID: "adjudicator",
		Role:       governance.RoleEvaluationResolve,
		Rationale:  "The evidence dispute is resolved in favor of the implementation.",
		CreatedAt:  time.Now().UTC(),
	})
	if err == nil {
		t.Fatal("human disposition must not override a failed mechanical gate")
	}
}

func TestReferenceClassUsesLatestOutcomePerTask(t *testing.T) {
	service, err := NewService(Config{Root: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	session := Session{ProjectID: "project-outcomes"}
	first := service.appendOutcomeRecordUnlocked(&session, OutcomeRequest{
		Kind:   "passed",
		TaskID: "task-one",
		Reason: "initial acceptance passed",
	}, "go-core")
	second := service.appendOutcomeRecordUnlocked(&session, OutcomeRequest{
		Kind:   "rolled_back",
		TaskID: "task-one",
		Reason: "post-deployment regression required rollback",
	}, "operator")
	if second.SupersedesID != first.ID {
		t.Fatalf("expected rollback to supersede pass, got %#v", second)
	}
	stats := ReferenceClassStats{ProjectID: session.ProjectID}
	accumulateReferenceClass(&stats, session)
	finalizeReferenceClass(&stats)
	if stats.Total != 1 || stats.Passed != 0 || stats.RolledBack != 1 ||
		stats.FailureRate != 1 {
		t.Fatalf("superseded outcome was double-counted: %#v", stats)
	}
}
