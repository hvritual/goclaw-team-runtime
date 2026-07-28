package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

var globalConfig *Config

// Load 加载配置文件
func Load(configPath string) (*Config, error) {
	// 创建 viper 实例
	v := viper.New()

	// 设置配置文件路径
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		// 默认配置文件路径
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}

		configDir := filepath.Join(home, ".goclaw")
		v.AddConfigPath(configDir)
		v.AddConfigPath(".")
		v.SetConfigName("config")
		v.SetConfigType("json")
	}

	// 设置环境变量前缀
	v.SetEnvPrefix("GOSKILLS")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// 设置默认值
	setDefaults(v)

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
		// 配置文件不存在，使用默认值和环境变量
	}

	// 解析配置
	var cfg Config
	if err := v.Unmarshal(&cfg, viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(
		mapstructure.StringToTimeDurationHookFunc(),
		mapstructure.TextUnmarshallerHookFunc(),
	))); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	globalConfig = &cfg
	return &cfg, nil
}

// setDefaults 设置默认配置值
func setDefaults(v *viper.Viper) {
	// Agent 默认配置
	v.SetDefault("agents.defaults.model", map[string]interface{}{
		"primary": "openrouter/anthropic/claude-opus-4-5",
	})
	v.SetDefault("agents.defaults.max_iterations", 15)
	v.SetDefault("agents.defaults.temperature", 0.7)
	v.SetDefault("agents.defaults.max_tokens", 4096)
	v.SetDefault("agents.defaults.max_history_messages", 100) // 默认保留最近100条消息

	// Gateway 默认配置
	v.SetDefault("gateway.host", "localhost")
	v.SetDefault("gateway.port", 8080)
	v.SetDefault("gateway.read_timeout", 30)
	v.SetDefault("gateway.write_timeout", 30)

	// 工具默认配置
	v.SetDefault("tools.shell.enabled", true)
	v.SetDefault("tools.shell.timeout", 120)
	v.SetDefault("tools.shell.sandbox.enabled", false)
	v.SetDefault("tools.shell.sandbox.image", "goclaw/sandbox:latest")
	v.SetDefault("tools.shell.sandbox.workdir", "/workspace")
	v.SetDefault("tools.shell.sandbox.remove", true)
	v.SetDefault("tools.shell.sandbox.network", "none")
	v.SetDefault("tools.shell.sandbox.privileged", false)
	v.SetDefault("tools.web.search_engine", "travily")
	v.SetDefault("tools.web.timeout", 10)
	v.SetDefault("tools.browser.enabled", false)
	v.SetDefault("browser.headless", true)
	v.SetDefault("browser.timeout", 30)

	// OpenClaw-compatible defaults
	v.SetDefault("auth.profiles", map[string]interface{}{})
	v.SetDefault("auth.order", map[string]interface{}{})
	v.SetDefault("models.mode", "merge")
	v.SetDefault("models.providers", map[string]interface{}{})

	// The catalog is the governed memory control plane. It is enabled by
	// default, requires no embedding API, and stores runtime state outside a
	// synchronized Obsidian Vault.
	v.SetDefault("memory.backend", "builtin")
	v.SetDefault("memory.builtin.enabled", true)
	v.SetDefault("memory.catalog.enabled", true)
	v.SetDefault("memory.catalog.default_project", "default")
	v.SetDefault("memory.catalog.review_after_days", 90)
	v.SetDefault("memory.catalog.max_context_records", 6)
	v.SetDefault("memory.catalog.max_context_chars", 8000)
	v.SetDefault("memory.catalog.auto_ingest", false)

	// Human decision governance. Authentication remains opt-in for backward
	// compatibility, while the example production configuration enables it.
	v.SetDefault("governance.enabled", false)
	v.SetDefault("governance.require_authenticated_reviewers", false)
	v.SetDefault("governance.require_rationale", true)
	v.SetDefault("governance.require_counterargument", false)
	v.SetDefault("governance.min_rationale_runes", 12)
	v.SetDefault("governance.forbid_self_approval", true)
	v.SetDefault("governance.seed_approval_quorum", 1)
	v.SetDefault("governance.high_risk_approval_quorum", 2)
	v.SetDefault("governance.evolution_approval_quorum", 1)
	v.SetDefault("governance.harness_approval_quorum", 1)
	v.SetDefault("governance.min_distinct_task_reviewers", 2)
	v.SetDefault("governance.max_task_review_kinds_per_reviewer", 2)
	v.SetDefault("governance.forbid_final_approver_from_task_review", true)
	v.SetDefault("governance.forbid_harness_promoter_from_approval", true)
	v.SetDefault("governance.reviewers", map[string]interface{}{})

	// Better-Harness defaults. The root is resolved by harness.NewService when
	// left empty so runtime state stays under ~/.goclaw and outside the vault.
	v.SetDefault("harness.enabled", false)
	v.SetDefault("harness.project_id", "default")
	v.SetDefault("harness.trace_enabled", true)
	v.SetDefault("harness.active_version", "v0.1.0")

	// Go-native Ouroboros defaults. Like Harness and Development state, this
	// event store is resolved under ~/.goclaw and must stay outside the vault.
	v.SetDefault("ouroboros.enabled", false)
	v.SetDefault("ouroboros.ambiguity_threshold", 0.20)
	v.SetDefault("ouroboros.convergence_threshold", 0.95)
	v.SetDefault("ouroboros.required_ready_streak", 2)
	v.SetDefault("ouroboros.max_generations", 30)
	v.SetDefault("ouroboros.consensus_reviewers", 3)
	v.SetDefault("ouroboros.max_questions_per_round", 5)
	v.SetDefault("ouroboros.max_context_bytes", 131072)
	v.SetDefault("ouroboros.max_output_tokens", 12000)
	v.SetDefault("ouroboros.assessment_reviewers", 2)
	v.SetDefault("ouroboros.assessment_max_spread", 0.15)
	v.SetDefault("ouroboros.assessment_gray_zone", 0.03)
	v.SetDefault("ouroboros.critical_finding_veto", true)
	v.SetDefault("ouroboros.consensus_max_spread", 0.25)
	v.SetDefault("ouroboros.evaluation_history_window", 5)
	v.SetDefault("ouroboros.required_passing_evaluations", 2)
	v.SetDefault("ouroboros.max_session_model_calls", 120)
	v.SetDefault("ouroboros.max_session_model_tokens", 2000000)

	// Orchestrator Lite defaults. Execution is disabled until explicitly
	// enabled and always uses an isolated Git worktree.
	v.SetDefault("development.enabled", false)
	v.SetDefault("development.codex_command", "codex")
	v.SetDefault("development.run_timeout_seconds", 21600)
	v.SetDefault("development.verify_timeout_seconds", 1800)
	v.SetDefault("development.max_repair_attempts", 2)
	v.SetDefault("development.default_max_changed_files", 40)
	v.SetDefault("development.default_max_changed_lines", 2000)
	v.SetDefault("development.allow_dirty_repo", false)
	v.SetDefault("development.independent_review", true)
	v.SetDefault("development.gateway_allow_execution", false)
	v.SetDefault("development.unsafe_host_verification", false)
	v.SetDefault("development.require_human_final_approval", true)

	// Team control and workstation scheduling remain opt-in for backward
	// compatibility. Production team mode requires both the shared Gateway
	// boundary and an authenticated per-user principal.
	v.SetDefault("team_control.enabled", false)
	v.SetDefault("workstation.enabled", false)
	v.SetDefault("workstation.lease_duration_seconds", 120)
	v.SetDefault("workstation.runner_offline_seconds", 300)
	v.SetDefault("workstation.default_max_attempts", 3)
	v.SetDefault("workstation.max_idempotency_receipts", 128)
}

