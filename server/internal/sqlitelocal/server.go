// Package sqlitelocal provides the opt-in, local-only SQLite server.
//
// It deliberately does not share the PostgreSQL/sqlc handler graph: the
// production server remains unchanged, while this package implements the
// stable local contracts for workspace, member, project, issue, task, and
// skill data. It does not start any external execution process.
package sqlitelocal

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/multica-ai/multica/server/internal/bootstrap"
	"github.com/multica-ai/multica/server/internal/knowledge"
	knowledgeSqlite "github.com/multica-ai/multica/server/internal/knowledge/adapter/sqlite"
	"github.com/multica-ai/multica/server/internal/knowledge/outbox"
	authcontract "github.com/multica-ai/multica/server/internal/modules/auth/contract"
	"github.com/multica-ai/multica/server/internal/projectrequirements"
	projectRequirementsSQLite "github.com/multica-ai/multica/server/internal/projectrequirements/adapter/sqlite"
	"github.com/multica-ai/multica/server/internal/workspacepermissions"
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schema string

const (
	defaultVerificationCode = "888888"
	authCookieName          = "multica_auth"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type Options struct {
	VerificationCode        string
	FrontendOrigin          string
	KnowledgeDatabasePath   string
	DisableKnowledge        bool
	PublicURL               string
	MCPAuthorizationServers []string
	MCPTokenVerifier        mcpauth.TokenVerifier
}

type knowledgeOperationalStore interface {
	knowledge.Repository
	knowledge.SearchIndex
	Capabilities(context.Context) (knowledge.StoreCapabilities, error)
	Close() error
}

type Server struct {
	db                        *sql.DB
	handler                   http.Handler
	verificationCode          string
	knowledgeStore            knowledgeOperationalStore
	knowledgeService          *knowledge.Service
	knowledgeUnavailable      error
	knowledgeDispatcher       *outbox.Dispatcher
	knowledgeCancel           context.CancelFunc
	knowledgeDone             chan struct{}
	knowledgeDispatchMu       sync.RWMutex
	knowledgeDispatchError    string
	mcpHandler                http.Handler
	mcpPublicURL              string
	mcpAuthorizationServers   []string
	mcpExternalVerifier       mcpauth.TokenVerifier
	authMembers               authcontract.MemberService
	requirements              *projectrequirements.Service
	requirementTracking       *projectrequirements.TrackingService
	requirementTrackingSQLite *projectRequirementsSQLite.TrackingRepository
}

type principal struct {
	UserID string
}

type contextKey int

const principalKey contextKey = iota

func Open(path string, options Options) (*Server, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("sqlite database path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create sqlite directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}
	db.SetMaxOpenConns(1)
	nativeApplication, err := bootstrap.NewSQLiteApplication(context.Background(), db)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("assemble sqlite modules: %w", err)
	}
	if err := migrateLegacyTaskSchema(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate sqlite task schema: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("initialize sqlite schema: %w", err)
	}

	code := strings.TrimSpace(options.VerificationCode)
	if code == "" {
		code = defaultVerificationCode
	}
	trackingRepository := projectRequirementsSQLite.NewTracking(db)
	server := &Server{db: db, verificationCode: code, authMembers: nativeApplication.AuthMembers(), requirementTracking: projectrequirements.NewTrackingService(trackingRepository), requirementTrackingSQLite: trackingRepository}
	server.requirements = projectrequirements.NewService(projectRequirementsSQLite.NewWithApprovalHook(db, server.enqueueApprovedRequirementEvidence))
	if !options.DisableKnowledge {
		knowledgePath := strings.TrimSpace(options.KnowledgeDatabasePath)
		if knowledgePath == "" {
			extension := filepath.Ext(path)
			knowledgePath = strings.TrimSuffix(path, extension) + ".knowledge" + extension
		}
		knowledgeStore, err := knowledgeSqlite.Open(knowledgePath)
		if err != nil {
			server.knowledgeUnavailable = fmt.Errorf("open knowledge store: %w", err)
		} else {
			server.knowledgeStore = knowledgeStore
			server.knowledgeService = knowledge.NewService(
				knowledgeStore,
				knowledge.DefaultPromotionPolicy{},
				server,
			)
			server.knowledgeDispatcher = outbox.NewDispatcher(
				&sqliteEvidenceOutbox{db: db},
				server.knowledgeService,
			)
			server.startKnowledgeDispatcher()
			server.configureKnowledgeMCP(options)
		}
	} else {
		server.knowledgeUnavailable = errKnowledgeDisabled
	}
	server.handler = server.routes(options.FrontendOrigin)
	return server, nil
}

func (s *Server) ValidateProject(ctx context.Context, workspaceID, projectID string) error {
	if s.belongsToWorkspace(ctx, "projects", projectID, workspaceID) {
		return nil
	}
	return knowledge.ErrProjectScope
}

func migrateLegacyTaskSchema(db *sql.DB) error {
	rows, err := db.Query(`PRAGMA table_info(tasks)`)
	if err != nil {
		return err
	}
	defer rows.Close()

	hasTaskTable := false
	hasTitle := false
	for rows.Next() {
		var (
			columnID     int
			name         string
			columnType   string
			notNull      int
			defaultValue any
			primaryKey   int
		)
		if err := rows.Scan(
			&columnID,
			&name,
			&columnType,
			&notNull,
			&defaultValue,
			&primaryKey,
		); err != nil {
			return err
		}
		hasTaskTable = true
		hasTitle = hasTitle || name == "title"
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if !hasTaskTable || hasTitle {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	statements := []string{
		`CREATE TABLE tasks_six_domain (
			id TEXT PRIMARY KEY,
			workspace_id TEXT NOT NULL,
			project_id TEXT,
			issue_id TEXT,
			title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'todo',
			priority TEXT NOT NULL DEFAULT 'none',
			assignee_id TEXT,
			creator_id TEXT NOT NULL,
			position REAL NOT NULL DEFAULT 0,
			start_date TEXT,
			due_date TEXT,
			completed_at TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`INSERT INTO tasks_six_domain(
			id, workspace_id, project_id, issue_id, title, description,
			status, priority, assignee_id, creator_id, position,
			start_date, due_date, completed_at, created_at, updated_at
		)
		SELECT
			id,
			workspace_id,
			NULL,
			issue_id,
			'Imported task ' || substr(id, 1, 8),
			'',
			CASE status
				WHEN 'running' THEN 'in_progress'
				WHEN 'dispatched' THEN 'in_progress'
				WHEN 'completed' THEN 'done'
				WHEN 'cancelled' THEN 'cancelled'
				ELSE 'todo'
			END,
			CASE
				WHEN priority >= 4 THEN 'urgent'
				WHEN priority = 3 THEN 'high'
				WHEN priority = 2 THEN 'medium'
				WHEN priority = 1 THEN 'low'
				ELSE 'none'
			END,
			NULL,
			COALESCE(
				(SELECT user_id FROM members WHERE members.workspace_id = tasks.workspace_id LIMIT 1),
				'00000000-0000-0000-0000-000000000000'
			),
			0,
			started_at,
			NULL,
			completed_at,
			created_at,
			updated_at
		FROM tasks`,
		`DROP TABLE IF EXISTS task_messages`,
		`DROP TABLE tasks`,
		`ALTER TABLE tasks_six_domain RENAME TO tasks`,
	}
	for _, statement := range statements {
		if _, err := tx.Exec(statement); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Server) Handler() http.Handler {
	return s.handler
}

func (s *Server) Close() error {
	if s.knowledgeCancel != nil {
		s.knowledgeCancel()
		<-s.knowledgeDone
	}
	var closeErrors []error
	if s.knowledgeStore != nil {
		closeErrors = append(closeErrors, s.knowledgeStore.Close())
	}
	closeErrors = append(closeErrors, s.db.Close())
	return errors.Join(closeErrors...)
}

func (s *Server) routes(frontendOrigin string) http.Handler {
	r := chi.NewRouter()
	allowedOrigins := []string{"http://localhost:*", "http://127.0.0.1:*"}
	if strings.TrimSpace(frontendOrigin) != "" {
		allowedOrigins = append(allowedOrigins, frontendOrigin)
	}
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Workspace-ID", "X-Workspace-Slug", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "database": "sqlite"})
	})
	r.Get("/ws", s.unsupported)
	r.Get("/api/config", s.getConfig)
	r.Get("/.well-known/oauth-protected-resource", s.getMCPProtectedResourceMetadata)
	r.Get("/.well-known/oauth-protected-resource/mcp/{workspaceSlug}/knowledge", s.getMCPProtectedResourceMetadata)
	r.Handle("/mcp/{workspaceSlug}/knowledge", s.handleKnowledgeMCP())
	r.Post("/auth/send-code", s.sendCode)
	r.Post("/auth/verify-code", s.verifyCode)
	r.Post("/auth/logout", s.logout)

	r.Group(func(api chi.Router) {
		api.Use(s.authenticate)
		api.Get("/api/me", s.getMe)
		api.Patch("/api/me", s.updateMe)
		api.Patch("/api/me/onboarding", s.patchOnboarding)
		api.Post("/api/me/onboarding/complete", s.completeOnboarding)

		api.Route("/api/workspaces", func(workspaces chi.Router) {
			workspaces.Get("/", s.listWorkspaces)
			workspaces.Post("/", s.createWorkspace)
			workspaces.Get("/{id}", s.getWorkspace)
			workspaces.Patch("/{id}", s.updateWorkspace)
			workspaces.Delete("/{id}", s.deleteWorkspace)
			workspaces.Get("/{id}/permissions", s.getWorkspacePermissions)
			workspaces.Get("/{id}/members", s.listMembers)
			workspaces.Post("/{id}/members", s.createMember)
			workspaces.Patch("/{id}/members/{memberID}", s.updateMember)
			workspaces.Delete("/{id}/members/{memberID}", s.deleteMember)
			workspaces.Get("/{id}/invitations", s.listWorkspaceInvitations)
			workspaces.Delete("/{id}/invitations/{invitationID}", s.revokeInvitation)
			workspaces.Post("/{id}/leave", s.leaveWorkspace)
		})

		api.Get("/api/invitations", s.listMyInvitations)
		api.Get("/api/invitations/{id}", s.getMyInvitation)
		api.Post("/api/invitations/{id}/accept", s.acceptInvitation)
		api.Post("/api/invitations/{id}/decline", s.declineInvitation)
		api.Get("/api/properties", s.listProperties)
		api.Route("/api/pins", func(pins chi.Router) {
			pins.Get("/", s.listPins)
			pins.Post("/", s.createPin)
			pins.Put("/reorder", s.reorderPins)
			pins.Delete("/{itemType}/{itemId}", s.deletePin)
		})

		api.Route("/api/projects", func(projects chi.Router) {
			projects.Get("/", s.listProjects)
			projects.Post("/", s.createProject)
			projects.Get("/{id}", s.getProject)
			projects.Put("/{id}", s.updateProject)
			projects.Delete("/{id}", s.deleteProject)
			projects.Get("/{id}/requirement-baseline", s.getProjectRequirementBaseline)
			projects.Put("/{id}/requirement-baseline", s.saveProjectRequirementDraft)
			projects.Post("/{id}/requirement-baseline/submit-review", s.submitProjectRequirementReview)
			projects.Post("/{id}/requirement-baseline/approve", s.approveProjectRequirement)
			projects.Post("/{id}/requirement-baseline/withdraw", s.withdrawProjectRequirementReview)
			projects.Get("/{id}/requirement-baseline/history", s.listProjectRequirementHistory)
			projects.Get("/{id}/requirement-baseline/coverage", s.getProjectRequirementCoverage)
			projects.Post("/{id}/requirement-baseline/links", s.linkProjectRequirementIssue)
			projects.Delete("/{id}/requirement-baseline/links/{requirementKey}/{issueID}", s.unlinkProjectRequirementIssue)
			projects.Post("/{id}/requirement-baseline/items/{requirementKey}/issues", s.createIssueForProjectRequirement)
			projects.Get("/{id}/retrospectives", s.listProjectRetrospectives)
			projects.Post("/{id}/retrospectives", s.createProjectRetrospective)
		})

		api.Route("/api/issues", func(issues chi.Router) {
			issues.Get("/", s.listIssues)
			issues.Post("/", s.createIssue)
			issues.Post("/query", s.listIssues)
			issues.Post("/table/facets", s.listIssueTableFacets)
			issues.Post("/table/rows", s.listIssueTableRows)
			issues.Get("/children", s.listIssueChildren)
			issues.Get("/child-progress", s.listIssueChildProgress)
			issues.Get("/{id}", s.getIssue)
			issues.Patch("/{id}", s.updateIssue)
			issues.Put("/{id}", s.updateIssue)
			issues.Delete("/{id}", s.deleteIssue)
			issues.Get("/{id}/acceptance-conclusions", s.listIssueAcceptanceConclusions)
			issues.Post("/{id}/acceptance-conclusions", s.createIssueAcceptanceConclusion)
			issues.Get("/{id}/children", s.listIssueChildren)
			issues.Get("/{id}/timeline", s.listIssueTimeline)
			issues.Get("/{id}/comments", s.listComments)
			issues.Post("/{id}/comments", s.createComment)
			issues.Post("/{id}/reactions", s.addIssueReaction)
			issues.Delete("/{id}/reactions", s.removeIssueReaction)
		})

		api.Route("/api/comments/{commentId}", func(comments chi.Router) {
			comments.Put("/", s.updateComment)
			comments.Post("/knowledge-proposals", s.proposeCommentDecision)
		})

		api.Route("/api/tasks", func(tasks chi.Router) {
			tasks.Get("/", s.listTasks)
			tasks.Post("/", s.createTask)
			tasks.Get("/{id}", s.getTask)
			tasks.Patch("/{id}", s.updateTask)
			tasks.Delete("/{id}", s.deleteTask)
		})

		api.Route("/api/skills", func(skills chi.Router) {
			skills.Get("/", s.listSkills)
			skills.Post("/", s.createSkill)
			skills.Get("/{id}", s.getSkill)
			skills.Put("/{id}", s.updateSkill)
			skills.Delete("/{id}", s.deleteSkill)
		})

		api.Route("/api/knowledge", func(knowledge chi.Router) {
			knowledge.Get("/", s.listKnowledge)
			knowledge.Get("/search", s.listKnowledge)
			knowledge.Post("/proposals", s.proposeKnowledge)
			knowledge.Get("/candidates", s.listKnowledgeCandidates)
			knowledge.Post("/candidates/{id}/review", s.reviewKnowledgeCandidate)
			knowledge.Get("/health", s.getKnowledgeHealth)
			knowledge.Get("/{id}/revisions", s.listKnowledgeRevisions)
			knowledge.Get("/{id}/sources", s.listKnowledgeSources)
			knowledge.Get("/{id}", s.getKnowledgeEntry)
		})

		api.NotFound(s.unsupported)
	})
	return r
}

func (s *Server) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := strings.TrimSpace(r.Header.Get("Authorization"))
		token := ""
		if strings.HasPrefix(auth, "Bearer ") {
			token = strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		} else if cookie, err := r.Cookie(authCookieName); err == nil {
			token = strings.TrimSpace(cookie.Value)
		}
		if token == "" {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		var userID string
		err := s.db.QueryRowContext(r.Context(), `SELECT user_id FROM auth_tokens WHERE token = ?`, token).Scan(&userID)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}
		ctx := context.WithValue(r.Context(), principalKey, principal{UserID: userID})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func currentUserID(r *http.Request) string {
	value, _ := r.Context().Value(principalKey).(principal)
	return value.UserID
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 2<<20))
	if err := decoder.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if value != nil {
		_ = json.NewEncoder(w).Encode(value)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"error": message})
}

func now() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func newID() string {
	return uuid.NewString()
}

func nullable(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func mapJSON(value string, fallback any) any {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	var decoded any
	if err := json.Unmarshal([]byte(value), &decoded); err != nil {
		return fallback
	}
	if decoded == nil {
		return fallback
	}
	return decoded
}

func encodeJSON(value any, fallback string) string {
	if value == nil {
		return fallback
	}
	encoded, err := json.Marshal(value)
	if err != nil || string(encoded) == "null" {
		return fallback
	}
	return string(encoded)
}

func (s *Server) getConfig(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"cdn_domain":                   "",
		"cdn_signed":                   false,
		"allow_signup":                 true,
		"google_client_id":             "",
		"posthog_key":                  "",
		"posthog_host":                 "",
		"analytics_environment":        "local",
		"daemon_server_url":            "",
		"daemon_app_url":               "",
		"workspace_creation_disabled":  false,
		"vcs_integration_available":    false,
		"feature_flags":                map[string]bool{},
		"app_mode":                     "sqlite_local",
		"cloud_features_enabled":       false,
		"agent_execution_enabled":      false,
		"local_verification_code_hint": s.verificationCode,
	})
}

func (s *Server) sendCode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if !strings.Contains(strings.TrimSpace(req.Email), "@") {
		writeError(w, http.StatusBadRequest, "valid email is required")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) verifyCode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if req.Code != s.verificationCode || !strings.Contains(email, "@") {
		writeError(w, http.StatusUnauthorized, "invalid verification code")
		return
	}

	user, err := s.findOrCreateUser(r.Context(), email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create local user")
		return
	}
	token := newID()
	if _, err := s.db.ExecContext(r.Context(), `INSERT INTO auth_tokens(token, user_id, created_at) VALUES (?, ?, ?)`, token, user.ID, now()); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create token")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   7 * 24 * 60 * 60,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
	})
	writeJSON(w, http.StatusOK, map[string]any{"token": token, "user": user.response()})
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	auth := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	if auth == "" {
		if cookie, err := r.Cookie(authCookieName); err == nil {
			auth = strings.TrimSpace(cookie.Value)
		}
	}
	if auth != "" {
		if _, err := s.db.ExecContext(r.Context(), `DELETE FROM auth_tokens WHERE token = ?`, auth); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to revoke token")
			return
		}
	}
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(1, 0),
	})
	w.WriteHeader(http.StatusNoContent)
}

type user struct {
	ID, Name, Email, CreatedAt, UpdatedAt                    string
	AvatarURL, Language, Timezone, OnboardedAt, StarterState sql.NullString
	Questionnaire, ProfileDescription                        string
}

func scanUser(scanner interface{ Scan(...any) error }) (user, error) {
	var value user
	err := scanner.Scan(
		&value.ID, &value.Name, &value.Email, &value.AvatarURL, &value.Language,
		&value.Timezone, &value.OnboardedAt, &value.Questionnaire,
		&value.StarterState, &value.ProfileDescription, &value.CreatedAt, &value.UpdatedAt,
	)
	return value, err
}

func userColumns() string {
	return `id, name, email, avatar_url, language, timezone, onboarded_at,
		onboarding_questionnaire, starter_content_state, profile_description,
		created_at, updated_at`
}

func (u user) response() map[string]any {
	return map[string]any{
		"id": u.ID, "name": u.Name, "email": u.Email,
		"avatar_url": nullable(u.AvatarURL.String), "language": nullable(u.Language.String),
		"timezone": nullable(u.Timezone.String), "onboarded_at": nullable(u.OnboardedAt.String),
		"onboarding_questionnaire": mapJSON(u.Questionnaire, map[string]any{}),
		"starter_content_state":    nullable(u.StarterState.String),
		"profile_description":      u.ProfileDescription,
		"created_at":               u.CreatedAt, "updated_at": u.UpdatedAt,
	}
}

func (s *Server) findOrCreateUser(ctx context.Context, email string) (user, error) {
	value, err := scanUser(s.db.QueryRowContext(ctx, `SELECT `+userColumns()+` FROM users WHERE email = ?`, email))
	if err == nil {
		return value, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return user{}, err
	}
	timestamp := now()
	name := strings.Split(email, "@")[0]
	id := newID()
	_, err = s.db.ExecContext(ctx, `INSERT INTO users(
		id, name, email, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?)`, id, name, email, timestamp, timestamp)
	if err != nil {
		return user{}, err
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE invitations SET invitee_user_id = ?
		WHERE invitee_email = ? AND invitee_user_id IS NULL AND status = 'pending'`, id, email); err != nil {
		return user{}, err
	}
	return scanUser(s.db.QueryRowContext(ctx, `SELECT `+userColumns()+` FROM users WHERE id = ?`, id))
}

func (s *Server) getMe(w http.ResponseWriter, r *http.Request) {
	value, err := scanUser(s.db.QueryRowContext(r.Context(), `SELECT `+userColumns()+` FROM users WHERE id = ?`, currentUserID(r)))
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	writeJSON(w, http.StatusOK, value.response())
}

func (s *Server) updateMe(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name               *string `json:"name"`
		AvatarURL          *string `json:"avatar_url"`
		Language           *string `json:"language"`
		Timezone           *string `json:"timezone"`
		ProfileDescription *string `json:"profile_description"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	value, err := scanUser(s.db.QueryRowContext(r.Context(), `SELECT `+userColumns()+` FROM users WHERE id = ?`, currentUserID(r)))
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	if req.Name != nil {
		value.Name = strings.TrimSpace(*req.Name)
	}
	if req.AvatarURL != nil {
		value.AvatarURL = sql.NullString{String: *req.AvatarURL, Valid: *req.AvatarURL != ""}
	}
	if req.Language != nil {
		value.Language = sql.NullString{String: *req.Language, Valid: *req.Language != ""}
	}
	if req.Timezone != nil {
		value.Timezone = sql.NullString{String: *req.Timezone, Valid: *req.Timezone != ""}
	}
	if req.ProfileDescription != nil {
		value.ProfileDescription = *req.ProfileDescription
	}
	value.UpdatedAt = now()
	_, err = s.db.ExecContext(r.Context(), `UPDATE users SET name = ?, avatar_url = ?, language = ?, timezone = ?,
		profile_description = ?, updated_at = ? WHERE id = ?`, value.Name, value.AvatarURL, value.Language,
		value.Timezone, value.ProfileDescription, value.UpdatedAt, value.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update user")
		return
	}
	writeJSON(w, http.StatusOK, value.response())
}

func (s *Server) patchOnboarding(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	var req struct {
		Questionnaire *json.RawMessage `json:"questionnaire"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	if req.Questionnaire != nil {
		var questionnaire map[string]any
		if err := json.Unmarshal(*req.Questionnaire, &questionnaire); err != nil {
			writeError(w, http.StatusBadRequest, "questionnaire must be a JSON object")
			return
		}
		encoded, err := json.Marshal(questionnaire)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid questionnaire")
			return
		}
		timestamp := now()
		if _, err := s.db.ExecContext(r.Context(), `UPDATE users
			SET onboarding_questionnaire = ?, updated_at = ? WHERE id = ?`,
			string(encoded), timestamp, currentUserID(r)); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to update onboarding")
			return
		}
	}

	value, err := scanUser(s.db.QueryRowContext(r.Context(), `SELECT `+userColumns()+` FROM users WHERE id = ?`, currentUserID(r)))
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	writeJSON(w, http.StatusOK, value.response())
}

func (s *Server) completeOnboarding(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CompletionPath string `json:"completion_path"`
		WorkspaceID    string `json:"workspace_id"`
	}
	if r.ContentLength != 0 && !decodeJSON(w, r, &req) {
		return
	}
	if req.WorkspaceID != "" {
		var exists int
		err := s.db.QueryRowContext(r.Context(), `SELECT 1 FROM members
			WHERE workspace_id = ? AND user_id = ?`, req.WorkspaceID, currentUserID(r)).Scan(&exists)
		if err != nil {
			writeError(w, http.StatusBadRequest, "workspace is not available to the current user")
			return
		}
	}

	timestamp := now()
	if _, err := s.db.ExecContext(r.Context(), `UPDATE users
		SET onboarded_at = COALESCE(onboarded_at, ?), updated_at = ? WHERE id = ?`,
		timestamp, timestamp, currentUserID(r)); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to complete onboarding")
		return
	}
	value, err := scanUser(s.db.QueryRowContext(r.Context(), `SELECT `+userColumns()+` FROM users WHERE id = ?`, currentUserID(r)))
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	writeJSON(w, http.StatusOK, value.response())
}

