package sqlitelocal

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestSQLiteLocalAuthModuleProtectsLastOwnerRole(t *testing.T) {
	app, err := Open(filepath.Join(t.TempDir(), "multica.db"), Options{
		VerificationCode: "888888",
		DisableKnowledge: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = app.Close() })

	owner := &testClient{t: t, app: app}
	login := owner.request(http.MethodPost, "/auth/verify-code", map[string]any{
		"email": "owner@example.com",
		"code":  "888888",
	}, http.StatusOK)
	owner.token = login["token"].(string)
	workspace := owner.request(http.MethodPost, "/api/workspaces", map[string]any{
		"name": "Auth Module",
		"slug": "auth-module",
	}, http.StatusCreated)

	members := owner.requestList(http.MethodGet, "/api/workspaces/"+workspace["id"].(string)+"/members", http.StatusOK)
	if len(members) != 1 {
		t.Fatalf("members = %#v", members)
	}
	owner.request(
		http.MethodPatch,
		"/api/workspaces/"+workspace["id"].(string)+"/members/"+members[0]["id"].(string),
		map[string]any{"role": "member"},
		http.StatusBadRequest,
	)

	unchanged := owner.requestList(http.MethodGet, "/api/workspaces/"+workspace["id"].(string)+"/members", http.StatusOK)
	if len(unchanged) != 1 || unchanged[0]["role"] != "owner" {
		t.Fatalf("last owner changed: %#v", unchanged)
	}
}

func TestSQLiteLocalAuthModuleHidesWorkspaceBeforeDecodingRoleChange(t *testing.T) {
	app, err := Open(filepath.Join(t.TempDir(), "multica.db"), Options{
		VerificationCode: "888888",
		DisableKnowledge: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = app.Close() })

	owner := &testClient{t: t, app: app}
	ownerLogin := owner.request(http.MethodPost, "/auth/verify-code", map[string]any{
		"email": "owner@example.com",
		"code":  "888888",
	}, http.StatusOK)
	owner.token = ownerLogin["token"].(string)
	workspace := owner.request(http.MethodPost, "/api/workspaces", map[string]any{
		"name": "Private Auth Module",
		"slug": "private-auth-module",
	}, http.StatusCreated)

	outsider := &testClient{t: t, app: app}
	outsiderLogin := outsider.request(http.MethodPost, "/auth/verify-code", map[string]any{
		"email": "outsider@example.com",
		"code":  "888888",
	}, http.StatusOK)
	outsider.token = outsiderLogin["token"].(string)
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPatch,
		"/api/workspaces/"+workspace["id"].(string)+"/members/member-1",
		bytes.NewBufferString("{"),
	)
	req.Header.Set("Authorization", "Bearer "+outsider.token)
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	app.Handler().ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusNotFound, recorder.Body.String())
	}
}
