package controlplane

import (
	"context"
	"database/sql"
	"time"
)

type Repository interface {
	CreateWorkspace(context.Context, Workspace, Member, AuditEntry) error
	UpdateWorkspace(context.Context, Workspace, int64, AuditEntry) error
	GetWorkspace(context.Context, string) (Workspace, error)
	GetMember(context.Context, string, string) (Member, error)
	ListMembers(context.Context, string, bool) ([]Member, error)
	SaveMember(context.Context, Member, int64, AuditEntry) error
	SaveRecord(context.Context, Record, int64, AuditEntry) (Record, error)
	GetRecord(context.Context, string, string, string) (Record, error)
	ListRecords(context.Context, string, string, Page) ([]Record, error)
	ListAudit(context.Context, string, Page) ([]AuditEntry, error)
	Close() error
}

type Dialect string

const (
	DialectSQLite   Dialect = "sqlite"
	DialectPostgres Dialect = "postgres"
)

type Clock func() time.Time

// NewPostgresRepository accepts an already configured production connection.
// The caller owns driver selection, credentials, pooling, and TLS policy.
func NewPostgresRepository(ctx context.Context, db *sql.DB) (Repository, error) {
	return newSQLRepository(ctx, db, DialectPostgres)
}