type workspace struct {
	ID, Name, Slug, IssuePrefix, CreatedAt, UpdatedAt string
	Description, Context, AvatarURL                   sql.NullString
	Settings, Repos                                   string
}

func workspaceColumns() string {
	return `id, name, slug, description, context, settings, repos, issue_prefix, avatar_url, created_at, updated_at`
}

func scanWorkspace(scanner interface{ Scan(...any) error }) (workspace, error) {
	var value workspace
	err := scanner.Scan(&value.ID, &value.Name, &value.Slug, &value.Description, &value.Context,
		&value.Settings, &value.Repos, &value.IssuePrefix, &value.AvatarURL, &value.CreatedAt, &value.UpdatedAt)
	return value, err
}

func (value workspace) response() map[string]any {
	return map[string]any{
		"id": value.ID, "name": value.Name, "slug": value.Slug,
		"description": nullable(value.Description.String), "context": nullable(value.Context.String),
		"settings": mapJSON(value.Settings, map[string]any{}), "repos": mapJSON(value.Repos, []any{}),
		"issue_prefix": value.IssuePrefix, "avatar_url": nullable(value.AvatarURL.String),
		"created_at": value.CreatedAt, "updated_at": value.UpdatedAt,
	}
}

func issuePrefix(name string) string {
	var letters strings.Builder
	for _, char := range name {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') {
			letters.WriteRune(char)
		}
	}
	value := strings.ToUpper(letters.String())
	if value == "" {
		return "WS"
	}
	if len(value) > 3 {
		return value[:3]
	}
	return value
}

