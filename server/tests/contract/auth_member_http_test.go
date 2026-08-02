package contract_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-kratos/kratos/v3/middleware"
	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	authv1 "github.com/multica-ai/multica/server/gen/go/auth/v1"
	"github.com/multica-ai/multica/server/internal/modules/auth"
	"github.com/multica-ai/multica/server/internal/modules/auth/contract"
)

type memberListJSONService struct{ successfulMemberService }

type emptyWorkspaceInvitationService struct{ successfulMemberService }

type forbiddenInvitationCreationService struct{ successfulMemberService }

type actorAwareInvitationCreationService struct{ successfulMemberService }

func (*forbiddenInvitationCreationService) AuthorizeCreateInvitation(context.Context, contract.Member_CreateInvitationRequest) error {
	return contract.ErrInsufficientWorkspaceRole
}

func (*actorAwareInvitationCreationService) AuthorizeCreateInvitation(ctx context.Context, _ contract.Member_CreateInvitationRequest) error {
	if _, ok := contract.MemberActor(ctx); !ok {
		return contract.ErrMemberActorRequired
	}
	return nil
}

func (*emptyWorkspaceInvitationService) ListWorkspaceInvitations(context.Context, contract.Member_ListWorkspaceInvitationsRequest) (contract.Member_ListWorkspaceInvitationsResponse, error) {
	return contract.Member_ListWorkspaceInvitationsResponse{Invitations: make([]contract.Member_Invitation, 0)}, nil
}

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

func TestAuthMemberHTTPInvitationListUsesTopLevelArray(t *testing.T) {
	extension := auth.NewMemberExtensionWithService(&successfulMemberService{})
	server := kratoshttp.NewServer()
	extension.RegisterHTTP(server)
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/workspaces/workspace-1/invitations", nil)
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d; body=%s", response.Code, response.Body.String())
	}
	var invitations []map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &invitations); err != nil {
		t.Fatalf("response is not an invitation array: %v; body=%s", err, response.Body.String())
	}
	if len(invitations) != 1 || invitations[0]["workspace_name"] != "Acme" {
		t.Fatalf("unexpected invitation array: %+v", invitations)
	}
	if inviteeUserID, exists := invitations[0]["invitee_user_id"]; !exists || inviteeUserID != nil {
		t.Fatalf("nil invitee_user_id must be present as JSON null: %+v", invitations[0])
	}
}

func TestAuthMemberHTTPInvitationListPreservesEmptyArray(t *testing.T) {
	extension := auth.NewMemberExtensionWithService(&emptyWorkspaceInvitationService{})
	server := kratoshttp.NewServer()
	extension.RegisterHTTP(server)
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/workspaces/workspace-1/invitations", nil)
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusOK || response.Body.String() != "[]" {
		t.Fatalf("status = %d; body=%q, want []", response.Code, response.Body.String())
	}
}

func TestAuthMemberHTTPCreateInvitationReturnsCreatedBody(t *testing.T) {
	service := &successfulMemberService{}
	extension := auth.NewMemberExtensionWithService(service)
	server := kratoshttp.NewServer()
	extension.RegisterHTTP(server)
	request := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/api/workspaces/workspace-1/members",
		strings.NewReader(`{"workspaceId":"body-workspace","email":"invitee@example.test","role":"admin"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusCreated, response.Body.String())
	}
	var invitation map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &invitation); err != nil {
		t.Fatalf("response is not an invitation object: %v; body=%s", err, response.Body.String())
	}
	if service.createInvitationRequest.WorkspaceId != "workspace-1" || service.createInvitationRequest.Role != "admin" {
		t.Fatalf("unexpected create request: %+v", service.createInvitationRequest)
	}
	if invitation["id"] != "invitation-created" || invitation["workspace_name"] != "Acme" {
		t.Fatalf("unexpected created invitation: %+v", invitation)
	}
	if inviteeUserID, exists := invitation["invitee_user_id"]; !exists || inviteeUserID != nil {
		t.Fatalf("nil invitee_user_id must be present as JSON null: %+v", invitation)
	}
}

func TestAuthMemberHTTPCreateInvitationAuthorizesBeforeBody(t *testing.T) {
	extension := auth.NewMemberExtensionWithService(&forbiddenInvitationCreationService{})
	server := kratoshttp.NewServer()
	extension.RegisterHTTP(server)
	request := httptest.NewRequestWithContext(
		t.Context(), http.MethodPost, "/api/workspaces/workspace-1/members", strings.NewReader("{"),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusForbidden, response.Body.String())
	}
}

func TestAuthMemberHTTPPreBodyAuthorizationReceivesMiddlewareContext(t *testing.T) {
	extension := auth.NewMemberExtensionWithService(&actorAwareInvitationCreationService{})
	server := kratoshttp.NewServer(kratoshttp.Middleware(func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, request any) (any, error) {
			return next(contract.WithMemberActor(ctx, "owner-user"), request)
		}
	}))
	extension.RegisterHTTP(server)
	request := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/api/workspaces/workspace-1/members",
		strings.NewReader(`{"email":"invitee@example.test","role":"admin"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusCreated, response.Body.String())
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
		{method: http.MethodDelete, path: "/api/workspaces/workspace-1/invitations/invitation-1"},
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

	if _, err := client.RevokeInvitation(t.Context(), &authv1.RevokeInvitationRequest{
		WorkspaceId:  "workspace-1",
		InvitationId: "invitation-1",
	}); err != nil {
		t.Fatal(err)
	}
	if service.revokeRequest.WorkspaceId != "workspace-1" || service.revokeRequest.InvitationId != "invitation-1" {
		t.Fatalf("unexpected revoke request: %+v", service.revokeRequest)
	}

	invitationList, err := client.ListWorkspaceInvitations(t.Context(), &authv1.ListWorkspaceInvitationsRequest{
		WorkspaceId: "workspace-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if service.listInvitationsRequest.WorkspaceId != "workspace-1" || len(invitationList.GetInvitations()) != 1 || invitationList.GetInvitations()[0].GetWorkspaceName() != "Acme" {
		t.Fatalf("unexpected generated-client invitation list request=%+v result=%+v", service.listInvitationsRequest, invitationList)
	}

	created, err := client.CreateInvitation(t.Context(), &authv1.CreateInvitationRequest{
		WorkspaceId: "workspace-1", Email: "invitee@example.test", Role: "admin",
	})
	if err != nil {
		t.Fatal(err)
	}
	if service.createInvitationRequest.Email != "invitee@example.test" || created.GetId() != "invitation-created" || created.GetRole() != "admin" {
		t.Fatalf("unexpected generated-client invitation create request=%+v result=%+v", service.createInvitationRequest, created)
	}
}
