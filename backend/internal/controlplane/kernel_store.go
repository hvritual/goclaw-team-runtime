package controlplane

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (r *sqlRepository) AppendCommand(ctx context.Context, command CommandEnvelope, proposed []ProposedEvent) (AppendResult, error) {
	const op = "append kernel command"
	if err := validateKernelEnvelope(command); err != nil {
		return AppendResult{}, err
	}
	requestHash := sha256.Sum256(command.Request)
	requestDigest := hex.EncodeToString(requestHash[:])
	var result AppendResult
	err := r.transaction(ctx, func(tx *sql.Tx) error {
		if err := r.lockKernelCommand(ctx, tx, command); err != nil {
			return err
		}
		storedHash, storedResult, found, err := r.loadCommand(ctx, tx, command)
		if err != nil {
			return err
		}
		if found {
			if storedHash != requestDigest {
				return conflict(op, "command_id was already used with a different request")
			}
			if err := json.Unmarshal([]byte(storedResult), &result); err != nil {
				return invariant(op, "stored command result is invalid")
			}
			result.Replayed = true
			return nil
		}
		if len(proposed) == 0 {
			return invalid(op, "events", "at least one event is required")
		}
		if _, err := tx.ExecContext(ctx, r.bind(`INSERT INTO cp_project_heads
            (workspace_id, project_id, head_seq, head_hash) VALUES (?, ?, 0, '')
            ON CONFLICT (workspace_id, project_id) DO NOTHING`), command.WorkspaceID, command.ProjectID); err != nil {
			return fmt.Errorf("create project head: %w", err)
		}
		headQuery := `SELECT head_seq, head_hash FROM cp_project_heads WHERE workspace_id = ? AND project_id = ?`
		if r.dialect == DialectPostgres {
			headQuery += ` FOR UPDATE`
		}
		var head int64
		var headHash string
		if err := tx.QueryRowContext(ctx, r.bind(headQuery), command.WorkspaceID, command.ProjectID).Scan(&head, &headHash); err != nil {
			return fmt.Errorf("load project head: %w", err)
		}
		if head != command.ExpectedHead {
			return conflict(op, "project head changed")
		}
		previousHash := headHash
		for index, candidate := range proposed {
			if candidate.Type == "" || !json.Valid(candidate.Payload) {
				return invalid(op, "event", "type and valid JSON payload are required")
			}
			event := SessionEvent{
				SchemaVersion: kernelSchemaVersion,
				WorkspaceID:   command.WorkspaceID,
				ProjectID:     command.ProjectID,
				Sequence:      head + int64(index) + 1,
				EventID:       uuid.NewString(),
				CommandID:     command.CommandID,
				Type:          candidate.Type,
				ActorID:       command.Actor.ID,
				ActorKind:     command.Actor.Kind,
				Payload:       append(json.RawMessage(nil), candidate.Payload...),
				PreviousHash:  previousHash,
				OccurredAt:    candidate.OccurredAt.UTC(),
			}
			event.Hash = hashSessionEvent(event)
			if event.Hash == "" {
				return invariant(op, "project head hash is invalid")
			}
			if _, err := tx.ExecContext(ctx, r.bind(`INSERT INTO cp_session_events
                (workspace_id, project_id, seq, event_id, command_id, schema_version, type, actor_id, actor_kind,
                 payload, previous_hash, hash, occurred_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`),
				event.WorkspaceID, event.ProjectID, event.Sequence, event.EventID, event.CommandID, event.SchemaVersion,
				event.Type, event.ActorID, event.ActorKind, string(event.Payload), event.PreviousHash, event.Hash,
				formatTime(event.OccurredAt)); err != nil {
				return fmt.Errorf("insert session event: %w", err)
			}
			result.Events = append(result.Events, event)
			previousHash = event.Hash
		}
		newHead := head + int64(len(result.Events))
		updated, err := tx.ExecContext(ctx, r.bind(`UPDATE cp_project_heads SET head_seq = ?, head_hash = ?
            WHERE workspace_id = ? AND project_id = ? AND head_seq = ? AND head_hash = ?`),
			newHead, previousHash, command.WorkspaceID, command.ProjectID, head, headHash)
		if err != nil {
			return fmt.Errorf("update project head: %w", err)
		}
		if err := requireChanged(updated, "update project head"); err != nil {
			return err
		}
		result.Head, result.HeadHash = newHead, previousHash
		encodedResult, err := json.Marshal(result)
		if err != nil {
			return fmt.Errorf("encode command result: %w", err)
		}
		if _, err := tx.ExecContext(ctx, r.bind(`INSERT INTO cp_kernel_commands
            (workspace_id, project_id, command_id, command_name, request_hash, result_json, created_at)
            VALUES (?, ?, ?, ?, ?, ?, ?)`), command.WorkspaceID, command.ProjectID, command.CommandID,
			command.Name, requestDigest, string(encodedResult), formatTime(result.Events[0].OccurredAt)); err != nil {
			return fmt.Errorf("insert kernel command: %w", err)
		}
		return nil
	})
	return result, err
}

