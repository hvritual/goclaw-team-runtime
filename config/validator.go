package config

import (
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/smallnest/goclaw/errors"
	"github.com/smallnest/goclaw/governance"
)

// Validator provides configuration validation
type Validator struct {
	strictMode bool
}

// NewValidator creates a new configuration validator
func NewValidator(strict bool) *Validator {
	return &Validator{
		strictMode: strict,
	}
}

// Validate performs comprehensive configuration validation
func (v *Validator) Validate(cfg *Config) error {
	if cfg == nil {
		return errors.InvalidConfig("configuration cannot be nil")
	}

	// Validate in order of dependency
	validators := []func(*Config) error{
		v.validateWorkspace,
		v.validateAgents,
		v.validateProviders,
		v.validateChannels,
		v.validateTools,
		v.validateGovernance,
		v.validateOuroboros,
		v.validateDevelopment,
		v.validateTeamRuntime,
		v.validateGateway,
		v.validateMemory,
	}

	for _, validator := range validators {
		if err := validator(cfg); err != nil {
			return err
		}
	}

	return nil
}

func (v *Validator) validateTeamRuntime(cfg *Config) error {
	if !cfg.TeamControl.Enabled && !cfg.Workstation.Enabled {
		return nil
	}
	if cfg.Workstation.Enabled && !cfg.TeamControl.Enabled {
		return errors.InvalidConfig("workstation requires team_control.enabled")
	}
	if cfg.TeamControl.Enabled {
		if !cfg.Gateway.WebSocket.EnableAuth ||
			len(strings.TrimSpace(cfg.Gateway.WebSocket.AuthToken)) < 24 {
			return errors.InvalidConfig(
				"team_control requires gateway.websocket authentication with an auth_token of at least 24 characters",
			)
		}
	}
	if cfg.Workstation.Enabled {
		if cfg.Workstation.LeaseDurationSeconds < 30 ||
			cfg.Workstation.LeaseDurationSeconds > 3600 {
			return errors.InvalidConfig(
				"workstation.lease_duration_seconds must be between 30 and 3600",
			)
		}
		if cfg.Workstation.RunnerOfflineSeconds < cfg.Workstation.LeaseDurationSeconds ||
			cfg.Workstation.RunnerOfflineSeconds > 86400 {
			return errors.InvalidConfig(
				"workstation.runner_offline_seconds must be at least the lease duration and no more than 86400",
			)
		}
		if cfg.Workstation.DefaultMaxAttempts < 1 ||
			cfg.Workstation.DefaultMaxAttempts > 20 {
			return errors.InvalidConfig(
				"workstation.default_max_attempts must be between 1 and 20",
			)
		}
		if cfg.Workstation.MaxIdempotencyReceipts < 16 ||
			cfg.Workstation.MaxIdempotencyReceipts > 4096 {
			return errors.InvalidConfig(
				"workstation.max_idempotency_receipts must be between 16 and 4096",
			)
		}
	}
	for label, root := range map[string]string{
		"team_control.root":         cfg.TeamControl.Root,
		"workstation.root":          cfg.Workstation.Root,
		"harness.root":              cfg.Harness.Root,
		"ouroboros.root":            cfg.Ouroboros.Root,
		"development.root":          cfg.Development.Root,
		"development.worktree_root": cfg.Development.WorktreeRoot,
	} {
		knowledgeRoot := strings.TrimSpace(cfg.Harness.KnowledgeRoot)
		if knowledgeRoot == "" {
			knowledgeRoot = strings.TrimSpace(cfg.Harness.VaultPath)
		}
		if root == "" || knowledgeRoot == "" {
			continue
		}
		runtimeRoot, err := filepath.Abs(root)
		if err != nil {
			return errors.InvalidConfig(label + " cannot be resolved")
		}
		vaultRoot, err := filepath.Abs(knowledgeRoot)
		if err != nil {
			return errors.InvalidConfig("harness.knowledge_root cannot be resolved")
		}
		if runtimeRoot == vaultRoot ||
			strings.HasPrefix(runtimeRoot, vaultRoot+string(filepath.Separator)) {
			return errors.InvalidConfig(label + " must remain outside the governed knowledge root")
		}
	}
	if cfg.Harness.Enabled &&
		cfg.Harness.KnowledgeBackend != "" &&
		cfg.Harness.KnowledgeBackend != "filesystem" &&
		cfg.Harness.KnowledgeBackend != "git" {
		return errors.InvalidConfig("harness.knowledge_backend must be filesystem or git")
	}
	return nil
}

