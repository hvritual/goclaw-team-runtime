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

func TestSQLiteRuntimeCertifiesPhaseOneEngineeringDigitalThread(t *testing.T) {
	now := time.Date(2026, 8, 28, 15, 15, 0, 0, time.UTC)
	runtime := newRuntimeForConfig(t, Config{
		Name: "backend-test", Version: "test",
		HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath:            filepath.Join(t.TempDir(), "engineering-phase-one-exit.db"),
		WorkspaceDependencies: FailClosedWorkspaceDependencies(),
		LocalAuth: auth.LocalAuthConfig{
			VerificationCode: "888888", SessionTTL: time.Hour, Now: func() time.Time { return now },
			NewID: func(context.Context) (string, error) { return "phase-one-token", nil },
		},
	})
	userID := verifyRuntimeUser(t, runtime, "phase-one@example.com")
	for _, statement := range []string{
		`INSERT INTO workspaces(id,name,slug,settings,repos,issue_prefix,created_at,updated_at) VALUES('workspace-one','One','one','{}','[]','ONE','2026-08-28T00:00:00Z','2026-08-28T00:00:00Z')`,
		`INSERT INTO auth_members(id,workspace_id,user_id,role,created_at) VALUES('member-one','workspace-one','` + userID + `','owner','2026-08-28T00:00:00Z')`,
		`INSERT INTO workspace_projects(id,workspace_id,name,description,status,asset_ids,created_at,updated_at) VALUES('project-one','workspace-one','Project One','','in_progress','[]','2026-08-28T00:00:00Z','2026-08-28T00:00:00Z')`,
		`INSERT INTO workspace_requirements(id,workspace_id,project_id,title,current_version,approval_status,coverage_status,issue_ids,created_at,updated_at) VALUES('requirement-one','workspace-one','project-one','Requirement One',1,'draft','uncovered','[]','2026-08-28T00:00:00Z','2026-08-28T00:00:00Z')`,
		`INSERT INTO workspace_todos(id,workspace_id,title,description,status,created_at,updated_at,priority,creator_type,creator_id,position,revision) VALUES('task-one','workspace-one','Task One','','todo','2026-08-28T00:00:00Z','2026-08-28T00:00:00Z','none','member','` + userID + `',0,1)`,
	} {
		if _, err := runtime.Database().Exec(statement); err != nil {
			t.Fatal(err)
		}
	}

	engineeringHeaders := map[string]string{"Authorization": "Bearer phase-one-token"}
	createdEntity := runtimeRequest(runtime, http.MethodPost, "/api/engineering/v1/workspaces/workspace-one/entities", `{"id":"service-one","type":"service","name":"Device Gateway","status":"active"}`, engineeringHeaders)
	if createdEntity.Code != http.StatusCreated {
		t.Fatalf("create entity = %d %s", createdEntity.Code, createdEntity.Body.String())
	}
	binding := runtimeRequest(runtime, http.MethodPost, "/api/engineering/v1/workspaces/workspace-one/source-bindings", `{"id":"github-service-one","entity_id":"service-one","provenance":{"source_type":"github","locator":"github://acme/device-gateway","revision":"abc123"},"authority":"authoritative"}`, engineeringHeaders)
	if binding.Code != http.StatusCreated || !containsJSON(binding.Body.Bytes(), `"entity_id":"service-one"`, `"authority":"authoritative"`, `"source_type":"github"`, `"revision":"abc123"`) {
		t.Fatalf("source binding = %d %s", binding.Code, binding.Body.String())
	}

	workHeaders := map[string]string{"Authorization": "Bearer phase-one-token", "X-Workspace-Slug": "one"}
	for _, path := range []string{
		"/api/projects/project-one/engineering-links",
		"/api/requirements/requirement-one/engineering-links",
		"/api/tasks/task-one/engineering-links",
	} {
		response := runtimeRequest(runtime, http.MethodPost, path, `{"entity_id":"service-one"}`, workHeaders)
		if response.Code != http.StatusCreated {
			t.Fatalf("work link %s = %d %s", path, response.Code, response.Body.String())
		}
	}

	change := runtimeRequest(runtime, http.MethodPost, "/api/engineering/v1/workspaces/workspace-one/changes", `{"id":"change-one","project_id":"project-one","requirement_id":"requirement-one","work_item":{"kind":"task","id":"task-one"},"summary":"Update reconnect handling","affected_entity_ids":["service-one"],"artifacts":[{"kind":"pull_request","locator":"github://acme/device-gateway/pull/7","revision":"abc123"}],"provenance":{"source_type":"workspace","locator":"workspace://workspace-one/tasks/task-one","revision":"1"}}`, engineeringHeaders)
	if change.Code != http.StatusCreated || !containsJSON(change.Body.Bytes(), `"id":"change-one"`, `"status":"proposed"`) {
		t.Fatalf("create change = %d %s", change.Code, change.Body.String())
	}
	accepted := runtimeRequest(runtime, http.MethodPost, "/api/engineering/v1/workspaces/workspace-one/changes/change-one/accept", "", engineeringHeaders)
	if accepted.Code != http.StatusOK || !containsJSON(accepted.Body.Bytes(), `"id":"change-one"`, `"status":"accepted"`) {
		t.Fatalf("accept change = %d %s", accepted.Code, accepted.Body.String())
	}

	frozen := runtimeRequest(runtime, http.MethodPost, "/api/engineering/v1/workspaces/workspace-one/context-packs", `{"id":"context-one","work_item":{"kind":"task","id":"task-one"},"work_item_revision":"1","target_entity_ids":["service-one"],"references":[{"kind":"change","id":"change-one","revision":"accepted","checksum":"sha256:change-one"}],"policy_version":"phase1-exit-v1"}`, engineeringHeaders)
	if frozen.Code != http.StatusCreated {
		t.Fatalf("freeze context pack = %d %s", frozen.Code, frozen.Body.String())
	}
	var pack struct {
		ID               string `json:"id"`
		Checksum         string `json:"checksum"`
		WorkItemRevision string `json:"work_item_revision"`
		PolicyVersion    string `json:"policy_version"`
	}
	if err := json.Unmarshal(frozen.Body.Bytes(), &pack); err != nil {
		t.Fatal(err)
	}
	if pack.ID != "context-one" || pack.Checksum == "" || pack.WorkItemRevision != "1" || pack.PolicyVersion != "phase1-exit-v1" {
		t.Fatalf("frozen context pack = %+v", pack)
	}
	readback := runtimeRequest(runtime, http.MethodGet, "/api/engineering/v1/workspaces/workspace-one/context-packs/context-one", "", engineeringHeaders)
	if readback.Code != http.StatusOK || !containsJSON(readback.Body.Bytes(), `"id":"context-one"`, `"checksum":"`+pack.Checksum+`"`, `"revision":"accepted"`) {
		t.Fatalf("context pack readback = %d %s", readback.Code, readback.Body.String())
	}
	replayed := runtimeRequest(runtime, http.MethodPost, "/api/engineering/v1/workspaces/workspace-one/context-packs", `{"id":"context-one","work_item":{"kind":"task","id":"task-one"},"work_item_revision":"1","target_entity_ids":["service-one"],"references":[{"kind":"change","id":"change-one","revision":"accepted","checksum":"sha256:change-one"}],"policy_version":"phase1-exit-v1"}`, engineeringHeaders)
	if replayed.Code != http.StatusCreated || !containsJSON(replayed.Body.Bytes(), `"checksum":"`+pack.Checksum+`"`) {
		t.Fatalf("context pack replay = %d %s", replayed.Code, replayed.Body.String())
	}

	var taskStatus string
	var taskRevision int64
	if err := runtime.Database().QueryRow(`SELECT status, revision FROM workspace_todos WHERE workspace_id='workspace-one' AND id='task-one'`).Scan(&taskStatus, &taskRevision); err != nil {
		t.Fatal(err)
	}
	if taskStatus != "todo" || taskRevision != 1 {
		t.Fatalf("Phase 1 exit mutated Task lifecycle: status=%s revision=%d", taskStatus, taskRevision)
	}
}
