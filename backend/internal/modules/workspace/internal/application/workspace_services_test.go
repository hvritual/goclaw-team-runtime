package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	projectDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/project"
	relationDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/relationship"
)

type accessAuthorizerStub struct {
	err         error
	permissions []string
}

func (s *accessAuthorizerStub) AuthorizeWorkspace(_ context.Context, _ string, permission string) error {
	s.permissions = append(s.permissions, permission)
	return s.err
}

type projectRepositoryStub struct {
	created       []projectDomain.Project
	project       projectDomain.Project
	projects      []projectDomain.Project
	searchResults []ProjectSearchResult
	searchTotal   int
	findErr       error
	createErr     error
	updateErr     error
	deleteErr     error
	findCalls     int
	createCalls   int
	listCalls     int
	searchCalls   int
	updateCalls   int
	deleteCalls   int
	searchQuery   ProjectSearchQuery
}

func (s *projectRepositoryStub) Create(_ context.Context, value projectDomain.Project) error {
	s.createCalls++
	s.created = append(s.created, value)
	return s.createErr
}

func (s *projectRepositoryStub) FindByID(context.Context, string, string) (projectDomain.Project, error) {
	s.findCalls++
	return s.project, s.findErr
}

func (s *projectRepositoryStub) List(context.Context, string, string) ([]projectDomain.Project, error) {
	s.listCalls++
	return append([]projectDomain.Project(nil), s.projects...), nil
}

func (s *projectRepositoryStub) Search(_ context.Context, query ProjectSearchQuery) ([]ProjectSearchResult, int, error) {
	s.searchCalls++
	s.searchQuery = query
	return append([]ProjectSearchResult(nil), s.searchResults...), s.searchTotal, nil
}

func (s *projectRepositoryStub) Update(_ context.Context, value projectDomain.Project) error {
	s.updateCalls++
	s.project = value
	return s.updateErr
}

func (s *projectRepositoryStub) DeleteWithDependents(context.Context, string, string, time.Time) error {
	s.deleteCalls++
	return s.deleteErr
}

type actorReaderStub struct {
	belongs bool
	err     error
	calls   int
}

func (s *actorReaderStub) ActorBelongsToWorkspace(context.Context, string, string, string) (bool, error) {
	s.calls++
	return s.belongs, s.err
}

type relationRepositoryStub struct {
	values      []relationDomain.Relation
	putCalls    int
	listCalls   int
	deleteCalls int
}

func (s *relationRepositoryStub) Put(context.Context, relationDomain.Relation) error {
	s.putCalls++
	return nil
}
func (s *relationRepositoryStub) List(context.Context, string, string) ([]relationDomain.Relation, error) {
	s.listCalls++
	return append([]relationDomain.Relation(nil), s.values...), nil
}
func (s *relationRepositoryStub) Delete(context.Context, string, string, string, string) error {
	s.deleteCalls++
	return nil
}

