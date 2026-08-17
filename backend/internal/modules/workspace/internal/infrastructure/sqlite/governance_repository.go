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

func WithGovernanceEventPolicies(provider application.GovernanceEventPolicyProvider) GovernanceRepositoryOption {
	return func(repository *GovernanceRepository) {
		repository.eventPolicies = provider
	}
}

type GovernanceRepository struct {
	db            *sql.DB
	failureHook   func(GovernancePhase) error
	eventPolicies application.GovernanceEventPolicyProvider
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
	command := prepared.Command()
	audit := prepared.Audit()
	outbox := prepared.Outbox()

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

	if command.IdempotencyKey != "" {
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
	if currentRevision != command.ExpectedRevision {
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
	if err := persistAuditRecord(ctx, connection, audit); err != nil {
		return contract.MutationResult{}, err
	}
	if err := r.afterPhase(GovernanceAfterAudit); err != nil {
		return contract.MutationResult{}, err
	}
	for _, event := range outbox {
		if err := persistOutboxEvent(ctx, connection, event); err != nil {
			return contract.MutationResult{}, err
		}
	}
	if err := r.afterPhase(GovernanceAfterOutbox); err != nil {
		return contract.MutationResult{}, err
	}
	if command.IdempotencyKey != "" {
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
	return prepared.Result(), nil
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

func (r *GovernanceRepository) ClaimOutbox(ctx context.Context, now time.Time, limit int, lease time.Duration, claimToken string) (events []contract.OutboxEvent, err error) {
	claimToken = strings.TrimSpace(claimToken)
	if now.IsZero() || lease <= 0 || claimToken == "" {
		return nil, fmt.Errorf("%w: invalid outbox claim", contract.ErrInvalidGovernanceMutation)
	}
	if r.eventPolicies == nil {
		return nil, contract.ErrGovernanceUnavailable
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	nowText := now.UTC().Format(time.RFC3339Nano)
	var available bool
	if err := r.db.QueryRowContext(ctx, `SELECT EXISTS(
		SELECT 1 FROM workspace_outbox_events
		WHERE ((state IN ('ready','retry_wait') AND available_at <= ?)
			OR (state='inflight' AND lease_expires_at <= ?))
		LIMIT 1)`, nowText, nowText).Scan(&available); err != nil {
		return nil, fmt.Errorf("check outbox claim availability: %w", err)
	}
	if !available {
		return []contract.OutboxEvent{}, nil
	}
	connection, err := r.db.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquire outbox claim connection: %w", err)
	}
	defer connection.Close()
	if _, err = connection.ExecContext(ctx, `PRAGMA busy_timeout = 5000`); err != nil {
		return nil, fmt.Errorf("configure outbox claim connection: %w", err)
	}
	if _, err = connection.ExecContext(ctx, `BEGIN IMMEDIATE`); err != nil {
		return nil, fmt.Errorf("begin outbox claim: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = connection.ExecContext(context.WithoutCancel(ctx), `ROLLBACK`)
		}
	}()

	rows, err := connection.QueryContext(ctx, `SELECT state, available_at, workspace_id, id, event_type,
		aggregate_kind, aggregate_id, aggregate_revision, payload_json, actor_type, actor_id, attempt_count,
		claim_token, lease_expires_at, last_error_code, created_at, delivered_at
		FROM workspace_outbox_events
		WHERE ((state IN ('ready','retry_wait') AND available_at <= ?)
			OR (state='inflight' AND lease_expires_at <= ?))
		ORDER BY available_at, workspace_id, id LIMIT ?`, nowText, nowText, limit)
	if err != nil {
		return nil, fmt.Errorf("select outbox claims: %w", err)
	}
	for rows.Next() {
		event, scanErr := scanOutboxEvent(rows)
		if scanErr != nil {
			_ = rows.Close()
			return nil, scanErr
		}
		events = append(events, event)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close outbox claims: %w", err)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate outbox claims: %w", err)
	}
	for _, event := range events {
		if err := application.ValidateGovernanceEventPolicy(ctx, r.eventPolicies, event); err != nil {
			return nil, err
		}
	}

	leaseExpiresAt := now.Add(lease).UTC()
	for index := range events {
		event := &events[index]
		result, err := connection.ExecContext(ctx, `UPDATE workspace_outbox_events
			SET state='inflight', claim_token=?, lease_expires_at=?, attempt_count=attempt_count+1
			WHERE state=? AND available_at=? AND workspace_id=? AND id=?`,
			claimToken, leaseExpiresAt.Format(time.RFC3339Nano), event.State,
			event.AvailableAt.UTC().Format(time.RFC3339Nano), event.WorkspaceID, event.ID)
		if err != nil {
			return nil, fmt.Errorf("claim outbox event: %w", err)
		}
		if err := requireOneOutboxRow(result); err != nil {
			return nil, err
		}
		event.State = contract.OutboxInflight
		event.ClaimToken = claimToken
		event.LeaseExpiresAt = &leaseExpiresAt
		event.AttemptCount++
	}
	if _, err = connection.ExecContext(ctx, `COMMIT`); err != nil {
		return nil, fmt.Errorf("commit outbox claim: %w", err)
	}
	committed = true
	return events, nil
}

func (r *GovernanceRepository) MarkOutboxDelivered(ctx context.Context, identity contract.OutboxClaimIdentity, deliveredAt time.Time) error {
	if err := identity.Validate(); err != nil || deliveredAt.IsZero() {
		return fmt.Errorf("%w: invalid outbox delivery", contract.ErrInvalidGovernanceMutation)
	}
	deliveredText := deliveredAt.UTC().Format(time.RFC3339Nano)
	result, err := r.db.ExecContext(ctx, `UPDATE workspace_outbox_events
		SET state='delivered', claim_token=NULL, lease_expires_at=NULL, delivered_at=?, last_error_code=NULL
		WHERE state=? AND available_at=? AND workspace_id=? AND id=?
			AND claim_token=? AND lease_expires_at=? AND lease_expires_at>?`,
		deliveredText, identity.State, identity.AvailableAt.UTC().Format(time.RFC3339Nano), identity.WorkspaceID, identity.ID,
		identity.ClaimToken, identity.LeaseExpiresAt.UTC().Format(time.RFC3339Nano), deliveredText)
	if err != nil {
		return fmt.Errorf("mark outbox delivered: %w", err)
	}
	if err := requireOneOutboxRow(result); err != nil {
		return err
	}
	return nil
}

func (r *GovernanceRepository) MarkOutboxFailed(ctx context.Context, identity contract.OutboxClaimIdentity, failedAt, retryAt time.Time, errorCode string, dead bool) error {
	if err := identity.Validate(); err != nil || failedAt.IsZero() || retryAt.IsZero() || strings.TrimSpace(errorCode) == "" {
		return fmt.Errorf("%w: invalid outbox failure", contract.ErrInvalidGovernanceMutation)
	}
	state := contract.OutboxRetryWait
	if dead {
		state = contract.OutboxDeadLetter
	}
	failedText := failedAt.UTC().Format(time.RFC3339Nano)
	result, err := r.db.ExecContext(ctx, `UPDATE workspace_outbox_events
		SET state=?, available_at=?, claim_token=NULL, lease_expires_at=NULL, last_error_code=?, delivered_at=NULL
		WHERE state=? AND available_at=? AND workspace_id=? AND id=?
			AND claim_token=? AND lease_expires_at=? AND lease_expires_at>?`,
		state, retryAt.UTC().Format(time.RFC3339Nano), errorCode,
		identity.State, identity.AvailableAt.UTC().Format(time.RFC3339Nano), identity.WorkspaceID, identity.ID,
		identity.ClaimToken, identity.LeaseExpiresAt.UTC().Format(time.RFC3339Nano), failedText)
	if err != nil {
		return fmt.Errorf("mark outbox failed: %w", err)
	}
	return requireOneOutboxRow(result)
}

func (r *GovernanceRepository) ReplayOutbox(ctx context.Context, identity contract.OutboxRowIdentity, availableAt time.Time) error {
	if err := identity.Validate(); err != nil || identity.State != contract.OutboxDeadLetter || availableAt.IsZero() {
		return fmt.Errorf("%w: invalid outbox replay", contract.ErrInvalidGovernanceMutation)
	}
	if r.eventPolicies == nil {
		return contract.ErrGovernanceUnavailable
	}
	connection, err := r.db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("acquire outbox replay connection: %w", err)
	}
	defer connection.Close()
	if _, err := connection.ExecContext(ctx, `PRAGMA busy_timeout = 5000`); err != nil {
		return fmt.Errorf("configure outbox replay connection: %w", err)
	}
	if _, err := connection.ExecContext(ctx, `BEGIN IMMEDIATE`); err != nil {
		return fmt.Errorf("begin outbox replay: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = connection.ExecContext(context.WithoutCancel(ctx), `ROLLBACK`)
		}
	}()
	event, err := scanOutboxEvent(connection.QueryRowContext(ctx, `SELECT state, available_at, workspace_id, id, event_type,
		aggregate_kind, aggregate_id, aggregate_revision, payload_json, actor_type, actor_id, attempt_count,
		claim_token, lease_expires_at, last_error_code, created_at, delivered_at
		FROM workspace_outbox_events
		WHERE state=? AND available_at=? AND workspace_id=? AND id=?`,
		identity.State, identity.AvailableAt.UTC().Format(time.RFC3339Nano), identity.WorkspaceID, identity.ID))
	if errors.Is(err, sql.ErrNoRows) {
		return contract.ErrOutboxClaimConflict
	}
	if err != nil {
		return err
	}
	if err := application.ValidateGovernanceEventPolicy(ctx, r.eventPolicies, event); err != nil {
		return err
	}
	result, err := connection.ExecContext(ctx, `UPDATE workspace_outbox_events
		SET state='ready', available_at=?, attempt_count=0, claim_token=NULL, lease_expires_at=NULL,
			last_error_code=NULL, delivered_at=NULL
		WHERE state=? AND available_at=? AND workspace_id=? AND id=?`,
		availableAt.UTC().Format(time.RFC3339Nano), identity.State,
		identity.AvailableAt.UTC().Format(time.RFC3339Nano), identity.WorkspaceID, identity.ID)
	if err != nil {
		return fmt.Errorf("replay dead-letter outbox event: %w", err)
	}
	if err := requireOneOutboxRow(result); err != nil {
		return err
	}
	if _, err := connection.ExecContext(ctx, `COMMIT`); err != nil {
		return fmt.Errorf("commit outbox replay: %w", err)
	}
	committed = true
	return nil
}

func (r *GovernanceRepository) ReadOutboxDiagnostics(ctx context.Context, workspaceID string, now time.Time) (contract.OutboxDiagnostics, error) {
	if strings.TrimSpace(workspaceID) == "" || now.IsZero() {
		return contract.OutboxDiagnostics{}, fmt.Errorf("%w: invalid governance diagnostics request", contract.ErrInvalidGovernanceMutation)
	}
	var diagnostics contract.OutboxDiagnostics
	var oldestReady, oldestLease, lastDelivered sql.NullString
	err := r.db.QueryRowContext(ctx, `SELECT
		COALESCE(SUM(CASE WHEN state='ready' THEN 1 ELSE 0 END), 0),
		MIN(CASE WHEN state='ready' THEN available_at END),
		COALESCE(SUM(CASE WHEN state='inflight' THEN 1 ELSE 0 END), 0),
		MIN(CASE WHEN state='inflight' THEN lease_expires_at END),
		COALESCE(SUM(CASE WHEN state='retry_wait' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN state='dead_letter' THEN 1 ELSE 0 END), 0),
		MAX(CASE WHEN state='delivered' THEN delivered_at END)
		FROM workspace_outbox_events WHERE workspace_id=?`, workspaceID).Scan(
		&diagnostics.ReadyCount, &oldestReady, &diagnostics.InflightCount, &oldestLease,
		&diagnostics.RetryWaitCount, &diagnostics.DeadLetterCount, &lastDelivered)
	if err != nil {
		return contract.OutboxDiagnostics{}, fmt.Errorf("read governance diagnostics: %w", err)
	}
	if diagnostics.OldestReadyAge, err = outboxAge(now, oldestReady); err != nil {
		return contract.OutboxDiagnostics{}, err
	}
	if diagnostics.OldestLeaseAge, err = outboxAge(now, oldestLease); err != nil {
		return contract.OutboxDiagnostics{}, err
	}
	if value, parseErr := parseNullableOutboxTime(lastDelivered); parseErr != nil {
		return contract.OutboxDiagnostics{}, parseErr
	} else if value != nil {
		diagnostics.LastSuccessfulDelivery = *value
	}
	diagnostics.SchemaVersion = "000009_workspace_governance"
	return diagnostics, diagnostics.Validate()
}

func outboxAge(now time.Time, value sql.NullString) (time.Duration, error) {
	parsed, err := parseNullableOutboxTime(value)
	if err != nil || parsed == nil {
		return 0, err
	}
	age := now.UTC().Sub(parsed.UTC())
	if age < 0 {
		return 0, nil
	}
	return age, nil
}

type outboxRowScanner interface {
	Scan(...any) error
}

func scanOutboxEvent(scanner outboxRowScanner) (contract.OutboxEvent, error) {
	var event contract.OutboxEvent
	var state, availableAt, payload, createdAt string
	var claimToken, leaseExpiresAt, lastErrorCode, deliveredAt sql.NullString
	if err := scanner.Scan(&state, &availableAt, &event.WorkspaceID, &event.ID, &event.EventType,
		&event.AggregateKind, &event.AggregateID, &event.AggregateRevision, &payload, &event.ActorType, &event.ActorID,
		&event.AttemptCount, &claimToken, &leaseExpiresAt, &lastErrorCode, &createdAt, &deliveredAt); err != nil {
		return contract.OutboxEvent{}, fmt.Errorf("scan outbox event: %w", err)
	}
	event.State = contract.OutboxState(state)
	event.Payload = json.RawMessage(payload)
	var err error
	if event.AvailableAt, err = time.Parse(time.RFC3339Nano, availableAt); err != nil {
		return contract.OutboxEvent{}, fmt.Errorf("parse outbox available time: %w", err)
	}
	if event.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt); err != nil {
		return contract.OutboxEvent{}, fmt.Errorf("parse outbox created time: %w", err)
	}
	event.ClaimToken = claimToken.String
	event.LastErrorCode = lastErrorCode.String
	if event.LeaseExpiresAt, err = parseNullableOutboxTime(leaseExpiresAt); err != nil {
		return contract.OutboxEvent{}, err
	}
	if event.DeliveredAt, err = parseNullableOutboxTime(deliveredAt); err != nil {
		return contract.OutboxEvent{}, err
	}
	if err := event.Validate(); err != nil {
		return contract.OutboxEvent{}, fmt.Errorf("validate stored outbox event: %w", err)
	}
	return event, nil
}

func parseNullableOutboxTime(value sql.NullString) (*time.Time, error) {
	if !value.Valid {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, value.String)
	if err != nil {
		return nil, fmt.Errorf("parse outbox timestamp: %w", err)
	}
	return &parsed, nil
}

func requireOneOutboxRow(result sql.Result) error {
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read outbox transition result: %w", err)
	}
	if rows != 1 {
		return contract.ErrOutboxClaimConflict
	}
	return nil
}

