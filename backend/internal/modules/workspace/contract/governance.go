package contract

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
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
	if err := validateGovernanceEnvelope(r.ResponseBody, "governance-replay-v1"); err != nil {
		return err
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
	if err := validateGovernanceEnvelope(r.Metadata, "governance-audit-v1"); err != nil {
		return err
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

// OutboxRowIdentity is the complete persisted primary-key tuple for an event.
// Event ID alone is never sufficient write authority.
type OutboxRowIdentity struct {
	State       OutboxState
	AvailableAt time.Time
	WorkspaceID string
	ID          string
}

func (i OutboxRowIdentity) Validate() error {
	if !validOutboxState(i.State) || i.AvailableAt.IsZero() || strings.TrimSpace(i.WorkspaceID) == "" || strings.TrimSpace(i.ID) == "" {
		return fmt.Errorf("%w: invalid outbox row identity", ErrInvalidGovernanceMutation)
	}
	return nil
}

// OutboxClaimIdentity binds an inflight row tuple to the observed claim token
// and lease. Both values participate in every acknowledgement or failure write.
type OutboxClaimIdentity struct {
	OutboxRowIdentity
	ClaimToken     string
	LeaseExpiresAt time.Time
}

func (i OutboxClaimIdentity) Validate() error {
	if err := i.OutboxRowIdentity.Validate(); err != nil {
		return err
	}
	if i.State != OutboxInflight || strings.TrimSpace(i.ClaimToken) == "" || i.LeaseExpiresAt.IsZero() {
		return fmt.Errorf("%w: invalid outbox claim identity", ErrInvalidGovernanceMutation)
	}
	return nil
}

func (e OutboxEvent) RowIdentity() OutboxRowIdentity {
	return OutboxRowIdentity{State: e.State, AvailableAt: e.AvailableAt, WorkspaceID: e.WorkspaceID, ID: e.ID}
}

func (e OutboxEvent) ClaimIdentity() (OutboxClaimIdentity, error) {
	if e.LeaseExpiresAt == nil {
		return OutboxClaimIdentity{}, fmt.Errorf("%w: outbox claim has no lease", ErrInvalidGovernanceMutation)
	}
	identity := OutboxClaimIdentity{
		OutboxRowIdentity: e.RowIdentity(),
		ClaimToken:        e.ClaimToken,
		LeaseExpiresAt:    e.LeaseExpiresAt.UTC(),
	}
	return identity, identity.Validate()
}

func (e OutboxEvent) Validate() error {
	if !validOutboxState(e.State) || e.AvailableAt.IsZero() || strings.TrimSpace(e.WorkspaceID) == "" || strings.TrimSpace(e.ID) == "" || strings.TrimSpace(e.EventType) == "" || strings.TrimSpace(e.AggregateKind) == "" || strings.TrimSpace(e.AggregateID) == "" || e.AggregateRevision < 1 || !json.Valid(e.Payload) || len(e.Payload) > MaxOutboxPayloadBytes || strings.TrimSpace(e.ActorID) == "" || e.AttemptCount < 0 || e.CreatedAt.IsZero() {
		return fmt.Errorf("%w: invalid outbox event", ErrInvalidGovernanceMutation)
	}
	if e.ActorType != "member" && e.ActorType != "agent" {
		return fmt.Errorf("%w: invalid outbox actor", ErrInvalidGovernanceMutation)
	}
	if err := validateGovernanceEnvelope(e.Payload, "governance-outbox-v1"); err != nil {
		return err
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

func validateGovernanceEnvelope(raw json.RawMessage, version string) error {
	if err := rejectDuplicateJSONKeys(raw); err != nil {
		return fmt.Errorf("%w: invalid governance envelope", ErrInvalidGovernanceMutation)
	}
	var envelope struct {
		Version string                     `json:"version"`
		Data    map[string]json.RawMessage `json:"data"`
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&envelope); err != nil {
		return fmt.Errorf("%w: invalid governance envelope", ErrInvalidGovernanceMutation)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("%w: invalid governance envelope", ErrInvalidGovernanceMutation)
	}
	if envelope.Version != version || envelope.Data == nil {
		return fmt.Errorf("%w: invalid governance envelope", ErrInvalidGovernanceMutation)
	}
	if governanceEnvelopeContainsForbidden(envelope.Data) {
		return fmt.Errorf("%w: forbidden governance material", ErrInvalidGovernanceMutation)
	}
	return nil
}

func ContainsForbiddenGovernanceMaterial(value string) bool {
	lower := strings.ToLower(value)
	for _, authorization := range []string{"basic ", "basic:"} {
		if strings.Contains(lower, authorization) {
			return true
		}
	}
	for _, forbidden := range []string{
		"authorization", "bearer", "credential", "password", "passwd", "secret",
		"token", "cookie", "api-key", "api_key", "apikey", "prompt", "archive",
		"attachment-body", "attachment_body", "raw-body", "raw_body",
	} {
		if strings.Contains(lower, forbidden) {
			return true
		}
	}
	return false
}

func governanceEnvelopeContainsForbidden(data map[string]json.RawMessage) bool {
	for key, raw := range data {
		if ContainsForbiddenGovernanceMaterial(key) {
			return true
		}
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.UseNumber()
		var value any
		if err := decoder.Decode(&value); err != nil || containsForbiddenGovernanceValue(value) {
			return true
		}
	}
	return false
}

func containsForbiddenGovernanceValue(value any) bool {
	switch typed := value.(type) {
	case string:
		return ContainsForbiddenGovernanceMaterial(typed)
	case []any:
		for _, item := range typed {
			if containsForbiddenGovernanceValue(item) {
				return true
			}
		}
	case map[string]any:
		for key, item := range typed {
			if ContainsForbiddenGovernanceMaterial(key) || containsForbiddenGovernanceValue(item) {
				return true
			}
		}
	}
	return false
}

func rejectDuplicateJSONKeys(raw json.RawMessage) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := consumeUniqueJSONValue(decoder); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return fmt.Errorf("trailing JSON token")
	}
	return nil
}

func consumeUniqueJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("object key is not a string")
			}
			if _, duplicate := seen[key]; duplicate {
				return fmt.Errorf("duplicate object key")
			}
			seen[key] = struct{}{}
			if err := consumeUniqueJSONValue(decoder); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim('}') {
			return fmt.Errorf("unterminated object")
		}
	case '[':
		for decoder.More() {
			if err := consumeUniqueJSONValue(decoder); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim(']') {
			return fmt.Errorf("unterminated array")
		}
	default:
		return fmt.Errorf("unexpected JSON delimiter")
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
