// dddgen:service-implementation MemberService; method bodies are user-owned.
package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/multica-ai/multica/server/internal/modules/auth/contract"
	"github.com/multica-ai/multica/server/internal/modules/auth/internal/domain/invitation"
	"github.com/multica-ai/multica/server/internal/modules/auth/internal/domain/member"
	workspacecontract "github.com/multica-ai/multica/server/internal/modules/workspace/contract"
)

var (
	ErrMembershipNotFound      = errors.New("membership not found")
	ErrAuthUserNotFound        = errors.New("auth user not found")
	ErrInvitationNotFound      = errors.New("invitation not found")
	ErrPendingInvitationExists = errors.New("pending invitation exists")
)

const defaultInvitationLifetime = 7 * 24 * time.Hour

type MemberRepository interface {
	FindByUserAndWorkspace(context.Context, string, string) (member.Member, error)
	FindByIDAndWorkspace(context.Context, string, string) (member.Member, error)
	ListByWorkspace(context.Context, string) ([]member.Member, error)
	ExistsByEmail(context.Context, string, string) (bool, error)
	FindUserIDByEmail(context.Context, string) (*string, error)
	FindUserByID(context.Context, string) (member.UserIdentity, error)
	CountOwners(context.Context, string) (int, error)
	UpdateRole(context.Context, string, string, member.Role) (member.Member, error)
	DeleteByIDAndWorkspace(context.Context, string, string) error
	DeleteByUserAndWorkspace(context.Context, string, string) error
}

type MemberUnitOfWork interface {
	WithinTransaction(context.Context, func(MemberRepository) error) error
}

type InvitationRepository interface {
	ExpirePendingByWorkspace(context.Context, string, time.Time) error
	ExpirePendingByWorkspaceAndEmail(context.Context, string, string, time.Time) error
	PendingExistsByWorkspaceAndEmail(context.Context, string, string) (bool, error)
	ListPendingByWorkspace(context.Context, string) ([]invitation.Invitation, error)
	Create(context.Context, invitation.Invitation) error
	RevokePending(context.Context, string, string, time.Time) error
	ExpirePendingByInvitee(context.Context, member.UserIdentity, time.Time) error
	ListPendingByInvitee(context.Context, member.UserIdentity) ([]invitation.Invitation, error)
	FindByID(context.Context, string) (invitation.Invitation, error)
}

type InvitationUnitOfWork interface {
	WithinInvitationTransaction(context.Context, func(MemberRepository, InvitationRepository) error) error
}

type MemberServiceOption func(*MemberService)

