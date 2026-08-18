package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	issueDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/issue"
	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

type IssueSearchQuery struct {
	WorkspaceID   string
	Phrase        string
	Terms         []string
	IncludeClosed bool
	Limit         int
	Offset        int
}

type IssueSearchResult struct {
	Issue              issueDomain.Issue
	MatchSource        string
	MatchedSnippet     string
	DescriptionSnippet string
}

type IssueSearchRepository interface {
	SearchIssues(context.Context, IssueSearchQuery) ([]IssueSearchResult, int, error)
}

type IssueSearchUseCase struct {
	repository IssueSearchRepository
	authorizer contract.WorkspaceAccessAuthorizer
}

func NewIssueSearchUseCase(repository IssueSearchRepository, authorizer contract.WorkspaceAccessAuthorizer) (*IssueSearchUseCase, error) {
	if repository == nil || authorizer == nil {
		return nil, errors.New("Issue search dependencies are required")
	}
	return &IssueSearchUseCase{repository: repository, authorizer: authorizer}, nil
}

func (s *IssueSearchUseCase) SearchIssues(ctx context.Context, request contract.SearchIssuesRequest) (contract.SearchIssuesResponse, error) {
	workspaceID := strings.TrimSpace(request.WorkspaceID)
	phrase := NormalizeIssueSearchText(request.Query)
	if workspaceID == "" || phrase == "" || request.Limit < 0 || request.Offset < 0 {
		return contract.SearchIssuesResponse{}, contract.ErrInvalidIssueSearch
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, workspaceID, contract.PermissionSearchReadable); err != nil {
		return contract.SearchIssuesResponse{}, err
	}
	limit := request.Limit
	if limit == 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}
	values, total, err := s.repository.SearchIssues(ctx, IssueSearchQuery{
		WorkspaceID: workspaceID, Phrase: phrase, Terms: strings.Fields(phrase),
		IncludeClosed: request.IncludeClosed, Limit: limit, Offset: request.Offset,
	})
	if err != nil {
		return contract.SearchIssuesResponse{}, fmt.Errorf("search Issues: %w", err)
	}
	results := make([]contract.IssueSearchResult, len(values))
	for index, value := range values {
		issue := issueToContract(value.Issue)
		results[index] = contract.IssueSearchResult{Issue: issue, MatchSource: value.MatchSource}
		if value.MatchedSnippet != "" {
			snippet := value.MatchedSnippet
			results[index].MatchedSnippet = &snippet
		}
		if value.DescriptionSnippet != "" {
			snippet := value.DescriptionSnippet
			results[index].DescriptionSnippet = &snippet
		}
	}
	return contract.SearchIssuesResponse{Issues: results, Total: total}, nil
}

var issueSearchFold = cases.Fold()

func NormalizeIssueSearchText(value string) string {
	value = issueSearchFold.String(norm.NFKC.String(value))
	var builder strings.Builder
	space := true
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			builder.WriteRune(r)
			space = false
			continue
		}
		if !space {
			builder.WriteByte(' ')
			space = true
		}
	}
	return strings.TrimSpace(builder.String())
}
