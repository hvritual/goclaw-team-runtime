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

type recordingTaskService struct {
	contract.TodoService
	listRequest    contract.ListTodosRequest
	getRequest     contract.GetTodoRequest
	createRequest  contract.CreateTodoRequest
	updateRequest  contract.UpdateTodoRequest
	updateErr      error
	deleteRequest  contract.DeleteTodoRequest
	restoreRequest contract.RestoreTodoRequest
	reorderRequest contract.ReorderTodosRequest
}

func (s *recordingTaskService) ReorderTodos(_ context.Context, request contract.ReorderTodosRequest) (contract.ReorderTodosResponse, error) {
	s.reorderRequest = request
	return contract.ReorderTodosResponse{Todos: []contract.Todo{{
		Id: request.Items[0].TodoId, WorkspaceId: request.WorkspaceId, Title: "Moved", Status: "todo",
		Priority: "none", CreatorType: "member", CreatorId: "member-1", Position: request.Items[0].Position,
		Revision: request.Items[0].ExpectedRevision + 1,
	}}}, nil
}

func (s *recordingTaskService) GetTodo(_ context.Context, request contract.GetTodoRequest) (contract.GetTodoResponse, error) {
	s.getRequest = request
	return contract.GetTodoResponse{Todo: &contract.Todo{
		Id: request.TodoId, WorkspaceId: request.WorkspaceId, Title: "Detail",
		Status: "todo", Priority: "none", CreatorType: "member", CreatorId: "member-1", Revision: 1,
	}}, nil
}

func (s *recordingTaskService) DeleteTodo(_ context.Context, request contract.DeleteTodoRequest) (contract.DeleteTodoResponse, error) {
	s.deleteRequest = request
	return contract.DeleteTodoResponse{}, nil
}

func (s *recordingTaskService) RestoreTodo(_ context.Context, request contract.RestoreTodoRequest) (contract.RestoreTodoResponse, error) {
	s.restoreRequest = request
	return contract.RestoreTodoResponse{Todo: &contract.Todo{
		Id: request.TodoId, WorkspaceId: request.WorkspaceId, Title: "Restored",
		Status: "cancelled", Priority: "none", CreatorType: "member", CreatorId: "member-1",
		Revision: request.ExpectedRevision + 1,
	}}, nil
}

func (s *recordingTaskService) UpdateTodo(_ context.Context, request contract.UpdateTodoRequest) (contract.UpdateTodoResponse, error) {
	s.updateRequest = request
	if s.updateErr != nil {
		return contract.UpdateTodoResponse{}, s.updateErr
	}
	return contract.UpdateTodoResponse{Todo: &contract.Todo{Id: request.TodoId, WorkspaceId: request.WorkspaceId, Title: "Updated", Status: "in_progress", Priority: "none", CreatorType: "member", CreatorId: "member-1", Revision: request.ExpectedRevision + 1}}, nil
}

func (s *recordingTaskService) ListTodos(_ context.Context, request contract.ListTodosRequest) (contract.ListTodosResponse, error) {
	s.listRequest = request
	return contract.ListTodosResponse{Todos: []contract.Todo{{
		Id: "task-1", WorkspaceId: "workspace-1", Title: "Ship", Status: "done",
		Priority: "high", CreatorType: "member", CreatorId: "member-1", Revision: 3,
		CreatedAt: "2026-08-18T00:00:00Z", UpdatedAt: "2026-08-18T00:01:00Z",
	}}, Total: 1}, nil
}

func (s *recordingTaskService) CreateTodo(_ context.Context, request contract.CreateTodoRequest) (contract.CreateTodoResponse, error) {
	s.createRequest = request
	return contract.CreateTodoResponse{Todo: &contract.Todo{
		Id: "task-1", WorkspaceId: request.WorkspaceId, Title: request.Title,
		Status: "todo", Priority: "none", CreatorType: "member", CreatorId: "member-1",
		Revision: 1, CreatedAt: "2026-08-18T00:00:00Z", UpdatedAt: "2026-08-18T00:00:00Z",
	}}, nil
}

