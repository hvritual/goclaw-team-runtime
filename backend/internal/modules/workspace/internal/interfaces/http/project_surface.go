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

type ProjectSurfaceHandler struct {
	service      contract.ProjectSurfaceService
	identity     contract.WorkspaceHTTPIdentityResolver
	authenticate func(*http.Request) (string, error)
	mutation     func(*http.Request) error
}

func NewProjectSurfaceHandler(service contract.ProjectSurfaceService, identity contract.WorkspaceHTTPIdentityResolver, authenticate func(*http.Request) (string, error), mutation func(*http.Request) error) *ProjectSurfaceHandler {
	return &ProjectSurfaceHandler{service: service, identity: identity, authenticate: authenticate, mutation: mutation}
}

func (h *ProjectSurfaceHandler) Register(server *kratoshttp.Server) {
	router := server.Route("/")
	router.GET("/api/projects", h.listProjects)
	router.POST("/api/projects", h.createProject)
	router.GET("/api/projects/{id}", h.getProject)
	router.PUT("/api/projects/{id}", h.updateProject)
	router.DELETE("/api/projects/{id}", h.deleteProject)
	router.GET("/api/pins", h.listPins)
	router.POST("/api/pins", h.createPin)
	router.DELETE("/api/pins/{item_type}/{item_id}", h.deletePin)
}

func (h *ProjectSurfaceHandler) listProjects(ctx kratoshttp.Context) error {
	if _, err := h.authenticateRequest(ctx.Request()); err != nil {
		return writeError(ctx, http.StatusUnauthorized, "user not authenticated")
	}
	if !hasWorkspaceIdentity(ctx) {
		return writeError(ctx, http.StatusBadRequest, "workspace is required")
	}
	identity, err := h.workspaceIdentity(ctx.Request())
	if err != nil {
		return issueReadIdentityError(ctx, err)
	}
	result, err := h.service.ListProjects(workspaceActorContext(ctx, identity), identity.WorkspaceID, ctx.Request().URL.Query().Get("status"))
	if errors.Is(err, application.ErrInvalidProjectSurfaceRequest) {
		return writeError(ctx, http.StatusBadRequest, "invalid project request")
	}
	if projectSurfaceHidden(err) {
		return writeError(ctx, http.StatusNotFound, "project not found")
	}
	if err != nil {
		return writeError(ctx, http.StatusInternalServerError, "failed to list projects")
	}
	return ctx.JSON(http.StatusOK, result)
}

func (h *ProjectSurfaceHandler) getProject(ctx kratoshttp.Context) error {
	identity, ok := h.resolveIdentity(ctx)
	if !ok {
		return nil
	}
	result, err := h.service.GetProject(workspaceActorContext(ctx, identity), identity.WorkspaceID, ctx.Vars().Get("id"))
	return h.writeProject(ctx, result, err, http.StatusOK)
}

func (h *ProjectSurfaceHandler) createProject(ctx kratoshttp.Context) error {
	identity, ok := h.resolveIdentity(ctx)
	if !ok {
		return nil
	}
	if h.mutation == nil || h.mutation(ctx.Request()) != nil {
		return writeError(ctx, http.StatusForbidden, "invalid CSRF token")
	}
	var request contract.CreateProjectSurfaceRequest
	if err := decodeJSON(ctx.Request().Body, &request); err != nil {
		return writeError(ctx, http.StatusBadRequest, "invalid request body")
	}
	result, err := h.service.CreateProject(workspaceActorContext(ctx, identity), identity.WorkspaceID, request)
	return h.writeProject(ctx, result, err, http.StatusCreated)
}

func (h *ProjectSurfaceHandler) updateProject(ctx kratoshttp.Context) error {
	identity, ok := h.resolveIdentity(ctx)
	if !ok {
		return nil
	}
	if h.mutation == nil || h.mutation(ctx.Request()) != nil {
		return writeError(ctx, http.StatusForbidden, "invalid CSRF token")
	}
	if _, err := h.service.GetProject(workspaceActorContext(ctx, identity), identity.WorkspaceID, ctx.Vars().Get("id")); err != nil {
		return h.writeProject(ctx, contract.ProjectSurfaceProject{}, err, http.StatusOK)
	}
	var request contract.UpdateProjectSurfaceRequest
	if err := decodeJSON(ctx.Request().Body, &request); err != nil {
		return writeError(ctx, http.StatusBadRequest, "invalid request body")
	}
	result, err := h.service.UpdateProject(workspaceActorContext(ctx, identity), identity.WorkspaceID, ctx.Vars().Get("id"), request)
	return h.writeProject(ctx, result, err, http.StatusOK)
}

