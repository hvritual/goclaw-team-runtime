package sqlitelocal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/multica-ai/multica/server/internal/storage"
)

type memoryUploadStorage struct {
	mu      sync.Mutex
	objects map[string][]byte
}

func (s *memoryUploadStorage) Upload(_ context.Context, key string, data []byte, _, _ string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.objects == nil {
		s.objects = make(map[string][]byte)
	}
	s.objects[key] = append([]byte(nil), data...)
	return "/uploads/" + key, nil
}
func (*memoryUploadStorage) Delete(context.Context, string) {}
func (s *memoryUploadStorage) DeleteObject(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.objects, key)
	return nil
}
func (*memoryUploadStorage) DeleteKeys(context.Context, []string) {}
func (*memoryUploadStorage) KeyFromURL(rawURL string) string {
	return strings.TrimPrefix(rawURL, "/uploads/")
}
func (*memoryUploadStorage) ObjectURL(key string) string { return "/uploads/" + key }
func (*memoryUploadStorage) CdnDomain() string           { return "" }
func (s *memoryUploadStorage) GetReader(_ context.Context, key string) (io.ReadCloser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.objects[key]
	if !ok {
		return nil, errors.New("object not found")
	}
	return io.NopCloser(bytes.NewReader(append([]byte(nil), value...))), nil
}

var _ storage.Storage = (*memoryUploadStorage)(nil)

