package main

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/hvritual/workspace/internal/modules/auth"
	"github.com/hvritual/workspace/internal/modules/workspace"
	_ "modernc.org/sqlite"
)

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
