package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	spacecontract "github.com/hvritual/workspace/internal/modules/space/contract"
)

func prepareOwnedIssueAttachmentCleanup(ctx context.Context, connection *sql.Conn, cleaner spacecontract.AttachmentCleanupService, workspaceID string, issueIDs []string) (spacecontract.AttachmentCleanup, error) {
	if cleaner == nil || len(issueIDs) == 0 {
		return nil, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(issueIDs)), ",")
	arguments := make([]any, 0, len(issueIDs)+1)
	arguments = append(arguments, workspaceID)
	for _, issueID := range issueIDs {
		arguments = append(arguments, issueID)
	}
	candidates := map[string]struct{}{}
	for _, query := range []string{
		`SELECT asset_ids FROM workspace_issues WHERE workspace_id=? AND id IN (` + placeholders + `)`,
		`SELECT asset_ids FROM workspace_issue_comments WHERE workspace_id=? AND issue_id IN (` + placeholders + `)`,
	} {
		rows, err := connection.QueryContext(ctx, query, arguments...)
		if err != nil {
			return nil, fmt.Errorf("list Issue attachment references for cleanup: %w", err)
		}
		for rows.Next() {
			var raw string
			if err := rows.Scan(&raw); err != nil {
				rows.Close()
				return nil, err
			}
			var ids []string
			if err := json.Unmarshal([]byte(raw), &ids); err != nil {
				rows.Close()
				return nil, fmt.Errorf("decode Issue attachment cleanup references: %w", err)
			}
			for _, id := range ids {
				if strings.TrimSpace(id) != "" {
					candidates[id] = struct{}{}
				}
			}
		}
		if err := rows.Close(); err != nil {
			return nil, err
		}
	}
	owned := make([]string, 0, len(candidates))
	for assetID := range candidates {
		referenceArguments := make([]any, 0, len(issueIDs)*2+7)
		referenceArguments = append(referenceArguments, workspaceID)
		for _, issueID := range issueIDs {
			referenceArguments = append(referenceArguments, issueID)
		}
		referenceArguments = append(referenceArguments, assetID, workspaceID)
		for _, issueID := range issueIDs {
			referenceArguments = append(referenceArguments, issueID)
		}
		referenceArguments = append(referenceArguments, assetID, workspaceID, assetID, workspaceID, assetID)
		var retained int
		err := connection.QueryRowContext(ctx, `SELECT
			EXISTS(SELECT 1 FROM workspace_issues WHERE workspace_id=? AND id NOT IN (`+placeholders+`) AND EXISTS(SELECT 1 FROM json_each(asset_ids) WHERE value=?)) OR
			EXISTS(SELECT 1 FROM workspace_issue_comments WHERE workspace_id=? AND issue_id NOT IN (`+placeholders+`) AND EXISTS(SELECT 1 FROM json_each(asset_ids) WHERE value=?)) OR
			EXISTS(SELECT 1 FROM workspace_projects WHERE workspace_id=? AND EXISTS(SELECT 1 FROM json_each(asset_ids) WHERE value=?)) OR
			EXISTS(SELECT 1 FROM workspace_knowledge WHERE workspace_id=? AND EXISTS(SELECT 1 FROM json_each(asset_ids) WHERE value=?))`, referenceArguments...).Scan(&retained)
		if err != nil {
			return nil, fmt.Errorf("inspect retained Workspace attachment references: %w", err)
		}
		if retained == 0 {
			owned = append(owned, assetID)
		}
	}
	return cleaner.PrepareDelete(ctx, connection, workspaceID, owned)
}

