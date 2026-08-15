package bootstrap

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/hvritual/workspace/internal/modules/auth"
)

type runtimeAttachmentResponse struct {
	ID            string  `json:"id"`
	WorkspaceID   string  `json:"workspace_id"`
	IssueID       *string `json:"issue_id"`
	CommentID     *string `json:"comment_id"`
	ChatSessionID *string `json:"chat_session_id"`
	ChatMessageID *string `json:"chat_message_id"`
	UploaderType  string  `json:"uploader_type"`
	UploaderID    string  `json:"uploader_id"`
	Filename      string  `json:"filename"`
	URL           string  `json:"url"`
	DownloadURL   string  `json:"download_url"`
	MarkdownURL   string  `json:"markdown_url"`
	ContentType   string  `json:"content_type"`
	SizeBytes     int64   `json:"size_bytes"`
	CreatedAt     string  `json:"created_at"`
}

func TestSQLiteRuntimeServesCanonicalIssueAttachments(t *testing.T) {
	fixture := newCollaborationRuntimeFixture(
		t,
		filepath.Join(t.TempDir(), "issue-attachment-red.db"),
		"issue-attachment-red",
		"issue-attachment-red@example.com",
	)
	filePath := filepath.Join(t.TempDir(), "evidence.txt")
	if err := os.WriteFile(filePath, []byte("canonical attachment evidence\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	upload := runtimeMultipartFileRequest(t, fixture.runtime, "/api/upload-file", filePath, map[string]string{
		"issue_id": fixture.issueID,
	}, fixture.headers)
	if upload.Code == http.StatusNotFound {
		t.Fatalf("Canonical attachment upload route is missing: %d %s", upload.Code, upload.Body.String())
	}
	if upload.Code != http.StatusOK {
		t.Fatalf("upload attachment = %d %s", upload.Code, upload.Body.String())
	}
	var attachment runtimeAttachmentResponse
	if err := json.Unmarshal(upload.Body.Bytes(), &attachment); err != nil {
		t.Fatalf("decode attachment: %v body=%s", err, upload.Body.String())
	}
	if attachment.ID == "" || attachment.WorkspaceID != fixture.workspaceID || attachment.IssueID == nil || *attachment.IssueID != fixture.issueID {
		t.Fatalf("attachment response = %#v", attachment)
	}

	listed := runtimeRequest(fixture.runtime, http.MethodGet, "/api/issues/"+fixture.issueID+"/attachments", "", fixture.headers)
	if listed.Code != http.StatusOK {
		t.Fatalf("list attachments = %d %s", listed.Code, listed.Body.String())
	}
	metadata := runtimeRequest(fixture.runtime, http.MethodGet, "/api/attachments/"+attachment.ID, "", fixture.headers)
	if metadata.Code != http.StatusOK {
		t.Fatalf("get attachment = %d %s", metadata.Code, metadata.Body.String())
	}
	preview := runtimeRequest(fixture.runtime, http.MethodGet, "/api/attachments/"+attachment.ID+"/content", "", fixture.headers)
	if preview.Code != http.StatusOK || preview.Body.String() != "canonical attachment evidence\n" {
		t.Fatalf("preview attachment = %d %q", preview.Code, preview.Body.String())
	}
	download := runtimeRequest(fixture.runtime, http.MethodGet, "/api/attachments/"+attachment.ID+"/download", "", fixture.headers)
	if download.Code != http.StatusOK || !bytes.Equal(download.Body.Bytes(), []byte("canonical attachment evidence\n")) {
		t.Fatalf("download attachment = %d %q", download.Code, download.Body.String())
	}
	config := runtimeRequest(fixture.runtime, http.MethodGet, "/api/config", "", nil)
	if config.Code != http.StatusOK || !containsJSON(config.Body.Bytes(), `"issue_attachments":true`) {
		t.Fatalf("attachment capability = %d %s", config.Code, config.Body.String())
	}
}