func validatePreparedGovernanceMutation(prepared application.PreparedGovernanceMutation) error {
	return prepared.Validate()
}

func loadGovernanceReplay(ctx context.Context, connection *sql.Conn, prepared application.PreparedGovernanceMutation) (contract.MutationResult, bool, error) {
	identity, command := prepared.Identity(), prepared.Command()
	var hash string
	var result contract.MutationResult
	var body string
	err := connection.QueryRowContext(ctx, `SELECT request_hash, resource_revision, response_status, response_body
		FROM workspace_mutation_idempotency WHERE workspace_id=? AND action=? AND idempotency_key=?`,
		identity.WorkspaceID, command.Action, command.IdempotencyKey,
	).Scan(&hash, &result.ResourceRevision, &result.ResponseStatus, &body)
	if errors.Is(err, sql.ErrNoRows) {
		return contract.MutationResult{}, false, nil
	}
	if err != nil {
		return contract.MutationResult{}, false, fmt.Errorf("load mutation replay: %w", err)
	}
	if !strings.EqualFold(hash, command.RequestHash) {
		return contract.MutationResult{}, false, contract.ErrIdempotencyConflict
	}
	result.ResponseBody = json.RawMessage(body)
	result.Replayed = true
	if err := result.Validate(); err != nil {
		return contract.MutationResult{}, false, fmt.Errorf("validate stored mutation replay: %w", err)
	}
	if err := prepared.ValidateReplayResponse(result.ResponseBody); err != nil {
		return contract.MutationResult{}, false, fmt.Errorf("validate stored replay policy: %w", err)
	}
	return result, true, nil
}

