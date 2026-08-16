package contract

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	MaxReplayResponseBytes = 64 * 1024
	MaxAuditMetadataBytes  = 16 * 1024
	MaxOutboxPayloadBytes  = 64 * 1024
)

// MutationIdentity is the trusted Workspace caller projection attached by a
// transport boundary before a governed mutation starts.
type MutationIdentity struct {
	WorkspaceID string
	ActorType   string
	ActorID     string
	RequestID   string
}

type MutationCommand struct {
	Action           string
	ResourceKind     string
	ResourceID       string
	ExpectedRevision int64
	IdempotencyKey   string
	RequestHash      string
}

func (c MutationCommand) Validate() error {
	if strings.TrimSpace(c.Action) == "" || strings.TrimSpace(c.ResourceKind) == "" || strings.TrimSpace(c.ResourceID) == "" {
		return fmt.Errorf("%w: action and resource identity are required", ErrInvalidGovernanceMutation)
	}
	if c.ExpectedRevision < 0 {
		return fmt.Errorf("%w: expected revision cannot be negative", ErrInvalidGovernanceMutation)
	}
	key, hash := strings.TrimSpace(c.IdempotencyKey), strings.TrimSpace(c.RequestHash)
	if key == "" && hash == "" {
		return nil
	}
	decoded, err := hex.DecodeString(hash)
	if key == "" || len(key) > 200 || err != nil || len(decoded) != 32 {
		return fmt.Errorf("%w: idempotency key and SHA-256 request hash must be supplied together", ErrInvalidGovernanceMutation)
	}
	return nil
}

type MutationResult struct {
	ResourceRevision int64
	ResponseStatus   int
	ResponseBody     json.RawMessage
	Replayed         bool
}

func (r MutationResult) Validate() error {
	if r.ResourceRevision < 1 || r.ResponseStatus < 100 || r.ResponseStatus > 599 || !json.Valid(r.ResponseBody) {
		return fmt.Errorf("%w: invalid mutation result", ErrInvalidGovernanceMutation)
	}
	if len(r.ResponseBody) > MaxReplayResponseBytes {
		return ErrIdempotencyResponseTooLarge
	}
	return nil
}

type AuditRecord struct {
	ID               string
	Identity         MutationIdentity
	Action           string
	ResourceKind     string
	ResourceID       string
	ResourceRevision int64
	OccurredAt       time.Time
	Metadata         json.RawMessage
}

func (r AuditRecord) Validate() error {
	if err := r.Identity.Validate(); err != nil {
		return err
	}
	metadata := strings.TrimSpace(string(r.Metadata))
	if strings.TrimSpace(r.ID) == "" || strings.TrimSpace(r.Action) == "" || strings.TrimSpace(r.ResourceKind) == "" || strings.TrimSpace(r.ResourceID) == "" || r.ResourceRevision < 1 || r.OccurredAt.IsZero() || !json.Valid(r.Metadata) || !strings.HasPrefix(metadata, "{") || len(r.Metadata) > MaxAuditMetadataBytes {
		return fmt.Errorf("%w: invalid audit record", ErrInvalidGovernanceMutation)
	}
	return nil
}

type OutboxState string

const (
	OutboxReady      OutboxState = "ready"
	OutboxInflight   OutboxState = "inflight"
	OutboxRetryWait  OutboxState = "retry_wait"
	OutboxDelivered  OutboxState = "delivered"
	OutboxDeadLetter OutboxState = "dead_letter"
)

type OutboxEvent struct {
	State             OutboxState
	AvailableAt       time.Time
	WorkspaceID       string
	ID                string
	EventType         string
	AggregateKind     string
	AggregateID       string
	AggregateRevision int64
	Payload           json.RawMessage
	ActorType         string
	ActorID           string
	AttemptCount      int
	ClaimToken        string
	LeaseExpiresAt    *time.Time
	LastErrorCode     string
	CreatedAt         time.Time
	DeliveredAt       *time.Time
}

func (e OutboxEvent) Validate() error {
	if !validOutboxState(e.State) || e.AvailableAt.IsZero() || strings.TrimSpace(e.WorkspaceID) == "" || strings.TrimSpace(e.ID) == "" || strings.TrimSpace(e.EventType) == "" || strings.TrimSpace(e.AggregateKind) == "" || strings.TrimSpace(e.AggregateID) == "" || e.AggregateRevision < 1 || !json.Valid(e.Payload) || len(e.Payload) > MaxOutboxPayloadBytes || strings.TrimSpace(e.ActorID) == "" || e.AttemptCount < 0 || e.CreatedAt.IsZero() {
		return fmt.Errorf("%w: invalid outbox event", ErrInvalidGovernanceMutation)
	}
	if e.ActorType != "member" && e.ActorType != "agent" {
		return fmt.Errorf("%w: invalid outbox actor", ErrInvalidGovernanceMutation)
	}
	claimToken := strings.TrimSpace(e.ClaimToken)
	if e.State == OutboxInflight {
		if claimToken == "" || e.LeaseExpiresAt == nil || e.LeaseExpiresAt.IsZero() {
			return fmt.Errorf("%w: inflight outbox event requires a claim and lease", ErrInvalidGovernanceMutation)
		}
	} else if claimToken != "" || e.LeaseExpiresAt != nil {
		return fmt.Errorf("%w: only inflight outbox events may hold a claim", ErrInvalidGovernanceMutation)
	}
	return nil
}

func validOutboxState(state OutboxState) bool {
	switch state {
	case OutboxReady, OutboxInflight, OutboxRetryWait, OutboxDelivered, OutboxDeadLetter:
		return true
	default:
		return false
	}
}

type RevisionConflictError struct {
	CurrentRevision int64
}

func (e RevisionConflictError) Error() string {
	return fmt.Sprintf("%s: current revision %d", ErrRevisionConflict, e.CurrentRevision)
}

func (e RevisionConflictError) Unwrap() error { return ErrRevisionConflict }

type OutboxDiagnostics struct {
	ReadyCount             int64
	OldestReadyAge         time.Duration
	InflightCount          int64
	OldestLeaseAge         time.Duration
	RetryWaitCount         int64
	DeadLetterCount        int64
	LastSuccessfulDelivery time.Time
	SchemaVersion          string
	DispatcherRunning      bool
}

func (d OutboxDiagnostics) Validate() error {
	if d.ReadyCount < 0 || d.OldestReadyAge < 0 || d.InflightCount < 0 || d.OldestLeaseAge < 0 || d.RetryWaitCount < 0 || d.DeadLetterCount < 0 {
		return fmt.Errorf("%w: invalid outbox diagnostics", ErrInvalidGovernanceMutation)
	}
	return nil
}

type OutboxSink interface {
	Publish(context.Context, OutboxEvent) error
}

type GovernanceDiagnosticsReader interface {
	ReadGovernanceDiagnostics(context.Context, string) (OutboxDiagnostics, error)
}

func (i MutationIdentity) Validate() error {
	if strings.TrimSpace(i.WorkspaceID) == "" || strings.TrimSpace(i.ActorID) == "" || strings.TrimSpace(i.RequestID) == "" {
		return fmt.Errorf("%w: workspace, actor, and request identity are required", ErrInvalidGovernanceMutation)
	}
	switch strings.TrimSpace(i.ActorType) {
	case "member", "agent":
		return nil
	default:
		return fmt.Errorf("%w: actor type is not trusted", ErrInvalidGovernanceMutation)
	}
}
