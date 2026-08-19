package bootstrap

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/auth"
	workspacecontract "github.com/hvritual/workspace/internal/modules/workspace/contract"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid",
			config: Config{
				Name: "backend", Version: "dev",
				HTTPAddress: "127.0.0.1:8000", GRPCAddress: "127.0.0.1:9000",
				SQLitePath: "test.db", WorkspaceDependencies: FailClosedWorkspaceDependencies(), LocalAuth: testLocalAuthConfig(),
			},
		},
		{
			name: "ephemeral test ports",
			config: Config{
				Name: "backend", Version: "test",
				HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
				SQLitePath: "test.db", WorkspaceDependencies: FailClosedWorkspaceDependencies(), LocalAuth: testLocalAuthConfig(),
			},
		},
		{
			name: "missing name",
			config: Config{
				Version: "dev", HTTPAddress: "127.0.0.1:8000", GRPCAddress: "127.0.0.1:9000",
			},
			wantErr: true,
		},
		{
			name: "invalid HTTP port",
			config: Config{
				Name: "backend", Version: "dev",
				HTTPAddress: "127.0.0.1:invalid", GRPCAddress: "127.0.0.1:9000",
			},
			wantErr: true,
		},
		{
			name: "invalid gRPC address",
			config: Config{
				Name: "backend", Version: "dev",
				HTTPAddress: "127.0.0.1:8000", GRPCAddress: "missing-port",
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.config.Validate()
			if (err != nil) != test.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestRuntimeRegistersHealthAndModuleRoutes(t *testing.T) {
	runtime := newTestRuntime(t)
	if got := len(runtime.Application().Modules()); got != 4 {
		t.Fatalf("module count = %d, want 4", got)
	}

	tests := []struct {
		path       string
		wantStatus int
		wantBody   map[string]string
	}{
		{path: "/healthz", wantStatus: http.StatusOK, wantBody: map[string]string{"status": "ok"}},
		{path: "/readyz", wantStatus: http.StatusOK, wantBody: map[string]string{"status": "ready"}},
		{path: "/v1/workspace/ping", wantStatus: http.StatusOK, wantBody: map[string]string{"message": "pong"}},
		{path: "/v1/auth/ping", wantStatus: http.StatusOK, wantBody: map[string]string{"message": "pong"}},
		{path: "/v1/space/ping", wantStatus: http.StatusOK, wantBody: map[string]string{"message": "pong"}},
		{path: "/v1/system/ping", wantStatus: http.StatusOK, wantBody: map[string]string{"message": "pong"}},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			response := httptest.NewRecorder()
			runtime.HTTPServer().ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, test.wantStatus, response.Body.String())
			}
			var body map[string]string
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			for key, value := range test.wantBody {
				if body[key] != value {
					t.Fatalf("body[%q] = %q, want %q", key, body[key], value)
				}
			}
		})
	}
}

func TestRuntimeUsesExplicitHTTPTimeout(t *testing.T) {
	runtime := newTestRuntime(t)
	var remaining time.Duration
	var hasDeadline bool
	runtime.HTTPServer().HandleFunc("/__test/runtime-http-deadline", func(response http.ResponseWriter, request *http.Request) {
		deadline, ok := request.Context().Deadline()
		hasDeadline = ok
		if ok {
			remaining = time.Until(deadline)
		}
		response.WriteHeader(http.StatusNoContent)
	})

	response := httptest.NewRecorder()
	runtime.HTTPServer().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/__test/runtime-http-deadline", nil))
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
	if !hasDeadline || remaining < 28*time.Second || remaining > 30*time.Second {
		t.Fatalf("HTTP request deadline remaining = %v, want explicit 30s budget", remaining)
	}
}

