package workspace

import (
	"context"
	"encoding/json"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

type publishingIssueCatalogService struct {
	contract.IssueCatalogService
	events contract.WorkspaceEventPublisher
}

func (s publishingIssueCatalogService) CreateIssueLabel(ctx context.Context, request contract.CreateIssueLabelRequest) (contract.IssueLabel, error) {
	value, err := s.IssueCatalogService.CreateIssueLabel(ctx, request)
	if err == nil {
		s.publish(ctx, request.WorkspaceID, "label:created", map[string]any{"label": realtimeLabel(value)})
	}
	return value, err
}

func (s publishingIssueCatalogService) UpdateIssueLabel(ctx context.Context, request contract.UpdateIssueLabelRequest) (contract.IssueLabel, error) {
	value, err := s.IssueCatalogService.UpdateIssueLabel(ctx, request)
	if err == nil {
		s.publish(ctx, request.WorkspaceID, "label:updated", map[string]any{"label": realtimeLabel(value)})
	}
	return value, err
}

func (s publishingIssueCatalogService) DeleteIssueLabel(ctx context.Context, workspaceID, labelID string) error {
	err := s.IssueCatalogService.DeleteIssueLabel(ctx, workspaceID, labelID)
	if err == nil {
		s.publish(ctx, workspaceID, "label:deleted", map[string]any{"label_id": labelID})
	}
	return err
}

func (s publishingIssueCatalogService) AttachLabelToIssue(ctx context.Context, workspaceID, issueID, labelID string) (string, []contract.IssueLabel, error) {
	resolved, labels, err := s.IssueCatalogService.AttachLabelToIssue(ctx, workspaceID, issueID, labelID)
	if err == nil {
		s.publish(ctx, workspaceID, "issue_labels:changed", map[string]any{"issue_id": resolved, "labels": realtimeLabels(labels)})
	}
	return resolved, labels, err
}

func (s publishingIssueCatalogService) DetachLabelFromIssue(ctx context.Context, workspaceID, issueID, labelID string) (string, []contract.IssueLabel, error) {
	resolved, labels, err := s.IssueCatalogService.DetachLabelFromIssue(ctx, workspaceID, issueID, labelID)
	if err == nil {
		s.publish(ctx, workspaceID, "issue_labels:changed", map[string]any{"issue_id": resolved, "labels": realtimeLabels(labels)})
	}
	return resolved, labels, err
}

func (s publishingIssueCatalogService) CreateIssueProperty(ctx context.Context, request contract.CreateIssuePropertyRequest) (contract.IssuePropertyDefinition, error) {
	value, err := s.IssueCatalogService.CreateIssueProperty(ctx, request)
	if err == nil {
		s.publish(ctx, request.WorkspaceID, "property:created", map[string]any{"property": realtimeProperty(value)})
	}
	return value, err
}

func (s publishingIssueCatalogService) UpdateIssueProperty(ctx context.Context, request contract.UpdateIssuePropertyRequest) (contract.IssuePropertyDefinition, error) {
	value, err := s.IssueCatalogService.UpdateIssueProperty(ctx, request)
	if err == nil {
		s.publish(ctx, request.WorkspaceID, "property:updated", map[string]any{"property": realtimeProperty(value)})
	}
	return value, err
}

func (s publishingIssueCatalogService) SetIssueProperty(ctx context.Context, workspaceID, issueID, propertyID string, raw json.RawMessage) (string, map[string]any, error) {
	resolved, values, err := s.IssueCatalogService.SetIssueProperty(ctx, workspaceID, issueID, propertyID, raw)
	if err == nil {
		s.publish(ctx, workspaceID, "issue_properties:changed", map[string]any{"issue_id": resolved, "properties": values})
	}
	return resolved, values, err
}

func (s publishingIssueCatalogService) UnsetIssueProperty(ctx context.Context, workspaceID, issueID, propertyID string) (string, map[string]any, error) {
	resolved, values, err := s.IssueCatalogService.UnsetIssueProperty(ctx, workspaceID, issueID, propertyID)
	if err == nil {
		s.publish(ctx, workspaceID, "issue_properties:changed", map[string]any{"issue_id": resolved, "properties": values})
	}
	return resolved, values, err
}

func (s publishingIssueCatalogService) CreateAcceptanceConclusion(ctx context.Context, workspaceID, issueID string, input contract.AcceptanceConclusionInput) (contract.AcceptanceConclusionMutation, error) {
	value, err := s.IssueCatalogService.CreateAcceptanceConclusion(ctx, workspaceID, issueID, input)
	if err == nil && value.Issue != nil {
		s.publish(ctx, workspaceID, "issue:updated", map[string]any{"issue": realtimeIssue(value.Issue)})
	}
	return value, err
}

func (s publishingIssueCatalogService) CompleteIssueWithAcceptance(ctx context.Context, workspaceID, issueID string, input contract.AcceptanceConclusionInput) (contract.AcceptanceConclusionMutation, error) {
	value, err := s.IssueCatalogService.CompleteIssueWithAcceptance(ctx, workspaceID, issueID, input)
	if err == nil && value.Issue != nil {
		s.publish(ctx, workspaceID, "issue:updated", map[string]any{"issue": realtimeIssue(value.Issue), "status_changed": true})
	}
	return value, err
}

func (s publishingIssueCatalogService) publish(ctx context.Context, workspaceID, event string, payload map[string]any) {
	actorID, actorType := realtimeActor(ctx)
	s.events.Publish(workspaceID, event, payload, actorID, actorType)
}

func realtimeLabels(values []contract.IssueLabel) []map[string]any {
	result := make([]map[string]any, len(values))
	for index := range values {
		result[index] = realtimeLabel(values[index])
	}
	return result
}

func realtimeLabel(value contract.IssueLabel) map[string]any {
	return map[string]any{"id": value.ID, "workspace_id": value.WorkspaceID, "resource_type": value.ResourceType, "name": value.Name, "description": value.Description, "color": value.Color, "usage_count": value.UsageCount, "created_at": value.CreatedAt, "updated_at": value.UpdatedAt}
}

func realtimeProperty(value contract.IssuePropertyDefinition) map[string]any {
	options := value.Config.Options
	if options == nil {
		options = []contract.IssuePropertyOption{}
	}
	return map[string]any{"id": value.ID, "workspace_id": value.WorkspaceID, "name": value.Name, "type": value.Type, "description": value.Description, "icon": value.Icon, "config": map[string]any{"options": options}, "position": value.Position, "archived": value.Archived, "archived_at": value.ArchivedAt, "usage_count": value.UsageCount, "created_at": value.CreatedAt, "updated_at": value.UpdatedAt}
}

var _ contract.IssueCatalogService = publishingIssueCatalogService{}
