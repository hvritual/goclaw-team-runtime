// dddgen:service-implementation IssueService; method bodies are user-owned.
package application

import (
	"context"

	"github.com/multica-ai/multica/server/internal/modules/workspace/contract"
)

type IssueService struct{}

func NewIssueService() *IssueService { return &IssueService{} }

func (s *IssueService) UpdateIssueStatus(ctx context.Context, request contract.Issue_UpdateIssueStatusRequest) (contract.Issue_Issue, error) {
	return contract.Issue_Issue{}, contract.ErrIssueNotImplemented
}