type MemberService struct {
	unitOfWork           MemberUnitOfWork
	invitationUnitOfWork InvitationUnitOfWork
	workspaceIdentities  workspacecontract.WorkspaceIdentityReader
	now                  func() time.Time
	newInvitationID      func() string
	invitationLifetime   time.Duration
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

func WithInvitationIDGenerator(generator func() string) MemberServiceOption {
	return func(service *MemberService) { service.newInvitationID = generator }
}

func WithInvitationLifetime(lifetime time.Duration) MemberServiceOption {
	return func(service *MemberService) { service.invitationLifetime = lifetime }
}

func WithWorkspaceIdentityReader(reader workspacecontract.WorkspaceIdentityReader) MemberServiceOption {
	return func(service *MemberService) { service.workspaceIdentities = reader }
}

func NewMemberService(options ...MemberServiceOption) *MemberService {
	service := &MemberService{now: time.Now, invitationLifetime: defaultInvitationLifetime}
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
func (s *MemberService) ListWorkspaceInvitations(ctx context.Context, request contract.Member_ListWorkspaceInvitationsRequest) (contract.Member_ListWorkspaceInvitationsResponse, error) {
	actorUserID, ok := contract.MemberActor(ctx)
	if !ok {
		return contract.Member_ListWorkspaceInvitationsResponse{}, contract.ErrMemberActorRequired
	}
	if s.invitationUnitOfWork == nil || s.workspaceIdentities == nil {
		return contract.Member_ListWorkspaceInvitationsResponse{}, contract.ErrMemberNotImplemented
	}
	var values []invitation.Invitation
	err := s.invitationUnitOfWork.WithinInvitationTransaction(
		ctx,
		func(members MemberRepository, invitations InvitationRepository) error {
			if _, findErr := findMemberManager(ctx, members, actorUserID, request.WorkspaceId); findErr != nil {
				return findErr
			}
			if expireErr := invitations.ExpirePendingByWorkspace(ctx, request.WorkspaceId, s.now().UTC()); expireErr != nil {
				return expireErr
			}
			var listErr error
			values, listErr = invitations.ListPendingByWorkspace(ctx, request.WorkspaceId)
			return listErr
		},
	)
	if err != nil {
		return contract.Member_ListWorkspaceInvitationsResponse{}, err
	}
	identity, err := s.workspaceIdentities.FindIdentity(ctx, request.WorkspaceId)
	if errors.Is(err, workspacecontract.ErrWorkspaceNotFound) {
		return contract.Member_ListWorkspaceInvitationsResponse{}, contract.ErrWorkspaceMembershipHidden
	}
	if err != nil {
		return contract.Member_ListWorkspaceInvitationsResponse{}, err
	}
	return contract.Member_ListWorkspaceInvitationsResponse{
		Invitations: invitationContracts(values, identity.Name),
	}, nil
}

func invitationContracts(values []invitation.Invitation, workspaceName string) []contract.Member_Invitation {
	result := make([]contract.Member_Invitation, 0, len(values))
	for _, value := range values {
		result = append(result, invitationContract(value, workspaceName))
	}
	return result
}

func invitationContract(value invitation.Invitation, workspaceName string) contract.Member_Invitation {
	return contract.Member_Invitation{
		Id: value.ID, WorkspaceId: value.WorkspaceID, InviterId: value.InviterID,
		InviteeEmail: value.InviteeEmail, InviteeUserId: value.InviteeUserID,
		Role: string(value.Role), Status: string(value.Status),
		CreatedAt:     value.CreatedAt.String(),
		UpdatedAt:     value.UpdatedAt.String(),
		ExpiresAt:     value.ExpiresAt.String(),
		WorkspaceName: workspaceName, InviterName: value.InviterName, InviterEmail: value.InviterEmail,
	}
}
func (s *MemberService) CreateInvitation(ctx context.Context, request contract.Member_CreateInvitationRequest) (contract.Member_Invitation, error) {
	actorUserID, ok := contract.MemberActor(ctx)
	if !ok {
		return contract.Member_Invitation{}, contract.ErrMemberActorRequired
	}
	if s.invitationUnitOfWork == nil || s.workspaceIdentities == nil || s.newInvitationID == nil {
		return contract.Member_Invitation{}, contract.ErrMemberNotImplemented
	}

	var created invitation.Invitation
	err := s.invitationUnitOfWork.WithinInvitationTransaction(
		ctx,
		func(members MemberRepository, invitations InvitationRepository) error {
			requester, findErr := findMemberManager(ctx, members, actorUserID, request.WorkspaceId)
			if findErr != nil {
				return findErr
			}
			pending, createErr := s.newPendingInvitation(actorUserID, request)
			if createErr != nil {
				return createErr
			}
			pending.InviterName = requester.Name
			pending.InviterEmail = requester.Email
			if createErr := persistPendingInvitation(ctx, members, invitations, &pending); createErr != nil {
				return createErr
			}
			created = pending
			return nil
		},
	)
	if err != nil {
		return contract.Member_Invitation{}, err
	}
	identity, err := s.workspaceIdentities.FindIdentity(ctx, request.WorkspaceId)
	if errors.Is(err, workspacecontract.ErrWorkspaceNotFound) {
		return contract.Member_Invitation{}, contract.ErrWorkspaceMembershipHidden
	}
	if err != nil {
		return contract.Member_Invitation{}, err
	}
	return invitationContracts([]invitation.Invitation{created}, identity.Name)[0], nil
}

func (s *MemberService) AuthorizeCreateInvitation(
	ctx context.Context,
	request contract.Member_CreateInvitationRequest,
) error {
	actorUserID, ok := contract.MemberActor(ctx)
	if !ok {
		return contract.ErrMemberActorRequired
	}
	if s.invitationUnitOfWork == nil {
		return contract.ErrMemberNotImplemented
	}
	return s.invitationUnitOfWork.WithinInvitationTransaction(
		ctx,
		func(members MemberRepository, _ InvitationRepository) error {
			_, err := findMemberManager(ctx, members, actorUserID, request.WorkspaceId)
			return err
		},
	)
}

func (s *MemberService) newPendingInvitation(
	actorUserID string,
	request contract.Member_CreateInvitationRequest,
) (invitation.Invitation, error) {
	role, err := invitationRole(request.Role)
	if err != nil {
		return invitation.Invitation{}, err
	}
	pending, err := invitation.NewPending(
		s.newInvitationID(), request.WorkspaceId, actorUserID, request.Email,
		role, nil, s.now(), s.invitationLifetime,
	)
	switch {
	case errors.Is(err, invitation.ErrInvalidEmail):
		return invitation.Invitation{}, contract.ErrInvalidInvitationEmail
	case errors.Is(err, invitation.ErrInvalidRole):
		return invitation.Invitation{}, contract.ErrInvalidInvitationRole
	default:
		return pending, err
	}
}

func persistPendingInvitation(
	ctx context.Context,
	members MemberRepository,
	invitations InvitationRepository,
	pending *invitation.Invitation,
) error {
	memberExists, err := members.ExistsByEmail(ctx, pending.WorkspaceID, pending.InviteeEmail)
	if err != nil {
		return err
	}
	if memberExists {
		return contract.ErrInviteeAlreadyMember
	}
	createdAt, err := pending.CreatedAt.Time()
	if err != nil {
		return fmt.Errorf("parse new invitation creation time: %w", err)
	}
	if err := invitations.ExpirePendingByWorkspaceAndEmail(
		ctx, pending.WorkspaceID, pending.InviteeEmail, createdAt,
	); err != nil {
		return err
	}
	pendingExists, err := invitations.PendingExistsByWorkspaceAndEmail(
		ctx, pending.WorkspaceID, pending.InviteeEmail,
	)
	if err != nil {
		return err
	}
	if pendingExists {
		return contract.ErrInvitationAlreadyPending
	}
	pending.InviteeUserID, err = members.FindUserIDByEmail(ctx, pending.InviteeEmail)
	if err != nil {
		return err
	}
	err = invitations.Create(ctx, *pending)
	if errors.Is(err, ErrPendingInvitationExists) {
		return contract.ErrInvitationAlreadyPending
	}
	return err
}

func invitationRole(value string) (member.Role, error) {
	if value == "" {
		return member.RoleMember, nil
	}
	role, err := member.ParseRole(value)
	if err != nil || role == member.RoleOwner {
		return "", contract.ErrInvalidInvitationRole
	}
	return role, nil
}
func (s *MemberService) ListMyInvitations(ctx context.Context, _ contract.Member_ListMyInvitationsRequest) (contract.Member_ListMyInvitationsResponse, error) {
	actorUserID, ok := contract.MemberActor(ctx)
	if !ok {
		return contract.Member_ListMyInvitationsResponse{}, contract.ErrMemberActorRequired
	}
	if s.invitationUnitOfWork == nil || s.workspaceIdentities == nil {
		return contract.Member_ListMyInvitationsResponse{}, contract.ErrMemberNotImplemented
	}

	var values []invitation.Invitation
	err := s.invitationUnitOfWork.WithinInvitationTransaction(
		ctx,
		func(members MemberRepository, invitations InvitationRepository) error {
			current, findErr := findAuthUserIdentity(ctx, members, actorUserID)
			if findErr != nil {
				return findErr
			}
			if expireErr := invitations.ExpirePendingByInvitee(ctx, current, s.now().UTC()); expireErr != nil {
				return expireErr
			}
			var listErr error
			values, listErr = invitations.ListPendingByInvitee(ctx, current)
			return listErr
		},
	)
	if err != nil {
		return contract.Member_ListMyInvitationsResponse{}, err
	}

	result, err := s.personalInvitationContracts(ctx, values)
	if err != nil {
		return contract.Member_ListMyInvitationsResponse{}, err
	}
	return contract.Member_ListMyInvitationsResponse{Invitations: result}, nil
}

func (s *MemberService) GetMyInvitation(ctx context.Context, request contract.Member_GetMyInvitationRequest) (contract.Member_GetMyInvitationResponse, error) {
	actorUserID, ok := contract.MemberActor(ctx)
	if !ok {
		return contract.Member_GetMyInvitationResponse{}, contract.ErrMemberActorRequired
	}
	if s.invitationUnitOfWork == nil || s.workspaceIdentities == nil {
		return contract.Member_GetMyInvitationResponse{}, contract.ErrMemberNotImplemented
	}

	var value invitation.Invitation
	err := s.invitationUnitOfWork.WithinInvitationTransaction(
		ctx,
		func(members MemberRepository, invitations InvitationRepository) error {
			current, findErr := findAuthUserIdentity(ctx, members, actorUserID)
			if findErr != nil {
				return findErr
			}
			value, findErr = invitations.FindByID(ctx, request.InvitationId)
			if errors.Is(findErr, ErrInvitationNotFound) {
				return contract.ErrInvitationNotFound
			}
			if findErr != nil {
				return findErr
			}
			if !value.BelongsTo(current.ID, current.Email) {
				return contract.ErrInvitationForbidden
			}
			return nil
		},
	)
	if err != nil {
		return contract.Member_GetMyInvitationResponse{}, err
	}

	identity, err := s.workspaceIdentities.FindIdentity(ctx, value.WorkspaceID)
	if errors.Is(err, workspacecontract.ErrWorkspaceNotFound) {
		return contract.Member_GetMyInvitationResponse{}, contract.ErrInvitationNotFound
	}
	if err != nil {
		return contract.Member_GetMyInvitationResponse{}, err
	}
	result := invitationContract(value, identity.Name)
	return contract.Member_GetMyInvitationResponse{Invitation: &result}, nil
}

func findAuthUserIdentity(
	ctx context.Context,
	members MemberRepository,
	userID string,
) (member.UserIdentity, error) {
	current, err := members.FindUserByID(ctx, userID)
	if errors.Is(err, ErrAuthUserNotFound) {
		return member.UserIdentity{}, contract.ErrAuthUserNotFound
	}
	return current, err
}

func (s *MemberService) personalInvitationContracts(
	ctx context.Context,
	values []invitation.Invitation,
) ([]contract.Member_Invitation, error) {
	result := make([]contract.Member_Invitation, 0, len(values))
	workspaceNames := make(map[string]string)
	missingWorkspaces := make(map[string]struct{})
	for _, value := range values {
		if _, missing := missingWorkspaces[value.WorkspaceID]; missing {
			continue
		}
		workspaceName, found := workspaceNames[value.WorkspaceID]
		if !found {
			identity, err := s.workspaceIdentities.FindIdentity(ctx, value.WorkspaceID)
			if errors.Is(err, workspacecontract.ErrWorkspaceNotFound) {
				missingWorkspaces[value.WorkspaceID] = struct{}{}
				continue
			}
			if err != nil {
				return nil, err
			}
			workspaceName = identity.Name
			workspaceNames[value.WorkspaceID] = workspaceName
		}
		result = append(result, invitationContract(value, workspaceName))
	}
	return result, nil
}
