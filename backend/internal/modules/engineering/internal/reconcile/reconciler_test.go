package reconcile

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/engineering/internal/domain"
	persistence "github.com/hvritual/workspace/internal/modules/engineering/internal/infrastructure/sqlite"
	githubsource "github.com/hvritual/workspace/internal/modules/engineering/internal/source/github"
	_ "modernc.org/sqlite"
)

const (
	reconcileSHA1 = "1111111111111111111111111111111111111111"
	reconcileSHA2 = "2222222222222222222222222222222222222222"
)

type fakeSource struct {
	repository githubsource.Repository
	commit     githubsource.Commit
	blob       githubsource.ManifestBlob
}

func (s *fakeSource) GetRepository(context.Context, string) (githubsource.Repository, error) {
	return s.repository, nil
}

func (s *fakeSource) ResolveCommit(_ context.Context, locator, _ string) (githubsource.Commit, error) {
	value := s.commit
	value.RepositoryLocator = locator
	return value, nil
}

func (s *fakeSource) ReadEngineeringManifestAtCommit(_ context.Context, locator, sha string) (githubsource.ManifestBlob, error) {
	value := s.blob
	value.RepositoryLocator = locator
	value.CommitSHA = sha
	return value, nil
}

func TestReconcileProjectsPinnedManifestAndRepairsStaleEdges(t *testing.T) {
	store, closeStore := newReconcileStore(t)
	defer closeStore()
	ctx := context.Background()
	workspaceID := "workspace-a"
	for _, entity := range []domain.EngineeringEntity{
		mustEntity(t, "service-dep", workspaceID, domain.EntityTypeService, "Dependency", domain.EntityStatusActive, ""),
		mustEntity(t, "system-cloud", workspaceID, domain.EntityTypeEngineeringSystem, "Device Cloud", domain.EntityStatusActive, ""),
		mustEntity(t, "api-session", workspaceID, domain.EntityTypeAPI, "Session API", domain.EntityStatusActive, ""),
	} {
		if err := store.PutEntity(ctx, entity); err != nil {
			t.Fatal(err)
		}
	}

	source := &fakeSource{
		repository: githubsource.Repository{Locator: "github://acme/device-gateway", DefaultBranch: "main"},
		commit:     githubsource.Commit{SHA: reconcileSHA1, RepositoryLocator: "github://acme/device-gateway"},
		blob:       githubsource.ManifestBlob{BlobSHA: "blob-one", Content: manifestV1("Device Gateway", "active", true, true)},
	}
	reconciler, err := New(store, source, func() time.Time { return time.Date(2026, 8, 28, 16, 0, 0, 0, time.UTC) })
	if err != nil {
		t.Fatal(err)
	}
	first, err := reconciler.Reconcile(ctx, Input{WorkspaceID: workspaceID, Locator: "github://acme/device-gateway"})
	if err != nil {
		t.Fatal(err)
	}
	if first.CommitSHA != reconcileSHA1 || first.EntityID != "service-gateway" || first.ManifestChecksum == "" || len(first.UpsertedEdgeIDs) != 3 {
		t.Fatalf("first result = %+v", first)
	}
	if len(first.Unresolved) != 1 || first.Unresolved[0].TargetID != "thing-model-coffee" || first.Unresolved[0].Reason != "not_found" {
		t.Fatalf("unresolved = %+v", first.Unresolved)
	}
	binding, err := store.GetSourceBinding(ctx, workspaceID, first.SourceBindingID)
	if err != nil {
		t.Fatal(err)
	}
	if binding.Authority() != domain.AuthorityAuthoritative || binding.Provenance().Revision() != reconcileSHA1 {
		t.Fatalf("binding = %+v provenance=%+v", binding, binding.Provenance())
	}

	if _, err := reconciler.Reconcile(ctx, Input{WorkspaceID: workspaceID, Locator: "github://acme/device-gateway"}); err != nil {
		t.Fatalf("idempotent reconcile: %v", err)
	}

	if err := store.PutEntity(ctx, mustEntity(t, "thing-model-coffee", workspaceID, domain.EntityTypeThingModel, "Coffee Thing Model", domain.EntityStatusActive, "")); err != nil {
		t.Fatal(err)
	}
	source.commit.SHA = reconcileSHA2
	source.blob.BlobSHA = "blob-two"
	source.blob.Content = manifestV1("Device Gateway v2", "active", false, true)
	second, err := reconciler.Reconcile(ctx, Input{WorkspaceID: workspaceID, Locator: "github://acme/device-gateway", Ref: "main"})
	if err != nil {
		t.Fatal(err)
	}
	if len(second.DeletedEdgeIDs) != 1 || len(second.UpsertedEdgeIDs) != 3 || len(second.Unresolved) != 0 {
		t.Fatalf("second result = %+v", second)
	}
	entity, err := store.GetEntity(ctx, workspaceID, "service-gateway")
	if err != nil {
		t.Fatal(err)
	}
	if entity.Name() != "Device Gateway v2" {
		t.Fatalf("source-owned entity was not refreshed: %+v", entity)
	}
	node, _ := domain.NewNodeRef(domain.NodeKindEngineeringEntity, "service-gateway")
	edges, err := store.ListThreadEdges(ctx, workspaceID, node)
	if err != nil {
		t.Fatal(err)
	}
	var sourceEdges int
	for _, edge := range edges {
		if edge.Provenance().SourceType() == manifestEdgeSourceType {
			sourceEdges++
			if edge.Provenance().Revision() != reconcileSHA2 {
				t.Fatalf("edge %s revision = %s", edge.ID(), edge.Provenance().Revision())
			}
			if edge.Relation() == domain.RelationDependsOn {
				t.Fatalf("removed dependency edge survived: %+v", edge)
			}
		}
	}
	if sourceEdges != 3 {
		t.Fatalf("source edge count = %d", sourceEdges)
	}
}

