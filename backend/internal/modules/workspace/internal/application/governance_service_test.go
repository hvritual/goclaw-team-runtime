package application

import (
	"context"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

func TestGovernanceServicePreparesRevisionAuditAndOutboxEnvelope(t *testing.T) {
	provider := governancePolicyProviderStub{
		identity: contract.MutationIdentity{WorkspaceID: "workspace-1", ActorType: "member", ActorID: "user-1", RequestID: "request-1"},
		policy:   taskGovernancePolicy(),
	}
	service := NewGovernanceService(provider)
	prepared, err := service.PrepareContext(context.Background(), GovernanceRequest{
		Identity:       contract.MutationIdentity{WorkspaceID: "workspace-1", RequestID: "request-1"},
		Command:        contract.MutationCommand{Action: "workspace.task.create", ResourceKind: "task", ResourceID: "task-1", ExpectedRevision: 0, IdempotencyKey: "command-1"},
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
	if prepared.Result().ResourceRevision != 1 || prepared.Result().Replayed {
		t.Fatalf("result = %+v", prepared.Result())
	}
	if string(prepared.Audit().Metadata) != `{"version":"governance-audit-v1","data":{"status":"todo"}}` {
		t.Fatalf("audit metadata = %s", prepared.Audit().Metadata)
	}
	outbox := prepared.Outbox()
	if len(outbox) != 1 || outbox[0].AggregateRevision != 1 || outbox[0].ActorID != "user-1" {
		t.Fatalf("outbox = %+v", outbox)
	}
}
