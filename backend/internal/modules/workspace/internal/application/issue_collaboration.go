package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

var (
	ErrIssueCommentNotFound      = errors.New("Issue comment not found")
	ErrIssueCollaborationInvalid = errors.New("invalid Issue collaboration request")
	ErrIssueCommentPermission    = errors.New("Issue comment permission denied")
)

type IssueCollaborationRepository interface {
	ResolveIssue(context.Context, string, string) (string, error)
	GetComment(context.Context, string, string) (contract.IssueComment, error)
	ListComments(context.Context, string, string) ([]contract.IssueComment, error)
	ListActivities(context.Context, string, string) ([]contract.IssueActivity, error)
	CreateComment(context.Context, contract.IssueComment) (contract.IssueComment, error)
	UpdateComment(context.Context, string, string, string, []string, string) (contract.IssueComment, error)
	DeleteComment(context.Context, string, string) error
	ResolveComment(context.Context, string, string, string, string, string, bool) (contract.IssueComment, error)
	ProposeCommentKnowledge(context.Context, string, string, string, string, string, string, string) (bool, error)
	AddCommentReaction(context.Context, string, string, contract.CommentReaction) (contract.CommentReaction, error)
	RemoveCommentReaction(context.Context, string, string, string, string, string) error
	ListIssueReactions(context.Context, string, string) ([]contract.IssueReaction, error)
	AddIssueReaction(context.Context, string, string, contract.IssueReaction) (contract.IssueReaction, error)
	RemoveIssueReaction(context.Context, string, string, string, string, string) error
	ListIssueSubscribers(context.Context, string, string) ([]contract.IssueSubscriber, error)
	SetIssueSubscriber(context.Context, string, string, contract.IssueSubscriber, bool) error
	RecordActivity(context.Context, string, contract.IssueActivity) (contract.IssueActivity, error)
}

type IssueCollaborationUseCase struct {
	repository  IssueCollaborationRepository
	authorizer  contract.WorkspaceAccessAuthorizer
	actors      contract.WorkspaceActorReader
	memberships contract.WorkspaceMembershipReader
	assets      contract.WorkspaceAssetReader
	attachments contract.IssueAttachmentProjectionReader
	newID       ProjectIDGenerator
	now         Clock
}

func NewIssueCollaborationUseCase(repository IssueCollaborationRepository, authorizer contract.WorkspaceAccessAuthorizer, actors contract.WorkspaceActorReader, memberships contract.WorkspaceMembershipReader, assets contract.WorkspaceAssetReader, attachments contract.IssueAttachmentProjectionReader, newID ProjectIDGenerator, now Clock) (*IssueCollaborationUseCase, error) {
	if repository == nil || authorizer == nil || actors == nil || memberships == nil || assets == nil || newID == nil || now == nil {
		return nil, errors.New("Issue collaboration dependencies are required")
	}
	if attachments == nil {
		attachments = emptyIssueAttachmentProjection{}
	}
	return &IssueCollaborationUseCase{repository: repository, authorizer: authorizer, actors: actors, memberships: memberships, assets: assets, attachments: attachments, newID: newID, now: now}, nil
}

type emptyIssueAttachmentProjection struct{}

func (emptyIssueAttachmentProjection) ReadAttachments(context.Context, string, []string) ([]map[string]any, error) {
	return []map[string]any{}, nil
}

func (u *IssueCollaborationUseCase) ResolveIssueID(ctx context.Context, workspaceID, issueID string) (string, error) {
	_, resolved, err := u.issue(ctx, workspaceID, issueID, "workspace.issue.get")
	return resolved, err
}

func (u *IssueCollaborationUseCase) GetIssueComment(ctx context.Context, workspaceID, commentID string) (contract.IssueComment, error) {
	workspaceID, commentID = strings.TrimSpace(workspaceID), strings.TrimSpace(commentID)
	if workspaceID == "" || commentID == "" {
		return contract.IssueComment{}, ErrIssueCollaborationInvalid
	}
	if err := u.authorizer.AuthorizeWorkspace(ctx, workspaceID, "workspace.issue.comment.get"); err != nil {
		return contract.IssueComment{}, err
	}
	comment, err := u.repository.GetComment(ctx, workspaceID, commentID)
	if err != nil {
		return contract.IssueComment{}, err
	}
	return u.hydrateComment(ctx, workspaceID, comment)
}

