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
	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
)

type recordingProjectRetrospectiveService struct {
	operation, workspaceID, projectID, retrospectiveID, actionItemID, key string
	actor                                                                 contract.WorkspaceActor
	limit                                                                 int
	cursor                                                                string
	includeArchived                                                       bool
	create                                                                contract.CreateProjectRetrospectiveRequest
	update                                                                contract.UpdateProjectRetrospectiveRequest
	target                                                                contract.CreateProjectRetrospectiveTargetRequest
	expectedRevision                                                      int64
	err                                                                   error
}

func (s *recordingProjectRetrospectiveService) record(ctx context.Context, operation, workspaceID, projectID string) {
	s.operation, s.workspaceID, s.projectID = operation, workspaceID, projectID
	s.actor, _ = contract.WorkspaceActorFromContext(ctx)
}

func (s *recordingProjectRetrospectiveService) ListProjectRetrospectives(ctx context.Context, workspaceID, projectID string, limit int, cursor string, includeArchived bool) (contract.ProjectRetrospectiveList, error) {
	s.record(ctx, "list", workspaceID, projectID)
	s.limit, s.cursor, s.includeArchived = limit, cursor, includeArchived
	return contract.ProjectRetrospectiveList{Retrospectives: []contract.ProjectRetrospective{}}, s.err
}

func (s *recordingProjectRetrospectiveService) GetProjectRetrospective(ctx context.Context, workspaceID, projectID, retrospectiveID string) (contract.ProjectRetrospective, error) {
	s.record(ctx, "get", workspaceID, projectID)
	s.retrospectiveID = retrospectiveID
	return projectRetrospectiveHTTPFixture(), s.err
}

func (s *recordingProjectRetrospectiveService) CreateProjectRetrospective(ctx context.Context, workspaceID, projectID, key string, request contract.CreateProjectRetrospectiveRequest) (contract.ProjectRetrospective, error) {
	s.record(ctx, "create", workspaceID, projectID)
	s.key, s.create = key, request
	return projectRetrospectiveHTTPFixture(), s.err
}

func (s *recordingProjectRetrospectiveService) UpdateProjectRetrospective(ctx context.Context, workspaceID, projectID, retrospectiveID string, request contract.UpdateProjectRetrospectiveRequest) (contract.ProjectRetrospective, error) {
	s.record(ctx, "update", workspaceID, projectID)
	s.retrospectiveID, s.update = retrospectiveID, request
	return projectRetrospectiveHTTPFixture(), s.err
}

func (s *recordingProjectRetrospectiveService) ArchiveProjectRetrospective(ctx context.Context, workspaceID, projectID, retrospectiveID string, expectedRevision int64) (contract.ProjectRetrospective, error) {
	s.record(ctx, "archive", workspaceID, projectID)
	s.retrospectiveID, s.expectedRevision = retrospectiveID, expectedRevision
	return projectRetrospectiveHTTPFixture(), s.err
}

func (s *recordingProjectRetrospectiveService) CreateProjectRetrospectiveTarget(ctx context.Context, workspaceID, projectID, retrospectiveID, actionItemID, key string, request contract.CreateProjectRetrospectiveTargetRequest) (contract.ProjectRetrospectiveActionLink, error) {
	s.record(ctx, "target", workspaceID, projectID)
	s.retrospectiveID, s.actionItemID, s.key, s.target = retrospectiveID, actionItemID, key, request
	return contract.ProjectRetrospectiveActionLink{RetrospectiveID: retrospectiveID, ActionItemID: actionItemID, SourceRevision: 2, State: "linked", TargetKind: "task", TargetID: "task-1", CreatedBy: "member-1", CreatedAt: "2026-08-20T00:00:00Z"}, s.err
}

