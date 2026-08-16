package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
)

type DomainMutation func(context.Context, *sql.Conn) error

type GovernancePhase string

const (
	GovernanceAfterDomain   GovernancePhase = "after_domain"
	GovernanceAfterRevision GovernancePhase = "after_revision"
	GovernanceAfterAudit    GovernancePhase = "after_audit"
	GovernanceAfterOutbox   GovernancePhase = "after_outbox"
	GovernanceAfterReplay   GovernancePhase = "after_replay"
)

type GovernanceRepositoryOption func(*GovernanceRepository)

func WithGovernanceFailureHook(hook func(GovernancePhase) error) GovernanceRepositoryOption {
	return func(repository *GovernanceRepository) {
		repository.failureHook = hook
	}
}

type GovernanceRepository struct {
	db          *sql.DB
	failureHook func(GovernancePhase) error
}

func NewGovernanceRepository(config Config, options ...GovernanceRepositoryOption) (*GovernanceRepository, error) {
	if config.DB == nil {
		return nil, errors.New("workspace sqlite database is required")
	}
	repository := &GovernanceRepository{db: config.DB}
	for _, option := range options {
		if option != nil {
			option(repository)
		}
	}
	return repository, nil
}

func (r *GovernanceRepository) Execute(ctx context.Context, prepared application.PreparedGovernanceMutation, apply DomainMutation) (result contract.MutationResult, err error) {
	if apply == nil {
		return contract.MutationResult{}, fmt.Errorf("%w: domain mutation is required", contract.ErrInvalidGovernanceMutation)
	}
	if err := validatePreparedGovernanceMutation(prepared); err != nil {
		return contract.MutationResult{}, err
	}

	connection, err := r.db.Conn(ctx)
	if err != nil {
		return contract.MutationResult{}, fmt.Errorf("acquire governance connection: %w", err)
	}
	defer connection.Close()
	if _, err := connection.ExecContext(ctx, `PRAGMA busy_timeout = 5000`); err != nil {
		return contract.MutationResult{}, fmt.Errorf("configure governance connection: %w", err)
	}
	if _, err := connection.ExecContext(ctx, `BEGIN IMMEDIATE`); err != nil {
		return contract.MutationResult{}, fmt.Errorf("begin governed mutation: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = connection.ExecContext(context.WithoutCancel(ctx), `ROLLBACK`)
		}
	}()

	if prepared.Command.IdempotencyKey != "" {
		replayed, found, err := loadGovernanceReplay(ctx, connection, prepared)
		if err != nil {
			return contract.MutationResult{}, err
		}
		if found {
			return replayed, nil
		}
	}

	currentRevision, err := loadResourceRevision(ctx, connection, prepared)
	if err != nil {
		return contract.MutationResult{}, err
	}
	if currentRevision != prepared.Command.ExpectedRevision {
		return contract.MutationResult{}, contract.RevisionConflictError{CurrentRevision: currentRevision}
	}
	if err := apply(ctx, connection); err != nil {
		return contract.MutationResult{}, fmt.Errorf("apply governed domain mutation: %w", err)
	}
	if err := r.afterPhase(GovernanceAfterDomain); err != nil {
		return contract.MutationResult{}, err
	}
	if err := persistResourceRevision(ctx, connection, prepared); err != nil {
		return contract.MutationResult{}, err
	}
	if err := r.afterPhase(GovernanceAfterRevision); err != nil {
		return contract.MutationResult{}, err
	}
	if err := persistAuditRecord(ctx, connection, prepared.Audit); err != nil {
		return contract.MutationResult{}, err
	}
	if err := r.afterPhase(GovernanceAfterAudit); err != nil {
		return contract.MutationResult{}, err
	}
	for _, event := range prepared.Outbox {
		if err := persistOutboxEvent(ctx, connection, event); err != nil {
			return contract.MutationResult{}, err
		}
	}
	if err := r.afterPhase(GovernanceAfterOutbox); err != nil {
		return contract.MutationResult{}, err
	}
	if prepared.Command.IdempotencyKey != "" {
		if err := persistGovernanceReplay(ctx, connection, prepared); err != nil {
			return contract.MutationResult{}, err
		}
	}
	if err := r.afterPhase(GovernanceAfterReplay); err != nil {
		return contract.MutationResult{}, err
	}
	if _, err := connection.ExecContext(ctx, `COMMIT`); err != nil {
		return contract.MutationResult{}, fmt.Errorf("commit governed mutation: %w", err)
	}
	committed = true
	return prepared.Result, nil
}

