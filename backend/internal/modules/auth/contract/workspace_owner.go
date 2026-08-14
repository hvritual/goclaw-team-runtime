package contract

import (
	"context"
	"database/sql"
	"time"
)

type SQLiteExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

type SQLiteWorkspaceOwnerWriter interface {
	CreateWorkspaceOwner(context.Context, SQLiteExecutor, string, string, string, time.Time) error
}
