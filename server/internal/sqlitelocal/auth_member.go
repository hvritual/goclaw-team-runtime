package sqlitelocal

import (
	"errors"
	"net/http"

	"github.com/multica-ai/multica/server/internal/modules/auth/contract"
)

func writeMemberError(w http.ResponseWriter, err error, fallback string) {
	switch {
	case errors.Is(err, contract.ErrInvalidMemberRole):
		writeError(w, http.StatusBadRequest, "role must be owner, admin, or member")
	case errors.Is(err, contract.ErrWorkspaceMembershipHidden):
		writeError(w, http.StatusNotFound, "workspace not found")
	case errors.Is(err, contract.ErrMemberNotFound):
		writeError(w, http.StatusNotFound, "member not found")
	case errors.Is(err, contract.ErrInvitationNotFound):
		writeError(w, http.StatusNotFound, "invitation not found")
	case errors.Is(err, contract.ErrInsufficientWorkspaceRole):
		writeError(w, http.StatusForbidden, "insufficient workspace role")
	case errors.Is(err, contract.ErrOwnerRoleRequiresOwner):
		writeError(w, http.StatusForbidden, "only owners can manage the owner role")
	case errors.Is(err, contract.ErrOwnerRemovalRequiresOwner):
		writeError(w, http.StatusForbidden, "only owners can remove another owner")
	case errors.Is(err, contract.ErrLastWorkspaceOwner):
		writeError(w, http.StatusBadRequest, "workspace must have at least one owner")
	default:
		writeError(w, http.StatusInternalServerError, fallback)
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

func memberContractResponses(values []contract.Member_Member) []map[string]any {
	result := make([]map[string]any, 0, len(values))
	for _, value := range values {
		result = append(result, memberContractResponse(value))
	}
	return result
}

func invitationContractResponse(value contract.Member_Invitation) map[string]any {
	var inviteeUserID any
	if value.InviteeUserId != nil {
		inviteeUserID = *value.InviteeUserId
	}
	return map[string]any{
		"id": value.Id, "workspace_id": value.WorkspaceId, "inviter_id": value.InviterId,
		"invitee_email": value.InviteeEmail, "invitee_user_id": inviteeUserID,
		"role": value.Role, "status": value.Status, "created_at": value.CreatedAt,
		"updated_at": value.UpdatedAt, "expires_at": value.ExpiresAt,
		"workspace_name": value.WorkspaceName, "inviter_name": value.InviterName,
		"inviter_email": value.InviterEmail,
	}
}

func invitationContractResponses(values []contract.Member_Invitation) []map[string]any {
	result := make([]map[string]any, 0, len(values))
	for _, value := range values {
		result = append(result, invitationContractResponse(value))
	}
	return result
}
