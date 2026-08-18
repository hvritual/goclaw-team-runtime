package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/system/contract"
)

type SkillCatalogHandler struct {
	service  contract.SkillCatalogService
	identity contract.SkillIdentityResolver
	mutation contract.SkillMutationAuthorizer
}

func NewSkillCatalogHandler(service contract.SkillCatalogService, identity contract.SkillIdentityResolver, mutation contract.SkillMutationAuthorizer) *SkillCatalogHandler {
	return &SkillCatalogHandler{service: service, identity: identity, mutation: mutation}
}

func (h *SkillCatalogHandler) Register(server *kratoshttp.Server) {
	router := server.Route("/")
	router.GET("/api/skills", h.list)
	router.POST("/api/skills", h.create)
	router.GET("/api/skills/{id}", h.get)
	router.GET("/api/skills/{id}/history", h.history)
	router.PUT("/api/skills/{id}", h.createVersion)
	router.DELETE("/api/skills/{id}", h.archive)
	router.POST("/api/skills/{id}/restore", h.restore)
	router.POST("/api/skills/{id}/versions/{version_id}/publish", h.publish)
	router.POST("/api/skills/{id}/versions/{version_id}/deprecate", h.deprecate)
}

func (h *SkillCatalogHandler) history(ctx kratoshttp.Context) error {
	identity, ok := h.resolveIdentity(ctx)
	if !ok {
		return nil
	}
	value, err := h.service.History(ctx.Request().Context(), identity, ctx.Vars().Get("id"))
	if err != nil {
		return h.writeError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, value)
}

func (h *SkillCatalogHandler) list(ctx kratoshttp.Context) error {
	identity, ok := h.resolveIdentity(ctx)
	if !ok {
		return nil
	}
	values, err := h.service.List(ctx.Request().Context(), identity)
	if err != nil {
		return h.writeError(ctx, err)
	}
	if values == nil {
		values = []contract.SkillCatalogEntry{}
	}
	return ctx.JSON(http.StatusOK, values)
}

func (h *SkillCatalogHandler) get(ctx kratoshttp.Context) error {
	identity, ok := h.resolveIdentity(ctx)
	if !ok {
		return nil
	}
	value, err := h.service.Get(ctx.Request().Context(), identity, ctx.Vars().Get("id"), ctx.Request().URL.Query().Get("version_id"))
	if err != nil {
		return h.writeError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, value)
}

func (h *SkillCatalogHandler) create(ctx kratoshttp.Context) error {
	identity, ok := h.resolveIdentity(ctx)
	if !ok {
		return nil
	}
	if h.mutation == nil || h.mutation(ctx.Request()) != nil {
		return writeSkillError(ctx, http.StatusForbidden, "invalid CSRF token")
	}
	var input struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		Config      map[string]any  `json:"config"`
		Files       json.RawMessage `json:"files"`
		Content     *string         `json:"content"`
		URL         *string         `json:"url"`
	}
	decoder := json.NewDecoder(io.LimitReader(ctx.Request().Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		return writeSkillError(ctx, http.StatusBadRequest, "invalid request body")
	}
	if len(input.Files) != 0 || input.Content != nil || input.URL != nil {
		return writeSkillError(ctx, http.StatusServiceUnavailable, "skill files and import are unavailable")
	}
	value, err := h.service.Create(ctx.Request().Context(), identity, contract.CreateSkillCatalogRequest{Name: input.Name, Description: input.Description, Config: input.Config})
	if err != nil {
		return h.writeError(ctx, err)
	}
	return ctx.JSON(http.StatusCreated, value)
}

func (h *SkillCatalogHandler) createVersion(ctx kratoshttp.Context) error {
	identity, ok := h.resolveMutation(ctx)
	if !ok {
		return nil
	}
	var input struct {
		Name             *string         `json:"name"`
		Description      *string         `json:"description"`
		Config           *map[string]any `json:"config"`
		ExpectedRevision int64           `json:"expected_revision"`
		Files            json.RawMessage `json:"files"`
		Content          *string         `json:"content"`
	}
	if err := decodeSkillJSON(ctx.Request(), &input); err != nil {
		return writeSkillError(ctx, http.StatusBadRequest, "invalid request body")
	}
	if len(input.Files) != 0 || input.Content != nil {
		return writeSkillError(ctx, http.StatusServiceUnavailable, "skill files and import are unavailable")
	}
	request := contract.UpdateSkillCatalogRequest{Name: input.Name, Description: input.Description, ExpectedRevision: input.ExpectedRevision}
	if input.Config != nil {
		request.Config, request.ConfigPresent = *input.Config, true
	}
	value, err := h.service.CreateVersion(ctx.Request().Context(), identity, ctx.Vars().Get("id"), request)
	if err != nil {
		return h.writeError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, value)
}

