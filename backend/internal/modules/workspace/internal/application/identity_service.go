package application

import (
	"context"
	"errors"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	workspaceDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/workspace"
)

var ErrWorkspaceNotFound = errors.New("workspace not found")

type WorkspaceIdentityRepository interface {
	FindByID(context.Context, string) (workspaceDomain.Workspace, error)
}

type WorkspaceIdentityService struct {
	repository WorkspaceIdentityRepository
}

func NewWorkspaceIdentityService(repository WorkspaceIdentityRepository) *WorkspaceIdentityService {
	return &WorkspaceIdentityService{repository: repository}
}

func (s *WorkspaceIdentityService) FindIdentity(ctx context.Context, workspaceID string) (contract.WorkspaceIdentity, error) {
	value, err := s.repository.FindByID(ctx, workspaceID)
	if errors.Is(err, ErrWorkspaceNotFound) {
		return contract.WorkspaceIdentity{}, contract.ErrWorkspaceNotFound
	}
	if err != nil {
		return contract.WorkspaceIdentity{}, err
	}
	return contract.WorkspaceIdentity{ID: value.ID(), Name: value.Name()}, nil
}
