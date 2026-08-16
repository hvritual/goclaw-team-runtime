package application

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

func TestGovernanceServicePreparesRevisionAuditAndOutboxEnvelope(t *testing.T) {
	service := NewGovernanceService()
	prepared, err := service.Prepare(GovernanceRequest{
		Identity:       contract.MutationIdentity{WorkspaceID: "workspace-1", ActorType: "member", ActorID: "member-1", RequestID: "request-1"},
		Command:        contract.MutationCommand{Action: "workspace.task.create", ResourceKind: "task", ResourceID: "task-1", ExpectedRevision: 0, IdempotencyKey: "command-1", RequestHash: strings.Repeat("a", 64)},
		ResponseStatus: 201,
		ResponseBody:   json.RawMessage(`{"id":"task-1"}`),
		AuditID:        "audit-1",
		OccurredAt:     time.Unix(1, 0).UTC(),
		AuditMetadata:  map[string]string{"status": "todo", "access_token": "secret"},
		Outbox:         []OutboxDraft{{ID: "event-1", EventType: "task:created", Payload: json.RawMessage(`{"id":"task-1"}`)}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if prepared.Result.ResourceRevision != 1 || prepared.Result.Replayed {
		t.Fatalf("result = %+v", prepared.Result)
	}
	if string(prepared.Audit.Metadata) != `{"status":"todo"}` {
		t.Fatalf("audit metadata = %s", prepared.Audit.Metadata)
	}
	if len(prepared.Outbox) != 1 || prepared.Outbox[0].AggregateRevision != 1 || prepared.Outbox[0].ActorID != "member-1" {
		t.Fatalf("outbox = %+v", prepared.Outbox)
	}
}