func (r *GovernanceRepository) afterPhase(phase GovernancePhase) error {
	if r.failureHook == nil {
		return nil
	}
	if err := r.failureHook(phase); err != nil {
		return fmt.Errorf("governance phase %s: %w", phase, err)
	}
	return nil
}

func validatePreparedGovernanceMutation(prepared application.PreparedGovernanceMutation) error {
	if err := prepared.Identity.Validate(); err != nil {
		return err
	}
	if err := prepared.Command.Validate(); err != nil {
		return err
	}
	if err := prepared.Result.Validate(); err != nil {
		return err
	}
	if prepared.Result.ResourceRevision != prepared.Command.ExpectedRevision+1 {
		return fmt.Errorf("%w: prepared revision is inconsistent", contract.ErrInvalidGovernanceMutation)
	}
	if prepared.Result.Replayed {
		return fmt.Errorf("%w: prepared mutation cannot already be replayed", contract.ErrInvalidGovernanceMutation)
	}
	if err := prepared.Audit.Validate(); err != nil {
		return err
	}
	if prepared.Audit.Identity != prepared.Identity ||
		prepared.Audit.Action != prepared.Command.Action ||
		prepared.Audit.ResourceKind != prepared.Command.ResourceKind ||
		prepared.Audit.ResourceID != prepared.Command.ResourceID ||
		prepared.Audit.ResourceRevision != prepared.Result.ResourceRevision {
		return fmt.Errorf("%w: audit envelope is inconsistent", contract.ErrInvalidGovernanceMutation)
	}
	for _, event := range prepared.Outbox {
		if err := event.Validate(); err != nil {
			return err
		}
		if event.WorkspaceID != prepared.Identity.WorkspaceID ||
			event.AggregateKind != prepared.Command.ResourceKind ||
			event.AggregateID != prepared.Command.ResourceID ||
			event.AggregateRevision != prepared.Result.ResourceRevision ||
			event.ActorType != prepared.Identity.ActorType ||
			event.ActorID != prepared.Identity.ActorID ||
			event.State != contract.OutboxReady {
			return fmt.Errorf("%w: outbox envelope is inconsistent", contract.ErrInvalidGovernanceMutation)
		}
	}
	return nil
}

func loadGovernanceReplay(ctx context.Context, connection *sql.Conn, prepared application.PreparedGovernanceMutation) (contract.MutationResult, bool, error) {
	var hash string
	var result contract.MutationResult
	var body string
	err := connection.QueryRowContext(ctx, `SELECT request_hash, resource_revision, response_status, response_body
		FROM workspace_mutation_idempotency WHERE workspace_id=? AND action=? AND idempotency_key=?`,
		prepared.Identity.WorkspaceID, prepared.Command.Action, prepared.Command.IdempotencyKey,
	).Scan(&hash, &result.ResourceRevision, &result.ResponseStatus, &body)
	if errors.Is(err, sql.ErrNoRows) {
		return contract.MutationResult{}, false, nil
	}
	if err != nil {
		return contract.MutationResult{}, false, fmt.Errorf("load mutation replay: %w", err)
	}
	if !strings.EqualFold(hash, prepared.Command.RequestHash) {
		return contract.MutationResult{}, false, contract.ErrIdempotencyConflict
	}
	result.ResponseBody = json.RawMessage(body)
	result.Replayed = true
	if err := result.Validate(); err != nil {
		return contract.MutationResult{}, false, fmt.Errorf("validate stored mutation replay: %w", err)
	}
	return result, true, nil
}

