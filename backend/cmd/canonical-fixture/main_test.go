package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/auth"
	"github.com/hvritual/workspace/internal/modules/workspace"
	_ "modernc.org/sqlite"
)

func TestCanonicalFixtureProjectsOnboardedUserAfterDatabaseRestart(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "canonical.db")
	db, err := sql.Open("sqlite", databasePath+"?_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatal(err)
	}
	if err := workspace.MigrateSqlite(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if err := auth.MigrateSqlite(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if result, err := seedFixture(context.Background(), db); err != nil || result != "created" {
		t.Fatalf("seed = %q %v", result, err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	db, err = sql.Open("sqlite", databasePath+"?_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	module, err := auth.NewWithSqliteLocalAuth(auth.SqlitePersistenceConfig{DB: db}, auth.LocalAuthConfig{
		VerificationCode: "888888", SessionTTL: time.Hour,
		NewID: func(context.Context) (string, error) { return "fixture-session", nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	server := kratoshttp.NewServer()
	module.RegisterHTTP(server)
	verifyRequest := httptest.NewRequest(http.MethodPost, "/auth/verify-code", strings.NewReader(`{"email":"`+fixtureEmail+`","code":"888888"}`))
	verify := httptest.NewRecorder()
	server.ServeHTTP(verify, verifyRequest)
	if verify.Code != http.StatusOK {
		t.Fatalf("verify = %d %s", verify.Code, verify.Body.String())
	}
	var login struct {
		Token string `json:"token"`
		User  struct {
			ID          string  `json:"id"`
			OnboardedAt *string `json:"onboarded_at"`
		} `json:"user"`
	}
	if err := json.Unmarshal(verify.Body.Bytes(), &login); err != nil {
		t.Fatal(err)
	}
	if login.Token != "fixture-session" || login.User.ID != fixtureUserID || login.User.OnboardedAt == nil || *login.User.OnboardedAt == "" {
		t.Fatalf("login = %#v", login)
	}
	meRequest := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	meRequest.Header.Set("Authorization", "Bearer fixture-session")
	me := httptest.NewRecorder()
	server.ServeHTTP(me, meRequest)
	if me.Code != http.StatusOK || !strings.Contains(me.Body.String(), `"id":"`+fixtureUserID+`"`) || !strings.Contains(me.Body.String(), `"onboarded_at":"`+*login.User.OnboardedAt+`"`) {
		t.Fatalf("me = %d %s", me.Code, me.Body.String())
	}
}

func TestSeedFixtureIsExplicitIdempotentAndNonOverwriting(t *testing.T) {
	db := fixtureDatabase(t)
	result, err := seedFixture(context.Background(), db)
	if err != nil || result != "created" {
		t.Fatalf("first seed=%q err=%v", result, err)
	}
	if _, err := db.Exec(`UPDATE workspace_issues SET metadata='{"browser_readback":"retained"}' WHERE id=?`, fixtureIssueID); err != nil {
		t.Fatal(err)
	}
	result, err = seedFixture(context.Background(), db)
	if err != nil || result != "already_present" {
		t.Fatalf("second seed=%q err=%v", result, err)
	}
	var metadata string
	if err := db.QueryRow(`SELECT metadata FROM workspace_issues WHERE id=?`, fixtureIssueID).Scan(&metadata); err != nil || metadata != `{"browser_readback":"retained"}` {
		t.Fatalf("metadata=%q err=%v", metadata, err)
	}
	var onboardedAt string
	if err := db.QueryRow(`SELECT onboarded_at FROM auth_users WHERE id=?`, fixtureUserID).Scan(&onboardedAt); err != nil || onboardedAt == "" {
		t.Fatalf("onboarded_at=%q err=%v", onboardedAt, err)
	}
}

func TestSeedFixtureRejectsPartialOrConflictingFootprintWithoutWrites(t *testing.T) {
	db := fixtureDatabase(t)
	if _, err := db.Exec(`INSERT INTO auth_users(id,name,email,created_at,updated_at) VALUES('conflict','Conflict',?,'2026-08-13T00:00:00Z','2026-08-13T00:00:00Z')`, fixtureEmail); err != nil {
		t.Fatal(err)
	}
	if _, err := seedFixture(context.Background(), db); err == nil {
		t.Fatal("conflicting seed unexpectedly succeeded")
	}
	for _, table := range []string{"workspaces", "auth_members", "workspace_issues"} {
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil || count != 0 {
			t.Fatalf("%s count=%d err=%v", table, count, err)
		}
	}
}

func TestSeedFixtureRejectsUnmigratedAndSkewedCompleteCount(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "unmigrated.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := seedFixture(context.Background(), db); err == nil {
		t.Fatal("unmigrated database unexpectedly accepted")
	}

	migrated := fixtureDatabase(t)
	if _, err := migrated.Exec(`INSERT INTO auth_users(id,name,email,created_at,updated_at) VALUES(?,?,?,?,?)`, fixtureUserID, "Wrong", "wrong@local", "2026-08-13T00:00:00Z", "2026-08-13T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	if _, err := migrated.Exec(`INSERT INTO workspaces(id,name,slug,settings,repos,issue_prefix,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?)`, fixtureWorkspaceID, "Wrong", "wrong", `{}`, `[]`, "BAD", "2026-08-13T00:00:00Z", "2026-08-13T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	if _, err := migrated.Exec(`INSERT INTO auth_members(id,workspace_id,user_id,role,created_at) VALUES(?,?,?,?,?)`, fixtureMemberID, fixtureWorkspaceID, fixtureUserID, "owner", "2026-08-13T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	if _, err := migrated.Exec(`INSERT INTO auth_workspace_membership_roots(workspace_id,user_id,member_id,created_at) VALUES(?,?,?,?)`, fixtureWorkspaceID, fixtureUserID, fixtureMemberID, "2026-08-13T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	if _, err := migrated.Exec(`INSERT INTO workspace_issues(id,workspace_id,number,identifier,title,status,priority,creator_type,creator_id,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, fixtureIssueID, fixtureWorkspaceID, 1, "BAD-1", "Wrong", "todo", "none", "member", fixtureMemberID, "2026-08-13T00:00:00Z", "2026-08-13T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	if _, err := seedFixture(context.Background(), migrated); err == nil {
		t.Fatal("skewed five-row footprint unexpectedly accepted")
	}
}

func fixtureDatabase(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "fixture.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := workspace.MigrateSqlite(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if err := auth.MigrateSqlite(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	return db
}
