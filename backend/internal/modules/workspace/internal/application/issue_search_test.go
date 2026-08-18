package application

import (
	"context"
	"errors"
	"testing"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

type issueSearchRepositoryStub struct {
	query IssueSearchQuery
	hits  []IssueSearchResult
	total int
	err   error
	calls int
}

func (s *issueSearchRepositoryStub) SearchIssues(_ context.Context, query IssueSearchQuery) ([]IssueSearchResult, int, error) {
	s.calls++
	s.query = query
	return s.hits, s.total, s.err
}

func TestIssueSearchUseCaseNormalizesBoundsAndMapsResults(t *testing.T) {
	repository := &issueSearchRepositoryStub{total: 73, hits: []IssueSearchResult{{
		Issue: applicationIssueValue(t), MatchSource: "identifier",
	}}}
	authorizer := &accessAuthorizerStub{}
	service, err := NewIssueSearchUseCase(repository, authorizer)
	if err != nil {
		t.Fatal(err)
	}

	result, err := service.SearchIssues(context.Background(), contract.SearchIssuesRequest{
		WorkspaceID: " workspace-1 ", Query: " ＷＳＰ－１ ", Limit: 99, Offset: 2, IncludeClosed: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if repository.query.WorkspaceID != "workspace-1" || repository.query.Phrase != "wsp 1" || repository.query.Limit != 50 || repository.query.Offset != 2 || !repository.query.IncludeClosed {
		t.Fatalf("query = %+v", repository.query)
	}
	if result.Total != 73 || len(result.Issues) != 1 || result.Issues[0].MatchSource != "identifier" || result.Issues[0].Issue.Identifier != "WSP-1" {
		t.Fatalf("result = %+v", result)
	}
	if len(authorizer.permissions) != 1 || authorizer.permissions[0] != contract.PermissionSearchReadable {
		t.Fatalf("permissions = %v", authorizer.permissions)
	}
}

func TestIssueSearchUseCaseRejectsInvalidInputBeforeAuthorization(t *testing.T) {
	repository := &issueSearchRepositoryStub{}
	authorizer := &accessAuthorizerStub{}
	service, err := NewIssueSearchUseCase(repository, authorizer)
	if err != nil {
		t.Fatal(err)
	}

	for _, request := range []contract.SearchIssuesRequest{
		{WorkspaceID: "", Query: "issue"},
		{WorkspaceID: "workspace-1", Query: " \t\n "},
		{WorkspaceID: "workspace-1", Query: "issue", Limit: -1},
		{WorkspaceID: "workspace-1", Query: "issue", Offset: -1},
	} {
		if _, err := service.SearchIssues(context.Background(), request); !errors.Is(err, contract.ErrInvalidIssueSearch) {
			t.Fatalf("request %+v error = %v", request, err)
		}
	}
	if repository.calls != 0 || len(authorizer.permissions) != 0 {
		t.Fatalf("invalid input reached dependencies: repo=%d auth=%v", repository.calls, authorizer.permissions)
	}
}

func TestIssueSearchUseCaseAuthorizesBeforeRepository(t *testing.T) {
	denied := errors.New("denied")
	repository := &issueSearchRepositoryStub{}
	service, err := NewIssueSearchUseCase(repository, &accessAuthorizerStub{err: denied})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.SearchIssues(context.Background(), contract.SearchIssuesRequest{WorkspaceID: "workspace-1", Query: "issue"})
	if !errors.Is(err, denied) || repository.calls != 0 {
		t.Fatalf("error=%v repository calls=%d", err, repository.calls)
	}
}
