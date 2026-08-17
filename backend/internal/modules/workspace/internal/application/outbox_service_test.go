package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

func TestOutboxServiceSchedulesDeterministicRetryWithinHardBatchCap(t *testing.T) {
	now := time.Unix(100, 0).UTC()
	repository := &outboxRepositoryStub{claimed: []contract.OutboxEvent{{
		State: contract.OutboxInflight, AvailableAt: now, WorkspaceID: "workspace-1", ID: "event-1",
		EventType: "task:created", AggregateKind: "task", AggregateID: "task-1", AggregateRevision: 7,
		Payload: []byte(`{"version":"governance-outbox-v1","data":{"id":"task-1"}}`), ActorType: "member", ActorID: "member-1", AttemptCount: 1,
		ClaimToken: "claim-1", LeaseExpiresAt: timePointer(now.Add(time.Minute)), CreatedAt: now,
	}}}
	service, err := NewOutboxService(OutboxServiceConfig{
		Repository:    repository,
		Sink:          outboxSinkStub{err: errors.New("downstream unavailable")},
		Authorizer:    outboxAuthorizerStub{},
		EventPolicies: governanceEventPolicyStub{},
		Now:           func() time.Time { return now },
		NewClaimToken: func() string {
			return "claim-1"
		},
		Jitter:    func(int) time.Duration { return 7 * time.Second },
		BatchSize: 700,
	})
	if err != nil {
		t.Fatal(err)
	}
	processed, err := service.DispatchOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if processed != 1 || repository.claimLimit != 500 || repository.claimLease != time.Minute {
		t.Fatalf("processed=%d limit=%d lease=%s", processed, repository.claimLimit, repository.claimLease)
	}
	if repository.failedDead || repository.failedCode != "publish_failed" || !repository.retryAt.Equal(now.Add(time.Minute+7*time.Second)) {
		t.Fatalf("failure transition = dead:%v code:%q retry:%s", repository.failedDead, repository.failedCode, repository.retryAt)
	}
}

