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
	"time"

	"github.com/hvritual/workspace/internal/modules/auth"
)

func TestSQLiteRuntimeServesIssueHierarchyAndBatchRoutes(t *testing.T) {
	runtime := newRuntimeForConfig(t, Config{
		Name: "backend-test", Version: "test",
		HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath:            filepath.Join(t.TempDir(), "issue-hierarchy.db"),
		WorkspaceDependencies: FailClosedWorkspaceDependencies(),
		LocalAuth:             auth.LocalAuthConfig{VerificationCode: "888888"},
	})
	login := verifyRuntimeLogin(t, runtime, "issue-hierarchy@example.com")

	t.Run("authenticates before requiring workspace", func(t *testing.T) {
		missingAuth := runtimeRequest(runtime, http.MethodGet, "/api/issues/child-progress", "", nil)
		assertRuntimeResponse(t, missingAuth.Code, missingAuth.Body.String(), http.StatusUnauthorized, `{"error":"user not authenticated"}`)

		missingWorkspace := runtimeRequest(runtime, http.MethodGet, "/api/issues/child-progress", "", map[string]string{
			"Authorization": "Bearer " + login.Token,
		})
		assertRuntimeResponse(t, missingWorkspace.Code, missingWorkspace.Body.String(), http.StatusBadRequest, `{"error":"workspace is required"}`)

		missingBatchWorkspace := runtimeRequest(runtime, http.MethodPost, "/api/issues/batch-update", `{"issue_ids":["missing"],"updates":{"priority":"urgent"}}`, map[string]string{
			"Authorization": "Bearer " + login.Token,
			"Content-Type":  "application/json",
		})
		assertRuntimeResponse(t, missingBatchWorkspace.Code, missingBatchWorkspace.Body.String(), http.StatusBadRequest, `{"error":"workspace is required"}`)

		missingBatchAuth := runtimeRequest(runtime, http.MethodPost, "/api/issues/batch-delete", `{"issue_ids":["missing"]}`, map[string]string{
			"X-Workspace-Slug": "issue-hierarchy", "Content-Type": "application/json",
		})
		assertRuntimeResponse(t, missingBatchAuth.Code, missingBatchAuth.Body.String(), http.StatusUnauthorized, `{"error":"user not authenticated"}`)
	})

	workspace := runtimeRequest(runtime, http.MethodPost, "/api/workspaces", `{"name":"Issue Hierarchy","slug":"issue-hierarchy"}`, map[string]string{
		"Authorization": "Bearer " + login.Token,
		"Content-Type":  "application/json",
	})
	if workspace.Code != http.StatusCreated {
		t.Fatalf("create workspace = %d %s", workspace.Code, workspace.Body.String())
	}
	var workspaceBody struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(workspace.Body.Bytes(), &workspaceBody); err != nil || workspaceBody.ID == "" {
		t.Fatalf("decode workspace: %v body=%s", err, workspace.Body.String())
	}
	headers := map[string]string{
		"Authorization":    "Bearer " + login.Token,
		"X-Workspace-Slug": "issue-hierarchy",
		"Content-Type":     "application/json",
	}
	create := func(title string, parentID *string, status string, position int) string {
		t.Helper()
		parent := "null"
		if parentID != nil {
			parent = fmt.Sprintf("%q", *parentID)
		}
		response := runtimeRequest(runtime, http.MethodPost, "/api/issues", fmt.Sprintf(`{"title":%q,"parent_issue_id":%s,"status":%q,"position":%d}`, title, parent, status, position), headers)
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
	parentID := create("Parent", nil, "todo", 10)
	// Sibling order is the monotonic Issue number, not mutable status-surface
	// position. Reversing the positions catches accidental position ordering.
	childOneID := create("Child one", &parentID, "done", 30)
	childTwoID := create("Child two", &parentID, "cancelled", 20)
	var parentIdentifier, childOneIdentifier string
	if err := runtime.Database().QueryRow(`SELECT identifier FROM workspace_issues WHERE id=?`, parentID).Scan(&parentIdentifier); err != nil {
		t.Fatal(err)
	}
	if err := runtime.Database().QueryRow(`SELECT identifier FROM workspace_issues WHERE id=?`, childOneID).Scan(&childOneIdentifier); err != nil {
		t.Fatal(err)
	}

	t.Run("per-parent children", func(t *testing.T) {
		response := runtimeRequest(runtime, http.MethodGet, "/api/issues/"+parentID+"/children", "", headers)
		if response.Code != http.StatusOK || !containsJSON(response.Body.Bytes(), `"id":"`+childOneID+`"`, `"id":"`+childTwoID+`"`) {
			t.Fatalf("children = %d %s", response.Code, response.Body.String())
		}
		var body struct {
			Issues []struct {
				ID string `json:"id"`
			} `json:"issues"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || len(body.Issues) != 2 || body.Issues[0].ID != childOneID || body.Issues[1].ID != childTwoID {
			t.Fatalf("ordered children: err=%v body=%s", err, response.Body.String())
		}
		byIdentifier := runtimeRequest(runtime, http.MethodGet, "/api/issues/"+parentIdentifier+"/children", "", headers)
		if byIdentifier.Code != http.StatusOK || byIdentifier.Body.String() != response.Body.String() {
			t.Fatalf("identifier children = %d %s, want %s", byIdentifier.Code, byIdentifier.Body.String(), response.Body.String())
		}
	})
	t.Run("batched children", func(t *testing.T) {
		response := runtimeRequest(runtime, http.MethodGet, "/api/issues/children?parent_ids="+parentID, "", headers)
		if response.Code != http.StatusOK || !containsJSON(response.Body.Bytes(), `"id":"`+childOneID+`"`, `"id":"`+childTwoID+`"`) {
			t.Fatalf("children by parents = %d %s", response.Code, response.Body.String())
		}
	})
	t.Run("child progress", func(t *testing.T) {
		response := runtimeRequest(runtime, http.MethodGet, "/api/issues/child-progress", "", headers)
		if response.Code != http.StatusOK || !containsJSON(response.Body.Bytes(), `"parent_issue_id":"`+parentID+`"`, `"total":2`, `"done":2`) {
			t.Fatalf("child progress = %d %s", response.Code, response.Body.String())
		}
	})
	t.Run("validates hidden parents and parent list", func(t *testing.T) {
		empty := runtimeRequest(runtime, http.MethodGet, "/api/issues/children?parent_ids=", "", headers)
		assertRuntimeResponse(t, empty.Code, empty.Body.String(), http.StatusBadRequest, `{"error":"invalid issue request"}`)
		missing := runtimeRequest(runtime, http.MethodGet, "/api/issues/missing/children", "", headers)
		assertRuntimeResponse(t, missing.Code, missing.Body.String(), http.StatusNotFound, `{"error":"issue not found"}`)

		otherWorkspace := runtimeRequest(runtime, http.MethodPost, "/api/workspaces", `{"name":"Other Hierarchy","slug":"other-hierarchy"}`, map[string]string{
			"Authorization": "Bearer " + login.Token, "Content-Type": "application/json",
		})
		if otherWorkspace.Code != http.StatusCreated {
			t.Fatalf("create other workspace = %d %s", otherWorkspace.Code, otherWorkspace.Body.String())
		}
		var otherWorkspaceBody struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(otherWorkspace.Body.Bytes(), &otherWorkspaceBody); err != nil || otherWorkspaceBody.ID == "" {
			t.Fatalf("decode other workspace: %v body=%s", err, otherWorkspace.Body.String())
		}
		otherHeaders := map[string]string{
			"Authorization": "Bearer " + login.Token, "X-Workspace-Slug": "other-hierarchy", "Content-Type": "application/json",
		}
		otherParent := runtimeRequest(runtime, http.MethodPost, "/api/issues", `{"title":"Other parent"}`, otherHeaders)
		if otherParent.Code != http.StatusCreated {
			t.Fatalf("create other parent = %d %s", otherParent.Code, otherParent.Body.String())
		}
		var other struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(otherParent.Body.Bytes(), &other); err != nil || other.ID == "" {
			t.Fatalf("decode other parent: %v body=%s", err, otherParent.Body.String())
		}
		foreign := runtimeRequest(runtime, http.MethodGet, "/api/issues/"+other.ID+"/children", "", headers)
		assertRuntimeResponse(t, foreign.Code, foreign.Body.String(), http.StatusNotFound, `{"error":"issue not found"}`)
		foreignBatch := runtimeRequest(runtime, http.MethodGet, "/api/issues/children?parent_ids="+parentID+","+other.ID, "", headers)
		assertRuntimeResponse(t, foreignBatch.Code, foreignBatch.Body.String(), http.StatusNotFound, `{"error":"issue not found"}`)
		foreignUpdate := runtimeRequest(runtime, http.MethodPost, "/api/issues/batch-update", `{"issue_ids":["`+other.ID+`"],"updates":{"priority":"urgent"}}`, headers)
		assertRuntimeResponse(t, foreignUpdate.Code, foreignUpdate.Body.String(), http.StatusNotFound, `{"error":"issue not found"}`)
		foreignDelete := runtimeRequest(runtime, http.MethodPost, "/api/issues/batch-delete", `{"issue_ids":["`+other.ID+`"]}`, headers)
		assertRuntimeResponse(t, foreignDelete.Code, foreignDelete.Body.String(), http.StatusNotFound, `{"error":"issue not found"}`)
		mismatch := runtimeRequest(runtime, http.MethodGet, "/api/issues/child-progress", "", map[string]string{
			"Authorization": "Bearer " + login.Token, "X-Workspace-ID": workspaceBody.ID, "X-Workspace-Slug": "other-hierarchy",
		})
		assertRuntimeResponse(t, mismatch.Code, mismatch.Body.String(), http.StatusNotFound, `{"error":"issue not found"}`)
		var foreignPriority string
		if err := runtime.Database().QueryRow(`SELECT priority FROM workspace_issues WHERE workspace_id=? AND id=?`, otherWorkspaceBody.ID, other.ID).Scan(&foreignPriority); err != nil || foreignPriority != "none" {
			t.Fatalf("foreign batch changed Issue: priority=%q err=%v", foreignPriority, err)
		}
		progress := runtimeRequest(runtime, http.MethodGet, "/api/issues/child-progress", "", headers)
		if strings.Contains(progress.Body.String(), other.ID) {
			t.Fatalf("foreign progress leaked: %s", progress.Body.String())
		}
	})
	t.Run("cookie batch mutation requires csrf", func(t *testing.T) {
		cookie := "multica_auth=" + login.Token + "; multica_csrf=" + login.CSRF
		withoutCSRF := runtimeRequest(runtime, http.MethodPost, "/api/issues/batch-update", `{"issue_ids":["`+childOneID+`"],"updates":{"priority":"high"}}`, map[string]string{
			"Cookie": cookie, "X-Workspace-Slug": "issue-hierarchy", "Content-Type": "application/json",
		})
		assertRuntimeResponse(t, withoutCSRF.Code, withoutCSRF.Body.String(), http.StatusForbidden, `{"error":"invalid CSRF token"}`)
		withCSRF := runtimeRequest(runtime, http.MethodPost, "/api/issues/batch-update", `{"issue_ids":["`+childOneID+`"],"updates":{"priority":"high"}}`, map[string]string{
			"Cookie": cookie, "X-CSRF-Token": login.CSRF, "X-Workspace-Slug": "issue-hierarchy", "Content-Type": "application/json",
		})
		assertRuntimeResponse(t, withCSRF.Code, withCSRF.Body.String(), http.StatusOK, `{"updated":1}`)

		cookieDeleteID := create("Cookie delete", nil, "todo", 40)
		deleteWithoutCSRF := runtimeRequest(runtime, http.MethodPost, "/api/issues/batch-delete", `{"issue_ids":["`+cookieDeleteID+`"]}`, map[string]string{
			"Cookie": cookie, "X-Workspace-Slug": "issue-hierarchy", "Content-Type": "application/json",
		})
		assertRuntimeResponse(t, deleteWithoutCSRF.Code, deleteWithoutCSRF.Body.String(), http.StatusForbidden, `{"error":"invalid CSRF token"}`)
		deleteWithCSRF := runtimeRequest(runtime, http.MethodPost, "/api/issues/batch-delete", `{"issue_ids":["`+cookieDeleteID+`"]}`, map[string]string{
			"Cookie": cookie, "X-CSRF-Token": login.CSRF, "X-Workspace-Slug": "issue-hierarchy", "Content-Type": "application/json",
		})
		assertRuntimeResponse(t, deleteWithCSRF.Code, deleteWithCSRF.Body.String(), http.StatusOK, `{"deleted":1}`)
	})
	t.Run("updates the full supported patch and clears nullable fields", func(t *testing.T) {
		targetID := create("Patch target", nil, "todo", 50)
		project := runtimeRequest(runtime, http.MethodPost, "/api/projects", `{"title":"Batch project"}`, headers)
		if project.Code != http.StatusCreated {
			t.Fatalf("create project = %d %s", project.Code, project.Body.String())
		}
		var projectBody struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(project.Body.Bytes(), &projectBody); err != nil || projectBody.ID == "" {
			t.Fatalf("decode project: %v body=%s", err, project.Body.String())
		}
		updated := runtimeRequest(runtime, http.MethodPost, "/api/issues/batch-update", `{"issue_ids":["`+targetID+`"],"updates":{"title":"Patched title","description":"Patched description","status":"in_review","priority":"high","assignee_type":"member","assignee_id":"`+login.UserID+`","parent_issue_id":"`+parentIdentifier+`","project_id":"`+projectBody.ID+`","stage":2,"start_date":"2026-08-14","due_date":"2026-08-31"}}`, headers)
		assertRuntimeResponse(t, updated.Code, updated.Body.String(), http.StatusOK, `{"updated":1}`)
		readback := runtimeRequest(runtime, http.MethodGet, "/api/issues/"+targetID, "", headers)
		if readback.Code != http.StatusOK || !containsJSON(readback.Body.Bytes(), `"title":"Patched title"`, `"description":"Patched description"`, `"status":"in_review"`, `"priority":"high"`, `"assignee_type":"member"`, `"assignee_id":"`+login.UserID+`"`, `"parent_issue_id":"`+parentID+`"`, `"project_id":"`+projectBody.ID+`"`, `"stage":2`, `"start_date":"2026-08-14"`, `"due_date":"2026-08-31"`) {
			t.Fatalf("batch patch readback = %d %s", readback.Code, readback.Body.String())
		}
		cleared := runtimeRequest(runtime, http.MethodPost, "/api/issues/batch-update", `{"issue_ids":["`+targetID+`"],"updates":{"assignee_type":null,"assignee_id":null,"parent_issue_id":null,"project_id":null,"stage":null,"start_date":null,"due_date":null}}`, headers)
		assertRuntimeResponse(t, cleared.Code, cleared.Body.String(), http.StatusOK, `{"updated":1}`)
		clearedReadback := runtimeRequest(runtime, http.MethodGet, "/api/issues/"+targetID, "", headers)
		if clearedReadback.Code != http.StatusOK || !containsJSON(clearedReadback.Body.Bytes(), `"assignee_type":null`, `"assignee_id":null`, `"parent_issue_id":null`, `"project_id":null`, `"stage":null`, `"start_date":null`, `"due_date":null`) {
			t.Fatalf("batch clear readback = %d %s", clearedReadback.Code, clearedReadback.Body.String())
		}
	})
	t.Run("strict batch validation", func(t *testing.T) {
		cases := []struct {
			name string
			body string
			want int
		}{
			{name: "empty ids", body: `{"issue_ids":[],"updates":{"priority":"urgent"}}`, want: http.StatusBadRequest},
			{name: "duplicate token", body: `{"issue_ids":["` + childOneID + `","` + childOneID + `"],"updates":{"priority":"urgent"}}`, want: http.StatusBadRequest},
			{name: "resolved duplicate", body: `{"issue_ids":["` + childOneID + `","` + childOneIdentifier + `"],"updates":{"priority":"urgent"}}`, want: http.StatusConflict},
			{name: "position unsupported", body: `{"issue_ids":["` + childOneID + `"],"updates":{"position":42}}`, want: http.StatusBadRequest},
			{name: "invalid status", body: `{"issue_ids":["` + childOneID + `"],"updates":{"status":"invented"}}`, want: http.StatusBadRequest},
			{name: "invalid date", body: `{"issue_ids":["` + childOneID + `"],"updates":{"due_date":"tomorrow"}}`, want: http.StatusBadRequest},
			{name: "invalid stage", body: `{"issue_ids":["` + childOneID + `"],"updates":{"stage":-1}}`, want: http.StatusBadRequest},
			{name: "unpaired assignee", body: `{"issue_ids":["` + childOneID + `"],"updates":{"assignee_id":"` + login.UserID + `"}}`, want: http.StatusBadRequest},
			{name: "missing parent", body: `{"issue_ids":["` + childOneID + `"],"updates":{"parent_issue_id":"missing"}}`, want: http.StatusNotFound},
			{name: "missing project", body: `{"issue_ids":["` + childOneID + `"],"updates":{"project_id":"missing"}}`, want: http.StatusNotFound},
			{name: "unknown top field", body: `{"issue_ids":["` + childOneID + `"],"updates":{"priority":"urgent"},"unknown":true}`, want: http.StatusBadRequest},
			{name: "unknown update field", body: `{"issue_ids":["` + childOneID + `"],"updates":{"unknown":true}}`, want: http.StatusBadRequest},
			{name: "trailing json", body: `{"issue_ids":["` + childOneID + `"],"updates":{"priority":"urgent"}} {}`, want: http.StatusBadRequest},
		}
		for _, probe := range cases {
			t.Run(probe.name, func(t *testing.T) {
				response := runtimeRequest(runtime, http.MethodPost, "/api/issues/batch-update", probe.body, headers)
				if response.Code != probe.want {
					t.Fatalf("response = %d %s, want %d", response.Code, response.Body.String(), probe.want)
				}
			})
		}
		oversized := runtimeRequest(runtime, http.MethodPost, "/api/issues/batch-update", `{"issue_ids":["`+childOneID+`"],"updates":{"title":"`+strings.Repeat("x", (1<<20)+1)+`"}}`, headers)
		if oversized.Code != http.StatusBadRequest {
			t.Fatalf("oversized = %d %s", oversized.Code, oversized.Body.String())
		}
		tooMany := make([]string, 201)
		for index := range tooMany {
			tooMany[index] = fmt.Sprintf("missing-%d", index)
		}
		encoded, err := json.Marshal(map[string]any{"issue_ids": tooMany, "updates": map[string]any{"priority": "urgent"}})
		if err != nil {
			t.Fatal(err)
		}
		tooManyResponse := runtimeRequest(runtime, http.MethodPost, "/api/issues/batch-update", string(encoded), headers)
		if tooManyResponse.Code != http.StatusBadRequest {
			t.Fatalf("too many = %d %s", tooManyResponse.Code, tooManyResponse.Body.String())
		}
		noOp := runtimeRequest(runtime, http.MethodPost, "/api/issues/batch-update", `{"issue_ids":["missing"],"updates":{}}`, headers)
		assertRuntimeResponse(t, noOp.Code, noOp.Body.String(), http.StatusOK, `{"updated":0}`)

		deleteCases := []struct {
			name string
			body string
			want int
		}{
			{name: "empty delete ids", body: `{"issue_ids":[]}`, want: http.StatusBadRequest},
			{name: "duplicate delete token", body: `{"issue_ids":["` + childOneID + `","` + childOneID + `"]}`, want: http.StatusBadRequest},
			{name: "resolved duplicate delete", body: `{"issue_ids":["` + childOneID + `","` + childOneIdentifier + `"]}`, want: http.StatusConflict},
			{name: "unknown delete field", body: `{"issue_ids":["` + childOneID + `"],"unknown":true}`, want: http.StatusBadRequest},
			{name: "trailing delete json", body: `{"issue_ids":["` + childOneID + `"]} {}`, want: http.StatusBadRequest},
		}
		for _, probe := range deleteCases {
			t.Run(probe.name, func(t *testing.T) {
				response := runtimeRequest(runtime, http.MethodPost, "/api/issues/batch-delete", probe.body, headers)
				if response.Code != probe.want {
					t.Fatalf("response = %d %s, want %d", response.Code, response.Body.String(), probe.want)
				}
			})
		}
		oversizedDelete := runtimeRequest(runtime, http.MethodPost, "/api/issues/batch-delete", `{"issue_ids":["`+strings.Repeat("x", (1<<20)+1)+`"]}`, headers)
		if oversizedDelete.Code != http.StatusBadRequest {
			t.Fatalf("oversized delete = %d %s", oversizedDelete.Code, oversizedDelete.Body.String())
		}
	})
	t.Run("expired identity is rejected", func(t *testing.T) {
		expired := verifyRuntimeLogin(t, runtime, "expired-hierarchy@example.com")
		if _, err := runtime.Database().Exec(`UPDATE auth_sessions SET expires_at_unix_nano=0 WHERE user_id=?`, expired.UserID); err != nil {
			t.Fatal(err)
		}
		response := runtimeRequest(runtime, http.MethodGet, "/api/issues/child-progress", "", map[string]string{
			"Authorization": "Bearer " + expired.Token, "X-Workspace-Slug": "issue-hierarchy",
		})
		assertRuntimeResponse(t, response.Code, response.Body.String(), http.StatusUnauthorized, `{"error":"user not authenticated"}`)
	})
	t.Run("batch update", func(t *testing.T) {
		response := runtimeRequest(runtime, http.MethodPost, "/api/issues/batch-update", `{"issue_ids":["`+childOneID+`","`+childTwoID+`"],"updates":{"priority":"urgent"}}`, headers)
		assertRuntimeResponse(t, response.Code, response.Body.String(), http.StatusOK, `{"updated":2}`)
	})
	t.Run("batch delete", func(t *testing.T) {
		response := runtimeRequest(runtime, http.MethodPost, "/api/issues/batch-delete", `{"issue_ids":["`+childTwoID+`"]}`, headers)
		assertRuntimeResponse(t, response.Code, response.Body.String(), http.StatusOK, `{"deleted":1}`)
	})

	config := runtimeRequest(runtime, http.MethodGet, "/api/config", "", nil)
	if config.Code != http.StatusOK || !containsJSON(config.Body.Bytes(), `"issue_children":true`, `"issue_child_progress":true`, `"issue_batch":true`) {
		t.Fatalf("hierarchy capabilities = %d %s", config.Code, config.Body.String())
	}
}

func TestSQLiteRuntimeRollsBackIssueBatchesAndPublishesOnlyCommittedEvents(t *testing.T) {
	runtime := newRuntimeForConfig(t, Config{
		Name: "backend-test", Version: "test",
		HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath:            filepath.Join(t.TempDir(), "issue-batch-events.db"),
		WorkspaceDependencies: FailClosedWorkspaceDependencies(),
		LocalAuth:             auth.LocalAuthConfig{VerificationCode: "888888"},
	})
	login := verifyRuntimeLogin(t, runtime, "issue-batch-events@example.com")
	workspace := runtimeRequest(runtime, http.MethodPost, "/api/workspaces", `{"name":"Batch Events","slug":"batch-events"}`, map[string]string{
		"Authorization": "Bearer " + login.Token, "Content-Type": "application/json",
	})
	if workspace.Code != http.StatusCreated {
		t.Fatalf("create workspace = %d %s", workspace.Code, workspace.Body.String())
	}
	var workspaceBody struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(workspace.Body.Bytes(), &workspaceBody); err != nil || workspaceBody.ID == "" {
		t.Fatalf("decode workspace: %v body=%s", err, workspace.Body.String())
	}
	headers := map[string]string{
		"Authorization": "Bearer " + login.Token, "X-Workspace-Slug": "batch-events", "Content-Type": "application/json",
	}
	type createdIssue struct{ ID, Identifier string }
	create := func(title string, parentID *string) createdIssue {
		t.Helper()
		parent := "null"
		if parentID != nil {
			parent = fmt.Sprintf("%q", *parentID)
		}
		response := runtimeRequest(runtime, http.MethodPost, "/api/issues", fmt.Sprintf(`{"title":%q,"parent_issue_id":%s}`, title, parent), headers)
		if response.Code != http.StatusCreated {
			t.Fatalf("create %s = %d %s", title, response.Code, response.Body.String())
		}
		var body createdIssue
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body.ID == "" || body.Identifier == "" {
			t.Fatalf("decode %s: %v body=%s", title, err, response.Body.String())
		}
		return body
	}
	updateOne := create("Update one", nil)
	updateTwo := create("Update two", nil)

	server := httptest.NewServer(runtime.HTTPServer())
	t.Cleanup(server.Close)
	failureSocket := dialTokenRealtime(t, server.URL, "batch-events", login.Token)
	if _, err := runtime.Database().Exec(`CREATE TRIGGER fail_issue_batch_update BEFORE UPDATE ON workspace_issues WHEN OLD.id='` + updateTwo.ID + `' BEGIN SELECT RAISE(ABORT,'forced batch update rollback'); END`); err != nil {
		t.Fatal(err)
	}
	failedUpdate := runtimeRequest(runtime, http.MethodPost, "/api/issues/batch-update", `{"issue_ids":["`+updateOne.ID+`","`+updateTwo.ID+`"],"updates":{"priority":"urgent"}}`, headers)
	assertRuntimeResponse(t, failedUpdate.Code, failedUpdate.Body.String(), http.StatusInternalServerError, `{"error":"failed to batch update issues"}`)
	assertNoRealtimeEvent(t, failureSocket)
	_ = failureSocket.Close()
	for _, issueID := range []string{updateOne.ID, updateTwo.ID} {
		var priority string
		if err := runtime.Database().QueryRow(`SELECT priority FROM workspace_issues WHERE id=?`, issueID).Scan(&priority); err != nil || priority != "none" {
			t.Fatalf("rolled back priority for %s = %q err=%v", issueID, priority, err)
		}
	}
	if _, err := runtime.Database().Exec(`DROP TRIGGER fail_issue_batch_update`); err != nil {
		t.Fatal(err)
	}

	cycleParent := create("Cycle parent", nil)
	cycleChild := create("Cycle child", &cycleParent.ID)
	cycleSocket := dialTokenRealtime(t, server.URL, "batch-events", login.Token)
	cycle := runtimeRequest(runtime, http.MethodPost, "/api/issues/batch-update", `{"issue_ids":["`+cycleParent.ID+`"],"updates":{"parent_issue_id":"`+cycleChild.ID+`"}}`, headers)
	if cycle.Code != http.StatusBadRequest {
		t.Fatalf("cycle = %d %s", cycle.Code, cycle.Body.String())
	}
	assertNoRealtimeEvent(t, cycleSocket)
	_ = cycleSocket.Close()
	var cycleParentValue *string
	if err := runtime.Database().QueryRow(`SELECT parent_issue_id FROM workspace_issues WHERE id=?`, cycleParent.ID).Scan(&cycleParentValue); err != nil || cycleParentValue != nil {
		t.Fatalf("cycle parent changed: value=%v err=%v", cycleParentValue, err)
	}

	successSocket := dialTokenRealtime(t, server.URL, "batch-events", login.Token)
	successUpdate := runtimeRequest(runtime, http.MethodPost, "/api/issues/batch-update", `{"issue_ids":["`+updateOne.ID+`","`+updateTwo.Identifier+`"],"updates":{"priority":"urgent","status":"in_progress"}}`, headers)
	assertRuntimeResponse(t, successUpdate.Code, successUpdate.Body.String(), http.StatusOK, `{"updated":2}`)
	assertRealtimeEvent(t, successSocket, "issue:updated", `"id":"`+updateOne.ID+`"`)
	assertRealtimeEvent(t, successSocket, "activity:created", `"action":"status_changed"`)
	assertRealtimeEvent(t, successSocket, "activity:created", `"action":"priority_changed"`)
	assertRealtimeEvent(t, successSocket, "issue:updated", `"id":"`+updateTwo.ID+`"`)
	assertRealtimeEvent(t, successSocket, "activity:created", `"action":"status_changed"`)
	assertRealtimeEvent(t, successSocket, "activity:created", `"action":"priority_changed"`)
	_ = successSocket.Close()

	deleteOne := create("Delete one", nil)
	deleteTwo := create("Delete two", nil)
	dependent := create("Dependent child", &deleteOne.ID)
	retained := create("Retained Issue", nil)
	project := runtimeRequest(runtime, http.MethodPost, "/api/projects", `{"title":"Requirement project"}`, headers)
	if project.Code != http.StatusCreated {
		t.Fatalf("create project = %d %s", project.Code, project.Body.String())
	}
	var projectBody struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(project.Body.Bytes(), &projectBody); err != nil || projectBody.ID == "" {
		t.Fatalf("decode project: %v body=%s", err, project.Body.String())
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := runtime.Database().Exec(`INSERT INTO workspace_todos(id,workspace_id,title,status,issue_id,created_at,updated_at) VALUES(?,?,?,?,?,?,?)`, "batch-todo", workspaceBody.ID, "Batch Todo", "todo", deleteOne.Identifier, now, now); err != nil {
		t.Fatal(err)
	}
	for _, requirement := range []struct {
		id       string
		issueIDs string
	}{
		{id: "batch-requirement-retained", issueIDs: fmt.Sprintf(`[%q,%q,%q]`, deleteOne.Identifier, deleteTwo.ID, retained.ID)},
		{id: "batch-requirement-empty", issueIDs: fmt.Sprintf(`[%q,%q]`, deleteOne.ID, deleteTwo.Identifier)},
	} {
		if _, err := runtime.Database().Exec(`INSERT INTO workspace_requirements(id,workspace_id,project_id,title,current_version,approval_status,coverage_status,issue_ids,created_at,updated_at) VALUES(?,?,?,?,1,'draft','covered',?,?,?)`, requirement.id, workspaceBody.ID, projectBody.ID, requirement.id, requirement.issueIDs, now, now); err != nil {
			t.Fatal(err)
		}
		if _, err := runtime.Database().Exec(`INSERT INTO workspace_requirement_versions(id,requirement_id,version,content,created_at) VALUES(?,?,?,?,?)`, requirement.id+"-v1", requirement.id, 1, "retained content", now); err != nil {
			t.Fatal(err)
		}
	}
	pin := runtimeRequest(runtime, http.MethodPost, "/api/pins", `{"item_type":"issue","item_id":"`+deleteOne.ID+`"}`, headers)
	if pin.Code != http.StatusCreated {
		t.Fatalf("create Issue pin = %d %s", pin.Code, pin.Body.String())
	}

	deleteFailureSocket := dialTokenRealtime(t, server.URL, "batch-events", login.Token)
	if _, err := runtime.Database().Exec(`CREATE TRIGGER fail_issue_batch_delete BEFORE DELETE ON workspace_issues WHEN OLD.id='` + deleteTwo.ID + `' BEGIN SELECT RAISE(ABORT,'forced batch delete rollback'); END`); err != nil {
		t.Fatal(err)
	}
	failedDelete := runtimeRequest(runtime, http.MethodPost, "/api/issues/batch-delete", `{"issue_ids":["`+deleteOne.ID+`","`+deleteTwo.Identifier+`"]}`, headers)
	assertRuntimeResponse(t, failedDelete.Code, failedDelete.Body.String(), http.StatusInternalServerError, `{"error":"failed to batch delete issues"}`)
	assertNoRealtimeEvent(t, deleteFailureSocket)
	_ = deleteFailureSocket.Close()
	assertBatchDeleteFixture(t, runtime, deleteOne, deleteTwo, dependent.ID, retained.ID, true)
	if _, err := runtime.Database().Exec(`DROP TRIGGER fail_issue_batch_delete`); err != nil {
		t.Fatal(err)
	}

	deleteSuccessSocket := dialTokenRealtime(t, server.URL, "batch-events", login.Token)
	successDelete := runtimeRequest(runtime, http.MethodPost, "/api/issues/batch-delete", `{"issue_ids":["`+deleteOne.ID+`","`+deleteTwo.Identifier+`"]}`, headers)
	assertRuntimeResponse(t, successDelete.Code, successDelete.Body.String(), http.StatusOK, `{"deleted":2}`)
	assertRealtimeEvent(t, deleteSuccessSocket, "issue:deleted", `"issue_id":"`+deleteOne.ID+`"`)
	assertRealtimeEvent(t, deleteSuccessSocket, "issue:deleted", `"issue_id":"`+deleteTwo.ID+`"`)
	_ = deleteSuccessSocket.Close()
	assertBatchDeleteFixture(t, runtime, deleteOne, deleteTwo, dependent.ID, retained.ID, false)
}

func assertBatchDeleteFixture(t *testing.T, runtime *Runtime, deleteOne, deleteTwo struct{ ID, Identifier string }, dependentID, retainedID string, rolledBack bool) {
	t.Helper()
	var issueCount, pinCount int
	if err := runtime.Database().QueryRow(`SELECT COUNT(*) FROM workspace_issues WHERE id IN (?,?)`, deleteOne.ID, deleteTwo.ID).Scan(&issueCount); err != nil {
		t.Fatal(err)
	}
	if err := runtime.Database().QueryRow(`SELECT COUNT(*) FROM workspace_pins WHERE item_type='issue' AND item_id=?`, deleteOne.ID).Scan(&pinCount); err != nil {
		t.Fatal(err)
	}
	var childParent, todoIssue *string
	if err := runtime.Database().QueryRow(`SELECT parent_issue_id FROM workspace_issues WHERE id=?`, dependentID).Scan(&childParent); err != nil {
		t.Fatal(err)
	}
	if err := runtime.Database().QueryRow(`SELECT issue_id FROM workspace_todos WHERE id='batch-todo'`).Scan(&todoIssue); err != nil {
		t.Fatal(err)
	}
	type requirementState struct {
		version  int
		coverage string
		issueIDs string
		versions int
	}
	readRequirement := func(id string) requirementState {
		t.Helper()
		var state requirementState
		if err := runtime.Database().QueryRow(`SELECT current_version,coverage_status,issue_ids FROM workspace_requirements WHERE id=?`, id).Scan(&state.version, &state.coverage, &state.issueIDs); err != nil {
			t.Fatal(err)
		}
		if err := runtime.Database().QueryRow(`SELECT COUNT(*) FROM workspace_requirement_versions WHERE requirement_id=?`, id).Scan(&state.versions); err != nil {
			t.Fatal(err)
		}
		return state
	}
	retainedRequirement := readRequirement("batch-requirement-retained")
	emptyRequirement := readRequirement("batch-requirement-empty")
	if rolledBack {
		if issueCount != 2 || pinCount != 1 || childParent == nil || *childParent != deleteOne.ID || todoIssue == nil || *todoIssue != deleteOne.Identifier || retainedRequirement.version != 1 || retainedRequirement.versions != 1 || emptyRequirement.version != 1 || emptyRequirement.versions != 1 {
			t.Fatalf("batch delete rollback lost data: issues=%d pins=%d parent=%v todo=%v retained=%+v empty=%+v", issueCount, pinCount, childParent, todoIssue, retainedRequirement, emptyRequirement)
		}
		return
	}
	if issueCount != 0 || pinCount != 0 || childParent != nil || todoIssue != nil {
		t.Fatalf("batch delete cleanup: issues=%d pins=%d parent=%v todo=%v", issueCount, pinCount, childParent, todoIssue)
	}
	var retainedIDs []string
	if err := json.Unmarshal([]byte(retainedRequirement.issueIDs), &retainedIDs); err != nil {
		t.Fatal(err)
	}
	var emptyIDs []string
	if err := json.Unmarshal([]byte(emptyRequirement.issueIDs), &emptyIDs); err != nil {
		t.Fatal(err)
	}
	if retainedRequirement.version != 2 || retainedRequirement.versions != 2 || retainedRequirement.coverage != "covered" || len(retainedIDs) != 1 || retainedIDs[0] != retainedID {
		t.Fatalf("retained requirement cleanup = %+v ids=%v", retainedRequirement, retainedIDs)
	}
	if emptyRequirement.version != 2 || emptyRequirement.versions != 2 || emptyRequirement.coverage != "uncovered" || len(emptyIDs) != 0 {
		t.Fatalf("empty requirement cleanup = %+v ids=%v", emptyRequirement, emptyIDs)
	}
}

func TestSQLiteRuntimeSerializesIssueBatchesAndPersistsHierarchyRestart(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "issue-batch-concurrent.db")
	config := Config{
		Name: "backend-test", Version: "test",
		HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath:            databasePath,
		WorkspaceDependencies: FailClosedWorkspaceDependencies(),
		LocalAuth:             auth.LocalAuthConfig{VerificationCode: "888888"},
	}
	runtime := newRuntimeForConfig(t, config)
	login := verifyRuntimeLogin(t, runtime, "issue-batch-concurrent@example.com")
	workspace := runtimeRequest(runtime, http.MethodPost, "/api/workspaces", `{"name":"Concurrent Batch","slug":"concurrent-batch"}`, map[string]string{
		"Authorization": "Bearer " + login.Token, "Content-Type": "application/json",
	})
	if workspace.Code != http.StatusCreated {
		t.Fatalf("create workspace = %d %s", workspace.Code, workspace.Body.String())
	}
	headers := map[string]string{
		"Authorization": "Bearer " + login.Token, "X-Workspace-Slug": "concurrent-batch", "Content-Type": "application/json",
	}
	create := func(title string, parentID *string) string {
		t.Helper()
		parent := "null"
		if parentID != nil {
			parent = fmt.Sprintf("%q", *parentID)
		}
		response := runtimeRequest(runtime, http.MethodPost, "/api/issues", fmt.Sprintf(`{"title":%q,"parent_issue_id":%s}`, title, parent), headers)
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
	parentID := create("Concurrent parent", nil)
	firstID := create("Concurrent first", &parentID)
	secondID := create("Concurrent second", &parentID)
	if _, err := runtime.Database().Exec(`UPDATE workspace_issues SET metadata='{"retained":true}',properties='{"estimate":3}' WHERE id IN (?,?)`, firstID, secondID); err != nil {
		t.Fatal(err)
	}

	type result struct {
		code int
		body string
	}
	results := make(chan result, 2)
	start := make(chan struct{})
	var workers sync.WaitGroup
	for _, updates := range []string{
		`{"status":"done","priority":"high"}`,
		`{"status":"blocked","priority":"urgent"}`,
	} {
		workers.Add(1)
		go func(updates string) {
			defer workers.Done()
			<-start
			response := runtimeRequest(runtime, http.MethodPost, "/api/issues/batch-update", `{"issue_ids":["`+firstID+`","`+secondID+`"],"updates":`+updates+`}`, headers)
			results <- result{code: response.Code, body: response.Body.String()}
		}(updates)
	}
	close(start)
	workers.Wait()
	close(results)
	for response := range results {
		if response.code != http.StatusOK || strings.TrimSpace(response.body) != `{"updated":2}` {
			t.Fatalf("concurrent batch update = %d %s", response.code, response.body)
		}
	}
	type state struct{ status, priority, metadata, properties string }
	states := make([]state, 0, 2)
	for _, issueID := range []string{firstID, secondID} {
		var value state
		if err := runtime.Database().QueryRow(`SELECT status,priority,metadata,properties FROM workspace_issues WHERE id=?`, issueID).Scan(&value.status, &value.priority, &value.metadata, &value.properties); err != nil {
			t.Fatal(err)
		}
		states = append(states, value)
	}
	if states[0].status != states[1].status || states[0].priority != states[1].priority || !((states[0].status == "done" && states[0].priority == "high") || (states[0].status == "blocked" && states[0].priority == "urgent")) {
		t.Fatalf("concurrent batches mixed commits: %+v", states)
	}
	for _, value := range states {
		if value.metadata != `{"retained":true}` || value.properties != `{"estimate":3}` {
			t.Fatalf("batch update overwrote unrelated bags: %+v", value)
		}
	}

	missingDelete := runtimeRequest(runtime, http.MethodPost, "/api/issues/batch-delete", `{"issue_ids":["`+firstID+`","missing"]}`, headers)
	assertRuntimeResponse(t, missingDelete.Code, missingDelete.Body.String(), http.StatusNotFound, `{"error":"issue not found"}`)
	var retainedCount int
	if err := runtime.Database().QueryRow(`SELECT COUNT(*) FROM workspace_issues WHERE id IN (?,?)`, firstID, secondID).Scan(&retainedCount); err != nil || retainedCount != 2 {
		t.Fatalf("missing-target batch delete was partial: count=%d err=%v", retainedCount, err)
	}

	deleteOne := create("Concurrent delete one", nil)
	deleteTwo := create("Concurrent delete two", nil)
	deleteResults := make(chan result, 2)
	deleteStart := make(chan struct{})
	for index := 0; index < 2; index++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-deleteStart
			response := runtimeRequest(runtime, http.MethodPost, "/api/issues/batch-delete", `{"issue_ids":["`+deleteOne+`","`+deleteTwo+`"]}`, headers)
			deleteResults <- result{code: response.Code, body: response.Body.String()}
		}()
	}
	close(deleteStart)
	workers.Wait()
	close(deleteResults)
	statusCounts := map[int]int{}
	for response := range deleteResults {
		statusCounts[response.code]++
	}
	if statusCounts[http.StatusOK] != 1 || statusCounts[http.StatusNotFound] != 1 {
		t.Fatalf("concurrent batch delete statuses = %#v", statusCounts)
	}

	if err := runtime.Close(); err != nil {
		t.Fatal(err)
	}
	restarted := newRuntimeForConfig(t, config)
	children := runtimeRequest(restarted, http.MethodGet, "/api/issues/"+parentID+"/children", "", headers)
	if children.Code != http.StatusOK || !containsJSON(children.Body.Bytes(), `"id":"`+firstID+`"`, `"id":"`+secondID+`"`) {
		t.Fatalf("restart children = %d %s", children.Code, children.Body.String())
	}
	readback := runtimeRequest(restarted, http.MethodGet, "/api/issues/"+firstID, "", headers)
	if readback.Code != http.StatusOK || !containsJSON(readback.Body.Bytes(), `"status":"`+states[0].status+`"`, `"priority":"`+states[0].priority+`"`, `"metadata":{"retained":true}`, `"properties":{"estimate":3}`) {
		t.Fatalf("restart batch readback = %d %s", readback.Code, readback.Body.String())
	}
}