func (v *Validator) validateGovernance(cfg *Config) error {
	policy := cfg.Governance
	if !policy.Enabled {
		return nil
	}
	if policy.MinRationaleRunes < 1 || policy.MinRationaleRunes > 2000 {
		return errors.InvalidConfig("governance.min_rationale_runes must be between 1 and 2000")
	}
	for label, value := range map[string]int{
		"governance.seed_approval_quorum":               policy.SeedApprovalQuorum,
		"governance.high_risk_approval_quorum":          policy.HighRiskApprovalQuorum,
		"governance.evolution_approval_quorum":          policy.EvolutionApprovalQuorum,
		"governance.harness_approval_quorum":            policy.HarnessApprovalQuorum,
		"governance.min_distinct_task_reviewers":        policy.MinDistinctTaskReviewers,
		"governance.max_task_review_kinds_per_reviewer": policy.MaxTaskReviewKindsPerReviewer,
	} {
		if value < 1 || value > 10 {
			return errors.InvalidConfig(label + " must be between 1 and 10")
		}
	}
	if policy.MinDistinctTaskReviewers > 4 {
		return errors.InvalidConfig("governance.min_distinct_task_reviewers cannot exceed four review kinds")
	}
	if policy.MaxTaskReviewKindsPerReviewer > 4 {
		return errors.InvalidConfig("governance.max_task_review_kinds_per_reviewer cannot exceed four")
	}
	if policy.RequireAuthenticatedReviewers && len(policy.Reviewers) == 0 {
		return errors.InvalidConfig("governance.reviewers is required when authenticated reviewers are enabled")
	}
	seenDigests := make(map[string]string)
	seenTeamUsers := make(map[string]string)
	for id, reviewer := range policy.Reviewers {
		if strings.TrimSpace(id) == "" {
			return errors.InvalidConfig("governance reviewer id cannot be empty")
		}
		digestText := strings.ToLower(strings.TrimSpace(reviewer.TokenSHA256))
		digest, err := hex.DecodeString(digestText)
		if err != nil || len(digest) != 32 {
			return errors.InvalidConfig(fmt.Sprintf("governance reviewer %q token_sha256 must be a 64-character SHA-256", id))
		}
		if digestText == strings.Repeat("0", 64) {
			return errors.InvalidConfig(fmt.Sprintf(
				"governance reviewer %q still uses the all-zero placeholder token digest",
				id,
			))
		}
		if priorID, exists := seenDigests[digestText]; exists {
			return errors.InvalidConfig(fmt.Sprintf(
				"governance reviewers %q and %q cannot share a token digest",
				priorID,
				id,
			))
		}
		seenDigests[digestText] = id
		if teamUserID := strings.TrimSpace(reviewer.TeamUserID); teamUserID != "" {
			normalizedTeamUserID := strings.ToLower(teamUserID)
			if priorID, exists := seenTeamUsers[normalizedTeamUserID]; exists {
				return errors.InvalidConfig(fmt.Sprintf(
					"governance reviewers %q and %q cannot bind the same team_user_id %q",
					priorID,
					id,
					teamUserID,
				))
			}
			seenTeamUsers[normalizedTeamUserID] = id
		}
		if len(reviewer.Roles) == 0 {
			return errors.InvalidConfig(fmt.Sprintf("governance reviewer %q requires at least one role", id))
		}
		for _, role := range reviewer.Roles {
			if !governance.IsKnownRole(strings.TrimSpace(role)) {
				return errors.InvalidConfig(fmt.Sprintf(
					"governance reviewer %q has unknown role %q",
					id,
					role,
				))
			}
		}
	}
	if policy.RequireAuthenticatedReviewers {
		if err := validateGovernanceCapacity(cfg); err != nil {
			return err
		}
	}
	return nil
}