func TestOutboxServiceDeadLettersFourthFailedAttempt(t *testing.T) {
	now := time.Unix(100, 0).UTC()
	repository := &outboxRepositoryStub{claimed: []contract.OutboxEvent{{
		State: contract.OutboxInflight, AvailableAt: now, WorkspaceID: "workspace-1", ID: "event-1",
		EventType: "task:created", AggregateKind: "task", AggregateID: "task-1", AggregateRevision: 7,
		Payload: []byte(`{"version":"governance-outbox-v1","data":{"id":"task-1"}}`), ActorType: "member", ActorID: "member-1", AttemptCount: 4,
		ClaimToken: "claim-1", LeaseExpiresAt: timePointer(now.Add(time.Minute)), CreatedAt: now,
	}}}
	service, err := NewOutboxService(OutboxServiceConfig{
		Repository: repository, Sink: outboxSinkStub{err: errors.New("failed")}, Authorizer: outboxAuthorizerStub{}, EventPolicies: governanceEventPolicyStub{},
		Now: func() time.Time { return now }, NewClaimToken: func() string { return "claim-1" },
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.DispatchOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !repository.failedDead {
		t.Fatal("fourth failed attempt was not dead-lettered")
	}
}

func TestOutboxServiceMarksSuccessfulPublishDelivered(t *testing.T) {
	now := time.Unix(100, 0).UTC()
	repository := &outboxRepositoryStub{claimed: []contract.OutboxEvent{{
		State: contract.OutboxInflight, AvailableAt: now, WorkspaceID: "workspace-1", ID: "event-1",
		EventType: "task:created", AggregateKind: "task", AggregateID: "task-1", AggregateRevision: 7,
		Payload: []byte(`{"version":"governance-outbox-v1","data":{"id":"task-1"}}`), ActorType: "member", ActorID: "member-1", AttemptCount: 1,
		ClaimToken: "claim-1", LeaseExpiresAt: timePointer(now.Add(time.Minute)), CreatedAt: now,
	}}}
	service, err := NewOutboxService(OutboxServiceConfig{
		Repository: repository, Sink: outboxSinkStub{}, Authorizer: outboxAuthorizerStub{}, EventPolicies: governanceEventPolicyStub{},
		Now: func() time.Time { return now }, NewClaimToken: func() string { return "claim-1" },
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.DispatchOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !repository.delivered {
		t.Fatal("successful publish was not marked delivered")
	}
}

func TestOutboxServiceUsesPostPublishClockAndCompleteClaimIdentity(t *testing.T) {
	claimedAt := time.Unix(100, 0).UTC()
	transitionedAt := claimedAt.Add(OutboxLeaseDuration + time.Second)
	clockCalls := 0
	repository := &outboxRepositoryStub{enforceCurrentLease: true, claimed: []contract.OutboxEvent{{
		State: contract.OutboxInflight, AvailableAt: claimedAt, WorkspaceID: "workspace-1", ID: "event-1",
		EventType: "task:created", AggregateKind: "task", AggregateID: "task-1", AggregateRevision: 7,
		Payload: []byte(`{"version":"governance-outbox-v1","data":{"id":"task-1"}}`), ActorType: "member", ActorID: "member-1", AttemptCount: 1,
		ClaimToken: "claim-1", LeaseExpiresAt: timePointer(claimedAt.Add(OutboxLeaseDuration)), CreatedAt: claimedAt,
	}}}
	service, err := NewOutboxService(OutboxServiceConfig{
		Repository: repository, Sink: outboxSinkStub{}, Authorizer: outboxAuthorizerStub{}, EventPolicies: governanceEventPolicyStub{},
		Now: func() time.Time {
			clockCalls++
			if clockCalls == 1 {
				return claimedAt
			}
			return transitionedAt
		},
		NewClaimToken: func() string { return "claim-1" },
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.DispatchOnce(context.Background()); !errors.Is(err, contract.ErrOutboxClaimConflict) {
		t.Fatalf("dispatch error = %v, want expired claim conflict", err)
	}
	if !repository.transitionedAt.Equal(transitionedAt) {
		t.Fatalf("transition time = %s, want %s", repository.transitionedAt, transitionedAt)
	}
	if repository.claimIdentity.State != contract.OutboxInflight ||
		!repository.claimIdentity.AvailableAt.Equal(claimedAt) ||
		repository.claimIdentity.WorkspaceID != "workspace-1" || repository.claimIdentity.ID != "event-1" ||
		repository.claimIdentity.ClaimToken != "claim-1" ||
		!repository.claimIdentity.LeaseExpiresAt.Equal(claimedAt.Add(OutboxLeaseDuration)) {
		t.Fatalf("claim identity = %+v", repository.claimIdentity)
	}
}

func TestOutboxServiceRejectsUnknownEventPolicyBeforePublish(t *testing.T) {
	now := time.Unix(100, 0).UTC()
	repository := &outboxRepositoryStub{claimed: []contract.OutboxEvent{{
		State: contract.OutboxInflight, AvailableAt: now, WorkspaceID: "workspace-1", ID: "event-1",
		EventType: "unknown:event", AggregateKind: "task", AggregateID: "task-1", AggregateRevision: 1,
		Payload: []byte(`{"version":"governance-outbox-v1","data":{"id":"task-1"}}`), ActorType: "member", ActorID: "user-1", AttemptCount: 1,
		ClaimToken: "claim-1", LeaseExpiresAt: timePointer(now.Add(OutboxLeaseDuration)), CreatedAt: now,
	}}}
	sink := &recordingOutboxSink{}
	service, err := NewOutboxService(OutboxServiceConfig{
		Repository: repository, Sink: sink, Authorizer: outboxAuthorizerStub{},
		EventPolicies: governanceEventPolicyStub{err: contract.ErrGovernanceUnavailable},
		Now:           func() time.Time { return now }, NewClaimToken: func() string { return "claim-1" },
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.DispatchOnce(context.Background()); !errors.Is(err, contract.ErrGovernanceUnavailable) {
		t.Fatalf("dispatch error = %v, want governance unavailable", err)
	}
	if len(sink.events) != 0 || repository.delivered || repository.failedCode != "" {
		t.Fatalf("unknown policy escaped: events=%d delivered=%v failure=%q", len(sink.events), repository.delivered, repository.failedCode)
	}
}

func TestOutboxServiceCrashWindowRedeliversStableEventIdentity(t *testing.T) {
	now := time.Unix(100, 0).UTC()
	event := contract.OutboxEvent{
		State: contract.OutboxInflight, AvailableAt: now, WorkspaceID: "workspace-1", ID: "event-1",
		EventType: "task:created", AggregateKind: "task", AggregateID: "task-1", AggregateRevision: 7,
		Payload: []byte(`{"version":"governance-outbox-v1","data":{"id":"task-1"}}`), ActorType: "member", ActorID: "member-1", AttemptCount: 1,
		ClaimToken: "claim-1", LeaseExpiresAt: timePointer(now.Add(time.Minute)), CreatedAt: now,
	}
	repository := &outboxRepositoryStub{claimed: []contract.OutboxEvent{event}, deliveryErr: errors.New("ack interrupted")}
	sink := &recordingOutboxSink{}
	service, err := NewOutboxService(OutboxServiceConfig{
		Repository: repository, Sink: sink, Authorizer: outboxAuthorizerStub{}, EventPolicies: governanceEventPolicyStub{},
		Now: func() time.Time { return now }, NewClaimToken: func() string { return "claim-1" },
	})
	if err != nil {
		t.Fatal(err)
	}
	for attempt := 0; attempt < 2; attempt++ {
		if _, err := service.DispatchOnce(context.Background()); err == nil {
			t.Fatal("expected simulated publish/ack crash window")
		}
	}
	if len(sink.events) != 2 || sink.events[0].ID != sink.events[1].ID || sink.events[0].AggregateRevision != sink.events[1].AggregateRevision {
		t.Fatalf("redelivered events = %+v", sink.events)
	}
}

func TestOutboxServiceReplayRequiresOperatorAuthority(t *testing.T) {
	now := time.Unix(100, 0).UTC()
	repository := &outboxRepositoryStub{}
	service, err := NewOutboxService(OutboxServiceConfig{
		Repository: repository, Sink: outboxSinkStub{}, Authorizer: outboxAuthorizerStub{err: contract.ErrWorkspacePermissionDenied}, EventPolicies: governanceEventPolicyStub{},
		Now: func() time.Time { return now }, NewClaimToken: func() string { return "claim-1" },
	})
	if err != nil {
		t.Fatal(err)
	}
	identity := contract.OutboxRowIdentity{State: contract.OutboxDeadLetter, AvailableAt: now, WorkspaceID: "workspace-1", ID: "event-1"}
	if err := service.Replay(context.Background(), identity); !errors.Is(err, contract.ErrWorkspacePermissionDenied) {
		t.Fatalf("denied replay error = %v", err)
	}
	if repository.replayed {
		t.Fatal("denied replay reached repository")
	}
	service.authorizer = outboxAuthorizerStub{}
	if err := service.Replay(context.Background(), identity); err != nil {
		t.Fatal(err)
	}
	if !repository.replayed {
		t.Fatal("authorized replay did not reach repository")
	}
}

type outboxRepositoryStub struct {
	claimed             []contract.OutboxEvent
	claimLimit          int
	claimLease          time.Duration
	delivered           bool
	failedDead          bool
	failedCode          string
	retryAt             time.Time
	diagnostics         contract.OutboxDiagnostics
	diagnosticErr       error
	deliveryErr         error
	replayed            bool
	transitionedAt      time.Time
	claimIdentity       contract.OutboxClaimIdentity
	enforceCurrentLease bool
}

func (s *outboxRepositoryStub) ClaimOutbox(_ context.Context, _ time.Time, limit int, lease time.Duration, _ string) ([]contract.OutboxEvent, error) {
	s.claimLimit, s.claimLease = limit, lease
	return s.claimed, nil
}
func (s *outboxRepositoryStub) MarkOutboxDelivered(_ context.Context, identity contract.OutboxClaimIdentity, transitionedAt time.Time) error {
	s.claimIdentity, s.transitionedAt = identity, transitionedAt
	if s.enforceCurrentLease && !transitionedAt.Before(identity.LeaseExpiresAt) {
		return contract.ErrOutboxClaimConflict
	}
	s.delivered = true
	return s.deliveryErr
}
func (s *outboxRepositoryStub) MarkOutboxFailed(_ context.Context, identity contract.OutboxClaimIdentity, transitionedAt, retryAt time.Time, code string, dead bool) error {
	s.claimIdentity, s.transitionedAt = identity, transitionedAt
	if s.enforceCurrentLease && !transitionedAt.Before(identity.LeaseExpiresAt) {
		return contract.ErrOutboxClaimConflict
	}
	s.retryAt, s.failedCode, s.failedDead = retryAt, code, dead
	return nil
}

func (s *outboxRepositoryStub) ReplayOutbox(context.Context, contract.OutboxRowIdentity, time.Time) error {
	s.replayed = true
	return nil
}
func (s *outboxRepositoryStub) ReadOutboxDiagnostics(context.Context, string, time.Time) (contract.OutboxDiagnostics, error) {
	return s.diagnostics, s.diagnosticErr
}

type outboxSinkStub struct{ err error }

func (s outboxSinkStub) Publish(context.Context, contract.OutboxEvent) error { return s.err }

type recordingOutboxSink struct{ events []contract.OutboxEvent }

func (s *recordingOutboxSink) Publish(_ context.Context, event contract.OutboxEvent) error {
	s.events = append(s.events, event)
	return nil
}

type outboxAuthorizerStub struct{ err error }

func (s outboxAuthorizerStub) AuthorizeWorkspace(context.Context, string, string) error { return s.err }

type governanceEventPolicyStub struct{ err error }

func (s governanceEventPolicyStub) ResolveGovernanceEventPolicy(_ context.Context, eventType, aggregateKind string) (GovernanceEventPolicy, error) {
	if s.err != nil {
		return GovernanceEventPolicy{}, s.err
	}
	return GovernanceEventPolicy{
		EventType: eventType, AggregateKind: aggregateKind,
		Schema: EnvelopeSchema{"id": {Kind: SafeIdentifier, MaxLength: 64, Required: true}},
	}, nil
}

func timePointer(value time.Time) *time.Time { return &value }
