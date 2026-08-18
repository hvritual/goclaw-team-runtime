package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

type recordingProjectSurfaceSearchService struct {
	contract.ProjectSurfaceService
	request     contract.ProjectSurfaceSearchRequest
	workspaceID string
	err         error
	calls       int
}

func (s *recordingProjectSurfaceSearchService) SearchProjects(_ context.Context, workspaceID string, request contract.ProjectSurfaceSearchRequest) (contract.ProjectSurfaceSearchResponse, error) {
	s.calls++
	s.workspaceID, s.request = workspaceID, request
	if s.err != nil {
		return contract.ProjectSurfaceSearchResponse{}, s.err
	}
	description := "Project description"
	return contract.ProjectSurfaceSearchResponse{Projects: []contract.ProjectSurfaceSearchResult{{
		ProjectSurfaceProject: contract.ProjectSurfaceProject{
			ID: "project-1", WorkspaceID: workspaceID, Title: "Alpha project", Description: &description,
			Status: "planned", Priority: "none", CreatedAt: "2026-08-18T00:00:00Z", UpdatedAt: "2026-08-18T00:00:00Z",
		},
		MatchSource: "title",
	}}, Total: 1}, nil
}

func TestProjectSurfaceSearchRoutePrecedesIDAndReturnsExactShape(t *testing.T) {
	service := &recordingProjectSurfaceSearchService{}
	server := kratoshttp.NewServer()
	NewProjectSurfaceHandler(service,
		func(*http.Request) (contract.WorkspaceHTTPIdentity, error) {
			return contract.WorkspaceHTTPIdentity{WorkspaceID: "workspace-1", ActorType: "member", ActorID: "member-1"}, nil
		},
		func(*http.Request) (string, error) { return "user-1", nil }, nil,
	).Register(server)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/projects/search?q=alpha&limit=50&offset=2&include_closed=true", nil)
	request.Header.Set("X-Workspace-Slug", "workspace-1")
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
	}
	if service.calls != 1 || service.workspaceID != "workspace-1" || service.request.Query != "alpha" || service.request.Limit != 50 || service.request.Offset != 2 || !service.request.IncludeClosed {
		t.Fatalf("call = %d workspace=%q request=%+v", service.calls, service.workspaceID, service.request)
	}
	body := strings.TrimSpace(response.Body.String())
	for _, fragment := range []string{`"projects":[{`, `"workspace_id":"workspace-1"`, `"title":"Alpha project"`, `"match_source":"title"`, `"total":1`} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("body missing %s: %s", fragment, body)
		}
	}
	for _, forbidden := range []string{`"project":`, `"workspaceId":`, `"name":`, `"matchSource":`} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("body contains %s: %s", forbidden, body)
		}
	}
}

func TestProjectSurfaceSearchRejectsInvalidInputAndMapsDenial(t *testing.T) {
	for _, path := range []string{
		"/api/projects/search", "/api/projects/search?q=project&limit=0", "/api/projects/search?q=project&offset=-1", "/api/projects/search?q=project&include_closed=maybe",
	} {
		service := &recordingProjectSurfaceSearchService{}
		response := serveProjectSurfaceSearch(t, service, path)
		if response.Code != http.StatusBadRequest || service.calls != 0 {
			t.Fatalf("%s = %d calls=%d body=%s", path, response.Code, service.calls, response.Body.String())
		}
	}
	service := &recordingProjectSurfaceSearchService{err: contract.ErrWorkspacePermissionDenied}
	response := serveProjectSurfaceSearch(t, service, "/api/projects/search?q=project")
	if response.Code != http.StatusForbidden || service.calls != 1 {
		t.Fatalf("denied = %d calls=%d body=%s", response.Code, service.calls, response.Body.String())
	}
}

func TestProjectSurfaceSearchRejectsAuthenticationAndTrustedIdentityFailures(t *testing.T) {
	tests := []struct {
		name         string
		authenticate func(*http.Request) (string, error)
		identity     contract.WorkspaceHTTPIdentityResolver
		wantStatus   int
	}{
		{
			name: "authentication failure",
			authenticate: func(*http.Request) (string, error) {
				return "", errors.New("invalid session")
			},
			identity: func(*http.Request) (contract.WorkspaceHTTPIdentity, error) {
				return contract.WorkspaceHTTPIdentity{}, errors.New("identity resolver must not run")
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "trusted identity failure",
			authenticate: func(*http.Request) (string, error) {
				return "user-1", nil
			},
			identity: func(*http.Request) (contract.WorkspaceHTTPIdentity, error) {
				return contract.WorkspaceHTTPIdentity{}, contract.ErrActorOutsideWorkspace
			},
			wantStatus: http.StatusNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &recordingProjectSurfaceSearchService{}
			server := kratoshttp.NewServer()
			NewProjectSurfaceHandler(service, tt.identity, tt.authenticate, nil).Register(server)
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/api/projects/search?q=project", nil)
			request.Header.Set("X-Workspace-Slug", "workspace-1")
			server.ServeHTTP(response, request)
			if response.Code != tt.wantStatus || service.calls != 0 {
				t.Fatalf("status = %d calls=%d body=%s", response.Code, service.calls, response.Body.String())
			}
		})
	}
}

func serveProjectSurfaceSearch(t *testing.T, service *recordingProjectSurfaceSearchService, path string) *httptest.ResponseRecorder {
	t.Helper()
	server := kratoshttp.NewServer()
	NewProjectSurfaceHandler(service,
		func(*http.Request) (contract.WorkspaceHTTPIdentity, error) {
			return contract.WorkspaceHTTPIdentity{WorkspaceID: "workspace-1", ActorType: "member", ActorID: "member-1"}, nil
		},
		func(*http.Request) (string, error) { return "user-1", nil }, nil,
	).Register(server)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.Header.Set("X-Workspace-Slug", "workspace-1")
	server.ServeHTTP(response, request)
	return response
}

var _ contract.ProjectSurfaceSearchService = (*recordingProjectSurfaceSearchService)(nil)
