package ouroboros

import (
	"context"
	"encoding/json"
	"time"

	"github.com/smallnest/goclaw/governance"
)

const (
	SchemaVersion                = 1
	DefaultAmbiguityThreshold    = 0.20
	DefaultConvergenceThreshold  = 0.95
	DefaultRequiredReadyStreak   = 2
	DefaultMaxGenerations        = 30
	DefaultConsensusReviewers    = 3
	DefaultMaxQuestionsPerRound  = 5
	DefaultMaxContextBytes       = 128 * 1024
	DefaultMaxOutputTokens       = 12000
	DefaultAssessmentReviewers   = 2
	DefaultAssessmentMaxSpread   = 0.15
	DefaultAssessmentGrayZone    = 0.03
	DefaultConsensusMaxSpread    = 0.25
	DefaultEvaluationWindow      = 5
	DefaultPassingWindow         = 2
	DefaultMaxSessionModelCalls  = 120
	DefaultMaxSessionModelTokens = 2_000_000
	CalibrationVersion           = "clarity-v1"
)

// Config controls the Go-native Ouroboros specification runtime. Root must
// stay on the single-writer host and outside a synchronized Obsidian vault.
type Config struct {
	Enabled                    bool     `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	Root                       string   `mapstructure:"root" json:"root" yaml:"root"`
	Model                      string   `mapstructure:"model" json:"model" yaml:"model"`
	AssessmentModels           []string `mapstructure:"assessment_models" json:"assessment_models,omitempty" yaml:"assessment_models,omitempty"`
	EvaluationModels           []string `mapstructure:"evaluation_models" json:"evaluation_models,omitempty" yaml:"evaluation_models,omitempty"`
	AmbiguityThreshold         float64  `mapstructure:"ambiguity_threshold" json:"ambiguity_threshold" yaml:"ambiguity_threshold"`
	ConvergenceThreshold       float64  `mapstructure:"convergence_threshold" json:"convergence_threshold" yaml:"convergence_threshold"`
	RequiredReadyStreak        int      `mapstructure:"required_ready_streak" json:"required_ready_streak" yaml:"required_ready_streak"`
	MaxGenerations             int      `mapstructure:"max_generations" json:"max_generations" yaml:"max_generations"`
	ConsensusReviewers         int      `mapstructure:"consensus_reviewers" json:"consensus_reviewers" yaml:"consensus_reviewers"`
	MaxQuestionsPerRound       int      `mapstructure:"max_questions_per_round" json:"max_questions_per_round" yaml:"max_questions_per_round"`
	MaxContextBytes            int      `mapstructure:"max_context_bytes" json:"max_context_bytes" yaml:"max_context_bytes"`
	MaxOutputTokens            int      `mapstructure:"max_output_tokens" json:"max_output_tokens" yaml:"max_output_tokens"`
	AssessmentReviewers        int      `mapstructure:"assessment_reviewers" json:"assessment_reviewers" yaml:"assessment_reviewers"`
	AssessmentMaxSpread        float64  `mapstructure:"assessment_max_spread" json:"assessment_max_spread" yaml:"assessment_max_spread"`
	AssessmentGrayZone         float64  `mapstructure:"assessment_gray_zone" json:"assessment_gray_zone" yaml:"assessment_gray_zone"`
	CriticalFindingVeto        bool     `mapstructure:"critical_finding_veto" json:"critical_finding_veto" yaml:"critical_finding_veto"`
	ConsensusMaxSpread         float64  `mapstructure:"consensus_max_spread" json:"consensus_max_spread" yaml:"consensus_max_spread"`
	EvaluationHistoryWindow    int      `mapstructure:"evaluation_history_window" json:"evaluation_history_window" yaml:"evaluation_history_window"`
	RequiredPassingEvaluations int      `mapstructure:"required_passing_evaluations" json:"required_passing_evaluations" yaml:"required_passing_evaluations"`
	MaxSessionModelCalls       int      `mapstructure:"max_session_model_calls" json:"max_session_model_calls" yaml:"max_session_model_calls"`
	MaxSessionModelTokens      int      `mapstructure:"max_session_model_tokens" json:"max_session_model_tokens" yaml:"max_session_model_tokens"`
}

type SessionStatus string

const (
	StatusInterviewing         SessionStatus = "interviewing"
	StatusClarificationNeeded  SessionStatus = "clarification_required"
	StatusSeedReady            SessionStatus = "seed_ready"
	StatusAwaitingSeedApproval SessionStatus = "awaiting_seed_approval"
	StatusApproved             SessionStatus = "approved"
	StatusCompiled             SessionStatus = "compiled"
	StatusEvaluated            SessionStatus = "evaluated"
	StatusEvolutionPending     SessionStatus = "evolution_pending"
	StatusConverged            SessionStatus = "converged"
	StatusBlocked              SessionStatus = "blocked"
	StatusRejected             SessionStatus = "rejected"
	StatusCancelled            SessionStatus = "cancelled"
)

type Dimension string

const (
	DimensionGoal       Dimension = "goal"
	DimensionConstraint Dimension = "constraint"
	DimensionSuccess    Dimension = "success"
	DimensionContext    Dimension = "context"
)

type StartRequest struct {
	ID             string   `json:"id,omitempty"`
	ProjectID      string   `json:"project_id"`
	TopicID        string   `json:"topic_id,omitempty"`
	Title          string   `json:"title"`
	RepoPath       string   `json:"repo_path"`
	BaseRef        string   `json:"base_ref,omitempty"`
	RawRequest     string   `json:"raw_request"`
	ContextSummary string   `json:"context_summary,omitempty"`
	Brownfield     bool     `json:"brownfield"`
	CreatedBy      string   `json:"created_by"`
	Stakeholders   []string `json:"stakeholders,omitempty"`
}

type Question struct {
	ID        string    `json:"id"`
	Dimension Dimension `json:"dimension"`
	Text      string    `json:"text"`
	Why       string    `json:"why,omitempty"`
	Blocking  bool      `json:"blocking"`
}

type Answer struct {
	QuestionID string    `json:"question_id"`
	Text       string    `json:"text"`
	AnsweredBy string    `json:"answered_by"`
	CreatedAt  time.Time `json:"created_at"`
}

type AnswerRequest struct {
	Answers  []Answer `json:"answers"`
	Actor    string   `json:"actor"`
	Reassess bool     `json:"reassess"`
}

type AssumptionStatus string

const (
	AssumptionProposed AssumptionStatus = "proposed"
	AssumptionAccepted AssumptionStatus = "accepted"
	AssumptionRejected AssumptionStatus = "rejected"
	AssumptionDeferred AssumptionStatus = "deferred"
)

type Assumption struct {
	ID        string           `json:"id"`
	Text      string           `json:"text"`
	Status    AssumptionStatus `json:"status"`
	Source    string           `json:"source,omitempty"`
	UpdatedAt time.Time        `json:"updated_at"`
}

type DimensionScore struct {
	Dimension     Dimension `json:"dimension"`
	Clarity       float64   `json:"clarity"`
	Weight        float64   `json:"weight"`
	WeightedValue float64   `json:"weighted_value"`
	Justification string    `json:"justification"`
	Floor         float64   `json:"floor"`
	FloorPassed   bool      `json:"floor_passed"`
}

type AmbiguityAssessment struct {
	Round                 int                `json:"round"`
	Overall               float64            `json:"overall"`
	Threshold             float64            `json:"threshold"`
	Dimensions            []DimensionScore   `json:"dimensions"`
	Ready                 bool               `json:"ready"`
	ReadyStreak           int                `json:"ready_streak"`
	RequiredReadyStreak   int                `json:"required_ready_streak"`
	Summary               string             `json:"summary"`
	Unresolved            []string           `json:"unresolved,omitempty"`
	RepeatedQuestionRatio float64            `json:"repeated_question_ratio,omitempty"`
	HumanDecisionRequired bool               `json:"human_decision_required,omitempty"`
	AssessorVotes         []AssessmentVote   `json:"assessor_votes,omitempty"`
	ScoreSpread           float64            `json:"score_spread,omitempty"`
	DistinctModels        int                `json:"distinct_models,omitempty"`
	GrayZone              bool               `json:"gray_zone,omitempty"`
	CalibrationVersion    string             `json:"calibration_version,omitempty"`
	HumanOverride         *ReadinessOverride `json:"human_override,omitempty"`
	CreatedAt             time.Time          `json:"created_at"`
}

type AssessmentVote struct {
	Role       string                `json:"role"`
	Model      string                `json:"model,omitempty"`
	Overall    float64               `json:"overall"`
	Dimensions map[Dimension]float64 `json:"dimensions"`
	Summary    string                `json:"summary"`
}

type ReadinessOverride struct {
	Ready     bool                      `json:"ready"`
	Reviewer  string                    `json:"reviewer"`
	Rationale string                    `json:"rationale"`
	Decision  governance.DecisionRecord `json:"decision"`
	CreatedAt time.Time                 `json:"created_at"`
}

type InterviewRound struct {
	Number     int                 `json:"number"`
	Questions  []Question          `json:"questions"`
	Answers    []Answer            `json:"answers,omitempty"`
	Assessment AmbiguityAssessment `json:"assessment"`
	CreatedAt  time.Time           `json:"created_at"`
}

type RepositoryContext struct {
	TopLevel   []string          `json:"top_level,omitempty"`
	Manifests  map[string]string `json:"manifests,omitempty"`
	GitPresent bool              `json:"git_present"`
	Truncated  bool              `json:"truncated,omitempty"`
	CapturedAt time.Time         `json:"captured_at"`
}

type ProblemFrame struct {
	ID              string   `json:"id"`
	Perspective     string   `json:"perspective"`
	Problem         string   `json:"problem"`
	ExpectedBenefit string   `json:"expected_benefit"`
	CostOfInaction  string   `json:"cost_of_inaction"`
	Risks           []string `json:"risks,omitempty"`
	Assumptions     []string `json:"assumptions,omitempty"`
}

type StakeholderClaim struct {
	ID          string    `json:"id"`
	Stakeholder string    `json:"stakeholder"`
	Statement   string    `json:"statement"`
	Source      string    `json:"source"`
	Status      string    `json:"status"`
	Round       int       `json:"round"`
	CreatedAt   time.Time `json:"created_at"`
}

type DecisionConflict struct {
	ID          string     `json:"id"`
	Description string     `json:"description"`
	ClaimIDs    []string   `json:"claim_ids,omitempty"`
	Status      string     `json:"status"`
	Resolution  string     `json:"resolution,omitempty"`
	ResolvedBy  string     `json:"resolved_by,omitempty"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
}

