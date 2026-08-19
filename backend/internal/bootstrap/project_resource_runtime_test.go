package bootstrap

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/hvritual/workspace/internal/modules/auth"
)

type runtimeProjectResource struct {
	ID           string `json:"id"`
	ResourceType string `json:"resource_type"`
	ResourceRef  struct {
		URL string `json:"url"`
		Ref string `json:"ref"`
	} `json:"resource_ref"`
	Status     string `json:"status"`
	Revision   int64  `json:"revision"`
	Connection struct {
		State          string `json:"state"`
		DiagnosticCode string `json:"diagnostic_code"`
	} `json:"connection"`
}

type runtimeProjectResourceList struct {
	Resources []runtimeProjectResource `json:"resources"`
	Total     int                      `json:"total"`
	Revision  int64                    `json:"revision"`
}

func TestSQLiteRuntimeProjectResourceInstalledJourneyAndRestart(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "project-resource-runtime.db")
	config := Config{
		Name: "backend-test", Version: "test", HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath: databasePath, WorkspaceDependencies: FailClosedWorkspaceDependencies(),
		LocalAuth: auth.LocalAuthConfig{VerificationCode: "888888"},
	}
	runtime := newRuntimeForConfig(t, config)
	owner := verifyRuntimeLogin(t, runtime, "project-resource-owner@example.com")
	workspaceResponse := runtimeRequest(runtime, http.MethodPost, "/api/workspaces", `{"name":"Resources","slug":"project-resources"}`, map[string]string{
		"Authorization": "Bearer " + owner.Token, "Content-Type": "application/json",
	})
	var workspace struct {
		ID string `json:"id"`
	}
	if workspaceResponse.Code != http.StatusCreated || json.Unmarshal(workspaceResponse.Body.Bytes(), &workspace) != nil || workspace.ID == "" {
		t.Fatalf("create workspace = %d %s", workspaceResponse.Code, workspaceResponse.Body.String())
	}
	ownerHeaders := collaborationHeaders(owner.Token, "project-resources")
	projectResponse := runtimeRequest(runtime, http.MethodPost, "/api/projects", `{
		"title":"Runtime resources",
		"resources":[
			{"resource_type":"github_repo","resource_ref":{"url":"git@github.com:Acme/Runtime.git","ref":"main"},"label":"Code"},
			{"resource_type":"url","resource_ref":{"url":"https://docs.example.com/runtime/"},"label":"Docs"}
		]
	}`, ownerHeaders)
	var project struct {
		ID            string `json:"id"`
		ResourceCount int    `json:"resource_count"`
	}
	if projectResponse.Code != http.StatusCreated || json.Unmarshal(projectResponse.Body.Bytes(), &project) != nil || project.ID == "" || project.ResourceCount != 2 {
		t.Fatalf("create project = %d %s", projectResponse.Code, projectResponse.Body.String())
	}

	listed := runtimeProjectResourceRequest(t, runtime, http.MethodGet, "/api/projects/"+project.ID+"/resources?include_archived=true", "", ownerHeaders, http.StatusOK)
	if listed.Total != 2 || listed.Revision != 1 || len(listed.Resources) != 2 ||
		listed.Resources[0].ResourceRef.URL != "https://github.com/acme/runtime" || listed.Resources[0].ResourceRef.Ref != "main" ||
		listed.Resources[1].ResourceRef.URL != "https://docs.example.com/runtime" {
		t.Fatalf("initial resources = %#v", listed)
	}

	createHeaders := collaborationHeaders(owner.Token, "project-resources")
	createHeaders["Idempotency-Key"] = "project-resource-runtime-create-1"
	createBody := `{"resource_type":"url","resource_ref":{"url":"https://example.com/runbook"},"label":"Runbook"}`
	createdResponse := runtimeRequest(runtime, http.MethodPost, "/api/projects/"+project.ID+"/resources", createBody, createHeaders)
	var created runtimeProjectResource
	if createdResponse.Code != http.StatusCreated || json.Unmarshal(createdResponse.Body.Bytes(), &created) != nil || created.ID == "" || created.Revision != 2 {
		t.Fatalf("create resource = %d %s", createdResponse.Code, createdResponse.Body.String())
	}
	replayResponse := runtimeRequest(runtime, http.MethodPost, "/api/projects/"+project.ID+"/resources", createBody, createHeaders)
	var replay runtimeProjectResource
	if replayResponse.Code != http.StatusCreated || json.Unmarshal(replayResponse.Body.Bytes(), &replay) != nil || replay.ID != created.ID || replay.Revision != created.Revision {
		t.Fatalf("replay resource = %d %s", replayResponse.Code, replayResponse.Body.String())
	}

	refreshResponse := runtimeRequest(runtime, http.MethodPut, "/api/projects/"+project.ID+"/resources/"+created.ID, `{"action":"refresh","expected_revision":2}`, ownerHeaders)
	var refreshed runtimeProjectResource
	if refreshResponse.Code != http.StatusOK || json.Unmarshal(refreshResponse.Body.Bytes(), &refreshed) != nil || refreshed.Revision != 3 ||
		refreshed.Connection.State != "unavailable" || refreshed.Connection.DiagnosticCode != "connection_not_configured" {
		t.Fatalf("refresh resource = %d %s", refreshResponse.Code, refreshResponse.Body.String())
	}
	stale := runtimeRequest(runtime, http.MethodPut, "/api/projects/"+project.ID+"/resources/"+created.ID, `{"action":"update","expected_revision":2,"label":"Stale"}`, ownerHeaders)
	var conflict struct {
		Code            string `json:"code"`
		CurrentRevision int64  `json:"current_revision"`
	}
	if stale.Code != http.StatusConflict || json.Unmarshal(stale.Body.Bytes(), &conflict) != nil || conflict.Code != "revision_conflict" || conflict.CurrentRevision != 3 {
		t.Fatalf("stale resource update = %d %s", stale.Code, stale.Body.String())
	}

	member := verifyRuntimeLogin(t, runtime, "project-resource-member@example.com")
	memberID := "project-resource-member-row"
	if _, err := runtime.Database().Exec(`INSERT INTO auth_members(id,workspace_id,user_id,role,created_at) VALUES(?,?,?,?,?)`, memberID, workspace.ID, member.UserID, "member", "2026-08-19T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	memberHeaders := collaborationHeaders(member.Token, "project-resources")
	if response := runtimeRequest(runtime, http.MethodGet, "/api/projects/"+project.ID+"/resources", "", memberHeaders); response.Code != http.StatusOK {
		t.Fatalf("member read = %d %s", response.Code, response.Body.String())
	}
	memberCreateHeaders := collaborationHeaders(member.Token, "project-resources")
	memberCreateHeaders["Idempotency-Key"] = "member-denied"
	if response := runtimeRequest(runtime, http.MethodPost, "/api/projects/"+project.ID+"/resources", `{"resource_type":"url","resource_ref":{"url":"https://example.com/denied"}}`, memberCreateHeaders); response.Code != http.StatusForbidden {
		t.Fatalf("member manage before lead = %d %s", response.Code, response.Body.String())
	}
	if response := runtimeRequest(runtime, http.MethodPut, "/api/projects/"+project.ID, `{"lead_type":"member","lead_id":"`+member.UserID+`"}`, ownerHeaders); response.Code != http.StatusOK {
		t.Fatalf("assign project lead = %d %s", response.Code, response.Body.String())
	}
	memberCreateHeaders["Idempotency-Key"] = "member-lead-create"
	if response := runtimeRequest(runtime, http.MethodPost, "/api/projects/"+project.ID+"/resources", `{"resource_type":"url","resource_ref":{"url":"https://example.com/lead"}}`, memberCreateHeaders); response.Code != http.StatusCreated {
		t.Fatalf("project lead manage = %d %s", response.Code, response.Body.String())
	}

	archiveResponse := runtimeRequest(runtime, http.MethodDelete, "/api/projects/"+project.ID+"/resources/"+created.ID, `{"expected_revision":4}`, ownerHeaders)
	if archiveResponse.Code != http.StatusNoContent || archiveResponse.Body.Len() != 0 {
		t.Fatalf("archive resource = %d %s", archiveResponse.Code, archiveResponse.Body.String())
	}
	active := runtimeProjectResourceRequest(t, runtime, http.MethodGet, "/api/projects/"+project.ID+"/resources", "", ownerHeaders, http.StatusOK)
	if active.Total != 3 || active.Revision != 5 {
		t.Fatalf("active resources after archive = %#v", active)
	}

	if err := runtime.Close(); err != nil {
		t.Fatal(err)
	}
	restarted := newRuntimeForConfig(t, config)
	persisted := runtimeProjectResourceRequest(t, restarted, http.MethodGet, "/api/projects/"+project.ID+"/resources?include_archived=true", "", ownerHeaders, http.StatusOK)
	if persisted.Total != 4 || persisted.Revision != 5 || persisted.Resources[len(persisted.Resources)-1].Status != "archived" {
		t.Fatalf("persisted resources = %#v", persisted)
	}
	configResponse := runtimeRequest(restarted, http.MethodGet, "/api/config", "", nil)
	var runtimeConfig struct {
		FeatureFlags map[string]bool `json:"feature_flags"`
	}
	if configResponse.Code != http.StatusOK || json.Unmarshal(configResponse.Body.Bytes(), &runtimeConfig) != nil || !runtimeConfig.FeatureFlags["project_resources"] {
		t.Fatalf("project_resources runtime flag = %d %s", configResponse.Code, configResponse.Body.String())
	}
}

func runtimeProjectResourceRequest(t *testing.T, runtime *Runtime, method, path, body string, headers map[string]string, wantStatus int) runtimeProjectResourceList {
	t.Helper()
	response := runtimeRequest(runtime, method, path, body, headers)
	var result runtimeProjectResourceList
	if response.Code != wantStatus || json.Unmarshal(response.Body.Bytes(), &result) != nil {
		t.Fatalf("%s %s = %d %s", method, path, response.Code, response.Body.String())
	}
	return result
}
