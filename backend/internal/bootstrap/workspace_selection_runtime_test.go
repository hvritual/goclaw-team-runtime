package bootstrap

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/auth"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

func TestSQLiteRuntimeListsOnlyAuthenticatedWorkspaceMemberships(t *testing.T) {
	now := time.Date(2026, 8, 13, 1, 2, 3, 0, time.UTC)
	sequence := 0
	runtime := newRuntimeForConfig(t, Config{
		Name: "backend-test", Version: "test",
		HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath:            filepath.Join(t.TempDir(), "workspace-selection.db"),
		WorkspaceDependencies: FailClosedWorkspaceDependencies(),
		LocalAuth: auth.LocalAuthConfig{
			VerificationCode: "888888", SessionTTL: time.Hour, Now: func() time.Time { return now },
			NewID: func(context.Context) (string, error) {
				sequence++
				return []string{"user-one-token", "user-two-token"}[sequence-1], nil
			},
		},
	})
	userOne := verifyRuntimeUser(t, runtime, "one@example.com")
	userTwo := verifyRuntimeUser(t, runtime, "two@example.com")
	for _, statement := range []string{
		`INSERT INTO workspaces(id,name,slug,settings,repos,issue_prefix,created_at,updated_at) VALUES
			('workspace-owner','Owner Space','owner-space','{}','[]','OWN','2026-08-13T00:00:01Z','2026-08-13T00:00:01Z'),
			('workspace-admin','Admin Space','admin-space','{}','[]','ADM','2026-08-13T00:00:02Z','2026-08-13T00:00:02Z'),
			('workspace-member','Member Space','member-space','{}','[]','MEM','2026-08-13T00:00:03Z','2026-08-13T00:00:03Z'),
			('workspace-foreign','Foreign Space','foreign-space','{}','[]','FOR','2026-08-13T00:00:04Z','2026-08-13T00:00:04Z')`,
		`INSERT INTO auth_members(id,workspace_id,user_id,role,created_at) VALUES
			('member-owner','workspace-owner','` + userOne + `','owner','2026-08-13T00:00:01Z'),
			('member-admin','workspace-admin','` + userOne + `','admin','2026-08-13T00:00:02Z'),
			('member-member','workspace-member','` + userOne + `','member','2026-08-13T00:00:03Z'),
			('member-foreign','workspace-foreign','` + userTwo + `','owner','2026-08-13T00:00:04Z')`,
	} {
		if _, err := runtime.Database().Exec(statement); err != nil {
			t.Fatal(err)
		}
	}

	missing := runtimeRequest(runtime, http.MethodGet, "/api/workspaces", "", nil)
	if missing.Code != http.StatusUnauthorized {
		t.Fatalf("missing = %d %s", missing.Code, missing.Body.String())
	}
	authorized := runtimeRequest(runtime, http.MethodGet, "/api/workspaces", "", map[string]string{"Authorization": "Bearer user-one-token"})
	if authorized.Code != http.StatusOK {
		t.Fatalf("authorized = %d %s", authorized.Code, authorized.Body.String())
	}
	var rawWorkspaces []map[string]json.RawMessage
	if err := json.Unmarshal(authorized.Body.Bytes(), &rawWorkspaces); err != nil {
		t.Fatal(err)
	}
	exactKeys := map[string]bool{
		"id": true, "name": true, "slug": true, "description": true,
		"context": true, "settings": true, "repos": true, "issue_prefix": true,
		"avatar_url": true, "created_at": true, "updated_at": true,
	}
	for _, rawWorkspace := range rawWorkspaces {
		if len(rawWorkspace) != len(exactKeys) {
			t.Fatalf("workspace keys = %#v, want exactly %#v", rawWorkspace, exactKeys)
		}
		for key := range rawWorkspace {
			if !exactKeys[key] {
				t.Fatalf("unexpected workspace key %q in %#v", key, rawWorkspace)
			}
		}
	}
	var workspaces []contract.WorkspaceSelection
	if err := json.Unmarshal(authorized.Body.Bytes(), &workspaces); err != nil {
		t.Fatal(err)
	}
	if len(workspaces) != 3 || workspaces[0].ID != "workspace-owner" || workspaces[2].ID != "workspace-member" {
		t.Fatalf("workspaces = %#v", workspaces)
	}

	for _, probe := range []struct {
		name, workspaceID, slug string
	}{
		{name: "same user authorized id and slug mismatch", workspaceID: "workspace-owner", slug: "admin-space"},
		{name: "missing slug", slug: "missing-space"},
		{name: "foreign id", workspaceID: "workspace-foreign"},
	} {
		t.Run(probe.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/issues/issue-1/metadata", nil)
			request.Header.Set("Authorization", "Bearer user-one-token")
			if probe.workspaceID != "" {
				request.Header.Set("X-Workspace-ID", probe.workspaceID)
			}
			if probe.slug != "" {
				request.Header.Set("X-Workspace-Slug", probe.slug)
			}
			response := httptest.NewRecorder()
			runtime.HTTPServer().ServeHTTP(response, request)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("workspace identity = %d %s", response.Code, response.Body.String())
			}
		})
	}

	detail := runtimeRequest(runtime, http.MethodGet, "/api/workspaces/workspace-owner", "", map[string]string{"Authorization": "Bearer user-one-token"})
	if detail.Code != http.StatusNotFound {
		t.Fatalf("workspace detail route = %d %s", detail.Code, detail.Body.String())
	}

	now = now.Add(2 * time.Hour)
	expired := runtimeRequest(runtime, http.MethodGet, "/api/workspaces", "", map[string]string{"Authorization": "Bearer user-one-token"})
	if expired.Code != http.StatusUnauthorized {
		t.Fatalf("expired = %d %s", expired.Code, expired.Body.String())
	}
}

func verifyRuntimeUser(t *testing.T, runtime *Runtime, email string) string {
	t.Helper()
	response := runtimeRequest(runtime, http.MethodPost, "/auth/verify-code", `{"email":"`+email+`","code":"888888"}`, nil)
	if response.Code != http.StatusOK {
		t.Fatalf("verify %s = %d %s", email, response.Code, response.Body.String())
	}
	var login struct {
		User struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &login); err != nil {
		t.Fatal(err)
	}
	return login.User.ID
}

func runtimeRequest(runtime *Runtime, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	response := httptest.NewRecorder()
	runtime.HTTPServer().ServeHTTP(response, request)
	return response
}
