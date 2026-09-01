package contextcompiler

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/engineering/contract"
	"github.com/hvritual/workspace/internal/modules/engineering/internal/domain"
	persistence "github.com/hvritual/workspace/internal/modules/engineering/internal/infrastructure/sqlite"
	"github.com/hvritual/workspace/internal/modules/engineering/internal/scope"
	_ "modernc.org/sqlite"
)

type publishedReader []contract.ContextReferenceCandidate

func (r publishedReader) ListPublishedContextReferences(context.Context, string, []string) ([]contract.ContextReferenceCandidate, error) {
	return append([]contract.ContextReferenceCandidate(nil), r...), nil
}

type incidentReader []contract.ContextReferenceCandidate

func (r incidentReader) ListIncidentContextReferences(context.Context, string, []string) ([]contract.ContextReferenceCandidate, error) {
	return append([]contract.ContextReferenceCandidate(nil), r...), nil
}

func TestCompileBuildsReproduciblePinnedContextPack(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 29, 0, 0, 0, 0, time.UTC)
	store, closeDB := newCompilerStore(t)
	defer closeDB()
	seedCompilerGraph(t, ctx, store, now)

	governed := publishedReader{
		{Kind: "architecture", ID: "arch-service-1", Revision: "r3", Checksum: "sha256:arch", EntityIDs: []string{"service-1"}, UpdatedAt: now.Add(-time.Hour), EstimatedTokens: 450},
		{Kind: "standard", ID: "std-go-1", Revision: "r8", Checksum: "sha256:std", Global: true, UpdatedAt: now.Add(-2 * time.Hour), EstimatedTokens: 300},
		{Kind: "knowledge", ID: "old-note", Revision: "r1", Checksum: "sha256:old", EntityIDs: []string{"service-1"}, UpdatedAt: now.Add(-90 * 24 * time.Hour), EstimatedTokens: 600},
	}
	incidents := incidentReader{{Kind: "incident", ID: "inc-9", Revision: "closed-r2", Checksum: "sha256:incident", EntityIDs: []string{"service-1"}, UpdatedAt: now.Add(-3 * time.Hour), EstimatedTokens: 350}}
	resolver, err := scope.New(store, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	compiler, err := New(store, resolver, governed, incidents, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}

	request := contract.CompileContextRequest{
		WorkspaceID: "workspace-1", PackID: "context-1", WorkItem: contract.NodeRef{Kind: "task", ID: "task-1"}, WorkItemRevision: "7",
		Policy: contract.ContextCompilePolicy{
			Version: "context-policy-v1", SourceStaleAfter: 24 * time.Hour, KnowledgeStaleAfter: 30 * 24 * time.Hour,
			MaxReferences: 8, MaxEstimatedTokens: 4096, MaxRecentChanges: 4,
		},
	}
	result, err := compiler.Compile(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if result.Pack.WorkItem.ID != "task-1" || result.Pack.WorkItemRevision != "7" || result.Pack.PolicyVersion != "context-policy-v1" || result.Pack.Checksum == "" {
		t.Fatalf("pack = %+v", result.Pack)
	}
	if len(result.ScopeEntityIDs) != 2 || result.ScopeEntityIDs[0] != "repo-1" || result.ScopeEntityIDs[1] != "service-1" {
		t.Fatalf("scope entities = %#v", result.ScopeEntityIDs)
	}
	assertReference(t, result.Pack.References, "engineering_entity", "service-1")
	assertReference(t, result.Pack.References, "engineering_entity", "repo-1")
	assertReference(t, result.Pack.References, "standard", "std-go-1")
	assertReference(t, result.Pack.References, "architecture", "arch-service-1")
	assertReference(t, result.Pack.References, "change", "change-accepted")
	assertReference(t, result.Pack.References, "incident", "inc-9")
	assertNoReference(t, result.Pack.References, "change", "change-proposed")
	if !hasWarning(result.Warnings, "reference_stale", "old-note") {
		t.Fatalf("warnings = %#v, want stale knowledge warning", result.Warnings)
	}

	replayed, err := compiler.Compile(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.Pack.Checksum != result.Pack.Checksum || !replayed.Pack.CreatedAt.Equal(result.Pack.CreatedAt) {
		t.Fatalf("replayed pack drifted: first=%+v replay=%+v", result.Pack, replayed.Pack)
	}

	request.PackID = "context-2"
	reordered := publishedReader{governed[2], governed[1], governed[0]}
	compiler2, err := New(store, resolver, reordered, incidents, func() time.Time { return now.Add(time.Hour) })
	if err != nil {
		t.Fatal(err)
	}
	second, err := compiler2.Compile(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if second.Pack.Checksum != result.Pack.Checksum {
		t.Fatalf("same semantic inputs changed checksum: %s != %s", second.Pack.Checksum, result.Pack.Checksum)
	}
}

func TestCompileConflictsWhenExistingPackContentChanges(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 29, 1, 0, 0, 0, time.UTC)
	store, closeDB := newCompilerStore(t)
	defer closeDB()
	seedCompilerGraph(t, ctx, store, now)
	resolver, _ := scope.New(store, func() time.Time { return now })
	compiler, _ := New(store, resolver, nil, nil, func() time.Time { return now })
	request := contract.CompileContextRequest{
		WorkspaceID: "workspace-1", PackID: "context-1", WorkItem: contract.NodeRef{Kind: "task", ID: "task-1"}, WorkItemRevision: "7",
		Policy: contract.ContextCompilePolicy{Version: "v1", MaxEstimatedTokens: 4096},
	}
	if _, err := compiler.Compile(ctx, request); err != nil {
		t.Fatal(err)
	}
	request.Policy.Version = "v2"
	if _, err := compiler.Compile(ctx, request); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("error = %v, want context-pack conflict", err)
	}
}

func TestCompileEnforcesBudgetForRequiredSourcePins(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 29, 2, 0, 0, 0, time.UTC)
	store, closeDB := newCompilerStore(t)
	defer closeDB()
	seedCompilerGraph(t, ctx, store, now)
	resolver, _ := scope.New(store, func() time.Time { return now })
	compiler, _ := New(store, resolver, nil, nil, func() time.Time { return now })
	_, err := compiler.Compile(ctx, contract.CompileContextRequest{
		WorkspaceID: "workspace-1", PackID: "tiny", WorkItem: contract.NodeRef{Kind: "task", ID: "task-1"}, WorkItemRevision: "7",
		Policy: contract.ContextCompilePolicy{Version: "tiny", MaxEstimatedTokens: 32, MaxReferences: 8},
	})
	if !errors.Is(err, ErrBudgetTooSmall) {
		t.Fatalf("error = %v, want required-source budget error", err)
	}
}

