package http

import (
	"errors"
	"net/http"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

type TaskHandler struct {
	service      contract.TodoService
	identity     contract.WorkspaceHTTPIdentityResolver
	authenticate func(*http.Request) (string, error)
	mutation     func(*http.Request) error
}

func NewTaskHandler(service contract.TodoService, identity contract.WorkspaceHTTPIdentityResolver, authenticate func(*http.Request) (string, error), mutation func(*http.Request) error) *TaskHandler {
	return &TaskHandler{service: service, identity: identity, authenticate: authenticate, mutation: mutation}
}

func (h *TaskHandler) Register(server *kratoshttp.Server) {
	router := server.Route("/")
	router.GET("/api/tasks", h.list)
	router.POST("/api/tasks/reorder", h.reorder)
	router.GET("/api/tasks/{id}", h.get)
	router.POST("/api/tasks", h.create)
	router.PATCH("/api/tasks/{id}", h.update)
	router.DELETE("/api/tasks/{id}", h.archive)
	router.POST("/api/tasks/{id}/restore", h.restore)
}

func (h *TaskHandler) reorder(ctx kratoshttp.Context) error {
	identity, ok := h.resolveIdentity(ctx)
	if !ok {
		return nil
	}
	if h.mutation == nil || h.mutation(ctx.Request()) != nil {
		return writeError(ctx, http.StatusForbidden, "invalid CSRF token")
	}
	var input struct {
		Items []struct {
			ID               string  `json:"id"`
			Position         float64 `json:"position"`
			ExpectedRevision int64   `json:"expected_revision"`
		} `json:"items"`
	}
	if err := decodeJSON(ctx.Request().Body, &input); err != nil {
		return writeError(ctx, http.StatusBadRequest, "invalid request body")
	}
	request := contract.ReorderTodosRequest{WorkspaceId: identity.WorkspaceID, Items: make([]contract.ReorderTodoItem, len(input.Items))}
	for index, item := range input.Items {
		request.Items[index] = contract.ReorderTodoItem{TodoId: item.ID, Position: item.Position, ExpectedRevision: item.ExpectedRevision}
	}
	result, err := h.service.ReorderTodos(workspaceActorContext(ctx, identity), request)
	if err != nil {
		return h.writeError(ctx, err)
	}
	items := make([]taskResponse, len(result.Todos))
	for index := range result.Todos {
		items[index] = taskToResponse(result.Todos[index])
	}
	return ctx.JSON(http.StatusOK, map[string]any{"tasks": items})
}

func (h *TaskHandler) get(ctx kratoshttp.Context) error {
	identity, ok := h.resolveIdentity(ctx)
	if !ok {
		return nil
	}
	result, err := h.service.GetTodo(workspaceActorContext(ctx, identity), contract.GetTodoRequest{
		WorkspaceId: identity.WorkspaceID, TodoId: ctx.Vars().Get("id"),
	})
	if err != nil {
		return h.writeError(ctx, err)
	}
	if result.Todo == nil {
		return writeError(ctx, http.StatusInternalServerError, "task operation failed")
	}
	return ctx.JSON(http.StatusOK, taskToResponse(*result.Todo))
}

func (h *TaskHandler) archive(ctx kratoshttp.Context) error {
	identity, ok := h.resolveIdentity(ctx)
	if !ok {
		return nil
	}
	if h.mutation == nil || h.mutation(ctx.Request()) != nil {
		return writeError(ctx, http.StatusForbidden, "invalid CSRF token")
	}
	var input taskCommandRequest
	if err := decodeJSON(ctx.Request().Body, &input); err != nil {
		return writeError(ctx, http.StatusBadRequest, "invalid request body")
	}
	_, err := h.service.DeleteTodo(workspaceActorContext(ctx, identity), contract.DeleteTodoRequest{
		WorkspaceId: identity.WorkspaceID, TodoId: ctx.Vars().Get("id"), ExpectedRevision: input.ExpectedRevision,
	})
	if err != nil {
		return h.writeError(ctx, err)
	}
	ctx.Response().WriteHeader(http.StatusNoContent)
	return nil
}

func (h *TaskHandler) restore(ctx kratoshttp.Context) error {
	identity, ok := h.resolveIdentity(ctx)
	if !ok {
		return nil
	}
	if h.mutation == nil || h.mutation(ctx.Request()) != nil {
		return writeError(ctx, http.StatusForbidden, "invalid CSRF token")
	}
	var input taskCommandRequest
	if err := decodeJSON(ctx.Request().Body, &input); err != nil {
		return writeError(ctx, http.StatusBadRequest, "invalid request body")
	}
	result, err := h.service.RestoreTodo(workspaceActorContext(ctx, identity), contract.RestoreTodoRequest{
		WorkspaceId: identity.WorkspaceID, TodoId: ctx.Vars().Get("id"), ExpectedRevision: input.ExpectedRevision,
	})
	if err != nil {
		return h.writeError(ctx, err)
	}
	if result.Todo == nil {
		return writeError(ctx, http.StatusInternalServerError, "task operation failed")
	}
	return ctx.JSON(http.StatusOK, taskToResponse(*result.Todo))
}

func (h *TaskHandler) update(ctx kratoshttp.Context) error {
	identity, ok := h.resolveIdentity(ctx)
	if !ok {
		return nil
	}
	if h.mutation == nil || h.mutation(ctx.Request()) != nil {
		return writeError(ctx, http.StatusForbidden, "invalid CSRF token")
	}
	var input taskUpdateRequest
	if err := decodeJSON(ctx.Request().Body, &input); err != nil {
		return writeError(ctx, http.StatusBadRequest, "invalid request body")
	}
	result, err := h.service.UpdateTodo(workspaceActorContext(ctx, identity), contract.UpdateTodoRequest{
		WorkspaceId: identity.WorkspaceID, TodoId: ctx.Vars().Get("id"),
		Title: input.Title, Description: input.Description, ProjectId: input.ProjectID,
		IssueId: input.IssueID, Status: input.Status, Priority: input.Priority,
		AssigneeType: input.AssigneeType, AssigneeId: input.AssigneeID,
		Position: input.Position, StartDate: input.StartDate, DueDate: input.DueDate,
		ExpectedRevision: input.ExpectedRevision,
	})
	if err != nil {
		return h.writeError(ctx, err)
	}
	if result.Todo == nil {
		return writeError(ctx, http.StatusInternalServerError, "task operation failed")
	}
	return ctx.JSON(http.StatusOK, taskToResponse(*result.Todo))
}

func (h *TaskHandler) create(ctx kratoshttp.Context) error {
	identity, ok := h.resolveIdentity(ctx)
	if !ok {
		return nil
	}
	if h.mutation == nil || h.mutation(ctx.Request()) != nil {
		return writeError(ctx, http.StatusForbidden, "invalid CSRF token")
	}
	idempotencyKey := ctx.Request().Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		return writeError(ctx, http.StatusBadRequest, "idempotency key is required")
	}
	var input taskMutationRequest
	if err := decodeJSON(ctx.Request().Body, &input); err != nil {
		return writeError(ctx, http.StatusBadRequest, "invalid request body")
	}
	result, err := h.service.CreateTodo(workspaceActorContext(ctx, identity), contract.CreateTodoRequest{
		IdempotencyKey: idempotencyKey,
		WorkspaceId:    identity.WorkspaceID, Title: input.Title, Description: input.Description,
		ProjectId: input.ProjectID, IssueId: input.IssueID, AssigneeType: input.AssigneeType,
		AssigneeId: input.AssigneeID, Status: input.Status, Priority: input.Priority,
		Position: input.Position, StartDate: input.StartDate, DueDate: input.DueDate,
	})
	if err != nil {
		return h.writeError(ctx, err)
	}
	if result.Todo == nil {
		return writeError(ctx, http.StatusInternalServerError, "task operation failed")
	}
	return ctx.JSON(http.StatusCreated, taskToResponse(*result.Todo))
}

