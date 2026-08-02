package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/modules/space/application"
	"github.com/multica-ai/multica/server/modules/space/domain"
	spacehttp "github.com/multica-ai/multica/server/modules/space/interfaces/http"
)

type fakeUploader struct {
	available bool
	result    spacehttp.UploadResult
	err       error
	request   spacehttp.UploadRequest
}

type fakeSigner struct{}

func (fakeSigner) SignedURL(rawURL string, _ time.Time) string { return "signed:" + rawURL }

func (f *fakeUploader) Available() bool { return f.available }

func (f *fakeUploader) Upload(_ context.Context, request spacehttp.UploadRequest) (spacehttp.UploadResult, error) {
	f.request = request
	return f.result, f.err
}

func multipartRequest(t *testing.T, filename string, body []byte, fields map[string]string) *stdhttp.Request {
	t.Helper()
	var payload bytes.Buffer
	writer := multipart.NewWriter(&payload)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := part.Write(body); err != nil {
		t.Fatalf("write file: %v", err)
	}
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("WriteField: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	req := httptest.NewRequestWithContext(context.Background(), stdhttp.MethodPost, "/api/upload-file", &payload)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-User-ID", "member-1")
	return req
}

func TestUploadHandlerMapsMultipartToWorkspaceUseCaseAndPreservesResponse(t *testing.T) {
	createdAt := time.Date(2026, time.August, 2, 10, 30, 0, 0, time.UTC)
	issueID := "01980000-0000-7000-8000-000000000002"
	asset, err := domain.NewUploadedAsset(domain.UploadedAssetParams{
		ID:           "0198-0000-7000-8000-000000000001",
		WorkspaceID:  "workspace-1",
		UploaderType: domain.UploaderMember,
		UploaderID:   "member-1",
		Filename:     "diagram.svg",
		URL:          "/uploads/workspaces/workspace-1/asset.svg",
		ContentType:  "image/svg+xml",
		SizeBytes:    11,
		Checksum:     "sha256:test",
		CreatedAt:    createdAt,
	})
	if err != nil {
		t.Fatalf("NewUploadedAsset: %v", err)
	}
	uploader := &fakeUploader{available: true, result: spacehttp.UploadResult{Asset: &asset, IssueID: &issueID}}
	handler := spacehttp.NewUploadHandler(
		uploader,
		func(*stdhttp.Request) string { return "workspace-1" },
		spacehttp.URLPolicy{PublicURL: "https://api.example.test"},
	)
	req := multipartRequest(t, "diagram.svg", []byte("<svg></svg>"), map[string]string{"issue_id": issueID})
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != stdhttp.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if uploader.request.WorkspaceID != "workspace-1" || uploader.request.UserID != "member-1" {
		t.Fatalf("upload request = %#v", uploader.request)
	}
	if uploader.request.ContentType != "image/svg+xml" || uploader.request.Filename != "diagram.svg" {
		t.Fatalf("upload metadata = %#v", uploader.request)
	}
	if uploader.request.IssueID == nil || *uploader.request.IssueID != issueID {
		t.Fatalf("Issue request = %#v", uploader.request.IssueID)
	}
	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response["id"] != asset.ID() || response["download_url"] != "/api/attachments/"+asset.ID()+"/download" {
		t.Fatalf("response = %#v", response)
	}
	if response["markdown_url"] != "https://api.example.test/api/attachments/"+asset.ID()+"/download" {
		t.Fatalf("markdown_url = %#v", response["markdown_url"])
	}
}

func TestUploadHandlerMapsMalformedIssueIDFromAuthorizedWorkflow(t *testing.T) {
	uploader := &fakeUploader{available: true, err: spacehttp.ErrInvalidIssueID}
	handler := spacehttp.NewUploadHandler(uploader, func(*stdhttp.Request) string { return "workspace-1" }, spacehttp.URLPolicy{})
	req := multipartRequest(t, "notes.txt", []byte("hello"), map[string]string{"issue_id": "not-a-uuid"})
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != stdhttp.StatusBadRequest || recorder.Body.String() != "{\"error\":\"invalid issue_id\"}\n" {
		t.Fatalf("response = %d %q", recorder.Code, recorder.Body.String())
	}
	if uploader.request.Filename != "notes.txt" || uploader.request.IssueID == nil {
		t.Fatalf("workflow request = %#v", uploader.request)
	}
}

func TestUploadHandlerPassesExtraneousIssueIDThroughForPersonalCompatibility(t *testing.T) {
	uploader := &fakeUploader{
		available: true,
		result:    spacehttp.UploadResult{ID: "asset-1", URL: "/uploads/users/member-1/asset-1.txt", Filename: "notes.txt"},
	}
	handler := spacehttp.NewUploadHandler(uploader, func(*stdhttp.Request) string { return "" }, spacehttp.URLPolicy{})
	req := multipartRequest(t, "notes.txt", []byte("hello"), map[string]string{"issue_id": "not-a-uuid"})
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != stdhttp.StatusOK {
		t.Fatalf("response = %d %q", recorder.Code, recorder.Body.String())
	}
	if uploader.request.IssueID == nil || *uploader.request.IssueID != "not-a-uuid" {
		t.Fatalf("workflow request = %#v", uploader.request)
	}
}

