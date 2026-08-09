package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	projectDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/project"
)

const (
	PermissionProjectCreate = "workspace.project.create"
	PermissionProjectGet    = "workspace.project.get"
	PermissionProjectList   = "workspace.project.list"
	PermissionProjectSearch = "workspace.project.search"
	PermissionProjectUpdate = "workspace.project.update"
	PermissionProjectDelete = "workspace.project.delete"
)

var ErrProjectRecordNotFound = errors.New("project record not found")

type ProjectRepository interface {
	Create(context.Context, projectDomain.Project) error
	FindByID(context.Context, string, string) (projectDomain.Project, error)
	List(context.Context, string, string) ([]projectDomain.Project, error)
	Search(context.Context, ProjectSearchQuery) ([]ProjectSearchResult, int, error)
	Update(context.Context, projectDomain.Project) error
	DeleteWithDependents(context.Context, string, string, time.Time) error
}

type ProjectSearchQuery struct {
	WorkspaceID   string
	Phrase        string
	Terms         []string
	IncludeClosed bool
	Limit         int
	Offset        int
}

type ProjectSearchResult struct {
	Project        projectDomain.Project
	MatchSource    string
	MatchedSnippet string
}

type ProjectIDGenerator func(context.Context) (string, error)
type Clock func() time.Time

type ProjectUseCase struct {
	repository ProjectRepository
	authorizer contract.WorkspaceAccessAuthorizer
	newID      ProjectIDGenerator
	now        Clock
}

func NewProjectUseCase(repository ProjectRepository, authorizer contract.WorkspaceAccessAuthorizer, newID ProjectIDGenerator, now Clock) (*ProjectUseCase, error) {
	if repository == nil {
		return nil, errors.New("project repository is required")
	}
	if authorizer == nil {
		return nil, errors.New("workspace access authorizer is required")
	}
	if newID == nil {
		return nil, errors.New("project id generator is required")
	}
	if now == nil {
		return nil, errors.New("project clock is required")
	}
	return &ProjectUseCase{repository: repository, authorizer: authorizer, newID: newID, now: now}, nil
}

func (s *ProjectUseCase) CreateProject(ctx context.Context, request contract.CreateProjectRequest) (contract.CreateProjectResponse, error) {
	workspaceID := strings.TrimSpace(request.WorkspaceId)
	if workspaceID == "" {
		return contract.CreateProjectResponse{}, fmt.Errorf("%w: workspace id is required", contract.ErrInvalidProject)
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, workspaceID, PermissionProjectCreate); err != nil {
		return contract.CreateProjectResponse{}, err
	}
	id, err := s.newID(ctx)
	if err != nil {
		return contract.CreateProjectResponse{}, fmt.Errorf("generate project id: %w", err)
	}
	value, err := projectDomain.New(id, workspaceID, request.Name, request.Description, request.Status, request.AssetIds, s.now())
	if err != nil {
		return contract.CreateProjectResponse{}, fmt.Errorf("%w: %v", contract.ErrInvalidProject, err)
	}
	if err := s.repository.Create(ctx, value); err != nil {
		return contract.CreateProjectResponse{}, fmt.Errorf("create project: %w", err)
	}
	result := projectToContract(value)
	return contract.CreateProjectResponse{Project: &result}, nil
}

func (s *ProjectUseCase) GetProject(ctx context.Context, request contract.GetProjectRequest) (contract.GetProjectResponse, error) {
	workspaceID := strings.TrimSpace(request.WorkspaceId)
	projectID := strings.TrimSpace(request.ProjectId)
	if workspaceID == "" || projectID == "" {
		return contract.GetProjectResponse{}, fmt.Errorf("%w: workspace id and project id are required", contract.ErrInvalidProject)
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, workspaceID, PermissionProjectGet); err != nil {
		return contract.GetProjectResponse{}, err
	}
	value, err := s.repository.FindByID(ctx, workspaceID, projectID)
	if errors.Is(err, ErrProjectRecordNotFound) {
		return contract.GetProjectResponse{}, contract.ErrProjectNotFound
	}
	if err != nil {
		return contract.GetProjectResponse{}, fmt.Errorf("get project: %w", err)
	}
	result := projectToContract(value)
	return contract.GetProjectResponse{Project: &result}, nil
}

func (s *ProjectUseCase) ListProjects(ctx context.Context, request contract.ListProjectsRequest) (contract.ListProjectsResponse, error) {
	workspaceID := strings.TrimSpace(request.WorkspaceId)
	status := strings.TrimSpace(request.Status)
	if workspaceID == "" || (status != "" && !projectDomain.IsValidStatus(status)) {
		return contract.ListProjectsResponse{}, fmt.Errorf("%w: workspace id and status must be valid", contract.ErrInvalidProject)
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, workspaceID, PermissionProjectList); err != nil {
		return contract.ListProjectsResponse{}, err
	}
	values, err := s.repository.List(ctx, workspaceID, status)
	if err != nil {
		return contract.ListProjectsResponse{}, fmt.Errorf("list projects: %w", err)
	}
	projects := make([]contract.Project, len(values))
	for index, value := range values {
		projects[index] = projectToContract(value)
	}
	return contract.ListProjectsResponse{Projects: projects, Total: countToInt32(len(projects))}, nil
}

