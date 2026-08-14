package http

import (
	"context"
	"errors"
	"net/http"
	"strings"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
)

type batchUpdateIssuesHTTPRequest struct {
	IssueIDs []string               `json:"issue_ids"`
	Updates  updateIssueHTTPRequest `json:"updates"`
}

type batchDeleteIssuesHTTPRequest struct {
	IssueIDs []string `json:"issue_ids"`
}

func (h *IssueReadHandler) hierarchyService() contract.IssueHierarchyService {
	service, _ := h.service.(contract.IssueHierarchyService)
	return service
}

func (h *IssueReadHandler) listChildren(ctx kratoshttp.Context) error {
	requestContext, workspaceID, ok := h.issueHierarchyReadIdentity(ctx)
	if !ok {
		return nil
	}
	result, err := h.hierarchyService().ListIssueChildren(requestContext, contract.ListIssueChildrenRequest{
		WorkspaceID: workspaceID,
		IssueID:     ctx.Vars().Get("id"),
	})
	if err != nil {
		return h.writeHierarchyError(ctx, err, "list child issues")
	}
	issues := make([]publicIssue, len(result.Issues))
	for index := range result.Issues {
		issues[index] = toPublicIssue(result.Issues[index])
	}
	return ctx.JSON(http.StatusOK, map[string]any{"issues": issues})
}

func (h *IssueReadHandler) listChildrenByParents(ctx kratoshttp.Context) error {
	requestContext, workspaceID, ok := h.issueHierarchyReadIdentity(ctx)
	if !ok {
		return nil
	}
	raw := ctx.Request().URL.Query().Get("parent_ids")
	parentIDs := strings.Split(raw, ",")
	if strings.TrimSpace(raw) == "" {
		parentIDs = nil
	}
	result, err := h.hierarchyService().ListIssueChildrenByParents(requestContext, contract.ListIssueChildrenByParentsRequest{
		WorkspaceID: workspaceID,
		ParentIDs:   parentIDs,
	})
	if err != nil {
		return h.writeHierarchyError(ctx, err, "list child issues")
	}
	issues := make([]publicIssue, len(result.Issues))
	for index := range result.Issues {
		issues[index] = toPublicIssue(result.Issues[index])
	}
	return ctx.JSON(http.StatusOK, map[string]any{"issues": issues})
}

func (h *IssueReadHandler) childProgress(ctx kratoshttp.Context) error {
	requestContext, workspaceID, ok := h.issueHierarchyReadIdentity(ctx)
	if !ok {
		return nil
	}
	result, err := h.hierarchyService().ListIssueChildProgress(requestContext, contract.ListIssueChildProgressRequest{WorkspaceID: workspaceID})
	if err != nil {
		return h.writeHierarchyError(ctx, err, "list child issue progress")
	}
	progress := make([]map[string]any, len(result.Progress))
	for index, row := range result.Progress {
		progress[index] = map[string]any{"parent_issue_id": row.ParentIssueID, "total": row.Total, "done": row.Done}
	}
	return ctx.JSON(http.StatusOK, map[string]any{"progress": progress})
}

func (h *IssueReadHandler) batchUpdate(ctx kratoshttp.Context) error {
	requestContext, workspaceID, ok := h.issueMutationIdentity(ctx)
	if !ok {
		return nil
	}
	var request batchUpdateIssuesHTTPRequest
	if err := decodeJSON(ctx.Request().Body, &request); err != nil {
		return writeError(ctx, http.StatusBadRequest, "invalid request body")
	}
	updates := contract.UpdateIssueRequest{
		Title: request.Updates.Title, Description: request.Updates.Description,
		Status: request.Updates.Status, Priority: request.Updates.Priority,
		AssigneeType: nullableStringUpdate(request.Updates.AssigneeType), AssigneeId: nullableStringUpdate(request.Updates.AssigneeID),
		ParentIssueId: nullableStringUpdate(request.Updates.ParentIssueID), ProjectId: nullableStringUpdate(request.Updates.ProjectID),
		Position: request.Updates.Position, Stage: nullableStageUpdate(request.Updates.Stage),
		StartDate: nullableStringUpdate(request.Updates.StartDate), DueDate: nullableStringUpdate(request.Updates.DueDate),
	}
	result, err := h.hierarchyService().BatchUpdateIssues(requestContext, contract.BatchUpdateIssuesRequest{
		WorkspaceID: workspaceID,
		IssueIDs:    request.IssueIDs,
		Updates:     updates,
		HasMutation: hasIssueUpdate(request.Updates),
	})
	if err != nil {
		return h.writeHierarchyError(ctx, err, "batch update issues")
	}
	return ctx.JSON(http.StatusOK, map[string]any{"updated": result.Updated})
}

