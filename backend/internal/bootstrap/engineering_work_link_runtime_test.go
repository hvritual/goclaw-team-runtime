package bootstrap

import (
	"context"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/auth"
)

func TestSQLiteRuntimeLinksWorkspaceWorkToEngineeringThread(t *testing.T) {
	now := time.Date(2026, 8, 28, 14, 0, 0, 0, time.UTC)
	runtime := newRuntimeForConfig(t, Config{
		Name: "backend-test", Version: "test",
		HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath:            filepath.Join(t.TempDir(), "engineering-work-links.db"),
		WorkspaceDependencies: FailClosedWorkspaceDependencies(),
		LocalAuth: auth.LocalAuthConfig{
			VerificationCode: "888888", SessionTTL: time.Hour, Now: func() time.Time { return now },
			NewID: func(context.Context) (string, error) { return "engineering-link-token", nil },
		},
	})
	userID := verifyRuntimeUser(t, runtime, "engineering-links@example.com")
	for _, statement := range []string{
		`INSERT INTO workspaces(id,name,slug,settings,repos,issue_prefix,created_at,updated_at) VALUES('workspace-one','One','one','{}','[]','ONE','2026-08-28T00:00:00Z','2026-08-28T00:00:00Z')`,
		`INSERT INTO auth_members(id,workspace_id,user_id,role,created_at) VALUES('member-one','workspace-one','` + userID + `','owner','2026-08-28T00:00:00Z')`,
		`INSERT INTO workspace_projects(id,workspace_id,name,description,status,asset_ids,created_at,updated_at) VALUES('project-one','workspace-one','Project One','','in_progress','[]','2026-08-28T00:00:00Z','2026-08-28T00:00:00Z')`,
		`INSERT INTO workspace_projects(id,workspace_id,name,description,status,asset_ids,created_at,updated_at) VALUES('project-two','workspace-one','Project Two','','planned','[]','2026-08-28T00:00:00Z','2026-08-28T00:00:00Z')`,
		`INSERT INTO workspace_requirements(id,workspace_id,project_id,title,current_version,approval_status,coverage_status,issue_ids,created_at,updated_at) VALUES('requirement-one','workspace-one','project-one','Requirement One',1,'draft','uncovered','[]','2026-08-28T00:00:00Z','2026-08-28T00:00:00Z')`,
		`INSERT INTO workspace_todos(id,workspace_id,title,description,status,created_at,updated_at,priority,creator_type,creator_id,position,revision) VALUES('task-one','workspace-one','Task One','','todo','2026-08-28T00:00:00Z','2026-08-28T00:00:00Z','none','member','` + userID + `',0,1)`,
	} {
		if _, err := runtime.Database().Exec(statement); err != nil {
			t.Fatal(err)
		}
	}

	engineeringHeaders := map[string]string{"Authorization": "Bearer engineering-link-token"}
	createdEntity := runtimeRequest(runtime, http.MethodPost, "/api/engineering/v1/workspaces/workspace-one/entities", `{"id":"service-one","type":"service","name":"Service One","status":"active"}`, engineeringHeaders)
	if createdEntity.Code != http.StatusCreated {
		t.Fatalf("create EngineeringEntity = %d %s", createdEntity.Code, createdEntity.Body.String())
	}

	workHeaders := map[string]string{"Authorization": "Bearer engineering-link-token", "X-Workspace-Slug": "one"}
	for _, test := range []struct {
		path     string
		relation string
		kind     string
	}{
		{path: "/api/projects/project-one/engineering-links", relation: "changes", kind: "project"},
		{path: "/api/requirements/requirement-one/engineering-links", relation: "affects", kind: "requirement"},
		{path: "/api/tasks/task-one/engineering-links", relation: "affects", kind: "task"},
	} {
		response := runtimeRequest(runtime, http.MethodPost, test.path, `{"entity_id":"service-one"}`, workHeaders)
		if response.Code != http.StatusCreated || !containsJSON(response.Body.Bytes(),
			`"work_kind":"`+test.kind+`"`,
			`"entity_id":"service-one"`,
			`"relation":"`+test.relation+`"`,
			`"authority":"authoritative"`,
			`"source":"workspace"`,
		) {
			t.Fatalf("link %s = %d %s", test.kind, response.Code, response.Body.String())
		}
	}
	var taskStatus string
	var taskRevision int64
	if err := runtime.Database().QueryRow(`SELECT status, revision FROM workspace_todos WHERE workspace_id='workspace-one' AND id='task-one'`).Scan(&taskStatus, &taskRevision); err != nil {
		t.Fatal(err)
	}
	if taskStatus != "todo" || taskRevision != 1 {
		t.Fatalf("Task lifecycle changed by engineering link: status=%s revision=%d", taskStatus, taskRevision)
	}

	projectLinks := runtimeRequest(runtime, http.MethodGet, "/api/projects/project-one/engineering-links", "", workHeaders)
	if projectLinks.Code != http.StatusOK || !containsJSON(projectLinks.Body.Bytes(), `"relation":"changes"`, `"entity_id":"service-one"`) {
		t.Fatalf("project links = %d %s", projectLinks.Code, projectLinks.Body.String())
	}

	if _, err := runtime.Database().Exec(`DELETE FROM workspace_todos WHERE id='task-one' AND workspace_id='workspace-one'`); err != nil {
		t.Fatal(err)
	}
	historicalTaskLinks := runtimeRequest(runtime, http.MethodGet, "/api/tasks/task-one/engineering-links", "", workHeaders)
	if historicalTaskLinks.Code != http.StatusOK || !containsJSON(historicalTaskLinks.Body.Bytes(), `"entity_id":"service-one"`) {
		t.Fatalf("historical task links = %d %s", historicalTaskLinks.Code, historicalTaskLinks.Body.String())
	}
	unlinkTask := runtimeRequest(runtime, http.MethodDelete, "/api/tasks/task-one/engineering-links/service-one", "", workHeaders)
	if unlinkTask.Code != http.StatusNoContent {
		t.Fatalf("unlink deleted task = %d %s", unlinkTask.Code, unlinkTask.Body.String())
	}
	afterUnlink := runtimeRequest(runtime, http.MethodGet, "/api/tasks/task-one/engineering-links", "", workHeaders)
	if afterUnlink.Code != http.StatusOK || !containsJSON(afterUnlink.Body.Bytes(), `"links":[]`) {
		t.Fatalf("links after unlink = %d %s", afterUnlink.Code, afterUnlink.Body.String())
	}
	linkDeletedTask := runtimeRequest(runtime, http.MethodPost, "/api/tasks/task-one/engineering-links", `{"entity_id":"service-one"}`, workHeaders)
	if linkDeletedTask.Code != http.StatusNotFound {
		t.Fatalf("link deleted task = %d %s", linkDeletedTask.Code, linkDeletedTask.Body.String())
	}

	archived := runtimeRequest(runtime, http.MethodPatch, "/api/engineering/v1/workspaces/workspace-one/entities/service-one", `{"status":"archived"}`, engineeringHeaders)
	if archived.Code != http.StatusOK {
		t.Fatalf("archive EngineeringEntity = %d %s", archived.Code, archived.Body.String())
	}
	historicalProjectLinks := runtimeRequest(runtime, http.MethodGet, "/api/projects/project-one/engineering-links", "", workHeaders)
	if historicalProjectLinks.Code != http.StatusOK || !containsJSON(historicalProjectLinks.Body.Bytes(), `"entity_id":"service-one"`) {
		t.Fatalf("historical project links after entity archive = %d %s", historicalProjectLinks.Code, historicalProjectLinks.Body.String())
	}
	linkArchivedEntity := runtimeRequest(runtime, http.MethodPost, "/api/projects/project-two/engineering-links", `{"entity_id":"service-one"}`, workHeaders)
	if linkArchivedEntity.Code != http.StatusConflict {
		t.Fatalf("link archived entity = %d %s", linkArchivedEntity.Code, linkArchivedEntity.Body.String())
	}
}
