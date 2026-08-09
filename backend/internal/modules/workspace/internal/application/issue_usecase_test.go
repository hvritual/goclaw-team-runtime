package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	issueDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/issue"
)

type issueRepositoryStub struct {
	value                                      issueDomain.Issue
	values                                     []issueDomain.Issue
	query                                      IssueListQuery
	err                                        error
	createCalls, findCalls, listCalls, updates int
	cycle                                      bool
}

func (s *issueRepositoryStub) Create(_ context.Context, value issueDomain.Issue) (issueDomain.Issue, error) {
	s.createCalls++
	if s.err != nil {
		return issueDomain.Issue{}, s.err
	}
	created, err := value.AssignIdentity(1, "WSP")
	s.value = created
	return created, err
}
func (s *issueRepositoryStub) FindByIDOrIdentifier(context.Context, string, string) (issueDomain.Issue, error) {
	s.findCalls++
	return s.value, s.err
}
func (s *issueRepositoryStub) List(_ context.Context, query IssueListQuery) ([]issueDomain.Issue, error) {
	s.listCalls++
	s.query = query
	return append([]issueDomain.Issue(nil), s.values...), s.err
}
func (s *issueRepositoryStub) Update(_ context.Context, value issueDomain.Issue) error {
	s.updates++
	s.value = value
	return s.err
}
func (s *issueRepositoryStub) WouldCreateParentCycle(context.Context, string, string, string) (bool, error) {
	return s.cycle, s.err
}

type issueAssetReaderStub struct {
	belongs bool
	calls   int
}

func (s *issueAssetReaderStub) AssetBelongsToWorkspace(context.Context, string, string) (bool, error) {
	s.calls++
	return s.belongs, nil
}

