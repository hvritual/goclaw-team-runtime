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
	ErrAuthUserNotFound          = errors.New("user not found")
	ErrInvitationNotFound        = errors.New("invitation not found")
	ErrInvitationForbidden       = errors.New("invitation does not belong to you")
	ErrInvalidInvitationEmail    = errors.New("valid email is required")
	ErrInvalidInvitationRole     = errors.New("role must be admin or member")
	ErrInviteeAlreadyMember      = errors.New("user is already a member")
	ErrInvitationAlreadyPending  = errors.New("invitation already pending for this email")
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

// InvitationCreationAuthorizer is the HTTP pre-body authorization seam for
// invitation creation. CreateInvitation repeats this check inside its mutation
// transaction and remains authoritative.
type InvitationCreationAuthorizer interface {
	AuthorizeCreateInvitation(context.Context, Member_CreateInvitationRequest) error
}
