package contract

import "context"

type ListIssueChildrenRequest struct {
	WorkspaceID string
	IssueID     string
}

type ListIssueChildrenResponse struct {
	Issues []Issue
}

type ListIssueChildrenByParentsRequest struct {
	WorkspaceID string
	ParentIDs   []string
}

type IssueChildProgress struct {
	ParentIssueID string
	Total         int32
	Done          int32
}

type ListIssueChildProgressRequest struct {
	WorkspaceID string
}

type ListIssueChildProgressResponse struct {
	Progress []IssueChildProgress
}

type BatchUpdateIssuesRequest struct {
	WorkspaceID string
	IssueIDs    []string
	Updates     UpdateIssueRequest
	HasMutation bool
}

type BatchUpdateIssuesResponse struct {
	Updated int32
	Issues  []Issue
}

type BatchDeleteIssuesRequest struct {
	WorkspaceID string
	IssueIDs    []string
}

type BatchDeleteIssuesResponse struct {
	Deleted  int32
	IssueIDs []string
}

type IssueHierarchyService interface {
	ListIssueChildren(context.Context, ListIssueChildrenRequest) (ListIssueChildrenResponse, error)
	ListIssueChildrenByParents(context.Context, ListIssueChildrenByParentsRequest) (ListIssueChildrenResponse, error)
	ListIssueChildProgress(context.Context, ListIssueChildProgressRequest) (ListIssueChildProgressResponse, error)
	BatchUpdateIssues(context.Context, BatchUpdateIssuesRequest) (BatchUpdateIssuesResponse, error)
	BatchDeleteIssues(context.Context, BatchDeleteIssuesRequest) (BatchDeleteIssuesResponse, error)
}
