package bootstrap

import (
	"encoding/json"
	"net/http"
	"net/url"
	"path/filepath"
	"testing"
)

func TestSQLiteRuntimeIssueSearchInstalledJourney(t *testing.T) {
	fixture := newCollaborationRuntimeFixture(t, filepath.Join(t.TempDir(), "issue-search-runtime.db"), "issue-search", "issue-search@example.com")
	english := createRuntimeIssue(t, fixture.runtime, fixture.headers, "Alpha—Beta delivery")
	chinese := createRuntimeIssue(t, fixture.runtime, fixture.headers, "修复咖啡机搜索")
	closed := createRuntimeIssue(t, fixture.runtime, fixture.headers, "Alpha beta closed")
	updated := runtimeRequest(fixture.runtime, http.MethodPut, "/api/issues/"+closed.ID, `{"status":"done"}`, fixture.headers)
	if updated.Code != http.StatusOK {
		t.Fatalf("close Issue = %d %s", updated.Code, updated.Body.String())
	}

	assertRuntimeIssueSearch(t, fixture, "alpha beta", false, []string{english.ID})
	assertRuntimeIssueSearch(t, fixture, "alpha beta", true, []string{closed.ID, english.ID})
	assertRuntimeIssueSearch(t, fixture, "咖啡机", false, []string{chinese.ID})
	assertRuntimeIssueSearch(t, fixture, english.Identifier, false, []string{english.ID})
	assertRuntimeIssueSearch(t, fixture, "2", false, []string{english.ID})

	other := runtimeRequest(fixture.runtime, http.MethodPost, "/api/workspaces", `{"name":"Other Search","slug":"other-search"}`, map[string]string{
		"Authorization": "Bearer " + fixture.login.Token, "Content-Type": "application/json",
	})
	if other.Code != http.StatusCreated {
		t.Fatalf("create other Workspace = %d %s", other.Code, other.Body.String())
	}
	otherHeaders := collaborationHeaders(fixture.login.Token, "other-search")
	foreign := createRuntimeIssue(t, fixture.runtime, otherHeaders, "Alpha beta foreign")
	assertRuntimeIssueSearch(t, fixture, "alpha beta", true, []string{closed.ID, english.ID})
	if foreign.ID == "" {
		t.Fatal("foreign fixture missing")
	}

	config := runtimeRequest(fixture.runtime, http.MethodGet, "/api/config", "", nil)
	var configBody struct {
		FeatureFlags map[string]bool `json:"feature_flags"`
	}
	if config.Code != http.StatusOK || json.Unmarshal(config.Body.Bytes(), &configBody) != nil {
		t.Fatalf("config = %d %s", config.Code, config.Body.String())
	}
	if !configBody.FeatureFlags["issue_search"] || configBody.FeatureFlags["project_search"] {
		t.Fatalf("search flags = %+v", configBody.FeatureFlags)
	}
}

func assertRuntimeIssueSearch(t *testing.T, fixture collaborationRuntimeFixture, query string, includeClosed bool, ids []string) {
	t.Helper()
	path := "/api/issues/search?q=" + url.QueryEscape(query) + "&limit=50"
	if includeClosed {
		path += "&include_closed=true"
	}
	response := runtimeRequest(fixture.runtime, http.MethodGet, path, "", fixture.headers)
	var body struct {
		Issues []struct {
			ID string `json:"id"`
		} `json:"issues"`
		Total int `json:"total"`
	}
	if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &body) != nil {
		t.Fatalf("search %q = %d %s", query, response.Code, response.Body.String())
	}
	if body.Total != len(ids) || len(body.Issues) != len(ids) {
		t.Fatalf("search %q = %+v, want %v", query, body, ids)
	}
	for index, id := range ids {
		if body.Issues[index].ID != id {
			t.Fatalf("search %q result %d = %s, want %s", query, index, body.Issues[index].ID, id)
		}
	}
}
