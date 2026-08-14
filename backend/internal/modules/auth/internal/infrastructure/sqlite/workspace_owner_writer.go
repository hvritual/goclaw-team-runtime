package sqlite

import (
	"context"
	"fmt"
	"time"

	"github.com/hvritual/workspace/internal/modules/auth/contract"
)

type WorkspaceOwnerWriter struct{}

func NewWorkspaceOwnerWriter() *WorkspaceOwnerWriter { return &WorkspaceOwnerWriter{} }

func (*WorkspaceOwnerWriter) CreateWorkspaceOwner(ctx context.Context, executor contract.SQLiteExecutor, workspaceID, userID, memberID string, createdAt time.Time) error {
	timestamp := createdAt.UTC().Format(time.RFC3339Nano)
	if _, err := executor.ExecContext(ctx, `INSERT INTO auth_workspace_membership_roots(
		workspace_id,user_id,member_id,created_at
	) VALUES(?,?,?,?)`, workspaceID, userID, memberID, timestamp); err != nil {
		return fmt.Errorf("create workspace membership root: %w", err)
	}
	if _, err := executor.ExecContext(ctx, `INSERT INTO auth_members(
		id,workspace_id,user_id,role,created_at
	) VALUES(?,?,?,'owner',?)`, memberID, workspaceID, userID, timestamp); err != nil {
		return fmt.Errorf("create workspace owner: %w", err)
	}
	return nil
}

var _ contract.SQLiteWorkspaceOwnerWriter = (*WorkspaceOwnerWriter)(nil)
