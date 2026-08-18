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

type recordingPinReorderService struct {
	contract.ProjectSurfaceService
	request     contract.ReorderPinsRequest
	workspaceID string
	userID      string
	err         error
	calls       int
}

func (s *recordingPinReorderService) ReorderPins(_ context.Context, workspaceID, userID string, request contract.ReorderPinsRequest) error {
	s.calls++
	s.workspaceID, s.userID, s.request = workspaceID, userID, request
	return s.err
}

func TestPinReorderUsesTrustedIdentityAndExactRequest(t *testing.T) {
	service := &recordingPinReorderService{}
	response := servePinReorder(t, service, `{"items":[{"id":"pin-2"},{"id":"pin-1"}],"expected_revision":7}`,
		func(*http.Request) (string, error) { return "user-1", nil },
		func(*http.Request) (contract.WorkspaceHTTPIdentity, error) {
			return contract.WorkspaceHTTPIdentity{WorkspaceID: "workspace-1", ActorType: "member", ActorID: "member-1"}, nil
		}, func(*http.Request) error { return nil })
	if response.Code != http.StatusNoContent || service.calls != 1 || service.workspaceID != "workspace-1" || service.userID != "user-1" || service.request.ExpectedRevision != 7 || len(service.request.Items) != 2 {
		t.Fatalf("response/call = %d %+v body=%s", response.Code, service, response.Body.String())
	}
}

func TestPinReorderMapsRevisionConflictAndStopsAtSecurityGates(t *testing.T) {
	service := &recordingPinReorderService{err: contract.RevisionConflictError{CurrentRevision: 9}}
	response := servePinReorder(t, service, `{"items":[{"id":"pin-1"}],"expected_revision":8}`,
		func(*http.Request) (string, error) { return "user-1", nil },
		func(*http.Request) (contract.WorkspaceHTTPIdentity, error) {
			return contract.WorkspaceHTTPIdentity{WorkspaceID: "workspace-1", ActorType: "member", ActorID: "member-1"}, nil
		}, func(*http.Request) error { return nil })
	if response.Code != http.StatusConflict || strings.TrimSpace(response.Body.String()) != `{"code":"revision_conflict","current_revision":9,"error":"revision conflict"}` {
		t.Fatalf("conflict = %d %s", response.Code, response.Body.String())
	}

	for _, test := range []struct {
		name         string
		authenticate func(*http.Request) (string, error)
		identity     contract.WorkspaceHTTPIdentityResolver
		mutation     func(*http.Request) error
		want         int
	}{
		{name: "authentication", authenticate: func(*http.Request) (string, error) { return "", contract.ErrWorkspaceActorRequired }, identity: nil, mutation: nil, want: http.StatusUnauthorized},
		{name: "identity", authenticate: func(*http.Request) (string, error) { return "user-1", nil }, identity: func(*http.Request) (contract.WorkspaceHTTPIdentity, error) {
			return contract.WorkspaceHTTPIdentity{}, contract.ErrActorOutsideWorkspace
		}, mutation: nil, want: http.StatusNotFound},
		{name: "csrf", authenticate: func(*http.Request) (string, error) { return "user-1", nil }, identity: func(*http.Request) (contract.WorkspaceHTTPIdentity, error) {
			return contract.WorkspaceHTTPIdentity{WorkspaceID: "workspace-1"}, nil
		}, mutation: func(*http.Request) error { return contract.ErrWorkspacePermissionDenied }, want: http.StatusForbidden},
	} {
		t.Run(test.name, func(t *testing.T) {
			blocked := &recordingPinReorderService{}
			response := servePinReorder(t, blocked, `{"items":[{"id":"pin-1"}],"expected_revision":1}`, test.authenticate, test.identity, test.mutation)
			if response.Code != test.want || blocked.calls != 0 {
				t.Fatalf("status/calls = %d/%d body=%s", response.Code, blocked.calls, response.Body.String())
			}
		})
	}
}

func TestPinReorderRejectsUnknownAndTrailingRequestData(t *testing.T) {
	for _, body := range []string{
		`{"items":[{"id":"pin-1","position":1}],"expected_revision":1}`,
		`{"items":[{"id":"pin-1"}],"expected_revision":1,"unknown":true}`,
		`{"items":[{"id":"pin-1"}],"expected_revision":1} {}`,
	} {
		service := &recordingPinReorderService{}
		response := servePinReorder(t, service, body,
			func(*http.Request) (string, error) { return "user-1", nil },
			func(*http.Request) (contract.WorkspaceHTTPIdentity, error) {
				return contract.WorkspaceHTTPIdentity{WorkspaceID: "workspace-1", ActorType: "member", ActorID: "member-1"}, nil
			}, func(*http.Request) error { return nil })
		if response.Code != http.StatusBadRequest || service.calls != 0 {
			t.Fatalf("body %s status/calls = %d/%d response=%s", body, response.Code, service.calls, response.Body.String())
		}
	}
}

func servePinReorder(t *testing.T, service *recordingPinReorderService, body string, authenticate func(*http.Request) (string, error), identity contract.WorkspaceHTTPIdentityResolver, mutation func(*http.Request) error) *httptest.ResponseRecorder {
	t.Helper()
	server := kratoshttp.NewServer()
	NewProjectSurfaceHandler(service, identity, authenticate, mutation).Register(server)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/api/pins/reorder", strings.NewReader(body))
	request.Header.Set("X-Workspace-Slug", "workspace-1")
	server.ServeHTTP(response, request)
	return response
}

var _ contract.PinReorderService = (*recordingPinReorderService)(nil)