func TestProjectUseCasePreservesLegacyCreateRules(t *testing.T) {
	repository := &projectRepositoryStub{}
	authorizer := &accessAuthorizerStub{}
	now := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)
	service, err := NewProjectUseCase(repository, authorizer, func(context.Context) (string, error) {
		return "project-1", nil
	}, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	response, err := service.CreateProject(context.Background(), contract.CreateProjectRequest{
		WorkspaceId: "workspace-1", Name: "  Delivery  ", AssetIds: []string{"asset-1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.Project == nil || response.Project.Status != "planned" || response.Project.Name != "Delivery" {
		t.Fatalf("unexpected Project response: %+v", response.Project)
	}
	if repository.createCalls != 1 || len(authorizer.permissions) != 1 || authorizer.permissions[0] != PermissionProjectCreate {
		t.Fatalf("create/auth calls = %d/%v", repository.createCalls, authorizer.permissions)
	}

	_, err = service.CreateProject(context.Background(), contract.CreateProjectRequest{
		WorkspaceId: "workspace-1", Name: "Delivery", Status: "active",
	})
	if !errors.Is(err, contract.ErrInvalidProject) || repository.createCalls != 1 {
		t.Fatalf("invalid status error/calls = %v/%d", err, repository.createCalls)
	}
}

func TestProjectUseCaseAuthorizesBeforeRepositoryAccess(t *testing.T) {
	denied := errors.New("denied")
	repository := &projectRepositoryStub{}
	service, err := NewProjectUseCase(repository, &accessAuthorizerStub{err: denied}, func(context.Context) (string, error) {
		return "project-1", nil
	}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetProject(context.Background(), contract.GetProjectRequest{WorkspaceId: "workspace-1", ProjectId: "project-1"}); !errors.Is(err, denied) {
		t.Fatalf("GetProject() error = %v", err)
	}
	if repository.findCalls != 0 {
		t.Fatalf("repository read before authorization: %d", repository.findCalls)
	}
	if _, err := service.ListProjects(context.Background(), contract.ListProjectsRequest{WorkspaceId: "workspace-1"}); !errors.Is(err, denied) {
		t.Fatalf("ListProjects() error = %v", err)
	}
	if _, err := service.SearchProjects(context.Background(), contract.SearchProjectsRequest{WorkspaceId: "workspace-1", Query: "delivery"}); !errors.Is(err, denied) {
		t.Fatalf("SearchProjects() error = %v", err)
	}
	name := "Launch"
	if _, err := service.UpdateProject(context.Background(), contract.UpdateProjectRequest{WorkspaceId: "workspace-1", ProjectId: "project-1", Name: &name}); !errors.Is(err, denied) {
		t.Fatalf("UpdateProject() error = %v", err)
	}
	if _, err := service.DeleteProject(context.Background(), contract.DeleteProjectRequest{WorkspaceId: "workspace-1", ProjectId: "project-1"}); !errors.Is(err, denied) {
		t.Fatalf("DeleteProject() error = %v", err)
	}
	if repository.findCalls != 0 || repository.listCalls != 0 || repository.searchCalls != 0 || repository.updateCalls != 0 || repository.deleteCalls != 0 {
		t.Fatalf("repository accessed before authorization: %+v", repository)
	}
}

func TestProjectUseCaseMapsWorkspaceScopedMiss(t *testing.T) {
	service, err := NewProjectUseCase(&projectRepositoryStub{findErr: ErrProjectRecordNotFound}, &accessAuthorizerStub{}, func(context.Context) (string, error) {
		return "project-1", nil
	}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.GetProject(context.Background(), contract.GetProjectRequest{WorkspaceId: "workspace-2", ProjectId: "project-1"})
	if !errors.Is(err, contract.ErrProjectNotFound) {
		t.Fatalf("GetProject() error = %v", err)
	}
}

func TestProjectUseCaseLifecycleQueriesAndMutations(t *testing.T) {
	createdAt := time.Date(2026, 8, 3, 1, 0, 0, 0, time.UTC)
	now := createdAt.Add(time.Hour)
	value, err := projectDomain.New("project-1", "workspace-1", "Delivery", "ship", projectDomain.StatusPlanned, []string{"asset-1"}, createdAt)
	if err != nil {
		t.Fatal(err)
	}
	repository := &projectRepositoryStub{
		project: value, projects: []projectDomain.Project{value}, searchTotal: 1,
		searchResults: []ProjectSearchResult{{Project: value, MatchSource: "name", MatchedSnippet: "Delivery"}},
	}
	authorizer := &accessAuthorizerStub{}
	service, err := NewProjectUseCase(repository, authorizer, func(context.Context) (string, error) { return "unused", nil }, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	listed, err := service.ListProjects(context.Background(), contract.ListProjectsRequest{WorkspaceId: "workspace-1", Status: projectDomain.StatusPlanned})
	if err != nil || listed.Total != 1 || repository.listCalls != 1 {
		t.Fatalf("ListProjects() = %+v, %v; calls=%d", listed, err, repository.listCalls)
	}
	searched, err := service.SearchProjects(context.Background(), contract.SearchProjectsRequest{WorkspaceId: "workspace-1", Query: "  DELIVERY plan  ", Limit: 100})
	if err != nil || searched.Total != 1 || len(searched.Hits) != 1 {
		t.Fatalf("SearchProjects() = %+v, %v", searched, err)
	}
	if repository.searchQuery.Phrase != "delivery plan" || repository.searchQuery.Limit != 50 || len(repository.searchQuery.Terms) != 2 {
		t.Fatalf("search query = %+v", repository.searchQuery)
	}
	name := "  Launch  "
	description := ""
	updated, err := service.UpdateProject(context.Background(), contract.UpdateProjectRequest{
		WorkspaceId: "workspace-1", ProjectId: "project-1", Name: &name, Description: &description,
	})
	if err != nil || updated.Project == nil || updated.Project.Name != "Launch" || updated.Project.Description != "" || updated.Project.AssetIds[0] != "asset-1" {
		t.Fatalf("UpdateProject() = %+v, %v", updated.Project, err)
	}
	if _, err := service.DeleteProject(context.Background(), contract.DeleteProjectRequest{WorkspaceId: "workspace-1", ProjectId: "project-1"}); err != nil {
		t.Fatal(err)
	}
	if repository.updateCalls != 1 || repository.deleteCalls != 1 {
		t.Fatalf("update/delete calls = %d/%d", repository.updateCalls, repository.deleteCalls)
	}
	wantPermissions := []string{PermissionProjectList, PermissionProjectSearch, PermissionProjectUpdate, PermissionProjectDelete}
	if len(authorizer.permissions) != len(wantPermissions) {
		t.Fatalf("permissions = %v", authorizer.permissions)
	}
	for index := range wantPermissions {
		if authorizer.permissions[index] != wantPermissions[index] {
			t.Fatalf("permissions = %v", authorizer.permissions)
		}
	}
}

func TestProjectUseCaseRejectsInvalidLifecycleRequestsBeforePersistence(t *testing.T) {
	repository := &projectRepositoryStub{}
	service, err := NewProjectUseCase(repository, &accessAuthorizerStub{}, func(context.Context) (string, error) { return "unused", nil }, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ListProjects(context.Background(), contract.ListProjectsRequest{WorkspaceId: "workspace-1", Status: "active"}); !errors.Is(err, contract.ErrInvalidProject) {
		t.Fatalf("ListProjects() error = %v", err)
	}
	if _, err := service.SearchProjects(context.Background(), contract.SearchProjectsRequest{WorkspaceId: "workspace-1", Query: " "}); !errors.Is(err, contract.ErrInvalidProject) {
		t.Fatalf("SearchProjects() error = %v", err)
	}
	invalidName := " "
	project, _ := projectDomain.New("project-1", "workspace-1", "Delivery", "", projectDomain.StatusPlanned, nil, time.Now())
	repository.project = project
	if _, err := service.UpdateProject(context.Background(), contract.UpdateProjectRequest{WorkspaceId: "workspace-1", ProjectId: "project-1", Name: &invalidName}); !errors.Is(err, contract.ErrInvalidProject) {
		t.Fatalf("UpdateProject() error = %v", err)
	}
	if repository.listCalls != 0 || repository.searchCalls != 0 || repository.updateCalls != 0 || repository.deleteCalls != 0 {
		t.Fatalf("persistence calls = list:%d search:%d update:%d delete:%d", repository.listCalls, repository.searchCalls, repository.updateCalls, repository.deleteCalls)
	}
}

func TestRelationshipUseCaseAuthorizationAndActorBoundary(t *testing.T) {
	project, err := projectDomain.New("project-1", "workspace-1", "Delivery", "", "planned", nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name       string
		authErr    error
		belongs    bool
		wantErr    error
		wantFind   int
		wantActor  int
		wantWrites int
	}{
		{name: "authorized same workspace", belongs: true, wantFind: 1, wantActor: 1, wantWrites: 1},
		{name: "foreign actor", belongs: false, wantErr: contract.ErrActorOutsideWorkspace, wantFind: 1, wantActor: 1},
		{name: "authorization denied", authErr: errors.New("denied"), belongs: true, wantErr: errors.New("denied")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projects := &projectRepositoryStub{project: project}
			relations := &relationRepositoryStub{}
			actors := &actorReaderStub{belongs: tt.belongs}
			authorizer := &accessAuthorizerStub{err: tt.authErr}
			service, constructorErr := NewRelationshipUseCase(projects, relations, authorizer, actors)
			if constructorErr != nil {
				t.Fatal(constructorErr)
			}
			_, callErr := service.PutProjectActorRelation(context.Background(), contract.PutProjectActorRelationRequest{
				Relation: &contract.ProjectActorRelation{WorkspaceId: "workspace-1", ProjectId: "project-1", ActorType: "member", ActorId: "member-1", Role: "lead"},
			})
			if tt.wantErr == nil && callErr != nil {
				t.Fatalf("PutProjectActorRelation() error = %v", callErr)
			}
			if tt.wantErr != nil && callErr == nil {
				t.Fatal("PutProjectActorRelation() error = nil")
			}
			if tt.wantErr == contract.ErrActorOutsideWorkspace && !errors.Is(callErr, tt.wantErr) {
				t.Fatalf("PutProjectActorRelation() error = %v", callErr)
			}
			if projects.findCalls != tt.wantFind || actors.calls != tt.wantActor || relations.putCalls != tt.wantWrites {
				t.Fatalf("find/actor/write calls = %d/%d/%d", projects.findCalls, actors.calls, relations.putCalls)
			}
		})
	}
}

func TestRelationshipDeleteDoesNotRequireCurrentActorMembership(t *testing.T) {
	project, err := projectDomain.New("project-1", "workspace-1", "Delivery", "", "planned", nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	actors := &actorReaderStub{belongs: false}
	relations := &relationRepositoryStub{}
	service, err := NewRelationshipUseCase(&projectRepositoryStub{project: project}, relations, &accessAuthorizerStub{}, actors)
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.DeleteProjectActorRelation(context.Background(), contract.DeleteProjectActorRelationRequest{
		WorkspaceId: "workspace-1", ProjectId: "project-1", ActorType: "member", ActorId: "former-member",
	})
	if err != nil {
		t.Fatal(err)
	}
	if actors.calls != 0 || relations.deleteCalls != 1 {
		t.Fatalf("actor/delete calls = %d/%d", actors.calls, relations.deleteCalls)
	}
}

func TestRelationshipListIsDeterministic(t *testing.T) {
	project, err := projectDomain.New("project-1", "workspace-1", "Delivery", "", "planned", nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	agent, _ := relationDomain.New("workspace-1", "project-1", "agent", "agent-1", "agent")
	member, _ := relationDomain.New("workspace-1", "project-1", "member", "member-1", "lead")
	service, err := NewRelationshipUseCase(
		&projectRepositoryStub{project: project},
		&relationRepositoryStub{values: []relationDomain.Relation{member, agent}},
		&accessAuthorizerStub{}, &actorReaderStub{belongs: true},
	)
	if err != nil {
		t.Fatal(err)
	}
	response, err := service.ListProjectActorRelations(context.Background(), contract.ListProjectActorRelationsRequest{WorkspaceId: "workspace-1", ProjectId: "project-1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Relations) != 2 || response.Relations[0].Role != "agent" || response.Relations[1].Role != "lead" {
		t.Fatalf("relations = %+v", response.Relations)
	}
}
