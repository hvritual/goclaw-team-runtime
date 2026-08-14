package bootstrap

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/auth"
)

func TestSQLiteRuntimeServesTrustedIssueMetadataAndPersistsAcrossRestart(t *testing.T) {
	now := time.Date(2026, 8, 13, 6, 7, 8, 0, time.UTC)
	sequence := 0
	databasePath := filepath.Join(t.TempDir(), "issue-metadata.db")
	config := Config{
		Name: "backend-test", Version: "test",
		HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath: databasePath, WorkspaceDependencies: FailClosedWorkspaceDependencies(),
		LocalAuth: auth.LocalAuthConfig{
			VerificationCode: "888888", SessionTTL: 24 * time.Hour, Now: func() time.Time { return now },
			NewID: func(context.Context) (string, error) {
				sequence++
				return []string{"token-one", "token-two"}[sequence-1], nil
			},
		},
	}
	runtime := newRuntimeForConfig(t, config)
	loginOne := verifyRuntimeLogin(t, runtime, "one@example.com")
	loginTwo := verifyRuntimeLogin(t, runtime, "two@example.com")
	for _, statement := range []string{
		`INSERT INTO workspaces(id,name,slug,settings,repos,issue_prefix,created_at,updated_at) VALUES
			('workspace-one','One','one','{}','[]','ONE','2026-08-13T00:00:00Z','2026-08-13T00:00:00Z'),
			('workspace-two','Two','two','{}','[]','TWO','2026-08-13T00:00:00Z','2026-08-13T00:00:00Z')`,
		`INSERT INTO auth_members(id,workspace_id,user_id,role,created_at) VALUES
			('member-one','workspace-one','` + loginOne.UserID + `','member','2026-08-13T00:00:00Z'),
			('member-two','workspace-two','` + loginTwo.UserID + `','member','2026-08-13T00:00:00Z')`,
		`INSERT INTO workspace_issues(id,workspace_id,number,identifier,title,status,priority,creator_type,creator_id,position,metadata,properties,asset_ids,created_at,updated_at) VALUES
			('issue-one','workspace-one',1,'ONE-1','First issue','todo','high','member','member-one',1,'{"seed":true}','{}','[]','2026-08-13T00:00:01Z','2026-08-13T00:00:01Z'),
			('issue-two','workspace-two',1,'TWO-1','Foreign issue','todo','none','member','member-two',1,'{}','{}','[]','2026-08-13T00:00:01Z','2026-08-13T00:00:01Z')`,
	} {
		if _, err := runtime.Database().Exec(statement); err != nil {
			t.Fatal(err)
		}
	}

	missingWorkspace := runtimeRequest(runtime, http.MethodGet, "/api/issues/ONE-1/metadata", "", nil)
	assertRuntimeResponse(t, missingWorkspace.Code, missingWorkspace.Body.String(), http.StatusUnauthorized, `{"error":"user not authenticated"}`)
	authenticatedMissingWorkspace := runtimeRequest(runtime, http.MethodGet, "/api/issues/ONE-1/metadata", "", map[string]string{"Authorization": "Bearer " + loginOne.Token})
	assertRuntimeResponse(t, authenticatedMissingWorkspace.Code, authenticatedMissingWorkspace.Body.String(), http.StatusBadRequest, `{"error":"workspace_id is required"}`)
	missingAuth := runtimeRequest(runtime, http.MethodGet, "/api/issues/ONE-1/metadata", "", map[string]string{"X-Workspace-Slug": "one"})
	assertRuntimeResponse(t, missingAuth.Code, missingAuth.Body.String(), http.StatusUnauthorized, `{"error":"user not authenticated"}`)

	bearerHeaders := map[string]string{"Authorization": "Bearer " + loginOne.Token, "X-Workspace-Slug": "one"}
	get := runtimeRequest(runtime, http.MethodGet, "/api/issues/ONE-1/metadata", "", bearerHeaders)
	assertRuntimeResponse(t, get.Code, get.Body.String(), http.StatusOK, `{"metadata":{"seed":true}}`)
	put := runtimeRequest(runtime, http.MethodPut, "/api/issues/issue-one/metadata/count", `{"value":2}`, bearerHeaders)
	assertRuntimeResponse(t, put.Code, put.Body.String(), http.StatusOK, `{"metadata":{"count":2,"seed":true}}`)

	foreign := runtimeRequest(runtime, http.MethodGet, "/api/issues/issue-two/metadata", "", bearerHeaders)
	assertRuntimeResponse(t, foreign.Code, foreign.Body.String(), http.StatusNotFound, `{"error":"issue not found"}`)
	mismatch := runtimeRequest(runtime, http.MethodGet, "/api/issues/ONE-1/metadata", "", map[string]string{
		"Authorization": "Bearer " + loginOne.Token, "X-Workspace-ID": "workspace-one", "X-Workspace-Slug": "two",
	})
	assertRuntimeResponse(t, mismatch.Code, mismatch.Body.String(), http.StatusNotFound, `{"error":"issue not found"}`)
	foreignMembership := runtimeRequest(runtime, http.MethodGet, "/api/issues/ONE-1/metadata", "", map[string]string{
		"Authorization": "Bearer " + loginTwo.Token, "X-Workspace-Slug": "one",
	})
	assertRuntimeResponse(t, foreignMembership.Code, foreignMembership.Body.String(), http.StatusNotFound, `{"error":"issue not found"}`)

	cookieHeaders := map[string]string{"Cookie": "multica_auth=" + loginOne.Token, "X-Workspace-Slug": "one"}
	noCSRF := runtimeRequest(runtime, http.MethodPut, "/api/issues/ONE-1/metadata/cookie", `{"value":true}`, cookieHeaders)
	assertRuntimeResponse(t, noCSRF.Code, noCSRF.Body.String(), http.StatusForbidden, `{"error":"invalid CSRF token"}`)
	cookieHeaders["X-CSRF-Token"] = loginOne.CSRF
	cookiePut := runtimeRequest(runtime, http.MethodPut, "/api/issues/ONE-1/metadata/cookie", `{"value":true}`, cookieHeaders)
	assertRuntimeResponse(t, cookiePut.Code, cookiePut.Body.String(), http.StatusOK, `{"metadata":{"cookie":true,"count":2,"seed":true}}`)
	noDeleteCSRF := runtimeRequest(runtime, http.MethodDelete, "/api/issues/ONE-1/metadata/cookie", "", map[string]string{
		"Cookie": "multica_auth=" + loginOne.Token, "X-Workspace-Slug": "one",
	})
	assertRuntimeResponse(t, noDeleteCSRF.Code, noDeleteCSRF.Body.String(), http.StatusForbidden, `{"error":"invalid CSRF token"}`)
	deleted := runtimeRequest(runtime, http.MethodDelete, "/api/issues/ONE-1/metadata/cookie", "", cookieHeaders)
	assertRuntimeResponse(t, deleted.Code, deleted.Body.String(), http.StatusOK, `{"metadata":{"count":2,"seed":true}}`)

	readback := runtimeRequest(runtime, http.MethodGet, "/api/issues/ONE-1", "", bearerHeaders)
	if readback.Code != http.StatusOK || !strings.Contains(readback.Body.String(), `"metadata":{"count":2,"seed":true}`) {
		t.Fatalf("detail readback = %d %s", readback.Code, readback.Body.String())
	}
	capabilities := runtimeRequest(runtime, http.MethodGet, "/api/config", "", nil)
	if capabilities.Code != http.StatusOK || !strings.Contains(capabilities.Body.String(), `"issue_metadata":true`) {
		t.Fatalf("metadata capability = %d %s", capabilities.Code, capabilities.Body.String())
	}

	if err := runtime.Close(); err != nil {
		t.Fatal(err)
	}
	restarted := newRuntimeForConfig(t, config)
	restartedGet := runtimeRequest(restarted, http.MethodGet, "/api/issues/ONE-1/metadata", "", bearerHeaders)
	assertRuntimeResponse(t, restartedGet.Code, restartedGet.Body.String(), http.StatusOK, `{"metadata":{"count":2,"seed":true}}`)

	now = now.Add(25 * time.Hour)
	expired := runtimeRequest(restarted, http.MethodGet, "/api/issues/ONE-1/metadata", "", bearerHeaders)
	assertRuntimeResponse(t, expired.Code, expired.Body.String(), http.StatusUnauthorized, `{"error":"user not authenticated"}`)
}

