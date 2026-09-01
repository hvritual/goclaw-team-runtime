package changeproposal

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/engineering/internal/domain"
	persistence "github.com/hvritual/workspace/internal/modules/engineering/internal/infrastructure/sqlite"
	githubsource "github.com/hvritual/workspace/internal/modules/engineering/internal/source/github"
	_ "modernc.org/sqlite"
)

const proposalMergeSHA = "abcdefabcdefabcdefabcdefabcdefabcdefabcd"

type fakePullSource struct {
	pull githubsource.PullRequest
	err  error
}

func (s fakePullSource) GetPullRequest(context.Context, string, int) (githubsource.PullRequest, error) {
	return s.pull, s.err
}

func TestProposeCreatesOnlyProposedChangeFromMergedPRWithExplicitWorkLink(t *testing.T) {
	store, closeStore := newProposalStore(t)
	defer closeStore()
	ctx := context.Background()
	workspaceID := "workspace-a"
	entity := proposalEntity(t, "service-gateway", workspaceID)
	if err := store.PutEntity(ctx, entity); err != nil {
		t.Fatal(err)
	}
	bindingProvenance, _ := domain.NewProvenance("github", "github://acme/device-gateway", proposalMergeSHA, time.Now())
	binding, _ := domain.NewSourceBinding("source-binding", workspaceID, entity.ID(), bindingProvenance, domain.AuthorityAuthoritative)
	if err := store.PutSourceBinding(ctx, binding); err != nil {
		t.Fatal(err)
	}
	work, _ := domain.NewNodeRef(domain.NodeKindTask, "task-one")
	entityNode, _ := domain.NewNodeRef(domain.NodeKindEngineeringEntity, entity.ID())
	workProvenance, _ := domain.NewProvenance("workspace", "workspace://workspace-a/work/task/task-one", "", time.Now())
	workEdge, _ := domain.NewThreadEdge("work-link", workspaceID, work, domain.RelationAffects, entityNode, domain.AuthorityAuthoritative, workProvenance)
	if err := store.PutThreadEdge(ctx, workEdge); err != nil {
		t.Fatal(err)
	}

	source := fakePullSource{pull: githubsource.PullRequest{
		RepositoryLocator: "github://acme/device-gateway", Number: 7, Merged: true, MergeCommitSHA: proposalMergeSHA,
		HeadSHA: proposalMergeSHA, BaseSHA: "1111111111111111111111111111111111111111",
	}}
	now := time.Date(2026, 8, 28, 17, 0, 0, 0, time.UTC)
	proposer, err := New(store, source, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	input := Input{WorkspaceID: workspaceID, RepositoryLocator: "github://acme/device-gateway", PullRequestNumber: 7, ProjectID: "project-one", RequirementID: "requirement-one", WorkItem: work}
	first, err := proposer.Propose(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if first.Change.Status() != domain.ChangeStatusProposed || first.Change.AcceptedAt() != nil || first.Change.RunID() != "" {
		t.Fatalf("proposed change = %+v", first.Change)
	}
	if affected := first.Change.AffectedEntityIDs(); len(affected) != 1 || affected[0] != entity.ID() {
		t.Fatalf("affected = %#v", affected)
	}
	artifacts := first.Change.Artifacts()
	if len(artifacts) != 2 || artifacts[0].Kind() != "pull_request" || artifacts[1].Kind() != "commit" || artifacts[1].Revision() != proposalMergeSHA {
		t.Fatalf("artifacts = %#v", artifacts)
	}

	second, err := proposer.Propose(ctx, input)
	if err != nil || second.Change.ID() != first.Change.ID() {
		t.Fatalf("idempotent propose = %+v err=%v", second.Change, err)
	}
	accepted, err := first.Change.Accept(now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.PutChange(ctx, accepted); err != nil {
		t.Fatal(err)
	}
	afterAcceptance, err := proposer.Propose(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if afterAcceptance.Change.Status() != domain.ChangeStatusAccepted {
		t.Fatalf("proposal rewrote accepted change: %s", afterAcceptance.Change.Status())
	}
}

func TestProposeRequiresMergedPRSourceBindingAndWorkLink(t *testing.T) {
	store, closeStore := newProposalStore(t)
	defer closeStore()
	ctx := context.Background()
	workspaceID := "workspace-a"
	work, _ := domain.NewNodeRef(domain.NodeKindTask, "task-one")

	openSource := fakePullSource{pull: githubsource.PullRequest{RepositoryLocator: "github://acme/device-gateway", Number: 7, Merged: false}}
	proposer, _ := New(store, openSource, time.Now)
	if _, err := proposer.Propose(ctx, Input{WorkspaceID: workspaceID, RepositoryLocator: "github://acme/device-gateway", PullRequestNumber: 7, WorkItem: work}); !errors.Is(err, ErrPullRequestNotMerged) {
		t.Fatalf("open PR error = %v", err)
	}

	mergedSource := fakePullSource{pull: githubsource.PullRequest{RepositoryLocator: "github://acme/device-gateway", Number: 7, Merged: true, MergeCommitSHA: proposalMergeSHA}}
	proposer, _ = New(store, mergedSource, time.Now)
	if _, err := proposer.Propose(ctx, Input{WorkspaceID: workspaceID, RepositoryLocator: "github://acme/device-gateway", PullRequestNumber: 7, WorkItem: work}); !errors.Is(err, ErrSourceBindingMissing) {
		t.Fatalf("missing binding error = %v", err)
	}

	entity := proposalEntity(t, "service-gateway", workspaceID)
	if err := store.PutEntity(ctx, entity); err != nil {
		t.Fatal(err)
	}
	provenance, _ := domain.NewProvenance("github", "github://acme/device-gateway", proposalMergeSHA, time.Now())
	binding, _ := domain.NewSourceBinding("binding", workspaceID, entity.ID(), provenance, domain.AuthorityAuthoritative)
	if err := store.PutSourceBinding(ctx, binding); err != nil {
		t.Fatal(err)
	}
	if _, err := proposer.Propose(ctx, Input{WorkspaceID: workspaceID, RepositoryLocator: "github://acme/device-gateway", PullRequestNumber: 7, WorkItem: work}); !errors.Is(err, ErrWorkLinkMissing) {
		t.Fatalf("missing work link error = %v", err)
	}
}

func TestSourceBindingLookupRejectsAmbiguousRepositoryOwnership(t *testing.T) {
	store, closeStore := newProposalStore(t)
	defer closeStore()
	ctx := context.Background()
	workspaceID := "workspace-a"
	for _, id := range []string{"service-a", "service-b"} {
		if err := store.PutEntity(ctx, proposalEntity(t, id, workspaceID)); err != nil {
			t.Fatal(err)
		}
		provenance, _ := domain.NewProvenance("github", "github://acme/shared", proposalMergeSHA, time.Now())
		binding, _ := domain.NewSourceBinding("binding-"+id, workspaceID, id, provenance, domain.AuthorityAuthoritative)
		if err := store.PutSourceBinding(ctx, binding); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.FindSourceBindingBySource(ctx, workspaceID, "github", "github://acme/shared"); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("ambiguous source error = %v", err)
	}
}

func newProposalStore(t *testing.T) (*persistence.Store, func()) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:engineering-change-proposal?mode=memory&cache=shared")
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

func proposalEntity(t *testing.T, id, workspaceID string) domain.EngineeringEntity {
	t.Helper()
	value, err := domain.NewEngineeringEntity(id, workspaceID, domain.EntityTypeService, id, domain.EntityStatusActive, "")
	if err != nil {
		t.Fatal(err)
	}
	return value
}
