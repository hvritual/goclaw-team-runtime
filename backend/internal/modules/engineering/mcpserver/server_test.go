package mcpserver

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/engineering/contract"
)

type fakeService struct {
	contract.Service
	entities      []contract.Entity
	edges         map[string][]contract.ThreadEdge
	changes       []contract.Change
	pack          contract.ContextPack
	lastActor     contract.Actor
	lastWorkspace string
	err           error
}

func (f *fakeService) GetEntity(_ context.Context, actor contract.Actor, workspaceID, id string) (contract.Entity, error) {
	f.lastActor, f.lastWorkspace = actor, workspaceID
	if f.err != nil {
		return contract.Entity{}, f.err
	}
	for _, value := range f.entities {
		if value.ID == id {
			return value, nil
		}
	}
	return contract.Entity{}, contract.ErrNotFound
}

func (f *fakeService) ListEntities(_ context.Context, actor contract.Actor, workspaceID string) ([]contract.Entity, error) {
	f.lastActor, f.lastWorkspace = actor, workspaceID
	if f.err != nil {
		return nil, f.err
	}
	return append([]contract.Entity(nil), f.entities...), nil
}

func (f *fakeService) ListThreadEdges(_ context.Context, actor contract.Actor, workspaceID string, node contract.NodeRef) ([]contract.ThreadEdge, error) {
	f.lastActor, f.lastWorkspace = actor, workspaceID
	if f.err != nil {
		return nil, f.err
	}
	return append([]contract.ThreadEdge(nil), f.edges[nodeKey(node)]...), nil
}

func (f *fakeService) GetChange(_ context.Context, actor contract.Actor, workspaceID, id string) (contract.Change, error) {
	f.lastActor, f.lastWorkspace = actor, workspaceID
	for _, value := range f.changes {
		if value.ID == id {
			return value, nil
		}
	}
	return contract.Change{}, contract.ErrNotFound
}

func (f *fakeService) ListChanges(_ context.Context, actor contract.Actor, workspaceID, _ string) ([]contract.Change, error) {
	f.lastActor, f.lastWorkspace = actor, workspaceID
	return append([]contract.Change(nil), f.changes...), nil
}

func (f *fakeService) GetContextPack(_ context.Context, actor contract.Actor, workspaceID, _ string) (contract.ContextPack, error) {
	f.lastActor, f.lastWorkspace = actor, workspaceID
	if f.err != nil {
		return contract.ContextPack{}, f.err
	}
	return f.pack, nil
}

type fakeCompiler struct {
	actor   contract.Actor
	request contract.CompileContextRequest
	result  contract.CompileContextResult
	err     error
}

func (f *fakeCompiler) CompileContext(_ context.Context, actor contract.Actor, request contract.CompileContextRequest) (contract.CompileContextResult, error) {
	f.actor, f.request = actor, request
	return f.result, f.err
}