type OutcomeRecord struct {
	ID           string    `json:"id"`
	SupersedesID string    `json:"supersedes_id,omitempty"`
	Kind         string    `json:"kind"`
	EvaluationID string    `json:"evaluation_id,omitempty"`
	TaskID       string    `json:"task_id,omitempty"`
	SeedHash     string    `json:"seed_hash,omitempty"`
	RiskLevel    string    `json:"risk_level,omitempty"`
	Passed       bool      `json:"passed"`
	Reason       string    `json:"reason"`
	EvidenceRefs []string  `json:"evidence_refs,omitempty"`
	Actor        string    `json:"actor"`
	CreatedAt    time.Time `json:"created_at"`
}

type OutcomeRequest struct {
	Kind         string            `json:"kind"`
	EvaluationID string            `json:"evaluation_id,omitempty"`
	TaskID       string            `json:"task_id,omitempty"`
	SeedHash     string            `json:"seed_hash,omitempty"`
	RiskLevel    string            `json:"risk_level,omitempty"`
	Reason       string            `json:"reason"`
	EvidenceRefs []string          `json:"evidence_refs,omitempty"`
	Review       governance.Review `json:"-"`
}

type ReferenceClassStats struct {
	ProjectID   string  `json:"project_id"`
	Total       int     `json:"total"`
	Passed      int     `json:"passed"`
	Failed      int     `json:"failed"`
	Cancelled   int     `json:"cancelled"`
	RolledBack  int     `json:"rolled_back"`
	NoFeedback  int     `json:"no_feedback"`
	PassRate    float64 `json:"pass_rate"`
	FailureRate float64 `json:"failure_rate"`
}

