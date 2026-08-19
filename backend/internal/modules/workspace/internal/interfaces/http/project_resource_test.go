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
	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
)

type recordingProjectResourceService struct {
	listWorkspaceID  string
	listProjectID    string
	includeArchived  bool
	createWorkspace  string
	createProject    string
	createKey        string
	createRequest    contract.CreateProjectResourceRequest
	updateWorkspace  string
	updateProject    string
	updateResource   string
	updateRequest    contract.UpdateProjectResourceRequest
	archiveWorkspace string
	archiveProject   string
	archiveResource  string
	archiveRevision  int64
	createCalls      int
	updateCalls      int
	archiveCalls     int
	listErr          error
	createErr        error
	updateErr        error
	archiveErr       error
}

func (s *recordingProjectResourceService) ListProjectResources(_ context.Context, workspaceID, projectID string, includeArchived bool) (contract.ProjectResourceList, error) {
	s.listWorkspaceID, s.listProjectID, s.includeArchived = workspaceID, projectID, includeArchived
	if s.listErr != nil {
		return contract.ProjectResourceList{}, s.listErr
	}
	return contract.ProjectResourceList{Resources: []contract.ProjectResource{}, Total: 0, Revision: 4}, nil
}

func (s *recordingProjectResourceService) CreateProjectResource(_ context.Context, workspaceID, projectID, key string, request contract.CreateProjectResourceRequest) (contract.ProjectResource, error) {
	s.createCalls++
	s.createWorkspace, s.createProject, s.createKey, s.createRequest = workspaceID, projectID, key, request
	if s.createErr != nil {
		return contract.ProjectResource{}, s.createErr
	}
	return projectResourceHTTPFixture("resource-created", 5), nil
}

func (s *recordingProjectResourceService) UpdateProjectResource(_ context.Context, workspaceID, projectID, resourceID string, request contract.UpdateProjectResourceRequest) (contract.ProjectResource, error) {
	s.updateCalls++
	s.updateWorkspace, s.updateProject, s.updateResource, s.updateRequest = workspaceID, projectID, resourceID, request
	if s.updateErr != nil {
		return contract.ProjectResource{}, s.updateErr
	}
	return projectResourceHTTPFixture(resourceID, request.ExpectedRevision+1), nil
}

func (s *recordingProjectResourceService) ArchiveProjectResource(_ context.Context, workspaceID, projectID, resourceID string, expectedRevision int64) error {
	s.archiveCalls++
	s.archiveWorkspace, s.archiveProject, s.archiveResource, s.archiveRevision = workspaceID, projectID, resourceID, expectedRevision
	return s.archiveErr
}

func TestProjectResourceRoutesUseTrustedIdentityAndExactContracts(t *testing.T) {
	service := &recordingProjectResourceService{}
	server := newProjectResourceHTTPServer(service, func(*http.Request) error { return nil })

	list := newProjectResourceRequest(http.MethodGet, "/api/projects/project-1/resources?include_archived=true", "")
	list.Header.Set("X-Workspace-ID", "forged-workspace")
	listResponse := httptest.NewRecorder()
	server.ServeHTTP(listResponse, list)
	if listResponse.Code != http.StatusOK || service.listWorkspaceID != "workspace-trusted" || service.listProjectID != "project-1" || !service.includeArchived {
		t.Fatalf("list = %d %s workspace=%q project=%q archived=%t", listResponse.Code, listResponse.Body.String(), service.listWorkspaceID, service.listProjectID, service.includeArchived)
	}
	if strings.TrimSpace(listResponse.Body.String()) != `{"resources":[],"total":0,"revision":4}` {
		t.Fatalf("list body = %s", listResponse.Body.String())
	}

	create := newProjectResourceRequest(http.MethodPost, "/api/projects/project-1/resources", `{"resource_type":"github_repo","resource_ref":{"url":"https://github.com/acme/repo"},"label":"Repo"}`)
	create.Header.Set("Idempotency-Key", "resource-create-1")
	createResponse := httptest.NewRecorder()
	server.ServeHTTP(createResponse, create)
	if createResponse.Code != http.StatusCreated || service.createCalls != 1 || service.createWorkspace != "workspace-trusted" || service.createProject != "project-1" || service.createKey != "resource-create-1" || service.createRequest.ResourceType != "github_repo" {
		t.Fatalf("create = %d %s calls=%d workspace=%q project=%q key=%q request=%+v", createResponse.Code, createResponse.Body.String(), service.createCalls, service.createWorkspace, service.createProject, service.createKey, service.createRequest)
	}

	update := newProjectResourceRequest(http.MethodPut, "/api/projects/project-1/resources/resource-1", `{"action":"reorder","expected_revision":5,"before_resource_id":"resource-2"}`)
	updateResponse := httptest.NewRecorder()
	server.ServeHTTP(updateResponse, update)
	if updateResponse.Code != http.StatusOK || service.updateCalls != 1 || service.updateWorkspace != "workspace-trusted" || service.updateProject != "project-1" || service.updateResource != "resource-1" || service.updateRequest.ExpectedRevision != 5 || service.updateRequest.BeforeResourceID == nil || *service.updateRequest.BeforeResourceID != "resource-2" {
		t.Fatalf("update = %d %s calls=%d request=%+v", updateResponse.Code, updateResponse.Body.String(), service.updateCalls, service.updateRequest)
	}

	archive := newProjectResourceRequest(http.MethodDelete, "/api/projects/project-1/resources/resource-1", `{"expected_revision":6}`)
	archiveResponse := httptest.NewRecorder()
	server.ServeHTTP(archiveResponse, archive)
	if archiveResponse.Code != http.StatusNoContent || service.archiveCalls != 1 || service.archiveWorkspace != "workspace-trusted" || service.archiveProject != "project-1" || service.archiveResource != "resource-1" || service.archiveRevision != 6 {
		t.Fatalf("archive = %d %s calls=%d revision=%d", archiveResponse.Code, archiveResponse.Body.String(), service.archiveCalls, service.archiveRevision)
	}
}

