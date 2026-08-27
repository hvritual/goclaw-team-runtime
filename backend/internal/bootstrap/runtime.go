package bootstrap

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v3"
	kratosgrpc "github.com/go-kratos/kratos/v3/transport/grpc"
	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/auth"
	"github.com/hvritual/workspace/internal/modules/workspace"
	workspacecontract "github.com/hvritual/workspace/internal/modules/workspace/contract"
	canonicalrealtime "github.com/hvritual/workspace/internal/realtime"
)

const canonicalHTTPRequestTimeout = 30 * time.Second

// Config defines the standalone backend process identity and listen addresses.
type Config struct {
	Name                      string
	Version                   string
	HTTPAddress               string
	GRPCAddress               string
	SQLitePath                string
	AttachmentRoot            string
	WorkspaceDependencies     workspace.WorkspaceServiceDependencies
	LocalAuth                 auth.LocalAuthConfig
	IssueMetadataEnabled      *bool
	IssueCreateEnabled        *bool
	IssueAttachmentsEnabled   *bool
	RoadmapCapabilityProvider workspacecontract.RoadmapCapabilityProvider
}

// Validate rejects incomplete process identity and malformed TCP addresses.
func (c Config) Validate() error {
	if strings.TrimSpace(c.Name) == "" {
		return fmt.Errorf("service name is required")
	}
	if strings.TrimSpace(c.Version) == "" {
		return fmt.Errorf("service version is required")
	}
	if err := validateTCPAddress("HTTP", c.HTTPAddress); err != nil {
		return err
	}
	if err := validateTCPAddress("gRPC", c.GRPCAddress); err != nil {
		return err
	}
	if strings.TrimSpace(c.SQLitePath) == "" {
		return fmt.Errorf("SQLite path is required")
	}
	if strings.TrimSpace(c.LocalAuth.VerificationCode) == "" {
		return fmt.Errorf("development verification code is required")
	}
	return nil
}