func (s *Server) listWorkspaces(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.QueryContext(r.Context(), `SELECT `+workspaceColumns()+` FROM workspaces
		WHERE id IN (SELECT workspace_id FROM members WHERE user_id = ?) ORDER BY created_at`, currentUserID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list workspaces")
		return
	}
	defer rows.Close()
	result := make([]map[string]any, 0)
	for rows.Next() {
		value, err := scanWorkspace(rows)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list workspaces")
			return
		}
		result = append(result, value.response())
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) createWorkspace(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string  `json:"name"`
		Slug        string  `json:"slug"`
		Description *string `json:"description"`
		Context     *string `json:"context"`
		IssuePrefix *string `json:"issue_prefix"`
		AvatarURL   *string `json:"avatar_url"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Slug = strings.ToLower(strings.TrimSpace(req.Slug))
	if req.Name == "" || !slugPattern.MatchString(req.Slug) {
		writeError(w, http.StatusBadRequest, "valid name and slug are required")
		return
	}
	prefix := issuePrefix(req.Name)
	if req.IssuePrefix != nil && strings.TrimSpace(*req.IssuePrefix) != "" {
		prefix = strings.ToUpper(strings.TrimSpace(*req.IssuePrefix))
	}
	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create workspace")
		return
	}
	defer tx.Rollback()
	id, timestamp := newID(), now()
	_, err = tx.ExecContext(r.Context(), `INSERT INTO workspaces(
		id, name, slug, description, context, issue_prefix, avatar_url, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, id, req.Name, req.Slug, req.Description, req.Context, prefix, req.AvatarURL, timestamp, timestamp)
	if err != nil {
		writeError(w, http.StatusConflict, "workspace slug already exists")
		return
	}
	_, err = tx.ExecContext(r.Context(), `INSERT INTO members(id, workspace_id, user_id, role, created_at)
		VALUES (?, ?, ?, 'owner', ?)`, newID(), id, currentUserID(r), timestamp)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create workspace")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create workspace")
		return
	}
	value, _ := scanWorkspace(s.db.QueryRowContext(r.Context(), `SELECT `+workspaceColumns()+` FROM workspaces WHERE id = ?`, id))
	writeJSON(w, http.StatusCreated, value.response())
}

func (s *Server) getWorkspace(w http.ResponseWriter, r *http.Request) {
	value, ok := s.authorizedWorkspace(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, value.response())
}

func (s *Server) authorizedWorkspace(w http.ResponseWriter, r *http.Request, id string) (workspace, bool) {
	value, err := scanWorkspace(s.db.QueryRowContext(r.Context(), `SELECT `+workspaceColumns()+` FROM workspaces
		WHERE id = ? AND id IN (SELECT workspace_id FROM members WHERE user_id = ?)`, id, currentUserID(r)))
	if err != nil {
		writeError(w, http.StatusNotFound, "workspace not found")
		return workspace{}, false
	}
	return value, true
}

func (s *Server) resolveWorkspace(w http.ResponseWriter, r *http.Request) (workspace, bool) {
	slug := strings.TrimSpace(r.Header.Get("X-Workspace-Slug"))
	id := strings.TrimSpace(r.Header.Get("X-Workspace-ID"))
	if id == "" {
		id = strings.TrimSpace(r.URL.Query().Get("workspace_id"))
	}
	var row *sql.Row
	switch {
	case slug != "":
		row = s.db.QueryRowContext(r.Context(), `SELECT `+workspaceColumns()+` FROM workspaces
			WHERE slug = ? AND id IN (SELECT workspace_id FROM members WHERE user_id = ?)`, slug, currentUserID(r))
	case id != "":
		row = s.db.QueryRowContext(r.Context(), `SELECT `+workspaceColumns()+` FROM workspaces
			WHERE id = ? AND id IN (SELECT workspace_id FROM members WHERE user_id = ?)`, id, currentUserID(r))
	default:
		row = s.db.QueryRowContext(r.Context(), `SELECT `+workspaceColumns()+` FROM workspaces
			WHERE id IN (SELECT workspace_id FROM members WHERE user_id = ?) ORDER BY created_at LIMIT 1`, currentUserID(r))
	}
	value, err := scanWorkspace(row)
	if err != nil {
		writeError(w, http.StatusBadRequest, "workspace is required")
		return workspace{}, false
	}
	return value, true
}

