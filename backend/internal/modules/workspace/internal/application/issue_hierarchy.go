package application

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

const maximumIssueBatchSize = 200

func (s *IssueUseCase) ListIssueChildren(ctx context.Context, request contract.ListIssueChildrenRequest) (contract.ListIssueChildrenResponse, error) {
	workspaceID, issueID, err := validateIssueIdentity(request.WorkspaceID, request.IssueID)
	if err != nil {
		return contract.ListIssueChildrenResponse{}, err
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, workspaceID, PermissionIssueList); err != nil {
		return contract.ListIssueChildrenResponse{}, err
	}
	parent, err := s.findIssue(ctx, workspaceID, issueID)
	if err != nil {
		return contract.ListIssueChildrenResponse{}, err
	}
	return s.listIssueChildren(ctx, workspaceID, []string{parent.ID})
}

func (s *IssueUseCase) ListIssueChildrenByParents(ctx context.Context, request contract.ListIssueChildrenByParentsRequest) (contract.ListIssueChildrenResponse, error) {
	workspaceID := strings.TrimSpace(request.WorkspaceID)
	if workspaceID == "" {
		return contract.ListIssueChildrenResponse{}, fmt.Errorf("%w: workspace id is required", contract.ErrInvalidIssue)
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, workspaceID, PermissionIssueList); err != nil {
		return contract.ListIssueChildrenResponse{}, err
	}
	parentIDs, err := cleanIssueIDs(request.ParentIDs)
	if err != nil || len(parentIDs) == 0 {
		return contract.ListIssueChildrenResponse{}, fmt.Errorf("%w: parent_ids is required", contract.ErrInvalidIssue)
	}
	canonical := make([]string, 0, len(parentIDs))
	for _, parentID := range parentIDs {
		parent, findErr := s.findIssue(ctx, workspaceID, parentID)
		if findErr != nil {
			return contract.ListIssueChildrenResponse{}, findErr
		}
		canonical = append(canonical, parent.ID)
	}
	return s.listIssueChildren(ctx, workspaceID, canonical)
}

func (s *IssueUseCase) listIssueChildren(ctx context.Context, workspaceID string, parentIDs []string) (contract.ListIssueChildrenResponse, error) {
	if s.hierarchy == nil {
		return contract.ListIssueChildrenResponse{}, contract.ErrIssueNotImplemented
	}
	values, err := s.hierarchy.ListChildren(ctx, workspaceID, parentIDs)
	if err != nil {
		return contract.ListIssueChildrenResponse{}, fmt.Errorf("list child Issues: %w", err)
	}
	result := make([]contract.Issue, len(values))
	for index := range values {
		result[index] = issueToContract(values[index])
	}
	return contract.ListIssueChildrenResponse{Issues: result}, nil
}

func (s *IssueUseCase) ListIssueChildProgress(ctx context.Context, request contract.ListIssueChildProgressRequest) (contract.ListIssueChildProgressResponse, error) {
	workspaceID := strings.TrimSpace(request.WorkspaceID)
	if workspaceID == "" {
		return contract.ListIssueChildProgressResponse{}, fmt.Errorf("%w: workspace id is required", contract.ErrInvalidIssue)
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, workspaceID, PermissionIssueList); err != nil {
		return contract.ListIssueChildProgressResponse{}, err
	}
	if s.hierarchy == nil {
		return contract.ListIssueChildProgressResponse{}, contract.ErrIssueNotImplemented
	}
	rows, err := s.hierarchy.ChildProgress(ctx, workspaceID)
	if err != nil {
		return contract.ListIssueChildProgressResponse{}, fmt.Errorf("list child Issue progress: %w", err)
	}
	progress := make([]contract.IssueChildProgress, len(rows))
	for index, row := range rows {
		progress[index] = contract.IssueChildProgress{ParentIssueID: row.ParentIssueID, Total: row.Total, Done: row.Done}
	}
	return contract.ListIssueChildProgressResponse{Progress: progress}, nil
}

