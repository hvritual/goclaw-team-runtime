package controlplane

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestKernelPersistsChainAndCommandIdempotency(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "kernel.db")
	kernel, repository := openTestKernel(t, path)
	actor := Actor{ID: "owner-1", WorkspaceID: "workspace-1", Kind: ActorHuman}
	node := WorkNode{ID: "task-1", Kind: "task", Revision: 1, State: "validation", CreatorID: actor.ID, AssigneeIDs: []string{"agent-1"}}
	first, err := kernel.UpsertNode(ctx, actor, "command-1", "project-1", 0, node)
	if err != nil { t.Fatal(err) }
	if first.Head != 1 || first.Replayed { t.Fatalf("first result = %#v", first) }
	if err := repository.Close(); err != nil { t.Fatal(err) }

	kernel, repository = openTestKernel(t, path)
	defer repository.Close()
	replayed, err := kernel.UpsertNode(ctx, actor, "command-1", "project-1", 0, node)
	if err != nil { t.Fatal(err) }
	if !replayed.Replayed || replayed.HeadHash != first.HeadHash || replayed.Events[0].EventID != first.Events[0].EventID {
		t.Fatalf("replayed result = %#v, first = %#v", replayed, first)
	}
	changed := node
	changed.State = "running"
	if _, err := kernel.UpsertNode(ctx, actor, "command-1", "project-1", 1, changed); !errors.Is(err, ErrConflict) {
		t.Fatalf("changed command error = %v, want conflict", err)
	}
	projection, err := kernel.Replay(ctx, actor.WorkspaceID, "project-1")
	if err != nil { t.Fatal(err) }
	if projection.Head != 1 || projection.Nodes[node.ID].State != node.State { t.Fatalf("projection = %#v", projection) }
	firstDigest := stableProjectionDigest(projection)
	again, err := kernel.Replay(ctx, actor.WorkspaceID, "project-1")
	if err != nil { t.Fatal(err) }
	if stableProjectionDigest(again) != firstDigest { t.Fatal("replay digest changed") }
}

func TestKernelCASAllowsOneConcurrentCommand(t *testing.T) {
	kernel, repository := openTestKernel(t, filepath.Join(t.TempDir(), "kernel.db"))
	defer repository.Close()
	actor := Actor{ID: "owner-1", WorkspaceID: "workspace-1", Kind: ActorHuman}
	results := make(chan error, 2)
	var wait sync.WaitGroup
	for _, id := range []string{"task-1", "task-2"} {
		wait.Add(1)
		go func(nodeID string) {
			defer wait.Done()
			_, err := kernel.UpsertNode(context.Background(), actor, "command-"+nodeID, "project-1", 0,
				WorkNode{ID: nodeID, Kind: "task", Revision: 1, State: "draft", CreatorID: actor.ID})
			results <- err
		}(id)
	}
	wait.Wait()
	close(results)
	var successes, conflicts int
	for err := range results {
		if err == nil { successes++ } else if errors.Is(err, ErrConflict) { conflicts++ } else { t.Fatalf("unexpected error: %v", err) }
	}
	if successes != 1 || conflicts != 1 { t.Fatalf("successes=%d conflicts=%d", successes, conflicts) }
}

func TestKernelTamperingAndUnknownEventsFailClosed(t *testing.T) {
	kernel, repository := openTestKernel(t, filepath.Join(t.TempDir(), "kernel.db"))
	defer repository.Close()
	actor := Actor{ID: "owner-1", WorkspaceID: "workspace-1", Kind: ActorHuman}
	if _, err := kernel.UpsertNode(context.Background(), actor, "command-1", "project-1", 0,
		WorkNode{ID: "task-1", Kind: "task", Revision: 1, State: "draft", CreatorID: actor.ID}); err != nil { t.Fatal(err) }
	sqlStore := repository.(*sqlRepository)
	if _, err := sqlStore.db.Exec(`UPDATE cp_session_events SET payload = '{"tampered":true}' WHERE workspace_id = 'workspace-1'`); err != nil { t.Fatal(err) }
	if _, err := kernel.Replay(context.Background(), actor.WorkspaceID, "project-1"); !errors.Is(err, ErrInvariant) {
		t.Fatalf("tampered replay error = %v, want invariant", err)
	}
}

