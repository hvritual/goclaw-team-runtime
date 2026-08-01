// Package sqlite adapts the project requirement repository to SQLite.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/multica-ai/multica/server/internal/projectrequirements"
)

type ApprovalHook func(context.Context, *sql.Tx, projectrequirements.Record) error

type Repository struct {
	db         *sql.DB
	onApproved ApprovalHook
}

func New(db *sql.DB) *Repository { return &Repository{db: db} }

// NewWithApprovalHook keeps source-transaction hooks in the storage adapter,
// so application services do not learn SQLite transaction details.
func NewWithApprovalHook(db *sql.DB, hook ApprovalHook) *Repository {
	return &Repository{db: db, onApproved: hook}
}

func (r *Repository) Get(ctx context.Context, workspaceID, projectID string) (projectrequirements.Record, error) {
	return r.get(ctx, r.db, workspaceID, projectID)
}

func (r *Repository) SaveDraft(ctx context.Context, input projectrequirements.SaveDraftInput) (projectrequirements.Record, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return projectrequirements.Record{}, err
	}
	defer tx.Rollback()

	record, err := r.get(ctx, tx, input.WorkspaceID, input.ProjectID)
	if errors.Is(err, projectrequirements.ErrNotFound) {
		if input.ExpectedRevision != 0 {
			return projectrequirements.Record{}, projectrequirements.ErrRevisionConflict
		}
		return r.insertDraft(ctx, tx, input)
	}
	if err != nil {
		return projectrequirements.Record{}, err
	}
	if record.Baseline.CurrentRevision != input.ExpectedRevision {
		return projectrequirements.Record{}, projectrequirements.ErrRevisionConflict
	}
	if record.Baseline.Status == projectrequirements.StatusInReview {
		return projectrequirements.Record{}, projectrequirements.ErrInvalidTransition
	}

	now := timestamp()
	content, err := marshalContent(input.Content)
	if err != nil {
		return projectrequirements.Record{}, err
	}
	if record.Baseline.Status == projectrequirements.StatusApproved {
		next := record.Baseline.CurrentRevision + 1
		if _, err := tx.ExecContext(ctx, `UPDATE project_requirement_baseline
			SET status = ?, current_revision = ?, submitted_by = NULL, submitted_at = NULL,
			approved_by = NULL, approved_at = NULL, updated_at = ? WHERE id = ? AND current_revision = ?`,
			projectrequirements.StatusDraft, next, now, record.Baseline.ID, input.ExpectedRevision); err != nil {
			return projectrequirements.Record{}, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO project_requirement_revision(
			baseline_id, revision, content, change_summary, actor_id, created_at
		) VALUES (?, ?, ?, ?, ?, ?)`, record.Baseline.ID, next, content, input.ChangeSummary, input.ActorID, now); err != nil {
			return projectrequirements.Record{}, err
		}
	} else {
		result, err := tx.ExecContext(ctx, `UPDATE project_requirement_revision
			SET content = ?, change_summary = ?, actor_id = ?, created_at = ?
			WHERE baseline_id = ? AND revision = ?`, content, input.ChangeSummary, input.ActorID, now, record.Baseline.ID, input.ExpectedRevision)
		if err != nil {
			return projectrequirements.Record{}, err
		}
		affected, _ := result.RowsAffected()
		if affected != 1 {
			return projectrequirements.Record{}, projectrequirements.ErrRevisionConflict
		}
		if _, err := tx.ExecContext(ctx, `UPDATE project_requirement_baseline SET updated_at = ? WHERE id = ?`, now, record.Baseline.ID); err != nil {
			return projectrequirements.Record{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return projectrequirements.Record{}, err
	}
	return r.Get(ctx, input.WorkspaceID, input.ProjectID)
}

func (r *Repository) insertDraft(ctx context.Context, tx *sql.Tx, input projectrequirements.SaveDraftInput) (projectrequirements.Record, error) {
	now := timestamp()
	id := uuid.NewString()
	content, err := marshalContent(input.Content)
	if err != nil {
		return projectrequirements.Record{}, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO project_requirement_baseline(
		id, workspace_id, project_id, status, current_revision, approved_revision, submitted_by, submitted_at, approved_by, approved_at, created_at, updated_at
	) VALUES (?, ?, ?, ?, 1, NULL, NULL, NULL, NULL, NULL, ?, ?)`, id, input.WorkspaceID, input.ProjectID, projectrequirements.StatusDraft, now, now); err != nil {
		return projectrequirements.Record{}, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO project_requirement_revision(
		baseline_id, revision, content, change_summary, actor_id, created_at
	) VALUES (?, 1, ?, ?, ?, ?)`, id, content, input.ChangeSummary, input.ActorID, now); err != nil {
		return projectrequirements.Record{}, err
	}
	if err := tx.Commit(); err != nil {
		return projectrequirements.Record{}, err
	}
	return r.Get(ctx, input.WorkspaceID, input.ProjectID)
}

func (r *Repository) SubmitReview(ctx context.Context, input projectrequirements.TransitionInput) (projectrequirements.Record, error) {
	return r.transition(ctx, input, projectrequirements.StatusDraft, projectrequirements.StatusInReview, false)
}

func (r *Repository) Approve(ctx context.Context, input projectrequirements.TransitionInput) (projectrequirements.Record, error) {
	return r.transition(ctx, input, projectrequirements.StatusInReview, projectrequirements.StatusApproved, true)
}

func (r *Repository) Withdraw(ctx context.Context, input projectrequirements.TransitionInput) (projectrequirements.Record, error) {
	return r.transition(ctx, input, projectrequirements.StatusInReview, projectrequirements.StatusDraft, false)
}

func (r *Repository) transition(ctx context.Context, input projectrequirements.TransitionInput, from, to projectrequirements.Status, approve bool) (projectrequirements.Record, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return projectrequirements.Record{}, err
	}
	defer tx.Rollback()
	record, err := r.get(ctx, tx, input.WorkspaceID, input.ProjectID)
	if err != nil {
		return projectrequirements.Record{}, err
	}
	if record.Baseline.CurrentRevision != input.ExpectedRevision {
		return projectrequirements.Record{}, projectrequirements.ErrRevisionConflict
	}
	if record.Baseline.Status != from {
		return projectrequirements.Record{}, projectrequirements.ErrInvalidTransition
	}
	now := timestamp()
	isSubmit := to == projectrequirements.StatusInReview
	isWithdraw := to == projectrequirements.StatusDraft
	result, err := tx.ExecContext(ctx, `UPDATE project_requirement_baseline
		SET status = ?, approved_revision = CASE WHEN ? THEN ? ELSE approved_revision END,
			submitted_by = CASE WHEN ? THEN ? WHEN ? THEN NULL ELSE submitted_by END,
			submitted_at = CASE WHEN ? THEN ? WHEN ? THEN NULL ELSE submitted_at END,
			approved_by = CASE WHEN ? THEN ? ELSE approved_by END,
			approved_at = CASE WHEN ? THEN ? ELSE approved_at END, updated_at = ?
		WHERE id = ? AND status = ? AND current_revision = ?`,
		to, approve, input.ExpectedRevision,
		isSubmit, input.ActorID, isWithdraw,
		isSubmit, now, isWithdraw,
		approve, input.ActorID,
		approve, now,
		now, record.Baseline.ID, from, input.ExpectedRevision)
	if err != nil {
		return projectrequirements.Record{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return projectrequirements.Record{}, err
	}
	if affected != 1 {
		return projectrequirements.Record{}, projectrequirements.ErrRevisionConflict
	}
	if approve {
		_, err = tx.ExecContext(ctx, `UPDATE project_requirement_revision SET approved_by = ?, approved_at = ? WHERE baseline_id = ? AND revision = ?`, input.ActorID, now, record.Baseline.ID, input.ExpectedRevision)
	} else if to == projectrequirements.StatusInReview {
		_, err = tx.ExecContext(ctx, `UPDATE project_requirement_revision SET submitted_by = ?, submitted_at = ? WHERE baseline_id = ? AND revision = ?`, input.ActorID, now, record.Baseline.ID, input.ExpectedRevision)
	} else {
		_, err = tx.ExecContext(ctx, `UPDATE project_requirement_revision SET submitted_by = NULL, submitted_at = NULL WHERE baseline_id = ? AND revision = ?`, record.Baseline.ID, input.ExpectedRevision)
	}
	if err != nil {
		return projectrequirements.Record{}, err
	}
	if approve && r.onApproved != nil {
		approved, err := r.get(ctx, tx, input.WorkspaceID, input.ProjectID)
		if err != nil {
			return projectrequirements.Record{}, err
		}
		if err := r.onApproved(ctx, tx, approved); err != nil {
			return projectrequirements.Record{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return projectrequirements.Record{}, err
	}
	return r.Get(ctx, input.WorkspaceID, input.ProjectID)
}

type rowQuerier interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func (r *Repository) get(ctx context.Context, q rowQuerier, workspaceID, projectID string) (projectrequirements.Record, error) {
	var record projectrequirements.Record
	var approved sql.NullInt64
	var submittedBy, submittedAt, approvedBy, approvedAt sql.NullString
	err := q.QueryRowContext(ctx, `SELECT id, workspace_id, project_id, status, current_revision, approved_revision, submitted_by, submitted_at, approved_by, approved_at, created_at, updated_at
		FROM project_requirement_baseline WHERE workspace_id = ? AND project_id = ?`, workspaceID, projectID).Scan(
		&record.Baseline.ID, &record.Baseline.WorkspaceID, &record.Baseline.ProjectID, &record.Baseline.Status,
		&record.Baseline.CurrentRevision, &approved, &submittedBy, &submittedAt, &approvedBy, &approvedAt, &record.Baseline.CreatedAt, &record.Baseline.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return projectrequirements.Record{}, projectrequirements.ErrNotFound
	}
	if err != nil {
		return projectrequirements.Record{}, err
	}
	if approved.Valid {
		value := int(approved.Int64)
		record.Baseline.ApprovedRevision = &value
	}
	if submittedBy.Valid {
		value := submittedBy.String
		record.Baseline.SubmittedBy = &value
	}
	if submittedAt.Valid {
		value := submittedAt.String
		record.Baseline.SubmittedAt = &value
	}
	if approvedBy.Valid {
		value := approvedBy.String
		record.Baseline.ApprovedBy = &value
	}
	if approvedAt.Valid {
		value := approvedAt.String
		record.Baseline.ApprovedAt = &value
	}
	var currentRaw string
	if err := q.QueryRowContext(ctx, `SELECT content FROM project_requirement_revision WHERE baseline_id = ? AND revision = ?`, record.Baseline.ID, record.Baseline.CurrentRevision).Scan(&currentRaw); err != nil {
		return projectrequirements.Record{}, fmt.Errorf("load current project requirement revision: %w", err)
	}
	currentContent, err := unmarshalContent(currentRaw)
	if err != nil {
		return projectrequirements.Record{}, err
	}
	record.CurrentContent = currentContent
	if record.Baseline.ApprovedRevision != nil {
		var content string
		if err := q.QueryRowContext(ctx, `SELECT content FROM project_requirement_revision WHERE baseline_id = ? AND revision = ?`, record.Baseline.ID, *record.Baseline.ApprovedRevision).Scan(&content); err != nil {
			return projectrequirements.Record{}, fmt.Errorf("load effective project requirement revision: %w", err)
		}
		parsed, err := unmarshalContent(content)
		if err != nil {
			return projectrequirements.Record{}, err
		}
		record.EffectiveContent = &parsed
	}
	rows, err := q.QueryContext(ctx, `SELECT revision, content, change_summary, actor_id, created_at, submitted_by, submitted_at, approved_by, approved_at
		FROM project_requirement_revision WHERE baseline_id = ? ORDER BY revision DESC`, record.Baseline.ID)
	if err != nil {
		return projectrequirements.Record{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var revision projectrequirements.Revision
		var raw string
		var submittedBy, submittedAt, approvedBy, approvedAt sql.NullString
		if err := rows.Scan(&revision.Revision, &raw, &revision.ChangeSummary, &revision.ActorID, &revision.CreatedAt, &submittedBy, &submittedAt, &approvedBy, &approvedAt); err != nil {
			return projectrequirements.Record{}, err
		}
		content, err := unmarshalContent(raw)
		if err != nil {
			return projectrequirements.Record{}, err
		}
		revision.BaselineID, revision.Content = record.Baseline.ID, content
		if submittedBy.Valid {
			value := submittedBy.String
			revision.SubmittedBy = &value
		}
		if submittedAt.Valid {
			value := submittedAt.String
			revision.SubmittedAt = &value
		}
		if approvedBy.Valid {
			value := approvedBy.String
			revision.ApprovedBy = &value
		}
		if approvedAt.Valid {
			value := approvedAt.String
			revision.ApprovedAt = &value
		}
		revision.State = stateFor(revision.Revision, record.Baseline)
		record.History = append(record.History, revision)
	}
	if err := rows.Err(); err != nil {
		return projectrequirements.Record{}, err
	}
	return record, nil
}

func stateFor(revision int, baseline projectrequirements.Baseline) projectrequirements.Status {
	if revision == baseline.CurrentRevision {
		return baseline.Status
	}
	if baseline.ApprovedRevision != nil && revision == *baseline.ApprovedRevision {
		return projectrequirements.StatusApproved
	}
	return projectrequirements.StatusSuperseded
}

func marshalContent(content projectrequirements.Content) (string, error) {
	bytes, err := json.Marshal(content)
	return string(bytes), err
}
func unmarshalContent(raw string) (projectrequirements.Content, error) {
	var content projectrequirements.Content
	err := json.Unmarshal([]byte(raw), &content)
	return content, err
}
func timestamp() string { return time.Now().UTC().Format(time.RFC3339Nano) }
