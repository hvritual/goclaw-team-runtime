package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/hvritual/workspace/internal/modules/engineering/internal/domain"
)

func (s *Store) PutEvidence(ctx context.Context, value domain.EvidenceEnvelope) error {
	source := value.Source()
	subject := value.Subject()
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO engineering_evidence_envelopes(
		workspace_id,id,kind,subject_kind,subject_id,source_type,source_locator,source_revision,source_digest,
		source_observed_at,producer_id,artifact_uri,artifact_digest,captured_at,content_checksum
	) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		value.WorkspaceID(), value.ID(), string(value.Kind()), string(subject.Kind()), subject.ID(),
		source.SourceType(), source.Locator(), source.Revision(), source.Digest(), formatTime(source.ObservedAt()),
		value.ProducerID(), value.ArtifactURI(), value.ArtifactDigest(), formatTime(value.CapturedAt()), value.ContentChecksum(),
	)
	if err != nil {
		return fmt.Errorf("put engineering evidence: %w", err)
	}
	var checksum string
	if err := s.db.QueryRowContext(ctx, `SELECT content_checksum FROM engineering_evidence_envelopes WHERE workspace_id=? AND id=?`, value.WorkspaceID(), value.ID()).Scan(&checksum); err != nil {
		return fmt.Errorf("verify engineering evidence: %w", err)
	}
	if checksum != value.ContentChecksum() {
		return domain.ErrConflict
	}
	return nil
}

func (s *Store) GetEvidence(ctx context.Context, workspaceID, id string) (domain.EvidenceEnvelope, error) {
	row := s.db.QueryRowContext(ctx, `SELECT kind,subject_kind,subject_id,source_type,source_locator,source_revision,source_digest,
		source_observed_at,producer_id,artifact_uri,artifact_digest,captured_at,content_checksum
		FROM engineering_evidence_envelopes WHERE workspace_id=? AND id=?`, workspaceID, id)
	value, err := scanEvidence(row, workspaceID, id)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.EvidenceEnvelope{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.EvidenceEnvelope{}, fmt.Errorf("get engineering evidence: %w", err)
	}
	return value, nil
}

func (s *Store) ListEvidence(ctx context.Context, workspaceID string, subject *domain.NodeRef) ([]domain.EvidenceEnvelope, error) {
	query := `SELECT id,kind,subject_kind,subject_id,source_type,source_locator,source_revision,source_digest,
		source_observed_at,producer_id,artifact_uri,artifact_digest,captured_at,content_checksum
		FROM engineering_evidence_envelopes WHERE workspace_id=?`
	args := []any{workspaceID}
	if subject != nil {
		query += ` AND subject_kind=? AND subject_id=?`
		args = append(args, string(subject.Kind()), subject.ID())
	}
	query += ` ORDER BY id`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list engineering evidence: %w", err)
	}
	defer rows.Close()
	var values []domain.EvidenceEnvelope
	for rows.Next() {
		var id string
		var kind, subjectKind, subjectID, sourceType, locator, revision, digest string
		var observedAt, producerID, artifactURI, artifactDigest, capturedAt, checksum string
		if err := rows.Scan(&id, &kind, &subjectKind, &subjectID, &sourceType, &locator, &revision, &digest, &observedAt, &producerID, &artifactURI, &artifactDigest, &capturedAt, &checksum); err != nil {
			return nil, fmt.Errorf("scan engineering evidence: %w", err)
		}
		value, err := buildEvidence(workspaceID, id, kind, subjectKind, subjectID, sourceType, locator, revision, digest, observedAt, producerID, artifactURI, artifactDigest, capturedAt, checksum)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate engineering evidence: %w", err)
	}
	return values, nil
}

func scanEvidence(row scanner, workspaceID, id string) (domain.EvidenceEnvelope, error) {
	var kind, subjectKind, subjectID, sourceType, locator, revision, digest string
	var observedAt, producerID, artifactURI, artifactDigest, capturedAt, checksum string
	if err := row.Scan(&kind, &subjectKind, &subjectID, &sourceType, &locator, &revision, &digest, &observedAt, &producerID, &artifactURI, &artifactDigest, &capturedAt, &checksum); err != nil {
		return domain.EvidenceEnvelope{}, err
	}
	return buildEvidence(workspaceID, id, kind, subjectKind, subjectID, sourceType, locator, revision, digest, observedAt, producerID, artifactURI, artifactDigest, capturedAt, checksum)
}

func buildEvidence(workspaceID, id, kind, subjectKind, subjectID, sourceType, locator, revision, digest, observedAt, producerID, artifactURI, artifactDigest, capturedAt, checksum string) (domain.EvidenceEnvelope, error) {
	subject, err := domain.NewNodeRef(domain.NodeKind(subjectKind), subjectID)
	if err != nil {
		return domain.EvidenceEnvelope{}, fmt.Errorf("rehydrate evidence subject: %w", err)
	}
	observed, err := parseTime(observedAt)
	if err != nil {
		return domain.EvidenceEnvelope{}, fmt.Errorf("parse evidence observed_at: %w", err)
	}
	source, err := domain.NewEvidenceSource(sourceType, locator, revision, digest, observed)
	if err != nil {
		return domain.EvidenceEnvelope{}, fmt.Errorf("rehydrate evidence source: %w", err)
	}
	captured, err := parseTime(capturedAt)
	if err != nil {
		return domain.EvidenceEnvelope{}, fmt.Errorf("parse evidence captured_at: %w", err)
	}
	value, err := domain.RehydrateEvidenceEnvelope(id, workspaceID, domain.EvidenceKind(kind), subject, source, producerID, artifactURI, artifactDigest, captured, checksum)
	if err != nil {
		return domain.EvidenceEnvelope{}, fmt.Errorf("rehydrate evidence: %w", err)
	}
	return value, nil
}
