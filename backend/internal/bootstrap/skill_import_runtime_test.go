package bootstrap

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/hvritual/workspace/internal/modules/auth"
	workspacecontract "github.com/hvritual/workspace/internal/modules/workspace/contract"
)

func TestSQLiteRuntimeSkillArchivePreviewImportIdempotencyAndNewVersion(t *testing.T) {
	runtime := newRuntimeForConfig(t, Config{
		Name: "backend-test", Version: "test", HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath: filepath.Join(t.TempDir(), "skill-import-runtime.db"), AttachmentRoot: filepath.Join(t.TempDir(), "objects"),
		WorkspaceDependencies: FailClosedWorkspaceDependencies(), LocalAuth: auth.LocalAuthConfig{VerificationCode: "888888"},
		RoadmapCapabilityProvider: runtimeCapabilityProviderStub{workspacecontract.PermissionSkillImport: true},
	})
	owner := verifyRuntimeLogin(t, runtime, "skill-import-owner@example.com")
	if response := runtimeRequest(runtime, http.MethodPost, "/api/workspaces", `{"name":"Skill Import","slug":"skill-import"}`, map[string]string{"Authorization": "Bearer " + owner.Token, "Content-Type": "application/json"}); response.Code != http.StatusCreated {
		t.Fatalf("create workspace = %d %s", response.Code, response.Body.String())
	}
	headers := collaborationHeaders(owner.Token, "skill-import")
	archive := runtimeSkillArchive(t, map[string]string{"SKILL.md": "---\nname: Imported Helper\ndescription: v1\n---\n# Imported", "scripts/run.py": "print('v1')\n"})
	preview := runtimeMultipartSkillImport(t, runtime, "/api/skills/import/preview", archive, nil, headers)
	var previewBody struct {
		Token    string `json:"preview_token"`
		Checksum string `json:"checksum"`
	}
	if preview.Code != http.StatusOK || json.Unmarshal(preview.Body.Bytes(), &previewBody) != nil || previewBody.Token == "" || previewBody.Checksum == "" || bytes.Contains(preview.Body.Bytes(), []byte("print('v1')")) {
		t.Fatalf("preview = %d %s", preview.Code, preview.Body.String())
	}
	fields := map[string]string{"preview_token": previewBody.Token, "conflict_mode": "new_version"}
	commitHeaders := cloneHeaders(headers)
	commitHeaders["Idempotency-Key"] = "import-1"
	imported := runtimeMultipartSkillImport(t, runtime, "/api/skills/import", archive, fields, commitHeaders)
	var first struct {
		ID        string `json:"id"`
		VersionID string `json:"version_id"`
		Revision  int64  `json:"revision"`
	}
	if imported.Code != http.StatusCreated || json.Unmarshal(imported.Body.Bytes(), &first) != nil || first.ID == "" || first.Revision != 1 {
		t.Fatalf("import = %d %s", imported.Code, imported.Body.String())
	}
	replay := runtimeMultipartSkillImport(t, runtime, "/api/skills/import", archive, fields, commitHeaders)
	if replay.Code != http.StatusCreated || !containsJSON(replay.Body.Bytes(), `"id":"`+first.ID+`"`, `"version_id":"`+first.VersionID+`"`) {
		t.Fatalf("replay = %d %s", replay.Code, replay.Body.String())
	}
	changed := runtimeSkillArchive(t, map[string]string{"SKILL.md": "# Changed"})
	conflict := runtimeMultipartSkillImport(t, runtime, "/api/skills/import", changed, fields, commitHeaders)
	if conflict.Code != http.StatusConflict {
		t.Fatalf("idempotency conflict = %d %s", conflict.Code, conflict.Body.String())
	}

	secondArchive := runtimeSkillArchive(t, map[string]string{"SKILL.md": "---\nname: Imported Helper\ndescription: v2\n---\n# Imported", "scripts/run.py": "print('v2')\n"})
	secondPreview := runtimeMultipartSkillImport(t, runtime, "/api/skills/import/preview", secondArchive, nil, headers)
	if secondPreview.Code != http.StatusOK || json.Unmarshal(secondPreview.Body.Bytes(), &previewBody) != nil {
		t.Fatalf("second preview = %d %s", secondPreview.Code, secondPreview.Body.String())
	}
	secondHeaders := cloneHeaders(headers)
	secondHeaders["Idempotency-Key"] = "import-2"
	second := runtimeMultipartSkillImport(t, runtime, "/api/skills/import", secondArchive, map[string]string{"preview_token": previewBody.Token}, secondHeaders)
	if second.Code != http.StatusCreated || !containsJSON(second.Body.Bytes(), `"id":"`+first.ID+`"`, `"version":"2"`, `"revision":2`, `"description":"v2"`) {
		t.Fatalf("second import = %d %s", second.Code, second.Body.String())
	}
	oldFiles := runtimeRequest(runtime, http.MethodGet, "/api/skills/"+first.ID+"/files?version_id="+first.VersionID, "", headers)
	if oldFiles.Code != http.StatusOK || !containsJSON(oldFiles.Body.Bytes(), `"checksum":"`) {
		t.Fatalf("retained old files = %d %s", oldFiles.Code, oldFiles.Body.String())
	}
	replacementArchive := runtimeSkillArchive(t, map[string]string{"SKILL.md": "---\nname: Imported Helper\ndescription: replacement\n---\n# Imported", "scripts/run.py": "print('replacement')\n"})
	replacementPreview := runtimeMultipartSkillImport(t, runtime, "/api/skills/import/preview", replacementArchive, nil, headers)
	if replacementPreview.Code != http.StatusOK || json.Unmarshal(replacementPreview.Body.Bytes(), &previewBody) != nil {
		t.Fatalf("replacement preview = %d %s", replacementPreview.Code, replacementPreview.Body.String())
	}
	replacementHeaders := cloneHeaders(headers)
	replacementHeaders["Idempotency-Key"] = "import-3"
	replacement := runtimeMultipartSkillImport(t, runtime, "/api/skills/import", replacementArchive, map[string]string{
		"preview_token": previewBody.Token, "conflict_mode": "replace", "expected_revision": "2",
	}, replacementHeaders)
	if replacement.Code != http.StatusCreated || !containsJSON(replacement.Body.Bytes(), `"version":"3"`, `"revision":3`, `"description":"replacement"`) {
		t.Fatalf("replacement = %d %s", replacement.Code, replacement.Body.String())
	}
	var replacedStatus string
	if err := runtime.Database().QueryRow(`SELECT status FROM system_skill_versions WHERE skill_id=? AND version_number=2`, first.ID).Scan(&replacedStatus); err != nil || replacedStatus != "archived" {
		t.Fatalf("replaced draft status = %q, %v", replacedStatus, err)
	}
	for table, want := range map[string]int{"system_skills": 1, "system_skill_versions": 3, "system_skill_file_manifests": 6, "system_skill_import_idempotency": 3, "space_skill_objects": 6} {
		var count int
		if err := runtime.Database().QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil || count != want {
			t.Fatalf("%s count = %d, %v; want %d", table, count, err, want)
		}
	}
}

