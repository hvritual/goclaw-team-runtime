package bootstrap

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestSQLiteRuntimeIssueSimilarityReturnsOnlySameWorkspaceCandidates(t *testing.T) {
	fixture := newCollaborationRuntimeFixture(t, filepath.Join(t.TempDir(), "issue-similarity-runtime.db"), "issue-similarity", "issue-similarity@example.com")
	sameWorkspace := createRuntimeIssue(t, fixture.runtime, fixture.headers, "Alpha—Beta delivery")

	other := runtimeRequest(fixture.runtime, http.MethodPost, "/api/workspaces", `{"name":"Other Similarity","slug":"other-similarity"}`, map[string]string{
		"Authorization": "Bearer " + fixture.login.Token,
		"Content-Type":  "application/json",
	})
	if other.Code != http.StatusCreated {
		t.Fatalf("create other workspace = %d %s", other.Code, other.Body.String())
	}
	foreign := createRuntimeIssue(t, fixture.runtime, collaborationHeaders(fixture.login.Token, "other-similarity"), "Alpha beta delivery")

	response := runtimeRequest(fixture.runtime, http.MethodPost, "/api/issues/similarity/check", `{"title":"Alpha beta delivery"}`, fixture.headers)
	if response.Code != http.StatusOK {
		t.Fatalf("similarity = %d %s", response.Code, response.Body.String())
	}
	var body struct {
		RankingVersion string `json:"ranking_version"`
		Candidates     []struct {
			ID string `json:"id"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.RankingVersion == "" {
		t.Fatal("ranking_version is required")
	}
	if len(body.Candidates) != 1 || body.Candidates[0].ID != sameWorkspace.ID {
		t.Fatalf("candidates = %+v, want only %s and never %s", body.Candidates, sameWorkspace.ID, foreign.ID)
	}
}

func TestSQLiteRuntimeIssueSimilarityCanonicalHTTPAcceptance(t *testing.T) {
	fixture := newCollaborationRuntimeFixture(t, filepath.Join(t.TempDir(), "issue-similarity-http.db"), "issue-similarity-http", "issue-similarity-http@example.com")
	current := createRuntimeIssue(t, fixture.runtime, fixture.headers, "Alpha beta delivery")
	candidate := createRuntimeIssue(t, fixture.runtime, fixture.headers, "Alpha beta delivery follow up")

	other := runtimeRequest(fixture.runtime, http.MethodPost, "/api/workspaces", `{"name":"Other Similarity HTTP","slug":"other-similarity-http"}`, map[string]string{
		"Authorization": "Bearer " + fixture.login.Token,
		"Content-Type":  "application/json",
	})
	if other.Code != http.StatusCreated {
		t.Fatalf("create other workspace = %d %s", other.Code, other.Body.String())
	}
	foreign := createRuntimeIssue(t, fixture.runtime, collaborationHeaders(fixture.login.Token, "other-similarity-http"), "Alpha beta delivery")

	server := httptest.NewServer(fixture.runtime.HTTPServer())
	t.Cleanup(server.Close)
	status, body := canonicalHTTPRuntimeRequest(t, server, http.MethodPost, "/api/issues/"+current.ID+"/similarity/check", "", fixture.headers)
	if status != http.StatusOK {
		t.Fatalf("canonical HTTP similarity = %d %s", status, body)
	}
	var result struct {
		DetectorAvailable bool `json:"detector_available"`
		Candidates        []struct {
			ID string `json:"id"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatal(err)
	}
	ids := make(map[string]bool, len(result.Candidates))
	for _, item := range result.Candidates {
		ids[item.ID] = true
	}
	if !result.DetectorAvailable || ids[current.ID] || !ids[candidate.ID] || ids[foreign.ID] {
		t.Fatalf("canonical HTTP candidates = %+v, want self excluded, %s included, and %s excluded", result.Candidates, candidate.ID, foreign.ID)
	}
}

func canonicalHTTPRuntimeRequest(t *testing.T, server *httptest.Server, method, path, body string, headers map[string]string) (int, []byte) {
	t.Helper()
	request, err := http.NewRequest(method, server.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	result, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return response.StatusCode, result
}
