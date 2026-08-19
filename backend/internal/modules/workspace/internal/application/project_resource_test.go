package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

type projectResourceRepositoryStub struct {
	access       ProjectResourceProjectAccess
	list         contract.ProjectResourceList
	resource     contract.ProjectResource
	create       ProjectResourceCreate
	mutation     ProjectResourceMutation
	mutateErr    error
	createCalls  int
	mutateCalls  int
	archiveCalls int
	refreshCalls int
	refreshErr   error
}

func (r *projectResourceRepositoryStub) ProjectResourceAccess(context.Context, string, string) (ProjectResourceProjectAccess, error) {
	return r.access, nil
}

func (r *projectResourceRepositoryStub) ListProjectResources(context.Context, string, string, bool) (contract.ProjectResourceList, error) {
	return r.list, nil
}

func (r *projectResourceRepositoryStub) GetProjectResource(context.Context, string, string, string) (contract.ProjectResource, error) {
	return r.resource, nil
}

func (r *projectResourceRepositoryStub) CreateProjectResource(_ context.Context, command ProjectResourceCreate) (contract.ProjectResource, error) {
	r.createCalls++
	r.create = command
	if r.resource.ID == "" {
		r.resource = contract.ProjectResource{
			ID: command.ID, WorkspaceID: command.WorkspaceID, ProjectID: command.ProjectID,
			ResourceType: command.ResourceType, ResourceRef: command.ResourceRef,
			Label: command.Label, Status: "active", Revision: 1,
		}
	}
	return r.resource, nil
}

func (r *projectResourceRepositoryStub) MutateProjectResource(_ context.Context, command ProjectResourceMutation) (contract.ProjectResource, error) {
	r.mutateCalls++
	r.mutation = command
	if r.mutateErr != nil {
		return contract.ProjectResource{}, r.mutateErr
	}
	result := r.resource
	result.Connection = command.Connection
	result.Revision = command.ExpectedRevision + 1
	return result, nil
}

func (r *projectResourceRepositoryStub) RefreshProjectResource(ctx context.Context, command ProjectResourceMutation, resolver ProjectResourceConnectionResolver) (contract.ProjectResource, error) {
	r.refreshCalls++
	if r.refreshErr != nil {
		return contract.ProjectResource{}, r.refreshErr
	}
	command.Connection = resolver(ctx, contract.ProjectResourceConnectionRequest{
		WorkspaceID: command.WorkspaceID, ProjectID: command.ProjectID, ResourceID: command.ResourceID,
		ResourceType: r.resource.ResourceType, ResourceRef: r.resource.ResourceRef,
	})
	r.mutation = command
	result := r.resource
	result.Connection = command.Connection
	result.Revision = command.ExpectedRevision + 1
	return result, nil
}

