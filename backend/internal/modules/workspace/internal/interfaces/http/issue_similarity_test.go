package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

type recordingIssueSimilarityService struct {
	request  contract.CheckIssueSimilarityRequest
	response contract.CheckIssueSimilarityResponse
	err      error
	calls    int
}

func (s *recordingIssueSimilarityService) CheckIssueSimilarity(_ context.Context, request contract.CheckIssueSimilarityRequest) (contract.CheckIssueSimilarityResponse, error) {
	s.calls++
	s.request = request
	return s.response, s.err
}

type recordingIssueSimilarityIssueService struct {
	contract.IssueMutationService
	request  contract.GetIssueRequest
	response contract.GetIssueResponse
	err      error
	calls    int
}

func (s *recordingIssueSimilarityIssueService) GetIssue(_ context.Context, request contract.GetIssueRequest) (contract.GetIssueResponse, error) {
	s.calls++
	s.request = request
	return s.response, s.err
}

func TestIssueSimilarityCheckRoutePassesAuthorizedRequestAndResponse(t *testing.T) {
	service := &recordingIssueSimilarityService{response: issueSimilarityResponse()}
	response := serveIssueSimilarity(t, service, &recordingIssueSimilarityIssueService{}, http.MethodPost, "/api/issues/similarity/check", `{"title":"Alpha beta","description":"Details","project_id":"project-1","include_closed":true}`)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
	}
	if service.calls != 1 || service.request.WorkspaceID != "workspace-1" || service.request.Title != "Alpha beta" || service.request.Description == nil || *service.request.Description != "Details" || service.request.ProjectID == nil || *service.request.ProjectID != "project-1" || !service.request.IncludeClosed {
		t.Fatalf("request = %+v calls=%d", service.request, service.calls)
	}
	body := strings.TrimSpace(response.Body.String())
	for _, fragment := range []string{`"ranking_version":"lexical-v1"`, `"candidates":[{`, `"id":"issue-1"`, `"component_scores":{"exact_normalized_title":1}`, `"same_project":true`, `"closed":false`, `"detector_available":true`} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("body missing %s: %s", fragment, body)
		}
	}
}

func TestIssueSimilarityExistingRouteReadsCanonicalIssueAndExcludesSelf(t *testing.T) {
	description, projectID := "Canonical description", "project-1"
	issues := &recordingIssueSimilarityIssueService{response: contract.GetIssueResponse{Issue: &contract.Issue{
		Id: "issue-1", WorkspaceId: "workspace-1", Title: "Canonical title", Description: &description, ProjectId: &projectID,
	}}}
	service := &recordingIssueSimilarityService{response: issueSimilarityResponse()}
	response := serveIssueSimilarity(t, service, issues, http.MethodPost, "/api/issues/issue-1/similarity/check", "")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
	}
	if issues.calls != 1 || issues.request.WorkspaceId != "workspace-1" || issues.request.IssueId != "issue-1" {
		t.Fatalf("canonical issue request = %+v calls=%d", issues.request, issues.calls)
	}
	if service.calls != 1 || service.request.WorkspaceID != "workspace-1" || service.request.Title != "Canonical title" || service.request.Description == nil || *service.request.Description != description || service.request.ProjectID == nil || *service.request.ProjectID != projectID || service.request.ExcludeIssueID != "issue-1" {
		t.Fatalf("similarity request = %+v calls=%d", service.request, service.calls)
	}
}

func TestIssueSimilarityRoutesRejectMalformedRequestsAndMapFailures(t *testing.T) {
	malformed := &recordingIssueSimilarityService{}
	response := serveIssueSimilarity(t, malformed, &recordingIssueSimilarityIssueService{}, http.MethodPost, "/api/issues/similarity/check", `{"unknown":true}`)
	if response.Code != http.StatusBadRequest || malformed.calls != 0 {
		t.Fatalf("malformed = %d calls=%d body=%s", response.Code, malformed.calls, response.Body.String())
	}

	denied := &recordingIssueSimilarityService{err: contract.ErrWorkspacePermissionDenied}
	response = serveIssueSimilarity(t, denied, &recordingIssueSimilarityIssueService{}, http.MethodPost, "/api/issues/similarity/check", `{"title":"Alpha"}`)
	if response.Code != http.StatusForbidden || denied.calls != 1 {
		t.Fatalf("denied = %d calls=%d body=%s", response.Code, denied.calls, response.Body.String())
	}

	notFound := &recordingIssueSimilarityIssueService{err: contract.ErrIssueNotFound}
	response = serveIssueSimilarity(t, &recordingIssueSimilarityService{}, notFound, http.MethodPost, "/api/issues/missing/similarity/check", "")
	if response.Code != http.StatusNotFound || notFound.calls != 1 {
		t.Fatalf("missing = %d calls=%d body=%s", response.Code, notFound.calls, response.Body.String())
	}
}

func TestIssueSimilarityRouteRejectsAuthenticationBeforeService(t *testing.T) {
	service := &recordingIssueSimilarityService{}
	server := kratoshttp.NewServer()
	NewIssueSimilarityHandler(service, &recordingIssueSimilarityIssueService{},
		func(*http.Request) (contract.WorkspaceHTTPIdentity, error) {
			return contract.WorkspaceHTTPIdentity{}, errors.New("identity must not be called")
		},
		func(*http.Request) (string, error) { return "", errors.New("bad session") },
	).Register(server)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/issues/similarity/check", strings.NewReader(`{"title":"Alpha"}`))
	request.Header.Set("X-Workspace-Slug", "workspace-1")
	server.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized || service.calls != 0 {
		t.Fatalf("authentication = %d calls=%d body=%s", response.Code, service.calls, response.Body.String())
	}
}

func serveIssueSimilarity(t *testing.T, service *recordingIssueSimilarityService, issues *recordingIssueSimilarityIssueService, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	server := kratoshttp.NewServer()
	NewIssueSimilarityHandler(service, issues,
		func(*http.Request) (contract.WorkspaceHTTPIdentity, error) {
			return contract.WorkspaceHTTPIdentity{WorkspaceID: "workspace-1", ActorType: "member", ActorID: "member-1"}, nil
		},
		func(*http.Request) (string, error) { return "user-1", nil },
	).Register(server)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Workspace-Slug", "workspace-1")
	server.ServeHTTP(response, request)
	return response
}

func issueSimilarityResponse() contract.CheckIssueSimilarityResponse {
	return contract.CheckIssueSimilarityResponse{
		RankingVersion: "lexical-v1", DetectorAvailable: true,
		Candidates: []contract.IssueSimilarityCandidate{{
			Issue: contract.Issue{Id: "issue-1", WorkspaceId: "workspace-1", Identifier: "WSP-1", Title: "Alpha beta", Status: "todo", Priority: "none", CreatorType: "member", CreatorId: "member-1"},
			Score: 100, ComponentScores: map[string]float64{"exact_normalized_title": 1}, SameProject: true,
		}},
	}
}

var _ contract.IssueSimilarityService = (*recordingIssueSimilarityService)(nil)
