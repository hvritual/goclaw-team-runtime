package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

type GovernanceRequest struct {
	Identity       contract.MutationIdentity
	Command        contract.MutationCommand
	RequestFields  map[string]any
	ResponseStatus int
	ResponseBody   json.RawMessage
	ResponseFields map[string]any
	AuditID        string
	OccurredAt     time.Time
	AuditMetadata  map[string]string
	AuditFields    map[string]any
	Outbox         []OutboxDraft
}

type OutboxDraft struct {
	ID        string
	EventType string
	Payload   json.RawMessage
	Fields    map[string]any
}

type PreparedGovernanceMutation struct {
	identity contract.MutationIdentity
	command  contract.MutationCommand
	result   contract.MutationResult
	audit    contract.AuditRecord
	outbox   []contract.OutboxEvent
	policy   GovernanceActionPolicy
	prepared bool
}

func (p PreparedGovernanceMutation) Identity() contract.MutationIdentity { return p.identity }
func (p PreparedGovernanceMutation) Command() contract.MutationCommand   { return p.command }
func (p PreparedGovernanceMutation) Result() contract.MutationResult {
	result := p.result
	result.ResponseBody = cloneGovernanceJSON(p.result.ResponseBody)
	return result
}
func (p PreparedGovernanceMutation) Audit() contract.AuditRecord {
	audit := p.audit
	audit.Metadata = cloneGovernanceJSON(p.audit.Metadata)
	return audit
}

func (p PreparedGovernanceMutation) Outbox() []contract.OutboxEvent {
	outbox := append([]contract.OutboxEvent(nil), p.outbox...)
	for index := range outbox {
		outbox[index].Payload = cloneGovernanceJSON(p.outbox[index].Payload)
	}
	return outbox
}

func cloneGovernanceJSON(raw json.RawMessage) json.RawMessage {
	return append(json.RawMessage(nil), raw...)
}

func (p PreparedGovernanceMutation) Validate() error {
	if !p.prepared {
		return fmt.Errorf("%w: mutation was not policy prepared", contract.ErrInvalidGovernanceMutation)
	}
	if err := p.identity.Validate(); err != nil {
		return err
	}
	if err := p.policy.validate(); err != nil || p.policy.Action != p.command.Action || p.policy.ResourceKind != p.command.ResourceKind {
		return contract.ErrGovernanceUnavailable
	}
	if err := p.command.Validate(); err != nil {
		return err
	}
	if err := p.result.Validate(); err != nil {
		return err
	}
	if err := validateGovernanceEnvelopeFields("governance-replay-v1", p.policy.ReplaySchema, p.result.ResponseBody); err != nil {
		return err
	}
	if p.result.ResourceRevision != p.command.ExpectedRevision+1 || p.result.Replayed {
		return fmt.Errorf("%w: prepared revision is inconsistent", contract.ErrInvalidGovernanceMutation)
	}
	if err := p.audit.Validate(); err != nil {
		return err
	}
	if err := validateGovernanceEnvelopeFields("governance-audit-v1", p.policy.AuditSchema, p.audit.Metadata); err != nil {
		return err
	}
	if p.audit.Identity != p.identity || p.audit.Action != p.command.Action || p.audit.ResourceKind != p.command.ResourceKind ||
		p.audit.ResourceID != p.command.ResourceID || p.audit.ResourceRevision != p.result.ResourceRevision {
		return fmt.Errorf("%w: audit envelope is inconsistent", contract.ErrInvalidGovernanceMutation)
	}
	for _, event := range p.outbox {
		if err := event.Validate(); err != nil {
			return err
		}
		schema, ok := p.policy.EventSchemas[event.EventType]
		if !ok {
			return contract.ErrGovernanceUnavailable
		}
		if err := validateGovernanceEnvelopeFields("governance-outbox-v1", schema, event.Payload); err != nil {
			return err
		}
		if event.WorkspaceID != p.identity.WorkspaceID || event.AggregateKind != p.command.ResourceKind || event.AggregateID != p.command.ResourceID ||
			event.AggregateRevision != p.result.ResourceRevision || event.ActorType != p.identity.ActorType || event.ActorID != p.identity.ActorID || event.State != contract.OutboxReady {
			return fmt.Errorf("%w: outbox envelope is inconsistent", contract.ErrInvalidGovernanceMutation)
		}
	}
	return nil
}

