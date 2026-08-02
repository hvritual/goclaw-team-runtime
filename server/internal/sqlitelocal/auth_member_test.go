package sqlitelocal

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestSQLiteLocalAuthMemberFallbackErrorsRemainOperationSpecific(t *testing.T) {
	tests := []string{
		"failed to list members",
		"failed to update member",
		"failed to delete member",
		"failed to leave workspace",
		"failed to revoke invitation",
		"failed to list invitations",
		"failed to create invitation",
		"failed to authorize invitation",
	}
	for _, fallback := range tests {
		t.Run(fallback, func(t *testing.T) {
			response := httptest.NewRecorder()
			writeMemberError(response, errors.New("database unavailable"), fallback)
			if response.Code != http.StatusInternalServerError || !strings.Contains(response.Body.String(), fallback) {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
		})
	}
}

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

func TestSQLiteLocalInvitationCreationAuthorizesBeforeDecodingBody(t *testing.T) {
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
		"email": "owner@example.com", "code": "888888",
	}, http.StatusOK)
	owner.token = ownerLogin["token"].(string)
	workspace := owner.request(http.MethodPost, "/api/workspaces", map[string]any{
		"name": "Invitation Authorization", "slug": "invitation-authorization",
	}, http.StatusCreated)
	workspaceID := workspace["id"].(string)
	created := owner.request(http.MethodPost, "/api/workspaces/"+workspaceID+"/members", map[string]any{
		"email": "member@example.com", "role": "member",
	}, http.StatusCreated)

	memberClient := &testClient{t: t, app: app}
	memberLogin := memberClient.request(http.MethodPost, "/auth/verify-code", map[string]any{
		"email": "member@example.com", "code": "888888",
	}, http.StatusOK)
	memberClient.token = memberLogin["token"].(string)
	memberClient.request(http.MethodPost, "/api/invitations/"+created["id"].(string)+"/accept", nil, http.StatusOK)

	request := httptest.NewRequestWithContext(
		t.Context(), http.MethodPost, "/api/workspaces/"+workspaceID+"/members", bytes.NewBufferString("{"),
	)
	request.Header.Set("Authorization", "Bearer "+memberClient.token)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	app.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusForbidden, response.Body.String())
	}
}

func TestSQLiteLocalAuthModuleRemovesAndLeavesMemberships(t *testing.T) {
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
		"name": "Membership Lifecycle",
		"slug": "membership-lifecycle",
	}, http.StatusCreated)
	workspaceID := workspace["id"].(string)

	acceptMember := func(email string) (*testClient, string) {
		t.Helper()
		invitation := owner.request(http.MethodPost, "/api/workspaces/"+workspaceID+"/members", map[string]any{
			"email": email,
			"role":  "member",
		}, http.StatusCreated)
		client := &testClient{t: t, app: app}
		login := client.request(http.MethodPost, "/auth/verify-code", map[string]any{
			"email": email,
			"code":  "888888",
		}, http.StatusOK)
		client.token = login["token"].(string)
		membership := client.request(http.MethodPost, "/api/invitations/"+invitation["id"].(string)+"/accept", nil, http.StatusOK)
		return client, membership["id"].(string)
	}

	_, removedMemberID := acceptMember("removed@example.com")
	owner.request(http.MethodDelete, "/api/workspaces/"+workspaceID+"/members/"+removedMemberID, nil, http.StatusNoContent)
	leavingMember, _ := acceptMember("leaving@example.com")
	leavingMember.request(http.MethodPost, "/api/workspaces/"+workspaceID+"/leave", nil, http.StatusNoContent)

	members := owner.requestList(http.MethodGet, "/api/workspaces/"+workspaceID+"/members", http.StatusOK)
	if len(members) != 1 || members[0]["role"] != "owner" {
		t.Fatalf("unexpected remaining memberships: %#v", members)
	}
}
