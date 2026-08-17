package bootstrap

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/auth"
)

func TestSQLiteRuntimeServesInstalledTaskSlice(t *testing.T) {
	now := time.Date(2026, 8, 18, 1, 2, 3, 0, time.UTC)
	sequence := 0
	runtime := newRuntimeForConfig(t, Config{
		Name: "backend-test", Version: "test",
		HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath:            filepath.Join(t.TempDir(), "tasks.db"),
		WorkspaceDependencies: FailClosedWorkspaceDependencies(),
		LocalAuth: auth.LocalAuthConfig{
			VerificationCode: "888888", SessionTTL: time.Hour, Now: func() time.Time { return now },
			NewID: func(context.Context) (string, error) {
				sequence++
				return []string{"user-one-token", "user-two-token"}[sequence-1], nil
			},
		},
	})
	userID := verifyRuntimeUser(t, runtime, "one@example.com")
	userTwoID := verifyRuntimeUser(t, runtime, "two@example.com")
	for _, statement := range []string{
		`INSERT INTO workspaces(id,name,slug,settings,repos,issue_prefix,created_at,updated_at) VALUES
			('workspace-one','One','one','{}','[]','ONE','2026-08-18T00:00:00Z','2026-08-18T00:00:00Z')`,
		`INSERT INTO auth_members(id,workspace_id,user_id,role,created_at) VALUES
			('member-one','workspace-one','` + userID + `','member','2026-08-18T00:00:00Z'),
			('member-two','workspace-one','` + userTwoID + `','member','2026-08-18T00:00:00Z')`,
		`INSERT INTO workspace_projects(id,workspace_id,name,status,created_at,updated_at) VALUES
			('project-one','workspace-one','Release 1','in_progress','2026-08-18T00:00:00Z','2026-08-18T00:00:00Z')`,
		`INSERT INTO workspace_project_actor_relations(workspace_id,project_id,actor_type,actor_id,role) VALUES
			('workspace-one','project-one','agent','agent-one','agent')`,
	} {
		if _, err := runtime.Database().Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	headers := map[string]string{"Authorization": "Bearer user-one-token", "X-Workspace-Slug": "one", "Idempotency-Key": "create-ship-s02a"}
	headersTwo := map[string]string{"Authorization": "Bearer user-two-token", "X-Workspace-Slug": "one"}
	unauthorized := runtimeRequest(runtime, http.MethodGet, "/api/tasks", "", nil)
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized list = %d %s", unauthorized.Code, unauthorized.Body.String())
	}

	created := runtimeRequest(runtime, http.MethodPost, "/api/tasks", `{"title":"Ship S02A","due_date":"2026-08-20"}`, headers)
	if created.Code != http.StatusCreated {
		t.Fatalf("create = %d %s", created.Code, created.Body.String())
	}
	var task struct {
		ID          string `json:"id"`
		WorkspaceID string `json:"workspace_id"`
		Status      string `json:"status"`
		Revision    int64  `json:"revision"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &task); err != nil {
		t.Fatal(err)
	}
	if task.ID == "" || task.WorkspaceID != "workspace-one" || task.Status != "todo" || task.Revision != 1 {
		t.Fatalf("created body = %s", created.Body.String())
	}
	replayed := runtimeRequest(runtime, http.MethodPost, "/api/tasks", `{"title":"Ship S02A","due_date":"2026-08-20"}`, headers)
	if replayed.Code != http.StatusCreated || replayed.Body.String() != created.Body.String() {
		t.Fatalf("idempotent replay = %d %s, want original %s", replayed.Code, replayed.Body.String(), created.Body.String())
	}
	conflictingReplay := runtimeRequest(runtime, http.MethodPost, "/api/tasks", `{"title":"Different body"}`, headers)
	if conflictingReplay.Code != http.StatusConflict || !containsJSON(conflictingReplay.Body.Bytes(), `"code":"idempotency_conflict"`) {
		t.Fatalf("idempotency conflict = %d %s", conflictingReplay.Code, conflictingReplay.Body.String())
	}
	headers["Idempotency-Key"] = "create-agent-task"
	assigned := runtimeRequest(runtime, http.MethodPost, "/api/tasks", `{"title":"Agent task","project_id":"project-one","assignee_type":"agent","assignee_id":"agent-one"}`, headers)
	if assigned.Code != http.StatusCreated || !containsJSON(assigned.Body.Bytes(), `"assignee_type":"agent"`, `"assignee_id":"agent-one"`) {
		t.Fatalf("assigned create = %d %s", assigned.Code, assigned.Body.String())
	}
	foreignUpdate := runtimeRequest(runtime, http.MethodPatch, "/api/tasks/"+task.ID, `{"title":"Taken over","expected_revision":1}`, headersTwo)
	if foreignUpdate.Code != http.StatusForbidden {
		t.Fatalf("foreign member update = %d %s", foreignUpdate.Code, foreignUpdate.Body.String())
	}
	inProgress := runtimeRequest(runtime, http.MethodPatch, "/api/tasks/"+task.ID, `{"status":"in_progress","expected_revision":1}`, headers)
	if inProgress.Code != http.StatusOK || !containsJSON(inProgress.Body.Bytes(), `"status":"in_progress"`, `"revision":2`) {
		t.Fatalf("in progress = %d %s", inProgress.Code, inProgress.Body.String())
	}
	done := runtimeRequest(runtime, http.MethodPatch, "/api/tasks/"+task.ID, `{"status":"done","expected_revision":2}`, headers)
	if done.Code != http.StatusOK || !containsJSON(done.Body.Bytes(), `"status":"done"`, `"revision":3`) {
		t.Fatalf("done = %d %s", done.Code, done.Body.String())
	}
	stale := runtimeRequest(runtime, http.MethodPatch, "/api/tasks/"+task.ID, `{"title":"Stale","expected_revision":2}`, headers)
	if stale.Code != http.StatusConflict || !containsJSON(stale.Body.Bytes(), `"code":"revision_conflict"`, `"current_revision":3`) {
		t.Fatalf("stale = %d %s", stale.Code, stale.Body.String())
	}
	archived := runtimeRequest(runtime, http.MethodDelete, "/api/tasks/"+task.ID, `{"expected_revision":3}`, headers)
	if archived.Code != http.StatusNoContent {
		t.Fatalf("archive = %d %s", archived.Code, archived.Body.String())
	}
	restored := runtimeRequest(runtime, http.MethodPost, "/api/tasks/"+task.ID+"/restore", `{"expected_revision":4}`, headers)
	if restored.Code != http.StatusOK || !containsJSON(restored.Body.Bytes(), `"status":"done"`, `"revision":5`) {
		t.Fatalf("restore = %d %s", restored.Code, restored.Body.String())
	}
	var assignedBody struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(assigned.Body.Bytes(), &assignedBody); err != nil {
		t.Fatal(err)
	}
	reordered := runtimeRequest(runtime, http.MethodPost, "/api/tasks/reorder", `{"items":[{"id":"`+assignedBody.ID+`","position":10,"expected_revision":1},{"id":"`+task.ID+`","position":20,"expected_revision":5}]}`, headers)
	if reordered.Code != http.StatusOK || !containsJSON(reordered.Body.Bytes(), `"revision":2`, `"revision":6`) {
		t.Fatalf("reorder = %d %s", reordered.Code, reordered.Body.String())
	}
	failedReorder := runtimeRequest(runtime, http.MethodPost, "/api/tasks/reorder", `{"items":[{"id":"`+assignedBody.ID+`","position":30,"expected_revision":2},{"id":"`+task.ID+`","position":40,"expected_revision":5}]}`, headers)
	if failedReorder.Code != http.StatusConflict || !containsJSON(failedReorder.Body.Bytes(), `"current_revision":6`) {
		t.Fatalf("failed reorder = %d %s", failedReorder.Code, failedReorder.Body.String())
	}
	unchanged := runtimeRequest(runtime, http.MethodGet, "/api/tasks/"+assignedBody.ID, "", headers)
	if unchanged.Code != http.StatusOK || !containsJSON(unchanged.Body.Bytes(), `"position":10`, `"revision":2`) {
		t.Fatalf("reorder rollback = %d %s", unchanged.Code, unchanged.Body.String())
	}
	afterReorderUpdate := runtimeRequest(runtime, http.MethodPatch, "/api/tasks/"+task.ID, `{"title":"Ship S02A after reorder","expected_revision":6}`, headers)
	if afterReorderUpdate.Code != http.StatusOK || !containsJSON(afterReorderUpdate.Body.Bytes(), `"title":"Ship S02A after reorder"`, `"revision":7`) {
		t.Fatalf("update after reorder = %d %s", afterReorderUpdate.Code, afterReorderUpdate.Body.String())
	}
	for table, want := range map[string]int{
		"workspace_resource_revisions":   3,
		"workspace_mutation_idempotency": 2,
		"workspace_audit_entries":        8,
		"workspace_outbox_events":        8,
	} {
		var count int
		if err := runtime.Database().QueryRow(`SELECT COUNT(*) FROM ` + table + ` WHERE workspace_id='workspace-one'`).Scan(&count); err != nil || count != want {
			t.Fatalf("%s count = %d, %v", table, count, err)
		}
	}

	listed := runtimeRequest(runtime, http.MethodGet, "/api/tasks", "", headers)
	if listed.Code != http.StatusOK || !containsJSON(listed.Body.Bytes(), `"total":2`, `"title":"Ship S02A after reorder"`, `"workspace_id":"workspace-one"`) {
		t.Fatalf("list = %d %s", listed.Code, listed.Body.String())
	}
}

func TestSQLiteRuntimeTaskPersistsAcrossRestart(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "tasks-restart.db")
	now := time.Date(2026, 8, 18, 5, 0, 0, 0, time.UTC)
	config := Config{
		Name: "backend-test", Version: "test",
		HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath: databasePath, WorkspaceDependencies: FailClosedWorkspaceDependencies(),
		LocalAuth: auth.LocalAuthConfig{
			VerificationCode: "888888", SessionTTL: time.Hour, Now: func() time.Time { return now },
			NewID: func(context.Context) (string, error) { return "restart-token", nil },
		},
	}
	first := newRuntimeForConfig(t, config)
	userID := verifyRuntimeUser(t, first, "restart@example.com")
	for _, statement := range []string{
		`INSERT INTO workspaces(id,name,slug,settings,repos,issue_prefix,created_at,updated_at) VALUES('workspace-one','One','one','{}','[]','ONE','2026-08-18T00:00:00Z','2026-08-18T00:00:00Z')`,
		`INSERT INTO auth_members(id,workspace_id,user_id,role,created_at) VALUES('member-one','workspace-one','` + userID + `','member','2026-08-18T00:00:00Z')`,
	} {
		if _, err := first.Database().Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	headers := map[string]string{"Authorization": "Bearer restart-token", "X-Workspace-Slug": "one", "Idempotency-Key": "create-restart-task"}
	created := runtimeRequest(first, http.MethodPost, "/api/tasks", `{"title":"Survive restart"}`, headers)
	if created.Code != http.StatusCreated {
		t.Fatalf("create = %d %s", created.Code, created.Body.String())
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}

	second := newRuntimeForConfig(t, config)
	listed := runtimeRequest(second, http.MethodGet, "/api/tasks", "", headers)
	if listed.Code != http.StatusOK || !containsJSON(listed.Body.Bytes(), `"title":"Survive restart"`, `"revision":1`, `"total":1`) {
		t.Fatalf("restart list = %d %s", listed.Code, listed.Body.String())
	}
}
