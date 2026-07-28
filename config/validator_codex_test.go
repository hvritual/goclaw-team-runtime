package config

import "testing"

func TestValidatorAcceptsCodexRuntimeWithoutBaseURL(t *testing.T) {
	cfg := &Config{
		Workspace: WorkspaceConfig{Path: "/tmp/workspace"},
		Agents: AgentsConfig{Defaults: AgentDefaults{
			Model:         ModelSelection{Primary: "codex/default"},
			MaxIterations: 20,
			Temperature:   0.2,
			MaxTokens:     16000,
		}},
		Models: ModelsConfig{
			Mode: "merge",
			Providers: map[string]ModelProviderConfig{
				"codex": {
					API:     ModelAPICodexAppServer,
					Auth:    ModelProviderAuthModeOAuth,
					Runtime: &ModelRuntimeConfig{Command: "codex"},
					Models: []ModelDefinitionConfig{{
						ID:        "default",
						Name:      "ChatGPT Workspace Codex Default",
						MaxTokens: 16000,
					}},
				},
			},
		},
		Tools: ToolsConfig{Web: WebToolConfig{Timeout: 10}},
		Gateway: GatewayConfig{
			Port:         8080,
			ReadTimeout:  30,
			WriteTimeout: 30,
		},
		Memory: MemoryConfig{Backend: "builtin"},
	}
	if err := NewValidator(true).Validate(cfg); err != nil {
		t.Fatalf("Codex runtime config should be valid: %v", err)
	}
}
