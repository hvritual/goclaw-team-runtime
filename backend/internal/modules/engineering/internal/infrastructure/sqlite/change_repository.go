package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/hvritual/workspace/internal/modules/engineering/internal/domain"
)

func (s *Store) PutChange(ctx context.Context, value domain.Change) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin put engineering change: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var workKind, workID any
	if workItem := value.WorkItem(); workItem != nil {
		workKind = string(workItem.Kind())
		workID = workItem.ID()
	}
	var acceptedAt any
	if timestamp := value.AcceptedAt(); timestamp != nil {
		acceptedAt = formatTime(*timestamp)
	}
	provenance := value.Provenance()
	if _, err := tx.ExecContext(ctx, `INSERT INTO engineering_changes(
		workspace_id,id,project_id,requirement_id,work_item_kind,work_item_id,run_id,summary,status,
		source_type,source_locator,source_revision,observed_at,created_at,updated_at,accepted_at
	) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
	ON CONFLICT(workspace_id,id) DO UPDATE SET
		project_id=excluded.project_id,
		requirement_id=excluded.requirement_id,
		work_item_kind=excluded.work_item_kind,
		work_item_id=excluded.work_item_id,
		run_id=excluded.run_id,
		summary=excluded.summary,
		status=excluded.status,
		source_type=excluded.source_type,
		source_locator=excluded.source_locator,
		source_revision=excluded.source_revision,
		observed_at=excluded.observed_at,
		created_at=excluded.created_at,
		updated_at=excluded.updated_at,
		accepted_at=excluded.accepted_at`,
		value.WorkspaceID(), value.ID(), value.ProjectID(), value.RequirementID(), workKind, workID, value.RunID(), value.Summary(), string(value.Status()),
		provenance.SourceType(), provenance.Locator(), provenance.Revision(), formatTime(provenance.ObservedAt()),
		formatTime(value.CreatedAt()), formatTime(value.UpdatedAt()), acceptedAt,
	); err != nil {
		return fmt.Errorf("put engineering change: %w", err)
	}
	for _, table := range []string{"engineering_change_entities", "engineering_change_artifacts"} {
		if _, err := tx.ExecContext(ctx, `DELETE FROM `+table+` WHERE workspace_id=? AND change_id=?`, value.WorkspaceID(), value.ID()); err != nil {
			return fmt.Errorf("replace engineering change children: %w", err)
		}
	}
	for ordinal, entityID := range value.AffectedEntityIDs() {
		if _, err := tx.ExecContext(ctx, `INSERT INTO engineering_change_entities(workspace_id,change_id,ordinal,entity_id) VALUES(?,?,?,?)`,
			value.WorkspaceID(), value.ID(), ordinal, entityID); err != nil {
			return fmt.Errorf("put engineering change entity: %w", err)
		}
	}
	for ordinal, artifact := range value.Artifacts() {
		if _, err := tx.ExecContext(ctx, `INSERT INTO engineering_change_artifacts(workspace_id,change_id,ordinal,artifact_kind,locator,revision) VALUES(?,?,?,?,?,?)`,
			value.WorkspaceID(), value.ID(), ordinal, artifact.Kind(), artifact.Locator(), artifact.Revision()); err != nil {
			return fmt.Errorf("put engineering change artifact: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit engineering change: %w", err)
	}
	return nil
}

func (s *Store) GetChange(ctx context.Context, workspaceID, id string) (domain.Change, error) {
	row := s.db.QueryRowContext(ctx, `SELECT project_id,requirement_id,work_item_kind,work_item_id,run_id,summary,status,
		source_type,source_locator,source_revision,observed_at,created_at,updated_at,accepted_at
		FROM engineering_changes WHERE workspace_id=? AND id=?`, workspaceID, id)
	value, err := s.scanChange(ctx, row, workspaceID, id)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Change{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Change{}, fmt.Errorf("get engineering change: %w", err)
	}
	return value, nil
}

