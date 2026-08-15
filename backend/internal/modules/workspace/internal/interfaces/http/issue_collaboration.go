package http

import (
	"context"
	"errors"
	"net/http"
	"strings"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
)

type IssueCollaborationHandler struct {
	service            contract.IssueCollaborationService
	identity           contract.WorkspaceHTTPIdentityResolver
	authenticate       func(*http.Request) (string, error)
	mutation           func(*http.Request) error
	attachmentsEnabled bool
}

func NewIssueCollaborationHandler(service contract.IssueCollaborationService, identity contract.WorkspaceHTTPIdentityResolver, authenticate func(*http.Request) (string, error), mutation func(*http.Request) error, attachmentsEnabled bool) *IssueCollaborationHandler {
	return &IssueCollaborationHandler{service: service, identity: identity, authenticate: authenticate, mutation: mutation, attachmentsEnabled: attachmentsEnabled}
}

func (h *IssueCollaborationHandler) Register(server *kratoshttp.Server) {
	router := server.Route("/")
	router.GET("/api/issues/{id}/comments", h.listComments)
	router.POST("/api/issues/{id}/comments", h.createComment)
	router.GET("/api/issues/{id}/timeline", h.listTimeline)
	router.GET("/api/issues/{id}/reactions", h.listIssueReactions)
	router.POST("/api/issues/{id}/reactions", h.addIssueReaction)
	router.DELETE("/api/issues/{id}/reactions", h.removeIssueReaction)
	router.GET("/api/issues/{id}/subscribers", h.listSubscribers)
	router.POST("/api/issues/{id}/subscribe", h.subscribe)
	router.POST("/api/issues/{id}/unsubscribe", h.unsubscribe)
	router.PUT("/api/comments/{commentId}", h.updateComment)
	router.DELETE("/api/comments/{commentId}", h.deleteComment)
	router.POST("/api/comments/{commentId}/resolve", h.resolveComment)
	router.DELETE("/api/comments/{commentId}/resolve", h.unresolveComment)
	router.POST("/api/comments/{commentId}/reactions", h.addCommentReaction)
	router.DELETE("/api/comments/{commentId}/reactions", h.removeCommentReaction)
	router.POST("/api/comments/{commentId}/knowledge-proposals", h.proposeKnowledge)
}

type createCommentHTTPRequest struct {
	Content       string           `json:"content"`
	Type          string           `json:"type"`
	ParentID      *string          `json:"parent_id"`
	AttachmentIDs stringSlicePatch `json:"attachment_ids"`
}

type updateCommentHTTPRequest struct {
	Content       string           `json:"content"`
	AttachmentIDs stringSlicePatch `json:"attachment_ids"`
}

type reactionHTTPRequest struct {
	Emoji string `json:"emoji"`
}
type subscriberHTTPRequest struct {
	UserID   string `json:"user_id"`
	UserType string `json:"user_type"`
}

func (h *IssueCollaborationHandler) listComments(ctx kratoshttp.Context) error {
	requestContext, workspaceID, ok := h.readIdentity(ctx)
	if !ok {
		return nil
	}
	response, err := h.service.ListIssueComments(requestContext, contract.ListIssueCommentsRequest{WorkspaceID: workspaceID, IssueID: ctx.Vars().Get("id")})
	if err != nil {
		return h.writeError(ctx, err, "issue", "list comments")
	}
	comments := make([]map[string]any, len(response.Comments))
	for index := range response.Comments {
		comments[index] = publicIssueComment(response.Comments[index])
	}
	return ctx.JSON(http.StatusOK, comments)
}

func (h *IssueCollaborationHandler) listTimeline(ctx kratoshttp.Context) error {
	requestContext, workspaceID, ok := h.readIdentity(ctx)
	if !ok {
		return nil
	}
	response, err := h.service.ListIssueTimeline(requestContext, contract.ListIssueTimelineRequest{WorkspaceID: workspaceID, IssueID: ctx.Vars().Get("id")})
	if err != nil {
		return h.writeError(ctx, err, "issue", "list timeline")
	}
	entries := make([]map[string]any, len(response.Entries))
	for index := range response.Entries {
		entries[index] = publicTimelineEntry(response.Entries[index])
	}
	return ctx.JSON(http.StatusOK, entries)
}

