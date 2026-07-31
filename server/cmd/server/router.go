package main

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/multica-ai/multica/server/internal/analytics"
	"github.com/multica-ai/multica/server/internal/auth"
	"github.com/multica-ai/multica/server/internal/events"
	"github.com/multica-ai/multica/server/internal/handler"
	"github.com/multica-ai/multica/server/internal/middleware"
	"github.com/multica-ai/multica/server/internal/realtime"
	"github.com/multica-ai/multica/server/internal/service"
	"github.com/multica-ai/multica/server/internal/storage"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

var defaultOrigins = []string{
	"http://localhost:3000",
	"http://localhost:5173",
	"http://localhost:5174",
}

var corsAllowedHeaders = []string{
	"Accept",
	"Authorization",
	"Content-Type",
	"X-Workspace-ID",
	"X-Workspace-Slug",
	"X-Request-ID",
	"X-CSRF-Token",
	"X-Client-Platform",
	"X-Client-Version",
	"X-Client-OS",
}

func allowedOrigins() []string {
	raw := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv("FRONTEND_ORIGIN"))
	}
	if raw == "" {
		return defaultOrigins
	}
	var origins []string
	for _, part := range strings.Split(raw, ",") {
		if origin := strings.TrimSpace(part); origin != "" {
			origins = append(origins, origin)
		}
	}
	if len(origins) == 0 {
		return defaultOrigins
	}
	return origins
}

func normalizeServerVersion(value string) string {
	if value == "dev" {
		return ""
	}
	return value
}

