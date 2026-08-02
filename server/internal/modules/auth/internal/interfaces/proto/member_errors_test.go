package proto

import (
	"context"
	"errors"
	"net/http"
	"testing"

	kratoserrors "github.com/go-kratos/kratos/v3/errors"
	authv1 "github.com/multica-ai/multica/server/gen/go/auth/v1"
	"github.com/multica-ai/multica/server/internal/modules/auth/contract"
)

type fakeMemberService struct {
	request contract.Member_UpdateMemberRoleRequest
	result  contract.Member_Member
	err     error
}

func (s *fakeMemberService) ListMembers(context.Context, contract.Member_ListMembersRequest) (contract.Member_ListMembersResponse, error) {
	return contract.Member_ListMembersResponse{}, s.err
}

func (s *fakeMemberService) UpdateMemberRole(_ context.Context, request contract.Member_UpdateMemberRoleRequest) (contract.Member_Member, error) {
	s.request = request
	return s.result, s.err
}

func (s *fakeMemberService) DeleteMember(context.Context, contract.Member_DeleteMemberRequest) (contract.Member_DeleteMemberResponse, error) {
	return contract.Member_DeleteMemberResponse{}, s.err
}

func (s *fakeMemberService) LeaveWorkspace(context.Context, contract.Member_LeaveWorkspaceRequest) (contract.Member_LeaveWorkspaceResponse, error) {
	return contract.Member_LeaveWorkspaceResponse{}, s.err
}

func (s *fakeMemberService) RevokeInvitation(context.Context, contract.Member_RevokeInvitationRequest) (contract.Member_RevokeInvitationResponse, error) {
	return contract.Member_RevokeInvitationResponse{}, s.err
}

func (s *fakeMemberService) ListWorkspaceInvitations(context.Context, contract.Member_ListWorkspaceInvitationsRequest) (contract.Member_ListWorkspaceInvitationsResponse, error) {
	return contract.Member_ListWorkspaceInvitationsResponse{}, s.err
}

func TestMemberTransportServiceMapsDomainErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code int
	}{
		{name: "actor", err: contract.ErrMemberActorRequired, code: http.StatusUnauthorized},
		{name: "invalid role", err: contract.ErrInvalidMemberRole, code: http.StatusBadRequest},
		{name: "hidden workspace", err: contract.ErrWorkspaceMembershipHidden, code: http.StatusNotFound},
		{name: "member missing", err: contract.ErrMemberNotFound, code: http.StatusNotFound},
		{name: "invitation missing", err: contract.ErrInvitationNotFound, code: http.StatusNotFound},
		{name: "insufficient role", err: contract.ErrInsufficientWorkspaceRole, code: http.StatusForbidden},
		{name: "owner role", err: contract.ErrOwnerRoleRequiresOwner, code: http.StatusForbidden},
		{name: "owner removal", err: contract.ErrOwnerRemovalRequiresOwner, code: http.StatusForbidden},
		{name: "last owner", err: contract.ErrLastWorkspaceOwner, code: http.StatusBadRequest},
		{name: "unexpected", err: errors.New("database unavailable"), code: http.StatusInternalServerError},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := NewMemberTransportService(&fakeMemberService{err: test.err})
			_, err := service.UpdateMemberRole(t.Context(), contract.Member_UpdateMemberRoleRequest{})
			if code := kratoserrors.Code(err); code != test.code {
				t.Fatalf("error code = %d, want %d: %v", code, test.code, err)
			}
			if !errors.Is(err, test.err) {
				t.Fatalf("transport error does not retain cause: %v", err)
			}
		})
	}
}

func TestMemberServerPreservesRoleStrings(t *testing.T) {
	service := &fakeMemberService{result: contract.Member_Member{
		Id:          "member-1",
		WorkspaceId: "workspace-1",
		UserId:      "user-1",
		Role:        "admin",
	}}
	server := NewMemberServer(service)

	response, err := server.UpdateMemberRole(t.Context(), &authv1.UpdateMemberRoleRequest{
		WorkspaceId: "workspace-1",
		MemberId:    "member-1",
		Role:        "admin",
	})
	if err != nil {
		t.Fatal(err)
	}
	if service.request.Role != "admin" || response.GetRole() != "admin" {
		t.Fatalf("role conversion request=%q response=%q", service.request.Role, response.GetRole())
	}
}