func TestSQLiteRuntimeBindsAttachmentsToIssuesAndComments(t *testing.T) {
	fixture := newCollaborationRuntimeFixture(
		t,
		filepath.Join(t.TempDir(), "issue-attachment-binding.db"),
		"issue-attachment-binding",
		"issue-attachment-binding@example.com",
	)
	writeFile := func(name, content string) string {
		t.Helper()
		filePath := filepath.Join(t.TempDir(), name)
		if err := os.WriteFile(filePath, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		return filePath
	}

	t.Run("Issue create binds an unbound upload", func(t *testing.T) {
		upload := runtimeMultipartFileRequest(t, fixture.runtime, "/api/upload-file", writeFile("create.txt", "create attachment"), nil, fixture.headers)
		attachment := decodeRuntimeAttachment(t, upload)
		if attachment.IssueID != nil || attachment.CommentID != nil {
			t.Fatalf("unbound upload = %#v", attachment)
		}
		created := runtimeRequest(fixture.runtime, http.MethodPost, "/api/issues", `{"title":"Created with attachment","attachment_ids":["`+attachment.ID+`"]}`, fixture.headers)
		if created.Code != http.StatusCreated {
			t.Fatalf("create Issue with attachment = %d %s", created.Code, created.Body.String())
		}
		var issue runtimeIssueResponse
		if err := json.Unmarshal(created.Body.Bytes(), &issue); err != nil || issue.ID == "" {
			t.Fatalf("decode created Issue: %v body=%s", err, created.Body.String())
		}
		listed := runtimeRequest(fixture.runtime, http.MethodGet, "/api/issues/"+issue.ID+"/attachments", "", fixture.headers)
		attachments := decodeRuntimeAttachmentList(t, listed)
		if len(attachments) != 1 || attachments[0].ID != attachment.ID || attachments[0].IssueID == nil || *attachments[0].IssueID != issue.ID {
			t.Fatalf("created Issue attachments = %#v", attachments)
		}
	})

	t.Run("Comment create binds an Issue upload and renders it in the timeline", func(t *testing.T) {
		upload := runtimeMultipartFileRequest(t, fixture.runtime, "/api/upload-file", writeFile("comment.txt", "comment attachment"), map[string]string{"issue_id": fixture.issueID}, fixture.headers)
		attachment := decodeRuntimeAttachment(t, upload)
		comment := runtimeRequest(fixture.runtime, http.MethodPost, "/api/issues/"+fixture.issueID+"/comments", `{"content":"with file","attachment_ids":["`+attachment.ID+`"]}`, fixture.headers)
		if comment.Code != http.StatusCreated || !containsJSON(comment.Body.Bytes(), `"id":"`+attachment.ID+`"`, `"comment_id":`) {
			t.Fatalf("create Comment with attachment = %d %s", comment.Code, comment.Body.String())
		}
		timeline := runtimeRequest(fixture.runtime, http.MethodGet, "/api/issues/"+fixture.issueID+"/timeline", "", fixture.headers)
		if timeline.Code != http.StatusOK || !containsJSON(timeline.Body.Bytes(), `"filename":"comment.txt"`, `"content_type":"text/plain; charset=utf-8"`) {
			t.Fatalf("timeline attachment = %d %s", timeline.Code, timeline.Body.String())
		}
	})

	t.Run("Issue update replaces and clears the attachment bag atomically", func(t *testing.T) {
		issue := createRuntimeIssue(t, fixture.runtime, fixture.headers, "Attachment update")
		first := decodeRuntimeAttachment(t, runtimeMultipartFileRequest(t, fixture.runtime, "/api/upload-file", writeFile("update-first.txt", "first"), nil, fixture.headers))
		second := decodeRuntimeAttachment(t, runtimeMultipartFileRequest(t, fixture.runtime, "/api/upload-file", writeFile("update-second.txt", "second"), nil, fixture.headers))
		updated := runtimeRequest(fixture.runtime, http.MethodPut, "/api/issues/"+issue.ID, `{"attachment_ids":["`+first.ID+`","`+second.ID+`"]}`, fixture.headers)
		if updated.Code != http.StatusOK {
			t.Fatalf("bind Issue attachments = %d %s", updated.Code, updated.Body.String())
		}
		values := decodeRuntimeAttachmentList(t, runtimeRequest(fixture.runtime, http.MethodGet, "/api/issues/"+issue.ID+"/attachments", "", fixture.headers))
		if len(values) != 2 || values[0].ID != first.ID || values[1].ID != second.ID {
			t.Fatalf("bound Issue attachments = %#v", values)
		}
		invalid := runtimeRequest(fixture.runtime, http.MethodPut, "/api/issues/"+issue.ID, `{"attachment_ids":["missing-attachment"]}`, fixture.headers)
		if invalid.Code != http.StatusNotFound {
			t.Fatalf("invalid Issue attachment = %d %s", invalid.Code, invalid.Body.String())
		}
		values = decodeRuntimeAttachmentList(t, runtimeRequest(fixture.runtime, http.MethodGet, "/api/issues/"+issue.ID+"/attachments", "", fixture.headers))
		if len(values) != 2 {
			t.Fatalf("failed Issue update changed attachments = %#v", values)
		}
		cleared := runtimeRequest(fixture.runtime, http.MethodPut, "/api/issues/"+issue.ID, `{"attachment_ids":[]}`, fixture.headers)
		if cleared.Code != http.StatusOK {
			t.Fatalf("clear Issue attachments = %d %s", cleared.Code, cleared.Body.String())
		}
		if values = decodeRuntimeAttachmentList(t, runtimeRequest(fixture.runtime, http.MethodGet, "/api/issues/"+issue.ID+"/attachments", "", fixture.headers)); len(values) != 0 {
			t.Fatalf("cleared Issue attachments = %#v", values)
		}
	})

	t.Run("Comment update replaces and clears only Issue-bound attachments", func(t *testing.T) {
		issue := createRuntimeIssue(t, fixture.runtime, fixture.headers, "Comment attachment update")
		first := decodeRuntimeAttachment(t, runtimeMultipartFileRequest(t, fixture.runtime, "/api/upload-file", writeFile("comment-first.txt", "first"), map[string]string{"issue_id": issue.ID}, fixture.headers))
		second := decodeRuntimeAttachment(t, runtimeMultipartFileRequest(t, fixture.runtime, "/api/upload-file", writeFile("comment-second.txt", "second"), map[string]string{"issue_id": issue.ID}, fixture.headers))
		comment := createRuntimeComment(t, fixture.runtime, fixture.headers, issue.ID, `{"content":"first","attachment_ids":["`+first.ID+`"]}`)
		updated := runtimeRequest(fixture.runtime, http.MethodPut, "/api/comments/"+comment.ID, `{"content":"second","attachment_ids":["`+second.ID+`"]}`, fixture.headers)
		if updated.Code != http.StatusOK || !containsJSON(updated.Body.Bytes(), `"id":"`+second.ID+`"`, `"filename":"comment-second.txt"`) || containsJSON(updated.Body.Bytes(), `"id":"`+first.ID+`"`) {
			t.Fatalf("replace Comment attachments = %d %s", updated.Code, updated.Body.String())
		}
		invalid := runtimeRequest(fixture.runtime, http.MethodPut, "/api/comments/"+comment.ID, `{"content":"invalid","attachment_ids":["missing-attachment"]}`, fixture.headers)
		if invalid.Code != http.StatusBadRequest {
			t.Fatalf("invalid Comment attachment = %d %s", invalid.Code, invalid.Body.String())
		}
		readback := runtimeRequest(fixture.runtime, http.MethodGet, "/api/issues/"+issue.ID+"/comments", "", fixture.headers)
		if readback.Code != http.StatusOK || !containsJSON(readback.Body.Bytes(), `"id":"`+second.ID+`"`) {
			t.Fatalf("failed Comment update changed attachments = %d %s", readback.Code, readback.Body.String())
		}
		cleared := runtimeRequest(fixture.runtime, http.MethodPut, "/api/comments/"+comment.ID, `{"content":"clear","attachment_ids":[]}`, fixture.headers)
		if cleared.Code != http.StatusOK || !containsJSON(cleared.Body.Bytes(), `"attachments":[]`) {
			t.Fatalf("clear Comment attachments = %d %s", cleared.Code, cleared.Body.String())
		}
	})
}

func TestSQLiteRuntimeHonorsCanonicalAttachmentBoundary(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "issue-attachment-contract.db")
	fixture := newCollaborationRuntimeFixture(t, databasePath, "issue-attachment-contract", "issue-attachment-owner@example.com")
	member := addCollaborationRuntimeMember(t, fixture, "issue-attachment-member@example.com", "member")
	outsider := verifyRuntimeLogin(t, fixture.runtime, "issue-attachment-outsider@example.com")
	filePath := filepath.Join(t.TempDir(), "contract.md")
	content := []byte("# Canonical attachment\n\n<script>alert(1)</script>\n")
	if err := os.WriteFile(filePath, content, 0o600); err != nil {
		t.Fatal(err)
	}

	t.Run("identity is resolved before Workspace and multipart", func(t *testing.T) {
		missingAuth := runtimeMultipartFileRequest(t, fixture.runtime, "/api/upload-file", filePath, map[string]string{"issue_id": fixture.issueID}, map[string]string{"X-Workspace-Slug": fixture.workspaceSlug})
		assertRuntimeResponse(t, missingAuth.Code, missingAuth.Body.String(), http.StatusUnauthorized, `{"error":"user not authenticated"}`)
		missingWorkspace := runtimeMultipartFileRequest(t, fixture.runtime, "/api/upload-file", filePath, nil, map[string]string{"Authorization": "Bearer " + fixture.login.Token})
		assertRuntimeResponse(t, missingWorkspace.Code, missingWorkspace.Body.String(), http.StatusBadRequest, `{"error":"workspace is required"}`)
		foreign := runtimeMultipartFileRequest(t, fixture.runtime, "/api/upload-file", filePath, nil, collaborationHeaders(outsider.Token, fixture.workspaceSlug))
		assertRuntimeResponse(t, foreign.Code, foreign.Body.String(), http.StatusNotFound, `{"error":"attachment not found"}`)
		if _, err := fixture.runtime.Database().Exec(`UPDATE auth_sessions SET expires_at_unix_nano=0 WHERE user_id=?`, outsider.UserID); err != nil {
			t.Fatal(err)
		}
		expired := runtimeMultipartFileRequest(t, fixture.runtime, "/api/upload-file", filePath, nil, collaborationHeaders(outsider.Token, fixture.workspaceSlug))
		assertRuntimeResponse(t, expired.Code, expired.Body.String(), http.StatusUnauthorized, `{"error":"user not authenticated"}`)
		cookieNoCSRF := runtimeMultipartFileRequest(t, fixture.runtime, "/api/upload-file", filePath, nil, map[string]string{"Cookie": "multica_auth=" + fixture.login.Token, "X-Workspace-Slug": fixture.workspaceSlug})
		assertRuntimeResponse(t, cookieNoCSRF.Code, cookieNoCSRF.Body.String(), http.StatusForbidden, `{"error":"invalid CSRF token"}`)
	})

	cookieHeaders := map[string]string{
		"Cookie":       "multica_auth=" + fixture.login.Token + "; multica_csrf=" + fixture.login.CSRF,
		"X-CSRF-Token": fixture.login.CSRF, "X-Workspace-Slug": fixture.workspaceSlug,
	}
	upload := runtimeMultipartFileRequest(t, fixture.runtime, "/api/upload-file", filePath, map[string]string{"issue_id": fixture.identifier}, cookieHeaders)
	attachment := decodeRuntimeAttachment(t, upload)
	if attachment.Filename != "contract.md" || attachment.ContentType != "text/plain; charset=utf-8" || attachment.SizeBytes != int64(len(content)) || attachment.UploaderID != fixture.login.UserID {
		t.Fatalf("normalized attachment = %#v", attachment)
	}
	var exact map[string]any
	if err := json.Unmarshal(upload.Body.Bytes(), &exact); err != nil || len(exact) != 15 {
		t.Fatalf("attachment exact body (%d keys) = %s err=%v", len(exact), upload.Body.String(), err)
	}
	for _, key := range []string{"id", "workspace_id", "issue_id", "comment_id", "chat_session_id", "chat_message_id", "uploader_type", "uploader_id", "filename", "url", "download_url", "markdown_url", "content_type", "size_bytes", "created_at"} {
		if _, ok := exact[key]; !ok {
			t.Fatalf("attachment response missing %s: %s", key, upload.Body.String())
		}
	}

	t.Run("metadata preview and download reauthorize from stored ownership", func(t *testing.T) {
		metadata := runtimeRequest(fixture.runtime, http.MethodGet, "/api/attachments/"+attachment.ID, "", map[string]string{"Authorization": "Bearer " + member.Token})
		if metadata.Code != http.StatusOK || !containsJSON(metadata.Body.Bytes(), `"workspace_id":"`+fixture.workspaceID+`"`) {
			t.Fatalf("member metadata = %d %s", metadata.Code, metadata.Body.String())
		}
		preview := runtimeRequest(fixture.runtime, http.MethodGet, "/api/attachments/"+attachment.ID+"/content", "", map[string]string{"Authorization": "Bearer " + member.Token})
		if preview.Code != http.StatusOK || !bytes.Equal(preview.Body.Bytes(), content) || preview.Header().Get("Content-Type") != "text/plain; charset=utf-8" || preview.Header().Get("X-Original-Content-Type") != attachment.ContentType || preview.Header().Get("X-Content-Type-Options") != "nosniff" || preview.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("member preview = %d headers=%v body=%q", preview.Code, preview.Header(), preview.Body.String())
		}
		download := runtimeRequest(fixture.runtime, http.MethodGet, "/api/attachments/"+attachment.ID+"/download", "", map[string]string{"Cookie": "multica_auth=" + member.Token})
		if download.Code != http.StatusOK || !bytes.Equal(download.Body.Bytes(), content) || !strings.Contains(download.Header().Get("Content-Disposition"), "contract.md") {
			t.Fatalf("member download = %d headers=%v body=%q", download.Code, download.Header(), download.Body.String())
		}
		missing := runtimeRequest(fixture.runtime, http.MethodGet, "/api/attachments/"+attachment.ID, "", nil)
		assertRuntimeResponse(t, missing.Code, missing.Body.String(), http.StatusUnauthorized, `{"error":"user not authenticated"}`)
		foreign := runtimeRequest(fixture.runtime, http.MethodGet, "/api/attachments/"+attachment.ID, "", map[string]string{"Authorization": "Bearer " + outsider.Token})
		assertRuntimeResponse(t, foreign.Code, foreign.Body.String(), http.StatusUnauthorized, `{"error":"user not authenticated"}`)
	})

	t.Run("cross Workspace targets are hidden", func(t *testing.T) {
		second := runtimeRequest(fixture.runtime, http.MethodPost, "/api/workspaces", `{"name":"Attachment Other","slug":"attachment-other"}`, map[string]string{"Authorization": "Bearer " + fixture.login.Token, "Content-Type": "application/json"})
		if second.Code != http.StatusCreated {
			t.Fatalf("create other Workspace = %d %s", second.Code, second.Body.String())
		}
		otherHeaders := collaborationHeaders(fixture.login.Token, "attachment-other")
		crossTarget := runtimeMultipartFileRequest(t, fixture.runtime, "/api/upload-file", filePath, map[string]string{"issue_id": fixture.issueID}, otherHeaders)
		assertRuntimeResponse(t, crossTarget.Code, crossTarget.Body.String(), http.StatusNotFound, `{"error":"attachment not found"}`)
		listed := runtimeRequest(fixture.runtime, http.MethodGet, "/api/issues/"+fixture.issueID+"/attachments", "", otherHeaders)
		assertRuntimeResponse(t, listed.Code, listed.Body.String(), http.StatusNotFound, `{"error":"attachment not found"}`)
	})

	t.Run("delete rollback restores metadata relation and object", func(t *testing.T) {
		objectPath := onlyAttachmentObjectPath(t, databasePath+".files")
		before, err := os.ReadFile(objectPath)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fixture.runtime.Database().Exec(`CREATE TRIGGER block_space_attachment_delete BEFORE DELETE ON space_assets BEGIN SELECT RAISE(ABORT,'blocked'); END`); err != nil {
			t.Fatal(err)
		}
		blocked := runtimeRequest(fixture.runtime, http.MethodDelete, "/api/attachments/"+attachment.ID, "", fixture.headers)
		if blocked.Code != http.StatusInternalServerError {
			t.Fatalf("blocked delete = %d %s", blocked.Code, blocked.Body.String())
		}
		if _, err := fixture.runtime.Database().Exec(`DROP TRIGGER block_space_attachment_delete`); err != nil {
			t.Fatal(err)
		}
		after, err := os.ReadFile(objectPath)
		if err != nil || !bytes.Equal(before, after) {
			t.Fatalf("restored object err=%v before=%x after=%x", err, sha256.Sum256(before), sha256.Sum256(after))
		}
		if values := decodeRuntimeAttachmentList(t, runtimeRequest(fixture.runtime, http.MethodGet, "/api/issues/"+fixture.issueID+"/attachments", "", fixture.headers)); len(values) != 1 || values[0].ID != attachment.ID {
			t.Fatalf("rollback attachment list = %#v", values)
		}
	})

	t.Run("only uploader or admin can delete", func(t *testing.T) {
		denied := runtimeRequest(fixture.runtime, http.MethodDelete, "/api/attachments/"+attachment.ID, "", collaborationHeaders(member.Token, fixture.workspaceSlug))
		assertRuntimeResponse(t, denied.Code, denied.Body.String(), http.StatusForbidden, `{"error":"insufficient permissions"}`)
		cookieNoCSRF := runtimeRequest(fixture.runtime, http.MethodDelete, "/api/attachments/"+attachment.ID, "", map[string]string{"Cookie": "multica_auth=" + fixture.login.Token})
		assertRuntimeResponse(t, cookieNoCSRF.Code, cookieNoCSRF.Body.String(), http.StatusForbidden, `{"error":"invalid CSRF token"}`)
		deleted := runtimeRequest(fixture.runtime, http.MethodDelete, "/api/attachments/"+attachment.ID, "", cookieHeaders)
		if deleted.Code != http.StatusNoContent || deleted.Body.Len() != 0 {
			t.Fatalf("delete attachment = %d %q", deleted.Code, deleted.Body.String())
		}
		missing := runtimeRequest(fixture.runtime, http.MethodGet, "/api/attachments/"+attachment.ID, "", fixture.headers)
		assertRuntimeResponse(t, missing.Code, missing.Body.String(), http.StatusNotFound, `{"error":"attachment not found"}`)
		if _, err := os.Stat(databasePath + ".files"); err != nil {
			t.Fatalf("attachment root disappeared: %v", err)
		}
		if count := countRegularFiles(t, databasePath+".files"); count != 0 {
			t.Fatalf("attachment files after delete = %d", count)
		}
	})
}

