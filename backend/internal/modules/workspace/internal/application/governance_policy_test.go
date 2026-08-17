package application

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

func TestGovernanceServiceFailsClosedWithoutPolicyProvider(t *testing.T) {
	service := NewGovernanceService()
	_, err := service.PrepareContext(context.Background(), GovernanceRequest{})
	if !errors.Is(err, contract.ErrGovernanceUnavailable) {
		t.Fatalf("PrepareContext() error = %v, want %v", err, contract.ErrGovernanceUnavailable)
	}
}

func TestPreparedGovernanceMutationHasNoCallerWritableFields(t *testing.T) {
	typeOfPrepared := reflect.TypeOf(PreparedGovernanceMutation{})
	for index := 0; index < typeOfPrepared.NumField(); index++ {
		if field := typeOfPrepared.Field(index); field.IsExported() {
			t.Fatalf("PreparedGovernanceMutation field %q is exported", field.Name)
		}
	}
}

func TestGovernanceInputsExposeNoUnrestrictedRawEnvelopeFields(t *testing.T) {
	for _, candidate := range []struct {
		value     any
		forbidden map[string]struct{}
	}{
		{value: GovernanceRequest{}, forbidden: map[string]struct{}{"ResponseBody": {}, "AuditMetadata": {}}},
		{value: OutboxDraft{}, forbidden: map[string]struct{}{"Payload": {}}},
	} {
		typeOfCandidate := reflect.TypeOf(candidate.value)
		for index := 0; index < typeOfCandidate.NumField(); index++ {
			field := typeOfCandidate.Field(index)
			if _, forbidden := candidate.forbidden[field.Name]; forbidden {
				t.Fatalf("%s still exposes unrestricted field %s", typeOfCandidate.Name(), field.Name)
			}
		}
	}
}

