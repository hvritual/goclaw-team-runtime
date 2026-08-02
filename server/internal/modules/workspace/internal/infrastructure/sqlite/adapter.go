// Package sqlite contains Sqlite adapters for the workspace module.
// Implement domain or application ports here and keep driver types inside this package.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/multica-ai/multica/server/internal/modules/workspace/contract"
	"github.com/multica-ai/multica/server/internal/modules/workspace/internal/application"
	workspaceDomain "github.com/multica-ai/multica/server/internal/modules/workspace/internal/domain/workspace"
)

// Config is the provider-owned composition input. Add the native connection
// and provider settings here; never pass them into domain or application APIs.
type Config struct {
	DB *sql.DB
}

func New(Config) contract.Service { return application.New() }

func NewIdentity(config Config) (contract.WorkspaceIdentityReader, error) {
	if config.DB == nil {
		return nil, errors.New("workspace sqlite database is required")
	}
	return application.NewWorkspaceIdentityService(&workspaceIdentityRepository{db: config.DB}), nil
}

type workspaceIdentityRepository struct {
	db *sql.DB
}

func (r *workspaceIdentityRepository) FindByID(ctx context.Context, workspaceID string) (workspaceDomain.Workspace, error) {
	var id, name string
	if err := r.db.QueryRowContext(ctx,
		`SELECT id, name FROM workspaces WHERE id = ?`, workspaceID,
	).Scan(&id, &name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return workspaceDomain.Workspace{}, application.ErrWorkspaceNotFound
		}
		return workspaceDomain.Workspace{}, fmt.Errorf("find workspace identity: %w", err)
	}
	return *workspaceDomain.New(id, name), nil
}
