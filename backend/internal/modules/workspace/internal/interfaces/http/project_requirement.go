package http

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
)

type ProjectRequirementHandler struct {
	service      contract.ProjectRequirementService
	identity     contract.WorkspaceHTTPIdentityResolver
	authenticate func(*http.Request) (string, error)
	mutation     func(*http.Request) error
}

func NewProjectRequirementHandler(service contract.ProjectRequirementService, identity contract.WorkspaceHTTPIdentityResolver, authenticate func(*http.Request) (string, error), mutation func(*http.Request) error) *ProjectRequirementHandler {
	return &ProjectRequirementHandler{service: service, identity: identity, authenticate: authenticate, mutation: mutation}
}

func (h *ProjectRequirementHandler) Register(server *kratoshttp.Server) {
	router := server.Route("/")
	router.GET("/api/projects/{id}/requirement-baseline", h.get)
	router.GET("/api/projects/{id}/requirement-baseline/coverage", h.getCoverage)
	router.PUT("/api/projects/{id}/requirement-baseline", h.save)
	router.POST("/api/projects/{id}/requirement-baseline/submit-review", h.transition("submit-review"))
	router.POST("/api/projects/{id}/requirement-baseline/withdraw", h.transition("withdraw"))
	router.POST("/api/projects/{id}/requirement-baseline/approve", h.transition("approve"))
	router.POST("/api/projects/{id}/requirement-baseline/freeze", h.transition("freeze"))
	router.POST("/api/projects/{id}/requirement-baseline/retire", h.transition("retire"))
	router.POST("/api/projects/{id}/requirement-baseline/links", h.mutateIssueLink(false))
	router.DELETE("/api/projects/{id}/requirement-baseline/links/{requirement_key}/{issue_id}", h.mutateIssueLink(true))
	router.POST("/api/projects/{id}/requirement-baseline/outline-links", h.mutateOutlineLink(false))
	router.DELETE("/api/projects/{id}/requirement-baseline/outline-links/{requirement_key}/{node_id}", h.mutateOutlineLink(true))
	router.GET("/api/projects/{id}/requirement-baseline/access", h.getAccess)
	router.PUT("/api/projects/{id}/requirement-baseline/access", h.replaceAccess)
	router.GET("/api/projects/{id}/outline", h.getOutline)
	router.POST("/api/projects/{id}/outline", h.createOutline)

	// S10 and Requirement-driven Issue creation are intentionally frozen.
	router.PATCH("/api/projects/{id}/outline/{node_id}", h.featureUnavailable)
	router.DELETE("/api/projects/{id}/outline/{node_id}", h.featureUnavailable)
	router.POST("/api/projects/{id}/outline/reorder", h.featureUnavailable)
	router.POST("/api/projects/{id}/outline/{node_id}/issues", h.featureUnavailable)
	router.DELETE("/api/projects/{id}/outline/{node_id}/issues/{issue_id}", h.featureUnavailable)
	router.POST("/api/projects/{id}/requirement-baseline/items/{requirement_key}/issues", h.featureUnavailable)
}

func (h *ProjectRequirementHandler) getCoverage(ctx kratoshttp.Context) error {
	identity, ok := h.resolveIdentity(ctx)
	if !ok {
		return nil
	}
	result, err := h.service.GetProjectRequirementCoverage(workspaceActorContext(ctx, identity), identity.WorkspaceID, ctx.Vars().Get("id"))
	if err != nil {
		return h.writeFailure(ctx, err, "failed to read Project Requirement coverage")
	}
	return ctx.JSON(http.StatusOK, result)
}

func (h *ProjectRequirementHandler) get(ctx kratoshttp.Context) error {
	identity, ok := h.resolveIdentity(ctx)
	if !ok {
		return nil
	}
	result, err := h.service.GetProjectRequirement(workspaceActorContext(ctx, identity), identity.WorkspaceID, ctx.Vars().Get("id"))
	if err != nil {
		return h.writeFailure(ctx, err, "failed to read Project Requirement")
	}
	return ctx.JSON(http.StatusOK, result)
}

