// dddgen:service-implementation MemberService; method bodies are user-owned.
package application

import (
	"context"
	"errors"
	"time"

	"github.com/multica-ai/multica/server/internal/modules/auth/contract"
	"github.com/multica-ai/multica/server/internal/modules/auth/internal/domain/member"
)

var (
	ErrMembershipNotFound = errors.New("membership not found")
	ErrInvitationNotFound = errors.New("invitation not found")
)

type MemberRepository interface {
	FindByUserAndWorkspace(context.Context, string, string) (member.Member, error)
	FindByIDAndWorkspace(context.Context, string, string) (member.Member, error)
	ListByWorkspace(context.Context, string) ([]member.Member, error)
	CountOwners(context.Context, string) (int, error)
	UpdateRole(context.Context, string, string, member.Role) (member.Member, error)
	DeleteByIDAndWorkspace(context.Context, string, string) error
	DeleteByUserAndWorkspace(context.Context, string, string) error
}

type MemberUnitOfWork interface {
	WithinTransaction(context.Context, func(MemberRepository) error) error
}

type InvitationRepository interface {
	RevokePending(context.Context, string, string, time.Time) error
}

type InvitationUnitOfWork interface {
	WithinInvitationTransaction(context.Context, func(MemberRepository, InvitationRepository) error) error
}

type MemberServiceOption func(*MemberService)

type MemberService struct {
	unitOfWork           MemberUnitOfWork
	invitationUnitOfWork InvitationUnitOfWork
	now                  func() time.Time
}

func WithMemberUnitOfWork(unitOfWork MemberUnitOfWork) MemberServiceOption {
	return func(service *MemberService) { service.unitOfWork = unitOfWork }
}

func WithInvitationUnitOfWork(unitOfWork InvitationUnitOfWork) MemberServiceOption {
	return func(service *MemberService) { service.invitationUnitOfWork = unitOfWork }
}

func WithInvitationClock(now func() time.Time) MemberServiceOption {
	return func(service *MemberService) { service.now = now }
}

func NewMemberService(options ...MemberServiceOption) *MemberService {
	service := &MemberService{now: time.Now}
	for _, option := range options {
		option(service)
	}
	return service
}

func (s *MemberService) UpdateMemberRole(ctx context.Context, request contract.Member_UpdateMemberRoleRequest) (contract.Member_Member, error) {
	nextRole, err := member.ParseRole(request.Role)
	if err != nil {
		return contract.Member_Member{}, contract.ErrInvalidMemberRole
	}
	actorUserID, ok := contract.MemberActor(ctx)
	if !ok {
		return contract.Member_Member{}, contract.ErrMemberActorRequired
	}
	if s.unitOfWork == nil {
		return contract.Member_Member{}, contract.ErrMemberNotImplemented
	}

	var updated member.Member
	err = s.unitOfWork.WithinTransaction(ctx, func(repository MemberRepository) error {
		requester, findErr := findMemberManager(ctx, repository, actorUserID, request.WorkspaceId)
		if findErr != nil {
			return findErr
		}
		target, findErr := repository.FindByIDAndWorkspace(ctx, request.MemberId, request.WorkspaceId)
		if errors.Is(findErr, ErrMembershipNotFound) {
			return contract.ErrMemberNotFound
		}
		if findErr != nil {
			return findErr
		}

		ownerCount := 0
		if target.Role == member.RoleOwner && nextRole != member.RoleOwner {
			ownerCount, findErr = repository.CountOwners(ctx, request.WorkspaceId)
			if findErr != nil {
				return findErr
			}
		}
		if policyErr := member.ValidateRoleChange(requester.Role, target.Role, nextRole, ownerCount); policyErr != nil {
			return mapMemberPolicyError(policyErr)
		}
		updated, findErr = repository.UpdateRole(ctx, request.WorkspaceId, target.ID, nextRole)
		return findErr
	})
	if err != nil {
		return contract.Member_Member{}, err
	}
	return memberContract(updated), nil
}

func mapMemberPolicyError(err error) error {
	switch {
	case errors.Is(err, member.ErrInsufficientWorkspaceRole):
		return contract.ErrInsufficientWorkspaceRole
	case errors.Is(err, member.ErrOwnerRoleRequiresOwner):
		return contract.ErrOwnerRoleRequiresOwner
	case errors.Is(err, member.ErrOwnerRemovalRequiresOwner):
		return contract.ErrOwnerRemovalRequiresOwner
	case errors.Is(err, member.ErrLastOwner):
		return contract.ErrLastWorkspaceOwner
	default:
		return err
	}
}

func memberContract(value member.Member) contract.Member_Member {
	return contract.Member_Member{
		Id:          value.ID,
		WorkspaceId: value.WorkspaceID,
		UserId:      value.UserID,
		Role:        string(value.Role),
		CreatedAt:   value.CreatedAt,
		Name:        value.Name,
		Email:       value.Email,
		AvatarUrl:   value.AvatarURL,
	}
}

