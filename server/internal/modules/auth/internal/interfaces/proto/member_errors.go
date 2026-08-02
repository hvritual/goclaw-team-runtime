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
