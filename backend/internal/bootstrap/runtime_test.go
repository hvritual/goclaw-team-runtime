package bootstrap

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
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
			},
		},
		{
			name: "ephemeral test ports",
			config: Config{
				Name: "backend", Version: "test",
				HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
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
	}, logger)
	if err != nil {
		t.Fatal(err)
	}
	return runtime
}
