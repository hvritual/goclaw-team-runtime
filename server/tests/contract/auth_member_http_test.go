package contract_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	authv1 "github.com/multica-ai/multica/server/gen/go/auth/v1"
	"github.com/multica-ai/multica/server/internal/modules/auth"
)

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
	service := &successfulMemberService{}
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