func TestSQLiteLocalNativeSpaceUploadAndDownload(t *testing.T) {
	store := &memoryUploadStorage{}
	app, err := Open(t.TempDir()+"/space.db", Options{VerificationCode: "888888", Storage: store})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = app.Close() })
	client := &testClient{t: t, app: app}
	client.request(http.MethodPost, "/auth/send-code", map[string]any{"email": "owner@example.com"}, http.StatusNoContent)
	login := client.request(http.MethodPost, "/auth/verify-code", map[string]any{"email": "owner@example.com", "code": "888888"}, http.StatusOK)
	client.token = login["token"].(string)
	workspace := client.request(http.MethodPost, "/api/workspaces", map[string]any{"name": "Space", "slug": "space"}, http.StatusCreated)
	client.slug = "space"
	issue := client.request(http.MethodPost, "/api/issues", map[string]any{"title": "Asset owner"}, http.StatusCreated)

	request := sqliteMultipartRequest(t, issue["id"].(string), []byte("hello sqlite"))
	request.Header.Set("Authorization", "Bearer "+client.token)
	request.Header.Set("X-Workspace-Slug", client.slug)
	recorder := httptest.NewRecorder()
	app.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("upload status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	response := decodeJSONMap(t, recorder.Body.Bytes())
	assetID, _ := response["id"].(string)
	if assetID == "" || response["issue_id"] != issue["id"] || response["download_url"] == "" {
		t.Fatalf("upload response=%+v", response)
	}
	var assetCount, versionCount, relationCount int
	if err := app.db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM space_assets WHERE id = ?`, assetID).Scan(&assetCount); err != nil {
		t.Fatal(err)
	}
	if err := app.db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM space_asset_versions WHERE asset_id = ?`, assetID).Scan(&versionCount); err != nil {
		t.Fatal(err)
	}
	if err := app.db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM issue_asset_refs WHERE asset_id = ? AND issue_id = ?`, assetID, issue["id"]).Scan(&relationCount); err != nil {
		t.Fatal(err)
	}
	if assetCount != 1 || versionCount != 1 || relationCount != 1 {
		t.Fatalf("asset=%d version=%d relation=%d", assetCount, versionCount, relationCount)
	}
	attachments := client.requestList(http.MethodGet, "/api/issues/"+issue["id"].(string)+"/attachments", http.StatusOK)
	if len(attachments) != 1 || attachments[0]["id"] != assetID || attachments[0]["issue_id"] != issue["id"] {
		t.Fatalf("attachments=%+v", attachments)
	}

	download := httptest.NewRequestWithContext(t.Context(), http.MethodGet, response["download_url"].(string), nil)
	download.Header.Set("Authorization", "Bearer "+client.token)
	downloadRecorder := httptest.NewRecorder()
	app.Handler().ServeHTTP(downloadRecorder, download)
	if downloadRecorder.Code != http.StatusOK || downloadRecorder.Body.String() != "hello sqlite" {
		t.Fatalf("download status=%d body=%q", downloadRecorder.Code, downloadRecorder.Body.String())
	}
	_ = workspace
}

func TestSQLiteLocalSpaceUploadChecksMembershipBeforeIssueVisibility(t *testing.T) {
	app, err := Open(t.TempDir()+"/space-deny.db", Options{VerificationCode: "888888", Storage: &memoryUploadStorage{}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = app.Close() })
	owner := &testClient{t: t, app: app}
	owner.request(http.MethodPost, "/auth/send-code", map[string]any{"email": "owner@example.com"}, http.StatusNoContent)
	ownerLogin := owner.request(http.MethodPost, "/auth/verify-code", map[string]any{"email": "owner@example.com", "code": "888888"}, http.StatusOK)
	owner.token = ownerLogin["token"].(string)
	owner.request(http.MethodPost, "/api/workspaces", map[string]any{"name": "Private", "slug": "private"}, http.StatusCreated)

	outsider := &testClient{t: t, app: app}
	outsider.request(http.MethodPost, "/auth/send-code", map[string]any{"email": "outsider@example.com"}, http.StatusNoContent)
	outsiderLogin := outsider.request(http.MethodPost, "/auth/verify-code", map[string]any{"email": "outsider@example.com", "code": "888888"}, http.StatusOK)
	request := sqliteMultipartRequest(t, "01980000-0000-7000-8000-000000000099", []byte("secret"))
	request.Header.Set("Authorization", "Bearer "+outsiderLogin["token"].(string))
	request.Header.Set("X-Workspace-Slug", "private")
	recorder := httptest.NewRecorder()
	app.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden || !strings.Contains(recorder.Body.String(), "not a member") {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestSQLiteLocalNativeSpacePersonalUploadIsPubliclyReadable(t *testing.T) {
	store := &memoryUploadStorage{}
	app, err := Open(t.TempDir()+"/space-personal.db", Options{VerificationCode: "888888", Storage: store})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = app.Close() })
	client := &testClient{t: t, app: app}
	client.request(http.MethodPost, "/auth/send-code", map[string]any{"email": "avatar@example.com"}, http.StatusNoContent)
	login := client.request(http.MethodPost, "/auth/verify-code", map[string]any{"email": "avatar@example.com", "code": "888888"}, http.StatusOK)
	client.token = login["token"].(string)

	request := sqliteMultipartRequest(t, "", []byte("avatar bytes"))
	request.Header.Set("Authorization", "Bearer "+client.token)
	recorder := httptest.NewRecorder()
	app.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("upload status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	response := decodeJSONMap(t, recorder.Body.Bytes())
	rawURL, _ := response["url"].(string)
	if !strings.HasPrefix(rawURL, "/uploads/users/") || response["download_url"] != rawURL || response["filename"] != "notes.txt" {
		t.Fatalf("response=%+v", response)
	}

	download := httptest.NewRequestWithContext(t.Context(), http.MethodGet, rawURL, nil)
	downloadRecorder := httptest.NewRecorder()
	app.Handler().ServeHTTP(downloadRecorder, download)
	if downloadRecorder.Code != http.StatusOK || downloadRecorder.Body.String() != "avatar bytes" {
		t.Fatalf("download status=%d body=%q", downloadRecorder.Code, downloadRecorder.Body.String())
	}
	if downloadRecorder.Header().Get("Content-Type") != "text/plain; charset=utf-8" {
		t.Fatalf("content-type=%q", downloadRecorder.Header().Get("Content-Type"))
	}

	workspaceObject := httptest.NewRequestWithContext(t.Context(), http.MethodGet,
		"/uploads/workspaces/01980000-0000-7000-8000-000000000001/01980000-0000-7000-8000-000000000002.txt", nil)
	workspaceRecorder := httptest.NewRecorder()
	app.Handler().ServeHTTP(workspaceRecorder, workspaceObject)
	if workspaceRecorder.Code != http.StatusNotFound {
		t.Fatalf("workspace object status=%d", workspaceRecorder.Code)
	}
}

func sqliteMultipartRequest(t *testing.T, issueID string, content []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "notes.txt")
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
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/upload-file", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}

func decodeJSONMap(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal(body, &value); err != nil {
		t.Fatalf("decode JSON: %v; body=%s", err, body)
	}
	return value
}
