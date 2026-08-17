package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/auth"
	authcontract "github.com/hvritual/workspace/internal/modules/auth/contract"
	workspacecontract "github.com/hvritual/workspace/internal/modules/workspace/contract"
)

func TestSQLiteRuntimeRegistersTrustedLocalAuthJourney(t *testing.T) {
	runtime := newRuntimeForConfig(t, Config{
		Name: "backend-test", Version: "test",
		HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath:            filepath.Join(t.TempDir(), "auth-runtime.db"),
		WorkspaceDependencies: FailClosedWorkspaceDependencies(),
		LocalAuth: auth.LocalAuthConfig{
			VerificationCode: "888888", SessionTTL: time.Hour,
			Now:   func() time.Time { return time.Date(2026, 8, 13, 1, 2, 3, 0, time.UTC) },
			NewID: func(context.Context) (string, error) { return "runtime-token", nil },
		},
	})
	response := httptest.NewRecorder()
	runtime.HTTPServer().ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/auth/verify-code", strings.NewReader(`{"email":"owner@example.com","code":"888888"}`)))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"token":"runtime-token"`) {
		t.Fatalf("verify = %d %s", response.Code, response.Body.String())
	}
	me := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	me.Header.Set("Authorization", "Bearer runtime-token")
	meResponse := httptest.NewRecorder()
	runtime.HTTPServer().ServeHTTP(meResponse, me)
	if meResponse.Code != http.StatusOK || !strings.Contains(meResponse.Body.String(), `"email":"owner@example.com"`) {
		t.Fatalf("me = %d %s", meResponse.Code, meResponse.Body.String())
	}
}