type KillTrigger struct {
	ConditionID  string    `json:"condition_id"`
	Reason       string    `json:"reason"`
	EvidenceRefs []string  `json:"evidence_refs,omitempty"`
	TriggeredBy  string    `json:"triggered_by"`
	TriggeredAt  time.Time `json:"triggered_at"`
}

type Session struct {
	SchemaVersion     int                    `json:"schema_version"`
	ID                string                 `json:"id"`
	ProjectID         string                 `json:"project_id"`
	TopicID           string                 `json:"topic_id,omitempty"`
	Title             string                 `json:"title"`
	RepoPath          string                 `json:"repo_path"`
	BaseRef           string                 `json:"base_ref"`
	RawRequest        string                 `json:"raw_request"`
	ContextSummary    string                 `json:"context_summary,omitempty"`
	RepositoryContext RepositoryContext      `json:"repository_context,omitempty"`
	Brownfield        bool                   `json:"brownfield"`
	Stakeholders      []string               `json:"stakeholders,omitempty"`
	Status            SessionStatus          `json:"status"`
	Rounds            []InterviewRound       `json:"rounds"`
	Assumptions       []Assumption           `json:"assumptions,omitempty"`
	SeedHistory       []SeedReference        `json:"seed_history,omitempty"`
	ActiveSeedHash    string                 `json:"active_seed_hash,omitempty"`
	PendingSeedHash   string                 `json:"pending_seed_hash,omitempty"`
	CompiledTasks     []CompiledTask         `json:"compiled_tasks,omitempty"`
	Evaluations       []Evaluation           `json:"evaluations,omitempty"`
	PendingEvolution  *EvolutionProposal     `json:"pending_evolution,omitempty"`
	LastEvolution     *EvolutionProposal     `json:"last_evolution,omitempty"`
	CreatedBy         string                 `json:"created_by"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
	LastError         string                 `json:"last_error,omitempty"`
	BlockedReasons    []string               `json:"blocked_reasons,omitempty"`
	ModelUsage        ModelUsage             `json:"model_usage,omitempty"`
	DecisionLedger    []DecisionLedgerRecord `json:"decision_ledger,omitempty"`
	ProblemFrames     []ProblemFrame         `json:"problem_frames,omitempty"`
	StakeholderClaims []StakeholderClaim     `json:"stakeholder_claims,omitempty"`
	DecisionConflicts []DecisionConflict     `json:"decision_conflicts,omitempty"`
	Outcomes          []OutcomeRecord        `json:"outcomes,omitempty"`
	ReferenceClass    ReferenceClassStats    `json:"reference_class,omitempty"`
	KillTriggers      []KillTrigger          `json:"kill_triggers,omitempty"`
}

type DecisionLedgerRecord struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	Decision  string    `json:"decision"`
	Rationale string    `json:"rationale"`
	Actor     string    `json:"actor"`
	CreatedAt time.Time `json:"created_at"`
}

type AcceptanceCriterion struct {
	ID                string   `json:"id"`
	Description       string   `json:"description"`
	VerifyCommand     []string `json:"verify_command,omitempty"`
	ExpectedArtifacts []string `json:"expected_artifacts,omitempty"`
	OutputAssertion   string   `json:"output_assertion,omitempty"`
}

type OntologyField struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

type Ontology struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Fields      []OntologyField `json:"fields"`
}

type EvaluationPrinciple struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ExitCondition struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Criteria    string `json:"criteria"`
}

type SeedWorkItem struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Instructions string   `json:"instructions"`
	DependsOn    []string `json:"depends_on,omitempty"`
	CriteriaIDs  []string `json:"criteria_ids,omitempty"`
}

type SeedMilestone struct {
	ID        string         `json:"id"`
	Title     string         `json:"title"`
	WorkItems []SeedWorkItem `json:"work_items"`
}

type SeedPlan struct {
	Summary    string          `json:"summary"`
	Milestones []SeedMilestone `json:"milestones"`
}

type SeedScope struct {
	AllowedPaths       []string `json:"allowed_paths"`
	DeniedPaths        []string `json:"denied_paths,omitempty"`
	MaxChangedFiles    int      `json:"max_changed_files"`
	MaxChangedLines    int      `json:"max_changed_lines"`
	AllowNewDependency bool     `json:"allow_new_dependency"`
}

type SeedRisk struct {
	Level          string   `json:"level"`
	Forbidden      []string `json:"forbidden,omitempty"`
	Rollback       string   `json:"rollback"`
	HumanEscalates []string `json:"human_escalates,omitempty"`
}

type SeedCost struct {
	MaxRepairAttempts int `json:"max_repair_attempts"`
	MaxInputTokens    int `json:"max_input_tokens,omitempty"`
	MaxOutputTokens   int `json:"max_output_tokens,omitempty"`
}

type Alternative struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Summary   string   `json:"summary"`
	Tradeoffs []string `json:"tradeoffs,omitempty"`
	Selected  bool     `json:"selected"`
}

type Falsifier struct {
	CriterionID      string `json:"criterion_id"`
	Condition        string `json:"condition"`
	EvidenceRequired string `json:"evidence_required"`
}

type KillCondition struct {
	ID        string `json:"id"`
	Condition string `json:"condition"`
	Metric    string `json:"metric"`
	Threshold string `json:"threshold"`
	Action    string `json:"action"`
}

type ReferenceClassForecast struct {
	Basis              string  `json:"basis"`
	SampleSize         int     `json:"sample_size"`
	BaseFailureRate    float64 `json:"base_failure_rate"`
	P50DurationMinutes int     `json:"p50_duration_minutes,omitempty"`
	P90DurationMinutes int     `json:"p90_duration_minutes,omitempty"`
	P50InputTokens     int     `json:"p50_input_tokens,omitempty"`
	P90InputTokens     int     `json:"p90_input_tokens,omitempty"`
}

type Prediction struct {
	ID              string  `json:"id"`
	Claim           string  `json:"claim"`
	ExpectedOutcome string  `json:"expected_outcome"`
	Horizon         string  `json:"horizon"`
	Confidence      float64 `json:"confidence"`
}

// Seed is an immutable execution specification. Once written under seeds/,
// no operation overwrites it; successors point to ParentHash.
type Seed struct {
	SchemaVersion        int                    `json:"schema_version"`
	ID                   string                 `json:"id"`
	SessionID            string                 `json:"session_id"`
	Generation           int                    `json:"generation"`
	ParentHash           string                 `json:"parent_hash,omitempty"`
	Title                string                 `json:"title"`
	Goal                 string                 `json:"goal"`
	TaskType             string                 `json:"task_type"`
	ContextSummary       string                 `json:"context_summary,omitempty"`
	Brownfield           bool                   `json:"brownfield"`
	Constraints          []string               `json:"constraints"`
	NonGoals             []string               `json:"non_goals,omitempty"`
	Assumptions          []string               `json:"assumptions,omitempty"`
	AcceptanceCriteria   []AcceptanceCriterion  `json:"acceptance_criteria"`
	Ontology             Ontology               `json:"ontology"`
	EvaluationPrinciples []EvaluationPrinciple  `json:"evaluation_principles"`
	ExitConditions       []ExitCondition        `json:"exit_conditions"`
	Plan                 SeedPlan               `json:"plan"`
	Scope                SeedScope              `json:"scope"`
	Risk                 SeedRisk               `json:"risk"`
	Cost                 SeedCost               `json:"cost"`
	Alternatives         []Alternative          `json:"alternatives"`
	Falsifiers           []Falsifier            `json:"falsifiers"`
	CostOfInaction       []string               `json:"cost_of_inaction"`
	KillConditions       []KillCondition        `json:"kill_conditions"`
	PreMortem            []string               `json:"pre_mortem"`
	ReferenceClass       ReferenceClassForecast `json:"reference_class"`
	Predictions          []Prediction           `json:"predictions"`
	StakeholderClaimIDs  []string               `json:"stakeholder_claim_ids,omitempty"`
	AmbiguityScore       float64                `json:"ambiguity_score"`
	CreatedAt            time.Time              `json:"created_at"`
	CreatedBy            string                 `json:"created_by"`
	Hash                 string                 `json:"hash"`
}

type SeedReference struct {
	Hash       string                      `json:"hash"`
	ID         string                      `json:"id"`
	Generation int                         `json:"generation"`
	ParentHash string                      `json:"parent_hash,omitempty"`
	Approved   bool                        `json:"approved"`
	ApprovedBy string                      `json:"approved_by,omitempty"`
	ApprovedAt *time.Time                  `json:"approved_at,omitempty"`
	Comment    string                      `json:"comment,omitempty"`
	Approvals  []governance.DecisionRecord `json:"approvals,omitempty"`
	CreatedAt  time.Time                   `json:"created_at"`
}

type CompiledTask struct {
	SeedHash   string    `json:"seed_hash"`
	Generation int       `json:"generation"`
	TaskID     string    `json:"task_id"`
	CompiledBy string    `json:"compiled_by"`
	CompiledAt time.Time `json:"compiled_at"`
}

type EvaluationStage struct {
	Name             string            `json:"name"`
	Passed           bool              `json:"passed"`
	Score            float64           `json:"score"`
	Summary          string            `json:"summary"`
	Findings         []string          `json:"findings,omitempty"`
	UnmetCriteria    []string          `json:"unmet_criteria,omitempty"`
	EvidenceRefs     []string          `json:"evidence_refs,omitempty"`
	Reviewer         string            `json:"reviewer"`
	Model            string            `json:"model,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
	CriticalFindings []string          `json:"critical_findings,omitempty"`
	Blinded          bool              `json:"blinded,omitempty"`
	IndependenceKey  string            `json:"independence_key,omitempty"`
}

