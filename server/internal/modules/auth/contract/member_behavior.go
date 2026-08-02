package contract

import (
	"context"
	"errors"
)

var (
	ErrMemberActorRequired       = errors.New("member actor is required")
	ErrInvalidMemberRole         = errors.New("role must be owner, admin, or member")
	ErrWorkspaceMembershipHidden = errors.New("workspace not found")
	ErrMemberNotFound            = errors.New("member not found")
	ErrInvitationNotFound        = errors.New("invitation not found")
	ErrInsufficientWorkspaceRole = errors.New("insufficient workspace role")
	ErrOwnerRoleRequiresOwner    = errors.New("only owners can manage the owner role")
	ErrOwnerRemovalRequiresOwner = errors.New("only owners can remove another owner")
	ErrLastWorkspaceOwner        = errors.New("workspace must have at least one owner")
)

type memberActorContextKey struct{}

// WithMemberActor attaches the authenticated user identity to a MemberService call.
func WithMemberActor(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, memberActorContextKey{}, userID)
}

// MemberActor returns the authenticated user identity for a MemberService call.
func MemberActor(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(memberActorContextKey{}).(string)
	return userID, ok && userID != ""
}
