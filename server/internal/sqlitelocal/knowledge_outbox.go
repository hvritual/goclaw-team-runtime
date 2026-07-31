package sqlitelocal

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/multica-ai/multica/server/internal/knowledge"
	"github.com/multica-ai/multica/server/internal/knowledge/outbox"
)

type contextExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

type sqliteEvidenceOutbox struct {
	db *sql.DB
}

func enqueueKnowledgeEvidence(
	ctx context.Context,
	executor contextExecutor,
	evidence knowledge.Evidence,
) error {
	payload, err := json.Marshal(evidence)
	if err != nil {
		return err
	}
	timestamp := now()
	_, err = executor.ExecContext(ctx, `
		INSERT OR IGNORE INTO knowledge_evidence_outbox(
			id, workspace_id, evidence_id, idempotency_key, payload_json,
			available_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		newID(),
		evidence.WorkspaceID,
		evidence.ID,
		evidence.IdempotencyKey,
		string(payload),
		timestamp,
		timestamp,
		timestamp,
	)
	return err
}

func (s *sqliteEvidenceOutbox) NextBatch(ctx context.Context, limit int) ([]outbox.Message, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, workspace_id, payload_json, attempts, created_at
		FROM knowledge_evidence_outbox
		WHERE delivered_at IS NULL AND available_at <= ?
		ORDER BY created_at, id
		LIMIT ?`,
		now(),
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	messages := make([]outbox.Message, 0)
	for rows.Next() {
		var (
			message   outbox.Message
			payload   string
			createdAt string
		)
		if err := rows.Scan(
			&message.ID,
			&message.WorkspaceID,
			&payload,
			&message.Attempts,
			&createdAt,
		); err != nil {
			return nil, err
		}
		message.Payload = []byte(payload)
		message.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	return messages, rows.Err()
}

func (s *sqliteEvidenceOutbox) MarkDelivered(ctx context.Context, id string, deliveredAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE knowledge_evidence_outbox
		SET delivered_at = ?, last_error = '', updated_at = ?
		WHERE id = ? AND delivered_at IS NULL`,
		deliveredAt.UTC().Format(time.RFC3339Nano),
		deliveredAt.UTC().Format(time.RFC3339Nano),
		id,
	)
	return err
}

func (s *sqliteEvidenceOutbox) MarkFailed(
	ctx context.Context,
	id string,
	failedAt time.Time,
	deliveryError error,
) error {
	message := strings.TrimSpace(deliveryError.Error())
	if len(message) > 500 {
		message = message[:500]
	}
	retryAt := failedAt.UTC().Add(5 * time.Second).Format(time.RFC3339Nano)
	_, err := s.db.ExecContext(ctx, `
		UPDATE knowledge_evidence_outbox
		SET attempts = attempts + 1, available_at = ?, last_error = ?, updated_at = ?
		WHERE id = ? AND delivered_at IS NULL`,
		retryAt,
		message,
		failedAt.UTC().Format(time.RFC3339Nano),
		id,
	)
	return err
}

func (s *Server) startKnowledgeDispatcher() {
	ctx, cancel := context.WithCancel(context.Background())
	s.knowledgeCancel = cancel
	s.knowledgeDone = make(chan struct{})
	go func() {
		defer close(s.knowledgeDone)
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			_, err := s.knowledgeDispatcher.Drain(ctx, 50)
			s.recordKnowledgeDispatch(err)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func (s *Server) dispatchKnowledgeEvidence(ctx context.Context) {
	if s.knowledgeDispatcher == nil {
		return
	}
	_, err := s.knowledgeDispatcher.Drain(ctx, 50)
	s.recordKnowledgeDispatch(err)
}

func (s *Server) recordKnowledgeDispatch(err error) {
	s.knowledgeDispatchMu.Lock()
	defer s.knowledgeDispatchMu.Unlock()
	s.knowledgeDispatchError = ""
	if err != nil {
		s.knowledgeDispatchError = err.Error()
	}
}

func (s *Server) knowledgeDispatchStatus() string {
	s.knowledgeDispatchMu.RLock()
	defer s.knowledgeDispatchMu.RUnlock()
	return s.knowledgeDispatchError
}

func (s *Server) knowledgeOutboxStats(ctx context.Context, workspaceID string) (map[string]any, error) {
	var pending, failed int
	var lastDelivered sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT
			COUNT(CASE WHEN delivered_at IS NULL THEN 1 END),
			COUNT(CASE WHEN delivered_at IS NULL AND attempts > 0 THEN 1 END),
			MAX(delivered_at)
		FROM knowledge_evidence_outbox
		WHERE workspace_id = ?`,
		workspaceID,
	).Scan(&pending, &failed, &lastDelivered)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"pending":           pending,
		"failed":            failed,
		"last_delivered_at": nullable(lastDelivered.String),
		"last_error":        nullable(s.knowledgeDispatchStatus()),
	}, nil
}