func TestRuntimeReportsOnlyInstalledRoadmapCapabilities(t *testing.T) {
	runtime := newTestRuntime(t)
	response := httptest.NewRecorder()
	runtime.HTTPServer().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/config", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}
	var body struct {
		FeatureFlags map[string]bool `json:"feature_flags"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !body.FeatureFlags["project_resources"] {
		t.Error("project_resources flag = false after the installed S07A runtime")
	}
	if !body.FeatureFlags["project_requirements"] {
		t.Error("project_requirements flag = false after the installed S07B runtime")
	}
	for _, capability := range []string{
		"project_retrospectives",
		"issue_similarity",
		"notifications",
		"overdue_reminders",
		"project_phases",
		"project_outline",
		"project_phase_board",
	} {
		enabled, present := body.FeatureFlags[capability]
		if !present {
			t.Errorf("feature flag %q is absent", capability)
			continue
		}
		if enabled {
			t.Errorf("uninstalled feature flag %q = true, want false", capability)
		}
	}
	if !body.FeatureFlags["tasks"] {
		t.Error("tasks flag = false after the installed S02A runtime")
	}
	if !body.FeatureFlags["skill_import"] {
		t.Error("skill_import flag = false after the installed S05B runtime")
	}
	if !body.FeatureFlags["knowledge_query"] {
		t.Error("knowledge_query flag = false after the installed S06A runtime")
	}
	if !body.FeatureFlags["knowledge_review"] {
		t.Error("knowledge_review flag = false after the installed S06B runtime")
	}
	if !body.FeatureFlags["issue_search"] {
		t.Error("issue_search flag = false after the installed S03A runtime")
	}
	if !body.FeatureFlags["project_search"] {
		t.Error("project_search flag = false after the installed S03B runtime")
	}
	if !body.FeatureFlags["pin_reorder"] {
		t.Error("pin_reorder flag = false after the installed S04 runtime")
	}
	if !body.FeatureFlags["skill_administration"] {
		t.Error("skill_administration flag = false after the installed S05A runtime")
	}
}

func TestRoadmapFeatureFlagsRequireAnInstalledProvider(t *testing.T) {
	provider := runtimeCapabilityProviderStub{
		workspacecontract.PermissionTaskRead:       true,
		workspacecontract.PermissionSearchReadable: true,
	}
	flags := roadmapFeatureFlags(provider)
	if !flags["tasks"] {
		t.Error("tasks flag = false after its provider reported installed")
	}
	if flags["issue_search"] {
		t.Error("issue_search flag = true from shared permission without per-feature evidence")
	}
	if flags["project_search"] {
		t.Error("project_search flag = true from shared permission without per-feature evidence")
	}
	for name, enabled := range roadmapFeatureFlags(nil) {
		if enabled {
			t.Errorf("feature flag %q = true without an injected provider", name)
		}
	}
}

func TestInstalledRuntimeKeepsKnowledgeReviewOpenAgainstInjectedFeatureProvider(t *testing.T) {
	provider := featureEnabledProvider{runtimeCapabilityProviderStub: runtimeCapabilityProviderStub{
		workspacecontract.PermissionKnowledgeReview: true,
	}}
	flags := roadmapFeatureFlags(installedRuntimeCapabilities{next: provider})
	if !flags["knowledge_review"] {
		t.Fatal("knowledge_review flag = false after S06B installation")
	}
}

type featureEnabledProvider struct{ runtimeCapabilityProviderStub }

func (featureEnabledProvider) RoadmapFeatureInstalled(string) bool { return true }

func TestRuntimeReportsOnlyCapabilitiesProvenByInjectedProvider(t *testing.T) {
	runtime := newRuntimeForConfig(t, Config{
		Name: "backend-test", Version: "test",
		HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath:            filepath.Join(t.TempDir(), "capability-runtime.db"),
		WorkspaceDependencies: FailClosedWorkspaceDependencies(),
		RoadmapCapabilityProvider: runtimeCapabilityProviderStub{
			workspacecontract.PermissionTaskRead: true,
		},
	})
	response := httptest.NewRecorder()
	runtime.HTTPServer().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/config", nil))
	var body struct {
		FeatureFlags map[string]bool `json:"feature_flags"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !body.FeatureFlags["tasks"] {
		t.Error("tasks flag = false after runtime provider installation")
	}
	if !body.FeatureFlags["skill_import"] {
		t.Error("skill_import flag = false after installed S05B runtime")
	}
}

type runtimeCapabilityProviderStub map[string]bool

func (s runtimeCapabilityProviderStub) RoadmapCapabilityInstalled(permission string) bool {
	return s[permission]
}

func TestRuntimeRegistersGRPCServices(t *testing.T) {
	runtime := newTestRuntime(t)
	services := runtime.GRPCServer().GetServiceInfo()
	expected := []string{
		"grpc.health.v1.Health",
		"workspace.v1.WorkspaceService",
		"workspace.v1.ProjectService",
		"workspace.v1.TodoService",
		"workspace.v1.IssueService",
		"workspace.v1.KnowledgeService",
		"workspace.v1.RequirementService",
		"workspace.v1.SettingService",
		"workspace.v1.RelationshipService",
		"auth.v1.AuthService",
		"auth.v1.MemberService",
		"auth.v1.AgentService",
		"space.v1.SpaceService",
		"space.v1.AssetService",
		"system.v1.SystemService",
		"system.v1.AgentReleaseService",
		"system.v1.SkillService",
	}
	for _, service := range expected {
		if _, ok := services[service]; !ok {
			t.Errorf("gRPC service %q is not registered", service)
		}
	}
}

func newTestRuntime(t *testing.T) *Runtime {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	runtime, err := NewRuntime(Config{
		Name: "backend-test", Version: "test",
		HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath: filepath.Join(t.TempDir(), "runtime.db"), WorkspaceDependencies: FailClosedWorkspaceDependencies(), LocalAuth: testLocalAuthConfig(),
	}, logger)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = runtime.Close() })
	return runtime
}

func testLocalAuthConfig() auth.LocalAuthConfig {
	return auth.LocalAuthConfig{VerificationCode: "888888", SessionTTL: time.Hour}
}
