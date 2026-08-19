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

func TestSqliteWorkspaceChainComposesCanonicalProjectRequirementHTTP(t *testing.T) {
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
	for _, statement := range []string{
		`INSERT INTO auth_members(id,workspace_id,user_id,role,created_at) VALUES('owner-member','workspace-1','user-1','owner','2026-08-19T00:00:00Z')`,
		`INSERT INTO workspace_projects(id,workspace_id,name,status,priority,asset_ids,created_at,updated_at)
		 VALUES('project-1','workspace-1','Runtime','in_progress','none','[]','2026-08-19T00:00:00Z','2026-08-19T00:00:00Z')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	module, err := NewWithSqliteWorkspaceChain(SqlitePersistenceConfig{DB: db}, WorkspaceServiceDependencies{
		Authorizer:           &workspaceAccessStub{},
		Actors:               &workspaceActorCatalog{actors: map[string]bool{"workspace-1/member/user-1": true}},
		Assets:               &workspaceAssetCatalog{assets: map[string]bool{}},
		Skills:               &skillReferenceCatalog{references: map[string]bool{}},
		WorkspaceMemberships: selectionMemberships{"user-1": {{MemberID: "owner-member", UserID: "user-1", WorkspaceID: "workspace-1", Role: "owner"}}},
		NewProjectRequirementID: func(context.Context) (string, error) {
			return "baseline-1", nil
		},
		NewProjectOutlineNodeID: func(context.Context) (string, error) {
			return "outline-1", nil
		},
		Now: func() time.Time { return time.Date(2026, 8, 19, 14, 0, 0, 0, time.UTC) },
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

	create := projectRequirementCompositionRequest(http.MethodPut, "/api/projects/project-1/requirement-baseline", `{"expected_revision":0,"content":{"problem_statement":"Govern delivery","goals":[{"key":"goal-1","text":"Ship"}],"in_scope":[],"out_of_scope":[],"constraints":[],"acceptance_criteria":[],"dependencies":[]},"change_summary":"Initial baseline"}`)
	create.Header.Set("Idempotency-Key", "baseline-create-key")
	createResponse := httptest.NewRecorder()
	server.ServeHTTP(createResponse, create)
	if createResponse.Code != http.StatusCreated || !strings.Contains(createResponse.Body.String(), `"id":"baseline-1"`) || !strings.Contains(createResponse.Body.String(), `"can_manage_access":true`) {
		t.Fatalf("create baseline = %d %s", createResponse.Code, createResponse.Body.String())
	}

	outline := projectRequirementCompositionRequest(http.MethodPost, "/api/projects/project-1/outline", `{"expected_revision":0,"title":"Delivery root"}`)
	outline.Header.Set("Idempotency-Key", "outline-create-key")
	outlineResponse := httptest.NewRecorder()
	server.ServeHTTP(outlineResponse, outline)
	if outlineResponse.Code != http.StatusCreated || !strings.Contains(outlineResponse.Body.String(), `"id":"outline-1"`) {
		t.Fatalf("create outline = %d %s", outlineResponse.Code, outlineResponse.Body.String())
	}

	read := projectRequirementCompositionRequest(http.MethodGet, "/api/projects/project-1/requirement-baseline", "")
	readResponse := httptest.NewRecorder()
	server.ServeHTTP(readResponse, read)
	if readResponse.Code != http.StatusOK || !strings.Contains(readResponse.Body.String(), `"problem_statement":"Govern delivery"`) || !strings.Contains(readResponse.Body.String(), `"revision":1`) {
		t.Fatalf("read baseline = %d %s", readResponse.Code, readResponse.Body.String())
	}

	coverage := projectRequirementCompositionRequest(http.MethodGet, "/api/projects/project-1/requirement-baseline/coverage", "")
	coverageResponse := httptest.NewRecorder()
	server.ServeHTTP(coverageResponse, coverage)
	for _, fragment := range []string{
		`"baseline_status":"draft"`, `"revision":1`, `"total":1`, `"linked":0`,
		`"implemented":0`, `"accepted":0`, `"unlinked":1`, `"stage":"unlinked"`,
		`"effective":null`,
	} {
		if coverageResponse.Code != http.StatusOK || !strings.Contains(coverageResponse.Body.String(), fragment) {
			t.Fatalf("coverage = %d %s, missing %s", coverageResponse.Code, coverageResponse.Body.String(), fragment)
		}
	}

	if _, err = db.Exec(`UPDATE workspace_requirement_revisions SET content_json='{}' WHERE baseline_id='baseline-1' AND revision=1`); err != nil {
		t.Fatal(err)
	}
	corruptResponse := httptest.NewRecorder()
	server.ServeHTTP(corruptResponse, coverage)
	if corruptResponse.Code != http.StatusInternalServerError ||
		!strings.Contains(corruptResponse.Body.String(), `"code":"internal_error"`) ||
		!strings.Contains(corruptResponse.Body.String(), `"error":"failed to read Project Requirement coverage"`) {
		t.Fatalf("corrupt persisted coverage = %d %s", corruptResponse.Code, corruptResponse.Body.String())
	}
	for _, forbidden := range []string{`"total"`, `"items"`, `"problem_statement"`, "Govern delivery"} {
		if strings.Contains(corruptResponse.Body.String(), forbidden) {
			t.Fatalf("corrupt persisted coverage leaked fallback fragment %q: %s", forbidden, corruptResponse.Body.String())
		}
	}

	if _, err = db.Exec(`UPDATE workspace_requirement_revisions SET content_json='{"problem_statement":"Govern delivery","goals":[{"key":"goal-1","text":"Ship"}],"in_scope":[],"out_of_scope":[],"constraints":[],"acceptance_criteria":[],"dependencies":[]}' WHERE baseline_id='baseline-1' AND revision=1`); err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`INSERT INTO workspaces(id,name,slug,issue_prefix,created_at,updated_at) VALUES
			('workspace-2','Workspace Two','workspace-two','TWO','2026-08-19T00:00:00Z','2026-08-19T00:00:00Z')`,
		`INSERT INTO workspace_projects(id,workspace_id,name,status,priority,asset_ids,created_at,updated_at) VALUES
			('project-2','workspace-2','Foreign Project','in_progress','none','[]','2026-08-19T00:00:00Z','2026-08-19T00:00:00Z')`,
		`INSERT INTO workspace_issues(
			id,workspace_id,number,identifier,title,status,priority,creator_type,creator_id,project_id,created_at,updated_at
		) VALUES('foreign-issue','workspace-2',1,'TWO-1','Foreign secret title','done','none','member','foreign-owner','project-2','2026-08-19T00:00:00Z','2026-08-19T00:00:00Z')`,
		`INSERT INTO workspace_requirement_issue_links(
			workspace_id,project_id,baseline_id,requirement_key,issue_id,linked_revision,
			unlinked_revision,linked_by,linked_at,unlinked_by,unlinked_at
		) VALUES('workspace-2','project-2','baseline-1','goal-1','foreign-issue',1,NULL,'owner-member','2026-08-19T00:00:00Z',NULL,NULL)`,
	} {
		if _, err = db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	driftReadResponse := httptest.NewRecorder()
	server.ServeHTTP(driftReadResponse, read)
	if driftReadResponse.Code != http.StatusInternalServerError ||
		!strings.Contains(driftReadResponse.Body.String(), `"code":"internal_error"`) ||
		!strings.Contains(driftReadResponse.Body.String(), `"error":"failed to read Project Requirement"`) {
		t.Fatalf("ownership-drift baseline = %d %s", driftReadResponse.Code, driftReadResponse.Body.String())
	}
	driftCoverageResponse := httptest.NewRecorder()
	server.ServeHTTP(driftCoverageResponse, coverage)
	if driftCoverageResponse.Code != http.StatusInternalServerError ||
		!strings.Contains(driftCoverageResponse.Body.String(), `"code":"internal_error"`) ||
		!strings.Contains(driftCoverageResponse.Body.String(), `"error":"failed to read Project Requirement coverage"`) {
		t.Fatalf("ownership-drift coverage = %d %s", driftCoverageResponse.Code, driftCoverageResponse.Body.String())
	}
	for _, response := range []*httptest.ResponseRecorder{driftReadResponse, driftCoverageResponse} {
		for _, forbidden := range []string{`"issue_links"`, `"total"`, `"items"`, "TWO-1", "Foreign secret title", "Foreign Project"} {
			if strings.Contains(response.Body.String(), forbidden) {
				t.Fatalf("ownership drift leaked fragment %q: %s", forbidden, response.Body.String())
			}
		}
	}
}

func projectRequirementCompositionRequest(method, path, body string) *http.Request {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("X-Workspace-Slug", "acme")
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	return request
}
