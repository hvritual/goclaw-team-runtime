package auth

import (
	"database/sql"

	"github.com/hvritual/workspace/internal/modules/auth/contract"
	persistence "github.com/hvritual/workspace/internal/modules/auth/internal/infrastructure/sqlite"
)

// NewSQLiteWorkspaceMembershipReader exposes the Auth-owned read projection to
// composition roots without exposing Auth persistence internals to consumers.
func NewSQLiteWorkspaceMembershipReader(db *sql.DB) (contract.WorkspaceMembershipReader, error) {
	return persistence.NewWorkspaceMembershipStore(db)
}
