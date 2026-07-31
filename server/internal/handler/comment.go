package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/knowledge"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
	"github.com/multica-ai/multica/server/pkg/protocol"
)

type CommentResponse struct {
	ID             string               `json:"id"`
	IssueID        string               `json:"issue_id"`
	AuthorType     string               `json:"author_type"`
	AuthorID       string               `json:"author_id"`
	Content        string               `json:"content"`
	Type           string               `json:"type"`
	ParentID       *string              `json:"parent_id"`
	CreatedAt      string               `json:"created_at"`
	UpdatedAt      string               `json:"updated_at"`
	ResolvedAt     *string              `json:"resolved_at"`
	ResolvedByType *string              `json:"resolved_by_type"`
	ResolvedByID   *string              `json:"resolved_by_id"`
	Reactions      []ReactionResponse   `json:"reactions"`
	Attachments    []AttachmentResponse `json:"attachments"`
}

type CommentKnowledgeProposalResponse struct {
	Queued         bool    `json:"queued"`
	EvidenceID     *string `json:"evidence_id"`
	SourceRevision string  `json:"source_revision"`
}

func (h *Handler) commentResponse(c db.Comment, reactions []ReactionResponse, attachments []AttachmentResponse) CommentResponse {
	if reactions == nil {
		reactions = []ReactionResponse{}
	}
	if attachments == nil {
		attachments = []AttachmentResponse{}
	}
	return CommentResponse{
		ID:             uuidToString(c.ID),
		IssueID:        uuidToString(c.IssueID),
		AuthorType:     c.AuthorType,
		AuthorID:       uuidToString(c.AuthorID),
		Content:        c.Content,
		Type:           c.Type,
		ParentID:       uuidToPtr(c.ParentID),
		CreatedAt:      timestampToString(c.CreatedAt),
		UpdatedAt:      timestampToString(c.UpdatedAt),
		ResolvedAt:     timestampToPtr(c.ResolvedAt),
		ResolvedByType: textToPtr(c.ResolvedByType),
		ResolvedByID:   uuidToPtr(c.ResolvedByID),
		Reactions:      reactions,
		Attachments:    attachments,
	}
}

