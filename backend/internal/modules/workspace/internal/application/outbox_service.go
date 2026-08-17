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
	MarkOutboxDelivered(context.Context, contract.OutboxClaimIdentity, time.Time) error
	MarkOutboxFailed(context.Context, contract.OutboxClaimIdentity, time.Time, time.Time, string, bool) error
	ReplayOutbox(context.Context, contract.OutboxRowIdentity, time.Time) error
	ReadOutboxDiagnostics(context.Context, string, time.Time) (contract.OutboxDiagnostics, error)
}

type OutboxServiceConfig struct {
	Repository    OutboxRepository
	Sink          contract.OutboxSink
	Authorizer    contract.WorkspaceAccessAuthorizer
	EventPolicies GovernanceEventPolicyProvider
	Now           func() time.Time
	NewClaimToken func() string
	Jitter        func(int) time.Duration
	BatchSize     int
}

type OutboxService struct {
	repository        OutboxRepository
	sink              contract.OutboxSink
	authorizer        contract.WorkspaceAccessAuthorizer
	eventPolicies     GovernanceEventPolicyProvider
	now               func() time.Time
	newClaimToken     func() string
	jitter            func(int) time.Duration
	batchSize         int
	dispatcherRunning func() bool
}

func NewOutboxService(config OutboxServiceConfig) (*OutboxService, error) {
	if config.Repository == nil || config.Sink == nil || config.Authorizer == nil || config.EventPolicies == nil {
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
		repository: config.Repository, sink: config.Sink, authorizer: config.Authorizer, eventPolicies: config.EventPolicies,
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
		if err := event.Validate(); err != nil {
			return 0, err
		}
		if err := validateGovernanceEventPolicy(ctx, s.eventPolicies, event); err != nil {
			return 0, err
		}
		claimIdentity, err := event.ClaimIdentity()
		if err != nil {
			return 0, err
		}
		if err := s.sink.Publish(ctx, event); err == nil {
			transitionedAt := s.now().UTC()
			if err := s.repository.MarkOutboxDelivered(ctx, claimIdentity, transitionedAt); err != nil {
				return 0, fmt.Errorf("acknowledge outbox delivery: %w", err)
			}
			continue
		}
		transitionedAt := s.now().UTC()
		dead := event.AttemptCount >= MaxOutboxAttempts
		retryAt := transitionedAt
		if !dead {
			retryAt = transitionedAt.Add(outboxRetryDelay(event.AttemptCount) + nonNegativeJitter(s.jitter(event.AttemptCount)))
		}
		if err := s.repository.MarkOutboxFailed(ctx, claimIdentity, transitionedAt, retryAt, "publish_failed", dead); err != nil {
			return 0, fmt.Errorf("record outbox delivery failure: %w", err)
		}
	}
	return len(events), nil
}

func (s *OutboxService) Replay(ctx context.Context, identity contract.OutboxRowIdentity) error {
	if err := identity.Validate(); err != nil || identity.State != contract.OutboxDeadLetter {
		return fmt.Errorf("%w: invalid dead-letter replay identity", contract.ErrInvalidGovernanceMutation)
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, identity.WorkspaceID, contract.PermissionReminderReplayRepair); err != nil {
		return err
	}
	return s.repository.ReplayOutbox(ctx, identity, s.now().UTC())
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