func NewRouter(
	pool *pgxpool.Pool,
	hub *realtime.Hub,
	bus *events.Bus,
	analyticsClient analytics.Client,
	rdb *redis.Client,
	knowledgeRuntime *knowledgeRuntime,
) chi.Router {
	queries := db.New(pool)
	var store storage.Storage
	if s3 := storage.NewS3StorageFromEnv(); s3 != nil {
		store = s3
	} else if local := storage.NewLocalStorageFromEnv(); local != nil {
		store = local
	}
	cfSigner := auth.NewCloudFrontSignerFromEnv()
	config := handler.Config{
		AllowSignup:              os.Getenv("ALLOW_SIGNUP") != "false",
		DisableWorkspaceCreation: os.Getenv("DISABLE_WORKSPACE_CREATION") == "true",
		PublicURL:                strings.TrimRight(strings.TrimSpace(os.Getenv("MULTICA_PUBLIC_URL")), "/"),
		AttachmentDownloadMode:   os.Getenv("ATTACHMENT_DOWNLOAD_MODE"),
		AttachmentDownloadURLTTL: 30 * time.Minute,
		AttachmentFrameAncestors: allowedOrigins(),
		ServerVersion:            normalizeServerVersion(version),
	}
	h := handler.New(
		queries,
		pool,
		hub,
		bus,
		service.NewEmailService(),
		store,
		cfSigner,
		analyticsClient,
		config,
	)
	patCache := auth.NewPATCache(rdb)
	h.PATCache = patCache
	h.MembershipCache = auth.NewMembershipCache(rdb)
	if knowledgeRuntime != nil {
		h.ConfigureKnowledge(
			knowledgeRuntime.store,
			knowledgeRuntime.service,
			knowledgeRuntime.unavailable,
		)
		h.ConfigureKnowledgeHealth(knowledgeRuntime)
		h.ConfigureKnowledgeEvidence(knowledgeRuntime.enabled)
		h.ConfigureKnowledgeMCP(
			config.PublicURL,
			splitKnowledgeAuthorizationServers(os.Getenv("MULTICA_MCP_AUTHORIZATION_SERVERS")),
		)
	}

	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(middleware.ClientMetadata)
	r.Use(middleware.RequestLogger)
	r.Use(chimw.Recoverer)
	r.Use(middleware.ContentSecurityPolicy)
	origins := allowedOrigins()
	realtime.SetAllowedOrigins(origins)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   corsAllowedHeaders,
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if err := pool.Ping(r.Context()); err != nil {
			http.Error(w, `{"status":"unavailable"}`, http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := pool.Ping(r.Context()); err != nil {
			http.Error(w, `{"status":"unavailable"}`, http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	r.Get("/api/config", h.GetConfig)
	r.Get("/.well-known/oauth-protected-resource", h.KnowledgeMCPMetadata)
	r.Get("/.well-known/oauth-protected-resource/mcp/{workspaceSlug}/knowledge", h.KnowledgeMCPMetadata)
	r.Handle("/mcp/{workspaceSlug}/knowledge", h.KnowledgeMCPHandler())
	r.Post("/auth/send-code", h.SendCode)
	r.Post("/auth/verify-code", h.VerifyCode)
	r.Post("/auth/google", h.GoogleLogin)
	r.Post("/auth/logout", h.Logout)
	r.Get("/api/attachments/{id}/signed-download", h.DownloadAttachmentWithCapability)
	r.Get("/api/avatars/{sig}/*", h.ServeAvatar)
	if _, ok := store.(*storage.LocalStorage); ok {
		r.Get("/uploads/*", h.ServeLocalUpload)
	}

	membership := &membershipChecker{queries: queries}
	resolver := &patResolver{queries: queries, cache: patCache}
	slugResolver := realtime.SlugResolver(func(ctx context.Context, slug string) (string, error) {
		workspace, err := queries.GetWorkspaceBySlug(ctx, slug)
		if err != nil {
			return "", err
		}
		return util.UUIDToString(workspace.ID), nil
	})
	r.Get("/ws", func(w http.ResponseWriter, r *http.Request) {
		realtime.HandleWebSocket(hub, membership, resolver, slugResolver, w, r)
	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(queries, patCache))
		r.Use(middleware.RefreshCloudFrontCookies(cfSigner))

		r.Get("/api/me", h.GetMe)
		r.Patch("/api/me", h.UpdateMe)
		r.Post("/api/cli-token", h.IssueCliToken)
		r.Post("/api/upload-file", h.UploadFile)
		r.Get("/api/attachments/{id}/download", h.DownloadAttachment)

		r.Route("/api/workspaces", func(r chi.Router) {
			r.Get("/", h.ListWorkspaces)
			r.Post("/", h.CreateWorkspace)
			r.Route("/{id}", func(r chi.Router) {
				r.Group(func(r chi.Router) {
					r.Use(middleware.RequireWorkspaceMemberFromURL(queries, "id"))
					r.Get("/", h.GetWorkspace)
					r.Get("/members", h.ListMembersWithUser)
					r.Post("/leave", h.LeaveWorkspace)
				})
				r.Group(func(r chi.Router) {
					r.Use(middleware.RequireWorkspaceRoleFromURL(queries, "id", "owner", "admin"))
					r.Put("/", h.UpdateWorkspace)
					r.Patch("/", h.UpdateWorkspace)
					r.Get("/permissions", h.GetWorkspacePermissions)
					r.Get("/invitations", h.ListWorkspaceInvitations)
					r.Post("/members", h.CreateInvitation)
					r.Patch("/members/{memberId}", h.UpdateMember)
					r.Delete("/members/{memberId}", h.DeleteMember)
					r.Delete("/invitations/{invitationId}", h.RevokeInvitation)
				})
				r.With(middleware.RequireWorkspaceRoleFromURL(queries, "id", "owner")).
					Delete("/", h.DeleteWorkspace)
			})
		})

		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireWorkspaceMember(queries))

			r.Route("/api/knowledge", h.RegisterKnowledgeRoutes)

			r.Route("/api/issues", func(r chi.Router) {
				r.Post("/table/groups", h.ListIssueTableGroups)
				r.Post("/table/rows", h.ListIssueTableRows)
				r.Post("/table/facets", h.ListIssueTableFacets)
				r.Get("/search", h.SearchIssues)
				r.Get("/child-progress", h.ChildIssueProgress)
				r.Get("/children", h.ListChildrenByParents)
				r.Get("/grouped", h.ListGroupedIssues)
				r.Get("/", h.ListIssues)
				r.Post("/query", h.QueryIssues)
				r.Post("/", h.CreateIssue)
				r.Post("/batch-update", h.BatchUpdateIssues)
				r.Post("/batch-delete", h.BatchDeleteIssues)
				r.Route("/{id}", func(r chi.Router) {
					r.Get("/", h.GetIssue)
					r.Put("/", h.UpdateIssue)
					r.Post("/move", h.MoveIssue)
					r.Delete("/", h.DeleteIssue)
					r.Get("/timeline", h.ListTimeline)
					r.Get("/comments", h.ListComments)
					r.Post("/comments", h.CreateComment)
					r.Get("/subscribers", h.ListIssueSubscribers)
					r.Post("/subscribe", h.SubscribeToIssue)
					r.Post("/unsubscribe", h.UnsubscribeFromIssue)
					r.Post("/reactions", h.AddIssueReaction)
					r.Delete("/reactions", h.RemoveIssueReaction)
					r.Get("/attachments", h.ListAttachments)
					r.Get("/children", h.ListChildIssues)
					r.Get("/labels", h.ListLabelsForIssue)
					r.Post("/labels", h.AttachLabel)
					r.Delete("/labels/{labelId}", h.DetachLabel)
					r.Get("/metadata", h.ListIssueMetadata)
					r.Put("/metadata/{key}", h.SetIssueMetadataKey)
					r.Delete("/metadata/{key}", h.DeleteIssueMetadataKey)
					r.Put("/properties/{propertyId}", h.SetIssueProperty)
					r.Delete("/properties/{propertyId}", h.DeleteIssueProperty)
				})
			})

			r.Route("/api/comments/{commentId}", func(r chi.Router) {
				r.Put("/", h.UpdateComment)
				r.Delete("/", h.DeleteComment)
				r.Post("/knowledge-proposals", h.ProposeCommentDecision)
				r.Post("/resolve", h.ResolveComment)
				r.Delete("/resolve", h.UnresolveComment)
				r.Post("/reactions", h.AddReaction)
				r.Delete("/reactions", h.RemoveReaction)
			})

			r.Route("/api/projects", func(r chi.Router) {
				r.Get("/search", h.SearchProjects)
				r.Get("/", h.ListProjects)
				r.Post("/", h.CreateProject)
				r.Route("/{id}", func(r chi.Router) {
					r.Get("/", h.GetProject)
					r.Put("/", h.UpdateProject)
					r.Delete("/", h.DeleteProject)
				})
			})

			r.Route("/api/tasks", func(r chi.Router) {
				r.Get("/", h.ListTasks)
				r.Post("/", h.CreateTask)
				r.Route("/{id}", func(r chi.Router) {
					r.Get("/", h.GetTask)
					r.Put("/", h.UpdateTask)
					r.Patch("/", h.UpdateTask)
					r.Delete("/", h.DeleteTask)
				})
			})

			r.Route("/api/skills", func(r chi.Router) {
				r.Get("/", h.ListSkills)
				r.Post("/", h.CreateSkill)
				r.Get("/search", h.SearchSkills)
				r.Post("/import", h.ImportSkill)
				r.Route("/{id}", func(r chi.Router) {
					r.Get("/", h.GetSkill)
					r.Put("/", h.UpdateSkill)
					r.Delete("/", h.DeleteSkill)
					r.Get("/labels", h.ListLabelsForSkill)
					r.Post("/labels", h.AttachLabelToSkill)
					r.Delete("/labels/{labelId}", h.DetachLabelFromSkill)
					r.Get("/files", h.ListSkillFiles)
					r.Put("/files", h.UpsertSkillFile)
					r.Delete("/files/{fileId}", h.DeleteSkillFile)
				})
			})

			r.Route("/api/labels", func(r chi.Router) {
				r.Get("/", h.ListLabels)
				r.Post("/", h.CreateLabel)
				r.Get("/{id}", h.GetLabel)
				r.Put("/{id}", h.UpdateLabel)
				r.Delete("/{id}", h.DeleteLabel)
			})

			r.Route("/api/properties", func(r chi.Router) {
				r.Get("/", h.ListProperties)
				r.Post("/", h.CreateProperty)
				r.Get("/{id}", h.GetProperty)
				r.Patch("/{id}", h.UpdateProperty)
			})

			r.Get("/api/attachments/{id}", h.GetAttachmentByID)
			r.Delete("/api/attachments/{id}", h.DeleteAttachment)
		})
	})
	return r
}