func TestSQLiteRuntimeCanDisableIssueMetadataCapabilityAndRoutes(t *testing.T) {
	runtime := newRuntimeForConfig(t, Config{
		Name: "backend-test", Version: "test",
		HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath:            filepath.Join(t.TempDir(), "metadata-disabled.db"),
		WorkspaceDependencies: FailClosedWorkspaceDependencies(),
		LocalAuth:             auth.LocalAuthConfig{VerificationCode: "888888"},
		IssueMetadataEnabled:  boolPointer(false),
	})
	capabilities := runtimeRequest(runtime, http.MethodGet, "/api/config", "", nil)
	if capabilities.Code != http.StatusOK || !strings.Contains(capabilities.Body.String(), `"issue_metadata":false`) {
		t.Fatalf("disabled capability = %d %s", capabilities.Code, capabilities.Body.String())
	}
	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
		path := "/api/issues/issue-one/metadata"
		if method != http.MethodGet {
			path += "/key"
		}
		response := runtimeRequest(runtime, method, path, `{"value":true}`, nil)
		if response.Code != http.StatusNotFound {
			t.Fatalf("disabled %s route = %d %s", method, response.Code, response.Body.String())
		}
	}
}

func TestSQLiteRuntimeCanDisableIssueCreateCapabilityAndRoute(t *testing.T) {
	runtime := newRuntimeForConfig(t, Config{
		Name: "backend-test", Version: "test",
		HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath:            filepath.Join(t.TempDir(), "issue-create-disabled.db"),
		WorkspaceDependencies: FailClosedWorkspaceDependencies(),
		LocalAuth:             auth.LocalAuthConfig{VerificationCode: "888888"},
		IssueCreateEnabled:    boolPointer(false),
	})
	capabilities := runtimeRequest(runtime, http.MethodGet, "/api/config", "", nil)
	if capabilities.Code != http.StatusOK || !strings.Contains(capabilities.Body.String(), `"issue_create":false`) {
		t.Fatalf("disabled capability = %d %s", capabilities.Code, capabilities.Body.String())
	}
	response := runtimeRequest(runtime, http.MethodPost, "/api/issues", `{"title":"disabled"}`, nil)
	if response.Code != http.StatusNotFound {
		t.Fatalf("disabled create route = %d %s", response.Code, response.Body.String())
	}
}

