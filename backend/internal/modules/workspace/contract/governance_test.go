package contract

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestMutationIdentityValidate(t *testing.T) {
	tests := []struct {
		name     string
		identity MutationIdentity
		wantErr  bool
	}{
		{name: "member", identity: MutationIdentity{WorkspaceID: "workspace-1", ActorType: "member", ActorID: "member-1", RequestID: "request-1"}},
		{name: "agent", identity: MutationIdentity{WorkspaceID: "workspace-1", ActorType: "agent", ActorID: "agent-1", RequestID: "request-1"}},
		{name: "missing workspace", identity: MutationIdentity{ActorType: "member", ActorID: "member-1", RequestID: "request-1"}, wantErr: true},
		{name: "untrusted actor type", identity: MutationIdentity{WorkspaceID: "workspace-1", ActorType: "service", ActorID: "service-1", RequestID: "request-1"}, wantErr: true},
		{name: "missing actor", identity: MutationIdentity{WorkspaceID: "workspace-1", ActorType: "member", RequestID: "request-1"}, wantErr: true},
		{name: "missing request", identity: MutationIdentity{WorkspaceID: "workspace-1", ActorType: "member", ActorID: "member-1"}, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.identity.Validate()
			if (err != nil) != test.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, test.wantErr)
			}
			if test.wantErr && !errors.Is(err, ErrInvalidGovernanceMutation) {
				t.Fatalf("Validate() error = %v, want %v", err, ErrInvalidGovernanceMutation)
			}
		})
	}
}

func TestMutationResultValidate(t *testing.T) {
	valid := MutationResult{ResourceRevision: 1, ResponseStatus: 201, ResponseBody: json.RawMessage(`{"version":"governance-replay-v1","data":{"id":"task-1"}}`)}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid result: %v", err)
	}
	for _, test := range []struct {
		name   string
		result MutationResult
	}{
		{name: "zero revision", result: MutationResult{ResponseStatus: 200, ResponseBody: json.RawMessage(`{}`)}},
		{name: "invalid status", result: MutationResult{ResourceRevision: 1, ResponseStatus: 99, ResponseBody: json.RawMessage(`{}`)}},
		{name: "invalid json", result: MutationResult{ResourceRevision: 1, ResponseStatus: 200, ResponseBody: json.RawMessage(`{`)}},
		{name: "oversized response", result: MutationResult{ResourceRevision: 1, ResponseStatus: 200, ResponseBody: json.RawMessage(`"` + string(bytes.Repeat([]byte("x"), MaxReplayResponseBytes)) + `"`)}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := test.result.Validate(); err == nil {
				t.Fatal("Validate() error = nil")
			} else if test.name == "oversized response" && !errors.Is(err, ErrIdempotencyResponseTooLarge) {
				t.Fatalf("Validate() error = %v, want %v", err, ErrIdempotencyResponseTooLarge)
			}
		})
	}
}

func TestAuditRecordValidate(t *testing.T) {
	identity := MutationIdentity{WorkspaceID: "workspace-1", ActorType: "member", ActorID: "member-1", RequestID: "request-1"}
	valid := AuditRecord{ID: "audit-1", Identity: identity, Action: "workspace.task.create", ResourceKind: "task", ResourceID: "task-1", ResourceRevision: 1, OccurredAt: time.Unix(1, 0).UTC(), Metadata: json.RawMessage(`{"version":"governance-audit-v1","data":{"status":"todo"}}`)}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid audit: %v", err)
	}
	for _, mutate := range []func(*AuditRecord){
		func(value *AuditRecord) { value.ID = "" },
		func(value *AuditRecord) { value.ResourceRevision = 0 },
		func(value *AuditRecord) { value.OccurredAt = time.Time{} },
		func(value *AuditRecord) { value.Metadata = json.RawMessage(`[]`) },
		func(value *AuditRecord) {
			value.Metadata = json.RawMessage(`{"value":"` + string(bytes.Repeat([]byte("x"), MaxAuditMetadataBytes)) + `"}`)
		},
	} {
		candidate := valid
		mutate(&candidate)
		if err := candidate.Validate(); !errors.Is(err, ErrInvalidGovernanceMutation) {
			t.Errorf("Validate() error = %v, want %v", err, ErrInvalidGovernanceMutation)
		}
	}
}