func validateTCPAddress(label, address string) error {
	trimmed := strings.TrimSpace(address)
	if trimmed == "" {
		return fmt.Errorf("%s address is required", label)
	}
	_, port, err := net.SplitHostPort(trimmed)
	if err != nil {
		return fmt.Errorf("parse %s address %q: %w", label, address, err)
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 0 || portNumber > 65535 {
		return fmt.Errorf("parse %s address %q: port must be between 0 and 65535", label, address)
	}
	return nil
}

// Runtime owns the Kratos application and its two transports.
type Runtime struct {
	application             *Application
	app                     *kratos.App
	httpServer              *kratoshttp.Server
	grpcServer              *kratosgrpc.Server
	db                      *sql.DB
	realtime                *canonicalrealtime.Hub
	governance              *workspace.GovernanceOutbox
	ephemeralAttachmentRoot string
	closeOnce               sync.Once
	closeErr                error
}

// NewRuntime creates the shared HTTP and gRPC servers and registers all modules.
func NewRuntime(config Config, logger *slog.Logger) (*Runtime, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if logger == nil {
		logger = slog.Default()
	}
	config.RoadmapCapabilityProvider = installedRuntimeCapabilities{next: config.RoadmapCapabilityProvider}

	ephemeralAttachmentRoot := ""
	if strings.TrimSpace(config.AttachmentRoot) == "" && strings.TrimSpace(config.SQLitePath) == ":memory:" {
		root, err := os.MkdirTemp("", "goclaw-canonical-space-")
		if err != nil {
			return nil, fmt.Errorf("create ephemeral Space attachment root: %w", err)
		}
		config.AttachmentRoot, ephemeralAttachmentRoot = root, root
	}
	db, application, realtimeHub, governance, err := newSQLiteApplication(context.Background(), config)
	if err != nil {
		if ephemeralAttachmentRoot != "" {
			_ = os.RemoveAll(ephemeralAttachmentRoot)
		}
		return nil, err
	}
	httpServer := kratoshttp.NewServer(
		kratoshttp.Address(config.HTTPAddress),
		kratoshttp.Timeout(canonicalHTTPRequestTimeout),
	)
	grpcServer := kratosgrpc.NewServer(kratosgrpc.Address(config.GRPCAddress))
	application.RegisterHTTP(httpServer)
	realtimeHub.RegisterHTTP(httpServer)
	application.RegisterGRPC(grpcServer)
	registerHealthRoutes(httpServer, db, governance)
	registerConfigRoute(httpServer, config.Version, capabilityEnabled(config.IssueMetadataEnabled), capabilityEnabled(config.IssueCreateEnabled), capabilityEnabled(config.IssueAttachmentsEnabled), config.RoadmapCapabilityProvider)

	app := kratos.New(
		kratos.Name(config.Name),
		kratos.Version(config.Version),
		kratos.Logger(logger),
		kratos.StopTimeout(5*time.Second),
		kratos.Server(httpServer, grpcServer),
	)
	governance.Start()
	return &Runtime{
		application:             application,
		app:                     app,
		httpServer:              httpServer,
		grpcServer:              grpcServer,
		db:                      db,
		realtime:                realtimeHub,
		governance:              governance,
		ephemeralAttachmentRoot: ephemeralAttachmentRoot,
	}, nil
}

type installedRuntimeCapabilities struct {
	next workspacecontract.RoadmapCapabilityProvider
}

func (p installedRuntimeCapabilities) RoadmapCapabilityInstalled(permission string) bool {
	switch permission {
	case workspacecontract.PermissionTaskRead,
		workspacecontract.PermissionTaskCreate,
		workspacecontract.PermissionTaskUpdateOwn,
		workspacecontract.PermissionTaskManageWorkspace,
		workspacecontract.PermissionSearchReadable,
		workspacecontract.PermissionPinReorder,
		workspacecontract.PermissionSkillReadPublished,
		workspacecontract.PermissionSkillCreate,
		workspacecontract.PermissionSkillImport,
		workspacecontract.PermissionSkillVersion,
		workspacecontract.PermissionSkillArchive,
		workspacecontract.PermissionKnowledgeQuery,
		workspacecontract.PermissionKnowledgePropose,
		workspacecontract.PermissionKnowledgeReview,
		workspacecontract.PermissionKnowledgeSelfReviewOverride,
		workspacecontract.PermissionResourceRead,
		workspacecontract.PermissionResourceManage,
		workspacecontract.PermissionRequirementEditDraft,
		workspacecontract.PermissionRequirementApproveFreeze,
		workspacecontract.PermissionRetrospectiveDraft,
		workspacecontract.PermissionRetrospectivePublish,
		workspacecontract.PermissionSimilarityCheck:
		return true
	default:
		return p.next != nil && p.next.RoadmapCapabilityInstalled(permission)
	}
}

func (p installedRuntimeCapabilities) RoadmapFeatureInstalled(feature string) bool {
	switch feature {
	case "tasks", "issue_search", "project_search", "pin_reorder", "skill_administration", "skill_import", "knowledge_query", "knowledge_review", "project_resources", "project_requirements", "project_retrospectives", "issue_similarity":
		return true
	default:
		if next, ok := p.next.(workspacecontract.RoadmapFeatureProvider); ok {
			return next.RoadmapFeatureInstalled(feature)
		}
		return false
	}
}

func registerConfigRoute(server *kratoshttp.Server, version string, issueMetadataEnabled, issueCreateEnabled, issueAttachmentsEnabled bool, roadmapProvider workspacecontract.RoadmapCapabilityProvider) {
	server.Route("/").GET("/api/config", func(ctx kratoshttp.Context) error {
		featureFlags := map[string]bool{
			"issue_list": true, "issue_base_detail": true,
			"issue_detail_pull_requests": false,
			"issue_timeline":             true, "issue_members": true,
			"issue_reactions": true, "issue_subscribers": true,
			"issue_attachments": issueAttachmentsEnabled, "issue_labels": true,
			"issue_properties": true, "issue_pins": true,
			"issue_children": true, "issue_project": true,
			"issue_child_progress": true, "issue_batch": true, "issue_acceptance": true,
			"issue_metadata": issueMetadataEnabled, "issue_realtime": true,
			"issue_create":    issueCreateEnabled,
			"project_control": false,
		}
		for name, enabled := range roadmapFeatureFlags(roadmapProvider) {
			featureFlags[name] = enabled
		}
		return ctx.JSON(http.StatusOK, map[string]any{
			"cdn_domain": "", "allow_signup": true, "server_version": version,
			"feature_flags": featureFlags,
		})
	})
}

func roadmapFeatureFlags(provider workspacecontract.RoadmapCapabilityProvider) map[string]bool {
	permissions := map[string]string{
		"tasks":                  workspacecontract.PermissionTaskRead,
		"issue_search":           workspacecontract.PermissionSearchReadable,
		"project_search":         workspacecontract.PermissionSearchReadable,
		"pin_reorder":            workspacecontract.PermissionPinReorder,
		"skill_administration":   workspacecontract.PermissionSkillCreate,
		"skill_import":           workspacecontract.PermissionSkillImport,
		"knowledge_query":        workspacecontract.PermissionKnowledgeQuery,
		"knowledge_review":       workspacecontract.PermissionKnowledgeReview,
		"project_resources":      workspacecontract.PermissionResourceRead,
		"project_requirements":   workspacecontract.PermissionRequirementEditDraft,
		"project_retrospectives": workspacecontract.PermissionRetrospectiveDraft,
		"issue_similarity":       workspacecontract.PermissionSimilarityCheck,
		"notifications":          workspacecontract.PermissionNotificationReadUpdateOwn,
		"overdue_reminders":      workspacecontract.PermissionReminderReplayRepair,
		"project_phases":         workspacecontract.PermissionProjectPhaseTransition,
		"project_outline":        workspacecontract.PermissionOutlineEditReorderLink,
		"project_phase_board":    workspacecontract.PermissionProjectPhaseTransition,
	}
	flags := make(map[string]bool, len(permissions))
	for name, permission := range permissions {
		if features, ok := provider.(workspacecontract.RoadmapFeatureProvider); ok {
			flags[name] = features.RoadmapFeatureInstalled(name)
		} else if name == "issue_search" || name == "project_search" {
			flags[name] = false
		} else {
			flags[name] = workspacecontract.RoadmapCapabilityInstalled(permission, provider)
		}
	}
	return flags
}

func capabilityEnabled(value *bool) bool { return value == nil || *value }

func registerHealthRoutes(server *kratoshttp.Server, db *sql.DB, governance *workspace.GovernanceOutbox) {
	router := server.Route("/")
	router.GET("/healthz", func(ctx kratoshttp.Context) error {
		return ctx.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})
	router.GET("/readyz", func(ctx kratoshttp.Context) error {
		if err := db.PingContext(ctx.Request().Context()); err != nil {
			return ctx.JSON(http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
		}
		if governance == nil || governance.Ready(ctx.Request().Context()) != nil {
			return ctx.JSON(http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
		}
		return ctx.JSON(http.StatusOK, map[string]string{"status": "ready"})
	})
}

// Run starts both transports and blocks until shutdown or an error.
func (r *Runtime) Run() error {
	defer r.Close()
	return r.app.Run()
}

// Stop requests graceful shutdown of both transports.
func (r *Runtime) Stop() error {
	stopErr := r.app.Stop()
	closeErr := r.Close()
	if stopErr != nil {
		return stopErr
	}
	return closeErr
}

// Close releases the product database. It is safe to call more than once.
func (r *Runtime) Close() error {
	r.closeOnce.Do(func() {
		if r.governance != nil {
			r.closeErr = r.governance.Close()
		}
		if r.realtime != nil {
			if err := r.realtime.Close(); r.closeErr == nil && err != nil {
				r.closeErr = err
			}
		}
		if r.db != nil {
			r.closeErr = r.db.Close()
		}
		if r.ephemeralAttachmentRoot != "" {
			if err := os.RemoveAll(r.ephemeralAttachmentRoot); r.closeErr == nil && err != nil {
				r.closeErr = err
			}
		}
	})
	return r.closeErr
}

// Database exposes the owned product database for composition and lifecycle tests.
func (r *Runtime) Database() *sql.DB { return r.db }

// Realtime exposes the Canonical in-memory publisher for composition tests.
func (r *Runtime) Realtime() *canonicalrealtime.Hub { return r.realtime }

// Application returns the assembled bounded contexts.
func (r *Runtime) Application() *Application {
	return r.application
}

// HTTPServer exposes the registered server for transport tests and embedding.
func (r *Runtime) HTTPServer() *kratoshttp.Server {
	return r.httpServer
}

// GRPCServer exposes the registered server for transport tests and embedding.
func (r *Runtime) GRPCServer() *kratosgrpc.Server {
	return r.grpcServer
}
