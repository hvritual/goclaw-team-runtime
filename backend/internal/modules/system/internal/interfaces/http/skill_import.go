package http

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/system/contract"
)

const maxSkillArchiveRequestBytes = 10 << 20

type SkillImportHandler struct {
	service  contract.SkillImportService
	identity contract.SkillIdentityResolver
	mutation contract.SkillMutationAuthorizer
}

func NewSkillImportHandler(service contract.SkillImportService, identity contract.SkillIdentityResolver, mutation contract.SkillMutationAuthorizer) *SkillImportHandler {
	return &SkillImportHandler{service: service, identity: identity, mutation: mutation}
}

func (h *SkillImportHandler) Register(server *kratoshttp.Server) {
	router := server.Route("/")
	router.POST("/api/skills/import/preview", h.preview)
	router.POST("/api/skills/import", h.commit)
}

func (h *SkillImportHandler) preview(ctx kratoshttp.Context) error {
	identity, ok := h.resolveMutation(ctx)
	if !ok {
		return nil
	}
	if strings.HasPrefix(strings.ToLower(ctx.Request().Header.Get("Content-Type")), "multipart/form-data") {
		data, _, _, _, err := decodeSkillImportMultipart(ctx)
		if err != nil {
			return writeSkillError(ctx, http.StatusBadRequest, err.Error())
		}
		value, err := h.service.PreviewArchive(ctx.Request().Context(), identity, data)
		if err != nil {
			return writeSkillImportError(ctx, err)
		}
		return ctx.JSON(http.StatusOK, value)
	}
	var input struct {
		URL string `json:"url"`
	}
	if err := decodeSkillJSON(ctx.Request(), &input); err != nil || strings.TrimSpace(input.URL) == "" {
		return writeSkillError(ctx, http.StatusBadRequest, "invalid request body")
	}
	value, err := h.service.PreviewURL(ctx.Request().Context(), identity, input.URL)
	if err != nil {
		return writeSkillImportError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, value)
}

func (h *SkillImportHandler) commit(ctx kratoshttp.Context) error {
	identity, ok := h.resolveMutation(ctx)
	if !ok {
		return nil
	}
	idempotencyKey := strings.TrimSpace(ctx.Request().Header.Get("Idempotency-Key"))
	if idempotencyKey == "" || len(idempotencyKey) > 200 {
		return writeSkillError(ctx, http.StatusBadRequest, "Idempotency-Key is required")
	}
	if strings.HasPrefix(strings.ToLower(ctx.Request().Header.Get("Content-Type")), "multipart/form-data") {
		data, token, mode, expected, err := decodeSkillImportMultipart(ctx)
		if err != nil {
			return writeSkillError(ctx, http.StatusBadRequest, err.Error())
		}
		value, err := h.service.ImportArchive(ctx.Request().Context(), identity, data, token, mode, expected, idempotencyKey)
		if err != nil {
			return writeSkillImportError(ctx, err)
		}
		return ctx.JSON(http.StatusCreated, value)
	}
	var input struct {
		URL              string `json:"url"`
		PreviewToken     string `json:"preview_token"`
		ConflictMode     string `json:"conflict_mode"`
		ExpectedRevision int64  `json:"expected_revision"`
	}
	if err := decodeSkillJSON(ctx.Request(), &input); err != nil {
		return writeSkillError(ctx, http.StatusBadRequest, "invalid request body")
	}
	value, err := h.service.ImportURL(ctx.Request().Context(), identity, input.URL, input.PreviewToken, input.ConflictMode, input.ExpectedRevision, idempotencyKey)
	if err != nil {
		return writeSkillImportError(ctx, err)
	}
	return ctx.JSON(http.StatusCreated, value)
}

func decodeSkillImportMultipart(ctx kratoshttp.Context) ([]byte, string, string, int64, error) {
	request := ctx.Request()
	request.Body = http.MaxBytesReader(ctx.Response(), request.Body, maxSkillArchiveRequestBytes+(1<<20))
	if err := request.ParseMultipartForm(maxSkillArchiveRequestBytes + (1 << 20)); err != nil {
		return nil, "", "", 0, errors.New("invalid or oversized Skill archive")
	}
	defer func() {
		if request.MultipartForm != nil {
			_ = request.MultipartForm.RemoveAll()
		}
	}()
	file, _, err := request.FormFile("file")
	if err != nil {
		return nil, "", "", 0, errors.New("Skill archive file is required")
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxSkillArchiveRequestBytes+1))
	if err != nil || len(data) > maxSkillArchiveRequestBytes {
		return nil, "", "", 0, errors.New("Skill archive exceeds compressed size limit")
	}
	expected, _ := strconv.ParseInt(request.FormValue("expected_revision"), 10, 64)
	return data, request.FormValue("preview_token"), request.FormValue("conflict_mode"), expected, nil
}

func (h *SkillImportHandler) resolveMutation(ctx kratoshttp.Context) (contract.SkillIdentity, bool) {
	if h.identity == nil {
		_ = writeSkillError(ctx, http.StatusUnauthorized, "authentication required")
		return contract.SkillIdentity{}, false
	}
	identity, err := h.identity(ctx.Request())
	if err != nil {
		_ = writeSkillError(ctx, http.StatusUnauthorized, "authentication required")
		return contract.SkillIdentity{}, false
	}
	if h.mutation == nil || h.mutation(ctx.Request()) != nil {
		_ = writeSkillError(ctx, http.StatusForbidden, "invalid CSRF token")
		return contract.SkillIdentity{}, false
	}
	return identity, true
}

func writeSkillImportError(ctx kratoshttp.Context, err error) error {
	var conflict contract.SkillRevisionConflict
	switch {
	case errors.As(err, &conflict):
		return ctx.JSON(http.StatusConflict, map[string]any{"code": "revision_conflict", "current_revision": conflict.CurrentRevision, "error": "revision conflict"})
	case errors.Is(err, contract.ErrInvalidSkill):
		return writeSkillError(ctx, http.StatusBadRequest, "invalid Skill import")
	case errors.Is(err, contract.ErrSkillAccessDenied):
		return writeSkillError(ctx, http.StatusForbidden, "insufficient permissions")
	case errors.Is(err, contract.ErrSkillImportConflict):
		return writeSkillError(ctx, http.StatusConflict, "Skill import idempotency conflict")
	case errors.Is(err, contract.ErrSkillTransition):
		return writeSkillError(ctx, http.StatusBadRequest, "only the latest draft can be replaced")
	default:
		return writeSkillError(ctx, http.StatusInternalServerError, "Skill import failed")
	}
}
