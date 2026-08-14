package bootstrap

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/hvritual/workspace/internal/modules/auth"
)

func TestSQLiteRuntimeExposesPublicUserIDForIssueMemberActors(t *testing.T) {
	runtime := newRuntimeForConfig(t, Config{
		Name: "backend-test", Version: "test",
		HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath:            filepath.Join(t.TempDir(), "issue-public-actor.db"),
		WorkspaceDependencies: FailClosedWorkspaceDependencies(),
		LocalAuth:             auth.LocalAuthConfig{VerificationCode: "888888"},
	})
	login := verifyRuntimeLogin(t, runtime, "issue-public-actor@example.com")
	createdWorkspace := runtimeRequest(runtime, http.MethodPost, "/api/workspaces", `{"name":"Actor Contract","slug":"actor-contract"}`, map[string]string{
		"Authorization": "Bearer " + login.Token,
		"Content-Type":  "application/json",
	})
	if createdWorkspace.Code != http.StatusCreated {
		t.Fatalf("create workspace = %d %s", createdWorkspace.Code, createdWorkspace.Body.String())
	}

	createdIssue := runtimeRequest(runtime, http.MethodPost, "/api/issues", `{"title":"Public actor"}`, map[string]string{
		"Authorization":    "Bearer " + login.Token,
		"X-Workspace-Slug": "actor-contract",
		"Content-Type":     "application/json",
	})
	if createdIssue.Code != http.StatusCreated {
		t.Fatalf("create issue = %d %s", createdIssue.Code, createdIssue.Body.String())
	}
	var body struct {
		CreatorType string `json:"creator_type"`
		CreatorID   string `json:"creator_id"`
	}
	if err := json.Unmarshal(createdIssue.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.CreatorType != "member" || body.CreatorID != login.UserID {
		t.Fatalf("creator actor = %s:%s, want member:%s; body=%s", body.CreatorType, body.CreatorID, login.UserID, createdIssue.Body.String())
	}
}

func TestSQLiteRuntimeUpdatesIssueFromCurrentDetailContract(t *testing.T) {
	runtime := newRuntimeForConfig(t, Config{
		Name: "backend-test", Version: "test",
		HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath:            filepath.Join(t.TempDir(), "issue-update.db"),
		WorkspaceDependencies: FailClosedWorkspaceDependencies(),
		LocalAuth:             auth.LocalAuthConfig{VerificationCode: "888888"},
	})
	login := verifyRuntimeLogin(t, runtime, "issue-update@example.com")
	workspace := runtimeRequest(runtime, http.MethodPost, "/api/workspaces", `{"name":"Issue Update","slug":"issue-update"}`, map[string]string{
		"Authorization": "Bearer " + login.Token,
		"Content-Type":  "application/json",
	})
	if workspace.Code != http.StatusCreated {
		t.Fatalf("create workspace = %d %s", workspace.Code, workspace.Body.String())
	}
	headers := map[string]string{
		"Authorization":    "Bearer " + login.Token,
		"X-Workspace-Slug": "issue-update",
		"Content-Type":     "application/json",
	}
	created := runtimeRequest(runtime, http.MethodPost, "/api/issues", `{"title":"Before update"}`, headers)
	if created.Code != http.StatusCreated {
		t.Fatalf("create issue = %d %s", created.Code, created.Body.String())
	}
	var original struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &original); err != nil || original.ID == "" {
		t.Fatalf("decode created issue: %v body=%s", err, created.Body.String())
	}

	updated := runtimeRequest(runtime, http.MethodPut, "/api/issues/"+original.ID,
		`{"title":"After update","description":"Full detail","status":"in_progress","priority":"urgent","assignee_type":"member","assignee_id":"`+login.UserID+`","start_date":"2026-08-14","due_date":"2026-08-31"}`,
		headers,
	)
	if updated.Code != http.StatusOK {
		t.Fatalf("update issue = %d %s", updated.Code, updated.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(updated.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["id"] != original.ID || body["title"] != "After update" || body["description"] != "Full detail" || body["status"] != "in_progress" || body["priority"] != "urgent" || body["assignee_type"] != "member" || body["assignee_id"] != login.UserID || body["start_date"] != "2026-08-14" || body["due_date"] != "2026-08-31" {
		t.Fatalf("updated issue = %s", updated.Body.String())
	}
	readback := runtimeRequest(runtime, http.MethodGet, "/api/issues/"+original.ID, "", headers)
	if readback.Code != http.StatusOK || readback.Body.String() != updated.Body.String() {
		t.Fatalf("readback = %d %s, want %s", readback.Code, readback.Body.String(), updated.Body.String())
	}
	cleared := runtimeRequest(runtime, http.MethodPut, "/api/issues/"+original.ID,
		`{"assignee_type":null,"assignee_id":null,"stage":null,"start_date":null,"due_date":null}`,
		headers,
	)
	if cleared.Code != http.StatusOK || !containsJSON(cleared.Body.Bytes(), `"assignee_type":null`, `"assignee_id":null`, `"stage":null`, `"start_date":null`, `"due_date":null`) {
		t.Fatalf("clear nullable issue fields = %d %s", cleared.Code, cleared.Body.String())
	}
	for name, response := range map[string]*httptest.ResponseRecorder{
		"missing auth": runtimeRequest(runtime, http.MethodPut, "/api/issues/"+original.ID, `{"title":"ignored"}`, map[string]string{
			"X-Workspace-Slug": "issue-update", "Content-Type": "application/json",
		}),
		"missing workspace": runtimeRequest(runtime, http.MethodPut, "/api/issues/"+original.ID, `{"title":"ignored"}`, map[string]string{
			"Authorization": "Bearer " + login.Token, "Content-Type": "application/json",
		}),
		"cookie without csrf": runtimeRequest(runtime, http.MethodPut, "/api/issues/"+original.ID, `{"title":"ignored"}`, map[string]string{
			"Cookie": "multica_auth=" + login.Token, "X-Workspace-Slug": "issue-update", "Content-Type": "application/json",
		}),
		"unknown field":                        runtimeRequest(runtime, http.MethodPut, "/api/issues/"+original.ID, `{"unknown":true}`, headers),
		"trailing body":                        runtimeRequest(runtime, http.MethodPut, "/api/issues/"+original.ID, `{"title":"ignored"} {}`, headers),
		"missing target before malformed body": runtimeRequest(runtime, http.MethodPut, "/api/issues/missing", `{"unknown":true}`, headers),
	} {
		want := map[string]int{
			"missing auth": http.StatusUnauthorized, "missing workspace": http.StatusBadRequest,
			"cookie without csrf": http.StatusForbidden, "unknown field": http.StatusBadRequest,
			"trailing body": http.StatusBadRequest, "missing target before malformed body": http.StatusNotFound,
		}[name]
		if response.Code != want {
			t.Fatalf("%s = %d %s, want %d", name, response.Code, response.Body.String(), want)
		}
	}
}

func TestSQLiteRuntimeMovesIssueUsingRelativeAnchors(t *testing.T) {
	runtime := newRuntimeForConfig(t, Config{
		Name: "backend-test", Version: "test",
		HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath:            filepath.Join(t.TempDir(), "issue-move.db"),
		WorkspaceDependencies: FailClosedWorkspaceDependencies(),
		LocalAuth:             auth.LocalAuthConfig{VerificationCode: "888888"},
	})
	login := verifyRuntimeLogin(t, runtime, "issue-move@example.com")
	workspace := runtimeRequest(runtime, http.MethodPost, "/api/workspaces", `{"name":"Issue Move","slug":"issue-move"}`, map[string]string{
		"Authorization": "Bearer " + login.Token,
		"Content-Type":  "application/json",
	})
	if workspace.Code != http.StatusCreated {
		t.Fatalf("create workspace = %d %s", workspace.Code, workspace.Body.String())
	}
	headers := map[string]string{
		"Authorization":    "Bearer " + login.Token,
		"X-Workspace-Slug": "issue-move",
		"Content-Type":     "application/json",
	}
	create := func(title string, position float64) string {
		t.Helper()
		response := runtimeRequest(runtime, http.MethodPost, "/api/issues", `{"title":"`+title+`","position":`+formatTestFloat(position)+`}`, headers)
		if response.Code != http.StatusCreated {
			t.Fatalf("create %s = %d %s", title, response.Code, response.Body.String())
		}
		var body struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body.ID == "" {
			t.Fatalf("decode %s: %v body=%s", title, err, response.Body.String())
		}
		return body.ID
	}
	beforeID := create("Before", 10)
	afterID := create("After", 20)
	movedID := create("Moved", 30)

	moved := runtimeRequest(runtime, http.MethodPost, "/api/issues/"+movedID+"/move",
		`{"status":"done","before_id":"`+beforeID+`","after_id":"`+afterID+`"}`,
		headers,
	)
	if moved.Code != http.StatusOK {
		t.Fatalf("move issue = %d %s", moved.Code, moved.Body.String())
	}
	var body struct {
		ID       string  `json:"id"`
		Status   string  `json:"status"`
		Position float64 `json:"position"`
	}
	if err := json.Unmarshal(moved.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.ID != movedID || body.Status != "done" || body.Position != 15 {
		t.Fatalf("moved issue = %s", moved.Body.String())
	}
	readback := runtimeRequest(runtime, http.MethodGet, "/api/issues/"+movedID, "", headers)
	if readback.Code != http.StatusOK || !containsJSON(readback.Body.Bytes(), `"status":"done"`, `"position":15`) {
		t.Fatalf("move readback = %d %s", readback.Code, readback.Body.String())
	}
	emptyAnchor := runtimeRequest(runtime, http.MethodPost, "/api/issues/"+movedID+"/move",
		`{"before_id":"","after_id":null}`,
		headers,
	)
	assertRuntimeResponse(t, emptyAnchor.Code, emptyAnchor.Body.String(), http.StatusBadRequest, `{"error":"invalid move anchor"}`)
	for name, body := range map[string]string{
		"missing anchor":    `{"before_id":null}`,
		"client position":   `{"before_id":null,"after_id":null,"position":99}`,
		"self anchor":       `{"before_id":"` + movedID + `","after_id":null}`,
		"duplicate anchor":  `{"before_id":"` + beforeID + `","after_id":"` + beforeID + `"}`,
		"stale anchors":     `{"before_id":"` + afterID + `","after_id":"` + beforeID + `"}`,
		"missing anchor id": `{"before_id":"missing","after_id":null}`,
	} {
		response := runtimeRequest(runtime, http.MethodPost, "/api/issues/"+movedID+"/move", body, headers)
		want := http.StatusBadRequest
		if name == "self anchor" || name == "duplicate anchor" || name == "stale anchors" {
			want = http.StatusConflict
		}
		if response.Code != want {
			t.Fatalf("%s = %d %s, want %d", name, response.Code, response.Body.String(), want)
		}
	}
}

func TestSQLiteRuntimePublishesOnlyCommittedIssueMoves(t *testing.T) {
	runtime := newRuntimeForConfig(t, Config{
		Name: "backend-test", Version: "test",
		HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath:            filepath.Join(t.TempDir(), "issue-move-events.db"),
		WorkspaceDependencies: FailClosedWorkspaceDependencies(),
		LocalAuth:             auth.LocalAuthConfig{VerificationCode: "888888"},
	})
	login := verifyRuntimeLogin(t, runtime, "issue-move-events@example.com")
	workspace := runtimeRequest(runtime, http.MethodPost, "/api/workspaces", `{"name":"Move Events","slug":"move-events"}`, map[string]string{
		"Authorization": "Bearer " + login.Token, "Content-Type": "application/json",
	})
	if workspace.Code != http.StatusCreated {
		t.Fatalf("create workspace = %d %s", workspace.Code, workspace.Body.String())
	}
	headers := map[string]string{"Authorization": "Bearer " + login.Token, "X-Workspace-Slug": "move-events", "Content-Type": "application/json"}
	create := func(title string, position int) string {
		t.Helper()
		response := runtimeRequest(runtime, http.MethodPost, "/api/issues", fmt.Sprintf(`{"title":%q,"position":%d}`, title, position), headers)
		var body struct {
			ID string `json:"id"`
		}
		if response.Code != http.StatusCreated || json.Unmarshal(response.Body.Bytes(), &body) != nil || body.ID == "" {
			t.Fatalf("create %s = %d %s", title, response.Code, response.Body.String())
		}
		return body.ID
	}
	beforeID := create("Event before", 10)
	afterID := create("Event after", 20)
	movedID := create("Event moved", 30)
	server := httptest.NewServer(runtime.HTTPServer())
	t.Cleanup(server.Close)
	socket := dialTokenRealtime(t, server.URL, "move-events", login.Token)

	success := runtimeRequest(runtime, http.MethodPost, "/api/issues/"+movedID+"/move", `{"status":"done","before_id":"`+beforeID+`","after_id":"`+afterID+`"}`, headers)
	if success.Code != http.StatusOK {
		t.Fatalf("successful move = %d %s", success.Code, success.Body.String())
	}
	assertRealtimeEvent(t, socket, "issue:updated", `"position":15`)
	_ = socket.Close()

	failureSocket := dialTokenRealtime(t, server.URL, "move-events", login.Token)
	if _, err := runtime.Database().Exec(`CREATE TRIGGER fail_issue_move BEFORE UPDATE ON workspace_issues WHEN OLD.id='` + movedID + `' BEGIN SELECT RAISE(ABORT,'forced move rollback'); END`); err != nil {
		t.Fatal(err)
	}
	failed := runtimeRequest(runtime, http.MethodPost, "/api/issues/"+movedID+"/move", `{"status":"cancelled","before_id":null,"after_id":null}`, headers)
	assertRuntimeResponse(t, failed.Code, failed.Body.String(), http.StatusInternalServerError, `{"error":"failed to move issue"}`)
	assertNoRealtimeEvent(t, failureSocket)
	_ = failureSocket.Close()
	var status string
	var position float64
	if err := runtime.Database().QueryRow(`SELECT status,position FROM workspace_issues WHERE id=?`, movedID).Scan(&status, &position); err != nil || status != "done" || position != 15 {
		t.Fatalf("rolled back move = status %q position %v err %v", status, position, err)
	}
	if _, err := runtime.Database().Exec(`DROP TRIGGER fail_issue_move`); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(success.Body.String()) == "" {
		t.Fatal("successful move returned an empty Issue")
	}
}

func TestSQLiteRuntimeSerializesConcurrentIssueMovesAndPersistsRestart(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "issue-move-concurrent.db")
	config := Config{
		Name: "backend-test", Version: "test",
		HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath:            databasePath,
		WorkspaceDependencies: FailClosedWorkspaceDependencies(),
		LocalAuth:             auth.LocalAuthConfig{VerificationCode: "888888"},
	}
	runtime := newRuntimeForConfig(t, config)
	login := verifyRuntimeLogin(t, runtime, "issue-move-concurrent@example.com")
	workspace := runtimeRequest(runtime, http.MethodPost, "/api/workspaces", `{"name":"Concurrent Move","slug":"concurrent-move"}`, map[string]string{
		"Authorization": "Bearer " + login.Token, "Content-Type": "application/json",
	})
	if workspace.Code != http.StatusCreated {
		t.Fatalf("create workspace = %d %s", workspace.Code, workspace.Body.String())
	}
	headers := map[string]string{"Authorization": "Bearer " + login.Token, "X-Workspace-Slug": "concurrent-move", "Content-Type": "application/json"}
	create := func(title string, position int) string {
		t.Helper()
		response := runtimeRequest(runtime, http.MethodPost, "/api/issues", fmt.Sprintf(`{"title":%q,"position":%d}`, title, position), headers)
		var body struct {
			ID string `json:"id"`
		}
		if response.Code != http.StatusCreated || json.Unmarshal(response.Body.Bytes(), &body) != nil || body.ID == "" {
			t.Fatalf("create %s = %d %s", title, response.Code, response.Body.String())
		}
		return body.ID
	}
	leftID := create("Left", 10)
	middleID := create("Middle", 20)
	rightID := create("Right", 40)
	movedID := create("Concurrent target", 50)
	type result struct {
		code int
		body string
	}
	results := make(chan result, 2)
	start := make(chan struct{})
	var workers sync.WaitGroup
	for _, body := range []string{
		`{"status":"done","before_id":"` + leftID + `","after_id":"` + middleID + `"}`,
		`{"status":"blocked","before_id":"` + middleID + `","after_id":"` + rightID + `"}`,
	} {
		workers.Add(1)
		go func(body string) {
			defer workers.Done()
			<-start
			response := runtimeRequest(runtime, http.MethodPost, "/api/issues/"+movedID+"/move", body, headers)
			results <- result{code: response.Code, body: response.Body.String()}
		}(body)
	}
	close(start)
	workers.Wait()
	close(results)
	for response := range results {
		if response.code != http.StatusOK {
			t.Fatalf("concurrent move = %d %s", response.code, response.body)
		}
	}
	var status string
	var position float64
	if err := runtime.Database().QueryRow(`SELECT status,position FROM workspace_issues WHERE id=?`, movedID).Scan(&status, &position); err != nil {
		t.Fatal(err)
	}
	if !((status == "done" && position == 15) || (status == "blocked" && position == 30)) {
		t.Fatalf("concurrent move mixed commits: status=%q position=%v", status, position)
	}
	if err := runtime.Close(); err != nil {
		t.Fatal(err)
	}
	restarted := newRuntimeForConfig(t, config)
	readback := runtimeRequest(restarted, http.MethodGet, "/api/issues/"+movedID, "", headers)
	if readback.Code != http.StatusOK || !containsJSON(readback.Body.Bytes(), `"status":"`+status+`"`, `"position":`+formatTestFloat(position)) {
		t.Fatalf("restart readback = %d %s", readback.Code, readback.Body.String())
	}
}

func TestSQLiteRuntimeNormalizesRetainedPrivateMemberActorIDs(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "issue-actor-upgrade.db")
	config := Config{
		Name: "backend-test", Version: "test",
		HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath:            databasePath,
		WorkspaceDependencies: FailClosedWorkspaceDependencies(),
		LocalAuth:             auth.LocalAuthConfig{VerificationCode: "888888"},
	}
	runtime := newRuntimeForConfig(t, config)
	login := verifyRuntimeLogin(t, runtime, "issue-actor-upgrade@example.com")
	workspaceResponse := runtimeRequest(runtime, http.MethodPost, "/api/workspaces", `{"name":"Actor Upgrade","slug":"actor-upgrade"}`, map[string]string{
		"Authorization": "Bearer " + login.Token,
		"Content-Type":  "application/json",
	})
	if workspaceResponse.Code != http.StatusCreated {
		t.Fatalf("create workspace = %d %s", workspaceResponse.Code, workspaceResponse.Body.String())
	}
	var workspace struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(workspaceResponse.Body.Bytes(), &workspace); err != nil {
		t.Fatal(err)
	}
	var memberID string
	if err := runtime.Database().QueryRow(`SELECT id FROM auth_members WHERE workspace_id=? AND user_id=?`, workspace.ID, login.UserID).Scan(&memberID); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Database().Exec(`INSERT INTO workspace_issues(
		id,workspace_id,number,identifier,title,status,priority,assignee_type,assignee_id,creator_type,creator_id,position,metadata,properties,asset_ids,created_at,updated_at
	) VALUES('retained-private-actor',?,1,'ACT-1','Retained private actor','todo','none','member',?,'member',?,1,'{}','{}','[]','2026-08-13T00:00:00Z','2026-08-13T00:00:00Z')`, workspace.ID, memberID, memberID); err != nil {
		t.Fatal(err)
	}
	if err := runtime.Close(); err != nil {
		t.Fatal(err)
	}

	restarted := newRuntimeForConfig(t, config)
	response := runtimeRequest(restarted, http.MethodGet, "/api/issues/retained-private-actor", "", map[string]string{
		"Authorization":    "Bearer " + login.Token,
		"X-Workspace-Slug": "actor-upgrade",
	})
	if response.Code != http.StatusOK || !containsJSON(response.Body.Bytes(), `"creator_id":"`+login.UserID+`"`, `"assignee_id":"`+login.UserID+`"`) {
		t.Fatalf("normalized retained actor = %d %s", response.Code, response.Body.String())
	}
}

func formatTestFloat(value float64) string {
	return fmt.Sprintf("%g", value)
}
