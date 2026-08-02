package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	projectrequirements "github.com/multica-ai/multica/server/modules/projectrequirements/application"
)

// TrackingRepository implements requirement coverage and issue links with SQLite.
type TrackingRepository struct{ db *sql.DB }

// NewTracking constructs a SQLite tracking repository.
func NewTracking(db *sql.DB) *TrackingRepository { return &TrackingRepository{db: db} }

// Link atomically associates an issue with a valid requirement item.
func (r *TrackingRepository) Link(ctx context.Context, input projectrequirements.LinkInput) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := r.LinkTx(ctx, tx, input); err != nil {
		return err
	}
	return tx.Commit()
}

// LinkTx is the SQLite adapter seam used when an issue and its link must commit together.
func (*TrackingRepository) LinkTx(ctx context.Context, tx *sql.Tx, input projectrequirements.LinkInput) error {
	if err := activeRevision(ctx, tx, input.WorkspaceID, input.ProjectID, input.Revision); err != nil {
		return err
	}
	if _, err := trackableSection(ctx, tx, input.WorkspaceID, input.ProjectID, input.RequirementKey, input.Revision); err != nil {
		return err
	}
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM issues WHERE id = ? AND workspace_id = ? AND project_id = ?`, input.IssueID, input.WorkspaceID, input.ProjectID).Scan(&count); err != nil {
		return err
	}
	if count != 1 {
		return projectrequirements.ErrInvalidTracking
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO project_requirement_issue_link(workspace_id, project_id, requirement_key, issue_id, created_by, created_at)
		VALUES (?, ?, ?, ?, ?, strftime('%Y-%m-%dT%H:%M:%fZ','now')) ON CONFLICT(workspace_id, project_id, requirement_key, issue_id) DO NOTHING`,
		input.WorkspaceID, input.ProjectID, input.RequirementKey, input.IssueID, input.ActorID)
	return err
}

// Unlink atomically removes a valid issue-to-requirement association.
func (r *TrackingRepository) Unlink(ctx context.Context, input projectrequirements.LinkInput) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := trackableSection(ctx, tx, input.WorkspaceID, input.ProjectID, input.RequirementKey, input.Revision); err != nil {
		return err
	}
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM issues WHERE id = ? AND workspace_id = ? AND project_id = ?`, input.IssueID, input.WorkspaceID, input.ProjectID).Scan(&count); err != nil {
		return err
	}
	if count != 1 {
		return projectrequirements.ErrInvalidTracking
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM project_requirement_issue_link WHERE workspace_id = ? AND project_id = ? AND requirement_key = ? AND issue_id = ?`, input.WorkspaceID, input.ProjectID, input.RequirementKey, input.IssueID); err != nil {
		return err
	}
	return tx.Commit()
}

// Coverage returns coverage for current and effective baseline revisions.
func (r *TrackingRepository) Coverage(ctx context.Context, workspaceID, projectID string) (projectrequirements.Coverage, error) {
	var current int
	var approved sql.NullInt64
	err := r.db.QueryRowContext(ctx, `SELECT current_revision, approved_revision FROM project_requirement_baseline WHERE workspace_id = ? AND project_id = ?`, workspaceID, projectID).Scan(&current, &approved)
	if errors.Is(err, sql.ErrNoRows) {
		return projectrequirements.Coverage{}, nil
	}
	if err != nil {
		return projectrequirements.Coverage{}, err
	}
	linkedIssues, err := r.linkedIssues(ctx, workspaceID, projectID)
	if err != nil {
		return projectrequirements.Coverage{}, err
	}
	currentSnapshot, err := r.snapshot(ctx, workspaceID, projectID, current, linkedIssues)
	if err != nil {
		return projectrequirements.Coverage{}, err
	}
	coverage := projectrequirements.Coverage{Current: &currentSnapshot}
	if approved.Valid {
		if int(approved.Int64) == current {
			coverage.Effective = coverage.Current
		} else {
			effective, err := r.snapshot(ctx, workspaceID, projectID, int(approved.Int64), linkedIssues)
			if err != nil {
				return projectrequirements.Coverage{}, err
			}
			coverage.Effective = &effective
		}
	}
	return coverage, nil
}