func (r *sqlRepository) lockKernelCommand(ctx context.Context, tx *sql.Tx, command CommandEnvelope) error {
	if r.dialect != DialectPostgres {
		return nil
	}
	key := command.WorkspaceID + "/" + command.ProjectID + "/" + command.CommandID
	if _, err := tx.ExecContext(ctx, r.bind(`SELECT pg_advisory_xact_lock(hashtextextended(?, 0))`), key); err != nil {
		return fmt.Errorf("lock kernel command: %w", err)
	}
	return nil
}

func (r *sqlRepository) loadCommand(ctx context.Context, tx *sql.Tx, command CommandEnvelope) (string, string, bool, error) {
	row := tx.QueryRowContext(ctx, r.bind(`SELECT request_hash, result_json FROM cp_kernel_commands
        WHERE workspace_id = ? AND project_id = ? AND command_id = ?`), command.WorkspaceID, command.ProjectID, command.CommandID)
	var requestHash, result string
	if err := row.Scan(&requestHash, &result); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", false, nil
		}
		return "", "", false, fmt.Errorf("load kernel command: %w", err)
	}
	return requestHash, result, true, nil
}

func (r *sqlRepository) ProjectHead(ctx context.Context, workspaceID, projectID string) (int64, string, error) {
	row := r.db.QueryRowContext(ctx, r.bind(`SELECT head_seq, head_hash FROM cp_project_heads
        WHERE workspace_id = ? AND project_id = ?`), workspaceID, projectID)
	var head int64
	var hash string
	if err := row.Scan(&head, &hash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, "", nil
		}
		return 0, "", fmt.Errorf("load project head: %w", err)
	}
	return head, hash, nil
}

func (r *sqlRepository) ListSessionEvents(ctx context.Context, workspaceID, projectID string) ([]SessionEvent, error) {
	rows, err := r.db.QueryContext(ctx, r.bind(`SELECT schema_version, workspace_id, project_id, seq, event_id,
        command_id, type, actor_id, actor_kind, payload, previous_hash, hash, occurred_at
        FROM cp_session_events WHERE workspace_id = ? AND project_id = ? ORDER BY seq`), workspaceID, projectID)
	if err != nil {
		return nil, fmt.Errorf("list session events: %w", err)
	}
	defer rows.Close()
	var events []SessionEvent
	for rows.Next() {
		var event SessionEvent
		var payload, occurred string
		if err := rows.Scan(&event.SchemaVersion, &event.WorkspaceID, &event.ProjectID, &event.Sequence, &event.EventID,
			&event.CommandID, &event.Type, &event.ActorID, &event.ActorKind, &payload, &event.PreviousHash,
			&event.Hash, &occurred); err != nil {
			return nil, fmt.Errorf("scan session event: %w", err)
		}
		parsed, err := time.Parse(time.RFC3339Nano, occurred)
		if err != nil {
			return nil, invariant("list session events", "invalid occurred_at")
		}
		event.Payload, event.OccurredAt = json.RawMessage(payload), parsed.UTC()
		events = append(events, event)
	}
	return events, rows.Err()
}

