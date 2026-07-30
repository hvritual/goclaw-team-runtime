package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/netip"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/analytics"
	"github.com/multica-ai/multica/server/internal/auth"
	"github.com/multica-ai/multica/server/internal/events"
	obsmetrics "github.com/multica-ai/multica/server/internal/metrics"
	"github.com/multica-ai/multica/server/internal/middleware"
	"github.com/multica-ai/multica/server/internal/realtime"
	"github.com/multica-ai/multica/server/internal/service"
	"github.com/multica-ai/multica/server/internal/storage"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
	"github.com/multica-ai/multica/server/pkg/featureflag"
)

type txStarter interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

type dbExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Config struct {
	AllowSignup         bool
	AllowedEmails       []string
	AllowedEmailDomains []string
	// DisableWorkspaceCreation, when true, makes POST /api/workspaces return
	// 403 for every caller. There is no role/owner exception because the repo
	// has no platform-admin concept; operators bootstrap the workspace with
	// the flag off, then flip it on and restart so subsequent users join via
	// invitation only. The public /api/config endpoint mirrors this flag so
	// the UI can hide every "Create workspace" affordance — see #3433.
	DisableWorkspaceCreation bool
	// VCSIntegrationEnabled gates the self-hosted Git provider integration
	// (Forgejo / Gitea / GitLab) at the deployment level, independent of whether
	// MULTICA_VCS_SECRET_KEY is set. It is the product boundary: the feature is
	// intended for self-hosted Multica only (where Multica and the Git instance
	// can share a network), and is left off on the managed cloud — connect,
	// rotate, and webhook handlers reject when it is false, and /api/config
	// omits it so the UI hides the whole section rather than showing a
	// "missing key" message a cloud user cannot act on. Populated from
	// MULTICA_VCS_INTEGRATION_ENABLED; the self-host compose defaults it on.
	VCSIntegrationEnabled bool
	// PublicURL is the absolute base URL the API is reachable at from the
	// public internet, with no trailing slash (e.g. "https://multica.ai").
	// Used only to build webhook_url responses for autopilot webhook triggers
	// — never for auth, routing, or workspace resolution. Empty when unset,
	// in which case clients fall back to webhook_path + their own origin.
	// Reading the public host from request headers (Host / X-Forwarded-Host)
	// is intentionally avoided so a misconfigured reverse proxy cannot trick
	// the server into minting webhook URLs pointing at an attacker-controlled
	// host.
	PublicURL string
	// TrustedProxies are CIDRs whose source IP we trust to set
	// X-Forwarded-For / X-Real-IP. Empty means "trust nothing": the rate
	// limiter uses r.RemoteAddr exclusively. Populated via the
	// MULTICA_TRUSTED_PROXIES env var (comma-separated CIDRs, e.g.
	// "10.0.0.0/8,127.0.0.1/32"). This is specifically to keep the per-IP
	// webhook limiter from being bypassed by a spoofed XFF on deployments
	// without a header-stripping reverse proxy in front.
	TrustedProxies           []netip.Prefix
	AttachmentDownloadMode   string
	AttachmentDownloadURLTTL time.Duration
	// AttachmentFrameAncestors are trusted browser origins allowed to embed
	// attachment preview responses. In production this should mirror the
	// frontend/CORS origin allowlist so split app/api self-hosted deployments
	// can frame API-hosted PDFs without allowing arbitrary third-party frames.
	AttachmentFrameAncestors []string
	// ServerVersion is the build version of the running API binary (the same
	// value main.go stamps via -X main.version and reports on /metrics).
	// Surfaced through /api/config so self-hosted operators can confirm which
	// server build is deployed. Empty in dev builds.
	ServerVersion string
}

type Handler struct {
	Queries         *db.Queries
	DB              dbExecutor
	TxStarter       txStarter
	Hub             *realtime.Hub
	Bus             *events.Bus
	IssueService    *service.IssueService
	EmailService    *service.EmailService
	FeatureFlags    *featureflag.Service
	Storage         storage.Storage
	CFSigner        *auth.CloudFrontSigner
	Analytics       analytics.Client
	Metrics         *obsmetrics.BusinessMetrics
	PATCache        *auth.PATCache
	MembershipCache *auth.MembershipCache
	cfg             Config
}