func (h *SkillCatalogHandler) publish(ctx kratoshttp.Context) error {
	return h.transition(ctx, "publish")
}

func (h *SkillCatalogHandler) deprecate(ctx kratoshttp.Context) error {
	return h.transition(ctx, "deprecate")
}

func (h *SkillCatalogHandler) transition(ctx kratoshttp.Context, action string) error {
	identity, ok := h.resolveMutation(ctx)
	if !ok {
		return nil
	}
	expected, ok := decodeExpectedRevision(ctx)
	if !ok {
		return nil
	}
	value, err := h.service.TransitionVersion(ctx.Request().Context(), identity, ctx.Vars().Get("id"), ctx.Vars().Get("version_id"), action, expected)
	if err != nil {
		return h.writeError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, value)
}

func (h *SkillCatalogHandler) archive(ctx kratoshttp.Context) error {
	identity, ok := h.resolveMutation(ctx)
	if !ok {
		return nil
	}
	expected, ok := decodeExpectedRevision(ctx)
	if !ok {
		return nil
	}
	if err := h.service.Archive(ctx.Request().Context(), identity, ctx.Vars().Get("id"), expected); err != nil {
		return h.writeError(ctx, err)
	}
	ctx.Response().WriteHeader(http.StatusNoContent)
	return nil
}

func (h *SkillCatalogHandler) restore(ctx kratoshttp.Context) error {
	identity, ok := h.resolveMutation(ctx)
	if !ok {
		return nil
	}
	expected, ok := decodeExpectedRevision(ctx)
	if !ok {
		return nil
	}
	value, err := h.service.Restore(ctx.Request().Context(), identity, ctx.Vars().Get("id"), expected)
	if err != nil {
		return h.writeError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, value)
}

func (h *SkillCatalogHandler) resolveIdentity(ctx kratoshttp.Context) (contract.SkillIdentity, bool) {
	identity, err := h.identity(ctx.Request())
	if err != nil {
		_ = writeSkillError(ctx, http.StatusUnauthorized, "authentication required")
		return contract.SkillIdentity{}, false
	}
	return identity, true
}

func (h *SkillCatalogHandler) resolveMutation(ctx kratoshttp.Context) (contract.SkillIdentity, bool) {
	identity, ok := h.resolveIdentity(ctx)
	if !ok {
		return contract.SkillIdentity{}, false
	}
	if h.mutation == nil || h.mutation(ctx.Request()) != nil {
		_ = writeSkillError(ctx, http.StatusForbidden, "invalid CSRF token")
		return contract.SkillIdentity{}, false
	}
	return identity, true
}

func decodeSkillJSON(request *http.Request, output any) error {
	decoder := json.NewDecoder(io.LimitReader(request.Body, 1<<20))
	decoder.DisallowUnknownFields()
	return decoder.Decode(output)
}

func decodeExpectedRevision(ctx kratoshttp.Context) (int64, bool) {
	var input struct {
		ExpectedRevision int64 `json:"expected_revision"`
	}
	if err := decodeSkillJSON(ctx.Request(), &input); err != nil || input.ExpectedRevision <= 0 {
		_ = writeSkillError(ctx, http.StatusBadRequest, "invalid request body")
		return 0, false
	}
	return input.ExpectedRevision, true
}

func (h *SkillCatalogHandler) writeError(ctx kratoshttp.Context, err error) error {
	var conflict contract.SkillRevisionConflict
	switch {
	case errors.As(err, &conflict):
		return ctx.JSON(http.StatusConflict, map[string]any{"code": "revision_conflict", "current_revision": conflict.CurrentRevision, "error": "revision conflict"})
	case errors.Is(err, contract.ErrInvalidSkill):
		return writeSkillError(ctx, http.StatusBadRequest, "invalid skill")
	case errors.Is(err, contract.ErrSkillAccessDenied):
		return writeSkillError(ctx, http.StatusForbidden, "insufficient permissions")
	case errors.Is(err, contract.ErrSkillAlreadyExists):
		return writeSkillError(ctx, http.StatusConflict, "skill already exists")
	case errors.Is(err, contract.ErrSkillNotFound):
		return writeSkillError(ctx, http.StatusNotFound, "skill not found")
	case errors.Is(err, contract.ErrSkillTransition):
		return writeSkillError(ctx, http.StatusConflict, "invalid skill transition")
	default:
		return writeSkillError(ctx, http.StatusInternalServerError, "skill operation failed")
	}
}

func writeSkillError(ctx kratoshttp.Context, status int, message string) error {
	return ctx.JSON(status, map[string]string{"error": message})
}
