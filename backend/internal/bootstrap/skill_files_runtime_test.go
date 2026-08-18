package bootstrap

import (
	"encoding/json"
	"net/http"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/hvritual/workspace/internal/modules/auth"
	workspacecontract "github.com/hvritual/workspace/internal/modules/workspace/contract"
)

func TestSQLiteRuntimeSkillFilesCreateImmutableVersionsAndDownload(t *testing.T) {
	runtime := newRuntimeForConfig(t, Config{
		Name: "backend-test", Version: "test",
		HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath: filepath.Join(t.TempDir(), "skill-files-runtime.db"), AttachmentRoot: filepath.Join(t.TempDir(), "objects"),
		WorkspaceDependencies: FailClosedWorkspaceDependencies(),
		LocalAuth:             auth.LocalAuthConfig{VerificationCode: "888888"},
		RoadmapCapabilityProvider: runtimeCapabilityProviderStub{
			workspacecontract.PermissionSkillImport: true,
		},
	})
	owner := verifyRuntimeLogin(t, runtime, "skill-files-owner@example.com")
	if response := runtimeRequest(runtime, http.MethodPost, "/api/workspaces", `{"name":"Skill Files","slug":"skill-files"}`, map[string]string{
		"Authorization": "Bearer " + owner.Token, "Content-Type": "application/json",
	}); response.Code != http.StatusCreated {
		t.Fatalf("create workspace = %d %s", response.Code, response.Body.String())
	}
	headers := collaborationHeaders(owner.Token, "skill-files")
	created := runtimeRequest(runtime, http.MethodPost, "/api/skills", `{"name":"File helper"}`, headers)
	var skill struct {
		ID        string `json:"id"`
		VersionID string `json:"version_id"`
	}
	if created.Code != http.StatusCreated || json.Unmarshal(created.Body.Bytes(), &skill) != nil {
		t.Fatalf("create Skill = %d %s", created.Code, created.Body.String())
	}
	missingManifest := runtimeRequest(runtime, http.MethodPost, "/api/skills/"+skill.ID+"/files", `{"path":"scripts/run.py","content":"print('unsafe')\n","expected_revision":1}`, headers)
	if missingManifest.Code != http.StatusBadRequest {
		t.Fatalf("add script before SKILL.md = %d %s", missingManifest.Code, missingManifest.Body.String())
	}
	metadataWithoutManifest := runtimeRequest(runtime, http.MethodPut, "/api/skills/"+skill.ID, `{"description":"unsafe metadata version","expected_revision":1}`, headers)
	if metadataWithoutManifest.Code != http.StatusBadRequest {
		t.Fatalf("metadata version before SKILL.md = %d %s", metadataWithoutManifest.Code, metadataWithoutManifest.Body.String())
	}

	mainFile := runtimeRequest(runtime, http.MethodPost, "/api/skills/"+skill.ID+"/files", `{"path":"SKILL.md","content":"# File helper","expected_revision":1}`, headers)
	var mainVersion struct {
		VersionID string `json:"version_id"`
		Revision  int64  `json:"revision"`
	}
	if mainFile.Code != http.StatusCreated || json.Unmarshal(mainFile.Body.Bytes(), &mainVersion) != nil || mainVersion.Revision != 2 {
		t.Fatalf("add SKILL.md = %d %s", mainFile.Code, mainFile.Body.String())
	}
	script := runtimeRequest(runtime, http.MethodPost, "/api/skills/"+skill.ID+"/files", `{"path":"scripts/run.py","content":"print('ok')\n","expected_revision":2}`, headers)
	var scriptVersion struct {
		VersionID string `json:"version_id"`
		Revision  int64  `json:"revision"`
	}
	if script.Code != http.StatusCreated || json.Unmarshal(script.Body.Bytes(), &scriptVersion) != nil || scriptVersion.Revision != 3 {
		t.Fatalf("add script = %d %s", script.Code, script.Body.String())
	}
	metadata := runtimeRequest(runtime, http.MethodPut, "/api/skills/"+skill.ID, `{"description":"metadata with manifest","expected_revision":3}`, headers)
	var metadataVersion struct {
		VersionID string `json:"version_id"`
		Revision  int64  `json:"revision"`
	}
	if metadata.Code != http.StatusOK || json.Unmarshal(metadata.Body.Bytes(), &metadataVersion) != nil || metadataVersion.Revision != 4 {
		t.Fatalf("metadata version with manifest = %d %s", metadata.Code, metadata.Body.String())
	}
	metadataFiles := runtimeRequest(runtime, http.MethodGet, "/api/skills/"+skill.ID+"/files?version_id="+url.QueryEscape(metadataVersion.VersionID), "", headers)
	if metadataFiles.Code != http.StatusOK || !containsJSON(metadataFiles.Body.Bytes(), `"path":"SKILL.md"`, `"path":"scripts/run.py"`) {
		t.Fatalf("metadata manifest copy = %d %s", metadataFiles.Code, metadataFiles.Body.String())
	}

	oldList := runtimeRequest(runtime, http.MethodGet, "/api/skills/"+skill.ID+"/files?version_id="+url.QueryEscape(mainVersion.VersionID), "", headers)
	currentList := runtimeRequest(runtime, http.MethodGet, "/api/skills/"+skill.ID+"/files?version_id="+url.QueryEscape(scriptVersion.VersionID), "", headers)
	if oldList.Code != http.StatusOK || !containsJSON(oldList.Body.Bytes(), `"path":"SKILL.md"`) || containsJSON(oldList.Body.Bytes(), `scripts/run.py`) {
		t.Fatalf("old manifest = %d %s", oldList.Code, oldList.Body.String())
	}
	if currentList.Code != http.StatusOK || !containsJSON(currentList.Body.Bytes(), `"path":"SKILL.md"`, `"path":"scripts/run.py"`) {
		t.Fatalf("current manifest = %d %s", currentList.Code, currentList.Body.String())
	}

	encodedPath := url.PathEscape("scripts/run.py")
	read := runtimeRequest(runtime, http.MethodGet, "/api/skills/"+skill.ID+"/files/"+encodedPath+"?version_id="+url.QueryEscape(scriptVersion.VersionID), "", headers)
	if read.Code != http.StatusOK || !containsJSON(read.Body.Bytes(), `"content":"print('ok')\n"`, `"checksum":"`) {
		t.Fatalf("read = %d %s", read.Code, read.Body.String())
	}
	download := runtimeRequest(runtime, http.MethodGet, "/api/skills/"+skill.ID+"/files/"+encodedPath+"?version_id="+url.QueryEscape(scriptVersion.VersionID)+"&download=true", "", headers)
	if download.Code != http.StatusOK || download.Body.String() != "print('ok')\n" || download.Header().Get("X-Content-Type-Options") != "nosniff" || download.Header().Get("ETag") == "" {
		t.Fatalf("download = %d headers=%v body=%q", download.Code, download.Header(), download.Body.String())
	}

	stale := runtimeRequest(runtime, http.MethodPut, "/api/skills/"+skill.ID+"/files/"+encodedPath, `{"content":"new","expected_revision":2}`, headers)
	if stale.Code != http.StatusConflict || !containsJSON(stale.Body.Bytes(), `"current_revision":4`) {
		t.Fatalf("stale replace = %d %s", stale.Code, stale.Body.String())
	}
	deleteMain := runtimeRequest(runtime, http.MethodDelete, "/api/skills/"+skill.ID+"/files/"+url.PathEscape("SKILL.md"), `{"expected_revision":4}`, headers)
	if deleteMain.Code != http.StatusBadRequest {
		t.Fatalf("delete SKILL.md = %d %s", deleteMain.Code, deleteMain.Body.String())
	}
}
