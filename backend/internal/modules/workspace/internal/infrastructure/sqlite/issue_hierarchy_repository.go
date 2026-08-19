package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	spacecontract "github.com/hvritual/workspace/internal/modules/space/contract"
	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
	issueDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/issue"
)

func (r *issueRepository) ListChildren(ctx context.Context, workspaceID string, parentIDs []string) ([]issueDomain.Issue, error) {
	if len(parentIDs) == 0 {
		return []issueDomain.Issue{}, nil
	}
	placeholders := make([]string, len(parentIDs))
	arguments := make([]any, 0, len(parentIDs)+1)
	arguments = append(arguments, workspaceID)
	for index, parentID := range parentIDs {
		placeholders[index] = "?"
		arguments = append(arguments, parentID)
	}
	rows, err := r.db.QueryContext(ctx, `SELECT `+issueColumns+` FROM workspace_issues
		WHERE workspace_id=? AND parent_issue_id IN (`+strings.Join(placeholders, ",")+`)
		ORDER BY parent_issue_id ASC, number ASC, id ASC`, arguments...)
	if err != nil {
		return nil, fmt.Errorf("list Workspace child Issues: %w", err)
	}
	defer rows.Close()
	values := make([]issueDomain.Issue, 0)
	for rows.Next() {
		value, scanErr := scanIssue(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Workspace child Issues: %w", err)
	}
	return values, nil
}

func (r *issueRepository) ChildProgress(ctx context.Context, workspaceID string) ([]application.IssueChildProgress, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT parent_issue_id,COUNT(*),
		SUM(CASE WHEN status IN ('done','cancelled') THEN 1 ELSE 0 END)
		FROM workspace_issues WHERE workspace_id=? AND parent_issue_id IS NOT NULL
		GROUP BY parent_issue_id ORDER BY parent_issue_id`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list Workspace child Issue progress: %w", err)
	}
	defer rows.Close()
	result := make([]application.IssueChildProgress, 0)
	for rows.Next() {
		var row application.IssueChildProgress
		if err := rows.Scan(&row.ParentIssueID, &row.Total, &row.Done); err != nil {
			return nil, fmt.Errorf("scan Workspace child Issue progress: %w", err)
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Workspace child Issue progress: %w", err)
	}
	return result, nil
}

func (r *issueRepository) BatchUpdate(ctx context.Context, command application.IssueBatchUpdateCommand) (updated []issueDomain.Issue, err error) {
	connection, err := r.issueWriteConnection(ctx, "batch update")
	if err != nil {
		return nil, err
	}
	defer connection.Close()
	committed := false
	defer func() {
		if !committed {
			_, _ = connection.ExecContext(context.WithoutCancel(ctx), `ROLLBACK`)
		}
	}()

	updated = make([]issueDomain.Issue, 0, len(command.IssueIDs))
	seen := make(map[string]struct{}, len(command.IssueIDs))
	for _, issueID := range command.IssueIDs {
		current, findErr := scanIssue(connection.QueryRowContext(ctx, `SELECT `+issueColumns+`
			FROM workspace_issues WHERE workspace_id=? AND (id=? OR identifier=?)`, command.WorkspaceID, issueID, issueID))
		if findErr != nil {
			return nil, findErr
		}
		if _, duplicate := seen[current.ID]; duplicate {
			return nil, fmt.Errorf("%w: duplicate resolved Issue", application.ErrIssueBatchConflict)
		}
		seen[current.ID] = struct{}{}
		value, applyErr := current.Apply(command.Patch, command.Now)
		if applyErr != nil {
			return nil, fmt.Errorf("%w: %v", application.ErrIssueBatchInvalid, applyErr)
		}
		if value.ParentIssueID != nil {
			cycle, cycleErr := wouldCreateParentCycleOnConnection(ctx, connection, command.WorkspaceID, value.ID, *value.ParentIssueID)
			if cycleErr != nil {
				return nil, cycleErr
			}
			if cycle {
				return nil, fmt.Errorf("%w: circular parent relationship", application.ErrIssueBatchInvalid)
			}
		}
		if updateErr := updateIssueOnConnection(ctx, connection, value); updateErr != nil {
			return nil, updateErr
		}
		updated = append(updated, value)
	}
	if _, err := connection.ExecContext(ctx, `COMMIT`); err != nil {
		return nil, fmt.Errorf("commit Workspace Issue batch update: %w", err)
	}
	committed = true
	return updated, nil
}

func (r *issueRepository) BatchDelete(ctx context.Context, command application.IssueBatchDeleteCommand) (deleted []string, err error) {
	connection, err := r.issueWriteConnection(ctx, "batch delete")
	if err != nil {
		return nil, err
	}
	defer connection.Close()
	committed := false
	var attachmentCleanup spacecontract.AttachmentCleanup
	defer func() {
		if !committed {
			_, _ = connection.ExecContext(context.WithoutCancel(ctx), `ROLLBACK`)
			if attachmentCleanup != nil {
				err = errors.Join(err, attachmentCleanup.Rollback(context.WithoutCancel(ctx)))
			}
		}
	}()

	tokens := make([]string, 0, len(command.IssueIDs)*2)
	deleted = make([]string, 0, len(command.IssueIDs))
	seen := make(map[string]struct{}, len(command.IssueIDs))
	for _, issueID := range command.IssueIDs {
		var resolvedID, identifier string
		if err := connection.QueryRowContext(ctx, `SELECT id,identifier FROM workspace_issues
			WHERE workspace_id=? AND (id=? OR identifier=?)`, command.WorkspaceID, issueID, issueID).Scan(&resolvedID, &identifier); errors.Is(err, sql.ErrNoRows) {
			return nil, application.ErrIssueRecordNotFound
		} else if err != nil {
			return nil, fmt.Errorf("resolve Workspace Issue batch delete target: %w", err)
		}
		if _, duplicate := seen[resolvedID]; duplicate {
			return nil, fmt.Errorf("%w: duplicate resolved Issue", application.ErrIssueBatchConflict)
		}
		seen[resolvedID] = struct{}{}
		deleted = append(deleted, resolvedID)
		tokens = append(tokens, resolvedID, identifier)
	}
	attachmentCleanup, err = prepareOwnedIssueAttachmentCleanup(ctx, connection, r.attachmentCleanup, command.WorkspaceID, deleted)
	if err != nil {
		return nil, fmt.Errorf("prepare batch Issue attachment cleanup: %w", err)
	}
	if err := clearBatchIssueDependents(ctx, connection, command.WorkspaceID, deleted, tokens, command.Now); err != nil {
		return nil, err
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(deleted)), ",")
	arguments := make([]any, 0, len(deleted)+1)
	arguments = append(arguments, command.WorkspaceID)
	for _, issueID := range deleted {
		arguments = append(arguments, issueID)
	}
	result, err := connection.ExecContext(ctx, `DELETE FROM workspace_issues WHERE workspace_id=? AND id IN (`+placeholders+`)`, arguments...)
	if err != nil {
		return nil, fmt.Errorf("delete Workspace Issue batch: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("inspect Workspace Issue batch deletion: %w", err)
	}
	if rows != int64(len(deleted)) {
		return nil, fmt.Errorf("inspect Workspace Issue batch deletion: expected %d rows, got %d", len(deleted), rows)
	}
	if _, err := connection.ExecContext(ctx, `COMMIT`); err != nil {
		return nil, fmt.Errorf("commit Workspace Issue batch deletion: %w", err)
	}
	committed = true
	if attachmentCleanup != nil {
		attachmentCleanup.Commit(context.WithoutCancel(ctx))
	}
	return deleted, nil
}

func (r *issueRepository) issueWriteConnection(ctx context.Context, operation string) (*sql.Conn, error) {
	connection, err := r.db.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquire Issue %s connection: %w", operation, err)
	}
	if _, err := connection.ExecContext(ctx, `PRAGMA busy_timeout = 5000`); err != nil {
		_ = connection.Close()
		return nil, fmt.Errorf("configure Issue %s lock wait: %w", operation, err)
	}
	if _, err := connection.ExecContext(ctx, `BEGIN IMMEDIATE`); err != nil {
		_ = connection.Close()
		return nil, fmt.Errorf("begin immediate Issue %s: %w", operation, err)
	}
	return connection, nil
}

func updateIssueOnConnection(ctx context.Context, connection *sql.Conn, value issueDomain.Issue) error {
	_, _, assets, err := encodeIssueJSON(value)
	if err != nil {
		return err
	}
	result, err := connection.ExecContext(ctx, `UPDATE workspace_issues SET
		title=?,description=?,status=?,priority=?,assignee_type=?,assignee_id=?,
		parent_issue_id=?,project_id=?,position=?,stage=?,start_date=?,due_date=?,asset_ids=?,updated_at=?
		WHERE workspace_id=? AND id=?`,
		value.Title, nullableString(value.Description), value.Status, value.Priority,
		nullableString(value.AssigneeType), nullableString(value.AssigneeID), nullableString(value.ParentIssueID), nullableString(value.ProjectID),
		value.Position, nullableInt32(value.Stage), nullableString(value.StartDate), nullableString(value.DueDate), assets,
		value.UpdatedAt.Format(time.RFC3339Nano), value.WorkspaceID, value.ID,
	)
	if err != nil {
		return fmt.Errorf("update Workspace Issue batch row: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect Workspace Issue batch row: %w", err)
	}
	if rows != 1 {
		return application.ErrIssueRecordNotFound
	}
	return nil
}

func clearBatchIssueDependents(ctx context.Context, connection *sql.Conn, workspaceID string, issueIDs, tokens []string, now time.Time) error {
	tokenPlaceholders := strings.TrimSuffix(strings.Repeat("?,", len(tokens)), ",")
	tokenArguments := make([]any, 0, len(tokens)+1)
	tokenArguments = append(tokenArguments, workspaceID)
	for _, token := range tokens {
		tokenArguments = append(tokenArguments, token)
	}
	if _, err := connection.ExecContext(ctx, `UPDATE workspace_todos SET issue_id=NULL WHERE workspace_id=? AND issue_id IN (`+tokenPlaceholders+`)`, tokenArguments...); err != nil {
		return fmt.Errorf("clear Workspace Todo batch Issue references: %w", err)
	}
	if _, err := connection.ExecContext(ctx, `UPDATE workspace_issues SET parent_issue_id=NULL WHERE workspace_id=? AND parent_issue_id IN (`+tokenPlaceholders+`)`, tokenArguments...); err != nil {
		return fmt.Errorf("clear child Issue batch parent references: %w", err)
	}
	if err := clearRequirementIssueReferencesMany(ctx, connection, workspaceID, tokens, now); err != nil {
		return fmt.Errorf("clear Workspace Requirement batch Issue references: %w", err)
	}
	idPlaceholders := strings.TrimSuffix(strings.Repeat("?,", len(issueIDs)), ",")
	idArguments := make([]any, 0, len(issueIDs)+1)
	idArguments = append(idArguments, workspaceID)
	for _, issueID := range issueIDs {
		idArguments = append(idArguments, issueID)
	}
	if _, err := connection.ExecContext(ctx, `DELETE FROM workspace_pins WHERE workspace_id=? AND item_type='issue' AND item_id IN (`+idPlaceholders+`)`, idArguments...); err != nil {
		return fmt.Errorf("clear Workspace batch Issue pins: %w", err)
	}
	if err := clearIssueCollaborationDependents(ctx, connection, workspaceID, issueIDs); err != nil {
		return err
	}
	return nil
}

var _ application.IssueHierarchyRepository = (*issueRepository)(nil)
