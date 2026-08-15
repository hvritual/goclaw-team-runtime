package http

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/space/contract"
	"github.com/hvritual/workspace/internal/modules/space/internal/application"
)

const (
	maxMultipartOverhead = 1 << 20
	maxPreviewBytes      = 2 << 20
)

type AttachmentHandler struct {
	service     contract.AttachmentService
	identity    contract.HTTPIdentityResolver
	user        contract.HTTPUserResolver
	mutation    contract.HTTPMutationAuthorizer
	memberships contract.WorkspaceMembershipReader
}

func NewAttachmentHandler(service contract.AttachmentService, identity contract.HTTPIdentityResolver, user contract.HTTPUserResolver, mutation contract.HTTPMutationAuthorizer, memberships contract.WorkspaceMembershipReader) *AttachmentHandler {
	return &AttachmentHandler{service: service, identity: identity, user: user, mutation: mutation, memberships: memberships}
}

func (h *AttachmentHandler) Register(server *kratoshttp.Server) {
	router := server.Route("/")
	router.POST("/api/upload-file", h.upload)
	router.GET("/api/issues/{id}/attachments", h.listIssue)
	router.GET("/api/attachments/{id}", h.get)
	router.GET("/api/attachments/{id}/content", h.preview)
	router.GET("/api/attachments/{id}/download", h.download)
	router.DELETE("/api/attachments/{id}", h.delete)
}

func (h *AttachmentHandler) upload(ctx kratoshttp.Context) error {
	if _, ok := h.authenticate(ctx); !ok {
		return nil
	}
	if !hasWorkspace(ctx.Request()) {
		return writeError(ctx, http.StatusBadRequest, "workspace is required")
	}
	identity, ok := h.resolveIdentity(ctx)
	if !ok {
		return nil
	}
	if h.mutation == nil || h.mutation(ctx.Request()) != nil {
		return writeError(ctx, http.StatusForbidden, "invalid CSRF token")
	}
	decoded, err := decodeMultipart(ctx)
	if err != nil {
		return writeAttachmentError(ctx, err)
	}
	requestContext := contract.WithAttachmentActor(ctx.Request().Context(), identity.ActorType, identity.ActorID)
	value, err := h.service.Upload(requestContext, contract.UploadAttachmentRequest{
		WorkspaceID: identity.WorkspaceID, UploaderType: identity.ActorType, UploaderID: identity.ActorID,
		Filename: decoded.filename, ContentType: decoded.contentType, Content: decoded.content,
		IssueID: decoded.issueID, CommentID: decoded.commentID,
	})
	if err != nil {
		return writeAttachmentError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, value)
}

func (h *AttachmentHandler) listIssue(ctx kratoshttp.Context) error {
	if _, ok := h.authenticate(ctx); !ok {
		return nil
	}
	if !hasWorkspace(ctx.Request()) {
		return writeError(ctx, http.StatusBadRequest, "workspace is required")
	}
	identity, ok := h.resolveIdentity(ctx)
	if !ok {
		return nil
	}
	values, err := h.service.ListIssue(ctx.Request().Context(), identity.WorkspaceID, ctx.Vars().Get("id"))
	if err != nil {
		return writeAttachmentError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, values)
}

func (h *AttachmentHandler) get(ctx kratoshttp.Context) error {
	_, value, ok := h.authorizeStored(ctx)
	if !ok {
		return nil
	}
	return ctx.JSON(http.StatusOK, value)
}

func (h *AttachmentHandler) preview(ctx kratoshttp.Context) error {
	_, value, ok := h.authorizeStored(ctx)
	if !ok {
		return nil
	}
	if !textPreviewable(value.ContentType, value.Filename) {
		return writeError(ctx, http.StatusUnsupportedMediaType, "preview not supported for this file type")
	}
	if value.SizeBytes > maxPreviewBytes {
		return writeError(ctx, http.StatusRequestEntityTooLarge, "file too large for inline preview")
	}
	_, reader, err := h.service.Open(ctx.Request().Context(), value.ID)
	if err != nil {
		return writeAttachmentError(ctx, err)
	}
	defer reader.Close()
	body, err := io.ReadAll(io.LimitReader(reader, maxPreviewBytes+1))
	if err != nil {
		return writeError(ctx, http.StatusBadGateway, "failed to read attachment body")
	}
	if len(body) > maxPreviewBytes {
		return writeError(ctx, http.StatusRequestEntityTooLarge, "file too large for inline preview")
	}
	response := ctx.Response()
	response.Header().Set("Content-Type", "text/plain; charset=utf-8")
	response.Header().Set("X-Original-Content-Type", value.ContentType)
	setSecurityHeaders(response.Header())
	response.Header().Set("Content-Length", strconv.Itoa(len(body)))
	response.WriteHeader(http.StatusOK)
	_, _ = response.Write(body)
	return nil
}