func TestSQLiteRuntimeRejectsCorruptedCrossWorkspaceAttachmentReference(t *testing.T) {
	fixture := newCollaborationRuntimeFixture(t, filepath.Join(t.TempDir(), "issue-attachment-corrupt-reference.db"), "issue-attachment-corrupt-reference", "issue-attachment-corrupt-reference@example.com")
	second := runtimeRequest(fixture.runtime, http.MethodPost, "/api/workspaces", `{"name":"Foreign Attachment","slug":"foreign-attachment"}`, map[string]string{
		"Authorization": "Bearer " + fixture.login.Token, "Content-Type": "application/json",
	})
	if second.Code != http.StatusCreated {
		t.Fatalf("create foreign attachment Workspace = %d %s", second.Code, second.Body.String())
	}
	filePath := filepath.Join(t.TempDir(), "foreign.txt")
	if err := os.WriteFile(filePath, []byte("foreign workspace attachment"), 0o600); err != nil {
		t.Fatal(err)
	}
	foreign := decodeRuntimeAttachment(t, runtimeMultipartFileRequest(t, fixture.runtime, "/api/upload-file", filePath, nil, collaborationHeaders(fixture.login.Token, "foreign-attachment")))
	if _, err := fixture.runtime.Database().Exec(`UPDATE workspace_issues SET asset_ids=? WHERE workspace_id=? AND id=?`, `["`+foreign.ID+`"]`, fixture.workspaceID, fixture.issueID); err != nil {
		t.Fatal(err)
	}
	listed := runtimeRequest(fixture.runtime, http.MethodGet, "/api/issues/"+fixture.issueID+"/attachments", "", fixture.headers)
	if listed.Code != http.StatusNotFound || strings.TrimSpace(listed.Body.String()) != `{"error":"attachment not found"}` {
		t.Fatalf("corrupted cross-Workspace attachment reference = %d %s", listed.Code, listed.Body.String())
	}
}

