package contract_test

import (
	"context"
	"net"
	"testing"

	"github.com/multica-ai/multica/server/internal/modules/auth"
	"github.com/multica-ai/multica/server/internal/modules/auth/contract"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

type successfulMemberService struct {
	listRequest              contract.Member_ListMembersRequest
	updateRequest            contract.Member_UpdateMemberRoleRequest
	deleteRequest            contract.Member_DeleteMemberRequest
	leaveRequest             contract.Member_LeaveWorkspaceRequest
	revokeRequest            contract.Member_RevokeInvitationRequest
	listInvitationsRequest   contract.Member_ListWorkspaceInvitationsRequest
	createInvitationRequest  contract.Member_CreateInvitationRequest
	listMyInvitationsRequest contract.Member_ListMyInvitationsRequest
	getMyInvitationRequest   contract.Member_GetMyInvitationRequest
}

func (s *successfulMemberService) ListMembers(_ context.Context, request contract.Member_ListMembersRequest) (contract.Member_ListMembersResponse, error) {
	s.listRequest = request
	return contract.Member_ListMembersResponse{Members: []contract.Member_Member{
		{
			Id:          "member-1",
			WorkspaceId: request.WorkspaceId,
			UserId:      "user-1",
			Role:        "owner",
			CreatedAt:   "2026-08-02T00:00:00Z",
			Name:        "Owner",
			Email:       "owner@example.test",
		},
	}}, nil
}

func (s *successfulMemberService) UpdateMemberRole(_ context.Context, request contract.Member_UpdateMemberRoleRequest) (contract.Member_Member, error) {
	s.updateRequest = request
	return contract.Member_Member{
		Id:          request.MemberId,
		WorkspaceId: request.WorkspaceId,
		UserId:      "user-1",
		Role:        request.Role,
		CreatedAt:   "2026-08-02T00:00:00Z",
		Name:        "Member",
		Email:       "member@example.test",
	}, nil
}

func (s *successfulMemberService) DeleteMember(_ context.Context, request contract.Member_DeleteMemberRequest) (contract.Member_DeleteMemberResponse, error) {
	s.deleteRequest = request
	return contract.Member_DeleteMemberResponse{}, nil
}

func (s *successfulMemberService) LeaveWorkspace(_ context.Context, request contract.Member_LeaveWorkspaceRequest) (contract.Member_LeaveWorkspaceResponse, error) {
	s.leaveRequest = request
	return contract.Member_LeaveWorkspaceResponse{}, nil
}

func (s *successfulMemberService) RevokeInvitation(_ context.Context, request contract.Member_RevokeInvitationRequest) (contract.Member_RevokeInvitationResponse, error) {
	s.revokeRequest = request
	return contract.Member_RevokeInvitationResponse{}, nil
}

func (s *successfulMemberService) ListWorkspaceInvitations(_ context.Context, request contract.Member_ListWorkspaceInvitationsRequest) (contract.Member_ListWorkspaceInvitationsResponse, error) {
	s.listInvitationsRequest = request
	return contract.Member_ListWorkspaceInvitationsResponse{Invitations: []contract.Member_Invitation{{
		Id: "invitation-1", WorkspaceId: request.WorkspaceId, InviterId: "user-1",
		InviteeEmail: "invitee@example.test", Role: "member", Status: "pending",
		CreatedAt: "2026-08-02T00:00:00Z", UpdatedAt: "2026-08-02T00:00:00Z", ExpiresAt: "2026-08-09T00:00:00Z",
		WorkspaceName: "Acme", InviterName: "Owner", InviterEmail: "owner@example.test",
	}}}, nil
}

func (s *successfulMemberService) CreateInvitation(_ context.Context, request contract.Member_CreateInvitationRequest) (contract.Member_Invitation, error) {
	s.createInvitationRequest = request
	return contract.Member_Invitation{
		Id: "invitation-created", WorkspaceId: request.WorkspaceId, InviterId: "user-1",
		InviteeEmail: request.Email, Role: request.Role, Status: "pending",
		CreatedAt: "2026-08-02T00:00:00Z", UpdatedAt: "2026-08-02T00:00:00Z", ExpiresAt: "2026-08-09T00:00:00Z",
		WorkspaceName: "Acme", InviterName: "Owner", InviterEmail: "owner@example.test",
	}, nil
}

func (s *successfulMemberService) ListMyInvitations(_ context.Context, request contract.Member_ListMyInvitationsRequest) (contract.Member_ListMyInvitationsResponse, error) {
	s.listMyInvitationsRequest = request
	return contract.Member_ListMyInvitationsResponse{Invitations: []contract.Member_Invitation{personalInvitationContract()}}, nil
}

func (s *successfulMemberService) GetMyInvitation(_ context.Context, request contract.Member_GetMyInvitationRequest) (contract.Member_GetMyInvitationResponse, error) {
	s.getMyInvitationRequest = request
	value := personalInvitationContract()
	value.Id = request.InvitationId
	return contract.Member_GetMyInvitationResponse{Invitation: &value}, nil
}

func personalInvitationContract() contract.Member_Invitation {
	return contract.Member_Invitation{
		Id: "invitation-personal", WorkspaceId: "workspace-1", InviterId: "user-1",
		InviteeEmail: "invitee@example.test", Role: "member", Status: "pending",
		CreatedAt: "2026-08-02T00:00:00Z", UpdatedAt: "2026-08-02T00:00:00Z", ExpiresAt: "2026-08-09T00:00:00Z",
		WorkspaceName: "Acme", InviterName: "Owner", InviterEmail: "owner@example.test",
	}
}

func (*successfulMemberService) AuthorizeCreateInvitation(context.Context, contract.Member_CreateInvitationRequest) error {
	return nil
}

func TestAuthMemberGRPCRoundTrips(t *testing.T) {
	service := &successfulMemberService{}
	extension := auth.NewMemberExtensionWithService(service)
	listener := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer()
	extension.RegisterGRPC(grpcServer)
	go func() { _ = grpcServer.Serve(listener) }()
	t.Cleanup(grpcServer.Stop)
	connection, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = connection.Close() })

	request := contract.Member_UpdateMemberRoleRequest{
		WorkspaceId: "workspace-1",
		MemberId:    "member-1",
		Role:        "admin",
	}
	result, err := auth.NewMemberGRPCClient(connection).UpdateMemberRole(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	if service.updateRequest.Role != "admin" || result.Role != "admin" {
		t.Fatalf("role round trip request=%q result=%q", service.updateRequest.Role, result.Role)
	}
	if result.Id != "member-1" || result.WorkspaceId != "workspace-1" {
		t.Fatalf("unexpected member result: %+v", result)
	}
	client := auth.NewMemberGRPCClient(connection)
	listed, err := client.ListMembers(t.Context(), contract.Member_ListMembersRequest{WorkspaceId: "workspace-1"})
	if err != nil {
		t.Fatal(err)
	}
	if service.listRequest.WorkspaceId != "workspace-1" || len(listed.Members) != 1 || listed.Members[0].Role != "owner" {
		t.Fatalf("unexpected list round trip request=%+v result=%+v", service.listRequest, listed)
	}
	if _, err := client.DeleteMember(t.Context(), contract.Member_DeleteMemberRequest{
		WorkspaceId: "workspace-1",
		MemberId:    "member-2",
	}); err != nil {
		t.Fatal(err)
	}
	if service.deleteRequest.WorkspaceId != "workspace-1" || service.deleteRequest.MemberId != "member-2" {
		t.Fatalf("unexpected delete request: %+v", service.deleteRequest)
	}
	if _, err := client.LeaveWorkspace(t.Context(), contract.Member_LeaveWorkspaceRequest{
		WorkspaceId: "workspace-1",
	}); err != nil {
		t.Fatal(err)
	}
	if service.leaveRequest.WorkspaceId != "workspace-1" {
		t.Fatalf("unexpected leave request: %+v", service.leaveRequest)
	}
	if _, err := client.RevokeInvitation(t.Context(), contract.Member_RevokeInvitationRequest{
		WorkspaceId: "workspace-1", InvitationId: "invitation-1",
	}); err != nil {
		t.Fatal(err)
	}
	if service.revokeRequest.WorkspaceId != "workspace-1" || service.revokeRequest.InvitationId != "invitation-1" {
		t.Fatalf("unexpected revoke request: %+v", service.revokeRequest)
	}
	invitationList, err := client.ListWorkspaceInvitations(t.Context(), contract.Member_ListWorkspaceInvitationsRequest{
		WorkspaceId: "workspace-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if service.listInvitationsRequest.WorkspaceId != "workspace-1" || len(invitationList.Invitations) != 1 || invitationList.Invitations[0].WorkspaceName != "Acme" {
		t.Fatalf("unexpected invitation list round trip request=%+v result=%+v", service.listInvitationsRequest, invitationList)
	}
	assertCreateInvitationGRPCRoundTrip(t, client, service)
	assertPersonalInvitationGRPCRoundTrip(t, client, service)
}

func assertPersonalInvitationGRPCRoundTrip(t *testing.T, client contract.MemberService, service *successfulMemberService) {
	t.Helper()
	personalList, err := client.ListMyInvitations(t.Context(), contract.Member_ListMyInvitationsRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(personalList.Invitations) != 1 || personalList.Invitations[0].Id != "invitation-personal" {
		t.Fatalf("unexpected personal invitation list round trip: %+v", personalList)
	}
	personalDetail, err := client.GetMyInvitation(t.Context(), contract.Member_GetMyInvitationRequest{InvitationId: "invitation-detail"})
	if err != nil {
		t.Fatal(err)
	}
	if service.getMyInvitationRequest.InvitationId != "invitation-detail" || personalDetail.Invitation == nil || personalDetail.Invitation.Id != "invitation-detail" {
		t.Fatalf("unexpected personal invitation detail round trip request=%+v result=%+v", service.getMyInvitationRequest, personalDetail)
	}
}

func assertCreateInvitationGRPCRoundTrip(t *testing.T, client contract.MemberService, service *successfulMemberService) {
	t.Helper()
	created, err := client.CreateInvitation(t.Context(), contract.Member_CreateInvitationRequest{
		WorkspaceId: "workspace-1", Email: "invitee@example.test", Role: "admin",
	})
	if err != nil {
		t.Fatal(err)
	}
	if service.createInvitationRequest.Email != "invitee@example.test" || created.Id != "invitation-created" || created.Role != "admin" {
		t.Fatalf("unexpected invitation create round trip request=%+v result=%+v", service.createInvitationRequest, created)
	}
}