func validateGovernanceCapacity(cfg *Config) error {
	policy := cfg.Governance
	count := func(role string) int {
		total := 0
		for _, reviewer := range policy.Reviewers {
			if reviewerHasRole(reviewer, role) {
				total++
			}
		}
		return total
	}
	require := func(role string, minimum int) error {
		if count(role) < minimum {
			return errors.InvalidConfig(fmt.Sprintf(
				"governance role %q requires at least %d configured reviewer(s)",
				role,
				minimum,
			))
		}
		return nil
	}
	if cfg.Ouroboros.Enabled {
		seedQuorum := policy.SeedApprovalQuorum
		if policy.HighRiskApprovalQuorum > seedQuorum {
			seedQuorum = policy.HighRiskApprovalQuorum
		}
		evolutionQuorum := policy.EvolutionApprovalQuorum
		if policy.HighRiskApprovalQuorum > evolutionQuorum {
			evolutionQuorum = policy.HighRiskApprovalQuorum
		}
		for role, minimum := range map[string]int{
			governance.RoleSeedApprove:       seedQuorum,
			governance.RoleEvolutionApprove:  evolutionQuorum,
			governance.RoleReadinessOverride: 1,
			governance.RoleConflictResolve:   1,
			governance.RoleEvaluationResolve: 1,
			governance.RoleOutcomeRecord:     1,
			governance.RoleKillSwitch:        1,
			governance.RoleSessionCancel:     1,
		} {
			if err := require(role, minimum); err != nil {
				return err
			}
		}
	}
	if cfg.Development.Enabled {
		reviewRoles := []string{
			governance.RoleScenarioReview,
			governance.RoleCapacityReview,
			governance.RoleRiskReview,
			governance.RoleCostReview,
		}
		reviewCapable := make(map[string]struct{})
		acceptCapable := make(map[string]struct{})
		for _, role := range reviewRoles {
			if err := require(role, 1); err != nil {
				return err
			}
		}
		if err := require(governance.RoleTaskAccept, 1); err != nil {
			return err
		}
		if err := require(governance.RoleTaskCancel, 1); err != nil {
			return err
		}
		for id, reviewer := range policy.Reviewers {
			for _, role := range reviewRoles {
				if reviewerHasRole(reviewer, role) {
					reviewCapable[id] = struct{}{}
				}
			}
			if reviewerHasRole(reviewer, governance.RoleTaskAccept) {
				acceptCapable[id] = struct{}{}
			}
		}
		if len(reviewCapable) < policy.MinDistinctTaskReviewers {
			return errors.InvalidConfig(
				"governance reviewers cannot satisfy min_distinct_task_reviewers",
			)
		}
		if policy.ForbidFinalApproverFromTaskReview {
			possible := false
			for acceptID := range acceptCapable {
				reviewersOtherThanAcceptor := 0
				for reviewID := range reviewCapable {
					if reviewID != acceptID {
						reviewersOtherThanAcceptor++
					}
				}
				if reviewersOtherThanAcceptor >= policy.MinDistinctTaskReviewers {
					possible = true
					break
				}
			}
			if !possible {
				return errors.InvalidConfig(
					"governance reviewers cannot separate task review from final acceptance",
				)
			}
		}
	}
	if cfg.Harness.Enabled {
		if err := require(governance.RoleHarnessApprove, policy.HarnessApprovalQuorum); err != nil {
			return err
		}
		if err := require(governance.RoleHarnessPromote, 1); err != nil {
			return err
		}
		if err := require(governance.RoleHarnessRollback, 1); err != nil {
			return err
		}
		if err := require(governance.RoleKnowledgeApprove, 1); err != nil {
			return err
		}
		if policy.ForbidHarnessPromoterFromApproval {
			possible := false
			for promoterID, promoter := range policy.Reviewers {
				if !reviewerHasRole(promoter, governance.RoleHarnessPromote) {
					continue
				}
				approvers := 0
				for approverID, approver := range policy.Reviewers {
					if approverID != promoterID &&
						reviewerHasRole(approver, governance.RoleHarnessApprove) {
						approvers++
					}
				}
				if approvers >= policy.HarnessApprovalQuorum {
					possible = true
					break
				}
			}
			if !possible {
				return errors.InvalidConfig(
					"governance reviewers cannot separate Harness approval from promotion",
				)
			}
		}
	}
	return nil
}

func reviewerHasRole(reviewer governance.ReviewerConfig, expected string) bool {
	for _, role := range reviewer.Roles {
		if strings.TrimSpace(role) == governance.RoleAny ||
			strings.TrimSpace(role) == expected {
			return true
		}
	}
	return false
}