func (p PreparedGovernanceMutation) ValidateReplayResponse(raw json.RawMessage) error {
	if !p.prepared {
		return fmt.Errorf("%w: mutation was not policy prepared", contract.ErrInvalidGovernanceMutation)
	}
	return validateGovernanceEnvelopeFields("governance-replay-v1", p.policy.ReplaySchema, raw)
}

type GovernanceService struct {
	policyProvider GovernancePolicyProvider
}

func NewGovernanceService(providers ...GovernancePolicyProvider) GovernanceService {
	service := GovernanceService{}
	if len(providers) == 1 {
		service.policyProvider = providers[0]
	}
	return service
}

// PrepareContext is the policy-resolved governance entry point. Until an
// explicit server-side policy provider is installed it fails closed.
func (s GovernanceService) PrepareContext(ctx context.Context, request GovernanceRequest) (PreparedGovernanceMutation, error) {
	identity, policy, err := resolveGovernancePolicy(ctx, s.policyProvider, request)
	if err != nil {
		return PreparedGovernanceMutation{}, err
	}
	if strings.TrimSpace(request.Command.RequestHash) != "" {
		return PreparedGovernanceMutation{}, fmt.Errorf("%w: caller request hash is forbidden", contract.ErrInvalidGovernanceMutation)
	}
	requestHash, err := governanceRequestHash(request.Command.Action, policy.RequestSchema, request.RequestFields)
	if err != nil {
		return PreparedGovernanceMutation{}, fmt.Errorf("canonicalize governance request: %w", err)
	}
	if strings.TrimSpace(request.Command.IdempotencyKey) != "" {
		request.Command.RequestHash = requestHash
	}
	responseBody, err := encodeGovernanceEnvelope("governance-replay-v1", policy.ReplaySchema, request.ResponseFields)
	if err != nil {
		return PreparedGovernanceMutation{}, fmt.Errorf("validate governance replay: %w", err)
	}
	auditMetadata, err := encodeGovernanceEnvelope("governance-audit-v1", policy.AuditSchema, request.AuditFields)
	if err != nil {
		return PreparedGovernanceMutation{}, fmt.Errorf("validate governance audit: %w", err)
	}
	outbox := make([]contract.OutboxEvent, 0, len(request.Outbox))
	revision := request.Command.ExpectedRevision + 1
	for _, draft := range request.Outbox {
		schema, ok := policy.EventSchemas[draft.EventType]
		if !ok {
			return PreparedGovernanceMutation{}, contract.ErrGovernanceUnavailable
		}
		payload, err := encodeGovernanceEnvelope("governance-outbox-v1", schema, draft.Fields)
		if err != nil {
			return PreparedGovernanceMutation{}, fmt.Errorf("validate governance outbox: %w", err)
		}
		outbox = append(outbox, contract.OutboxEvent{
			State: contract.OutboxReady, AvailableAt: request.OccurredAt, WorkspaceID: identity.WorkspaceID,
			ID: draft.ID, EventType: draft.EventType, AggregateKind: request.Command.ResourceKind,
			AggregateID: request.Command.ResourceID, AggregateRevision: revision, Payload: payload,
			ActorType: identity.ActorType, ActorID: identity.ActorID, CreatedAt: request.OccurredAt,
		})
	}
	result := contract.MutationResult{ResourceRevision: revision, ResponseStatus: request.ResponseStatus, ResponseBody: responseBody}
	if err := result.Validate(); err != nil {
		return PreparedGovernanceMutation{}, fmt.Errorf("validate mutation result: %w", err)
	}
	audit := contract.AuditRecord{
		ID: request.AuditID, Identity: identity, Action: request.Command.Action,
		ResourceKind: request.Command.ResourceKind, ResourceID: request.Command.ResourceID,
		ResourceRevision: revision, OccurredAt: request.OccurredAt, Metadata: auditMetadata,
	}
	if err := audit.Validate(); err != nil {
		return PreparedGovernanceMutation{}, fmt.Errorf("validate audit record: %w", err)
	}
	for _, event := range outbox {
		if err := event.Validate(); err != nil {
			return PreparedGovernanceMutation{}, fmt.Errorf("validate outbox event: %w", err)
		}
	}
	prepared := PreparedGovernanceMutation{
		identity: identity, command: request.Command, result: result, audit: audit,
		outbox: outbox, policy: policy, prepared: true,
	}
	if err := prepared.Validate(); err != nil {
		return PreparedGovernanceMutation{}, err
	}
	return prepared, nil
}

func (GovernanceService) Prepare(request GovernanceRequest) (PreparedGovernanceMutation, error) {
	return PreparedGovernanceMutation{}, contract.ErrGovernanceUnavailable
}
