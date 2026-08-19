package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

type projectRetrospectiveCleanupExecutor interface {
	projectRequirementCleanupExecutor
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func deleteProjectRetrospectiveAuthority(ctx context.Context, executor projectRetrospectiveCleanupExecutor, workspaceID, projectID string) error {
	var conflictingID string
	err := executor.QueryRowContext(ctx, `SELECT owned.id
		FROM workspace_project_retrospectives owned
		JOIN workspace_project_retrospectives foreign_copy
		  ON foreign_copy.workspace_id=owned.workspace_id
		 AND foreign_copy.id=owned.id
		 AND foreign_copy.project_id<>owned.project_id
		WHERE owned.workspace_id=? AND owned.project_id=? LIMIT 1`, workspaceID, projectID).Scan(&conflictingID)
	if err == nil {
		return fmt.Errorf("delete Project Retrospective authority ownership drift for %s: %w", conflictingID, contract.ErrInvalidGovernanceMutation)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("validate Project Retrospective cleanup ownership: %w", err)
	}
	statements := []struct {
		query string
		args  []any
	}{
		{`DELETE FROM workspace_resource_revisions
			WHERE workspace_id=? AND resource_kind='project_retrospective' AND resource_id IN (
				SELECT id FROM workspace_project_retrospectives WHERE workspace_id=? AND project_id=?
			)`, []any{workspaceID, workspaceID, projectID}},
		{`DELETE FROM workspace_project_retrospective_action_links WHERE workspace_id=? AND project_id=?`, []any{workspaceID, projectID}},
		{`DELETE FROM workspace_project_retrospective_participants WHERE workspace_id=? AND project_id=?`, []any{workspaceID, projectID}},
		{`DELETE FROM workspace_project_retrospective_revisions WHERE workspace_id=? AND project_id=?`, []any{workspaceID, projectID}},
		{`DELETE FROM workspace_project_retrospectives WHERE workspace_id=? AND project_id=?`, []any{workspaceID, projectID}},
	}
	for _, statement := range statements {
		if _, err := executor.ExecContext(ctx, statement.query, statement.args...); err != nil {
			return fmt.Errorf("delete Project Retrospective authority: %w", err)
		}
	}
	return nil
}