func TestSQLiteRuntimeRetainsAttachmentBytesAcrossRestartAndReconcilesOrphans(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "issue-attachment-restart.db")
	fixture := newCollaborationRuntimeFixture(t, databasePath, "issue-attachment-restart", "issue-attachment-restart@example.com")
	filePath := filepath.Join(t.TempDir(), "restart.txt")
	content := []byte("retained attachment bytes\n")
	if err := os.WriteFile(filePath, content, 0o600); err != nil {
		t.Fatal(err)
	}
	attachment := decodeRuntimeAttachment(t, runtimeMultipartFileRequest(t, fixture.runtime, "/api/upload-file", filePath, map[string]string{"issue_id": fixture.identifier}, fixture.headers))
	objectPath := onlyAttachmentObjectPath(t, databasePath+".files")
	before, err := os.ReadFile(objectPath)
	if err != nil {
		t.Fatal(err)
	}
	beforeHash := sha256.Sum256(before)
	if err := fixture.runtime.Close(); err != nil {
		t.Fatal(err)
	}

	crashTombstone := objectPath + ".deleting-crash"
	if err := os.Rename(objectPath, crashTombstone); err != nil {
		t.Fatal(err)
	}
	orphan := filepath.Join(databasePath+".files", "orphan", "unreferenced.blob")
	if err := os.MkdirAll(filepath.Dir(orphan), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(orphan, []byte("orphan"), 0o600); err != nil {
		t.Fatal(err)
	}

	restarted := newRuntimeForConfig(t, fixture.config)
	if _, err := os.Stat(crashTombstone); !os.IsNotExist(err) {
		t.Fatalf("crash tombstone remains: %v", err)
	}
	if _, err := os.Stat(orphan); !os.IsNotExist(err) {
		t.Fatalf("orphan remains: %v", err)
	}
	after, err := os.ReadFile(objectPath)
	if err != nil || sha256.Sum256(after) != beforeHash {
		t.Fatalf("retained object hash changed: %v before=%x after=%x", err, beforeHash, sha256.Sum256(after))
	}
	listed := decodeRuntimeAttachmentList(t, runtimeRequest(restarted, http.MethodGet, "/api/issues/"+fixture.identifier+"/attachments", "", fixture.headers))
	if len(listed) != 1 || listed[0].ID != attachment.ID {
		t.Fatalf("restart attachment list = %#v", listed)
	}
	preview := runtimeRequest(restarted, http.MethodGet, "/api/attachments/"+attachment.ID+"/content", "", fixture.headers)
	if preview.Code != http.StatusOK || !bytes.Equal(preview.Body.Bytes(), content) {
		t.Fatalf("restart preview = %d %q", preview.Code, preview.Body.String())
	}
}

