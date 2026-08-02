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