func (h *ProjectSurfaceHandler) deleteProject(ctx kratoshttp.Context) error {
	identity, ok := h.resolveIdentity(ctx)
	if !ok {
		return nil
	}
	if h.mutation == nil || h.mutation(ctx.Request()) != nil {
		return writeError(ctx, http.StatusForbidden, "invalid CSRF token")
	}
	err := h.service.DeleteProject(workspaceActorContext(ctx, identity), identity.WorkspaceID, ctx.Vars().Get("id"))
	if errors.Is(err, contract.ErrWorkspacePermissionDenied) {
		return writeError(ctx, http.StatusForbidden, contract.ErrWorkspacePermissionDenied.Error())
	}
	if errors.Is(err, application.ErrProjectSurfaceNotFound) {
		return writeError(ctx, http.StatusNotFound, "project not found")
	}
	if projectSurfaceHidden(err) {
		return writeError(ctx, http.StatusNotFound, "project not found")
	}
	if errors.Is(err, application.ErrInvalidProjectSurfaceRequest) {
		return writeError(ctx, http.StatusBadRequest, "invalid project request")
	}
	if err != nil {
		return writeError(ctx, http.StatusInternalServerError, "failed to delete project")
	}
	ctx.Response().WriteHeader(http.StatusNoContent)
	return nil
}

func (h *ProjectSurfaceHandler) listPins(ctx kratoshttp.Context) error {
	userID, err := h.authenticateRequest(ctx.Request())
	if err != nil {
		return writeError(ctx, http.StatusUnauthorized, "user not authenticated")
	}
	if !hasWorkspaceIdentity(ctx) {
		return writeError(ctx, http.StatusBadRequest, "workspace is required")
	}
	identity, err := h.workspaceIdentity(ctx.Request())
	if err != nil {
		return issueReadIdentityError(ctx, err)
	}
	result, err := h.service.ListPins(workspaceActorContext(ctx, identity), identity.WorkspaceID, userID)
	if projectSurfaceHidden(err) {
		return writeError(ctx, http.StatusNotFound, "workspace not found")
	}
	if err != nil {
		return writeError(ctx, http.StatusInternalServerError, "failed to list pins")
	}
	return ctx.JSON(http.StatusOK, result)
}

func (h *ProjectSurfaceHandler) createPin(ctx kratoshttp.Context) error {
	userID, err := h.authenticateRequest(ctx.Request())
	if err != nil {
		return writeError(ctx, http.StatusUnauthorized, "user not authenticated")
	}
	if !hasWorkspaceIdentity(ctx) {
		return writeError(ctx, http.StatusBadRequest, "workspace is required")
	}
	identity, err := h.workspaceIdentity(ctx.Request())
	if err != nil {
		return issueReadIdentityError(ctx, err)
	}
	if h.mutation == nil || h.mutation(ctx.Request()) != nil {
		return writeError(ctx, http.StatusForbidden, "invalid CSRF token")
	}
	var request contract.CreatePinRequest
	if err := decodeJSON(ctx.Request().Body, &request); err != nil {
		return writeError(ctx, http.StatusBadRequest, "invalid request body")
	}
	result, err := h.service.CreatePin(workspaceActorContext(ctx, identity), identity.WorkspaceID, userID, request)
	if projectSurfaceHidden(err) {
		return writeError(ctx, http.StatusNotFound, "workspace not found")
	}
	if errors.Is(err, application.ErrPinConflict) {
		return writeError(ctx, http.StatusConflict, application.ErrPinConflict.Error())
	}
	if errors.Is(err, application.ErrPinTargetNotFound) {
		return writeError(ctx, http.StatusNotFound, strings.TrimSpace(request.ItemType)+" not found")
	}
	if errors.Is(err, application.ErrInvalidProjectSurfaceRequest) {
		return writeError(ctx, http.StatusBadRequest, "invalid pin request")
	}
	if err != nil {
		return writeError(ctx, http.StatusInternalServerError, "failed to create pin")
	}
	return ctx.JSON(http.StatusCreated, result)
}