func TestProjectRetrospectiveHTTPRoutesUseTrustedIdentityAndExactInputs(t *testing.T) {
	service := &recordingProjectRetrospectiveService{}
	server := newProjectRetrospectiveHTTPServer(service, func(*http.Request) error { return nil })
	tests := []struct {
		name, method, path, body, key, operation string
		status                                   int
	}{
		{name: "list", method: http.MethodGet, path: "/api/projects/project-1/retrospectives?limit=25&cursor=opaque&include_archived=true", operation: "list", status: http.StatusOK},
		{name: "get", method: http.MethodGet, path: "/api/projects/project-1/retrospectives/retro-1", operation: "get", status: http.StatusOK},
		{name: "create", method: http.MethodPost, path: "/api/projects/project-1/retrospectives", key: "create-key", body: `{"content":{"summary":"Summary","successes":[],"problems":[],"lessons":["Lesson"],"action_items":[]},"participants":[]}`, operation: "create", status: http.StatusCreated},
		{name: "update", method: http.MethodPut, path: "/api/projects/project-1/retrospectives/retro-1", body: `{"expected_revision":2,"action":"publish"}`, operation: "update", status: http.StatusOK},
		{name: "archive", method: http.MethodDelete, path: "/api/projects/project-1/retrospectives/retro-1?expected_revision=3", operation: "archive", status: http.StatusOK},
		{name: "default task target", method: http.MethodPost, path: "/api/projects/project-1/retrospectives/retro-1/action-items/action-1/target", key: "target-key", body: `{}`, operation: "target", status: http.StatusCreated},
		{name: "explicit issue target", method: http.MethodPost, path: "/api/projects/project-1/retrospectives/retro-1/action-items/action-1/target", key: "issue-key", body: `{"target_kind":"issue"}`, operation: "target", status: http.StatusCreated},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service.operation = ""
			request := projectRetrospectiveHTTPRequest(test.method, test.path, test.body)
			request.Header.Set("X-Workspace-ID", "forged-workspace")
			if test.key != "" {
				request.Header.Set("Idempotency-Key", test.key)
			}
			response := httptest.NewRecorder()
			server.ServeHTTP(response, request)
			if response.Code != test.status || service.operation != test.operation {
				t.Fatalf("response = %d %s operation %q", response.Code, response.Body.String(), service.operation)
			}
			if service.workspaceID != "workspace-trusted" || service.projectID != "project-1" || service.actor != (contract.WorkspaceActor{Type: "member", ID: "member-1"}) {
				t.Fatalf("trusted identity = workspace %q project %q actor %#v", service.workspaceID, service.projectID, service.actor)
			}
		})
	}
	if service.limit != 25 || service.cursor != "opaque" || !service.includeArchived || service.expectedRevision != 3 {
		t.Fatalf("list/archive inputs = limit %d cursor %q archived %t revision %d", service.limit, service.cursor, service.includeArchived, service.expectedRevision)
	}
	if service.target.TargetKind == nil || *service.target.TargetKind != "issue" || service.retrospectiveID != "retro-1" || service.actionItemID != "action-1" {
		t.Fatalf("target inputs = %#v", service)
	}
}

func TestProjectRetrospectiveHTTPRejectsStrictInputAndCSRF(t *testing.T) {
	for _, test := range []struct{ name, method, path, body, key string }{
		{name: "unknown list query", method: http.MethodGet, path: "/api/projects/project-1/retrospectives?offset=1"},
		{name: "invalid list limit", method: http.MethodGet, path: "/api/projects/project-1/retrospectives?limit=101"},
		{name: "duplicate list cursor", method: http.MethodGet, path: "/api/projects/project-1/retrospectives?cursor=a&cursor=b"},
		{name: "invalid archived flag", method: http.MethodGet, path: "/api/projects/project-1/retrospectives?include_archived=maybe"},
		{name: "missing create key", method: http.MethodPost, path: "/api/projects/project-1/retrospectives", body: `{}`},
		{name: "unknown create field", method: http.MethodPost, path: "/api/projects/project-1/retrospectives", key: "key", body: `{"unknown":true}`},
		{name: "trailing target body", method: http.MethodPost, path: "/api/projects/project-1/retrospectives/retro-1/action-items/action-1/target", key: "key", body: `{} {}`},
		{name: "delete alias", method: http.MethodDelete, path: "/api/projects/project-1/retrospectives/retro-1?revision=1"},
	} {
		t.Run(test.name, func(t *testing.T) {
			service := &recordingProjectRetrospectiveService{}
			server := newProjectRetrospectiveHTTPServer(service, func(*http.Request) error { return nil })
			request := projectRetrospectiveHTTPRequest(test.method, test.path, test.body)
			if test.key != "" {
				request.Header.Set("Idempotency-Key", test.key)
			}
			response := httptest.NewRecorder()
			server.ServeHTTP(response, request)
			if response.Code != http.StatusBadRequest || service.operation != "" {
				t.Fatalf("response = %d %s operation %q", response.Code, response.Body.String(), service.operation)
			}
		})
	}
	service := &recordingProjectRetrospectiveService{}
	server := newProjectRetrospectiveHTTPServer(service, func(*http.Request) error { return errors.New("bad csrf") })
	request := projectRetrospectiveHTTPRequest(http.MethodPut, "/api/projects/project-1/retrospectives/retro-1", `{"expected_revision":1,"action":"publish"}`)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || service.operation != "" {
		t.Fatalf("csrf response = %d %s operation %q", response.Code, response.Body.String(), service.operation)
	}
}

