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

type recordingProjectRequirementService struct {
	last        string
	workspaceID string
	projectID   string
	key         string
	action      string
	actor       contract.WorkspaceActor
	save        contract.SaveProjectRequirementDraftRequest
	issueLink   contract.ProjectRequirementIssueLinkRequest
	outlineLink contract.ProjectRequirementOutlineLinkRequest
	access      contract.ReplaceProjectRequirementAccessRequest
	outline     contract.CreateProjectOutlineNodeRequest
	err         error
}

func (s *recordingProjectRequirementService) record(ctx context.Context, operation, workspaceID, projectID string) {
	s.last, s.workspaceID, s.projectID = operation, workspaceID, projectID
	s.actor, _ = contract.WorkspaceActorFromContext(ctx)
}

func (s *recordingProjectRequirementService) GetProjectRequirement(ctx context.Context, workspaceID, projectID string) (contract.ProjectRequirementBaselineResponse, error) {
	s.record(ctx, "get", workspaceID, projectID)
	return projectRequirementHTTPFixture(), s.err
}

func (s *recordingProjectRequirementService) SaveProjectRequirement(ctx context.Context, workspaceID, projectID, key string, request contract.SaveProjectRequirementDraftRequest) (contract.ProjectRequirementBaselineResponse, error) {
	s.record(ctx, "save", workspaceID, projectID)
	s.key, s.save = key, request
	return projectRequirementHTTPFixture(), s.err
}

func (s *recordingProjectRequirementService) TransitionProjectRequirement(ctx context.Context, workspaceID, projectID, action string, request contract.ProjectRequirementTransitionRequest) (contract.ProjectRequirementBaselineResponse, error) {
	s.record(ctx, "transition", workspaceID, projectID)
	s.action = action
	return projectRequirementHTTPFixture(), s.err
}

func (s *recordingProjectRequirementService) MutateProjectRequirementIssueLink(ctx context.Context, workspaceID, projectID string, request contract.ProjectRequirementIssueLinkRequest, unlink bool) (contract.ProjectRequirementBaselineResponse, error) {
	operation := "link-issue"
	if unlink {
		operation = "unlink-issue"
	}
	s.record(ctx, operation, workspaceID, projectID)
	s.issueLink = request
	return projectRequirementHTTPFixture(), s.err
}

func (s *recordingProjectRequirementService) MutateProjectRequirementOutlineLink(ctx context.Context, workspaceID, projectID string, request contract.ProjectRequirementOutlineLinkRequest, unlink bool) (contract.ProjectRequirementBaselineResponse, error) {
	operation := "link-outline"
	if unlink {
		operation = "unlink-outline"
	}
	s.record(ctx, operation, workspaceID, projectID)
	s.outlineLink = request
	return projectRequirementHTTPFixture(), s.err
}

func (s *recordingProjectRequirementService) GetProjectRequirementAccess(ctx context.Context, workspaceID, projectID string) (contract.ProjectRequirementAccessSet, error) {
	s.record(ctx, "get-access", workspaceID, projectID)
	return contract.ProjectRequirementAccessSet{Revision: 2, Grants: []contract.ProjectRequirementGrant{}}, s.err
}

func (s *recordingProjectRequirementService) ReplaceProjectRequirementAccess(ctx context.Context, workspaceID, projectID string, request contract.ReplaceProjectRequirementAccessRequest) (contract.ProjectRequirementAccessSet, error) {
	s.record(ctx, "put-access", workspaceID, projectID)
	s.access = request
	return contract.ProjectRequirementAccessSet{Revision: request.ExpectedRevision + 1, Grants: []contract.ProjectRequirementGrant{}}, s.err
}

func (s *recordingProjectRequirementService) GetProjectOutline(ctx context.Context, workspaceID, projectID string) (contract.ProjectOutline, error) {
	s.record(ctx, "get-outline", workspaceID, projectID)
	return contract.ProjectOutline{Revision: 3, Nodes: []contract.ProjectOutlineNode{}}, s.err
}

