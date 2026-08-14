package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	authcontract "github.com/hvritual/workspace/internal/modules/auth/contract"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
)

type WorkspaceCreationRepository struct {
	db          *sql.DB
	ownerWriter authcontract.SQLiteWorkspaceOwnerWriter
}

func NewWorkspaceCreationRepository(config Config, ownerWriter authcontract.SQLiteWorkspaceOwnerWriter) (*WorkspaceCreationRepository, error) {
	if config.DB == nil {
		return nil, errors.New("workspace sqlite database is required")
	}
	if ownerWriter == nil {
		return nil, errors.New("workspace owner writer is required")
	}
	return &WorkspaceCreationRepository{db: config.DB, ownerWriter: ownerWriter}, nil
}

func (r *WorkspaceCreationRepository) Create(ctx context.Context, value application.WorkspaceCreation) (contract.WorkspaceSelection, error) {
	connection, err := r.db.Conn(ctx)
	if err != nil {
		return contract.WorkspaceSelection{}, fmt.Errorf("acquire workspace creation connection: %w", err)
	}
	defer connection.Close()
	if _, err := connection.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return contract.WorkspaceSelection{}, fmt.Errorf("begin workspace creation transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = connection.ExecContext(context.Background(), "ROLLBACK")
		}
	}()
	timestamp := value.CreatedAt.Format(time.RFC3339Nano)
	if _, err := connection.ExecContext(ctx, `INSERT INTO workspaces(
		id,name,slug,description,context,settings,repos,issue_prefix,avatar_url,created_at,updated_at
	) VALUES(?,?,?,?,?,'{}','[]',?,NULL,?,?)`, value.WorkspaceID, value.Name, value.Slug, value.Description, value.Context, value.IssuePrefix, timestamp, timestamp); err != nil {
		if isWorkspaceSlugConflict(err) {
			return contract.WorkspaceSelection{}, application.ErrWorkspaceSlugConflict
		}
		return contract.WorkspaceSelection{}, fmt.Errorf("create workspace: %w", err)
	}
	if err := r.ownerWriter.CreateWorkspaceOwner(ctx, connection, value.WorkspaceID, value.UserID, value.MemberID, value.CreatedAt); err != nil {
		return contract.WorkspaceSelection{}, err
	}
	if _, err := connection.ExecContext(ctx, "COMMIT"); err != nil {
		return contract.WorkspaceSelection{}, fmt.Errorf("commit workspace creation transaction: %w", err)
	}
	committed = true
	return contract.WorkspaceSelection{
		ID: value.WorkspaceID, Name: value.Name, Slug: value.Slug,
		Description: value.Description, Context: value.Context, Settings: map[string]any{}, Repos: []contract.WorkspaceRepo{},
		IssuePrefix: value.IssuePrefix, CreatedAt: timestamp, UpdatedAt: timestamp,
	}, nil
}

func isWorkspaceSlugConflict(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed: workspaces.slug")
}

var _ application.WorkspaceCreationRepository = (*WorkspaceCreationRepository)(nil)
