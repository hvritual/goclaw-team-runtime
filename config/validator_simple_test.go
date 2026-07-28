package config

import (
	"path/filepath"
	"testing"

	"github.com/smallnest/goclaw/memory/catalog"
	"github.com/smallnest/goclaw/orchestratorlite"
	"github.com/smallnest/goclaw/teamcontrol"
	"github.com/smallnest/goclaw/workstation"
)

func TestValidatorValidConfig(t *testing.T) {
	validator := NewValidator(true)

	cfg := &Config{
		Workspace: WorkspaceConfig{
			Path: "/tmp/test-workspace",
		},
		Agents: AgentsConfig{
			Defaults: AgentDefaults{
				Model:         ModelSelection{Primary: "qianfan:test-model"},
				MaxIterations: 11,
				Temperature:   1.7,
				MaxTokens:     4096,
			},
		},
		Models: ModelsConfig{
			Mode: "merge",
			Providers: map[string]ModelProviderConfig{
				"qianfan": {
					BaseURL: "https://qianfan.baidubce.com/v2",
					APIKey:  "test-valid-api-key-12345",
					API:     ModelAPIOpenAICompletions,
					Models: []ModelDefinitionConfig{
						{
							ID:            "test-model",
							Name:          "Test Model",
							ContextWindow: 128000,
							MaxTokens:     8192,
							Input:         []string{"text", "image"},
						},
					},
				},
			},
		},
		Tools: ToolsConfig{
			Web: WebToolConfig{
				Timeout: 30,
			},
		},
		Gateway: GatewayConfig{
			Port:         8080,
			ReadTimeout:  30,
			WriteTimeout: 30,
		},
		Memory: MemoryConfig{
			Backend: "builtin",
		},
	}

	err := validator.Validate(cfg)
	if err != nil {
		t.Fatalf("Expected valid config to pass, got error: %v", err)
	}
}

func TestTeamRuntimeRequiresLayeredAuthenticationAndSafeQueueSettings(t *testing.T) {
	cfg := &Config{
		TeamControl: teamcontrol.Config{Enabled: true},
		Workstation: workstation.Config{
			Enabled:                true,
			LeaseDurationSeconds:   120,
			RunnerOfflineSeconds:   300,
			DefaultMaxAttempts:     3,
			MaxIdempotencyReceipts: 128,
		},
	}
	if err := NewValidator(true).validateTeamRuntime(cfg); err == nil {
		t.Fatal("team runtime accepted a Gateway without authentication")
	}
	cfg.Gateway.WebSocket.EnableAuth = true
	cfg.Gateway.WebSocket.AuthToken = "0123456789abcdef0123456789abcdef"
	if err := NewValidator(true).validateTeamRuntime(cfg); err != nil {
		t.Fatalf("valid team runtime settings were rejected: %v", err)
	}

	cfg.Workstation.RunnerOfflineSeconds = 60
	if err := NewValidator(true).validateTeamRuntime(cfg); err == nil {
		t.Fatal("runner offline timeout shorter than the lease was accepted")
	}
}

func TestMemoryCatalogMultipleSourcesRequireStableRoot(t *testing.T) {
	cfg := &Config{
		Memory: MemoryConfig{
			Backend: "builtin",
			Catalog: catalog.Config{
				Enabled:           true,
				DefaultProject:    "alpha",
				ReviewAfterDays:   90,
				MaxContextRecords: 6,
				MaxContextChars:   8000,
				AutoIngest:        true,
				SourcePaths:       []string{"/vault/01-goals", "/vault/02-decisions"},
			},
		},
	}
	if err := NewValidator(true).validateMemory(cfg); err == nil {
		t.Fatal("multiple auto-ingest paths without source_root were accepted")
	}
	cfg.Memory.Catalog.SourceRoot = "/vault"
	if err := NewValidator(true).validateMemory(cfg); err != nil {
		t.Fatalf("stable source root was rejected: %v", err)
	}
}

func TestDevelopmentVerificationSandboxConfiguration(t *testing.T) {
	root := t.TempDir()
	cfg := &Config{
		Development: orchestratorlite.Config{
			Enabled:                   true,
			Root:                      filepath.Join(root, "runtime"),
			WorktreeRoot:              filepath.Join(root, "worktrees"),
			RepoPath:                  filepath.Join(root, "repo"),
			CodexCommand:              "codex",
			RunTimeoutSeconds:         60,
			VerifyTimeoutSeconds:      60,
			MaxRepairAttempts:         2,
			DefaultMaxChangedFiles:    40,
			DefaultMaxChangedLines:    2000,
			VerificationSandbox:       []string{"relative-wrapper"},
			RequireHumanFinalApproval: true,
		},
	}
	validator := NewValidator(true)
	if err := validator.validateDevelopment(cfg); err == nil {
		t.Fatal("relative development verification wrapper was accepted")
	}
	cfg.Development.VerificationSandbox = []string{
		filepath.Join(root, "verify-wrapper"),
	}
	cfg.Development.UnsafeHostVerification = true
	if err := validator.validateDevelopment(cfg); err == nil {
		t.Fatal("mutually exclusive development verification modes were accepted")
	}
	cfg.Development.UnsafeHostVerification = false
	if err := validator.validateDevelopment(cfg); err != nil {
		t.Fatalf("absolute development verification wrapper was rejected: %v", err)
	}
}
