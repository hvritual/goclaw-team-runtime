package sqlitelocal

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

type testClient struct {
	t     *testing.T
	app   *Server
	token string
	slug  string
}

func (c *testClient) request(method, path string, body any, wantStatus int) map[string]any {
	c.t.Helper()

	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			c.t.Fatal(err)
		}
	}

	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if c.slug != "" {
		req.Header.Set("X-Workspace-Slug", c.slug)
	}
	rec := httptest.NewRecorder()
	c.app.Handler().ServeHTTP(rec, req)
	if rec.Code != wantStatus {
		c.t.Fatalf("%s %s: got status %d, want %d; body=%s", method, path, rec.Code, wantStatus, rec.Body.String())
	}
	if rec.Body.Len() == 0 {
		return nil
	}
	var result map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		c.t.Fatalf("%s %s: decode response: %v; body=%s", method, path, err, rec.Body.String())
	}
	return result
}

func (c *testClient) requestList(method, path string, wantStatus int) []map[string]any {
	c.t.Helper()

	req := httptest.NewRequest(method, path, nil)
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if c.slug != "" {
		req.Header.Set("X-Workspace-Slug", c.slug)
	}
	rec := httptest.NewRecorder()
	c.app.Handler().ServeHTTP(rec, req)
	if rec.Code != wantStatus {
		c.t.Fatalf("%s %s: got status %d, want %d; body=%s", method, path, rec.Code, wantStatus, rec.Body.String())
	}
	var result []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		c.t.Fatalf("%s %s: decode response: %v; body=%s", method, path, err, rec.Body.String())
	}
	return result
}

func TestSQLiteLocalSixDomainAPI(t *testing.T) {
	path := filepath.Join(t.TempDir(), "multica.db")
	app, err := Open(path, Options{VerificationCode: "888888"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = app.Close() })

	client := &testClient{t: t, app: app}
	client.request(http.MethodGet, "/health", nil, http.StatusOK)
	client.request(http.MethodPost, "/auth/send-code", map[string]any{"email": "owner@example.com"}, http.StatusNoContent)
	login := client.request(http.MethodPost, "/auth/verify-code", map[string]any{
		"email": "owner@example.com",
		"code":  "888888",
	}, http.StatusOK)
	client.token = login["token"].(string)

	workspace := client.request(http.MethodPost, "/api/workspaces", map[string]any{
		"name": "Local Team",
		"slug": "local-team",
	}, http.StatusCreated)
	client.slug = workspace["slug"].(string)

	members := client.requestList(http.MethodGet, "/api/workspaces/"+workspace["id"].(string)+"/members", http.StatusOK)
	if len(members) != 1 || members[0]["role"] != "owner" {
		t.Fatalf("owner membership not created: %#v", members)
	}

	project := client.request(http.MethodPost, "/api/projects", map[string]any{
		"title":    "SQLite integration",
		"status":   "in_progress",
		"priority": "high",
	}, http.StatusCreated)
	issue := client.request(http.MethodPost, "/api/issues", map[string]any{
		"title":      "Persist six domains",
		"project_id": project["id"],
		"priority":   "high",
	}, http.StatusCreated)
	if issue["identifier"] != "LOC-1" {
		t.Fatalf("unexpected issue identifier: %#v", issue["identifier"])
	}

	task := client.request(http.MethodPost, "/api/tasks", map[string]any{
		"title":      "Verify SQLite task domain",
		"project_id": project["id"],
		"issue_id":   issue["id"],
		"status":     "todo",
		"priority":   "high",
	}, http.StatusCreated)

	skill := client.request(http.MethodPost, "/api/skills", map[string]any{
		"name":        "sqlite-local",
		"description": "Local SQLite workflow",
		"content":     "# SQLite Local",
		"files": []map[string]any{
			{"path": "references/notes.md", "content": "local only"},
		},
	}, http.StatusCreated)

	if got := client.request(http.MethodGet, "/api/projects", nil, http.StatusOK); got["total"] != float64(1) {
		t.Fatalf("project list total: %#v", got)
	}
	if got := client.request(http.MethodGet, "/api/issues", nil, http.StatusOK); got["total"] != float64(1) {
		t.Fatalf("issue list total: %#v", got)
	}
	if got := client.request(http.MethodGet, "/api/tasks?status=todo", nil, http.StatusOK); got["total"] != float64(1) {
		t.Fatalf("task list: %#v", got)
	}
	if got := client.request(http.MethodGet, "/api/skills/"+skill["id"].(string), nil, http.StatusOK); len(got["files"].([]any)) != 1 {
		t.Fatalf("skill files: %#v", got)
	}

	completed := client.request(http.MethodPatch, "/api/tasks/"+task["id"].(string), map[string]any{
		"status": "done",
	}, http.StatusOK)
	if completed["status"] != "done" || completed["completed_at"] == nil {
		t.Fatalf("task not completed: %#v", completed)
	}

	unsupported := client.request(http.MethodGet, "/api/dashboard/usage", nil, http.StatusNotImplemented)
	if unsupported["code"] != "sqlite_local_unsupported" {
		t.Fatalf("unsupported response: %#v", unsupported)
	}
}

func TestSQLiteLocalPersistsAcrossRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "multica.db")
	app, err := Open(path, Options{VerificationCode: "888888"})
	if err != nil {
		t.Fatal(err)
	}
	client := &testClient{t: t, app: app}
	client.request(http.MethodPost, "/auth/send-code", map[string]any{"email": "owner@example.com"}, http.StatusNoContent)
	login := client.request(http.MethodPost, "/auth/verify-code", map[string]any{"email": "owner@example.com", "code": "888888"}, http.StatusOK)
	client.token = login["token"].(string)
	client.request(http.MethodPost, "/api/workspaces", map[string]any{"name": "Persistent", "slug": "persistent"}, http.StatusCreated)
	// Simulate a database created before the invitation table existed. Open
	// must re-apply additive schema statements even when older data is present.
	if _, err := app.db.Exec(`DROP TABLE invitations`); err != nil {
		t.Fatal(err)
	}
	if err := app.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(path, Options{VerificationCode: "888888"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	client.app = reopened

	workspaces := client.requestList(http.MethodGet, "/api/workspaces", http.StatusOK)
	if len(workspaces) != 1 || workspaces[0]["slug"] != "persistent" {
		t.Fatalf("workspace did not persist: %#v", workspaces)
	}
	client.request(http.MethodPost, "/api/workspaces/"+workspaces[0]["id"].(string)+"/members", map[string]any{
		"email": "invitee@example.com",
		"role":  "member",
	}, http.StatusCreated)
}

func TestSQLiteLocalMigratesLegacyExecutionTasks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "multica.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`
		CREATE TABLE members (
			id TEXT PRIMARY KEY,
			workspace_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			role TEXT NOT NULL,
			created_at TEXT NOT NULL
		);
		CREATE TABLE tasks (
			id TEXT PRIMARY KEY,
			workspace_id TEXT NOT NULL,
			issue_id TEXT,
			agent_id TEXT NOT NULL DEFAULT '',
			runtime_id TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL,
			priority INTEGER NOT NULL DEFAULT 0,
			dispatched_at TEXT,
			started_at TEXT,
			completed_at TEXT,
			result TEXT,
			error TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);
		CREATE TABLE task_messages (
			id TEXT PRIMARY KEY,
			task_id TEXT NOT NULL,
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at TEXT NOT NULL
		);
		INSERT INTO members(id, workspace_id, user_id, role, created_at)
		VALUES ('member-1', 'workspace-1', 'user-1', 'owner', '2026-01-01T00:00:00Z');
		INSERT INTO tasks(
			id, workspace_id, issue_id, agent_id, runtime_id, status, priority,
			dispatched_at, started_at, completed_at, result, error, created_at, updated_at
		) VALUES (
			'task-legacy', 'workspace-1', NULL, 'legacy-worker', 'legacy-machine',
			'completed', 3, NULL, '2026-01-01T00:00:00Z', '2026-01-02T00:00:00Z',
			NULL, NULL, '2026-01-01T00:00:00Z', '2026-01-02T00:00:00Z'
		);
	`)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	app, err := Open(path, Options{VerificationCode: "888888"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = app.Close() })

	var title, status, priority, creatorID string
	if err := app.db.QueryRow(
		`SELECT title, status, priority, creator_id FROM tasks WHERE id = 'task-legacy'`,
	).Scan(&title, &status, &priority, &creatorID); err != nil {
		t.Fatal(err)
	}
	if title == "" || status != "done" || priority != "high" || creatorID != "user-1" {
		t.Fatalf(
			"legacy task migration mismatch: title=%q status=%q priority=%q creator=%q",
			title,
			status,
			priority,
			creatorID,
		)
	}

	var legacyMessageTable string
	err = app.db.QueryRow(
		`SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'task_messages'`,
	).Scan(&legacyMessageTable)
	if err != sql.ErrNoRows {
		t.Fatalf("legacy task_messages table still exists: %v", err)
	}
}

func TestSQLiteLocalRejectsCrossWorkspaceRelationships(t *testing.T) {
	app, err := Open(filepath.Join(t.TempDir(), "multica.db"), Options{VerificationCode: "888888"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = app.Close() })

	client := &testClient{t: t, app: app}
	client.request(http.MethodPost, "/auth/send-code", map[string]any{"email": "owner@example.com"}, http.StatusNoContent)
	login := client.request(http.MethodPost, "/auth/verify-code", map[string]any{"email": "owner@example.com", "code": "888888"}, http.StatusOK)
	client.token = login["token"].(string)

	first := client.request(http.MethodPost, "/api/workspaces", map[string]any{"name": "First", "slug": "first"}, http.StatusCreated)
	client.slug = first["slug"].(string)
	project := client.request(http.MethodPost, "/api/projects", map[string]any{"title": "First project"}, http.StatusCreated)

	second := client.request(http.MethodPost, "/api/workspaces", map[string]any{"name": "Second", "slug": "second"}, http.StatusCreated)
	client.slug = second["slug"].(string)
	client.request(http.MethodPost, "/api/issues", map[string]any{
		"title":      "Cross-workspace issue",
		"project_id": project["id"],
	}, http.StatusBadRequest)

	client.slug = first["slug"].(string)
	issue := client.request(http.MethodPost, "/api/issues", map[string]any{"title": "First issue"}, http.StatusCreated)
	client.slug = second["slug"].(string)
	client.request(http.MethodPost, "/api/tasks", map[string]any{
		"issue_id": issue["id"],
		"title":    "Cross-workspace task",
		"status":   "todo",
	}, http.StatusBadRequest)
}

func TestSQLiteLocalMemberCannotDeleteWorkspace(t *testing.T) {
	app, err := Open(filepath.Join(t.TempDir(), "multica.db"), Options{VerificationCode: "888888"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = app.Close() })

	owner := &testClient{t: t, app: app}
	owner.request(http.MethodPost, "/auth/send-code", map[string]any{"email": "owner@example.com"}, http.StatusNoContent)
	login := owner.request(http.MethodPost, "/auth/verify-code", map[string]any{"email": "owner@example.com", "code": "888888"}, http.StatusOK)
	owner.token = login["token"].(string)
	workspace := owner.request(http.MethodPost, "/api/workspaces", map[string]any{"name": "Protected", "slug": "protected"}, http.StatusCreated)
	owner.slug = workspace["slug"].(string)
	invitation := owner.request(http.MethodPost, "/api/workspaces/"+workspace["id"].(string)+"/members", map[string]any{
		"email": "member@example.com",
		"role":  "member",
	}, http.StatusCreated)

	member := &testClient{t: t, app: app, slug: owner.slug}
	member.request(http.MethodPost, "/auth/send-code", map[string]any{"email": "member@example.com"}, http.StatusNoContent)
	memberLogin := member.request(http.MethodPost, "/auth/verify-code", map[string]any{"email": "member@example.com", "code": "888888"}, http.StatusOK)
	member.token = memberLogin["token"].(string)
	member.request(http.MethodPost, "/api/invitations/"+invitation["id"].(string)+"/accept", nil, http.StatusOK)
	member.request(http.MethodDelete, "/api/workspaces/"+workspace["id"].(string), nil, http.StatusForbidden)
}

func TestSQLiteLocalBrowserCookieLogin(t *testing.T) {
	app, err := Open(filepath.Join(t.TempDir(), "multica.db"), Options{VerificationCode: "888888"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = app.Close() })

	body := bytes.NewBufferString(`{"email":"browser@example.com","code":"888888"}`)
	verifyReq := httptest.NewRequest(http.MethodPost, "/auth/verify-code", body)
	verifyReq.Header.Set("Content-Type", "application/json")
	verifyRec := httptest.NewRecorder()
	app.Handler().ServeHTTP(verifyRec, verifyReq)
	if verifyRec.Code != http.StatusOK {
		t.Fatalf("verify code: got status %d; body=%s", verifyRec.Code, verifyRec.Body.String())
	}

	var authCookie *http.Cookie
	for _, cookie := range verifyRec.Result().Cookies() {
		if cookie.Name == "multica_auth" {
			authCookie = cookie
			break
		}
	}
	if authCookie == nil {
		t.Fatal("verify code did not set browser auth cookie")
	}

	meReq := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	meReq.AddCookie(authCookie)
	meRec := httptest.NewRecorder()
	app.Handler().ServeHTTP(meRec, meReq)
	if meRec.Code != http.StatusOK {
		t.Fatalf("cookie-authenticated /api/me: got status %d; body=%s", meRec.Code, meRec.Body.String())
	}
}

func TestSQLiteLocalOnboardingUnlocksWorkspace(t *testing.T) {
	app, err := Open(filepath.Join(t.TempDir(), "multica.db"), Options{VerificationCode: "888888"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = app.Close() })

	client := &testClient{t: t, app: app}
	login := client.request(http.MethodPost, "/auth/verify-code", map[string]any{
		"email": "onboarding@example.com",
		"code":  "888888",
	}, http.StatusOK)
	client.token = login["token"].(string)

	updated := client.request(http.MethodPatch, "/api/me/onboarding", map[string]any{
		"questionnaire": map[string]any{
			"role":     "engineer",
			"use_case": []string{"task_management"},
		},
	}, http.StatusOK)
	questionnaire, ok := updated["onboarding_questionnaire"].(map[string]any)
	if !ok || questionnaire["role"] != "engineer" {
		t.Fatalf("questionnaire was not persisted: %#v", updated)
	}
	if updated["onboarded_at"] != nil {
		t.Fatalf("saving questionnaire must not complete onboarding: %#v", updated)
	}

	workspace := client.request(http.MethodPost, "/api/workspaces", map[string]any{
		"name": "Onboarding Team",
		"slug": "onboarding-team",
	}, http.StatusCreated)
	completed := client.request(http.MethodPost, "/api/me/onboarding/complete", map[string]any{
		"completion_path": "skip_existing",
		"workspace_id":    workspace["id"],
	}, http.StatusOK)
	if completed["onboarded_at"] == nil || completed["onboarded_at"] == "" {
		t.Fatalf("onboarding completion did not unlock the workspace: %#v", completed)
	}

	me := client.request(http.MethodGet, "/api/me", nil, http.StatusOK)
	if me["onboarded_at"] != completed["onboarded_at"] {
		t.Fatalf("completed onboarding was not persisted: completed=%#v me=%#v", completed, me)
	}
}

func TestSQLiteLocalInvitationAcceptanceAddsMember(t *testing.T) {
	app, err := Open(filepath.Join(t.TempDir(), "multica.db"), Options{VerificationCode: "888888"})
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
		"name": "Invitation Team",
		"slug": "invitation-team",
	}, http.StatusCreated)

	invitation := owner.request(http.MethodPost, "/api/workspaces/"+workspace["id"].(string)+"/members", map[string]any{
		"email": "invitee@example.com",
		"role":  "member",
	}, http.StatusCreated)
	if invitation["status"] != "pending" {
		t.Fatalf("new invitation must stay pending until accepted: %#v", invitation)
	}
	owner.request(http.MethodPost, "/api/workspaces/"+workspace["id"].(string)+"/members", map[string]any{
		"email": "invitee@example.com",
		"role":  "member",
	}, http.StatusConflict)

	membersBefore := owner.requestList(http.MethodGet, "/api/workspaces/"+workspace["id"].(string)+"/members", http.StatusOK)
	if len(membersBefore) != 1 {
		t.Fatalf("pending invitation must not create a member: %#v", membersBefore)
	}
	workspaceInvitations := owner.requestList(http.MethodGet, "/api/workspaces/"+workspace["id"].(string)+"/invitations", http.StatusOK)
	if len(workspaceInvitations) != 1 || workspaceInvitations[0]["id"] != invitation["id"] {
		t.Fatalf("workspace invitation was not listed: %#v", workspaceInvitations)
	}

	intruder := &testClient{t: t, app: app}
	intruderLogin := intruder.request(http.MethodPost, "/auth/verify-code", map[string]any{
		"email": "intruder@example.com",
		"code":  "888888",
	}, http.StatusOK)
	intruder.token = intruderLogin["token"].(string)
	intruder.request(http.MethodGet, "/api/invitations/"+invitation["id"].(string), nil, http.StatusForbidden)
	intruder.request(http.MethodPost, "/api/invitations/"+invitation["id"].(string)+"/accept", nil, http.StatusForbidden)

	invitee := &testClient{t: t, app: app}
	inviteeLogin := invitee.request(http.MethodPost, "/auth/verify-code", map[string]any{
		"email": "invitee@example.com",
		"code":  "888888",
	}, http.StatusOK)
	invitee.token = inviteeLogin["token"].(string)
	pending := invitee.requestList(http.MethodGet, "/api/invitations", http.StatusOK)
	if len(pending) != 1 || pending[0]["id"] != invitation["id"] {
		t.Fatalf("invitee cannot see pending invitation: %#v", pending)
	}

	acceptedMember := invitee.request(http.MethodPost, "/api/invitations/"+invitation["id"].(string)+"/accept", nil, http.StatusOK)
	if acceptedMember["user_id"] != inviteeLogin["user"].(map[string]any)["id"] {
		t.Fatalf("accepted membership belongs to the wrong user: %#v", acceptedMember)
	}
	membersAfter := owner.requestList(http.MethodGet, "/api/workspaces/"+workspace["id"].(string)+"/members", http.StatusOK)
	if len(membersAfter) != 2 {
		t.Fatalf("accepted invitation did not add a member: %#v", membersAfter)
	}

	me := invitee.request(http.MethodGet, "/api/me", nil, http.StatusOK)
	if me["onboarded_at"] == nil || me["onboarded_at"] == "" {
		t.Fatalf("accepting the first invitation must complete onboarding: %#v", me)
	}
}

func TestSQLiteLocalInvitationDeclineAndRevoke(t *testing.T) {
	app, err := Open(filepath.Join(t.TempDir(), "multica.db"), Options{VerificationCode: "888888"})
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
		"name": "Invitation Decisions",
		"slug": "invitation-decisions",
	}, http.StatusCreated)

	declined := owner.request(http.MethodPost, "/api/workspaces/"+workspace["id"].(string)+"/members", map[string]any{
		"email": "decline@example.com",
		"role":  "member",
	}, http.StatusCreated)
	decliningUser := &testClient{t: t, app: app}
	decliningLogin := decliningUser.request(http.MethodPost, "/auth/verify-code", map[string]any{
		"email": "decline@example.com",
		"code":  "888888",
	}, http.StatusOK)
	decliningUser.token = decliningLogin["token"].(string)
	decliningUser.request(http.MethodPost, "/api/invitations/"+declined["id"].(string)+"/decline", nil, http.StatusNoContent)
	if pending := decliningUser.requestList(http.MethodGet, "/api/invitations", http.StatusOK); len(pending) != 0 {
		t.Fatalf("declined invitation is still pending: %#v", pending)
	}

	revoked := owner.request(http.MethodPost, "/api/workspaces/"+workspace["id"].(string)+"/members", map[string]any{
		"email": "revoke@example.com",
		"role":  "admin",
	}, http.StatusCreated)
	owner.request(http.MethodDelete, "/api/workspaces/"+workspace["id"].(string)+"/invitations/"+revoked["id"].(string), nil, http.StatusNoContent)
	if pending := owner.requestList(http.MethodGet, "/api/workspaces/"+workspace["id"].(string)+"/invitations", http.StatusOK); len(pending) != 0 {
		t.Fatalf("revoked invitation is still pending: %#v", pending)
	}
}

func TestSQLiteLocalPermissionManagementIsAdminOnly(t *testing.T) {
	app, err := Open(filepath.Join(t.TempDir(), "multica.db"), Options{VerificationCode: "888888"})
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
		"name": "Permission Team",
		"slug": "permission-team",
	}, http.StatusCreated)
	owner.slug = workspace["slug"].(string)

	catalog := owner.request(http.MethodGet, "/api/workspaces/"+workspace["id"].(string)+"/permissions", nil, http.StatusOK)
	if len(catalog["roles"].([]any)) != 3 {
		t.Fatalf("permission catalog must describe all fixed roles: %#v", catalog)
	}
	if len(catalog["capabilities"].([]any)) == 0 {
		t.Fatalf("permission catalog must include capabilities: %#v", catalog)
	}

	invitation := owner.request(http.MethodPost, "/api/workspaces/"+workspace["id"].(string)+"/members", map[string]any{
		"email": "member@example.com",
		"role":  "member",
	}, http.StatusCreated)
	member := &testClient{t: t, app: app, slug: owner.slug}
	memberLogin := member.request(http.MethodPost, "/auth/verify-code", map[string]any{
		"email": "member@example.com",
		"code":  "888888",
	}, http.StatusOK)
	member.token = memberLogin["token"].(string)
	member.request(http.MethodPost, "/api/invitations/"+invitation["id"].(string)+"/accept", nil, http.StatusOK)
	member.request(http.MethodGet, "/api/workspaces/"+workspace["id"].(string)+"/permissions", nil, http.StatusForbidden)
	member.request(http.MethodGet, "/api/workspaces/"+workspace["id"].(string)+"/invitations", nil, http.StatusForbidden)
}

func TestSQLiteLocalOwnerManagementMatchesWorkspaceInvariant(t *testing.T) {
	app, err := Open(filepath.Join(t.TempDir(), "multica.db"), Options{VerificationCode: "888888"})
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
		"name": "Owner Invariant",
		"slug": "owner-invariant",
	}, http.StatusCreated)
	owner.slug = workspace["slug"].(string)

	invitation := owner.request(http.MethodPost, "/api/workspaces/"+workspace["id"].(string)+"/members", map[string]any{
		"email": "admin@example.com",
		"role":  "admin",
	}, http.StatusCreated)
	admin := &testClient{t: t, app: app, slug: owner.slug}
	adminLogin := admin.request(http.MethodPost, "/auth/verify-code", map[string]any{
		"email": "admin@example.com",
		"code":  "888888",
	}, http.StatusOK)
	admin.token = adminLogin["token"].(string)
	accepted := admin.request(http.MethodPost, "/api/invitations/"+invitation["id"].(string)+"/accept", nil, http.StatusOK)

	members := owner.requestList(http.MethodGet, "/api/workspaces/"+workspace["id"].(string)+"/members", http.StatusOK)
	ownerMemberID := ""
	for _, member := range members {
		if member["role"] == "owner" {
			ownerMemberID = member["id"].(string)
		}
	}
	if ownerMemberID == "" {
		t.Fatalf("owner membership not found: %#v", members)
	}
	admin.request(http.MethodDelete, "/api/workspaces/"+workspace["id"].(string)+"/members/"+ownerMemberID, nil, http.StatusForbidden)

	promoted := owner.request(http.MethodPatch, "/api/workspaces/"+workspace["id"].(string)+"/members/"+accepted["id"].(string), map[string]any{
		"role": "owner",
	}, http.StatusOK)
	if promoted["role"] != "owner" {
		t.Fatalf("member was not promoted to owner: %#v", promoted)
	}

	owner.request(http.MethodPost, "/api/workspaces/"+workspace["id"].(string)+"/leave", nil, http.StatusNoContent)
	admin.request(http.MethodPost, "/api/workspaces/"+workspace["id"].(string)+"/leave", nil, http.StatusBadRequest)
}

func TestSQLiteLocalProjectAndSkillPermissionsMatchCatalog(t *testing.T) {
	app, err := Open(filepath.Join(t.TempDir(), "multica.db"), Options{VerificationCode: "888888"})
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
		"name": "Permission parity",
		"slug": "permission-parity",
	}, http.StatusCreated)
	owner.slug = workspace["slug"].(string)

	project := owner.request(http.MethodPost, "/api/projects", map[string]any{
		"title": "Owner project",
	}, http.StatusCreated)
	ownerSkill := owner.request(http.MethodPost, "/api/skills", map[string]any{
		"name":        "Owner skill",
		"description": "Created by the owner",
	}, http.StatusCreated)

	invitation := owner.request(http.MethodPost, "/api/workspaces/"+workspace["id"].(string)+"/members", map[string]any{
		"email": "member@example.com",
		"role":  "member",
	}, http.StatusCreated)
	member := &testClient{t: t, app: app, slug: owner.slug}
	memberLogin := member.request(http.MethodPost, "/auth/verify-code", map[string]any{
		"email": "member@example.com",
		"code":  "888888",
	}, http.StatusOK)
	member.token = memberLogin["token"].(string)
	member.request(http.MethodPost, "/api/invitations/"+invitation["id"].(string)+"/accept", nil, http.StatusOK)

	member.request(http.MethodDelete, "/api/projects/"+project["id"].(string), nil, http.StatusForbidden)
	member.request(http.MethodPut, "/api/skills/"+ownerSkill["id"].(string), map[string]any{
		"name":        "Changed by member",
		"description": "Must be rejected",
	}, http.StatusForbidden)
	member.request(http.MethodDelete, "/api/skills/"+ownerSkill["id"].(string), nil, http.StatusForbidden)

	memberSkill := member.request(http.MethodPost, "/api/skills", map[string]any{
		"name":        "Member skill",
		"description": "Created by the member",
	}, http.StatusCreated)
	member.request(http.MethodPut, "/api/skills/"+memberSkill["id"].(string), map[string]any{
		"name":        "Updated member skill",
		"description": "Members can maintain their own skills",
	}, http.StatusOK)
	member.request(http.MethodDelete, "/api/skills/"+memberSkill["id"].(string), nil, http.StatusNoContent)
}
