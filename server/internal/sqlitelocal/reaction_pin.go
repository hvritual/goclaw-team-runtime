package sqlitelocal

import (
	"context"
	"database/sql"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

type issueReaction struct {
	ID, IssueID, WorkspaceID, ActorType, ActorID, Emoji, CreatedAt string
}

func scanIssueReaction(scanner interface{ Scan(...any) error }) (issueReaction, error) {
	var value issueReaction
	err := scanner.Scan(
		&value.ID,
		&value.IssueID,
		&value.WorkspaceID,
		&value.ActorType,
		&value.ActorID,
		&value.Emoji,
		&value.CreatedAt,
	)
	return value, err
}

func (value issueReaction) response() map[string]any {
	return map[string]any{
		"id":         value.ID,
		"issue_id":   value.IssueID,
		"actor_type": value.ActorType,
		"actor_id":   value.ActorID,
		"emoji":      value.Emoji,
		"created_at": value.CreatedAt,
	}
}

func (s *Server) listIssueReactions(ctx context.Context, issueID, workspaceID string) ([]map[string]any, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, issue_id, workspace_id, actor_type, actor_id, emoji, created_at
		FROM issue_reactions WHERE issue_id = ? AND workspace_id = ? ORDER BY created_at, id`, issueID, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]map[string]any, 0)
	for rows.Next() {
		value, err := scanIssueReaction(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, value.response())
	}
	return items, rows.Err()
}

func (s *Server) addIssueReaction(w http.ResponseWriter, r *http.Request) {
	issueValue, _, ok := s.loadIssue(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var request struct {
		Emoji string `json:"emoji"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	if request.Emoji == "" {
		writeError(w, http.StatusBadRequest, "emoji is required")
		return
	}

	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to add reaction")
		return
	}
	defer tx.Rollback()

	value := issueReaction{
		ID:          newID(),
		IssueID:     issueValue.ID,
		WorkspaceID: issueValue.WorkspaceID,
		ActorType:   "member",
		ActorID:     currentUserID(r),
		Emoji:       request.Emoji,
		CreatedAt:   now(),
	}
	if _, err := tx.ExecContext(r.Context(), `INSERT OR IGNORE INTO issue_reactions(
		id, issue_id, workspace_id, actor_type, actor_id, emoji, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?)`, value.ID, value.IssueID, value.WorkspaceID, value.ActorType, value.ActorID, value.Emoji, value.CreatedAt); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to add reaction")
		return
	}
	value, err = scanIssueReaction(tx.QueryRowContext(r.Context(), `SELECT id, issue_id, workspace_id, actor_type, actor_id, emoji, created_at
		FROM issue_reactions WHERE issue_id = ? AND workspace_id = ? AND actor_type = ? AND actor_id = ? AND emoji = ?`,
		value.IssueID, value.WorkspaceID, value.ActorType, value.ActorID, value.Emoji))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to add reaction")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to add reaction")
		return
	}
	writeJSON(w, http.StatusCreated, value.response())
}