func (s *Server) updateWorkspace(w http.ResponseWriter, r *http.Request) {
	value, ok := s.authorizedWorkspace(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	if !s.requireWorkspaceRole(
		w,
		r,
		value.ID,
		workspacepermissions.RoleOwner,
		workspacepermissions.RoleAdmin,
	) {
		return
	}
	var req struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		Context     *string `json:"context"`
		IssuePrefix *string `json:"issue_prefix"`
		AvatarURL   *string `json:"avatar_url"`
		Settings    any     `json:"settings"`
		Repos       any     `json:"repos"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Name != nil && strings.TrimSpace(*req.Name) != "" {
		value.Name = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		value.Description = sql.NullString{String: *req.Description, Valid: *req.Description != ""}
	}
	if req.Context != nil {
		value.Context = sql.NullString{String: *req.Context, Valid: *req.Context != ""}
	}
	if req.IssuePrefix != nil && strings.TrimSpace(*req.IssuePrefix) != "" {
		value.IssuePrefix = strings.ToUpper(strings.TrimSpace(*req.IssuePrefix))
	}
	if req.AvatarURL != nil {
		value.AvatarURL = sql.NullString{String: *req.AvatarURL, Valid: *req.AvatarURL != ""}
	}
	if req.Settings != nil {
		value.Settings = encodeJSON(req.Settings, "{}")
	}
	if req.Repos != nil {
		value.Repos = encodeJSON(req.Repos, "[]")
	}
	value.UpdatedAt = now()
	_, err := s.db.ExecContext(r.Context(), `UPDATE workspaces SET name = ?, description = ?, context = ?,
		settings = ?, repos = ?, issue_prefix = ?, avatar_url = ?, updated_at = ? WHERE id = ?`,
		value.Name, value.Description, value.Context, value.Settings, value.Repos, value.IssuePrefix, value.AvatarURL, value.UpdatedAt, value.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update workspace")
		return
	}
	writeJSON(w, http.StatusOK, value.response())
}

func (s *Server) deleteWorkspace(w http.ResponseWriter, r *http.Request) {
	value, ok := s.authorizedWorkspace(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	if !s.requireWorkspaceRole(
		w,
		r,
		value.ID,
		workspacepermissions.RoleOwner,
	) {
		return
	}
	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete workspace")
		return
	}
	defer tx.Rollback()
	statements := []string{
		`DELETE FROM tasks WHERE workspace_id = ?`,
		`DELETE FROM issue_reactions WHERE workspace_id = ?`,
		`DELETE FROM pinned_items WHERE workspace_id = ?`,
		`DELETE FROM project_requirement_issue_link WHERE workspace_id = ?`,
		`DELETE FROM project_requirement_revision WHERE baseline_id IN (SELECT id FROM project_requirement_baseline WHERE workspace_id = ?)`,
		`DELETE FROM project_requirement_baseline WHERE workspace_id = ?`,
		`DELETE FROM skill_files WHERE skill_id IN (SELECT id FROM skills WHERE workspace_id = ?)`,
		`DELETE FROM skills WHERE workspace_id = ?`,
		`DELETE FROM issues WHERE workspace_id = ?`,
		`DELETE FROM projects WHERE workspace_id = ?`,
		`DELETE FROM invitations WHERE workspace_id = ?`,
		`DELETE FROM members WHERE workspace_id = ?`,
		`DELETE FROM workspaces WHERE id = ?`,
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(r.Context(), statement, value.ID); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to delete workspace")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete workspace")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type member struct {
	ID, WorkspaceID, UserID, Role, CreatedAt, Name, Email string
	AvatarURL                                             sql.NullString
}

func memberResponse(value member) map[string]any {
	return map[string]any{
		"id": value.ID, "workspace_id": value.WorkspaceID, "user_id": value.UserID,
		"role": value.Role, "created_at": value.CreatedAt, "name": value.Name,
		"email": value.Email, "avatar_url": nullable(value.AvatarURL.String),
	}
}

type invitation struct {
	ID, WorkspaceID, InviterID, InviteeEmail, Role, Status string
	CreatedAt, UpdatedAt, ExpiresAt                        string
	WorkspaceName, InviterName, InviterEmail               string
	InviteeUserID                                          sql.NullString
}

func invitationSelect() string {
	return `i.id, i.workspace_id, i.inviter_id, i.invitee_email, i.invitee_user_id,
		i.role, i.status, i.created_at, i.updated_at, i.expires_at,
		w.name, u.name, u.email`
}

func scanInvitation(scanner interface{ Scan(...any) error }) (invitation, error) {
	var value invitation
	err := scanner.Scan(
		&value.ID, &value.WorkspaceID, &value.InviterID, &value.InviteeEmail, &value.InviteeUserID,
		&value.Role, &value.Status, &value.CreatedAt, &value.UpdatedAt, &value.ExpiresAt,
		&value.WorkspaceName, &value.InviterName, &value.InviterEmail,
	)
	return value, err
}

func invitationResponse(value invitation) map[string]any {
	return map[string]any{
		"id": value.ID, "workspace_id": value.WorkspaceID, "inviter_id": value.InviterID,
		"invitee_email": value.InviteeEmail, "invitee_user_id": nullable(value.InviteeUserID.String),
		"role": value.Role, "status": value.Status, "created_at": value.CreatedAt,
		"updated_at": value.UpdatedAt, "expires_at": value.ExpiresAt,
		"workspace_name": value.WorkspaceName, "inviter_name": value.InviterName,
		"inviter_email": value.InviterEmail,
	}
}

func (s *Server) listMembers(w http.ResponseWriter, r *http.Request) {
	workspaceValue, ok := s.authorizedWorkspace(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	ctx := authcontract.WithMemberActor(r.Context(), currentUserID(r))
	result, err := s.authMembers.ListMembers(ctx, authcontract.Member_ListMembersRequest{
		WorkspaceId: workspaceValue.ID,
	})
	if err != nil {
		writeMemberError(w, err, "failed to list members")
		return
	}
	writeJSON(w, http.StatusOK, memberContractResponses(result.Members))
}

func (s *Server) createMember(w http.ResponseWriter, r *http.Request) {
	workspaceValue, ok := s.authorizedWorkspace(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	authorizer, ok := s.authMembers.(authcontract.InvitationCreationAuthorizer)
	if !ok {
		writeError(w, http.StatusInternalServerError, "failed to authorize invitation")
		return
	}
	ctx := authcontract.WithMemberActor(r.Context(), currentUserID(r))
	if err := authorizer.AuthorizeCreateInvitation(ctx, authcontract.Member_CreateInvitationRequest{
		WorkspaceId: workspaceValue.ID,
	}); err != nil {
		writeMemberError(w, err, "failed to authorize invitation")
		return
	}
	var req struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	value, err := s.authMembers.CreateInvitation(ctx, authcontract.Member_CreateInvitationRequest{
		WorkspaceId: workspaceValue.ID,
		Email:       req.Email,
		Role:        req.Role,
	})
	if err != nil {
		writeMemberError(w, err, "failed to create invitation")
		return
	}
	writeJSON(w, http.StatusCreated, invitationContractResponse(value))
}

func (s *Server) listWorkspaceInvitations(w http.ResponseWriter, r *http.Request) {
	workspaceValue, ok := s.authorizedWorkspace(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	ctx := authcontract.WithMemberActor(r.Context(), currentUserID(r))
	result, err := s.authMembers.ListWorkspaceInvitations(ctx, authcontract.Member_ListWorkspaceInvitationsRequest{
		WorkspaceId: workspaceValue.ID,
	})
	if err != nil {
		writeMemberError(w, err, "failed to list invitations")
		return
	}
	writeJSON(w, http.StatusOK, invitationContractResponses(result.Invitations))
}

func (s *Server) revokeInvitation(w http.ResponseWriter, r *http.Request) {
	workspaceValue, ok := s.authorizedWorkspace(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	ctx := authcontract.WithMemberActor(r.Context(), currentUserID(r))
	_, err := s.authMembers.RevokeInvitation(ctx, authcontract.Member_RevokeInvitationRequest{
		WorkspaceId:  workspaceValue.ID,
		InvitationId: chi.URLParam(r, "invitationID"),
	})
	if err != nil {
		writeMemberError(w, err, "failed to revoke invitation")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listMyInvitations(w http.ResponseWriter, r *http.Request) {
	ctx := authcontract.WithMemberActor(r.Context(), currentUserID(r))
	result, err := s.authMembers.ListMyInvitations(ctx, authcontract.Member_ListMyInvitationsRequest{})
	if err != nil {
		writeMemberError(w, err, "failed to list invitations")
		return
	}
	writeJSON(w, http.StatusOK, invitationContractResponses(result.Invitations))
}

func (s *Server) getMyInvitation(w http.ResponseWriter, r *http.Request) {
	ctx := authcontract.WithMemberActor(r.Context(), currentUserID(r))
	result, err := s.authMembers.GetMyInvitation(ctx, authcontract.Member_GetMyInvitationRequest{
		InvitationId: chi.URLParam(r, "id"),
	})
	if err != nil {
		writeMemberError(w, err, "failed to load invitation")
		return
	}
	if result.Invitation == nil {
		writeError(w, http.StatusInternalServerError, "failed to load invitation")
		return
	}
	writeJSON(w, http.StatusOK, invitationContractResponse(*result.Invitation))
}

func (s *Server) acceptInvitation(w http.ResponseWriter, r *http.Request) {
	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to accept invitation")
		return
	}
	defer tx.Rollback()

	current, err := scanUser(tx.QueryRowContext(r.Context(), `SELECT `+userColumns()+` FROM users WHERE id = ?`, currentUserID(r)))
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	value, err := scanInvitation(tx.QueryRowContext(r.Context(), `SELECT `+invitationSelect()+`
		FROM invitations i
		JOIN workspaces w ON w.id = i.workspace_id
		JOIN users u ON u.id = i.inviter_id
		WHERE i.id = ?`, chi.URLParam(r, "id")))
	if err != nil {
		writeError(w, http.StatusNotFound, "invitation not found")
		return
	}
	if value.InviteeEmail != current.Email && (!value.InviteeUserID.Valid || value.InviteeUserID.String != current.ID) {
		writeError(w, http.StatusForbidden, "invitation does not belong to you")
		return
	}
	if value.Status != "pending" {
		writeError(w, http.StatusBadRequest, "invitation is not pending")
		return
	}
	expiresAt, err := time.Parse(time.RFC3339Nano, value.ExpiresAt)
	if err != nil || !expiresAt.After(time.Now()) {
		if _, updateErr := tx.ExecContext(r.Context(), `UPDATE invitations
			SET status = 'expired', updated_at = ? WHERE id = ? AND status = 'pending'`,
			now(), value.ID); updateErr == nil {
			_ = tx.Commit()
		}
		writeError(w, http.StatusGone, "invitation has expired")
		return
	}

	memberID, timestamp := newID(), now()
	if _, err := tx.ExecContext(r.Context(), `INSERT INTO members(id, workspace_id, user_id, role, created_at)
		VALUES (?, ?, ?, ?, ?)`, memberID, value.WorkspaceID, current.ID, value.Role, timestamp); err != nil {
		writeError(w, http.StatusConflict, "you are already a member of this workspace")
		return
	}
	result, err := tx.ExecContext(r.Context(), `UPDATE invitations
		SET status = 'accepted', invitee_user_id = ?, updated_at = ?
		WHERE id = ? AND status = 'pending'`, current.ID, timestamp, value.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to accept invitation")
		return
	}
	affected, err := result.RowsAffected()
	if err != nil || affected == 0 {
		writeError(w, http.StatusConflict, "invitation is no longer pending")
		return
	}
	if _, err := tx.ExecContext(r.Context(), `UPDATE users
		SET onboarded_at = COALESCE(onboarded_at, ?), updated_at = ? WHERE id = ?`,
		timestamp, timestamp, current.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to complete onboarding")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to accept invitation")
		return
	}

	writeJSON(w, http.StatusOK, memberResponse(member{
		ID: memberID, WorkspaceID: value.WorkspaceID, UserID: current.ID,
		Role: value.Role, CreatedAt: timestamp, Name: current.Name, Email: current.Email,
		AvatarURL: current.AvatarURL,
	}))
}

func (s *Server) declineInvitation(w http.ResponseWriter, r *http.Request) {
	current, err := scanUser(s.db.QueryRowContext(r.Context(), `SELECT `+userColumns()+` FROM users WHERE id = ?`, currentUserID(r)))
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	value, err := scanInvitation(s.db.QueryRowContext(r.Context(), `SELECT `+invitationSelect()+`
		FROM invitations i
		JOIN workspaces w ON w.id = i.workspace_id
		JOIN users u ON u.id = i.inviter_id
		WHERE i.id = ?`, chi.URLParam(r, "id")))
	if err != nil {
		writeError(w, http.StatusNotFound, "invitation not found")
		return
	}
	if value.InviteeEmail != current.Email && (!value.InviteeUserID.Valid || value.InviteeUserID.String != current.ID) {
		writeError(w, http.StatusForbidden, "invitation does not belong to you")
		return
	}
	result, err := s.db.ExecContext(r.Context(), `UPDATE invitations
		SET status = 'declined', invitee_user_id = ?, updated_at = ?
		WHERE id = ? AND status = 'pending'`, current.ID, now(), value.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to decline invitation")
		return
	}
	affected, err := result.RowsAffected()
	if err != nil || affected == 0 {
		writeError(w, http.StatusBadRequest, "invitation is not pending")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) updateMember(w http.ResponseWriter, r *http.Request) {
	workspaceValue, ok := s.authorizedWorkspace(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req struct {
		Role string `json:"role"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	ctx := authcontract.WithMemberActor(r.Context(), currentUserID(r))
	updated, err := s.authMembers.UpdateMemberRole(ctx, authcontract.Member_UpdateMemberRoleRequest{
		WorkspaceId: workspaceValue.ID,
		MemberId:    chi.URLParam(r, "memberID"),
		Role:        req.Role,
	})
	if err != nil {
		writeMemberError(w, err, "failed to update member")
		return
	}
	writeJSON(w, http.StatusOK, memberContractResponse(updated))
}

func (s *Server) deleteMember(w http.ResponseWriter, r *http.Request) {
	workspaceValue, ok := s.authorizedWorkspace(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	ctx := authcontract.WithMemberActor(r.Context(), currentUserID(r))
	_, err := s.authMembers.DeleteMember(ctx, authcontract.Member_DeleteMemberRequest{
		WorkspaceId: workspaceValue.ID,
		MemberId:    chi.URLParam(r, "memberID"),
	})
	if err != nil {
		writeMemberError(w, err, "failed to delete member")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) leaveWorkspace(w http.ResponseWriter, r *http.Request) {
	workspaceValue, ok := s.authorizedWorkspace(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	ctx := authcontract.WithMemberActor(r.Context(), currentUserID(r))
	_, err := s.authMembers.LeaveWorkspace(ctx, authcontract.Member_LeaveWorkspaceRequest{
		WorkspaceId: workspaceValue.ID,
	})
	if err != nil {
		writeMemberError(w, err, "failed to leave workspace")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type project struct {
	ID, WorkspaceID, Title, Status, Priority, CreatedAt, UpdatedAt string
	Description, Icon, LeadType, LeadID, StartDate, DueDate        sql.NullString
}

func projectColumns() string {
	return `id, workspace_id, title, description, icon, status, priority, lead_type, lead_id,
		start_date, due_date, created_at, updated_at`
}

func scanProject(scanner interface{ Scan(...any) error }) (project, error) {
	var value project
	err := scanner.Scan(&value.ID, &value.WorkspaceID, &value.Title, &value.Description, &value.Icon,
		&value.Status, &value.Priority, &value.LeadType, &value.LeadID, &value.StartDate, &value.DueDate,
		&value.CreatedAt, &value.UpdatedAt)
	return value, err
}

func (s *Server) projectResponse(ctx context.Context, value project) map[string]any {
	var issueCount, doneCount, resourceCount int64
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*), COUNT(CASE WHEN status = 'done' THEN 1 END)
		FROM issues WHERE project_id = ?`, value.ID).Scan(&issueCount, &doneCount)
	return map[string]any{
		"id": value.ID, "workspace_id": value.WorkspaceID, "title": value.Title,
		"description": nullable(value.Description.String), "icon": nullable(value.Icon.String),
		"status": value.Status, "priority": value.Priority, "lead_type": nullable(value.LeadType.String),
		"lead_id": nullable(value.LeadID.String), "start_date": nullable(value.StartDate.String),
		"due_date": nullable(value.DueDate.String), "created_at": value.CreatedAt, "updated_at": value.UpdatedAt,
		"issue_count": issueCount, "done_count": doneCount, "resource_count": resourceCount,
	}
}

type projectRequest struct {
	Title       string  `json:"title"`
	Status      string  `json:"status"`
	Priority    string  `json:"priority"`
	Description *string `json:"description"`
	Icon        *string `json:"icon"`
	LeadType    *string `json:"lead_type"`
	LeadID      *string `json:"lead_id"`
	StartDate   *string `json:"start_date"`
	DueDate     *string `json:"due_date"`
}

func (s *Server) listProjects(w http.ResponseWriter, r *http.Request) {
	workspaceValue, ok := s.resolveWorkspace(w, r)
	if !ok {
		return
	}
	status := r.URL.Query().Get("status")
	query := `SELECT ` + projectColumns() + ` FROM projects WHERE workspace_id = ?`
	args := []any{workspaceValue.ID}
	if status != "" {
		query += ` AND status = ?`
		args = append(args, status)
	}
	query += ` ORDER BY updated_at DESC`
	rows, err := s.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list projects")
		return
	}
	values := make([]project, 0)
	for rows.Next() {
		value, err := scanProject(rows)
		if err != nil {
			_ = rows.Close()
			writeError(w, http.StatusInternalServerError, "failed to list projects")
			return
		}
		values = append(values, value)
	}
	if err := rows.Close(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list projects")
		return
	}
	projects := make([]map[string]any, 0, len(values))
	for _, value := range values {
		projects = append(projects, s.projectResponse(r.Context(), value))
	}
	writeJSON(w, http.StatusOK, map[string]any{"projects": projects, "total": len(projects)})
}

func (s *Server) createProject(w http.ResponseWriter, r *http.Request) {
	workspaceValue, ok := s.resolveWorkspace(w, r)
	if !ok {
		return
	}
	var req projectRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	if req.Status == "" {
		req.Status = "planned"
	}
	if req.Priority == "" {
		req.Priority = "none"
	}
	id, timestamp := newID(), now()
	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create project")
		return
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(r.Context(), `INSERT INTO projects(
		id, workspace_id, title, description, icon, status, priority, lead_type, lead_id,
		start_date, due_date, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, id, workspaceValue.ID, req.Title,
		req.Description, req.Icon, req.Status, req.Priority, req.LeadType, req.LeadID,
		req.StartDate, req.DueDate, timestamp, timestamp)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create project")
		return
	}
	value, err := scanProject(tx.QueryRowContext(r.Context(), `SELECT `+projectColumns()+` FROM projects WHERE id = ?`, id))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load project")
		return
	}
	if err := enqueueKnowledgeEvidence(r.Context(), tx, projectEvidence(value, currentUserID(r), "project.created")); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to record project evidence")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create project")
		return
	}
	s.dispatchKnowledgeEvidence(r.Context())
	writeJSON(w, http.StatusCreated, s.projectResponse(r.Context(), value))
}

func (s *Server) loadProject(w http.ResponseWriter, r *http.Request, id string) (project, bool) {
	workspaceValue, ok := s.resolveWorkspace(w, r)
	if !ok {
		return project{}, false
	}
	value, err := scanProject(s.db.QueryRowContext(r.Context(), `SELECT `+projectColumns()+` FROM projects WHERE id = ? AND workspace_id = ?`, id, workspaceValue.ID))
	if err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return project{}, false
	}
	return value, true
}

