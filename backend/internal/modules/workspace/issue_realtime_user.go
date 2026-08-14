package workspace

import (
	"context"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

type publishingIssueService struct {
	contract.IssueMutationService
	events contract.WorkspaceEventPublisher
}

func (s publishingIssueService) CreateIssue(ctx context.Context, request contract.CreateIssueRequest) (contract.CreateIssueResponse, error) {
	response, err := s.IssueMutationService.CreateIssue(ctx, request)
	if err == nil && response.Issue != nil {
		actorID, actorType := realtimeActor(ctx)
		s.events.Publish(request.WorkspaceId, "issue:created", map[string]any{"issue": realtimeIssue(response.Issue)}, actorID, actorType)
	}
	return response, err
}

func (s publishingIssueService) UpdateIssue(ctx context.Context, request contract.UpdateIssueRequest) (contract.UpdateIssueResponse, error) {
	before, beforeErr := s.IssueMutationService.GetIssue(ctx, contract.GetIssueRequest{WorkspaceId: request.WorkspaceId, IssueId: request.IssueId})
	if beforeErr != nil {
		return contract.UpdateIssueResponse{}, beforeErr
	}
	response, err := s.IssueMutationService.UpdateIssue(ctx, request)
	if err == nil && response.Issue != nil {
		actorID, actorType := realtimeActor(ctx)
		s.events.Publish(request.WorkspaceId, "issue:updated", map[string]any{
			"issue":            realtimeIssue(response.Issue),
			"assignee_changed": before.Issue != nil && (pointerValue(before.Issue.AssigneeType) != pointerValue(response.Issue.AssigneeType) || pointerValue(before.Issue.AssigneeId) != pointerValue(response.Issue.AssigneeId)),
			"status_changed":   before.Issue != nil && before.Issue.Status != response.Issue.Status,
			"project_changed":  before.Issue != nil && pointerValue(before.Issue.ProjectId) != pointerValue(response.Issue.ProjectId),
		}, actorID, actorType)
	}
	return response, err
}

func (s publishingIssueService) UpdateIssueStatus(ctx context.Context, request contract.UpdateIssueStatusRequest) (contract.UpdateIssueStatusResponse, error) {
	before, beforeErr := s.IssueMutationService.GetIssue(ctx, contract.GetIssueRequest{WorkspaceId: request.WorkspaceId, IssueId: request.IssueId})
	if beforeErr != nil {
		return contract.UpdateIssueStatusResponse{}, beforeErr
	}
	response, err := s.IssueMutationService.UpdateIssueStatus(ctx, request)
	if err == nil && response.Issue != nil {
		actorID, actorType := realtimeActor(ctx)
		s.events.Publish(request.WorkspaceId, "issue:updated", map[string]any{"issue": realtimeIssue(response.Issue), "status_changed": before.Issue != nil && before.Issue.Status != response.Issue.Status}, actorID, actorType)
	}
	return response, err
}

func (s publishingIssueService) MoveIssue(ctx context.Context, request contract.MoveIssueRequest) (contract.MoveIssueResponse, error) {
	before, beforeErr := s.IssueMutationService.GetIssue(ctx, contract.GetIssueRequest{WorkspaceId: request.WorkspaceID, IssueId: request.IssueID})
	if beforeErr != nil {
		return contract.MoveIssueResponse{}, beforeErr
	}
	response, err := s.IssueMutationService.MoveIssue(ctx, request)
	if err == nil && response.Issue != nil {
		actorID, actorType := realtimeActor(ctx)
		s.events.Publish(request.WorkspaceID, "issue:updated", map[string]any{
			"issue":            realtimeIssue(response.Issue),
			"assignee_changed": before.Issue != nil && (pointerValue(before.Issue.AssigneeType) != pointerValue(response.Issue.AssigneeType) || pointerValue(before.Issue.AssigneeId) != pointerValue(response.Issue.AssigneeId)),
			"status_changed":   before.Issue != nil && before.Issue.Status != response.Issue.Status,
			"project_changed":  before.Issue != nil && pointerValue(before.Issue.ProjectId) != pointerValue(response.Issue.ProjectId),
		}, actorID, actorType)
	}
	return response, err
}

func pointerValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

type publishingIssueMetadataService struct {
	contract.IssueMetadataService
	events contract.WorkspaceEventPublisher
}

func (s publishingIssueMetadataService) PutIssueMetadata(ctx context.Context, request contract.PutIssueMetadataRequest) (contract.IssueMetadataSnapshot, error) {
	response, err := s.IssueMetadataService.PutIssueMetadata(ctx, request)
	if err == nil {
		actorID, actorType := realtimeActor(ctx)
		s.events.Publish(request.WorkspaceId, "issue_metadata:changed", map[string]any{"issue_id": response.IssueId, "metadata": response.Metadata}, actorID, actorType)
	}
	return response, err
}

func (s publishingIssueMetadataService) DeleteIssueMetadata(ctx context.Context, request contract.DeleteIssueMetadataRequest) (contract.IssueMetadataSnapshot, error) {
	response, err := s.IssueMetadataService.DeleteIssueMetadata(ctx, request)
	if err == nil {
		actorID, actorType := realtimeActor(ctx)
		s.events.Publish(request.WorkspaceId, "issue_metadata:changed", map[string]any{"issue_id": response.IssueId, "metadata": response.Metadata}, actorID, actorType)
	}
	return response, err
}

func realtimeActor(ctx context.Context) (string, string) {
	actor, ok := contract.WorkspaceActorFromContext(ctx)
	if !ok {
		return "", ""
	}
	return actor.ID, actor.Type
}

func realtimeIssue(issue *contract.Issue) map[string]any {
	return map[string]any{
		"id": issue.Id, "workspace_id": issue.WorkspaceId, "number": issue.Number,
		"identifier": issue.Identifier, "title": issue.Title, "description": issue.Description,
		"status": issue.Status, "priority": issue.Priority, "assignee_type": issue.AssigneeType,
		"assignee_id": issue.AssigneeId, "creator_type": issue.CreatorType, "creator_id": issue.CreatorId,
		"parent_issue_id": issue.ParentIssueId, "project_id": issue.ProjectId, "position": issue.Position,
		"stage": issue.Stage, "start_date": issue.StartDate, "due_date": issue.DueDate,
		"metadata": issue.Metadata, "properties": issue.Properties, "created_at": issue.CreatedAt, "updated_at": issue.UpdatedAt,
	}
}
