package controlplane

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestRunContextBinderPinsFrozenContextAndSurvivesRunLifecycle(t *testing.T) {
	kernel, repository := openTestKernel(t, filepath.Join(t.TempDir(), "contextual-run.db"))
	defer repository.Close()

	ctx := context.Background()
	creator := Actor{ID: "creator-1", WorkspaceID: "workspace-1", Kind: ActorHuman}
	runner := Actor{ID: "agent-1", WorkspaceID: "workspace-1", Kind: ActorAgent}
	checksum := strings.Repeat("a", 64)
	readerCalls := 0
	readerAvailable := true
	reader := RuntimeContextPackReaderFunc(func(_ context.Context, workspaceID, packID string) (FrozenRuntimeContextPack, error) {
		readerCalls++
		if !readerAvailable {
			return FrozenRuntimeContextPack{}, errors.New("engineering unavailable")
		}
		return FrozenRuntimeContextPack{
			ID: packID, WorkspaceID: workspaceID,
			WorkItemKind: "task", WorkItemID: "task-1", WorkItemRevision: "7",
			PolicyVersion: "context-v1", Checksum: checksum,
		}, nil
	})
	binder, err := NewRunContextBinder(kernel, reader)
	if err != nil {
		t.Fatal(err)
	}
	request := RunExecutionContextRequest{
		ContextPackID: "context-1", ContextPackChecksum: checksum, AgentReleaseID: "release-1",
		SkillVersions: []SkillVersionPin{{SkillID: "skill-b", VersionID: "v2"}, {SkillID: "skill-a", VersionID: "v1"}},
	}
	secretRefs := []string{"secret://github/11111111-1111-4111-8111-111111111111"}
	queued, err := binder.QueueRun(ctx, creator, "command-1", "project-1", 0, "run-1", "worktree://task-1", secretRefs, 2, request)
	if err != nil {
		t.Fatal(err)
	}
	if queued.Head != 3 || len(queued.Events) != 3 || queued.Replayed {
		t.Fatalf("queued = %#v", queued)
	}
	projection, err := kernel.Replay(ctx, creator.WorkspaceID, "project-1")
	if err != nil {
		t.Fatal(err)
	}
	pinned, found, err := ResolveRunExecutionContext(projection, "run-1")
	if err != nil || !found {
		t.Fatalf("resolve pinned context: found=%v err=%v", found, err)
	}
	want := RunExecutionContextData{
		ContextPackID: "context-1", ContextPackChecksum: checksum,
		WorkItemKind: "task", WorkItemID: "task-1", WorkItemRevision: "7", ContextPolicy: "context-v1",
		AgentReleaseID: "release-1",
		SkillVersions:  []SkillVersionPin{{SkillID: "skill-a", VersionID: "v1"}, {SkillID: "skill-b", VersionID: "v2"}},
	}
	if !reflect.DeepEqual(pinned, want) {
		t.Fatalf("pinned = %#v, want %#v", pinned, want)
	}
	contextNodeID := runContextNodeID(creator.WorkspaceID, "project-1", "run-1")
	beforeContextNode := projection.Nodes[contextNodeID]
	if beforeContextNode.Kind != "run_context" || beforeContextNode.State != "frozen" || beforeContextNode.Revision != 1 {
		t.Fatalf("context node = %#v", beforeContextNode)
	}

	flows, err := NewP2Flows(kernel)
	if err != nil {
		t.Fatal(err)
	}
	claimed, err := flows.ClaimRun(ctx, runner, "command-claim", "project-1", 3, "run-1", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if claimed.Head != 4 {
		t.Fatalf("claim head = %d, want 4", claimed.Head)
	}
	projection, err = kernel.Replay(ctx, creator.WorkspaceID, "project-1")
	if err != nil {
		t.Fatal(err)
	}
	afterContextNode := projection.Nodes[contextNodeID]
	if !reflect.DeepEqual(beforeContextNode, afterContextNode) {
		t.Fatalf("context node mutated by Run lifecycle: before=%#v after=%#v", beforeContextNode, afterContextNode)
	}
	pinnedAfterClaim, found, err := ResolveRunExecutionContext(projection, "run-1")
	if err != nil || !found || !reflect.DeepEqual(pinnedAfterClaim, want) {
		t.Fatalf("context after claim = %#v found=%v err=%v", pinnedAfterClaim, found, err)
	}

	callsBeforeReplay := readerCalls
	readerAvailable = false
	replayed, err := binder.QueueRun(ctx, creator, "command-1", "project-1", 0, "run-1", "worktree://task-1", secretRefs, 2, request)
	if err != nil {
		t.Fatal(err)
	}
	if !replayed.Replayed || replayed.Head != 3 || readerCalls != callsBeforeReplay {
		t.Fatalf("receipt replay = %#v reader calls %d -> %d", replayed, callsBeforeReplay, readerCalls)
	}
}

func TestRunContextBinderRejectsInvalidFrozenContext(t *testing.T) {
	tests := []struct {
		name    string
		request RunExecutionContextRequest
		reader  RuntimeContextPackReaderFunc
		want    error
	}{
		{
			name:    "missing ContextPack",
			request: RunExecutionContextRequest{ContextPackID: "missing", ContextPackChecksum: strings.Repeat("a", 64), AgentReleaseID: "release-1"},
			reader: func(context.Context, string, string) (FrozenRuntimeContextPack, error) {
				return FrozenRuntimeContextPack{}, ErrRuntimeContextPackNotFound
			},
			want: ErrNotFound,
		},
		{
			name:    "checksum mismatch",
			request: RunExecutionContextRequest{ContextPackID: "context-1", ContextPackChecksum: strings.Repeat("b", 64), AgentReleaseID: "release-1"},
			reader: func(_ context.Context, workspaceID, packID string) (FrozenRuntimeContextPack, error) {
				return FrozenRuntimeContextPack{ID: packID, WorkspaceID: workspaceID, WorkItemKind: "task", WorkItemID: "task-1", WorkItemRevision: "7", PolicyVersion: "context-v1", Checksum: strings.Repeat("a", 64)}, nil
			},
			want: ErrConflict,
		},
		{
			name:    "workspace mismatch",
			request: RunExecutionContextRequest{ContextPackID: "context-1", ContextPackChecksum: strings.Repeat("a", 64), AgentReleaseID: "release-1"},
			reader: func(_ context.Context, _ string, packID string) (FrozenRuntimeContextPack, error) {
				return FrozenRuntimeContextPack{ID: packID, WorkspaceID: "workspace-other", WorkItemKind: "task", WorkItemID: "task-1", WorkItemRevision: "7", PolicyVersion: "context-v1", Checksum: strings.Repeat("a", 64)}, nil
			},
			want: ErrInvariant,
		},
		{
			name:    "reader unavailable",
			request: RunExecutionContextRequest{ContextPackID: "context-1", ContextPackChecksum: strings.Repeat("a", 64), AgentReleaseID: "release-1"},
			reader: func(context.Context, string, string) (FrozenRuntimeContextPack, error) {
				return FrozenRuntimeContextPack{}, errors.New("offline")
			},
			want: ErrUnavailable,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			kernel, repository := openTestKernel(t, filepath.Join(t.TempDir(), "invalid-context.db"))
			defer repository.Close()
			binder, err := NewRunContextBinder(kernel, test.reader)
			if err != nil {
				t.Fatal(err)
			}
			actor := Actor{ID: "creator-1", WorkspaceID: "workspace-1", Kind: ActorHuman}
			_, err = binder.QueueRun(context.Background(), actor, "command-1", "project-1", 0, "run-1", "worktree://task-1", nil, 1, test.request)
			if !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestRunContextBinderRejectsRebindingAndLegacyRunRemainsUnbound(t *testing.T) {
	kernel, repository := openTestKernel(t, filepath.Join(t.TempDir(), "binding-boundaries.db"))
	defer repository.Close()
	ctx := context.Background()
	creator := Actor{ID: "creator-1", WorkspaceID: "workspace-1", Kind: ActorHuman}
	checksum := strings.Repeat("a", 64)
	reader := RuntimeContextPackReaderFunc(func(_ context.Context, workspaceID, packID string) (FrozenRuntimeContextPack, error) {
		return FrozenRuntimeContextPack{ID: packID, WorkspaceID: workspaceID, WorkItemKind: "task", WorkItemID: "task-1", WorkItemRevision: "7", PolicyVersion: "context-v1", Checksum: checksum}, nil
	})
	binder, _ := NewRunContextBinder(kernel, reader)
	request := RunExecutionContextRequest{ContextPackID: "context-1", ContextPackChecksum: checksum, AgentReleaseID: "release-1"}
	if _, err := binder.QueueRun(ctx, creator, "command-1", "project-1", 0, "run-1", "worktree://task-1", nil, 1, request); err != nil {
		t.Fatal(err)
	}
	rebound := request
	rebound.AgentReleaseID = "release-2"
	if _, err := binder.QueueRun(ctx, creator, "command-2", "project-1", 3, "run-1", "worktree://task-1", nil, 1, rebound); !errors.Is(err, ErrConflict) {
		t.Fatalf("rebind error = %v, want conflict", err)
	}

	flows, _ := NewP2Flows(kernel)
	if _, err := flows.QueueRun(ctx, creator, "command-legacy", "project-1", 3, "legacy-run", "worktree://task-2", nil, 1); err != nil {
		t.Fatal(err)
	}
	projection, err := kernel.Replay(ctx, creator.WorkspaceID, "project-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, found, err := ResolveRunExecutionContext(projection, "legacy-run"); err != nil || found {
		t.Fatalf("legacy binding: found=%v err=%v", found, err)
	}
}
