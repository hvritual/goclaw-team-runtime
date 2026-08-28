package application

import (
	"context"
	"strings"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

type EngineeringWorkReferenceReader interface {
	WorkExists(context.Context, string, contract.EngineeringWorkKind, string) (bool, error)
}

type WorkEngineeringLinkUseCase struct {
	work         EngineeringWorkReferenceReader
	memberships  contract.WorkspaceMembershipReader
	engineering  contract.EngineeringLinkGateway
}

func NewWorkEngineeringLinkUseCase(work EngineeringWorkReferenceReader, memberships contract.WorkspaceMembershipReader, engineering contract.EngineeringLinkGateway) (*WorkEngineeringLinkUseCase, error) {
	if work == nil || memberships == nil || engineering == nil {
		return nil, contract.ErrEngineeringWorkLinkUnavailable
	}
	return &WorkEngineeringLinkUseCase{work: work, memberships: memberships, engineering: engineering}, nil
}

func (u *WorkEngineeringLinkUseCase) LinkEngineeringEntity(ctx context.Context, workspaceID string, kind contract.EngineeringWorkKind, workID, entityID string) (contract.EngineeringWorkLink, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	workID = strings.TrimSpace(workID)
	entityID = strings.TrimSpace(entityID)
	if !validEngineeringWorkKind(kind) || workspaceID == "" || workID == "" || entityID == "" {
		return contract.EngineeringWorkLink{}, contract.ErrEngineeringWorkLinkInvalid
	}
	if err := u.authorize(ctx, workspaceID, true); err != nil {
		return contract.EngineeringWorkLink{}, err
	}
	exists, err := u.work.WorkExists(ctx, workspaceID, kind, workID)
	if err != nil {
		return contract.EngineeringWorkLink{}, contract.ErrEngineeringWorkLinkUnavailable
	}
	if !exists {
		return contract.EngineeringWorkLink{}, contract.ErrEngineeringWorkNotFound
	}
	return u.engineering.PutEngineeringWorkLink(ctx, workspaceID, kind, workID, entityID)
}

func (u *WorkEngineeringLinkUseCase) ListEngineeringLinks(ctx context.Context, workspaceID string, kind contract.EngineeringWorkKind, workID string) ([]contract.EngineeringWorkLink, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	workID = strings.TrimSpace(workID)
	if !validEngineeringWorkKind(kind) || workspaceID == "" || workID == "" {
		return nil, contract.ErrEngineeringWorkLinkInvalid
	}
	if err := u.authorize(ctx, workspaceID, false); err != nil {
		return nil, err
	}
	// Historical links remain readable after the Workspace source is archived
	// or deleted; reads therefore do not revalidate source existence.
	return u.engineering.ListEngineeringWorkLinks(ctx, workspaceID, kind, workID)
}

func (u *WorkEngineeringLinkUseCase) UnlinkEngineeringEntity(ctx context.Context, workspaceID string, kind contract.EngineeringWorkKind, workID, entityID string) error {
	workspaceID = strings.TrimSpace(workspaceID)
	workID = strings.TrimSpace(workID)
	entityID = strings.TrimSpace(entityID)
	if !validEngineeringWorkKind(kind) || workspaceID == "" || workID == "" || entityID == "" {
		return contract.ErrEngineeringWorkLinkInvalid
	}
	if err := u.authorize(ctx, workspaceID, true); err != nil {
		return err
	}
	// Explicit unlink remains possible after source removal. There is no
	// cascade from Workspace lifecycle events into Engineering Thread history.
	return u.engineering.DeleteEngineeringWorkLink(ctx, workspaceID, kind, workID, entityID)
}

func (u *WorkEngineeringLinkUseCase) authorize(ctx context.Context, workspaceID string, write bool) error {
	actor, ok := contract.WorkspaceActorFromContext(ctx)
	if !ok || actor.Type != "member" || strings.TrimSpace(actor.ID) == "" {
		return contract.ErrWorkspaceActorRequired
	}
	membership, found, err := u.memberships.FindForUserAndWorkspace(ctx, actor.ID, workspaceID)
	if err != nil {
		return contract.ErrEngineeringWorkLinkUnavailable
	}
	if !found {
		membership, found, err = u.memberships.FindByMemberAndWorkspace(ctx, actor.ID, workspaceID)
		if err != nil {
			return contract.ErrEngineeringWorkLinkUnavailable
		}
	}
	if !found {
		return contract.ErrActorOutsideWorkspace
	}
	if !write {
		return nil
	}
	if membership.Role != "owner" && membership.Role != "admin" {
		return contract.ErrWorkspacePermissionDenied
	}
	return nil
}

func validEngineeringWorkKind(kind contract.EngineeringWorkKind) bool {
	switch kind {
	case contract.EngineeringWorkProject, contract.EngineeringWorkRequirement, contract.EngineeringWorkTask:
		return true
	default:
		return false
	}
}
