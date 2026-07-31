package sqlitelocal

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/multica-ai/multica/server/internal/knowledge"
	"github.com/multica-ai/multica/server/internal/workspacepermissions"
)

type comment struct {
	ID, IssueID, WorkspaceID, AuthorType, AuthorID, Content, Type, CreatedAt, UpdatedAt string
	ParentID                                                                            sql.NullString
	ProjectID                                                                           sql.NullString
}

func commentColumns() string {
	return `c.id, c.issue_id, c.workspace_id, c.author_type, c.author_id, c.content, c.type,
		c.parent_id, c.created_at, c.updated_at`
}

func scanComment(scanner interface{ Scan(...any) error }) (comment, error) {
	var value comment
	err := scanner.Scan(
		&value.ID, &value.IssueID, &value.WorkspaceID, &value.AuthorType, &value.AuthorID,
		&value.Content, &value.Type, &value.ParentID, &value.CreatedAt, &value.UpdatedAt,
	)
	return value, err
}

func (value comment) response() map[string]any {
	return map[string]any{
		"id": value.ID, "issue_id": value.IssueID, "author_type": value.AuthorType,
		"author_id": value.AuthorID, "content": value.Content, "type": value.Type,
		"parent_id": nullable(value.ParentID.String), "reactions": []any{}, "attachments": []any{},
		"created_at": value.CreatedAt, "updated_at": value.UpdatedAt,
		"resolved_at": nil, "resolved_by_type": nil, "resolved_by_id": nil,
	}
}

func (value comment) timelineResponse() map[string]any {
	return map[string]any{
		"type": "comment", "id": value.ID, "actor_type": value.AuthorType,
		"actor_id": value.AuthorID, "content": value.Content,
		"parent_id": nullable(value.ParentID.String), "created_at": value.CreatedAt,
		"updated_at": value.UpdatedAt, "comment_type": value.Type,
		"reactions": []any{}, "attachments": []any{},
		"resolved_at": nil, "resolved_by_type": nil, "resolved_by_id": nil,
	}
}