func TestProjectRetrospectiveHTTPMapsTypedProblemsWithoutPayloadEcho(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "invalid", err: application.ErrInvalidProjectRetrospectiveRequest, wantStatus: http.StatusBadRequest, wantCode: "invalid_request"},
		{name: "permission", err: contract.ErrWorkspacePermissionDenied, wantStatus: http.StatusForbidden, wantCode: "permission_denied"},
		{name: "missing", err: contract.ErrProjectRetrospectiveNotFound, wantStatus: http.StatusNotFound, wantCode: "not_found"},
		{name: "revision", err: contract.RevisionConflictError{CurrentRevision: 7}, wantStatus: http.StatusConflict, wantCode: "revision_conflict"},
		{name: "state", err: contract.ErrProjectRetrospectiveStateConflict, wantStatus: http.StatusConflict, wantCode: "state_conflict"},
		{name: "target", err: contract.ErrProjectRetrospectiveTargetConflict, wantStatus: http.StatusConflict, wantCode: "target_conflict"},
		{name: "idempotency", err: contract.ErrIdempotencyConflict, wantStatus: http.StatusConflict, wantCode: "idempotency_conflict"},
		{name: "internal", err: errors.New("contains project-secret payload-secret"), wantStatus: http.StatusInternalServerError, wantCode: "internal_error"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &recordingProjectRetrospectiveService{err: test.err}
			server := newProjectRetrospectiveHTTPServer(service, func(*http.Request) error { return nil })
			response := httptest.NewRecorder()
			server.ServeHTTP(response, projectRetrospectiveHTTPRequest(http.MethodGet, "/api/projects/project-secret/retrospectives/retro-secret", ""))
			body := response.Body.String()
			if response.Code != test.wantStatus || !strings.Contains(body, `"code":"`+test.wantCode+`"`) || strings.Contains(body, "project-secret") || strings.Contains(body, "payload-secret") {
				t.Fatalf("response = %d %s", response.Code, body)
			}
		})
	}
}

func newProjectRetrospectiveHTTPServer(service contract.ProjectRetrospectiveService, mutation func(*http.Request) error) *kratoshttp.Server {
	server := kratoshttp.NewServer()
	NewProjectRetrospectiveHandler(service,
		func(*http.Request) (contract.WorkspaceHTTPIdentity, error) {
			return contract.WorkspaceHTTPIdentity{WorkspaceID: "workspace-trusted", ActorType: "member", ActorID: "member-1"}, nil
		},
		func(*http.Request) (string, error) { return "user-1", nil }, mutation,
	).Register(server)
	return server
}

func projectRetrospectiveHTTPRequest(method, path, body string) *http.Request {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("X-Workspace-Slug", "workspace-trusted")
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	return request
}

func projectRetrospectiveHTTPFixture() contract.ProjectRetrospective {
	return contract.ProjectRetrospective{
		ID: "retro-1", WorkspaceID: "workspace-trusted", ProjectID: "project-1", Status: "draft", CurrentRevision: 1,
		CreatedBy: "member-1", CreatedAt: "2026-08-20T00:00:00Z", UpdatedAt: "2026-08-20T00:00:00Z",
		History: []contract.ProjectRetrospectiveRevision{}, ActionLinks: []contract.ProjectRetrospectiveActionLink{},
	}
}
