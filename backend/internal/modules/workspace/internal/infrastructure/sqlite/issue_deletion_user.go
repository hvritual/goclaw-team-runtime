package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	spacecontract "github.com/hvritual/workspace/internal/modules/space/contract"
	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
)

type issueDeletionRepository struct {
	db                *sql.DB
	attachmentCleanup spacecontract.AttachmentCleanupService
}

func NewIssueDeletionRepository(config Config) (application.IssueDeletionRepository, error) {
	if config.DB == nil {
		return nil, errors.New("workspace sqlite database is required")
	}
	return &issueDeletionRepository{db: config.DB, attachmentCleanup: config.AttachmentCleanup}, nil
}

func (r *issueDeletionRepository) Delete(ctx context.Context, workspaceID, issueID string) (resolvedID string, err error) {
	var resolvedIdentifier string
	connection, err := r.db.Conn(ctx)
	if err != nil {
		return "", fmt.Errorf("acquire Issue deletion connection: %w", err)
	}
	defer connection.Close()
	if _, err = connection.ExecContext(ctx, `PRAGMA busy_timeout = 5000`); err != nil {
		return "", fmt.Errorf("configure Issue deletion lock wait: %w", err)
	}
	if _, err = connection.ExecContext(ctx, `BEGIN IMMEDIATE`); err != nil {
		return "", fmt.Errorf("begin Issue deletion: %w", err)
	}
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
	if err = connection.QueryRowContext(ctx, `SELECT id,identifier FROM workspace_issues WHERE workspace_id=? AND (id=? OR identifier=?)`, workspaceID, issueID, issueID).Scan(&resolvedID, &resolvedIdentifier); errors.Is(err, sql.ErrNoRows) {
		return "", application.ErrIssueRecordNotFound
	} else if err != nil {
		return "", fmt.Errorf("resolve Issue deletion target: %w", err)
	}
	attachmentCleanup, err = prepareOwnedIssueAttachmentCleanup(ctx, connection, r.attachmentCleanup, workspaceID, []string{resolvedID})
	if err != nil {
		return "", fmt.Errorf("prepare Issue attachment cleanup: %w", err)
	}
	if _, err = connection.ExecContext(ctx, `UPDATE workspace_todos SET issue_id=NULL WHERE workspace_id=? AND issue_id IN (?,?)`, workspaceID, resolvedID, resolvedIdentifier); err != nil {
		return "", fmt.Errorf("clear Workspace Todo Issue references: %w", err)
	}
	if _, err = connection.ExecContext(ctx, `UPDATE workspace_issues SET parent_issue_id=NULL WHERE workspace_id=? AND parent_issue_id IN (?,?)`, workspaceID, resolvedID, resolvedIdentifier); err != nil {
		return "", fmt.Errorf("clear child Issue parent references: %w", err)
	}
	if err = clearRequirementIssueReferences(ctx, connection, workspaceID, resolvedID, resolvedIdentifier); err != nil {
		return "", fmt.Errorf("clear Workspace Requirement Issue references: %w", err)
	}
	if _, err = connection.ExecContext(ctx, `DELETE FROM workspace_pins WHERE workspace_id=? AND item_type='issue' AND item_id=?`, workspaceID, resolvedID); err != nil {
		return "", fmt.Errorf("clear Workspace Issue pins: %w", err)
	}
	if err = clearIssueCollaborationDependents(ctx, connection, workspaceID, []string{resolvedID}); err != nil {
		return "", err
	}
	result, err := connection.ExecContext(ctx, `DELETE FROM workspace_issues WHERE workspace_id=? AND id=?`, workspaceID, resolvedID)
	if err != nil {
		return "", fmt.Errorf("delete Workspace Issue: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return "", fmt.Errorf("inspect Workspace Issue deletion: %w", err)
	}
	if rows != 1 {
		return "", fmt.Errorf("inspect Workspace Issue deletion: expected 1 row, got %d", rows)
	}
	if _, err = connection.ExecContext(ctx, `COMMIT`); err != nil {
		return "", fmt.Errorf("commit Workspace Issue deletion: %w", err)
	}
	committed = true
	if attachmentCleanup != nil {
		attachmentCleanup.Commit(context.WithoutCancel(ctx))
	}
	return resolvedID, nil
}
