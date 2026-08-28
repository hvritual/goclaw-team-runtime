package engineering

import (
	"context"
	"database/sql"

	persistence "github.com/hvritual/workspace/internal/modules/engineering/internal/infrastructure/sqlite"
)

func MigrateSqlite(ctx context.Context, db *sql.DB) error {
	return persistence.Migrate(ctx, db)
}

func RollbackSqliteLatest(ctx context.Context, db *sql.DB) error {
	return persistence.RollbackLatest(ctx, db)
}
