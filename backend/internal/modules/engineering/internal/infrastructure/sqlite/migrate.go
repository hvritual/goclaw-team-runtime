package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

const migrationDir = "migrations"

func Migrate(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return ErrDatabaseRequired
	}
	if err := configure(ctx, db); err != nil {
		return err
	}
	paths, err := fs.Glob(migrationFiles, migrationDir+"/*.up.sql")
	if err != nil {
		return fmt.Errorf("list engineering sqlite migrations: %w", err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin engineering sqlite migration: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS engineering_schema_migrations (
		version TEXT PRIMARY KEY
	)`); err != nil {
		return fmt.Errorf("create engineering sqlite migration catalog: %w", err)
	}
	for _, migrationPath := range paths {
		version := path.Base(migrationPath)
		var applied string
		err := tx.QueryRowContext(ctx, `SELECT version FROM engineering_schema_migrations WHERE version = ?`, version).Scan(&applied)
		if err == nil {
			continue
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("check engineering sqlite migration %s: %w", version, err)
		}
		migration, err := fs.ReadFile(migrationFiles, migrationPath)
		if err != nil {
			return fmt.Errorf("read engineering sqlite migration %s: %w", version, err)
		}
		if _, err := tx.ExecContext(ctx, string(migration)); err != nil {
			return fmt.Errorf("apply engineering sqlite migration %s: %w", version, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO engineering_schema_migrations(version) VALUES (?)`, version); err != nil {
			return fmt.Errorf("record engineering sqlite migration %s: %w", version, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit engineering sqlite migrations: %w", err)
	}
	return nil
}

func RollbackLatest(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return ErrDatabaseRequired
	}
	if err := configure(ctx, db); err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin engineering sqlite rollback: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var version string
	if err := tx.QueryRowContext(ctx, `SELECT version FROM engineering_schema_migrations ORDER BY version DESC LIMIT 1`).Scan(&version); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("select latest engineering sqlite migration: %w", err)
	}
	base := strings.TrimSuffix(version, ".up.sql")
	if base == version {
		return fmt.Errorf("invalid engineering sqlite migration version %q", version)
	}
	downPath := migrationDir + "/" + base + ".down.sql"
	migration, err := fs.ReadFile(migrationFiles, downPath)
	if err != nil {
		return fmt.Errorf("read engineering sqlite rollback %s: %w", version, err)
	}
	if _, err := tx.ExecContext(ctx, string(migration)); err != nil {
		return fmt.Errorf("apply engineering sqlite rollback %s: %w", version, err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM engineering_schema_migrations WHERE version = ?`, version); err != nil {
		return fmt.Errorf("remove engineering sqlite migration %s: %w", version, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit engineering sqlite rollback: %w", err)
	}
	return nil
}

func configure(ctx context.Context, db *sql.DB) error {
	for _, statement := range []string{
		`PRAGMA foreign_keys = OFF`,
		`PRAGMA busy_timeout = 5000`,
		`PRAGMA journal_mode = WAL`,
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("configure engineering sqlite: %w", err)
		}
	}
	return nil
}
