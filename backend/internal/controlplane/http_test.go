package controlplane

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestHTTPCommandsProjectionAndProblems(t *testing.T) {
	kernel, repository := openTestKernel(t, filepath.Join(t.TempDir(), "http.db"))
	defer repository.Close()
	flows, _ := NewP2Flows(kernel)
	api, err := NewHTTPAPI(kernel, flows, func(request *http.Request) (Actor, error) {
		return Actor{ID: request.Header.Get("X-Test-Actor"), WorkspaceID: "workspace-1", Kind: ActorHuman}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := api.Handler()

	health := httptest.NewRecorder()
	handler.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if health.Code != http.StatusOK {
		t.Fatalf("health status=%d", health.Code)
	}
	body := []byte(`{"type":"requirement.start","command_id":"command-1","expected_head":0,"payload":{"id":"requirement-1","text":"Need API"}}`)
	request := httptest.NewRequest(http.MethodPost, "/v1/workspaces/workspace-1/projects/project-1/commands", bytes.NewReader(body))
	request.Header.Set("X-Test-Actor", "owner-1")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", response.Code, response.Body.String())
	}

	projectionRequest := httptest.NewRequest(http.MethodGet, "/v1/workspaces/workspace-1/projects/project-1/projection", nil)
	projectionRequest.Header.Set("X-Test-Actor", "owner-1")
	projection := httptest.NewRecorder()
	handler.ServeHTTP(projection, projectionRequest)
	if projection.Code != http.StatusOK || !bytes.Contains(projection.Body.Bytes(), []byte("requirement-1")) {
		t.Fatalf("projection status=%d body=%s", projection.Code, projection.Body.String())
	}

	staleRequest := httptest.NewRequest(http.MethodPost, "/v1/workspaces/workspace-1/projects/project-1/commands", bytes.NewReader([]byte(`{"type":"requirement.start","command_id":"command-2","expected_head":0,"payload":{"id":"requirement-2","text":"Stale"}}`)))
	staleRequest.Header.Set("X-Test-Actor", "owner-1")
	stale := httptest.NewRecorder()
	handler.ServeHTTP(stale, staleRequest)
	if stale.Code != http.StatusConflict {
		t.Fatalf("stale status=%d body=%s", stale.Code, stale.Body.String())
	}
	badWorkspace := httptest.NewRequest(http.MethodGet, "/v1/workspaces/workspace-2/projects/project-1/projection", nil)
	badWorkspace.Header.Set("X-Test-Actor", "owner-1")
	deniedResponse := httptest.NewRecorder()
	handler.ServeHTTP(deniedResponse, badWorkspace)
	if deniedResponse.Code != http.StatusForbidden {
		t.Fatalf("denied status=%d", deniedResponse.Code)
	}
	bad := httptest.NewRecorder()
	handler.ServeHTTP(bad, httptest.NewRequest(http.MethodPost, "/v1/workspaces/workspace-1/projects/project-1/commands", bytes.NewReader([]byte(`{`))))
	if bad.Code != http.StatusBadRequest && bad.Code != http.StatusForbidden {
		t.Fatalf("bad status=%d", bad.Code)
	}
}
