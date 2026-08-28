package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

type EngineeringWorkReferenceReader struct {
	db *sql.DB
}

func NewEngineeringWorkReferenceReader(db *sql.DB) (*EngineeringWorkReferenceReader, error) {
	if db == nil {
		return nil, contract.ErrEngineeringWorkLinkUnavailable
	}
	return &EngineeringWorkReferenceReader{db: db}, nil
}

func (r *EngineeringWorkReferenceReader) WorkExists(ctx context.Context, workspaceID string, kind contract.EngineeringWorkKind, workID string) (bool, error) {
	var query string
	switch kind {
	case contract.EngineeringWorkProject:
		query = `SELECT EXISTS(SELECT 1 FROM workspace_projects WHERE workspace_id=? AND id=? LIMIT 1)`
	case contract.EngineeringWorkRequirement:
		query = `SELECT EXISTS(SELECT 1 FROM workspace_requirements WHERE workspace_id=? AND id=? LIMIT 1)`
	case contract.EngineeringWorkTask:
		query = `SELECT EXISTS(SELECT 1 FROM workspace_todos WHERE workspace_id=? AND id=? AND status <> 'archived' LIMIT 1)`
	default:
		return false, contract.ErrEngineeringWorkLinkInvalid
	}
	var exists bool
	if err := r.db.QueryRowContext(ctx, query, workspaceID, workID).Scan(&exists); err != nil {
		return false, fmt.Errorf("read Workspace engineering work reference: %w", err)
	}
	return exists, nil
}
