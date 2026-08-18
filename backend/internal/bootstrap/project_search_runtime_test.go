package bootstrap

import (
	"encoding/json"
	"net/http"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/hvritual/workspace/internal/modules/auth"
)

func TestSQLiteRuntimeProjectSearchInstalledJourneyAndRestart(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "project-search-runtime.db")
	config := Config{
		Name: "backend-test", Version: "test", HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath: databasePath, WorkspaceDependencies: FailClosedWorkspaceDependencies(),
		LocalAuth: auth.LocalAuthConfig{VerificationCode: "888888"},
	}
	runtime := newRuntimeForConfig(t, config)
	login := verifyRuntimeLogin(t, runtime, "project-search@example.com")
	if response := runtimeRequest(runtime, http.MethodPost, "/api/workspaces", `{"name":"Search","slug":"project-search"}`, map[string]string{
		"Authorization": "Bearer " + login.Token, "Content-Type": "application/json",
	}); response.Code != http.StatusCreated {
		t.Fatalf("create workspace = %d %s", response.Code, response.Body.String())
	}
	headers := collaborationHeaders(login.Token, "project-search")
	english := createRuntimeSearchProject(t, runtime, headers, `{"title":"Alpha—Beta delivery","description":"English project"}`)
	chinese := createRuntimeSearchProject(t, runtime, headers, `{"title":"修复咖啡机搜索","description":"中文项目"}`)
	closed := createRuntimeSearchProject(t, runtime, headers, `{"title":"Alpha beta closed"}`)
	if response := runtimeRequest(runtime, http.MethodPut, "/api/projects/"+closed, `{"status":"completed"}`, headers); response.Code != http.StatusOK {
		t.Fatalf("close Project = %d %s", response.Code, response.Body.String())
	}
	assertRuntimeProjectSearch(t, runtime, headers, "alpha beta", false, []string{english})
	assertRuntimeProjectSearch(t, runtime, headers, "alpha beta", true, []string{closed, english})
	assertRuntimeProjectSearch(t, runtime, headers, "咖啡机", false, []string{chinese})

	if response := runtimeRequest(runtime, http.MethodPost, "/api/workspaces", `{"name":"Other","slug":"project-search-other"}`, map[string]string{
		"Authorization": "Bearer " + login.Token, "Content-Type": "application/json",
	}); response.Code != http.StatusCreated {
		t.Fatalf("create other workspace = %d %s", response.Code, response.Body.String())
	}
	otherHeaders := collaborationHeaders(login.Token, "project-search-other")
	_ = createRuntimeSearchProject(t, runtime, otherHeaders, `{"title":"Alpha beta foreign"}`)
	assertRuntimeProjectSearch(t, runtime, headers, "alpha beta", true, []string{closed, english})

	if err := runtime.Close(); err != nil {
		t.Fatal(err)
	}
	restarted := newRuntimeForConfig(t, config)
	assertRuntimeProjectSearch(t, restarted, headers, "咖啡机", false, []string{chinese})

	configResponse := runtimeRequest(restarted, http.MethodGet, "/api/config", "", nil)
	var body struct {
		FeatureFlags map[string]bool `json:"feature_flags"`
	}
	if configResponse.Code != http.StatusOK || json.Unmarshal(configResponse.Body.Bytes(), &body) != nil {
		t.Fatalf("config = %d %s", configResponse.Code, configResponse.Body.String())
	}
	if !body.FeatureFlags["issue_search"] || !body.FeatureFlags["project_search"] || !body.FeatureFlags["pin_reorder"] {
		t.Fatalf("feature flags = %+v", body.FeatureFlags)
	}
}

func createRuntimeSearchProject(t *testing.T, runtime *Runtime, headers map[string]string, body string) string {
	t.Helper()
	response := runtimeRequest(runtime, http.MethodPost, "/api/projects", body, headers)
	var project struct {
		ID string `json:"id"`
	}
	if response.Code != http.StatusCreated || json.Unmarshal(response.Body.Bytes(), &project) != nil || project.ID == "" {
		t.Fatalf("create Project = %d %s", response.Code, response.Body.String())
	}
	return project.ID
}

func assertRuntimeProjectSearch(t *testing.T, runtime *Runtime, headers map[string]string, query string, includeClosed bool, ids []string) {
	t.Helper()
	path := "/api/projects/search?q=" + url.QueryEscape(query) + "&limit=50"
	if includeClosed {
		path += "&include_closed=true"
	}
	response := runtimeRequest(runtime, http.MethodGet, path, "", headers)
	var body struct {
		Projects []struct {
			ID          string `json:"id"`
			MatchSource string `json:"match_source"`
		} `json:"projects"`
		Total int `json:"total"`
	}
	if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &body) != nil {
		t.Fatalf("search %q = %d %s", query, response.Code, response.Body.String())
	}
	if body.Total != len(ids) || len(body.Projects) != len(ids) {
		t.Fatalf("search %q = %+v, want %v", query, body, ids)
	}
	for index, id := range ids {
		if body.Projects[index].ID != id || body.Projects[index].MatchSource == "" {
			t.Fatalf("search %q result %d = %+v, want %s", query, index, body.Projects[index], id)
		}
	}
}