func (h *ProjectRequirementHandler) save(ctx kratoshttp.Context) error {
	identity, ok := h.resolveMutationIdentity(ctx)
	if !ok {
		return nil
	}
	var request contract.SaveProjectRequirementDraftRequest
	if err := decodeJSON(ctx.Request().Body, &request); err != nil {
		return writeError(ctx, http.StatusBadRequest, "invalid request body")
	}
	key := strings.TrimSpace(ctx.Request().Header.Get("Idempotency-Key"))
	if request.ExpectedRevision == 0 && key == "" {
		return writeError(ctx, http.StatusBadRequest, "idempotency key is required")
	}
	result, err := h.service.SaveProjectRequirement(workspaceActorContext(ctx, identity), identity.WorkspaceID, ctx.Vars().Get("id"), key, request)
	if err != nil {
		return h.writeFailure(ctx, err, "failed to save Project Requirement")
	}
	statusCode := http.StatusOK
	if request.ExpectedRevision == 0 {
		statusCode = http.StatusCreated
	}
	return ctx.JSON(statusCode, result)
}

func (h *ProjectRequirementHandler) transition(action string) func(kratoshttp.Context) error {
	return func(ctx kratoshttp.Context) error {
		identity, ok := h.resolveMutationIdentity(ctx)
		if !ok {
			return nil
		}
		var request contract.ProjectRequirementTransitionRequest
		if err := decodeJSON(ctx.Request().Body, &request); err != nil {
			return writeError(ctx, http.StatusBadRequest, "invalid request body")
		}
		result, err := h.service.TransitionProjectRequirement(workspaceActorContext(ctx, identity), identity.WorkspaceID, ctx.Vars().Get("id"), action, request)
		if err != nil {
			return h.writeFailure(ctx, err, "failed to transition Project Requirement")
		}
		return ctx.JSON(http.StatusOK, result)
	}
}

func (h *ProjectRequirementHandler) mutateIssueLink(unlink bool) func(kratoshttp.Context) error {
	return func(ctx kratoshttp.Context) error {
		identity, ok := h.resolveMutationIdentity(ctx)
		if !ok {
			return nil
		}
		var request contract.ProjectRequirementIssueLinkRequest
		if unlink {
			expected, valid := exactExpectedRevisionQuery(ctx)
			if !valid {
				return writeError(ctx, http.StatusBadRequest, "expected_revision is required")
			}
			request = contract.ProjectRequirementIssueLinkRequest{
				ExpectedRevision: expected,
				RequirementKey:   ctx.Vars().Get("requirement_key"),
				IssueID:          ctx.Vars().Get("issue_id"),
			}
		} else if err := decodeJSON(ctx.Request().Body, &request); err != nil {
			return writeError(ctx, http.StatusBadRequest, "invalid request body")
		}
		result, err := h.service.MutateProjectRequirementIssueLink(workspaceActorContext(ctx, identity), identity.WorkspaceID, ctx.Vars().Get("id"), request, unlink)
		if err != nil {
			return h.writeFailure(ctx, err, "failed to update Project Requirement Issue link")
		}
		if unlink {
			ctx.Response().WriteHeader(http.StatusNoContent)
			return nil
		}
		return ctx.JSON(http.StatusOK, result)
	}
}