func (h *ProjectSurfaceHandler) deletePin(ctx kratoshttp.Context) error {
	userID, err := h.authenticateRequest(ctx.Request())
	if err != nil {
		return writeError(ctx, http.StatusUnauthorized, "user not authenticated")
	}
	if !hasWorkspaceIdentity(ctx) {
		return writeError(ctx, http.StatusBadRequest, "workspace is required")
	}
	identity, err := h.workspaceIdentity(ctx.Request())
	if err != nil {
		return issueReadIdentityError(ctx, err)
	}
	if h.mutation == nil || h.mutation(ctx.Request()) != nil {
		return writeError(ctx, http.StatusForbidden, "invalid CSRF token")
	}
	err = h.service.DeletePin(workspaceActorContext(ctx, identity), identity.WorkspaceID, userID, ctx.Vars().Get("item_type"), ctx.Vars().Get("item_id"))
	if projectSurfaceHidden(err) {
		return writeError(ctx, http.StatusNotFound, "workspace not found")
	}
	if errors.Is(err, application.ErrInvalidProjectSurfaceRequest) {
		return writeError(ctx, http.StatusBadRequest, "invalid pin request")
	}
	if err != nil {
		return writeError(ctx, http.StatusInternalServerError, "failed to delete pin")
	}
	ctx.Response().WriteHeader(http.StatusNoContent)
	return nil
}

func (h *ProjectSurfaceHandler) resolveIdentity(ctx kratoshttp.Context) (contract.WorkspaceHTTPIdentity, bool) {
	if _, err := h.authenticateRequest(ctx.Request()); err != nil {
		_ = writeError(ctx, http.StatusUnauthorized, "user not authenticated")
		return contract.WorkspaceHTTPIdentity{}, false
	}
	if !hasWorkspaceIdentity(ctx) {
		_ = writeError(ctx, http.StatusBadRequest, "workspace is required")
		return contract.WorkspaceHTTPIdentity{}, false
	}
	identity, err := h.workspaceIdentity(ctx.Request())
	if err != nil {
		_ = issueReadIdentityError(ctx, err)
		return contract.WorkspaceHTTPIdentity{}, false
	}
	return identity, true
}

func (h *ProjectSurfaceHandler) writeProject(ctx kratoshttp.Context, result contract.ProjectSurfaceProject, err error, status int) error {
	if errors.Is(err, application.ErrProjectSurfaceNotFound) {
		return writeError(ctx, http.StatusNotFound, "project not found")
	}
	if errors.Is(err, application.ErrInvalidProjectSurfaceRequest) {
		return writeError(ctx, http.StatusBadRequest, "invalid project request")
	}
	if projectSurfaceHidden(err) {
		return writeError(ctx, http.StatusNotFound, "project not found")
	}
	if err != nil {
		return writeError(ctx, http.StatusInternalServerError, "project operation failed")
	}
	return ctx.JSON(status, result)
}

func (h *ProjectSurfaceHandler) authenticateRequest(request *http.Request) (string, error) {
	if h.authenticate == nil {
		return "", contract.ErrWorkspaceActorRequired
	}
	return h.authenticate(request)
}

func (h *ProjectSurfaceHandler) workspaceIdentity(request *http.Request) (contract.WorkspaceHTTPIdentity, error) {
	if h.identity == nil {
		return contract.WorkspaceHTTPIdentity{}, contract.ErrWorkspaceActorRequired
	}
	return h.identity(request)
}

func projectSurfaceHidden(err error) bool {
	return errors.Is(err, contract.ErrWorkspaceNotFound) || errors.Is(err, contract.ErrActorOutsideWorkspace)
}

func workspaceActorContext(ctx kratoshttp.Context, identity contract.WorkspaceHTTPIdentity) context.Context {
	return contract.WithWorkspaceActor(ctx.Request().Context(), identity.ActorType, identity.ActorID)
}
