// dddgen:service-implementation MemberService; method bodies are user-owned.
package application

import (
	"context"
	"errors"

	"github.com/multica-ai/multica/server/internal/modules/auth/contract"
	"github.com/multica-ai/multica/server/internal/modules/auth/internal/domain/member"
)

var ErrMembershipNotFound = errors.New("membership not found")

type MemberRepository interface {
	FindByUserAndWorkspace(context.Context, string, string) (member.Member, error)
	FindByIDAndWorkspace(context.Context, string, string) (member.Member, error)
	CountOwners(context.Context, string) (int, error)
	UpdateRole(context.Context, string, string, member.Role) (member.Member, error)
}

type MemberUnitOfWork interface {
	WithinTransaction(context.Context, func(MemberRepository) error) error
}

type MemberServiceOption func(*MemberService)

type MemberService struct {
	unitOfWork MemberUnitOfWork
}

func WithMemberUnitOfWork(unitOfWork MemberUnitOfWork) MemberServiceOption {
	return func(service *MemberService) { service.unitOfWork = unitOfWork }
}

func NewMemberService(options ...MemberServiceOption) *MemberService {
	service := &MemberService{}
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
		requester, findErr := repository.FindByUserAndWorkspace(ctx, actorUserID, request.WorkspaceId)
		if errors.Is(findErr, ErrMembershipNotFound) {
			return contract.ErrWorkspaceMembershipHidden
		}
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