type Evaluation struct {
	ID                    string                 `json:"id"`
	SessionID             string                 `json:"session_id"`
	SeedHash              string                 `json:"seed_hash"`
	TaskID                string                 `json:"task_id"`
	Mechanical            EvaluationStage        `json:"mechanical"`
	Semantic              EvaluationStage        `json:"semantic"`
	Reviews               []EvaluationStage      `json:"reviews,omitempty"`
	Consensus             EvaluationStage        `json:"consensus"`
	Passed                bool                   `json:"passed"`
	CreatedAt             time.Time              `json:"created_at"`
	ModelUsage            ModelUsage             `json:"model_usage,omitempty"`
	ScoreSpread           float64                `json:"score_spread,omitempty"`
	DistinctModels        int                    `json:"distinct_models,omitempty"`
	HumanDecisionRequired bool                   `json:"human_decision_required,omitempty"`
	HumanDisposition      *EvaluationDisposition `json:"human_disposition,omitempty"`
}

// EvaluationDisposition adjudicates disputed evidence. It does not approve
// deployment, promote a Harness candidate, or bypass deterministic gates.
type EvaluationDisposition struct {
	Accepted bool                      `json:"accepted"`
	Decision governance.DecisionRecord `json:"decision"`
}

type EvolutionStatus string