func (h *IssueCollaborationHandler) createComment(ctx kratoshttp.Context) error {
	requestContext, workspaceID, ok := h.mutationIdentity(ctx)
	if !ok {
		return nil
	}
	issueID, err := h.service.ResolveIssueID(requestContext, workspaceID, ctx.Vars().Get("id"))
	if err != nil {
		return h.writeError(ctx, err, "issue", "create comment")
	}
	var request createCommentHTTPRequest
	if err := decodeJSON(ctx.Request().Body, &request); err != nil {
		return writeError(ctx, http.StatusBadRequest, "invalid request body")
	}
	if !h.attachmentsEnabled && request.AttachmentIDs.Set {
		return writeError(ctx, http.StatusBadRequest, "unsupported comment attachment field")
	}
	response, err := h.service.CreateIssueComment(requestContext, contract.CreateIssueCommentRequest{WorkspaceID: workspaceID, IssueID: issueID, Content: request.Content, Type: request.Type, ParentID: request.ParentID, AttachmentIDs: attachmentIDValues(request.AttachmentIDs)})
	if err != nil {
		return h.writeError(ctx, err, "issue", "create comment")
	}
	return ctx.JSON(http.StatusCreated, publicIssueComment(response))
}

func (h *IssueCollaborationHandler) updateComment(ctx kratoshttp.Context) error {
	requestContext, workspaceID, ok := h.mutationIdentity(ctx)
	if !ok {
		return nil
	}
	comment, err := h.service.GetIssueComment(requestContext, workspaceID, ctx.Vars().Get("commentId"))
	if err != nil {
		return h.writeError(ctx, err, "comment", "update comment")
	}
	var request updateCommentHTTPRequest
	if err := decodeJSON(ctx.Request().Body, &request); err != nil {
		return writeError(ctx, http.StatusBadRequest, "invalid request body")
	}
	if !h.attachmentsEnabled && request.AttachmentIDs.Set {
		return writeError(ctx, http.StatusBadRequest, "unsupported comment attachment field")
	}
	response, err := h.service.UpdateIssueComment(requestContext, contract.UpdateIssueCommentRequest{WorkspaceID: workspaceID, CommentID: comment.ID, Content: request.Content, AttachmentIDs: attachmentIDValues(request.AttachmentIDs)})
	if err != nil {
		return h.writeError(ctx, err, "comment", "update comment")
	}
	return ctx.JSON(http.StatusOK, publicIssueComment(response))
}

func attachmentIDValues(values stringSlicePatch) []string {
	if !values.Set || values.Value == nil {
		return nil
	}
	return append([]string(nil), (*values.Value)...)
}

func (h *IssueCollaborationHandler) deleteComment(ctx kratoshttp.Context) error {
	requestContext, workspaceID, ok := h.mutationIdentity(ctx)
	if !ok {
		return nil
	}
	err := h.service.DeleteIssueComment(requestContext, contract.DeleteIssueCommentRequest{WorkspaceID: workspaceID, CommentID: ctx.Vars().Get("commentId")})
	if err != nil {
		return h.writeError(ctx, err, "comment", "delete comment")
	}
	ctx.Response().WriteHeader(http.StatusNoContent)
	return nil
}

func (h *IssueCollaborationHandler) resolveComment(ctx kratoshttp.Context) error {
	return h.changeResolution(ctx, true)
}

func (h *IssueCollaborationHandler) unresolveComment(ctx kratoshttp.Context) error {
	return h.changeResolution(ctx, false)
}

func (h *IssueCollaborationHandler) changeResolution(ctx kratoshttp.Context, resolved bool) error {
	requestContext, workspaceID, ok := h.mutationIdentity(ctx)
	if !ok {
		return nil
	}
	response, err := h.service.ResolveIssueComment(requestContext, contract.ResolveIssueCommentRequest{WorkspaceID: workspaceID, CommentID: ctx.Vars().Get("commentId"), Resolved: resolved})
	if err != nil {
		return h.writeError(ctx, err, "comment", "update comment")
	}
	return ctx.JSON(http.StatusOK, publicIssueComment(response))
}

