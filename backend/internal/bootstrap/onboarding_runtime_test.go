package bootstrap

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/auth"
)

func TestSQLiteRuntimeCompletesNewUserOnboarding(t *testing.T) {
	now := time.Date(2026, 8, 14, 2, 3, 4, 0, time.UTC)
	databasePath := filepath.Join(t.TempDir(), "onboarding.db")
	dependencies := FailClosedWorkspaceDependencies()
	dependencies.Now = func() time.Time { return now }
	config := Config{
		Name: "backend-test", Version: "test",
		HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath:            databasePath,
		WorkspaceDependencies: dependencies,
		LocalAuth: auth.LocalAuthConfig{
			VerificationCode: "888888", SessionTTL: 24 * time.Hour,
			Now: func() time.Time { return now },
		},
	}
	runtime := newRuntimeForConfig(t, config)
	login := verifyRuntimeLogin(t, runtime, "new-user@example.com")
	headers := map[string]string{
		"Cookie":       "multica_auth=" + login.Token,
		"X-CSRF-Token": login.CSRF,
		"Content-Type": "application/json",
		"X-Request-ID": "onboarding-create",
	}

	missing := runtimeRequest(runtime, http.MethodPost, "/api/workspaces", `{"name":"New Team","slug":"new-team"}`, nil)
	assertRuntimeResponse(t, missing.Code, missing.Body.String(), http.StatusUnauthorized, `{"error":"user not authenticated"}`)
	noCSRF := runtimeRequest(runtime, http.MethodPost, "/api/workspaces", `{"name":"New Team","slug":"new-team"}`, map[string]string{"Cookie": "multica_auth=" + login.Token})
	assertRuntimeResponse(t, noCSRF.Code, noCSRF.Body.String(), http.StatusForbidden, `{"error":"invalid CSRF token"}`)
	invalid := runtimeRequest(runtime, http.MethodPost, "/api/workspaces", `{"name":"","slug":"Bad Slug"}`, headers)
	assertRuntimeResponse(t, invalid.Code, invalid.Body.String(), http.StatusBadRequest, `{"error":"valid name and slug are required"}`)
	unknown := runtimeRequest(runtime, http.MethodPost, "/api/workspaces", `{"name":"New Team","slug":"new-team","unknown":true}`, headers)
	assertRuntimeResponse(t, unknown.Code, unknown.Body.String(), http.StatusBadRequest, `{"error":"invalid request body"}`)
	trailing := runtimeRequest(runtime, http.MethodPost, "/api/workspaces", `{"name":"New Team","slug":"new-team"}{}`, headers)
	assertRuntimeResponse(t, trailing.Code, trailing.Body.String(), http.StatusBadRequest, `{"error":"invalid request body"}`)
	expired := verifyRuntimeLogin(t, runtime, "expired-onboarding@example.com")
	if _, err := runtime.Database().Exec(`UPDATE auth_sessions SET expires_at_unix_nano=0 WHERE user_id=?`, expired.UserID); err != nil {
		t.Fatal(err)
	}
	expiredCreate := runtimeRequest(runtime, http.MethodPost, "/api/workspaces", `{"name":"Expired","slug":"expired"}`, map[string]string{
		"Authorization": "Bearer " + expired.Token, "Content-Type": "application/json",
	})
	assertRuntimeResponse(t, expiredCreate.Code, expiredCreate.Body.String(), http.StatusUnauthorized, `{"error":"user not authenticated"}`)

	created := runtimeRequest(runtime, http.MethodPost, "/api/workspaces", `{"name":"New Team","slug":"new-team","description":"First workspace"}`, headers)
	if created.Code != http.StatusCreated {
		t.Fatalf("create workspace = %d %s", created.Code, created.Body.String())
	}
	var workspace struct {
		ID   string `json:"id"`
		Slug string `json:"slug"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &workspace); err != nil {
		t.Fatal(err)
	}
	if workspace.ID == "" || workspace.Slug != "new-team" {
		t.Fatalf("workspace = %s", created.Body.String())
	}
	var rawWorkspace map[string]json.RawMessage
	if err := json.Unmarshal(created.Body.Bytes(), &rawWorkspace); err != nil {
		t.Fatal(err)
	}
	if len(rawWorkspace) != 11 {
		t.Fatalf("workspace keys = %#v", rawWorkspace)
	}
	for _, key := range []string{"id", "name", "slug", "description", "context", "settings", "repos", "issue_prefix", "avatar_url", "created_at", "updated_at"} {
		if _, ok := rawWorkspace[key]; !ok {
			t.Fatalf("workspace missing key %q: %s", key, created.Body.String())
		}
	}
	for key, expected := range map[string]string{
		"name": `"New Team"`, "slug": `"new-team"`, "description": `"First workspace"`, "context": `null`,
		"settings": `{}`, "repos": `[]`, "issue_prefix": `"NEW"`, "avatar_url": `null`,
		"created_at": `"2026-08-14T02:03:04Z"`, "updated_at": `"2026-08-14T02:03:04Z"`,
	} {
		if string(rawWorkspace[key]) != expected {
			t.Fatalf("workspace %s = %s, want %s", key, rawWorkspace[key], expected)
		}
	}
	for table, query := range map[string]string{
		"workspace": `SELECT COUNT(*) FROM workspaces WHERE id=?`,
		"member":    `SELECT COUNT(*) FROM auth_members WHERE workspace_id=? AND user_id=? AND role='owner'`,
		"root":      `SELECT COUNT(*) FROM auth_workspace_membership_roots WHERE workspace_id=? AND user_id=?`,
	} {
		arguments := []any{workspace.ID}
		if table != "workspace" {
			arguments = append(arguments, login.UserID)
		}
		var count int
		if err := runtime.Database().QueryRow(query, arguments...).Scan(&count); err != nil || count != 1 {
			t.Fatalf("%s count/error = %d/%v", table, count, err)
		}
	}
	duplicate := runtimeRequest(runtime, http.MethodPost, "/api/workspaces", `{"name":"Duplicate","slug":"new-team"}`, headers)
	assertRuntimeResponse(t, duplicate.Code, duplicate.Body.String(), http.StatusConflict, `{"error":"workspace slug already exists"}`)
	expiredComplete := runtimeRequest(runtime, http.MethodPost, "/api/me/onboarding/complete", `{"workspace_id":"`+workspace.ID+`"}`, map[string]string{
		"Authorization": "Bearer " + expired.Token, "Content-Type": "application/json",
	})
	assertRuntimeResponse(t, expiredComplete.Code, expiredComplete.Body.String(), http.StatusUnauthorized, `{"error":"invalid token"}`)

	outsider := verifyRuntimeLogin(t, runtime, "outsider@example.com")
	foreignComplete := runtimeRequest(runtime, http.MethodPost, "/api/me/onboarding/complete", `{"completion_path":"full","workspace_id":"`+workspace.ID+`"}`, map[string]string{
		"Authorization": "Bearer " + outsider.Token, "Content-Type": "application/json",
	})
	assertRuntimeResponse(t, foreignComplete.Code, foreignComplete.Body.String(), http.StatusBadRequest, `{"error":"workspace is not available to the current user"}`)
	completionNoCSRF := runtimeRequest(runtime, http.MethodPost, "/api/me/onboarding/complete", `{"workspace_id":"`+workspace.ID+`"}`, map[string]string{"Cookie": "multica_auth=" + login.Token})
	assertRuntimeResponse(t, completionNoCSRF.Code, completionNoCSRF.Body.String(), http.StatusForbidden, `{"error":"invalid CSRF token"}`)

	completed := runtimeRequest(runtime, http.MethodPost, "/api/me/onboarding/complete", `{"completion_path":"full","workspace_id":"`+workspace.ID+`"}`, headers)
	if completed.Code != http.StatusOK {
		t.Fatalf("complete onboarding = %d %s", completed.Code, completed.Body.String())
	}
	var user struct {
		ID          string  `json:"id"`
		OnboardedAt *string `json:"onboarded_at"`
	}
	if err := json.Unmarshal(completed.Body.Bytes(), &user); err != nil {
		t.Fatal(err)
	}
	if user.ID != login.UserID || user.OnboardedAt == nil || *user.OnboardedAt == "" {
		t.Fatalf("completed user = %s", completed.Body.String())
	}
	firstOnboardedAt := *user.OnboardedAt
	var rawUser map[string]json.RawMessage
	if err := json.Unmarshal(completed.Body.Bytes(), &rawUser); err != nil {
		t.Fatal(err)
	}
	if len(rawUser) != 12 {
		t.Fatalf("user keys = %#v", rawUser)
	}
	for _, key := range []string{"id", "name", "email", "avatar_url", "onboarded_at", "onboarding_questionnaire", "starter_content_state", "language", "profile_description", "timezone", "created_at", "updated_at"} {
		if _, ok := rawUser[key]; !ok {
			t.Fatalf("completed user missing key %q: %s", key, completed.Body.String())
		}
	}
	unknownCompletion := runtimeRequest(runtime, http.MethodPost, "/api/me/onboarding/complete", `{"workspace_id":"`+workspace.ID+`","unknown":true}`, headers)
	assertRuntimeResponse(t, unknownCompletion.Code, unknownCompletion.Body.String(), http.StatusBadRequest, `{"error":"invalid request body"}`)
	trailingCompletion := runtimeRequest(runtime, http.MethodPost, "/api/me/onboarding/complete", `{"workspace_id":"`+workspace.ID+`"}{}`, headers)
	assertRuntimeResponse(t, trailingCompletion.Code, trailingCompletion.Body.String(), http.StatusBadRequest, `{"error":"invalid request body"}`)

	now = now.Add(time.Hour)
	retried := runtimeRequest(runtime, http.MethodPost, "/api/me/onboarding/complete", `{"completion_path":"full","workspace_id":"`+workspace.ID+`"}`, headers)
	if retried.Code != http.StatusOK || !strings.Contains(retried.Body.String(), `"onboarded_at":"`+firstOnboardedAt+`"`) {
		t.Fatalf("retry = %d %s", retried.Code, retried.Body.String())
	}
	if err := runtime.Close(); err != nil {
		t.Fatal(err)
	}
	restarted := newRuntimeForConfig(t, config)
	me := runtimeRequest(restarted, http.MethodGet, "/api/me", "", map[string]string{"Cookie": "multica_auth=" + login.Token})
	if me.Code != http.StatusOK || !strings.Contains(me.Body.String(), `"onboarded_at":"`+firstOnboardedAt+`"`) {
		t.Fatalf("restart me = %d %s", me.Code, me.Body.String())
	}
	restartedRetry := runtimeRequest(restarted, http.MethodPost, "/api/me/onboarding/complete", `{"workspace_id":"`+workspace.ID+`"}`, headers)
	if restartedRetry.Code != http.StatusOK || !strings.Contains(restartedRetry.Body.String(), `"onboarded_at":"`+firstOnboardedAt+`"`) {
		t.Fatalf("restart retry = %d %s", restartedRetry.Code, restartedRetry.Body.String())
	}
}

func TestSQLiteRuntimeDoesNotReportDuplicateWorkspaceIDAsSlugConflict(t *testing.T) {
	dependencies := FailClosedWorkspaceDependencies()
	dependencies.NewWorkspaceID = func(context.Context) (string, error) { return "fixed-workspace-id", nil }
	var memberSequence int
	dependencies.NewWorkspaceMemberID = func(context.Context) (string, error) {
		memberSequence++
		return "member-" + string(rune('0'+memberSequence)), nil
	}
	runtime := newRuntimeForConfig(t, Config{
		Name: "backend-test", Version: "test", HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath: filepath.Join(t.TempDir(), "onboarding-id-conflict.db"), WorkspaceDependencies: dependencies,
		LocalAuth: auth.LocalAuthConfig{VerificationCode: "888888"},
	})
	login := verifyRuntimeLogin(t, runtime, "id-conflict@example.com")
	headers := map[string]string{"Authorization": "Bearer " + login.Token, "Content-Type": "application/json"}
	first := runtimeRequest(runtime, http.MethodPost, "/api/workspaces", `{"name":"First","slug":"first"}`, headers)
	if first.Code != http.StatusCreated {
		t.Fatalf("first create = %d %s", first.Code, first.Body.String())
	}
	second := runtimeRequest(runtime, http.MethodPost, "/api/workspaces", `{"name":"Second","slug":"second"}`, headers)
	assertRuntimeResponse(t, second.Code, second.Body.String(), http.StatusInternalServerError, `{"error":"failed to create workspace"}`)
}

func TestSQLiteRuntimeRollsBackWorkspaceWhenOwnerCreationFails(t *testing.T) {
	runtime := newRuntimeForConfig(t, Config{
		Name: "backend-test", Version: "test",
		HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath:            filepath.Join(t.TempDir(), "onboarding-rollback.db"),
		WorkspaceDependencies: FailClosedWorkspaceDependencies(),
		LocalAuth:             auth.LocalAuthConfig{VerificationCode: "888888"},
	})
	login := verifyRuntimeLogin(t, runtime, "rollback@example.com")
	if _, err := runtime.Database().Exec(`CREATE TRIGGER fail_workspace_owner BEFORE INSERT ON auth_members BEGIN SELECT RAISE(ABORT,'forced owner failure'); END`); err != nil {
		t.Fatal(err)
	}
	response := runtimeRequest(runtime, http.MethodPost, "/api/workspaces", `{"name":"Rollback","slug":"rollback"}`, map[string]string{
		"Cookie": "multica_auth=" + login.Token, "X-CSRF-Token": login.CSRF, "Content-Type": "application/json",
	})
	assertRuntimeResponse(t, response.Code, response.Body.String(), http.StatusInternalServerError, `{"error":"failed to create workspace"}`)
	for table := range map[string]bool{"workspaces": true, "auth_members": true, "auth_workspace_membership_roots": true} {
		var count int
		if err := runtime.Database().QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil || count != 0 {
			t.Fatalf("%s count/error = %d/%v", table, count, err)
		}
	}
}

func TestSQLiteRuntimeSerializesConcurrentWorkspaceSlugCreation(t *testing.T) {
	runtime := newRuntimeForConfig(t, Config{
		Name: "backend-test", Version: "test",
		HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath:            filepath.Join(t.TempDir(), "onboarding-concurrent.db"),
		WorkspaceDependencies: FailClosedWorkspaceDependencies(),
		LocalAuth:             auth.LocalAuthConfig{VerificationCode: "888888"},
	})
	login := verifyRuntimeLogin(t, runtime, "concurrent@example.com")
	responses := make([]int, 2)
	var wait sync.WaitGroup
	for index := range responses {
		wait.Add(1)
		go func() {
			defer wait.Done()
			response := runtimeRequest(runtime, http.MethodPost, "/api/workspaces", `{"name":"Concurrent","slug":"concurrent"}`, map[string]string{
				"Authorization": "Bearer " + login.Token, "Content-Type": "application/json",
			})
			responses[index] = response.Code
		}()
	}
	wait.Wait()
	if !((responses[0] == http.StatusCreated && responses[1] == http.StatusConflict) ||
		(responses[1] == http.StatusCreated && responses[0] == http.StatusConflict)) {
		t.Fatalf("concurrent statuses = %v", responses)
	}
	for table := range map[string]bool{"workspaces": true, "auth_members": true, "auth_workspace_membership_roots": true} {
		var count int
		if err := runtime.Database().QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil || count != 1 {
			t.Fatalf("%s count/error = %d/%v", table, count, err)
		}
	}
}
