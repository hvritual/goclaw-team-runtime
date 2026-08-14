package contract

import "context"

type MoveIssueRequest struct {
	WorkspaceID   string
	IssueID       string
	Status        *string
	AssigneeType  *string
	AssigneeID    *string
	ParentIssueID *string
	ProjectID     *string
	BeforeID      *string
	AfterID       *string
}

type MoveIssueResponse struct {
	Issue *Issue
}

type IssueMoveService interface {
	MoveIssue(context.Context, MoveIssueRequest) (MoveIssueResponse, error)
}

type IssueMutationService interface {
	IssueService
	IssueMoveService
}