func loadResourceRevision(ctx context.Context, connection *sql.Conn, prepared application.PreparedGovernanceMutation) (int64, error) {
	identity, command := prepared.Identity(), prepared.Command()
	var revision int64
	err := connection.QueryRowContext(ctx, `SELECT revision FROM workspace_resource_revisions
		WHERE workspace_id=? AND resource_kind=? AND resource_id=?`,
		identity.WorkspaceID, command.ResourceKind, command.ResourceID,
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
	identity, command, result, audit := prepared.Identity(), prepared.Command(), prepared.Result(), prepared.Audit()
	_, err := connection.ExecContext(ctx, `INSERT INTO workspace_resource_revisions
		(workspace_id, resource_kind, resource_id, revision, updated_at) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(workspace_id, resource_kind, resource_id) DO UPDATE SET revision=excluded.revision, updated_at=excluded.updated_at`,
		identity.WorkspaceID, command.ResourceKind, command.ResourceID,
		result.ResourceRevision, audit.OccurredAt.UTC().Format(time.RFC3339Nano),
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
	identity, command, result, audit := prepared.Identity(), prepared.Command(), prepared.Result(), prepared.Audit()
	_, err := connection.ExecContext(ctx, `INSERT INTO workspace_mutation_idempotency
		(workspace_id, action, idempotency_key, request_hash, resource_kind, resource_id, resource_revision,
		 response_status, response_body, created_at, expires_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL)`,
		identity.WorkspaceID, command.Action, command.IdempotencyKey, command.RequestHash,
		command.ResourceKind, command.ResourceID, result.ResourceRevision,
		result.ResponseStatus, string(result.ResponseBody), audit.OccurredAt.UTC().Format(time.RFC3339Nano))
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
