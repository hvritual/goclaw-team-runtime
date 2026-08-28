package application

import (
	"context"
	"errors"
	"strings"

	"github.com/hvritual/workspace/internal/modules/engineering/contract"
	"github.com/hvritual/workspace/internal/modules/engineering/internal/domain"
)

func (s *Service) AcceptChange(ctx context.Context, actor contract.Actor, workspaceID, id string) (contract.Change, error) {
	if err := s.authorize(ctx, actor, workspaceID, true); err != nil {
		return contract.Change{}, err
	}
	workspaceID = strings.TrimSpace(workspaceID)
	id = strings.TrimSpace(id)
	if id == "" {
		return contract.Change{}, contract.ErrInvalidArgument
	}
	current, err := s.repository.GetChange(ctx, workspaceID, id)
	if err != nil {
		return contract.Change{}, repositoryError(err)
	}
	accepted, err := current.Accept(s.now().UTC())
	if err != nil {
		return contract.Change{}, contract.ErrConflict
	}
	if err := s.repository.PutChange(ctx, accepted); err != nil {
		return contract.Change{}, repositoryError(err)
	}
	return toChange(accepted), nil
}

func (s *Service) FreezeContextPack(ctx context.Context, actor contract.Actor, workspaceID string, request contract.FreezeContextPackRequest) (contract.ContextPack, error) {
	if err := s.authorize(ctx, actor, workspaceID, true); err != nil {
		return contract.ContextPack{}, err
	}
	workspaceID = strings.TrimSpace(workspaceID)
	for _, entityID := range request.TargetEntityIDs {
		if _, err := s.repository.GetEntity(ctx, workspaceID, strings.TrimSpace(entityID)); err != nil {
			return contract.ContextPack{}, repositoryError(err)
		}
	}
	workItem, err := domain.NewNodeRef(domain.NodeKind(request.WorkItem.Kind), request.WorkItem.ID)
	if err != nil {
		return contract.ContextPack{}, invalid(err)
	}
	references := make([]domain.ContextReference, len(request.References))
	for index, reference := range request.References {
		value, err := domain.NewContextReference(domain.ContextKind(reference.Kind), reference.ID, reference.Revision, reference.Checksum)
		if err != nil {
			return contract.ContextPack{}, invalid(err)
		}
		references[index] = value
	}

	createdAt := s.now().UTC()
	existing, getErr := s.repository.GetContextPack(ctx, workspaceID, strings.TrimSpace(request.ID))
	if getErr == nil {
		createdAt = existing.CreatedAt()
	} else if !errors.Is(getErr, domain.ErrNotFound) {
		return contract.ContextPack{}, repositoryError(getErr)
	}
	pack, err := domain.NewContextPack(
		request.ID,
		workspaceID,
		workItem,
		request.WorkItemRevision,
		request.TargetEntityIDs,
		references,
		request.PolicyVersion,
		createdAt,
	)
	if err != nil {
		return contract.ContextPack{}, invalid(err)
	}
	if getErr == nil {
		if existing.Checksum() != pack.Checksum() {
			return contract.ContextPack{}, contract.ErrConflict
		}
		return toContextPack(existing), nil
	}
	if err := s.repository.PutContextPack(ctx, pack); err != nil {
		return contract.ContextPack{}, repositoryError(err)
	}
	return toContextPack(pack), nil
}
