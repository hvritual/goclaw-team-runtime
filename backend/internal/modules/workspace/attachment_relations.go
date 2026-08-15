package workspace

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	spacecontract "github.com/hvritual/workspace/internal/modules/space/contract"
)

type sqliteAttachmentRelations struct{ db *sql.DB }

func NewSQLiteAttachmentRelations(db *sql.DB) (spacecontract.AttachmentRelations, error) {
	if db == nil {
		return nil, errors.New("workspace sqlite database is required")
	}
	return &sqliteAttachmentRelations{db: db}, nil
}

func (r *sqliteAttachmentRelations) ResolveBinding(ctx context.Context, workspaceID string, issueID, commentID *string) (spacecontract.AttachmentBinding, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return spacecontract.AttachmentBinding{}, spacecontract.ErrAttachmentInvalid
	}
	var binding spacecontract.AttachmentBinding
	if commentID != nil && strings.TrimSpace(*commentID) != "" {
		var resolvedComment, resolvedIssue string
		err := r.db.QueryRowContext(ctx, `SELECT id,issue_id FROM workspace_issue_comments WHERE workspace_id=? AND id=?`, workspaceID, strings.TrimSpace(*commentID)).Scan(&resolvedComment, &resolvedIssue)
		if errors.Is(err, sql.ErrNoRows) {
			return spacecontract.AttachmentBinding{}, spacecontract.ErrAttachmentTargetNotFound
		}
		if err != nil {
			return spacecontract.AttachmentBinding{}, fmt.Errorf("resolve attachment Comment: %w", err)
		}
		binding.CommentID, binding.IssueID = stringPointer(resolvedComment), stringPointer(resolvedIssue)
	}
	if issueID != nil && strings.TrimSpace(*issueID) != "" {
		var resolvedIssue string
		err := r.db.QueryRowContext(ctx, `SELECT id FROM workspace_issues WHERE workspace_id=? AND (id=? OR identifier=?)`, workspaceID, strings.TrimSpace(*issueID), strings.TrimSpace(*issueID)).Scan(&resolvedIssue)
		if errors.Is(err, sql.ErrNoRows) {
			return spacecontract.AttachmentBinding{}, spacecontract.ErrAttachmentTargetNotFound
		}
		if err != nil {
			return spacecontract.AttachmentBinding{}, fmt.Errorf("resolve attachment Issue: %w", err)
		}
		if binding.IssueID != nil && *binding.IssueID != resolvedIssue {
			return spacecontract.AttachmentBinding{}, spacecontract.ErrAttachmentTargetNotFound
		}
		binding.IssueID = stringPointer(resolvedIssue)
	}
	return binding, nil
}

func (r *sqliteAttachmentRelations) Bind(ctx context.Context, executor spacecontract.AttachmentExecutor, workspaceID, assetID string, binding spacecontract.AttachmentBinding) error {
	if binding.IssueID != nil {
		if err := appendAttachmentID(ctx, executor, "workspace_issues", workspaceID, *binding.IssueID, assetID); err != nil {
			return err
		}
	}
	if binding.CommentID != nil {
		if binding.IssueID == nil {
			return spacecontract.ErrAttachmentInvalid
		}
		var issueID string
		if err := executor.QueryRowContext(ctx, `SELECT issue_id FROM workspace_issue_comments WHERE workspace_id=? AND id=?`, workspaceID, *binding.CommentID).Scan(&issueID); errors.Is(err, sql.ErrNoRows) {
			return spacecontract.ErrAttachmentTargetNotFound
		} else if err != nil {
			return err
		}
		if issueID != *binding.IssueID {
			return spacecontract.ErrAttachmentTargetNotFound
		}
		if err := appendAttachmentID(ctx, executor, "workspace_issue_comments", workspaceID, *binding.CommentID, assetID); err != nil {
			return err
		}
	}
	return nil
}