func TestWorkGraphRejectsDependencyCycle(t *testing.T) {
	kernel, repository := openTestKernel(t, filepath.Join(t.TempDir(), "kernel.db"))
	defer repository.Close()
	ctx := context.Background()
	actor := Actor{ID: "owner-1", WorkspaceID: "workspace-1", Kind: ActorHuman}
	for index, id := range []string{"task-1", "task-2"} {
		if _, err := kernel.UpsertNode(ctx, actor, "node-"+id, "project-1", int64(index), WorkNode{ID: id, Kind: "task", Revision: 1, State: "draft", CreatorID: actor.ID}); err != nil { t.Fatal(err) }
	}
	if _, err := kernel.AddEdge(ctx, actor, "edge-1", "project-1", 2, WorkEdge{ID: "edge-1", From: "task-1", To: "task-2", Kind: "depends_on"}); err != nil { t.Fatal(err) }
	if _, err := kernel.AddEdge(ctx, actor, "edge-2", "project-1", 3, WorkEdge{ID: "edge-2", From: "task-2", To: "task-1", Kind: "depends_on"}); !errors.Is(err, ErrInvariant) {
		t.Fatalf("cycle error = %v, want invariant", err)
	}
}

func TestDoneGateUsesChecksEvidenceAndIndependentHuman(t *testing.T) {
	kernel, repository := openTestKernel(t, filepath.Join(t.TempDir(), "kernel.db"))
	defer repository.Close()
	ctx := context.Background()
	creator := Actor{ID: "creator-1", WorkspaceID: "workspace-1", Kind: ActorHuman}
	runner := Actor{ID: "agent-1", WorkspaceID: "workspace-1", Kind: ActorAgent}
	checker := Actor{ID: "checker-1", WorkspaceID: "workspace-1", Kind: ActorHuman}
	acceptor := Actor{ID: "acceptor-1", WorkspaceID: "workspace-1", Kind: ActorHuman}
	node := WorkNode{ID: "task-1", Kind: "task", Revision: 1, State: "validation", CreatorID: creator.ID, AssigneeIDs: []string{runner.ID}, ExecutorIDs: []string{runner.ID}}
	if _, err := kernel.UpsertNode(ctx, creator, "command-1", "project-1", 0, node); err != nil { t.Fatal(err) }
	digest := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	evidence := EvidenceRef{ID: "evidence-1", SubjectID: node.ID, Kind: "test-report", URI: "artifact://run-1/report.json", SHA256: digest, Size: 42, MediaType: "application/json", ProducedBy: runner.ID, RunID: "run-1", Sanitized: true}
	if _, err := kernel.AttachEvidence(ctx, runner, "command-2", "project-1", 1, evidence); err != nil { t.Fatal(err) }
	if _, err := kernel.AcceptDone(ctx, acceptor, "command-3", "project-1", node.ID, node.Revision, 2, []string{"policy-test"}); !errors.Is(err, ErrInvariant) {
		t.Fatalf("accept without check error = %v, want invariant", err)
	}
	check := CheckResult{ID: "check-1", PolicyID: "policy-test", SubjectID: node.ID, Revision: node.Revision, Outcome: CheckPassed, EvidenceIDs: []string{evidence.ID}, CheckerID: checker.ID, Deterministic: true}
	if _, err := kernel.RecordCheck(ctx, checker, "command-4", "project-1", 2, check); err != nil { t.Fatal(err) }
	if _, err := kernel.AcceptDone(ctx, runner, "command-5", "project-1", node.ID, node.Revision, 3, []string{"policy-test"}); !errors.Is(err, ErrDenied) {
		t.Fatalf("runner accept error = %v, want denied", err)
	}
	if _, err := kernel.AcceptDone(ctx, creator, "command-6", "project-1", node.ID, node.Revision, 3, []string{"policy-test"}); !errors.Is(err, ErrDenied) {
		t.Fatalf("creator accept error = %v, want denied", err)
	}
	accepted, err := kernel.AcceptDone(ctx, acceptor, "command-7", "project-1", node.ID, node.Revision, 3, []string{"policy-test"})
	if err != nil { t.Fatal(err) }
	if accepted.Head != 4 { t.Fatalf("accepted result = %#v", accepted) }
	projection, err := kernel.Replay(ctx, creator.WorkspaceID, "project-1")
	if err != nil { t.Fatal(err) }
	if projection.Nodes[node.ID].State != "done" || projection.Acceptances[node.ID].AcceptorID != acceptor.ID { t.Fatalf("projection = %#v", projection) }
}

func openTestKernel(t *testing.T, path string) (*DeliveryKernel, Repository) {
	t.Helper()
	repository, err := OpenSQLite(context.Background(), path)
	if err != nil { t.Fatal(err) }
	store, err := KernelStoreFrom(repository)
	if err != nil { t.Fatal(err) }
	fixed := time.Date(2026, 8, 11, 14, 0, 0, 0, time.UTC)
	kernel, err := NewDeliveryKernel(store, func() time.Time { fixed = fixed.Add(time.Second); return fixed }, func(_ context.Context, actor Actor, permission string) error {
		if actor.Kind == ActorAgent && permission == PermissionAccept { return ErrDenied }
		return nil
	})
	if err != nil { t.Fatal(err) }
	return kernel, repository
}
