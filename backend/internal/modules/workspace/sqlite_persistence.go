package workspace

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"

	persistence "github.com/hvritual/workspace/internal/modules/workspace/internal/infrastructure/sqlite"
	"github.com/hvritual/workspace/internal/modules/workspace/internal/interfaces/local"
	protoadapter "github.com/hvritual/workspace/internal/modules/workspace/internal/interfaces/proto"
)

//go:embed internal/infrastructure/sqlite/migrations/*.sql
var sqliteMigrationFiles embed.FS

// SqliteMigrationFS exposes only the explicitly selected provider's migrations.
func SqliteMigrationFS() fs.FS { return sqliteMigrationFiles }

func SqliteMigrationDir() string {
	return "internal/infrastructure/sqlite/migrations"
}

func MigrateSqlite(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("workspace sqlite database is required")
	}
	migrationPaths, err := fs.Glob(sqliteMigrationFiles, SqliteMigrationDir()+"/*.up.sql")
	if err != nil {
		return fmt.Errorf("list Workspace SQLite migrations: %w", err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin Workspace SQLite migration: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS workspace_schema_migrations (
		version TEXT PRIMARY KEY
	)`); err != nil {
		return fmt.Errorf("create Workspace SQLite migration catalog: %w", err)
	}
	for _, migrationPath := range migrationPaths {
		version := migrationPath[len(SqliteMigrationDir())+1:]
		var applied string
		err := tx.QueryRowContext(ctx, `SELECT version FROM workspace_schema_migrations WHERE version = ?`, version).Scan(&applied)
		if err == nil {
			continue
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("check Workspace SQLite migration %s: %w", version, err)
		}
		migration, err := fs.ReadFile(sqliteMigrationFiles, migrationPath)
		if err != nil {
			return fmt.Errorf("read Workspace SQLite migration %s: %w", version, err)
		}
		if _, err := tx.ExecContext(ctx, string(migration)); err != nil {
			return fmt.Errorf("apply Workspace SQLite migration %s: %w", version, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO workspace_schema_migrations(version) VALUES (?)`, version); err != nil {
			return fmt.Errorf("record Workspace SQLite migration %s: %w", version, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit Workspace SQLite migrations: %w", err)
	}
	return nil
}

type SqlitePersistenceConfig = persistence.Config

func NewWithSqlitePersistence(config SqlitePersistenceConfig) (*Module, error) {
	identity, err := persistence.NewIdentity(config)
	if err != nil {
		return nil, fmt.Errorf("configure Workspace identity SQLite persistence: %w", err)
	}
	client := local.New(persistence.New(config))
	module := New()
	module.local = client
	module.identity = identity
	module.server = protoadapter.New(client)
	return module, nil
}