func TestSQLiteRuntimeMigratesSharedProductDatabaseAndRetainsData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "multica-canonical.db")
	config := Config{
		Name: "backend-test", Version: "test",
		HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath: path, WorkspaceDependencies: FailClosedWorkspaceDependencies(),
	}
	runtime := newRuntimeForConfig(t, config)
	assertSQLiteTables(t, runtime, "auth_users", "auth_members", "workspaces", "workspace_issues")
	if _, err := runtime.Database().Exec(`INSERT INTO workspaces(
		id,name,slug,issue_prefix,created_at,updated_at
	) VALUES ('workspace-1','Acme','acme','ACM','2026-08-13T00:00:00Z','2026-08-13T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if err := runtime.Close(); err != nil {
		t.Fatal(err)
	}

	restarted := newRuntimeForConfig(t, config)
	t.Cleanup(func() { _ = restarted.Close() })
	var name string
	if err := restarted.Database().QueryRow(`SELECT name FROM workspaces WHERE id='workspace-1'`).Scan(&name); err != nil || name != "Acme" {
		t.Fatalf("retained workspace = %q, %v", name, err)
	}
	var authMigrations, workspaceMigrations int
	if err := restarted.Database().QueryRow(`SELECT COUNT(*) FROM auth_schema_migrations`).Scan(&authMigrations); err != nil {
		t.Fatal(err)
	}
	if err := restarted.Database().QueryRow(`SELECT COUNT(*) FROM workspace_schema_migrations`).Scan(&workspaceMigrations); err != nil {
		t.Fatal(err)
	}
	if authMigrations == 0 || workspaceMigrations == 0 {
		t.Fatalf("migration counts = auth:%d workspace:%d", authMigrations, workspaceMigrations)
	}
}

func TestSQLiteRuntimeRequiresBoundaryProvidersAndReadinessTracksDatabase(t *testing.T) {
	config := Config{
		Name: "backend-test", Version: "test",
		HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath: filepath.Join(t.TempDir(), "missing-providers.db"),
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	if _, err := NewRuntime(config, logger); err == nil {
		t.Fatal("NewRuntime() succeeded without Workspace boundary providers")
	}

	config.WorkspaceDependencies = FailClosedWorkspaceDependencies()
	runtime := newRuntimeForConfig(t, config)
	assertProbe(t, runtime, "/healthz", http.StatusOK, "ok")
	assertProbe(t, runtime, "/readyz", http.StatusOK, "ready")
	if err := runtime.Database().Close(); err != nil {
		t.Fatal(err)
	}
	assertProbe(t, runtime, "/healthz", http.StatusOK, "ok")
	assertProbe(t, runtime, "/readyz", http.StatusServiceUnavailable, "not_ready")
}

func TestSQLiteRuntimeCloseIsIdempotent(t *testing.T) {
	runtime := newRuntimeForConfig(t, Config{
		Name: "backend-test", Version: "test",
		HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath:            filepath.Join(t.TempDir(), "close.db"),
		WorkspaceDependencies: FailClosedWorkspaceDependencies(),
	})
	if err := runtime.Close(); err != nil {
		t.Fatal(err)
	}
	if err := runtime.Close(); err != nil {
		t.Fatalf("second Close() = %v", err)
	}
}

func TestSQLiteRuntimeStartsAndStopsGovernanceOutbox(t *testing.T) {
	runtime := newRuntimeForConfig(t, Config{
		Name: "backend-test", Version: "test",
		HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath: filepath.Join(t.TempDir(), "governance-runtime.db"), WorkspaceDependencies: FailClosedWorkspaceDependencies(),
	})
	if runtime.governance == nil || !runtime.governance.Running() {
		t.Fatal("governance outbox worker is not running")
	}
	assertProbe(t, runtime, "/readyz", http.StatusOK, "ready")
	if err := runtime.Close(); err != nil {
		t.Fatal(err)
	}
	if runtime.governance.Running() {
		t.Fatal("governance outbox worker remained running after runtime close")
	}
}

func TestRealtimeOutboxSinkPreservesStableDeliveryIdentity(t *testing.T) {
	recorder := &outboxEventRecorder{}
	sink := realtimeOutboxSink{events: recorder}
	event := workspacecontract.OutboxEvent{
		State: workspacecontract.OutboxInflight, AvailableAt: time.Unix(1, 0).UTC(), WorkspaceID: "workspace-1", ID: "event-1",
		EventType: "task:created", AggregateKind: "task", AggregateID: "task-1", AggregateRevision: 7,
		Payload: json.RawMessage(`{"version":"governance-outbox-v1","data":{"id":"task-1"}}`), ActorType: "member", ActorID: "member-1", AttemptCount: 1,
		ClaimToken: "claim-1", LeaseExpiresAt: timePointerBootstrap(time.Unix(61, 0).UTC()), CreatedAt: time.Unix(1, 0).UTC(),
	}
	if err := sink.Publish(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	if recorder.eventType != "task:created" || recorder.workspaceID != "workspace-1" {
		t.Fatalf("published target = %s %s", recorder.workspaceID, recorder.eventType)
	}
	payload, ok := recorder.payload.(map[string]any)
	if !ok || payload["event_id"] != "event-1" || payload["aggregate_revision"] != int64(7) {
		t.Fatalf("published payload = %#v", recorder.payload)
	}
}

func TestRealtimeOutboxSinkRejectsLegacyAndSecretBearingPayloads(t *testing.T) {
	for _, payload := range []json.RawMessage{
		json.RawMessage(`{"id":"task-1"}`),
		json.RawMessage(`{"version":"governance-outbox-v1","data":{"id":"Bearer secret-value"}}`),
	} {
		recorder := &outboxEventRecorder{}
		event := workspacecontract.OutboxEvent{
			State: workspacecontract.OutboxInflight, AvailableAt: time.Unix(1, 0).UTC(), WorkspaceID: "workspace-1", ID: "event-1",
			EventType: "task:created", AggregateKind: "task", AggregateID: "task-1", AggregateRevision: 1,
			Payload: payload, ActorType: "member", ActorID: "member-1", AttemptCount: 1,
			ClaimToken: "claim-1", LeaseExpiresAt: timePointerBootstrap(time.Unix(61, 0).UTC()), CreatedAt: time.Unix(1, 0).UTC(),
		}
		if err := (realtimeOutboxSink{events: recorder}).Publish(context.Background(), event); !errors.Is(err, workspacecontract.ErrInvalidGovernanceMutation) {
			t.Fatalf("payload %s error = %v", payload, err)
		}
		if recorder.eventType != "" {
			t.Fatalf("payload %s was published", payload)
		}
	}
}

func TestSQLiteRuntimeStopClosesDatabase(t *testing.T) {
	runtime := newRuntimeForConfig(t, Config{
		Name: "backend-test", Version: "test",
		HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath:            filepath.Join(t.TempDir(), "stop.db"),
		WorkspaceDependencies: FailClosedWorkspaceDependencies(),
	})
	if err := runtime.Stop(); err != nil {
		t.Fatal(err)
	}
	if err := runtime.Database().Ping(); err == nil {
		t.Fatal("database remained open after Stop")
	}
}

func TestSQLiteRuntimeMemoryDatabaseUsesOneConnection(t *testing.T) {
	runtime := newRuntimeForConfig(t, Config{
		Name: "backend-test", Version: "test",
		HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath: ":memory:", WorkspaceDependencies: FailClosedWorkspaceDependencies(),
	})
	if stats := runtime.Database().Stats(); stats.MaxOpenConnections != 1 {
		t.Fatalf("memory database max connections = %d, want 1", stats.MaxOpenConnections)
	}
	assertSQLiteTables(t, runtime, "auth_users", "workspaces")
}

func TestSQLiteAuthorizationExplicitlyDeniesUninstalledRoadmapCapabilities(t *testing.T) {
	reader := roadmapMembershipReader{membership: authcontract.WorkspaceMembership{
		MemberID: "member-1", UserID: "user-1", WorkspaceID: "workspace-1", Role: "owner",
	}}
	authorizer := authMembershipAdapter{reader: reader}
	tests := []struct {
		name       string
		ctx        context.Context
		permission string
	}{
		{
			name:       "owner member",
			ctx:        workspacecontract.WithWorkspaceActor(context.Background(), "member", "user-1"),
			permission: workspacecontract.PermissionTaskRead,
		},
		{
			name:       "agent",
			ctx:        workspacecontract.WithWorkspaceActor(context.Background(), "agent", "agent-1"),
			permission: workspacecontract.PermissionSearchReadable,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := authorizer.AuthorizeWorkspace(test.ctx, "workspace-1", test.permission)
			if !errors.Is(err, workspacecontract.ErrWorkspacePermissionDenied) {
				t.Fatalf("AuthorizeWorkspace() error = %v, want %v", err, workspacecontract.ErrWorkspacePermissionDenied)
			}
		})
	}
}

func TestSQLiteAuthorizationAppliesRoadmapRoleDefaultsOnlyAfterProviderInstallation(t *testing.T) {
	reader := roadmapMembershipReader{membership: authcontract.WorkspaceMembership{
		MemberID: "member-1", UserID: "user-1", WorkspaceID: "workspace-1", Role: "member",
	}}
	provider := runtimeCapabilityProviderStub{
		workspacecontract.PermissionTaskRead:    true,
		workspacecontract.PermissionSkillImport: true,
	}
	authorizer := authMembershipAdapter{reader: reader, roadmapProvider: provider}
	tests := []struct {
		name       string
		ctx        context.Context
		workspace  string
		permission string
		wantErr    error
	}{
		{
			name:       "member task read",
			ctx:        workspacecontract.WithWorkspaceActor(context.Background(), "member", "user-1"),
			workspace:  "workspace-1",
			permission: workspacecontract.PermissionTaskRead,
		},
		{
			name:       "member skill import denied",
			ctx:        workspacecontract.WithWorkspaceActor(context.Background(), "member", "user-1"),
			workspace:  "workspace-1",
			permission: workspacecontract.PermissionSkillImport,
			wantErr:    workspacecontract.ErrWorkspacePermissionDenied,
		},
		{
			name:       "client supplied agent type denied",
			ctx:        workspacecontract.WithWorkspaceActor(context.Background(), "agent", "user-1"),
			workspace:  "workspace-1",
			permission: workspacecontract.PermissionTaskRead,
			wantErr:    workspacecontract.ErrWorkspacePermissionDenied,
		},
		{
			name:       "client supplied foreign workspace denied",
			ctx:        workspacecontract.WithWorkspaceActor(context.Background(), "member", "user-1"),
			workspace:  "workspace-foreign",
			permission: workspacecontract.PermissionTaskRead,
			wantErr:    workspacecontract.ErrActorOutsideWorkspace,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := authorizer.AuthorizeWorkspace(test.ctx, test.workspace, test.permission)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("AuthorizeWorkspace() error = %v, want %v", err, test.wantErr)
			}
		})
	}
}

type roadmapMembershipReader struct {
	membership authcontract.WorkspaceMembership
}

func (r roadmapMembershipReader) ListForUser(context.Context, string) ([]authcontract.WorkspaceMembership, error) {
	return []authcontract.WorkspaceMembership{r.membership}, nil
}

func (r roadmapMembershipReader) FindForUserAndWorkspace(_ context.Context, userID, workspaceID string) (authcontract.WorkspaceMembership, bool, error) {
	return r.membership, r.membership.UserID == userID && r.membership.WorkspaceID == workspaceID, nil
}

func (r roadmapMembershipReader) FindByMemberAndWorkspace(_ context.Context, memberID, workspaceID string) (authcontract.WorkspaceMembership, bool, error) {
	return r.membership, r.membership.MemberID == memberID && r.membership.WorkspaceID == workspaceID, nil
}

func newRuntimeForConfig(t *testing.T, config Config) *Runtime {
	t.Helper()
	if config.LocalAuth.VerificationCode == "" {
		config.LocalAuth = testLocalAuthConfig()
	}
	runtime, err := NewRuntime(config, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = runtime.Close() })
	return runtime
}

func assertSQLiteTables(t *testing.T, runtime *Runtime, names ...string) {
	t.Helper()
	for _, name := range names {
		var found string
		if err := runtime.Database().QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, name).Scan(&found); err != nil {
			t.Fatalf("table %s: %v", name, err)
		}
	}
}

func assertProbe(t *testing.T, runtime *Runtime, path string, status int, state string) {
	t.Helper()
	response := httptest.NewRecorder()
	runtime.HTTPServer().ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
	if response.Code != status {
		t.Fatalf("%s status = %d, want %d: %s", path, response.Code, status, response.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body["status"] != state {
		t.Fatalf("%s body = %#v, %v", path, body, err)
	}
}

type outboxEventRecorder struct {
	workspaceID string
	eventType   string
	payload     any
}

func (r *outboxEventRecorder) Publish(workspaceID, eventType string, payload any, _, _ string) {
	r.workspaceID, r.eventType, r.payload = workspaceID, eventType, payload
}

func timePointerBootstrap(value time.Time) *time.Time { return &value }
