package contract_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	authv1 "github.com/multica-ai/multica/server/gen/go/auth/v1"
	"github.com/multica-ai/multica/server/internal/modules/auth"
	"github.com/multica-ai/multica/server/internal/modules/auth/contract"
)

type memberListJSONService struct{ successfulMemberService }

func (*memberListJSONService) ListMembers(_ context.Context, request contract.Member_ListMembersRequest) (contract.Member_ListMembersResponse, error) {
	avatarURL := "https://cdn.example.test/avatar.png"
	return contract.Member_ListMembersResponse{Members: []contract.Member_Member{
		{Id: "member-1", WorkspaceId: request.WorkspaceId, UserId: "user-1", Role: "owner", AvatarUrl: nil},
		{Id: "member-2", WorkspaceId: request.WorkspaceId, UserId: "user-2", Role: "member", AvatarUrl: &avatarURL},
	}}, nil
}

func TestAuthMemberHTTPListUsesTopLevelArray(t *testing.T) {
	extension := auth.NewMemberExtensionWithService(&memberListJSONService{})
	server := kratoshttp.NewServer()
	extension.RegisterHTTP(server)
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/workspaces/workspace-1/members", nil)
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d; body=%s", response.Code, response.Body.String())
	}
	var members []map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &members); err != nil {
		t.Fatalf("response is not a member array: %v; body=%s", err, response.Body.String())
	}
	if len(members) != 2 || members[0]["workspace_id"] != "workspace-1" {
		t.Fatalf("unexpected member array: %+v", members)
	}
	if avatarURL, exists := members[0]["avatar_url"]; !exists || avatarURL != nil {
		t.Fatalf("nil avatar must be present as JSON null: %+v", members[0])
	}
	if members[1]["avatar_url"] != "https://cdn.example.test/avatar.png" {
		t.Fatalf("avatar must be a JSON string: %+v", members[1])
	}
}

func TestAuthMemberHTTPMutationsReturnNoContent(t *testing.T) {
	extension := auth.NewMemberExtensionWithService(&successfulMemberService{})
	server := kratoshttp.NewServer()
	extension.RegisterHTTP(server)

	tests := []struct {
		method string
		path   string
	}{
		{method: http.MethodDelete, path: "/api/workspaces/workspace-1/members/member-1"},
		{method: http.MethodPost, path: "/api/workspaces/workspace-1/leave"},
	}
	for _, test := range tests {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			request := httptest.NewRequestWithContext(t.Context(), test.method, test.path, nil)
			response := httptest.NewRecorder()

			server.ServeHTTP(response, request)

			if response.Code != http.StatusNoContent {
				t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusNoContent, response.Body.String())
			}
			if response.Body.Len() != 0 {
				t.Fatalf("204 response body = %q", response.Body.String())
			}
		})
	}
}

func TestAuthMemberGeneratedHTTPClientHandlesNoContent(t *testing.T) {
	service := &memberListJSONService{}
	extension := auth.NewMemberExtensionWithService(service)
	server := kratoshttp.NewServer()
	extension.RegisterHTTP(server)
	testServer := httptest.NewServer(server)
	t.Cleanup(testServer.Close)

	transport, err := kratoshttp.NewClient(t.Context(), kratoshttp.WithEndpoint(testServer.URL))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = transport.Close() })
	client := authv1.NewMemberServiceHTTPClient(transport)

	listed, err := client.ListMembers(t.Context(), &authv1.ListMembersRequest{WorkspaceId: "workspace-1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(listed.GetMembers()) != 2 || listed.GetMembers()[0].GetRole() != "owner" {
		t.Fatalf("unexpected generated-client member list: %+v", listed.GetMembers())
	}
	if listed.GetMembers()[0].AvatarUrl != nil || listed.GetMembers()[1].GetAvatarUrl() != "https://cdn.example.test/avatar.png" {
		t.Fatalf("generated client lost nullable avatar values: %+v", listed.GetMembers())
	}

	if _, err := client.DeleteMember(t.Context(), &authv1.DeleteMemberRequest{
		WorkspaceId: "workspace-1",
		MemberId:    "member-1",
	}); err != nil {
		t.Fatal(err)
	}
	if service.deleteRequest.WorkspaceId != "workspace-1" || service.deleteRequest.MemberId != "member-1" {
		t.Fatalf("unexpected delete request: %+v", service.deleteRequest)
	}

	if _, err := client.LeaveWorkspace(t.Context(), &authv1.LeaveWorkspaceRequest{
		WorkspaceId: "workspace-1",
	}); err != nil {
		t.Fatal(err)
	}
	if service.leaveRequest.WorkspaceId != "workspace-1" {
		t.Fatalf("unexpected leave request: %+v", service.leaveRequest)
	}
}
