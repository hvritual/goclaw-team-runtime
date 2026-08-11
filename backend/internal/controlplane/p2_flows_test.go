package controlplane

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestRequirementFourReviewFreezeAndChangeIntent(t *testing.T) {
	kernel, repository := openTestKernel(t, filepath.Join(t.TempDir(), "flows.db"))
	defer repository.Close()
	flows, err := NewP2Flows(kernel)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	creator := Actor{ID: "creator-1", WorkspaceID: "workspace-1", Kind: ActorHuman}
	runner := Actor{ID: "agent-1", WorkspaceID: "workspace-1", Kind: ActorAgent}
	checker := Actor{ID: "checker-1", WorkspaceID: "workspace-1", Kind: ActorHuman}
	acceptor := Actor{ID: "acceptor-1", WorkspaceID: "workspace-1", Kind: ActorHuman}
	if _, err := flows.StartRequirement(ctx, creator, "command-1", "project-1", 0, "requirement-1", "Need a governed delivery flow"); err != nil {
		t.Fatal(err)
	}
	if _, err := flows.FinalizeIntent(ctx, creator, "command-2", "project-1", 1, "requirement-1", "Deliver safely"); err != nil {
		t.Fatal(err)
	}
	if _, err := flows.ProposeSolution(ctx, creator, "command-3", "project-1", 2, "requirement-1", "ADR-001 typed kernel"); err != nil {
		t.Fatal(err)
	}
	evidence := EvidenceRef{ID: "evidence-1", SubjectID: "requirement-1", Kind: "review-pack", URI: "artifact://review/pack.json", SHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Size: 100, MediaType: "application/json", ProducedBy: runner.ID, Sanitized: true}
	if _, err := kernel.AttachEvidence(ctx, runner, "command-4", "project-1", 3, evidence); err != nil {
		t.Fatal(err)
	}
	head := int64(4)
	for index, policy := range RequirementReviewPolicies {
		check := CheckResult{ID: "check-" + policy, PolicyID: policy, SubjectID: "requirement-1", Revision: 3, Outcome: CheckPassed, EvidenceIDs: []string{evidence.ID}, CheckerID: checker.ID, Deterministic: true}
		if _, err := kernel.RecordCheck(ctx, checker, "review-command-"+policy, "project-1", head, check); err != nil {
			t.Fatalf("review %d: %v", index, err)
		}
		head++
	}
	if _, err := flows.FreezeRequirement(ctx, creator, "command-creator", "project-1", "requirement-1", 3, head); !errors.Is(err, ErrDenied) {
		t.Fatalf("creator freeze error = %v", err)
	}
	if _, err := flows.FreezeRequirement(ctx, acceptor, "command-9", "project-1", "requirement-1", 3, head); err != nil {
		t.Fatal(err)
	}
	projection, err := kernel.Replay(ctx, creator.WorkspaceID, "project-1")
	if err != nil {
		t.Fatal(err)
	}
	if projection.Nodes["requirement-1"].State != "done" {
		t.Fatalf("requirement = %#v", projection.Nodes["requirement-1"])
	}
	if _, err := flows.ChangeIntent(ctx, creator, "command-10", "project-1", head+1, "requirement-1", "Material scope changed"); err != nil {
		t.Fatal(err)
	}
	projection, err = kernel.Replay(ctx, creator.WorkspaceID, "project-1")
	if err != nil {
		t.Fatal(err)
	}
	if projection.Nodes["requirement-1"].Revision != 4 || projection.Nodes["requirement-1"].State != "clarifying" {
		t.Fatalf("changed requirement = %#v", projection.Nodes["requirement-1"])
	}
}

func TestQualityKnowledgeAndRunBoundaries(t *testing.T) {
	kernel, repository := openTestKernel(t, filepath.Join(t.TempDir(), "flows.db"))
	defer repository.Close()
	flows, _ := NewP2Flows(kernel)
	ctx := context.Background()
	creator := Actor{ID: "creator-1", WorkspaceID: "workspace-1", Kind: ActorHuman}
	runner := Actor{ID: "agent-1", WorkspaceID: "workspace-1", Kind: ActorAgent}
	if _, err := flows.CreateDefect(ctx, creator, "command-1", "project-1", 0, "defect-1", QualityData{Summary: "failure", Severity: "high"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("defect error = %v", err)
	}
	if _, err := flows.CreateRisk(ctx, creator, "command-2", "project-1", 0, "risk-1", QualityData{Summary: "capacity", Probability: 4, Impact: 5, ResponsePlan: "scale", ReviewDueAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if _, err := flows.QueueRun(ctx, creator, "command-3", "project-1", 1, "run-1", "worktree://task-1", []string{"raw-token"}, 2); !errors.Is(err, ErrInvalid) {
		t.Fatalf("raw secret error = %v", err)
	}
	if _, err := flows.QueueRun(ctx, creator, "command-4", "project-1", 1, "run-1", "worktree://task-1", []string{"secret://github/token"}, 2); err != nil {
		t.Fatal(err)
	}
	if _, err := flows.ClaimRun(ctx, runner, "command-5", "project-1", 2, "run-1", time.Minute); err != nil {
		t.Fatal(err)
	}
	if _, err := flows.ClaimRun(ctx, Actor{ID: "agent-2", WorkspaceID: "workspace-1", Kind: ActorAgent}, "command-6", "project-1", 3, "run-1", time.Minute); !errors.Is(err, ErrConflict) {
		t.Fatalf("double claim error = %v", err)
	}
	runEvidence := EvidenceRef{ID: "run-evidence-1", SubjectID: "run-1", Kind: "execution-log", URI: "artifact://run-1/log.json", SHA256: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", Size: 10, MediaType: "application/json", ProducedBy: runner.ID, Sanitized: true}
	if _, err := kernel.AttachEvidence(ctx, runner, "command-7", "project-1", 3, runEvidence); err != nil {
		t.Fatal(err)
	}
	if _, err := flows.CompleteRun(ctx, runner, "command-8", "project-1", 4, "run-1"); err != nil {
		t.Fatal(err)
	}
	projection, err := kernel.Replay(ctx, creator.WorkspaceID, "project-1")
	if err != nil {
		t.Fatal(err)
	}
	if projection.Nodes["run-1"].State != "validation" {
		t.Fatalf("run = %#v", projection.Nodes["run-1"])
	}
}
