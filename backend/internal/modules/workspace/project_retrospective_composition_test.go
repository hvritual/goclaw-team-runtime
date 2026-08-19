package workspace

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

func TestSqliteWorkspaceChainComposesCompleteProjectRetrospectiveHTTPVertical(t *testing.T) {
	db := openWorkspaceTestDB(t)
	seedWorkspace(t, db, "workspace-1", "Acme", "acme")
	if _, err := db.Exec(`CREATE TABLE auth_members (
		id TEXT PRIMARY KEY, workspace_id TEXT NOT NULL, user_id TEXT NOT NULL,
		role TEXT NOT NULL CHECK (role IN ('owner','admin','member')), created_at TEXT NOT NULL,
		UNIQUE (workspace_id,user_id)
	)`); err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`INSERT INTO auth_members(id,workspace_id,user_id,role,created_at) VALUES('member-1','workspace-1','user-1','owner','2026-08-20T00:00:00Z')`,
		`INSERT INTO workspace_projects(id,workspace_id,name,status,priority,asset_ids,created_at,updated_at)
		 VALUES('project-1','workspace-1','Runtime','in_progress','none','[]','2026-08-20T00:00:00Z','2026-08-20T00:00:00Z')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	actionIDs := &chainIDSequence{values: []string{"action-1", "action-2"}}
	module, err := NewWithSqliteWorkspaceChain(SqlitePersistenceConfig{DB: db}, WorkspaceServiceDependencies{
		Authorizer: &workspaceAccessStub{}, Actors: &workspaceActorCatalog{actors: map[string]bool{
			"workspace-1/member/user-1": true,
		}},
		Assets: &workspaceAssetCatalog{assets: map[string]bool{}}, Skills: &skillReferenceCatalog{references: map[string]bool{}},
		WorkspaceMemberships:                selectionMemberships{"user-1": {{MemberID: "member-1", UserID: "user-1", WorkspaceID: "workspace-1", Role: "owner"}}},
		NewProjectRetrospectiveID:           func(context.Context) (string, error) { return "retro-1", nil },
		NewProjectRetrospectiveActionItemID: actionIDs.next,
		NewTodoID:                           func(context.Context) (string, error) { return "todo-target-1", nil },
		NewIssueID:                          func(context.Context) (string, error) { return "issue-target-1", nil },
		Now:                                 func() time.Time { return time.Date(2026, 8, 20, 11, 0, 0, 0, time.UTC) },
		HTTPIdentity: func(*http.Request) (contract.WorkspaceHTTPIdentity, error) {
			return contract.WorkspaceHTTPIdentity{WorkspaceID: "workspace-1", ActorType: "member", ActorID: "user-1"}, nil
		},
		HTTPUserIdentity: func(*http.Request) (string, error) { return "user-1", nil }, HTTPMutationAuthorizer: func(*http.Request) error { return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	server := kratoshttp.NewServer()
	module.RegisterHTTP(server)

	create := projectRetrospectiveCompositionRequest(http.MethodPost, "/api/projects/project-1/retrospectives", `{"content":{"summary":"Release learning","successes":["Shipped"],"problems":[],"lessons":["Review earlier"],"action_items":[{"title":"Create task"},{"title":"Create issue"}]},"participants":[]}`)
	create.Header.Set("Idempotency-Key", "retro-create-key")
	created := httptest.NewRecorder()
	server.ServeHTTP(created, create)
	for _, fragment := range []string{`"id":"retro-1"`, `"status":"draft"`, `"id":"action-1"`, `"id":"action-2"`, `"can_publish":true`} {
		if created.Code != http.StatusCreated || !strings.Contains(created.Body.String(), fragment) {
			t.Fatalf("create = %d %s, missing %s", created.Code, created.Body.String(), fragment)
		}
	}

	publish := projectRetrospectiveCompositionRequest(http.MethodPut, "/api/projects/project-1/retrospectives/retro-1", `{"expected_revision":1,"action":"publish"}`)
	published := httptest.NewRecorder()
	server.ServeHTTP(published, publish)
	if published.Code != http.StatusOK || !strings.Contains(published.Body.String(), `"status":"published"`) || !strings.Contains(published.Body.String(), `"current_revision":2`) {
		t.Fatalf("publish = %d %s", published.Code, published.Body.String())
	}

	for _, target := range []struct{ action, key, body, kind, id string }{
		{action: "action-1", key: "task-target-key", body: `{}`, kind: "task", id: "todo-target-1"},
		{action: "action-2", key: "issue-target-key", body: `{"target_kind":"issue"}`, kind: "issue", id: "issue-target-1"},
	} {
		for attempt := 0; attempt < 2; attempt++ {
			request := projectRetrospectiveCompositionRequest(http.MethodPost, "/api/projects/project-1/retrospectives/retro-1/action-items/"+target.action+"/target", target.body)
			request.Header.Set("Idempotency-Key", target.key)
			response := httptest.NewRecorder()
			server.ServeHTTP(response, request)
			for _, fragment := range []string{`"retrospective_id":"retro-1"`, `"action_item_id":"` + target.action + `"`, `"source_revision":2`, `"target_kind":"` + target.kind + `"`, `"target_id":"` + target.id + `"`} {
				if response.Code != http.StatusCreated || !strings.Contains(response.Body.String(), fragment) {
					t.Fatalf("target %s attempt %d = %d %s, missing %s", target.kind, attempt, response.Code, response.Body.String(), fragment)
				}
			}
		}
	}

	list := projectRetrospectiveCompositionRequest(http.MethodGet, "/api/projects/project-1/retrospectives", "")
	listed := httptest.NewRecorder()
	server.ServeHTTP(listed, list)
	if listed.Code != http.StatusOK || !strings.Contains(listed.Body.String(), `"retrospectives":[{"id":"retro-1"`) || !strings.Contains(listed.Body.String(), `"action_links":[`) {
		t.Fatalf("list = %d %s", listed.Code, listed.Body.String())
	}

	archive := projectRetrospectiveCompositionRequest(http.MethodDelete, "/api/projects/project-1/retrospectives/retro-1?expected_revision=2", "")
	archived := httptest.NewRecorder()
	server.ServeHTTP(archived, archive)
	if archived.Code != http.StatusOK || !strings.Contains(archived.Body.String(), `"status":"archived"`) || !strings.Contains(archived.Body.String(), `"current_revision":3`) {
		t.Fatalf("archive = %d %s", archived.Code, archived.Body.String())
	}
	var todos, issues, links int
	if err = db.QueryRow(`SELECT COUNT(*) FROM workspace_todos WHERE workspace_id='workspace-1' AND id='todo-target-1'`).Scan(&todos); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(`SELECT COUNT(*) FROM workspace_issues WHERE workspace_id='workspace-1' AND id='issue-target-1'`).Scan(&issues); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(`SELECT COUNT(*) FROM workspace_project_retrospective_action_links WHERE workspace_id='workspace-1' AND retrospective_id='retro-1' AND state='linked'`).Scan(&links); err != nil {
		t.Fatal(err)
	}
	if todos != 1 || issues != 1 || links != 2 {
		t.Fatalf("installed target rows = todos %d issues %d links %d", todos, issues, links)
	}
}

func projectRetrospectiveCompositionRequest(method, path, body string) *http.Request {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("X-Workspace-Slug", "acme")
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	return request
}
