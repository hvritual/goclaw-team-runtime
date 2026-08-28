package http

import (
	"errors"
	"net/http"
	"strings"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

type WorkEngineeringLinkHandler struct {
	service      contract.WorkEngineeringLinkService
	identity     contract.WorkspaceHTTPIdentityResolver
	authenticate func(*http.Request) (string, error)
	mutation     func(*http.Request) error
}

func NewWorkEngineeringLinkHandler(service contract.WorkEngineeringLinkService, identity contract.WorkspaceHTTPIdentityResolver, authenticate func(*http.Request) (string, error), mutation func(*http.Request) error) *WorkEngineeringLinkHandler {
	return &WorkEngineeringLinkHandler{service: service, identity: identity, authenticate: authenticate, mutation: mutation}
}

func (h *WorkEngineeringLinkHandler) Register(server *kratoshttp.Server) {
	if h == nil || server == nil {
		return
	}
	router := server.Route("/")
	router.GET("/api/projects/{id}/engineering-links", func(ctx kratoshttp.Context) error { return h.list(ctx, contract.EngineeringWorkProject) })
	router.POST("/api/projects/{id}/engineering-links", func(ctx kratoshttp.Context) error { return h.link(ctx, contract.EngineeringWorkProject) })
	router.DELETE("/api/projects/{id}/engineering-links/{entity_id}", func(ctx kratoshttp.Context) error { return h.unlink(ctx, contract.EngineeringWorkProject) })
	router.GET("/api/requirements/{id}/engineering-links", func(ctx kratoshttp.Context) error { return h.list(ctx, contract.EngineeringWorkRequirement) })
	router.POST("/api/requirements/{id}/engineering-links", func(ctx kratoshttp.Context) error { return h.link(ctx, contract.EngineeringWorkRequirement) })
	router.DELETE("/api/requirements/{id}/engineering-links/{entity_id}", func(ctx kratoshttp.Context) error { return h.unlink(ctx, contract.EngineeringWorkRequirement) })
	router.GET("/api/tasks/{id}/engineering-links", func(ctx kratoshttp.Context) error { return h.list(ctx, contract.EngineeringWorkTask) })
	router.POST("/api/tasks/{id}/engineering-links", func(ctx kratoshttp.Context) error { return h.link(ctx, contract.EngineeringWorkTask) })
	router.DELETE("/api/tasks/{id}/engineering-links/{entity_id}", func(ctx kratoshttp.Context) error { return h.unlink(ctx, contract.EngineeringWorkTask) })
}

func (h *WorkEngineeringLinkHandler) list(ctx kratoshttp.Context, kind contract.EngineeringWorkKind) error {
	identity, ok := h.resolveIdentity(ctx)
	if !ok {
		return nil
	}
	links, err := h.service.ListEngineeringLinks(workspaceActorContext(ctx, identity), identity.WorkspaceID, kind, strings.TrimSpace(ctx.Vars().Get("id")))
	if err != nil {
		return h.writeFailure(ctx, err)
	}
	return ctx.JSON(http.StatusOK, map[string]any{"links": links})
}

func (h *WorkEngineeringLinkHandler) link(ctx kratoshttp.Context, kind contract.EngineeringWorkKind) error {
	identity, ok := h.resolveIdentity(ctx)
	if !ok {
		return nil
	}
	if !h.authorizeMutation(ctx) {
		return nil
	}
	var request struct {
		EntityID string `json:"entity_id"`
	}
	if err := decodeJSON(ctx.Request().Body, &request); err != nil || strings.TrimSpace(request.EntityID) == "" {
		return writeError(ctx, http.StatusBadRequest, contract.ErrEngineeringWorkLinkInvalid.Error())
	}
	value, err := h.service.LinkEngineeringEntity(workspaceActorContext(ctx, identity), identity.WorkspaceID, kind, strings.TrimSpace(ctx.Vars().Get("id")), strings.TrimSpace(request.EntityID))
	if err != nil {
		return h.writeFailure(ctx, err)
	}
	return ctx.JSON(http.StatusCreated, value)
}

func (h *WorkEngineeringLinkHandler) unlink(ctx kratoshttp.Context, kind contract.EngineeringWorkKind) error {
	identity, ok := h.resolveIdentity(ctx)
	if !ok {
		return nil
	}
	if !h.authorizeMutation(ctx) {
		return nil
	}
	if err := h.service.UnlinkEngineeringEntity(workspaceActorContext(ctx, identity), identity.WorkspaceID, kind, strings.TrimSpace(ctx.Vars().Get("id")), strings.TrimSpace(ctx.Vars().Get("entity_id"))); err != nil {
		return h.writeFailure(ctx, err)
	}
	ctx.Response().WriteHeader(http.StatusNoContent)
	return nil
}

func (h *WorkEngineeringLinkHandler) resolveIdentity(ctx kratoshttp.Context) (contract.WorkspaceHTTPIdentity, bool) {
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
		_ = writeError(ctx, http.StatusNotFound, "workspace not found")
		return contract.WorkspaceHTTPIdentity{}, false
	}
	if err != nil {
		_ = writeError(ctx, http.StatusUnauthorized, "user not authenticated")
		return contract.WorkspaceHTTPIdentity{}, false
	}
	return identity, true
}

func (h *WorkEngineeringLinkHandler) authorizeMutation(ctx kratoshttp.Context) bool {
	if h.mutation != nil && h.mutation(ctx.Request()) == nil {
		return true
	}
	_ = writeError(ctx, http.StatusForbidden, "invalid CSRF token")
	return false
}

func (h *WorkEngineeringLinkHandler) writeFailure(ctx kratoshttp.Context, err error) error {
	switch {
	case errors.Is(err, contract.ErrEngineeringWorkLinkInvalid):
		return writeError(ctx, http.StatusBadRequest, contract.ErrEngineeringWorkLinkInvalid.Error())
	case errors.Is(err, contract.ErrEngineeringWorkNotFound):
		return writeError(ctx, http.StatusNotFound, contract.ErrEngineeringWorkNotFound.Error())
	case errors.Is(err, contract.ErrEngineeringEntityReferenceMissing):
		return writeError(ctx, http.StatusNotFound, contract.ErrEngineeringEntityReferenceMissing.Error())
	case errors.Is(err, contract.ErrEngineeringWorkLinkNotFound):
		return writeError(ctx, http.StatusNotFound, contract.ErrEngineeringWorkLinkNotFound.Error())
	case errors.Is(err, contract.ErrEngineeringEntityArchived):
		return writeError(ctx, http.StatusConflict, contract.ErrEngineeringEntityArchived.Error())
	case errors.Is(err, contract.ErrWorkspacePermissionDenied), errors.Is(err, contract.ErrWorkspaceActorRequired):
		return writeError(ctx, http.StatusForbidden, contract.ErrWorkspacePermissionDenied.Error())
	case errors.Is(err, contract.ErrActorOutsideWorkspace), errors.Is(err, contract.ErrWorkspaceNotFound):
		return writeError(ctx, http.StatusNotFound, "workspace not found")
	case errors.Is(err, contract.ErrEngineeringWorkLinkUnavailable):
		return writeError(ctx, http.StatusServiceUnavailable, contract.ErrEngineeringWorkLinkUnavailable.Error())
	default:
		return writeError(ctx, http.StatusInternalServerError, "failed to manage engineering work link")
	}
}
