package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/engineering/contract"
)

type recordingService struct {
	createEntityCalls int
	actor             contract.Actor
	workspaceID       string
	createRequest     contract.CreateEntityRequest
	createEntityErr   error
}

func (s *recordingService) CreateEntity(_ context.Context, actor contract.Actor, workspaceID string, request contract.CreateEntityRequest) (contract.Entity, error) {
	s.createEntityCalls++
	s.actor = actor
	s.workspaceID = workspaceID
	s.createRequest = request
	if s.createEntityErr != nil {
		return contract.Entity{}, s.createEntityErr
	}
	return contract.Entity{ID: request.ID, WorkspaceID: workspaceID, Type: request.Type, Name: request.Name, Status: request.Status, OwnerRef: request.OwnerRef}, nil
}
func (*recordingService) GetEntity(context.Context, contract.Actor, string, string) (contract.Entity, error) {
	return contract.Entity{}, contract.ErrNotFound
}
func (*recordingService) ListEntities(context.Context, contract.Actor, string) ([]contract.Entity, error) {
	return []contract.Entity{}, nil
}
func (*recordingService) UpdateEntity(context.Context, contract.Actor, string, string, contract.UpdateEntityRequest) (contract.Entity, error) {
	return contract.Entity{}, nil
}
func (*recordingService) CreateSourceBinding(context.Context, contract.Actor, string, contract.CreateSourceBindingRequest) (contract.SourceBinding, error) {
	return contract.SourceBinding{}, nil
}
func (*recordingService) GetSourceBinding(context.Context, contract.Actor, string, string) (contract.SourceBinding, error) {
	return contract.SourceBinding{}, contract.ErrNotFound
}
func (*recordingService) ListSourceBindings(context.Context, contract.Actor, string, string) ([]contract.SourceBinding, error) {
	return []contract.SourceBinding{}, nil
}
func (*recordingService) CreateThreadEdge(context.Context, contract.Actor, string, contract.CreateThreadEdgeRequest) (contract.ThreadEdge, error) {
	return contract.ThreadEdge{}, nil
}
func (*recordingService) ListThreadEdges(context.Context, contract.Actor, string, contract.NodeRef) ([]contract.ThreadEdge, error) {
	return []contract.ThreadEdge{}, nil
}
func (*recordingService) CreateChange(context.Context, contract.Actor, string, contract.CreateChangeRequest) (contract.Change, error) {
	return contract.Change{}, nil
}
func (*recordingService) GetChange(context.Context, contract.Actor, string, string) (contract.Change, error) {
	return contract.Change{}, contract.ErrNotFound
}
func (*recordingService) ListChanges(context.Context, contract.Actor, string, string) ([]contract.Change, error) {
	return []contract.Change{}, nil
}
func (*recordingService) GetContextPack(context.Context, contract.Actor, string, string) (contract.ContextPack, error) {
	return contract.ContextPack{}, contract.ErrNotFound
}
func (*recordingService) RecordEvidence(context.Context, contract.Actor, string, contract.RecordEvidenceRequest) (contract.Evidence, error) {
	return contract.Evidence{}, nil
}
func (*recordingService) GetEvidence(context.Context, contract.Actor, string, string) (contract.Evidence, error) {
	return contract.Evidence{}, contract.ErrNotFound
}
func (*recordingService) ListEvidence(context.Context, contract.Actor, string, *contract.NodeRef) ([]contract.Evidence, error) {
	return []contract.Evidence{}, nil
}

func TestCreateEntityPassesAuthenticatedWorkspaceActor(t *testing.T) {
	service := &recordingService{}
	response := serve(t, service, func(*http.Request) (string, error) { return "user-1", nil }, http.MethodPost,
		"/api/engineering/v1/workspaces/workspace-1/entities", `{"id":"service-1","type":"service","name":"Device Gateway","status":"active"}`)
	if response.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if service.createEntityCalls != 1 || service.actor.UserID != "user-1" || service.workspaceID != "workspace-1" {
		t.Fatalf("calls=%d actor=%+v workspace=%q", service.createEntityCalls, service.actor, service.workspaceID)
	}
	if service.createRequest.ID != "service-1" || service.createRequest.Type != "service" || service.createRequest.Name != "Device Gateway" {
		t.Fatalf("request=%+v", service.createRequest)
	}
	if !strings.Contains(response.Body.String(), `"workspace_id":"workspace-1"`) {
		t.Fatalf("body=%s", response.Body.String())
	}
}

func TestMalformedEntityRequestFailsBeforeApplicationService(t *testing.T) {
	service := &recordingService{}
	response := serve(t, service, func(*http.Request) (string, error) { return "user-1", nil }, http.MethodPost,
		"/api/engineering/v1/workspaces/workspace-1/entities", `{"id":"service-1","type":"service","name":"Device Gateway","unknown":true}`)
	if response.Code != http.StatusBadRequest || service.createEntityCalls != 0 {
		t.Fatalf("status=%d calls=%d body=%s", response.Code, service.createEntityCalls, response.Body.String())
	}
}

func TestAuthenticationFailureStopsBeforeApplicationService(t *testing.T) {
	service := &recordingService{}
	response := serve(t, service, func(*http.Request) (string, error) { return "", errors.New("expired") }, http.MethodPost,
		"/api/engineering/v1/workspaces/workspace-1/entities", `{"id":"service-1","type":"service","name":"Device Gateway"}`)
	if response.Code != http.StatusUnauthorized || service.createEntityCalls != 0 {
		t.Fatalf("status=%d calls=%d body=%s", response.Code, service.createEntityCalls, response.Body.String())
	}
}

func TestApplicationAuthorizationErrorsMapToHTTP(t *testing.T) {
	service := &recordingService{createEntityErr: contract.ErrForbidden}
	response := serve(t, service, func(*http.Request) (string, error) { return "member-1", nil }, http.MethodPost,
		"/api/engineering/v1/workspaces/workspace-1/entities", `{"id":"service-1","type":"service","name":"Device Gateway"}`)
	if response.Code != http.StatusForbidden || service.createEntityCalls != 1 {
		t.Fatalf("status=%d calls=%d body=%s", response.Code, service.createEntityCalls, response.Body.String())
	}
}

func TestQueryRoutesRejectMissingTypedSelectors(t *testing.T) {
	service := &recordingService{}
	authenticate := func(*http.Request) (string, error) { return "user-1", nil }
	response := serve(t, service, authenticate, http.MethodGet, "/api/engineering/v1/workspaces/workspace-1/source-bindings", "")
	if response.Code != http.StatusBadRequest {
		t.Fatalf("source-binding status=%d body=%s", response.Code, response.Body.String())
	}
	response = serve(t, service, authenticate, http.MethodGet, "/api/engineering/v1/workspaces/workspace-1/thread-edges?node_kind=engineering_entity", "")
	if response.Code != http.StatusBadRequest {
		t.Fatalf("thread-edge status=%d body=%s", response.Code, response.Body.String())
	}
}

func serve(t *testing.T, service contract.Service, authenticate func(*http.Request) (string, error), method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	server := kratoshttp.NewServer()
	NewHandler(service, authenticate).Register(server)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	server.ServeHTTP(response, request)
	return response
}

var _ contract.Service = (*recordingService)(nil)
