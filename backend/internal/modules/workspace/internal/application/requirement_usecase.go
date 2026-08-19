package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	requirementDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/requirement"
)

const (
	PermissionRequirementSave = "workspace.requirement.save_version"
	PermissionRequirementGet  = "workspace.requirement.get"
)

var ErrRequirementRecordNotFound = errors.New("requirement record not found")

type RequirementRepository interface {
	FindByID(context.Context, string, string) (requirementDomain.Requirement, requirementDomain.Version, error)
	SaveVersion(context.Context, requirementDomain.Requirement, requirementDomain.Version, bool) error
}
type RequirementUseCase struct {
	repository                     RequirementRepository
	projects                       ProjectRepository
	issues                         IssueReferenceRepository
	authorizer                     contract.WorkspaceAccessAuthorizer
	newRequirementID, newVersionID ProjectIDGenerator
	now                            Clock
}

func NewRequirementUseCase(repository RequirementRepository, projects ProjectRepository, issues IssueReferenceRepository, authorizer contract.WorkspaceAccessAuthorizer, newRequirementID, newVersionID ProjectIDGenerator, now Clock) (*RequirementUseCase, error) {
	if repository == nil || projects == nil || issues == nil || authorizer == nil || newRequirementID == nil || newVersionID == nil || now == nil {
		return nil, errors.New("Requirement dependencies are required")
	}
	return &RequirementUseCase{repository: repository, projects: projects, issues: issues, authorizer: authorizer, newRequirementID: newRequirementID, newVersionID: newVersionID, now: now}, nil
}
func (s *RequirementUseCase) SaveRequirementVersion(ctx context.Context, r contract.SaveRequirementVersionRequest) (contract.SaveRequirementVersionResponse, error) {
	return contract.SaveRequirementVersionResponse{}, contract.LegacyRequirementMutationDisabledError{}
}
func (s *RequirementUseCase) GetRequirement(ctx context.Context, r contract.GetRequirementRequest) (contract.GetRequirementResponse, error) {
	w, id := strings.TrimSpace(r.WorkspaceId), strings.TrimSpace(r.RequirementId)
	if w == "" || id == "" {
		return contract.GetRequirementResponse{}, fmt.Errorf("%w: workspace id and Requirement id are required", contract.ErrInvalidRequirement)
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, w, PermissionRequirementGet); err != nil {
		return contract.GetRequirementResponse{}, err
	}
	req, version, err := s.repository.FindByID(ctx, w, id)
	if errors.Is(err, ErrRequirementRecordNotFound) {
		return contract.GetRequirementResponse{}, contract.ErrRequirementNotFound
	}
	if err != nil {
		return contract.GetRequirementResponse{}, fmt.Errorf("get Requirement: %w", err)
	}
	out, current := requirementToContract(req), requirementVersionToContract(version)
	return contract.GetRequirementResponse{Requirement: &out, CurrentVersion: &current}, nil
}
func requirementToContract(v requirementDomain.Requirement) contract.Requirement {
	return contract.Requirement{Id: v.ID, WorkspaceId: v.WorkspaceID, ProjectId: v.ProjectID, Title: v.Title, CurrentVersion: v.CurrentVersion, ApprovalStatus: v.ApprovalStatus, CoverageStatus: v.CoverageStatus, IssueIds: append([]string(nil), v.IssueIDs...), CreatedAt: v.CreatedAt.Format(time.RFC3339Nano), UpdatedAt: v.UpdatedAt.Format(time.RFC3339Nano)}
}
func requirementVersionToContract(v requirementDomain.Version) contract.RequirementVersion {
	return contract.RequirementVersion{Id: v.ID, RequirementId: v.RequirementID, Version: v.Version, Content: v.Content, CreatedAt: v.CreatedAt.Format(time.RFC3339Nano)}
}

var _ contract.RequirementService = (*RequirementUseCase)(nil)
