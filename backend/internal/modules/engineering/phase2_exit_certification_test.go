package engineering

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/controlplane"
	"github.com/hvritual/workspace/internal/modules/engineering/contract"
	"github.com/hvritual/workspace/internal/modules/engineering/internal/application"
	"github.com/hvritual/workspace/internal/modules/engineering/internal/changeproposal"
	"github.com/hvritual/workspace/internal/modules/engineering/internal/contextcompiler"
	"github.com/hvritual/workspace/internal/modules/engineering/internal/domain"
	persistence "github.com/hvritual/workspace/internal/modules/engineering/internal/infrastructure/sqlite"
	"github.com/hvritual/workspace/internal/modules/engineering/internal/reconcile"
	"github.com/hvritual/workspace/internal/modules/engineering/internal/scope"
	githubsource "github.com/hvritual/workspace/internal/modules/engineering/internal/source/github"
	_ "modernc.org/sqlite"
)

const phase2Manifest = `schema_version: v1
entity:
  id: service-device-gateway
  type: service
  name: Device Gateway
  status: active
  owner_ref: team:iot-platform
source:
  type: github
  locator: github://acme/device-gateway
dependencies:
  - service-device-session
knowledge:
  standards:
    - docs/standards/mqtt.md
`

type phase2Source struct {
	repository githubsource.Repository
	commit     githubsource.Commit
	manifest   githubsource.ManifestBlob
}

func (s phase2Source) GetRepository(_ context.Context, locator string) (githubsource.Repository, error) {
	if locator != s.repository.Locator {
		return githubsource.Repository{}, githubsource.ErrNotFound
	}
	return s.repository, nil
}

func (s phase2Source) ResolveCommit(_ context.Context, locator, _ string) (githubsource.Commit, error) {
	if locator != s.repository.Locator {
		return githubsource.Commit{}, githubsource.ErrNotFound
	}
	return s.commit, nil
}

func (s phase2Source) ReadEngineeringManifestAtCommit(_ context.Context, locator, commitSHA string) (githubsource.ManifestBlob, error) {
	if locator != s.repository.Locator || commitSHA != s.commit.SHA {
		return githubsource.ManifestBlob{}, githubsource.ErrNotFound
	}
	return s.manifest, nil
}

type phase2PullSource struct {
	pull githubsource.PullRequest
}

func (s phase2PullSource) GetPullRequest(_ context.Context, locator string, number int) (githubsource.PullRequest, error) {
	if locator != s.pull.RepositoryLocator || number != s.pull.Number {
		return githubsource.PullRequest{}, githubsource.ErrNotFound
	}
	return s.pull, nil
}

type phase2PublishedRefs struct {
	refs []contract.VersionedContextReference
}

func (p phase2PublishedRefs) ListPublishedContextReferences(_ context.Context, _ string, entityIDs []string) ([]contract.VersionedContextReference, error) {
	allowed := make(map[string]struct{}, len(entityIDs))
	for _, id := range entityIDs {
		allowed[id] = struct{}{}
	}
	result := make([]contract.VersionedContextReference, 0, len(p.refs))
	for _, ref := range p.refs {
		for _, entityID := range ref.EntityIDs {
			if _, ok := allowed[entityID]; ok {
				result = append(result, ref)
				break
			}
		}
	}
	return result, nil
}