func (s *Server) getProject(w http.ResponseWriter, r *http.Request) {
	value, ok := s.loadProject(w, r, chi.URLParam(r, "id"))
	if ok {
		writeJSON(w, http.StatusOK, s.projectResponse(r.Context(), value))
	}
}

func (s *Server) updateProject(w http.ResponseWriter, r *http.Request) {
	value, ok := s.loadProject(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	previousStatus := value.Status
	var req projectRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Title) != "" {
		value.Title = strings.TrimSpace(req.Title)
	}
	if req.Status != "" {
		value.Status = req.Status
	}
	if req.Priority != "" {
		value.Priority = req.Priority
	}
	applyNullString(&value.Description, req.Description)
	applyNullString(&value.Icon, req.Icon)
	applyNullString(&value.LeadType, req.LeadType)
	applyNullString(&value.LeadID, req.LeadID)
	applyNullString(&value.StartDate, req.StartDate)
	applyNullString(&value.DueDate, req.DueDate)
	value.UpdatedAt = now()
	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update project")
		return
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(r.Context(), `UPDATE projects SET title = ?, description = ?, icon = ?, status = ?,
		priority = ?, lead_type = ?, lead_id = ?, start_date = ?, due_date = ?, updated_at = ? WHERE id = ?`,
		value.Title, value.Description, value.Icon, value.Status, value.Priority, value.LeadType,
		value.LeadID, value.StartDate, value.DueDate, value.UpdatedAt, value.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update project")
		return
	}
	if previousStatus != value.Status && (value.Status == "completed" || value.Status == "done") {
		if err := enqueueKnowledgeEvidence(r.Context(), tx, projectEvidence(value, currentUserID(r), "project.completed")); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to record project evidence")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update project")
		return
	}
	s.dispatchKnowledgeEvidence(r.Context())
	writeJSON(w, http.StatusOK, s.projectResponse(r.Context(), value))
}

func applyNullString(target *sql.NullString, input *string) {
	if input != nil {
		target.String = *input
		target.Valid = strings.TrimSpace(*input) != ""
	}
}