const (
	EvolutionPending   EvolutionStatus = "pending"
	EvolutionApproved  EvolutionStatus = "approved"
	EvolutionRejected  EvolutionStatus = "rejected"
	EvolutionConverged EvolutionStatus = "converged"
	EvolutionBlocked   EvolutionStatus = "blocked"
)

type EvolutionProposal struct {
	ID                    string                      `json:"id"`
	SessionID             string                      `json:"session_id"`
	FromSeedHash          string                      `json:"from_seed_hash"`
	CandidateSeedHash     string                      `json:"candidate_seed_hash,omitempty"`
	FromGeneration        int                         `json:"from_generation"`
	CandidateGeneration   int                         `json:"candidate_generation,omitempty"`
	OntologySimilarity    float64                     `json:"ontology_similarity"`
	ConvergenceThreshold  float64                     `json:"convergence_threshold"`
	Status                EvolutionStatus             `json:"status"`
	Action                string                      `json:"action"`
	Reasons               []string                    `json:"reasons,omitempty"`
	KnowledgeGaps         []string                    `json:"knowledge_gaps,omitempty"`
	PossibleRegressions   []string                    `json:"possible_regressions,omitempty"`
	OscillationDetected   bool                        `json:"oscillation_detected,omitempty"`
	HardCapReached        bool                        `json:"hard_cap_reached,omitempty"`
	CreatedBy             string                      `json:"created_by"`
	CreatedAt             time.Time                   `json:"created_at"`
	ReviewedBy            string                      `json:"reviewed_by,omitempty"`
	ReviewedAt            *time.Time                  `json:"reviewed_at,omitempty"`
	ReviewComment         string                      `json:"review_comment,omitempty"`
	Approvals             []governance.DecisionRecord `json:"approvals,omitempty"`
	HistoryWindow         int                         `json:"history_window,omitempty"`
	PassingInWindow       int                         `json:"passing_in_window,omitempty"`
	CumulativeModelCalls  int                         `json:"cumulative_model_calls,omitempty"`
	CumulativeModelTokens int                         `json:"cumulative_model_tokens,omitempty"`
}