func (r *TrackingRepository) snapshot(ctx context.Context, workspaceID, projectID string, revision int, linkedIssues map[string][]projectrequirements.LinkedIssue) (projectrequirements.CoverageSnapshot, error) {
	var raw string
	err := r.db.QueryRowContext(ctx, `SELECT content FROM project_requirement_revision WHERE baseline_id = (SELECT id FROM project_requirement_baseline WHERE workspace_id = ? AND project_id = ?) AND revision = ?`, workspaceID, projectID, revision).Scan(&raw)
	if err != nil {
		return projectrequirements.CoverageSnapshot{}, err
	}
	var content projectrequirements.Content
	if err := json.Unmarshal([]byte(raw), &content); err != nil {
		return projectrequirements.CoverageSnapshot{}, err
	}
	snapshot := projectrequirements.CoverageSnapshot{Revision: revision, Items: make([]projectrequirements.CoverageItem, 0)}
	for _, tracked := range projectrequirements.TrackableItems(content) {
		item := tracked.Item
		issues := linkedIssues[item.Key]
		if issues == nil {
			issues = make([]projectrequirements.LinkedIssue, 0)
		}
		entry := projectrequirements.CoverageItem{RequirementKey: item.Key, Section: tracked.Section, Issues: issues}
		snapshot.Total++
		if len(entry.Issues) > 0 {
			snapshot.Linked++
		} else {
			snapshot.Unlinked++
		}
		for _, issue := range entry.Issues {
			switch issue.Status {
			case "done":
				snapshot.LinkedIssueDone++
			case "blocked":
				snapshot.LinkedIssueBlocked++
			}
		}
		snapshot.Items = append(snapshot.Items, entry)
	}
	return snapshot, nil
}

func (r *TrackingRepository) linkedIssues(ctx context.Context, workspaceID, projectID string) (map[string][]projectrequirements.LinkedIssue, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT l.requirement_key, i.id, i.number, i.title, i.status, w.issue_prefix, l.created_by, l.created_at
		FROM project_requirement_issue_link l JOIN issues i ON i.id = l.issue_id JOIN workspaces w ON w.id = i.workspace_id
		WHERE l.workspace_id = ? AND l.project_id = ? AND i.workspace_id = ? AND i.project_id = ? ORDER BY i.number`, workspaceID, projectID, workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	result := make(map[string][]projectrequirements.LinkedIssue)
	for rows.Next() {
		var key, prefix string
		var number int
		var issue projectrequirements.LinkedIssue
		if err := rows.Scan(&key, &issue.ID, &number, &issue.Title, &issue.Status, &prefix, &issue.CreatedBy, &issue.CreatedAt); err != nil {
			return nil, err
		}
		issue.Identifier = fmt.Sprintf("%s-%d", prefix, number)
		result[key] = append(result[key], issue)
	}
	return result, rows.Err()
}

func trackableSection(ctx context.Context, q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, workspaceID, projectID, key string, revision int) (projectrequirements.TrackableSection, error) {
	if revision < 1 || strings.TrimSpace(key) == "" {
		return "", projectrequirements.ErrInvalidTracking
	}
	var raw string
	err := q.QueryRowContext(ctx, `SELECT content FROM project_requirement_revision WHERE baseline_id = (SELECT id FROM project_requirement_baseline WHERE workspace_id = ? AND project_id = ?) AND revision = ?`, workspaceID, projectID, revision).Scan(&raw)
	if err != nil {
		return "", projectrequirements.ErrInvalidTracking
	}
	var content projectrequirements.Content
	if json.Unmarshal([]byte(raw), &content) != nil {
		return "", projectrequirements.ErrInvalidTracking
	}
	if item, ok := projectrequirements.FindTrackableItem(content, key); ok {
		return item.Section, nil
	}
	return "", projectrequirements.ErrInvalidTracking
}

func activeRevision(ctx context.Context, q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, workspaceID, projectID string, revision int) error {
	var current int
	var approved sql.NullInt64
	if err := q.QueryRowContext(ctx, `SELECT current_revision, approved_revision FROM project_requirement_baseline WHERE workspace_id = ? AND project_id = ?`, workspaceID, projectID).Scan(&current, &approved); err != nil {
		return projectrequirements.ErrInvalidTracking
	}
	if revision == current || (approved.Valid && revision == int(approved.Int64)) {
		return nil
	}
	return projectrequirements.ErrInvalidTracking
}
