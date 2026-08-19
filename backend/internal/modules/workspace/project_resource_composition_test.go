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

func TestSqliteWorkspaceChainComposesProjectResourcesWithUnavailableDefaultChecker(t *testing.T) {
	db := openWorkspaceTestDB(t)
	seedWorkspace(t, db, "workspace-1", "Acme", "acme")
	if _, err := db.Exec(`CREATE TABLE auth_members (
		id TEXT PRIMARY KEY,
		workspace_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		role TEXT NOT NULL CHECK (role IN ('owner','admin','member')),
		created_at TEXT NOT NULL,
		UNIQUE (workspace_id,user_id)
	)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO auth_members(id,workspace_id,user_id,role,created_at) VALUES('member-1','workspace-1','user-1','owner','2026-08-19T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO workspace_projects(id,workspace_id,name,status,priority,asset_ids,created_at,updated_at) VALUES('project-1','workspace-1','Runtime','in_progress','none','[]','2026-08-19T00:00:00Z','2026-08-19T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	module, err := NewWithSqliteWorkspaceChain(SqlitePersistenceConfig{DB: db}, WorkspaceServiceDependencies{
		Authorizer: &workspaceAccessStub{},
		Actors: &workspaceActorCatalog{actors: map[string]bool{
			"workspace-1/member/user-1": true,
		}},
		Assets: &workspaceAssetCatalog{assets: map[string]bool{}},
		Skills: &skillReferenceCatalog{references: map[string]bool{}},
		WorkspaceMemberships: selectionMemberships{
			"user-1": {{MemberID: "member-1", UserID: "user-1", WorkspaceID: "workspace-1", Role: "owner"}},
		},
		NewProjectResourceID: func(context.Context) (string, error) { return "resource-1", nil },
		Now:                  func() time.Time { return time.Date(2026, 8, 19, 7, 0, 0, 0, time.UTC) },
		HTTPIdentity: func(*http.Request) (contract.WorkspaceHTTPIdentity, error) {
			return contract.WorkspaceHTTPIdentity{WorkspaceID: "workspace-1", ActorType: "member", ActorID: "user-1"}, nil
		},
		HTTPUserIdentity:       func(*http.Request) (string, error) { return "user-1", nil },
		HTTPMutationAuthorizer: func(*http.Request) error { return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	server := kratoshttp.NewServer()
	module.RegisterHTTP(server)

	create := httptest.NewRequest(http.MethodPost, "/api/projects/project-1/resources", strings.NewReader(`{"resource_type":"url","resource_ref":{"url":"https://example.com/docs"},"label":"Docs"}`))
	create.Header.Set("X-Workspace-Slug", "acme")
	create.Header.Set("Content-Type", "application/json")
	create.Header.Set("Idempotency-Key", "create-resource-1")
	createResponse := httptest.NewRecorder()
	server.ServeHTTP(createResponse, create)
	if createResponse.Code != http.StatusCreated || !strings.Contains(createResponse.Body.String(), `"id":"resource-1"`) {
		t.Fatalf("create = %d %s", createResponse.Code, createResponse.Body.String())
	}

	refresh := httptest.NewRequest(http.MethodPut, "/api/projects/project-1/resources/resource-1", strings.NewReader(`{"action":"refresh","expected_revision":1}`))
	refresh.Header.Set("X-Workspace-Slug", "acme")
	refresh.Header.Set("Content-Type", "application/json")
	refreshResponse := httptest.NewRecorder()
	server.ServeHTTP(refreshResponse, refresh)
	if refreshResponse.Code != http.StatusOK || !strings.Contains(refreshResponse.Body.String(), `"state":"unavailable"`) || !strings.Contains(refreshResponse.Body.String(), `"diagnostic_code":"connection_not_configured"`) {
		t.Fatalf("refresh = %d %s", refreshResponse.Code, refreshResponse.Body.String())
	}
}
