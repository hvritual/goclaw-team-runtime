package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/engineering/contract"
)

type Handler struct {
	service      contract.Service
	authenticate func(*http.Request) (string, error)
}

func NewHandler(service contract.Service, authenticate func(*http.Request) (string, error)) *Handler {
	return &Handler{service: service, authenticate: authenticate}
}

func (h *Handler) Register(server *kratoshttp.Server) {
	if h == nil || server == nil {
		return
	}
	router := server.Route("/")
	router.POST("/api/engineering/v1/workspaces/{workspace}/entities", h.createEntity)
	router.GET("/api/engineering/v1/workspaces/{workspace}/entities", h.listEntities)
	router.GET("/api/engineering/v1/workspaces/{workspace}/entities/{id}", h.getEntity)
	router.PATCH("/api/engineering/v1/workspaces/{workspace}/entities/{id}", h.updateEntity)

	router.POST("/api/engineering/v1/workspaces/{workspace}/source-bindings", h.createSourceBinding)
	router.GET("/api/engineering/v1/workspaces/{workspace}/source-bindings", h.listSourceBindings)
	router.GET("/api/engineering/v1/workspaces/{workspace}/source-bindings/{id}", h.getSourceBinding)

	router.POST("/api/engineering/v1/workspaces/{workspace}/thread-edges", h.createThreadEdge)
	router.GET("/api/engineering/v1/workspaces/{workspace}/thread-edges", h.listThreadEdges)

	router.POST("/api/engineering/v1/workspaces/{workspace}/changes", h.createChange)
	router.GET("/api/engineering/v1/workspaces/{workspace}/changes", h.listChanges)
	router.GET("/api/engineering/v1/workspaces/{workspace}/changes/{id}", h.getChange)

	router.GET("/api/engineering/v1/workspaces/{workspace}/context-packs/{id}", h.getContextPack)
}

func (h *Handler) createEntity(ctx kratoshttp.Context) error {
	actor, workspaceID, ok := h.authorizedActor(ctx)
	if !ok {
		return nil
	}
	var request contract.CreateEntityRequest
	if err := decodeJSON(ctx.Request(), &request); err != nil {
		return writeError(ctx, http.StatusBadRequest, "invalid engineering entity request")
	}
	value, err := h.service.CreateEntity(ctx.Request().Context(), actor, workspaceID, request)
	if err != nil {
		return h.writeServiceError(ctx, err)
	}
	return ctx.JSON(http.StatusCreated, value)
}

func (h *Handler) listEntities(ctx kratoshttp.Context) error {
	actor, workspaceID, ok := h.authorizedActor(ctx)
	if !ok {
		return nil
	}
	values, err := h.service.ListEntities(ctx.Request().Context(), actor, workspaceID)
	if err != nil {
		return h.writeServiceError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, map[string]any{"entities": values})
}

func (h *Handler) getEntity(ctx kratoshttp.Context) error {
	actor, workspaceID, ok := h.authorizedActor(ctx)
	if !ok {
		return nil
	}
	value, err := h.service.GetEntity(ctx.Request().Context(), actor, workspaceID, strings.TrimSpace(ctx.Vars().Get("id")))
	if err != nil {
		return h.writeServiceError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, value)
}

func (h *Handler) updateEntity(ctx kratoshttp.Context) error {
	actor, workspaceID, ok := h.authorizedActor(ctx)
	if !ok {
		return nil
	}
	var request contract.UpdateEntityRequest
	if err := decodeJSON(ctx.Request(), &request); err != nil {
		return writeError(ctx, http.StatusBadRequest, "invalid engineering entity update")
	}
	value, err := h.service.UpdateEntity(ctx.Request().Context(), actor, workspaceID, strings.TrimSpace(ctx.Vars().Get("id")), request)
	if err != nil {
		return h.writeServiceError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, value)
}

func (h *Handler) createSourceBinding(ctx kratoshttp.Context) error {
	actor, workspaceID, ok := h.authorizedActor(ctx)
	if !ok {
		return nil
	}
	var request contract.CreateSourceBindingRequest
	if err := decodeJSON(ctx.Request(), &request); err != nil {
		return writeError(ctx, http.StatusBadRequest, "invalid source binding request")
	}
	value, err := h.service.CreateSourceBinding(ctx.Request().Context(), actor, workspaceID, request)
	if err != nil {
		return h.writeServiceError(ctx, err)
	}
	return ctx.JSON(http.StatusCreated, value)
}

