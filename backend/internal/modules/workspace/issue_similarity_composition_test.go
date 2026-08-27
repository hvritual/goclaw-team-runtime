package workspace

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

func TestSqliteWorkspaceChainComposesIssueSimilarityHTTP(t *testing.T) {
	db := openWorkspaceTestDB(t)
	seedWorkspace(t, db, "workspace-1", "Acme", "acme")
	for _, statement := range []string{
		`INSERT INTO workspace_issues(
			id,workspace_id,number,identifier,title,status,priority,creator_type,creator_id,metadata,properties,asset_ids,created_at,updated_at
		) VALUES('issue-1','workspace-1',1,'WSP-1','Canonical delivery','todo','none','member','member-1','{}','{}','[]','2026-08-21T00:00:00Z','2026-08-21T00:00:00Z')`,
		`INSERT INTO workspace_issues(
			id,workspace_id,number,identifier,title,status,priority,creator_type,creator_id,metadata,properties,asset_ids,created_at,updated_at
		) VALUES('issue-2','workspace-1',2,'WSP-2','Canonical delivery follow-up','todo','none','member','member-1','{}','{}','[]','2026-08-20T00:00:00Z','2026-08-20T00:00:00Z')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	module, err := NewWithSqliteWorkspaceChain(SqlitePersistenceConfig{DB: db}, WorkspaceServiceDependencies{
		Authorizer:           &workspaceAccessStub{},
		Actors:               &workspaceActorCatalog{actors: map[string]bool{}},
		Assets:               &workspaceAssetCatalog{assets: map[string]bool{}},
		Skills:               &skillReferenceCatalog{references: map[string]bool{}},
		WorkspaceMemberships: selectionMemberships{},
		Now:                  func() time.Time { return time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC) },
		HTTPIdentity: func(*http.Request) (contract.WorkspaceHTTPIdentity, error) {
			return contract.WorkspaceHTTPIdentity{WorkspaceID: "workspace-1", ActorType: "member", ActorID: "member-1"}, nil
		},
		HTTPUserIdentity: func(*http.Request) (string, error) { return "user-1", nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	server := kratoshttp.NewServer()
	module.RegisterHTTP(server)

	request := httptest.NewRequest(http.MethodPost, "/api/issues/issue-1/similarity/check", nil)
	request.Header.Set("X-Workspace-Slug", "acme")
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"id":"issue-2"`) || strings.Contains(response.Body.String(), `"id":"issue-1"`) || !strings.Contains(response.Body.String(), `"detector_available":true`) {
		t.Fatalf("similarity = %d %s", response.Code, response.Body.String())
	}
}
