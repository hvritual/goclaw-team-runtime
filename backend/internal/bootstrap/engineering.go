package bootstrap

import (
	"context"
	"errors"

	engineeringcontract "github.com/hvritual/workspace/internal/modules/engineering/contract"
	workspacecontract "github.com/hvritual/workspace/internal/modules/workspace/contract"
)

func (a authMembershipAdapter) ResolveWorkspaceRole(ctx context.Context, userID, workspaceID string) (string, bool, error) {
	membership, found, err := a.FindForUserAndWorkspace(ctx, userID, workspaceID)
	if err != nil || !found {
		return "", found, err
	}
	return membership.Role, true, nil
}

type workspaceEngineeringLinkAdapter struct {
	provider engineeringcontract.WorkLinkProvider
}

var _ workspacecontract.EngineeringLinkGateway = workspaceEngineeringLinkAdapter{}

func (a workspaceEngineeringLinkAdapter) PutEngineeringWorkLink(ctx context.Context, workspaceID string, kind workspacecontract.EngineeringWorkKind, workID, entityID string) (workspacecontract.EngineeringWorkLink, error) {
	if a.provider == nil {
		return workspacecontract.EngineeringWorkLink{}, workspacecontract.ErrEngineeringWorkLinkUnavailable
	}
	engineeringKind, err := engineeringWorkKind(kind)
	if err != nil {
		return workspacecontract.EngineeringWorkLink{}, err
	}
	value, err := a.provider.PutWorkLink(ctx, engineeringcontract.PutWorkLinkRequest{
		WorkspaceID: workspaceID,
		WorkKind:    engineeringKind,
		WorkID:      workID,
		EntityID:    entityID,
	})
	if err != nil {
		return workspacecontract.EngineeringWorkLink{}, mapEngineeringPutError(err)
	}
	return projectEngineeringWorkLink(value, kind), nil
}

func (a workspaceEngineeringLinkAdapter) ListEngineeringWorkLinks(ctx context.Context, workspaceID string, kind workspacecontract.EngineeringWorkKind, workID string) ([]workspacecontract.EngineeringWorkLink, error) {
	if a.provider == nil {
		return nil, workspacecontract.ErrEngineeringWorkLinkUnavailable
	}
	engineeringKind, err := engineeringWorkKind(kind)
	if err != nil {
		return nil, err
	}
	values, err := a.provider.ListWorkLinks(ctx, engineeringcontract.ListWorkLinksRequest{
		WorkspaceID: workspaceID,
		WorkKind:    engineeringKind,
		WorkID:      workID,
	})
	if err != nil {
		return nil, mapEngineeringReadError(err)
	}
	result := make([]workspacecontract.EngineeringWorkLink, len(values))
	for index, value := range values {
		result[index] = projectEngineeringWorkLink(value, kind)
	}
	return result, nil
}

func (a workspaceEngineeringLinkAdapter) DeleteEngineeringWorkLink(ctx context.Context, workspaceID string, kind workspacecontract.EngineeringWorkKind, workID, entityID string) error {
	if a.provider == nil {
		return workspacecontract.ErrEngineeringWorkLinkUnavailable
	}
	engineeringKind, err := engineeringWorkKind(kind)
	if err != nil {
		return err
	}
	err = a.provider.DeleteWorkLink(ctx, engineeringcontract.DeleteWorkLinkRequest{
		WorkspaceID: workspaceID,
		WorkKind:    engineeringKind,
		WorkID:      workID,
		EntityID:    entityID,
	})
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, engineeringcontract.ErrInvalidArgument):
		return workspacecontract.ErrEngineeringWorkLinkInvalid
	case errors.Is(err, engineeringcontract.ErrNotFound):
		return workspacecontract.ErrEngineeringWorkLinkNotFound
	case errors.Is(err, engineeringcontract.ErrUnavailable), errors.Is(err, engineeringcontract.ErrConflict):
		return workspacecontract.ErrEngineeringWorkLinkUnavailable
	default:
		return workspacecontract.ErrEngineeringWorkLinkUnavailable
	}
}

func engineeringWorkKind(kind workspacecontract.EngineeringWorkKind) (engineeringcontract.WorkLinkKind, error) {
	switch kind {
	case workspacecontract.EngineeringWorkProject:
		return engineeringcontract.WorkLinkProject, nil
	case workspacecontract.EngineeringWorkRequirement:
		return engineeringcontract.WorkLinkRequirement, nil
	case workspacecontract.EngineeringWorkTask:
		return engineeringcontract.WorkLinkTask, nil
	default:
		return "", workspacecontract.ErrEngineeringWorkLinkInvalid
	}
}

func projectEngineeringWorkLink(value engineeringcontract.WorkLink, kind workspacecontract.EngineeringWorkKind) workspacecontract.EngineeringWorkLink {
	return workspacecontract.EngineeringWorkLink{
		ID:          value.ID,
		WorkspaceID: value.WorkspaceID,
		WorkKind:    kind,
		WorkID:      value.WorkID,
		EntityID:    value.EntityID,
		Relation:    value.Relation,
		Authority:   value.Authority,
		Source:      value.Provenance.SourceType,
		Locator:     value.Provenance.Locator,
	}
}

func mapEngineeringPutError(err error) error {
	switch {
	case errors.Is(err, engineeringcontract.ErrInvalidArgument):
		return workspacecontract.ErrEngineeringWorkLinkInvalid
	case errors.Is(err, engineeringcontract.ErrNotFound):
		return workspacecontract.ErrEngineeringEntityReferenceMissing
	case errors.Is(err, engineeringcontract.ErrConflict):
		return workspacecontract.ErrEngineeringEntityArchived
	case errors.Is(err, engineeringcontract.ErrUnavailable):
		return workspacecontract.ErrEngineeringWorkLinkUnavailable
	default:
		return workspacecontract.ErrEngineeringWorkLinkUnavailable
	}
}

func mapEngineeringReadError(err error) error {
	switch {
	case errors.Is(err, engineeringcontract.ErrInvalidArgument):
		return workspacecontract.ErrEngineeringWorkLinkInvalid
	case errors.Is(err, engineeringcontract.ErrUnavailable):
		return workspacecontract.ErrEngineeringWorkLinkUnavailable
	default:
		return workspacecontract.ErrEngineeringWorkLinkUnavailable
	}
}