func (v *Validator) validateOuroboros(cfg *Config) error {
	ouroborosConfig := cfg.Ouroboros
	if !ouroborosConfig.Enabled {
		return nil
	}
	if ouroborosConfig.Root != "" && !filepath.IsAbs(ouroborosConfig.Root) {
		return errors.InvalidConfig("ouroboros.root must be absolute when set")
	}
	if ouroborosConfig.AmbiguityThreshold <= 0 || ouroborosConfig.AmbiguityThreshold >= 1 {
		return errors.InvalidConfig("ouroboros.ambiguity_threshold must be between 0 and 1")
	}
	if ouroborosConfig.ConvergenceThreshold <= 0 || ouroborosConfig.ConvergenceThreshold > 1 {
		return errors.InvalidConfig("ouroboros.convergence_threshold must be greater than 0 and at most 1")
	}
	if ouroborosConfig.RequiredReadyStreak < 1 || ouroborosConfig.RequiredReadyStreak > 10 {
		return errors.InvalidConfig("ouroboros.required_ready_streak must be between 1 and 10")
	}
	if ouroborosConfig.MaxGenerations < 1 || ouroborosConfig.MaxGenerations > 100 {
		return errors.InvalidConfig("ouroboros.max_generations must be between 1 and 100")
	}
	if ouroborosConfig.ConsensusReviewers < 1 || ouroborosConfig.ConsensusReviewers > 9 {
		return errors.InvalidConfig("ouroboros.consensus_reviewers must be between 1 and 9")
	}
	if ouroborosConfig.MaxQuestionsPerRound < 1 || ouroborosConfig.MaxQuestionsPerRound > 20 {
		return errors.InvalidConfig("ouroboros.max_questions_per_round must be between 1 and 20")
	}
	if ouroborosConfig.MaxContextBytes < 4096 || ouroborosConfig.MaxContextBytes > 4*1024*1024 {
		return errors.InvalidConfig("ouroboros.max_context_bytes must be between 4096 and 4194304")
	}
	if ouroborosConfig.MaxOutputTokens < 256 || ouroborosConfig.MaxOutputTokens > 128000 {
		return errors.InvalidConfig("ouroboros.max_output_tokens must be between 256 and 128000")
	}
	if ouroborosConfig.AssessmentReviewers < 1 || ouroborosConfig.AssessmentReviewers > 5 {
		return errors.InvalidConfig("ouroboros.assessment_reviewers must be between 1 and 5")
	}
	if ouroborosConfig.AssessmentMaxSpread <= 0 || ouroborosConfig.AssessmentMaxSpread > 1 {
		return errors.InvalidConfig("ouroboros.assessment_max_spread must be greater than 0 and at most 1")
	}
	if ouroborosConfig.AssessmentGrayZone < 0 || ouroborosConfig.AssessmentGrayZone > 0.25 {
		return errors.InvalidConfig("ouroboros.assessment_gray_zone must be between 0 and 0.25")
	}
	if ouroborosConfig.ConsensusMaxSpread <= 0 || ouroborosConfig.ConsensusMaxSpread > 1 {
		return errors.InvalidConfig("ouroboros.consensus_max_spread must be greater than 0 and at most 1")
	}
	if ouroborosConfig.EvaluationHistoryWindow < 1 || ouroborosConfig.EvaluationHistoryWindow > 100 {
		return errors.InvalidConfig("ouroboros.evaluation_history_window must be between 1 and 100")
	}
	if ouroborosConfig.RequiredPassingEvaluations < 1 ||
		ouroborosConfig.RequiredPassingEvaluations > ouroborosConfig.EvaluationHistoryWindow {
		return errors.InvalidConfig("ouroboros.required_passing_evaluations must be within the evaluation history window")
	}
	if ouroborosConfig.MaxSessionModelCalls < 1 || ouroborosConfig.MaxSessionModelCalls > 10000 {
		return errors.InvalidConfig("ouroboros.max_session_model_calls must be between 1 and 10000")
	}
	if ouroborosConfig.MaxSessionModelTokens < 1000 || ouroborosConfig.MaxSessionModelTokens > 1_000_000_000 {
		return errors.InvalidConfig("ouroboros.max_session_model_tokens must be between 1000 and 1000000000")
	}
	return nil
}

