package http

import (
	"errors"
	"net/http"
	"strings"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/engineering/contract"
)

type LifecycleHandler struct {
	service      contract.LifecycleService
	authenticate func(*http.Request) (string, error)
}

func NewLifecycleHandler(service contract.LifecycleService, authenticate func(*http.Request) (string, error)) *LifecycleHandler {
	return &LifecycleHandler{service: service, authenticate: authenticate}
}

func (h *LifecycleHandler) Register(server *kratoshttp.Server) {
	if h == nil || server == nil {
		return
	}
	router := server.Route("/")
	router.POST("/api/engineering/v1/workspaces/{workspace}/changes/{id}/accept", h.acceptChange)
	router.POST("/api/engineering/v1/workspaces/{workspace}/context-packs", h.freezeContextPack)
}

func (h *LifecycleHandler) acceptChange(ctx kratoshttp.Context) error {
	actor, workspaceID, ok := h.authorizedActor(ctx)
	if !ok {
		return nil
	}
	value, err := h.service.AcceptChange(ctx.Request().Context(), actor, workspaceID, strings.TrimSpace(ctx.Vars().Get("id")))
	if err != nil {
		return writeLifecycleError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, value)
}

func (h *LifecycleHandler) freezeContextPack(ctx kratoshttp.Context) error {
	actor, workspaceID, ok := h.authorizedActor(ctx)
	if !ok {
		return nil
	}
	var request contract.FreezeContextPackRequest
	if err := decodeJSON(ctx.Request(), &request); err != nil {
		return writeError(ctx, http.StatusBadRequest, "invalid context pack request")
	}
	value, err := h.service.FreezeContextPack(ctx.Request().Context(), actor, workspaceID, request)
	if err != nil {
		return writeLifecycleError(ctx, err)
	}
	return ctx.JSON(http.StatusCreated, value)
}

func (h *LifecycleHandler) authorizedActor(ctx kratoshttp.Context) (contract.Actor, string, bool) {
	if h == nil || h.service == nil || h.authenticate == nil {
		_ = writeError(ctx, http.StatusServiceUnavailable, "engineering service unavailable")
		return contract.Actor{}, "", false
	}
	userID, err := h.authenticate(ctx.Request())
	if err != nil || strings.TrimSpace(userID) == "" {
		_ = writeError(ctx, http.StatusUnauthorized, "user not authenticated")
		return contract.Actor{}, "", false
	}
	workspaceID := strings.TrimSpace(ctx.Vars().Get("workspace"))
	if workspaceID == "" {
		_ = writeError(ctx, http.StatusBadRequest, "workspace is required")
		return contract.Actor{}, "", false
	}
	return contract.Actor{UserID: strings.TrimSpace(userID)}, workspaceID, true
}

func writeLifecycleError(ctx kratoshttp.Context, err error) error {
	switch {
	case errors.Is(err, contract.ErrInvalidArgument):
		return writeError(ctx, http.StatusBadRequest, "invalid engineering request")
	case errors.Is(err, contract.ErrForbidden):
		return writeError(ctx, http.StatusForbidden, "engineering access denied")
	case errors.Is(err, contract.ErrNotFound):
		return writeError(ctx, http.StatusNotFound, "engineering record not found")
	case errors.Is(err, contract.ErrConflict):
		return writeError(ctx, http.StatusConflict, "engineering record conflict")
	case errors.Is(err, contract.ErrUnavailable):
		return writeError(ctx, http.StatusServiceUnavailable, "engineering service unavailable")
	default:
		return writeError(ctx, http.StatusInternalServerError, "engineering request failed")
	}
}