func New(
	queries *db.Queries,
	txStarter txStarter,
	hub *realtime.Hub,
	bus *events.Bus,
	emailService *service.EmailService,
	store storage.Storage,
	cfSigner *auth.CloudFrontSigner,
	analyticsClient analytics.Client,
	cfg Config,
) *Handler {
	var executor dbExecutor
	if candidate, ok := txStarter.(dbExecutor); ok {
		executor = candidate
	}
	if analyticsClient == nil {
		analyticsClient = analytics.NoopClient{}
	}
	if mode, ok := normalizeAttachmentDownloadMode(cfg.AttachmentDownloadMode); ok {
		cfg.AttachmentDownloadMode = string(mode)
	} else {
		slog.Warn("invalid ATTACHMENT_DOWNLOAD_MODE, using auto", "value", cfg.AttachmentDownloadMode)
		cfg.AttachmentDownloadMode = string(attachmentDownloadModeAuto)
	}
	if cfg.AttachmentDownloadURLTTL <= 0 {
		cfg.AttachmentDownloadURLTTL = defaultAttachmentDownloadURLTTL
	}
	return &Handler{
		Queries:      queries,
		DB:           executor,
		TxStarter:    txStarter,
		Hub:          hub,
		Bus:          bus,
		IssueService: service.NewIssueService(queries, txStarter, bus),
		EmailService: emailService,
		Storage:      store,
		CFSigner:     cfSigner,
		Analytics:    analyticsClient,
		cfg:          cfg,
	}
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	// Marshal the body up front so we can advertise an accurate Content-Length
	// header. Streaming straight into the ResponseWriter after WriteHeader forces
	// net/http into chunked transfer encoding, which omits Content-Length; buffering
	// first lets clients (and proxies) see the exact body size.
	body, err := json.Marshal(v)
	if err != nil {
		// Fall back to a minimal, self-describing error payload rather than leaving
		// the client with a half-written response.
		body = []byte(`{"error":"failed to encode response"}`)
		status = http.StatusInternalServerError
	}
	// Match the trailing newline that json.Encoder.Encode historically appended.
	body = append(body, '\n')
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

// writeMeasuredJSON behaves like writeJSON but returns the encoded body size so
// callers can record payload bytes in slow-endpoint diagnostics. It measures the
// uncompressed JSON length and is unrelated to transport compression.
func writeMeasuredJSON(w http.ResponseWriter, status int, v any) (int, error) {
	body, err := json.Marshal(v)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to encode response")
		return 0, err
	}
	body = append(body, '\n')
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(status)
	if _, err := w.Write(body); err != nil {
		return len(body), err
	}
	return len(body), nil
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// Thin wrappers around util functions.
//
// parseUUID is intentionally the panicking variant: any handler call site
// reachable here is expected to feed a UUID that is either (a) a sqlc round-trip
// of a DB-sourced value, or (b) a raw request input that has already been
// validated upstream. A panic here means an unguarded user-input string slipped
// in — that is a real bug we want surfaced loudly (chi's middleware.Recoverer
// converts it to a 500) instead of silently corrupting data via a zero UUID.
//
// For unvalidated user input at request boundaries, use parseUUIDOrBadRequest
// (writes 400) — never feed raw chi.URLParam / request-body strings into
// parseUUID directly when the call writes to the database.
func parseUUID(s string) pgtype.UUID                { return util.MustParseUUID(s) }
func uuidToString(u pgtype.UUID) string             { return util.UUIDToString(u) }
func textToPtr(t pgtype.Text) *string               { return util.TextToPtr(t) }
func ptrToText(s *string) pgtype.Text               { return util.PtrToText(s) }
func strToText(s string) pgtype.Text                { return util.StrToText(s) }
func timestampToString(t pgtype.Timestamptz) string { return util.TimestampToString(t) }
func timestampToPtr(t pgtype.Timestamptz) *string   { return util.TimestampToPtr(t) }
func dateToPtr(d pgtype.Date) *string               { return util.DateToPtr(d) }
func uuidToPtr(u pgtype.UUID) *string               { return util.UUIDToPtr(u) }

// uuidsToStrings maps a UUID array column to string ids, skipping NULL/invalid
// entries. Returns nil (not an empty slice) when there is nothing to emit so
// `omitempty` JSON fields drop out cleanly (MUL-4195).
func uuidsToStrings(us []pgtype.UUID) []string {
	if len(us) == 0 {
		return nil
	}
	out := make([]string, 0, len(us))
	for _, u := range us {
		if u.Valid {
			out = append(out, uuidToString(u))
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// uuidStringsOrEmpty preserves the distinction between a modern, authoritative
// empty UUID-array value (`[]`) and a field omitted by a legacy server. Delivery
// receipts use this so clients never mistake zero delivered comments for an
// unknown receipt and fall back to the enqueue-time plan.
func uuidStringsOrEmpty(us []pgtype.UUID) []string {
	out := uuidsToStrings(us)
	if out == nil {
		return []string{}
	}
	return out
}

func int8ToPtr(v pgtype.Int8) *int64 { return util.Int8ToPtr(v) }
func int4ToPtr(v pgtype.Int4) *int32 { return util.Int4ToPtr(v) }
func ptrToInt4(v *int32) pgtype.Int4 { return util.PtrToInt4(v) }

// parseUUIDOrBadRequest validates a UUID string sourced from user input
// (URL params, request body, headers). On invalid input it writes a 400
// response and returns ok=false; callers must return immediately.
//
// Use this anywhere a malformed UUID would otherwise reach a write query
// (DELETE / UPDATE) — the silent zero-UUID behavior of the old ParseUUID
// caused real silent-data-loss bugs (#1661).
func parseUUIDOrBadRequest(w http.ResponseWriter, s, fieldName string) (pgtype.UUID, bool) {
	u, err := util.ParseUUID(s)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid "+fieldName)
		return pgtype.UUID{}, false
	}
	return u, true
}

func parseUUIDSliceOrBadRequest(w http.ResponseWriter, ids []string, fieldName string) ([]pgtype.UUID, bool) {
	uuids := make([]pgtype.UUID, len(ids))
	for i, id := range ids {
		u, err := util.ParseUUID(id)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid "+fieldName)
			return nil, false
		}
		uuids[i] = u
	}
	return uuids, true
}

// publish sends a domain event through the event bus.
func (h *Handler) publish(eventType, workspaceID, actorType, actorID string, payload any) {
	h.Bus.Publish(events.Event{
		Type:        eventType,
		WorkspaceID: workspaceID,
		ActorType:   actorType,
		ActorID:     actorID,
		Payload:     payload,
	})
}

func isNotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// isCheckViolation reports whether err is a PostgreSQL CHECK constraint
// violation (SQLSTATE 23514). Used to translate column-level CHECK failures
// into a 4xx instead of a generic 500.
func isCheckViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23514"
}

func requestUserID(r *http.Request) string {
	return r.Header.Get("X-User-ID")
}

func (h *Handler) resolveActor(_ *http.Request, userID, _ string) (actorType, actorID string) {
	return "member", userID
}

func requireUserID(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID := requestUserID(r)
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "user not authenticated")
		return "", false
	}
	return userID, true
}

