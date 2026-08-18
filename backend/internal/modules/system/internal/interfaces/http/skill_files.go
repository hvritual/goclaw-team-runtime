package http

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/system/contract"
)

type SkillFileHandler struct {
	service  contract.SkillFileService
	identity contract.SkillIdentityResolver
	mutation contract.SkillMutationAuthorizer
}

func NewSkillFileHandler(service contract.SkillFileService, identity contract.SkillIdentityResolver, mutation contract.SkillMutationAuthorizer) *SkillFileHandler {
	return &SkillFileHandler{service: service, identity: identity, mutation: mutation}
}

func (h *SkillFileHandler) Register(server *kratoshttp.Server) {
	router := server.Route("/")
	router.GET("/api/skills/{id}/files", h.list)
	router.POST("/api/skills/{id}/files", h.add)
	router.GET("/api/skills/{id}/files/{path:.*}", h.get)
	router.PUT("/api/skills/{id}/files/{path:.*}", h.replace)
	router.DELETE("/api/skills/{id}/files/{path:.*}", h.delete)
}

func (h *SkillFileHandler) list(ctx kratoshttp.Context) error {
	identity, ok := h.resolve(ctx, false)
	if !ok {
		return nil
	}
	values, err := h.service.List(ctx.Request().Context(), identity, ctx.Vars().Get("id"), ctx.Request().URL.Query().Get("version_id"))
	if err != nil {
		return writeSkillFileError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, values)
}

func (h *SkillFileHandler) get(ctx kratoshttp.Context) error {
	identity, ok := h.resolve(ctx, false)
	if !ok {
		return nil
	}
	pathValue, err := url.PathUnescape(ctx.Vars().Get("path"))
	if err != nil {
		return writeSkillError(ctx, http.StatusBadRequest, "invalid Skill file path")
	}
	manifest, body, err := h.service.Read(ctx.Request().Context(), identity, ctx.Vars().Get("id"), ctx.Request().URL.Query().Get("version_id"), pathValue)
	if err != nil {
		return writeSkillFileError(ctx, err)
	}
	if ctx.Request().URL.Query().Get("download") == "true" {
		response := ctx.Response()
		response.Header().Set("Content-Type", manifest.MediaType)
		response.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, strings.ReplaceAll(pathBase(manifest.Path), `"`, "")))
		response.Header().Set("Content-Length", strconv.FormatInt(manifest.SizeBytes, 10))
		response.Header().Set("ETag", `"sha256-`+manifest.Checksum+`"`)
		response.Header().Set("X-Content-Type-Options", "nosniff")
		response.WriteHeader(http.StatusOK)
		_, err = response.Write(body)
		return err
	}
	if !utf8.Valid(body) {
		return writeSkillError(ctx, http.StatusUnsupportedMediaType, "Skill file is download-only")
	}
	return ctx.JSON(http.StatusOK, contract.SkillFileContent{SkillFileManifest: manifest, Content: string(body)})
}

func (h *SkillFileHandler) add(ctx kratoshttp.Context) error {
	identity, ok := h.resolve(ctx, true)
	if !ok {
		return nil
	}
	var input struct {
		Path             string `json:"path"`
		Content          string `json:"content"`
		ExpectedRevision int64  `json:"expected_revision"`
	}
	if err := decodeSkillJSON(ctx.Request(), &input); err != nil {
		return writeSkillError(ctx, http.StatusBadRequest, "invalid request body")
	}
	value, err := h.service.Mutate(ctx.Request().Context(), identity, ctx.Vars().Get("id"), "add", input.Path, []byte(input.Content), input.ExpectedRevision)
	if err != nil {
		return writeSkillFileError(ctx, err)
	}
	return ctx.JSON(http.StatusCreated, value)
}

func (h *SkillFileHandler) replace(ctx kratoshttp.Context) error {
	identity, ok := h.resolve(ctx, true)
	if !ok {
		return nil
	}
	var input struct {
		Content          string `json:"content"`
		ExpectedRevision int64  `json:"expected_revision"`
	}
	if err := decodeSkillJSON(ctx.Request(), &input); err != nil {
		return writeSkillError(ctx, http.StatusBadRequest, "invalid request body")
	}
	pathValue, err := url.PathUnescape(ctx.Vars().Get("path"))
	if err != nil {
		return writeSkillError(ctx, http.StatusBadRequest, "invalid Skill file path")
	}
	value, err := h.service.Mutate(ctx.Request().Context(), identity, ctx.Vars().Get("id"), "replace", pathValue, []byte(input.Content), input.ExpectedRevision)
	if err != nil {
		return writeSkillFileError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, value)
}

func (h *SkillFileHandler) delete(ctx kratoshttp.Context) error {
	identity, ok := h.resolve(ctx, true)
	if !ok {
		return nil
	}
	var input struct {
		ExpectedRevision int64 `json:"expected_revision"`
	}
	if err := decodeSkillJSON(ctx.Request(), &input); err != nil {
		return writeSkillError(ctx, http.StatusBadRequest, "invalid request body")
	}
	pathValue, err := url.PathUnescape(ctx.Vars().Get("path"))
	if err != nil {
		return writeSkillError(ctx, http.StatusBadRequest, "invalid Skill file path")
	}
	value, err := h.service.Delete(ctx.Request().Context(), identity, ctx.Vars().Get("id"), pathValue, input.ExpectedRevision)
	if err != nil {
		return writeSkillFileError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, value)
}

func (h *SkillFileHandler) resolve(ctx kratoshttp.Context, mutation bool) (contract.SkillIdentity, bool) {
	if h.identity == nil {
		_ = writeSkillError(ctx, http.StatusUnauthorized, "authentication required")
		return contract.SkillIdentity{}, false
	}
	identity, err := h.identity(ctx.Request())
	if err != nil {
		_ = writeSkillError(ctx, http.StatusUnauthorized, "authentication required")
		return contract.SkillIdentity{}, false
	}
	if mutation && (h.mutation == nil || h.mutation(ctx.Request()) != nil) {
		_ = writeSkillError(ctx, http.StatusForbidden, "invalid CSRF token")
		return contract.SkillIdentity{}, false
	}
	return identity, true
}

func writeSkillFileError(ctx kratoshttp.Context, err error) error {
	var conflict contract.SkillRevisionConflict
	switch {
	case errors.As(err, &conflict):
		return ctx.JSON(http.StatusConflict, map[string]any{"code": "revision_conflict", "current_revision": conflict.CurrentRevision, "error": "revision conflict"})
	case errors.Is(err, contract.ErrInvalidSkill):
		return writeSkillError(ctx, http.StatusBadRequest, "invalid Skill file")
	case errors.Is(err, contract.ErrSkillAccessDenied):
		return writeSkillError(ctx, http.StatusForbidden, "insufficient permissions")
	case errors.Is(err, contract.ErrSkillAlreadyExists):
		return writeSkillError(ctx, http.StatusConflict, "Skill file conflict")
	case errors.Is(err, contract.ErrSkillNotFound):
		return writeSkillError(ctx, http.StatusNotFound, "Skill file not found")
	default:
		return writeSkillError(ctx, http.StatusInternalServerError, "Skill file operation failed")
	}
}

func pathBase(value string) string {
	if index := strings.LastIndex(value, "/"); index >= 0 {
		return value[index+1:]
	}
	return value
}