func TestSQLiteRuntimeSkillArchiveRejectionLeavesNoImportState(t *testing.T) {
	runtime := newRuntimeForConfig(t, Config{
		Name: "backend-test", Version: "test", HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath: filepath.Join(t.TempDir(), "skill-import-reject.db"), WorkspaceDependencies: FailClosedWorkspaceDependencies(),
		LocalAuth: auth.LocalAuthConfig{VerificationCode: "888888"}, RoadmapCapabilityProvider: runtimeCapabilityProviderStub{workspacecontract.PermissionSkillImport: true},
	})
	owner := verifyRuntimeLogin(t, runtime, "skill-import-reject@example.com")
	_ = runtimeRequest(runtime, http.MethodPost, "/api/workspaces", `{"name":"Reject","slug":"skill-import-reject"}`, map[string]string{"Authorization": "Bearer " + owner.Token, "Content-Type": "application/json"})
	archive := runtimeSkillArchive(t, map[string]string{"SKILL.md": "# Safe", "../escape": "bad"})
	response := runtimeMultipartSkillImport(t, runtime, "/api/skills/import/preview", archive, nil, collaborationHeaders(owner.Token, "skill-import-reject"))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("rejection = %d %s", response.Code, response.Body.String())
	}
	for _, table := range []string{"system_skill_import_previews", "system_skill_file_manifests", "space_skill_objects"} {
		var count int
		if err := runtime.Database().QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil || count != 0 {
			t.Fatalf("%s count = %d, %v", table, count, err)
		}
	}
}