func (h *IssueCollaborationHandler) proposeKnowledge(ctx kratoshttp.Context) error {
	requestContext, workspaceID, ok := h.mutationIdentity(ctx)
	if !ok {
		return nil
	}
	response, err := h.service.ProposeCommentKnowledge(requestContext, contract.ProposeCommentKnowledgeRequest{WorkspaceID: workspaceID, CommentID: ctx.Vars().Get("commentId")})
	if err != nil {
		return h.writeError(ctx, err, "comment", "capture comment knowledge")
	}
	status := http.StatusOK
	if response.Queued {
		status = http.StatusAccepted
	}
	return ctx.JSON(status, map[string]any{"queued": response.Queued, "evidence_id": response.EvidenceID, "source_revision": response.SourceRevision})
}

func (h *IssueCollaborationHandler) addCommentReaction(ctx kratoshttp.Context) error {
	requestContext, workspaceID, ok := h.mutationIdentity(ctx)
	if !ok {
		return nil
	}
	comment, err := h.service.GetIssueComment(requestContext, workspaceID, ctx.Vars().Get("commentId"))
	if err != nil {
		return h.writeError(ctx, err, "comment", "add reaction")
	}
	request, ok := h.decodeReaction(ctx)
	if !ok {
		return nil
	}
	response, err := h.service.AddCommentReaction(requestContext, contract.ChangeCommentReactionRequest{WorkspaceID: workspaceID, CommentID: comment.ID, Emoji: request.Emoji})
	if err != nil {
		return h.writeError(ctx, err, "comment", "add reaction")
	}
	return ctx.JSON(http.StatusCreated, publicCommentReaction(response))
}

func (h *IssueCollaborationHandler) removeCommentReaction(ctx kratoshttp.Context) error {
	requestContext, workspaceID, ok := h.mutationIdentity(ctx)
	if !ok {
		return nil
	}
	comment, err := h.service.GetIssueComment(requestContext, workspaceID, ctx.Vars().Get("commentId"))
	if err != nil {
		return h.writeError(ctx, err, "comment", "remove reaction")
	}
	request, ok := h.decodeReaction(ctx)
	if !ok {
		return nil
	}
	err = h.service.RemoveCommentReaction(requestContext, contract.ChangeCommentReactionRequest{WorkspaceID: workspaceID, CommentID: comment.ID, Emoji: request.Emoji})
	if err != nil {
		return h.writeError(ctx, err, "comment", "remove reaction")
	}
	ctx.Response().WriteHeader(http.StatusNoContent)
	return nil
}

func (h *IssueCollaborationHandler) listIssueReactions(ctx kratoshttp.Context) error {
	requestContext, workspaceID, ok := h.readIdentity(ctx)
	if !ok {
		return nil
	}
	response, err := h.service.ListIssueReactions(requestContext, contract.ListIssueReactionsRequest{WorkspaceID: workspaceID, IssueID: ctx.Vars().Get("id")})
	if err != nil {
		return h.writeError(ctx, err, "issue", "list reactions")
	}
	values := make([]map[string]any, len(response.Reactions))
	for index := range response.Reactions {
		values[index] = publicIssueReaction(response.Reactions[index])
	}
	return ctx.JSON(http.StatusOK, values)
}

func (h *IssueCollaborationHandler) addIssueReaction(ctx kratoshttp.Context) error {
	requestContext, workspaceID, ok := h.mutationIdentity(ctx)
	if !ok {
		return nil
	}
	issueID, err := h.service.ResolveIssueID(requestContext, workspaceID, ctx.Vars().Get("id"))
	if err != nil {
		return h.writeError(ctx, err, "issue", "add reaction")
	}
	request, ok := h.decodeReaction(ctx)
	if !ok {
		return nil
	}
	response, err := h.service.AddIssueReaction(requestContext, contract.ChangeIssueReactionRequest{WorkspaceID: workspaceID, IssueID: issueID, Emoji: request.Emoji})
	if err != nil {
		return h.writeError(ctx, err, "issue", "add reaction")
	}
	return ctx.JSON(http.StatusCreated, publicIssueReaction(response))
}

