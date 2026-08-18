package system

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
)

//go:embed internal/infrastructure/sqlite/migrations/*.sql
var sqliteMigrationFiles embed.FS

func MigrateSqlite(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return errors.New("system sqlite database is required")
	}
	paths, err := fs.Glob(sqliteMigrationFiles, "internal/infrastructure/sqlite/migrations/*.up.sql")
	if err != nil {
		return fmt.Errorf("list System SQLite migrations: %w", err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin System SQLite migration: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS system_schema_migrations(version TEXT PRIMARY KEY)`); err != nil {
		return fmt.Errorf("create System SQLite migration catalog: %w", err)
	}
	for _, path := range paths {
		var applied string
		if err := tx.QueryRowContext(ctx, `SELECT version FROM system_schema_migrations WHERE version=?`, path).Scan(&applied); err == nil {
			continue
		} else if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("check System SQLite migration %s: %w", path, err)
		}
		body, err := fs.ReadFile(sqliteMigrationFiles, path)
		if err != nil {
			return fmt.Errorf("read System SQLite migration %s: %w", path, err)
		}
		if _, err := tx.ExecContext(ctx, string(body)); err != nil {
			return fmt.Errorf("apply System SQLite migration %s: %w", path, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO system_schema_migrations(version) VALUES(?)`, path); err != nil {
			return fmt.Errorf("record System SQLite migration %s: %w", path, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit System SQLite migrations: %w", err)
	}
	return nil
}
