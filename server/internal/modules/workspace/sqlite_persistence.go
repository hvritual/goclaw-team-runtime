package workspace

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"

	persistence "github.com/multica-ai/multica/server/internal/modules/workspace/internal/infrastructure/sqlite"
	"github.com/multica-ai/multica/server/internal/modules/workspace/internal/interfaces/local"
	protoadapter "github.com/multica-ai/multica/server/internal/modules/workspace/internal/interfaces/proto"
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
	migrationPath := SqliteMigrationDir() + "/000001_workspace.up.sql"
	migration, err := fs.ReadFile(sqliteMigrationFiles, migrationPath)
	if err != nil {
		return fmt.Errorf("read Workspace SQLite migration: %w", err)
	}
	if _, err := db.ExecContext(ctx, string(migration)); err != nil {
		return fmt.Errorf("apply Workspace SQLite migration: %w", err)
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
