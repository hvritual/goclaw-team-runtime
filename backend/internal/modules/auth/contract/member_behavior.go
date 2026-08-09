package contract

import (
	"context"
	"errors"
	"strings"
)

var (
	ErrInvalidMember                  = errors.New("invalid member")
	ErrMemberActorRequired            = errors.New("member actor is required")
	ErrInvalidMemberRole              = errors.New("role must be owner, admin, or member")
	ErrWorkspaceMembershipHidden      = errors.New("workspace not found")
	ErrWorkspaceMembershipInitialized = errors.New("workspace membership is already initialized")
	ErrMemberNotFound                 = errors.New("member not found")
	ErrAuthUserNotFound               = errors.New("auth user not found")
	ErrInsufficientWorkspaceRole      = errors.New("insufficient workspace role")
	ErrOwnerRoleRequiresOwner         = errors.New("only owners can manage the owner role")
	ErrLastWorkspaceOwner             = errors.New("workspace must have at least one owner")
)

type memberActorContextKey struct{}

// WithMemberActor attaches an authenticated Auth user identity to a local
// MemberService call. Transport authentication is responsible for setting it.
func WithMemberActor(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, memberActorContextKey{}, strings.TrimSpace(userID))
}

// MemberActor returns the authenticated Auth user identity for a MemberService
// call without exposing transport-specific principal types.
func MemberActor(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(memberActorContextKey{}).(string)
	userID = strings.TrimSpace(userID)
	return userID, ok && userID != ""
}