func TestCompileWarnsAndSkipsInvalidExternalReferences(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 29, 3, 0, 0, 0, time.UTC)
	store, closeDB := newCompilerStore(t)
	defer closeDB()
	seedCompilerGraph(t, ctx, store, now)
	resolver, _ := scope.New(store, func() time.Time { return now })
	governed := publishedReader{
		{Kind: "architecture", ID: "no-revision", Checksum: "sha256:x", EntityIDs: []string{"service-1"}},
		{Kind: "architecture", ID: "other-service", Revision: "r1", Checksum: "sha256:y", EntityIDs: []string{"outside"}},
	}
	compiler, _ := New(store, resolver, governed, nil, func() time.Time { return now })
	result, err := compiler.Compile(ctx, contract.CompileContextRequest{
		WorkspaceID: "workspace-1", PackID: "warnings", WorkItem: contract.NodeRef{Kind: "task", ID: "task-1"}, WorkItemRevision: "7",
		Policy: contract.ContextCompilePolicy{Version: "warnings", MaxEstimatedTokens: 4096},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !hasWarning(result.Warnings, "reference_invalid", "no-revision") || !hasWarning(result.Warnings, "reference_invalid", "other-service") {
		t.Fatalf("warnings = %#v", result.Warnings)
	}
}

func newCompilerStore(t *testing.T) (*persistence.Store, func()) {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "context-compiler.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := persistence.Migrate(context.Background(), db); err != nil {
		db.Close()
		t.Fatal(err)
	}
	store, err := persistence.New(db)
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	return store, func() { _ = db.Close() }
}

