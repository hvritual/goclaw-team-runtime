// dddgen:service-implementation IssueService; method bodies are user-owned.
package application

import (
	"context"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

type IssueService struct{}

func NewIssueService() *IssueService { return &IssueService{} }

func (s *IssueService) UpdateIssueStatus(ctx context.Context, request contract.UpdateIssueStatusRequest) (contract.UpdateIssueStatusResponse, error) {
	return contract.UpdateIssueStatusResponse{}, contract.ErrIssueNotImplemented
}
func (s *IssueService) CreateIssue(ctx context.Context, request contract.CreateIssueRequest) (contract.CreateIssueResponse, error) {
	return contract.CreateIssueResponse{}, contract.ErrIssueNotImplemented
}
func (s *IssueService) GetIssue(ctx context.Context, request contract.GetIssueRequest) (contract.GetIssueResponse, error) {
	return contract.GetIssueResponse{}, contract.ErrIssueNotImplemented
}
func (s *IssueService) ListIssues(ctx context.Context, request contract.ListIssuesRequest) (contract.ListIssuesResponse, error) {
	return contract.ListIssuesResponse{}, contract.ErrIssueNotImplemented
}
func (s *IssueService) UpdateIssue(ctx context.Context, request contract.UpdateIssueRequest) (contract.UpdateIssueResponse, error) {
	return contract.UpdateIssueResponse{}, contract.ErrIssueNotImplemented
}