func (h *AttachmentHandler) download(ctx kratoshttp.Context) error {
	_, value, ok := h.authorizeStored(ctx)
	if !ok {
		return nil
	}
	_, reader, err := h.service.Open(ctx.Request().Context(), value.ID)
	if err != nil {
		return writeAttachmentError(ctx, err)
	}
	defer reader.Close()
	response := ctx.Response()
	response.Header().Set("Content-Type", value.ContentType)
	response.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, strings.ReplaceAll(value.Filename, `"`, "")))
	response.Header().Set("Content-Length", strconv.FormatInt(value.SizeBytes, 10))
	setSecurityHeaders(response.Header())
	response.WriteHeader(http.StatusOK)
	_, _ = io.Copy(response, reader)
	return nil
}

func (h *AttachmentHandler) delete(ctx kratoshttp.Context) error {
	userID, value, ok := h.authorizeStored(ctx)
	if !ok {
		return nil
	}
	if h.mutation == nil || h.mutation(ctx.Request()) != nil {
		return writeError(ctx, http.StatusForbidden, "invalid CSRF token")
	}
	role, found, err := h.membership(ctx.Request().Context(), userID, value.WorkspaceID)
	if err != nil || !found {
		return writeError(ctx, http.StatusNotFound, "attachment not found")
	}
	if value.UploaderType != "member" || (value.UploaderID != userID && role != "owner" && role != "admin") {
		return writeError(ctx, http.StatusForbidden, "insufficient permissions")
	}
	requestContext := contract.WithAttachmentActor(ctx.Request().Context(), "member", userID)
	if err := h.service.Delete(requestContext, value.ID); err != nil {
		return writeAttachmentError(ctx, err)
	}
	ctx.Response().WriteHeader(http.StatusNoContent)
	return nil
}

func (h *AttachmentHandler) authenticate(ctx kratoshttp.Context) (string, bool) {
	if h.user == nil {
		_ = writeError(ctx, http.StatusUnauthorized, "user not authenticated")
		return "", false
	}
	userID, err := h.user(ctx.Request())
	if err != nil || strings.TrimSpace(userID) == "" {
		_ = writeError(ctx, http.StatusUnauthorized, "user not authenticated")
		return "", false
	}
	return userID, true
}

func (h *AttachmentHandler) resolveIdentity(ctx kratoshttp.Context) (contract.HTTPIdentity, bool) {
	if h.identity == nil {
		_ = writeError(ctx, http.StatusNotFound, "attachment not found")
		return contract.HTTPIdentity{}, false
	}
	identity, err := h.identity(ctx.Request())
	if err != nil {
		_ = writeError(ctx, http.StatusNotFound, "attachment not found")
		return contract.HTTPIdentity{}, false
	}
	return identity, true
}

func (h *AttachmentHandler) authorizeStored(ctx kratoshttp.Context) (string, contract.Attachment, bool) {
	userID, ok := h.authenticate(ctx)
	if !ok {
		return "", contract.Attachment{}, false
	}
	if !hasWorkspace(ctx.Request()) {
		_ = writeError(ctx, http.StatusBadRequest, "workspace is required")
		return "", contract.Attachment{}, false
	}
	identity, ok := h.resolveIdentity(ctx)
	if !ok {
		return "", contract.Attachment{}, false
	}
	value, err := h.service.Get(ctx.Request().Context(), ctx.Vars().Get("id"))
	if err != nil {
		_ = writeAttachmentError(ctx, err)
		return "", contract.Attachment{}, false
	}
	if identity.WorkspaceID != value.WorkspaceID {
		_ = writeError(ctx, http.StatusNotFound, "attachment not found")
		return "", contract.Attachment{}, false
	}
	_, found, err := h.membership(ctx.Request().Context(), userID, identity.WorkspaceID)
	if err != nil || !found {
		_ = writeError(ctx, http.StatusNotFound, "attachment not found")
		return "", contract.Attachment{}, false
	}
	return userID, value, true
}

func (h *AttachmentHandler) membership(ctx context.Context, userID, workspaceID string) (string, bool, error) {
	if h.memberships == nil {
		return "", false, nil
	}
	return h.memberships(ctx, userID, workspaceID)
}

type decodedMultipart struct {
	filename, contentType string
	content               []byte
	issueID, commentID    *string
}

func decodeMultipart(ctx kratoshttp.Context) (decodedMultipart, error) {
	return decodeMultipartRequest(ctx.Response(), ctx.Request(), application.MaxAttachmentSize)
}

