package contract

import "context"

type DeleteIssueRequest struct {
	WorkspaceID string
	IssueID     string
}

type DeleteIssueResponse struct{ IssueID string }

type IssueDeletionService interface {
	DeleteIssue(context.Context, DeleteIssueRequest) (DeleteIssueResponse, error)
}
