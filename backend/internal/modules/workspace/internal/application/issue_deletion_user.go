package application

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

const PermissionIssueDelete = "workspace.issue.delete"

type IssueDeletionRepository interface {
	Delete(context.Context, string, string) (string, error)
}

type IssueDeletionUseCase struct {
	repository IssueDeletionRepository
	authorizer contract.WorkspaceAccessAuthorizer
}

func NewIssueDeletionUseCase(repository IssueDeletionRepository, authorizer contract.WorkspaceAccessAuthorizer) (*IssueDeletionUseCase, error) {
	if repository == nil || authorizer == nil {
		return nil, errors.New("Issue deletion dependencies are required")
	}
	return &IssueDeletionUseCase{repository: repository, authorizer: authorizer}, nil
}

func (s *IssueDeletionUseCase) DeleteIssue(ctx context.Context, request contract.DeleteIssueRequest) (contract.DeleteIssueResponse, error) {
	workspaceID, issueID := strings.TrimSpace(request.WorkspaceID), strings.TrimSpace(request.IssueID)
	if workspaceID == "" || issueID == "" {
		return contract.DeleteIssueResponse{}, fmt.Errorf("%w: workspace id and issue id are required", contract.ErrInvalidIssue)
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, workspaceID, PermissionIssueDelete); err != nil {
		return contract.DeleteIssueResponse{}, err
	}
	resolvedID, err := s.repository.Delete(ctx, workspaceID, issueID)
	if errors.Is(err, ErrIssueRecordNotFound) {
		return contract.DeleteIssueResponse{}, contract.ErrIssueNotFound
	}
	if err != nil {
		return contract.DeleteIssueResponse{}, fmt.Errorf("delete Issue: %w", err)
	}
	return contract.DeleteIssueResponse{IssueID: resolvedID}, nil
}