func (s *Server) deleteProject(w http.ResponseWriter, r *http.Request) {
	value, ok := s.loadProject(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	if !s.requireWorkspaceRole(
		w,
		r,
		value.WorkspaceID,
		workspacepermissions.RoleOwner,
		workspacepermissions.RoleAdmin,
	) {
		return
	}
	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete project")
		return
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(r.Context(), `DELETE FROM project_requirement_revision
		WHERE baseline_id IN (SELECT id FROM project_requirement_baseline WHERE workspace_id = ? AND project_id = ?)`, value.WorkspaceID, value.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete project requirements")
		return
	}
	if _, err := tx.ExecContext(r.Context(), `DELETE FROM project_requirement_issue_link WHERE workspace_id = ? AND project_id = ?`, value.WorkspaceID, value.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete project requirement links")
		return
	}
	if _, err := tx.ExecContext(r.Context(), `DELETE FROM project_requirement_baseline WHERE workspace_id = ? AND project_id = ?`, value.WorkspaceID, value.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete project requirements")
		return
	}
	if _, err := tx.ExecContext(r.Context(), `DELETE FROM project_retrospective WHERE project_id = ?`, value.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete project")
		return
	}
	if _, err := tx.ExecContext(r.Context(), `DELETE FROM pinned_items WHERE item_type = 'project' AND item_id = ?`, value.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete project")
		return
	}
	if _, err := tx.ExecContext(r.Context(), `UPDATE issues SET project_id = NULL WHERE project_id = ?`, value.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete project")
		return
	}
	if _, err := tx.ExecContext(r.Context(), `DELETE FROM projects WHERE id = ?`, value.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete project")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete project")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type issue struct {
	ID, WorkspaceID, Title, Status, Priority, CreatorType, CreatorID, CreatedAt, UpdatedAt string
	Number                                                                                 int64
	Description, AssigneeType, AssigneeID, ParentIssueID, ProjectID, StartDate, DueDate    sql.NullString
	Position                                                                               float64
	Stage                                                                                  sql.NullInt64
	Metadata, Properties                                                                   string
}

func issueColumns() string {
	return `id, workspace_id, number, title, description, status, priority, assignee_type, assignee_id,
		creator_type, creator_id, parent_issue_id, project_id, position, stage, start_date, due_date,
		metadata, properties, created_at, updated_at`
}

func scanIssue(scanner interface{ Scan(...any) error }) (issue, error) {
	var value issue
	err := scanner.Scan(&value.ID, &value.WorkspaceID, &value.Number, &value.Title, &value.Description,
		&value.Status, &value.Priority, &value.AssigneeType, &value.AssigneeID, &value.CreatorType,
		&value.CreatorID, &value.ParentIssueID, &value.ProjectID, &value.Position, &value.Stage,
		&value.StartDate, &value.DueDate, &value.Metadata, &value.Properties, &value.CreatedAt, &value.UpdatedAt)
	return value, err
}

func (value issue) response(prefix string) map[string]any {
	var stage any
	if value.Stage.Valid {
		stage = value.Stage.Int64
	}
	return map[string]any{
		"id": value.ID, "workspace_id": value.WorkspaceID, "number": value.Number,
		"identifier": prefix + "-" + strconv.FormatInt(value.Number, 10), "title": value.Title,
		"description": nullable(value.Description.String), "status": value.Status, "priority": value.Priority,
		"assignee_type": nullable(value.AssigneeType.String), "assignee_id": nullable(value.AssigneeID.String),
		"creator_type": value.CreatorType, "creator_id": value.CreatorID,
		"parent_issue_id": nullable(value.ParentIssueID.String), "project_id": nullable(value.ProjectID.String),
		"position": value.Position, "stage": stage, "start_date": nullable(value.StartDate.String),
		"due_date": nullable(value.DueDate.String), "metadata": mapJSON(value.Metadata, map[string]any{}),
		"properties": mapJSON(value.Properties, map[string]any{}), "created_at": value.CreatedAt, "updated_at": value.UpdatedAt,
	}
}

type issueRequest struct {
	Title                string                       `json:"title"`
	Status               string                       `json:"status"`
	Priority             string                       `json:"priority"`
	Description          *string                      `json:"description"`
	AssigneeType         *string                      `json:"assignee_type"`
	AssigneeID           *string                      `json:"assignee_id"`
	ParentIssueID        *string                      `json:"parent_issue_id"`
	ProjectID            *string                      `json:"project_id"`
	StartDate            *string                      `json:"start_date"`
	DueDate              *string                      `json:"due_date"`
	Stage                *int64                       `json:"stage"`
	Metadata             map[string]any               `json:"metadata"`
	Properties           map[string]any               `json:"properties"`
	AcceptanceConclusion *acceptanceConclusionRequest `json:"acceptance_conclusion"`
}

func (s *Server) listIssues(w http.ResponseWriter, r *http.Request) {
	workspaceValue, ok := s.resolveWorkspace(w, r)
	if !ok {
		return
	}
	query := `SELECT ` + issueColumns() + ` FROM issues WHERE workspace_id = ?`
	args := []any{workspaceValue.ID}
	if status := r.URL.Query().Get("status"); status != "" {
		query += ` AND status = ?`
		args = append(args, status)
	}
	if projectID := r.URL.Query().Get("project_id"); projectID != "" {
		query += ` AND project_id = ?`
		args = append(args, projectID)
	}
	query += ` ORDER BY number DESC`
	rows, err := s.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list issues")
		return
	}
	defer rows.Close()
	issues := make([]map[string]any, 0)
	for rows.Next() {
		value, err := scanIssue(rows)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list issues")
			return
		}
		issues = append(issues, value.response(workspaceValue.IssuePrefix))
	}
	writeJSON(w, http.StatusOK, map[string]any{"issues": issues, "total": len(issues)})
}

func (s *Server) listProperties(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.resolveWorkspace(w, r); !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"properties": []any{}, "total": 0})
}

func (s *Server) createIssue(w http.ResponseWriter, r *http.Request) {
	workspaceValue, ok := s.resolveWorkspace(w, r)
	if !ok {
		return
	}
	var req issueRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	if req.Status == "" {
		req.Status = "todo"
	}
	if req.Priority == "" {
		req.Priority = "none"
	}
	if req.ProjectID != nil && !s.belongsToWorkspace(r.Context(), "projects", *req.ProjectID, workspaceValue.ID) {
		writeError(w, http.StatusBadRequest, "project not found in workspace")
		return
	}
	if req.ParentIssueID != nil && !s.belongsToWorkspace(r.Context(), "issues", *req.ParentIssueID, workspaceValue.ID) {
		writeError(w, http.StatusBadRequest, "parent issue not found in workspace")
		return
	}
	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create issue")
		return
	}
	defer tx.Rollback()
	value, _, err := s.createIssueTx(r.Context(), tx, workspaceValue.ID, currentUserID(r), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create issue")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create issue")
		return
	}
	s.dispatchKnowledgeEvidence(r.Context())
	writeJSON(w, http.StatusCreated, value.response(workspaceValue.IssuePrefix))
}

