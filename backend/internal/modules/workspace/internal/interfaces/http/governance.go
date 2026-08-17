package http

import (
	"context"
	"net/http"
	"time"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

const governanceBacklogDegradedAfter = 15 * time.Minute

type GovernanceHandler struct {
	service      contract.GovernanceDiagnosticsReader
	identity     contract.WorkspaceHTTPIdentityResolver
	authenticate func(*http.Request) (string, error)
	memberships  contract.WorkspaceMembershipReader
}

func NewGovernanceHandler(service contract.GovernanceDiagnosticsReader, identity contract.WorkspaceHTTPIdentityResolver, authenticate func(*http.Request) (string, error), memberships contract.WorkspaceMembershipReader) *GovernanceHandler {
	return &GovernanceHandler{service: service, identity: identity, authenticate: authenticate, memberships: memberships}
}

func (h *GovernanceHandler) Register(server *kratoshttp.Server) {
	server.Route("/").GET("/api/operations/governance", h.read)
}

func (h *GovernanceHandler) read(ctx kratoshttp.Context) error {
	if h.authenticate == nil {
		return writeError(ctx, http.StatusUnauthorized, "user not authenticated")
	}
	userID, err := h.authenticate(ctx.Request())
	if err != nil {
		return writeError(ctx, http.StatusUnauthorized, "user not authenticated")
	}
	if !hasWorkspaceIdentity(ctx) {
		return writeError(ctx, http.StatusBadRequest, "workspace is required")
	}
	if h.identity == nil {
		return writeError(ctx, http.StatusServiceUnavailable, "governance unavailable")
	}
	identity, err := h.identity(ctx.Request())
	if err != nil {
		return issueReadIdentityError(ctx, err)
	}
	if identity.ActorType != "member" || h.memberships == nil {
		return writeError(ctx, http.StatusForbidden, "governance access denied")
	}
	membership, found, err := h.findMembership(ctx.Request().Context(), userID, identity)
	if err != nil {
		return writeError(ctx, http.StatusServiceUnavailable, "governance unavailable")
	}
	if !found {
		return writeError(ctx, http.StatusNotFound, "workspace not found")
	}
	if membership.Role != "owner" && membership.Role != "admin" {
		return writeError(ctx, http.StatusForbidden, "governance access denied")
	}
	if h.service == nil {
		return writeError(ctx, http.StatusServiceUnavailable, "governance unavailable")
	}
	diagnostics, err := h.service.ReadGovernanceDiagnostics(ctx.Request().Context(), identity.WorkspaceID)
	if err != nil {
		return writeError(ctx, http.StatusServiceUnavailable, "governance unavailable")
	}
	response := map[string]any{
		"status":                      "ok",
		"degraded":                    diagnostics.OldestReadyAge > governanceBacklogDegradedAfter,
		"ready_count":                 diagnostics.ReadyCount,
		"oldest_ready_age_seconds":    int64(diagnostics.OldestReadyAge / time.Second),
		"inflight_count":              diagnostics.InflightCount,
		"oldest_lease_age_seconds":    int64(diagnostics.OldestLeaseAge / time.Second),
		"retry_wait_count":            diagnostics.RetryWaitCount,
		"dead_letter_count":           diagnostics.DeadLetterCount,
		"last_successful_delivery_at": nil,
		"schema_version":              diagnostics.SchemaVersion,
		"dispatcher_running":          diagnostics.DispatcherRunning,
	}
	if response["degraded"].(bool) {
		response["status"] = "degraded"
	}
	if !diagnostics.LastSuccessfulDelivery.IsZero() {
		response["last_successful_delivery_at"] = diagnostics.LastSuccessfulDelivery.UTC().Format(time.RFC3339Nano)
	}
	return ctx.JSON(http.StatusOK, response)
}

func (h *GovernanceHandler) findMembership(ctx context.Context, userID string, identity contract.WorkspaceHTTPIdentity) (contract.WorkspaceMembership, bool, error) {
	membership, found, err := h.memberships.FindForUserAndWorkspace(ctx, userID, identity.WorkspaceID)
	if err != nil || found {
		return membership, found, err
	}
	return h.memberships.FindByMemberAndWorkspace(ctx, identity.ActorID, identity.WorkspaceID)
}
