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
	listRequest   contract.Member_ListMembersRequest
	updateRequest contract.Member_UpdateMemberRoleRequest
	deleteRequest contract.Member_DeleteMemberRequest
	leaveRequest  contract.Member_LeaveWorkspaceRequest
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
}
