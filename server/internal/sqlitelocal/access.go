package sqlitelocal

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/multica-ai/multica/server/internal/workspacepermissions"
)

type sqlRowQuerier interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func workspaceRole(ctx context.Context, q sqlRowQuerier, workspaceID, userID string) (workspacepermissions.RoleKey, error) {
	var rawRole string
	err := q.QueryRowContext(ctx, `SELECT role FROM members WHERE workspace_id = ? AND user_id = ?`,
		workspaceID, userID).Scan(&rawRole)
	return workspacepermissions.RoleKey(rawRole), err
}

func workspaceMemberRole(ctx context.Context, q sqlRowQuerier, workspaceID, memberID string) (workspacepermissions.RoleKey, error) {
	var rawRole string
	err := q.QueryRowContext(ctx, `SELECT role FROM members WHERE workspace_id = ? AND id = ?`,
		workspaceID, memberID).Scan(&rawRole)
	return workspacepermissions.RoleKey(rawRole), err
}

func countWorkspaceOwners(ctx context.Context, q sqlRowQuerier, workspaceID string) (int, error) {
	var count int
	err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM members WHERE workspace_id = ? AND role = 'owner'`,
		workspaceID).Scan(&count)
	return count, err
}

func requireAnotherWorkspaceOwner(w http.ResponseWriter, r *http.Request, q sqlRowQuerier, workspaceID string) bool {
	ownerCount, err := countWorkspaceOwners(r.Context(), q, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to verify workspace owners")
		return false
	}
	if ownerCount <= 1 {
		writeError(w, http.StatusBadRequest, "workspace must have at least one owner")
		return false
	}
	return true
}

func (s *Server) getWorkspacePermissions(w http.ResponseWriter, r *http.Request) {
	workspaceValue, ok := s.authorizedWorkspace(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	if !s.requireWorkspaceRole(
		w,
		r,
		workspaceValue.ID,
		workspacepermissions.RoleOwner,
		workspacepermissions.RoleAdmin,
	) {
		return
	}
	writeJSON(w, http.StatusOK, workspacepermissions.FixedCatalog())
}

func (s *Server) requireWorkspaceRole(
	w http.ResponseWriter,
	r *http.Request,
	workspaceID string,
	allowed ...workspacepermissions.RoleKey,
) bool {
	role, err := workspaceRole(r.Context(), s.db, workspaceID, currentUserID(r))
	if err != nil {
		writeError(w, http.StatusForbidden, "workspace access denied")
		return false
	}
	for _, candidate := range allowed {
		if role == candidate {
			return true
		}
	}
	writeError(w, http.StatusForbidden, "insufficient workspace role")
	return false
}

func (s *Server) requireSkillManager(w http.ResponseWriter, r *http.Request, workspaceID string, createdBy sql.NullString) bool {
	role, err := workspaceRole(r.Context(), s.db, workspaceID, currentUserID(r))
	if err != nil {
		writeError(w, http.StatusForbidden, "workspace access denied")
		return false
	}
	if role == workspacepermissions.RoleOwner ||
		role == workspacepermissions.RoleAdmin ||
		(createdBy.Valid && createdBy.String == currentUserID(r)) {
		return true
	}
	writeError(w, http.StatusForbidden, "only workspace administrators or the skill creator can manage this skill")
	return false
}