func decodeMultipartRequest(response http.ResponseWriter, request *http.Request, maxFileBytes int64) (decodedMultipart, error) {
	request.Body = http.MaxBytesReader(response, request.Body, maxFileBytes+maxMultipartOverhead)
	reader, err := request.MultipartReader()
	if err != nil {
		return decodedMultipart{}, contract.ErrAttachmentInvalid
	}
	var result decodedMultipart
	seen := map[string]bool{}
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			var maximum *http.MaxBytesError
			if errors.As(err, &maximum) {
				return decodedMultipart{}, contract.ErrAttachmentTooLarge
			}
			return decodedMultipart{}, contract.ErrAttachmentInvalid
		}
		name := part.FormName()
		if seen[name] || (name != "file" && name != "issue_id" && name != "comment_id") {
			part.Close()
			return decodedMultipart{}, contract.ErrAttachmentInvalid
		}
		seen[name] = true
		if name == "file" {
			filename := part.FileName()
			if filename == "" {
				part.Close()
				return decodedMultipart{}, contract.ErrAttachmentInvalid
			}
			content, err := io.ReadAll(io.LimitReader(part, maxFileBytes+1))
			part.Close()
			if err != nil {
				return decodedMultipart{}, contract.ErrAttachmentInvalid
			}
			if int64(len(content)) > maxFileBytes {
				return decodedMultipart{}, contract.ErrAttachmentTooLarge
			}
			result.filename = filename
			result.content = content
			result.contentType = detectContentType(content, result.filename)
			if !supportedContentType(result.contentType) {
				return decodedMultipart{}, contract.ErrAttachmentUnsupported
			}
			continue
		}
		value, err := io.ReadAll(io.LimitReader(part, 4097))
		part.Close()
		if err != nil || len(value) > 4096 {
			return decodedMultipart{}, contract.ErrAttachmentInvalid
		}
		trimmed := strings.TrimSpace(string(value))
		if trimmed != "" {
			copy := trimmed
			if name == "issue_id" {
				result.issueID = &copy
			} else {
				result.commentID = &copy
			}
		}
	}
	if result.filename == "" || len(result.content) == 0 {
		return decodedMultipart{}, contract.ErrAttachmentInvalid
	}
	return result, nil
}

func detectContentType(content []byte, filename string) string {
	value := http.DetectContentType(content)
	extension := strings.ToLower(path.Ext(filename))
	switch extension {
	case ".md", ".markdown", ".txt", ".log", ".csv", ".tsv":
		return "text/plain; charset=utf-8"
	case ".json":
		return "application/json"
	case ".html", ".htm":
		return "text/html; charset=utf-8"
	case ".svg":
		return "image/svg+xml"
	}
	return value
}

func supportedContentType(value string) bool {
	media := strings.ToLower(strings.TrimSpace(strings.Split(value, ";")[0]))
	if strings.HasPrefix(media, "text/") || strings.HasPrefix(media, "image/") || strings.HasPrefix(media, "audio/") || strings.HasPrefix(media, "video/") {
		return true
	}
	switch media {
	case "application/json", "application/pdf", "application/zip", "application/octet-stream", "application/xml", "application/x-yaml", "application/yaml":
		return true
	default:
		return false
	}
}

func textPreviewable(contentType, filename string) bool {
	media := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	if strings.HasPrefix(media, "text/") {
		return true
	}
	switch media {
	case "application/json", "application/xml", "application/x-yaml", "application/yaml":
		return true
	}
	extension := strings.ToLower(path.Ext(filename))
	switch extension {
	case ".md", ".markdown", ".txt", ".log", ".csv", ".tsv", ".html", ".htm", ".json", ".xml", ".yml", ".yaml", ".toml", ".ini", ".conf", ".sh", ".py", ".rb", ".go", ".rs", ".ts", ".tsx", ".js", ".jsx", ".css", ".sql":
		return true
	default:
		return false
	}
}

func hasWorkspace(request *http.Request) bool {
	return strings.TrimSpace(request.Header.Get("X-Workspace-ID")) != "" || strings.TrimSpace(request.Header.Get("X-Workspace-Slug")) != ""
}

func setSecurityHeaders(header http.Header) {
	header.Set("Cache-Control", "no-store")
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'self'; object-src 'none'; base-uri 'none'; form-action 'none'")
}

func writeAttachmentError(ctx kratoshttp.Context, err error) error {
	switch {
	case errors.Is(err, contract.ErrAttachmentNotFound), errors.Is(err, contract.ErrAttachmentTargetNotFound):
		return writeError(ctx, http.StatusNotFound, "attachment not found")
	case errors.Is(err, contract.ErrAttachmentTooLarge):
		return writeError(ctx, http.StatusRequestEntityTooLarge, "file too large")
	case errors.Is(err, contract.ErrAttachmentUnsupported):
		return writeError(ctx, http.StatusUnsupportedMediaType, "unsupported file type")
	case errors.Is(err, contract.ErrAttachmentInvalid):
		return writeError(ctx, http.StatusBadRequest, "invalid multipart form")
	case errors.Is(err, contract.ErrAttachmentForbidden):
		return writeError(ctx, http.StatusForbidden, "insufficient permissions")
	default:
		return writeError(ctx, http.StatusInternalServerError, "attachment operation failed")
	}
}

func writeError(ctx kratoshttp.Context, status int, message string) error {
	return ctx.JSON(status, map[string]string{"error": message})
}