func TestSQLiteRuntimeSkillImportFailureRollsBackMetadataObjectsBindingAndPreview(t *testing.T) {
	runtime := newRuntimeForConfig(t, Config{
		Name: "backend-test", Version: "test", HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath: filepath.Join(t.TempDir(), "skill-import-rollback.db"), WorkspaceDependencies: FailClosedWorkspaceDependencies(),
		LocalAuth: auth.LocalAuthConfig{VerificationCode: "888888"}, RoadmapCapabilityProvider: runtimeCapabilityProviderStub{workspacecontract.PermissionSkillImport: true},
	})
	owner := verifyRuntimeLogin(t, runtime, "skill-import-rollback@example.com")
	_ = runtimeRequest(runtime, http.MethodPost, "/api/workspaces", `{"name":"Rollback","slug":"skill-import-rollback"}`, map[string]string{"Authorization": "Bearer " + owner.Token, "Content-Type": "application/json"})
	headers := collaborationHeaders(owner.Token, "skill-import-rollback")
	archive := runtimeSkillArchive(t, map[string]string{"SKILL.md": "# Rollback", "note.txt": "body"})
	preview := runtimeMultipartSkillImport(t, runtime, "/api/skills/import/preview", archive, nil, headers)
	var body struct {
		Token string `json:"preview_token"`
	}
	if preview.Code != http.StatusOK || json.Unmarshal(preview.Body.Bytes(), &body) != nil {
		t.Fatalf("preview = %d %s", preview.Code, preview.Body.String())
	}
	if _, err := runtime.Database().Exec(`CREATE TRIGGER reject_import_audit BEFORE INSERT ON system_skill_audit BEGIN SELECT RAISE(ABORT,'reject import audit'); END`); err != nil {
		t.Fatal(err)
	}
	commitHeaders := cloneHeaders(headers)
	commitHeaders["Idempotency-Key"] = "rollback-import"
	failed := runtimeMultipartSkillImport(t, runtime, "/api/skills/import", archive, map[string]string{"preview_token": body.Token}, commitHeaders)
	if failed.Code != http.StatusInternalServerError {
		t.Fatalf("failed import = %d %s", failed.Code, failed.Body.String())
	}
	for _, table := range []string{"system_skills", "system_skill_versions", "system_skill_file_manifests", "system_skill_audit", "workspace_skill_bindings", "system_skill_import_previews", "system_skill_import_idempotency", "space_skill_objects"} {
		var count int
		if err := runtime.Database().QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil || count != 0 {
			t.Fatalf("%s count = %d, %v; want 0", table, count, err)
		}
	}
}