func TestUploadHandlerMapsIssueWorkflowDenial(t *testing.T) {
	uploader := &fakeUploader{available: true, err: spacehttp.ErrIssueNotAccessible}
	handler := spacehttp.NewUploadHandler(
		uploader,
		func(*stdhttp.Request) string { return "workspace-1" },
		spacehttp.URLPolicy{},
	)
	req := multipartRequest(t, "notes.txt", []byte("hello"), map[string]string{
		"issue_id": "01980000-0000-7000-8000-000000000002",
	})
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != stdhttp.StatusForbidden || recorder.Body.String() != "{\"error\":\"invalid issue_id\"}\n" {
		t.Fatalf("response = %d %q", recorder.Code, recorder.Body.String())
	}
	if uploader.request.Filename != "notes.txt" {
		t.Fatalf("workflow request = %#v", uploader.request)
	}
}

func TestUploadHandlerPreservesAuthorizationAndStorageErrors(t *testing.T) {
	tests := []struct {
		name       string
		available  bool
		err        error
		wantStatus int
		wantBody   string
	}{
		{name: "storage unavailable", available: false, wantStatus: stdhttp.StatusServiceUnavailable, wantBody: "{\"error\":\"file upload not configured\"}\n"},
		{name: "not a member", available: true, err: application.ErrNotWorkspaceMember, wantStatus: stdhttp.StatusForbidden, wantBody: "{\"error\":\"not a member of this workspace\"}\n"},
		{name: "upload failure", available: true, err: errors.Join(application.ErrUploadFailed, errors.New("offline")), wantStatus: stdhttp.StatusInternalServerError, wantBody: "{\"error\":\"upload failed\"}\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			uploader := &fakeUploader{available: test.available, err: test.err}
			handler := spacehttp.NewUploadHandler(uploader, func(*stdhttp.Request) string { return "workspace-1" }, spacehttp.URLPolicy{})
			req := multipartRequest(t, "notes.txt", []byte("hello"), nil)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, req)

			if recorder.Code != test.wantStatus || recorder.Body.String() != test.wantBody {
				t.Fatalf("response = %d %q", recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestUploadHandlerPreservesSignedAndStableDownloadURLModes(t *testing.T) {
	asset, err := domain.NewUploadedAsset(domain.UploadedAssetParams{
		ID:           "asset-1",
		WorkspaceID:  "workspace-1",
		UploaderType: domain.UploaderMember,
		UploaderID:   "member-1",
		Filename:     "notes.txt",
		URL:          "https://cdn.example.test/uploads/asset-1.txt",
		ContentType:  "text/plain",
		SizeBytes:    5,
		Checksum:     "sha256:test",
		CreatedAt:    time.Date(2026, time.August, 2, 10, 30, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("NewUploadedAsset: %v", err)
	}
	tests := []struct {
		name       string
		capability string
		wantURL    string
	}{
		{name: "signed by default", wantURL: "signed:" + asset.URL()},
		{name: "stable capability", capability: "stable_attachment_urls", wantURL: "/api/attachments/asset-1/download"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			uploader := &fakeUploader{available: true, result: spacehttp.UploadResult{Asset: &asset}}
			handler := spacehttp.NewUploadHandler(
				uploader,
				func(*stdhttp.Request) string { return "workspace-1" },
				spacehttp.URLPolicy{Signer: fakeSigner{}},
			)
			req := multipartRequest(t, "notes.txt", []byte("hello"), nil)
			req.Header.Set("X-Client-Capabilities", test.capability)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, req)

			var response map[string]any
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if response["download_url"] != test.wantURL {
				t.Fatalf("download_url = %#v, want %q", response["download_url"], test.wantURL)
			}
		})
	}
}

func TestUploadHandlerReturnsLegacyDirectLinkForMetadataFailure(t *testing.T) {
	fallback := application.UploadResult{URL: "/uploads/workspaces/workspace-1/asset.txt", Filename: "notes.txt"}
	uploader := &fakeUploader{
		available: true,
		err: &application.MetadataPersistenceError{
			Result: fallback,
			Err:    errors.New("database unavailable"),
		},
	}
	handler := spacehttp.NewUploadHandler(uploader, func(*stdhttp.Request) string { return "workspace-1" }, spacehttp.URLPolicy{})
	req := multipartRequest(t, "notes.txt", []byte("hello"), nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != stdhttp.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var response map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response["id"] != "" || response["url"] != fallback.URL || response["filename"] != fallback.Filename {
		t.Fatalf("response = %#v", response)
	}
}
