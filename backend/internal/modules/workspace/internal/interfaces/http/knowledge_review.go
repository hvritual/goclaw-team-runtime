package http

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

type KnowledgeReviewHandler struct {
	service      contract.KnowledgeReviewService
	identity     contract.WorkspaceHTTPIdentityResolver
	authenticate func(*http.Request) (string, error)
	mutation     func(*http.Request) error
}

func NewKnowledgeReviewHandler(service contract.KnowledgeReviewService, identity contract.WorkspaceHTTPIdentityResolver, authenticate func(*http.Request) (string, error), mutation func(*http.Request) error) *KnowledgeReviewHandler {
	return &KnowledgeReviewHandler{service: service, identity: identity, authenticate: authenticate, mutation: mutation}
}
func (h *KnowledgeReviewHandler) Register(server *kratoshttp.Server) {
	router := server.Route("/")
	router.POST("/api/knowledge/proposals", h.propose)
	router.GET("/api/knowledge/candidates", h.list)
	router.POST("/api/knowledge/candidates/{id}/review", h.review)
}

func (h *KnowledgeReviewHandler) propose(ctx kratoshttp.Context) error {
	identity, ok := h.resolve(ctx)
	if !ok {
		return nil
	}
	if h.mutation == nil || h.mutation(ctx.Request()) != nil {
		return writeError(ctx, http.StatusForbidden, "invalid CSRF token")
	}
	key := strings.TrimSpace(ctx.Request().Header.Get("Idempotency-Key"))
	if key == "" {
		return writeError(ctx, http.StatusBadRequest, "idempotency key is required")
	}
	var input struct {
		ProjectID   *string                       `json:"project_id"`
		KnowledgeID *string                       `json:"knowledge_id"`
		Kind        string                        `json:"kind"`
		Title       string                        `json:"title"`
		Content     string                        `json:"content"`
		Reason      string                        `json:"reason"`
		SourceRefs  []contract.KnowledgeSourceRef `json:"source_refs"`
	}
	if decodeJSON(ctx.Request().Body, &input) != nil {
		return writeError(ctx, http.StatusBadRequest, "invalid request body")
	}
	value, err := h.service.ProposeKnowledge(workspaceActorContext(ctx, identity), contract.ProposeKnowledgeRequest{WorkspaceID: identity.WorkspaceID, IdempotencyKey: key, ProjectID: input.ProjectID, KnowledgeID: input.KnowledgeID, Kind: input.Kind, Title: input.Title, Content: input.Content, Reason: input.Reason, SourceRefs: input.SourceRefs})
	if err != nil {
		return h.writeError(ctx, err)
	}
	return ctx.JSON(http.StatusCreated, value)
}

func (h *KnowledgeReviewHandler) list(ctx kratoshttp.Context) error {
	identity, ok := h.resolve(ctx)
	if !ok {
		return nil
	}
	values := ctx.Request().URL.Query()
	for key := range values {
		if key != "limit" && key != "cursor" {
			return writeError(ctx, http.StatusBadRequest, contract.ErrInvalidKnowledgeReview.Error())
		}
	}
	limit := 0
	if raw, present := values["limit"]; present {
		if len(raw) != 1 || strings.TrimSpace(raw[0]) == "" {
			return writeError(ctx, http.StatusBadRequest, contract.ErrInvalidKnowledgeReview.Error())
		}
		parsed, err := strconv.Atoi(raw[0])
		if err != nil || parsed <= 0 {
			return writeError(ctx, http.StatusBadRequest, contract.ErrInvalidKnowledgeReview.Error())
		}
		limit = parsed
	}
	if raw, present := values["cursor"]; present && (len(raw) != 1 || strings.TrimSpace(raw[0]) == "") {
		return writeError(ctx, http.StatusBadRequest, contract.ErrInvalidKnowledgeReview.Error())
	}
	result, err := h.service.ListKnowledgeCandidates(workspaceActorContext(ctx, identity), contract.ListKnowledgeCandidatesRequest{WorkspaceID: identity.WorkspaceID, Limit: limit, Cursor: values.Get("cursor")})
	if err != nil {
		return h.writeError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, result)
}

func (h *KnowledgeReviewHandler) review(ctx kratoshttp.Context) error {
	identity, ok := h.resolve(ctx)
	if !ok {
		return nil
	}
	if h.mutation == nil || h.mutation(ctx.Request()) != nil {
		return writeError(ctx, http.StatusForbidden, "invalid CSRF token")
	}
	var input struct {
		Action           string `json:"action"`
		ExpectedRevision int    `json:"expected_revision"`
		Rationale        string `json:"rationale"`
		Emergency        bool   `json:"emergency"`
	}
	if decodeJSON(ctx.Request().Body, &input) != nil {
		return writeError(ctx, http.StatusBadRequest, "invalid request body")
	}
	result, err := h.service.ReviewKnowledge(workspaceActorContext(ctx, identity), contract.ReviewKnowledgeRequest{WorkspaceID: identity.WorkspaceID, CandidateID: ctx.Vars().Get("id"), Action: input.Action, ExpectedRevision: input.ExpectedRevision, Rationale: input.Rationale, Emergency: input.Emergency})
	if err != nil {
		return h.writeError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, result)
}

func (h *KnowledgeReviewHandler) resolve(ctx kratoshttp.Context) (contract.WorkspaceHTTPIdentity, bool) {
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
		_ = writeError(ctx, http.StatusForbidden, "Knowledge review denied")
		return contract.WorkspaceHTTPIdentity{}, false
	}
	identity, err := h.identity(ctx.Request())
	if err != nil {
		_ = issueReadIdentityError(ctx, err)
		return contract.WorkspaceHTTPIdentity{}, false
	}
	return identity, true
}

func (h *KnowledgeReviewHandler) writeError(ctx kratoshttp.Context, err error) error {
	var conflict *contract.KnowledgeRevisionConflictError
	switch {
	case errors.Is(err, contract.ErrInvalidKnowledgeReview):
		return writeError(ctx, http.StatusBadRequest, err.Error())
	case errors.Is(err, contract.ErrKnowledgeCandidateNotFound):
		return writeError(ctx, http.StatusNotFound, err.Error())
	case errors.As(err, &conflict):
		return ctx.JSON(http.StatusConflict, map[string]any{"code": "revision_conflict", "resource": conflict.Resource, "current_revision": conflict.CurrentRevision})
	case errors.Is(err, contract.ErrKnowledgeIdempotencyConflict):
		return ctx.JSON(http.StatusConflict, map[string]any{"code": "idempotency_conflict"})
	case errors.Is(err, contract.ErrKnowledgeSelfReview):
		return writeError(ctx, http.StatusForbidden, err.Error())
	case errors.Is(err, contract.ErrWorkspacePermissionDenied), errors.Is(err, contract.ErrWorkspaceActorRequired), errors.Is(err, contract.ErrAssetOutsideWorkspace):
		return writeError(ctx, http.StatusForbidden, "Knowledge review denied")
	default:
		return writeError(ctx, http.StatusInternalServerError, "Knowledge review failed")
	}
}
