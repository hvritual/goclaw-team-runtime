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

type ProjectResourceHandler struct {
	service      contract.ProjectResourceService
	identity     contract.WorkspaceHTTPIdentityResolver
	authenticate func(*http.Request) (string, error)
	mutation     func(*http.Request) error
}

func NewProjectResourceHandler(service contract.ProjectResourceService, identity contract.WorkspaceHTTPIdentityResolver, authenticate func(*http.Request) (string, error), mutation func(*http.Request) error) *ProjectResourceHandler {
	return &ProjectResourceHandler{service: service, identity: identity, authenticate: authenticate, mutation: mutation}
}

func (h *ProjectResourceHandler) Register(server *kratoshttp.Server) {
	router := server.Route("/")
	router.GET("/api/projects/{id}/resources", h.list)
	router.POST("/api/projects/{id}/resources", h.create)
	router.PUT("/api/projects/{id}/resources/{resource_id}", h.update)
	router.DELETE("/api/projects/{id}/resources/{resource_id}", h.archive)
}

func (h *ProjectResourceHandler) list(ctx kratoshttp.Context) error {
	identity, ok := h.resolveIdentity(ctx)
	if !ok {
		return nil
	}
	includeArchived := false
	if raw := strings.TrimSpace(ctx.Request().URL.Query().Get("include_archived")); raw != "" {
		var err error
		includeArchived, err = strconv.ParseBool(raw)
		if err != nil {
			return writeError(ctx, http.StatusBadRequest, "invalid include_archived")
		}
	}
	result, err := h.service.ListProjectResources(workspaceActorContext(ctx, identity), identity.WorkspaceID, ctx.Vars().Get("id"), includeArchived)
	if err != nil {
		return h.writeFailure(ctx, err, "failed to list Project Resources")
	}
	return ctx.JSON(http.StatusOK, result)
}

func (h *ProjectResourceHandler) create(ctx kratoshttp.Context) error {
	identity, ok := h.resolveIdentity(ctx)
	if !ok {
		return nil
	}
	if !h.authorizeMutation(ctx) {
		return nil
	}
	key := strings.TrimSpace(ctx.Request().Header.Get("Idempotency-Key"))
	if key == "" {
		return writeError(ctx, http.StatusBadRequest, "idempotency key is required")
	}
	var request contract.CreateProjectResourceRequest
	if err := decodeJSON(ctx.Request().Body, &request); err != nil {
		return writeError(ctx, http.StatusBadRequest, "invalid request body")
	}
	result, err := h.service.CreateProjectResource(workspaceActorContext(ctx, identity), identity.WorkspaceID, ctx.Vars().Get("id"), key, request)
	if err != nil {
		return h.writeFailure(ctx, err, "failed to create Project Resource")
	}
	return ctx.JSON(http.StatusCreated, result)
}

func (h *ProjectResourceHandler) update(ctx kratoshttp.Context) error {
	identity, ok := h.resolveIdentity(ctx)
	if !ok {
		return nil
	}
	if !h.authorizeMutation(ctx) {
		return nil
	}
	var request contract.UpdateProjectResourceRequest
	if err := decodeJSON(ctx.Request().Body, &request); err != nil {
		return writeError(ctx, http.StatusBadRequest, "invalid request body")
	}
	result, err := h.service.UpdateProjectResource(workspaceActorContext(ctx, identity), identity.WorkspaceID, ctx.Vars().Get("id"), ctx.Vars().Get("resource_id"), request)
	if err != nil {
		return h.writeFailure(ctx, err, "failed to update Project Resource")
	}
	return ctx.JSON(http.StatusOK, result)
}

func (h *ProjectResourceHandler) archive(ctx kratoshttp.Context) error {
	identity, ok := h.resolveIdentity(ctx)
	if !ok {
		return nil
	}
	if !h.authorizeMutation(ctx) {
		return nil
	}
	var request struct {
		ExpectedRevision int64 `json:"expected_revision"`
	}
	if err := decodeJSON(ctx.Request().Body, &request); err != nil {
		return writeError(ctx, http.StatusBadRequest, "invalid request body")
	}
	err := h.service.ArchiveProjectResource(workspaceActorContext(ctx, identity), identity.WorkspaceID, ctx.Vars().Get("id"), ctx.Vars().Get("resource_id"), request.ExpectedRevision)
	if err != nil {
		return h.writeFailure(ctx, err, "failed to archive Project Resource")
	}
	ctx.Response().WriteHeader(http.StatusNoContent)
	return nil
}

func (h *ProjectResourceHandler) resolveIdentity(ctx kratoshttp.Context) (contract.WorkspaceHTTPIdentity, bool) {
	if h.authenticate == nil {
		_ = writeError(ctx, http.StatusUnauthorized, "user not authenticated")
		return contract.WorkspaceHTTPIdentity{}, false
	}
	if _, err := h.authenticate(ctx.Request()); err != nil {
		_ = writeError(ctx, http.StatusUnauthorized, "user not authenticated")
		return contract.WorkspaceHTTPIdentity{}, false
	}
	if !hasWorkspaceIdentity(ctx) {
		_ = writeError(ctx, http.StatusBadRequest, "workspace is required")
		return contract.WorkspaceHTTPIdentity{}, false
	}
	if h.identity == nil {
		_ = writeError(ctx, http.StatusUnauthorized, "user not authenticated")
		return contract.WorkspaceHTTPIdentity{}, false
	}
	identity, err := h.identity(ctx.Request())
	if errors.Is(err, contract.ErrActorOutsideWorkspace) || errors.Is(err, contract.ErrWorkspaceNotFound) {
		_ = writeError(ctx, http.StatusNotFound, "project not found")
		return contract.WorkspaceHTTPIdentity{}, false
	}
	if err != nil {
		_ = writeError(ctx, http.StatusUnauthorized, "user not authenticated")
		return contract.WorkspaceHTTPIdentity{}, false
	}
	return identity, true
}

func (h *ProjectResourceHandler) authorizeMutation(ctx kratoshttp.Context) bool {
	if h.mutation != nil && h.mutation(ctx.Request()) == nil {
		return true
	}
	_ = writeError(ctx, http.StatusForbidden, "invalid CSRF token")
	return false
}

func (h *ProjectResourceHandler) writeFailure(ctx kratoshttp.Context, err error, internalMessage string) error {
	var conflict contract.RevisionConflictError
	switch {
	case errors.As(err, &conflict):
		return ctx.JSON(http.StatusConflict, map[string]any{
			"code": "revision_conflict", "current_revision": conflict.CurrentRevision,
			"error": contract.ErrRevisionConflict.Error(),
		})
	case errors.Is(err, application.ErrProjectResourceConflict):
		return writeError(ctx, http.StatusConflict, application.ErrProjectResourceConflict.Error())
	case errors.Is(err, application.ErrInvalidProjectResourceRequest):
		return writeError(ctx, http.StatusBadRequest, application.ErrInvalidProjectResourceRequest.Error())
	case errors.Is(err, application.ErrProjectResourceNotFound), errors.Is(err, application.ErrProjectSurfaceNotFound), errors.Is(err, contract.ErrActorOutsideWorkspace), errors.Is(err, contract.ErrWorkspaceNotFound):
		return writeError(ctx, http.StatusNotFound, "Project Resource not found")
	case errors.Is(err, contract.ErrWorkspacePermissionDenied), errors.Is(err, contract.ErrWorkspaceActorRequired):
		return writeError(ctx, http.StatusForbidden, contract.ErrWorkspacePermissionDenied.Error())
	default:
		return writeError(ctx, http.StatusInternalServerError, internalMessage)
	}
}
