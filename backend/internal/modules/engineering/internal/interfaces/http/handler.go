package application

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/engineering/contract"
	"github.com/hvritual/workspace/internal/modules/engineering/internal/domain"
	persistence "github.com/hvritual/workspace/internal/modules/engineering/internal/infrastructure/sqlite"
	_ "modernc.org/sqlite"
)

func TestServiceAuthorizationMatrixAndWorkspaceIsolation(t *testing.T) {
	service, _, closeDB := newTestService(t)
	defer closeDB()
	ctx := context.Background()
	owner := contract.Actor{UserID: "owner"}
	member := contract.Actor{UserID: "member"}
	admin := contract.Actor{UserID: "admin"}
	outsider := contract.Actor{UserID: "outsider"}

	created, err := service.CreateEntity(ctx, owner, "workspace-a", contract.CreateEntityRequest{ID: "service-1", Type: "service", Name: "Device Gateway", Status: "active"})
	if err != nil {
		t.Fatal(err)
	}
	if created.WorkspaceID != "workspace-a" || created.Status != "active" {
		t.Fatalf("created = %+v", created)
	}
	if _, err := service.GetEntity(ctx, member, "workspace-a", "service-1"); err != nil {
		t.Fatalf("member read: %v", err)
	}
	if _, err := service.CreateEntity(ctx, admin, "workspace-a", contract.CreateEntityRequest{ID: "service-admin", Type: "service", Name: "Admin managed service"}); err != nil {
		t.Fatalf("admin write: %v", err)
	}
	name := "Forbidden rename"
	if _, err := service.UpdateEntity(ctx, member, "workspace-a", "service-1", contract.UpdateEntityRequest{Name: &name}); !errors.Is(err, contract.ErrForbidden) {
		t.Fatalf("member write error = %v, want forbidden", err)
	}
	if _, err := service.ListEntities(ctx, outsider, "workspace-a"); !errors.Is(err, contract.ErrForbidden) {
		t.Fatalf("outsider read error = %v, want forbidden", err)
	}
	if _, err := service.GetEntity(ctx, owner, "workspace-b", "service-1"); !errors.Is(err, contract.ErrNotFound) {
		t.Fatalf("cross-workspace read error = %v, want not found", err)
	}
}

