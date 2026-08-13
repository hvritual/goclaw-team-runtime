package controlplane

import (
	"context"
	"encoding/json"
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
	if _, err := flows.FinalizeIntent(ctx, creator, "command-too-early", "project-1", 1, "requirement-1", "Deliver safely"); !errors.Is(err, ErrInvariant) {
		t.Fatalf("intent before context error = %v", err)
	}
	if _, err := flows.StartContextDiscovery(ctx, creator, "command-2", "project-1", 1, "requirement-1", "Resolve delivery constraints", 4); err != nil {
		t.Fatal(err)
	}
	iteration := ContextIterationData{Needs: []ContextNeed{{ID: "need-scope", Description: "Delivery scope is explicit", Required: true, Status: ContextNeedResolved, Resolution: "Governed control-plane delivery", SourceRefs: []string{"repo://backend/internal/controlplane"}}}, Summary: "Required scope context resolved"}
	if _, err := flows.IterateContextDiscovery(ctx, creator, "command-3", "project-1", 2, "requirement-1", iteration); err != nil {
		t.Fatal(err)
	}
	if _, err := flows.FinalizeIntent(ctx, creator, "command-4", "project-1", 3, "requirement-1", "Deliver safely"); err != nil {
		t.Fatal(err)
	}
	if _, err := flows.ProposeSolution(ctx, creator, "command-5", "project-1", 4, "requirement-1", "ADR-001 typed kernel"); err != nil {
		t.Fatal(err)
	}
	evidence := EvidenceRef{ID: "evidence-1", SubjectID: "requirement-1", Kind: "review-pack", URI: "artifact://review/11111111-1111-4111-8111-111111111111", SHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Size: 100, MediaType: "application/json", ProducedBy: runner.ID, Sanitized: true}
	if _, err := kernel.AttachEvidence(ctx, runner, "command-6", "project-1", 5, evidence); err != nil {
		t.Fatal(err)
	}
	head := int64(6)
	for index, policy := range RequirementReviewPolicies {
		check := CheckResult{ID: "check-" + policy, PolicyID: policy, SubjectID: "requirement-1", Revision: 5, Outcome: CheckPassed, EvidenceIDs: []string{evidence.ID}, CheckerID: checker.ID, Deterministic: true}
		if _, err := kernel.RecordCheck(ctx, checker, "review-command-"+policy, "project-1", head, check); err != nil {
			t.Fatalf("review %d: %v", index, err)
		}
		head++
	}
	if _, err := flows.FreezeRequirement(ctx, creator, "command-creator", "project-1", "requirement-1", 5, head); !errors.Is(err, ErrDenied) {
		t.Fatalf("creator freeze error = %v", err)
	}
	if _, err := flows.FreezeRequirement(ctx, acceptor, "command-11", "project-1", "requirement-1", 5, head); err != nil {
		t.Fatal(err)
	}
	projection, err := kernel.Replay(ctx, creator.WorkspaceID, "project-1")
	if err != nil {
		t.Fatal(err)
	}
	if projection.Nodes["requirement-1"].State != "done" {
		t.Fatalf("requirement = %#v", projection.Nodes["requirement-1"])
	}
	if _, err := flows.ChangeIntent(ctx, creator, "command-12", "project-1", head+1, "requirement-1", "Material scope changed"); err != nil {
		t.Fatal(err)
	}
	projection, err = kernel.Replay(ctx, creator.WorkspaceID, "project-1")
	if err != nil {
		t.Fatal(err)
	}
	node := projection.Nodes["requirement-1"]
	if node.Revision != 6 || node.State != "clarifying" {
		t.Fatalf("changed requirement = %#v", node)
	}
	var changed RequirementData
	if err := json.Unmarshal(node.Data, &changed); err != nil {
		t.Fatal(err)
	}
	if changed.Context == nil || changed.Context.State != ContextStateDiscovering || changed.Context.Iteration != 0 || changed.Context.Needs[0].Status != ContextNeedOpen || changed.Context.Needs[0].Resolution != "" {
		t.Fatalf("changed context = %#v", changed.Context)
	}
	if _, err := flows.FinalizeIntent(ctx, creator, "command-stale-context", "project-1", head+2, "requirement-1", "Reuse stale context"); !errors.Is(err, ErrInvariant) {
		t.Fatalf("stale context intent error = %v", err)
	}
}

