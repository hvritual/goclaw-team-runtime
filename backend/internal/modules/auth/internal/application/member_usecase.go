package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hvritual/workspace/internal/modules/auth/contract"
	memberDomain "github.com/hvritual/workspace/internal/modules/auth/internal/domain/member"
)

var (
	ErrMemberRecordNotFound   = errors.New("member record not found")
	ErrMemberRecordConflict   = errors.New("member record conflicts with existing membership")
	ErrAuthUserRecordNotFound = errors.New("auth user record not found")
	ErrWorkspaceRootNotFound  = errors.New("workspace membership root not found")
)

type MemberRepository interface {
	FindWorkspaceRoot(context.Context, string) (memberDomain.WorkspaceRoot, error)
	CreateWorkspaceRoot(context.Context, memberDomain.WorkspaceRoot) error
	FindUserByID(context.Context, string) (memberDomain.User, error)
	Create(context.Context, memberDomain.Member) error
	FindByUserAndWorkspace(context.Context, string, string) (memberDomain.Member, error)
	FindByIDAndWorkspace(context.Context, string, string) (memberDomain.Member, error)
	ListByWorkspace(context.Context, string) ([]memberDomain.Member, error)
	CountOwners(context.Context, string) (int, error)
	UpdateRole(context.Context, memberDomain.Member) error
}

type MemberUnitOfWork interface {
	WithinTransaction(context.Context, func(MemberRepository) error) error
}

type MemberIDGenerator func(context.Context) (string, error)
type MemberClock func() time.Time

type MemberUseCase struct {
	unitOfWork MemberUnitOfWork
	newID      MemberIDGenerator
	now        MemberClock
}

func NewMemberUseCase(unitOfWork MemberUnitOfWork, newID MemberIDGenerator, now MemberClock) (*MemberUseCase, error) {
	if unitOfWork == nil {
		return nil, errors.New("member unit of work is required")
	}
	if newID == nil {
		return nil, errors.New("member id generator is required")
	}
	if now == nil {
		return nil, errors.New("member clock is required")
	}
	return &MemberUseCase{unitOfWork: unitOfWork, newID: newID, now: now}, nil
}

func (s *MemberUseCase) ProvisionWorkspaceOwner(ctx context.Context, request contract.ProvisionWorkspaceOwnerRequest) (contract.ProvisionWorkspaceOwnerResponse, error) {
	workspaceID := strings.TrimSpace(request.WorkspaceId)
	userID := strings.TrimSpace(request.UserId)
	if workspaceID == "" || userID == "" {
		return contract.ProvisionWorkspaceOwnerResponse{}, fmt.Errorf("%w: workspace id and user id are required", contract.ErrInvalidMember)
	}
	var provisioned memberDomain.Member
	created := false
	err := s.unitOfWork.WithinTransaction(ctx, func(repository MemberRepository) error {
		existingRoot, findErr := repository.FindWorkspaceRoot(ctx, workspaceID)
		if findErr == nil {
			if existingRoot.UserID() != userID {
				return contract.ErrWorkspaceMembershipInitialized
			}
			existing, findErr := repository.FindByIDAndWorkspace(ctx, existingRoot.MemberID(), workspaceID)
			if errors.Is(findErr, ErrMemberRecordNotFound) {
				return contract.ErrWorkspaceMembershipInitialized
			}
			if findErr != nil {
				return findErr
			}
			if existing.UserID() != userID || existing.Role() != memberDomain.RoleOwner {
				return contract.ErrWorkspaceMembershipInitialized
			}
			provisioned = existing
			return nil
		}
		if !errors.Is(findErr, ErrWorkspaceRootNotFound) {
			return findErr
		}
		user, findErr := repository.FindUserByID(ctx, userID)
		if errors.Is(findErr, ErrAuthUserRecordNotFound) {
			return contract.ErrAuthUserNotFound
		}
		if findErr != nil {
			return findErr
		}
		memberID, generateErr := s.newID(ctx)
		if generateErr != nil {
			return fmt.Errorf("generate member id: %w", generateErr)
		}
		now := s.now().UTC()
		owner, createErr := memberDomain.NewInitialOwner(memberID, workspaceID, user, now)
		if createErr != nil {
			return fmt.Errorf("build initial workspace owner: %w", createErr)
		}
		root, createErr := memberDomain.NewWorkspaceRoot(workspaceID, userID, memberID, now)
		if createErr != nil {
			return fmt.Errorf("build workspace membership root: %w", createErr)
		}
		if createErr := repository.CreateWorkspaceRoot(ctx, root); createErr != nil {
			return createErr
		}
		if createErr := repository.Create(ctx, owner); createErr != nil {
			if errors.Is(createErr, ErrMemberRecordConflict) {
				return contract.ErrWorkspaceMembershipInitialized
			}
			return createErr
		}
		provisioned = owner
		created = true
		return nil
	})
	if err != nil {
		return contract.ProvisionWorkspaceOwnerResponse{}, fmt.Errorf("provision workspace owner: %w", err)
	}
	result := memberToContract(provisioned)
	return contract.ProvisionWorkspaceOwnerResponse{Member: &result, Created: created}, nil
}

