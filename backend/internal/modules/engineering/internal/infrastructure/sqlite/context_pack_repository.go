package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/hvritual/workspace/internal/modules/engineering/internal/domain"
)

func (s *Store) PutContextPack(ctx context.Context, value domain.ContextPack) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin put engineering context pack: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var existingChecksum, existingCreatedAt string
	err = tx.QueryRowContext(ctx, `SELECT checksum,created_at FROM engineering_context_packs WHERE workspace_id=? AND id=?`,
		value.WorkspaceID(), value.ID()).Scan(&existingChecksum, &existingCreatedAt)
	if err == nil {
		if existingChecksum == value.Checksum() && existingCreatedAt == formatTime(value.CreatedAt()) {
			return nil
		}
		return domain.ErrConflict
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("check engineering context pack identity: %w", err)
	}
	workItem := value.WorkItem()
	if _, err := tx.ExecContext(ctx, `INSERT INTO engineering_context_packs(
		workspace_id,id,work_item_kind,work_item_id,work_item_revision,policy_version,checksum,created_at
	) VALUES(?,?,?,?,?,?,?,?)`, value.WorkspaceID(), value.ID(), string(workItem.Kind()), workItem.ID(), value.WorkItemRevision(),
		value.PolicyVersion(), value.Checksum(), formatTime(value.CreatedAt())); err != nil {
		return fmt.Errorf("put engineering context pack: %w", err)
	}
	for ordinal, entityID := range value.TargetEntityIDs() {
		if _, err := tx.ExecContext(ctx, `INSERT INTO engineering_context_pack_targets(workspace_id,context_pack_id,ordinal,entity_id) VALUES(?,?,?,?)`,
			value.WorkspaceID(), value.ID(), ordinal, entityID); err != nil {
			return fmt.Errorf("put engineering context pack target: %w", err)
		}
	}
	for ordinal, reference := range value.References() {
		if _, err := tx.ExecContext(ctx, `INSERT INTO engineering_context_pack_references(
			workspace_id,context_pack_id,ordinal,context_kind,reference_id,revision,checksum
		) VALUES(?,?,?,?,?,?,?)`, value.WorkspaceID(), value.ID(), ordinal, string(reference.Kind()), reference.ID(), reference.Revision(), reference.Checksum()); err != nil {
			return fmt.Errorf("put engineering context pack reference: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit engineering context pack: %w", err)
	}
	return nil
}

func (s *Store) GetContextPack(ctx context.Context, workspaceID, id string) (domain.ContextPack, error) {
	var workKind, workID, workRevision, policyVersion, checksum, createdAt string
	if err := s.db.QueryRowContext(ctx, `SELECT work_item_kind,work_item_id,work_item_revision,policy_version,checksum,created_at
		FROM engineering_context_packs WHERE workspace_id=? AND id=?`, workspaceID, id,
	).Scan(&workKind, &workID, &workRevision, &policyVersion, &checksum, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ContextPack{}, domain.ErrNotFound
		}
		return domain.ContextPack{}, fmt.Errorf("get engineering context pack: %w", err)
	}
	workItem, err := domain.NewNodeRef(domain.NodeKind(workKind), workID)
	if err != nil {
		return domain.ContextPack{}, fmt.Errorf("rehydrate engineering context pack work item: %w", err)
	}
	targets, err := s.loadContextPackTargets(ctx, workspaceID, id)
	if err != nil {
		return domain.ContextPack{}, err
	}
	references, err := s.loadContextPackReferences(ctx, workspaceID, id)
	if err != nil {
		return domain.ContextPack{}, err
	}
	created, err := parseTime(createdAt)
	if err != nil {
		return domain.ContextPack{}, fmt.Errorf("parse engineering context pack created_at: %w", err)
	}
	value, err := domain.RehydrateContextPack(id, workspaceID, workItem, workRevision, targets, references, policyVersion, checksum, created)
	if err != nil {
		return domain.ContextPack{}, fmt.Errorf("rehydrate engineering context pack: %w", err)
	}
	return value, nil
}

func (s *Store) loadContextPackTargets(ctx context.Context, workspaceID, contextPackID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT entity_id FROM engineering_context_pack_targets WHERE workspace_id=? AND context_pack_id=? ORDER BY ordinal`, workspaceID, contextPackID)
	if err != nil {
		return nil, fmt.Errorf("load engineering context pack targets: %w", err)
	}
	defer rows.Close()
	var values []string
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, fmt.Errorf("scan engineering context pack target: %w", err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate engineering context pack targets: %w", err)
	}
	return values, nil
}

func (s *Store) loadContextPackReferences(ctx context.Context, workspaceID, contextPackID string) ([]domain.ContextReference, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT context_kind,reference_id,revision,checksum
		FROM engineering_context_pack_references WHERE workspace_id=? AND context_pack_id=? ORDER BY ordinal`, workspaceID, contextPackID)
	if err != nil {
		return nil, fmt.Errorf("load engineering context pack references: %w", err)
	}
	defer rows.Close()
	var values []domain.ContextReference
	for rows.Next() {
		var kind, id, revision, checksum string
		if err := rows.Scan(&kind, &id, &revision, &checksum); err != nil {
			return nil, fmt.Errorf("scan engineering context pack reference: %w", err)
		}
		value, err := domain.NewContextReference(domain.ContextKind(kind), id, revision, checksum)
		if err != nil {
			return nil, fmt.Errorf("rehydrate engineering context pack reference: %w", err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate engineering context pack references: %w", err)
	}
	return values, nil
}
