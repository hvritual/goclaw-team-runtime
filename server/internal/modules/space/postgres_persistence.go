package space

import (
	"embed"
	"io/fs"

	persistence "github.com/multica-ai/multica/server/internal/modules/space/internal/infrastructure/postgres"
	"github.com/multica-ai/multica/server/internal/modules/space/internal/interfaces/local"
	protoadapter "github.com/multica-ai/multica/server/internal/modules/space/internal/interfaces/proto"
)

//go:embed internal/infrastructure/postgres/migrations/*.sql
var postgresMigrationFiles embed.FS

// PostgresMigrationFS exposes only the explicitly selected provider's migrations.
func PostgresMigrationFS() fs.FS { return postgresMigrationFiles }

func PostgresMigrationDir() string {
	return "internal/infrastructure/postgres/migrations"
}

type PostgresPersistenceConfig = persistence.Config

func NewWithPostgresPersistence(config PostgresPersistenceConfig) *Module {
	client := local.New(persistence.New(config))
	module := New()
	module.local = client
	module.server = protoadapter.New(client)
	return module
}