func (s *Server) listComments(w http.ResponseWriter, r *http.Request) {
	issueValue, _, ok := s.loadIssue(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	rows, err := s.db.QueryContext(r.Context(), `SELECT `+commentColumns()+` FROM comments c
		WHERE c.issue_id = ? AND c.workspace_id = ? ORDER BY c.created_at, c.id`, issueValue.ID, issueValue.WorkspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list comments")
		return
	}
	defer rows.Close()
	items := make([]map[string]any, 0)
	for rows.Next() {
		value, err := scanComment(rows)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list comments")
			return
		}
		items = append(items, value.response())
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list comments")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) listIssueTimeline(w http.ResponseWriter, r *http.Request) {
	issueValue, _, ok := s.loadIssue(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	rows, err := s.db.QueryContext(r.Context(), `SELECT `+commentColumns()+` FROM comments c
		WHERE c.issue_id = ? AND c.workspace_id = ? ORDER BY c.created_at, c.id`, issueValue.ID, issueValue.WorkspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list issue timeline")
		return
	}
	defer rows.Close()
	items := make([]map[string]any, 0)
	for rows.Next() {
		value, err := scanComment(rows)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list issue timeline")
			return
		}
		items = append(items, value.timelineResponse())
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list issue timeline")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) createComment(w http.ResponseWriter, r *http.Request) {
	issueValue, _, ok := s.loadIssue(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var request struct {
		Content  string  `json:"content"`
		Type     string  `json:"type"`
		ParentID *string `json:"parent_id"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	request.Content = strings.TrimSpace(request.Content)
	if request.Content == "" {
		writeError(w, http.StatusBadRequest, "content is required")
		return
	}
	if request.Type == "" {
		request.Type = "comment"
	}
	if request.Type != "comment" {
		writeError(w, http.StatusBadRequest, "type must be comment")
		return
	}
	if request.ParentID != nil {
		var parentCount int
		err := s.db.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM comments
			WHERE id = ? AND issue_id = ? AND workspace_id = ?`, *request.ParentID, issueValue.ID, issueValue.WorkspaceID).Scan(&parentCount)
		if err != nil || parentCount == 0 {
			writeError(w, http.StatusBadRequest, "parent comment does not belong to this issue")
			return
		}
	}
	timestamp := now()
	value := comment{
		ID: newID(), IssueID: issueValue.ID, WorkspaceID: issueValue.WorkspaceID,
		AuthorType: "member", AuthorID: currentUserID(r), Content: request.Content,
		Type: request.Type, CreatedAt: timestamp, UpdatedAt: timestamp,
	}
	if request.ParentID != nil {
		value.ParentID = sql.NullString{String: *request.ParentID, Valid: true}
	}
	_, err := s.db.ExecContext(r.Context(), `INSERT INTO comments(
		id, issue_id, workspace_id, author_type, author_id, content, type, parent_id, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		value.ID, value.IssueID, value.WorkspaceID, value.AuthorType, value.AuthorID, value.Content,
		value.Type, value.ParentID, value.CreatedAt, value.UpdatedAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create comment")
		return
	}
	writeJSON(w, http.StatusCreated, value.response())
}

func (s *Server) loadComment(w http.ResponseWriter, r *http.Request) (comment, bool) {
	workspaceValue, ok := s.resolveWorkspace(w, r)
	if !ok {
		return comment{}, false
	}
	value, err := scanComment(s.db.QueryRowContext(r.Context(), `SELECT `+commentColumns()+`
		FROM comments c WHERE c.id = ? AND c.workspace_id = ?`, chi.URLParam(r, "commentId"), workspaceValue.ID))
	if err != nil {
		writeError(w, http.StatusNotFound, "comment not found")
		return comment{}, false
	}
	if err := s.db.QueryRowContext(r.Context(), `SELECT project_id FROM issues WHERE id = ? AND workspace_id = ?`, value.IssueID, value.WorkspaceID).Scan(&value.ProjectID); err != nil {
		writeError(w, http.StatusNotFound, "comment not found")
		return comment{}, false
	}
	return value, true
}

func (s *Server) proposeCommentDecision(w http.ResponseWriter, r *http.Request) {
	if !s.requireKnowledge(w) {
		return
	}
	value, ok := s.loadComment(w, r)
	if !ok {
		return
	}
	updatedAt, err := time.Parse(time.RFC3339Nano, value.UpdatedAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "invalid comment revision")
		return
	}
	evidence := knowledge.NewCommentDecisionEvidence(knowledge.CommentDecisionEvidenceDraft{
		WorkspaceID: value.WorkspaceID, ProjectID: nullableString(value.ProjectID), CommentID: value.ID,
		Content: value.Content, UpdatedAt: updatedAt, ActorID: currentUserID(r),
	})
	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to capture comment decision")
		return
	}
	defer tx.Rollback()
	queued, err := appendKnowledgeEvidence(r.Context(), tx, evidence)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to capture comment decision")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to capture comment decision")
		return
	}
	s.dispatchKnowledgeEvidence(r.Context())
	response := map[string]any{"queued": queued, "source_revision": evidence.SourceRevision, "evidence_id": nil}
	status := http.StatusOK
	if queued {
		response["evidence_id"] = evidence.ID
		status = http.StatusAccepted
	}
	writeJSON(w, status, response)
}

func (s *Server) updateComment(w http.ResponseWriter, r *http.Request) {
	value, ok := s.loadComment(w, r)
	if !ok {
		return
	}
	if value.AuthorType != "member" || value.AuthorID != currentUserID(r) {
		if !s.requireWorkspaceRole(
			w,
			r,
			value.WorkspaceID,
			workspacepermissions.RoleOwner,
			workspacepermissions.RoleAdmin,
		) {
			return
		}
	}
	var request struct {
		Content string `json:"content"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	request.Content = strings.TrimSpace(request.Content)
	if request.Content == "" {
		writeError(w, http.StatusBadRequest, "content is required")
		return
	}
	value.Content = request.Content
	value.UpdatedAt = now()
	_, err := s.db.ExecContext(r.Context(), `UPDATE comments SET content = ?, updated_at = ?
		WHERE id = ? AND workspace_id = ?`, value.Content, value.UpdatedAt, value.ID, value.WorkspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update comment")
		return
	}
	writeJSON(w, http.StatusOK, value.response())
}

func nullableString(value sql.NullString) string {
	if value.Valid {
		return value.String
	}
	return ""
}
