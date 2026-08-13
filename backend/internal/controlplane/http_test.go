package controlplane

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
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

	earlyIntentRequest := httptest.NewRequest(http.MethodPost, "/v1/workspaces/workspace-1/projects/project-1/commands", bytes.NewReader([]byte(`{"type":"requirement.intent","command_id":"command-early-intent","expected_head":1,"payload":{"id":"requirement-1","text":"Deliver the API"}}`)))
	earlyIntentRequest.Header.Set("X-Test-Actor", "owner-1")
	earlyIntent := httptest.NewRecorder()
	handler.ServeHTTP(earlyIntent, earlyIntentRequest)
	if earlyIntent.Code != http.StatusUnprocessableEntity {
		t.Fatalf("early intent status=%d body=%s", earlyIntent.Code, earlyIntent.Body.String())
	}

	contextStartRequest := httptest.NewRequest(http.MethodPost, "/v1/workspaces/workspace-1/projects/project-1/commands", bytes.NewReader([]byte(`{"type":"context.start","command_id":"command-context-start","expected_head":1,"payload":{"requirement_id":"requirement-1","objective":"Resolve API delivery context","max_iterations":4}}`)))
	contextStartRequest.Header.Set("X-Test-Actor", "owner-1")
	contextStart := httptest.NewRecorder()
	handler.ServeHTTP(contextStart, contextStartRequest)
	if contextStart.Code != http.StatusCreated {
		t.Fatalf("context start status=%d body=%s", contextStart.Code, contextStart.Body.String())
	}

	contextIterateRequest := httptest.NewRequest(http.MethodPost, "/v1/workspaces/workspace-1/projects/project-1/commands", bytes.NewReader([]byte(`{"type":"context.iterate","command_id":"command-context-iterate","expected_head":2,"payload":{"requirement_id":"requirement-1","needs":[{"id":"need-contract","description":"API contract is known","required":true,"status":"resolved","resolution":"typed command API","source_refs":["repo://backend/internal/controlplane/http.go"]}],"summary":"Required API context resolved"}}`)))
	contextIterateRequest.Header.Set("X-Test-Actor", "owner-1")
	contextIterate := httptest.NewRecorder()
	handler.ServeHTTP(contextIterate, contextIterateRequest)
	if contextIterate.Code != http.StatusCreated {
		t.Fatalf("context iterate status=%d body=%s", contextIterate.Code, contextIterate.Body.String())
	}

	intentRequest := httptest.NewRequest(http.MethodPost, "/v1/workspaces/workspace-1/projects/project-1/commands", bytes.NewReader([]byte(`{"type":"requirement.intent","command_id":"command-intent","expected_head":3,"payload":{"id":"requirement-1","text":"Deliver the API"}}`)))
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
	if projection.Code != http.StatusOK || !bytes.Contains(projection.Body.Bytes(), []byte("requirement-1")) || !bytes.Contains(projection.Body.Bytes(), []byte(`\"state\":\"ready\"`)) {
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

func TestHTTPCommandRejectsUnknownPayloadFields(t *testing.T) {
	kernel, repository := openTestKernel(t, filepath.Join(t.TempDir(), "strict-payload.db"))
	defer repository.Close()
	service, _ := NewService(repository, nil)
	flows, _ := NewP2Flows(kernel)
	api, _ := NewHTTPAPI(service, kernel, flows, func(request *http.Request) (ResolvedIdentity, error) {
		return ResolvedIdentity{Actor: Actor{ID: "owner-1", WorkspaceID: request.PathValue("workspace"), Kind: ActorHuman}}, nil
	})
	body := []byte(`{"type":"requirement.start","command_id":"command-1","expected_head":0,"payload":{"id":"requirement-1","text":"Need API","unexpected":"secret"}}`)
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/workspaces/workspace-1/projects/project-1/commands", bytes.NewReader(body)))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}

	start := []byte(`{"type":"requirement.start","command_id":"command-2","expected_head":0,"payload":{"id":"requirement-1","text":"Need API"}}`)
	startResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(startResponse, httptest.NewRequest(http.MethodPost, "/v1/workspaces/workspace-1/projects/project-1/commands", bytes.NewReader(start)))
	if startResponse.Code != http.StatusCreated {
		t.Fatalf("start status=%d body=%s", startResponse.Code, startResponse.Body.String())
	}
	contextBody := []byte(`{"type":"context.start","command_id":"command-3","expected_head":1,"payload":{"requirement_id":"requirement-1","objective":"Discover","unexpected":"secret"}}`)
	contextResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(contextResponse, httptest.NewRequest(http.MethodPost, "/v1/workspaces/workspace-1/projects/project-1/commands", bytes.NewReader(contextBody)))
	if contextResponse.Code != http.StatusBadRequest {
		t.Fatalf("context status=%d body=%s", contextResponse.Code, contextResponse.Body.String())
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

func TestHTTPEventStreamReauthorizesAndStopsAfterRevocation(t *testing.T) {
	kernel, repository := openTestKernel(t, filepath.Join(t.TempDir(), "revoked-events.db"))
	defer repository.Close()
	service, _ := NewService(repository, nil)
	flows, _ := NewP2Flows(kernel)
	var resolutions atomic.Int32
	api, err := NewHTTPAPI(service, kernel, flows, func(request *http.Request) (ResolvedIdentity, error) {
		if resolutions.Add(1) > 1 {
			return ResolvedIdentity{}, denied("test identity", "revoked")
		}
		return ResolvedIdentity{Actor: Actor{ID: "owner-1", WorkspaceID: request.PathValue("workspace"), Kind: ActorHuman}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	api.sseReauthorizeInterval = 10 * time.Millisecond
	api.ssePollInterval = time.Hour
	api.sseHeartbeatInterval = time.Hour
	request := httptest.NewRequest(http.MethodGet, "/v1/workspaces/workspace-1/projects/project-1/events", nil)
	response := &flushingRecorder{ResponseRecorder: httptest.NewRecorder(), flushed: make(chan struct{})}
	done := make(chan struct{})
	go func() { api.Handler().ServeHTTP(response, request); close(done) }()
	<-response.flushed
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("event stream did not close after authorization was revoked")
	}
	if resolutions.Load() < 2 {
		t.Fatalf("identity resolutions = %d, want reauthorization", resolutions.Load())
	}
}

func TestSessionEventCursorQueryIsBounded(t *testing.T) {
	kernel, repository := openTestKernel(t, filepath.Join(t.TempDir(), "bounded-events.db"))
	defer repository.Close()
	actor := Actor{ID: "owner-1", WorkspaceID: "workspace-1", Kind: ActorHuman}
	for index := 0; index < 5; index++ {
		if _, err := kernel.UpsertNode(context.Background(), actor, "command-"+string(rune('a'+index)), "project-1", int64(index), WorkNode{ID: "task-" + string(rune('a'+index)), Kind: "task", Revision: 1, State: "draft", CreatorID: actor.ID}); err != nil {
			t.Fatal(err)
		}
	}
	events, err := kernel.store.ListSessionEventsAfter(context.Background(), "workspace-1", "project-1", 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0].Sequence != 3 || events[1].Sequence != 4 {
		t.Fatalf("events = %#v", events)
	}
}