func (h *IssueCollaborationHandler) removeIssueReaction(ctx kratoshttp.Context) error {
	requestContext, workspaceID, ok := h.mutationIdentity(ctx)
	if !ok {
		return nil
	}
	issueID, err := h.service.ResolveIssueID(requestContext, workspaceID, ctx.Vars().Get("id"))
	if err != nil {
		return h.writeError(ctx, err, "issue", "remove reaction")
	}
	request, ok := h.decodeReaction(ctx)
	if !ok {
		return nil
	}
	err = h.service.RemoveIssueReaction(requestContext, contract.ChangeIssueReactionRequest{WorkspaceID: workspaceID, IssueID: issueID, Emoji: request.Emoji})
	if err != nil {
		return h.writeError(ctx, err, "issue", "remove reaction")
	}
	ctx.Response().WriteHeader(http.StatusNoContent)
	return nil
}

func (h *IssueCollaborationHandler) listSubscribers(ctx kratoshttp.Context) error {
	requestContext, workspaceID, ok := h.readIdentity(ctx)
	if !ok {
		return nil
	}
	response, err := h.service.ListIssueSubscribers(requestContext, contract.ListIssueSubscribersRequest{WorkspaceID: workspaceID, IssueID: ctx.Vars().Get("id")})
	if err != nil {
		return h.writeError(ctx, err, "issue", "list subscribers")
	}
	values := make([]map[string]any, len(response.Subscribers))
	for index, value := range response.Subscribers {
		values[index] = map[string]any{"issue_id": value.IssueID, "user_type": value.UserType, "user_id": value.UserID, "reason": value.Reason, "created_at": value.CreatedAt}
	}
	return ctx.JSON(http.StatusOK, values)
}

func (h *IssueCollaborationHandler) subscribe(ctx kratoshttp.Context) error {
	return h.changeSubscriber(ctx, true)
}

func (h *IssueCollaborationHandler) unsubscribe(ctx kratoshttp.Context) error {
	return h.changeSubscriber(ctx, false)
}

func (h *IssueCollaborationHandler) changeSubscriber(ctx kratoshttp.Context, subscribed bool) error {
	requestContext, workspaceID, ok := h.mutationIdentity(ctx)
	if !ok {
		return nil
	}
	issueID, err := h.service.ResolveIssueID(requestContext, workspaceID, ctx.Vars().Get("id"))
	if err != nil {
		return h.writeError(ctx, err, "issue", "change subscriber")
	}
	var request subscriberHTTPRequest
	if err := decodeJSON(ctx.Request().Body, &request); err != nil {
		return writeError(ctx, http.StatusBadRequest, "invalid request body")
	}
	change := contract.ChangeIssueSubscriberRequest{WorkspaceID: workspaceID, IssueID: issueID, UserType: request.UserType, UserID: request.UserID}
	if subscribed {
		err = h.service.SubscribeToIssue(requestContext, change)
	} else {
		err = h.service.UnsubscribeFromIssue(requestContext, change)
	}
	if err != nil {
		return h.writeError(ctx, err, "issue", "change subscriber")
	}
	return ctx.JSON(http.StatusOK, map[string]bool{"subscribed": subscribed})
}

func (h *IssueCollaborationHandler) decodeReaction(ctx kratoshttp.Context) (reactionHTTPRequest, bool) {
	var request reactionHTTPRequest
	if err := decodeJSON(ctx.Request().Body, &request); err != nil {
		_ = writeError(ctx, http.StatusBadRequest, "invalid request body")
		return reactionHTTPRequest{}, false
	}
	return request, true
}

func (h *IssueCollaborationHandler) readIdentity(ctx kratoshttp.Context) (context.Context, string, bool) {
	if h.authenticate == nil {
		_ = writeError(ctx, http.StatusUnauthorized, "user not authenticated")
		return nil, "", false
	}
	if _, err := h.authenticate(ctx.Request()); err != nil {
		_ = writeError(ctx, http.StatusUnauthorized, "user not authenticated")
		return nil, "", false
	}
	if !hasWorkspaceIdentity(ctx) {
		_ = writeError(ctx, http.StatusBadRequest, "workspace is required")
		return nil, "", false
	}
	if h.identity == nil {
		_ = writeError(ctx, http.StatusUnauthorized, "user not authenticated")
		return nil, "", false
	}
	identity, err := h.identity(ctx.Request())
	if err != nil {
		_ = issueReadIdentityError(ctx, err)
		return nil, "", false
	}
	if strings.TrimSpace(identity.WorkspaceID) == "" || strings.TrimSpace(identity.ActorID) == "" {
		_ = writeError(ctx, http.StatusUnauthorized, "user not authenticated")
		return nil, "", false
	}
	return contract.WithWorkspaceActor(ctx.Request().Context(), identity.ActorType, identity.ActorID), identity.WorkspaceID, true
}

