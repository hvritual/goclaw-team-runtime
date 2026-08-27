package application

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	issueDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/issue"
)

const (
	issueSimilarityRankingVersion = "lexical-v1"
	issueSimilarityCandidateLimit = 5
	issueSimilarityPoolLimit      = 50
)

type IssueSimilarityQuery struct {
	WorkspaceID    string
	Title          string
	Description    string
	ProjectID      string
	IncludeClosed  bool
	ExcludeIssueID string
	Limit          int
}

type IssueSimilarityCandidate struct {
	Issue           issueDomain.Issue
	Score           int
	ComponentScores map[string]float64
	SameProject     bool
	Closed          bool
}

type IssueSimilarityRepository interface {
	FindIssueSimilarityCandidates(context.Context, IssueSimilarityQuery) ([]IssueSimilarityCandidate, bool, error)
}

type IssueSimilarityUseCase struct {
	repository IssueSimilarityRepository
	authorizer contract.WorkspaceAccessAuthorizer
}

func NewIssueSimilarityUseCase(repository IssueSimilarityRepository, authorizer contract.WorkspaceAccessAuthorizer) (*IssueSimilarityUseCase, error) {
	if repository == nil || authorizer == nil {
		return nil, errors.New("Issue similarity dependencies are required")
	}
	return &IssueSimilarityUseCase{repository: repository, authorizer: authorizer}, nil
}

func (s *IssueSimilarityUseCase) CheckIssueSimilarity(ctx context.Context, request contract.CheckIssueSimilarityRequest) (contract.CheckIssueSimilarityResponse, error) {
	workspaceID := strings.TrimSpace(request.WorkspaceID)
	title := NormalizeIssueSearchText(request.Title)
	if workspaceID == "" || title == "" {
		return contract.CheckIssueSimilarityResponse{}, contract.ErrInvalidIssueSimilarity
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, workspaceID, contract.PermissionSimilarityCheck); err != nil {
		return contract.CheckIssueSimilarityResponse{}, err
	}
	description := ""
	if request.Description != nil {
		description = NormalizeIssueSearchText(*request.Description)
	}
	projectID := ""
	if request.ProjectID != nil {
		projectID = strings.TrimSpace(*request.ProjectID)
	}
	candidates, truncated, err := s.repository.FindIssueSimilarityCandidates(ctx, IssueSimilarityQuery{
		WorkspaceID: workspaceID, Title: title, Description: description, ProjectID: projectID,
		IncludeClosed: request.IncludeClosed, ExcludeIssueID: strings.TrimSpace(request.ExcludeIssueID), Limit: issueSimilarityPoolLimit,
	})
	if errors.Is(err, contract.ErrIssueSimilarityUnavailable) {
		return contract.CheckIssueSimilarityResponse{
			RankingVersion:    issueSimilarityRankingVersion,
			Candidates:        make([]contract.IssueSimilarityCandidate, 0),
			DetectorAvailable: false,
		}, nil
	}
	if err != nil {
		return contract.CheckIssueSimilarityResponse{}, fmt.Errorf("check Issue similarity: %w", err)
	}
	if len(candidates) > issueSimilarityCandidateLimit {
		candidates = candidates[:issueSimilarityCandidateLimit]
		truncated = true
	}
	result := make([]contract.IssueSimilarityCandidate, len(candidates))
	for index, candidate := range candidates {
		result[index] = contract.IssueSimilarityCandidate{
			Issue: issueToContract(candidate.Issue), Score: candidate.Score,
			ComponentScores: candidate.ComponentScores, SameProject: candidate.SameProject, Closed: candidate.Closed,
		}
	}
	return contract.CheckIssueSimilarityResponse{
		RankingVersion: issueSimilarityRankingVersion, Candidates: result, Truncated: truncated, DetectorAvailable: true,
	}, nil
}