func (v *Validator) validateDevelopment(cfg *Config) error {
	development := cfg.Development
	if !development.Enabled {
		return nil
	}
	if strings.TrimSpace(development.RepoPath) == "" {
		return errors.InvalidConfig("development.repo_path is required when development is enabled")
	}
	if !filepath.IsAbs(development.RepoPath) {
		return errors.InvalidConfig("development.repo_path must be absolute")
	}
	for label, value := range map[string]string{
		"development.root":          development.Root,
		"development.worktree_root": development.WorktreeRoot,
	} {
		if value != "" && !filepath.IsAbs(value) {
			return errors.InvalidConfig(label + " must be absolute when set")
		}
	}
	if strings.TrimSpace(development.CodexCommand) == "" {
		return errors.InvalidConfig("development.codex_command is required")
	}
	if development.RunTimeoutSeconds < 1 || development.RunTimeoutSeconds > 24*60*60 {
		return errors.InvalidConfig("development.run_timeout_seconds must be between 1 and 86400")
	}
	if development.VerifyTimeoutSeconds < 1 || development.VerifyTimeoutSeconds > 6*60*60 {
		return errors.InvalidConfig("development.verify_timeout_seconds must be between 1 and 21600")
	}
	for _, value := range development.VerificationSandbox {
		if strings.TrimSpace(value) == "" {
			return errors.InvalidConfig(
				"development.verification_sandbox entries must not be empty",
			)
		}
	}
	if len(development.VerificationSandbox) > 0 {
		if !filepath.IsAbs(strings.TrimSpace(
			development.VerificationSandbox[0],
		)) {
			return errors.InvalidConfig(
				"development.verification_sandbox executable must be absolute",
			)
		}
		if development.UnsafeHostVerification {
			return errors.InvalidConfig(
				"development.verification_sandbox and unsafe_host_verification are mutually exclusive",
			)
		}
	}
	if development.MaxRepairAttempts < 1 || development.MaxRepairAttempts > 10 {
		return errors.InvalidConfig("development.max_repair_attempts must be between 1 and 10")
	}
	if development.DefaultMaxChangedFiles < 1 || development.DefaultMaxChangedFiles > 10000 {
		return errors.InvalidConfig("development.default_max_changed_files must be between 1 and 10000")
	}
	if development.DefaultMaxChangedLines < 1 || development.DefaultMaxChangedLines > 1_000_000 {
		return errors.InvalidConfig("development.default_max_changed_lines must be between 1 and 1000000")
	}
	return nil
}

// validateWorkspace validates workspace configuration
func (v *Validator) validateWorkspace(cfg *Config) error {
	// Check workspace path
	if cfg.Workspace.Path != "" {
		// Check if path is absolute
		if !filepath.IsAbs(cfg.Workspace.Path) {
			return errors.InvalidConfig("workspace path must be absolute")
		}

		// Check if directory exists or can be created
		if err := os.MkdirAll(cfg.Workspace.Path, 0755); err != nil {
			return errors.Wrap(err, errors.ErrCodeInvalidConfig,
				"cannot create workspace directory")
		}
	}

	return nil
}

// validateAgents validates agent configuration
func (v *Validator) validateAgents(cfg *Config) error {
	// Check default configuration
	if err := v.validateAgentDefaults(&cfg.Agents.Defaults); err != nil {
		return errors.Wrap(err, errors.ErrCodeInvalidConfig, "invalid agent defaults")
	}

	// Check individual agents
	agentIDs := make(map[string]bool)
	for i, agent := range cfg.Agents.List {
		if agent.ID == "" {
			return errors.InvalidConfig(fmt.Sprintf("agent at index %d has empty ID", i))
		}

		// Check for duplicate IDs
		if agentIDs[agent.ID] {
			return errors.InvalidConfig(fmt.Sprintf("duplicate agent ID: %s", agent.ID))
		}
		agentIDs[agent.ID] = true

		// Validate agent configuration
		if err := v.validateAgentConfig(&agent); err != nil {
			return errors.Wrap(err, errors.ErrCodeInvalidConfig,
				fmt.Sprintf("invalid agent '%s'", agent.ID))
		}
	}

	// Check bindings
	for _, binding := range cfg.Bindings {
		if !agentIDs[binding.AgentID] {
			return errors.InvalidConfig(fmt.Sprintf("binding references non-existent agent: %s",
				binding.AgentID))
		}
	}

	return nil
}

// validateAgentDefaults validates default agent configuration
func (v *Validator) validateAgentDefaults(defaults *AgentDefaults) error {
	// Check model
	if strings.TrimSpace(defaults.Model.Effective()) == "" {
		return errors.InvalidConfig("default agent model cannot be empty")
	}

	// Check max iterations
	if defaults.MaxIterations < 1 || defaults.MaxIterations > 100 {
		return errors.InvalidConfig("max_iterations must be between 1 and 100")
	}

	// Check temperature
	if defaults.Temperature < 0 || defaults.Temperature > 2 {
		return errors.InvalidConfig("temperature must be between 0 and 2")
	}

	// Check max tokens
	if defaults.MaxTokens < 1 || defaults.MaxTokens > 128000 {
		return errors.InvalidConfig("max_tokens must be between 1 and 128000")
	}

	// Validate subagents configuration
	// Note: Subagents is of type *SubagentsConfig, not *AgentSubagentConfig
	// Skip validation for now as the structure differs
	_ = defaults.Subagents

	return nil
}