func membersContract(values []member.Member) []contract.Member_Member {
	result := make([]contract.Member_Member, 0, len(values))
	for _, value := range values {
		result = append(result, memberContract(value))
	}
	return result
}
func (s *MemberService) DeleteMember(ctx context.Context, request contract.Member_DeleteMemberRequest) (contract.Member_DeleteMemberResponse, error) {
	actorUserID, ok := contract.MemberActor(ctx)
	if !ok {
		return contract.Member_DeleteMemberResponse{}, contract.ErrMemberActorRequired
	}
	if s.unitOfWork == nil {
		return contract.Member_DeleteMemberResponse{}, contract.ErrMemberNotImplemented
	}
	err := s.unitOfWork.WithinTransaction(ctx, func(repository MemberRepository) error {
		requester, findErr := findMemberManager(ctx, repository, actorUserID, request.WorkspaceId)
		if findErr != nil {
			return findErr
		}
		target, findErr := repository.FindByIDAndWorkspace(ctx, request.MemberId, request.WorkspaceId)
		if errors.Is(findErr, ErrMembershipNotFound) {
			return contract.ErrMemberNotFound
		}
		if findErr != nil {
			return findErr
		}
		ownerCount := 0
		if target.Role == member.RoleOwner {
			ownerCount, findErr = repository.CountOwners(ctx, request.WorkspaceId)
			if findErr != nil {
				return findErr
			}
		}
		if policyErr := member.ValidateRemoval(requester.Role, target.Role, ownerCount); policyErr != nil {
			return mapMemberPolicyError(policyErr)
		}
		return repository.DeleteByIDAndWorkspace(ctx, request.WorkspaceId, target.ID)
	})
	if err != nil {
		return contract.Member_DeleteMemberResponse{}, err
	}
	return contract.Member_DeleteMemberResponse{}, nil
}

func findMemberManager(ctx context.Context, repository MemberRepository, actorUserID, workspaceID string) (member.Member, error) {
	requester, err := repository.FindByUserAndWorkspace(ctx, actorUserID, workspaceID)
	if errors.Is(err, ErrMembershipNotFound) {
		return member.Member{}, contract.ErrWorkspaceMembershipHidden
	}
	if err != nil {
		return member.Member{}, err
	}
	if err := member.ValidateManager(requester.Role); err != nil {
		return member.Member{}, mapMemberPolicyError(err)
	}
	return requester, nil
}
func (s *MemberService) LeaveWorkspace(ctx context.Context, request contract.Member_LeaveWorkspaceRequest) (contract.Member_LeaveWorkspaceResponse, error) {
	actorUserID, ok := contract.MemberActor(ctx)
	if !ok {
		return contract.Member_LeaveWorkspaceResponse{}, contract.ErrMemberActorRequired
	}
	if s.unitOfWork == nil {
		return contract.Member_LeaveWorkspaceResponse{}, contract.ErrMemberNotImplemented
	}
	err := s.unitOfWork.WithinTransaction(ctx, func(repository MemberRepository) error {
		membership, findErr := repository.FindByUserAndWorkspace(ctx, actorUserID, request.WorkspaceId)
		if errors.Is(findErr, ErrMembershipNotFound) {
			return contract.ErrWorkspaceMembershipHidden
		}
		if findErr != nil {
			return findErr
		}
		ownerCount := 0
		if membership.Role == member.RoleOwner {
			ownerCount, findErr = repository.CountOwners(ctx, request.WorkspaceId)
			if findErr != nil {
				return findErr
			}
		}
		if policyErr := member.ValidateDeparture(membership.Role, ownerCount); policyErr != nil {
			return mapMemberPolicyError(policyErr)
		}
		return repository.DeleteByUserAndWorkspace(ctx, request.WorkspaceId, actorUserID)
	})
	if err != nil {
		return contract.Member_LeaveWorkspaceResponse{}, err
	}
	return contract.Member_LeaveWorkspaceResponse{}, nil
}
func (s *MemberService) ListMembers(ctx context.Context, request contract.Member_ListMembersRequest) (contract.Member_ListMembersResponse, error) {
	actorUserID, ok := contract.MemberActor(ctx)
	if !ok {
		return contract.Member_ListMembersResponse{}, contract.ErrMemberActorRequired
	}
	if s.unitOfWork == nil {
		return contract.Member_ListMembersResponse{}, contract.ErrMemberNotImplemented
	}
	var memberships []member.Member
	err := s.unitOfWork.WithinTransaction(ctx, func(repository MemberRepository) error {
		_, findErr := repository.FindByUserAndWorkspace(ctx, actorUserID, request.WorkspaceId)
		if errors.Is(findErr, ErrMembershipNotFound) {
			return contract.ErrWorkspaceMembershipHidden
		}
		if findErr != nil {
			return findErr
		}
		memberships, findErr = repository.ListByWorkspace(ctx, request.WorkspaceId)
		return findErr
	})
	if err != nil {
		return contract.Member_ListMembersResponse{}, err
	}
	return contract.Member_ListMembersResponse{Members: membersContract(memberships)}, nil
}
func (s *MemberService) RevokeInvitation(ctx context.Context, request contract.Member_RevokeInvitationRequest) (contract.Member_RevokeInvitationResponse, error) {
	actorUserID, ok := contract.MemberActor(ctx)
	if !ok {
		return contract.Member_RevokeInvitationResponse{}, contract.ErrMemberActorRequired
	}
	if s.invitationUnitOfWork == nil {
		return contract.Member_RevokeInvitationResponse{}, contract.ErrMemberNotImplemented
	}
	err := s.invitationUnitOfWork.WithinInvitationTransaction(
		ctx,
		func(members MemberRepository, invitations InvitationRepository) error {
			if _, findErr := findMemberManager(ctx, members, actorUserID, request.WorkspaceId); findErr != nil {
				return findErr
			}
			if revokeErr := invitations.RevokePending(ctx, request.WorkspaceId, request.InvitationId, s.now().UTC()); revokeErr != nil {
				if errors.Is(revokeErr, ErrInvitationNotFound) {
					return contract.ErrInvitationNotFound
				}
				return revokeErr
			}
			return nil
		},
	)
	if err != nil {
		return contract.Member_RevokeInvitationResponse{}, err
	}
	return contract.Member_RevokeInvitationResponse{}, nil
}