func (s *recordingProjectRequirementService) CreateProjectOutlineNode(ctx context.Context, workspaceID, projectID, key string, request contract.CreateProjectOutlineNodeRequest) (contract.ProjectOutline, error) {
	s.record(ctx, "create-outline", workspaceID, projectID)
	s.key, s.outline = key, request
	return contract.ProjectOutline{Revision: 4, Nodes: []contract.ProjectOutlineNode{}}, s.err
}

func TestProjectRequirementRoutesUseTrustedIdentityAndExactRevisionContracts(t *testing.T) {
	service := &recordingProjectRequirementService{}
	server := newProjectRequirementHTTPServer(service, func(*http.Request) error { return nil })

	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		key        string
		wantStatus int
		wantLast   string
	}{
		{name: "get baseline", method: http.MethodGet, path: "/api/projects/project-1/requirement-baseline", wantStatus: http.StatusOK, wantLast: "get"},
		{name: "create baseline", method: http.MethodPut, path: "/api/projects/project-1/requirement-baseline", body: `{"expected_revision":0,"content":{"problem_statement":"Need","goals":[],"in_scope":[],"out_of_scope":[],"constraints":[],"acceptance_criteria":[],"dependencies":[]},"change_summary":"Initial"}`, key: "baseline-key", wantStatus: http.StatusCreated, wantLast: "save"},
		{name: "update baseline", method: http.MethodPut, path: "/api/projects/project-1/requirement-baseline", body: `{"expected_revision":1,"content":{"problem_statement":"Need","goals":[],"in_scope":[],"out_of_scope":[],"constraints":[],"acceptance_criteria":[],"dependencies":[]},"change_summary":"Edit"}`, wantStatus: http.StatusOK, wantLast: "save"},
		{name: "submit", method: http.MethodPost, path: "/api/projects/project-1/requirement-baseline/submit-review", body: `{"expected_revision":2}`, wantStatus: http.StatusOK, wantLast: "transition"},
		{name: "withdraw", method: http.MethodPost, path: "/api/projects/project-1/requirement-baseline/withdraw", body: `{"expected_revision":3}`, wantStatus: http.StatusOK, wantLast: "transition"},
		{name: "approve", method: http.MethodPost, path: "/api/projects/project-1/requirement-baseline/approve", body: `{"expected_revision":4}`, wantStatus: http.StatusOK, wantLast: "transition"},
		{name: "freeze", method: http.MethodPost, path: "/api/projects/project-1/requirement-baseline/freeze", body: `{"expected_revision":5}`, wantStatus: http.StatusOK, wantLast: "transition"},
		{name: "retire", method: http.MethodPost, path: "/api/projects/project-1/requirement-baseline/retire", body: `{"expected_revision":6}`, wantStatus: http.StatusOK, wantLast: "transition"},
		{name: "link issue", method: http.MethodPost, path: "/api/projects/project-1/requirement-baseline/links", body: `{"expected_revision":7,"requirement_key":"goal-1","issue_id":"issue-1"}`, wantStatus: http.StatusOK, wantLast: "link-issue"},
		{name: "unlink issue", method: http.MethodDelete, path: "/api/projects/project-1/requirement-baseline/links/goal-1/issue-1?expected_revision=8", wantStatus: http.StatusNoContent, wantLast: "unlink-issue"},
		{name: "link outline", method: http.MethodPost, path: "/api/projects/project-1/requirement-baseline/outline-links", body: `{"expected_revision":9,"requirement_key":"goal-1","node_id":"node-1"}`, wantStatus: http.StatusOK, wantLast: "link-outline"},
		{name: "unlink outline", method: http.MethodDelete, path: "/api/projects/project-1/requirement-baseline/outline-links/goal-1/node-1?expected_revision=10", wantStatus: http.StatusNoContent, wantLast: "unlink-outline"},
		{name: "get access", method: http.MethodGet, path: "/api/projects/project-1/requirement-baseline/access", wantStatus: http.StatusOK, wantLast: "get-access"},
		{name: "put access", method: http.MethodPut, path: "/api/projects/project-1/requirement-baseline/access", body: `{"expected_revision":2,"grants":[]}`, wantStatus: http.StatusOK, wantLast: "put-access"},
		{name: "get outline", method: http.MethodGet, path: "/api/projects/project-1/outline", wantStatus: http.StatusOK, wantLast: "get-outline"},
		{name: "create outline", method: http.MethodPost, path: "/api/projects/project-1/outline", body: `{"expected_revision":3,"title":"Root"}`, key: "outline-key", wantStatus: http.StatusCreated, wantLast: "create-outline"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service.last = ""
			request := newProjectRequirementRequest(test.method, test.path, test.body)
			request.Header.Set("X-Workspace-ID", "forged-workspace")
			if test.key != "" {
				request.Header.Set("Idempotency-Key", test.key)
			}
			response := httptest.NewRecorder()
			server.ServeHTTP(response, request)
			if response.Code != test.wantStatus || service.last != test.wantLast {
				t.Fatalf("response = %d %s operation=%q", response.Code, response.Body.String(), service.last)
			}
			if service.workspaceID != "workspace-trusted" || service.projectID != "project-1" || service.actor.ID != "member-1" {
				t.Fatalf("trusted identity = workspace %q project %q actor %+v", service.workspaceID, service.projectID, service.actor)
			}
		})
	}
	if service.issueLink.ExpectedRevision != 8 || service.outlineLink.ExpectedRevision != 10 {
		t.Fatalf("unlink revisions = issue %d outline %d", service.issueLink.ExpectedRevision, service.outlineLink.ExpectedRevision)
	}
}