func TestSQLiteRuntimeCanDisableIssueAttachmentCapabilityAndRouteFamily(t *testing.T) {
	runtime := newRuntimeForConfig(t, Config{
		Name: "backend-test", Version: "test", HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath: filepath.Join(t.TempDir(), "issue-attachment-disabled.db"), WorkspaceDependencies: FailClosedWorkspaceDependencies(),
		LocalAuth: auth.LocalAuthConfig{VerificationCode: "888888"}, IssueAttachmentsEnabled: boolPointer(false),
	})
	capabilities := runtimeRequest(runtime, http.MethodGet, "/api/config", "", nil)
	if capabilities.Code != http.StatusOK || !containsJSON(capabilities.Body.Bytes(), `"issue_attachments":false`) {
		t.Fatalf("disabled attachment capability = %d %s", capabilities.Code, capabilities.Body.String())
	}
	for _, probe := range []struct{ method, path string }{
		{http.MethodPost, "/api/upload-file"},
		{http.MethodGet, "/api/issues/issue-id/attachments"},
		{http.MethodGet, "/api/attachments/attachment-id"},
		{http.MethodGet, "/api/attachments/attachment-id/content"},
		{http.MethodGet, "/api/attachments/attachment-id/download"},
		{http.MethodDelete, "/api/attachments/attachment-id"},
	} {
		response := runtimeRequest(runtime, probe.method, probe.path, "", nil)
		if response.Code != http.StatusNotFound {
			t.Fatalf("disabled %s %s = %d %s", probe.method, probe.path, response.Code, response.Body.String())
		}
	}
}

func TestSQLiteRuntimeDisabledAttachmentCapabilityRejectsIssueRelationWrites(t *testing.T) {
	fixture := newCollaborationRuntimeFixture(t, filepath.Join(t.TempDir(), "issue-attachment-disabled-relations.db"), "issue-attachment-disabled-relations", "issue-attachment-disabled-relations@example.com")
	filePath := filepath.Join(t.TempDir(), "retained-unbound.txt")
	if err := os.WriteFile(filePath, []byte("retained while capability is disabled"), 0o600); err != nil {
		t.Fatal(err)
	}
	attachment := decodeRuntimeAttachment(t, runtimeMultipartFileRequest(t, fixture.runtime, "/api/upload-file", filePath, map[string]string{"issue_id": fixture.issueID}, fixture.headers))
	commentResponse := runtimeRequest(fixture.runtime, http.MethodPost, "/api/issues/"+fixture.issueID+"/comments", `{"content":"retained comment"}`, fixture.headers)
	if commentResponse.Code != http.StatusCreated {
		t.Fatalf("create retained comment = %d %s", commentResponse.Code, commentResponse.Body.String())
	}
	var comment struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(commentResponse.Body.Bytes(), &comment); err != nil || comment.ID == "" {
		t.Fatalf("decode retained comment = %#v err=%v body=%s", comment, err, commentResponse.Body.String())
	}
	if err := fixture.runtime.Close(); err != nil {
		t.Fatal(err)
	}
	disabled := false
	fixture.config.IssueAttachmentsEnabled = &disabled
	restarted := newRuntimeForConfig(t, fixture.config)
	for _, probe := range []struct {
		method, path, body string
	}{
		{http.MethodPost, "/api/issues", `{"title":"Disabled attachment create","attachment_ids":["` + attachment.ID + `"]}`},
		{http.MethodPut, "/api/issues/" + fixture.issueID, `{"attachment_ids":["` + attachment.ID + `"]}`},
		{http.MethodPut, "/api/issues/" + fixture.issueID, `{"attachment_ids":[]}`},
	} {
		response := runtimeRequest(restarted, probe.method, probe.path, probe.body, fixture.headers)
		if response.Code != http.StatusBadRequest || strings.TrimSpace(response.Body.String()) != `{"error":"unsupported issue attachment field"}` {
			t.Fatalf("disabled relation %s %s = %d %s", probe.method, probe.path, response.Code, response.Body.String())
		}
	}
	for _, probe := range []struct {
		method, path, body string
	}{
		{http.MethodPost, "/api/issues/" + fixture.issueID + "/comments", `{"content":"blocked comment","attachment_ids":["` + attachment.ID + `"]}`},
		{http.MethodPut, "/api/comments/" + comment.ID, `{"content":"retained comment","attachment_ids":[]}`},
	} {
		response := runtimeRequest(restarted, probe.method, probe.path, probe.body, fixture.headers)
		if response.Code != http.StatusBadRequest || strings.TrimSpace(response.Body.String()) != `{"error":"unsupported comment attachment field"}` {
			t.Fatalf("disabled comment relation %s %s = %d %s", probe.method, probe.path, response.Code, response.Body.String())
		}
	}
}

func TestSQLiteRuntimeConcurrentAttachmentUploadsLoseNoReferencesOrFiles(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "issue-attachment-concurrent.db")
	fixture := newCollaborationRuntimeFixture(t, databasePath, "issue-attachment-concurrent", "issue-attachment-concurrent@example.com")
	const writers = 12
	paths := make([]string, writers)
	for index := range paths {
		paths[index] = filepath.Join(t.TempDir(), fmt.Sprintf("concurrent-%02d.txt", index))
		if err := os.WriteFile(paths[index], []byte(fmt.Sprintf("concurrent attachment %02d", index)), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	type result struct {
		response *httptest.ResponseRecorder
		err      error
	}
	results := make(chan result, writers)
	start := make(chan struct{})
	var wait sync.WaitGroup
	for _, filePath := range paths {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			response, err := runtimeMultipartFileRequestE(fixture.runtime, "/api/upload-file", filePath, map[string]string{"issue_id": fixture.issueID}, fixture.headers)
			results <- result{response: response, err: err}
		}()
	}
	close(start)
	wait.Wait()
	close(results)
	ids := map[string]struct{}{}
	for value := range results {
		if value.err != nil {
			t.Fatal(value.err)
		}
		attachment := decodeRuntimeAttachment(t, value.response)
		ids[attachment.ID] = struct{}{}
	}
	listed := decodeRuntimeAttachmentList(t, runtimeRequest(fixture.runtime, http.MethodGet, "/api/issues/"+fixture.issueID+"/attachments", "", fixture.headers))
	if len(ids) != writers || len(listed) != writers || countRegularFiles(t, databasePath+".files") != writers {
		t.Fatalf("concurrent uploads: ids=%d listed=%d files=%d", len(ids), len(listed), countRegularFiles(t, databasePath+".files"))
	}
}

