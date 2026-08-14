package http

import (
	"errors"
	"net/http"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
)

type UserIDResolver func(*http.Request) (string, error)

type WorkspaceSelectionHandler struct {
	service           contract.WorkspaceSelectionService
	creator           contract.WorkspaceCreationService
	identity          UserIDResolver
	authorizeMutation func(*http.Request) error
}

func NewWorkspaceSelectionHandler(service contract.WorkspaceSelectionService, creator contract.WorkspaceCreationService, identity UserIDResolver, authorizeMutation func(*http.Request) error) *WorkspaceSelectionHandler {
	return &WorkspaceSelectionHandler{service: service, creator: creator, identity: identity, authorizeMutation: authorizeMutation}
}

func (h *WorkspaceSelectionHandler) Register(server *kratoshttp.Server) {
	router := server.Route("/")
	router.GET("/api/workspaces", h.list)
	if h.creator != nil && h.authorizeMutation != nil {
		router.POST("/api/workspaces", h.create)
	}
}

func (h *WorkspaceSelectionHandler) create(ctx kratoshttp.Context) error {
	userID, err := h.identity(ctx.Request())
	if err != nil {
		return writeError(ctx, http.StatusUnauthorized, "user not authenticated")
	}
	if err := h.authorizeMutation(ctx.Request()); err != nil {
		return writeError(ctx, http.StatusForbidden, "invalid CSRF token")
	}
	var request contract.CreateWorkspaceRequest
	if err := decodeJSON(ctx.Request().Body, &request); err != nil {
		return writeError(ctx, http.StatusBadRequest, "invalid request body")
	}
	value, err := h.creator.Create(ctx.Request().Context(), userID, request)
	if errors.Is(err, application.ErrInvalidWorkspace) {
		return writeError(ctx, http.StatusBadRequest, application.ErrInvalidWorkspace.Error())
	}
	if errors.Is(err, application.ErrWorkspaceSlugConflict) {
		return writeError(ctx, http.StatusConflict, application.ErrWorkspaceSlugConflict.Error())
	}
	if err != nil {
		return writeError(ctx, http.StatusInternalServerError, "failed to create workspace")
	}
	return ctx.JSON(http.StatusCreated, value)
}

func (h *WorkspaceSelectionHandler) list(ctx kratoshttp.Context) error {
	userID, err := h.identity(ctx.Request())
	if err != nil {
		return writeError(ctx, http.StatusUnauthorized, "user not authenticated")
	}
	values, err := h.service.List(ctx.Request().Context(), userID)
	if err != nil {
		return writeError(ctx, http.StatusInternalServerError, "failed to list workspaces")
	}
	return ctx.JSON(http.StatusOK, values)
}
