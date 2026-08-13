package bootstrap

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/auth"
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