func TestSQLiteRuntimePublishesCompleteAttachmentBagOnlyAfterCommit(t *testing.T) {
	fixture := newCollaborationRuntimeFixture(t, filepath.Join(t.TempDir(), "issue-attachment-events.db"), "issue-attachment-events", "issue-attachment-events@example.com")
	server := httptest.NewServer(fixture.runtime.HTTPServer())
	defer server.Close()
	filePath := filepath.Join(t.TempDir(), "event.txt")
	if err := os.WriteFile(filePath, []byte("event"), 0o600); err != nil {
		t.Fatal(err)
	}

	failedUploadSocket := dialTokenRealtime(t, server.URL, fixture.workspaceSlug, fixture.login.Token)
	defer failedUploadSocket.Close()
	if _, err := fixture.runtime.Database().Exec(`CREATE TRIGGER block_space_attachment_create BEFORE INSERT ON space_assets BEGIN SELECT RAISE(ABORT,'blocked'); END`); err != nil {
		t.Fatal(err)
	}
	failedUpload := runtimeMultipartFileRequest(t, fixture.runtime, "/api/upload-file", filePath, map[string]string{"issue_id": fixture.issueID}, fixture.headers)
	if failedUpload.Code != http.StatusInternalServerError {
		t.Fatalf("blocked upload = %d %s", failedUpload.Code, failedUpload.Body.String())
	}
	assertNoRealtimeEvent(t, failedUploadSocket)
	if _, err := fixture.runtime.Database().Exec(`DROP TRIGGER block_space_attachment_create`); err != nil {
		t.Fatal(err)
	}

	uploadSocket := dialTokenRealtime(t, server.URL, fixture.workspaceSlug, fixture.login.Token)
	defer uploadSocket.Close()
	attachment := decodeRuntimeAttachment(t, runtimeMultipartFileRequest(t, fixture.runtime, "/api/upload-file", filePath, map[string]string{"issue_id": fixture.issueID}, fixture.headers))
	assertRealtimeEvent(t, uploadSocket, "issue_attachments:changed", `"id":"`+attachment.ID+`"`)

	failedDeleteSocket := dialTokenRealtime(t, server.URL, fixture.workspaceSlug, fixture.login.Token)
	defer failedDeleteSocket.Close()
	if _, err := fixture.runtime.Database().Exec(`CREATE TRIGGER block_space_attachment_delete_event BEFORE DELETE ON space_assets BEGIN SELECT RAISE(ABORT,'blocked'); END`); err != nil {
		t.Fatal(err)
	}
	failedDelete := runtimeRequest(fixture.runtime, http.MethodDelete, "/api/attachments/"+attachment.ID, "", fixture.headers)
	if failedDelete.Code != http.StatusInternalServerError {
		t.Fatalf("blocked delete = %d %s", failedDelete.Code, failedDelete.Body.String())
	}
	assertNoRealtimeEvent(t, failedDeleteSocket)
	if _, err := fixture.runtime.Database().Exec(`DROP TRIGGER block_space_attachment_delete_event`); err != nil {
		t.Fatal(err)
	}

	deleteSocket := dialTokenRealtime(t, server.URL, fixture.workspaceSlug, fixture.login.Token)
	defer deleteSocket.Close()
	deleted := runtimeRequest(fixture.runtime, http.MethodDelete, "/api/attachments/"+attachment.ID, "", fixture.headers)
	if deleted.Code != http.StatusNoContent {
		t.Fatalf("delete = %d %s", deleted.Code, deleted.Body.String())
	}
	assertRealtimeEvent(t, deleteSocket, "issue_attachments:changed", `"attachments":[]`)
}

func TestSQLiteRuntimeIssueDeletionCleansOwnedSpaceAttachmentAndFile(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "issue-attachment-parent-delete.db")
	fixture := newCollaborationRuntimeFixture(t, databasePath, "issue-attachment-parent-delete", "issue-attachment-parent-delete@example.com")
	filePath := filepath.Join(t.TempDir(), "owned.txt")
	if err := os.WriteFile(filePath, []byte("owned by Issue"), 0o600); err != nil {
		t.Fatal(err)
	}
	attachment := decodeRuntimeAttachment(t, runtimeMultipartFileRequest(t, fixture.runtime, "/api/upload-file", filePath, map[string]string{"issue_id": fixture.issueID}, fixture.headers))
	deleted := runtimeRequest(fixture.runtime, http.MethodDelete, "/api/issues/"+fixture.issueID, "", fixture.headers)
	if deleted.Code != http.StatusNoContent {
		t.Fatalf("delete Issue = %d %s", deleted.Code, deleted.Body.String())
	}
	metadata := runtimeRequest(fixture.runtime, http.MethodGet, "/api/attachments/"+attachment.ID, "", fixture.headers)
	if metadata.Code != http.StatusNotFound {
		t.Fatalf("deleted Issue attachment remains readable = %d %s", metadata.Code, metadata.Body.String())
	}
	var assets, versions int
	if err := fixture.runtime.Database().QueryRow(`SELECT COUNT(*) FROM space_assets WHERE id=?`, attachment.ID).Scan(&assets); err != nil {
		t.Fatal(err)
	}
	if err := fixture.runtime.Database().QueryRow(`SELECT COUNT(*) FROM space_asset_versions WHERE asset_id=?`, attachment.ID).Scan(&versions); err != nil {
		t.Fatal(err)
	}
	if assets != 0 || versions != 0 || countRegularFiles(t, databasePath+".files") != 0 {
		t.Fatalf("deleted Issue attachment assets=%d versions=%d files=%d", assets, versions, countRegularFiles(t, databasePath+".files"))
	}
}

func TestSQLiteRuntimeCommentDeletionCleansOwnedSpaceAttachmentAndRollsBack(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "issue-attachment-comment-delete.db")
	fixture := newCollaborationRuntimeFixture(t, databasePath, "issue-attachment-comment-delete", "issue-attachment-comment-delete@example.com")
	filePath := filepath.Join(t.TempDir(), "comment-owned.txt")
	if err := os.WriteFile(filePath, []byte("owned by Comment"), 0o600); err != nil {
		t.Fatal(err)
	}
	attachment := decodeRuntimeAttachment(t, runtimeMultipartFileRequest(t, fixture.runtime, "/api/upload-file", filePath, map[string]string{"issue_id": fixture.issueID}, fixture.headers))
	created := runtimeRequest(fixture.runtime, http.MethodPost, "/api/issues/"+fixture.issueID+"/comments", `{"content":"Comment with attachment","attachment_ids":["`+attachment.ID+`"]}`, fixture.headers)
	if created.Code != http.StatusCreated {
		t.Fatalf("create attached Comment = %d %s", created.Code, created.Body.String())
	}
	var comment runtimeCommentResponse
	if err := json.Unmarshal(created.Body.Bytes(), &comment); err != nil || comment.ID == "" {
		t.Fatalf("decode attached Comment = %#v err=%v body=%s", comment, err, created.Body.String())
	}
	objectPath := onlyAttachmentObjectPath(t, databasePath+".files")
	if _, err := fixture.runtime.Database().Exec(`CREATE TRIGGER block_comment_attachment_delete BEFORE DELETE ON workspace_issue_comments WHEN OLD.id='` + comment.ID + `' BEGIN SELECT RAISE(ABORT,'blocked'); END`); err != nil {
		t.Fatal(err)
	}
	blocked := runtimeRequest(fixture.runtime, http.MethodDelete, "/api/comments/"+comment.ID, "", fixture.headers)
	if blocked.Code != http.StatusInternalServerError {
		t.Fatalf("blocked Comment delete = %d %s", blocked.Code, blocked.Body.String())
	}
	if _, err := fixture.runtime.Database().Exec(`DROP TRIGGER block_comment_attachment_delete`); err != nil {
		t.Fatal(err)
	}
	if metadata := runtimeRequest(fixture.runtime, http.MethodGet, "/api/attachments/"+attachment.ID, "", fixture.headers); metadata.Code != http.StatusOK {
		t.Fatalf("rolled back Comment attachment metadata = %d %s", metadata.Code, metadata.Body.String())
	}
	if _, err := os.Stat(objectPath); err != nil {
		t.Fatalf("rolled back Comment attachment object: %v", err)
	}
	deleted := runtimeRequest(fixture.runtime, http.MethodDelete, "/api/comments/"+comment.ID, "", fixture.headers)
	if deleted.Code != http.StatusNoContent {
		t.Fatalf("delete attached Comment = %d %s", deleted.Code, deleted.Body.String())
	}
	if metadata := runtimeRequest(fixture.runtime, http.MethodGet, "/api/attachments/"+attachment.ID, "", fixture.headers); metadata.Code != http.StatusNotFound {
		t.Fatalf("deleted Comment attachment remains readable = %d %s", metadata.Code, metadata.Body.String())
	}
	listed := decodeRuntimeAttachmentList(t, runtimeRequest(fixture.runtime, http.MethodGet, "/api/issues/"+fixture.issueID+"/attachments", "", fixture.headers))
	if len(listed) != 0 || countRegularFiles(t, databasePath+".files") != 0 {
		t.Fatalf("Comment attachment cleanup list=%#v files=%d", listed, countRegularFiles(t, databasePath+".files"))
	}
}