func (s *Server) createIssueTx(ctx context.Context, tx *sql.Tx, workspaceID, actorID string, req issueRequest) (issue, workspace, error) {
	workspaceValue, err := scanWorkspace(tx.QueryRowContext(ctx, `SELECT `+workspaceColumns()+` FROM workspaces WHERE id = ?`, workspaceID))
	if err != nil {
		return issue{}, workspace{}, err
	}
	var number int64
	if err := tx.QueryRowContext(ctx, `UPDATE workspaces SET next_issue_number = next_issue_number + 1,
		updated_at = ? WHERE id = ? RETURNING next_issue_number - 1`, now(), workspaceValue.ID).Scan(&number); err != nil {
		return issue{}, workspace{}, err
	}
	id, timestamp := newID(), now()
	_, err = tx.ExecContext(ctx, `INSERT INTO issues(
		id, workspace_id, number, title, description, status, priority, assignee_type, assignee_id,
		creator_type, creator_id, parent_issue_id, project_id, stage, start_date, due_date,
		metadata, properties, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'member', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, workspaceValue.ID, number, req.Title, req.Description, req.Status, req.Priority,
		req.AssigneeType, req.AssigneeID, actorID, req.ParentIssueID, req.ProjectID,
		req.Stage, req.StartDate, req.DueDate, encodeJSON(req.Metadata, "{}"),
		encodeJSON(req.Properties, "{}"), timestamp, timestamp)
	if err != nil {
		return issue{}, workspace{}, err
	}
	value, err := scanIssue(tx.QueryRowContext(ctx, `SELECT `+issueColumns()+` FROM issues WHERE id = ?`, id))
	if err != nil {
		return issue{}, workspace{}, err
	}
	if err := enqueueKnowledgeEvidence(ctx, tx, issueEvidence(value, actorID, "issue.created")); err != nil {
		return issue{}, workspace{}, err
	}
	return value, workspaceValue, nil
}

func (s *Server) loadIssue(w http.ResponseWriter, r *http.Request, id string) (issue, workspace, bool) {
	workspaceValue, ok := s.resolveWorkspace(w, r)
	if !ok {
		return issue{}, workspace{}, false
	}
	value, err := scanIssue(s.db.QueryRowContext(r.Context(), `SELECT `+issueColumns()+` FROM issues WHERE id = ? AND workspace_id = ?`, id, workspaceValue.ID))
	if err != nil {
		writeError(w, http.StatusNotFound, "issue not found")
		return issue{}, workspace{}, false
	}
	return value, workspaceValue, true
}

func (s *Server) getIssue(w http.ResponseWriter, r *http.Request) {
	value, workspaceValue, ok := s.loadIssue(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	response := value.response(workspaceValue.IssuePrefix)
	reactions, err := s.listIssueReactions(r.Context(), value.ID, value.WorkspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list issue reactions")
		return
	}
	response["reactions"] = reactions
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) updateIssue(w http.ResponseWriter, r *http.Request) {
	value, workspaceValue, ok := s.loadIssue(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req issueRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.ProjectID != nil && *req.ProjectID != "" && !s.belongsToWorkspace(r.Context(), "projects", *req.ProjectID, workspaceValue.ID) {
		writeError(w, http.StatusBadRequest, "project not found in workspace")
		return
	}
	if req.ParentIssueID != nil && *req.ParentIssueID != "" && !s.belongsToWorkspace(r.Context(), "issues", *req.ParentIssueID, workspaceValue.ID) {
		writeError(w, http.StatusBadRequest, "parent issue not found in workspace")
		return
	}
	if strings.TrimSpace(req.Title) != "" {
		value.Title = strings.TrimSpace(req.Title)
	}
	if req.Status != "" {
		value.Status = req.Status
	}
	if req.Priority != "" {
		value.Priority = req.Priority
	}
	applyNullString(&value.Description, req.Description)
	applyNullString(&value.AssigneeType, req.AssigneeType)
	applyNullString(&value.AssigneeID, req.AssigneeID)
	applyNullString(&value.ParentIssueID, req.ParentIssueID)
	applyNullString(&value.ProjectID, req.ProjectID)
	applyNullString(&value.StartDate, req.StartDate)
	applyNullString(&value.DueDate, req.DueDate)
	if req.Stage != nil {
		value.Stage = sql.NullInt64{Int64: *req.Stage, Valid: *req.Stage > 0}
	}
	if req.Metadata != nil {
		value.Metadata = encodeJSON(req.Metadata, "{}")
	}
	if req.Properties != nil {
		value.Properties = encodeJSON(req.Properties, "{}")
	}
	if req.AcceptanceConclusion != nil {
		if value.Status != "done" {
			writeError(w, http.StatusBadRequest, "acceptance conclusion requires done status")
			return
		}
		if err := req.AcceptanceConclusion.validate(); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	value.UpdatedAt = now()
	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update issue")
		return
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(r.Context(), `UPDATE issues SET title = ?, description = ?, status = ?, priority = ?,
		assignee_type = ?, assignee_id = ?, parent_issue_id = ?, project_id = ?, stage = ?, start_date = ?,
		due_date = ?, metadata = ?, properties = ?, updated_at = ? WHERE id = ?`, value.Title, value.Description,
		value.Status, value.Priority, value.AssigneeType, value.AssigneeID, value.ParentIssueID, value.ProjectID,
		value.Stage, value.StartDate, value.DueDate, value.Metadata, value.Properties, value.UpdatedAt, value.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update issue")
		return
	}
	if req.AcceptanceConclusion != nil {
		conclusion, err := insertIssueAcceptanceConclusion(r.Context(), tx, value, currentUserID(r), *req.AcceptanceConclusion)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to record issue evidence")
			return
		}
		if err := enqueueKnowledgeEvidence(r.Context(), tx, acceptanceConclusionEvidence(value, conclusion)); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to record issue evidence")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update issue")
		return
	}
	s.dispatchKnowledgeEvidence(r.Context())
	writeJSON(w, http.StatusOK, value.response(workspaceValue.IssuePrefix))
}

func (s *Server) deleteIssue(w http.ResponseWriter, r *http.Request) {
	value, _, ok := s.loadIssue(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete issue")
		return
	}
	defer tx.Rollback()
	statements := []string{
		`DELETE FROM tasks WHERE issue_id = ?`,
		`DELETE FROM issue_reactions WHERE issue_id = ?`,
		`DELETE FROM pinned_items WHERE item_type = 'issue' AND item_id = ?`,
		`DELETE FROM project_requirement_issue_link WHERE issue_id = ?`,
		`DELETE FROM issue_acceptance_conclusion WHERE issue_id = ?`,
		`UPDATE issues SET parent_issue_id = NULL WHERE parent_issue_id = ?`,
		`DELETE FROM issues WHERE id = ?`,
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(r.Context(), statement, value.ID); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to delete issue")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete issue")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type task struct {
	ID, WorkspaceID, Title, Description, Status, Priority, CreatorID string
	ProjectID, IssueID, AssigneeID, StartDate, DueDate, CompletedAt  sql.NullString
	Position                                                         float64
	CreatedAt, UpdatedAt                                             string
}

var validTaskStatuses = map[string]struct{}{
	"todo": {}, "in_progress": {}, "done": {}, "cancelled": {},
}

var validTaskPriorities = map[string]struct{}{
	"urgent": {}, "high": {}, "medium": {}, "low": {}, "none": {},
}

func taskColumns() string {
	return `id, workspace_id, project_id, issue_id, title, description, status, priority,
		assignee_id, creator_id, position, start_date, due_date, completed_at, created_at, updated_at`
}

func scanTask(scanner interface{ Scan(...any) error }) (task, error) {
	var value task
	err := scanner.Scan(
		&value.ID,
		&value.WorkspaceID,
		&value.ProjectID,
		&value.IssueID,
		&value.Title,
		&value.Description,
		&value.Status,
		&value.Priority,
		&value.AssigneeID,
		&value.CreatorID,
		&value.Position,
		&value.StartDate,
		&value.DueDate,
		&value.CompletedAt,
		&value.CreatedAt,
		&value.UpdatedAt,
	)
	return value, err
}

func (value task) response() map[string]any {
	return map[string]any{
		"id":           value.ID,
		"workspace_id": value.WorkspaceID,
		"project_id":   nullable(value.ProjectID.String),
		"issue_id":     nullable(value.IssueID.String),
		"title":        value.Title,
		"description":  value.Description,
		"status":       value.Status,
		"priority":     value.Priority,
		"assignee_id":  nullable(value.AssigneeID.String),
		"creator_id":   value.CreatorID,
		"position":     value.Position,
		"start_date":   nullable(value.StartDate.String),
		"due_date":     nullable(value.DueDate.String),
		"completed_at": nullable(value.CompletedAt.String),
		"created_at":   value.CreatedAt,
		"updated_at":   value.UpdatedAt,
	}
}

func validTaskStatus(value string) bool {
	_, ok := validTaskStatuses[value]
	return ok
}

func validTaskPriority(value string) bool {
	_, ok := validTaskPriorities[value]
	return ok
}

func (s *Server) workspaceHasMember(ctx context.Context, workspaceID, userID string) bool {
	var exists int
	err := s.db.QueryRowContext(
		ctx,
		`SELECT 1 FROM members WHERE workspace_id = ? AND user_id = ?`,
		workspaceID,
		userID,
	).Scan(&exists)
	return err == nil
}

func (s *Server) createTask(w http.ResponseWriter, r *http.Request) {
	workspaceValue, ok := s.resolveWorkspace(w, r)
	if !ok {
		return
	}
	var req struct {
		ProjectID   string  `json:"project_id"`
		IssueID     string  `json:"issue_id"`
		Title       string  `json:"title"`
		Description string  `json:"description"`
		Status      string  `json:"status"`
		Priority    string  `json:"priority"`
		AssigneeID  string  `json:"assignee_id"`
		Position    float64 `json:"position"`
		StartDate   string  `json:"start_date"`
		DueDate     string  `json:"due_date"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	if req.Status == "" {
		req.Status = "todo"
	}
	if req.Priority == "" {
		req.Priority = "none"
	}
	if !validTaskStatus(req.Status) {
		writeError(w, http.StatusBadRequest, "invalid task status")
		return
	}
	if !validTaskPriority(req.Priority) {
		writeError(w, http.StatusBadRequest, "invalid task priority")
		return
	}
	if req.ProjectID != "" && !s.belongsToWorkspace(r.Context(), "projects", req.ProjectID, workspaceValue.ID) {
		writeError(w, http.StatusBadRequest, "project does not belong to workspace")
		return
	}
	if req.IssueID != "" && !s.belongsToWorkspace(r.Context(), "issues", req.IssueID, workspaceValue.ID) {
		writeError(w, http.StatusBadRequest, "issue does not belong to workspace")
		return
	}
	if req.AssigneeID != "" && !s.workspaceHasMember(r.Context(), workspaceValue.ID, req.AssigneeID) {
		writeError(w, http.StatusBadRequest, "assignee must be a workspace member")
		return
	}

	id, timestamp := newID(), now()
	completedAt := any(nil)
	if req.Status == "done" || req.Status == "cancelled" {
		completedAt = timestamp
	}
	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create task")
		return
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(r.Context(), `INSERT INTO tasks(
		id, workspace_id, project_id, issue_id, title, description, status, priority,
		assignee_id, creator_id, position, start_date, due_date, completed_at, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id,
		workspaceValue.ID,
		nullable(req.ProjectID),
		nullable(req.IssueID),
		req.Title,
		req.Description,
		req.Status,
		req.Priority,
		nullable(req.AssigneeID),
		currentUserID(r),
		req.Position,
		nullable(req.StartDate),
		nullable(req.DueDate),
		completedAt,
		timestamp,
		timestamp,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create task")
		return
	}
	value, err := scanTask(tx.QueryRowContext(
		r.Context(),
		`SELECT `+taskColumns()+` FROM tasks WHERE id = ?`,
		id,
	))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load task")
		return
	}
	if value.Status == "done" || value.Status == "cancelled" {
		if err := enqueueKnowledgeEvidence(r.Context(), tx, taskEvidence(value, currentUserID(r), "task.completed")); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to record task evidence")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create task")
		return
	}
	s.dispatchKnowledgeEvidence(r.Context())
	writeJSON(w, http.StatusCreated, value.response())
}

func (s *Server) listTasks(w http.ResponseWriter, r *http.Request) {
	workspaceValue, ok := s.resolveWorkspace(w, r)
	if !ok {
		return
	}
	clauses := []string{"workspace_id = ?"}
	args := []any{workspaceValue.ID}
	for queryKey, column := range map[string]string{
		"project_id": "project_id",
		"issue_id":   "issue_id",
	} {
		if value := strings.TrimSpace(r.URL.Query().Get(queryKey)); value != "" {
			clauses = append(clauses, column+" = ?")
			args = append(args, value)
		}
	}
	if status := strings.TrimSpace(r.URL.Query().Get("status")); status != "" {
		if !validTaskStatus(status) {
			writeError(w, http.StatusBadRequest, "invalid task status")
			return
		}
		clauses = append(clauses, "status = ?")
		args = append(args, status)
	}

	rows, err := s.db.QueryContext(
		r.Context(),
		`SELECT `+taskColumns()+` FROM tasks WHERE `+strings.Join(clauses, " AND ")+
			` ORDER BY position, created_at DESC`,
		args...,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list tasks")
		return
	}
	defer rows.Close()
	result := make([]map[string]any, 0)
	for rows.Next() {
		value, err := scanTask(rows)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list tasks")
			return
		}
		result = append(result, value.response())
	}
	writeJSON(w, http.StatusOK, map[string]any{"tasks": result, "total": len(result)})
}

func (s *Server) loadTask(w http.ResponseWriter, r *http.Request, id string) (task, bool) {
	workspaceValue, ok := s.resolveWorkspace(w, r)
	if !ok {
		return task{}, false
	}
	value, err := scanTask(s.db.QueryRowContext(
		r.Context(),
		`SELECT `+taskColumns()+` FROM tasks WHERE id = ? AND workspace_id = ?`,
		id,
		workspaceValue.ID,
	))
	if err != nil {
		writeError(w, http.StatusNotFound, "task not found")
		return task{}, false
	}
	return value, true
}

func (s *Server) getTask(w http.ResponseWriter, r *http.Request) {
	value, ok := s.loadTask(w, r, chi.URLParam(r, "id"))
	if ok {
		writeJSON(w, http.StatusOK, value.response())
	}
}

func (s *Server) updateTask(w http.ResponseWriter, r *http.Request) {
	value, ok := s.loadTask(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	previousStatus := value.Status
	var req struct {
		ProjectID   *string  `json:"project_id"`
		IssueID     *string  `json:"issue_id"`
		Title       *string  `json:"title"`
		Description *string  `json:"description"`
		Status      *string  `json:"status"`
		Priority    *string  `json:"priority"`
		AssigneeID  *string  `json:"assignee_id"`
		Position    *float64 `json:"position"`
		StartDate   *string  `json:"start_date"`
		DueDate     *string  `json:"due_date"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Title != nil {
		value.Title = strings.TrimSpace(*req.Title)
		if value.Title == "" {
			writeError(w, http.StatusBadRequest, "title is required")
			return
		}
	}
	if req.Description != nil {
		value.Description = *req.Description
	}
	if req.Status != nil {
		if !validTaskStatus(*req.Status) {
			writeError(w, http.StatusBadRequest, "invalid task status")
			return
		}
		value.Status = *req.Status
	}
	if req.Priority != nil {
		if !validTaskPriority(*req.Priority) {
			writeError(w, http.StatusBadRequest, "invalid task priority")
			return
		}
		value.Priority = *req.Priority
	}
	if req.Position != nil {
		value.Position = *req.Position
	}
	if req.ProjectID != nil {
		if *req.ProjectID != "" && !s.belongsToWorkspace(r.Context(), "projects", *req.ProjectID, value.WorkspaceID) {
			writeError(w, http.StatusBadRequest, "project does not belong to workspace")
			return
		}
		value.ProjectID = sql.NullString{String: *req.ProjectID, Valid: *req.ProjectID != ""}
	}
	if req.IssueID != nil {
		if *req.IssueID != "" && !s.belongsToWorkspace(r.Context(), "issues", *req.IssueID, value.WorkspaceID) {
			writeError(w, http.StatusBadRequest, "issue does not belong to workspace")
			return
		}
		value.IssueID = sql.NullString{String: *req.IssueID, Valid: *req.IssueID != ""}
	}
	if req.AssigneeID != nil {
		if *req.AssigneeID != "" && !s.workspaceHasMember(r.Context(), value.WorkspaceID, *req.AssigneeID) {
			writeError(w, http.StatusBadRequest, "assignee must be a workspace member")
			return
		}
		value.AssigneeID = sql.NullString{String: *req.AssigneeID, Valid: *req.AssigneeID != ""}
	}
	if req.StartDate != nil {
		value.StartDate = sql.NullString{String: *req.StartDate, Valid: *req.StartDate != ""}
	}
	if req.DueDate != nil {
		value.DueDate = sql.NullString{String: *req.DueDate, Valid: *req.DueDate != ""}
	}

	value.UpdatedAt = now()
	if value.Status == "done" || value.Status == "cancelled" {
		value.CompletedAt = sql.NullString{String: value.UpdatedAt, Valid: true}
	} else {
		value.CompletedAt = sql.NullString{}
	}
	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update task")
		return
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(r.Context(), `UPDATE tasks SET
		project_id = ?, issue_id = ?, title = ?, description = ?, status = ?, priority = ?,
		assignee_id = ?, position = ?, start_date = ?, due_date = ?, completed_at = ?, updated_at = ?
		WHERE id = ?`,
		value.ProjectID,
		value.IssueID,
		value.Title,
		value.Description,
		value.Status,
		value.Priority,
		value.AssigneeID,
		value.Position,
		value.StartDate,
		value.DueDate,
		value.CompletedAt,
		value.UpdatedAt,
		value.ID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update task")
		return
	}
	if previousStatus != value.Status && (value.Status == "done" || value.Status == "cancelled") {
		if err := enqueueKnowledgeEvidence(r.Context(), tx, taskEvidence(value, currentUserID(r), "task.completed")); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to record task evidence")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update task")
		return
	}
	s.dispatchKnowledgeEvidence(r.Context())
	writeJSON(w, http.StatusOK, value.response())
}

func (s *Server) deleteTask(w http.ResponseWriter, r *http.Request) {
	value, ok := s.loadTask(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	if _, err := s.db.ExecContext(r.Context(), `DELETE FROM tasks WHERE id = ?`, value.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete task")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type skill struct {
	ID, WorkspaceID, Name, Description, Content, Config, CreatedAt, UpdatedAt string
	CreatedBy                                                                 sql.NullString
}

func skillColumns() string {
	return `id, workspace_id, name, description, content, config, created_by, created_at, updated_at`
}

func scanSkill(scanner interface{ Scan(...any) error }) (skill, error) {
	var value skill
	err := scanner.Scan(&value.ID, &value.WorkspaceID, &value.Name, &value.Description, &value.Content,
		&value.Config, &value.CreatedBy, &value.CreatedAt, &value.UpdatedAt)
	return value, err
}

func (value skill) response(includeContent bool) map[string]any {
	result := map[string]any{
		"id": value.ID, "workspace_id": value.WorkspaceID, "name": value.Name,
		"description": value.Description, "config": mapJSON(value.Config, map[string]any{}),
		"created_by": nullable(value.CreatedBy.String), "created_at": value.CreatedAt, "updated_at": value.UpdatedAt,
	}
	if includeContent {
		result["content"] = value.Content
	}
	return result
}

type skillRequest struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Content     string         `json:"content"`
	Config      map[string]any `json:"config"`
	Files       []struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	} `json:"files"`
}

func (s *Server) listSkills(w http.ResponseWriter, r *http.Request) {
	workspaceValue, ok := s.resolveWorkspace(w, r)
	if !ok {
		return
	}
	rows, err := s.db.QueryContext(r.Context(), `SELECT `+skillColumns()+` FROM skills WHERE workspace_id = ? ORDER BY name`, workspaceValue.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list skills")
		return
	}
	defer rows.Close()
	result := make([]map[string]any, 0)
	for rows.Next() {
		value, err := scanSkill(rows)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list skills")
			return
		}
		result = append(result, value.response(false))
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) createSkill(w http.ResponseWriter, r *http.Request) {
	workspaceValue, ok := s.resolveWorkspace(w, r)
	if !ok {
		return
	}
	var req skillRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create skill")
		return
	}
	defer tx.Rollback()
	id, timestamp := newID(), now()
	_, err = tx.ExecContext(r.Context(), `INSERT INTO skills(
		id, workspace_id, name, description, content, config, created_by, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, id, workspaceValue.ID, req.Name, req.Description,
		req.Content, encodeJSON(req.Config, "{}"), currentUserID(r), timestamp, timestamp)
	if err != nil {
		writeError(w, http.StatusConflict, "a skill with this name already exists")
		return
	}
	for _, file := range req.Files {
		if strings.TrimSpace(file.Path) == "" {
			writeError(w, http.StatusBadRequest, "skill file path is required")
			return
		}
		_, err = tx.ExecContext(r.Context(), `INSERT INTO skill_files(
			id, skill_id, path, content, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?)`, newID(), id, file.Path, file.Content, timestamp, timestamp)
		if err != nil {
			writeError(w, http.StatusConflict, "duplicate skill file path")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create skill")
		return
	}
	s.writeSkill(w, r, id, http.StatusCreated)
}

func (s *Server) writeSkill(w http.ResponseWriter, r *http.Request, id string, status int) {
	workspaceValue, ok := s.resolveWorkspace(w, r)
	if !ok {
		return
	}
	value, err := scanSkill(s.db.QueryRowContext(r.Context(), `SELECT `+skillColumns()+` FROM skills WHERE id = ? AND workspace_id = ?`, id, workspaceValue.ID))
	if err != nil {
		writeError(w, http.StatusNotFound, "skill not found")
		return
	}
	result := value.response(true)
	rows, err := s.db.QueryContext(r.Context(), `SELECT id, skill_id, path, content, created_at, updated_at
		FROM skill_files WHERE skill_id = ? ORDER BY path`, value.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list skill files")
		return
	}
	defer rows.Close()
	files := make([]map[string]any, 0)
	for rows.Next() {
		var fileID, skillID, path, content, createdAt, updatedAt string
		if err := rows.Scan(&fileID, &skillID, &path, &content, &createdAt, &updatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list skill files")
			return
		}
		files = append(files, map[string]any{"id": fileID, "skill_id": skillID, "path": path, "content": content, "created_at": createdAt, "updated_at": updatedAt})
	}
	result["files"] = files
	writeJSON(w, status, result)
}

func (s *Server) getSkill(w http.ResponseWriter, r *http.Request) {
	s.writeSkill(w, r, chi.URLParam(r, "id"), http.StatusOK)
}

func (s *Server) updateSkill(w http.ResponseWriter, r *http.Request) {
	workspaceValue, ok := s.resolveWorkspace(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	current, err := scanSkill(s.db.QueryRowContext(r.Context(), `SELECT `+skillColumns()+` FROM skills WHERE id = ? AND workspace_id = ?`, id, workspaceValue.ID))
	if err != nil {
		writeError(w, http.StatusNotFound, "skill not found")
		return
	}
	if !s.requireSkillManager(w, r, workspaceValue.ID, current.CreatedBy) {
		return
	}
	var req skillRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Name) != "" {
		current.Name = strings.TrimSpace(req.Name)
	}
	current.Description, current.Content, current.UpdatedAt = req.Description, req.Content, now()
	if req.Config != nil {
		current.Config = encodeJSON(req.Config, "{}")
	}
	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update skill")
		return
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(r.Context(), `UPDATE skills SET name = ?, description = ?, content = ?, config = ?, updated_at = ? WHERE id = ?`,
		current.Name, current.Description, current.Content, current.Config, current.UpdatedAt, current.ID)
	if err != nil {
		writeError(w, http.StatusConflict, "failed to update skill")
		return
	}
	if req.Files != nil {
		if _, err := tx.ExecContext(r.Context(), `DELETE FROM skill_files WHERE skill_id = ?`, current.ID); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to update skill files")
			return
		}
		for _, file := range req.Files {
			_, err = tx.ExecContext(r.Context(), `INSERT INTO skill_files(id, skill_id, path, content, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?)`, newID(), current.ID, file.Path, file.Content, current.UpdatedAt, current.UpdatedAt)
			if err != nil {
				writeError(w, http.StatusConflict, "failed to update skill files")
				return
			}
		}
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update skill")
		return
	}
	s.writeSkill(w, r, current.ID, http.StatusOK)
}

func (s *Server) deleteSkill(w http.ResponseWriter, r *http.Request) {
	workspaceValue, ok := s.resolveWorkspace(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	current, err := scanSkill(s.db.QueryRowContext(r.Context(), `SELECT `+skillColumns()+` FROM skills WHERE id = ? AND workspace_id = ?`, id, workspaceValue.ID))
	if err != nil {
		writeError(w, http.StatusNotFound, "skill not found")
		return
	}
	if !s.requireSkillManager(w, r, workspaceValue.ID, current.CreatedBy) {
		return
	}
	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete skill")
		return
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(r.Context(), `DELETE FROM skill_files WHERE skill_id = ?`, id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete skill")
		return
	}
	result, err := tx.ExecContext(r.Context(), `DELETE FROM skills WHERE id = ? AND workspace_id = ?`, id, workspaceValue.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete skill")
		return
	}
	affected, err := result.RowsAffected()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete skill")
		return
	}
	if affected == 0 {
		writeError(w, http.StatusNotFound, "skill not found")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete skill")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) unsupported(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusNotImplemented, map[string]any{
		"error": "endpoint is not available in SQLite local mode",
		"code":  "sqlite_local_unsupported",
	})
}

func (s *Server) belongsToWorkspace(ctx context.Context, table, id, workspaceID string) bool {
	if table != "projects" && table != "issues" {
		return false
	}
	var found int
	query := `SELECT 1 FROM ` + table + ` WHERE id = ? AND workspace_id = ?`
	return s.db.QueryRowContext(ctx, query, id, workspaceID).Scan(&found) == nil
}