func TestProjectResourceStaleRestoreReturnsRevisionConflictBeforeStateError(t *testing.T) {
	t.Parallel()

	repository := &projectResourceRepositoryStub{
		access:    ProjectResourceProjectAccess{Status: "in_progress"},
		mutateErr: contract.RevisionConflictError{CurrentRevision: 5},
		resource: contract.ProjectResource{
			ID: "resource-1", WorkspaceID: "workspace-1", ProjectID: "project-1",
			ResourceType: "url", ResourceRef: contract.ProjectResourceRef{URL: "https://example.com/docs"},
			Status: "active", Revision: 5,
		},
	}
	service, err := NewProjectResourceUseCase(
		repository,
		&projectResourceAuthorizerStub{},
		projectResourceMembershipsStub{membership: contract.WorkspaceMembership{UserID: "owner-1", Role: "owner"}, found: true},
		&projectResourceCheckerStub{},
		func(context.Context) (string, error) { return "resource-2", nil },
		time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx := contract.WithWorkspaceActor(context.Background(), "member", "owner-1")
	_, err = service.UpdateProjectResource(ctx, "workspace-1", "project-1", "resource-1", contract.UpdateProjectResourceRequest{
		Action: "restore", ExpectedRevision: 4,
	})
	var conflict contract.RevisionConflictError
	if !errors.As(err, &conflict) || conflict.CurrentRevision != 5 {
		t.Fatalf("UpdateProjectResource() error = %v", err)
	}
	if repository.mutateCalls != 1 {
		t.Fatalf("mutate calls = %d", repository.mutateCalls)
	}
}

func (r *projectResourceRepositoryStub) ArchiveProjectResource(context.Context, string, string, string, int64, contract.WorkspaceActor, time.Time) error {
	r.archiveCalls++
	return nil
}

type projectResourceAuthorizerStub struct {
	err         error
	permissions []string
}

func (a *projectResourceAuthorizerStub) AuthorizeWorkspace(_ context.Context, _ string, permission string) error {
	a.permissions = append(a.permissions, permission)
	return a.err
}

type projectResourceMembershipsStub struct {
	membership contract.WorkspaceMembership
	found      bool
}

func (m projectResourceMembershipsStub) ListForUser(context.Context, string) ([]contract.WorkspaceMembership, error) {
	return nil, nil
}

func (m projectResourceMembershipsStub) FindForUserAndWorkspace(context.Context, string, string) (contract.WorkspaceMembership, bool, error) {
	return m.membership, m.found, nil
}

func (m projectResourceMembershipsStub) FindByMemberAndWorkspace(context.Context, string, string) (contract.WorkspaceMembership, bool, error) {
	return m.membership, m.found, nil
}

type projectResourceCheckerStub struct {
	connection contract.ProjectResourceConnection
	err        error
	calls      int
}

func (c *projectResourceCheckerStub) Check(context.Context, contract.ProjectResourceConnectionRequest) (contract.ProjectResourceConnection, error) {
	c.calls++
	return c.connection, c.err
}

func TestProjectResourceLeadCreatesCanonicalResource(t *testing.T) {
	t.Parallel()

	repository := &projectResourceRepositoryStub{
		access: ProjectResourceProjectAccess{Status: "in_progress", LeadType: "member", LeadID: "user-1"},
	}
	authorizer := &projectResourceAuthorizerStub{}
	service, err := NewProjectResourceUseCase(
		repository,
		authorizer,
		projectResourceMembershipsStub{membership: contract.WorkspaceMembership{UserID: "user-1", Role: "member"}, found: true},
		&projectResourceCheckerStub{},
		func(context.Context) (string, error) { return "resource-1", nil },
		func() time.Time { return time.Date(2026, 8, 19, 1, 2, 3, 0, time.UTC) },
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx := contract.WithWorkspaceActor(context.Background(), "member", "user-1")
	result, err := service.CreateProjectResource(ctx, "workspace-1", "project-1", "idem-1", contract.CreateProjectResourceRequest{
		ResourceType: "github_repo",
		ResourceRef:  contract.ProjectResourceRef{URL: "git@github.com:Acme/Repo.git", Ref: " main "},
		Label:        " Runtime ",
	})
	if err != nil {
		t.Fatalf("CreateProjectResource() error = %v", err)
	}
	if result.ID != "resource-1" || repository.createCalls != 1 {
		t.Fatalf("result=%#v createCalls=%d", result, repository.createCalls)
	}
	if repository.create.ResourceRef.URL != "https://github.com/acme/repo" || repository.create.ResourceRef.Ref != "main" {
		t.Fatalf("canonical ref = %#v", repository.create.ResourceRef)
	}
	if repository.create.Label != "Runtime" || repository.create.IdempotencyKey != "idem-1" || len(repository.create.RequestHash) != 64 {
		t.Fatalf("create command = %#v", repository.create)
	}
	if len(authorizer.permissions) != 1 || authorizer.permissions[0] != contract.PermissionResourceRead {
		t.Fatalf("permissions = %#v", authorizer.permissions)
	}
}

func TestProjectResourceNonLeadMemberCannotManage(t *testing.T) {
	t.Parallel()

	repository := &projectResourceRepositoryStub{
		access: ProjectResourceProjectAccess{Status: "in_progress", LeadType: "member", LeadID: "another-user"},
	}
	service, err := NewProjectResourceUseCase(
		repository,
		&projectResourceAuthorizerStub{},
		projectResourceMembershipsStub{membership: contract.WorkspaceMembership{UserID: "user-1", Role: "member"}, found: true},
		&projectResourceCheckerStub{},
		func(context.Context) (string, error) { return "resource-1", nil },
		time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx := contract.WithWorkspaceActor(context.Background(), "member", "user-1")
	_, err = service.CreateProjectResource(ctx, "workspace-1", "project-1", "idem-1", contract.CreateProjectResourceRequest{
		ResourceType: "url", ResourceRef: contract.ProjectResourceRef{URL: "https://example.com/docs"},
	})
	if !errors.Is(err, contract.ErrWorkspacePermissionDenied) {
		t.Fatalf("CreateProjectResource() error = %v", err)
	}
	if repository.createCalls != 0 {
		t.Fatalf("create calls = %d", repository.createCalls)
	}
}

func TestProjectResourceOwnerRefreshFailurePersistsSafeUnavailableProjection(t *testing.T) {
	t.Parallel()

	repository := &projectResourceRepositoryStub{
		access: ProjectResourceProjectAccess{Status: "in_progress"},
		resource: contract.ProjectResource{
			ID: "resource-1", WorkspaceID: "workspace-1", ProjectID: "project-1",
			ResourceType: "url", ResourceRef: contract.ProjectResourceRef{URL: "https://example.com/docs"},
			Status: "active", Revision: 2,
		},
	}
	checker := &projectResourceCheckerStub{err: errors.New("provider response contained secret=do-not-return")}
	service, err := NewProjectResourceUseCase(
		repository,
		&projectResourceAuthorizerStub{},
		projectResourceMembershipsStub{membership: contract.WorkspaceMembership{UserID: "owner-1", Role: "owner"}, found: true},
		checker,
		func(context.Context) (string, error) { return "resource-2", nil },
		func() time.Time { return time.Date(2026, 8, 19, 1, 2, 3, 0, time.UTC) },
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx := contract.WithWorkspaceActor(context.Background(), "member", "owner-1")
	result, err := service.UpdateProjectResource(ctx, "workspace-1", "project-1", "resource-1", contract.UpdateProjectResourceRequest{
		Action: "refresh", ExpectedRevision: 2,
	})
	if err != nil {
		t.Fatalf("UpdateProjectResource() error = %v", err)
	}
	if checker.calls != 1 || repository.refreshCalls != 1 {
		t.Fatalf("checker=%d refreshes=%d", checker.calls, repository.refreshCalls)
	}
	if result.Connection.State != "unavailable" || result.Connection.DiagnosticCode != "connection_check_failed" {
		t.Fatalf("connection = %#v", result.Connection)
	}
	if result.Connection.DiagnosticCode == checker.err.Error() {
		t.Fatalf("raw provider error escaped into projection")
	}
}

func TestProjectResourceArchiveValidatesRevisionBeforeRepository(t *testing.T) {
	t.Parallel()

	repository := &projectResourceRepositoryStub{
		access: ProjectResourceProjectAccess{Status: "in_progress"},
	}
	service, err := NewProjectResourceUseCase(
		repository,
		&projectResourceAuthorizerStub{},
		projectResourceMembershipsStub{membership: contract.WorkspaceMembership{UserID: "admin-1", Role: "admin"}, found: true},
		&projectResourceCheckerStub{},
		func(context.Context) (string, error) { return "resource-1", nil },
		time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx := contract.WithWorkspaceActor(context.Background(), "member", "admin-1")
	if err = service.ArchiveProjectResource(ctx, "workspace-1", "project-1", "resource-1", 0); !errors.Is(err, ErrInvalidProjectResourceRequest) {
		t.Fatalf("ArchiveProjectResource() error = %v", err)
	}
	if repository.archiveCalls != 0 {
		t.Fatalf("archive calls = %d", repository.archiveCalls)
	}
}

func TestProjectResourceRejectsIdempotencyKeysOutsideGovernanceLimit(t *testing.T) {
	t.Parallel()

	repository := &projectResourceRepositoryStub{
		access: ProjectResourceProjectAccess{Status: "in_progress"},
	}
	service, err := NewProjectResourceUseCase(
		repository,
		&projectResourceAuthorizerStub{},
		projectResourceMembershipsStub{membership: contract.WorkspaceMembership{UserID: "owner-1", Role: "owner"}, found: true},
		&projectResourceCheckerStub{},
		func(context.Context) (string, error) { return "resource-1", nil },
		time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx := contract.WithWorkspaceActor(context.Background(), "member", "owner-1")
	_, err = service.CreateProjectResource(ctx, "workspace-1", "project-1", strings.Repeat("k", 201), contract.CreateProjectResourceRequest{
		ResourceType: "url", ResourceRef: contract.ProjectResourceRef{URL: "https://example.com/docs"},
	})
	if !errors.Is(err, ErrInvalidProjectResourceRequest) {
		t.Fatalf("CreateProjectResource() error = %v", err)
	}
	if repository.createCalls != 0 {
		t.Fatalf("create calls = %d", repository.createCalls)
	}
}

func TestProjectResourceStaleRefreshDoesNotCallConnectionAdapter(t *testing.T) {
	t.Parallel()

	repository := &projectResourceRepositoryStub{
		access:     ProjectResourceProjectAccess{Status: "in_progress"},
		refreshErr: contract.RevisionConflictError{CurrentRevision: 5},
		resource: contract.ProjectResource{
			ID: "resource-1", WorkspaceID: "workspace-1", ProjectID: "project-1", ResourceType: "url",
			ResourceRef: contract.ProjectResourceRef{URL: "https://example.com/docs"}, Status: "active", Revision: 3,
		},
	}
	checker := &projectResourceCheckerStub{connection: contract.ProjectResourceConnection{State: "available"}}
	service, err := NewProjectResourceUseCase(
		repository,
		&projectResourceAuthorizerStub{},
		projectResourceMembershipsStub{membership: contract.WorkspaceMembership{UserID: "owner-1", Role: "owner"}, found: true},
		checker,
		func(context.Context) (string, error) { return "resource-2", nil },
		time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx := contract.WithWorkspaceActor(context.Background(), "member", "owner-1")
	_, err = service.UpdateProjectResource(ctx, "workspace-1", "project-1", "resource-1", contract.UpdateProjectResourceRequest{Action: "refresh", ExpectedRevision: 4})
	var conflict contract.RevisionConflictError
	if !errors.As(err, &conflict) || conflict.CurrentRevision != 5 {
		t.Fatalf("UpdateProjectResource() error = %v", err)
	}
	if checker.calls != 0 || repository.refreshCalls != 1 {
		t.Fatalf("checker calls = %d, refresh calls = %d", checker.calls, repository.refreshCalls)
	}
}

func TestProjectResourceManagementDeniesClosedProjects(t *testing.T) {
	t.Parallel()

	for _, status := range []string{"completed", "cancelled"} {
		t.Run(status, func(t *testing.T) {
			repository := &projectResourceRepositoryStub{access: ProjectResourceProjectAccess{Status: status}}
			service, err := NewProjectResourceUseCase(
				repository,
				&projectResourceAuthorizerStub{},
				projectResourceMembershipsStub{membership: contract.WorkspaceMembership{UserID: "owner-1", Role: "owner"}, found: true},
				&projectResourceCheckerStub{},
				func(context.Context) (string, error) { return "resource-1", nil },
				time.Now,
			)
			if err != nil {
				t.Fatal(err)
			}
			ctx := contract.WithWorkspaceActor(context.Background(), "member", "owner-1")
			_, err = service.CreateProjectResource(ctx, "workspace-1", "project-1", "create-1", contract.CreateProjectResourceRequest{
				ResourceType: "url", ResourceRef: contract.ProjectResourceRef{URL: "https://example.com"},
			})
			if !errors.Is(err, contract.ErrWorkspacePermissionDenied) {
				t.Fatalf("CreateProjectResource() error = %v", err)
			}
			if repository.createCalls != 0 {
				t.Fatalf("create calls = %d", repository.createCalls)
			}
		})
	}
}

func TestProjectResourceUpdateRejectsReorderOnlyFields(t *testing.T) {
	t.Parallel()

	repository := &projectResourceRepositoryStub{
		access: ProjectResourceProjectAccess{Status: "in_progress"},
		resource: contract.ProjectResource{
			ID: "resource-1", WorkspaceID: "workspace-1", ProjectID: "project-1",
			ResourceType: "url", ResourceRef: contract.ProjectResourceRef{URL: "https://example.com/docs"},
			Status: "active", Revision: 2,
		},
	}
	service, err := NewProjectResourceUseCase(
		repository,
		&projectResourceAuthorizerStub{},
		projectResourceMembershipsStub{membership: contract.WorkspaceMembership{UserID: "owner-1", Role: "owner"}, found: true},
		&projectResourceCheckerStub{},
		func(context.Context) (string, error) { return "resource-2", nil },
		time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	beforeID := "resource-2"
	label := "Changed"
	ctx := contract.WithWorkspaceActor(context.Background(), "member", "owner-1")
	_, err = service.UpdateProjectResource(ctx, "workspace-1", "project-1", "resource-1", contract.UpdateProjectResourceRequest{
		Action: "update", ExpectedRevision: 2, Label: &label, BeforeResourceID: &beforeID,
	})
	if !errors.Is(err, ErrInvalidProjectResourceRequest) {
		t.Fatalf("UpdateProjectResource() error = %v", err)
	}
	if repository.mutateCalls != 0 {
		t.Fatalf("mutate calls = %d", repository.mutateCalls)
	}
}
