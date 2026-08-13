package http

import (
	"errors"
	"net/http"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

type IssueDeletionHandler struct {
	service      contract.IssueDeletionService
	identity     contract.WorkspaceHTTPIdentityResolver
	authenticate func(*http.Request) (string, error)
	mutation     func(*http.Request) error
}

func NewIssueDeletionHandler(service contract.IssueDeletionService, identity contract.WorkspaceHTTPIdentityResolver, authenticate func(*http.Request) (string, error), mutation func(*http.Request) error) *IssueDeletionHandler {
	return &IssueDeletionHandler{service: service, identity: identity, authenticate: authenticate, mutation: mutation}
}

func (h *IssueDeletionHandler) Register(server *kratoshttp.Server) {
	server.Route("/").DELETE("/api/issues/{id}", h.delete)
}

func (h *IssueDeletionHandler) delete(ctx kratoshttp.Context) error {
	if h.authenticate == nil {
		return writeError(ctx, 401, "user not authenticated")
	}
	if _, err := h.authenticate(ctx.Request()); err != nil {
		return writeError(ctx, 401, "user not authenticated")
	}
	if !hasWorkspaceIdentity(ctx) {
		return writeError(ctx, 400, "workspace_id is required")
	}
	if h.identity == nil {
		return writeError(ctx, 401, "user not authenticated")
	}
	identity, err := h.identity(ctx.Request())
	if err != nil {
		return issueReadIdentityError(ctx, err)
	}
	if h.mutation != nil && h.mutation(ctx.Request()) != nil {
		return writeError(ctx, 403, "invalid CSRF token")
	}
	requestContext := contract.WithWorkspaceActor(ctx.Request().Context(), identity.ActorType, identity.ActorID)
	_, err = h.service.DeleteIssue(requestContext, contract.DeleteIssueRequest{WorkspaceID: identity.WorkspaceID, IssueID: ctx.Vars().Get("id")})
	if errors.Is(err, contract.ErrIssueNotFound) || errors.Is(err, contract.ErrActorOutsideWorkspace) {
		return writeError(ctx, 404, "issue not found")
	}
	if err != nil {
		return writeError(ctx, 500, "failed to delete issue")
	}
	ctx.Response().WriteHeader(http.StatusNoContent)
	return nil
}
