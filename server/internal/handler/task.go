package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
	"github.com/multica-ai/multica/server/pkg/protocol"
)

var validTaskStatuses = map[string]struct{}{
	"todo":        {},
	"in_progress": {},
	"done":        {},
	"cancelled":   {},
}

var validTaskPriorities = map[string]struct{}{
	"urgent": {},
	"high":   {},
	"medium": {},
	"low":    {},
	"none":   {},
}

type TaskResponse struct {
	ID          string  `json:"id"`
	WorkspaceID string  `json:"workspace_id"`
	ProjectID   *string `json:"project_id"`
	IssueID     *string `json:"issue_id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Status      string  `json:"status"`
	Priority    string  `json:"priority"`
	AssigneeID  *string `json:"assignee_id"`
	CreatorID   string  `json:"creator_id"`
	Position    float64 `json:"position"`
	StartDate   *string `json:"start_date"`
	DueDate     *string `json:"due_date"`
	CompletedAt *string `json:"completed_at"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

type CreateTaskRequest struct {
	ProjectID   *string  `json:"project_id"`
	IssueID     *string  `json:"issue_id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
	Priority    string   `json:"priority"`
	AssigneeID  *string  `json:"assignee_id"`
	Position    *float64 `json:"position"`
	StartDate   *string  `json:"start_date"`
	DueDate     *string  `json:"due_date"`
}

type UpdateTaskRequest struct {
	ProjectID   *string  `json:"project_id"`
	IssueID     *string  `json:"issue_id"`
	Title       *string  `json:"title"`
	Description *string  `json:"description"`
	Status      *string  `json:"status"`
	Priority    *string  `json:"priority"`
	AssigneeID  *string  `json:"assignee_id"`
	Position    *float64 `json:"position"`
	StartDate   *string  `json:"start_date"`
	DueDate     *string  `json:"due_date"`
}

func taskToResponse(task db.Task) TaskResponse {
	return TaskResponse{
		ID:          uuidToString(task.ID),
		WorkspaceID: uuidToString(task.WorkspaceID),
		ProjectID:   uuidToPtr(task.ProjectID),
		IssueID:     uuidToPtr(task.IssueID),
		Title:       task.Title,
		Description: task.Description,
		Status:      task.Status,
		Priority:    task.Priority,
		AssigneeID:  uuidToPtr(task.AssigneeID),
		CreatorID:   uuidToString(task.CreatorID),
		Position:    task.Position,
		StartDate:   timestampToPtr(task.StartDate),
		DueDate:     timestampToPtr(task.DueDate),
		CompletedAt: timestampToPtr(task.CompletedAt),
		CreatedAt:   timestampToString(task.CreatedAt),
		UpdatedAt:   timestampToString(task.UpdatedAt),
	}
}

func parseOptionalTimestamp(
	w http.ResponseWriter,
	value *string,
	field string,
) (pgtype.Timestamptz, bool) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return pgtype.Timestamptz{}, true
	}
	parsed, err := time.Parse(time.RFC3339, *value)
	if err != nil {
		writeError(w, http.StatusBadRequest, field+" must be an RFC3339 timestamp")
		return pgtype.Timestamptz{}, false
	}
	return pgtype.Timestamptz{Time: parsed, Valid: true}, true
}

func validateTaskStatus(w http.ResponseWriter, value string) bool {
	if _, ok := validTaskStatuses[value]; ok {
		return true
	}
	writeError(w, http.StatusBadRequest, "invalid task status")
	return false
}

func validateTaskPriority(w http.ResponseWriter, value string) bool {
	if _, ok := validTaskPriorities[value]; ok {
		return true
	}
	writeError(w, http.StatusBadRequest, "invalid task priority")
	return false
}

func (h *Handler) ListTasks(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := parseUUIDOrBadRequest(
		w,
		h.resolveWorkspaceID(r),
		"workspace_id",
	)
	if !ok {
		return
	}
	params := db.ListTasksParams{WorkspaceID: workspaceID}
	if value := r.URL.Query().Get("project_id"); value != "" {
		params.ProjectID, ok = parseUUIDOrBadRequest(w, value, "project_id")
		if !ok {
			return
		}
	}
	if value := r.URL.Query().Get("issue_id"); value != "" {
		params.IssueID, ok = parseUUIDOrBadRequest(w, value, "issue_id")
		if !ok {
			return
		}
	}
	if value := r.URL.Query().Get("status"); value != "" {
		if !validateTaskStatus(w, value) {
			return
		}
		params.Status = pgtype.Text{String: value, Valid: true}
	}

	tasks, err := h.Queries.ListTasks(r.Context(), params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list tasks")
		return
	}
	response := make([]TaskResponse, len(tasks))
	for index, task := range tasks {
		response[index] = taskToResponse(task)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"tasks": response,
		"total": len(response),
	})
}

func (h *Handler) GetTask(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "id"), "task id")
	if !ok {
		return
	}
	workspaceID, ok := parseUUIDOrBadRequest(
		w,
		h.resolveWorkspaceID(r),
		"workspace_id",
	)
	if !ok {
		return
	}
	task, err := h.Queries.GetTask(r.Context(), db.GetTaskParams{
		ID:          id,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	writeJSON(w, http.StatusOK, taskToResponse(task))
}

func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	var request CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	request.Title = strings.TrimSpace(request.Title)
	if request.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	if request.Status == "" {
		request.Status = "todo"
	}
	if request.Priority == "" {
		request.Priority = "none"
	}
	if !validateTaskStatus(w, request.Status) ||
		!validateTaskPriority(w, request.Priority) {
		return
	}

	workspaceID, ok := parseUUIDOrBadRequest(
		w,
		h.resolveWorkspaceID(r),
		"workspace_id",
	)
	if !ok {
		return
	}
	creatorID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	params := db.CreateTaskParams{
		WorkspaceID: workspaceID,
		Title:       request.Title,
		Description: request.Description,
		Status:      request.Status,
		Priority:    request.Priority,
		CreatorID:   parseUUID(creatorID),
	}
	if request.Position != nil {
		params.Position = *request.Position
	}
	if request.ProjectID != nil {
		params.ProjectID, ok = parseUUIDOrBadRequest(w, *request.ProjectID, "project_id")
		if !ok {
			return
		}
		if _, err := h.Queries.GetProjectInWorkspace(r.Context(), db.GetProjectInWorkspaceParams{
			ID: params.ProjectID, WorkspaceID: workspaceID,
		}); err != nil {
			writeError(w, http.StatusBadRequest, "project does not belong to workspace")
			return
		}
	}
	if request.IssueID != nil {
		params.IssueID, ok = parseUUIDOrBadRequest(w, *request.IssueID, "issue_id")
		if !ok {
			return
		}
		if _, err := h.Queries.GetIssueInWorkspace(r.Context(), db.GetIssueInWorkspaceParams{
			ID: params.IssueID, WorkspaceID: workspaceID,
		}); err != nil {
			writeError(w, http.StatusBadRequest, "issue does not belong to workspace")
			return
		}
	}
	if request.AssigneeID != nil {
		params.AssigneeID, ok = parseUUIDOrBadRequest(w, *request.AssigneeID, "assignee_id")
		if !ok {
			return
		}
		if !h.isWorkspaceEntity(
			r.Context(),
			"member",
			*request.AssigneeID,
			uuidToString(workspaceID),
		) {
			writeError(w, http.StatusBadRequest, "assignee must be a workspace member")
			return
		}
	}
	params.StartDate, ok = parseOptionalTimestamp(w, request.StartDate, "start_date")
	if !ok {
		return
	}
	params.DueDate, ok = parseOptionalTimestamp(w, request.DueDate, "due_date")
	if !ok {
		return
	}

	task, err := h.Queries.CreateTask(r.Context(), params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create task")
		return
	}
	response := taskToResponse(task)
	h.publish(protocol.EventTaskCreated, uuidToString(workspaceID), "member", creatorID, map[string]any{"task": response})
	writeJSON(w, http.StatusCreated, response)
}

func (h *Handler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "id"), "task id")
	if !ok {
		return
	}
	workspaceID, ok := parseUUIDOrBadRequest(
		w,
		h.resolveWorkspaceID(r),
		"workspace_id",
	)
	if !ok {
		return
	}
	existing, err := h.Queries.GetTask(r.Context(), db.GetTaskParams{
		ID: id, WorkspaceID: workspaceID,
	})
	if err != nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	var request UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	params := db.UpdateTaskParams{
		ID:          existing.ID,
		WorkspaceID: existing.WorkspaceID,
		ProjectID:   existing.ProjectID,
		IssueID:     existing.IssueID,
		AssigneeID:  existing.AssigneeID,
		StartDate:   existing.StartDate,
		DueDate:     existing.DueDate,
	}
	if request.Title != nil {
		title := strings.TrimSpace(*request.Title)
		if title == "" {
			writeError(w, http.StatusBadRequest, "title is required")
			return
		}
		params.Title = pgtype.Text{String: title, Valid: true}
	}
	if request.Description != nil {
		params.Description = pgtype.Text{String: *request.Description, Valid: true}
	}
	if request.Status != nil {
		if !validateTaskStatus(w, *request.Status) {
			return
		}
		params.Status = pgtype.Text{String: *request.Status, Valid: true}
	}
	if request.Priority != nil {
		if !validateTaskPriority(w, *request.Priority) {
			return
		}
		params.Priority = pgtype.Text{String: *request.Priority, Valid: true}
	}
	if request.Position != nil {
		params.Position = pgtype.Float8{Float64: *request.Position, Valid: true}
	}
	if request.ProjectID != nil {
		params.ProjectID, ok = parseUUIDOrBadRequest(w, *request.ProjectID, "project_id")
		if !ok {
			return
		}
		if _, err := h.Queries.GetProjectInWorkspace(
			r.Context(),
			db.GetProjectInWorkspaceParams{
				ID: params.ProjectID, WorkspaceID: workspaceID,
			},
		); err != nil {
			writeError(w, http.StatusBadRequest, "project does not belong to workspace")
			return
		}
	}
	if request.IssueID != nil {
		params.IssueID, ok = parseUUIDOrBadRequest(w, *request.IssueID, "issue_id")
		if !ok {
			return
		}
		if _, err := h.Queries.GetIssueInWorkspace(
			r.Context(),
			db.GetIssueInWorkspaceParams{
				ID: params.IssueID, WorkspaceID: workspaceID,
			},
		); err != nil {
			writeError(w, http.StatusBadRequest, "issue does not belong to workspace")
			return
		}
	}
	if request.AssigneeID != nil {
		params.AssigneeID, ok = parseUUIDOrBadRequest(w, *request.AssigneeID, "assignee_id")
		if !ok {
			return
		}
		if !h.isWorkspaceEntity(
			r.Context(),
			"member",
			*request.AssigneeID,
			uuidToString(workspaceID),
		) {
			writeError(w, http.StatusBadRequest, "assignee must be a workspace member")
			return
		}
	}
	if request.StartDate != nil {
		params.StartDate, ok = parseOptionalTimestamp(w, request.StartDate, "start_date")
		if !ok {
			return
		}
	}
	if request.DueDate != nil {
		params.DueDate, ok = parseOptionalTimestamp(w, request.DueDate, "due_date")
		if !ok {
			return
		}
	}
	task, err := h.Queries.UpdateTask(r.Context(), params)
	if err != nil {
		if err == pgx.ErrNoRows {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update task")
		return
	}
	response := taskToResponse(task)
	h.publish(protocol.EventTaskUpdated, uuidToString(workspaceID), "member", requestUserID(r), map[string]any{"task": response})
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "id"), "task id")
	if !ok {
		return
	}
	workspaceID, ok := parseUUIDOrBadRequest(
		w,
		h.resolveWorkspaceID(r),
		"workspace_id",
	)
	if !ok {
		return
	}
	affected, err := h.Queries.DeleteTask(r.Context(), db.DeleteTaskParams{
		ID: id, WorkspaceID: workspaceID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete task")
		return
	}
	if affected == 0 {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	h.publish(protocol.EventTaskDeleted, uuidToString(workspaceID), "member", requestUserID(r), map[string]any{"task_id": uuidToString(id)})
	w.WriteHeader(http.StatusNoContent)
}
