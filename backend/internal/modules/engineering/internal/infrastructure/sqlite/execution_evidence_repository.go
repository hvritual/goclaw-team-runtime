package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hvritual/workspace/internal/modules/engineering/internal/domain"
)

func (s *Store) CreateExecutionItem(ctx context.Context, value domain.ExecutionItem) error {
	normalized, err := domain.NewExecutionItem(
		value.ID(), value.WorkspaceID(), value.Kind(), value.SourceType(), value.SourceID(), value.SourceLocator(), value.CreatedAt(),
	)
	if err != nil {
		return err
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin create engineering execution item: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var kind, sourceType, sourceID, sourceLocator, createdAt string
	err = tx.QueryRowContext(ctx, `SELECT kind,source_type,source_id,source_locator,created_at
        FROM engineering_execution_items WHERE workspace_id=? AND id=?`, normalized.WorkspaceID(), normalized.ID()).
		Scan(&kind, &sourceType, &sourceID, &sourceLocator, &createdAt)
	if err == nil {
		if kind == string(normalized.Kind()) && sourceType == normalized.SourceType() && sourceID == normalized.SourceID() &&
			sourceLocator == normalized.SourceLocator() && createdAt == formatTime(normalized.CreatedAt()) {
			return nil
		}
		return domain.ErrConflict
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("check engineering execution item identity: %w", err)
	}

	var existingID string
	err = tx.QueryRowContext(ctx, `SELECT id FROM engineering_execution_items
        WHERE workspace_id=? AND kind=? AND source_type=? AND source_id=?`,
		normalized.WorkspaceID(), string(normalized.Kind()), normalized.SourceType(), normalized.SourceID()).Scan(&existingID)
	if err == nil {
		return domain.ErrConflict
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("check engineering execution item source identity: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `INSERT INTO engineering_execution_items(
        workspace_id,id,kind,source_type,source_id,source_locator,created_at
    ) VALUES(?,?,?,?,?,?,?)`, normalized.WorkspaceID(), normalized.ID(), string(normalized.Kind()), normalized.SourceType(),
		normalized.SourceID(), normalized.SourceLocator(), formatTime(normalized.CreatedAt())); err != nil {
		return fmt.Errorf("create engineering execution item: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit engineering execution item: %w", err)
	}
	return nil
}

func (s *Store) GetExecutionItem(ctx context.Context, workspaceID, id string) (domain.ExecutionItem, error) {
	var kind, sourceType, sourceID, sourceLocator, createdAt string
	if err := s.db.QueryRowContext(ctx, `SELECT kind,source_type,source_id,source_locator,created_at
        FROM engineering_execution_items WHERE workspace_id=? AND id=?`, workspaceID, id).
		Scan(&kind, &sourceType, &sourceID, &sourceLocator, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ExecutionItem{}, domain.ErrNotFound
		}
		return domain.ExecutionItem{}, fmt.Errorf("get engineering execution item: %w", err)
	}
	created, err := parseTime(createdAt)
	if err != nil {
		return domain.ExecutionItem{}, fmt.Errorf("parse engineering execution item created_at: %w", err)
	}
	value, err := domain.NewExecutionItem(id, workspaceID, domain.ExecutionItemKind(kind), sourceType, sourceID, sourceLocator, created)
	if err != nil {
		return domain.ExecutionItem{}, fmt.Errorf("rehydrate engineering execution item: %w", err)
	}
	return value, nil
}

func (s *Store) CreateEvidence(ctx context.Context, value domain.EvidenceEnvelope) error {
	normalized, err := domain.RehydrateEvidenceEnvelope(
		value.SchemaVersion(), value.ID(), value.WorkspaceID(), value.Kind(), value.Outcome(), value.Source(), value.ProducerID(),
		value.Artifact(), value.Payload(), value.CapturedAt(), value.ContentChecksum(),
	)
	if err != nil {
		return err
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin create engineering evidence: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var existingChecksum, existingObservedAt, existingCapturedAt string
	err = tx.QueryRowContext(ctx, `SELECT content_checksum,source_observed_at,captured_at
        FROM engineering_evidence WHERE workspace_id=? AND id=?`, normalized.WorkspaceID(), normalized.ID()).
		Scan(&existingChecksum, &existingObservedAt, &existingCapturedAt)
	if err == nil {
		if existingChecksum == normalized.ContentChecksum() && existingObservedAt == formatTime(normalized.Source().ObservedAt()) &&
			existingCapturedAt == formatTime(normalized.CapturedAt()) {
			return nil
		}
		return domain.ErrConflict
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("check engineering evidence identity: %w", err)
	}

	artifactURI, artifactDigest := "", ""
	if artifact := normalized.Artifact(); artifact != nil {
		artifactURI = artifact.URI()
		artifactDigest = artifact.Digest()
	}
	source := normalized.Source()
	if _, err := tx.ExecContext(ctx, `INSERT INTO engineering_evidence(
        workspace_id,id,schema_version,kind,outcome,source_type,source_id,source_locator,source_revision,source_digest,
        source_observed_at,producer_id,artifact_uri,artifact_digest,payload_json,captured_at,content_checksum
    ) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		normalized.WorkspaceID(), normalized.ID(), normalized.SchemaVersion(), string(normalized.Kind()), string(normalized.Outcome()),
		source.Type(), source.ID(), source.Locator(), source.Revision(), source.Digest(), formatTime(source.ObservedAt()), normalized.ProducerID(),
		artifactURI, artifactDigest, string(normalized.Payload()), formatTime(normalized.CapturedAt()), normalized.ContentChecksum()); err != nil {
		return fmt.Errorf("create engineering evidence: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit engineering evidence: %w", err)
	}
	return nil
}

func (s *Store) GetEvidence(ctx context.Context, workspaceID, id string) (domain.EvidenceEnvelope, error) {
	var schemaVersion, kind, outcome string
	var sourceType, sourceID, sourceLocator, sourceRevision, sourceDigest, sourceObservedAt string
	var producerID, artifactURI, artifactDigest, payloadJSON, capturedAt, checksum string
	if err := s.db.QueryRowContext(ctx, `SELECT schema_version,kind,outcome,source_type,source_id,source_locator,source_revision,source_digest,
        source_observed_at,producer_id,artifact_uri,artifact_digest,payload_json,captured_at,content_checksum
        FROM engineering_evidence WHERE workspace_id=? AND id=?`, workspaceID, id).Scan(
		&schemaVersion, &kind, &outcome, &sourceType, &sourceID, &sourceLocator, &sourceRevision, &sourceDigest,
		&sourceObservedAt, &producerID, &artifactURI, &artifactDigest, &payloadJSON, &capturedAt, &checksum,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.EvidenceEnvelope{}, domain.ErrNotFound
		}
		return domain.EvidenceEnvelope{}, fmt.Errorf("get engineering evidence: %w", err)
	}

	observed, err := parseTime(sourceObservedAt)
	if err != nil {
		return domain.EvidenceEnvelope{}, fmt.Errorf("parse engineering evidence source_observed_at: %w", err)
	}
	captured, err := parseTime(capturedAt)
	if err != nil {
		return domain.EvidenceEnvelope{}, fmt.Errorf("parse engineering evidence captured_at: %w", err)
	}
	source, err := domain.NewEvidenceSource(sourceType, sourceID, sourceLocator, sourceRevision, sourceDigest, observed)
	if err != nil {
		return domain.EvidenceEnvelope{}, fmt.Errorf("rehydrate engineering evidence source: %w", err)
	}
	var artifact *domain.EvidenceArtifact
	if artifactURI != "" || artifactDigest != "" {
		value, err := domain.NewEvidenceArtifact(artifactURI, artifactDigest)
		if err != nil {
			return domain.EvidenceEnvelope{}, fmt.Errorf("rehydrate engineering evidence artifact: %w", err)
		}
		artifact = &value
	}
	value, err := domain.RehydrateEvidenceEnvelope(
		schemaVersion, id, workspaceID, domain.EvidenceKind(kind), domain.EvidenceOutcome(outcome), source, producerID, artifact,
		json.RawMessage(payloadJSON), captured, checksum,
	)
	if err != nil {
		return domain.EvidenceEnvelope{}, fmt.Errorf("rehydrate engineering evidence: %w", err)
	}
	return value, nil
}