// TestPrepareIssueMetadataBrowserFixture prepares an explicitly requested,
// persistent SQLite fixture for real-listener browser acceptance. It is skipped
// during normal test runs and refuses to overwrite an existing file.
func TestPrepareIssueMetadataBrowserFixture(t *testing.T) {
	databasePath := strings.TrimSpace(os.Getenv("MULTICA_METADATA_BROWSER_FIXTURE_DB"))
	if databasePath == "" {
		t.Skip("set MULTICA_METADATA_BROWSER_FIXTURE_DB to prepare the browser fixture")
	}
	if _, err := os.Stat(databasePath); err == nil {
		t.Fatalf("fixture path already exists: %s", databasePath)
	} else if !os.IsNotExist(err) {
		t.Fatal(err)
	}
	runtime := newRuntimeForConfig(t, Config{
		Name: "backend-test", Version: "browser-fixture",
		HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath: databasePath, WorkspaceDependencies: FailClosedWorkspaceDependencies(),
		LocalAuth: auth.LocalAuthConfig{VerificationCode: "888888", SessionTTL: 24 * time.Hour},
	})
	login := verifyRuntimeLogin(t, runtime, "browser@example.com")
	for _, statement := range []string{
		`INSERT INTO workspaces(id,name,slug,settings,repos,issue_prefix,created_at,updated_at) VALUES
			('workspace-browser','Browser Fixture','browser','{}','[]','BROWSE','2026-08-13T00:00:00Z','2026-08-13T00:00:00Z')`,
		`INSERT INTO auth_members(id,workspace_id,user_id,role,created_at) VALUES
			('member-browser','workspace-browser','` + login.UserID + `','owner','2026-08-13T00:00:00Z')`,
		`INSERT INTO workspace_issues(id,workspace_id,number,identifier,title,description,status,priority,creator_type,creator_id,position,metadata,properties,asset_ids,created_at,updated_at) VALUES
			('issue-browser','workspace-browser',1,'BROWSE-1','Metadata browser readback','S5 real-listener fixture','todo','high','member','member-browser',1,'{"fixture":"ready"}','{}','[]','2026-08-13T00:00:01Z','2026-08-13T00:00:01Z')`,
	} {
		if _, err := runtime.Database().Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("fixture ready: %s; login browser@example.com / 888888; detail /browser/issues/issue-browser", databasePath)
}

func boolPointer(value bool) *bool { return &value }

type runtimeLogin struct{ Token, CSRF, UserID string }

func verifyRuntimeLogin(t *testing.T, runtime *Runtime, email string) runtimeLogin {
	t.Helper()
	response := runtimeRequest(runtime, http.MethodPost, "/auth/verify-code", `{"email":"`+email+`","code":"888888"}`, nil)
	if response.Code != http.StatusOK {
		t.Fatalf("verify %s = %d %s", email, response.Code, response.Body.String())
	}
	var login struct {
		Token string `json:"token"`
		User  struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &login); err != nil {
		t.Fatal(err)
	}
	result := runtimeLogin{Token: login.Token, UserID: login.User.ID}
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == "multica_csrf" {
			result.CSRF = cookie.Value
		}
	}
	if result.Token == "" || result.CSRF == "" || result.UserID == "" {
		t.Fatalf("missing login token or CSRF cookie: %s", response.Body.String())
	}
	return result
}

func assertRuntimeResponse(t *testing.T, gotStatus int, gotBody string, wantStatus int, wantBody string) {
	t.Helper()
	if gotStatus != wantStatus || strings.TrimSpace(gotBody) != wantBody {
		t.Fatalf("response = %d %s, want %d %s", gotStatus, gotBody, wantStatus, wantBody)
	}
}
