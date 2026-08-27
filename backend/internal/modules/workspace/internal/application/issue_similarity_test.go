package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

type unavailableIssueSimilarityRepository struct{}

func (unavailableIssueSimilarityRepository) FindIssueSimilarityCandidates(context.Context, IssueSimilarityQuery) ([]IssueSimilarityCandidate, bool, error) {
	return nil, false, contract.ErrIssueSimilarityUnavailable
}

type issueSimilarityRepositoryStub struct {
	query      IssueSimilarityQuery
	candidates []IssueSimilarityCandidate
	truncated  bool
	err        error
	calls      int
}

func (s *issueSimilarityRepositoryStub) FindIssueSimilarityCandidates(_ context.Context, query IssueSimilarityQuery) ([]IssueSimilarityCandidate, bool, error) {
	s.calls++
	s.query = query
	return s.candidates, s.truncated, s.err
}

func TestIssueSimilarityUseCaseReturnsTruthfulDegradedResult(t *testing.T) {
	authorizer := &accessAuthorizerStub{}
	service, err := NewIssueSimilarityUseCase(unavailableIssueSimilarityRepository{}, authorizer)
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.CheckIssueSimilarity(context.Background(), contract.CheckIssueSimilarityRequest{
		WorkspaceID: "workspace-1",
		Title:       "Alpha beta",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.DetectorAvailable || result.RankingVersion == "" || result.Candidates == nil || len(result.Candidates) != 0 {
		t.Fatalf("result = %+v, want empty degraded response", result)
	}
	if len(authorizer.permissions) != 1 || authorizer.permissions[0] != contract.PermissionSimilarityCheck {
		t.Fatalf("permissions = %v", authorizer.permissions)
	}
}

func TestIssueSimilarityUseCaseNormalizesBoundsAndMapsCandidates(t *testing.T) {
	repository := &issueSimilarityRepositoryStub{}
	for index := 0; index < 6; index++ {
		issue := applicationIssueValue(t)
		issue.ID = fmt.Sprintf("issue-%d", index+1)
		repository.candidates = append(repository.candidates, IssueSimilarityCandidate{
			Issue: issue, Score: 100 - index, ComponentScores: map[string]float64{"title_terms": 1}, SameProject: index == 0,
		})
	}
	authorizer := &accessAuthorizerStub{}
	service, err := NewIssueSimilarityUseCase(repository, authorizer)
	if err != nil {
		t.Fatal(err)
	}
	description, projectID := " Alpha—Beta ", " project-1 "
	result, err := service.CheckIssueSimilarity(context.Background(), contract.CheckIssueSimilarityRequest{
		WorkspaceID: " workspace-1 ", Title: " Ａｌｐｈａ—Ｂｅｔａ ", Description: &description, ProjectID: &projectID,
		IncludeClosed: true, ExcludeIssueID: " issue-self ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if repository.calls != 1 || repository.query.WorkspaceID != "workspace-1" || repository.query.Title != "alpha beta" || repository.query.Description != "alpha beta" || repository.query.ProjectID != "project-1" || !repository.query.IncludeClosed || repository.query.ExcludeIssueID != "issue-self" || repository.query.Limit != 50 {
		t.Fatalf("repository query = %+v calls=%d", repository.query, repository.calls)
	}
	if len(result.Candidates) != 5 || !result.Truncated || !result.DetectorAvailable || result.RankingVersion == "" || result.Candidates[0].Issue.Id != "issue-1" || !result.Candidates[0].SameProject {
		t.Fatalf("response = %+v", result)
	}
	if len(authorizer.permissions) != 1 || authorizer.permissions[0] != contract.PermissionSimilarityCheck {
		t.Fatalf("permissions = %v", authorizer.permissions)
	}
}

func TestIssueSimilarityUseCaseRejectsInvalidInputBeforeAuthorization(t *testing.T) {
	repository := &issueSimilarityRepositoryStub{}
	authorizer := &accessAuthorizerStub{}
	service, err := NewIssueSimilarityUseCase(repository, authorizer)
	if err != nil {
		t.Fatal(err)
	}
	for _, request := range []contract.CheckIssueSimilarityRequest{
		{Title: "Issue"},
		{WorkspaceID: "workspace-1", Title: " \t\n "},
	} {
		if _, err := service.CheckIssueSimilarity(context.Background(), request); !errors.Is(err, contract.ErrInvalidIssueSimilarity) {
			t.Fatalf("request %+v error = %v", request, err)
		}
	}
	if repository.calls != 0 || len(authorizer.permissions) != 0 {
		t.Fatalf("invalid input reached dependencies: repository=%d authorization=%v", repository.calls, authorizer.permissions)
	}
}

func TestIssueSimilarityUseCaseRejectsOversizedInputBeforeAuthorization(t *testing.T) {
	validDescription := "Description"
	stringPointer := func(value string) *string { return &value }
	for _, test := range []struct {
		name    string
		request contract.CheckIssueSimilarityRequest
	}{
		{
			name: "title exceeds normalized rune limit",
			request: contract.CheckIssueSimilarityRequest{
				WorkspaceID: "workspace-1",
				Title:       strings.Repeat("界", 1025),
			},
		},
		{
			name: "description exceeds normalized rune limit",
			request: contract.CheckIssueSimilarityRequest{
				WorkspaceID: "workspace-1",
				Title:       "Issue",
				Description: stringPointer(strings.Repeat("界", 4097)),
			},
		},
		{
			name: "title exceeds normalized term limit",
			request: contract.CheckIssueSimilarityRequest{
				WorkspaceID: "workspace-1",
				Title:       strings.TrimSpace(strings.Repeat("term ", 33)),
				Description: &validDescription,
			},
		},
		{
			name: "description exceeds normalized term limit",
			request: contract.CheckIssueSimilarityRequest{
				WorkspaceID: "workspace-1",
				Title:       "Issue",
				Description: stringPointer(strings.TrimSpace(strings.Repeat("term ", 33))),
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := &issueSimilarityRepositoryStub{}
			authorizer := &accessAuthorizerStub{}
			service, err := NewIssueSimilarityUseCase(repository, authorizer)
			if err != nil {
				t.Fatal(err)
			}

			if _, err := service.CheckIssueSimilarity(context.Background(), test.request); !errors.Is(err, contract.ErrInvalidIssueSimilarity) {
				t.Fatalf("error = %v, want ErrInvalidIssueSimilarity", err)
			}
			if repository.calls != 0 || len(authorizer.permissions) != 0 {
				t.Fatalf("oversized input reached dependencies: repository=%d authorization=%v", repository.calls, authorizer.permissions)
			}
		})
	}
}

func TestIssueSimilarityUseCaseAuthorizesBeforeRepository(t *testing.T) {
	denied := errors.New("denied")
	repository := &issueSimilarityRepositoryStub{}
	service, err := NewIssueSimilarityUseCase(repository, &accessAuthorizerStub{err: denied})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.CheckIssueSimilarity(context.Background(), contract.CheckIssueSimilarityRequest{WorkspaceID: "workspace-1", Title: "Issue"})
	if !errors.Is(err, denied) || repository.calls != 0 {
		t.Fatalf("error=%v repository calls=%d", err, repository.calls)
	}
}