func TestContextDiscoveryHumanRequiredReadyAndExhausted(t *testing.T) {
	t.Run("human answer leads to ready", func(t *testing.T) {
		kernel, repository := openTestKernel(t, filepath.Join(t.TempDir(), "human.db"))
		defer repository.Close()
		flows, _ := NewP2Flows(kernel)
		ctx := context.Background()
		actor := Actor{ID: "creator-1", WorkspaceID: "workspace-1", Kind: ActorHuman}
		if _, err := flows.StartRequirement(ctx, actor, "command-1", "project-1", 0, "requirement-1", "Need deployment behavior"); err != nil {
			t.Fatal(err)
		}
		if _, err := flows.StartContextDiscovery(ctx, actor, "command-2", "project-1", 1, "requirement-1", "Discover deployment constraints", 2); err != nil {
			t.Fatal(err)
		}
		first := ContextIterationData{
			Needs: []ContextNeed{{ID: "need-region", Description: "Deployment region is known", Required: true, Status: ContextNeedOpen}},
			Questions: []ContextQuestion{{ID: "question-region", Question: "Which deployment region is required?", Required: true, Status: ContextQuestionOpen}},
			Summary: "Region requires a human decision",
		}
		if _, err := flows.IterateContextDiscovery(ctx, actor, "command-3", "project-1", 2, "requirement-1", first); err != nil {
			t.Fatal(err)
		}
		projection, err := kernel.Replay(ctx, actor.WorkspaceID, "project-1")
		if err != nil {
			t.Fatal(err)
		}
		var data RequirementData
		if err := json.Unmarshal(projection.Nodes["requirement-1"].Data, &data); err != nil {
			t.Fatal(err)
		}
		if data.Context == nil || data.Context.State != ContextStateHumanRequired || data.Context.Iteration != 1 {
			t.Fatalf("human-required context = %#v", data.Context)
		}
		second := ContextIterationData{
			Needs: []ContextNeed{{ID: "need-region", Description: "Deployment region is known", Required: true, Status: ContextNeedResolved, Resolution: "eu-central", SourceRefs: []string{"decision://deployment-region"}}},
			Questions: []ContextQuestion{{ID: "question-region", Question: "Which deployment region is required?", Required: true, Status: ContextQuestionAnswered, Answer: "eu-central"}},
			Summary: "Human decision resolved the deployment region",
		}
		if _, err := flows.IterateContextDiscovery(ctx, actor, "command-4", "project-1", 3, "requirement-1", second); err != nil {
			t.Fatal(err)
		}
		projection, err = kernel.Replay(ctx, actor.WorkspaceID, "project-1")
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(projection.Nodes["requirement-1"].Data, &data); err != nil {
			t.Fatal(err)
		}
		if data.Context.State != ContextStateReady || data.Context.Iteration != 2 {
			t.Fatalf("ready context = %#v", data.Context)
		}
		if _, err := flows.FinalizeIntent(ctx, actor, "command-5", "project-1", 4, "requirement-1", "Deploy in eu-central"); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("unresolved need exhausts bounded loop", func(t *testing.T) {
		kernel, repository := openTestKernel(t, filepath.Join(t.TempDir(), "exhausted.db"))
		defer repository.Close()
		flows, _ := NewP2Flows(kernel)
		ctx := context.Background()
		actor := Actor{ID: "creator-1", WorkspaceID: "workspace-1", Kind: ActorHuman}
		if _, err := flows.StartRequirement(ctx, actor, "command-1", "project-1", 0, "requirement-1", "Need an unknown dependency"); err != nil {
			t.Fatal(err)
		}
		if _, err := flows.StartContextDiscovery(ctx, actor, "command-2", "project-1", 1, "requirement-1", "Resolve dependency", 1); err != nil {
			t.Fatal(err)
		}
		iteration := ContextIterationData{Needs: []ContextNeed{{ID: "need-contract", Description: "Dependency contract is known", Required: true, Status: ContextNeedBlocked}}}
		if _, err := flows.IterateContextDiscovery(ctx, actor, "command-3", "project-1", 2, "requirement-1", iteration); err != nil {
			t.Fatal(err)
		}
		projection, err := kernel.Replay(ctx, actor.WorkspaceID, "project-1")
		if err != nil {
			t.Fatal(err)
		}
		var data RequirementData
		if err := json.Unmarshal(projection.Nodes["requirement-1"].Data, &data); err != nil {
			t.Fatal(err)
		}
		if data.Context == nil || data.Context.State != ContextStateExhausted {
			t.Fatalf("exhausted context = %#v", data.Context)
		}
		if _, err := flows.FinalizeIntent(ctx, actor, "command-4", "project-1", 3, "requirement-1", "Guess the dependency"); !errors.Is(err, ErrInvariant) {
			t.Fatalf("intent after exhausted context error = %v", err)
		}
		if _, err := flows.IterateContextDiscovery(ctx, actor, "command-5", "project-1", 3, "requirement-1", iteration); !errors.Is(err, ErrInvariant) {
			t.Fatalf("iteration after budget error = %v", err)
		}
	})
}

func TestContextDiscoveryRejectsEmptyAndSecretShapedContext(t *testing.T) {
	kernel, repository := openTestKernel(t, filepath.Join(t.TempDir(), "invalid-context.db"))
	defer repository.Close()
	flows, _ := NewP2Flows(kernel)
	ctx := context.Background()
	actor := Actor{ID: "creator-1", WorkspaceID: "workspace-1", Kind: ActorHuman}
	if _, err := flows.StartRequirement(ctx, actor, "command-1", "project-1", 0, "requirement-1", "Need context"); err != nil {
		t.Fatal(err)
	}
	if _, err := flows.StartContextDiscovery(ctx, actor, "command-2", "project-1", 1, "requirement-1", "Discover context", 2); err != nil {
		t.Fatal(err)
	}
	if _, err := flows.IterateContextDiscovery(ctx, actor, "command-empty", "project-1", 2, "requirement-1", ContextIterationData{}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("empty context error = %v", err)
	}
	optionalOnly := ContextIterationData{Needs: []ContextNeed{{ID: "need-optional", Description: "Optional hint", Status: ContextNeedResolved, Resolution: "known"}}}
	if _, err := flows.IterateContextDiscovery(ctx, actor, "command-optional", "project-1", 2, "requirement-1", optionalOnly); !errors.Is(err, ErrInvalid) {
		t.Fatalf("optional-only context error = %v", err)
	}
	secret := ContextIterationData{Needs: []ContextNeed{{ID: "need-source", Description: "Source is known", Required: true, Status: ContextNeedResolved, Resolution: "known", SourceRefs: []string{"secret://github/11111111-1111-4111-8111-111111111111"}}}}
	if _, err := flows.IterateContextDiscovery(ctx, actor, "command-secret", "project-1", 2, "requirement-1", secret); !errors.Is(err, ErrInvalid) {
		t.Fatalf("secret-shaped source error = %v", err)
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
	if _, err := flows.QueueRun(ctx, creator, "command-4", "project-1", 1, "run-1", "worktree://task-1", []string{"secret://github/11111111-1111-4111-8111-111111111111"}, 2); err != nil {
		t.Fatal(err)
	}
	if _, err := flows.ClaimRun(ctx, runner, "command-5", "project-1", 2, "run-1", time.Minute); err != nil {
		t.Fatal(err)
	}
	if _, err := flows.ClaimRun(ctx, Actor{ID: "agent-2", WorkspaceID: "workspace-1", Kind: ActorAgent}, "command-6", "project-1", 3, "run-1", time.Minute); !errors.Is(err, ErrConflict) {
		t.Fatalf("double claim error = %v", err)
	}
	runEvidence := EvidenceRef{ID: "run-evidence-1", SubjectID: "run-1", Kind: "execution-log", URI: "artifact://runner/22222222-2222-4222-8222-222222222222", SHA256: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", Size: 10, MediaType: "application/json", ProducedBy: runner.ID, Sanitized: true}
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

func TestRunReferencesRejectCredentialShapedOrNonCanonicalValues(t *testing.T) {
	kernel, repository := openTestKernel(t, filepath.Join(t.TempDir(), "references.db"))
	defer repository.Close()
	flows, _ := NewP2Flows(kernel)
	actor := Actor{ID: "creator-1", WorkspaceID: "workspace-1", Kind: ActorHuman}
	badSecrets := []string{
		"secret://", "secret://github/token", "secret://ghp_example", "secret://user@github/11111111-1111-4111-8111-111111111111",
		"secret://github/11111111-1111-4111-8111-111111111111?token=x", "secret://github/11111111-1111-4111-8111-111111111111#token",
	}
	for index, secret := range badSecrets {
		if _, err := flows.QueueRun(context.Background(), actor, "bad-secret-"+string(rune('a'+index)), "project-1", 0, "run-bad", "worktree://task-1", []string{secret}, 1); !errors.Is(err, ErrInvalid) {
			t.Fatalf("secret %q error = %v, want invalid", secret, err)
		}
	}
	badWorkspaces := []string{"https://user:token@example.test/repo", "worktree://task-1?token=x", "worktree://task-1#token", "worktree://"}
	for index, workspace := range badWorkspaces {
		if _, err := flows.QueueRun(context.Background(), actor, "bad-workspace-"+string(rune('a'+index)), "project-1", 0, "run-bad", workspace, nil, 1); !errors.Is(err, ErrInvalid) {
			t.Fatalf("workspace %q error = %v, want invalid", workspace, err)
		}
	}
}
