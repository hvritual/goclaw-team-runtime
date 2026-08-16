package workspace

import (
	"context"
	"database/sql"
	"errors"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	_ "modernc.org/sqlite"
)

func TestWorkspaceGovernanceMigrationUpgradesRetainedVersionEightDatabase(t *testing.T) {
	db := openUnmigratedWorkspaceDB(t, "governance-upgrade")
	applyWorkspaceMigrationsBeforeGovernance(t, db)
	if _, err := db.Exec(`INSERT INTO workspaces(id,name,slug,issue_prefix,created_at,updated_at) VALUES('workspace-1','Acme','acme','ACM','2026-08-16T00:00:00Z','2026-08-16T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if err := MigrateSqlite(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	var migrationCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM workspace_schema_migrations`).Scan(&migrationCount); err != nil || migrationCount != 9 {
		t.Fatalf("migration count = %d, %v", migrationCount, err)
	}
	var workspaceName string
	if err := db.QueryRow(`SELECT name FROM workspaces WHERE id='workspace-1'`).Scan(&workspaceName); err != nil || workspaceName != "Acme" {
		t.Fatalf("retained workspace = %q, %v", workspaceName, err)
	}
}

func TestWorkspaceGovernanceMigrationRollsBackPartialVersionNineFailure(t *testing.T) {
	db := openUnmigratedWorkspaceDB(t, "governance-rollback")
	applyWorkspaceMigrationsBeforeGovernance(t, db)
	if _, err := db.Exec(`CREATE VIEW workspace_audit_entries AS SELECT 'blocked' AS id`); err != nil {
		t.Fatal(err)
	}
	if err := MigrateSqlite(context.Background(), db); err == nil {
		t.Fatal("MigrateSqlite() error = nil")
	}
	for _, table := range []string{"workspace_resource_revisions", "workspace_mutation_idempotency", "workspace_outbox_events"} {
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&count); err != nil || count != 0 {
			t.Fatalf("rolled-back table %s count = %d, %v", table, count, err)
		}
	}
	var migrationCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM workspace_schema_migrations`).Scan(&migrationCount); err != nil || migrationCount != 8 {
		t.Fatalf("migration count after rollback = %d, %v", migrationCount, err)
	}
}

func TestWorkspaceGovernanceRowsPersistAcrossRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "governance-restart.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if err := MigrateSqlite(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO workspace_resource_revisions(workspace_id,resource_kind,resource_id,revision,updated_at) VALUES('workspace-1','task','task-1',3,'2026-08-16T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	restarted, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = restarted.Close() })
	if err := MigrateSqlite(context.Background(), restarted); err != nil {
		t.Fatal(err)
	}
	var revision int64
	if err := restarted.QueryRow(`SELECT revision FROM workspace_resource_revisions WHERE workspace_id='workspace-1' AND resource_kind='task' AND resource_id='task-1'`).Scan(&revision); err != nil || revision != 3 {
		t.Fatalf("retained revision = %d, %v", revision, err)
	}
}

func openUnmigratedWorkspaceDB(t *testing.T, name string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+name+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func applyWorkspaceMigrationsBeforeGovernance(t *testing.T, db *sql.DB) {
	t.Helper()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`CREATE TABLE workspace_schema_migrations(version TEXT PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	paths, err := fs.Glob(sqliteMigrationFiles, SqliteMigrationDir()+"/*.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, migrationPath := range paths {
		if strings.Contains(migrationPath, "000009_workspace_governance") {
			continue
		}
		migration, err := sqliteMigrationFiles.ReadFile(migrationPath)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(string(migration)); err != nil {
			t.Fatalf("apply %s: %v", migrationPath, err)
		}
		version := migrationPath[len(SqliteMigrationDir())+1:]
		if _, err := tx.Exec(`INSERT INTO workspace_schema_migrations(version) VALUES(?)`, version); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
}

func TestSqliteWorkspaceIdentityReaderFindsOnlyRequestedWorkspace(t *testing.T) {
	db, err := sql.Open("sqlite", "file:workspace-identity?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()
	if err := MigrateSqlite(ctx, db); err != nil {
		t.Fatal(err)
	}
	for _, workspace := range []struct {
		id   string
		name string
		slug string
	}{
		{id: "workspace-1", name: "Acme", slug: "acme"},
		{id: "workspace-2", name: "Globex", slug: "globex"},
	} {
		_, err = db.ExecContext(ctx, `INSERT INTO workspaces(
			id, name, slug, issue_prefix, created_at, updated_at
		) VALUES (?, ?, ?, 'WSP', '2026-08-03T00:00:00Z', '2026-08-03T00:00:00Z')`,
			workspace.id, workspace.name, workspace.slug)
		if err != nil {
			t.Fatal(err)
		}
	}

	module, err := NewWithSqlitePersistence(SqlitePersistenceConfig{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	message, err := module.Local().Ping(ctx)
	if err != nil || message != "pong" {
		t.Fatalf("persistent module Ping() = %q, %v", message, err)
	}
	identity, err := module.IdentityLocal().FindIdentity(ctx, "workspace-1")
	if err != nil {
		t.Fatal(err)
	}
	if identity != (contract.WorkspaceIdentity{ID: "workspace-1", Name: "Acme"}) {
		t.Fatalf("unexpected workspace identity: %+v", identity)
	}
	if _, err := module.IdentityLocal().FindIdentity(ctx, "missing"); !errors.Is(err, contract.ErrWorkspaceNotFound) {
		t.Fatalf("missing workspace error = %v", err)
	}
}

func TestWorkspaceGovernanceMigrationUsesOnlyCompositePrimaryKeyAccessPaths(t *testing.T) {
	contents, err := sqliteMigrationFiles.ReadFile(SqliteMigrationDir() + "/000009_workspace_governance.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sqlText := strings.ToUpper(string(contents))
	if got := strings.Count(sqlText, "WITHOUT ROWID"); got != 4 {
		t.Fatalf("WITHOUT ROWID count = %d, want 4", got)
	}
	for _, forbidden := range []string{
		"FOREIGN KEY", "REFERENCES", "CASCADE", "TRIGGER",
		"CREATE INDEX", "CREATE UNIQUE INDEX",
	} {
		if strings.Contains(sqlText, forbidden) {
			t.Errorf("governance migration contains forbidden DDL %q", forbidden)
		}
	}
}

func TestNewWithSqlitePersistenceRejectsMissingWorkspaceDatabase(t *testing.T) {
	if _, err := NewWithSqlitePersistence(SqlitePersistenceConfig{}); err == nil {
		t.Fatal("NewWithSqlitePersistence() error = nil")
	}
}
