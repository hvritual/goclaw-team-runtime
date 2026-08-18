package contract

import (
	"context"
	"errors"
)

var ErrInvalidIssueSearch = errors.New("invalid Issue search")

type SearchIssuesRequest struct {
	WorkspaceID   string
	Query         string
	Limit         int
	Offset        int
	IncludeClosed bool
}

type IssueSearchResult struct {
	Issue              Issue
	MatchSource        string
	MatchedSnippet     *string
	DescriptionSnippet *string
}

type SearchIssuesResponse struct {
	Issues []IssueSearchResult
	Total  int
}

type IssueSearchService interface {
	SearchIssues(context.Context, SearchIssuesRequest) (SearchIssuesResponse, error)
}