func TestEntitySearchUsesFixedActorAndDeterministicOrder(t *testing.T) {
	service := &fakeService{entities: []contract.Entity{
		{ID: "service-z", Name: "Device Gateway", Type: "service", Status: "active"},
		{ID: "api-a", Name: "Device Session API", Type: "api", Status: "active"},
		{ID: "service-a", Name: "Session Service", Type: "service", Status: "archived"},
	}}
	tools := &toolset{service: service, actor: contract.Actor{UserID: "user-1"}}
	_, output, err := tools.entitySearch(context.Background(), nil, entitySearchInput{WorkspaceID: "workspace-1", Query: "device", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if service.lastActor.UserID != "user-1" || service.lastWorkspace != "workspace-1" {
		t.Fatalf("authorization identity drift: actor=%#v workspace=%q", service.lastActor, service.lastWorkspace)
	}
	got := []string{output.Items[0].ID, output.Items[1].ID}
	want := []string{"api-a", "service-z"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("search order = %v, want %v", got, want)
	}
}

func TestThreadTraverseIsBoundedAndDeterministic(t *testing.T) {
	a := contract.NodeRef{Kind: "engineering_entity", ID: "a"}
	b := contract.NodeRef{Kind: "engineering_entity", ID: "b"}
	c := contract.NodeRef{Kind: "engineering_entity", ID: "c"}
	edgeAB := contract.ThreadEdge{ID: "edge-b", From: a, Relation: "depends_on", To: b, Authority: "authoritative"}
	edgeBC := contract.ThreadEdge{ID: "edge-a", From: b, Relation: "depends_on", To: c, Authority: "authoritative"}
	service := &fakeService{edges: map[string][]contract.ThreadEdge{
		nodeKey(a): {edgeAB},
		nodeKey(b): {edgeBC, edgeAB},
		nodeKey(c): {edgeBC},
	}}
	tools := &toolset{service: service, actor: contract.Actor{UserID: "user-1"}}
	_, output, err := tools.threadTraverse(context.Background(), nil, threadTraverseInput{WorkspaceID: "workspace-1", NodeID: "a", MaxDepth: 2, MaxNodes: 2})
	if err != nil {
		t.Fatal(err)
	}
	if !output.Truncated || len(output.Nodes) != 2 {
		t.Fatalf("bounded output = %#v", output)
	}
	if output.Nodes[0].ID != "a" || output.Nodes[1].ID != "b" {
		t.Fatalf("nodes = %#v", output.Nodes)
	}
	if len(output.Edges) != 2 || output.Edges[0].ID != "edge-a" || output.Edges[1].ID != "edge-b" {
		t.Fatalf("edge ordering = %#v", output.Edges)
	}
}

func TestContextPackCompileUsesFixedActorAndConvertsPolicy(t *testing.T) {
	compiler := &fakeCompiler{result: contract.CompileContextResult{Pack: contract.ContextPack{ID: "pack-1"}}}
	tools := &toolset{compiler: compiler, actor: contract.Actor{UserID: "user-1"}}
	_, output, err := tools.contextPackCompile(context.Background(), nil, contextPackCompileInput{
		WorkspaceID: "workspace-1", PackID: "pack-1", WorkItemKind: "task", WorkItemID: "task-1", WorkItemRevision: "7",
		PolicyVersion: "context-v1", SourceStaleSeconds: 60, KnowledgeStaleSeconds: 120,
	})
	if err != nil {
		t.Fatal(err)
	}
	if output.Result.Pack.ID != "pack-1" || compiler.actor.UserID != "user-1" {
		t.Fatalf("compile result/actor = %#v %#v", output, compiler.actor)
	}
	if compiler.request.Policy.SourceStaleAfter != time.Minute || compiler.request.Policy.KnowledgeStaleAfter != 2*time.Minute {
		t.Fatalf("policy duration conversion = %#v", compiler.request.Policy)
	}
}

func TestStableToolErrorDoesNotLeakDependencyDetail(t *testing.T) {
	service := &fakeService{err: errors.Join(contract.ErrUnavailable, errors.New("sqlite password=secret"))}
	tools := &toolset{service: service, actor: contract.Actor{UserID: "user-1"}}
	_, _, err := tools.entityGet(context.Background(), nil, entityGetInput{WorkspaceID: "workspace-1", EntityID: "service-a"})
	if err == nil || err.Error() != "unavailable" {
		t.Fatalf("error = %v, want sanitized unavailable", err)
	}
}

func TestNewRequiresFixedIdentityAndCapabilities(t *testing.T) {
	service := &fakeService{}
	compiler := &fakeCompiler{}
	if _, err := New(Dependencies{Service: service, Compiler: compiler}); err == nil {
		t.Fatal("expected missing user identity to fail")
	}
	if _, err := New(Dependencies{Service: service, Compiler: compiler, UserID: "user-1", Version: "test"}); err != nil {
		t.Fatal(err)
	}
}