func loadResourceRevision(ctx context.Context, connection *sql.Conn, prepared application.PreparedGovernanceMutation) (int64, error) {
	var revision int64
	err := connection.QueryRowContext(ctx, `SELECT revision FROM workspace_resource_revisions
		WHERE workspace_id=? AND resource_kind=? AND resource_id=?`,
		prepared.Identity.WorkspaceID, prepared.Command.ResourceKind, prepared.Command.ResourceID,
	).Scan(&revision)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("load resource revision: %w", err)
	}
	return revision, nil
}

func persistResourceRevision(ctx context.Context, connection *sql.Conn, prepared application.PreparedGovernanceMutation) error {
	_, err := connection.ExecContext(ctx, `INSERT INTO workspace_resource_revisions
		(workspace_id, resource_kind, resource_id, revision, updated_at) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(workspace_id, resource_kind, resource_id) DO UPDATE SET revision=excluded.revision, updated_at=excluded.updated_at`,
		prepared.Identity.WorkspaceID, prepared.Command.ResourceKind, prepared.Command.ResourceID,
		prepared.Result.ResourceRevision, prepared.Audit.OccurredAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("persist resource revision: %w", err)
	}
	return nil
}

func persistAuditRecord(ctx context.Context, connection *sql.Conn, record contract.AuditRecord) error {
	_, err := connection.ExecContext(ctx, `INSERT INTO workspace_audit_entries
		(workspace_id, occurred_at, id, actor_type, actor_id, action, resource_kind, resource_id, resource_revision, request_id, metadata_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, record.Identity.WorkspaceID, record.OccurredAt.UTC().Format(time.RFC3339Nano),
		record.ID, record.Identity.ActorType, record.Identity.ActorID, record.Action, record.ResourceKind, record.ResourceID,
		record.ResourceRevision, record.Identity.RequestID, string(record.Metadata))
	if err != nil {
		return fmt.Errorf("persist audit record: %w", err)
	}
	return nil
}

func persistOutboxEvent(ctx context.Context, connection *sql.Conn, event contract.OutboxEvent) error {
	_, err := connection.ExecContext(ctx, `INSERT INTO workspace_outbox_events
		(state, available_at, workspace_id, id, event_type, aggregate_kind, aggregate_id, aggregate_revision, payload_json,
		 actor_type, actor_id, attempt_count, claim_token, lease_expires_at, last_error_code, created_at, delivered_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, event.State, event.AvailableAt.UTC().Format(time.RFC3339Nano),
		event.WorkspaceID, event.ID, event.EventType, event.AggregateKind, event.AggregateID, event.AggregateRevision, string(event.Payload),
		event.ActorType, event.ActorID, event.AttemptCount, nullableGovernanceString(event.ClaimToken), nullableGovernanceTime(event.LeaseExpiresAt),
		nullableGovernanceString(event.LastErrorCode), event.CreatedAt.UTC().Format(time.RFC3339Nano), nullableGovernanceTime(event.DeliveredAt))
	if err != nil {
		return fmt.Errorf("persist outbox event: %w", err)
	}
	return nil
}

func persistGovernanceReplay(ctx context.Context, connection *sql.Conn, prepared application.PreparedGovernanceMutation) error {
	_, err := connection.ExecContext(ctx, `INSERT INTO workspace_mutation_idempotency
		(workspace_id, action, idempotency_key, request_hash, resource_kind, resource_id, resource_revision,
		 response_status, response_body, created_at, expires_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL)`,
		prepared.Identity.WorkspaceID, prepared.Command.Action, prepared.Command.IdempotencyKey, prepared.Command.RequestHash,
		prepared.Command.ResourceKind, prepared.Command.ResourceID, prepared.Result.ResourceRevision,
		prepared.Result.ResponseStatus, string(prepared.Result.ResponseBody), prepared.Audit.OccurredAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("persist mutation replay: %w", err)
	}
	return nil
}

func nullableGovernanceString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func nullableGovernanceTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC().Format(time.RFC3339Nano)
}
