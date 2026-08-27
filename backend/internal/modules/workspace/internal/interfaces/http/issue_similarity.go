package http

import (
	"errors"
	"net/http"
	"strings"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

type IssueSimilarityHandler struct {
	service      contract.IssueSimilarityService
	issues       contract.IssueMutationService
	identity     contract.WorkspaceHTTPIdentityResolver
	authenticate func(*http.Request) (string, error)
}

func NewIssueSimilarityHandler(service contract.IssueSimilarityService, issues contract.IssueMutationService, identity contract.WorkspaceHTTPIdentityResolver, authenticate func(*http.Request) (string, error)) *IssueSimilarityHandler {
	return &IssueSimilarityHandler{service: service, issues: issues, identity: identity, authenticate: authenticate}
}

func (h *IssueSimilarityHandler) Register(server *kratoshttp.Server) {
	router := server.Route("/")
	router.POST("/api/issues/similarity/check", h.check)
	router.POST("/api/issues/{id}/similarity/check", h.checkExisting)
}

type issueSimilarityHTTPRequest struct {
	Title         string  `json:"title"`
	Description   *string `json:"description"`
	ProjectID     *string `json:"project_id"`
	IncludeClosed bool    `json:"include_closed"`
}

func (h *IssueSimilarityHandler) check(ctx kratoshttp.Context) error {
	identity, ok := h.resolveIdentity(ctx)
	if !ok {
		return nil
	}
	var request issueSimilarityHTTPRequest
	if err := decodeJSON(ctx.Request().Body, &request); err != nil {
		return writeError(ctx, http.StatusBadRequest, "invalid issue similarity request")
	}
	return h.writeCheck(ctx, identity, contract.CheckIssueSimilarityRequest{
		WorkspaceID: identity.WorkspaceID, Title: request.Title, Description: request.Description,
		ProjectID: request.ProjectID, IncludeClosed: request.IncludeClosed,
	})
}

func (h *IssueSimilarityHandler) checkExisting(ctx kratoshttp.Context) error {
	identity, ok := h.resolveIdentity(ctx)
	if !ok {
		return nil
	}
	issueID := strings.TrimSpace(ctx.Vars().Get("id"))
	issue, err := h.issues.GetIssue(workspaceActorContext(ctx, identity), contract.GetIssueRequest{
		WorkspaceId: identity.WorkspaceID, IssueId: issueID,
	})
	if errors.Is(err, contract.ErrIssueNotFound) || issue.Issue == nil {
		return writeError(ctx, http.StatusNotFound, "issue not found")
	}
	if err != nil {
		return writeError(ctx, http.StatusInternalServerError, "failed to get issue")
	}
	return h.writeCheck(ctx, identity, contract.CheckIssueSimilarityRequest{
		WorkspaceID: identity.WorkspaceID, Title: issue.Issue.Title, Description: issue.Issue.Description,
		ProjectID: issue.Issue.ProjectId, ExcludeIssueID: issue.Issue.Id,
	})
}

func (h *IssueSimilarityHandler) writeCheck(ctx kratoshttp.Context, identity contract.WorkspaceHTTPIdentity, request contract.CheckIssueSimilarityRequest) error {
	result, err := h.service.CheckIssueSimilarity(workspaceActorContext(ctx, identity), request)
	if errors.Is(err, contract.ErrInvalidIssueSimilarity) {
		return writeError(ctx, http.StatusBadRequest, "invalid issue similarity request")
	}
	if errors.Is(err, contract.ErrWorkspacePermissionDenied) || errors.Is(err, contract.ErrWorkspaceActorRequired) {
		return writeError(ctx, http.StatusForbidden, "issue similarity denied")
	}
	if err != nil {
		return writeError(ctx, http.StatusInternalServerError, "failed to check issue similarity")
	}
	candidates := make([]publicIssueSimilarityCandidate, len(result.Candidates))
	for index, candidate := range result.Candidates {
		candidates[index] = publicIssueSimilarityCandidate{
			publicIssue: toPublicIssue(candidate.Issue), Score: candidate.Score, ComponentScores: candidate.ComponentScores,
			SameProject: candidate.SameProject, Closed: candidate.Closed,
		}
	}
	return ctx.JSON(http.StatusOK, map[string]any{
		"ranking_version": result.RankingVersion, "candidates": candidates, "truncated": result.Truncated,
		"detector_available": result.DetectorAvailable,
	})
}

type publicIssueSimilarityCandidate struct {
	publicIssue
	Score           int                `json:"score"`
	ComponentScores map[string]float64 `json:"component_scores"`
	SameProject     bool               `json:"same_project"`
	Closed          bool               `json:"closed"`
}

func (h *IssueSimilarityHandler) resolveIdentity(ctx kratoshttp.Context) (contract.WorkspaceHTTPIdentity, bool) {
	if h.authenticate == nil {
		_ = writeError(ctx, http.StatusUnauthorized, "user not authenticated")
		return contract.WorkspaceHTTPIdentity{}, false
	}
	if _, err := h.authenticate(ctx.Request()); err != nil {
		_ = writeError(ctx, http.StatusUnauthorized, "user not authenticated")
		return contract.WorkspaceHTTPIdentity{}, false
	}
	if !hasWorkspaceIdentity(ctx) {
		_ = writeError(ctx, http.StatusBadRequest, "workspace is required")
		return contract.WorkspaceHTTPIdentity{}, false
	}
	if h.identity == nil {
		_ = writeError(ctx, http.StatusUnauthorized, "user not authenticated")
		return contract.WorkspaceHTTPIdentity{}, false
	}
	identity, err := h.identity(ctx.Request())
	if errors.Is(err, contract.ErrActorOutsideWorkspace) || errors.Is(err, contract.ErrWorkspaceNotFound) {
		_ = writeError(ctx, http.StatusNotFound, "issue not found")
		return contract.WorkspaceHTTPIdentity{}, false
	}
	if err != nil {
		_ = writeError(ctx, http.StatusUnauthorized, "user not authenticated")
		return contract.WorkspaceHTTPIdentity{}, false
	}
	return identity, true
}
