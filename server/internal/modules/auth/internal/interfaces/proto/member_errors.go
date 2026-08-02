package proto

import (
	"context"
	"errors"
	"net/http"

	kratoserrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/multica-ai/multica/server/internal/modules/auth/contract"
)

type memberTransportService struct {
	next contract.MemberService
}

// NewMemberTransportService translates Auth domain failures at the transport
// boundary while keeping Kratos concerns out of the application layer.
func NewMemberTransportService(next contract.MemberService) contract.MemberService {
	return &memberTransportService{next: next}
}

func (s *memberTransportService) ListMembers(ctx context.Context, request contract.Member_ListMembersRequest) (contract.Member_ListMembersResponse, error) {
	result, err := s.next.ListMembers(ctx, request)
	if err != nil {
		return contract.Member_ListMembersResponse{}, memberTransportError(err)
	}
	return result, nil
}

func (s *memberTransportService) UpdateMemberRole(ctx context.Context, request contract.Member_UpdateMemberRoleRequest) (contract.Member_Member, error) {
	result, err := s.next.UpdateMemberRole(ctx, request)
	if err != nil {
		return contract.Member_Member{}, memberTransportError(err)
	}
	return result, nil
}

func (s *memberTransportService) DeleteMember(ctx context.Context, request contract.Member_DeleteMemberRequest) (contract.Member_DeleteMemberResponse, error) {
	result, err := s.next.DeleteMember(ctx, request)
	if err != nil {
		return contract.Member_DeleteMemberResponse{}, memberTransportError(err)
	}
	return result, nil
}

func (s *memberTransportService) LeaveWorkspace(ctx context.Context, request contract.Member_LeaveWorkspaceRequest) (contract.Member_LeaveWorkspaceResponse, error) {
	result, err := s.next.LeaveWorkspace(ctx, request)
	if err != nil {
		return contract.Member_LeaveWorkspaceResponse{}, memberTransportError(err)
	}
	return result, nil
}

func (s *memberTransportService) RevokeInvitation(ctx context.Context, request contract.Member_RevokeInvitationRequest) (contract.Member_RevokeInvitationResponse, error) {
	result, err := s.next.RevokeInvitation(ctx, request)
	if err != nil {
		return contract.Member_RevokeInvitationResponse{}, memberTransportError(err)
	}
	return result, nil
}

func (s *memberTransportService) ListWorkspaceInvitations(ctx context.Context, request contract.Member_ListWorkspaceInvitationsRequest) (contract.Member_ListWorkspaceInvitationsResponse, error) {
	result, err := s.next.ListWorkspaceInvitations(ctx, request)
	if err != nil {
		return contract.Member_ListWorkspaceInvitationsResponse{}, memberTransportError(err)
	}
	return result, nil
}

func (s *memberTransportService) CreateInvitation(ctx context.Context, request contract.Member_CreateInvitationRequest) (contract.Member_Invitation, error) {
	result, err := s.next.CreateInvitation(ctx, request)
	if err != nil {
		return contract.Member_Invitation{}, memberTransportError(err)
	}
	return result, nil
}

func (s *memberTransportService) ListMyInvitations(ctx context.Context, request contract.Member_ListMyInvitationsRequest) (contract.Member_ListMyInvitationsResponse, error) {
	result, err := s.next.ListMyInvitations(ctx, request)
	if err != nil {
		return contract.Member_ListMyInvitationsResponse{}, memberTransportError(err)
	}
	return result, nil
}

func (s *memberTransportService) GetMyInvitation(ctx context.Context, request contract.Member_GetMyInvitationRequest) (contract.Member_GetMyInvitationResponse, error) {
	result, err := s.next.GetMyInvitation(ctx, request)
	if err != nil {
		return contract.Member_GetMyInvitationResponse{}, memberTransportError(err)
	}
	return result, nil
}

func (s *memberTransportService) AcceptInvitation(ctx context.Context, request contract.Member_AcceptInvitationRequest) (contract.Member_AcceptInvitationResponse, error) {
	result, err := s.next.AcceptInvitation(ctx, request)
	if err != nil {
		return contract.Member_AcceptInvitationResponse{}, memberTransportError(err)
	}
	return result, nil
}

func (s *memberTransportService) DeclineInvitation(ctx context.Context, request contract.Member_DeclineInvitationRequest) (contract.Member_DeclineInvitationResponse, error) {
	result, err := s.next.DeclineInvitation(ctx, request)
	if err != nil {
		return contract.Member_DeclineInvitationResponse{}, memberTransportError(err)
	}
	return result, nil
}

func (s *memberTransportService) AuthorizeCreateInvitation(
	ctx context.Context,
	request contract.Member_CreateInvitationRequest,
) error {
	authorizer, ok := s.next.(contract.InvitationCreationAuthorizer)
	if !ok {
		return contract.ErrMemberNotImplemented
	}
	return authorizer.AuthorizeCreateInvitation(ctx, request)
}