func (s *Store) AttachEvidence(ctx context.Context, value domain.EvidenceAttachment) error {
	normalized, err := domain.NewEvidenceAttachment(value.WorkspaceID(), value.ExecutionItemID(), value.EvidenceID(), value.AttachedAt())
	if err != nil {
		return err
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin attach engineering evidence: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := requireExecutionEvidenceRecord(ctx, tx, `SELECT 1 FROM engineering_execution_items WHERE workspace_id=? AND id=?`,
		normalized.WorkspaceID(), normalized.ExecutionItemID()); err != nil {
		return err
	}
	if err := requireExecutionEvidenceRecord(ctx, tx, `SELECT 1 FROM engineering_evidence WHERE workspace_id=? AND id=?`,
		normalized.WorkspaceID(), normalized.EvidenceID()); err != nil {
		return err
	}

	var attachedAt string
	err = tx.QueryRowContext(ctx, `SELECT attached_at FROM engineering_execution_item_evidence
        WHERE workspace_id=? AND execution_item_id=? AND evidence_id=?`, normalized.WorkspaceID(), normalized.ExecutionItemID(), normalized.EvidenceID()).
		Scan(&attachedAt)
	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("check engineering evidence attachment: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO engineering_execution_item_evidence(
        workspace_id,execution_item_id,evidence_id,attached_at
    ) VALUES(?,?,?,?)`, normalized.WorkspaceID(), normalized.ExecutionItemID(), normalized.EvidenceID(), formatTime(normalized.AttachedAt())); err != nil {
		return fmt.Errorf("attach engineering evidence: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit engineering evidence attachment: %w", err)
	}
	return nil
}

func (s *Store) ListEvidenceAttachments(ctx context.Context, workspaceID, executionItemID string) ([]domain.EvidenceAttachment, error) {
	var exists int
	if err := s.db.QueryRowContext(ctx, `SELECT 1 FROM engineering_execution_items WHERE workspace_id=? AND id=?`, workspaceID, executionItemID).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("check engineering execution item for attachments: %w", err)
	}

	rows, err := s.db.QueryContext(ctx, `SELECT evidence_id,attached_at FROM engineering_execution_item_evidence
        WHERE workspace_id=? AND execution_item_id=? ORDER BY attached_at,evidence_id`, workspaceID, executionItemID)
	if err != nil {
		return nil, fmt.Errorf("list engineering evidence attachments: %w", err)
	}
	defer rows.Close()

	var values []domain.EvidenceAttachment
	for rows.Next() {
		var evidenceID, attachedAt string
		if err := rows.Scan(&evidenceID, &attachedAt); err != nil {
			return nil, fmt.Errorf("scan engineering evidence attachment: %w", err)
		}
		attached, err := parseTime(attachedAt)
		if err != nil {
			return nil, fmt.Errorf("parse engineering evidence attached_at: %w", err)
		}
		value, err := domain.NewEvidenceAttachment(workspaceID, executionItemID, evidenceID, attached)
		if err != nil {
			return nil, fmt.Errorf("rehydrate engineering evidence attachment: %w", err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate engineering evidence attachments: %w", err)
	}
	return values, nil
}

func requireExecutionEvidenceRecord(ctx context.Context, tx *sql.Tx, query string, args ...any) error {
	var exists int
	if err := tx.QueryRowContext(ctx, query, args...).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("check engineering evidence attachment reference: %w", err)
	}
	return nil
}
