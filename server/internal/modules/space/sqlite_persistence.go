package space

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"

	"github.com/multica-ai/multica/server/internal/modules/space/contract"
	persistence "github.com/multica-ai/multica/server/internal/modules/space/internal/infrastructure/sqlite"
	"github.com/multica-ai/multica/server/internal/modules/space/internal/interfaces/local"
	protoadapter "github.com/multica-ai/multica/server/internal/modules/space/internal/interfaces/proto"
)

//go:embed internal/infrastructure/sqlite/migrations/*.sql
var sqliteMigrationFiles embed.FS

func SqliteMigrationFS() fs.FS { return sqliteMigrationFiles }

func SqliteMigrationDir() string {
	return "internal/infrastructure/sqlite/migrations"
}

func MigrateSqlite(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("space sqlite database is required")
	}
	migrationPath := SqliteMigrationDir() + "/000001_space.up.sql"
	migration, err := fs.ReadFile(sqliteMigrationFiles, migrationPath)
	if err != nil {
		return fmt.Errorf("read Space SQLite migration: %w", err)
	}
	if _, err := db.ExecContext(ctx, string(migration)); err != nil {
		return fmt.Errorf("apply Space SQLite migration: %w", err)
	}
	return nil
}

type SqlitePersistenceConfig = persistence.Config

func NewWithSqlitePersistence(config SqlitePersistenceConfig) (*Module, error) {
	assetService, err := persistence.NewAsset(config)
	if err != nil {
		return nil, fmt.Errorf("configure Space Asset SQLite persistence: %w", err)
	}
	client := local.New(persistence.New(config))
	module := New()
	module.local = client
	module.server = protoadapter.New(client)
	for _, extension := range module.extensions {
		if typed, ok := extension.(*AssetExtension); ok {
			typed.local = assetService
			typed.server = protoadapter.NewAssetServer(assetService)
		}
	}
	return module, nil
}

// AssetUploads exposes the operational upload lifecycle used by the installed
// multipart adapter while AssetLocal remains the generated RPC contract.
func (m *Module) AssetUploads() contract.AssetUploadService {
	service := m.AssetLocal()
	typed, ok := service.(contract.AssetUploadService)
	if !ok {
		return nil
	}
	return typed
}
