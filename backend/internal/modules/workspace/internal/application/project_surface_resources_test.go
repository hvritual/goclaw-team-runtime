package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

type projectSurfaceResourceRepositoryStub struct {
	ProjectSurfaceRepository
	project         contract.ProjectSurfaceProject
	resources       []ProjectResourceSeed
	actor           contract.WorkspaceActor
	oldCreateCalled bool
	createErr       error
}

func (r *projectSurfaceResourceRepositoryStub) CreateProject(context.Context, contract.ProjectSurfaceProject) (contract.ProjectSurfaceProject, error) {
	r.oldCreateCalled = true
	return contract.ProjectSurfaceProject{}, errors.New("legacy non-atomic create called")
}

func (r *projectSurfaceResourceRepositoryStub) CreateProjectWithResources(_ context.Context, project contract.ProjectSurfaceProject, resources []ProjectResourceSeed, actor contract.WorkspaceActor) (contract.ProjectSurfaceProject, error) {
	r.project, r.resources, r.actor = project, resources, actor
	if r.createErr != nil {
		return contract.ProjectSurfaceProject{}, r.createErr
	}
	project.ResourceCount = len(resources)
	return project, nil
}

type projectSurfaceActorReaderStub struct{}

func (projectSurfaceActorReaderStub) ActorBelongsToWorkspace(context.Context, string, string, string) (bool, error) {
	return true, nil
}

func TestProjectSurfaceCreateNormalizesInitialResourcesForAtomicRepositoryCall(t *testing.T) {
	t.Parallel()

	repository := &projectSurfaceResourceRepositoryStub{}
	ids := []string{"project-1", "resource-1", "resource-2"}
	service, err := NewProjectSurfaceUseCase(
		repository,
		&projectResourceAuthorizerStub{},
		projectSurfaceActorReaderStub{},
		projectResourceMembershipsStub{membership: contract.WorkspaceMembership{UserID: "owner-1", Role: "owner"}, found: true},
		func(context.Context) (string, error) {
			id := ids[0]
			ids = ids[1:]
			return id, nil
		},
		func() time.Time { return time.Date(2026, 8, 19, 6, 0, 0, 0, time.UTC) },
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx := contract.WithWorkspaceActor(context.Background(), "member", "owner-1")
	created, err := service.CreateProject(ctx, "workspace-1", contract.CreateProjectSurfaceRequest{
		Title: " Runtime ",
		Resources: []contract.CreateProjectResourceRequest{
			{ResourceType: "github_repo", ResourceRef: contract.ProjectResourceRef{URL: "git@github.com:Acme/Runtime.git", Ref: " main "}, Label: " Code "},
			{ResourceType: "url", ResourceRef: contract.ProjectResourceRef{URL: "https://docs.example.com/guide/"}, Label: " Docs "},
		},
	})
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	if repository.oldCreateCalled || created.ResourceCount != 2 || repository.project.ID != "project-1" || repository.actor.ID != "owner-1" || len(repository.resources) != 2 {
		t.Fatalf("created=%#v project=%#v actor=%#v resources=%#v old=%t", created, repository.project, repository.actor, repository.resources, repository.oldCreateCalled)
	}
	if repository.resources[0].ID != "resource-1" || repository.resources[0].ResourceRef.URL != "https://github.com/acme/runtime" || repository.resources[0].ResourceRef.Ref != "main" || repository.resources[0].Label != "Code" {
		t.Fatalf("github seed = %#v", repository.resources[0])
	}
	if repository.resources[1].ID != "resource-2" || repository.resources[1].ResourceRef.URL != "https://docs.example.com/guide" || repository.resources[1].Label != "Docs" {
		t.Fatalf("url seed = %#v", repository.resources[1])
	}
}

func TestProjectSurfaceCreateRejectsDuplicateInitialResourcesBeforeRepository(t *testing.T) {
	t.Parallel()

	repository := &projectSurfaceResourceRepositoryStub{}
	service, err := NewProjectSurfaceUseCase(
		repository,
		&projectResourceAuthorizerStub{},
		projectSurfaceActorReaderStub{},
		projectResourceMembershipsStub{membership: contract.WorkspaceMembership{UserID: "owner-1", Role: "owner"}, found: true},
		func(context.Context) (string, error) { return "generated-id", nil },
		time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx := contract.WithWorkspaceActor(context.Background(), "member", "owner-1")
	_, err = service.CreateProject(ctx, "workspace-1", contract.CreateProjectSurfaceRequest{
		Title: "Runtime",
		Resources: []contract.CreateProjectResourceRequest{
			{ResourceType: "github_repo", ResourceRef: contract.ProjectResourceRef{URL: "https://github.com/acme/runtime.git"}},
			{ResourceType: "github_repo", ResourceRef: contract.ProjectResourceRef{URL: "git@github.com:acme/runtime.git"}},
		},
	})
	if !errors.Is(err, ErrInvalidProjectSurfaceRequest) {
		t.Fatalf("CreateProject() error = %v", err)
	}
	if repository.project.ID != "" || repository.oldCreateCalled {
		t.Fatalf("repository called: project=%#v old=%t", repository.project, repository.oldCreateCalled)
	}
}

func TestProjectSurfaceCreateRequiresResourceManagerForInitialResources(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		leadType *string
		leadID   *string
		wantErr  bool
	}{
		{name: "ordinary member without lead", wantErr: true},
		{name: "ordinary member leading another project actor", leadType: projectSurfaceString("member"), leadID: projectSurfaceString("other-member"), wantErr: true},
		{name: "ordinary member is requested lead", leadType: projectSurfaceString("member"), leadID: projectSurfaceString("user-1")},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			repository := &projectSurfaceResourceRepositoryStub{}
			service, err := NewProjectSurfaceUseCase(
				repository,
				&projectResourceAuthorizerStub{},
				projectSurfaceActorReaderStub{},
				projectResourceMembershipsStub{membership: contract.WorkspaceMembership{MemberID: "member-1", UserID: "user-1", Role: "member"}, found: true},
				func(context.Context) (string, error) { return "generated-id", nil },
				time.Now,
			)
			if err != nil {
				t.Fatal(err)
			}
			ctx := contract.WithWorkspaceActor(context.Background(), "member", "user-1")
			_, err = service.CreateProject(ctx, "workspace-1", contract.CreateProjectSurfaceRequest{
				Title: "Runtime", LeadType: test.leadType, LeadID: test.leadID,
				Resources: []contract.CreateProjectResourceRequest{{ResourceType: "url", ResourceRef: contract.ProjectResourceRef{URL: "https://example.com/docs"}}},
			})
			if test.wantErr {
				if !errors.Is(err, contract.ErrWorkspacePermissionDenied) || repository.project.ID != "" {
					t.Fatalf("CreateProject() error = %v, project = %#v", err, repository.project)
				}
				return
			}
			if err != nil || repository.project.ID == "" {
				t.Fatalf("CreateProject() error = %v, project = %#v", err, repository.project)
			}
		})
	}
}

func projectSurfaceString(value string) *string { return &value }