// validateAgentConfig validates individual agent configuration
func (v *Validator) validateAgentConfig(agent *AgentConfig) error {
	// Check model
	if strings.TrimSpace(agent.Model) == "" {
		return errors.InvalidConfig("agent model cannot be empty")
	}

	// Validate subagents configuration
	if agent.Subagents != nil {
		if err := v.validateSubagentsConfig(agent.Subagents); err != nil {
			return err
		}
	}

	return nil
}

// validateSubagentsConfig validates subagent configuration
func (v *Validator) validateSubagentsConfig(subagents *AgentSubagentConfig) error {
	// Check timeout
	if subagents.TimeoutSeconds < 1 || subagents.TimeoutSeconds > 3600 {
		return errors.InvalidConfig("subagent timeout must be between 1 and 3600 seconds")
	}

	// Check allowed tools and denied tools don't overlap
	for _, allowed := range subagents.AllowTools {
		if slices.Contains(subagents.DenyTools, allowed) {
			return errors.InvalidConfig(fmt.Sprintf(
				"tool '%s' is both allowed and denied", allowed))
		}
	}

	return nil
}

// validateProviders validates LLM provider configuration
func (v *Validator) validateProviders(cfg *Config) error {
	// Check if at least one provider is configured in models.providers
	if !cfg.Models.HasProviders() {
		return errors.InvalidConfig("at least one LLM provider must be configured in models.providers")
	}

	// Validate each provider in models.providers
	for providerName, provider := range cfg.Models.Providers {
		if provider.BaseURL == "" && provider.API != ModelAPICodexAppServer {
			return errors.InvalidConfig(fmt.Sprintf("provider '%s' has empty baseUrl", providerName))
		}

		// API key can be an environment variable reference, so empty is allowed
		if provider.APIKey != "" {
			if err := v.validateAPIKey(provider.APIKey); err != nil {
				return errors.Wrap(err, errors.ErrCodeInvalidConfig,
					fmt.Sprintf("invalid API key for provider '%s'", providerName))
			}
		}

		// Validate models
		for i, model := range provider.Models {
			if model.ID == "" {
				return errors.InvalidConfig(fmt.Sprintf("provider '%s' model at index %d has empty id", providerName, i))
			}
			if model.Name == "" {
				return errors.InvalidConfig(fmt.Sprintf("provider '%s' model '%s' has empty name", providerName, model.ID))
			}
		}
	}

	return nil
}

// validateAPIKey validates API key format
func (v *Validator) validateAPIKey(key string) error {
	key = strings.TrimSpace(key)

	if len(key) < 10 {
		return errors.InvalidInput("API key too short (minimum 10 characters)")
	}

	if strings.Contains(key, " ") {
		return errors.InvalidInput("API key cannot contain spaces")
	}

	return nil
}

// validateChannels validates channel configuration
func (v *Validator) validateChannels(cfg *Config) error {
	validators := []func(*ChannelsConfig) error{
		v.validateTelegram,
		v.validateWhatsApp,
		v.validateFeishu,
		v.validateQQ,
		v.validateWeWork,
		v.validateDingTalk,
		v.validateInfoflow,
	}

	for _, validator := range validators {
		if err := validator(&cfg.Channels); err != nil {
			return err
		}
	}

	return nil
}

// validateTelegram validates Telegram channel configuration
func (v *Validator) validateTelegram(channels *ChannelsConfig) error {
	if !channels.Telegram.Enabled {
		return nil
	}

	if channels.Telegram.Token == "" {
		return errors.InvalidConfig("telegram token is required when enabled")
	}

	return nil
}

// validateWhatsApp validates WhatsApp channel configuration
func (v *Validator) validateWhatsApp(channels *ChannelsConfig) error {
	if !channels.WhatsApp.Enabled {
		return nil
	}

	if channels.WhatsApp.BridgeURL == "" {
		return errors.InvalidConfig("whatsapp bridge_url is required when enabled")
	}

	if _, err := url.Parse(channels.WhatsApp.BridgeURL); err != nil {
		return errors.Wrap(err, errors.ErrCodeInvalidConfig, "invalid whatsapp bridge_url")
	}

	return nil
}

