package workspace

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

func TestSQLiteGovernanceOutboxOwnsExplicitWorkerLifecycle(t *testing.T) {
	db := openWorkspaceTestDB(t)
	now := time.Unix(100, 0).UTC()
	if _, err := db.Exec(`INSERT INTO workspace_outbox_events(
		state,available_at,workspace_id,id,event_type,aggregate_kind,aggregate_id,aggregate_revision,payload_json,
		actor_type,actor_id,attempt_count,created_at
	) VALUES('ready',?,'workspace-1','event-1','task:created','task','task-1',7,'{"version":"governance-outbox-v1","data":{"id":"task-1"}}','member','member-1',0,?)`,
		now.Add(-time.Second).Format(time.RFC3339Nano), now.Add(-time.Second).Format(time.RFC3339Nano)); err != nil {
		t.Fatal(err)
	}
	module := New()
	sink := &workspaceRecordingOutboxSink{}
	outbox, err := NewSQLiteGovernanceOutbox(module, SqlitePersistenceConfig{DB: db}, GovernanceOutboxDependencies{
		Sink: sink, Authorizer: &workspaceAccessStub{}, EventPolicies: workspaceGovernanceEventPolicies{}, Memberships: governanceMemberships{},
		HTTPIdentity: func(*http.Request) (contract.WorkspaceHTTPIdentity, error) {
			return contract.WorkspaceHTTPIdentity{WorkspaceID: "workspace-1", ActorType: "member", ActorID: "member-1"}, nil
		},
		HTTPUserIdentity: func(*http.Request) (string, error) { return "user-1", nil },
		Now:              func() time.Time { return now },
		NewClaimToken:    func() string { return "claim-1" },
		PollInterval:     5 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	if outbox.Running() {
		t.Fatal("worker started during construction")
	}
	outbox.Start()
	deadline := time.Now().Add(time.Second)
	for sink.Count() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if sink.Count() != 1 {
		t.Fatalf("published events = %d", sink.Count())
	}
	diagnostics, err := outbox.ReadGovernanceDiagnostics(context.Background(), "workspace-1")
	if err != nil || !diagnostics.DispatcherRunning || diagnostics.LastSuccessfulDelivery.IsZero() {
		t.Fatalf("running diagnostics = %+v, %v", diagnostics, err)
	}
	if err := outbox.Close(); err != nil {
		t.Fatal(err)
	}
	if outbox.Running() {
		t.Fatal("worker remained running after close")
	}
	if err := outbox.Close(); err != nil {
		t.Fatalf("second close = %v", err)
	}
}

func TestSQLiteGovernanceOutboxRejectsMissingProvider(t *testing.T) {
	db := openWorkspaceTestDB(t)
	_, err := NewSQLiteGovernanceOutbox(New(), SqlitePersistenceConfig{DB: db}, GovernanceOutboxDependencies{
		Authorizer: &workspaceAccessStub{}, EventPolicies: workspaceGovernanceEventPolicies{}, Memberships: governanceMemberships{},
		HTTPIdentity: func(*http.Request) (contract.WorkspaceHTTPIdentity, error) {
			return contract.WorkspaceHTTPIdentity{}, nil
		},
		HTTPUserIdentity: func(*http.Request) (string, error) { return "user-1", nil },
		NewClaimToken:    func() string { return "claim-1" },
	})
	if !errors.Is(err, contract.ErrGovernanceUnavailable) {
		t.Fatalf("missing sink error = %v", err)
	}
}

type workspaceRecordingOutboxSink struct {
	mu     sync.Mutex
	events []contract.OutboxEvent
}

func (s *workspaceRecordingOutboxSink) Publish(_ context.Context, event contract.OutboxEvent) error {
	if !json.Valid(event.Payload) {
		return errors.New("invalid payload")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
	return nil
}

func (s *workspaceRecordingOutboxSink) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.events)
}

type governanceMemberships struct{}

type workspaceGovernanceEventPolicies struct{}

func (workspaceGovernanceEventPolicies) ResolveGovernanceEventPolicy(_ context.Context, eventType, aggregateKind string) (GovernanceEventPolicy, error) {
	return GovernanceEventPolicy{
		EventType: eventType, AggregateKind: aggregateKind,
		Schema: GovernanceEnvelopeSchema{"id": {Kind: GovernanceSafeIdentifier, MaxLength: 64, Required: true}},
	}, nil
}

func (governanceMemberships) ListForUser(context.Context, string) ([]contract.WorkspaceMembership, error) {
	return nil, nil
}
func (governanceMemberships) FindForUserAndWorkspace(context.Context, string, string) (contract.WorkspaceMembership, bool, error) {
	return contract.WorkspaceMembership{MemberID: "member-1", UserID: "user-1", WorkspaceID: "workspace-1", Role: "owner"}, true, nil
}
func (governanceMemberships) FindByMemberAndWorkspace(context.Context, string, string) (contract.WorkspaceMembership, bool, error) {
	return contract.WorkspaceMembership{MemberID: "member-1", UserID: "user-1", WorkspaceID: "workspace-1", Role: "owner"}, true, nil
}