func (h *Handler) getSourceBinding(ctx kratoshttp.Context) error {
	actor, workspaceID, ok := h.authorizedActor(ctx)
	if !ok {
		return nil
	}
	value, err := h.service.GetSourceBinding(ctx.Request().Context(), actor, workspaceID, strings.TrimSpace(ctx.Vars().Get("id")))
	if err != nil {
		return h.writeServiceError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, value)
}

func (h *Handler) listSourceBindings(ctx kratoshttp.Context) error {
	actor, workspaceID, ok := h.authorizedActor(ctx)
	if !ok {
		return nil
	}
	entityID := strings.TrimSpace(ctx.Request().URL.Query().Get("entity_id"))
	if entityID == "" {
		return writeError(ctx, http.StatusBadRequest, "entity_id is required")
	}
	values, err := h.service.ListSourceBindings(ctx.Request().Context(), actor, workspaceID, entityID)
	if err != nil {
		return h.writeServiceError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, map[string]any{"source_bindings": values})
}

func (h *Handler) createThreadEdge(ctx kratoshttp.Context) error {
	actor, workspaceID, ok := h.authorizedActor(ctx)
	if !ok {
		return nil
	}
	var request contract.CreateThreadEdgeRequest
	if err := decodeJSON(ctx.Request(), &request); err != nil {
		return writeError(ctx, http.StatusBadRequest, "invalid thread edge request")
	}
	value, err := h.service.CreateThreadEdge(ctx.Request().Context(), actor, workspaceID, request)
	if err != nil {
		return h.writeServiceError(ctx, err)
	}
	return ctx.JSON(http.StatusCreated, value)
}

func (h *Handler) listThreadEdges(ctx kratoshttp.Context) error {
	actor, workspaceID, ok := h.authorizedActor(ctx)
	if !ok {
		return nil
	}
	node := contract.NodeRef{
		Kind: strings.TrimSpace(ctx.Request().URL.Query().Get("node_kind")),
		ID:   strings.TrimSpace(ctx.Request().URL.Query().Get("node_id")),
	}
	if node.Kind == "" || node.ID == "" {
		return writeError(ctx, http.StatusBadRequest, "node_kind and node_id are required")
	}
	values, err := h.service.ListThreadEdges(ctx.Request().Context(), actor, workspaceID, node)
	if err != nil {
		return h.writeServiceError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, map[string]any{"thread_edges": values})
}

func (h *Handler) createChange(ctx kratoshttp.Context) error {
	actor, workspaceID, ok := h.authorizedActor(ctx)
	if !ok {
		return nil
	}
	var request contract.CreateChangeRequest
	if err := decodeJSON(ctx.Request(), &request); err != nil {
		return writeError(ctx, http.StatusBadRequest, "invalid engineering change request")
	}
	value, err := h.service.CreateChange(ctx.Request().Context(), actor, workspaceID, request)
	if err != nil {
		return h.writeServiceError(ctx, err)
	}
	return ctx.JSON(http.StatusCreated, value)
}

func (h *Handler) getChange(ctx kratoshttp.Context) error {
	actor, workspaceID, ok := h.authorizedActor(ctx)
	if !ok {
		return nil
	}
	value, err := h.service.GetChange(ctx.Request().Context(), actor, workspaceID, strings.TrimSpace(ctx.Vars().Get("id")))
	if err != nil {
		return h.writeServiceError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, value)
}

func (h *Handler) listChanges(ctx kratoshttp.Context) error {
	actor, workspaceID, ok := h.authorizedActor(ctx)
	if !ok {
		return nil
	}
	values, err := h.service.ListChanges(ctx.Request().Context(), actor, workspaceID, strings.TrimSpace(ctx.Request().URL.Query().Get("affected_entity_id")))
	if err != nil {
		return h.writeServiceError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, map[string]any{"changes": values})
}

func (h *Handler) getContextPack(ctx kratoshttp.Context) error {
	actor, workspaceID, ok := h.authorizedActor(ctx)
	if !ok {
		return nil
	}
	value, err := h.service.GetContextPack(ctx.Request().Context(), actor, workspaceID, strings.TrimSpace(ctx.Vars().Get("id")))
	if err != nil {
		return h.writeServiceError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, value)
}

func (h *Handler) authorizedActor(ctx kratoshttp.Context) (contract.Actor, string, bool) {
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

func (h *Handler) writeServiceError(ctx kratoshttp.Context, err error) error {
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

func decodeJSON(request *http.Request, target any) error {
	if request == nil || request.Body == nil {
		return io.EOF
	}
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func writeError(ctx kratoshttp.Context, status int, message string) error {
	return ctx.JSON(status, map[string]any{"error": message})
}
