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
		Payload: []byte(`{"id":"task-1"}`), ActorType: "member", ActorID: "member-1", AttemptCount: 1,
		ClaimToken: "claim-1", LeaseExpiresAt: timePointer(now.Add(time.Minute)), CreatedAt: now,
	}}}
	service, err := NewOutboxService(OutboxServiceConfig{
		Repository: repository,
		Sink:       outboxSinkStub{err: errors.New("downstream unavailable")},
		Authorizer: outboxAuthorizerStub{},
		Now:        func() time.Time { return now },
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
		Payload: []byte(`{}`), ActorType: "member", ActorID: "member-1", AttemptCount: 4,
		ClaimToken: "claim-1", LeaseExpiresAt: timePointer(now.Add(time.Minute)), CreatedAt: now,
	}}}
	service, err := NewOutboxService(OutboxServiceConfig{
		Repository: repository, Sink: outboxSinkStub{err: errors.New("failed")}, Authorizer: outboxAuthorizerStub{},
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
		Payload: []byte(`{}`), ActorType: "member", ActorID: "member-1", AttemptCount: 1,
		ClaimToken: "claim-1", LeaseExpiresAt: timePointer(now.Add(time.Minute)), CreatedAt: now,
	}}}
	service, err := NewOutboxService(OutboxServiceConfig{
		Repository: repository, Sink: outboxSinkStub{}, Authorizer: outboxAuthorizerStub{},
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

func TestOutboxServiceCrashWindowRedeliversStableEventIdentity(t *testing.T) {
	now := time.Unix(100, 0).UTC()
	event := contract.OutboxEvent{
		State: contract.OutboxInflight, AvailableAt: now, WorkspaceID: "workspace-1", ID: "event-1",
		EventType: "task:created", AggregateKind: "task", AggregateID: "task-1", AggregateRevision: 7,
		Payload: []byte(`{}`), ActorType: "member", ActorID: "member-1", AttemptCount: 1,
		ClaimToken: "claim-1", LeaseExpiresAt: timePointer(now.Add(time.Minute)), CreatedAt: now,
	}
	repository := &outboxRepositoryStub{claimed: []contract.OutboxEvent{event}, deliveryErr: errors.New("ack interrupted")}
	sink := &recordingOutboxSink{}
	service, err := NewOutboxService(OutboxServiceConfig{
		Repository: repository, Sink: sink, Authorizer: outboxAuthorizerStub{},
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
		Repository: repository, Sink: outboxSinkStub{}, Authorizer: outboxAuthorizerStub{err: contract.ErrWorkspacePermissionDenied},
		Now: func() time.Time { return now }, NewClaimToken: func() string { return "claim-1" },
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Replay(context.Background(), "workspace-1", "event-1"); !errors.Is(err, contract.ErrWorkspacePermissionDenied) {
		t.Fatalf("denied replay error = %v", err)
	}
	if repository.replayed {
		t.Fatal("denied replay reached repository")
	}
	service.authorizer = outboxAuthorizerStub{}
	if err := service.Replay(context.Background(), "workspace-1", "event-1"); err != nil {
		t.Fatal(err)
	}
	if !repository.replayed {
		t.Fatal("authorized replay did not reach repository")
	}
}

type outboxRepositoryStub struct {
	claimed       []contract.OutboxEvent
	claimLimit    int
	claimLease    time.Duration
	delivered     bool
	failedDead    bool
	failedCode    string
	retryAt       time.Time
	diagnostics   contract.OutboxDiagnostics
	diagnosticErr error
	deliveryErr   error
	replayed      bool
}

func (s *outboxRepositoryStub) ClaimOutbox(_ context.Context, _ time.Time, limit int, lease time.Duration, _ string) ([]contract.OutboxEvent, error) {
	s.claimLimit, s.claimLease = limit, lease
	return s.claimed, nil
}
func (s *outboxRepositoryStub) MarkOutboxDelivered(context.Context, string, string, string, time.Time) error {
	s.delivered = true
	return s.deliveryErr
}
func (s *outboxRepositoryStub) MarkOutboxFailed(_ context.Context, _, _, _ string, _, retryAt time.Time, code string, dead bool) error {
	s.retryAt, s.failedCode, s.failedDead = retryAt, code, dead
	return nil
}

func (s *outboxRepositoryStub) ReplayOutbox(context.Context, string, string, time.Time) error {
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

func timePointer(value time.Time) *time.Time { return &value }
