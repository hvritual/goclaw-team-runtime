package auth

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"

	"github.com/hvritual/workspace/internal/modules/auth/contract"
	persistence "github.com/hvritual/workspace/internal/modules/auth/internal/infrastructure/sqlite"
	"github.com/hvritual/workspace/internal/modules/auth/internal/interfaces/local"
	protoadapter "github.com/hvritual/workspace/internal/modules/auth/internal/interfaces/proto"
)

//go:embed internal/infrastructure/sqlite/migrations/*.sql
var sqliteMigrationFiles embed.FS

func SqliteMigrationFS() fs.FS { return sqliteMigrationFiles }

func SqliteMigrationDir() string {
	return "internal/infrastructure/sqlite/migrations"
}

func MigrateSqlite(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return errors.New("auth sqlite database is required")
	}
	migrationPaths, err := fs.Glob(sqliteMigrationFiles, SqliteMigrationDir()+"/*.up.sql")
	if err != nil {
		return fmt.Errorf("list Auth SQLite migrations: %w", err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin Auth SQLite migration: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS auth_schema_migrations (
		version TEXT PRIMARY KEY
	)`); err != nil {
		return fmt.Errorf("create Auth SQLite migration catalog: %w", err)
	}
	for _, migrationPath := range migrationPaths {
		version := migrationPath[len(SqliteMigrationDir())+1:]
		var applied string
		err := tx.QueryRowContext(ctx, `SELECT version FROM auth_schema_migrations WHERE version = ?`, version).Scan(&applied)
		if err == nil {
			continue
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("check Auth SQLite migration %s: %w", version, err)
		}
		migration, err := fs.ReadFile(sqliteMigrationFiles, migrationPath)
		if err != nil {
			return fmt.Errorf("read Auth SQLite migration %s: %w", version, err)
		}
		if _, err := tx.ExecContext(ctx, string(migration)); err != nil {
			return fmt.Errorf("apply Auth SQLite migration %s: %w", version, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO auth_schema_migrations(version) VALUES (?)`, version); err != nil {
			return fmt.Errorf("record Auth SQLite migration %s: %w", version, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit Auth SQLite migrations: %w", err)
	}
	return nil
}

type SqlitePersistenceConfig = persistence.Config

func NewWithSqliteMemberServices(config SqlitePersistenceConfig) (*Module, error) {
	service, err := persistence.NewMember(config)
	if err != nil {
		return nil, fmt.Errorf("configure Auth Member SQLite persistence: %w", err)
	}
	module := New()
	replacement := newMemberExtension(service)
	replaced := false
	for index, extension := range module.extensions {
		if _, ok := extension.(*MemberExtension); ok {
			module.extensions[index] = replacement
			replaced = true
		}
	}
	if !replaced {
		return nil, errors.New("Auth generated Member extension is missing")
	}
	return module, nil
}

func newMemberExtension(service contract.MemberService) *MemberExtension {
	client := local.NewMember(service)
	return &MemberExtension{local: client, server: protoadapter.NewMemberServer(client)}
}