func (h *IssueReadHandler) batchDelete(ctx kratoshttp.Context) error {
	requestContext, workspaceID, ok := h.issueMutationIdentity(ctx)
	if !ok {
		return nil
	}
	var request batchDeleteIssuesHTTPRequest
	if err := decodeJSON(ctx.Request().Body, &request); err != nil {
		return writeError(ctx, http.StatusBadRequest, "invalid request body")
	}
	result, err := h.hierarchyService().BatchDeleteIssues(requestContext, contract.BatchDeleteIssuesRequest{WorkspaceID: workspaceID, IssueIDs: request.IssueIDs})
	if err != nil {
		return h.writeHierarchyError(ctx, err, "batch delete issues")
	}
	return ctx.JSON(http.StatusOK, map[string]any{"deleted": result.Deleted})
}

func (h *IssueReadHandler) issueMutationIdentity(ctx kratoshttp.Context) (context.Context, string, bool) {
	if h.authenticate == nil {
		_ = writeError(ctx, http.StatusUnauthorized, "user not authenticated")
		return nil, "", false
	}
	if _, err := h.authenticate(ctx.Request()); err != nil {
		_ = writeError(ctx, http.StatusUnauthorized, "user not authenticated")
		return nil, "", false
	}
	if !hasWorkspaceIdentity(ctx) {
		_ = writeError(ctx, http.StatusBadRequest, "workspace is required")
		return nil, "", false
	}
	requestContext, workspaceID, err := h.requestIdentity(ctx)
	if err != nil {
		_ = issueReadIdentityError(ctx, err)
		return nil, "", false
	}
	if h.mutation == nil || h.mutation(ctx.Request()) != nil {
		_ = writeError(ctx, http.StatusForbidden, "invalid CSRF token")
		return nil, "", false
	}
	return requestContext, workspaceID, true
}

func (h *IssueReadHandler) issueHierarchyReadIdentity(ctx kratoshttp.Context) (context.Context, string, bool) {
	if h.authenticate == nil {
		_ = writeError(ctx, http.StatusUnauthorized, "user not authenticated")
		return nil, "", false
	}
	if _, err := h.authenticate(ctx.Request()); err != nil {
		_ = writeError(ctx, http.StatusUnauthorized, "user not authenticated")
		return nil, "", false
	}
	if !hasWorkspaceIdentity(ctx) {
		_ = writeError(ctx, http.StatusBadRequest, "workspace is required")
		return nil, "", false
	}
	requestContext, workspaceID, err := h.requestIdentity(ctx)
	if err != nil {
		_ = issueReadIdentityError(ctx, err)
		return nil, "", false
	}
	return requestContext, workspaceID, true
}

func hasIssueUpdate(request updateIssueHTTPRequest) bool {
	return request.Title != nil || request.Description != nil || request.Status != nil || request.Priority != nil ||
		request.AssigneeType.Set || request.AssigneeID.Set || request.ParentIssueID.Set || request.ProjectID.Set ||
		request.Position != nil || request.Stage.Set || request.StartDate.Set || request.DueDate.Set
}

func (h *IssueReadHandler) writeHierarchyError(ctx kratoshttp.Context, err error, operation string) error {
	if errors.Is(err, contract.ErrInvalidIssue) || errors.Is(err, application.ErrIssueBatchInvalid) {
		return writeError(ctx, http.StatusBadRequest, "invalid issue request")
	}
	if errors.Is(err, application.ErrIssueBatchConflict) {
		return writeError(ctx, http.StatusConflict, "issue batch conflict")
	}
	if errors.Is(err, contract.ErrIssueNotFound) || errors.Is(err, contract.ErrProjectNotFound) || errors.Is(err, contract.ErrActorOutsideWorkspace) || errors.Is(err, contract.ErrWorkspaceNotFound) {
		return writeError(ctx, http.StatusNotFound, "issue not found")
	}
	return writeError(ctx, http.StatusInternalServerError, "failed to "+operation)
}
