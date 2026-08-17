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

func TestSQLiteRuntimePromotesTaskToIssueExactlyOnce(t *testing.T) {
	now := time.Date(2026, 8, 18, 10, 0, 0, 0, time.UTC)
	runtime := newRuntimeForConfig(t, Config{
		Name: "backend-test", Version: "test",
		HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath:            filepath.Join(t.TempDir(), "task-promotion.db"),
		WorkspaceDependencies: FailClosedWorkspaceDependencies(),
		LocalAuth: auth.LocalAuthConfig{
			VerificationCode: "888888", SessionTTL: time.Hour, Now: func() time.Time { return now },
			NewID: func(context.Context) (string, error) { return "promotion-token", nil },
		},
	})
	userID := verifyRuntimeUser(t, runtime, "promote@example.com")
	for _, statement := range []string{
		`INSERT INTO workspaces(id,name,slug,settings,repos,issue_prefix,created_at,updated_at) VALUES('workspace-one','One','one','{}','[]','ONE','2026-08-18T00:00:00Z','2026-08-18T00:00:00Z')`,
		`INSERT INTO auth_members(id,workspace_id,user_id,role,created_at) VALUES('member-one','workspace-one','` + userID + `','member','2026-08-18T00:00:00Z')`,
	} {
		if _, err := runtime.Database().Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	headers := map[string]string{"Authorization": "Bearer promotion-token", "X-Workspace-Slug": "one", "Idempotency-Key": "create-promotion-task"}
	created := runtimeRequest(runtime, http.MethodPost, "/api/tasks", `{"title":"Promote runtime task","description":"Frozen snapshot","priority":"high"}`, headers)
	if created.Code != http.StatusCreated {
		t.Fatalf("create = %d %s", created.Code, created.Body.String())
	}
	var task struct {
		ID       string `json:"id"`
		Revision int64  `json:"revision"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &task); err != nil {
		t.Fatal(err)
	}
	inProgress := runtimeRequest(runtime, http.MethodPatch, "/api/tasks/"+task.ID, `{"status":"in_progress","expected_revision":1}`, headers)
	if inProgress.Code != http.StatusOK {
		t.Fatalf("in progress = %d %s", inProgress.Code, inProgress.Body.String())
	}
	headers["Idempotency-Key"] = "promote-runtime-task"
	promoted := runtimeRequest(runtime, http.MethodPost, "/api/tasks/"+task.ID+"/promote", `{"expected_revision":2,"complete_task":true}`, headers)
	if promoted.Code != http.StatusCreated || !containsJSON(promoted.Body.Bytes(), `"source_task_id":"`+task.ID+`"`, `"status":"done"`, `"identifier":"ONE-1"`) {
		t.Fatalf("promote = %d %s", promoted.Code, promoted.Body.String())
	}
	replayed := runtimeRequest(runtime, http.MethodPost, "/api/tasks/"+task.ID+"/promote", `{"expected_revision":2,"complete_task":true}`, headers)
	if replayed.Code != http.StatusCreated || replayed.Body.String() != promoted.Body.String() {
		t.Fatalf("promotion replay = %d %s, want %s", replayed.Code, replayed.Body.String(), promoted.Body.String())
	}
	var promotion struct {
		Issue struct {
			ID string `json:"id"`
		} `json:"issue"`
	}
	if err := json.Unmarshal(promoted.Body.Bytes(), &promotion); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Database().Exec(`UPDATE workspace_todos SET title='Live Task edit' WHERE id=?`, task.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Database().Exec(`UPDATE workspace_issues SET title='Live Issue edit' WHERE id=?`, promotion.Issue.ID); err != nil {
		t.Fatal(err)
	}
	replayedAfterEdits := runtimeRequest(runtime, http.MethodPost, "/api/tasks/"+task.ID+"/promote", `{"expected_revision":2,"complete_task":true}`, headers)
	if replayedAfterEdits.Code != http.StatusCreated || replayedAfterEdits.Body.String() != promoted.Body.String() {
		t.Fatalf("promotion replay after live edits = %d %s, want %s", replayedAfterEdits.Code, replayedAfterEdits.Body.String(), promoted.Body.String())
	}
	conflict := runtimeRequest(runtime, http.MethodPost, "/api/tasks/"+task.ID+"/promote", `{"expected_revision":2,"complete_task":false}`, headers)
	if conflict.Code != http.StatusConflict || !containsJSON(conflict.Body.Bytes(), `"code":"idempotency_conflict"`) {
		t.Fatalf("promotion body conflict = %d %s", conflict.Code, conflict.Body.String())
	}
	for table, want := range map[string]int{
		"workspace_issues":                1,
		"workspace_task_issue_promotions": 1,
	} {
		var count int
		if err := runtime.Database().QueryRow(`SELECT COUNT(*) FROM ` + table + ` WHERE workspace_id='workspace-one'`).Scan(&count); err != nil || count != want {
			t.Fatalf("%s count = %d, %v", table, count, err)
		}
	}
}
