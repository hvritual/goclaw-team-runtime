package http

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/multica-ai/multica/server/internal/modules/space/contract"
)

type fakeUploader struct {
	available bool
	request   UploadRequest
	result    UploadResult
	err       error
}

func (u *fakeUploader) Available() bool { return u.available }
func (u *fakeUploader) Upload(_ context.Context, request UploadRequest) (UploadResult, error) {
	u.request = request
	return u.result, u.err
}

func TestUploadHandlerPreservesMultipartAndAttachmentResponse(t *testing.T) {
	issueID := "01980000-0000-7000-8000-000000000010"
	uploader := &fakeUploader{available: true, result: UploadResult{
		Asset: &contract.Asset_Asset{
			Id: "asset", WorkspaceId: "workspace", UploaderType: "member", UploaderId: "user",
			Filename: "diagram.svg", MediaType: "image/svg+xml", SizeBytes: "6",
			CreatedAt: "2026-08-02T15:00:00Z",
		},
		IssueID: &issueID,
	}}
	handler := NewUploadHandler(
		uploader,
		func(*stdhttp.Request) string { return "user" },
		func(*stdhttp.Request) (string, error) { return "workspace", nil },
		URLPolicy{PublicURL: "https://api.example.test"},
	)
	request := multipartUploadRequest(t, "diagram.svg", []byte("<svg/>"), issueID)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != stdhttp.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if uploader.request.ContentType != "image/svg+xml" || uploader.request.WorkspaceID != "workspace" || uploader.request.UserID != "user" {
		t.Fatalf("request=%+v", uploader.request)
	}
	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response["download_url"] != "/api/attachments/asset/download" ||
		response["markdown_url"] != "https://api.example.test/api/attachments/asset/download" ||
		response["issue_id"] != issueID {
		t.Fatalf("response=%+v", response)
	}
}

func TestUploadHandlerRejectsUnknownWorkspaceWithoutPersonalFallback(t *testing.T) {
	uploader := &fakeUploader{available: true}
	handler := NewUploadHandler(
		uploader,
		func(*stdhttp.Request) string { return "user" },
		func(*stdhttp.Request) (string, error) { return "", ErrWorkspaceNotFound },
		URLPolicy{},
	)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, multipartUploadRequest(t, "x.txt", []byte("x"), ""))
	if recorder.Code != stdhttp.StatusNotFound || uploader.request.Filename != "" {
		t.Fatalf("status=%d request=%+v body=%s", recorder.Code, uploader.request, recorder.Body.String())
	}
}

func multipartUploadRequest(t *testing.T, filename string, content []byte, issueID string) *stdhttp.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatal(err)
	}
	if issueID != "" {
		if err := writer.WriteField("issue_id", issueID); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequestWithContext(t.Context(), stdhttp.MethodPost, "/api/upload-file", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}
