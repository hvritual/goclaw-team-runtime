package http

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
)

type ProjectRetrospectiveHandler struct {
	service      contract.ProjectRetrospectiveService
	identity     contract.WorkspaceHTTPIdentityResolver
	authenticate func(*http.Request) (string, error)
	mutation     func(*http.Request) error
}

func NewProjectRetrospectiveHandler(service contract.ProjectRetrospectiveService, identity contract.WorkspaceHTTPIdentityResolver, authenticate func(*http.Request) (string, error), mutation func(*http.Request) error) *ProjectRetrospectiveHandler {
	return &ProjectRetrospectiveHandler{service: service, identity: identity, authenticate: authenticate, mutation: mutation}
}

func (h *ProjectRetrospectiveHandler) Register(server *kratoshttp.Server) {
	router := server.Route("/")
	router.GET("/api/projects/{project_id}/retrospectives", h.list)
	router.POST("/api/projects/{project_id}/retrospectives", h.create)
	router.GET("/api/projects/{project_id}/retrospectives/{retrospective_id}", h.get)
	router.PUT("/api/projects/{project_id}/retrospectives/{retrospective_id}", h.update)
	router.DELETE("/api/projects/{project_id}/retrospectives/{retrospective_id}", h.archive)
	router.POST("/api/projects/{project_id}/retrospectives/{retrospective_id}/action-items/{action_item_id}/target", h.createTarget)
}

func (h *ProjectRetrospectiveHandler) list(ctx kratoshttp.Context) error {
	identity, ok := h.resolveIdentity(ctx)
	if !ok {
		return nil
	}
	limit, cursor, includeArchived, valid := projectRetrospectiveListQuery(ctx)
	if !valid {
		return writeProjectRetrospectiveProblem(ctx, http.StatusBadRequest, "invalid_request", application.ErrInvalidProjectRetrospectiveRequest.Error())
	}
	result, err := h.service.ListProjectRetrospectives(workspaceActorContext(ctx, identity), identity.WorkspaceID, ctx.Vars().Get("project_id"), limit, cursor, includeArchived)
	if err != nil {
		return h.writeFailure(ctx, err, "failed to list Project Retrospectives")
	}
	return ctx.JSON(http.StatusOK, result)
}

func (h *ProjectRetrospectiveHandler) get(ctx kratoshttp.Context) error {
	identity, ok := h.resolveIdentity(ctx)
	if !ok {
		return nil
	}
	if len(ctx.Request().URL.Query()) != 0 {
		return writeProjectRetrospectiveProblem(ctx, http.StatusBadRequest, "invalid_request", application.ErrInvalidProjectRetrospectiveRequest.Error())
	}
	result, err := h.service.GetProjectRetrospective(workspaceActorContext(ctx, identity), identity.WorkspaceID, ctx.Vars().Get("project_id"), ctx.Vars().Get("retrospective_id"))
	if err != nil {
		return h.writeFailure(ctx, err, "failed to read Project Retrospective")
	}
	return ctx.JSON(http.StatusOK, result)
}

func (h *ProjectRetrospectiveHandler) create(ctx kratoshttp.Context) error {
	identity, ok := h.resolveMutationIdentity(ctx)
	if !ok {
		return nil
	}
	if len(ctx.Request().URL.Query()) != 0 {
		return writeProjectRetrospectiveProblem(ctx, http.StatusBadRequest, "invalid_request", application.ErrInvalidProjectRetrospectiveRequest.Error())
	}
	key := strings.TrimSpace(ctx.Request().Header.Get("Idempotency-Key"))
	if key == "" || len(key) > 200 {
		return writeProjectRetrospectiveProblem(ctx, http.StatusBadRequest, "invalid_request", "idempotency key is required")
	}
	var request contract.CreateProjectRetrospectiveRequest
	if err := decodeJSON(ctx.Request().Body, &request); err != nil {
		return writeProjectRetrospectiveProblem(ctx, http.StatusBadRequest, "invalid_request", "invalid request body")
	}
	result, err := h.service.CreateProjectRetrospective(workspaceActorContext(ctx, identity), identity.WorkspaceID, ctx.Vars().Get("project_id"), key, request)
	if err != nil {
		return h.writeFailure(ctx, err, "failed to create Project Retrospective")
	}
	return ctx.JSON(http.StatusCreated, result)
}