func memberTransportError(err error) error {
	switch {
	case errors.Is(err, contract.ErrMemberActorRequired):
		return kratoserrors.New(http.StatusUnauthorized, "MEMBER_ACTOR_REQUIRED", err.Error()).WithCause(err)
	case errors.Is(err, contract.ErrInvalidMemberRole):
		return kratoserrors.New(http.StatusBadRequest, "INVALID_MEMBER_ROLE", err.Error()).WithCause(err)
	case errors.Is(err, contract.ErrWorkspaceMembershipHidden):
		return kratoserrors.New(http.StatusNotFound, "WORKSPACE_NOT_FOUND", err.Error()).WithCause(err)
	case errors.Is(err, contract.ErrMemberNotFound):
		return kratoserrors.New(http.StatusNotFound, "MEMBER_NOT_FOUND", err.Error()).WithCause(err)
	case errors.Is(err, contract.ErrInvitationNotFound):
		return kratoserrors.New(http.StatusNotFound, "INVITATION_NOT_FOUND", err.Error()).WithCause(err)
	case errors.Is(err, contract.ErrAuthUserNotFound):
		return kratoserrors.New(http.StatusNotFound, "AUTH_USER_NOT_FOUND", err.Error()).WithCause(err)
	case errors.Is(err, contract.ErrInvitationForbidden):
		return kratoserrors.New(http.StatusForbidden, "INVITATION_FORBIDDEN", err.Error()).WithCause(err)
	case errors.Is(err, contract.ErrInvitationNotPending):
		return kratoserrors.New(http.StatusBadRequest, "INVITATION_NOT_PENDING", err.Error()).WithCause(err)
	case errors.Is(err, contract.ErrInvitationExpired):
		return kratoserrors.New(http.StatusGone, "INVITATION_EXPIRED", err.Error()).WithCause(err)
	case errors.Is(err, contract.ErrInvitationMemberExists):
		return kratoserrors.New(http.StatusConflict, "INVITATION_MEMBER_EXISTS", err.Error()).WithCause(err)
	case errors.Is(err, contract.ErrInvitationChanged):
		return kratoserrors.New(http.StatusConflict, "INVITATION_CHANGED", err.Error()).WithCause(err)
	case errors.Is(err, contract.ErrInvitationOnboarding):
		return kratoserrors.New(http.StatusInternalServerError, "INVITATION_ONBOARDING_FAILED", contract.ErrInvitationOnboarding.Error()).WithCause(err)
	case errors.Is(err, contract.ErrInvalidInvitationEmail):
		return kratoserrors.New(http.StatusBadRequest, "INVALID_INVITATION_EMAIL", err.Error()).WithCause(err)
	case errors.Is(err, contract.ErrInvalidInvitationRole):
		return kratoserrors.New(http.StatusBadRequest, "INVALID_INVITATION_ROLE", err.Error()).WithCause(err)
	case errors.Is(err, contract.ErrInviteeAlreadyMember):
		return kratoserrors.New(http.StatusConflict, "INVITEE_ALREADY_MEMBER", err.Error()).WithCause(err)
	case errors.Is(err, contract.ErrInvitationAlreadyPending):
		return kratoserrors.New(http.StatusConflict, "INVITATION_ALREADY_PENDING", err.Error()).WithCause(err)
	case errors.Is(err, contract.ErrInsufficientWorkspaceRole):
		return kratoserrors.New(http.StatusForbidden, "INSUFFICIENT_WORKSPACE_ROLE", err.Error()).WithCause(err)
	case errors.Is(err, contract.ErrOwnerRoleRequiresOwner):
		return kratoserrors.New(http.StatusForbidden, "OWNER_ROLE_REQUIRES_OWNER", err.Error()).WithCause(err)
	case errors.Is(err, contract.ErrOwnerRemovalRequiresOwner):
		return kratoserrors.New(http.StatusForbidden, "OWNER_REMOVAL_REQUIRES_OWNER", err.Error()).WithCause(err)
	case errors.Is(err, contract.ErrLastWorkspaceOwner):
		return kratoserrors.New(http.StatusBadRequest, "LAST_WORKSPACE_OWNER", err.Error()).WithCause(err)
	case errors.Is(err, contract.ErrMemberNotImplemented):
		return kratoserrors.New(http.StatusNotImplemented, "MEMBER_SERVICE_NOT_IMPLEMENTED", err.Error()).WithCause(err)
	default:
		return kratoserrors.New(http.StatusInternalServerError, "MEMBER_SERVICE_FAILURE", "member service failed").WithCause(err)
	}
}

var _ contract.MemberService = (*memberTransportService)(nil)
var _ contract.InvitationCreationAuthorizer = (*memberTransportService)(nil)
