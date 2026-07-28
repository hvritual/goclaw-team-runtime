package ouroboros

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	dev "github.com/smallnest/goclaw/orchestratorlite"
)

type queuedModel struct {
	mu        sync.Mutex
	responses []string
	calls     int
}

func (m *queuedModel) Generate(_ context.Context, request ModelRequest) (ModelResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.responses) == 0 {
		return ModelResponse{}, os.ErrNotExist
	}
	content := m.responses[0]
	m.responses = m.responses[1:]
	m.calls++
	return ModelResponse{
		Content: content,
		Model:   valueOr(request.Model, "test-model"),
		Usage:   ModelUsage{InputTokens: 10, OutputTokens: 10, TotalTokens: 20, Calls: 1},
	}, nil
}

func (m *queuedModel) add(values ...string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses = append(m.responses, values...)
}

func TestServiceLifecycleKeepsEvolutionCandidateInactiveUntilApproval(t *testing.T) {
	ctx := context.Background()
	model := &queuedModel{responses: []string{
		mustModelJSON(t, readyInterview()),
		mustModelJSON(t, readyInterview()),
		mustModelJSON(t, validSeedDraft()),
	}}
	service, err := NewService(Config{
		Root:                 t.TempDir(),
		Model:                "codex/default",
		RequiredReadyStreak:  2,
		AssessmentReviewers:  1,
		ConsensusReviewers:   3,
		EvaluationModels:     []string{"semantic-model", "review-model-a", "review-model-b", "review-model-c"},
		AmbiguityThreshold:   0.20,
		ConvergenceThreshold: 0.95,
	})
	if err != nil {
		t.Fatal(err)
	}
	service.SetModel(model)
	repoPath := t.TempDir()

	session, err := service.Start(ctx, StartRequest{
		ID:         "ouro-test",
		ProjectID:  "project-test",
		Title:      "Implement governed loop",
		RepoPath:   repoPath,
		RawRequest: "Implement a governed specification loop with deterministic tests.",
		Brownfield: true,
		CreatedBy:  "tester",
	})
	if err != nil {
		t.Fatal(err)
	}
	if session.Status != StatusInterviewing ||
		session.Rounds[len(session.Rounds)-1].Assessment.ReadyStreak != 1 {
		t.Fatalf("expected first readiness pass, got status=%s", session.Status)
	}

	session, err = service.Reassess(ctx, session.ID, "tester")
	if err != nil {
		t.Fatal(err)
	}
	if session.Status != StatusSeedReady {
		t.Fatalf("expected seed_ready, got %s", session.Status)
	}

	session, err = service.Crystallize(ctx, session.ID, "tester")
	if err != nil {
		t.Fatal(err)
	}
	if session.Status != StatusAwaitingSeedApproval || session.PendingSeedHash == "" {
		t.Fatalf("expected pending immutable Seed, got %#v", session)
	}
	firstHash := session.PendingSeedHash
	if _, err := service.BuildTaskRequest(session.ID, "tester"); err == nil {
		t.Fatal("unapproved Seed must not compile")
	}
	if _, err := service.ApproveSeed(session.ID, "human-reviewer", ""); err == nil {
		t.Fatal("Seed approval must include an auditable rationale")
	}

	session, err = service.ApproveSeed(session.ID, "human-reviewer", "scope and checks accepted")
	if err != nil {
		t.Fatal(err)
	}
	if session.ActiveSeedHash != firstHash || session.Status != StatusApproved {
		t.Fatalf("approved Seed was not activated: %#v", session)
	}

	development, err := dev.NewService(dev.Config{
		Root:     t.TempDir(),
		RepoPath: repoPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	task, err := service.CompileTask(session.ID, "human-reviewer", development)
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != dev.TaskReviewPending || len(task.Reviews) != len(dev.RequiredReviewKinds) {
		t.Fatalf("compiled task must retain four human reviews: %#v", task)
	}

	task.LastEvidence = "evidence.json"
	task.LastGate = &dev.DoneGateResult{
		Passed:         true,
		Verdict:        "pass",
		EvidencePath:   task.LastEvidence,
		EvidenceSHA256: strings.Repeat("a", 64),
		EvaluatedAt:    time.Now().UTC(),
		EvaluatedBy:    "go-core",
	}
	evidence := dev.EvidencePackage{
		SchemaVersion: 1,
		TaskID:        task.ID,
		RunID:         "run-test",
		TaskRevision:  task.Compile.Revision,
		Policy: dev.PolicyResult{
			Passed:       true,
			ChangedFiles: []string{"ouroboros/service.go"},
		},
		Verification: []dev.VerificationResult{
			{Name: "unit tests", Argv: []string{"go", "test", "./..."}, Passed: true},
		},
		Falsifiers: []dev.FalsifierResult{
			{
				CriterionID: "ac-1",
				Checked:     true,
				Triggered:   false,
				Reason:      "deterministic verification passed",
			},
		},
		Predictions: []dev.PredictionCheck{
			{
				PredictionID: "prediction-1",
				Horizon:      "before acceptance",
				Due:          true,
				Checked:      true,
				Satisfied:    true,
			},
		},
		KillChecks: []dev.KillConditionCheck{
			{
				ConditionID: "kill-1",
				Metric:      "changed_files",
				Observed:    1,
				Threshold:   20,
				Evaluated:   true,
			},
		},
		Review:   dev.IndependentReview{Passed: true, Summary: "evidence is sufficient"},
		DiffPath: "diff.patch",
	}
	passReview := mustModelJSON(t, evaluationModelOutput{
		Passed:  true,
		Score:   0.98,
		Summary: "all immutable criteria have concrete evidence",
	})
	model.add(passReview, passReview, passReview, passReview)
	session, err = service.EvaluateTask(ctx, session.ID, "evaluator", task, evidence, "diff --git a/a b/a")
	if err != nil {
		t.Fatal(err)
	}
	if session.Status != StatusEvaluated || !session.Evaluations[len(session.Evaluations)-1].Passed {
		t.Fatalf("expected passed evaluation, got %#v", session.Evaluations)
	}

	successor := validSeedDraft()
	successor.Ontology.Fields = append(successor.Ontology.Fields, OntologyField{
		Name: "State", Type: "string", Description: "governed lifecycle state", Required: true,
	})
	model.add(mustModelJSON(t, evolutionModelOutput{
		Action:  "continue",
		Reasons: []string{"make lifecycle state explicit"},
		Seed:    successor,
	}))
	session, err = service.ProposeEvolution(ctx, session.ID, "reflector")
	if err != nil {
		t.Fatal(err)
	}
	if session.Status != StatusEvolutionPending || session.PendingEvolution == nil {
		t.Fatalf("expected a pending successor, got %#v", session)
	}
	candidateHash := session.PendingEvolution.CandidateSeedHash
	if session.ActiveSeedHash != firstHash || candidateHash == firstHash {
		t.Fatal("candidate must remain inactive until explicit human approval")
	}
	if _, err := service.ApproveEvolution(session.ID, "human-reviewer", ""); err == nil {
		t.Fatal("evolution approval must include an auditable rationale")
	}

	session, err = service.ApproveEvolution(session.ID, "human-reviewer", "successor is justified by evidence")
	if err != nil {
		t.Fatal(err)
	}
	if session.Status != StatusApproved || session.ActiveSeedHash != candidateHash {
		t.Fatalf("approved successor was not activated: %#v", session)
	}

	events, err := service.ListEvents(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) < 9 || events[len(events)-1].Type != "evolution.approved" {
		t.Fatalf("unexpected event ledger: %#v", events)
	}
}

func TestModelJSONGetsOneBoundedRepairAttempt(t *testing.T) {
	model := &queuedModel{responses: []string{
		"not-json",
		mustModelJSON(t, readyInterview()),
	}}
	service, err := NewService(Config{
		Root:                t.TempDir(),
		RequiredReadyStreak: 1,
		AssessmentReviewers: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	service.SetModel(model)
	session, err := service.Start(context.Background(), StartRequest{
		ID:         "ouro-repair",
		ProjectID:  "project-test",
		RepoPath:   t.TempDir(),
		RawRequest: "Create a deterministic test.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if session.Status != StatusSeedReady || model.calls != 2 || session.ModelUsage.Calls != 2 {
		t.Fatalf("expected one repair and complete usage accounting: session=%#v calls=%d", session, model.calls)
	}
}

func TestStartRejectsOversizedContextBeforePersistingSession(t *testing.T) {
	model := &queuedModel{responses: []string{mustModelJSON(t, readyInterview())}}
	service, err := NewService(Config{
		Root:            t.TempDir(),
		MaxContextBytes: 4096,
	})
	if err != nil {
		t.Fatal(err)
	}
	service.SetModel(model)

	_, err = service.Start(context.Background(), StartRequest{
		ID:         "ouro-oversized",
		ProjectID:  "project-test",
		RepoPath:   t.TempDir(),
		RawRequest: strings.Repeat("bounded-input-", 1024),
		CreatedBy:  "alice",
	})
	if err == nil || !strings.Contains(err.Error(), "maximum is 4096") {
		t.Fatalf("expected a bounded-context error, got %v", err)
	}
	if model.calls != 0 {
		t.Fatalf("oversized context must be rejected before the model call, got %d calls", model.calls)
	}
	sessions, listErr := service.ListSessions("")
	if listErr != nil {
		t.Fatal(listErr)
	}
	if len(sessions) != 0 {
		t.Fatalf("oversized context must not leave a partial session, got %#v", sessions)
	}
}

func TestEventLedgerRejectsTampering(t *testing.T) {
	model := &queuedModel{responses: []string{mustModelJSON(t, readyInterview())}}
	service, err := NewService(Config{
		Root:                t.TempDir(),
		RequiredReadyStreak: 1,
		AssessmentReviewers: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	service.SetModel(model)
	session, err := service.Start(context.Background(), StartRequest{
		ID:         "ouro-tamper",
		ProjectID:  "project-test",
		RepoPath:   t.TempDir(),
		RawRequest: "Test the ledger.",
		CreatedBy:  "alice",
	})
	if err != nil {
		t.Fatal(err)
	}
	path := service.eventsPath(session.ID)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content = []byte(strings.Replace(string(content), `"actor":"alice"`, `"actor":"mallory"`, 1))
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ListEvents(session.ID); err == nil ||
		!strings.Contains(err.Error(), "hash") {
		t.Fatalf("expected hash-chain tamper error, got %v", err)
	}
}

func readyInterview() interviewModelOutput {
	return interviewModelOutput{
		Summary:    "goal, constraints, success tests, and repository context are explicit",
		Goal:       modelDimension{Clarity: 0.95, Justification: "specific outcome"},
		Constraint: modelDimension{Clarity: 0.95, Justification: "explicit boundaries"},
		Success:    modelDimension{Clarity: 0.95, Justification: "deterministic commands"},
		Context:    modelDimension{Clarity: 0.95, Justification: "repository identified"},
		Questions:  []Question{},
		Unresolved: []string{},
	}
}

func validSeedDraft() seedModelOutput {
	return seedModelOutput{
		Title:          "Implement governed loop",
		Goal:           "Implement the approved specification loop with deterministic tests.",
		TaskType:       "code",
		ContextSummary: "Existing Go repository.",
		Constraints:    []string{"Keep Go Core authoritative."},
		NonGoals:       []string{"Do not deploy remotely."},
		AcceptanceCriteria: []AcceptanceCriterion{
			{
				ID:            "ac-1",
				Description:   "All deterministic Go tests pass.",
				VerifyCommand: []string{"go", "test", "./ouroboros"},
			},
		},
		Ontology: Ontology{
			Name:        "Task",
			Description: "One governed development task.",
			Fields: []OntologyField{
				{Name: "ID", Type: "string", Description: "stable identifier", Required: true},
			},
		},
		EvaluationPrinciples: []EvaluationPrinciple{
			{Name: "evidence", Description: "Require deterministic evidence."},
		},
		ExitConditions: []ExitCondition{
			{Name: "verified", Description: "All criteria pass.", Criteria: "ac-1"},
		},
		Plan: SeedPlan{
			Summary: "Implement and verify.",
			Milestones: []SeedMilestone{
				{
					ID:    "m1",
					Title: "Implementation",
					WorkItems: []SeedWorkItem{
						{
							ID:           "w1",
							Title:        "Implement",
							Instructions: "Implement the approved scope and add tests.",
							CriteriaIDs:  []string{"ac-1"},
						},
					},
				},
			},
		},
		Scope: SeedScope{
			AllowedPaths:    []string{"ouroboros/**"},
			MaxChangedFiles: 20,
			MaxChangedLines: 1000,
		},
		Risk: SeedRisk{
			Level:    "medium",
			Rollback: "Discard the isolated worktree.",
		},
		Cost: SeedCost{MaxRepairAttempts: 2},
		Alternatives: []Alternative{
			{ID: "alt-build", Title: "Build governed loop", Summary: "Implement the requested loop.", Selected: true},
			{ID: "alt-status-quo", Title: "Keep current system", Summary: "Delay implementation.", Selected: false},
		},
		Falsifiers: []Falsifier{
			{CriterionID: "ac-1", Condition: "The deterministic Go tests fail.", EvidenceRequired: "go test output"},
		},
		CostOfInaction: []string{"Development tasks remain weakly governed."},
		KillConditions: []KillCondition{
			{
				ID:        "kill-1",
				Condition: "The implementation exceeds the approved scope.",
				Metric:    "changed_files",
				Threshold: "20",
				Action:    "stop",
			},
		},
		PreMortem: []string{"Review evidence may be incomplete."},
		ReferenceClass: ReferenceClassForecast{
			Basis:           "No comparable completed sessions are available.",
			SampleSize:      0,
			BaseFailureRate: 0,
		},
		Predictions: []Prediction{
			{
				ID:              "prediction-1",
				Claim:           "The scoped implementation is testable.",
				ExpectedOutcome: "All deterministic tests pass before acceptance.",
				Horizon:         "before acceptance",
				Confidence:      0.8,
			},
		},
	}
}

func mustModelJSON(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