func TestPreparedGovernanceMutationGettersDoNotExposeEnvelopeStorage(t *testing.T) {
	policy := taskGovernancePolicy()
	policy.RequestSchema["id"] = SafeFieldRule{Kind: SafeIdentifier, MaxLength: 64, Required: true}
	prepared, err := NewGovernanceService(governancePolicyProviderStub{
		identity: contract.MutationIdentity{WorkspaceID: "workspace-1", ActorType: "member", ActorID: "user-1", RequestID: "request-1"},
		policy:   policy,
	}).PrepareContext(context.Background(), GovernanceRequest{
		Identity:       contract.MutationIdentity{WorkspaceID: "workspace-1", RequestID: "request-1"},
		Command:        contract.MutationCommand{Action: "workspace.task.create", ResourceKind: "task", ResourceID: "task-1"},
		RequestFields:  map[string]any{"id": "task-1"},
		ResponseStatus: 201,
		ResponseFields: map[string]any{"id": "task-1"},
		AuditID:        "audit-1",
		OccurredAt:     time.Unix(1, 0).UTC(),
		AuditFields:    map[string]any{"status": "todo"},
		Outbox:         []OutboxDraft{{ID: "event-1", EventType: "task:created", Fields: map[string]any{"id": "task-1"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	result := prepared.Result()
	audit := prepared.Audit()
	outbox := prepared.Outbox()
	result.ResponseBody[0] = '['
	audit.Metadata[0] = '['
	outbox[0].Payload[0] = '['
	if prepared.Result().ResponseBody[0] != '{' || prepared.Audit().Metadata[0] != '{' || prepared.Outbox()[0].Payload[0] != '{' {
		t.Fatal("getter mutation changed prepared envelope storage")
	}
}

func TestPreparedGovernanceMutationFreezesResolvedPolicySnapshot(t *testing.T) {
	policy := taskGovernancePolicy()
	provider := governancePolicyProviderStub{
		identity: contract.MutationIdentity{WorkspaceID: "workspace-1", ActorType: "member", ActorID: "user-1", RequestID: "request-1"},
		policy:   policy,
	}
	prepared, err := NewGovernanceService(provider).PrepareContext(context.Background(), GovernanceRequest{
		Identity:       contract.MutationIdentity{WorkspaceID: "workspace-1", RequestID: "request-1"},
		Command:        contract.MutationCommand{Action: "workspace.task.create", ResourceKind: "task", ResourceID: "task-1"},
		RequestFields:  map[string]any{"id": "task-1", "status": "todo"},
		ResponseStatus: 201,
		ResponseFields: map[string]any{"id": "task-1"},
		AuditID:        "audit-1",
		OccurredAt:     time.Unix(1, 0).UTC(),
		AuditFields:    map[string]any{"status": "todo"},
		Outbox:         []OutboxDraft{{ID: "event-1", EventType: "task:created", Fields: map[string]any{"id": "task-1"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	policy.ReplaySchema["id"] = SafeFieldRule{Kind: SafeBoolean, Required: true}
	policy.EventSchemas["task:created"]["id"] = SafeFieldRule{Kind: SafeBoolean, Required: true}
	auditRule := policy.AuditSchema["status"]
	auditRule.EnumValues[0] = "changed"
	if err := prepared.Validate(); err != nil {
		t.Fatalf("provider mutation changed prepared policy: %v", err)
	}
}

func taskGovernancePolicy() GovernanceActionPolicy {
	return GovernanceActionPolicy{
		Action:       "workspace.task.create",
		ResourceKind: "task",
		RequestSchema: EnvelopeSchema{
			"id":     {Kind: SafeIdentifier, MaxLength: 64},
			"status": {Kind: SafeEnum, EnumValues: []string{"todo", "done"}},
		},
		ReplaySchema: EnvelopeSchema{
			"id": {Kind: SafeIdentifier, MaxLength: 64},
		},
		AuditSchema: EnvelopeSchema{
			"status":      {Kind: SafeEnum, EnumValues: []string{"todo", "done"}},
			"reason_code": {Kind: SafeIdentifier, MaxLength: 64},
		},
		EventSchemas: map[string]EnvelopeSchema{
			"task:created": {"id": {Kind: SafeIdentifier, MaxLength: 64}},
		},
	}
}

func TestGovernanceServiceRejectsSecretBearingValuesInEveryEnvelope(t *testing.T) {
	base := GovernanceRequest{
		Identity:       contract.MutationIdentity{WorkspaceID: "workspace-1", RequestID: "request-1"},
		Command:        contract.MutationCommand{Action: "workspace.task.create", ResourceKind: "task", ResourceID: "task-1", IdempotencyKey: "command-1"},
		RequestFields:  map[string]any{"id": "task-1", "status": "todo"},
		ResponseStatus: 201,
		ResponseFields: map[string]any{"id": "task-1"},
		AuditID:        "audit-1",
		OccurredAt:     time.Unix(1, 0).UTC(),
		AuditFields:    map[string]any{"status": "todo"},
		Outbox:         []OutboxDraft{{ID: "event-1", EventType: "task:created", Fields: map[string]any{"id": "task-1"}}},
	}
	tests := []struct {
		name   string
		mutate func(*GovernanceRequest)
	}{
		{name: "request", mutate: func(request *GovernanceRequest) { request.RequestFields["id"] = "secret-value" }},
		{name: "replay", mutate: func(request *GovernanceRequest) { request.ResponseFields["id"] = "Bearer-secret" }},
		{name: "audit", mutate: func(request *GovernanceRequest) { request.AuditFields["reason_code"] = "token-value" }},
		{name: "outbox", mutate: func(request *GovernanceRequest) { request.Outbox[0].Fields["id"] = "cookie-value" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := cloneGovernanceRequest(base)
			test.mutate(&request)
			provider := governancePolicyProviderStub{
				identity: contract.MutationIdentity{WorkspaceID: "workspace-1", ActorType: "member", ActorID: "user-1", RequestID: "request-1"},
				policy:   taskGovernancePolicy(),
			}
			_, err := NewGovernanceService(provider).PrepareContext(context.Background(), request)
			if !errors.Is(err, contract.ErrInvalidGovernanceMutation) {
				t.Fatalf("PrepareContext() error = %v, want %v", err, contract.ErrInvalidGovernanceMutation)
			}
		})
	}
}

func cloneGovernanceRequest(request GovernanceRequest) GovernanceRequest {
	clone := request
	clone.RequestFields = map[string]any{"id": request.RequestFields["id"], "status": request.RequestFields["status"]}
	clone.ResponseFields = map[string]any{"id": request.ResponseFields["id"]}
	clone.AuditFields = map[string]any{"status": request.AuditFields["status"]}
	clone.Outbox = []OutboxDraft{{ID: request.Outbox[0].ID, EventType: request.Outbox[0].EventType, Fields: map[string]any{"id": request.Outbox[0].Fields["id"]}}}
	return clone
}

type governancePolicyProviderStub struct {
	identity contract.MutationIdentity
	policy   GovernanceActionPolicy
	err      error
}

func TestGovernanceServiceComputesCanonicalVersionedRequestHash(t *testing.T) {
	provider := governancePolicyProviderStub{
		identity: contract.MutationIdentity{WorkspaceID: "workspace-1", ActorType: "member", ActorID: "user-1", RequestID: "request-1"},
		policy:   taskGovernancePolicy(),
	}
	service := NewGovernanceService(provider)
	prepared, err := service.PrepareContext(context.Background(), GovernanceRequest{
		Identity:       contract.MutationIdentity{WorkspaceID: "workspace-1", RequestID: "request-1"},
		Command:        contract.MutationCommand{Action: "workspace.task.create", ResourceKind: "task", ResourceID: "task-1", IdempotencyKey: "command-1"},
		RequestFields:  map[string]any{"status": "todo", "id": "task-1"},
		ResponseStatus: 201,
		ResponseFields: map[string]any{"id": "task-1"},
		AuditID:        "audit-1",
		OccurredAt:     time.Unix(1, 0).UTC(),
		AuditFields:    map[string]any{"status": "todo"},
		Outbox:         []OutboxDraft{{ID: "event-1", EventType: "task:created", Fields: map[string]any{"id": "task-1"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	canonical := `{"version":"governance-request-v1","action":"workspace.task.create","request":{"id":"task-1","status":"todo"}}`
	want := fmt.Sprintf("%x", sha256.Sum256([]byte(canonical)))
	if prepared.Command().RequestHash != want {
		t.Fatalf("request hash = %s, want %s", prepared.Command().RequestHash, want)
	}
}

func TestGovernanceServiceRejectsCallerSuppliedRequestHash(t *testing.T) {
	provider := governancePolicyProviderStub{
		identity: contract.MutationIdentity{WorkspaceID: "workspace-1", ActorType: "member", ActorID: "user-1", RequestID: "request-1"},
		policy:   taskGovernancePolicy(),
	}
	service := NewGovernanceService(provider)
	_, err := service.PrepareContext(context.Background(), GovernanceRequest{
		Identity:       contract.MutationIdentity{WorkspaceID: "workspace-1", RequestID: "request-1"},
		Command:        contract.MutationCommand{Action: "workspace.task.create", ResourceKind: "task", ResourceID: "task-1", IdempotencyKey: "command-1", RequestHash: fmt.Sprintf("%064x", 1)},
		RequestFields:  map[string]any{"id": "task-1", "status": "todo"},
		ResponseStatus: 201,
		ResponseFields: map[string]any{"id": "task-1"},
		AuditID:        "audit-1",
		OccurredAt:     time.Unix(1, 0).UTC(),
		AuditFields:    map[string]any{"status": "todo"},
	})
	if !errors.Is(err, contract.ErrInvalidGovernanceMutation) {
		t.Fatalf("PrepareContext() error = %v, want %v", err, contract.ErrInvalidGovernanceMutation)
	}
}

func TestGovernanceServicePersistsVersionedSafeEnvelopes(t *testing.T) {
	provider := governancePolicyProviderStub{
		identity: contract.MutationIdentity{WorkspaceID: "workspace-1", ActorType: "member", ActorID: "user-1", RequestID: "request-1"},
		policy:   taskGovernancePolicy(),
	}
	prepared, err := NewGovernanceService(provider).PrepareContext(context.Background(), GovernanceRequest{
		Identity:       contract.MutationIdentity{WorkspaceID: "workspace-1", RequestID: "request-1"},
		Command:        contract.MutationCommand{Action: "workspace.task.create", ResourceKind: "task", ResourceID: "task-1", IdempotencyKey: "command-1"},
		RequestFields:  map[string]any{"id": "task-1", "status": "todo"},
		ResponseStatus: 201,
		ResponseFields: map[string]any{"id": "task-1"},
		AuditID:        "audit-1",
		OccurredAt:     time.Unix(1, 0).UTC(),
		AuditFields:    map[string]any{"status": "todo"},
		Outbox:         []OutboxDraft{{ID: "event-1", EventType: "task:created", Fields: map[string]any{"id": "task-1"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := string(prepared.Result().ResponseBody); got != `{"version":"governance-replay-v1","data":{"id":"task-1"}}` {
		t.Fatalf("replay envelope = %s", got)
	}
	if got := string(prepared.Audit().Metadata); got != `{"version":"governance-audit-v1","data":{"status":"todo"}}` {
		t.Fatalf("audit envelope = %s", got)
	}
	if got := string(prepared.Outbox()[0].Payload); got != `{"version":"governance-outbox-v1","data":{"id":"task-1"}}` {
		t.Fatalf("outbox envelope = %s", got)
	}
}

func (s governancePolicyProviderStub) ResolveGovernancePolicy(context.Context, string, string, string, string) (contract.MutationIdentity, GovernanceActionPolicy, error) {
	return s.identity, s.policy, s.err
}

func TestGovernanceServiceUsesServerResolvedPolicyAndIdentity(t *testing.T) {
	provider := governancePolicyProviderStub{
		identity: contract.MutationIdentity{WorkspaceID: "workspace-1", ActorType: "member", ActorID: "user-1", RequestID: "request-1"},
		policy:   taskGovernancePolicy(),
	}
	service := NewGovernanceService(provider)
	prepared, err := service.PrepareContext(context.Background(), GovernanceRequest{
		Identity:       contract.MutationIdentity{WorkspaceID: "workspace-1", RequestID: "request-1"},
		Command:        contract.MutationCommand{Action: "workspace.task.create", ResourceKind: "task", ResourceID: "task-1"},
		RequestFields:  map[string]any{"id": "task-1", "status": "todo"},
		ResponseStatus: 201,
		ResponseFields: map[string]any{"id": "task-1"},
		AuditID:        "audit-1",
		OccurredAt:     time.Unix(1, 0).UTC(),
		AuditFields:    map[string]any{"status": "todo"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if prepared.Identity() != provider.identity {
		t.Fatalf("prepared identity = %+v, want %+v", prepared.Identity(), provider.identity)
	}
}

func TestGovernanceServiceRejectsMismatchedOrForeignPolicyResolution(t *testing.T) {
	tests := []struct {
		name     string
		identity contract.MutationIdentity
		policy   GovernanceActionPolicy
	}{
		{
			name:     "mismatched resource",
			identity: contract.MutationIdentity{WorkspaceID: "workspace-1", ActorType: "member", ActorID: "user-1", RequestID: "request-1"},
			policy:   GovernanceActionPolicy{Action: "workspace.task.create", ResourceKind: "credential"},
		},
		{
			name:     "foreign workspace actor",
			identity: contract.MutationIdentity{WorkspaceID: "workspace-2", ActorType: "member", ActorID: "user-1", RequestID: "request-1"},
			policy:   GovernanceActionPolicy{Action: "workspace.task.create", ResourceKind: "task"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewGovernanceService(governancePolicyProviderStub{identity: tt.identity, policy: tt.policy})
			_, err := service.PrepareContext(context.Background(), GovernanceRequest{
				Identity: contract.MutationIdentity{WorkspaceID: "workspace-1", RequestID: "request-1"},
				Command:  contract.MutationCommand{Action: "workspace.task.create", ResourceKind: "task", ResourceID: "task-1"},
			})
			if !errors.Is(err, contract.ErrGovernanceUnavailable) {
				t.Fatalf("PrepareContext() error = %v, want %v", err, contract.ErrGovernanceUnavailable)
			}
		})
	}
}
