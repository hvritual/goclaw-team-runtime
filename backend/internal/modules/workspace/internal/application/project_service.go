// dddgen:service-implementation ProjectService; method bodies are user-owned.
package application

import (
	"context"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

type ProjectService struct{}

func NewProjectService() *ProjectService { return &ProjectService{} }

func (s *ProjectService) CreateProject(ctx context.Context, request contract.CreateProjectRequest) (contract.CreateProjectResponse, error) {
	return contract.CreateProjectResponse{}, contract.ErrProjectNotImplemented
}

func (s *ProjectService) GetProject(ctx context.Context, request contract.GetProjectRequest) (contract.GetProjectResponse, error) {
	return contract.GetProjectResponse{}, contract.ErrProjectNotImplemented
}
func (s *ProjectService) ListProjects(ctx context.Context, request contract.ListProjectsRequest) (contract.ListProjectsResponse, error) {
	return contract.ListProjectsResponse{}, contract.ErrProjectNotImplemented
}
func (s *ProjectService) SearchProjects(ctx context.Context, request contract.SearchProjectsRequest) (contract.SearchProjectsResponse, error) {
	return contract.SearchProjectsResponse{}, contract.ErrProjectNotImplemented
}
func (s *ProjectService) UpdateProject(ctx context.Context, request contract.UpdateProjectRequest) (contract.UpdateProjectResponse, error) {
	return contract.UpdateProjectResponse{}, contract.ErrProjectNotImplemented
}
func (s *ProjectService) DeleteProject(ctx context.Context, request contract.DeleteProjectRequest) (contract.DeleteProjectResponse, error) {
	return contract.DeleteProjectResponse{}, contract.ErrProjectNotImplemented
}
