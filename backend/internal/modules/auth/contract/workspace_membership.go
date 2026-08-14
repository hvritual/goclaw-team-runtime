package contract

import "context"

// WorkspaceMembership is the Auth-owned projection used by consumers without
// exposing Auth persistence.
type WorkspaceMembership struct {
	MemberID    string
	UserID      string
	WorkspaceID string
	Role        string
}

type WorkspaceMembershipReader interface {
	ListForUser(context.Context, string) ([]WorkspaceMembership, error)
	FindForUserAndWorkspace(context.Context, string, string) (WorkspaceMembership, bool, error)
	FindByMemberAndWorkspace(context.Context, string, string) (WorkspaceMembership, bool, error)
}