func TestTaskListUsesTrustedWorkspaceAndSnakeCaseResponse(t *testing.T) {
	service := &recordingTaskService{}
	server := kratoshttp.NewServer()
	handler := NewTaskHandler(
		service,
		func(*http.Request) (contract.WorkspaceHTTPIdentity, error) {
			return contract.WorkspaceHTTPIdentity{WorkspaceID: "workspace-1", ActorType: "member", ActorID: "member-1"}, nil
		},
		func(*http.Request) (string, error) { return "member-1", nil },
		func(*http.Request) error { return nil },
	)
	handler.Register(server)

	request := httptest.NewRequest(http.MethodGet, "/api/tasks?status=done&limit=250", nil)
	request.Header.Set("X-Workspace-ID", "untrusted-workspace")
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("list = %d %s", response.Code, response.Body.String())
	}
	if service.listRequest.WorkspaceId != "workspace-1" || service.listRequest.Status != "done" || service.listRequest.Limit != 100 {
		t.Fatalf("list request = %+v", service.listRequest)
	}
	body := strings.TrimSpace(response.Body.String())
	if !strings.Contains(body, `"workspace_id":"workspace-1"`) || !strings.Contains(body, `"revision":3`) || strings.Contains(body, "workspaceId") {
		t.Fatalf("list body = %s", body)
	}
}

func TestTaskListDefaultsAndValidatesLimit(t *testing.T) {
	service := &recordingTaskService{}
	server := kratoshttp.NewServer()
	handler := NewTaskHandler(
		service,
		func(*http.Request) (contract.WorkspaceHTTPIdentity, error) {
			return contract.WorkspaceHTTPIdentity{WorkspaceID: "workspace-1", ActorType: "member", ActorID: "member-1"}, nil
		},
		func(*http.Request) (string, error) { return "member-1", nil }, nil,
	)
	handler.Register(server)

	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/tasks", nil))
	if response.Code != http.StatusOK || service.listRequest.Limit != 50 {
		t.Fatalf("default list = %d request=%+v", response.Code, service.listRequest)
	}

	for _, raw := range []string{"0", "-1", "invalid"} {
		response = httptest.NewRecorder()
		server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/tasks?limit="+raw, nil))
		if response.Code != http.StatusBadRequest || strings.TrimSpace(response.Body.String()) != `{"error":"limit must be a positive integer"}` {
			t.Fatalf("limit %q = %d %s", raw, response.Code, response.Body.String())
		}
	}
}

func TestTaskDetailUsesTrustedWorkspace(t *testing.T) {
	service := &recordingTaskService{}
	server := kratoshttp.NewServer()
	handler := NewTaskHandler(
		service,
		func(*http.Request) (contract.WorkspaceHTTPIdentity, error) {
			return contract.WorkspaceHTTPIdentity{WorkspaceID: "workspace-1", ActorType: "member", ActorID: "member-1"}, nil
		},
		func(*http.Request) (string, error) { return "member-1", nil }, nil,
	)
	handler.Register(server)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/tasks/task-1", nil))
	if response.Code != http.StatusOK || service.getRequest.WorkspaceId != "workspace-1" || service.getRequest.TodoId != "task-1" || !strings.Contains(response.Body.String(), `"title":"Detail"`) {
		t.Fatalf("detail = %d %s request=%+v", response.Code, response.Body.String(), service.getRequest)
	}
}

func TestTaskCreateUsesTrustedWorkspaceAndReturns201(t *testing.T) {
	service := &recordingTaskService{}
	server := kratoshttp.NewServer()
	handler := NewTaskHandler(
		service,
		func(*http.Request) (contract.WorkspaceHTTPIdentity, error) {
			return contract.WorkspaceHTTPIdentity{WorkspaceID: "workspace-1", ActorType: "member", ActorID: "member-1"}, nil
		},
		func(*http.Request) (string, error) { return "member-1", nil },
		func(*http.Request) error { return nil },
	)
	handler.Register(server)

	request := httptest.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(`{"title":"Ship"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "create-ship")
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("create = %d %s", response.Code, response.Body.String())
	}
	if service.createRequest.WorkspaceId != "workspace-1" || service.createRequest.Title != "Ship" || service.createRequest.IdempotencyKey != "create-ship" {
		t.Fatalf("create request = %+v", service.createRequest)
	}
	if !strings.Contains(response.Body.String(), `"workspace_id":"workspace-1"`) || !strings.Contains(response.Body.String(), `"revision":1`) {
		t.Fatalf("create body = %s", response.Body.String())
	}
}

func TestTaskUpdateReturnsCanonicalRevisionConflict(t *testing.T) {
	service := &recordingTaskService{updateErr: contract.RevisionConflictError{CurrentRevision: 7}}
	server := kratoshttp.NewServer()
	handler := NewTaskHandler(
		service,
		func(*http.Request) (contract.WorkspaceHTTPIdentity, error) {
			return contract.WorkspaceHTTPIdentity{WorkspaceID: "workspace-1", ActorType: "member", ActorID: "member-1"}, nil
		},
		func(*http.Request) (string, error) { return "member-1", nil },
		func(*http.Request) error { return nil },
	)
	handler.Register(server)

	request := httptest.NewRequest(http.MethodPatch, "/api/tasks/task-1", strings.NewReader(`{"title":"Updated","expected_revision":2}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)

	if response.Code != http.StatusConflict || strings.TrimSpace(response.Body.String()) != `{"code":"revision_conflict","current_revision":7,"error":"revision conflict"}` {
		t.Fatalf("update conflict = %d %s", response.Code, response.Body.String())
	}
	if service.updateRequest.WorkspaceId != "workspace-1" || service.updateRequest.TodoId != "task-1" || service.updateRequest.ExpectedRevision != 2 {
		t.Fatalf("update request = %+v", service.updateRequest)
	}
}