func TestProjectResourceHTTPRejectsMissingMutationGuardsAndInvalidQuery(t *testing.T) {
	service := &recordingProjectResourceService{}
	server := newProjectResourceHTTPServer(service, func(*http.Request) error { return nil })

	missingKey := newProjectResourceRequest(http.MethodPost, "/api/projects/project-1/resources", `{"resource_type":"url","resource_ref":{"url":"https://example.com"}}`)
	missingKeyResponse := httptest.NewRecorder()
	server.ServeHTTP(missingKeyResponse, missingKey)
	if missingKeyResponse.Code != http.StatusBadRequest || service.createCalls != 0 || strings.TrimSpace(missingKeyResponse.Body.String()) != `{"error":"idempotency key is required"}` {
		t.Fatalf("missing key = %d %s calls=%d", missingKeyResponse.Code, missingKeyResponse.Body.String(), service.createCalls)
	}

	invalidQuery := newProjectResourceRequest(http.MethodGet, "/api/projects/project-1/resources?include_archived=maybe", "")
	invalidQueryResponse := httptest.NewRecorder()
	server.ServeHTTP(invalidQueryResponse, invalidQuery)
	if invalidQueryResponse.Code != http.StatusBadRequest || strings.TrimSpace(invalidQueryResponse.Body.String()) != `{"error":"invalid include_archived"}` {
		t.Fatalf("invalid query = %d %s", invalidQueryResponse.Code, invalidQueryResponse.Body.String())
	}

	csrfService := &recordingProjectResourceService{}
	csrfServer := newProjectResourceHTTPServer(csrfService, func(*http.Request) error { return errors.New("bad csrf") })
	csrfRequest := newProjectResourceRequest(http.MethodDelete, "/api/projects/project-1/resources/resource-1", `{"expected_revision":1}`)
	csrfResponse := httptest.NewRecorder()
	csrfServer.ServeHTTP(csrfResponse, csrfRequest)
	if csrfResponse.Code != http.StatusForbidden || csrfService.archiveCalls != 0 {
		t.Fatalf("csrf = %d %s calls=%d", csrfResponse.Code, csrfResponse.Body.String(), csrfService.archiveCalls)
	}
}

func TestProjectResourceHTTPMapsRevisionAndResourceConflicts(t *testing.T) {
	revisionService := &recordingProjectResourceService{updateErr: contract.RevisionConflictError{CurrentRevision: 9}}
	revisionServer := newProjectResourceHTTPServer(revisionService, func(*http.Request) error { return nil })
	revisionRequest := newProjectResourceRequest(http.MethodPut, "/api/projects/project-1/resources/resource-1", `{"action":"update","expected_revision":2,"label":"New"}`)
	revisionResponse := httptest.NewRecorder()
	revisionServer.ServeHTTP(revisionResponse, revisionRequest)
	if revisionResponse.Code != http.StatusConflict || strings.TrimSpace(revisionResponse.Body.String()) != `{"code":"revision_conflict","current_revision":9,"error":"revision conflict"}` {
		t.Fatalf("revision conflict = %d %s", revisionResponse.Code, revisionResponse.Body.String())
	}

	duplicateService := &recordingProjectResourceService{createErr: application.ErrProjectResourceConflict}
	duplicateServer := newProjectResourceHTTPServer(duplicateService, func(*http.Request) error { return nil })
	duplicateRequest := newProjectResourceRequest(http.MethodPost, "/api/projects/project-1/resources", `{"resource_type":"url","resource_ref":{"url":"https://example.com"}}`)
	duplicateRequest.Header.Set("Idempotency-Key", "duplicate-1")
	duplicateResponse := httptest.NewRecorder()
	duplicateServer.ServeHTTP(duplicateResponse, duplicateRequest)
	if duplicateResponse.Code != http.StatusConflict || strings.TrimSpace(duplicateResponse.Body.String()) != `{"error":"Project Resource conflict"}` {
		t.Fatalf("resource conflict = %d %s", duplicateResponse.Code, duplicateResponse.Body.String())
	}
}

func newProjectResourceHTTPServer(service contract.ProjectResourceService, mutation func(*http.Request) error) *kratoshttp.Server {
	server := kratoshttp.NewServer()
	NewProjectResourceHandler(
		service,
		func(*http.Request) (contract.WorkspaceHTTPIdentity, error) {
			return contract.WorkspaceHTTPIdentity{WorkspaceID: "workspace-trusted", ActorType: "member", ActorID: "member-1"}, nil
		},
		func(*http.Request) (string, error) { return "user-1", nil },
		mutation,
	).Register(server)
	return server
}

func newProjectResourceRequest(method, path, body string) *http.Request {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("X-Workspace-Slug", "workspace-trusted")
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	return request
}

func projectResourceHTTPFixture(id string, revision int64) contract.ProjectResource {
	return contract.ProjectResource{
		ID: id, WorkspaceID: "workspace-trusted", ProjectID: "project-1", ResourceType: "url",
		ResourceRef: contract.ProjectResourceRef{URL: "https://example.com"}, Position: 0, Status: "active", Revision: revision,
		Connection: contract.ProjectResourceConnection{State: "unavailable"}, CreatedAt: "2026-08-19T00:00:00Z", CreatedBy: "member-1", UpdatedAt: "2026-08-19T00:00:00Z", UpdatedBy: "member-1",
	}
}