func prepareDeletedCommentAttachmentCleanup(ctx context.Context, connection *sql.Conn, cleaner spacecontract.AttachmentCleanupService, workspaceID, rootCommentID string) (spacecontract.AttachmentCleanup, error) {
	if cleaner == nil {
		return nil, nil
	}
	threadQuery := `WITH RECURSIVE descendants(id) AS (
		SELECT id FROM workspace_issue_comments WHERE workspace_id=? AND id=?
		UNION SELECT child.id FROM workspace_issue_comments child JOIN descendants parent ON child.parent_id=parent.id WHERE child.workspace_id=?
	) SELECT comment.id,comment.issue_id,comment.asset_ids FROM workspace_issue_comments comment JOIN descendants ON descendants.id=comment.id WHERE comment.workspace_id=?`
	rows, err := connection.QueryContext(ctx, threadQuery, workspaceID, rootCommentID, workspaceID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list deleted Comment attachment references: %w", err)
	}
	commentIDs := []string{}
	issueSet := map[string]struct{}{}
	candidateSet := map[string]struct{}{}
	for rows.Next() {
		var commentID, issueID, raw string
		if err := rows.Scan(&commentID, &issueID, &raw); err != nil {
			rows.Close()
			return nil, err
		}
		commentIDs = append(commentIDs, commentID)
		issueSet[issueID] = struct{}{}
		var ids []string
		if err := json.Unmarshal([]byte(raw), &ids); err != nil {
			rows.Close()
			return nil, fmt.Errorf("decode deleted Comment attachment references: %w", err)
		}
		for _, id := range ids {
			if id = strings.TrimSpace(id); id != "" {
				candidateSet[id] = struct{}{}
			}
		}
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if len(candidateSet) == 0 {
		return nil, nil
	}
	sort.Strings(commentIDs)
	issueIDs := make([]string, 0, len(issueSet))
	for issueID := range issueSet {
		issueIDs = append(issueIDs, issueID)
	}
	sort.Strings(issueIDs)
	candidates := make([]string, 0, len(candidateSet))
	for assetID := range candidateSet {
		candidates = append(candidates, assetID)
	}
	sort.Strings(candidates)

	commentPlaceholders := strings.TrimSuffix(strings.Repeat("?,", len(commentIDs)), ",")
	issuePlaceholders := strings.TrimSuffix(strings.Repeat("?,", len(issueIDs)), ",")
	owned := make([]string, 0, len(candidates))
	removeFromIssue := make(map[string]map[string]struct{}, len(issueIDs))
	for _, assetID := range candidates {
		arguments := make([]any, 0, len(commentIDs)+len(issueIDs)+8)
		arguments = append(arguments, workspaceID)
		for _, commentID := range commentIDs {
			arguments = append(arguments, commentID)
		}
		arguments = append(arguments, assetID, workspaceID)
		for _, issueID := range issueIDs {
			arguments = append(arguments, issueID)
		}
		arguments = append(arguments, assetID, workspaceID, assetID, workspaceID, assetID)
		var retained int
		err := connection.QueryRowContext(ctx, `SELECT
			EXISTS(SELECT 1 FROM workspace_issue_comments WHERE workspace_id=? AND id NOT IN (`+commentPlaceholders+`) AND EXISTS(SELECT 1 FROM json_each(asset_ids) WHERE value=?)) OR
			EXISTS(SELECT 1 FROM workspace_issues WHERE workspace_id=? AND id NOT IN (`+issuePlaceholders+`) AND EXISTS(SELECT 1 FROM json_each(asset_ids) WHERE value=?)) OR
			EXISTS(SELECT 1 FROM workspace_projects WHERE workspace_id=? AND EXISTS(SELECT 1 FROM json_each(asset_ids) WHERE value=?)) OR
			EXISTS(SELECT 1 FROM workspace_knowledge WHERE workspace_id=? AND EXISTS(SELECT 1 FROM json_each(asset_ids) WHERE value=?))`, arguments...).Scan(&retained)
		if err != nil {
			return nil, fmt.Errorf("inspect retained Comment attachment references: %w", err)
		}
		if retained == 0 {
			owned = append(owned, assetID)
		}
		for _, issueID := range issueIDs {
			externalArguments := make([]any, 0, len(commentIDs)+3)
			externalArguments = append(externalArguments, workspaceID, issueID)
			for _, commentID := range commentIDs {
				externalArguments = append(externalArguments, commentID)
			}
			externalArguments = append(externalArguments, assetID)
			var referencedByOtherComment int
			if err := connection.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM workspace_issue_comments WHERE workspace_id=? AND issue_id=? AND id NOT IN (`+commentPlaceholders+`) AND EXISTS(SELECT 1 FROM json_each(asset_ids) WHERE value=?))`, externalArguments...).Scan(&referencedByOtherComment); err != nil {
				return nil, fmt.Errorf("inspect sibling Comment attachment references: %w", err)
			}
			if referencedByOtherComment == 0 {
				if removeFromIssue[issueID] == nil {
					removeFromIssue[issueID] = map[string]struct{}{}
				}
				removeFromIssue[issueID][assetID] = struct{}{}
			}
		}
	}
	for issueID, removed := range removeFromIssue {
		var raw string
		if err := connection.QueryRowContext(ctx, `SELECT asset_ids FROM workspace_issues WHERE workspace_id=? AND id=?`, workspaceID, issueID).Scan(&raw); err != nil {
			return nil, fmt.Errorf("read parent Issue attachment references: %w", err)
		}
		var ids []string
		if err := json.Unmarshal([]byte(raw), &ids); err != nil {
			return nil, fmt.Errorf("decode parent Issue attachment references: %w", err)
		}
		retained := make([]string, 0, len(ids))
		for _, id := range ids {
			if _, remove := removed[id]; !remove {
				retained = append(retained, id)
			}
		}
		encoded, err := json.Marshal(retained)
		if err != nil {
			return nil, err
		}
		if _, err := connection.ExecContext(ctx, `UPDATE workspace_issues SET asset_ids=? WHERE workspace_id=? AND id=?`, string(encoded), workspaceID, issueID); err != nil {
			return nil, fmt.Errorf("clear parent Issue attachment references: %w", err)
		}
	}
	return cleaner.PrepareDelete(ctx, connection, workspaceID, owned)
}
