package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

const (
	DefaultOutboxBatchSize = 100
	MaxOutboxBatchSize     = 500
	OutboxLeaseDuration    = time.Minute
	MaxOutboxAttempts      = 4
)

type OutboxRepository interface {
	ClaimOutbox(context.Context, time.Time, int, time.Duration, string) ([]contract.OutboxEvent, error)
	MarkOutboxDelivered(context.Context, string, string, string, time.Time) error
	MarkOutboxFailed(context.Context, string, string, string, time.Time, time.Time, string, bool) error
	ReplayOutbox(context.Context, string, string, time.Time) error
	ReadOutboxDiagnostics(context.Context, string, time.Time) (contract.OutboxDiagnostics, error)
}

type OutboxServiceConfig struct {
	Repository    OutboxRepository
	Sink          contract.OutboxSink
	Authorizer    contract.WorkspaceAccessAuthorizer
	Now           func() time.Time
	NewClaimToken func() string
	Jitter        func(int) time.Duration
	BatchSize     int
}

type OutboxService struct {
	repository        OutboxRepository
	sink              contract.OutboxSink
	authorizer        contract.WorkspaceAccessAuthorizer
	now               func() time.Time
	newClaimToken     func() string
	jitter            func(int) time.Duration
	batchSize         int
	dispatcherRunning func() bool
}

func NewOutboxService(config OutboxServiceConfig) (*OutboxService, error) {
	if config.Repository == nil || config.Sink == nil || config.Authorizer == nil {
		return nil, contract.ErrGovernanceUnavailable
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	if config.NewClaimToken == nil {
		return nil, fmt.Errorf("%w: claim token generator is required", contract.ErrGovernanceUnavailable)
	}
	if config.Jitter == nil {
		config.Jitter = func(int) time.Duration { return 0 }
	}
	batchSize := config.BatchSize
	if batchSize <= 0 {
		batchSize = DefaultOutboxBatchSize
	}
	if batchSize > MaxOutboxBatchSize {
		batchSize = MaxOutboxBatchSize
	}
	return &OutboxService{
		repository: config.Repository, sink: config.Sink, authorizer: config.Authorizer,
		now: config.Now, newClaimToken: config.NewClaimToken, jitter: config.Jitter, batchSize: batchSize,
	}, nil
}

func (s *OutboxService) DispatchOnce(ctx context.Context) (int, error) {
	now := s.now().UTC()
	claimToken := strings.TrimSpace(s.newClaimToken())
	if claimToken == "" {
		return 0, fmt.Errorf("%w: empty claim token", contract.ErrGovernanceUnavailable)
	}
	events, err := s.repository.ClaimOutbox(ctx, now, s.batchSize, OutboxLeaseDuration, claimToken)
	if err != nil {
		return 0, fmt.Errorf("claim outbox batch: %w", err)
	}
	for _, event := range events {
		if err := s.sink.Publish(ctx, event); err == nil {
			if err := s.repository.MarkOutboxDelivered(ctx, event.WorkspaceID, event.ID, event.ClaimToken, now); err != nil {
				return 0, fmt.Errorf("acknowledge outbox delivery: %w", err)
			}
			continue
		}
		dead := event.AttemptCount >= MaxOutboxAttempts
		retryAt := now
		if !dead {
			retryAt = now.Add(outboxRetryDelay(event.AttemptCount) + nonNegativeJitter(s.jitter(event.AttemptCount)))
		}
		if err := s.repository.MarkOutboxFailed(ctx, event.WorkspaceID, event.ID, event.ClaimToken, now, retryAt, "publish_failed", dead); err != nil {
			return 0, fmt.Errorf("record outbox delivery failure: %w", err)
		}
	}
	return len(events), nil
}

func (s *OutboxService) Replay(ctx context.Context, workspaceID, eventID string) error {
	if err := s.authorizer.AuthorizeWorkspace(ctx, workspaceID, contract.PermissionReminderReplayRepair); err != nil {
		return err
	}
	return s.repository.ReplayOutbox(ctx, workspaceID, eventID, s.now().UTC())
}

func (s *OutboxService) ReadGovernanceDiagnostics(ctx context.Context, workspaceID string) (contract.OutboxDiagnostics, error) {
	diagnostics, err := s.repository.ReadOutboxDiagnostics(ctx, workspaceID, s.now().UTC())
	if err != nil {
		return contract.OutboxDiagnostics{}, err
	}
	if s.dispatcherRunning != nil {
		diagnostics.DispatcherRunning = s.dispatcherRunning()
	}
	return diagnostics, diagnostics.Validate()
}

func (s *OutboxService) SetDispatcherRunningProbe(probe func() bool) {
	s.dispatcherRunning = probe
}

func outboxRetryDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	if attempt > MaxOutboxAttempts {
		attempt = MaxOutboxAttempts
	}
	return time.Minute * time.Duration(1<<(attempt-1))
}

func nonNegativeJitter(value time.Duration) time.Duration {
	if value < 0 {
		return 0
	}
	return value
}
