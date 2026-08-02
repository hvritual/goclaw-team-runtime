// Package http exposes Space application use cases through the installed Chi HTTP boundary.
package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	stdhttp "net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/multica-ai/multica/server/modules/space/application"
	"github.com/multica-ai/multica/server/modules/space/domain"
)

const maxUploadSize = 100 << 20

var extensionContentTypes = map[string]string{
	".svg":  "image/svg+xml",
	".css":  "text/css",
	".js":   "application/javascript",
	".mjs":  "application/javascript",
	".json": "application/json",
	".wasm": "application/wasm",
}

// ErrIssueNotAccessible reports an invalid workspace-scoped Issue reference.
var ErrIssueNotAccessible = errors.New("issue not accessible in workspace")

// ErrInvalidIssueID reports a malformed Issue identity after authorization.
var ErrInvalidIssueID = errors.New("invalid issue id")

// UploadRequest is the transport-neutral input accepted by the HTTP boundary.
type UploadRequest struct {
	UserID      string
	WorkspaceID string
	IssueID     *string
	Filename    string
	ContentType string
	Content     []byte
}

// UploadResult is the transport-neutral result rendered by the HTTP boundary.
type UploadResult struct {
	Asset    *domain.Asset
	IssueID  *string
	ID       string
	URL      string
	Filename string
}

// Uploader is the HTTP adapter's single application-workflow contract.
type Uploader interface {
	Available() bool
	Upload(ctx context.Context, request UploadRequest) (UploadResult, error)
}

// WorkspaceResolver preserves the current request-to-workspace resolution policy.
type WorkspaceResolver func(request *stdhttp.Request) string

// URLSigner signs a storage URL for clients that cannot send authorization headers.
type URLSigner interface {
	SignedURL(rawURL string, expiry time.Time) string
}

// URLPolicy renders the three installed attachment URL variants.
type URLPolicy struct {
	PublicURL            string
	StorageURLsArePublic bool
	Signer               URLSigner
	TTL                  time.Duration
	Now                  func() time.Time
}

// UploadHandler adapts the multipart upload endpoint to the Space upload use case.
type UploadHandler struct {
	uploader         Uploader
	resolveWorkspace WorkspaceResolver
	urlPolicy        URLPolicy
}

// NewUploadHandler creates a Space multipart upload adapter.
func NewUploadHandler(
	uploader Uploader,
	resolver WorkspaceResolver,
	policy URLPolicy,
) *UploadHandler {
	return &UploadHandler{
		uploader:         uploader,
		resolveWorkspace: resolver,
		urlPolicy:        policy,
	}
}

// ServeHTTP handles POST /api/upload-file without changing its public contract.
func (h *UploadHandler) ServeHTTP(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
	if h.uploader == nil || !h.uploader.Available() {
		writeError(writer, stdhttp.StatusServiceUnavailable, "file upload not configured")
		return
	}

	userID := request.Header.Get("X-User-ID")
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
		workspaceID = h.resolveWorkspace(request)
	}
	result, err := h.uploader.Upload(request.Context(), UploadRequest{
		UserID:      userID,
		WorkspaceID: workspaceID,
		IssueID:     decoded.issueID,
		Filename:    decoded.filename,
		ContentType: decoded.contentType,
		Content:     decoded.content,
	})
	if err != nil {
		var metadataError *application.MetadataPersistenceError
		if !errors.As(err, &metadataError) {
			writeUploadError(writer, err)
			return
		}
		slog.Error("failed to create attachment record", "error", metadataError)
		result = UploadResult{
			ID:       metadataError.Result.ID,
			URL:      metadataError.Result.URL,
			Filename: metadataError.Result.Filename,
		}
	}
	if result.Asset != nil {
		writeJSON(writer, stdhttp.StatusOK, h.urlPolicy.responseFor(*result.Asset, result.IssueID, request))
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
			var maxBytesError *stdhttp.MaxBytesError
			if errors.As(err, &maxBytesError) {
				return invalidMultipartFailure()
			}
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
	return &requestFailure{
		status:  stdhttp.StatusBadRequest,
		message: "file too large or invalid multipart form",
	}
}

