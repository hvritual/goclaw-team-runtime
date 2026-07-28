package governance

import "time"

const (
	RoleAny               = "*"
	RoleSeedApprove       = "seed_approve"
	RoleEvolutionApprove  = "evolution_approve"
	RoleScenarioReview    = "scenario_review"
	RoleCapacityReview    = "capacity_review"
	RoleRiskReview        = "risk_review"
	RoleCostReview        = "cost_review"
	RoleTaskAccept        = "task_accept"
	RoleHarnessApprove    = "harness_approve"
	RoleHarnessPromote    = "harness_promote"
	RoleHarnessRollback   = "harness_rollback"
	RoleKnowledgeApprove  = "knowledge_approve"
	RoleMemoryApprove     = "memory_approve"
	RoleAuthorityManage   = "authority_manage"
	RoleReadinessOverride = "readiness_override"
	RoleConflictResolve   = "conflict_resolve"
	RoleEvaluationResolve = "evaluation_resolve"
	RoleOutcomeRecord     = "outcome_record"
	RoleKillSwitch        = "kill_switch"
	RoleSessionCancel     = "session_cancel"
	RoleTaskCancel        = "task_cancel"
)

// Config is the shared human-decision policy used by Ouroboros, Orchestrator
// Lite, Better Harness, and the Gateway. Reviewer secrets are never stored:
// TokenSHA256 contains a lowercase SHA-256 digest of the actual token.
type Config struct {
	Enabled                           bool                      `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	RequireAuthenticatedReviewers     bool                      `mapstructure:"require_authenticated_reviewers" json:"require_authenticated_reviewers" yaml:"require_authenticated_reviewers"`
	RequireRationale                  bool                      `mapstructure:"require_rationale" json:"require_rationale" yaml:"require_rationale"`
	RequireCounterargument            bool                      `mapstructure:"require_counterargument" json:"require_counterargument" yaml:"require_counterargument"`
	MinRationaleRunes                 int                       `mapstructure:"min_rationale_runes" json:"min_rationale_runes" yaml:"min_rationale_runes"`
	ForbidSelfApproval                bool                      `mapstructure:"forbid_self_approval" json:"forbid_self_approval" yaml:"forbid_self_approval"`
	SeedApprovalQuorum                int                       `mapstructure:"seed_approval_quorum" json:"seed_approval_quorum" yaml:"seed_approval_quorum"`
	HighRiskApprovalQuorum            int                       `mapstructure:"high_risk_approval_quorum" json:"high_risk_approval_quorum" yaml:"high_risk_approval_quorum"`
	EvolutionApprovalQuorum           int                       `mapstructure:"evolution_approval_quorum" json:"evolution_approval_quorum" yaml:"evolution_approval_quorum"`
	HarnessApprovalQuorum             int                       `mapstructure:"harness_approval_quorum" json:"harness_approval_quorum" yaml:"harness_approval_quorum"`
	MinDistinctTaskReviewers          int                       `mapstructure:"min_distinct_task_reviewers" json:"min_distinct_task_reviewers" yaml:"min_distinct_task_reviewers"`
	MaxTaskReviewKindsPerReviewer     int                       `mapstructure:"max_task_review_kinds_per_reviewer" json:"max_task_review_kinds_per_reviewer" yaml:"max_task_review_kinds_per_reviewer"`
	ForbidFinalApproverFromTaskReview bool                      `mapstructure:"forbid_final_approver_from_task_review" json:"forbid_final_approver_from_task_review" yaml:"forbid_final_approver_from_task_review"`
	ForbidHarnessPromoterFromApproval bool                      `mapstructure:"forbid_harness_promoter_from_approval" json:"forbid_harness_promoter_from_approval" yaml:"forbid_harness_promoter_from_approval"`
	Reviewers                         map[string]ReviewerConfig `mapstructure:"reviewers" json:"reviewers,omitempty" yaml:"reviewers,omitempty"`
}

type ReviewerConfig struct {
	TokenSHA256 string   `mapstructure:"token_sha256" json:"token_sha256" yaml:"token_sha256"`
	Roles       []string `mapstructure:"roles" json:"roles" yaml:"roles"`
	// TeamUserID binds a reviewer policy identity to the authenticated
	// TeamControl principal. It lets policy keys remain descriptive while the
	// Gateway derives the human identity from the personal access token.
	TeamUserID string `mapstructure:"team_user_id" json:"team_user_id,omitempty" yaml:"team_user_id,omitempty"`
}

// Review is constructed only after the caller boundary authenticates a human
// identity. Network clients must never be allowed to set Authenticated
// directly.
type Review struct {
	ReviewerID      string    `json:"reviewer_id"`
	Rationale       string    `json:"rationale"`
	Counterargument string    `json:"counterargument,omitempty"`
	EvidenceRefs    []string  `json:"evidence_refs,omitempty"`
	Role            string    `json:"role"`
	Source          string    `json:"source"`
	Authenticated   bool      `json:"authenticated"`
	CreatedAt       time.Time `json:"created_at"`
}

type Credential struct {
	ReviewerID string
	Token      string
	Source     string
}

type DecisionRecord struct {
	ReviewerID      string    `json:"reviewer_id"`
	Decision        string    `json:"decision"`
	Rationale       string    `json:"rationale"`
	Counterargument string    `json:"counterargument,omitempty"`
	EvidenceRefs    []string  `json:"evidence_refs,omitempty"`
	Role            string    `json:"role"`
	Source          string    `json:"source"`
	Authenticated   bool      `json:"authenticated"`
	CreatedAt       time.Time `json:"created_at"`
}

func DefaultConfig() Config {
	return Config{
		RequireRationale:              true,
		MinRationaleRunes:             12,
		SeedApprovalQuorum:            1,
		HighRiskApprovalQuorum:        2,
		EvolutionApprovalQuorum:       1,
		HarnessApprovalQuorum:         1,
		MinDistinctTaskReviewers:      2,
		MaxTaskReviewKindsPerReviewer: 2,
	}
}

func IsKnownRole(role string) bool {
	switch role {
	case RoleAny,
		RoleSeedApprove,
		RoleEvolutionApprove,
		RoleScenarioReview,
		RoleCapacityReview,
		RoleRiskReview,
		RoleCostReview,
		RoleTaskAccept,
		RoleHarnessApprove,
		RoleHarnessPromote,
		RoleHarnessRollback,
		RoleKnowledgeApprove,
		RoleMemoryApprove,
		RoleAuthorityManage,
		RoleReadinessOverride,
		RoleConflictResolve,
		RoleEvaluationResolve,
		RoleOutcomeRecord,
		RoleKillSwitch,
		RoleSessionCancel,
		RoleTaskCancel:
		return true
	default:
		return false
	}
}
