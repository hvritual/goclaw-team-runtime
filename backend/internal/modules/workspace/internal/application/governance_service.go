package application

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

type GovernanceRequest struct {
	Identity       contract.MutationIdentity
	Command        contract.MutationCommand
	ResponseStatus int
	ResponseBody   json.RawMessage
	AuditID        string
	OccurredAt     time.Time
	AuditMetadata  map[string]string
	Outbox         []OutboxDraft
}

type OutboxDraft struct {
	ID        string
	EventType string
	Payload   json.RawMessage
}

type PreparedGovernanceMutation struct {
	Identity contract.MutationIdentity
	Command  contract.MutationCommand
	Result   contract.MutationResult
	Audit    contract.AuditRecord
	Outbox   []contract.OutboxEvent
}

type GovernanceService struct{}

func NewGovernanceService() GovernanceService { return GovernanceService{} }

func (GovernanceService) Prepare(request GovernanceRequest) (PreparedGovernanceMutation, error) {
	if err := request.Identity.Validate(); err != nil {
		return PreparedGovernanceMutation{}, fmt.Errorf("validate mutation identity: %w", err)
	}
	if err := request.Command.Validate(); err != nil {
		return PreparedGovernanceMutation{}, fmt.Errorf("validate mutation command: %w", err)
	}

	revision := request.Command.ExpectedRevision + 1
	result := contract.MutationResult{
		ResourceRevision: revision,
		ResponseStatus:   request.ResponseStatus,
		ResponseBody:     request.ResponseBody,
	}
	if err := result.Validate(); err != nil {
		return PreparedGovernanceMutation{}, fmt.Errorf("validate mutation result: %w", err)
	}

	metadata, err := json.Marshal(allowlistedAuditMetadata(request.AuditMetadata))
	if err != nil {
		return PreparedGovernanceMutation{}, fmt.Errorf("encode audit metadata: %w", err)
	}
	audit := contract.AuditRecord{
		ID:               request.AuditID,
		Identity:         request.Identity,
		Action:           request.Command.Action,
		ResourceKind:     request.Command.ResourceKind,
		ResourceID:       request.Command.ResourceID,
		ResourceRevision: revision,
		OccurredAt:       request.OccurredAt,
		Metadata:         metadata,
	}
	if err := audit.Validate(); err != nil {
		return PreparedGovernanceMutation{}, fmt.Errorf("validate audit record: %w", err)
	}

	outbox := make([]contract.OutboxEvent, 0, len(request.Outbox))
	for _, draft := range request.Outbox {
		event := contract.OutboxEvent{
			State:             contract.OutboxReady,
			AvailableAt:       request.OccurredAt,
			WorkspaceID:       request.Identity.WorkspaceID,
			ID:                draft.ID,
			EventType:         draft.EventType,
			AggregateKind:     request.Command.ResourceKind,
			AggregateID:       request.Command.ResourceID,
			AggregateRevision: revision,
			Payload:           draft.Payload,
			ActorType:         request.Identity.ActorType,
			ActorID:           request.Identity.ActorID,
			CreatedAt:         request.OccurredAt,
		}
		if err := event.Validate(); err != nil {
			return PreparedGovernanceMutation{}, fmt.Errorf("validate outbox event: %w", err)
		}
		outbox = append(outbox, event)
	}

	return PreparedGovernanceMutation{
		Identity: request.Identity,
		Command:  request.Command,
		Result:   result,
		Audit:    audit,
		Outbox:   outbox,
	}, nil
}

func allowlistedAuditMetadata(metadata map[string]string) map[string]string {
	allowed := map[string]struct{}{
		"algorithm_version": {},
		"content_hash":      {},
		"count":             {},
		"from":              {},
		"reason_code":       {},
		"result":            {},
		"status":            {},
		"to":                {},
	}
	filtered := make(map[string]string)
	for key, value := range metadata {
		key = strings.TrimSpace(key)
		if _, ok := allowed[key]; ok {
			filtered[key] = value
		}
	}
	return filtered
}
