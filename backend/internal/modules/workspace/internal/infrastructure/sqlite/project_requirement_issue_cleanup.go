package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	requirementDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/requirement"
)

const projectRequirementIssueDeletionActorID = "system:issue-deletion"

type affectedProjectRequirementBaseline struct {
	ID        string
	ProjectID string
}

func clearRequirementIssueReferences(ctx context.Context, connection *sql.Conn, workspaceID, issueID, identifier string) error {
	return clearRequirementIssueReferencesMany(ctx, connection, workspaceID, []string{issueID, identifier}, time.Now().UTC())
}

func clearRequirementIssueReferencesMany(ctx context.Context, connection *sql.Conn, workspaceID string, removed []string, now time.Time) error {
	removed = compactProjectRequirementIssueTokens(removed)
	if len(removed) == 0 {
		return nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(removed)), ",")
	arguments := make([]any, 0, len(removed)+1)
	arguments = append(arguments, workspaceID)
	for _, value := range removed {
		arguments = append(arguments, value)
	}
	rows, err := connection.QueryContext(ctx, `SELECT DISTINCT b.id,b.project_id
		FROM workspace_requirement_baselines b
		JOIN workspace_requirement_issue_links l ON l.baseline_id=b.id
		WHERE b.workspace_id=? AND l.unlinked_revision IS NULL AND l.issue_id IN (`+placeholders+`)
		ORDER BY b.id`, arguments...)
	if err != nil {
		return fmt.Errorf("read Project Requirements affected by Issue deletion: %w", err)
	}
	affected := make([]affectedProjectRequirementBaseline, 0)
	for rows.Next() {
		var value affectedProjectRequirementBaseline
		if err = rows.Scan(&value.ID, &value.ProjectID); err != nil {
			_ = rows.Close()
			return fmt.Errorf("scan Project Requirement affected by Issue deletion: %w", err)
		}
		affected = append(affected, value)
	}
	if err = rows.Close(); err != nil {
		return fmt.Errorf("close Project Requirement Issue deletion scan: %w", err)
	}
	actor := contract.WorkspaceActor{Type: "agent", ID: projectRequirementIssueDeletionActorID}
	timestamp := now.UTC().Format(time.RFC3339Nano)
	for _, affectedBaseline := range affected {
		baseline, _, readErr := readProjectRequirementBaselineOnConnection(ctx, connection, workspaceID, affectedBaseline.ProjectID)
		if readErr != nil {
			return fmt.Errorf("read Project Requirement for Issue deletion: %w", readErr)
		}
		if baseline.ID != affectedBaseline.ID {
			return fmt.Errorf("Project Requirement Issue deletion baseline mismatch")
		}
		expectedRevision := baseline.CurrentRevision
		nextBaseline, revision, transitionErr := baseline.RecordTraceabilityMutation(
			expectedRevision,
			requirementDomain.ActionIssueDeleted,
			projectRequirementIssueDeletionActorID,
			now,
		)
		if transitionErr != nil {
			return fmt.Errorf("advance Project Requirement for Issue deletion: %w", transitionErr)
		}
		baseline = nextBaseline
		linkArguments := make([]any, 0, len(removed)+4)
		linkArguments = append(linkArguments, baseline.CurrentRevision, projectRequirementIssueDeletionActorID, timestamp, baseline.ID)
		for _, value := range removed {
			linkArguments = append(linkArguments, value)
		}
		result, updateErr := connection.ExecContext(ctx, `UPDATE workspace_requirement_issue_links
			SET unlinked_revision=?,unlinked_by=?,unlinked_at=?
			WHERE baseline_id=? AND unlinked_revision IS NULL AND issue_id IN (`+placeholders+`)`, linkArguments...)
		if updateErr != nil {
			return fmt.Errorf("close Project Requirement links for Issue deletion: %w", updateErr)
		}
		updated, updateErr := result.RowsAffected()
		if updateErr != nil {
			return fmt.Errorf("inspect Project Requirement links for Issue deletion: %w", updateErr)
		}
		if updated < 1 {
			return fmt.Errorf("Project Requirement Issue deletion found no active link")
		}
		projectionArguments := make([]any, 0, len(removed)+1)
		projectionArguments = append(projectionArguments, baseline.ID)
		for _, value := range removed {
			projectionArguments = append(projectionArguments, value)
		}
		if _, err = connection.ExecContext(ctx, `DELETE FROM workspace_requirement_review_projections
			WHERE baseline_id=? AND issue_id IN (`+placeholders+`)`, projectionArguments...); err != nil {
			return fmt.Errorf("delete Project Requirement review projections for Issue deletion: %w", err)
		}
		if err = updateProjectRequirementBaseline(ctx, connection, baseline, expectedRevision); err != nil {
			return err
		}
		if err = insertProjectRequirementRevision(ctx, connection, revision); err != nil {
			return err
		}
		requestID := projectRequirementRequestID(baseline.ID, string(requirementDomain.ActionIssueDeleted), baseline.CurrentRevision)
		if err = insertProjectRequirementAudit(ctx, connection, baseline, requirementDomain.ActionIssueDeleted, actor, requestID, now); err != nil {
			return err
		}
		if err = insertProjectRequirementOutbox(ctx, connection, baseline, requirementDomain.ActionIssueDeleted, actor, now); err != nil {
			return err
		}
	}
	return nil
}

func compactProjectRequirementIssueTokens(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, duplicate := seen[value]; duplicate {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
