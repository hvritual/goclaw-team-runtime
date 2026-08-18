package bootstrap

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/hvritual/workspace/internal/modules/auth"
)

type runtimePin struct {
	ID            string `json:"id"`
	Position      int    `json:"position"`
	OrderRevision int64  `json:"order_revision"`
}

func TestSQLiteRuntimePinReorderIsAtomicRevisionedAndPersistent(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "pin-reorder-runtime.db")
	config := Config{
		Name: "backend-test", Version: "test", HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath: databasePath, WorkspaceDependencies: FailClosedWorkspaceDependencies(),
		LocalAuth: auth.LocalAuthConfig{VerificationCode: "888888"},
	}
	runtime := newRuntimeForConfig(t, config)
	login := verifyRuntimeLogin(t, runtime, "pin-reorder@example.com")
	if response := runtimeRequest(runtime, http.MethodPost, "/api/workspaces", `{"name":"Pins","slug":"pins"}`, map[string]string{
		"Authorization": "Bearer " + login.Token, "Content-Type": "application/json",
	}); response.Code != http.StatusCreated {
		t.Fatalf("create workspace = %d %s", response.Code, response.Body.String())
	}
	headers := collaborationHeaders(login.Token, "pins")
	projectID := createRuntimeSearchProject(t, runtime, headers, `{"title":"Pinned Project"}`)
	issueResponse := runtimeRequest(runtime, http.MethodPost, "/api/issues", `{"title":"Pinned Issue"}`, headers)
	var issue struct {
		ID string `json:"id"`
	}
	if issueResponse.Code != http.StatusCreated || json.Unmarshal(issueResponse.Body.Bytes(), &issue) != nil || issue.ID == "" {
		t.Fatalf("create issue = %d %s", issueResponse.Code, issueResponse.Body.String())
	}
	issuePin := createRuntimePin(t, runtime, headers, "issue", issue.ID)
	projectPin := createRuntimePin(t, runtime, headers, "project", projectID)
	if issuePin.OrderRevision != 1 || projectPin.OrderRevision != 2 {
		t.Fatalf("created revisions = %d/%d", issuePin.OrderRevision, projectPin.OrderRevision)
	}
	reordered := runtimeRequest(runtime, http.MethodPut, "/api/pins/reorder", `{"items":[{"id":"`+projectPin.ID+`"},{"id":"`+issuePin.ID+`"}],"expected_revision":2}`, headers)
	assertRuntimeResponse(t, reordered.Code, reordered.Body.String(), http.StatusNoContent, ``)
	assertRuntimePins(t, runtime, headers, []string{projectPin.ID, issuePin.ID}, 3)

	stale := runtimeRequest(runtime, http.MethodPut, "/api/pins/reorder", `{"items":[{"id":"`+issuePin.ID+`"},{"id":"`+projectPin.ID+`"}],"expected_revision":2}`, headers)
	assertRuntimeResponse(t, stale.Code, stale.Body.String(), http.StatusConflict, `{"code":"revision_conflict","current_revision":3,"error":"revision conflict"}`)
	missing := runtimeRequest(runtime, http.MethodPut, "/api/pins/reorder", `{"items":[{"id":"`+projectPin.ID+`"}],"expected_revision":3}`, headers)
	assertRuntimeResponse(t, missing.Code, missing.Body.String(), http.StatusBadRequest, `{"error":"invalid pin reorder"}`)
	assertRuntimePins(t, runtime, headers, []string{projectPin.ID, issuePin.ID}, 3)

	if err := runtime.Close(); err != nil {
		t.Fatal(err)
	}
	restarted := newRuntimeForConfig(t, config)
	assertRuntimePins(t, restarted, headers, []string{projectPin.ID, issuePin.ID}, 3)
}

func createRuntimePin(t *testing.T, runtime *Runtime, headers map[string]string, itemType, itemID string) runtimePin {
	t.Helper()
	response := runtimeRequest(runtime, http.MethodPost, "/api/pins", `{"item_type":"`+itemType+`","item_id":"`+itemID+`"}`, headers)
	var pin runtimePin
	if response.Code != http.StatusCreated || json.Unmarshal(response.Body.Bytes(), &pin) != nil || pin.ID == "" {
		t.Fatalf("create %s pin = %d %s", itemType, response.Code, response.Body.String())
	}
	return pin
}

func assertRuntimePins(t *testing.T, runtime *Runtime, headers map[string]string, ids []string, revision int64) {
	t.Helper()
	response := runtimeRequest(runtime, http.MethodGet, "/api/pins", "", headers)
	var pins []runtimePin
	if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &pins) != nil || len(pins) != len(ids) {
		t.Fatalf("list pins = %d %s", response.Code, response.Body.String())
	}
	for index, id := range ids {
		if pins[index].ID != id || pins[index].Position != index+1 || pins[index].OrderRevision != revision {
			t.Fatalf("pin %d = %+v, want id=%s revision=%d", index, pins[index], id, revision)
		}
	}
}