func TestProjectRequirementHTTPRejectsAliasesGuardsAndUnavailableS10(t *testing.T) {
	service := &recordingProjectRequirementService{}
	server := newProjectRequirementHTTPServer(service, func(*http.Request) error { return nil })

	for _, test := range []struct {
		name, method, path, body string
		wantStatus               int
	}{
		{name: "old body revision alias", method: http.MethodPost, path: "/api/projects/project-1/requirement-baseline/links", body: `{"revision":1,"requirement_key":"goal-1","issue_id":"issue-1"}`, wantStatus: http.StatusBadRequest},
		{name: "old query revision alias", method: http.MethodDelete, path: "/api/projects/project-1/requirement-baseline/links/goal-1/issue-1?revision=1", wantStatus: http.StatusBadRequest},
		{name: "missing baseline key", method: http.MethodPut, path: "/api/projects/project-1/requirement-baseline", body: `{"expected_revision":0,"content":{"problem_statement":"Need","goals":[],"in_scope":[],"out_of_scope":[],"constraints":[],"acceptance_criteria":[],"dependencies":[]},"change_summary":"Initial"}`, wantStatus: http.StatusBadRequest},
		{name: "patch outline", method: http.MethodPatch, path: "/api/projects/project-1/outline/node-1", body: `{}`, wantStatus: http.StatusConflict},
		{name: "delete outline", method: http.MethodDelete, path: "/api/projects/project-1/outline/node-1", wantStatus: http.StatusConflict},
		{name: "reorder outline", method: http.MethodPost, path: "/api/projects/project-1/outline/reorder", body: `{}`, wantStatus: http.StatusConflict},
		{name: "outline issue link", method: http.MethodPost, path: "/api/projects/project-1/outline/node-1/issues", body: `{}`, wantStatus: http.StatusConflict},
		{name: "requirement issue create", method: http.MethodPost, path: "/api/projects/project-1/requirement-baseline/items/goal-1/issues", body: `{}`, wantStatus: http.StatusConflict},
	} {
		t.Run(test.name, func(t *testing.T) {
			service.last = ""
			request := newProjectRequirementRequest(test.method, test.path, test.body)
			response := httptest.NewRecorder()
			server.ServeHTTP(response, request)
			if response.Code != test.wantStatus || service.last != "" {
				t.Fatalf("response = %d %s operation=%q", response.Code, response.Body.String(), service.last)
			}
			if test.wantStatus == http.StatusConflict && strings.TrimSpace(response.Body.String()) != `{"code":"feature_unavailable","error":"feature unavailable"}` {
				t.Fatalf("feature body = %s", response.Body.String())
			}
		})
	}

	csrfService := &recordingProjectRequirementService{}
	csrfServer := newProjectRequirementHTTPServer(csrfService, func(*http.Request) error { return errors.New("bad csrf") })
	request := newProjectRequirementRequest(http.MethodPost, "/api/projects/project-1/requirement-baseline/retire", `{"expected_revision":1}`)
	response := httptest.NewRecorder()
	csrfServer.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || csrfService.last != "" {
		t.Fatalf("csrf response = %d %s operation=%q", response.Code, response.Body.String(), csrfService.last)
	}
}

