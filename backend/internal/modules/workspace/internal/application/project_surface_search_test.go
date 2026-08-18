package application

import (
	"context"
	"errors"
	"testing"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

type recordingProjectSurfaceSearchRepository struct {
	ProjectSurfaceRepository
	query   ProjectSurfaceSearchQuery
	results []ProjectSurfaceSearchResult
	total   int
	calls   int
}

func (r *recordingProjectSurfaceSearchRepository) SearchProjects(_ context.Context, query ProjectSurfaceSearchQuery) ([]ProjectSurfaceSearchResult, int, error) {
	r.calls++
	r.query = query
	return r.results, r.total, nil
}

type recordingProjectSurfaceSearchAuthorizer struct {
	permissions []string
	err         error
}

func (a *recordingProjectSurfaceSearchAuthorizer) AuthorizeWorkspace(_ context.Context, _ string, permission string) error {
	a.permissions = append(a.permissions, permission)
	return a.err
}

func TestProjectSurfaceSearchNormalizesAuthorizesAndBounds(t *testing.T) {
	repository := &recordingProjectSurfaceSearchRepository{
		results: []ProjectSurfaceSearchResult{{
			Project:     contract.ProjectSurfaceProject{ID: "project-1", Title: "Ｃａｆé API"},
			MatchSource: "title",
		}},
		total: 73,
	}
	authorizer := &recordingProjectSurfaceSearchAuthorizer{}
	service := &ProjectSurfaceUseCase{repository: repository, authorizer: authorizer}

	result, err := service.SearchProjects(context.Background(), " workspace-1 ", contract.ProjectSurfaceSearchRequest{
		Query: " Ｃａｆé—API ", Limit: 100, Offset: 2, IncludeClosed: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if repository.query.WorkspaceID != "workspace-1" || repository.query.Phrase != "café api" || len(repository.query.Terms) != 2 || repository.query.Limit != 50 || repository.query.Offset != 2 || !repository.query.IncludeClosed {
		t.Fatalf("query = %+v", repository.query)
	}
	if result.Total != 73 || len(result.Projects) != 1 || result.Projects[0].MatchSource != "title" {
		t.Fatalf("result = %+v", result)
	}
	if len(authorizer.permissions) != 1 || authorizer.permissions[0] != contract.PermissionSearchReadable {
		t.Fatalf("permissions = %v", authorizer.permissions)
	}
}

func TestProjectSurfaceSearchRejectsBeforeAuthorizationAndStopsOnDenial(t *testing.T) {
	repository := &recordingProjectSurfaceSearchRepository{}
	authorizer := &recordingProjectSurfaceSearchAuthorizer{}
	service := &ProjectSurfaceUseCase{repository: repository, authorizer: authorizer}

	for _, request := range []contract.ProjectSurfaceSearchRequest{
		{Query: " "}, {Query: "project", Limit: -1}, {Query: "project", Offset: -1},
	} {
		if _, err := service.SearchProjects(context.Background(), "workspace-1", request); !errors.Is(err, ErrInvalidProjectSurfaceRequest) {
			t.Fatalf("request %+v error = %v", request, err)
		}
	}
	if repository.calls != 0 || len(authorizer.permissions) != 0 {
		t.Fatalf("invalid input touched dependencies: repository=%d permissions=%v", repository.calls, authorizer.permissions)
	}

	authorizer.err = contract.ErrWorkspacePermissionDenied
	if _, err := service.SearchProjects(context.Background(), "workspace-1", contract.ProjectSurfaceSearchRequest{Query: "project"}); !errors.Is(err, contract.ErrWorkspacePermissionDenied) {
		t.Fatalf("denied error = %v", err)
	}
	if repository.calls != 0 {
		t.Fatalf("denied request repository calls = %d", repository.calls)
	}
}