func (h *ProjectRetrospectiveHandler) update(ctx kratoshttp.Context) error {
	identity, ok := h.resolveMutationIdentity(ctx)
	if !ok {
		return nil
	}
	if len(ctx.Request().URL.Query()) != 0 {
		return writeProjectRetrospectiveProblem(ctx, http.StatusBadRequest, "invalid_request", application.ErrInvalidProjectRetrospectiveRequest.Error())
	}
	var request contract.UpdateProjectRetrospectiveRequest
	if err := decodeJSON(ctx.Request().Body, &request); err != nil {
		return writeProjectRetrospectiveProblem(ctx, http.StatusBadRequest, "invalid_request", "invalid request body")
	}
	result, err := h.service.UpdateProjectRetrospective(workspaceActorContext(ctx, identity), identity.WorkspaceID, ctx.Vars().Get("project_id"), ctx.Vars().Get("retrospective_id"), request)
	if err != nil {
		return h.writeFailure(ctx, err, "failed to update Project Retrospective")
	}
	return ctx.JSON(http.StatusOK, result)
}

func (h *ProjectRetrospectiveHandler) archive(ctx kratoshttp.Context) error {
	identity, ok := h.resolveMutationIdentity(ctx)
	if !ok {
		return nil
	}
	expectedRevision, valid := exactExpectedRevisionQuery(ctx)
	if !valid {
		return writeProjectRetrospectiveProblem(ctx, http.StatusBadRequest, "invalid_request", "expected_revision is required")
	}
	result, err := h.service.ArchiveProjectRetrospective(workspaceActorContext(ctx, identity), identity.WorkspaceID, ctx.Vars().Get("project_id"), ctx.Vars().Get("retrospective_id"), expectedRevision)
	if err != nil {
		return h.writeFailure(ctx, err, "failed to archive Project Retrospective")
	}
	return ctx.JSON(http.StatusOK, result)
}

func (h *ProjectRetrospectiveHandler) createTarget(ctx kratoshttp.Context) error {
	identity, ok := h.resolveMutationIdentity(ctx)
	if !ok {
		return nil
	}
	if len(ctx.Request().URL.Query()) != 0 {
		return writeProjectRetrospectiveProblem(ctx, http.StatusBadRequest, "invalid_request", application.ErrInvalidProjectRetrospectiveRequest.Error())
	}
	key := strings.TrimSpace(ctx.Request().Header.Get("Idempotency-Key"))
	if key == "" || len(key) > 200 {
		return writeProjectRetrospectiveProblem(ctx, http.StatusBadRequest, "invalid_request", "idempotency key is required")
	}
	var request contract.CreateProjectRetrospectiveTargetRequest
	if err := decodeJSON(ctx.Request().Body, &request); err != nil {
		return writeProjectRetrospectiveProblem(ctx, http.StatusBadRequest, "invalid_request", "invalid request body")
	}
	result, err := h.service.CreateProjectRetrospectiveTarget(workspaceActorContext(ctx, identity), identity.WorkspaceID,
		ctx.Vars().Get("project_id"), ctx.Vars().Get("retrospective_id"), ctx.Vars().Get("action_item_id"), key, request)
	if err != nil {
		return h.writeFailure(ctx, err, "failed to create Project Retrospective action target")
	}
	return ctx.JSON(http.StatusCreated, result)
}

func (h *ProjectRetrospectiveHandler) resolveMutationIdentity(ctx kratoshttp.Context) (contract.WorkspaceHTTPIdentity, bool) {
	identity, ok := h.resolveIdentity(ctx)
	if !ok {
		return contract.WorkspaceHTTPIdentity{}, false
	}
	if h.mutation == nil || h.mutation(ctx.Request()) != nil {
		_ = writeProjectRetrospectiveProblem(ctx, http.StatusForbidden, "permission_denied", "invalid CSRF token")
		return contract.WorkspaceHTTPIdentity{}, false
	}
	return identity, true
}

func (h *ProjectRetrospectiveHandler) resolveIdentity(ctx kratoshttp.Context) (contract.WorkspaceHTTPIdentity, bool) {
	if h.authenticate == nil {
		_ = writeProjectRetrospectiveProblem(ctx, http.StatusUnauthorized, "unauthenticated", "user not authenticated")
		return contract.WorkspaceHTTPIdentity{}, false
	}
	if _, err := h.authenticate(ctx.Request()); err != nil {
		_ = writeProjectRetrospectiveProblem(ctx, http.StatusUnauthorized, "unauthenticated", "user not authenticated")
		return contract.WorkspaceHTTPIdentity{}, false
	}
	if !hasWorkspaceIdentity(ctx) {
		_ = writeProjectRetrospectiveProblem(ctx, http.StatusBadRequest, "invalid_request", "workspace is required")
		return contract.WorkspaceHTTPIdentity{}, false
	}
	if h.identity == nil {
		_ = writeProjectRetrospectiveProblem(ctx, http.StatusUnauthorized, "unauthenticated", "user not authenticated")
		return contract.WorkspaceHTTPIdentity{}, false
	}
	identity, err := h.identity(ctx.Request())
	if errors.Is(err, contract.ErrActorOutsideWorkspace) || errors.Is(err, contract.ErrWorkspaceNotFound) {
		_ = writeProjectRetrospectiveProblem(ctx, http.StatusNotFound, "not_found", contract.ErrProjectRetrospectiveNotFound.Error())
		return contract.WorkspaceHTTPIdentity{}, false
	}
	if err != nil || strings.TrimSpace(identity.WorkspaceID) == "" || strings.TrimSpace(identity.ActorID) == "" {
		_ = writeProjectRetrospectiveProblem(ctx, http.StatusUnauthorized, "unauthenticated", "user not authenticated")
		return contract.WorkspaceHTTPIdentity{}, false
	}
	return identity, true
}