func TestProjectRequirementHTTPMapsTypedFailuresWithoutPayloadEcho(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantBody   string
	}{
		{name: "revision", err: contract.RevisionConflictError{CurrentRevision: 7}, wantStatus: http.StatusConflict, wantBody: `{"code":"revision_conflict","current_revision":7,"error":"revision conflict"}`},
		{name: "invalid", err: application.ErrInvalidProjectRequirementRequest, wantStatus: http.StatusBadRequest, wantBody: `{"code":"invalid_request","error":"invalid Project Requirement request"}`},
		{name: "permission", err: contract.ErrWorkspacePermissionDenied, wantStatus: http.StatusForbidden, wantBody: `{"code":"permission_denied","error":"insufficient workspace role"}`},
		{name: "missing", err: application.ErrProjectRequirementNotFound, wantStatus: http.StatusNotFound, wantBody: `{"code":"not_found","error":"Project Requirement not found"}`},
		{name: "transition", err: application.ErrProjectRequirementTransition, wantStatus: http.StatusConflict, wantBody: `{"code":"invalid_transition","error":"invalid Project Requirement transition"}`},
		{name: "self approval", err: application.ErrProjectRequirementSelfApproval, wantStatus: http.StatusConflict, wantBody: `{"code":"independent_approval_required","error":"independent Project Requirement approval required"}`},
		{name: "idempotency", err: application.ErrProjectRequirementConflict, wantStatus: http.StatusConflict, wantBody: `{"code":"idempotency_conflict","error":"Project Requirement conflict"}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &recordingProjectRequirementService{err: test.err}
			server := newProjectRequirementHTTPServer(service, func(*http.Request) error { return nil })
			request := newProjectRequirementRequest(http.MethodGet, "/api/projects/project-secret/requirement-baseline", "")
			response := httptest.NewRecorder()
			server.ServeHTTP(response, request)
			if response.Code != test.wantStatus || strings.TrimSpace(response.Body.String()) != test.wantBody || strings.Contains(response.Body.String(), "project-secret") {
				t.Fatalf("response = %d %s", response.Code, response.Body.String())
			}
		})
	}
}

func newProjectRequirementHTTPServer(service contract.ProjectRequirementService, mutation func(*http.Request) error) *kratoshttp.Server {
	server := kratoshttp.NewServer()
	NewProjectRequirementHandler(
		service,
		func(*http.Request) (contract.WorkspaceHTTPIdentity, error) {
			return contract.WorkspaceHTTPIdentity{WorkspaceID: "workspace-trusted", ActorType: "member", ActorID: "member-1"}, nil
		},
		func(*http.Request) (string, error) { return "user-1", nil },
		mutation,
	).Register(server)
	return server
}

func newProjectRequirementRequest(method, path, body string) *http.Request {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("X-Workspace-Slug", "workspace-trusted")
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	return request
}

func projectRequirementHTTPFixture() contract.ProjectRequirementBaselineResponse {
	return contract.ProjectRequirementBaselineResponse{
		Baseline: &contract.ProjectRequirementBaseline{
			ID: "baseline-1", WorkspaceID: "workspace-trusted", ProjectID: "project-1", Status: "draft",
			CurrentRevision: 1, CreatedAt: "2026-08-19T00:00:00Z", UpdatedAt: "2026-08-19T00:00:00Z",
		},
		History: []contract.ProjectRequirementRevision{}, IssueLinks: []contract.ProjectRequirementIssueLink{},
		OutlineLinks: []contract.ProjectRequirementOutlineLink{},
	}
}