func evidenceSourceRef(sourceType, sourceID, revision string) []knowledge.SourceRef {
	return []knowledge.SourceRef{{
		Type:     sourceType,
		ID:       sourceID,
		Revision: revision,
		URI:      "multica://" + sourceType + "s/" + sourceID,
	}}
}

func optionalString(value sql.NullString) string {
	if value.Valid {
		return value.String
	}
	return ""
}

var errKnowledgeDisabled = errors.New("knowledge is disabled")

func projectEvidence(value project, actorID, eventType string) knowledge.Evidence {
	content := value.Title
	if value.Description.Valid && strings.TrimSpace(value.Description.String) != "" {
		content += "\n\n" + value.Description.String
	}
	return newKnowledgeEvidence(
		value.WorkspaceID,
		value.ID,
		value.ID,
		value.UpdatedAt,
		eventType,
		knowledge.KindGoal,
		value.Title,
		content,
		actorID,
		value.Status == "completed" || value.Status == "done",
	)
}

func issueEvidence(value issue, actorID, eventType string) knowledge.Evidence {
	content := value.Title
	if value.Description.Valid && strings.TrimSpace(value.Description.String) != "" {
		content += "\n\n" + value.Description.String
	}
	return newKnowledgeEvidence(
		value.WorkspaceID,
		optionalString(value.ProjectID),
		value.ID,
		value.UpdatedAt,
		eventType,
		knowledge.KindRequirement,
		value.Title,
		content,
		actorID,
		value.Status == "done",
	)
}

func taskEvidence(value task, actorID, eventType string) knowledge.Evidence {
	content := value.Title
	if strings.TrimSpace(value.Description) != "" {
		content += "\n\n" + value.Description
	}
	return newKnowledgeEvidence(
		value.WorkspaceID,
		optionalString(value.ProjectID),
		value.ID,
		value.UpdatedAt,
		eventType,
		knowledge.KindReference,
		value.Title,
		content,
		actorID,
		value.Status == "done" || value.Status == "cancelled",
	)
}

func newKnowledgeEvidence(
	workspaceID string,
	projectID string,
	sourceID string,
	revision string,
	eventType string,
	kind knowledge.Kind,
	title string,
	content string,
	actorID string,
	terminal bool,
) knowledge.Evidence {
	sourceType := strings.SplitN(eventType, ".", 2)[0]
	occurredAt, err := time.Parse(time.RFC3339Nano, revision)
	if err != nil {
		occurredAt = time.Now().UTC()
	}
	sum := sha256.Sum256([]byte(content))
	return knowledge.Evidence{
		ID:             newID(),
		WorkspaceID:    workspaceID,
		ProjectID:      projectID,
		SourceType:     sourceType,
		SourceID:       sourceID,
		SourceRevision: revision,
		EventType:      eventType,
		Kind:           kind,
		Title:          title,
		Content:        content,
		ActorID:        actorID,
		IdempotencyKey: fmt.Sprintf("%s:%s:%s", sourceID, revision, eventType),
		ProvenanceURI:  "multica://" + sourceType + "s/" + sourceID,
		Checksum:       fmt.Sprintf("sha256:%x", sum),
		OccurredAt:     occurredAt,
		Terminal:       terminal,
		Validated:      true,
		Confidence:     1,
		SourceRefs:     evidenceSourceRef(sourceType, sourceID, revision),
	}
}
