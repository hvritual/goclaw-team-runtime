package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

type deniedProjectSurfaceCreateService struct {
	contract.ProjectSurfaceService
	calls int
}

func (s *deniedProjectSurfaceCreateService) CreateProject(context.Context, string, contract.CreateProjectSurfaceRequest) (contract.ProjectSurfaceProject, error) {
	s.calls++
	return contract.ProjectSurfaceProject{}, contract.ErrWorkspacePermissionDenied
}

func TestProjectSurfaceHTTPMapsInitialResourcePermissionDenialToForbidden(t *testing.T) {
	service := &deniedProjectSurfaceCreateService{}
	server := kratoshttp.NewServer()
	NewProjectSurfaceHandler(
		service,
		func(*http.Request) (contract.WorkspaceHTTPIdentity, error) {
			return contract.WorkspaceHTTPIdentity{WorkspaceID: "workspace-1", ActorType: "member", ActorID: "member-1"}, nil
		},
		func(*http.Request) (string, error) { return "user-1", nil },
		func(*http.Request) error { return nil },
	).Register(server)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/projects", strings.NewReader(`{
		"title":"Denied",
		"resources":[{"resource_type":"url","resource_ref":{"url":"https://example.com/docs"}}]
	}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Workspace-Slug", "workspace-1")
	server.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || service.calls != 1 {
		t.Fatalf("status = %d calls=%d body=%s", response.Code, service.calls, response.Body.String())
	}
	if strings.TrimSpace(response.Body.String()) != `{"error":"insufficient workspace role"}` {
		t.Fatalf("body = %s", response.Body.String())
	}
}

var _ contract.ProjectSurfaceService = (*deniedProjectSurfaceCreateService)(nil)
