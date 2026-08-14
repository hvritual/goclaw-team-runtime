package workspace

import (
	"context"
	"strings"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

type publishingIssueCollaborationService struct {
	contract.IssueCollaborationService
	activities contract.IssueActivityRecorder
	events     contract.WorkspaceEventPublisher
}

func (s publishingIssueCollaborationService) CreateIssueComment(ctx context.Context, request contract.CreateIssueCommentRequest) (contract.IssueComment, error) {
	response, err := s.IssueCollaborationService.CreateIssueComment(ctx, request)
	if err == nil {
		s.publish(request.WorkspaceID, "comment:created", map[string]any{"comment": realtimeComment(response)}, ctx)
	}
	return response, err
}

func (s publishingIssueCollaborationService) UpdateIssueComment(ctx context.Context, request contract.UpdateIssueCommentRequest) (contract.IssueComment, error) {
	response, err := s.IssueCollaborationService.UpdateIssueComment(ctx, request)
	if err == nil {
		s.publish(request.WorkspaceID, "comment:updated", map[string]any{"comment": realtimeComment(response)}, ctx)
	}
	return response, err
}

func (s publishingIssueCollaborationService) DeleteIssueComment(ctx context.Context, request contract.DeleteIssueCommentRequest) error {
	before, beforeErr := s.IssueCollaborationService.GetIssueComment(ctx, request.WorkspaceID, request.CommentID)
	if beforeErr != nil {
		return beforeErr
	}
	err := s.IssueCollaborationService.DeleteIssueComment(ctx, request)
	if err == nil {
		s.publish(request.WorkspaceID, "comment:deleted", map[string]any{"comment_id": before.ID, "issue_id": before.IssueID}, ctx)
	}
	return err
}

func (s publishingIssueCollaborationService) ResolveIssueComment(ctx context.Context, request contract.ResolveIssueCommentRequest) (contract.IssueComment, error) {
	response, err := s.IssueCollaborationService.ResolveIssueComment(ctx, request)
	if err == nil {
		event := "comment:unresolved"
		if request.Resolved {
			event = "comment:resolved"
		}
		s.publish(request.WorkspaceID, event, map[string]any{"comment": realtimeComment(response)}, ctx)
	}
	return response, err
}

func (s publishingIssueCollaborationService) AddCommentReaction(ctx context.Context, request contract.ChangeCommentReactionRequest) (contract.CommentReaction, error) {
	comment, beforeErr := s.IssueCollaborationService.GetIssueComment(ctx, request.WorkspaceID, request.CommentID)
	if beforeErr != nil {
		return contract.CommentReaction{}, beforeErr
	}
	response, err := s.IssueCollaborationService.AddCommentReaction(ctx, request)
	if err == nil {
		s.publish(request.WorkspaceID, "reaction:added", map[string]any{"reaction": realtimeCommentReaction(response), "issue_id": comment.IssueID}, ctx)
	}
	return response, err
}

func (s publishingIssueCollaborationService) RemoveCommentReaction(ctx context.Context, request contract.ChangeCommentReactionRequest) error {
	comment, beforeErr := s.IssueCollaborationService.GetIssueComment(ctx, request.WorkspaceID, request.CommentID)
	if beforeErr != nil {
		return beforeErr
	}
	err := s.IssueCollaborationService.RemoveCommentReaction(ctx, request)
	if err == nil {
		actorID, actorType := realtimeActor(ctx)
		s.events.Publish(request.WorkspaceID, "reaction:removed", map[string]any{"comment_id": comment.ID, "issue_id": comment.IssueID, "emoji": request.Emoji, "actor_type": actorType, "actor_id": actorID}, actorID, actorType)
	}
	return err
}

func (s publishingIssueCollaborationService) AddIssueReaction(ctx context.Context, request contract.ChangeIssueReactionRequest) (contract.IssueReaction, error) {
	response, err := s.IssueCollaborationService.AddIssueReaction(ctx, request)
	if err == nil {
		s.publish(request.WorkspaceID, "issue_reaction:added", map[string]any{"reaction": realtimeIssueReaction(response), "issue_id": response.IssueID}, ctx)
	}
	return response, err
}

func (s publishingIssueCollaborationService) RemoveIssueReaction(ctx context.Context, request contract.ChangeIssueReactionRequest) error {
	issueID, resolveErr := s.IssueCollaborationService.ResolveIssueID(ctx, request.WorkspaceID, request.IssueID)
	if resolveErr != nil {
		return resolveErr
	}
	err := s.IssueCollaborationService.RemoveIssueReaction(ctx, request)
	if err == nil {
		actorID, actorType := realtimeActor(ctx)
		s.events.Publish(request.WorkspaceID, "issue_reaction:removed", map[string]any{"issue_id": issueID, "emoji": strings.TrimSpace(request.Emoji), "actor_type": actorType, "actor_id": actorID}, actorID, actorType)
	}
	return err
}

func (s publishingIssueCollaborationService) SubscribeToIssue(ctx context.Context, request contract.ChangeIssueSubscriberRequest) error {
	issueID, resolveErr := s.IssueCollaborationService.ResolveIssueID(ctx, request.WorkspaceID, request.IssueID)
	if resolveErr != nil {
		return resolveErr
	}
	err := s.IssueCollaborationService.SubscribeToIssue(ctx, request)
	if err == nil {
		actorID, actorType := realtimeActor(ctx)
		userType, userID := strings.TrimSpace(request.UserType), strings.TrimSpace(request.UserID)
		if userType == "" {
			userType = actorType
		}
		if userID == "" {
			userID = actorID
		}
		s.events.Publish(request.WorkspaceID, "subscriber:added", map[string]any{"issue_id": issueID, "user_type": userType, "user_id": userID, "reason": "manual"}, actorID, actorType)
	}
	return err
}

func (s publishingIssueCollaborationService) UnsubscribeFromIssue(ctx context.Context, request contract.ChangeIssueSubscriberRequest) error {
	issueID, resolveErr := s.IssueCollaborationService.ResolveIssueID(ctx, request.WorkspaceID, request.IssueID)
	if resolveErr != nil {
		return resolveErr
	}
	err := s.IssueCollaborationService.UnsubscribeFromIssue(ctx, request)
	if err == nil {
		actorID, actorType := realtimeActor(ctx)
		userType, userID := strings.TrimSpace(request.UserType), strings.TrimSpace(request.UserID)
		if userType == "" {
			userType = actorType
		}
		if userID == "" {
			userID = actorID
		}
		s.events.Publish(request.WorkspaceID, "subscriber:removed", map[string]any{"issue_id": issueID, "user_type": userType, "user_id": userID}, actorID, actorType)
	}
	return err
}

func (s publishingIssueCollaborationService) RecordIssueActivity(ctx context.Context, workspaceID, issueID, action string, details map[string]any) (contract.IssueActivity, error) {
	response, err := s.activities.RecordIssueActivity(ctx, workspaceID, issueID, action, details)
	if err == nil {
		s.publish(workspaceID, "activity:created", map[string]any{"issue_id": response.IssueID, "entry": realtimeActivity(response)}, ctx)
	}
	return response, err
}

func (s publishingIssueCollaborationService) publish(workspaceID, event string, payload map[string]any, ctx context.Context) {
	actorID, actorType := realtimeActor(ctx)
	s.events.Publish(workspaceID, event, payload, actorID, actorType)
}

func realtimeComment(value contract.IssueComment) map[string]any {
	reactions := make([]map[string]any, len(value.Reactions))
	for index := range value.Reactions {
		reactions[index] = realtimeCommentReaction(value.Reactions[index])
	}
	attachments := value.Attachments
	if attachments == nil {
		attachments = []map[string]any{}
	}
	return map[string]any{"id": value.ID, "issue_id": value.IssueID, "author_type": value.AuthorType, "author_id": value.AuthorID, "content": value.Content, "type": value.Type, "parent_id": value.ParentID, "reactions": reactions, "attachments": attachments, "created_at": value.CreatedAt, "updated_at": value.UpdatedAt, "resolved_at": value.ResolvedAt, "resolved_by_type": value.ResolvedByType, "resolved_by_id": value.ResolvedByID}
}

func realtimeCommentReaction(value contract.CommentReaction) map[string]any {
	return map[string]any{"id": value.ID, "comment_id": value.CommentID, "actor_type": value.ActorType, "actor_id": value.ActorID, "emoji": value.Emoji, "created_at": value.CreatedAt}
}

func realtimeIssueReaction(value contract.IssueReaction) map[string]any {
	return map[string]any{"id": value.ID, "issue_id": value.IssueID, "actor_type": value.ActorType, "actor_id": value.ActorID, "emoji": value.Emoji, "created_at": value.CreatedAt}
}

func realtimeActivity(value contract.IssueActivity) map[string]any {
	return map[string]any{"type": "activity", "id": value.ID, "actor_type": value.ActorType, "actor_id": value.ActorID, "action": value.Action, "details": value.Details, "created_at": value.CreatedAt}
}

var _ contract.IssueCollaborationService = publishingIssueCollaborationService{}
var _ contract.IssueActivityRecorder = publishingIssueCollaborationService{}
