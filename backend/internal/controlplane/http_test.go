package controlplane

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
)

type flushingRecorder struct {
	*httptest.ResponseRecorder
	once    sync.Once
	flushed chan struct{}
}

func (r *flushingRecorder) Flush() {
	r.ResponseRecorder.Flush()
	r.once.Do(func() { close(r.flushed) })
}

func TestHTTPCommandsProjectionAndProblems(t *testing.T) {
	kernel, repository := openTestKernel(t, filepath.Join(t.TempDir(), "http.db"))
	defer repository.Close()
	service, _ := NewService(repository, nil)
	flows, _ := NewP2Flows(kernel)
	api, err := NewHTTPAPI(service, kernel, flows, func(request *http.Request) (ResolvedIdentity, error) {
		return ResolvedIdentity{Actor: Actor{ID: request.Header.Get("X-Test-Actor"), WorkspaceID: "workspace-1", Kind: ActorHuman}}, nil
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
	intentRequest := httptest.NewRequest(http.MethodPost, "/v1/workspaces/workspace-1/projects/project-1/commands", bytes.NewReader([]byte(`{"type":"requirement.intent","command_id":"command-intent","expected_head":1,"payload":{"id":"requirement-1","text":"Deliver the API"}}`)))
	intentRequest.Header.Set("X-Test-Actor", "owner-1")
	intent := httptest.NewRecorder()
	handler.ServeHTTP(intent, intentRequest)
	if intent.Code != http.StatusCreated {
		t.Fatalf("intent status=%d body=%s", intent.Code, intent.Body.String())
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

func TestHTTPEventStreamResumesAndDisconnects(t *testing.T) {
	kernel, repository := openTestKernel(t, filepath.Join(t.TempDir(), "events.db"))
	defer repository.Close()
	service, _ := NewService(repository, nil)
	flows, _ := NewP2Flows(kernel)
	api, err := NewHTTPAPI(service, kernel, flows, func(request *http.Request) (ResolvedIdentity, error) {
		return ResolvedIdentity{Actor: Actor{ID: "owner-1", WorkspaceID: request.PathValue("workspace"), Kind: ActorHuman}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	actor := Actor{ID: "owner-1", WorkspaceID: "workspace-1", Kind: ActorHuman}
	for index, id := range []string{"task-1", "task-2"} {
		if _, err := kernel.UpsertNode(context.Background(), actor, "command-"+id, "project-1", int64(index), WorkNode{ID: id, Kind: "task", Revision: 1, State: "draft", CreatorID: actor.ID}); err != nil {
			t.Fatal(err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	request := httptest.NewRequest(http.MethodGet, "/v1/workspaces/workspace-1/projects/project-1/events?after=1", nil).WithContext(ctx)
	response := &flushingRecorder{ResponseRecorder: httptest.NewRecorder(), flushed: make(chan struct{})}
	done := make(chan struct{})
	go func() {
		api.Handler().ServeHTTP(response, request)
		close(done)
	}()
	<-response.flushed
	cancel()
	<-done
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte("id: 2\n")) || bytes.Contains(response.Body.Bytes(), []byte("id: 1\n")) {
		t.Fatalf("event stream status=%d body=%s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("content type = %q", got)
	}
}