func TestOutboxEventValidate(t *testing.T) {
	now := time.Unix(1, 0).UTC()
	lease := now.Add(time.Minute)
	valid := OutboxEvent{State: OutboxReady, AvailableAt: now, WorkspaceID: "workspace-1", ID: "event-1", EventType: "task:created", AggregateKind: "task", AggregateID: "task-1", AggregateRevision: 1, Payload: json.RawMessage(`{"version":"governance-outbox-v1","data":{"id":"task-1"}}`), ActorType: "member", ActorID: "member-1", CreatedAt: now}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid event: %v", err)
	}
	inflight := valid
	inflight.State, inflight.ClaimToken, inflight.LeaseExpiresAt = OutboxInflight, "claim-1", &lease
	if err := inflight.Validate(); err != nil {
		t.Fatalf("valid inflight event: %v", err)
	}
	for _, mutate := range []func(*OutboxEvent){
		func(value *OutboxEvent) { value.State = OutboxState("unknown") },
		func(value *OutboxEvent) { value.WorkspaceID = "" },
		func(value *OutboxEvent) { value.AggregateRevision = 0 },
		func(value *OutboxEvent) { value.Payload = json.RawMessage(`{`) },
		func(value *OutboxEvent) {
			value.Payload = json.RawMessage(`"` + string(bytes.Repeat([]byte("x"), MaxOutboxPayloadBytes)) + `"`)
		},
		func(value *OutboxEvent) { value.AttemptCount = -1 },
		func(value *OutboxEvent) { value.State = OutboxInflight },
		func(value *OutboxEvent) { value.ClaimToken, value.LeaseExpiresAt = "claim-1", &lease },
	} {
		candidate := valid
		mutate(&candidate)
		if err := candidate.Validate(); !errors.Is(err, ErrInvalidGovernanceMutation) {
			t.Errorf("Validate() error = %v, want %v", err, ErrInvalidGovernanceMutation)
		}
	}
}

func TestGovernanceEnvelopesRejectLegacyWrongVersionAndExtraTopLevelFields(t *testing.T) {
	identity := MutationIdentity{WorkspaceID: "workspace-1", ActorType: "member", ActorID: "user-1", RequestID: "request-1"}
	now := time.Unix(1, 0).UTC()
	tests := []struct {
		name     string
		validate func(json.RawMessage) error
	}{
		{name: "replay", validate: func(raw json.RawMessage) error {
			return (MutationResult{ResourceRevision: 1, ResponseStatus: 200, ResponseBody: raw}).Validate()
		}},
		{name: "audit", validate: func(raw json.RawMessage) error {
			return (AuditRecord{ID: "audit-1", Identity: identity, Action: "workspace.task.create", ResourceKind: "task", ResourceID: "task-1", ResourceRevision: 1, OccurredAt: now, Metadata: raw}).Validate()
		}},
		{name: "outbox", validate: func(raw json.RawMessage) error {
			return (OutboxEvent{State: OutboxReady, AvailableAt: now, WorkspaceID: "workspace-1", ID: "event-1", EventType: "task:created", AggregateKind: "task", AggregateID: "task-1", AggregateRevision: 1, Payload: raw, ActorType: "member", ActorID: "user-1", CreatedAt: now}).Validate()
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			version := "governance-" + test.name + "-v1"
			for _, raw := range []json.RawMessage{
				json.RawMessage(`{"id":"legacy"}`),
				json.RawMessage(`{"version":"wrong-v1","data":{}}`),
				json.RawMessage(`{"version":"` + version + `","data":{},"extra":true}`),
				json.RawMessage(`{"version":"` + version + `","version":"` + version + `","data":{}}`),
				json.RawMessage(`{"version":"` + version + `","data":{"id":"one","id":"two"}}`),
				json.RawMessage(`{"version":"` + version + `","data":{"id":"secret-value"}}`),
				json.RawMessage(`{"version":"` + version + `","data":{"access_token":"value"}}`),
			} {
				if err := test.validate(raw); !errors.Is(err, ErrInvalidGovernanceMutation) {
					t.Fatalf("Validate(%s) error = %v, want %v", raw, err, ErrInvalidGovernanceMutation)
				}
			}
		})
	}
}