func TestReconcilePreservesUnownedCanonicalEntityAndRejectsTypeDrift(t *testing.T) {
	store, closeStore := newReconcileStore(t)
	defer closeStore()
	ctx := context.Background()
	workspaceID := "workspace-a"
	if err := store.PutEntity(ctx, mustEntity(t, "service-gateway", workspaceID, domain.EntityTypeService, "Manual Name", domain.EntityStatusActive, "team:manual")); err != nil {
		t.Fatal(err)
	}
	source := &fakeSource{
		repository: githubsource.Repository{Locator: "github://acme/device-gateway", DefaultBranch: "main"},
		commit:     githubsource.Commit{SHA: reconcileSHA1},
		blob:       githubsource.ManifestBlob{BlobSHA: "blob", Content: manifestV1("Device Gateway", "active", false, false)},
	}
	reconciler, err := New(store, source, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reconciler.Reconcile(ctx, Input{WorkspaceID: workspaceID, Locator: source.repository.Locator}); !errors.Is(err, ErrCanonicalEntityConflict) {
		t.Fatalf("manual entity conflict = %v", err)
	}

	matching := mustEntity(t, "service-gateway", workspaceID, domain.EntityTypeService, "Device Gateway", domain.EntityStatusActive, "team:iot")
	if err := store.PutEntity(ctx, matching); err != nil {
		t.Fatal(err)
	}
	if _, err := reconciler.Reconcile(ctx, Input{WorkspaceID: workspaceID, Locator: source.repository.Locator}); err != nil {
		t.Fatalf("claim matching entity: %v", err)
	}
	source.commit.SHA = reconcileSHA2
	source.blob.Content = []byte(strings.ReplaceAll(string(source.blob.Content), "type: service", "type: application"))
	if _, err := reconciler.Reconcile(ctx, Input{WorkspaceID: workspaceID, Locator: source.repository.Locator}); !errors.Is(err, ErrCanonicalEntityConflict) {
		t.Fatalf("source-owned type drift = %v", err)
	}
}

func TestReconcileRejectsManifestSourceMismatchAndInterfaceTypeMismatchIsExplicit(t *testing.T) {
	store, closeStore := newReconcileStore(t)
	defer closeStore()
	ctx := context.Background()
	workspaceID := "workspace-a"
	if err := store.PutEntity(ctx, mustEntity(t, "api-session", workspaceID, domain.EntityTypeService, "Wrong Type", domain.EntityStatusActive, "")); err != nil {
		t.Fatal(err)
	}
	source := &fakeSource{
		repository: githubsource.Repository{Locator: "github://acme/device-gateway", DefaultBranch: "main"},
		commit:     githubsource.Commit{SHA: reconcileSHA1},
		blob:       githubsource.ManifestBlob{BlobSHA: "blob", Content: manifestV1("Device Gateway", "active", false, true)},
	}
	reconciler, _ := New(store, source, time.Now)
	result, err := reconciler.Reconcile(ctx, Input{WorkspaceID: workspaceID, Locator: source.repository.Locator})
	if err != nil {
		t.Fatal(err)
	}
	var foundTypeMismatch bool
	for _, unresolved := range result.Unresolved {
		if unresolved.TargetID == "api-session" && unresolved.Reason == "type_mismatch" && unresolved.ExpectedType == "api" {
			foundTypeMismatch = true
		}
	}
	if !foundTypeMismatch {
		t.Fatalf("unresolved = %+v", result.Unresolved)
	}

	source.repository.Locator = "github://acme/other"
	if _, err := reconciler.Reconcile(ctx, Input{WorkspaceID: workspaceID, Locator: source.repository.Locator}); !errors.Is(err, ErrSourceSnapshotInvalid) {
		t.Fatalf("source mismatch error = %v", err)
	}
}

func newReconcileStore(t *testing.T) (*persistence.Store, func()) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:engineering-reconcile?mode=memory&cache=shared")
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

func mustEntity(t *testing.T, id, workspaceID string, entityType domain.EntityType, name string, status domain.EntityStatus, owner string) domain.EngineeringEntity {
	t.Helper()
	value, err := domain.NewEngineeringEntity(id, workspaceID, entityType, name, status, owner)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func manifestV1(name, status string, includeDependency, includeInterfaces bool) []byte {
	var builder strings.Builder
	builder.WriteString("schema_version: v1\n")
	builder.WriteString("entity:\n  id: service-gateway\n  type: service\n  name: " + name + "\n  status: " + status + "\n  owner_ref: team:iot\n")
	builder.WriteString("source:\n  type: github\n  locator: github://acme/device-gateway\n")
	if includeDependency {
		builder.WriteString("dependencies:\n  - service-dep\n")
	}
	builder.WriteString("relations:\n  - relation: part_of\n    target: system-cloud\n")
	if includeInterfaces {
		builder.WriteString("interfaces:\n  - id: api-session\n    type: api\n    direction: provides\n  - id: thing-model-coffee\n    type: thing_model\n    direction: uses\n")
	}
	return []byte(builder.String())
}
