package auth

import (
	"context"

	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

// WorkspaceMemberships adapts the installed membership store to authorization ports.
type WorkspaceMemberships struct {
	queries *db.Queries
}

// NewWorkspaceMemberships creates the legacy Auth-side membership adapter.
func NewWorkspaceMemberships(queries *db.Queries) *WorkspaceMemberships {
	return &WorkspaceMemberships{queries: queries}
}

// IsMember reports whether a user belongs to a workspace.
func (a *WorkspaceMemberships) IsMember(ctx context.Context, userID, workspaceID string) bool {
	userUUID, err := util.ParseUUID(userID)
	if err != nil {
		return false
	}
	workspaceUUID, err := util.ParseUUID(workspaceID)
	if err != nil {
		return false
	}
	_, err = a.queries.GetMemberByUserAndWorkspace(ctx, db.GetMemberByUserAndWorkspaceParams{
		UserID:      userUUID,
		WorkspaceID: workspaceUUID,
	})
	return err == nil
}