func TestGovernanceStableErrors(t *testing.T) {
	for _, err := range []error{
		ErrRevisionConflict,
		ErrIdempotencyConflict,
		ErrIdempotencyResponseTooLarge,
		ErrGovernanceUnavailable,
		ErrOutboxClaimConflict,
	} {
		if err == nil || err.Error() == "" {
			t.Fatalf("stable governance error = %v", err)
		}
	}
	conflict := RevisionConflictError{CurrentRevision: 7}
	if !errors.Is(conflict, ErrRevisionConflict) || conflict.CurrentRevision != 7 {
		t.Fatalf("revision conflict = %#v", conflict)
	}
}

func TestOutboxDiagnosticsValidate(t *testing.T) {
	valid := OutboxDiagnostics{ReadyCount: 1, OldestReadyAge: time.Minute, InflightCount: 2, RetryWaitCount: 3, DeadLetterCount: 4}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid diagnostics: %v", err)
	}
	for _, diagnostics := range []OutboxDiagnostics{
		{ReadyCount: -1},
		{OldestReadyAge: -time.Second},
		{InflightCount: -1},
		{RetryWaitCount: -1},
		{DeadLetterCount: -1},
	} {
		if err := diagnostics.Validate(); !errors.Is(err, ErrInvalidGovernanceMutation) {
			t.Errorf("Validate() error = %v, want %v", err, ErrInvalidGovernanceMutation)
		}
	}
}

func TestMutationCommandValidate(t *testing.T) {
	validHash := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	tests := []struct {
		name    string
		command MutationCommand
		wantErr bool
	}{
		{name: "revision only", command: MutationCommand{Action: "workspace.task.create", ResourceKind: "task", ResourceID: "task-1", ExpectedRevision: 0}},
		{name: "idempotent", command: MutationCommand{Action: "workspace.task.create", ResourceKind: "task", ResourceID: "task-1", ExpectedRevision: 0, IdempotencyKey: "command-1", RequestHash: validHash}},
		{name: "missing action", command: MutationCommand{ResourceKind: "task", ResourceID: "task-1"}, wantErr: true},
		{name: "missing resource kind", command: MutationCommand{Action: "workspace.task.create", ResourceID: "task-1"}, wantErr: true},
		{name: "missing resource id", command: MutationCommand{Action: "workspace.task.create", ResourceKind: "task"}, wantErr: true},
		{name: "negative revision", command: MutationCommand{Action: "workspace.task.create", ResourceKind: "task", ResourceID: "task-1", ExpectedRevision: -1}, wantErr: true},
		{name: "key without hash", command: MutationCommand{Action: "workspace.task.create", ResourceKind: "task", ResourceID: "task-1", IdempotencyKey: "command-1"}, wantErr: true},
		{name: "invalid hash", command: MutationCommand{Action: "workspace.task.create", ResourceKind: "task", ResourceID: "task-1", IdempotencyKey: "command-1", RequestHash: "not-sha256"}, wantErr: true},
		{name: "hash without key", command: MutationCommand{Action: "workspace.task.create", ResourceKind: "task", ResourceID: "task-1", RequestHash: validHash}, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.command.Validate()
			if (err != nil) != test.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}