func (u *IssueCollaborationUseCase) ListIssueComments(ctx context.Context, request contract.ListIssueCommentsRequest) (contract.ListIssueCommentsResponse, error) {
	workspaceID, issueID, err := u.issue(ctx, request.WorkspaceID, request.IssueID, "workspace.issue.comment.list")
	if err != nil {
		return contract.ListIssueCommentsResponse{}, err
	}
	comments, err := u.repository.ListComments(ctx, workspaceID, issueID)
	if err != nil {
		return contract.ListIssueCommentsResponse{}, fmt.Errorf("list Issue comments: %w", err)
	}
	if comments == nil {
		comments = []contract.IssueComment{}
	}
	if err := u.hydrateComments(ctx, workspaceID, comments); err != nil {
		return contract.ListIssueCommentsResponse{}, err
	}
	return contract.ListIssueCommentsResponse{Comments: comments}, nil
}

func (u *IssueCollaborationUseCase) ListIssueTimeline(ctx context.Context, request contract.ListIssueTimelineRequest) (contract.ListIssueTimelineResponse, error) {
	workspaceID, issueID, err := u.issue(ctx, request.WorkspaceID, request.IssueID, "workspace.issue.timeline.list")
	if err != nil {
		return contract.ListIssueTimelineResponse{}, err
	}
	comments, err := u.repository.ListComments(ctx, workspaceID, issueID)
	if err != nil {
		return contract.ListIssueTimelineResponse{}, fmt.Errorf("list Issue timeline comments: %w", err)
	}
	if err := u.hydrateComments(ctx, workspaceID, comments); err != nil {
		return contract.ListIssueTimelineResponse{}, err
	}
	activities, err := u.repository.ListActivities(ctx, workspaceID, issueID)
	if err != nil {
		return contract.ListIssueTimelineResponse{}, fmt.Errorf("list Issue timeline activities: %w", err)
	}
	entries := mergeIssueTimeline(comments, activities)
	return contract.ListIssueTimelineResponse{Entries: entries}, nil
}

