package workspace

import (
	"context"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

type publishingIssueService struct {
	contract.IssueMutationService
	hierarchy  contract.IssueHierarchyService
	activities contract.IssueActivityRecorder
	events     contract.WorkspaceEventPublisher
}

func (s publishingIssueService) CreateIssue(ctx context.Context, request contract.CreateIssueRequest) (contract.CreateIssueResponse, error) {
	response, err := s.IssueMutationService.CreateIssue(ctx, request)
	if err == nil && response.Issue != nil {
		actorID, actorType := realtimeActor(ctx)
		s.events.Publish(request.WorkspaceId, "issue:created", map[string]any{"issue": realtimeIssue(response.Issue)}, actorID, actorType)
		s.recordActivity(ctx, request.WorkspaceId, response.Issue.Id, "created", map[string]any{})
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
		if before.Issue != nil {
			s.recordChanges(ctx, request.WorkspaceId, before.Issue, response.Issue)
		}
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
		if before.Issue != nil {
			s.recordChanges(ctx, request.WorkspaceId, before.Issue, response.Issue)
		}
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
		if before.Issue != nil {
			s.recordChanges(ctx, request.WorkspaceID, before.Issue, response.Issue)
		}
	}
	return response, err
}

func (s publishingIssueService) ListIssueChildren(ctx context.Context, request contract.ListIssueChildrenRequest) (contract.ListIssueChildrenResponse, error) {
	return s.hierarchy.ListIssueChildren(ctx, request)
}

func (s publishingIssueService) ListIssueChildrenByParents(ctx context.Context, request contract.ListIssueChildrenByParentsRequest) (contract.ListIssueChildrenResponse, error) {
	return s.hierarchy.ListIssueChildrenByParents(ctx, request)
}

func (s publishingIssueService) ListIssueChildProgress(ctx context.Context, request contract.ListIssueChildProgressRequest) (contract.ListIssueChildProgressResponse, error) {
	return s.hierarchy.ListIssueChildProgress(ctx, request)
}

func (s publishingIssueService) BatchUpdateIssues(ctx context.Context, request contract.BatchUpdateIssuesRequest) (contract.BatchUpdateIssuesResponse, error) {
	before := make(map[string]*contract.Issue, len(request.IssueIDs))
	for _, issueID := range request.IssueIDs {
		response, err := s.IssueMutationService.GetIssue(ctx, contract.GetIssueRequest{WorkspaceId: request.WorkspaceID, IssueId: issueID})
		if err == nil && response.Issue != nil {
			before[response.Issue.Id] = response.Issue
		}
	}
	response, err := s.hierarchy.BatchUpdateIssues(ctx, request)
	if err == nil {
		actorID, actorType := realtimeActor(ctx)
		for index := range response.Issues {
			issue := response.Issues[index]
			s.events.Publish(request.WorkspaceID, "issue:updated", map[string]any{"issue": realtimeIssue(&issue)}, actorID, actorType)
			if previous := before[issue.Id]; previous != nil {
				s.recordChanges(ctx, request.WorkspaceID, previous, &issue)
			}
		}
	}
	return response, err
}

func (s publishingIssueService) recordChanges(ctx context.Context, workspaceID string, before, after *contract.Issue) {
	if before.Title != after.Title {
		s.recordActivity(ctx, workspaceID, after.Id, "title_changed", map[string]any{"from": before.Title, "to": after.Title})
	}
	if pointerValue(before.Description) != pointerValue(after.Description) {
		s.recordActivity(ctx, workspaceID, after.Id, "description_updated", map[string]any{})
	}
	if before.Status != after.Status {
		s.recordActivity(ctx, workspaceID, after.Id, "status_changed", map[string]any{"from": before.Status, "to": after.Status})
	}
	if before.Priority != after.Priority {
		s.recordActivity(ctx, workspaceID, after.Id, "priority_changed", map[string]any{"from": before.Priority, "to": after.Priority})
	}
	if pointerValue(before.AssigneeType) != pointerValue(after.AssigneeType) || pointerValue(before.AssigneeId) != pointerValue(after.AssigneeId) {
		s.recordActivity(ctx, workspaceID, after.Id, "assignee_changed", map[string]any{"from_type": pointerValue(before.AssigneeType), "from_id": pointerValue(before.AssigneeId), "to_type": pointerValue(after.AssigneeType), "to_id": pointerValue(after.AssigneeId)})
	}
	if pointerValue(before.ParentIssueId) != pointerValue(after.ParentIssueId) {
		s.recordActivity(ctx, workspaceID, after.Id, "parent_changed", map[string]any{"from": pointerValue(before.ParentIssueId), "to": pointerValue(after.ParentIssueId)})
	}
	if pointerValue(before.ProjectId) != pointerValue(after.ProjectId) {
		s.recordActivity(ctx, workspaceID, after.Id, "project_changed", map[string]any{"from": pointerValue(before.ProjectId), "to": pointerValue(after.ProjectId)})
	}
	if intPointerValue(before.Stage) != intPointerValue(after.Stage) {
		s.recordActivity(ctx, workspaceID, after.Id, "stage_changed", map[string]any{"from": intPointerValue(before.Stage), "to": intPointerValue(after.Stage)})
	}
	if before.Position != after.Position {
		s.recordActivity(ctx, workspaceID, after.Id, "position_changed", map[string]any{"from": before.Position, "to": after.Position})
	}
	if pointerValue(before.StartDate) != pointerValue(after.StartDate) {
		s.recordActivity(ctx, workspaceID, after.Id, "start_date_changed", map[string]any{"from": pointerValue(before.StartDate), "to": pointerValue(after.StartDate)})
	}
	if pointerValue(before.DueDate) != pointerValue(after.DueDate) {
		s.recordActivity(ctx, workspaceID, after.Id, "due_date_changed", map[string]any{"from": pointerValue(before.DueDate), "to": pointerValue(after.DueDate)})
	}
}

func (s publishingIssueService) recordActivity(ctx context.Context, workspaceID, issueID, action string, details map[string]any) {
	if s.activities != nil {
		_, _ = s.activities.RecordIssueActivity(ctx, workspaceID, issueID, action, details)
	}
}

func (s publishingIssueService) BatchDeleteIssues(ctx context.Context, request contract.BatchDeleteIssuesRequest) (contract.BatchDeleteIssuesResponse, error) {
	response, err := s.hierarchy.BatchDeleteIssues(ctx, request)
	if err == nil {
		actorID, actorType := realtimeActor(ctx)
		for _, issueID := range response.IssueIDs {
			s.events.Publish(request.WorkspaceID, "issue:deleted", map[string]any{"issue_id": issueID}, actorID, actorType)
		}
	}
	return response, err
}

func pointerValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func intPointerValue(value *int32) any {
	if value == nil {
		return nil
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
