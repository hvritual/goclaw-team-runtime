// Package http preserves the installed multipart Space upload boundary.
package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	stdhttp "net/http"
	"path"
	"strconv"
	"strings"

	"github.com/multica-ai/multica/server/internal/modules/space/contract"
)

const maxUploadSize = 100 << 20

var (
	ErrIssueNotAccessible = errors.New("issue not accessible in workspace")
	ErrInvalidIssueID     = errors.New("invalid issue id")
	ErrAssetRelation      = errors.New("failed to link asset")
	ErrWorkspaceNotFound  = errors.New("workspace not found")
)

var extensionContentTypes = map[string]string{
	".svg": "image/svg+xml", ".css": "text/css", ".js": "application/javascript",
	".mjs": "application/javascript", ".json": "application/json", ".wasm": "application/wasm",
}

type UploadRequest struct {
	UserID      string
	WorkspaceID string
	IssueID     *string
	Filename    string
	ContentType string
	Content     []byte
}

type UploadResult struct {
	Asset    *contract.Asset_Asset
	IssueID  *string
	ID       string
	URL      string
	Filename string
}

type Uploader interface {
	Available() bool
	Upload(context.Context, UploadRequest) (UploadResult, error)
}

type ActorResolver func(*stdhttp.Request) string
type WorkspaceResolver func(*stdhttp.Request) (string, error)

type URLPolicy struct {
	PublicURL string
}

type UploadHandler struct {
	uploader         Uploader
	resolveActor     ActorResolver
	resolveWorkspace WorkspaceResolver
	urlPolicy        URLPolicy
}

func NewUploadHandler(
	uploader Uploader,
	actorResolver ActorResolver,
	workspaceResolver WorkspaceResolver,
	policy URLPolicy,
) *UploadHandler {
	return &UploadHandler{
		uploader: uploader, resolveActor: actorResolver,
		resolveWorkspace: workspaceResolver, urlPolicy: policy,
	}
}

func (h *UploadHandler) ServeHTTP(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
	if h.uploader == nil || !h.uploader.Available() {
		writeError(writer, stdhttp.StatusServiceUnavailable, "file upload not configured")
		return
	}
	userID := ""
	if h.resolveActor != nil {
		userID = h.resolveActor(request)
	}
	if userID == "" {
		writeError(writer, stdhttp.StatusUnauthorized, "user not authenticated")
		return
	}
	decoded, failure := decodeUploadRequest(writer, request)
	if failure != nil {
		writeError(writer, failure.status, failure.message)
		return
	}
	workspaceID := ""
	if h.resolveWorkspace != nil {
		var err error
		workspaceID, err = h.resolveWorkspace(request)
		if err != nil {
			writeError(writer, stdhttp.StatusNotFound, "workspace not found")
			return
		}
	}
	result, err := h.uploader.Upload(request.Context(), UploadRequest{
		UserID: userID, WorkspaceID: workspaceID, IssueID: decoded.issueID,
		Filename: decoded.filename, ContentType: decoded.contentType, Content: decoded.content,
	})
	if err != nil {
		writeUploadError(writer, err)
		return
	}
	if result.Asset != nil {
		writeJSON(writer, stdhttp.StatusOK, h.urlPolicy.ResponseFor(*result.Asset, result.IssueID))
		return
	}
	writeJSON(writer, stdhttp.StatusOK, directUploadResponse(result.ID, result.URL, result.Filename))
}

type decodedUpload struct {
	filename    string
	contentType string
	content     []byte
	issueID     *string
	issueSeen   bool
}

type requestFailure struct {
	status  int
	message string
}

func decodeUploadRequest(writer stdhttp.ResponseWriter, request *stdhttp.Request) (decodedUpload, *requestFailure) {
	request.Body = stdhttp.MaxBytesReader(writer, request.Body, maxUploadSize)
	reader, err := request.MultipartReader()
	if err != nil {
		return decodedUpload{}, invalidMultipartFailure()
	}
	var decoded decodedUpload
	for {
		part, nextErr := reader.NextPart()
		if errors.Is(nextErr, io.EOF) {
			break
		}
		if nextErr != nil {
			return decodedUpload{}, invalidMultipartFailure()
		}
		failure := readUploadPart(part.FormName(), part.FileName(), part, &decoded)
		_ = part.Close()
		if failure != nil {
			return decodedUpload{}, failure
		}
	}
	if decoded.filename == "" {
		return decodedUpload{}, &requestFailure{
			status:  stdhttp.StatusBadRequest,
			message: fmt.Sprintf("missing file field: %v", stdhttp.ErrMissingFile),
		}
	}
	return decoded, nil
}

