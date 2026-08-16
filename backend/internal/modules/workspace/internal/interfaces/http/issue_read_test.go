package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

type attachmentConflictIssueService struct{ contract.IssueMutationService }

type recordingIssueListService struct {
	contract.IssueMutationService
	request contract.ListIssuesRequest
}

func (s *recordingIssueListService) ListIssues(_ context.Context, request contract.ListIssuesRequest) (contract.ListIssuesResponse, error) {
	s.request = request
	return contract.ListIssuesResponse{Issues: []contract.Issue{{
		Id: "issue-blocked", WorkspaceId: "workspace-1", Number: 1, Identifier: "ONE-1",
		Title: "Blocked issue", Status: "blocked", Priority: "high", CreatorType: "member", CreatorId: "member-1",
	}}}, nil
}

func (attachmentConflictIssueService) GetIssue(context.Context, contract.GetIssueRequest) (contract.GetIssueResponse, error) {
	return contract.GetIssueResponse{Issue: &contract.Issue{Id: "issue-1", WorkspaceId: "workspace-1"}}, nil
}

func (attachmentConflictIssueService) UpdateIssue(context.Context, contract.UpdateIssueRequest) (contract.UpdateIssueResponse, error) {
	return contract.UpdateIssueResponse{}, contract.ErrIssueAttachmentConflict
}

func TestIssueAttachmentConflictReturnsCanonical409(t *testing.T) {
	server := kratoshttp.NewServer()
	handler := NewIssueReadHandler(
		attachmentConflictIssueService{}, nil,
		func(*http.Request) (contract.WorkspaceHTTPIdentity, error) {
			return contract.WorkspaceHTTPIdentity{WorkspaceID: "workspace-1", ActorType: "member", ActorID: "member-1"}, nil
		},
		func(*http.Request) (string, error) { return "user-1", nil },
		func(*http.Request) error { return nil }, true, true,
	)
	handler.Register(server)
	request := httptest.NewRequest(http.MethodPut, "/api/issues/issue-1", strings.NewReader(`{"attachment_ids":[]}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Workspace-Slug", "acme")
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusConflict || strings.TrimSpace(response.Body.String()) != `{"error":"issue attachments changed"}` {
		t.Fatalf("attachment conflict = %d %s", response.Code, response.Body.String())
	}
}

func TestIssueListPushesSingularStatusIntoServiceQuery(t *testing.T) {
	service := &recordingIssueListService{}
	server := kratoshttp.NewServer()
	handler := NewIssueReadHandler(
		service, nil,
		func(*http.Request) (contract.WorkspaceHTTPIdentity, error) {
			return contract.WorkspaceHTTPIdentity{WorkspaceID: "workspace-1", ActorType: "member", ActorID: "member-1"}, nil
		},
		nil, nil, false, false,
	)
	handler.Register(server)

	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/issues?status=blocked&limit=50", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("list = %d %s", response.Code, response.Body.String())
	}
	if service.request.Status != "blocked" {
		t.Fatalf("service status = %q, want blocked", service.request.Status)
	}
}

func TestSortIssuesUsesCanonicalRanksNullsAndTieBreakers(t *testing.T) {
	status := []publicIssue{{ID: "blocked", Status: "blocked"}, {ID: "done", Status: "done"}, {ID: "todo", Status: "todo"}, {ID: "backlog", Status: "backlog"}, {ID: "cancelled", Status: "cancelled"}, {ID: "review", Status: "in_review"}, {ID: "progress", Status: "in_progress"}}
	sortIssues(status, "status", "asc")
	assertIssueOrder(t, status, "backlog", "todo", "progress", "review", "done", "blocked", "cancelled")
	priority := []publicIssue{{ID: "none", Priority: "none"}, {ID: "low", Priority: "low"}, {ID: "urgent", Priority: "urgent"}, {ID: "medium", Priority: "medium"}, {ID: "high", Priority: "high"}}
	sortIssues(priority, "priority", "asc")
	assertIssueOrder(t, priority, "urgent", "high", "medium", "low", "none")
	a, b := "2026-01-01", "2026-01-02"
	dates := []publicIssue{{ID: "nil", CreatedAt: "2026-01-03"}, {ID: "a", StartDate: &a, CreatedAt: "2026-01-01"}, {ID: "b", StartDate: &b, CreatedAt: "2026-01-02"}}
	sortIssues(dates, "start_date", "asc")
	assertIssueOrder(t, dates, "a", "b", "nil")
	sortIssues(dates, "start_date", "desc")
	assertIssueOrder(t, dates, "b", "a", "nil")
	ties := []publicIssue{{ID: "a", Title: "same", CreatedAt: "2026-01-02"}, {ID: "c", Title: "same", CreatedAt: "2026-01-02"}, {ID: "b", Title: "same", CreatedAt: "2026-01-03"}}
	sortIssues(ties, "title", "asc")
	assertIssueOrder(t, ties, "b", "c", "a")
	positions := []publicIssue{{ID: "two", Position: 2}, {ID: "one", Position: 1}}
	sortIssues(positions, "position", "desc")
	assertIssueOrder(t, positions, "one", "two")
}

func assertIssueOrder(t *testing.T, issues []publicIssue, ids ...string) {
	t.Helper()
	for index, id := range ids {
		if issues[index].ID != id {
			t.Fatalf("index %d = %s, want %s", index, issues[index].ID, id)
		}
	}
}
