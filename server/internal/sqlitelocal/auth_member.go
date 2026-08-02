package sqlitelocal

import (
	"errors"
	"net/http"

	"github.com/multica-ai/multica/server/internal/modules/auth/contract"
)

func writeMemberRoleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, contract.ErrInvalidMemberRole):
		writeError(w, http.StatusBadRequest, "role must be owner, admin, or member")
	case errors.Is(err, contract.ErrWorkspaceMembershipHidden):
		writeError(w, http.StatusNotFound, "workspace not found")
	case errors.Is(err, contract.ErrMemberNotFound):
		writeError(w, http.StatusNotFound, "member not found")
	case errors.Is(err, contract.ErrInsufficientWorkspaceRole):
		writeError(w, http.StatusForbidden, "insufficient workspace role")
	case errors.Is(err, contract.ErrOwnerRoleRequiresOwner):
		writeError(w, http.StatusForbidden, "only owners can manage the owner role")
	case errors.Is(err, contract.ErrLastWorkspaceOwner):
		writeError(w, http.StatusBadRequest, "workspace must have at least one owner")
	default:
		writeError(w, http.StatusInternalServerError, "failed to update member")
	}
}

func memberContractResponse(value contract.Member_Member) map[string]any {
	var avatarURL any
	if value.AvatarUrl != nil {
		avatarURL = *value.AvatarUrl
	}
	return map[string]any{
		"id":           value.Id,
		"workspace_id": value.WorkspaceId,
		"user_id":      value.UserId,
		"role":         value.Role,
		"created_at":   value.CreatedAt,
		"name":         value.Name,
		"email":        value.Email,
		"avatar_url":   avatarURL,
	}
}