func (h *TaskHandler) list(ctx kratoshttp.Context) error {
	identity, ok := h.resolveIdentity(ctx)
	if !ok {
		return nil
	}
	query := ctx.Request().URL.Query()
	result, err := h.service.ListTodos(workspaceActorContext(ctx, identity), contract.ListTodosRequest{
		WorkspaceId: identity.WorkspaceID,
		ProjectId:   optionalTaskQuery(query.Get("project_id")),
		IssueId:     optionalTaskQuery(query.Get("issue_id")),
		Status:      query.Get("status"),
	})
	if err != nil {
		return h.writeError(ctx, err)
	}
	items := make([]taskResponse, len(result.Todos))
	for index := range result.Todos {
		items[index] = taskToResponse(result.Todos[index])
	}
	return ctx.JSON(http.StatusOK, map[string]any{"tasks": items, "total": result.Total})
}

func (h *TaskHandler) resolveIdentity(ctx kratoshttp.Context) (contract.WorkspaceHTTPIdentity, bool) {
	if h.authenticate == nil {
		_ = writeError(ctx, http.StatusUnauthorized, "user not authenticated")
		return contract.WorkspaceHTTPIdentity{}, false
	}
	if _, err := h.authenticate(ctx.Request()); err != nil {
		_ = writeError(ctx, http.StatusUnauthorized, "user not authenticated")
		return contract.WorkspaceHTTPIdentity{}, false
	}
	if h.identity == nil {
		_ = writeError(ctx, http.StatusBadRequest, "workspace is required")
		return contract.WorkspaceHTTPIdentity{}, false
	}
	identity, err := h.identity(ctx.Request())
	if err != nil {
		_ = issueReadIdentityError(ctx, err)
		return contract.WorkspaceHTTPIdentity{}, false
	}
	return identity, true
}