func (h *Handler) ListComments(w http.ResponseWriter, r *http.Request) {
	issue, ok := h.loadIssueForUser(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	comments, err := h.Queries.ListCommentsForIssue(r.Context(), db.ListCommentsForIssueParams{
		IssueID: issue.ID, WorkspaceID: issue.WorkspaceID, Limit: 2000,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list comments")
		return
	}
	ids := make([]pgtype.UUID, len(comments))
	for i := range comments {
		ids[i] = comments[i].ID
	}
	reactions := h.groupReactions(r, ids)
	attachments := h.groupAttachments(r, ids)
	response := make([]CommentResponse, len(comments))
	for i, comment := range comments {
		id := uuidToString(comment.ID)
		response[i] = h.commentResponse(comment, reactions[id], attachments[id])
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) CreateComment(w http.ResponseWriter, r *http.Request) {
	issue, ok := h.loadIssueForUser(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	var req struct {
		Content       string   `json:"content"`
		Type          string   `json:"type"`
		ParentID      *string  `json:"parent_id"`
		AttachmentIDs []string `json:"attachment_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Content = strings.TrimSpace(req.Content)
	if req.Content == "" {
		writeError(w, http.StatusBadRequest, "content is required")
		return
	}
	if req.Type == "" {
		req.Type = "comment"
	}
	if req.Type != "comment" {
		writeError(w, http.StatusBadRequest, "type must be comment")
		return
	}
	attachmentIDs, ok := parseUUIDSliceOrBadRequest(w, req.AttachmentIDs, "attachment_ids")
	if !ok {
		return
	}
	parentID := pgtype.UUID{}
	if req.ParentID != nil {
		parentID, ok = parseUUIDOrBadRequest(w, *req.ParentID, "parent_id")
		if !ok {
			return
		}
		parent, err := h.Queries.GetCommentInWorkspace(r.Context(), db.GetCommentInWorkspaceParams{
			ID: parentID, WorkspaceID: issue.WorkspaceID,
		})
		if err != nil || parent.IssueID != issue.ID {
			writeError(w, http.StatusBadRequest, "parent comment does not belong to this issue")
			return
		}
	}

	tx, err := h.TxStarter.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create comment")
		return
	}
	defer tx.Rollback(r.Context())
	qtx := h.Queries.WithTx(tx)
	comment, err := qtx.CreateComment(r.Context(), db.CreateCommentParams{
		IssueID: issue.ID, WorkspaceID: issue.WorkspaceID,
		AuthorID: parseUUID(userID), Content: req.Content, Type: req.Type, ParentID: parentID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create comment")
		return
	}
	if len(attachmentIDs) > 0 {
		if err := qtx.LinkAttachmentsToComment(r.Context(), db.LinkAttachmentsToCommentParams{
			CommentID: comment.ID, IssueID: issue.ID, Column3: attachmentIDs,
		}); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to attach files")
			return
		}
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create comment")
		return
	}
	attachments := h.groupAttachments(r, []pgtype.UUID{comment.ID})
	response := h.commentResponse(comment, nil, attachments[uuidToString(comment.ID)])
	h.publish(protocol.EventCommentCreated, uuidToString(issue.WorkspaceID), "member", userID, map[string]any{"comment": response})
	writeJSON(w, http.StatusCreated, response)
}

func (h *Handler) ProposeCommentDecision(w http.ResponseWriter, r *http.Request) {
	if !h.knowledgeEvidenceEnabled {
		writeError(w, http.StatusServiceUnavailable, "knowledge capture unavailable")
		return
	}
	workspaceID, _, ok := knowledgeAccess(r)
	if !ok {
		writeError(w, http.StatusForbidden, "workspace access denied")
		return
	}
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	commentID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "commentId"), "comment id")
	if !ok {
		return
	}

	tx, err := h.TxStarter.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to capture comment decision")
		return
	}
	defer tx.Rollback(r.Context())
	qtx := h.Queries.WithTx(tx)
	source, err := qtx.GetCommentKnowledgeSourceForUpdate(r.Context(), db.GetCommentKnowledgeSourceForUpdateParams{
		ID:          commentID,
		WorkspaceID: parseUUID(workspaceID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "comment not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to capture comment decision")
		return
	}
	evidence := knowledge.NewCommentDecisionEvidence(knowledge.CommentDecisionEvidenceDraft{
		WorkspaceID: uuidToString(source.WorkspaceID),
		ProjectID:   optionalKnowledgeUUID(source.IssueProjectID),
		CommentID:   uuidToString(source.ID),
		Content:     source.Content,
		UpdatedAt:   source.UpdatedAt.Time,
		ActorID:     userID,
	})
	queued, err := h.appendKnowledgeEvidence(r.Context(), tx, evidence)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to capture comment decision")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to capture comment decision")
		return
	}

	var evidenceID *string
	status := http.StatusOK
	if queued {
		evidenceID = &evidence.ID
		status = http.StatusAccepted
	}
	writeJSON(w, status, CommentKnowledgeProposalResponse{
		Queued: queued, EvidenceID: evidenceID, SourceRevision: evidence.SourceRevision,
	})
}

func (h *Handler) loadCommentForMember(w http.ResponseWriter, r *http.Request) (db.Comment, db.Member, bool) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return db.Comment{}, db.Member{}, false
	}
	workspaceID := h.resolveWorkspaceID(r)
	id, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "commentId"), "comment id")
	if !ok {
		return db.Comment{}, db.Member{}, false
	}
	comment, err := h.Queries.GetCommentInWorkspace(r.Context(), db.GetCommentInWorkspaceParams{
		ID: id, WorkspaceID: parseUUID(workspaceID),
	})
	if err != nil {
		writeError(w, http.StatusNotFound, "comment not found")
		return db.Comment{}, db.Member{}, false
	}
	member, err := h.getWorkspaceMember(r.Context(), userID, workspaceID)
	if err != nil {
		writeError(w, http.StatusForbidden, "not a member of this workspace")
		return db.Comment{}, db.Member{}, false
	}
	return comment, member, true
}

func canEditComment(comment db.Comment, member db.Member) bool {
	return comment.AuthorType == "member" && comment.AuthorID == member.UserID ||
		member.Role == "owner" || member.Role == "admin"
}

func (h *Handler) UpdateComment(w http.ResponseWriter, r *http.Request) {
	comment, member, ok := h.loadCommentForMember(w, r)
	if !ok {
		return
	}
	if !canEditComment(comment, member) {
		writeError(w, http.StatusForbidden, "insufficient permissions")
		return
	}
	var req struct {
		Content       string   `json:"content"`
		AttachmentIDs []string `json:"attachment_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Content = strings.TrimSpace(req.Content)
	if req.Content == "" {
		writeError(w, http.StatusBadRequest, "content is required")
		return
	}
	attachmentIDs, ok := parseUUIDSliceOrBadRequest(w, req.AttachmentIDs, "attachment_ids")
	if !ok {
		return
	}
	tx, err := h.TxStarter.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update comment")
		return
	}
	defer tx.Rollback(r.Context())
	qtx := h.Queries.WithTx(tx)
	updated, err := qtx.UpdateComment(r.Context(), db.UpdateCommentParams{
		ID: comment.ID, WorkspaceID: comment.WorkspaceID, Content: req.Content,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update comment")
		return
	}
	if err := qtx.ReplaceCommentAttachments(r.Context(), db.ReplaceCommentAttachmentsParams{
		CommentID: comment.ID, IssueID: comment.IssueID, AttachmentIds: attachmentIDs,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update attachments")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update comment")
		return
	}
	reactions := h.groupReactions(r, []pgtype.UUID{comment.ID})
	attachments := h.groupAttachments(r, []pgtype.UUID{comment.ID})
	id := uuidToString(comment.ID)
	response := h.commentResponse(updated, reactions[id], attachments[id])
	h.publish(protocol.EventCommentUpdated, uuidToString(comment.WorkspaceID), "member", uuidToString(member.UserID), map[string]any{"comment": response})
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	comment, member, ok := h.loadCommentForMember(w, r)
	if !ok {
		return
	}
	if !canEditComment(comment, member) {
		writeError(w, http.StatusForbidden, "insufficient permissions")
		return
	}
	urls, _ := h.Queries.ListAttachmentURLsByCommentID(r.Context(), comment.ID)
	if err := h.Queries.DeleteComment(r.Context(), db.DeleteCommentParams{ID: comment.ID, WorkspaceID: comment.WorkspaceID}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete comment")
		return
	}
	h.deleteS3Objects(r.Context(), urls)
	h.publish(protocol.EventCommentDeleted, uuidToString(comment.WorkspaceID), "member", uuidToString(member.UserID), map[string]any{
		"comment_id": uuidToString(comment.ID), "issue_id": uuidToString(comment.IssueID),
	})
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ResolveComment(w http.ResponseWriter, r *http.Request) {
	h.setCommentResolved(w, r, true)
}

func (h *Handler) UnresolveComment(w http.ResponseWriter, r *http.Request) {
	h.setCommentResolved(w, r, false)
}

func (h *Handler) setCommentResolved(w http.ResponseWriter, r *http.Request, resolved bool) {
	comment, member, ok := h.loadCommentForMember(w, r)
	if !ok {
		return
	}
	var updated db.Comment
	var err error
	if resolved {
		updated, err = h.Queries.ResolveComment(r.Context(), db.ResolveCommentParams{
			ID: comment.ID, WorkspaceID: comment.WorkspaceID, ResolvedByID: member.UserID,
		})
	} else {
		updated, err = h.Queries.UnresolveComment(r.Context(), db.UnresolveCommentParams{
			ID: comment.ID, WorkspaceID: comment.WorkspaceID,
		})
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "comment not found")
		} else {
			writeError(w, http.StatusInternalServerError, "failed to update comment")
		}
		return
	}
	id := uuidToString(comment.ID)
	reactions := h.groupReactions(r, []pgtype.UUID{comment.ID})
	attachments := h.groupAttachments(r, []pgtype.UUID{comment.ID})
	response := h.commentResponse(updated, reactions[id], attachments[id])
	event := protocol.EventCommentUnresolved
	if resolved {
		event = protocol.EventCommentResolved
	}
	h.publish(event, uuidToString(comment.WorkspaceID), "member", uuidToString(member.UserID), map[string]any{"comment": response})
	writeJSON(w, http.StatusOK, response)
}
