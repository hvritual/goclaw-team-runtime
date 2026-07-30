package handler

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

var (
	errLastWorkspaceOwner              = errors.New("workspace must have at least one owner")
	errWorkspaceMemberRemovalForbidden = errors.New("workspace member removal forbidden")
	errWorkspaceMemberRemovalNotFound  = errors.New("workspace member removal target not found")
)

type revocationResult struct{}

func (h *Handler) revokeAndRemoveMember(
	ctx context.Context,
	workspaceID, userID, memberID, removedBy pgtype.UUID,
) (revocationResult, error) {
	tx, err := h.TxStarter.Begin(ctx)
	if err != nil {
		return revocationResult{}, err
	}
	defer tx.Rollback(ctx)

	qtx := h.Queries.WithTx(tx)
	if _, err := qtx.LockWorkspaceForMemberMutation(ctx, workspaceID); err != nil {
		return revocationResult{}, err
	}

	target, err := qtx.GetMember(ctx, memberID)
	if err != nil || target.WorkspaceID != workspaceID || target.UserID != userID {
		return revocationResult{}, errWorkspaceMemberRemovalNotFound
	}
	if removedBy != target.UserID {
		requester, err := qtx.GetMemberByUserAndWorkspace(ctx, db.GetMemberByUserAndWorkspaceParams{
			UserID:      removedBy,
			WorkspaceID: workspaceID,
		})
		if err != nil || (requester.Role != "owner" && requester.Role != "admin") {
			return revocationResult{}, errWorkspaceMemberRemovalForbidden
		}
		if target.Role == "owner" && requester.Role != "owner" {
			return revocationResult{}, errWorkspaceMemberRemovalForbidden
		}
	}
	if target.Role == "owner" {
		members, err := qtx.ListMembers(ctx, workspaceID)
		if err != nil {
			return revocationResult{}, err
		}
		if countOwners(members) <= 1 {
			return revocationResult{}, errLastWorkspaceOwner
		}
	}

	if err := qtx.DeleteMember(ctx, memberID); err != nil {
		return revocationResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return revocationResult{}, err
	}
	return revocationResult{}, nil
}

func logRevocation(revocationResult, string, string, ...any) {}

func (h *Handler) publishRevocation(
	context.Context,
	revocationResult,
	string,
	string,
	string,
) {
}
