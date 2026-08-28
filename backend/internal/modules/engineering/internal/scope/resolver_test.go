package scope

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/engineering/internal/domain"
	persistence "github.com/hvritual/workspace/internal/modules/engineering/internal/infrastructure/sqlite"
	_ "modernc.org/sqlite"
)

func TestResolveTraversesAuthoritativeGraphAndReportsSourceWarnings(t *testing.T) {
	store, closeStore := newScopeStore(t)
	defer closeStore()
	ctx := context.Background()
	workspaceID := "workspace-a"
	now := time.Date(2026, 8, 28, 18, 0, 0, 0, time.UTC)

	for _, entity := range []domain.EngineeringEntity{
		scopeEntity(t, "service-a", workspaceID, domain.EntityTypeService, domain.EntityStatusActive),
		scopeEntity(t, "service-b", workspaceID, domain.EntityTypeService, domain.EntityStatusActive),
		scopeEntity(t, "api-c", workspaceID, domain.EntityTypeAPI, domain.EntityStatusActive),
		scopeEntity(t, "service-ignored", workspaceID, domain.EntityTypeService, domain.EntityStatusActive),
	} {
		if err := store.PutEntity(ctx, entity); err != nil {
			t.Fatal(err)
		}
	}
	work, _ := domain.NewNodeRef(domain.NodeKindTask, "task-one")
	putScopeEdge(t, store, workspaceID, "work-link", work, domain.RelationAffects, engineeringNode(t, "service-a"), domain.AuthorityAuthoritative, "workspace", "workspace://workspace-a/work/task/task-one", "", now)
	putScopeEdge(t, store, workspaceID, "depends", engineeringNode(t, "service-a"), domain.RelationDependsOn, engineeringNode(t, "service-b"), domain.AuthorityAuthoritative, "github_manifest", "github://acme/a/engineering.yaml", "sha-a", now)
	putScopeEdge(t, store, workspaceID, "provides", engineeringNode(t, "service-b"), domain.RelationProvides, engineeringNode(t, "api-c"), domain.AuthorityAuthoritative, "github_manifest", "github://acme/b/engineering.yaml", "sha-b", now)
	putScopeEdge(t, store, workspaceID, "observed-ignore", engineeringNode(t, "service-a"), domain.RelationDependsOn, engineeringNode(t, "service-ignored"), domain.AuthorityObserved, "observer", "observer://runtime", "obs-1", now)
	putScopeEdge(t, store, workspaceID, "dangling", engineeringNode(t, "service-a"), domain.RelationUses, engineeringNode(t, "missing-api"), domain.AuthorityAuthoritative, "github_manifest", "github://acme/a/engineering.yaml", "sha-a", now)

	putScopeBinding(t, store, workspaceID, "binding-a", "service-a", "github", "github://acme/a", "sha-a", domain.AuthorityAuthoritative, now.Add(-60*24*time.Hour))
	putScopeBinding(t, store, workspaceID, "binding-b", "service-b", "github", "github://acme/b", "", domain.AuthorityAuthoritative, now)

	resolver, err := New(store, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	result, err := resolver.Resolve(ctx, Input{WorkspaceID: workspaceID, WorkItem: work, Policy: Policy{MaxDepth: 2, MaxEntities: 16, SourceStaleAfter: 30 * 24 * time.Hour}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Truncated || len(result.Entities) != 3 {
		t.Fatalf("scope = %+v", result)
	}
	want := []struct {
		id    string
		depth int
	}{
		{id: "service-a", depth: 0},
		{id: "service-b", depth: 1},
		{id: "api-c", depth: 2},
	}
	for index, expected := range want {
		if result.Entities[index].ID != expected.id || result.Entities[index].Depth != expected.depth {
			t.Fatalf("entity[%d] = %+v, want %s depth %d", index, result.Entities[index], expected.id, expected.depth)
		}
	}
	if containsScopedEntity(result.Entities, "service-ignored") {
		t.Fatalf("observed edge expanded scope: %+v", result.Entities)
	}
	if len(result.Sources) != 2 || result.Sources[0].EntityID != "service-a" || !result.Sources[0].Stale || result.Sources[1].EntityID != "service-b" {
		t.Fatalf("sources = %+v", result.Sources)
	}
	for _, code := range []string{"dangling_edge", "stale_source", "unpinned_source"} {
		if !hasWarning(result.Warnings, code) {
			t.Fatalf("missing warning %q in %+v", code, result.Warnings)
		}
	}
}

func TestResolveTraversesSelectedInboundRelationsAndRetainsArchivedEntity(t *testing.T) {
	store, closeStore := newScopeStore(t)
	defer closeStore()
	ctx := context.Background()
	workspaceID := "workspace-a"
	now := time.Date(2026, 8, 28, 18, 0, 0, 0, time.UTC)
	for _, entity := range []domain.EngineeringEntity{
		scopeEntity(t, "api-c", workspaceID, domain.EntityTypeAPI, domain.EntityStatusActive),
		scopeEntity(t, "service-provider", workspaceID, domain.EntityTypeService, domain.EntityStatusArchived),
	} {
		if err := store.PutEntity(ctx, entity); err != nil {
			t.Fatal(err)
		}
	}
	work, _ := domain.NewNodeRef(domain.NodeKindTask, "task-api")
	putScopeEdge(t, store, workspaceID, "work-api", work, domain.RelationAffects, engineeringNode(t, "api-c"), domain.AuthorityAuthoritative, "workspace", "workspace://workspace-a/work/task/task-api", "", now)
	putScopeEdge(t, store, workspaceID, "provider", engineeringNode(t, "service-provider"), domain.RelationProvides, engineeringNode(t, "api-c"), domain.AuthorityAuthoritative, "github_manifest", "github://acme/provider/engineering.yaml", "sha-provider", now)

	resolver, _ := New(store, func() time.Time { return now })
	result, err := resolver.Resolve(ctx, Input{WorkspaceID: workspaceID, WorkItem: work, Policy: Policy{MaxDepth: 1, MaxEntities: 8}})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Entities) != 2 || result.Entities[1].ID != "service-provider" || result.Entities[1].Direction != "inbound" {
		t.Fatalf("inbound scope = %+v", result.Entities)
	}
	if !hasWarning(result.Warnings, "archived_entity") {
		t.Fatalf("warnings = %+v", result.Warnings)
	}
}

func TestResolveHonorsEntityLimitAndRequiresAuthoritativeWorkSeed(t *testing.T) {
	store, closeStore := newScopeStore(t)
	defer closeStore()
	ctx := context.Background()
	workspaceID := "workspace-a"
	now := time.Date(2026, 8, 28, 18, 0, 0, 0, time.UTC)
	for _, id := range []string{"service-a", "service-b", "service-c"} {
		if err := store.PutEntity(ctx, scopeEntity(t, id, workspaceID, domain.EntityTypeService, domain.EntityStatusActive)); err != nil {
			t.Fatal(err)
		}
	}
	work, _ := domain.NewNodeRef(domain.NodeKindTask, "task-limit")
	putScopeEdge(t, store, workspaceID, "work-limit", work, domain.RelationAffects, engineeringNode(t, "service-a"), domain.AuthorityAuthoritative, "workspace", "workspace://workspace-a/work/task/task-limit", "", now)
	putScopeEdge(t, store, workspaceID, "a-b", engineeringNode(t, "service-a"), domain.RelationDependsOn, engineeringNode(t, "service-b"), domain.AuthorityAuthoritative, "manifest", "manifest://a", "r1", now)
	putScopeEdge(t, store, workspaceID, "b-c", engineeringNode(t, "service-b"), domain.RelationDependsOn, engineeringNode(t, "service-c"), domain.AuthorityAuthoritative, "manifest", "manifest://b", "r1", now)
	resolver, _ := New(store, func() time.Time { return now })
	limited, err := resolver.Resolve(ctx, Input{WorkspaceID: workspaceID, WorkItem: work, Policy: Policy{MaxDepth: 4, MaxEntities: 2}})
	if err != nil {
		t.Fatal(err)
	}
	if !limited.Truncated || len(limited.Entities) != 2 || !hasWarning(limited.Warnings, "entity_limit") {
		t.Fatalf("limited scope = %+v", limited)
	}

	observedWork, _ := domain.NewNodeRef(domain.NodeKindTask, "task-observed")
	putScopeEdge(t, store, workspaceID, "observed-work", observedWork, domain.RelationAffects, engineeringNode(t, "service-a"), domain.AuthorityObserved, "workspace", "workspace://workspace-a/work/task/task-observed", "", now)
	if _, err := resolver.Resolve(ctx, Input{WorkspaceID: workspaceID, WorkItem: observedWork}); !errors.Is(err, ErrNoScope) {
		t.Fatalf("observed work seed error = %v", err)
	}
}

func TestResolveRejectsInvalidPolicyAndWorkKind(t *testing.T) {
	store, closeStore := newScopeStore(t)
	defer closeStore()
	resolver, _ := New(store, time.Now)
	badWork, _ := domain.NewNodeRef(domain.NodeKindRun, "run-one")
	if _, err := resolver.Resolve(context.Background(), Input{WorkspaceID: "workspace-a", WorkItem: badWork}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("bad work error = %v", err)
	}
	task, _ := domain.NewNodeRef(domain.NodeKindTask, "task-one")
	if _, err := resolver.Resolve(context.Background(), Input{WorkspaceID: "workspace-a", WorkItem: task, Policy: Policy{MaxDepth: HardMaxDepth + 1}}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("bad policy error = %v", err)
	}
}

func newScopeStore(t *testing.T) (*persistence.Store, func()) {
	t.Helper()
	name := strings.NewReplacer("/", "-", " ", "-").Replace(t.Name())
	db, err := sql.Open("sqlite", "file:"+name+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	if err := persistence.Migrate(context.Background(), db); err != nil {
		db.Close()
		t.Fatal(err)
	}
	store, err := persistence.New(db)
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	return store, func() { db.Close() }
}

func scopeEntity(t *testing.T, id, workspaceID string, entityType domain.EntityType, status domain.EntityStatus) domain.EngineeringEntity {
	t.Helper()
	value, err := domain.NewEngineeringEntity(id, workspaceID, entityType, id, status, "team:iot")
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func engineeringNode(t *testing.T, id string) domain.NodeRef {
	t.Helper()
	value, err := domain.NewNodeRef(domain.NodeKindEngineeringEntity, id)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func putScopeEdge(t *testing.T, store *persistence.Store, workspaceID, id string, from domain.NodeRef, relation domain.RelationType, to domain.NodeRef, authority domain.Authority, sourceType, locator, revision string, observedAt time.Time) {
	t.Helper()
	provenance, err := domain.NewProvenance(sourceType, locator, revision, observedAt)
	if err != nil {
		t.Fatal(err)
	}
	edge, err := domain.NewThreadEdge(id, workspaceID, from, relation, to, authority, provenance)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.PutThreadEdge(context.Background(), edge); err != nil {
		t.Fatal(err)
	}
}

func putScopeBinding(t *testing.T, store *persistence.Store, workspaceID, id, entityID, sourceType, locator, revision string, authority domain.Authority, observedAt time.Time) {
	t.Helper()
	provenance, err := domain.NewProvenance(sourceType, locator, revision, observedAt)
	if err != nil {
		t.Fatal(err)
	}
	binding, err := domain.NewSourceBinding(id, workspaceID, entityID, provenance, authority)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.PutSourceBinding(context.Background(), binding); err != nil {
		t.Fatal(err)
	}
}

func containsScopedEntity(values []ScopedEntity, id string) bool {
	for _, value := range values {
		if value.ID == id {
			return true
		}
	}
	return false
}

func hasWarning(values []Warning, code string) bool {
	for _, value := range values {
		if value.Code == code {
			return true
		}
	}
	return false
}