// resolveWorkspaceID returns the workspace UUID for this request. Delegates
// to middleware.ResolveWorkspaceIDFromRequest so middleware-protected routes
// and middleware-less routes (e.g. /api/upload-file) share identical
// resolution behavior — including slug → UUID translation via the DB.
//
// Returns "" when no workspace identifier was provided or a slug was provided
// but doesn't match any workspace.
func (h *Handler) resolveWorkspaceID(r *http.Request) string {
	return middleware.ResolveWorkspaceIDFromRequest(r, h.Queries)
}

// ctxMember returns the workspace member from context (set by workspace middleware).
func ctxMember(ctx context.Context) (db.Member, bool) {
	return middleware.MemberFromContext(ctx)
}

// ctxWorkspaceID returns the workspace ID from context (set by workspace middleware).
func ctxWorkspaceID(ctx context.Context) string {
	return middleware.WorkspaceIDFromContext(ctx)
}

// workspaceIDFromURL returns the workspace ID from context (preferred) or chi URL param (fallback).
func workspaceIDFromURL(r *http.Request, param string) string {
	if id := middleware.WorkspaceIDFromContext(r.Context()); id != "" {
		return id
	}
	return chi.URLParam(r, param)
}

// workspaceMember returns the member from middleware context, or falls back to a DB
// lookup when the handler is called directly (e.g. in tests).
func (h *Handler) workspaceMember(w http.ResponseWriter, r *http.Request, workspaceID string) (db.Member, bool) {
	if m, ok := ctxMember(r.Context()); ok {
		return m, true
	}
	return h.requireWorkspaceMember(w, r, workspaceID, "workspace not found")
}

func roleAllowed(role string, roles ...string) bool {
	for _, candidate := range roles {
		if role == candidate {
			return true
		}
	}
	return false
}

func countOwners(members []db.Member) int {
	owners := 0
	for _, member := range members {
		if member.Role == "owner" {
			owners++
		}
	}
	return owners
}

func (h *Handler) getWorkspaceMember(ctx context.Context, userID, workspaceID string) (db.Member, error) {
	userUUID, err := util.ParseUUID(userID)
	if err != nil {
		return db.Member{}, err
	}
	wsUUID, err := util.ParseUUID(workspaceID)
	if err != nil {
		return db.Member{}, err
	}
	return h.Queries.GetMemberByUserAndWorkspace(ctx, db.GetMemberByUserAndWorkspaceParams{
		UserID:      userUUID,
		WorkspaceID: wsUUID,
	})
}