func (h *IssueCollaborationHandler) mutationIdentity(ctx kratoshttp.Context) (context.Context, string, bool) {
	requestContext, workspaceID, ok := h.readIdentity(ctx)
	if !ok {
		return nil, "", false
	}
	if h.mutation == nil || h.mutation(ctx.Request()) != nil {
		_ = writeError(ctx, http.StatusForbidden, "invalid CSRF token")
		return nil, "", false
	}
	return requestContext, workspaceID, true
}

func (h *IssueCollaborationHandler) writeError(ctx kratoshttp.Context, err error, target, operation string) error {
	if errors.Is(err, application.ErrIssueCollaborationInvalid) || errors.Is(err, contract.ErrInvalidIssue) {
		return writeError(ctx, http.StatusBadRequest, "invalid request")
	}
	if errors.Is(err, application.ErrIssueCommentPermission) || errors.Is(err, contract.ErrWorkspacePermissionDenied) {
		return writeError(ctx, http.StatusForbidden, "insufficient permissions")
	}
	if errors.Is(err, application.ErrIssueCommentNotFound) {
		return writeError(ctx, http.StatusNotFound, "comment not found")
	}
	if errors.Is(err, application.ErrIssueRecordNotFound) || errors.Is(err, contract.ErrIssueNotFound) || errors.Is(err, contract.ErrActorOutsideWorkspace) || errors.Is(err, contract.ErrWorkspaceNotFound) {
		return writeError(ctx, http.StatusNotFound, target+" not found")
	}
	return writeError(ctx, http.StatusInternalServerError, "failed to "+operation)
}

func publicIssueComment(value contract.IssueComment) map[string]any {
	reactions := make([]map[string]any, len(value.Reactions))
	for index := range value.Reactions {
		reactions[index] = publicCommentReaction(value.Reactions[index])
	}
	attachments := value.Attachments
	if attachments == nil {
		attachments = []map[string]any{}
	}
	return map[string]any{"id": value.ID, "issue_id": value.IssueID, "author_type": value.AuthorType, "author_id": value.AuthorID, "content": value.Content, "type": value.Type, "parent_id": value.ParentID, "reactions": reactions, "attachments": attachments, "created_at": value.CreatedAt, "updated_at": value.UpdatedAt, "resolved_at": value.ResolvedAt, "resolved_by_type": value.ResolvedByType, "resolved_by_id": value.ResolvedByID}
}

func publicTimelineEntry(value contract.IssueTimelineEntry) map[string]any {
	result := map[string]any{"type": value.Type, "id": value.ID, "actor_type": value.ActorType, "actor_id": value.ActorID, "created_at": value.CreatedAt}
	if value.Type == "comment" {
		reactions := make([]map[string]any, len(value.Reactions))
		for index := range value.Reactions {
			reactions[index] = publicCommentReaction(value.Reactions[index])
		}
		attachments := value.Attachments
		if attachments == nil {
			attachments = []map[string]any{}
		}
		result["content"], result["parent_id"], result["updated_at"], result["comment_type"] = value.Content, value.ParentID, value.UpdatedAt, value.CommentType
		result["reactions"], result["attachments"] = reactions, attachments
		result["resolved_at"], result["resolved_by_type"], result["resolved_by_id"] = value.ResolvedAt, value.ResolvedByType, value.ResolvedByID
	} else {
		result["action"], result["details"] = value.Action, value.Details
	}
	return result
}

func publicCommentReaction(value contract.CommentReaction) map[string]any {
	return map[string]any{"id": value.ID, "comment_id": value.CommentID, "actor_type": value.ActorType, "actor_id": value.ActorID, "emoji": value.Emoji, "created_at": value.CreatedAt}
}

func publicIssueReaction(value contract.IssueReaction) map[string]any {
	return map[string]any{"id": value.ID, "issue_id": value.IssueID, "actor_type": value.ActorType, "actor_id": value.ActorID, "emoji": value.Emoji, "created_at": value.CreatedAt}
}
