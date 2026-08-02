package auth

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"

	persistence "github.com/multica-ai/multica/server/internal/modules/auth/internal/infrastructure/sqlite"
	"github.com/multica-ai/multica/server/internal/modules/auth/internal/interfaces/local"
	protoadapter "github.com/multica-ai/multica/server/internal/modules/auth/internal/interfaces/proto"
)

//go:embed internal/infrastructure/sqlite/migrations/*.sql
var sqliteMigrationFiles embed.FS

// SqliteMigrationFS exposes only the explicitly selected provider's migrations.
func SqliteMigrationFS() fs.FS { return sqliteMigrationFiles }

func SqliteMigrationDir() string {
	return "internal/infrastructure/sqlite/migrations"
}

// MigrateSqlite installs the Auth-owned SQLite schema before module assembly.
func MigrateSqlite(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("auth sqlite database is required")
	}
	migrationPath := SqliteMigrationDir() + "/000001_auth.up.sql"
	migration, err := fs.ReadFile(sqliteMigrationFiles, migrationPath)
	if err != nil {
		return fmt.Errorf("read auth sqlite migration: %w", err)
	}
	if _, err := db.ExecContext(ctx, string(migration)); err != nil {
		return fmt.Errorf("apply auth sqlite migration: %w", err)
	}
	return nil
}

type SqlitePersistenceConfig = persistence.Config

func NewWithSqlitePersistence(config SqlitePersistenceConfig) (*Module, error) {
	memberService, err := persistence.NewMember(config)
	if err != nil {
		return nil, fmt.Errorf("configure Auth member SQLite persistence: %w", err)
	}
	client := local.New(persistence.New(config))
	module := New()
	module.local = client
	module.server = protoadapter.New(client)
	memberClient := local.NewMember(memberService)
	for _, extension := range module.extensions {
		if typed, ok := extension.(*MemberExtension); ok {
			typed.local = memberClient
			typed.server = protoadapter.NewMemberServer(protoadapter.NewMemberTransportService(memberClient))
		}
	}
	return module, nil
}