// Save 保存配置到文件
func Save(cfg *Config, path string) error {
	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// 转换为 JSON（带缩进）
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Get 获取全局配置
func Get() *Config {
	return globalConfig
}

// GetDefaultConfigPath 获取默认配置文件路径
func GetDefaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".goclaw", "config.json"), nil
}

// GetWorkspacePath 获取 workspace 目录路径
func GetWorkspacePath(cfg *Config) (string, error) {
	if cfg.Workspace.Path != "" {
		// 使用配置中的自定义路径
		return cfg.Workspace.Path, nil
	}
	// 使用默认路径
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".goclaw", "workspace"), nil
}

// Validate 验证配置 (使用新的验证器)
func Validate(cfg *Config) error {
	validator := NewValidator(true)
	return validator.Validate(cfg)
}

// HasProvider 检查配置中是否有指定的提供商
func HasProvider(cfg *Config, provider string) bool {
	if cfg == nil {
		return false
	}
	_, exists := cfg.Models.Providers[provider]
	return exists
}

// GetGatewayWebSocketURL 获取 Gateway WebSocket URL
func GetGatewayWebSocketURL(cfg *Config) string {
	if cfg == nil {
		return "ws://localhost:28789/ws"
	}

	port := cfg.Gateway.WebSocket.Port
	if port == 0 {
		port = 28789
	}

	host := cfg.Gateway.WebSocket.Host
	if host == "" {
		host = "localhost"
	}

	path := cfg.Gateway.WebSocket.Path
	if path == "" {
		path = "/ws"
	}

	return fmt.Sprintf("ws://%s:%d%s", host, port, path)
}

// GetGatewayHTTPPort 获取 Gateway HTTP 端口
func GetGatewayHTTPPort(cfg *Config) int {
	if cfg == nil {
		return 28789
	}

	port := cfg.Gateway.WebSocket.Port
	if port == 0 {
		port = 28789
	}
	return port
}