func applicationIssueValue(t *testing.T) issueDomain.Issue {
	t.Helper()
	value, err := issueDomain.New("issue-1", "workspace-1", "Issue", nil, "todo", "none", nil, nil, nil, nil, "member", "member-1", 0, nil, nil, nil, nil, time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	value, err = value.AssignIdentity(1, "WSP")
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func newIssueApplicationService(t *testing.T, repository *issueRepositoryStub, authorizer *accessAuthorizerStub, actors *actorReaderStub, assets *issueAssetReaderStub) *IssueUseCase {
	t.Helper()
	service, err := NewIssueUseCase(repository, &projectRepositoryStub{}, authorizer, actors, assets,
		func(context.Context) (string, error) { return "issue-new", nil },
		func() time.Time { return time.Date(2026, 8, 3, 1, 0, 0, 0, time.UTC) },
	)
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func TestIssueUseCaseCreateUsesTrustedReferencesAndAtomicRepository(t *testing.T) {
	repository := &issueRepositoryStub{}
	actors := &actorReaderStub{belongs: true}
	assets := &issueAssetReaderStub{belongs: true}
	service := newIssueApplicationService(t, repository, &accessAuthorizerStub{}, actors, assets)
	ctx := contract.WithWorkspaceActor(context.Background(), "agent", "agent-creator")
	assigneeType, assigneeID, assetID := "member", "member-1", "asset-1"
	response, err := service.CreateIssue(ctx, contract.CreateIssueRequest{
		WorkspaceId: "workspace-1", Title: " Ship ", AssigneeType: &assigneeType, AssigneeId: &assigneeID, AssetIds: []string{assetID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.Issue == nil || response.Issue.Identifier != "WSP-1" || response.Issue.CreatorType != "agent" || response.Issue.CreatorId != "agent-creator" || response.Issue.Status != "todo" || response.Issue.Priority != "none" {
		t.Fatalf("CreateIssue() = %+v", response.Issue)
	}
	if repository.createCalls != 1 || actors.calls != 2 || assets.calls != 1 {
		t.Fatalf("calls = repo:%d actors:%d assets:%d", repository.createCalls, actors.calls, assets.calls)
	}
}

func TestIssueUseCaseGetListUpdateStatusAndClear(t *testing.T) {
	value := applicationIssueValue(t)
	repository := &issueRepositoryStub{value: value, values: []issueDomain.Issue{value}}
	authorizer := &accessAuthorizerStub{}
	service := newIssueApplicationService(t, repository, authorizer, &actorReaderStub{belongs: true}, &issueAssetReaderStub{belongs: true})
	got, err := service.GetIssue(context.Background(), contract.GetIssueRequest{WorkspaceId: "workspace-1", IssueId: "WSP-1"})
	if err != nil || got.Issue == nil || got.Issue.Id != "issue-1" {
		t.Fatalf("GetIssue() = %+v, %v", got.Issue, err)
	}
	projectID, actorType, actorID := "project-1", "member", "member-1"
	listed, err := service.ListIssues(context.Background(), contract.ListIssuesRequest{WorkspaceId: "workspace-1", ProjectId: &projectID, Status: "todo", AssigneeType: &actorType, AssigneeId: &actorID})
	if err != nil || listed.Total != 1 || repository.query.ProjectID == nil || repository.query.Status != "todo" {
		t.Fatalf("ListIssues() = %+v query=%+v, %v", listed, repository.query, err)
	}
	empty, title, done := "", "Updated", "done"
	updated, err := service.UpdateIssue(context.Background(), contract.UpdateIssueRequest{
		WorkspaceId: "workspace-1", IssueId: "WSP-1", Title: &title, Status: &done,
		AssigneeType: &empty, AssigneeId: &empty, ProjectId: &empty, StartDate: &empty,
		AssetIds: &contract.IssueAssetIDs{},
	})
	if err != nil || updated.Issue == nil || updated.Issue.AssigneeId != nil || updated.Issue.ProjectId != nil || len(updated.Issue.AssetIds) != 0 {
		t.Fatalf("UpdateIssue() = %+v, %v", updated.Issue, err)
	}
	statusResponse, err := service.UpdateIssueStatus(context.Background(), contract.UpdateIssueStatusRequest{WorkspaceId: "workspace-1", IssueId: "issue-1", Status: "blocked"})
	if err != nil || statusResponse.Issue == nil || statusResponse.Issue.Status != "blocked" {
		t.Fatalf("UpdateIssueStatus() = %+v, %v", statusResponse.Issue, err)
	}
}

func TestIssueUseCaseMapsIsolationAndRejectsCyclesAndForeignReferences(t *testing.T) {
	repository := &issueRepositoryStub{value: applicationIssueValue(t)}
	service := newIssueApplicationService(t, repository, &accessAuthorizerStub{}, &actorReaderStub{belongs: false}, &issueAssetReaderStub{belongs: false})
	ctx := contract.WithWorkspaceActor(context.Background(), "member", "outsider")
	if _, err := service.CreateIssue(ctx, contract.CreateIssueRequest{WorkspaceId: "workspace-1", Title: "Issue"}); !errors.Is(err, contract.ErrActorOutsideWorkspace) {
		t.Fatalf("foreign creator error = %v", err)
	}
	repository.err = ErrIssueRecordNotFound
	if _, err := service.GetIssue(context.Background(), contract.GetIssueRequest{WorkspaceId: "workspace-2", IssueId: "issue-1"}); !errors.Is(err, contract.ErrIssueNotFound) {
		t.Fatalf("foreign GetIssue error = %v", err)
	}
	repository.err = nil
	repository.cycle = true
	service = newIssueApplicationService(t, repository, &accessAuthorizerStub{}, &actorReaderStub{belongs: true}, &issueAssetReaderStub{belongs: true})
	parent := "WSP-1"
	if _, err := service.UpdateIssue(context.Background(), contract.UpdateIssueRequest{WorkspaceId: "workspace-1", IssueId: "issue-1", ParentIssueId: &parent}); !errors.Is(err, contract.ErrInvalidIssue) {
		t.Fatalf("cycle error = %v", err)
	}
	negativeStage := int32(-1)
	if _, err := service.UpdateIssue(context.Background(), contract.UpdateIssueRequest{WorkspaceId: "workspace-1", IssueId: "issue-1", Stage: &negativeStage}); !errors.Is(err, contract.ErrInvalidIssue) {
		t.Fatalf("negative stage error = %v", err)
	}
	paddedDate := " 2026-08-03"
	if _, err := service.UpdateIssue(context.Background(), contract.UpdateIssueRequest{WorkspaceId: "workspace-1", IssueId: "issue-1", StartDate: &paddedDate}); !errors.Is(err, contract.ErrInvalidIssue) {
		t.Fatalf("padded date error = %v", err)
	}
}

func TestIssueUseCaseAuthorizesBeforePersistence(t *testing.T) {
	denied := errors.New("denied")
	repository := &issueRepositoryStub{}
	service := newIssueApplicationService(t, repository, &accessAuthorizerStub{err: denied}, &actorReaderStub{belongs: true}, &issueAssetReaderStub{belongs: true})
	ctx := contract.WithWorkspaceActor(context.Background(), "member", "member-1")
	operations := []func() error{
		func() error {
			_, err := service.CreateIssue(ctx, contract.CreateIssueRequest{WorkspaceId: "workspace-1", Title: "Issue"})
			return err
		},
		func() error {
			_, err := service.GetIssue(ctx, contract.GetIssueRequest{WorkspaceId: "workspace-1", IssueId: "issue-1"})
			return err
		},
		func() error {
			_, err := service.ListIssues(ctx, contract.ListIssuesRequest{WorkspaceId: "workspace-1"})
			return err
		},
		func() error {
			_, err := service.UpdateIssue(ctx, contract.UpdateIssueRequest{WorkspaceId: "workspace-1", IssueId: "issue-1"})
			return err
		},
		func() error {
			_, err := service.UpdateIssueStatus(ctx, contract.UpdateIssueStatusRequest{WorkspaceId: "workspace-1", IssueId: "issue-1", Status: "done"})
			return err
		},
	}
	for index, operation := range operations {
		if err := operation(); !errors.Is(err, denied) {
			t.Fatalf("operation %d error = %v", index, err)
		}
	}
	if repository.createCalls+repository.findCalls+repository.listCalls+repository.updates != 0 {
		t.Fatalf("repository accessed before authorization: %+v", repository)
	}
}
