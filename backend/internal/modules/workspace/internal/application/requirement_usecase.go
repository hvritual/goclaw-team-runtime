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
	w, p := strings.TrimSpace(r.WorkspaceId), strings.TrimSpace(r.ProjectId)
	if w == "" || p == "" {
		return contract.SaveRequirementVersionResponse{}, fmt.Errorf("%w: workspace id and Project id are required", contract.ErrInvalidRequirement)
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, w, PermissionRequirementSave); err != nil {
		return contract.SaveRequirementVersionResponse{}, err
	}
	if _, err := s.projects.FindByID(ctx, w, p); errors.Is(err, ErrProjectRecordNotFound) {
		return contract.SaveRequirementVersionResponse{}, contract.ErrProjectNotFound
	} else if err != nil {
		return contract.SaveRequirementVersionResponse{}, fmt.Errorf("validate Requirement Project: %w", err)
	}
	for _, id := range r.IssueIds {
		issue, err := s.issues.FindByIDOrIdentifier(ctx, w, id)
		if errors.Is(err, ErrIssueRecordNotFound) {
			return contract.SaveRequirementVersionResponse{}, contract.ErrIssueNotFound
		} else if err != nil {
			return contract.SaveRequirementVersionResponse{}, fmt.Errorf("validate Requirement Issue: %w", err)
		}
		if issue.ProjectID == nil || *issue.ProjectID != p {
			return contract.SaveRequirementVersionResponse{}, fmt.Errorf("%w: Issue does not belong to Project", contract.ErrInvalidRequirement)
		}
	}
	creating := r.RequirementId == nil
	var req requirementDomain.Requirement
	var err error
	if creating {
		id, e := s.newRequirementID(ctx)
		if e != nil {
			return contract.SaveRequirementVersionResponse{}, fmt.Errorf("generate Requirement id: %w", e)
		}
		req, err = requirementDomain.New(id, w, p, r.Title, r.IssueIds, s.now())
	} else {
		req, _, err = s.repository.FindByID(ctx, w, strings.TrimSpace(*r.RequirementId))
		if errors.Is(err, ErrRequirementRecordNotFound) {
			return contract.SaveRequirementVersionResponse{}, contract.ErrRequirementNotFound
		}
		if err == nil && req.ProjectID != p {
			return contract.SaveRequirementVersionResponse{}, contract.ErrRequirementNotFound
		}
		if err == nil {
			req, err = req.NextVersion(r.Title, r.IssueIds, s.now())
		}
	}
	if err != nil {
		return contract.SaveRequirementVersionResponse{}, fmt.Errorf("%w: %v", contract.ErrInvalidRequirement, err)
	}
	versionID, e := s.newVersionID(ctx)
	if e != nil {
		return contract.SaveRequirementVersionResponse{}, fmt.Errorf("generate Requirement version id: %w", e)
	}
	version, err := requirementDomain.NewVersion(versionID, req.ID, req.CurrentVersion, r.Content, s.now())
	if err != nil {
		return contract.SaveRequirementVersionResponse{}, fmt.Errorf("%w: %v", contract.ErrInvalidRequirement, err)
	}
	if err := s.repository.SaveVersion(ctx, req, version, creating); err != nil {
		return contract.SaveRequirementVersionResponse{}, fmt.Errorf("save Requirement version: %w", err)
	}
	out, current := requirementToContract(req), requirementVersionToContract(version)
	return contract.SaveRequirementVersionResponse{Requirement: &out, Version: &current}, nil
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