func TestTaskArchiveAndRestoreUseExpectedRevision(t *testing.T) {
	service := &recordingTaskService{}
	server := kratoshttp.NewServer()
	handler := NewTaskHandler(
		service,
		func(*http.Request) (contract.WorkspaceHTTPIdentity, error) {
			return contract.WorkspaceHTTPIdentity{WorkspaceID: "workspace-1", ActorType: "member", ActorID: "member-1"}, nil
		},
		func(*http.Request) (string, error) { return "member-1", nil },
		func(*http.Request) error { return nil },
	)
	handler.Register(server)

	archive := httptest.NewRequest(http.MethodDelete, "/api/tasks/task-1", strings.NewReader(`{"expected_revision":3}`))
	archive.Header.Set("Content-Type", "application/json")
	archiveResponse := httptest.NewRecorder()
	server.ServeHTTP(archiveResponse, archive)
	if archiveResponse.Code != http.StatusNoContent || service.deleteRequest.ExpectedRevision != 3 {
		t.Fatalf("archive = %d %s request=%+v", archiveResponse.Code, archiveResponse.Body.String(), service.deleteRequest)
	}

	restore := httptest.NewRequest(http.MethodPost, "/api/tasks/task-1/restore", strings.NewReader(`{"expected_revision":4}`))
	restore.Header.Set("Content-Type", "application/json")
	restoreResponse := httptest.NewRecorder()
	server.ServeHTTP(restoreResponse, restore)
	if restoreResponse.Code != http.StatusOK || service.restoreRequest.ExpectedRevision != 4 || !strings.Contains(restoreResponse.Body.String(), `"revision":5`) {
		t.Fatalf("restore = %d %s request=%+v", restoreResponse.Code, restoreResponse.Body.String(), service.restoreRequest)
	}
}

func TestTaskReorderUsesTrustedWorkspaceAndExpectedRevisions(t *testing.T) {
	service := &recordingTaskService{}
	server := kratoshttp.NewServer()
	NewTaskHandler(
		service,
		func(*http.Request) (contract.WorkspaceHTTPIdentity, error) {
			return contract.WorkspaceHTTPIdentity{WorkspaceID: "workspace-1", ActorType: "member", ActorID: "member-1"}, nil
		},
		func(*http.Request) (string, error) { return "member-1", nil },
		func(*http.Request) error { return nil },
	).Register(server)

	request := httptest.NewRequest(http.MethodPost, "/api/tasks/reorder", strings.NewReader(`{"items":[{"id":"task-1","position":30,"expected_revision":2}]}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "reorder-1")
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("reorder = %d %s", response.Code, response.Body.String())
	}
	if service.reorderRequest.WorkspaceId != "workspace-1" || service.reorderRequest.IdempotencyKey != "reorder-1" || len(service.reorderRequest.Items) != 1 || service.reorderRequest.Items[0].TodoId != "task-1" || service.reorderRequest.Items[0].ExpectedRevision != 2 {
		t.Fatalf("reorder request = %+v", service.reorderRequest)
	}
	if !strings.Contains(response.Body.String(), `"position":30`) || !strings.Contains(response.Body.String(), `"revision":3`) {
		t.Fatalf("reorder body = %s", response.Body.String())
	}
}

func TestTaskReorderRequiresIdempotencyKey(t *testing.T) {
	service := &recordingTaskService{}
	server := kratoshttp.NewServer()
	NewTaskHandler(
		service,
		func(*http.Request) (contract.WorkspaceHTTPIdentity, error) {
			return contract.WorkspaceHTTPIdentity{WorkspaceID: "workspace-1", ActorType: "member", ActorID: "member-1"}, nil
		},
		func(*http.Request) (string, error) { return "member-1", nil },
		func(*http.Request) error { return nil },
	).Register(server)
	request := httptest.NewRequest(http.MethodPost, "/api/tasks/reorder", strings.NewReader(`{"items":[{"id":"task-1","position":30,"expected_revision":2}]}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || strings.TrimSpace(response.Body.String()) != `{"error":"idempotency key is required"}` {
		t.Fatalf("missing idempotency key = %d %s", response.Code, response.Body.String())
	}
}