func (u *IssueCollaborationUseCase) CreateIssueComment(ctx context.Context, request contract.CreateIssueCommentRequest) (contract.IssueComment, error) {
	workspaceID, issueID, err := u.issue(ctx, request.WorkspaceID, request.IssueID, "workspace.issue.comment.create")
	if err != nil {
		return contract.IssueComment{}, err
	}
	actor, err := u.actor(ctx, workspaceID)
	if err != nil {
		return contract.IssueComment{}, err
	}
	content := strings.TrimSpace(request.Content)
	commentType := strings.TrimSpace(request.Type)
	if commentType == "" {
		commentType = "comment"
	}
	if content == "" || commentType != "comment" {
		return contract.IssueComment{}, ErrIssueCollaborationInvalid
	}
	attachmentIDs, err := u.validateAttachmentIDs(ctx, workspaceID, request.AttachmentIDs)
	if err != nil {
		return contract.IssueComment{}, err
	}
	parentID := cleanStringPointer(request.ParentID)
	if parentID != nil {
		parent, getErr := u.repository.GetComment(ctx, workspaceID, *parentID)
		if getErr != nil || parent.IssueID != issueID {
			return contract.IssueComment{}, ErrIssueCollaborationInvalid
		}
	}
	id, err := u.newID(ctx)
	if err != nil {
		return contract.IssueComment{}, fmt.Errorf("generate Issue comment id: %w", err)
	}
	now := u.now().UTC().Format(time.RFC3339Nano)
	created, err := u.repository.CreateComment(ctx, contract.IssueComment{
		ID: id, WorkspaceID: workspaceID, IssueID: issueID, AuthorType: actor.Type, AuthorID: actor.ID,
		Content: content, Type: commentType, ParentID: parentID,
		Reactions: []contract.CommentReaction{}, Attachments: []map[string]any{}, AttachmentIDs: attachmentIDs, CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		return contract.IssueComment{}, err
	}
	return u.hydrateComment(ctx, workspaceID, created)
}

func (u *IssueCollaborationUseCase) UpdateIssueComment(ctx context.Context, request contract.UpdateIssueCommentRequest) (contract.IssueComment, error) {
	workspaceID := strings.TrimSpace(request.WorkspaceID)
	commentID := strings.TrimSpace(request.CommentID)
	content := strings.TrimSpace(request.Content)
	if workspaceID == "" || commentID == "" || content == "" {
		return contract.IssueComment{}, ErrIssueCollaborationInvalid
	}
	if err := u.authorizer.AuthorizeWorkspace(ctx, workspaceID, "workspace.issue.comment.update"); err != nil {
		return contract.IssueComment{}, err
	}
	comment, err := u.editableComment(ctx, workspaceID, commentID)
	if err != nil {
		return contract.IssueComment{}, err
	}
	attachmentIDs, err := u.validateAttachmentIDs(ctx, workspaceID, request.AttachmentIDs)
	if err != nil {
		return contract.IssueComment{}, err
	}
	updated, err := u.repository.UpdateComment(ctx, workspaceID, comment.ID, content, attachmentIDs, u.now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return contract.IssueComment{}, err
	}
	return u.hydrateComment(ctx, workspaceID, updated)
}

func (u *IssueCollaborationUseCase) DeleteIssueComment(ctx context.Context, request contract.DeleteIssueCommentRequest) error {
	workspaceID, commentID := strings.TrimSpace(request.WorkspaceID), strings.TrimSpace(request.CommentID)
	if workspaceID == "" || commentID == "" {
		return ErrIssueCollaborationInvalid
	}
	if err := u.authorizer.AuthorizeWorkspace(ctx, workspaceID, "workspace.issue.comment.delete"); err != nil {
		return err
	}
	comment, err := u.editableComment(ctx, workspaceID, commentID)
	if err != nil {
		return err
	}
	return u.repository.DeleteComment(ctx, workspaceID, comment.ID)
}

func (u *IssueCollaborationUseCase) validateAttachmentIDs(ctx context.Context, workspaceID string, raw []string) ([]string, error) {
	if len(raw) > 100 {
		return nil, ErrIssueCollaborationInvalid
	}
	result := make([]string, 0, len(raw))
	seen := make(map[string]struct{}, len(raw))
	for _, value := range raw {
		id := strings.TrimSpace(value)
		if id == "" || id != value {
			return nil, ErrIssueCollaborationInvalid
		}
		if _, duplicate := seen[id]; duplicate {
			continue
		}
		belongs, err := u.assets.AssetBelongsToWorkspace(ctx, workspaceID, id)
		if err != nil {
			return nil, err
		}
		if !belongs {
			return nil, ErrIssueCollaborationInvalid
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result, nil
}

func (u *IssueCollaborationUseCase) hydrateComment(ctx context.Context, workspaceID string, comment contract.IssueComment) (contract.IssueComment, error) {
	values, err := u.attachments.ReadAttachments(ctx, workspaceID, comment.AttachmentIDs)
	if err != nil {
		return contract.IssueComment{}, fmt.Errorf("read Issue comment attachments: %w", err)
	}
	comment.Attachments = values
	if comment.Attachments == nil {
		comment.Attachments = []map[string]any{}
	}
	return comment, nil
}

func (u *IssueCollaborationUseCase) hydrateComments(ctx context.Context, workspaceID string, comments []contract.IssueComment) error {
	for index := range comments {
		value, err := u.hydrateComment(ctx, workspaceID, comments[index])
		if err != nil {
			return err
		}
		comments[index] = value
	}
	return nil
}

func (u *IssueCollaborationUseCase) ResolveIssueComment(ctx context.Context, request contract.ResolveIssueCommentRequest) (contract.IssueComment, error) {
	workspaceID, commentID := strings.TrimSpace(request.WorkspaceID), strings.TrimSpace(request.CommentID)
	if workspaceID == "" || commentID == "" {
		return contract.IssueComment{}, ErrIssueCollaborationInvalid
	}
	if err := u.authorizer.AuthorizeWorkspace(ctx, workspaceID, "workspace.issue.comment.resolve"); err != nil {
		return contract.IssueComment{}, err
	}
	actor, err := u.actor(ctx, workspaceID)
	if err != nil {
		return contract.IssueComment{}, err
	}
	comment, err := u.repository.GetComment(ctx, workspaceID, commentID)
	if err != nil {
		return contract.IssueComment{}, err
	}
	now := u.now().UTC().Format(time.RFC3339Nano)
	updated, err := u.repository.ResolveComment(ctx, workspaceID, comment.ID, actor.Type, actor.ID, now, request.Resolved)
	if err != nil {
		return contract.IssueComment{}, err
	}
	return u.hydrateComment(ctx, workspaceID, updated)
}

func (u *IssueCollaborationUseCase) ProposeCommentKnowledge(ctx context.Context, request contract.ProposeCommentKnowledgeRequest) (contract.CommentKnowledgeProposalResponse, error) {
	workspaceID, commentID := strings.TrimSpace(request.WorkspaceID), strings.TrimSpace(request.CommentID)
	if workspaceID == "" || commentID == "" {
		return contract.CommentKnowledgeProposalResponse{}, ErrIssueCollaborationInvalid
	}
	if err := u.authorizer.AuthorizeWorkspace(ctx, workspaceID, "workspace.issue.comment.knowledge"); err != nil {
		return contract.CommentKnowledgeProposalResponse{}, err
	}
	actor, err := u.actor(ctx, workspaceID)
	if err != nil {
		return contract.CommentKnowledgeProposalResponse{}, err
	}
	comment, err := u.repository.GetComment(ctx, workspaceID, commentID)
	if err != nil {
		return contract.CommentKnowledgeProposalResponse{}, err
	}
	sum := sha256.Sum256([]byte(comment.Content))
	revision := comment.UpdatedAt + "@sha256:" + hex.EncodeToString(sum[:])
	evidenceID, err := u.newID(ctx)
	if err != nil {
		return contract.CommentKnowledgeProposalResponse{}, fmt.Errorf("generate comment evidence id: %w", err)
	}
	queued, err := u.repository.ProposeCommentKnowledge(ctx, workspaceID, comment.ID, evidenceID, revision, comment.Content, actor.ID, u.now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return contract.CommentKnowledgeProposalResponse{}, err
	}
	var resultID *string
	if queued {
		resultID = &evidenceID
	}
	return contract.CommentKnowledgeProposalResponse{Queued: queued, EvidenceID: resultID, SourceRevision: revision}, nil
}

func (u *IssueCollaborationUseCase) AddCommentReaction(ctx context.Context, request contract.ChangeCommentReactionRequest) (contract.CommentReaction, error) {
	workspaceID, commentID, emoji, actor, err := u.reactionIdentity(ctx, request.WorkspaceID, request.CommentID, request.Emoji, "workspace.issue.comment.react")
	if err != nil {
		return contract.CommentReaction{}, err
	}
	comment, err := u.repository.GetComment(ctx, workspaceID, commentID)
	if err != nil {
		return contract.CommentReaction{}, err
	}
	id, err := u.newID(ctx)
	if err != nil {
		return contract.CommentReaction{}, fmt.Errorf("generate comment reaction id: %w", err)
	}
	return u.repository.AddCommentReaction(ctx, workspaceID, comment.ID, contract.CommentReaction{ID: id, CommentID: comment.ID, ActorType: actor.Type, ActorID: actor.ID, Emoji: emoji, CreatedAt: u.now().UTC().Format(time.RFC3339Nano)})
}

func (u *IssueCollaborationUseCase) RemoveCommentReaction(ctx context.Context, request contract.ChangeCommentReactionRequest) error {
	workspaceID, commentID, emoji, actor, err := u.reactionIdentity(ctx, request.WorkspaceID, request.CommentID, request.Emoji, "workspace.issue.comment.react")
	if err != nil {
		return err
	}
	comment, err := u.repository.GetComment(ctx, workspaceID, commentID)
	if err != nil {
		return err
	}
	return u.repository.RemoveCommentReaction(ctx, workspaceID, comment.ID, actor.Type, actor.ID, emoji)
}

func (u *IssueCollaborationUseCase) ListIssueReactions(ctx context.Context, request contract.ListIssueReactionsRequest) (contract.ListIssueReactionsResponse, error) {
	workspaceID, issueID, err := u.issue(ctx, request.WorkspaceID, request.IssueID, "workspace.issue.reaction.list")
	if err != nil {
		return contract.ListIssueReactionsResponse{}, err
	}
	values, err := u.repository.ListIssueReactions(ctx, workspaceID, issueID)
	if err != nil {
		return contract.ListIssueReactionsResponse{}, err
	}
	if values == nil {
		values = []contract.IssueReaction{}
	}
	return contract.ListIssueReactionsResponse{Reactions: values}, nil
}

func (u *IssueCollaborationUseCase) AddIssueReaction(ctx context.Context, request contract.ChangeIssueReactionRequest) (contract.IssueReaction, error) {
	workspaceID, issueID, emoji, actor, err := u.issueReactionIdentity(ctx, request, "workspace.issue.reaction.put")
	if err != nil {
		return contract.IssueReaction{}, err
	}
	id, err := u.newID(ctx)
	if err != nil {
		return contract.IssueReaction{}, fmt.Errorf("generate Issue reaction id: %w", err)
	}
	return u.repository.AddIssueReaction(ctx, workspaceID, issueID, contract.IssueReaction{ID: id, IssueID: issueID, ActorType: actor.Type, ActorID: actor.ID, Emoji: emoji, CreatedAt: u.now().UTC().Format(time.RFC3339Nano)})
}

func (u *IssueCollaborationUseCase) RemoveIssueReaction(ctx context.Context, request contract.ChangeIssueReactionRequest) error {
	workspaceID, issueID, emoji, actor, err := u.issueReactionIdentity(ctx, request, "workspace.issue.reaction.delete")
	if err != nil {
		return err
	}
	return u.repository.RemoveIssueReaction(ctx, workspaceID, issueID, actor.Type, actor.ID, emoji)
}

func (u *IssueCollaborationUseCase) ListIssueSubscribers(ctx context.Context, request contract.ListIssueSubscribersRequest) (contract.ListIssueSubscribersResponse, error) {
	workspaceID, issueID, err := u.issue(ctx, request.WorkspaceID, request.IssueID, "workspace.issue.subscriber.list")
	if err != nil {
		return contract.ListIssueSubscribersResponse{}, err
	}
	values, err := u.repository.ListIssueSubscribers(ctx, workspaceID, issueID)
	if err != nil {
		return contract.ListIssueSubscribersResponse{}, err
	}
	if values == nil {
		values = []contract.IssueSubscriber{}
	}
	return contract.ListIssueSubscribersResponse{Subscribers: values}, nil
}

func (u *IssueCollaborationUseCase) SubscribeToIssue(ctx context.Context, request contract.ChangeIssueSubscriberRequest) error {
	return u.changeSubscriber(ctx, request, true)
}

func (u *IssueCollaborationUseCase) UnsubscribeFromIssue(ctx context.Context, request contract.ChangeIssueSubscriberRequest) error {
	return u.changeSubscriber(ctx, request, false)
}

func (u *IssueCollaborationUseCase) RecordIssueActivity(ctx context.Context, workspaceID, issueID, action string, details map[string]any) (contract.IssueActivity, error) {
	action = strings.TrimSpace(action)
	if action == "" {
		return contract.IssueActivity{}, ErrIssueCollaborationInvalid
	}
	workspaceID, issueID, err := u.issue(ctx, workspaceID, issueID, "workspace.issue.timeline.record")
	if err != nil {
		return contract.IssueActivity{}, err
	}
	actor, err := u.actor(ctx, workspaceID)
	if err != nil {
		return contract.IssueActivity{}, err
	}
	id, err := u.newID(ctx)
	if err != nil {
		return contract.IssueActivity{}, err
	}
	if details == nil {
		details = map[string]any{}
	}
	return u.repository.RecordActivity(ctx, workspaceID, contract.IssueActivity{ID: id, IssueID: issueID, ActorType: actor.Type, ActorID: actor.ID, Action: action, Details: details, CreatedAt: u.now().UTC().Format(time.RFC3339Nano)})
}

func (u *IssueCollaborationUseCase) issue(ctx context.Context, workspaceID, issueID, permission string) (string, string, error) {
	workspaceID, issueID = strings.TrimSpace(workspaceID), strings.TrimSpace(issueID)
	if workspaceID == "" || issueID == "" {
		return "", "", ErrIssueCollaborationInvalid
	}
	if err := u.authorizer.AuthorizeWorkspace(ctx, workspaceID, permission); err != nil {
		return "", "", err
	}
	resolved, err := u.repository.ResolveIssue(ctx, workspaceID, issueID)
	if err != nil {
		return "", "", err
	}
	return workspaceID, resolved, nil
}

func (u *IssueCollaborationUseCase) actor(ctx context.Context, workspaceID string) (contract.WorkspaceActor, error) {
	actor, ok := contract.WorkspaceActorFromContext(ctx)
	if !ok || (actor.Type != "member" && actor.Type != "agent") {
		return contract.WorkspaceActor{}, contract.ErrWorkspaceActorRequired
	}
	belongs, err := u.actors.ActorBelongsToWorkspace(ctx, workspaceID, actor.Type, actor.ID)
	if err != nil {
		return contract.WorkspaceActor{}, err
	}
	if !belongs {
		return contract.WorkspaceActor{}, contract.ErrActorOutsideWorkspace
	}
	return actor, nil
}

func (u *IssueCollaborationUseCase) editableComment(ctx context.Context, workspaceID, commentID string) (contract.IssueComment, error) {
	comment, err := u.repository.GetComment(ctx, workspaceID, commentID)
	if err != nil {
		return contract.IssueComment{}, err
	}
	actor, err := u.actor(ctx, workspaceID)
	if err != nil {
		return contract.IssueComment{}, err
	}
	if comment.AuthorType == actor.Type && comment.AuthorID == actor.ID {
		return comment, nil
	}
	if actor.Type != "member" {
		return contract.IssueComment{}, ErrIssueCommentPermission
	}
	membership, found, err := u.memberships.FindForUserAndWorkspace(ctx, actor.ID, workspaceID)
	if err != nil {
		return contract.IssueComment{}, err
	}
	if !found || (membership.Role != "owner" && membership.Role != "admin") {
		return contract.IssueComment{}, ErrIssueCommentPermission
	}
	return comment, nil
}

func (u *IssueCollaborationUseCase) reactionIdentity(ctx context.Context, workspaceID, commentID, emoji, permission string) (string, string, string, contract.WorkspaceActor, error) {
	workspaceID, commentID, emoji = strings.TrimSpace(workspaceID), strings.TrimSpace(commentID), strings.TrimSpace(emoji)
	if workspaceID == "" || commentID == "" || emoji == "" || len([]byte(emoji)) > 64 {
		return "", "", "", contract.WorkspaceActor{}, ErrIssueCollaborationInvalid
	}
	if err := u.authorizer.AuthorizeWorkspace(ctx, workspaceID, permission); err != nil {
		return "", "", "", contract.WorkspaceActor{}, err
	}
	actor, err := u.actor(ctx, workspaceID)
	return workspaceID, commentID, emoji, actor, err
}

func (u *IssueCollaborationUseCase) issueReactionIdentity(ctx context.Context, request contract.ChangeIssueReactionRequest, permission string) (string, string, string, contract.WorkspaceActor, error) {
	emoji := strings.TrimSpace(request.Emoji)
	if emoji == "" || len([]byte(emoji)) > 64 {
		return "", "", "", contract.WorkspaceActor{}, ErrIssueCollaborationInvalid
	}
	workspaceID, issueID, err := u.issue(ctx, request.WorkspaceID, request.IssueID, permission)
	if err != nil {
		return "", "", "", contract.WorkspaceActor{}, err
	}
	actor, err := u.actor(ctx, workspaceID)
	return workspaceID, issueID, emoji, actor, err
}

func (u *IssueCollaborationUseCase) changeSubscriber(ctx context.Context, request contract.ChangeIssueSubscriberRequest, subscribed bool) error {
	permission := "workspace.issue.subscriber.put"
	if !subscribed {
		permission = "workspace.issue.subscriber.delete"
	}
	workspaceID, issueID, err := u.issue(ctx, request.WorkspaceID, request.IssueID, permission)
	if err != nil {
		return err
	}
	actor, err := u.actor(ctx, workspaceID)
	if err != nil {
		return err
	}
	userType, userID := strings.TrimSpace(request.UserType), strings.TrimSpace(request.UserID)
	if userType == "" {
		userType = actor.Type
	}
	if userID == "" {
		userID = actor.ID
	}
	if userType != "member" {
		return ErrIssueCollaborationInvalid
	}
	belongs, err := u.actors.ActorBelongsToWorkspace(ctx, workspaceID, userType, userID)
	if err != nil {
		return err
	}
	if !belongs {
		return contract.ErrActorOutsideWorkspace
	}
	return u.repository.SetIssueSubscriber(ctx, workspaceID, issueID, contract.IssueSubscriber{IssueID: issueID, UserType: userType, UserID: userID, Reason: "manual", CreatedAt: u.now().UTC().Format(time.RFC3339Nano)}, subscribed)
}

func cleanStringPointer(value *string) *string {
	if value == nil {
		return nil
	}
	clean := strings.TrimSpace(*value)
	if clean == "" {
		return nil
	}
	return &clean
}

func mergeIssueTimeline(comments []contract.IssueComment, activities []contract.IssueActivity) []contract.IssueTimelineEntry {
	entries := make([]contract.IssueTimelineEntry, 0, len(comments)+len(activities))
	for _, comment := range comments {
		content, updatedAt, commentType := comment.Content, comment.UpdatedAt, comment.Type
		entries = append(entries, contract.IssueTimelineEntry{Type: "comment", ID: comment.ID, ActorType: comment.AuthorType, ActorID: comment.AuthorID, CreatedAt: comment.CreatedAt, Content: &content, ParentID: comment.ParentID, UpdatedAt: &updatedAt, CommentType: &commentType, Reactions: comment.Reactions, Attachments: comment.Attachments, ResolvedAt: comment.ResolvedAt, ResolvedByType: comment.ResolvedByType, ResolvedByID: comment.ResolvedByID})
	}
	for _, activity := range activities {
		action := activity.Action
		entries = append(entries, contract.IssueTimelineEntry{Type: "activity", ID: activity.ID, ActorType: activity.ActorType, ActorID: activity.ActorID, CreatedAt: activity.CreatedAt, Action: &action, Details: activity.Details})
	}
	sortTimelineEntries(entries)
	return entries
}

func sortTimelineEntries(values []contract.IssueTimelineEntry) {
	for index := 1; index < len(values); index++ {
		for cursor := index; cursor > 0; cursor-- {
			left, right := values[cursor-1], values[cursor]
			if left.CreatedAt < right.CreatedAt || (left.CreatedAt == right.CreatedAt && left.ID <= right.ID) {
				break
			}
			values[cursor-1], values[cursor] = right, left
		}
	}
}

var _ contract.IssueCollaborationService = (*IssueCollaborationUseCase)(nil)
var _ contract.IssueActivityRecorder = (*IssueCollaborationUseCase)(nil)
