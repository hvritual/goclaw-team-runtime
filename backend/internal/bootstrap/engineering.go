package bootstrap

import "context"

func (a authMembershipAdapter) ResolveWorkspaceRole(ctx context.Context, userID, workspaceID string) (string, bool, error) {
	membership, found, err := a.FindForUserAndWorkspace(ctx, userID, workspaceID)
	if err != nil || !found {
		return "", found, err
	}
	return membership.Role, true, nil
}
