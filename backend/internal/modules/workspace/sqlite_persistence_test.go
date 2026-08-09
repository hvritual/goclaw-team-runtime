package workspace

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	_ "modernc.org/sqlite"
)

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

func TestNewWithSqlitePersistenceRejectsMissingWorkspaceDatabase(t *testing.T) {
	if _, err := NewWithSqlitePersistence(SqlitePersistenceConfig{}); err == nil {
		t.Fatal("NewWithSqlitePersistence() error = nil")
	}
}