func TestPhase2ExitPreCertificationDigitalThread(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 9, 1, 7, 30, 0, 0, time.UTC)
	workspaceID := "workspace-1"
	repositoryLocator := "github://acme/device-gateway"
	commitSHA := strings.Repeat("a", 40)
	blobSHA := strings.Repeat("b", 40)
	mergeSHA := strings.Repeat("c", 40)

	engineeringDB, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "engineering.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer engineeringDB.Close()
	engineeringDB.SetMaxOpenConns(1)
	if err := MigrateSqlite(ctx, engineeringDB); err != nil {
		t.Fatal(err)
	}
	repository, err := persistence.New(engineeringDB)
	if err != nil {
		t.Fatal(err)
	}

	dependency, err := domain.NewEngineeringEntity("service-device-session", workspaceID, domain.EntityTypeService, "Device Session", domain.EntityStatusActive, "team:iot-platform")
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.PutEntity(ctx, dependency); err != nil {
		t.Fatal(err)
	}

	source := phase2Source{
		repository: githubsource.Repository{Locator: repositoryLocator, FullName: "acme/device-gateway", DefaultBranch: "main", UpdatedAt: now},
		commit:     githubsource.Commit{RepositoryLocator: repositoryLocator, SHA: commitSHA, TreeSHA: strings.Repeat("d", 40), HTMLURL: "https://github.com/acme/device-gateway/commit/" + commitSHA, CommittedAt: now},
		manifest:   githubsource.ManifestBlob{RepositoryLocator: repositoryLocator, CommitSHA: commitSHA, Path: "engineering.yaml", BlobSHA: blobSHA, Content: []byte(phase2Manifest)},
	}
	reconciler, err := reconcile.New(repository, source, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	reconciled, err := reconciler.Reconcile(ctx, reconcile.Input{WorkspaceID: workspaceID, Locator: repositoryLocator, Ref: "main"})
	if err != nil {
		t.Fatal(err)
	}
	if reconciled.EntityID != "service-device-gateway" || reconciled.CommitSHA != commitSHA || len(reconciled.UpsertedEdgeIDs) != 1 || len(reconciled.Unresolved) != 0 {
		t.Fatalf("reconcile result = %#v", reconciled)
	}
	binding, err := repository.GetSourceBinding(ctx, workspaceID, reconciled.SourceBindingID)
	if err != nil {
		t.Fatal(err)
	}
	if binding.Authority() != domain.AuthorityAuthoritative || binding.Provenance().Revision() != commitSHA {
		t.Fatalf("source binding = authority %s revision %q", binding.Authority(), binding.Provenance().Revision())
	}

	membership := contract.WorkspaceRoleResolverFunc(func(context.Context, string, string) (string, bool, error) {
		return "owner", true, nil
	})
	service, err := application.New(repository, membership, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	workLink, err := service.PutWorkLink(ctx, workspaceID, contract.WorkLinkTask, "task-1", "service-device-gateway")
	if err != nil {
		t.Fatal(err)
	}
	if workLink.Relation != "affects" || workLink.Authority != "authoritative" || workLink.Provenance.SourceType != "workspace" {
		t.Fatalf("work link = %#v", workLink)
	}
	workItem, err := domain.NewNodeRef(domain.NodeKindTask, "task-1")
	if err != nil {
		t.Fatal(err)
	}

	pull := githubsource.PullRequest{RepositoryLocator: repositoryLocator, Number: 42, State: "closed", Merged: true, MergeCommitSHA: mergeSHA, HeadSHA: strings.Repeat("e", 40), BaseSHA: commitSHA, HTMLURL: "https://github.com/acme/device-gateway/pull/42", UpdatedAt: now}
	proposer, err := changeproposal.New(repository, phase2PullSource{pull: pull}, func() time.Time { return now.Add(time.Minute) })
	if err != nil {
		t.Fatal(err)
	}
	proposed, err := proposer.Propose(ctx, changeproposal.Input{WorkspaceID: workspaceID, RepositoryLocator: repositoryLocator, PullRequestNumber: 42, ProjectID: "project-1", RequirementID: "requirement-1", WorkItem: workItem})
	if err != nil {
		t.Fatal(err)
	}
	if proposed.Change.Status() != domain.ChangeStatusProposed || proposed.Change.AcceptedAt() != nil {
		t.Fatalf("PR projection must remain proposed: status=%s accepted_at=%v", proposed.Change.Status(), proposed.Change.AcceptedAt())
	}

	published := phase2PublishedRefs{refs: []contract.VersionedContextReference{{
		Kind: "standard", ID: "std-mqtt", Revision: "std-v3", Checksum: strings.Repeat("f", 64),
		EntityIDs: []string{"service-device-gateway"}, EstimatedTokens: 128, UpdatedAt: now,
	}}}
	resolver, err := scope.New(repository, func() time.Time { return now.Add(10 * time.Minute) })
	if err != nil {
		t.Fatal(err)
	}
	compiler, err := contextcompiler.New(repository, resolver, published, nil, func() time.Time { return now.Add(10 * time.Minute) })
	if err != nil {
		t.Fatal(err)
	}
	policy := contract.ContextPolicy{
		Version: "context-v1", MaxDepth: 2, MaxEntities: 16,
		SourceStaleAfter: 24 * time.Hour, KnowledgeStaleAfter: 24 * time.Hour,
		MaxReferences: 16, MaxEstimatedTokens: 4096, MaxRecentChanges: 8,
	}
	compile := func(packID string) contract.CompileContextResult {
		result, compileErr := compiler.CompileContext(ctx, contract.CompileContextRequest{
			WorkspaceID: workspaceID, PackID: packID, WorkItemKind: "task", WorkItemID: "task-1", WorkItemRevision: "7", Policy: policy,
		})
		if compileErr != nil {
			t.Fatalf("compile %s: %v", packID, compileErr)
		}
		return result
	}
	first := compile("context-pack-1")
	second := compile("context-pack-2")
	if first.Pack.Checksum == "" || first.Pack.Checksum != second.Pack.Checksum {
		t.Fatalf("ContextPack checksum not reproducible: first=%q second=%q", first.Pack.Checksum, second.Pack.Checksum)
	}
	if !hasContextReference(first.Pack.References, "engineering_entity", "service-device-gateway", commitSHA) {
		t.Fatalf("ContextPack missing pinned source revision: %#v", first.Pack.References)
	}
	if !hasContextReference(first.Pack.References, "standard", "std-mqtt", "std-v3") {
		t.Fatalf("ContextPack missing governed standard: %#v", first.Pack.References)
	}
	for _, ref := range first.Pack.References {
		if ref.Kind == "change" && ref.ID == proposed.Change.ID() {
			t.Fatalf("proposed Change leaked into governed ContextPack: %#v", ref)
		}
	}

	staleResolver, err := scope.New(repository, func() time.Time { return now.Add(48 * time.Hour) })
	if err != nil {
		t.Fatal(err)
	}
	staleScope, err := staleResolver.Resolve(ctx, scope.Input{WorkspaceID: workspaceID, WorkItem: workItem, Policy: scope.Policy{MaxDepth: 2, MaxEntities: 16, SourceStaleAfter: time.Hour}})
	if err != nil {
		t.Fatal(err)
	}
	if !hasScopeWarning(staleScope.Warnings, "stale_source", reconciled.SourceBindingID) {
		t.Fatalf("stale source was not surfaced explicitly: %#v", staleScope.Warnings)
	}

	runtimeRepository, err := controlplane.OpenSQLite(ctx, filepath.Join(t.TempDir(), "runtime.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer runtimeRepository.Close()
	kernelStore, err := controlplane.KernelStoreFrom(runtimeRepository)
	if err != nil {
		t.Fatal(err)
	}
	kernel, err := controlplane.NewDeliveryKernel(kernelStore, func() time.Time { return now.Add(20 * time.Minute) }, func(context.Context, controlplane.Actor, string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	frozen := first.Pack
	reader := controlplane.RuntimeContextPackReaderFunc(func(_ context.Context, requestedWorkspace, packID string) (controlplane.FrozenRuntimeContextPack, error) {
		if requestedWorkspace != frozen.WorkspaceID || packID != frozen.ID {
			return controlplane.FrozenRuntimeContextPack{}, controlplane.ErrRuntimeContextPackNotFound
		}
		return controlplane.FrozenRuntimeContextPack{
			ID: frozen.ID, WorkspaceID: frozen.WorkspaceID, WorkItemKind: frozen.WorkItem.Kind, WorkItemID: frozen.WorkItem.ID,
			WorkItemRevision: frozen.WorkItemRevision, PolicyVersion: frozen.PolicyVersion, Checksum: frozen.Checksum,
		}, nil
	})
	binder, err := controlplane.NewRunContextBinder(kernel, reader)
	if err != nil {
		t.Fatal(err)
	}
	actor := controlplane.Actor{ID: "user-1", WorkspaceID: workspaceID, Kind: controlplane.ActorHuman}
	runRequest := controlplane.RunExecutionContextRequest{
		ContextPackID: frozen.ID, ContextPackChecksum: frozen.Checksum, AgentReleaseID: "release-7",
		SkillVersions: []controlplane.SkillVersionPin{{SkillID: "skill-go", VersionID: "v3"}},
	}
	queued, err := binder.QueueRun(ctx, actor, "command-run-1", "project-1", 0, "run-1", "worktree://task-1", nil, 2, runRequest)
	if err != nil {
		t.Fatal(err)
	}
	if queued.Head != 3 || len(queued.Events) != 3 {
		t.Fatalf("contextual run append = %#v", queued)
	}
	projection, err := kernel.Replay(ctx, workspaceID, "project-1")
	if err != nil {
		t.Fatal(err)
	}
	pinned, found, err := controlplane.ResolveRunExecutionContext(projection, "run-1")
	if err != nil || !found {
		t.Fatalf("resolve run context: found=%v err=%v", found, err)
	}
	wantSkills := []controlplane.SkillVersionPin{{SkillID: "skill-go", VersionID: "v3"}}
	if pinned.ContextPackID != frozen.ID || pinned.ContextPackChecksum != frozen.Checksum || pinned.WorkItemRevision != "7" || pinned.ContextPolicy != "context-v1" || pinned.AgentReleaseID != "release-7" || !reflect.DeepEqual(pinned.SkillVersions, wantSkills) {
		t.Fatalf("run context pins = %#v", pinned)
	}
}

func hasContextReference(values []contract.ContextReference, kind, id, revision string) bool {
	for _, value := range values {
		if value.Kind == kind && value.ID == id && value.Revision == revision && value.Checksum != "" {
			return true
		}
	}
	return false
}

func hasScopeWarning(values []scope.Warning, code, bindingID string) bool {
	for _, value := range values {
		if value.Code == code && (bindingID == "" || value.BindingID == bindingID) {
			return true
		}
	}
	return false
}

var _ reconcile.Source = phase2Source{}
var _ changeproposal.Source = phase2PullSource{}
var _ contract.PublishedContextReferenceReader = phase2PublishedRefs{}
var _ = errors.Is