func (s *MemberUseCase) ListMembers(ctx context.Context, request contract.ListMembersRequest) (contract.ListMembersResponse, error) {
	workspaceID := strings.TrimSpace(request.WorkspaceId)
	if workspaceID == "" {
		return contract.ListMembersResponse{}, fmt.Errorf("%w: workspace id is required", contract.ErrInvalidMember)
	}
	actorUserID, ok := contract.MemberActor(ctx)
	if !ok {
		return contract.ListMembersResponse{}, contract.ErrMemberActorRequired
	}
	var values []memberDomain.Member
	err := s.unitOfWork.WithinTransaction(ctx, func(repository MemberRepository) error {
		if _, findErr := repository.FindByUserAndWorkspace(ctx, actorUserID, workspaceID); errors.Is(findErr, ErrMemberRecordNotFound) {
			return contract.ErrWorkspaceMembershipHidden
		} else if findErr != nil {
			return findErr
		}
		var listErr error
		values, listErr = repository.ListByWorkspace(ctx, workspaceID)
		return listErr
	})
	if err != nil {
		return contract.ListMembersResponse{}, fmt.Errorf("list members: %w", err)
	}
	members := make([]contract.Member, len(values))
	for index, value := range values {
		members[index] = memberToContract(value)
	}
	return contract.ListMembersResponse{Members: members}, nil
}

func (s *MemberUseCase) UpdateMemberRole(ctx context.Context, request contract.UpdateMemberRoleRequest) (contract.UpdateMemberRoleResponse, error) {
	workspaceID := strings.TrimSpace(request.WorkspaceId)
	memberID := strings.TrimSpace(request.MemberId)
	nextRole, err := memberDomain.ParseRole(request.Role)
	if workspaceID == "" || memberID == "" || err != nil {
		return contract.UpdateMemberRoleResponse{}, fmt.Errorf("%w: workspace id, member id, and role must be valid", contract.ErrInvalidMemberRole)
	}
	actorUserID, ok := contract.MemberActor(ctx)
	if !ok {
		return contract.UpdateMemberRoleResponse{}, contract.ErrMemberActorRequired
	}
	var updated memberDomain.Member
	err = s.unitOfWork.WithinTransaction(ctx, func(repository MemberRepository) error {
		requester, findErr := repository.FindByUserAndWorkspace(ctx, actorUserID, workspaceID)
		if errors.Is(findErr, ErrMemberRecordNotFound) {
			return contract.ErrWorkspaceMembershipHidden
		}
		if findErr != nil {
			return findErr
		}
		if policyErr := memberDomain.ValidateManager(requester.Role()); policyErr != nil {
			return mapMemberPolicyError(policyErr)
		}
		target, findErr := repository.FindByIDAndWorkspace(ctx, memberID, workspaceID)
		if errors.Is(findErr, ErrMemberRecordNotFound) {
			return contract.ErrMemberNotFound
		}
		if findErr != nil {
			return findErr
		}
		ownerCount := 0
		if target.Role() == memberDomain.RoleOwner && nextRole != memberDomain.RoleOwner {
			ownerCount, findErr = repository.CountOwners(ctx, workspaceID)
			if findErr != nil {
				return findErr
			}
		}
		updated, findErr = target.ChangeRole(requester.Role(), nextRole, ownerCount)
		if findErr != nil {
			return mapMemberPolicyError(findErr)
		}
		return repository.UpdateRole(ctx, updated)
	})
	if err != nil {
		return contract.UpdateMemberRoleResponse{}, fmt.Errorf("update member role: %w", err)
	}
	result := memberToContract(updated)
	return contract.UpdateMemberRoleResponse{Member: &result}, nil
}

func mapMemberPolicyError(err error) error {
	switch {
	case errors.Is(err, memberDomain.ErrInsufficientWorkspaceRole):
		return contract.ErrInsufficientWorkspaceRole
	case errors.Is(err, memberDomain.ErrOwnerRoleRequiresOwner):
		return contract.ErrOwnerRoleRequiresOwner
	case errors.Is(err, memberDomain.ErrLastOwner):
		return contract.ErrLastWorkspaceOwner
	default:
		return err
	}
}

func memberToContract(value memberDomain.Member) contract.Member {
	return contract.Member{
		Id: value.ID(), WorkspaceId: value.WorkspaceID(), UserId: value.UserID(),
		Role: string(value.Role()), Name: value.Name(), Email: value.Email(),
		AvatarUrl: value.AvatarURL(), CreatedAt: value.CreatedAt().Format(time.RFC3339Nano),
	}
}

var _ contract.MemberService = (*MemberUseCase)(nil)
