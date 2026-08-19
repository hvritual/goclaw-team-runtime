package sqlite

import (
	"context"
	"database/sql"
	"fmt"
)

type projectRequirementCleanupExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func deleteProjectRequirementAuthority(ctx context.Context, executor projectRequirementCleanupExecutor, workspaceID, projectID string) error {
	statements := []struct {
		query string
		args  []any
	}{
		{`DELETE FROM workspace_audit_entries WHERE workspace_id=? AND (
			(resource_kind='requirement_baseline' AND resource_id IN (
				SELECT id FROM workspace_requirement_baselines WHERE workspace_id=? AND project_id=?
			)) OR (resource_kind IN ('project_outline','project_requirement_access') AND resource_id=?))`, []any{workspaceID, workspaceID, projectID, projectID}},
		{`DELETE FROM workspace_mutation_idempotency WHERE workspace_id=? AND (
			(resource_kind='requirement_baseline' AND resource_id IN (
				SELECT id FROM workspace_requirement_baselines WHERE workspace_id=? AND project_id=?
			)) OR (resource_kind='project_outline' AND resource_id=?))`, []any{workspaceID, workspaceID, projectID, projectID}},
		{`DELETE FROM workspace_outbox_events WHERE workspace_id=? AND (
			(aggregate_kind='requirement_baseline' AND aggregate_id IN (
				SELECT id FROM workspace_requirement_baselines WHERE workspace_id=? AND project_id=?
			)) OR (aggregate_kind IN ('project_outline','project_requirement_access') AND aggregate_id=?))`, []any{workspaceID, workspaceID, projectID, projectID}},
		{`DELETE FROM workspace_requirement_review_projections WHERE baseline_id IN (
			SELECT id FROM workspace_requirement_baselines WHERE workspace_id=? AND project_id=?)`, []any{workspaceID, projectID}},
		{`DELETE FROM workspace_requirement_issue_links WHERE workspace_id=? AND project_id=?`, []any{workspaceID, projectID}},
		{`DELETE FROM workspace_requirement_outline_links WHERE workspace_id=? AND project_id=?`, []any{workspaceID, projectID}},
		{`DELETE FROM workspace_requirement_revisions WHERE baseline_id IN (
			SELECT id FROM workspace_requirement_baselines WHERE workspace_id=? AND project_id=?)`, []any{workspaceID, projectID}},
		{`DELETE FROM workspace_requirement_baselines WHERE workspace_id=? AND project_id=?`, []any{workspaceID, projectID}},
		{`DELETE FROM workspace_project_requirement_grants WHERE workspace_id=? AND project_id=?`, []any{workspaceID, projectID}},
		{`DELETE FROM workspace_project_requirement_access_sets WHERE workspace_id=? AND project_id=?`, []any{workspaceID, projectID}},
		{`DELETE FROM workspace_project_outline_nodes WHERE workspace_id=? AND project_id=?`, []any{workspaceID, projectID}},
		{`DELETE FROM workspace_project_outline_sets WHERE workspace_id=? AND project_id=?`, []any{workspaceID, projectID}},
	}
	for _, statement := range statements {
		if _, err := executor.ExecContext(ctx, statement.query, statement.args...); err != nil {
			return fmt.Errorf("delete Project Requirement authority: %w", err)
		}
	}
	return nil
}