func (h *Handler) requireWorkspaceMember(w http.ResponseWriter, r *http.Request, workspaceID, notFoundMsg string) (db.Member, bool) {
	if workspaceID == "" {
		writeError(w, http.StatusBadRequest, "workspace_id is required")
		return db.Member{}, false
	}

	userID, ok := requireUserID(w, r)
	if !ok {
		return db.Member{}, false
	}

	member, err := h.getWorkspaceMember(r.Context(), userID, workspaceID)
	if err != nil {
		writeError(w, http.StatusNotFound, notFoundMsg)
		return db.Member{}, false
	}

	return member, true
}

func (h *Handler) requireWorkspaceRole(w http.ResponseWriter, r *http.Request, workspaceID, notFoundMsg string, roles ...string) (db.Member, bool) {
	member, ok := h.requireWorkspaceMember(w, r, workspaceID, notFoundMsg)
	if !ok {
		return db.Member{}, false
	}
	if !roleAllowed(member.Role, roles...) {
		writeError(w, http.StatusForbidden, "insufficient permissions")
		return db.Member{}, false
	}
	return member, true
}

func (h *Handler) isWorkspaceEntity(ctx context.Context, userType, userID, workspaceID string) bool {
	if userType != "member" {
		return false
	}
	_, err := h.getWorkspaceMember(ctx, userID, workspaceID)
	return err == nil
}

func (h *Handler) loadIssueForUser(w http.ResponseWriter, r *http.Request, issueID string) (db.Issue, bool) {
	if _, ok := requireUserID(w, r); !ok {
		return db.Issue{}, false
	}

	workspaceID := h.resolveWorkspaceID(r)
	if workspaceID == "" {
		writeError(w, http.StatusBadRequest, "workspace_id is required")
		return db.Issue{}, false
	}

	// Try identifier format first (e.g., "JIA-42"). resolveIssueByIdentifier
	// silently returns false for non-identifier strings, falling through to
	// the UUID path below.
	if issue, ok := h.resolveIssueByIdentifier(r.Context(), issueID, workspaceID); ok {
		return issue, true
	}

	issueUUID, err := util.ParseUUID(issueID)
	if err != nil {
		// Not a valid UUID and didn't match identifier format → 404 (consistent
		// with previous silent-zero behavior, which would also have produced 404).
		writeError(w, http.StatusNotFound, "issue not found")
		return db.Issue{}, false
	}
	wsUUID, err := util.ParseUUID(workspaceID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid workspace_id")
		return db.Issue{}, false
	}
	issue, err := h.Queries.GetIssueInWorkspace(r.Context(), db.GetIssueInWorkspaceParams{
		ID:          issueUUID,
		WorkspaceID: wsUUID,
	})
	if err != nil {
		writeError(w, http.StatusNotFound, "issue not found")
		return db.Issue{}, false
	}
	return issue, true
}

// resolveIssueByIdentifier tries to look up an issue by "PREFIX-NUMBER" format.
func (h *Handler) resolveIssueByIdentifier(ctx context.Context, id, workspaceID string) (db.Issue, bool) {
	parts := splitIdentifier(id)
	if parts == nil {
		return db.Issue{}, false
	}
	if workspaceID == "" {
		return db.Issue{}, false
	}
	wsUUID, err := util.ParseUUID(workspaceID)
	if err != nil {
		return db.Issue{}, false
	}
	issue, err := h.Queries.GetIssueByNumber(ctx, db.GetIssueByNumberParams{
		WorkspaceID: wsUUID,
		Number:      parts.number,
	})
	if err != nil {
		return db.Issue{}, false
	}
	return issue, true
}

type identifierParts struct {
	prefix string
	number int32
}

func splitIdentifier(id string) *identifierParts {
	idx := -1
	for i := len(id) - 1; i >= 0; i-- {
		if id[i] == '-' {
			idx = i
			break
		}
	}
	if idx <= 0 || idx >= len(id)-1 {
		return nil
	}
	numStr := id[idx+1:]
	num := 0
	for _, c := range numStr {
		if c < '0' || c > '9' {
			return nil
		}
		num = num*10 + int(c-'0')
	}
	if num <= 0 {
		return nil
	}
	return &identifierParts{prefix: id[:idx], number: int32(num)}
}

// getIssuePrefix fetches the issue_prefix for a workspace.
// Falls back to generating a prefix from the workspace name if the stored
// prefix is empty (e.g. workspaces created before the prefix was introduced).
func (h *Handler) getIssuePrefix(ctx context.Context, workspaceID pgtype.UUID) string {
	ws, err := h.Queries.GetWorkspace(ctx, workspaceID)
	if err != nil {
		return ""
	}
	if ws.IssuePrefix != "" {
		return ws.IssuePrefix
	}
	return generateIssuePrefix(ws.Name)
}