func (h *TaskHandler) writeError(ctx kratoshttp.Context, err error) error {
	var conflict contract.RevisionConflictError
	if errors.As(err, &conflict) {
		return ctx.JSON(http.StatusConflict, map[string]any{
			"code": "revision_conflict", "current_revision": conflict.CurrentRevision,
			"error": contract.ErrRevisionConflict.Error(),
		})
	}
	if errors.Is(err, contract.ErrIdempotencyConflict) {
		return ctx.JSON(http.StatusConflict, map[string]any{
			"code": "idempotency_conflict", "error": contract.ErrIdempotencyConflict.Error(),
		})
	}
	if errors.Is(err, contract.ErrWorkspacePermissionDenied) {
		return writeError(ctx, http.StatusForbidden, contract.ErrWorkspacePermissionDenied.Error())
	}
	if errors.Is(err, contract.ErrTodoNotFound) || errors.Is(err, contract.ErrActorOutsideWorkspace) {
		return writeError(ctx, http.StatusNotFound, "task not found")
	}
	if errors.Is(err, contract.ErrInvalidTodo) {
		return writeError(ctx, http.StatusBadRequest, contract.ErrInvalidTodo.Error())
	}
	return writeError(ctx, http.StatusInternalServerError, "task operation failed")
}

func optionalTaskQuery(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

type taskResponse struct {
	ID            string  `json:"id"`
	WorkspaceID   string  `json:"workspace_id"`
	ProjectID     *string `json:"project_id"`
	IssueID       *string `json:"issue_id"`
	Title         string  `json:"title"`
	Description   string  `json:"description"`
	Status        string  `json:"status"`
	Priority      string  `json:"priority"`
	AssigneeType  *string `json:"assignee_type"`
	AssigneeID    *string `json:"assignee_id"`
	CreatorType   string  `json:"creator_type"`
	CreatorID     string  `json:"creator_id"`
	Position      float64 `json:"position"`
	Revision      int64   `json:"revision"`
	StartDate     *string `json:"start_date"`
	DueDate       *string `json:"due_date"`
	CompletedAt   *string `json:"completed_at"`
	ArchivedAt    *string `json:"archived_at"`
	RestoreStatus string  `json:"restore_status"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

type taskMutationRequest struct {
	Title            string  `json:"title"`
	Description      string  `json:"description"`
	ProjectID        *string `json:"project_id"`
	IssueID          *string `json:"issue_id"`
	Status           string  `json:"status"`
	Priority         string  `json:"priority"`
	AssigneeType     *string `json:"assignee_type"`
	AssigneeID       *string `json:"assignee_id"`
	Position         float64 `json:"position"`
	StartDate        *string `json:"start_date"`
	DueDate          *string `json:"due_date"`
	ExpectedRevision int64   `json:"expected_revision"`
}

type taskUpdateRequest struct {
	Title            *string  `json:"title"`
	Description      *string  `json:"description"`
	ProjectID        *string  `json:"project_id"`
	IssueID          *string  `json:"issue_id"`
	Status           *string  `json:"status"`
	Priority         *string  `json:"priority"`
	AssigneeType     *string  `json:"assignee_type"`
	AssigneeID       *string  `json:"assignee_id"`
	Position         *float64 `json:"position"`
	StartDate        *string  `json:"start_date"`
	DueDate          *string  `json:"due_date"`
	ExpectedRevision int64    `json:"expected_revision"`
}

type taskCommandRequest struct {
	ExpectedRevision int64 `json:"expected_revision"`
}

func taskToResponse(value contract.Todo) taskResponse {
	return taskResponse{
		ID: value.Id, WorkspaceID: value.WorkspaceId, ProjectID: value.ProjectId,
		IssueID: value.IssueId, Title: value.Title, Description: value.Description,
		Status: value.Status, Priority: value.Priority, AssigneeType: value.AssigneeType,
		AssigneeID: value.AssigneeId, CreatorType: value.CreatorType, CreatorID: value.CreatorId,
		Position: value.Position, Revision: value.Revision, StartDate: value.StartDate,
		DueDate: value.DueDate, CompletedAt: value.CompletedAt, ArchivedAt: value.ArchivedAt,
		RestoreStatus: value.RestoreStatus, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt,
	}
}
