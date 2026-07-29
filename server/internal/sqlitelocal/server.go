// Package sqlitelocal provides the opt-in, local-only SQLite server.
//
// It deliberately does not share the PostgreSQL/sqlc handler graph: the
// production server remains unchanged, while this package implements the
// stable local contracts for workspace, member, project, issue, task, and
// skill data. It never starts or discovers an agent runtime.
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
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schema string

const (
	defaultVerificationCode = "888888"
	authCookieName          = "multica_auth"
	invitationLifetime      = 7 * 24 * time.Hour
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type Options struct {
	VerificationCode string
	FrontendOrigin   string
}

type Server struct {
	db               *sql.DB
	handler          http.Handler
	verificationCode string
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
	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("initialize sqlite schema: %w", err)
	}

	code := strings.TrimSpace(options.VerificationCode)
	if code == "" {
		code = defaultVerificationCode
	}
	server := &Server{db: db, verificationCode: code}
	server.handler = server.routes(options.FrontendOrigin)
	return server, nil
}

func (s *Server) Handler() http.Handler {
	return s.handler
}

func (s *Server) Close() error {
	return s.db.Close()
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

		api.Route("/api/projects", func(projects chi.Router) {
			projects.Get("/", s.listProjects)
			projects.Post("/", s.createProject)
			projects.Get("/{id}", s.getProject)
			projects.Put("/{id}", s.updateProject)
			projects.Delete("/{id}", s.deleteProject)
		})

		api.Route("/api/issues", func(issues chi.Router) {
			issues.Get("/", s.listIssues)
			issues.Post("/", s.createIssue)
			issues.Post("/query", s.listIssues)
			issues.Get("/{id}", s.getIssue)
			issues.Patch("/{id}", s.updateIssue)
			issues.Delete("/{id}", s.deleteIssue)
			issues.Get("/{id}/active-task", s.getActiveTask)
			issues.Get("/{id}/task-runs", s.listIssueTasks)
			issues.Post("/{id}/tasks/{taskID}/cancel", s.cancelTask)
		})

		api.Route("/api/tasks", func(tasks chi.Router) {
			tasks.Get("/", s.listTasks)
			tasks.Post("/", s.createTask)
			tasks.Get("/{id}", s.getTask)
			tasks.Patch("/{id}", s.updateTask)
			tasks.Delete("/{id}", s.deleteTask)
			tasks.Post("/{id}/cancel", s.cancelTask)
			tasks.Get("/{id}/messages", s.listTaskMessages)
			tasks.Post("/{id}/messages", s.createTaskMessage)
		})

		api.Route("/api/skills", func(skills chi.Router) {
			skills.Get("/", s.listSkills)
			skills.Post("/", s.createSkill)
			skills.Get("/{id}", s.getSkill)
			skills.Put("/{id}", s.updateSkill)
			skills.Delete("/{id}", s.deleteSkill)
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
	return decoded
}

func encodeJSON(value any, fallback string) string {
	if value == nil {
		return fallback
	}
	encoded, err := json.Marshal(value)
	if err != nil {
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
	if !s.requireWorkspaceRole(w, r, value.ID, "owner", "admin") {
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
	if !s.requireWorkspaceRole(w, r, value.ID, "owner") {
		return
	}
	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete workspace")
		return
	}
	defer tx.Rollback()
	statements := []string{
		`DELETE FROM task_messages WHERE task_id IN (SELECT id FROM tasks WHERE workspace_id = ?)`,
		`DELETE FROM tasks WHERE workspace_id = ?`,
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
	rows, err := s.db.QueryContext(r.Context(), `SELECT m.id, m.workspace_id, m.user_id, m.role, m.created_at,
		u.name, u.email, u.avatar_url FROM members m JOIN users u ON u.id = m.user_id
		WHERE m.workspace_id = ? ORDER BY m.created_at`, workspaceValue.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list members")
		return
	}
	defer rows.Close()
	result := make([]map[string]any, 0)
	for rows.Next() {
		var value member
		if err := rows.Scan(&value.ID, &value.WorkspaceID, &value.UserID, &value.Role, &value.CreatedAt,
			&value.Name, &value.Email, &value.AvatarURL); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list members")
			return
		}
		result = append(result, memberResponse(value))
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) createMember(w http.ResponseWriter, r *http.Request) {
	workspaceValue, ok := s.authorizedWorkspace(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	if !s.requireWorkspaceRole(w, r, workspaceValue.ID, "owner", "admin") {
		return
	}
	var req struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Role == "" {
		req.Role = "member"
	}
	if req.Role != "admin" && req.Role != "member" {
		writeError(w, http.StatusBadRequest, "role must be admin or member")
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if !strings.Contains(email, "@") {
		writeError(w, http.StatusBadRequest, "valid email is required")
		return
	}

	var memberCount int
	err := s.db.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM members m
		JOIN users u ON u.id = m.user_id
		WHERE m.workspace_id = ? AND lower(u.email) = ?`, workspaceValue.ID, email).Scan(&memberCount)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check workspace membership")
		return
	}
	if memberCount > 0 {
		writeError(w, http.StatusConflict, "user is already a member")
		return
	}

	timestamp := now()
	if _, err := s.db.ExecContext(r.Context(), `UPDATE invitations SET status = 'expired', updated_at = ?
		WHERE workspace_id = ? AND invitee_email = ? AND status = 'pending' AND expires_at <= ?`,
		timestamp, workspaceValue.ID, email, timestamp); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to expire old invitations")
		return
	}

	var pendingCount int
	if err := s.db.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM invitations
		WHERE workspace_id = ? AND invitee_email = ? AND status = 'pending'`,
		workspaceValue.ID, email).Scan(&pendingCount); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check pending invitation")
		return
	}
	if pendingCount > 0 {
		writeError(w, http.StatusConflict, "invitation already pending for this email")
		return
	}

	var inviteeUserID sql.NullString
	err = s.db.QueryRowContext(r.Context(), `SELECT id FROM users WHERE email = ?`, email).Scan(&inviteeUserID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusInternalServerError, "failed to resolve invitee")
		return
	}

	id := newID()
	expiresAt := time.Now().UTC().Add(invitationLifetime).Format(time.RFC3339Nano)
	_, err = s.db.ExecContext(r.Context(), `INSERT INTO invitations(
		id, workspace_id, inviter_id, invitee_email, invitee_user_id, role, status,
		created_at, updated_at, expires_at
	) VALUES (?, ?, ?, ?, ?, ?, 'pending', ?, ?, ?)`,
		id, workspaceValue.ID, currentUserID(r), email, inviteeUserID, req.Role,
		timestamp, timestamp, expiresAt)
	if err != nil {
		writeError(w, http.StatusConflict, "invitation already pending for this email")
		return
	}

	value, err := scanInvitation(s.db.QueryRowContext(r.Context(), `SELECT `+invitationSelect()+`
		FROM invitations i
		JOIN workspaces w ON w.id = i.workspace_id
		JOIN users u ON u.id = i.inviter_id
		WHERE i.id = ?`, id))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load invitation")
		return
	}
	writeJSON(w, http.StatusCreated, invitationResponse(value))
}

func (s *Server) listWorkspaceInvitations(w http.ResponseWriter, r *http.Request) {
	workspaceValue, ok := s.authorizedWorkspace(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	timestamp := now()
	if _, err := s.db.ExecContext(r.Context(), `UPDATE invitations SET status = 'expired', updated_at = ?
		WHERE workspace_id = ? AND status = 'pending' AND expires_at <= ?`,
		timestamp, workspaceValue.ID, timestamp); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to expire old invitations")
		return
	}
	rows, err := s.db.QueryContext(r.Context(), `SELECT `+invitationSelect()+`
		FROM invitations i
		JOIN workspaces w ON w.id = i.workspace_id
		JOIN users u ON u.id = i.inviter_id
		WHERE i.workspace_id = ? AND i.status = 'pending'
		ORDER BY i.created_at`, workspaceValue.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list invitations")
		return
	}
	defer rows.Close()

	result := make([]map[string]any, 0)
	for rows.Next() {
		value, err := scanInvitation(rows)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list invitations")
			return
		}
		result = append(result, invitationResponse(value))
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) revokeInvitation(w http.ResponseWriter, r *http.Request) {
	workspaceValue, ok := s.authorizedWorkspace(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	if !s.requireWorkspaceRole(w, r, workspaceValue.ID, "owner", "admin") {
		return
	}
	result, err := s.db.ExecContext(r.Context(), `UPDATE invitations
		SET status = 'revoked', updated_at = ?
		WHERE id = ? AND workspace_id = ? AND status = 'pending'`,
		now(), chi.URLParam(r, "invitationID"), workspaceValue.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to revoke invitation")
		return
	}
	affected, err := result.RowsAffected()
	if err != nil || affected == 0 {
		writeError(w, http.StatusNotFound, "invitation not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listMyInvitations(w http.ResponseWriter, r *http.Request) {
	current, err := scanUser(s.db.QueryRowContext(r.Context(), `SELECT `+userColumns()+` FROM users WHERE id = ?`, currentUserID(r)))
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	timestamp := now()
	if _, err := s.db.ExecContext(r.Context(), `UPDATE invitations SET status = 'expired', updated_at = ?
		WHERE status = 'pending' AND expires_at <= ?
			AND (invitee_user_id = ? OR invitee_email = ?)`,
		timestamp, timestamp, current.ID, current.Email); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to expire old invitations")
		return
	}
	rows, err := s.db.QueryContext(r.Context(), `SELECT `+invitationSelect()+`
		FROM invitations i
		JOIN workspaces w ON w.id = i.workspace_id
		JOIN users u ON u.id = i.inviter_id
		WHERE i.status = 'pending' AND (i.invitee_user_id = ? OR i.invitee_email = ?)
		ORDER BY i.created_at`, current.ID, current.Email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list invitations")
		return
	}
	defer rows.Close()

	result := make([]map[string]any, 0)
	for rows.Next() {
		value, err := scanInvitation(rows)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list invitations")
			return
		}
		result = append(result, invitationResponse(value))
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) getMyInvitation(w http.ResponseWriter, r *http.Request) {
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
	writeJSON(w, http.StatusOK, invitationResponse(value))
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
	if !s.requireWorkspaceRole(w, r, workspaceValue.ID, "owner", "admin") {
		return
	}
	var req struct {
		Role string `json:"role"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Role != "admin" && req.Role != "member" {
		writeError(w, http.StatusBadRequest, "role must be admin or member")
		return
	}
	memberID := chi.URLParam(r, "memberID")
	result, err := s.db.ExecContext(r.Context(), `UPDATE members SET role = ? WHERE id = ? AND workspace_id = ?`, req.Role, memberID, workspaceValue.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update member")
		return
	}
	affected, err := result.RowsAffected()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update member")
		return
	}
	if affected == 0 {
		writeError(w, http.StatusNotFound, "member not found")
		return
	}
	var value member
	err = s.db.QueryRowContext(r.Context(), `SELECT m.id, m.workspace_id, m.user_id, m.role, m.created_at,
		u.name, u.email, u.avatar_url FROM members m JOIN users u ON u.id = m.user_id
		WHERE m.id = ? AND m.workspace_id = ?`, memberID, workspaceValue.ID).Scan(
		&value.ID, &value.WorkspaceID, &value.UserID, &value.Role, &value.CreatedAt, &value.Name, &value.Email, &value.AvatarURL)
	if err != nil {
		writeError(w, http.StatusNotFound, "member not found")
		return
	}
	writeJSON(w, http.StatusOK, memberResponse(value))
}

func (s *Server) deleteMember(w http.ResponseWriter, r *http.Request) {
	workspaceValue, ok := s.authorizedWorkspace(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	if !s.requireWorkspaceRole(w, r, workspaceValue.ID, "owner", "admin") {
		return
	}
	_, err := s.db.ExecContext(r.Context(), `DELETE FROM members WHERE id = ? AND workspace_id = ?`,
		chi.URLParam(r, "memberID"), workspaceValue.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete member")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) leaveWorkspace(w http.ResponseWriter, r *http.Request) {
	workspaceValue, ok := s.authorizedWorkspace(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var role string
	if err := s.db.QueryRowContext(r.Context(), `SELECT role FROM members WHERE workspace_id = ? AND user_id = ?`,
		workspaceValue.ID, currentUserID(r)).Scan(&role); err != nil {
		writeError(w, http.StatusNotFound, "membership not found")
		return
	}
	if role == "owner" {
		writeError(w, http.StatusConflict, "workspace owner cannot leave")
		return
	}
	_, err := s.db.ExecContext(r.Context(), `DELETE FROM members WHERE workspace_id = ? AND user_id = ?`,
		workspaceValue.ID, currentUserID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to leave workspace")
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
	_, err := s.db.ExecContext(r.Context(), `INSERT INTO projects(
		id, workspace_id, title, description, icon, status, priority, lead_type, lead_id,
		start_date, due_date, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, id, workspaceValue.ID, req.Title,
		req.Description, req.Icon, req.Status, req.Priority, req.LeadType, req.LeadID,
		req.StartDate, req.DueDate, timestamp, timestamp)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create project")
		return
	}
	value, _ := scanProject(s.db.QueryRowContext(r.Context(), `SELECT `+projectColumns()+` FROM projects WHERE id = ?`, id))
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
	_, err := s.db.ExecContext(r.Context(), `UPDATE projects SET title = ?, description = ?, icon = ?, status = ?,
		priority = ?, lead_type = ?, lead_id = ?, start_date = ?, due_date = ?, updated_at = ? WHERE id = ?`,
		value.Title, value.Description, value.Icon, value.Status, value.Priority, value.LeadType,
		value.LeadID, value.StartDate, value.DueDate, value.UpdatedAt, value.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update project")
		return
	}
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
	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete project")
		return
	}
	defer tx.Rollback()
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
	Title         string         `json:"title"`
	Status        string         `json:"status"`
	Priority      string         `json:"priority"`
	Description   *string        `json:"description"`
	AssigneeType  *string        `json:"assignee_type"`
	AssigneeID    *string        `json:"assignee_id"`
	ParentIssueID *string        `json:"parent_issue_id"`
	ProjectID     *string        `json:"project_id"`
	StartDate     *string        `json:"start_date"`
	DueDate       *string        `json:"due_date"`
	Stage         *int64         `json:"stage"`
	Metadata      map[string]any `json:"metadata"`
	Properties    map[string]any `json:"properties"`
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
	var number int64
	if err := tx.QueryRowContext(r.Context(), `UPDATE workspaces SET next_issue_number = next_issue_number + 1,
		updated_at = ? WHERE id = ? RETURNING next_issue_number - 1`, now(), workspaceValue.ID).Scan(&number); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to allocate issue number")
		return
	}
	id, timestamp := newID(), now()
	_, err = tx.ExecContext(r.Context(), `INSERT INTO issues(
		id, workspace_id, number, title, description, status, priority, assignee_type, assignee_id,
		creator_type, creator_id, parent_issue_id, project_id, stage, start_date, due_date,
		metadata, properties, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'member', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, workspaceValue.ID, number, req.Title, req.Description, req.Status, req.Priority,
		req.AssigneeType, req.AssigneeID, currentUserID(r), req.ParentIssueID, req.ProjectID,
		req.Stage, req.StartDate, req.DueDate, encodeJSON(req.Metadata, "{}"),
		encodeJSON(req.Properties, "{}"), timestamp, timestamp)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create issue")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create issue")
		return
	}
	value, _ := scanIssue(s.db.QueryRowContext(r.Context(), `SELECT `+issueColumns()+` FROM issues WHERE id = ?`, id))
	writeJSON(w, http.StatusCreated, value.response(workspaceValue.IssuePrefix))
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
	if ok {
		writeJSON(w, http.StatusOK, value.response(workspaceValue.IssuePrefix))
	}
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
	value.UpdatedAt = now()
	_, err := s.db.ExecContext(r.Context(), `UPDATE issues SET title = ?, description = ?, status = ?, priority = ?,
		assignee_type = ?, assignee_id = ?, parent_issue_id = ?, project_id = ?, stage = ?, start_date = ?,
		due_date = ?, metadata = ?, properties = ?, updated_at = ? WHERE id = ?`, value.Title, value.Description,
		value.Status, value.Priority, value.AssigneeType, value.AssigneeID, value.ParentIssueID, value.ProjectID,
		value.Stage, value.StartDate, value.DueDate, value.Metadata, value.Properties, value.UpdatedAt, value.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update issue")
		return
	}
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
		`DELETE FROM task_messages WHERE task_id IN (SELECT id FROM tasks WHERE issue_id = ?)`,
		`DELETE FROM tasks WHERE issue_id = ?`,
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
	ID, WorkspaceID, AgentID, RuntimeID, Status, CreatedAt, UpdatedAt string
	IssueID, DispatchedAt, StartedAt, CompletedAt, Result, Error      sql.NullString
	Priority                                                          int64
}

func taskColumns() string {
	return `id, workspace_id, issue_id, agent_id, runtime_id, status, priority, dispatched_at,
		started_at, completed_at, result, error, created_at, updated_at`
}

func scanTask(scanner interface{ Scan(...any) error }) (task, error) {
	var value task
	err := scanner.Scan(&value.ID, &value.WorkspaceID, &value.IssueID, &value.AgentID, &value.RuntimeID,
		&value.Status, &value.Priority, &value.DispatchedAt, &value.StartedAt, &value.CompletedAt,
		&value.Result, &value.Error, &value.CreatedAt, &value.UpdatedAt)
	return value, err
}

func (value task) response() map[string]any {
	return map[string]any{
		"id": value.ID, "workspace_id": value.WorkspaceID, "issue_id": value.IssueID.String,
		"agent_id": value.AgentID, "runtime_id": value.RuntimeID, "status": value.Status,
		"priority": value.Priority, "dispatched_at": nullable(value.DispatchedAt.String),
		"started_at": nullable(value.StartedAt.String), "completed_at": nullable(value.CompletedAt.String),
		"result": mapJSON(value.Result.String, nil), "error": nullable(value.Error.String),
		"attempt": 1, "max_attempts": 1, "custom_env": map[string]string{},
		"created_at": value.CreatedAt,
	}
}

func (s *Server) createTask(w http.ResponseWriter, r *http.Request) {
	workspaceValue, ok := s.resolveWorkspace(w, r)
	if !ok {
		return
	}
	var req struct {
		IssueID   string `json:"issue_id"`
		AgentID   string `json:"agent_id"`
		RuntimeID string `json:"runtime_id"`
		Status    string `json:"status"`
		Priority  int64  `json:"priority"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Status == "" {
		req.Status = "queued"
	}
	if req.IssueID != "" && !s.belongsToWorkspace(r.Context(), "issues", req.IssueID, workspaceValue.ID) {
		writeError(w, http.StatusBadRequest, "issue not found in workspace")
		return
	}
	id, timestamp := newID(), now()
	_, err := s.db.ExecContext(r.Context(), `INSERT INTO tasks(
		id, workspace_id, issue_id, agent_id, runtime_id, status, priority, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, id, workspaceValue.ID, nullable(req.IssueID),
		req.AgentID, req.RuntimeID, req.Status, req.Priority, timestamp, timestamp)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create task")
		return
	}
	value, _ := scanTask(s.db.QueryRowContext(r.Context(), `SELECT `+taskColumns()+` FROM tasks WHERE id = ?`, id))
	writeJSON(w, http.StatusCreated, value.response())
}

func (s *Server) listTasks(w http.ResponseWriter, r *http.Request) {
	workspaceValue, ok := s.resolveWorkspace(w, r)
	if !ok {
		return
	}
	s.writeTaskList(w, r, `workspace_id = ?`, workspaceValue.ID)
}

func (s *Server) listIssueTasks(w http.ResponseWriter, r *http.Request) {
	issueValue, _, ok := s.loadIssue(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	tasks, ok := s.loadTaskList(w, r, `issue_id = ?`, issueValue.ID)
	if ok {
		writeJSON(w, http.StatusOK, tasks)
	}
}

func (s *Server) writeTaskList(w http.ResponseWriter, r *http.Request, predicate string, arg any) {
	tasks, ok := s.loadTaskList(w, r, predicate, arg)
	if ok {
		writeJSON(w, http.StatusOK, map[string]any{"tasks": tasks, "total": len(tasks)})
	}
}

func (s *Server) loadTaskList(w http.ResponseWriter, r *http.Request, predicate string, arg any) ([]map[string]any, bool) {
	rows, err := s.db.QueryContext(r.Context(), `SELECT `+taskColumns()+` FROM tasks WHERE `+predicate+` ORDER BY created_at DESC`, arg)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list tasks")
		return nil, false
	}
	defer rows.Close()
	tasks := make([]map[string]any, 0)
	for rows.Next() {
		value, err := scanTask(rows)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list tasks")
			return nil, false
		}
		tasks = append(tasks, value.response())
	}
	return tasks, true
}

func (s *Server) loadTask(w http.ResponseWriter, r *http.Request, id string) (task, bool) {
	workspaceValue, ok := s.resolveWorkspace(w, r)
	if !ok {
		return task{}, false
	}
	value, err := scanTask(s.db.QueryRowContext(r.Context(), `SELECT `+taskColumns()+` FROM tasks WHERE id = ? AND workspace_id = ?`, id, workspaceValue.ID))
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

func (s *Server) getActiveTask(w http.ResponseWriter, r *http.Request) {
	issueValue, _, ok := s.loadIssue(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	value, err := scanTask(s.db.QueryRowContext(r.Context(), `SELECT `+taskColumns()+` FROM tasks
		WHERE issue_id = ? AND status IN ('queued', 'dispatched', 'running') ORDER BY created_at DESC LIMIT 1`, issueValue.ID))
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusOK, map[string]any{"tasks": []any{}})
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load active task")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tasks": []any{value.response()}})
}

func (s *Server) updateTask(w http.ResponseWriter, r *http.Request) {
	value, ok := s.loadTask(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req struct {
		Status string  `json:"status"`
		Result any     `json:"result"`
		Error  *string `json:"error"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Status != "" {
		value.Status = req.Status
	}
	if req.Result != nil {
		value.Result = sql.NullString{String: encodeJSON(req.Result, "null"), Valid: true}
	}
	applyNullString(&value.Error, req.Error)
	value.UpdatedAt = now()
	if value.Status == "completed" || value.Status == "failed" || value.Status == "cancelled" {
		value.CompletedAt = sql.NullString{String: value.UpdatedAt, Valid: true}
	}
	_, err := s.db.ExecContext(r.Context(), `UPDATE tasks SET status = ?, result = ?, error = ?, completed_at = ?, updated_at = ? WHERE id = ?`,
		value.Status, value.Result, value.Error, value.CompletedAt, value.UpdatedAt, value.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update task")
		return
	}
	writeJSON(w, http.StatusOK, value.response())
}

func (s *Server) cancelTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")
	if taskID == "" {
		taskID = chi.URLParam(r, "taskID")
	}
	value, ok := s.loadTask(w, r, taskID)
	if !ok {
		return
	}
	if issueID := chi.URLParam(r, "id"); chi.URLParam(r, "taskID") != "" && value.IssueID.String != issueID {
		writeError(w, http.StatusNotFound, "task not found for issue")
		return
	}
	value.Status, value.UpdatedAt = "cancelled", now()
	value.CompletedAt = sql.NullString{String: value.UpdatedAt, Valid: true}
	_, err := s.db.ExecContext(r.Context(), `UPDATE tasks SET status = 'cancelled', completed_at = ?, updated_at = ? WHERE id = ?`,
		value.CompletedAt, value.UpdatedAt, value.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to cancel task")
		return
	}
	writeJSON(w, http.StatusOK, value.response())
}

func (s *Server) deleteTask(w http.ResponseWriter, r *http.Request) {
	value, ok := s.loadTask(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete task")
		return
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(r.Context(), `DELETE FROM task_messages WHERE task_id = ?`, value.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete task")
		return
	}
	if _, err := tx.ExecContext(r.Context(), `DELETE FROM tasks WHERE id = ?`, value.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete task")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete task")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listTaskMessages(w http.ResponseWriter, r *http.Request) {
	value, ok := s.loadTask(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	rows, err := s.db.QueryContext(r.Context(), `SELECT id, task_id, role, content, created_at
		FROM task_messages WHERE task_id = ? ORDER BY created_at`, value.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list task messages")
		return
	}
	defer rows.Close()
	result := make([]map[string]any, 0)
	seq := 0
	for rows.Next() {
		var id, taskID, role, content, createdAt string
		if err := rows.Scan(&id, &taskID, &role, &content, &createdAt); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list task messages")
			return
		}
		seq++
		result = append(result, map[string]any{
			"task_id": taskID, "issue_id": value.IssueID.String, "seq": seq,
			"type": "text", "content": content, "created_at": createdAt,
		})
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) createTaskMessage(w http.ResponseWriter, r *http.Request) {
	value, ok := s.loadTask(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Role == "" || strings.TrimSpace(req.Content) == "" {
		writeError(w, http.StatusBadRequest, "role and content are required")
		return
	}
	id, timestamp := newID(), now()
	_, err := s.db.ExecContext(r.Context(), `INSERT INTO task_messages(id, task_id, role, content, created_at)
		VALUES (?, ?, ?, ?, ?)`, id, value.ID, req.Role, req.Content, timestamp)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create task message")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": id, "task_id": value.ID, "role": req.Role, "content": req.Content, "created_at": timestamp})
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

func (s *Server) requireWorkspaceRole(w http.ResponseWriter, r *http.Request, workspaceID string, allowed ...string) bool {
	var role string
	err := s.db.QueryRowContext(r.Context(), `SELECT role FROM members WHERE workspace_id = ? AND user_id = ?`,
		workspaceID, currentUserID(r)).Scan(&role)
	if err != nil {
		writeError(w, http.StatusForbidden, "workspace access denied")
		return false
	}
	for _, candidate := range allowed {
		if role == candidate {
			return true
		}
	}
	writeError(w, http.StatusForbidden, "insufficient workspace role")
	return false
}
