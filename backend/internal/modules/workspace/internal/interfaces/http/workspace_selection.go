package http

import (
	"net/http"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

type UserIDResolver func(*http.Request) (string, error)

type WorkspaceSelectionHandler struct {
	service  contract.WorkspaceSelectionService
	identity UserIDResolver
}

func NewWorkspaceSelectionHandler(service contract.WorkspaceSelectionService, identity UserIDResolver) *WorkspaceSelectionHandler {
	return &WorkspaceSelectionHandler{service: service, identity: identity}
}

func (h *WorkspaceSelectionHandler) Register(server *kratoshttp.Server) {
	server.Route("/").GET("/api/workspaces", h.list)
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
