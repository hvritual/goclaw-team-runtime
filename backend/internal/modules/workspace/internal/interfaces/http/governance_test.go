package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

func TestGovernanceDiagnosticsAreOwnerAdminOnlyAndRedacted(t *testing.T) {
	for _, role := range []string{"owner", "admin", "member"} {
		t.Run(role, func(t *testing.T) {
			server := kratoshttp.NewServer()
			handler := NewGovernanceHandler(
				governanceDiagnosticsStub{value: contract.OutboxDiagnostics{
					ReadyCount: 2, OldestReadyAge: 16 * time.Minute, InflightCount: 1,
					RetryWaitCount: 3, DeadLetterCount: 4, SchemaVersion: "000009_workspace_governance", DispatcherRunning: true,
				}},
				func(*http.Request) (contract.WorkspaceHTTPIdentity, error) {
					return contract.WorkspaceHTTPIdentity{WorkspaceID: "workspace-1", ActorType: "member", ActorID: "member-1"}, nil
				},
				func(*http.Request) (string, error) { return "user-1", nil },
				governanceMembershipsStub{role: role},
			)
			handler.Register(server)
			request := httptest.NewRequest(http.MethodGet, "/api/operations/governance", nil)
			request.Header.Set("X-Workspace-ID", "workspace-1")
			response := httptest.NewRecorder()
			server.ServeHTTP(response, request)
			wantStatus := http.StatusOK
			if role == "member" {
				wantStatus = http.StatusForbidden
			}
			if response.Code != wantStatus {
				t.Fatalf("status = %d, want %d: %s", response.Code, wantStatus, response.Body.String())
			}
			if role != "member" {
				body := response.Body.String()
				for _, forbidden := range []string{"payload", "audit", "secret-value"} {
					if strings.Contains(body, forbidden) {
						t.Fatalf("diagnostics leaked %q: %s", forbidden, body)
					}
				}
				for _, required := range []string{`"ready_count":2`, `"dead_letter_count":4`, `"degraded":true`, `"dispatcher_running":true`} {
					if !strings.Contains(body, required) {
						t.Fatalf("diagnostics missing %s: %s", required, body)
					}
				}
			}
		})
	}
}

func TestGovernanceDiagnosticsFailClosedOutsideWorkspaceAndOnDatabaseError(t *testing.T) {
	tests := []struct {
		name        string
		memberships governanceMembershipsStub
		service     governanceDiagnosticsStub
		wantStatus  int
	}{
		{name: "outside workspace", memberships: governanceMembershipsStub{}, wantStatus: http.StatusNotFound},
		{name: "database unavailable", memberships: governanceMembershipsStub{role: "owner"}, service: governanceDiagnosticsStub{err: errors.New("database closed")}, wantStatus: http.StatusServiceUnavailable},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := kratoshttp.NewServer()
			handler := NewGovernanceHandler(test.service,
				func(*http.Request) (contract.WorkspaceHTTPIdentity, error) {
					return contract.WorkspaceHTTPIdentity{WorkspaceID: "workspace-1", ActorType: "member", ActorID: "member-1"}, nil
				},
				func(*http.Request) (string, error) { return "user-1", nil }, test.memberships)
			handler.Register(server)
			request := httptest.NewRequest(http.MethodGet, "/api/operations/governance", nil)
			request.Header.Set("X-Workspace-ID", "workspace-1")
			response := httptest.NewRecorder()
			server.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d: %s", response.Code, test.wantStatus, response.Body.String())
			}
		})
	}
}

type governanceDiagnosticsStub struct {
	value contract.OutboxDiagnostics
	err   error
}

func (s governanceDiagnosticsStub) ReadGovernanceDiagnostics(context.Context, string) (contract.OutboxDiagnostics, error) {
	return s.value, s.err
}

type governanceMembershipsStub struct{ role string }

func (s governanceMembershipsStub) ListForUser(context.Context, string) ([]contract.WorkspaceMembership, error) {
	return nil, nil
}
func (s governanceMembershipsStub) FindForUserAndWorkspace(context.Context, string, string) (contract.WorkspaceMembership, bool, error) {
	if s.role == "" {
		return contract.WorkspaceMembership{}, false, nil
	}
	return contract.WorkspaceMembership{MemberID: "member-1", UserID: "user-1", WorkspaceID: "workspace-1", Role: s.role}, true, nil
}
func (s governanceMembershipsStub) FindByMemberAndWorkspace(context.Context, string, string) (contract.WorkspaceMembership, bool, error) {
	if s.role == "" {
		return contract.WorkspaceMembership{}, false, nil
	}
	return contract.WorkspaceMembership{MemberID: "member-1", UserID: "user-1", WorkspaceID: "workspace-1", Role: s.role}, true, nil
}