func (s *ProjectUseCase) SearchProjects(ctx context.Context, request contract.SearchProjectsRequest) (contract.SearchProjectsResponse, error) {
	workspaceID := strings.TrimSpace(request.WorkspaceId)
	phrase := strings.ToLower(strings.TrimSpace(request.Query))
	if workspaceID == "" || phrase == "" || request.Limit < 0 || request.Offset < 0 {
		return contract.SearchProjectsResponse{}, fmt.Errorf("%w: workspace id, query, limit, and offset must be valid", contract.ErrInvalidProject)
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, workspaceID, PermissionProjectSearch); err != nil {
		return contract.SearchProjectsResponse{}, err
	}
	limit := int(request.Limit)
	if limit == 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}
	values, total, err := s.repository.Search(ctx, ProjectSearchQuery{
		WorkspaceID: workspaceID, Phrase: phrase, Terms: strings.Fields(phrase),
		IncludeClosed: request.IncludeClosed, Limit: limit, Offset: int(request.Offset),
	})
	if err != nil {
		return contract.SearchProjectsResponse{}, fmt.Errorf("search projects: %w", err)
	}
	hits := make([]contract.ProjectSearchHit, len(values))
	for index, value := range values {
		project := projectToContract(value.Project)
		snippet := value.MatchedSnippet
		hits[index] = contract.ProjectSearchHit{Project: &project, MatchSource: value.MatchSource, MatchedSnippet: &snippet}
	}
	return contract.SearchProjectsResponse{Hits: hits, Total: countToInt32(total)}, nil
}

func (s *ProjectUseCase) UpdateProject(ctx context.Context, request contract.UpdateProjectRequest) (contract.UpdateProjectResponse, error) {
	workspaceID := strings.TrimSpace(request.WorkspaceId)
	projectID := strings.TrimSpace(request.ProjectId)
	if workspaceID == "" || projectID == "" {
		return contract.UpdateProjectResponse{}, fmt.Errorf("%w: workspace id and project id are required", contract.ErrInvalidProject)
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, workspaceID, PermissionProjectUpdate); err != nil {
		return contract.UpdateProjectResponse{}, err
	}
	value, err := s.repository.FindByID(ctx, workspaceID, projectID)
	if errors.Is(err, ErrProjectRecordNotFound) {
		return contract.UpdateProjectResponse{}, contract.ErrProjectNotFound
	}
	if err != nil {
		return contract.UpdateProjectResponse{}, fmt.Errorf("get project for update: %w", err)
	}
	updated, err := value.Update(request.Name, request.Description, request.Status, s.now())
	if err != nil {
		return contract.UpdateProjectResponse{}, fmt.Errorf("%w: %v", contract.ErrInvalidProject, err)
	}
	if err := s.repository.Update(ctx, updated); errors.Is(err, ErrProjectRecordNotFound) {
		return contract.UpdateProjectResponse{}, contract.ErrProjectNotFound
	} else if err != nil {
		return contract.UpdateProjectResponse{}, fmt.Errorf("update project: %w", err)
	}
	result := projectToContract(updated)
	return contract.UpdateProjectResponse{Project: &result}, nil
}

func (s *ProjectUseCase) DeleteProject(ctx context.Context, request contract.DeleteProjectRequest) (contract.DeleteProjectResponse, error) {
	workspaceID := strings.TrimSpace(request.WorkspaceId)
	projectID := strings.TrimSpace(request.ProjectId)
	if workspaceID == "" || projectID == "" {
		return contract.DeleteProjectResponse{}, fmt.Errorf("%w: workspace id and project id are required", contract.ErrInvalidProject)
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, workspaceID, PermissionProjectDelete); err != nil {
		return contract.DeleteProjectResponse{}, err
	}
	if err := s.repository.DeleteWithDependents(ctx, workspaceID, projectID, s.now()); errors.Is(err, ErrProjectRecordNotFound) {
		return contract.DeleteProjectResponse{}, contract.ErrProjectNotFound
	} else if err != nil {
		return contract.DeleteProjectResponse{}, fmt.Errorf("delete project: %w", err)
	}
	return contract.DeleteProjectResponse{}, nil
}

func countToInt32(count int) int32 {
	const maxInt32 = int(^uint32(0) >> 1)
	if count > maxInt32 {
		return int32(maxInt32)
	}
	return int32(count)
}

func projectToContract(value projectDomain.Project) contract.Project {
	return contract.Project{
		Id: value.ID(), WorkspaceId: value.WorkspaceID(), Name: value.Name(),
		Description: value.Description(), Status: value.Status(), AssetIds: value.AssetIDs(),
		CreatedAt: value.CreatedAt().Format(time.RFC3339Nano),
		UpdatedAt: value.UpdatedAt().Format(time.RFC3339Nano),
	}
}

var _ contract.ProjectService = (*ProjectUseCase)(nil)