func splitKnowledgeAuthorizationServers(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			result = append(result, part)
		}
	}
	return result
}

type membershipChecker struct {
	queries *db.Queries
}

func (checker *membershipChecker) IsMember(ctx context.Context, userID, workspaceID string) bool {
	_, err := checker.queries.GetMemberByUserAndWorkspace(ctx, db.GetMemberByUserAndWorkspaceParams{
		UserID:      parseUUID(userID),
		WorkspaceID: parseUUID(workspaceID),
	})
	return err == nil
}

type patResolver struct {
	queries *db.Queries
	cache   *auth.PATCache
}

func (resolver *patResolver) ResolveToken(ctx context.Context, token string) (string, bool) {
	hash := auth.HashToken(token)
	if resolver.cache != nil {
		if userID, ok := resolver.cache.Get(ctx, hash); ok {
			return userID, true
		}
	}
	pat, err := resolver.queries.GetPersonalAccessTokenByHash(ctx, hash)
	if err != nil {
		return "", false
	}
	userID := util.UUIDToString(pat.UserID)
	if resolver.cache != nil {
		var expiresAt time.Time
		if pat.ExpiresAt.Valid {
			expiresAt = pat.ExpiresAt.Time
		}
		resolver.cache.Set(ctx, hash, userID, auth.TTLForExpiry(time.Now(), expiresAt))
	}
	go resolver.queries.UpdatePersonalAccessTokenLastUsed(context.Background(), pat.ID)
	return userID, true
}

func parseUUID(value string) pgtype.UUID {
	return util.MustParseUUID(value)
}
