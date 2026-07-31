package main

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/multica-ai/multica/server/internal/knowledge/outbox"
	"github.com/multica-ai/multica/server/internal/util"
)

type postgresEvidenceOutbox struct {
	pool *pgxpool.Pool
}

func (store *postgresEvidenceOutbox) NextBatch(
	ctx context.Context,
	limit int,
) ([]outbox.Message, error) {
	rows, err := store.pool.Query(ctx, `
		SELECT id, workspace_id, payload_json, attempts, created_at
		FROM knowledge_evidence_outbox
		WHERE delivered_at IS NULL AND available_at <= now()
		ORDER BY created_at, id
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	messages := make([]outbox.Message, 0)
	for rows.Next() {
		var id, workspaceID pgtype.UUID
		var message outbox.Message
		if err := rows.Scan(
			&id,
			&workspaceID,
			&message.Payload,
			&message.Attempts,
			&message.CreatedAt,
		); err != nil {
			return nil, err
		}
		message.ID = util.UUIDToString(id)
		message.WorkspaceID = util.UUIDToString(workspaceID)
		messages = append(messages, message)
	}
	return messages, rows.Err()
}

func (store *postgresEvidenceOutbox) MarkDelivered(
	ctx context.Context,
	id string,
	deliveredAt time.Time,
) error {
	_, err := store.pool.Exec(ctx, `
		UPDATE knowledge_evidence_outbox
		SET delivered_at = $2, last_error = '', updated_at = $2
		WHERE id = $1 AND delivered_at IS NULL`, id, deliveredAt.UTC())
	return err
}

func (store *postgresEvidenceOutbox) MarkFailed(
	ctx context.Context,
	id string,
	failedAt time.Time,
	deliveryError error,
) error {
	message := strings.TrimSpace(deliveryError.Error())
	if len(message) > 500 {
		message = message[:500]
	}
	_, err := store.pool.Exec(ctx, `
		UPDATE knowledge_evidence_outbox
		SET attempts = attempts + 1,
		    available_at = $2,
		    last_error = $3,
		    updated_at = $4
		WHERE id = $1 AND delivered_at IS NULL`,
		id, failedAt.UTC().Add(5*time.Second), message, failedAt.UTC())
	return err
}
