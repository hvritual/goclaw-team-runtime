package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	issue "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/issue"
)

type IssueMetadataHandler struct {
	service      contract.IssueMetadataService
	identity     contract.WorkspaceHTTPIdentityResolver
	authenticate func(*http.Request) (string, error)
	mutation     func(*http.Request) error
}

const maxIssueMetadataRequestBytes = 64 * 1024

func NewIssueMetadataHandler(service contract.IssueMetadataService, identity contract.WorkspaceHTTPIdentityResolver, authenticate func(*http.Request) (string, error), mutation func(*http.Request) error) *IssueMetadataHandler {
	return &IssueMetadataHandler{service: service, identity: identity, authenticate: authenticate, mutation: mutation}
}
func (h *IssueMetadataHandler) Register(server *kratoshttp.Server) {
	router := server.Route("/")
	router.GET("/api/issues/{id}/metadata", h.get)
	router.PUT("/api/issues/{id}/metadata/{key}", h.put)
	router.DELETE("/api/issues/{id}/metadata/{key}", h.delete)
}
func (h *IssueMetadataHandler) get(ctx kratoshttp.Context) error {
	if h.authenticate != nil {
		if _, err := h.authenticate(ctx.Request()); err != nil {
			return writeError(ctx, http.StatusUnauthorized, "user not authenticated")
		}
	}
	if !hasWorkspaceIdentity(ctx) {
		return writeError(ctx, 400, "workspace_id is required")
	}
	requestContext, workspaceID, err := h.requestIdentity(ctx)
	if err != nil {
		return issueReadIdentityError(ctx, err)
	}
	result, err := h.service.GetIssueMetadata(requestContext, contract.GetIssueMetadataRequest{WorkspaceId: workspaceID, IssueId: ctx.Vars().Get("id")})
	return writeResult(ctx, result, err, "issue metadata operation failed")
}
func (h *IssueMetadataHandler) put(ctx kratoshttp.Context) error {
	if h.authenticate != nil {
		if _, err := h.authenticate(ctx.Request()); err != nil {
			return writeError(ctx, http.StatusUnauthorized, "user not authenticated")
		}
	}
	if !hasWorkspaceIdentity(ctx) {
		return writeError(ctx, 400, "workspace_id is required")
	}
	requestContext, workspaceID, err := h.requestIdentity(ctx)
	if err != nil {
		return issueReadIdentityError(ctx, err)
	}
	if h.mutation != nil && h.mutation(ctx.Request()) != nil {
		return writeError(ctx, http.StatusForbidden, "invalid CSRF token")
	}
	key := ctx.Vars().Get("key")
	if key == "" {
		return writeError(ctx, 400, "key is required")
	}
	if err := issue.ValidateMetadataKey(key); err != nil {
		return writeError(ctx, 400, "key must match ^[a-zA-Z_][a-zA-Z0-9_.-]{0,63}$")
	}
	var body struct {
		Value json.RawMessage `json:"value"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(ctx.Response(), ctx.Request().Body, maxIssueMetadataRequestBytes))
	if err := decoder.Decode(&body); err != nil {
		if isMetadataBodyTooLarge(err) {
			return writeError(ctx, 400, "metadata exceeds the 8KB size limit")
		}
		return writeError(ctx, 400, "invalid request body")
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if isMetadataBodyTooLarge(err) {
			return writeError(ctx, 400, "metadata exceeds the 8KB size limit")
		}
		return writeError(ctx, 400, "invalid request body")
	}
	if len(body.Value) == 0 {
		return writeError(ctx, 400, "value is required")
	}
	if string(body.Value) == "null" {
		return writeError(ctx, 400, "value cannot be null (use DELETE to remove a key)")
	}
	var raw any
	if err := json.Unmarshal(body.Value, &raw); err != nil {
		return writeError(ctx, 400, "value must be valid JSON: "+err.Error())
	}
	switch raw.(type) {
	case string, float64, bool:
	default:
		return writeError(ctx, 400, "value must be a primitive: string, number, or bool")
	}
	result, err := h.service.PutIssueMetadata(requestContext, contract.PutIssueMetadataRequest{WorkspaceId: workspaceID, IssueId: ctx.Vars().Get("id"), Key: ctx.Vars().Get("key"), ValueJson: string(body.Value)})
	return writeResult(ctx, result, err, "failed to set metadata key")
}

func isMetadataBodyTooLarge(err error) bool {
	var maxBytesError *http.MaxBytesError
	return errors.As(err, &maxBytesError)
}
func (h *IssueMetadataHandler) delete(ctx kratoshttp.Context) error {
	if h.authenticate != nil {
		if _, err := h.authenticate(ctx.Request()); err != nil {
			return writeError(ctx, http.StatusUnauthorized, "user not authenticated")
		}
	}
	if !hasWorkspaceIdentity(ctx) {
		return writeError(ctx, 400, "workspace_id is required")
	}
	requestContext, workspaceID, err := h.requestIdentity(ctx)
	if err != nil {
		return issueReadIdentityError(ctx, err)
	}
	if h.mutation != nil && h.mutation(ctx.Request()) != nil {
		return writeError(ctx, http.StatusForbidden, "invalid CSRF token")
	}
	key := ctx.Vars().Get("key")
	if key == "" {
		return writeError(ctx, 400, "key is required")
	}
	if err := issue.ValidateMetadataKey(key); err != nil {
		return writeError(ctx, 400, "key must match ^[a-zA-Z_][a-zA-Z0-9_.-]{0,63}$")
	}
	result, err := h.service.DeleteIssueMetadata(requestContext, contract.DeleteIssueMetadataRequest{WorkspaceId: workspaceID, IssueId: ctx.Vars().Get("id"), Key: ctx.Vars().Get("key")})
	return writeResult(ctx, result, err, "failed to delete metadata key")
}
func hasWorkspaceIdentity(ctx kratoshttp.Context) bool {
	return strings.TrimSpace(ctx.Header().Get("X-Workspace-ID")) != "" || strings.TrimSpace(ctx.Header().Get("X-Workspace-Slug")) != ""
}
func (h *IssueMetadataHandler) requestIdentity(ctx kratoshttp.Context) (context.Context, string, error) {
	if h.identity == nil {
		return ctx.Request().Context(), "", contract.ErrWorkspaceActorRequired
	}
	identity, err := h.identity(ctx.Request())
	if err != nil {
		return ctx.Request().Context(), "", err
	}
	if identity.WorkspaceID == "" || identity.ActorID == "" {
		return ctx.Request().Context(), "", contract.ErrWorkspaceActorRequired
	}
	return contract.WithWorkspaceActor(ctx.Request().Context(), identity.ActorType, identity.ActorID), identity.WorkspaceID, nil
}
func writeResult(ctx kratoshttp.Context, result contract.IssueMetadataSnapshot, err error, failureMessage string) error {
	if err != nil {
		switch {
		case errors.Is(err, contract.ErrIssueNotFound):
			return writeError(ctx, 404, "issue not found")
		case errors.Is(err, contract.ErrInvalidIssueMetadata), errors.Is(err, contract.ErrInvalidIssue), strings.Contains(err.Error(), "metadata cannot exceed"), strings.Contains(err.Error(), "metadata exceeds"):
			message := err.Error()
			if strings.Contains(message, "metadata cannot exceed 50 keys") {
				message = "metadata cannot exceed 50 keys"
			}
			if strings.Contains(message, "metadata exceeds the 8KB size limit") {
				message = "metadata exceeds the 8KB size limit"
			}
			return writeError(ctx, 400, message)
		case errors.Is(err, contract.ErrWorkspaceActorRequired):
			return writeError(ctx, 401, "user not authenticated")
		case errors.Is(err, contract.ErrActorOutsideWorkspace):
			return writeError(ctx, 404, "issue not found")
		default:
			return writeError(ctx, 500, failureMessage)
		}
	}
	return ctx.JSON(http.StatusOK, map[string]any{"metadata": result.Metadata})
}
func writeError(ctx kratoshttp.Context, status int, message string) error {
	return ctx.JSON(status, map[string]string{"error": message})
}
