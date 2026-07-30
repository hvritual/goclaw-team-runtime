package handler

import (
	"net/http"

	"github.com/multica-ai/multica/server/internal/workspacepermissions"
)

// GetWorkspacePermissions returns the fixed role capability catalog. The
// handler repeats the owner/admin check as defense in depth so the catalog is
// not exposed if this method is mounted without the expected router middleware.
func (h *Handler) GetWorkspacePermissions(w http.ResponseWriter, r *http.Request) {
	workspaceID := workspaceIDFromURL(r, "id")
	member, ok := h.workspaceMember(w, r, workspaceID)
	if !ok {
		return
	}
	if !roleAllowed(member.Role, "owner", "admin") {
		writeError(w, http.StatusForbidden, "insufficient permissions")
		return
	}
	writeJSON(w, http.StatusOK, workspacepermissions.FixedCatalog())
}