func (h *ProjectRequirementHandler) mutateOutlineLink(unlink bool) func(kratoshttp.Context) error {
	return func(ctx kratoshttp.Context) error {
		identity, ok := h.resolveMutationIdentity(ctx)
		if !ok {
			return nil
		}
		var request contract.ProjectRequirementOutlineLinkRequest
		if unlink {
			expected, valid := exactExpectedRevisionQuery(ctx)
			if !valid {
				return writeError(ctx, http.StatusBadRequest, "expected_revision is required")
			}
			request = contract.ProjectRequirementOutlineLinkRequest{
				ExpectedRevision: expected,
				RequirementKey:   ctx.Vars().Get("requirement_key"),
				NodeID:           ctx.Vars().Get("node_id"),
			}
		} else if err := decodeJSON(ctx.Request().Body, &request); err != nil {
			return writeError(ctx, http.StatusBadRequest, "invalid request body")
		}
		result, err := h.service.MutateProjectRequirementOutlineLink(workspaceActorContext(ctx, identity), identity.WorkspaceID, ctx.Vars().Get("id"), request, unlink)
		if err != nil {
			return h.writeFailure(ctx, err, "failed to update Project Requirement outline link")
		}
		if unlink {
			ctx.Response().WriteHeader(http.StatusNoContent)
			return nil
		}
		return ctx.JSON(http.StatusOK, result)
	}
}

func (h *ProjectRequirementHandler) getAccess(ctx kratoshttp.Context) error {
	identity, ok := h.resolveIdentity(ctx)
	if !ok {
		return nil
	}
	result, err := h.service.GetProjectRequirementAccess(workspaceActorContext(ctx, identity), identity.WorkspaceID, ctx.Vars().Get("id"))
	if err != nil {
		return h.writeFailure(ctx, err, "failed to read Project Requirement access")
	}
	return ctx.JSON(http.StatusOK, result)
}

func (h *ProjectRequirementHandler) replaceAccess(ctx kratoshttp.Context) error {
	identity, ok := h.resolveMutationIdentity(ctx)
	if !ok {
		return nil
	}
	var request contract.ReplaceProjectRequirementAccessRequest
	if err := decodeJSON(ctx.Request().Body, &request); err != nil {
		return writeError(ctx, http.StatusBadRequest, "invalid request body")
	}
	result, err := h.service.ReplaceProjectRequirementAccess(workspaceActorContext(ctx, identity), identity.WorkspaceID, ctx.Vars().Get("id"), request)
	if err != nil {
		return h.writeFailure(ctx, err, "failed to replace Project Requirement access")
	}
	return ctx.JSON(http.StatusOK, result)
}

func (h *ProjectRequirementHandler) getOutline(ctx kratoshttp.Context) error {
	identity, ok := h.resolveIdentity(ctx)
	if !ok {
		return nil
	}
	result, err := h.service.GetProjectOutline(workspaceActorContext(ctx, identity), identity.WorkspaceID, ctx.Vars().Get("id"))
	if err != nil {
		return h.writeFailure(ctx, err, "failed to read Project outline")
	}
	return ctx.JSON(http.StatusOK, result)
}

func (h *ProjectRequirementHandler) createOutline(ctx kratoshttp.Context) error {
	identity, ok := h.resolveMutationIdentity(ctx)
	if !ok {
		return nil
	}
	key := strings.TrimSpace(ctx.Request().Header.Get("Idempotency-Key"))
	if key == "" {
		return writeError(ctx, http.StatusBadRequest, "idempotency key is required")
	}
	var request contract.CreateProjectOutlineNodeRequest
	if err := decodeJSON(ctx.Request().Body, &request); err != nil {
		return writeError(ctx, http.StatusBadRequest, "invalid request body")
	}
	result, err := h.service.CreateProjectOutlineNode(workspaceActorContext(ctx, identity), identity.WorkspaceID, ctx.Vars().Get("id"), key, request)
	if err != nil {
		return h.writeFailure(ctx, err, "failed to create Project outline root")
	}
	return ctx.JSON(http.StatusCreated, result)
}

func (h *ProjectRequirementHandler) featureUnavailable(ctx kratoshttp.Context) error {
	if _, ok := h.resolveMutationIdentity(ctx); !ok {
		return nil
	}
	return ctx.JSON(http.StatusConflict, map[string]string{"code": "feature_unavailable", "error": "feature unavailable"})
}

