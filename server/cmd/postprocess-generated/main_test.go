package main

import (
	"strings"
	"testing"
)

func TestApplyGoHTTPStatus(t *testing.T) {
	input := []byte(`package authv1

import (
	context "context"
	http "github.com/go-kratos/kratos/v3/transport/http"
)

// This is a compile-time assertion to ensure compatibility.
func _MemberService_DeleteMember0_HTTP_Handler(srv MemberServiceHTTPServer) func(ctx http.Context) error {
	return func(ctx http.Context) error {
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		reply := out.(*DeleteMemberResponse)
		return ctx.Result(200, reply)
	}
}

func (c *MemberServiceHTTPClientImpl) DeleteMember(ctx context.Context, in *DeleteMemberRequest, opts ...http.CallOption) (*DeleteMemberResponse, error) {
	var out DeleteMemberResponse
	err := c.cc.Invoke(ctx, "DELETE", path, nil, &out, opts...)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
`)
	output, err := applyGoHTTPStatus(input, statusOverride{serviceName: "MemberService", methodName: "DeleteMember", status: 204})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	if !strings.Contains(text, "ctx.Response().WriteHeader(204)") || strings.Contains(text, "ctx.Result(200") {
		t.Fatalf("unexpected generated handler:\n%s", text)
	}
	if !strings.Contains(text, `httpbody "google.golang.org/genproto/googleapis/api/httpbody"`) {
		t.Fatalf("HTTP body import missing:\n%s", text)
	}
	if !strings.Contains(text, "nil, &httpbody.HttpBody{}, opts...)") || strings.Contains(text, "nil, &out, opts...)") {
		t.Fatalf("unexpected generated client:\n%s", text)
	}
}

func TestApplyOpenAPIStatuses(t *testing.T) {
	input := []byte(`paths:
    /members/{id}:
        delete:
            operationId: MemberService_DeleteMember
            responses:
                "200":
                    description: OK
                    content:
                        application/json:
                            schema:
                                $ref: '#/components/schemas/DeleteMemberResponse'
                default:
                    description: Default error response
`)
	output, err := applyOpenAPIStatuses(input, []statusOverride{{
		serviceName: "MemberService",
		methodName:  "DeleteMember",
		status:      204,
	}})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	if !strings.Contains(text, `"204":`) || !strings.Contains(text, "description: No Content") || strings.Contains(text, `"200":`) {
		t.Fatalf("unexpected OpenAPI response:\n%s", text)
	}
	if !strings.Contains(text, "default:") {
		t.Fatalf("default response was removed:\n%s", text)
	}
}

func TestApplyGoHTTPClientPathVariables(t *testing.T) {
	input := []byte(`func call() {
	pattern := "/api/workspaces/{workspace_id}/members/{member_id}"
	path := http.BuildPath(pattern, in)
}
`)
	output := string(applyGoHTTPClientPathVariables(input))
	if !strings.Contains(output, `pattern := "/api/workspaces/{workspaceId}/members/{memberId}"`) {
		t.Fatalf("unexpected client path variables:\n%s", output)
	}
}

func TestApplyGoHTTPClientResponseBodies(t *testing.T) {
	input := []byte(`func (c *MemberServiceHTTPClientImpl) ListMembers(ctx context.Context, in *ListMembersRequest, opts ...http.CallOption) (*ListMembersResponse, error) {
	opts = append([]http.CallOption{
		http.Accept("application/protojson"),
	}, opts...)
	err := c.cc.Invoke(ctx, "GET", path, nil, &out.Members, opts...)
	return &out, err
}
`)
	output := string(applyGoHTTPClientResponseBodies(input))
	if !strings.Contains(output, `http.Accept("application/json")`) || strings.Contains(output, `http.Accept("application/protojson")`) {
		t.Fatalf("unexpected response-body client:\n%s", output)
	}
}

func TestApplyGoHTTPResponseBodyEmptyArray(t *testing.T) {
	input := []byte(`func _MemberService_ListWorkspaceInvitations0_HTTP_Handler(srv MemberServiceHTTPServer) func(ctx http.Context) error {
	return func(ctx http.Context) error {
		reply := out.(*ListWorkspaceInvitationsResponse)
		return ctx.Result(200, reply.Invitations)
	}
}
`)
	output, err := applyGoHTTPResponseBodyEmptyArray(input, responseBodyOverride{
		serviceName:     "MemberService",
		methodName:      "ListWorkspaceInvitations",
		schemaName:      "Invitation",
		responseGoField: "Invitations",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	if !strings.Contains(text, "reply.Invitations = []*Invitation{}") {
		t.Fatalf("empty response_body array is not normalized:\n%s", text)
	}
}

func TestApplyGoResponseBodyOptionalFields(t *testing.T) {
	input := []byte(`type Member struct {
	Id        string  ` + "`" + `json:"id,omitempty"` + "`" + `
	AvatarUrl *string ` + "`" + `json:"avatar_url,omitempty"` + "`" + `
}

`)
	output, err := applyGoResponseBodyOptionalFields(input, responseBodyOverride{
		schemaName:         "Member",
		optionalJSONFields: []string{"avatar_url"},
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	if !strings.Contains(text, `json:"avatar_url"`) || strings.Contains(text, `json:"avatar_url,omitempty"`) {
		t.Fatalf("optional response-body field still omits null:\n%s", text)
	}
	if !strings.Contains(text, `json:"id,omitempty"`) {
		t.Fatalf("unrelated field tag changed:\n%s", text)
	}
}

func TestApplyGoContractResponseBodyField(t *testing.T) {
	input := []byte(`type Member_ListWorkspaceInvitationsResponse struct {
	Invitations []Member_Invitation ` + "`" + `json:"invitations,omitempty"` + "`" + `
}
`)
	output, err := applyGoContractResponseBodyField(input, responseBodyOverride{
		serviceName:       "MemberService",
		responseName:      "ListWorkspaceInvitationsResponse",
		responseJSONField: "invitations",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	if !strings.Contains(text, `json:"invitations"`) || strings.Contains(text, `json:"invitations,omitempty"`) {
		t.Fatalf("response_body contract still omits an empty array:\n%s", text)
	}
}

func TestApplyOpenAPIResponseBodies(t *testing.T) {
	input := []byte(`paths:
    /members:
        get:
            operationId: MemberService_ListMembers
            responses:
                "200":
                    description: OK
                    content:
                        application/json:
                            schema:
                                $ref: '#/components/schemas/ListMembersResponse'
                default:
                    description: Default error response
components:
    schemas:
        Member:
            type: object
            properties:
                avatarUrl:
                    type: string
`)
	output, err := applyOpenAPIResponseBodies(input, []responseBodyOverride{{
		serviceName:           "MemberService",
		methodName:            "ListMembers",
		schemaName:            "Member",
		optionalOpenAPIFields: []string{"avatarUrl"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	if !strings.Contains(text, "type: array") || !strings.Contains(text, "#/components/schemas/Member'") {
		t.Fatalf("unexpected OpenAPI response body:\n%s", text)
	}
	if strings.Contains(text, "#/components/schemas/ListMembersResponse'") || !strings.Contains(text, "default:") {
		t.Fatalf("wrapper schema or default response mismatch:\n%s", text)
	}
	if !strings.Contains(text, "avatarUrl:\n                    type: string\n                    nullable: true") {
		t.Fatalf("optional response-body field is not nullable:\n%s", text)
	}
}