func (s *Store) ListChanges(ctx context.Context, workspaceID, affectedEntityID string) ([]domain.Change, error) {
	var rows *sql.Rows
	var err error
	if affectedEntityID == "" {
		rows, err = s.db.QueryContext(ctx, `SELECT id FROM engineering_changes WHERE workspace_id=? ORDER BY id`, workspaceID)
	} else {
		rows, err = s.db.QueryContext(ctx, `SELECT DISTINCT c.id
			FROM engineering_changes c
			JOIN engineering_change_entities e ON e.workspace_id=c.workspace_id AND e.change_id=c.id
			WHERE c.workspace_id=? AND e.entity_id=? ORDER BY c.id`, workspaceID, affectedEntityID)
	}
	if err != nil {
		return nil, fmt.Errorf("list engineering changes: %w", err)
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan engineering change id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, fmt.Errorf("iterate engineering change ids: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close engineering change rows: %w", err)
	}
	values := make([]domain.Change, 0, len(ids))
	for _, id := range ids {
		value, err := s.GetChange(ctx, workspaceID, id)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func (s *Store) scanChange(ctx context.Context, row scanner, workspaceID, id string) (domain.Change, error) {
	var projectID, requirementID, runID, summary, status string
	var workKind, workID, acceptedAt sql.NullString
	var sourceType, locator, revision, observedAt, createdAt, updatedAt string
	if err := row.Scan(&projectID, &requirementID, &workKind, &workID, &runID, &summary, &status,
		&sourceType, &locator, &revision, &observedAt, &createdAt, &updatedAt, &acceptedAt); err != nil {
		return domain.Change{}, err
	}
	var workItem *domain.NodeRef
	if workKind.Valid || workID.Valid {
		if !workKind.Valid || !workID.Valid {
			return domain.Change{}, errors.New("corrupt engineering change work item")
		}
		ref, err := domain.NewNodeRef(domain.NodeKind(workKind.String), workID.String)
		if err != nil {
			return domain.Change{}, fmt.Errorf("rehydrate engineering change work item: %w", err)
		}
		workItem = &ref
	}
	observed, err := parseTime(observedAt)
	if err != nil {
		return domain.Change{}, fmt.Errorf("parse engineering change observed_at: %w", err)
	}
	provenance, err := domain.NewProvenance(sourceType, locator, revision, observed)
	if err != nil {
		return domain.Change{}, fmt.Errorf("rehydrate engineering change provenance: %w", err)
	}
	created, err := parseTime(createdAt)
	if err != nil {
		return domain.Change{}, fmt.Errorf("parse engineering change created_at: %w", err)
	}
	updated, err := parseTime(updatedAt)
	if err != nil {
		return domain.Change{}, fmt.Errorf("parse engineering change updated_at: %w", err)
	}
	var acceptedPointer *time.Time
	if acceptedAt.Valid {
		parsed, err := parseTime(acceptedAt.String)
		if err != nil {
			return domain.Change{}, fmt.Errorf("parse engineering change accepted_at: %w", err)
		}
		acceptedPointer = &parsed
	}
	affected, err := s.loadChangeEntities(ctx, workspaceID, id)
	if err != nil {
		return domain.Change{}, err
	}
	artifacts, err := s.loadChangeArtifacts(ctx, workspaceID, id)
	if err != nil {
		return domain.Change{}, err
	}
	value, err := domain.RehydrateChange(id, workspaceID, projectID, requirementID, workItem, runID, summary, domain.ChangeStatus(status), affected, artifacts, provenance, created, updated, acceptedPointer)
	if err != nil {
		return domain.Change{}, fmt.Errorf("rehydrate engineering change: %w", err)
	}
	return value, nil
}

func (s *Store) loadChangeEntities(ctx context.Context, workspaceID, changeID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT entity_id FROM engineering_change_entities WHERE workspace_id=? AND change_id=? ORDER BY ordinal`, workspaceID, changeID)
	if err != nil {
		return nil, fmt.Errorf("load engineering change entities: %w", err)
	}
	defer rows.Close()
	var values []string
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, fmt.Errorf("scan engineering change entity: %w", err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate engineering change entities: %w", err)
	}
	return values, nil
}

func (s *Store) loadChangeArtifacts(ctx context.Context, workspaceID, changeID string) ([]domain.ArtifactRef, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT artifact_kind,locator,revision FROM engineering_change_artifacts WHERE workspace_id=? AND change_id=? ORDER BY ordinal`, workspaceID, changeID)
	if err != nil {
		return nil, fmt.Errorf("load engineering change artifacts: %w", err)
	}
	defer rows.Close()
	var values []domain.ArtifactRef
	for rows.Next() {
		var kind, locator, revision string
		if err := rows.Scan(&kind, &locator, &revision); err != nil {
			return nil, fmt.Errorf("scan engineering change artifact: %w", err)
		}
		value, err := domain.NewArtifactRef(kind, locator, revision)
		if err != nil {
			return nil, fmt.Errorf("rehydrate engineering change artifact: %w", err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate engineering change artifacts: %w", err)
	}
	return values, nil
}