func readUploadPart(formName, filename string, part io.Reader, decoded *decodedUpload) *requestFailure {
	switch {
	case formName == "file" && filename != "" && decoded.filename == "":
		content, err := io.ReadAll(part)
		if err != nil {
			return &requestFailure{status: stdhttp.StatusBadRequest, message: "failed to read file"}
		}
		decoded.filename = filename
		decoded.content = content
		decoded.contentType = stdhttp.DetectContentType(contentPrefix(content))
		if override, ok := extensionContentTypes[strings.ToLower(path.Ext(filename))]; ok {
			decoded.contentType = override
		}
	case formName == "issue_id" && !decoded.issueSeen:
		value, err := io.ReadAll(part)
		if err != nil {
			return invalidMultipartFailure()
		}
		decoded.issueSeen = true
		if string(value) != "" {
			issueID := string(value)
			decoded.issueID = &issueID
		}
	}
	return nil
}

func contentPrefix(content []byte) []byte {
	if len(content) > 512 {
		return content[:512]
	}
	return content
}

func invalidMultipartFailure() *requestFailure {
	return &requestFailure{status: stdhttp.StatusBadRequest, message: "file too large or invalid multipart form"}
}

func writeUploadError(writer stdhttp.ResponseWriter, err error) {
	switch {
	case errors.Is(err, contract.ErrAssetStorageUnavailable):
		writeError(writer, stdhttp.StatusServiceUnavailable, "file upload not configured")
	case errors.Is(err, contract.ErrAssetActorRequired):
		writeError(writer, stdhttp.StatusUnauthorized, "user not authenticated")
	case errors.Is(err, contract.ErrAssetWorkspaceForbidden):
		writeError(writer, stdhttp.StatusForbidden, "not a member of this workspace")
	case errors.Is(err, ErrIssueNotAccessible):
		writeError(writer, stdhttp.StatusForbidden, "invalid issue_id")
	case errors.Is(err, ErrInvalidIssueID), errors.Is(err, contract.ErrAssetInvalid):
		writeError(writer, stdhttp.StatusBadRequest, "invalid issue_id")
	case errors.Is(err, ErrAssetRelation):
		writeError(writer, stdhttp.StatusInternalServerError, "failed to create attachment record")
	default:
		writeError(writer, stdhttp.StatusInternalServerError, "upload failed")
	}
}

type AttachmentResponse struct {
	ID           string  `json:"id"`
	WorkspaceID  string  `json:"workspace_id"`
	IssueID      *string `json:"issue_id"`
	CommentID    *string `json:"comment_id"`
	UploaderType string  `json:"uploader_type"`
	UploaderID   string  `json:"uploader_id"`
	Filename     string  `json:"filename"`
	URL          string  `json:"url"`
	DownloadURL  string  `json:"download_url"`
	MarkdownURL  string  `json:"markdown_url"`
	ContentType  string  `json:"content_type"`
	SizeBytes    int64   `json:"size_bytes"`
	CreatedAt    string  `json:"created_at"`
}

func (policy URLPolicy) ResponseFor(value contract.Asset_Asset, issueID *string) AttachmentResponse {
	downloadPath := "/api/attachments/" + value.Id + "/download"
	markdownURL := downloadPath
	if publicURL := strings.TrimRight(policy.PublicURL, "/"); publicURL != "" {
		markdownURL = publicURL + downloadPath
	}
	sizeBytes, _ := strconv.ParseInt(value.SizeBytes, 10, 64)
	return AttachmentResponse{
		ID: value.Id, WorkspaceID: value.WorkspaceId, IssueID: cloneString(issueID),
		UploaderType: value.UploaderType, UploaderID: value.UploaderId,
		Filename: value.Filename, URL: downloadPath, DownloadURL: downloadPath,
		MarkdownURL: markdownURL, ContentType: value.MediaType,
		SizeBytes: sizeBytes, CreatedAt: value.CreatedAt,
	}
}

func directUploadResponse(id, rawURL, filename string) map[string]any {
	return map[string]any{
		"id": id, "url": rawURL, "download_url": rawURL,
		"markdown_url": rawURL, "filename": filename,
	}
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func writeJSON(writer stdhttp.ResponseWriter, status int, value any) {
	body, err := json.Marshal(value)
	if err != nil {
		writeError(writer, stdhttp.StatusInternalServerError, "failed to encode response")
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_, _ = writer.Write(append(body, '\n'))
}

func writeError(writer stdhttp.ResponseWriter, status int, message string) {
	writeJSON(writer, status, map[string]string{"error": message})
}