func TestSQLiteRuntimeSkillObjectReconciliationRestoresReferencesAndRemovesOrphans(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "skill-import-reconcile.db")
	config := Config{
		Name: "backend-test", Version: "test", HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath: databasePath, WorkspaceDependencies: FailClosedWorkspaceDependencies(),
		LocalAuth: auth.LocalAuthConfig{VerificationCode: "888888"}, RoadmapCapabilityProvider: runtimeCapabilityProviderStub{workspacecontract.PermissionSkillImport: true},
	}
	runtime := newRuntimeForConfig(t, config)
	owner := verifyRuntimeLogin(t, runtime, "skill-import-reconcile@example.com")
	_ = runtimeRequest(runtime, http.MethodPost, "/api/workspaces", `{"name":"Reconcile","slug":"skill-import-reconcile"}`, map[string]string{"Authorization": "Bearer " + owner.Token, "Content-Type": "application/json"})
	headers := collaborationHeaders(owner.Token, "skill-import-reconcile")
	archive := runtimeSkillArchive(t, map[string]string{"SKILL.md": "# Reconcile"})
	preview := runtimeMultipartSkillImport(t, runtime, "/api/skills/import/preview", archive, nil, headers)
	var previewBody struct {
		Token string `json:"preview_token"`
	}
	_ = json.Unmarshal(preview.Body.Bytes(), &previewBody)
	commitHeaders := cloneHeaders(headers)
	commitHeaders["Idempotency-Key"] = "reconcile-import"
	imported := runtimeMultipartSkillImport(t, runtime, "/api/skills/import", archive, map[string]string{"preview_token": previewBody.Token}, commitHeaders)
	var skill struct {
		ID        string `json:"id"`
		VersionID string `json:"version_id"`
	}
	if imported.Code != http.StatusCreated || json.Unmarshal(imported.Body.Bytes(), &skill) != nil {
		t.Fatalf("import = %d %s", imported.Code, imported.Body.String())
	}
	var referencedID string
	if err := runtime.Database().QueryRow(`SELECT space_object_id FROM system_skill_file_manifests WHERE skill_id=?`, skill.ID).Scan(&referencedID); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Database().Exec(`UPDATE space_skill_objects SET state='quarantined',committed_at=NULL WHERE id=?;
		INSERT INTO space_skill_objects(id,workspace_id,object_key,media_type,size_bytes,checksum,content,state,created_at) VALUES('orphan','workspace-orphan','skill/workspace-orphan/orphan','text/plain',6,'x',X'6F727068616E','quarantined','2026-08-18T00:00:00Z')`, referencedID); err != nil {
		t.Fatal(err)
	}
	if err := runtime.Close(); err != nil {
		t.Fatal(err)
	}
	restarted := newRuntimeForConfig(t, config)
	var state string
	if err := restarted.Database().QueryRow(`SELECT state FROM space_skill_objects WHERE id=?`, referencedID).Scan(&state); err != nil || state != "committed" {
		t.Fatalf("referenced state = %q, %v", state, err)
	}
	var orphanCount int
	if err := restarted.Database().QueryRow(`SELECT COUNT(*) FROM space_skill_objects WHERE id='orphan'`).Scan(&orphanCount); err != nil || orphanCount != 0 {
		t.Fatalf("orphan count = %d, %v", orphanCount, err)
	}
	listed := runtimeRequest(restarted, http.MethodGet, "/api/skills/"+skill.ID+"/files?version_id="+skill.VersionID, "", headers)
	if listed.Code != http.StatusOK || !containsJSON(listed.Body.Bytes(), `"path":"SKILL.md"`) {
		t.Fatalf("reconciled files = %d %s", listed.Code, listed.Body.String())
	}
}

func runtimeSkillArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var body bytes.Buffer
	writer := zip.NewWriter(&body)
	for name, content := range files {
		file, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return body.Bytes()
}

func runtimeMultipartSkillImport(t *testing.T, runtime *Runtime, target string, archive []byte, fields, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	file, err := writer.CreateFormFile("file", "skill.zip")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write(archive); err != nil {
		t.Fatal(err)
	}
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, target, &body)
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()
	runtime.HTTPServer().ServeHTTP(response, request)
	return response
}

func cloneHeaders(source map[string]string) map[string]string {
	result := make(map[string]string, len(source)+1)
	for key, value := range source {
		result[key] = value
	}
	return result
}