func (r *sqlRepository) ListSessionEventsAfter(ctx context.Context, workspaceID, projectID string, after int64, limit int) ([]SessionEvent, error) {
	if after < 0 || limit < 1 || limit > 1000 {
		return nil, invalid("list session events after", "cursor", "requires a non-negative cursor and a bounded limit")
	}
	rows, err := r.db.QueryContext(ctx, r.bind(`SELECT schema_version, workspace_id, project_id, seq, event_id,
        command_id, type, actor_id, actor_kind, payload, previous_hash, hash, occurred_at
        FROM cp_session_events WHERE workspace_id = ? AND project_id = ? AND seq > ? ORDER BY seq LIMIT ?`), workspaceID, projectID, after, limit)
	if err != nil {
		return nil, fmt.Errorf("list session events after: %w", err)
	}
	defer rows.Close()
	var events []SessionEvent
	for rows.Next() {
		var event SessionEvent
		var payload, occurred string
		if err := rows.Scan(&event.SchemaVersion, &event.WorkspaceID, &event.ProjectID, &event.Sequence, &event.EventID,
			&event.CommandID, &event.Type, &event.ActorID, &event.ActorKind, &payload, &event.PreviousHash,
			&event.Hash, &occurred); err != nil {
			return nil, fmt.Errorf("scan session event after: %w", err)
		}
		parsed, err := time.Parse(time.RFC3339Nano, occurred)
		if err != nil {
			return nil, invariant("list session events after", "invalid occurred_at")
		}
		event.Payload, event.OccurredAt = json.RawMessage(payload), parsed.UTC()
		events = append(events, event)
	}
	return events, rows.Err()
}

func hashSessionEvent(event SessionEvent) string {
	hash := sha256.New()
	writeHashField(hash, []byte("goclaw/session-event/v1"))
	for _, value := range []string{event.WorkspaceID, event.ProjectID, fmt.Sprint(event.Sequence), event.EventID,
		event.CommandID, event.Type, event.ActorID, string(event.ActorKind), event.OccurredAt.UTC().Format(time.RFC3339Nano)} {
		writeHashField(hash, []byte(value))
	}
	payloadHash := sha256.Sum256(event.Payload)
	writeHashField(hash, payloadHash[:])
	previous := make([]byte, sha256.Size)
	if event.PreviousHash != "" {
		decoded, err := hex.DecodeString(event.PreviousHash)
		if err != nil || len(decoded) != sha256.Size {
			return ""
		}
		previous = decoded
	}
	writeHashField(hash, previous)
	return hex.EncodeToString(hash.Sum(nil))
}

type hashWriter interface{ Write([]byte) (int, error) }

func writeHashField(target hashWriter, value []byte) {
	length := make([]byte, 8)
	binary.BigEndian.PutUint64(length, uint64(len(value)))
	_, _ = target.Write(length)
	_, _ = target.Write(value)
}

func validateKernelEnvelope(command CommandEnvelope) error {
	const op = "validate kernel command"
	for field, value := range map[string]string{"workspace_id": command.WorkspaceID, "project_id": command.ProjectID, "command_id": command.CommandID} {
		if err := validateIdentifier(op, field, value); err != nil {
			return err
		}
	}
	if command.Actor.WorkspaceID != command.WorkspaceID {
		return denied(op, "actor workspace does not match command workspace")
	}
	if command.Name == "" || !json.Valid(command.Request) || command.ExpectedHead < 0 {
		return invalid(op, "command", "name, valid request JSON, and a non-negative expected head are required")
	}
	return validateActor(command.Actor, true)
}

func canonicalKernelRequest(value any) (json.RawMessage, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(strings.TrimSpace(string(encoded))), nil
}
