package application

import (
	"context"
	"errors"
	"strings"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

type WorkspaceSelectionRepository interface {
	ListByIDs(context.Context, []string) ([]contract.WorkspaceSelection, error)
	FindIDBySlugAndIDs(context.Context, string, []string) (string, error)
}

type WorkspaceSelectionUseCase struct {
	memberships contract.WorkspaceMembershipReader
	repository  WorkspaceSelectionRepository
}

func NewWorkspaceSelectionUseCase(memberships contract.WorkspaceMembershipReader, repository WorkspaceSelectionRepository) (*WorkspaceSelectionUseCase, error) {
	if memberships == nil || repository == nil {
		return nil, errors.New("workspace selection dependencies are required")
	}
	return &WorkspaceSelectionUseCase{memberships: memberships, repository: repository}, nil
}

func (s *WorkspaceSelectionUseCase) List(ctx context.Context, userID string) ([]contract.WorkspaceSelection, error) {
	ids, err := s.authorizedIDs(ctx, userID)
	if err != nil || len(ids) == 0 {
		return []contract.WorkspaceSelection{}, err
	}
	return s.repository.ListByIDs(ctx, ids)
}

func (s *WorkspaceSelectionUseCase) ResolveSlug(ctx context.Context, userID, slug string) (string, error) {
	ids, err := s.authorizedIDs(ctx, userID)
	if err != nil {
		return "", err
	}
	if len(ids) == 0 || strings.TrimSpace(slug) == "" {
		return "", contract.ErrWorkspaceNotFound
	}
	return s.repository.FindIDBySlugAndIDs(ctx, strings.TrimSpace(slug), ids)
}

func (s *WorkspaceSelectionUseCase) MembershipForID(ctx context.Context, userID, workspaceID string) (contract.WorkspaceMembership, error) {
	membership, ok, err := s.memberships.FindForUserAndWorkspace(ctx, strings.TrimSpace(userID), strings.TrimSpace(workspaceID))
	if err != nil {
		return contract.WorkspaceMembership{}, err
	}
	if !ok {
		return contract.WorkspaceMembership{}, contract.ErrWorkspaceNotFound
	}
	return membership, nil
}

func (s *WorkspaceSelectionUseCase) authorizedIDs(ctx context.Context, userID string) ([]string, error) {
	memberships, err := s.memberships.ListForUser(ctx, strings.TrimSpace(userID))
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(memberships))
	for _, membership := range memberships {
		if id := strings.TrimSpace(membership.WorkspaceID); id != "" {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

var _ contract.WorkspaceSelectionService = (*WorkspaceSelectionUseCase)(nil)