type ModelRequest struct {
	Purpose     string  `json:"purpose"`
	System      string  `json:"system"`
	User        string  `json:"user"`
	Model       string  `json:"model,omitempty"`
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"max_tokens"`
}

type ModelResponse struct {
	Content string     `json:"content"`
	Model   string     `json:"model,omitempty"`
	Usage   ModelUsage `json:"usage,omitempty"`
}

type ModelUsage struct {
	InputTokens  int `json:"input_tokens,omitempty"`
	OutputTokens int `json:"output_tokens,omitempty"`
	TotalTokens  int `json:"total_tokens,omitempty"`
	Calls        int `json:"calls,omitempty"`
}

type Model interface {
	Generate(ctx context.Context, request ModelRequest) (ModelResponse, error)
}

type SessionEvent struct {
	SchemaVersion int             `json:"schema_version"`
	ID            string          `json:"id"`
	SessionID     string          `json:"session_id"`
	Sequence      int64           `json:"sequence"`
	Type          string          `json:"type"`
	Actor         string          `json:"actor"`
	CreatedAt     time.Time       `json:"created_at"`
	PreviousHash  string          `json:"previous_hash,omitempty"`
	Data          json.RawMessage `json:"data,omitempty"`
	Snapshot      Session         `json:"snapshot"`
	Hash          string          `json:"hash"`
}