// validateFeishu validates Feishu channel configuration
func (v *Validator) validateFeishu(channels *ChannelsConfig) error {
	if !channels.Feishu.Enabled {
		return nil
	}

	if channels.Feishu.AppID == "" {
		return errors.InvalidConfig("feishu app_id is required when enabled")
	}

	if channels.Feishu.AppSecret == "" {
		return errors.InvalidConfig("feishu app_secret is required when enabled")
	}

	// verification_token is optional (for webhook mode)
	// webhook_port is optional (defaults to 8765 if not set)
	if channels.Feishu.WebhookPort != 0 {
		if channels.Feishu.WebhookPort < 1024 || channels.Feishu.WebhookPort > 65535 {
			return errors.InvalidConfig("feishu webhook_port must be between 1024 and 65535")
		}
	}

	return nil
}

// validateQQ validates QQ channel configuration
func (v *Validator) validateQQ(channels *ChannelsConfig) error {
	if !channels.QQ.Enabled {
		return nil
	}

	if channels.QQ.AppID == "" {
		return errors.InvalidConfig("qq app_id is required when enabled")
	}

	if channels.QQ.AppSecret == "" {
		return errors.InvalidConfig("qq app_secret is required when enabled")
	}

	return nil
}

// validateWeWork validates WeWork channel configuration
func (v *Validator) validateWeWork(channels *ChannelsConfig) error {
	if !channels.WeWork.Enabled {
		return nil
	}

	if channels.WeWork.CorpID == "" {
		return errors.InvalidConfig("wework corp_id is required when enabled")
	}

	if channels.WeWork.Secret == "" {
		return errors.InvalidConfig("wework secret is required when enabled")
	}

	if channels.WeWork.AgentID == "" {
		return errors.InvalidConfig("wework agent_id is required when enabled")
	}

	if channels.WeWork.WebhookPort < 1024 || channels.WeWork.WebhookPort > 65535 {
		return errors.InvalidConfig("wework webhook_port must be between 1024 and 65535")
	}

	return nil
}

// validateDingTalk validates DingTalk channel configuration
func (v *Validator) validateDingTalk(channels *ChannelsConfig) error {
	if !channels.DingTalk.Enabled {
		return nil
	}

	if channels.DingTalk.ClientID == "" {
		return errors.InvalidConfig("dingtalk client_id is required when enabled")
	}

	if channels.DingTalk.ClientSecret == "" {
		return errors.InvalidConfig("dingtalk client_secret is required when enabled")
	}

	return nil
}

// validateInfoflow validates Infoflow channel configuration
func (v *Validator) validateInfoflow(channels *ChannelsConfig) error {
	if !channels.Infoflow.Enabled {
		return nil
	}

	if channels.Infoflow.WebhookURL == "" {
		return errors.InvalidConfig("infoflow webhook_url is required when enabled")
	}

	if _, err := url.Parse(channels.Infoflow.WebhookURL); err != nil {
		return errors.Wrap(err, errors.ErrCodeInvalidConfig, "invalid infoflow webhook_url")
	}

	if channels.Infoflow.WebhookPort < 1024 || channels.Infoflow.WebhookPort > 65535 {
		return errors.InvalidConfig("infoflow webhook_port must be between 1024 and 65535")
	}

	return nil
}

// validateTools validates tool configuration
func (v *Validator) validateTools(cfg *Config) error {
	if err := v.validateShellTool(&cfg.Tools.Shell); err != nil {
		return err
	}

	if err := v.validateWebTool(&cfg.Tools.Web); err != nil {
		return err
	}

	if err := v.validateBrowserTool(&cfg.Tools.Browser); err != nil {
		return err
	}

	return nil
}

// validateShellTool validates shell tool configuration
func (v *Validator) validateShellTool(shell *ShellToolConfig) error {
	if !shell.Enabled {
		return nil
	}

	// Check timeout
	if shell.Timeout < 1 || shell.Timeout > 3600 {
		return errors.InvalidConfig("shell timeout must be between 1 and 3600 seconds")
	}

	// Check for dangerous commands
	dangerousCmds := []string{"rm -rf", "dd", "mkfs"}
	for _, dangerous := range dangerousCmds {
		found := false
		for _, denied := range shell.DeniedCmds {
			if strings.Contains(denied, dangerous) {
				found = true
				break
			}
		}
		if !found {
			return errors.InvalidConfig(fmt.Sprintf(
				"dangerous command '%s' should be in denied_cmds list", dangerous))
		}
	}

	// Validate sandbox configuration
	if shell.Sandbox.Enabled {
		if shell.Sandbox.Image == "" {
			return errors.InvalidConfig("sandbox image is required when enabled")
		}
	}

	return nil
}

