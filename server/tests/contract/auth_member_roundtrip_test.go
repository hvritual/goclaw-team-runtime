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
	request contract.Member_UpdateMemberRoleRequest
}

func (s *successfulMemberService) UpdateMemberRole(_ context.Context, request contract.Member_UpdateMemberRoleRequest) (contract.Member_Member, error) {
	s.request = request
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

func TestAuthMemberGRPCRoundTripPreservesRoleString(t *testing.T) {
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
	if service.request.Role != "admin" || result.Role != "admin" {
		t.Fatalf("role round trip request=%q result=%q", service.request.Role, result.Role)
	}
	if result.Id != "member-1" || result.WorkspaceId != "workspace-1" {
		t.Fatalf("unexpected member result: %+v", result)
	}
}