func writeUploadError(writer stdhttp.ResponseWriter, err error) {
	switch {
	case errors.Is(err, application.ErrStorageUnavailable):
		writeError(writer, stdhttp.StatusServiceUnavailable, "file upload not configured")
	case errors.Is(err, application.ErrNotWorkspaceMember):
		writeError(writer, stdhttp.StatusForbidden, "not a member of this workspace")
	case errors.Is(err, ErrIssueNotAccessible):
		writeError(writer, stdhttp.StatusForbidden, "invalid issue_id")
	case errors.Is(err, ErrInvalidIssueID):
		writeError(writer, stdhttp.StatusBadRequest, "invalid issue_id")
	case errors.Is(err, application.ErrGenerateID):
		slog.Error("failed to generate uuid", "error", err)
		writeError(writer, stdhttp.StatusInternalServerError, "internal error")
	default:
		slog.Error("file upload failed", "error", err)
		writeError(writer, stdhttp.StatusInternalServerError, "upload failed")
	}
}

type attachmentResponse struct {
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

func (policy URLPolicy) responseFor(asset domain.Asset, issueID *string, request *stdhttp.Request) attachmentResponse {
	downloadPath := "/api/attachments/" + asset.ID() + "/download"
	downloadURL := downloadPath
	if policy.Signer != nil && !requestHasCapability(request, "stable_attachment_urls") {
		downloadURL = policy.Signer.SignedURL(asset.URL(), policy.now().Add(policy.ttl()))
	}
	markdownURL := downloadPath
	if policy.storageURLIsPublic(asset.URL()) {
		markdownURL = asset.URL()
	} else if publicURL := strings.TrimRight(policy.PublicURL, "/"); publicURL != "" {
		markdownURL = publicURL + downloadPath
	}
	return attachmentResponse{
		ID:           asset.ID(),
		WorkspaceID:  asset.WorkspaceID(),
		IssueID:      cloneString(issueID),
		UploaderType: string(asset.UploaderType()),
		UploaderID:   asset.UploaderID(),
		Filename:     asset.Filename(),
		URL:          asset.URL(),
		DownloadURL:  downloadURL,
		MarkdownURL:  markdownURL,
		ContentType:  asset.ContentType(),
		SizeBytes:    asset.SizeBytes(),
		CreatedAt:    asset.CreatedAt().Format(time.RFC3339),
	}
}

func directUploadResponse(id, rawURL, filename string) map[string]string {
	return map[string]string{"id": id, "url": rawURL, "filename": filename}
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func (policy URLPolicy) storageURLIsPublic(rawURL string) bool {
	if !policy.StorageURLsArePublic || policy.Signer != nil {
		return false
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return false
	}
	for _, key := range []string{"Signature", "X-Amz-Signature", "Key-Pair-Id", "Expires", "X-Amz-Expires"} {
		if parsed.Query().Get(key) != "" {
			return false
		}
	}
	return true
}

func (policy URLPolicy) now() time.Time {
	if policy.Now != nil {
		return policy.Now()
	}
	return time.Now()
}

func (policy URLPolicy) ttl() time.Duration {
	if policy.TTL > 0 {
		return policy.TTL
	}
	return 30 * time.Minute
}

func requestHasCapability(request *stdhttp.Request, capability string) bool {
	for _, item := range strings.Split(request.Header.Get("X-Client-Capabilities"), ",") {
		if strings.TrimSpace(item) == capability {
			return true
		}
	}
	return false
}

func writeJSON(writer stdhttp.ResponseWriter, status int, value any) {
	body, err := json.Marshal(value)
	if err != nil {
		body = []byte(`{"error":"failed to encode response"}`)
		status = stdhttp.StatusInternalServerError
	}
	body = append(body, '\n')
	writer.Header().Set("Content-Type", "application/json")
	writer.Header().Set("Content-Length", strconv.Itoa(len(body)))
	writer.WriteHeader(status)
	_, _ = writer.Write(body)
}

func writeError(writer stdhttp.ResponseWriter, status int, message string) {
	writeJSON(writer, status, map[string]string{"error": message})
}