func TestSQLiteRuntimeCommentDeletionPreservesSharedAttachmentUntilLastReference(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "issue-attachment-comment-shared.db")
	fixture := newCollaborationRuntimeFixture(t, databasePath, "issue-attachment-comment-shared", "issue-attachment-comment-shared@example.com")
	filePath := filepath.Join(t.TempDir(), "comment-shared.txt")
	if err := os.WriteFile(filePath, []byte("shared by Comments"), 0o600); err != nil {
		t.Fatal(err)
	}
	attachment := decodeRuntimeAttachment(t, runtimeMultipartFileRequest(t, fixture.runtime, "/api/upload-file", filePath, map[string]string{"issue_id": fixture.issueID}, fixture.headers))
	createComment := func(content string) runtimeCommentResponse {
		t.Helper()
		response := runtimeRequest(fixture.runtime, http.MethodPost, "/api/issues/"+fixture.issueID+"/comments", `{"content":`+strconv.Quote(content)+`,"attachment_ids":["`+attachment.ID+`"]}`, fixture.headers)
		if response.Code != http.StatusCreated {
			t.Fatalf("create shared Comment = %d %s", response.Code, response.Body.String())
		}
		var comment runtimeCommentResponse
		if err := json.Unmarshal(response.Body.Bytes(), &comment); err != nil || comment.ID == "" {
			t.Fatalf("decode shared Comment = %#v err=%v body=%s", comment, err, response.Body.String())
		}
		return comment
	}
	first, second := createComment("first shared Comment"), createComment("second shared Comment")
	if deleted := runtimeRequest(fixture.runtime, http.MethodDelete, "/api/comments/"+first.ID, "", fixture.headers); deleted.Code != http.StatusNoContent {
		t.Fatalf("delete first shared Comment = %d %s", deleted.Code, deleted.Body.String())
	}
	if metadata := runtimeRequest(fixture.runtime, http.MethodGet, "/api/attachments/"+attachment.ID, "", fixture.headers); metadata.Code != http.StatusOK {
		t.Fatalf("shared Comment attachment was deleted early = %d %s", metadata.Code, metadata.Body.String())
	}
	listed := decodeRuntimeAttachmentList(t, runtimeRequest(fixture.runtime, http.MethodGet, "/api/issues/"+fixture.issueID+"/attachments", "", fixture.headers))
	if len(listed) != 1 || listed[0].ID != attachment.ID || listed[0].CommentID == nil || *listed[0].CommentID != second.ID {
		t.Fatalf("shared Comment attachment projection = %#v", listed)
	}
	if deleted := runtimeRequest(fixture.runtime, http.MethodDelete, "/api/comments/"+second.ID, "", fixture.headers); deleted.Code != http.StatusNoContent {
		t.Fatalf("delete final shared Comment = %d %s", deleted.Code, deleted.Body.String())
	}
	if metadata := runtimeRequest(fixture.runtime, http.MethodGet, "/api/attachments/"+attachment.ID, "", fixture.headers); metadata.Code != http.StatusNotFound {
		t.Fatalf("final shared Comment attachment remains = %d %s", metadata.Code, metadata.Body.String())
	}
	if count := countRegularFiles(t, databasePath+".files"); count != 0 {
		t.Fatalf("shared Comment attachment files = %d", count)
	}
}

func TestSQLiteRuntimeIssueDeletionRestoresAttachmentWhenTransactionRollsBack(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "issue-attachment-parent-rollback.db")
	fixture := newCollaborationRuntimeFixture(t, databasePath, "issue-attachment-parent-rollback", "issue-attachment-parent-rollback@example.com")
	filePath := filepath.Join(t.TempDir(), "rollback.txt")
	content := []byte("restore attachment with Issue transaction")
	if err := os.WriteFile(filePath, content, 0o600); err != nil {
		t.Fatal(err)
	}
	attachment := decodeRuntimeAttachment(t, runtimeMultipartFileRequest(t, fixture.runtime, "/api/upload-file", filePath, map[string]string{"issue_id": fixture.issueID}, fixture.headers))
	objectPath := onlyAttachmentObjectPath(t, databasePath+".files")
	if _, err := fixture.runtime.Database().Exec(`CREATE TRIGGER block_issue_attachment_parent_delete BEFORE DELETE ON workspace_issues WHEN OLD.id='` + fixture.issueID + `' BEGIN SELECT RAISE(ABORT,'blocked'); END`); err != nil {
		t.Fatal(err)
	}
	blocked := runtimeRequest(fixture.runtime, http.MethodDelete, "/api/issues/"+fixture.issueID, "", fixture.headers)
	if blocked.Code != http.StatusInternalServerError {
		t.Fatalf("blocked Issue delete = %d %s", blocked.Code, blocked.Body.String())
	}
	if _, err := fixture.runtime.Database().Exec(`DROP TRIGGER block_issue_attachment_parent_delete`); err != nil {
		t.Fatal(err)
	}
	metadata := runtimeRequest(fixture.runtime, http.MethodGet, "/api/attachments/"+attachment.ID, "", fixture.headers)
	if metadata.Code != http.StatusOK {
		t.Fatalf("rollback attachment metadata = %d %s", metadata.Code, metadata.Body.String())
	}
	readback, err := os.ReadFile(objectPath)
	if err != nil || !bytes.Equal(readback, content) {
		t.Fatalf("rollback attachment bytes err=%v body=%q", err, string(readback))
	}
	if retained := runtimeRequest(fixture.runtime, http.MethodGet, "/api/issues/"+fixture.issueID, "", fixture.headers); retained.Code != http.StatusOK {
		t.Fatalf("rollback Issue = %d %s", retained.Code, retained.Body.String())
	}
}