func (r *sqliteAttachmentRelations) Unbind(ctx context.Context, executor spacecontract.AttachmentExecutor, workspaceID, assetID string) error {
	for _, table := range []string{"workspace_issues", "workspace_issue_comments"} {
		rows, err := executor.QueryContext(ctx, `SELECT id,asset_ids FROM `+table+` WHERE workspace_id=? AND EXISTS(SELECT 1 FROM json_each(asset_ids) WHERE value=?)`, workspaceID, assetID)
		if err != nil {
			return err
		}
		var updates []struct{ id, raw string }
		for rows.Next() {
			var value struct{ id, raw string }
			if err := rows.Scan(&value.id, &value.raw); err != nil {
				rows.Close()
				return err
			}
			updates = append(updates, value)
		}
		if err := rows.Close(); err != nil {
			return err
		}
		for _, update := range updates {
			ids, err := decodeAttachmentIDs(update.raw)
			if err != nil {
				return err
			}
			ids = removeAttachmentID(ids, assetID)
			encoded, _ := json.Marshal(ids)
			if _, err := executor.ExecContext(ctx, `UPDATE `+table+` SET asset_ids=? WHERE workspace_id=? AND id=?`, string(encoded), workspaceID, update.id); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *sqliteAttachmentRelations) Locate(ctx context.Context, workspaceID, assetID string) (spacecontract.AttachmentBinding, error) {
	var binding spacecontract.AttachmentBinding
	var issueID string
	err := r.db.QueryRowContext(ctx, `SELECT id FROM workspace_issues WHERE workspace_id=? AND EXISTS(SELECT 1 FROM json_each(asset_ids) WHERE value=?) ORDER BY created_at,id LIMIT 1`, workspaceID, assetID).Scan(&issueID)
	if err == nil {
		binding.IssueID = stringPointer(issueID)
	} else if !errors.Is(err, sql.ErrNoRows) {
		return spacecontract.AttachmentBinding{}, err
	}
	var commentID, commentIssueID string
	err = r.db.QueryRowContext(ctx, `SELECT id,issue_id FROM workspace_issue_comments WHERE workspace_id=? AND EXISTS(SELECT 1 FROM json_each(asset_ids) WHERE value=?) ORDER BY created_at,id LIMIT 1`, workspaceID, assetID).Scan(&commentID, &commentIssueID)
	if err == nil {
		binding.CommentID = stringPointer(commentID)
		if binding.IssueID == nil {
			binding.IssueID = stringPointer(commentIssueID)
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return spacecontract.AttachmentBinding{}, err
	}
	return binding, nil
}

func (r *sqliteAttachmentRelations) ListIssueAssetIDs(ctx context.Context, workspaceID, issueID string) ([]string, error) {
	var raw string
	err := r.db.QueryRowContext(ctx, `SELECT asset_ids FROM workspace_issues WHERE workspace_id=? AND (id=? OR identifier=?)`, workspaceID, issueID, issueID).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, spacecontract.ErrAttachmentTargetNotFound
	}
	if err != nil {
		return nil, err
	}
	return decodeAttachmentIDs(raw)
}

func appendAttachmentID(ctx context.Context, executor spacecontract.AttachmentExecutor, table, workspaceID, rowID, assetID string) error {
	var raw string
	if err := executor.QueryRowContext(ctx, `SELECT asset_ids FROM `+table+` WHERE workspace_id=? AND id=?`, workspaceID, rowID).Scan(&raw); errors.Is(err, sql.ErrNoRows) {
		return spacecontract.ErrAttachmentTargetNotFound
	} else if err != nil {
		return err
	}
	ids, err := decodeAttachmentIDs(raw)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if id == assetID {
			return nil
		}
	}
	ids = append(ids, assetID)
	encoded, _ := json.Marshal(ids)
	_, err = executor.ExecContext(ctx, `UPDATE `+table+` SET asset_ids=? WHERE workspace_id=? AND id=?`, string(encoded), workspaceID, rowID)
	return err
}

func decodeAttachmentIDs(raw string) ([]string, error) {
	result := []string{}
	if strings.TrimSpace(raw) == "" {
		return result, nil
	}
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, fmt.Errorf("decode Workspace attachment references: %w", err)
	}
	return result, nil
}

func removeAttachmentID(values []string, removed string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value != removed {
			result = append(result, value)
		}
	}
	return result
}

func stringPointer(value string) *string { result := value; return &result }

var _ spacecontract.AttachmentRelations = (*sqliteAttachmentRelations)(nil)