func (h *ProjectRequirementHandler) resolveMutationIdentity(ctx kratoshttp.Context) (contract.WorkspaceHTTPIdentity, bool) {
	identity, ok := h.resolveIdentity(ctx)
	if !ok {
		return contract.WorkspaceHTTPIdentity{}, false
	}
	if h.mutation == nil || h.mutation(ctx.Request()) != nil {
		_ = writeError(ctx, http.StatusForbidden, "invalid CSRF token")
		return contract.WorkspaceHTTPIdentity{}, false
	}
	return identity, true
}

func (h *ProjectRequirementHandler) resolveIdentity(ctx kratoshttp.Context) (contract.WorkspaceHTTPIdentity, bool) {
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
		_ = writeProjectRequirementProblem(ctx, http.StatusNotFound, "not_found", "Project Requirement not found")
		return contract.WorkspaceHTTPIdentity{}, false
	}
	if err != nil || strings.TrimSpace(identity.WorkspaceID) == "" || strings.TrimSpace(identity.ActorID) == "" {
		_ = writeError(ctx, http.StatusUnauthorized, "user not authenticated")
		return contract.WorkspaceHTTPIdentity{}, false
	}
	return identity, true
}

func (h *ProjectRequirementHandler) writeFailure(ctx kratoshttp.Context, err error, internalMessage string) error {
	var conflict contract.RevisionConflictError
	switch {
	case errors.As(err, &conflict):
		return ctx.JSON(http.StatusConflict, map[string]any{
			"code": "revision_conflict", "current_revision": conflict.CurrentRevision, "error": contract.ErrRevisionConflict.Error(),
		})
	case errors.Is(err, application.ErrInvalidProjectRequirementRequest):
		return writeProjectRequirementProblem(ctx, http.StatusBadRequest, "invalid_request", application.ErrInvalidProjectRequirementRequest.Error())
	case errors.Is(err, contract.ErrWorkspacePermissionDenied), errors.Is(err, contract.ErrWorkspaceActorRequired):
		return writeProjectRequirementProblem(ctx, http.StatusForbidden, "permission_denied", contract.ErrWorkspacePermissionDenied.Error())
	case errors.Is(err, application.ErrProjectRequirementNotFound), errors.Is(err, application.ErrProjectSurfaceNotFound), errors.Is(err, contract.ErrActorOutsideWorkspace), errors.Is(err, contract.ErrWorkspaceNotFound):
		return writeProjectRequirementProblem(ctx, http.StatusNotFound, "not_found", application.ErrProjectRequirementNotFound.Error())
	case errors.Is(err, application.ErrProjectRequirementTransition):
		return writeProjectRequirementProblem(ctx, http.StatusConflict, "invalid_transition", application.ErrProjectRequirementTransition.Error())
	case errors.Is(err, application.ErrProjectRequirementSelfApproval):
		return writeProjectRequirementProblem(ctx, http.StatusConflict, "independent_approval_required", application.ErrProjectRequirementSelfApproval.Error())
	case errors.Is(err, contract.ErrIdempotencyConflict):
		return writeProjectRequirementProblem(ctx, http.StatusConflict, "idempotency_conflict", contract.ErrIdempotencyConflict.Error())
	case errors.Is(err, application.ErrProjectRequirementConflict):
		return writeProjectRequirementProblem(ctx, http.StatusConflict, "conflict", application.ErrProjectRequirementConflict.Error())
	default:
		return writeProjectRequirementProblem(ctx, http.StatusInternalServerError, "internal_error", internalMessage)
	}
}

func exactExpectedRevisionQuery(ctx kratoshttp.Context) (int64, bool) {
	values := ctx.Request().URL.Query()
	if len(values) != 1 || len(values["expected_revision"]) != 1 {
		return 0, false
	}
	raw := strings.TrimSpace(values.Get("expected_revision"))
	revision, err := strconv.ParseInt(raw, 10, 64)
	return revision, err == nil && revision >= 1
}

func writeProjectRequirementProblem(ctx kratoshttp.Context, status int, code, message string) error {
	return ctx.JSON(status, map[string]string{"code": code, "error": message})
}