func TestSQLiteRuntimeIssueDeletionPreservesSharedAttachmentReferences(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "issue-attachment-shared.db")
	fixture := newCollaborationRuntimeFixture(t, databasePath, "issue-attachment-shared", "issue-attachment-shared@example.com")
	other := createRuntimeIssue(t, fixture.runtime, fixture.headers, "Retains shared attachment")
	filePath := filepath.Join(t.TempDir(), "shared.txt")
	if err := os.WriteFile(filePath, []byte("shared attachment"), 0o600); err != nil {
		t.Fatal(err)
	}
	attachment := decodeRuntimeAttachment(t, runtimeMultipartFileRequest(t, fixture.runtime, "/api/upload-file", filePath, map[string]string{"issue_id": fixture.issueID}, fixture.headers))
	bound := runtimeRequest(fixture.runtime, http.MethodPut, "/api/issues/"+other.ID, `{"attachment_ids":["`+attachment.ID+`"]}`, fixture.headers)
	if bound.Code != http.StatusOK {
		t.Fatalf("share attachment = %d %s", bound.Code, bound.Body.String())
	}
	deleted := runtimeRequest(fixture.runtime, http.MethodDelete, "/api/issues/"+fixture.issueID, "", fixture.headers)
	if deleted.Code != http.StatusNoContent {
		t.Fatalf("delete first Issue = %d %s", deleted.Code, deleted.Body.String())
	}
	listed := decodeRuntimeAttachmentList(t, runtimeRequest(fixture.runtime, http.MethodGet, "/api/issues/"+other.ID+"/attachments", "", fixture.headers))
	if len(listed) != 1 || listed[0].ID != attachment.ID || listed[0].IssueID == nil || *listed[0].IssueID != other.ID {
		t.Fatalf("retained shared attachment = %#v", listed)
	}
	if count := countRegularFiles(t, databasePath+".files"); count != 1 {
		t.Fatalf("shared attachment files = %d", count)
	}
}

func TestSQLiteRuntimeBatchIssueDeletionCleansOwnedSpaceAttachments(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "issue-attachment-batch-delete.db")
	fixture := newCollaborationRuntimeFixture(t, databasePath, "issue-attachment-batch-delete", "issue-attachment-batch-delete@example.com")
	second := createRuntimeIssue(t, fixture.runtime, fixture.headers, "Second attachment owner")
	writeAndUpload := func(name, issueID string) runtimeAttachmentResponse {
		t.Helper()
		filePath := filepath.Join(t.TempDir(), name)
		if err := os.WriteFile(filePath, []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
		return decodeRuntimeAttachment(t, runtimeMultipartFileRequest(t, fixture.runtime, "/api/upload-file", filePath, map[string]string{"issue_id": issueID}, fixture.headers))
	}
	firstAttachment := writeAndUpload("batch-first.txt", fixture.issueID)
	secondAttachment := writeAndUpload("batch-second.txt", second.ID)
	if _, err := fixture.runtime.Database().Exec(`CREATE TRIGGER block_issue_attachment_batch_delete BEFORE DELETE ON workspace_issues WHEN OLD.id='` + second.ID + `' BEGIN SELECT RAISE(ABORT,'blocked'); END`); err != nil {
		t.Fatal(err)
	}
	blocked := runtimeRequest(fixture.runtime, http.MethodPost, "/api/issues/batch-delete", `{"issue_ids":["`+fixture.issueID+`","`+second.Identifier+`"]}`, fixture.headers)
	if blocked.Code != http.StatusInternalServerError {
		t.Fatalf("blocked batch delete = %d %s", blocked.Code, blocked.Body.String())
	}
	if _, err := fixture.runtime.Database().Exec(`DROP TRIGGER block_issue_attachment_batch_delete`); err != nil {
		t.Fatal(err)
	}
	for _, attachmentID := range []string{firstAttachment.ID, secondAttachment.ID} {
		metadata := runtimeRequest(fixture.runtime, http.MethodGet, "/api/attachments/"+attachmentID, "", fixture.headers)
		if metadata.Code != http.StatusOK {
			t.Fatalf("rolled back batch attachment %s = %d %s", attachmentID, metadata.Code, metadata.Body.String())
		}
	}
	if count := countRegularFiles(t, databasePath+".files"); count != 2 {
		t.Fatalf("rolled back batch attachment files = %d", count)
	}
	deleted := runtimeRequest(fixture.runtime, http.MethodPost, "/api/issues/batch-delete", `{"issue_ids":["`+fixture.issueID+`","`+second.Identifier+`"]}`, fixture.headers)
	if deleted.Code != http.StatusOK || !containsJSON(deleted.Body.Bytes(), `"deleted":2`) {
		t.Fatalf("batch delete Issues = %d %s", deleted.Code, deleted.Body.String())
	}
	for _, attachmentID := range []string{firstAttachment.ID, secondAttachment.ID} {
		metadata := runtimeRequest(fixture.runtime, http.MethodGet, "/api/attachments/"+attachmentID, "", fixture.headers)
		if metadata.Code != http.StatusNotFound {
			t.Fatalf("batch deleted attachment %s = %d %s", attachmentID, metadata.Code, metadata.Body.String())
		}
	}
	if count := countRegularFiles(t, databasePath+".files"); count != 0 {
		t.Fatalf("batch deleted attachment files = %d", count)
	}
}

func onlyAttachmentObjectPath(t *testing.T, root string) string {
	t.Helper()
	var paths []string
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			paths = append(paths, path)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(paths) != 1 {
		t.Fatalf("attachment object paths = %v", paths)
	}
	return paths[0]
}

func countRegularFiles(t *testing.T, root string) int {
	t.Helper()
	count := 0
	if err := filepath.WalkDir(root, func(_ string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			count++
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return count
}

func decodeRuntimeAttachment(t *testing.T, response *httptest.ResponseRecorder) runtimeAttachmentResponse {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("attachment response = %d %s", response.Code, response.Body.String())
	}
	var value runtimeAttachmentResponse
	if err := json.Unmarshal(response.Body.Bytes(), &value); err != nil || value.ID == "" {
		t.Fatalf("decode attachment: %v body=%s", err, response.Body.String())
	}
	return value
}

func decodeRuntimeAttachmentList(t *testing.T, response *httptest.ResponseRecorder) []runtimeAttachmentResponse {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("attachment list = %d %s", response.Code, response.Body.String())
	}
	var values []runtimeAttachmentResponse
	if err := json.Unmarshal(response.Body.Bytes(), &values); err != nil {
		t.Fatalf("decode attachment list: %v body=%s", err, response.Body.String())
	}
	return values
}

func runtimeMultipartFileRequest(
	t *testing.T,
	runtime *Runtime,
	path string,
	filePath string,
	fields map[string]string,
	headers map[string]string,
) *httptest.ResponseRecorder {
	t.Helper()
	response, err := runtimeMultipartFileRequestE(runtime, path, filePath, fields, headers)
	if err != nil {
		t.Fatal(err)
	}
	return response
}

func runtimeMultipartFileRequestE(
	runtime *Runtime,
	path string,
	filePath string,
	fields map[string]string,
	headers map[string]string,
) (*httptest.ResponseRecorder, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, err
	}
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	request := httptest.NewRequest(http.MethodPost, path, &body)
	for key, value := range headers {
		if key != "Content-Type" {
			request.Header.Set(key, value)
		}
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()
	runtime.HTTPServer().ServeHTTP(response, request)
	return response, nil
}
