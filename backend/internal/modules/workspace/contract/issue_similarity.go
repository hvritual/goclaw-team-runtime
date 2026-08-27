package contract

import (
	"context"
	"errors"
)

var (
	ErrInvalidIssueSimilarity     = errors.New("invalid Issue similarity request")
	ErrIssueSimilarityUnavailable = errors.New("Issue similarity detector unavailable")
)

type CheckIssueSimilarityRequest struct {
	WorkspaceID    string
	Title          string
	Description    *string
	ProjectID      *string
	IncludeClosed  bool
	ExcludeIssueID string
}

type IssueSimilarityCandidate struct {
	Issue           Issue
	Score           int
	ComponentScores map[string]float64
	SameProject     bool
	Closed          bool
}

type CheckIssueSimilarityResponse struct {
	RankingVersion    string
	Candidates        []IssueSimilarityCandidate
	Truncated         bool
	DetectorAvailable bool
}

type IssueSimilarityService interface {
	CheckIssueSimilarity(context.Context, CheckIssueSimilarityRequest) (CheckIssueSimilarityResponse, error)
}