func (s *Server) removeIssueReaction(w http.ResponseWriter, r *http.Request) {
	issueValue, _, ok := s.loadIssue(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var request struct {
		Emoji string `json:"emoji"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	if request.Emoji == "" {
		writeError(w, http.StatusBadRequest, "emoji is required")
		return
	}
	if _, err := s.db.ExecContext(r.Context(), `DELETE FROM issue_reactions
		WHERE issue_id = ? AND workspace_id = ? AND actor_type = 'member' AND actor_id = ? AND emoji = ?`,
		issueValue.ID, issueValue.WorkspaceID, currentUserID(r), request.Emoji); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to remove reaction")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type pinnedItem struct {
	ID, WorkspaceID, UserID, ItemType, ItemID, CreatedAt string
	Position                                             float64
}

func scanPinnedItem(scanner interface{ Scan(...any) error }) (pinnedItem, error) {
	var value pinnedItem
	err := scanner.Scan(
		&value.ID,
		&value.WorkspaceID,
		&value.UserID,
		&value.ItemType,
		&value.ItemID,
		&value.Position,
		&value.CreatedAt,
	)
	return value, err
}

func (value pinnedItem) response() map[string]any {
	return map[string]any{
		"id":           value.ID,
		"workspace_id": value.WorkspaceID,
		"user_id":      value.UserID,
		"item_type":    value.ItemType,
		"item_id":      value.ItemID,
		"position":     value.Position,
		"created_at":   value.CreatedAt,
	}
}

func (s *Server) listPins(w http.ResponseWriter, r *http.Request) {
	workspaceValue, ok := s.resolveWorkspace(w, r)
	if !ok {
		return
	}
	rows, err := s.db.QueryContext(r.Context(), `SELECT id, workspace_id, user_id, item_type, item_id, position, created_at
		FROM pinned_items WHERE workspace_id = ? AND user_id = ? ORDER BY position, created_at, id`,
		workspaceValue.ID, currentUserID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list pins")
		return
	}
	defer rows.Close()

	items := make([]map[string]any, 0)
	for rows.Next() {
		value, err := scanPinnedItem(rows)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list pins")
			return
		}
		items = append(items, value.response())
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list pins")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) createPin(w http.ResponseWriter, r *http.Request) {
	workspaceValue, ok := s.resolveWorkspace(w, r)
	if !ok {
		return
	}
	var request struct {
		ItemType string `json:"item_type"`
		ItemID   string `json:"item_id"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	if request.ItemType != "issue" && request.ItemType != "project" {
		writeError(w, http.StatusBadRequest, "item_type must be 'issue' or 'project'")
		return
	}
	request.ItemID = strings.TrimSpace(request.ItemID)
	if request.ItemID == "" {
		writeError(w, http.StatusBadRequest, "item_id is required")
		return
	}
	table := request.ItemType + "s"
	if !s.belongsToWorkspace(r.Context(), table, request.ItemID, workspaceValue.ID) {
		writeError(w, http.StatusNotFound, request.ItemType+" not found")
		return
	}

	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create pin")
		return
	}
	defer tx.Rollback()
	userID := currentUserID(r)
	var existingID string
	err = tx.QueryRowContext(r.Context(), `SELECT id FROM pinned_items
		WHERE workspace_id = ? AND user_id = ? AND item_type = ? AND item_id = ?`,
		workspaceValue.ID, userID, request.ItemType, request.ItemID).Scan(&existingID)
	if err == nil {
		writeError(w, http.StatusConflict, "item already pinned")
		return
	}
	if err != sql.ErrNoRows {
		writeError(w, http.StatusInternalServerError, "failed to create pin")
		return
	}
	var maxPosition float64
	if err := tx.QueryRowContext(r.Context(), `SELECT COALESCE(MAX(position), 0) FROM pinned_items
		WHERE workspace_id = ? AND user_id = ?`, workspaceValue.ID, userID).Scan(&maxPosition); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create pin")
		return
	}
	value := pinnedItem{
		ID:          newID(),
		WorkspaceID: workspaceValue.ID,
		UserID:      userID,
		ItemType:    request.ItemType,
		ItemID:      request.ItemID,
		Position:    maxPosition + 1,
		CreatedAt:   now(),
	}
	if _, err := tx.ExecContext(r.Context(), `INSERT INTO pinned_items(
		id, workspace_id, user_id, item_type, item_id, position, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?)`, value.ID, value.WorkspaceID, value.UserID, value.ItemType, value.ItemID, value.Position, value.CreatedAt); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create pin")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create pin")
		return
	}
	writeJSON(w, http.StatusCreated, value.response())
}

func (s *Server) deletePin(w http.ResponseWriter, r *http.Request) {
	workspaceValue, ok := s.resolveWorkspace(w, r)
	if !ok {
		return
	}
	if _, err := s.db.ExecContext(r.Context(), `DELETE FROM pinned_items
		WHERE workspace_id = ? AND user_id = ? AND item_type = ? AND item_id = ?`,
		workspaceValue.ID, currentUserID(r), chi.URLParam(r, "itemType"), chi.URLParam(r, "itemId")); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete pin")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) reorderPins(w http.ResponseWriter, r *http.Request) {
	workspaceValue, ok := s.resolveWorkspace(w, r)
	if !ok {
		return
	}
	var request struct {
		Items []struct {
			ID       string  `json:"id"`
			Position float64 `json:"position"`
		} `json:"items"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to reorder pins")
		return
	}
	defer tx.Rollback()
	for _, item := range request.Items {
		if _, err := tx.ExecContext(r.Context(), `UPDATE pinned_items SET position = ?
			WHERE id = ? AND workspace_id = ? AND user_id = ?`,
			item.Position, item.ID, workspaceValue.ID, currentUserID(r)); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to reorder pins")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to reorder pins")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