func (h *ProjectRetrospectiveHandler) writeFailure(ctx kratoshttp.Context, err error, internalMessage string) error {
	var revisionConflict contract.RevisionConflictError
	switch {
	case errors.As(err, &revisionConflict):
		return ctx.JSON(http.StatusConflict, map[string]any{
			"code": "revision_conflict", "current_revision": revisionConflict.CurrentRevision, "error": contract.ErrRevisionConflict.Error(),
		})
	case errors.Is(err, application.ErrInvalidProjectRetrospectiveRequest), errors.Is(err, contract.ErrInvalidProjectRetrospective):
		return writeProjectRetrospectiveProblem(ctx, http.StatusBadRequest, "invalid_request", application.ErrInvalidProjectRetrospectiveRequest.Error())
	case errors.Is(err, contract.ErrWorkspacePermissionDenied), errors.Is(err, contract.ErrWorkspaceActorRequired):
		return writeProjectRetrospectiveProblem(ctx, http.StatusForbidden, "permission_denied", contract.ErrWorkspacePermissionDenied.Error())
	case errors.Is(err, contract.ErrProjectRetrospectiveNotFound), errors.Is(err, application.ErrProjectSurfaceNotFound), errors.Is(err, contract.ErrActorOutsideWorkspace), errors.Is(err, contract.ErrWorkspaceNotFound):
		return writeProjectRetrospectiveProblem(ctx, http.StatusNotFound, "not_found", contract.ErrProjectRetrospectiveNotFound.Error())
	case errors.Is(err, contract.ErrIdempotencyConflict):
		return writeProjectRetrospectiveProblem(ctx, http.StatusConflict, "idempotency_conflict", contract.ErrIdempotencyConflict.Error())
	case errors.Is(err, contract.ErrProjectRetrospectiveTargetConflict), errors.Is(err, contract.ErrInvalidTodo), errors.Is(err, contract.ErrInvalidIssue):
		return writeProjectRetrospectiveProblem(ctx, http.StatusConflict, "target_conflict", contract.ErrProjectRetrospectiveTargetConflict.Error())
	case errors.Is(err, contract.ErrProjectRetrospectiveStateConflict):
		return writeProjectRetrospectiveProblem(ctx, http.StatusConflict, "state_conflict", contract.ErrProjectRetrospectiveStateConflict.Error())
	default:
		return writeProjectRetrospectiveProblem(ctx, http.StatusInternalServerError, "internal_error", internalMessage)
	}
}

func projectRetrospectiveListQuery(ctx kratoshttp.Context) (limit int, cursor string, includeArchived bool, valid bool) {
	values := ctx.Request().URL.Query()
	allowed := map[string]struct{}{"limit": {}, "cursor": {}, "include_archived": {}}
	for key, entries := range values {
		if _, ok := allowed[key]; !ok || len(entries) != 1 {
			return 0, "", false, false
		}
	}
	if raw, present := values["limit"]; present {
		parsed, err := strconv.Atoi(strings.TrimSpace(raw[0]))
		if err != nil || parsed < 1 || parsed > application.MaxProjectRetrospectiveListLimit {
			return 0, "", false, false
		}
		limit = parsed
	}
	if raw, present := values["cursor"]; present {
		cursor = strings.TrimSpace(raw[0])
		if cursor == "" {
			return 0, "", false, false
		}
	}
	if raw, present := values["include_archived"]; present {
		var err error
		includeArchived, err = strconv.ParseBool(strings.TrimSpace(raw[0]))
		if err != nil {
			return 0, "", false, false
		}
	}
	return limit, cursor, includeArchived, true
}

func writeProjectRetrospectiveProblem(ctx kratoshttp.Context, status int, code, message string) error {
	return ctx.JSON(status, map[string]string{"code": code, "error": message})
}