func seedCompilerGraph(t *testing.T, ctx context.Context, store *persistence.Store, now time.Time) {
	t.Helper()
	for _, entity := range []domain.EngineeringEntity{
		mustEntity(t, "service-1", "workspace-1", domain.EntityTypeService, "Device Gateway"),
		mustEntity(t, "repo-1", "workspace-1", domain.EntityTypeRepository, "device-gateway"),
	} {
		if err := store.PutEntity(ctx, entity); err != nil {
			t.Fatal(err)
		}
	}
	workspaceProv := mustProvenance(t, "workspace", "workspace://workspace-1/tasks/task-1", "7", now)
	sourceProv := mustProvenance(t, "github", "github://acme/device-gateway", "abc123", now.Add(-time.Hour))
	observedProv := mustProvenance(t, "github", "github://acme/device-gateway", "abc122", now.Add(-30*time.Minute))
	task, _ := domain.NewNodeRef(domain.NodeKindTask, "task-1")
	service, _ := domain.NewNodeRef(domain.NodeKindEngineeringEntity, "service-1")
	repository, _ := domain.NewNodeRef(domain.NodeKindEngineeringEntity, "repo-1")
	workEdge, _ := domain.NewThreadEdge("work-1", "workspace-1", task, domain.RelationAffects, service, domain.AuthorityAuthoritative, workspaceProv)
	implements, _ := domain.NewThreadEdge("implements-1", "workspace-1", repository, domain.RelationImplements, service, domain.AuthorityAuthoritative, sourceProv)
	if err := store.PutThreadEdge(ctx, workEdge); err != nil {
		t.Fatal(err)
	}
	if err := store.PutThreadEdge(ctx, implements); err != nil {
		t.Fatal(err)
	}
	bindingService, _ := domain.NewSourceBinding("source-service", "workspace-1", "service-1", sourceProv, domain.AuthorityAuthoritative)
	bindingServiceObserved, _ := domain.NewSourceBinding("source-service-observed", "workspace-1", "service-1", observedProv, domain.AuthorityObserved)
	bindingRepo, _ := domain.NewSourceBinding("source-repo", "workspace-1", "repo-1", sourceProv, domain.AuthorityAuthoritative)
	for _, binding := range []domain.SourceBinding{bindingServiceObserved, bindingService, bindingRepo} {
		if err := store.PutSourceBinding(ctx, binding); err != nil {
			t.Fatal(err)
		}
	}

	accepted := mustChange(t, "change-accepted", now.Add(-2*time.Hour))
	accepted, err := accepted.Accept(now.Add(-90 * time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	proposed := mustChange(t, "change-proposed", now.Add(-time.Hour))
	if err := store.PutChange(ctx, accepted); err != nil {
		t.Fatal(err)
	}
	if err := store.PutChange(ctx, proposed); err != nil {
		t.Fatal(err)
	}
}

func mustEntity(t *testing.T, id, workspaceID string, kind domain.EntityType, name string) domain.EngineeringEntity {
	t.Helper()
	value, err := domain.NewEngineeringEntity(id, workspaceID, kind, name, domain.EntityStatusActive, "team:iot")
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func mustProvenance(t *testing.T, sourceType, locator, revision string, observedAt time.Time) domain.Provenance {
	t.Helper()
	value, err := domain.NewProvenance(sourceType, locator, revision, observedAt)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func mustChange(t *testing.T, id string, createdAt time.Time) domain.Change {
	t.Helper()
	work, _ := domain.NewNodeRef(domain.NodeKindTask, "task-1")
	artifact, _ := domain.NewArtifactRef("pull_request", "github://acme/device-gateway/pull/7", "abc123")
	prov := mustProvenance(t, "github", "github://acme/device-gateway/pull/7", "abc123", createdAt)
	value, err := domain.NewChange(id, "workspace-1", "project-1", "requirement-1", &work, "", "Update reconnect handling", []string{"service-1"}, []domain.ArtifactRef{artifact}, prov, createdAt)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func assertReference(t *testing.T, values []contract.ContextReference, kind, id string) {
	t.Helper()
	for _, value := range values {
		if value.Kind == kind && value.ID == id {
			if value.Revision == "" || value.Checksum == "" {
				t.Fatalf("reference %s/%s is not pinned: %+v", kind, id, value)
			}
			return
		}
	}
	t.Fatalf("missing reference %s/%s in %#v", kind, id, values)
}

func assertNoReference(t *testing.T, values []contract.ContextReference, kind, id string) {
	t.Helper()
	for _, value := range values {
		if value.Kind == kind && value.ID == id {
			t.Fatalf("unexpected reference %s/%s", kind, id)
		}
	}
}

func hasWarning(values []contract.ContextCompileWarning, code, refID string) bool {
	for _, value := range values {
		if value.Code == code && (refID == "" || value.RefID == refID) {
			return true
		}
	}
	return false
}