func TestServiceEngineeringThreadCRUDAndFrozenContextPackRead(t *testing.T) {
	service, repository, closeDB := newTestService(t)
	defer closeDB()
	ctx := context.Background()
	owner := contract.Actor{UserID: "owner"}
	member := contract.Actor{UserID: "member"}

	if _, err := service.CreateEntity(ctx, owner, "workspace-a", contract.CreateEntityRequest{ID: "service-1", Type: "service", Name: "Device Gateway", Status: "active", OwnerRef: "member:owner"}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateEntity(ctx, owner, "workspace-a", contract.CreateEntityRequest{ID: "repo-1", Type: "repository", Name: "gateway-repo", Status: "active"}); err != nil {
		t.Fatal(err)
	}

	binding, err := service.CreateSourceBinding(ctx, owner, "workspace-a", contract.CreateSourceBindingRequest{
		ID: "binding-1", EntityID: "repo-1", Authority: "authoritative",
		Provenance: contract.Provenance{SourceType: "github", Locator: "github.com/acme/gateway", Revision: "abc123"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if binding.Provenance.ObservedAt.IsZero() {
		t.Fatal("source binding observed_at was not defaulted")
	}
	bindings, err := service.ListSourceBindings(ctx, member, "workspace-a", "repo-1")
	if err != nil || len(bindings) != 1 || bindings[0].ID != "binding-1" {
		t.Fatalf("bindings = %+v err=%v", bindings, err)
	}

	edge, err := service.CreateThreadEdge(ctx, owner, "workspace-a", contract.CreateThreadEdgeRequest{
		ID:         "edge-1",
		From:       contract.NodeRef{Kind: "engineering_entity", ID: "service-1"},
		Relation:   "depends_on",
		To:         contract.NodeRef{Kind: "engineering_entity", ID: "repo-1"},
		Authority:  "authoritative",
		Provenance: contract.Provenance{SourceType: "catalog", Locator: "engineering.yaml", Revision: "r1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if edge.Relation != "depends_on" {
		t.Fatalf("edge = %+v", edge)
	}
	edges, err := service.ListThreadEdges(ctx, member, "workspace-a", contract.NodeRef{Kind: "engineering_entity", ID: "service-1"})
	if err != nil || len(edges) != 1 || edges[0].ID != "edge-1" {
		t.Fatalf("edges = %+v err=%v", edges, err)
	}

	change, err := service.CreateChange(ctx, owner, "workspace-a", contract.CreateChangeRequest{
		ID: "change-1", ProjectID: "project-1", RequirementID: "requirement-1",
		WorkItem: &contract.NodeRef{Kind: "task", ID: "task-1"}, RunID: "run-1",
		Summary: "Update gateway session handling", AffectedEntityIDs: []string{"service-1", "repo-1"},
		Artifacts:  []contract.ArtifactRef{{Kind: "commit", Locator: "github.com/acme/gateway/commit/abc123", Revision: "abc123"}},
		Provenance: contract.Provenance{SourceType: "task", Locator: "task-1", Revision: "7"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if change.Status != "proposed" || change.CreatedAt.IsZero() {
		t.Fatalf("change = %+v", change)
	}
	changes, err := service.ListChanges(ctx, member, "workspace-a", "service-1")
	if err != nil || len(changes) != 1 || changes[0].ID != "change-1" {
		t.Fatalf("changes = %+v err=%v", changes, err)
	}

	now := testNow()
	workItem, err := domain.NewNodeRef(domain.NodeKindTask, "task-1")
	if err != nil {
		t.Fatal(err)
	}
	reference, err := domain.NewContextReference(domain.ContextKindArchitecture, "architecture-1", "r3", "sha256:architecture")
	if err != nil {
		t.Fatal(err)
	}
	pack, err := domain.NewContextPack("context-1", "workspace-a", workItem, "7", []string{"service-1"}, []domain.ContextReference{reference}, "policy-v1", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.PutContextPack(ctx, pack); err != nil {
		t.Fatal(err)
	}
	manifest, err := service.GetContextPack(ctx, member, "workspace-a", "context-1")
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Checksum == "" || manifest.WorkItem.ID != "task-1" || len(manifest.References) != 1 {
		t.Fatalf("manifest = %+v", manifest)
	}
}

func TestServiceRejectsInvalidReferencesAndConflictingCreates(t *testing.T) {
	service, _, closeDB := newTestService(t)
	defer closeDB()
	ctx := context.Background()
	owner := contract.Actor{UserID: "owner"}
	if _, err := service.CreateEntity(ctx, owner, "workspace-a", contract.CreateEntityRequest{ID: "service-1", Type: "service", Name: "Gateway"}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateEntity(ctx, owner, "workspace-a", contract.CreateEntityRequest{ID: "service-1", Type: "service", Name: "Duplicate"}); !errors.Is(err, contract.ErrConflict) {
		t.Fatalf("duplicate create error = %v", err)
	}
	if _, err := service.CreateSourceBinding(ctx, owner, "workspace-a", contract.CreateSourceBindingRequest{
		ID: "binding-1", EntityID: "missing", Authority: "observed",
		Provenance: contract.Provenance{SourceType: "github", Locator: "missing"},
	}); !errors.Is(err, contract.ErrNotFound) {
		t.Fatalf("missing binding entity error = %v", err)
	}
	if _, err := service.CreateThreadEdge(ctx, owner, "workspace-a", contract.CreateThreadEdgeRequest{
		ID: "edge-1", From: contract.NodeRef{Kind: "engineering_entity", ID: "service-1"}, Relation: "depends_on",
		To: contract.NodeRef{Kind: "engineering_entity", ID: "missing"}, Authority: "observed",
		Provenance: contract.Provenance{SourceType: "catalog", Locator: "engineering.yaml"},
	}); !errors.Is(err, contract.ErrNotFound) {
		t.Fatalf("missing edge entity error = %v", err)
	}
}

func newTestService(t *testing.T) (*Service, *persistence.Store, func()) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:engineering-application?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	if err := persistence.Migrate(context.Background(), db); err != nil {
		_ = db.Close()
		t.Fatal(err)
	}
	repository, err := persistence.New(db)
	if err != nil {
		_ = db.Close()
		t.Fatal(err)
	}
	roles := map[string]string{
		"owner/workspace-a":  "owner",
		"owner/workspace-b":  "owner",
		"member/workspace-a": "member",
		"member/workspace-b": "member",
		"admin/workspace-a":  "admin",
	}
	memberships := contract.WorkspaceRoleResolverFunc(func(_ context.Context, userID, workspaceID string) (string, bool, error) {
		role, found := roles[userID+"/"+workspaceID]
		return role, found, nil
	})
	service, err := New(repository, memberships, testNow)
	if err != nil {
		_ = db.Close()
		t.Fatal(err)
	}
	return service, repository, func() { _ = db.Close() }
}

func testNow() time.Time {
	return time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
}