// validateWebTool validates web tool configuration
func (v *Validator) validateWebTool(web *WebToolConfig) error {
	// Check timeout
	if web.Timeout < 1 || web.Timeout > 300 {
		return errors.InvalidConfig("web timeout must be between 1 and 300 seconds")
	}

	return nil
}

// validateBrowserTool validates browser tool configuration
func (v *Validator) validateBrowserTool(browser *BrowserToolConfig) error {
	if !browser.Enabled {
		return nil
	}

	if browser.Timeout < 1 || browser.Timeout > 600 {
		return errors.InvalidConfig("browser timeout must be between 1 and 600 seconds")
	}

	return nil
}

// validateGateway validates gateway configuration
func (v *Validator) validateGateway(cfg *Config) error {
	if cfg.Gateway.Port < 1024 || cfg.Gateway.Port > 65535 {
		return errors.InvalidConfig("gateway port must be between 1024 and 65535")
	}

	if cfg.Gateway.ReadTimeout < 1 || cfg.Gateway.ReadTimeout > 300 {
		return errors.InvalidConfig("gateway read_timeout must be between 1 and 300 seconds")
	}

	if cfg.Gateway.WriteTimeout < 1 || cfg.Gateway.WriteTimeout > 300 {
		return errors.InvalidConfig("gateway write_timeout must be between 1 and 300 seconds")
	}

	// Validate WebSocket configuration
	if err := v.validateWebSocketConfig(&cfg.Gateway.WebSocket); err != nil {
		return err
	}

	return nil
}

// validateWebSocketConfig validates WebSocket configuration
func (v *Validator) validateWebSocketConfig(ws *WebSocketConfig) error {
	// Only validate WebSocket config if host is set (WebSocket is optional)
	if ws.Host == "" {
		return nil
	}

	if ws.Port < 1024 || ws.Port > 65535 {
		return errors.InvalidConfig("websocket port must be between 1024 and 65535")
	}

	if ws.PingInterval < 1*time.Second || ws.PingInterval > 5*time.Minute {
		return errors.InvalidConfig("websocket ping_interval must be between 1s and 5m")
	}

	if ws.PongTimeout < 1*time.Second || ws.PongTimeout > 5*time.Minute {
		return errors.InvalidConfig("websocket pong_timeout must be between 1s and 5m")
	}

	return nil
}

// validateMemory validates memory configuration
func (v *Validator) validateMemory(cfg *Config) error {
	if cfg.Memory.Backend == "" {
		return errors.InvalidConfig("memory backend cannot be empty")
	}

	validBackends := []string{"builtin", "qmd"}
	if !slices.Contains(validBackends, cfg.Memory.Backend) {
		return errors.InvalidConfig(fmt.Sprintf("invalid memory backend: %s", cfg.Memory.Backend))
	}

	if cfg.Memory.Catalog.Enabled {
		if strings.TrimSpace(cfg.Memory.Catalog.DefaultProject) == "" {
			return errors.InvalidConfig("memory catalog default_project cannot be empty")
		}
		if cfg.Memory.Catalog.ReviewAfterDays < 1 || cfg.Memory.Catalog.ReviewAfterDays > 3650 {
			return errors.InvalidConfig("memory catalog review_after_days must be between 1 and 3650")
		}
		if cfg.Memory.Catalog.MaxContextRecords < 1 || cfg.Memory.Catalog.MaxContextRecords > 50 {
			return errors.InvalidConfig("memory catalog max_context_records must be between 1 and 50")
		}
		if cfg.Memory.Catalog.MaxContextChars < 1000 || cfg.Memory.Catalog.MaxContextChars > 200000 {
			return errors.InvalidConfig("memory catalog max_context_chars must be between 1000 and 200000")
		}
		if cfg.Memory.Catalog.AutoIngest &&
			len(cfg.Memory.Catalog.SourcePaths) > 1 &&
			strings.TrimSpace(cfg.Memory.Catalog.SourceRoot) == "" {
			return errors.InvalidConfig(
				"memory catalog source_root is required when auto_ingest has multiple source_paths",
			)
		}
	}

	return nil
}
