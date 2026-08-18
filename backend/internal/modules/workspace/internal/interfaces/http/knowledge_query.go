package http

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

type KnowledgeQueryHandler struct {
	service      contract.KnowledgeQueryService
	identity     contract.WorkspaceHTTPIdentityResolver
	authenticate func(*http.Request) (string, error)
}

func NewKnowledgeQueryHandler(service contract.KnowledgeQueryService, identity contract.WorkspaceHTTPIdentityResolver, authenticate func(*http.Request) (string, error)) *KnowledgeQueryHandler {
	return &KnowledgeQueryHandler{service: service, identity: identity, authenticate: authenticate}
}

func (h *KnowledgeQueryHandler) Register(server *kratoshttp.Server) {
	router := server.Route("/")
	router.GET("/api/knowledge", h.list)
	router.GET("/api/knowledge/{id}", h.detail)
}

func (h *KnowledgeQueryHandler) list(ctx kratoshttp.Context) error {
	identity, ok := h.resolve(ctx)
	if !ok {
		return nil
	}
	values := ctx.Request().URL.Query()
	allowed := map[string]bool{"query": true, "status": true, "kind": true, "source_type": true, "source_id": true, "source_revision": true, "applicability": true, "project_id": true, "revision": true, "limit": true, "cursor": true}
	for key := range values {
		if !allowed[key] {
			return writeError(ctx, http.StatusBadRequest, contract.ErrInvalidKnowledgeQuery.Error())
		}
	}
	revision, err := strictKnowledgeInteger(values.Get("revision"))
	if err != nil {
		return writeError(ctx, http.StatusBadRequest, contract.ErrInvalidKnowledgeQuery.Error())
	}
	limit, err := strictKnowledgeInteger(values.Get("limit"))
	if err != nil {
		return writeError(ctx, http.StatusBadRequest, contract.ErrInvalidKnowledgeQuery.Error())
	}
	result, err := h.service.QueryKnowledge(workspaceActorContext(ctx, identity), contract.QueryKnowledgeRequest{
		WorkspaceID: identity.WorkspaceID, Query: values.Get("query"), Statuses: values["status"], Kinds: values["kind"],
		SourceType: values.Get("source_type"), SourceID: values.Get("source_id"), SourceRevision: values.Get("source_revision"),
		Applicability: values.Get("applicability"), ProjectID: values.Get("project_id"), Revision: revision, Limit: limit, Cursor: values.Get("cursor"),
	})
	if errors.Is(err, contract.ErrInvalidKnowledgeQuery) {
		return writeError(ctx, http.StatusBadRequest, err.Error())
	}
	if errors.Is(err, contract.ErrWorkspacePermissionDenied) || errors.Is(err, contract.ErrWorkspaceActorRequired) {
		return writeError(ctx, http.StatusForbidden, "Knowledge query denied")
	}
	if err != nil {
		return writeError(ctx, http.StatusInternalServerError, "failed to query Knowledge")
	}
	return ctx.JSON(http.StatusOK, result)
}

func (h *KnowledgeQueryHandler) detail(ctx kratoshttp.Context) error {
	identity, ok := h.resolve(ctx)
	if !ok {
		return nil
	}
	if len(ctx.Request().URL.Query()) > 0 {
		return writeError(ctx, http.StatusBadRequest, contract.ErrInvalidKnowledgeQuery.Error())
	}
	result, err := h.service.GetGovernedKnowledge(workspaceActorContext(ctx, identity), identity.WorkspaceID, ctx.Vars().Get("id"))
	if errors.Is(err, contract.ErrKnowledgeQueryHidden) {
		return writeError(ctx, http.StatusNotFound, "Knowledge not found")
	}
	if errors.Is(err, contract.ErrInvalidKnowledgeQuery) {
		return writeError(ctx, http.StatusBadRequest, err.Error())
	}
	if errors.Is(err, contract.ErrWorkspacePermissionDenied) || errors.Is(err, contract.ErrWorkspaceActorRequired) {
		return writeError(ctx, http.StatusForbidden, "Knowledge query denied")
	}
	if err != nil {
		return writeError(ctx, http.StatusInternalServerError, "failed to get Knowledge")
	}
	return ctx.JSON(http.StatusOK, result)
}

func (h *KnowledgeQueryHandler) resolve(ctx kratoshttp.Context) (contract.WorkspaceHTTPIdentity, bool) {
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
		_ = writeError(ctx, http.StatusForbidden, "Knowledge query denied")
		return contract.WorkspaceHTTPIdentity{}, false
	}
	identity, err := h.identity(ctx.Request())
	if err != nil {
		_ = issueReadIdentityError(ctx, err)
		return contract.WorkspaceHTTPIdentity{}, false
	}
	return identity, true
}

func strictKnowledgeInteger(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0, contract.ErrInvalidKnowledgeQuery
	}
	return value, nil
}