func (s *IssueUseCase) BatchUpdateIssues(ctx context.Context, request contract.BatchUpdateIssuesRequest) (contract.BatchUpdateIssuesResponse, error) {
	workspaceID := strings.TrimSpace(request.WorkspaceID)
	if workspaceID == "" {
		return contract.BatchUpdateIssuesResponse{}, fmt.Errorf("%w: workspace id is required", contract.ErrInvalidIssue)
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, workspaceID, PermissionIssueUpdate); err != nil {
		return contract.BatchUpdateIssuesResponse{}, err
	}
	issueIDs, err := cleanIssueIDs(request.IssueIDs)
	if err != nil || len(issueIDs) == 0 {
		return contract.BatchUpdateIssuesResponse{}, fmt.Errorf("%w: issue_ids is required", contract.ErrInvalidIssue)
	}
	if !request.HasMutation {
		return contract.BatchUpdateIssuesResponse{}, nil
	}
	if request.Updates.Position != nil || request.Updates.AssetIds != nil {
		return contract.BatchUpdateIssuesResponse{}, fmt.Errorf("%w: batch position and attachments are not supported", contract.ErrInvalidIssue)
	}
	patch, err := issuePatchFromUpdate(request.Updates)
	if err != nil {
		return contract.BatchUpdateIssuesResponse{}, err
	}
	if request.Updates.ProjectId != nil {
		if err := s.validateProject(ctx, workspaceID, patch.ProjectID.Value); err != nil {
			return contract.BatchUpdateIssuesResponse{}, err
		}
	}
	if request.Updates.ParentIssueId != nil && patch.ParentIssueID.Value != nil {
		parentID, canonicalErr := s.canonicalParent(ctx, workspaceID, patch.ParentIssueID.Value)
		if canonicalErr != nil {
			return contract.BatchUpdateIssuesResponse{}, canonicalErr
		}
		patch.ParentIssueID.Value = parentID
	}
	if (request.Updates.AssigneeType == nil) != (request.Updates.AssigneeId == nil) {
		return contract.BatchUpdateIssuesResponse{}, fmt.Errorf("%w: batch assignee type and id must be paired", contract.ErrInvalidIssue)
	}
	if request.Updates.AssigneeType != nil {
		if err := s.validateActorPair(ctx, workspaceID, patch.AssigneeType.Value, patch.AssigneeID.Value); err != nil {
			return contract.BatchUpdateIssuesResponse{}, err
		}
	}
	if s.hierarchy == nil {
		return contract.BatchUpdateIssuesResponse{}, contract.ErrIssueNotImplemented
	}
	values, err := s.hierarchy.BatchUpdate(ctx, IssueBatchUpdateCommand{WorkspaceID: workspaceID, IssueIDs: issueIDs, Patch: patch, Now: s.now()})
	if errors.Is(err, ErrIssueRecordNotFound) {
		return contract.BatchUpdateIssuesResponse{}, contract.ErrIssueNotFound
	}
	if err != nil {
		return contract.BatchUpdateIssuesResponse{}, fmt.Errorf("batch update Issues: %w", err)
	}
	result := make([]contract.Issue, len(values))
	for index := range values {
		result[index] = issueToContract(values[index])
	}
	return contract.BatchUpdateIssuesResponse{Updated: countToInt32(len(result)), Issues: result}, nil
}

func (s *IssueUseCase) BatchDeleteIssues(ctx context.Context, request contract.BatchDeleteIssuesRequest) (contract.BatchDeleteIssuesResponse, error) {
	workspaceID := strings.TrimSpace(request.WorkspaceID)
	if workspaceID == "" {
		return contract.BatchDeleteIssuesResponse{}, fmt.Errorf("%w: workspace id is required", contract.ErrInvalidIssue)
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, workspaceID, PermissionIssueDelete); err != nil {
		return contract.BatchDeleteIssuesResponse{}, err
	}
	issueIDs, err := cleanIssueIDs(request.IssueIDs)
	if err != nil || len(issueIDs) == 0 {
		return contract.BatchDeleteIssuesResponse{}, fmt.Errorf("%w: issue_ids is required", contract.ErrInvalidIssue)
	}
	if s.hierarchy == nil {
		return contract.BatchDeleteIssuesResponse{}, contract.ErrIssueNotImplemented
	}
	deleted, err := s.hierarchy.BatchDelete(ctx, IssueBatchDeleteCommand{WorkspaceID: workspaceID, IssueIDs: issueIDs, Now: s.now()})
	if errors.Is(err, ErrIssueRecordNotFound) {
		return contract.BatchDeleteIssuesResponse{}, contract.ErrIssueNotFound
	}
	if err != nil {
		return contract.BatchDeleteIssuesResponse{}, fmt.Errorf("batch delete Issues: %w", err)
	}
	return contract.BatchDeleteIssuesResponse{Deleted: countToInt32(len(deleted)), IssueIDs: deleted}, nil
}

func cleanIssueIDs(values []string) ([]string, error) {
	if len(values) > maximumIssueBatchSize {
		return nil, fmt.Errorf("%w: too many Issue ids", contract.ErrInvalidIssue)
	}
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, fmt.Errorf("%w: Issue id is required", contract.ErrInvalidIssue)
		}
		if _, duplicate := seen[value]; duplicate {
			return nil, fmt.Errorf("%w: duplicate Issue id", contract.ErrInvalidIssue)
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result, nil
}

var _ contract.IssueHierarchyService = (*IssueUseCase)(nil)
