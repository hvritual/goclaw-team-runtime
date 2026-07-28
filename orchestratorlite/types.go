package orchestratorlite

import (
	"encoding/json"
	"time"

	"github.com/smallnest/goclaw/governance"
)

const SchemaVersion = 1

// Config controls the single-node Orchestrator Lite development runtime.
// Runtime state and worktrees must remain outside synchronized Obsidian vaults.
type Config struct {
	Enabled                   bool     `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	Root                      string   `mapstructure:"root" json:"root" yaml:"root"`
	WorktreeRoot              string   `mapstructure:"worktree_root" json:"worktree_root" yaml:"worktree_root"`
	RepoPath                  string   `mapstructure:"repo_path" json:"repo_path" yaml:"repo_path"`
	CodexCommand              string   `mapstructure:"codex_command" json:"codex_command" yaml:"codex_command"`
	CodexModel                string   `mapstructure:"codex_model" json:"codex_model" yaml:"codex_model"`
	RunTimeoutSeconds         int      `mapstructure:"run_timeout_seconds" json:"run_timeout_seconds" yaml:"run_timeout_seconds"`
	VerifyTimeoutSeconds      int      `mapstructure:"verify_timeout_seconds" json:"verify_timeout_seconds" yaml:"verify_timeout_seconds"`
	VerificationSandbox       []string `mapstructure:"verification_sandbox" json:"verification_sandbox,omitempty" yaml:"verification_sandbox,omitempty"`
	UnsafeHostVerification    bool     `mapstructure:"unsafe_host_verification" json:"unsafe_host_verification,omitempty" yaml:"unsafe_host_verification,omitempty"`
	MaxRepairAttempts         int      `mapstructure:"max_repair_attempts" json:"max_repair_attempts" yaml:"max_repair_attempts"`
	DefaultMaxChangedFiles    int      `mapstructure:"default_max_changed_files" json:"default_max_changed_files" yaml:"default_max_changed_files"`
	DefaultMaxChangedLines    int      `mapstructure:"default_max_changed_lines" json:"default_max_changed_lines" yaml:"default_max_changed_lines"`
	DeniedPaths               []string `mapstructure:"denied_paths" json:"denied_paths" yaml:"denied_paths"`
	AllowDirtyRepo            bool     `mapstructure:"allow_dirty_repo" json:"allow_dirty_repo" yaml:"allow_dirty_repo"`
	IndependentReview         bool     `mapstructure:"independent_review" json:"independent_review" yaml:"independent_review"`
	GatewayAllowExecution     bool     `mapstructure:"gateway_allow_execution" json:"gateway_allow_execution" yaml:"gateway_allow_execution"`
	RequireHumanFinalApproval bool     `mapstructure:"require_human_final_approval" json:"require_human_final_approval" yaml:"require_human_final_approval"`
}

type TaskStatus string

const (
	TaskReviewPending      TaskStatus = "review_pending"
	TaskReadyToFreeze      TaskStatus = "ready_to_freeze"
	TaskBlocked            TaskStatus = "blocked"
	TaskFrozen             TaskStatus = "frozen"
	TaskRunning            TaskStatus = "running"
	TaskChecking           TaskStatus = "checking"
	TaskRepairPending      TaskStatus = "repair_pending"
	TaskAwaitingAcceptance TaskStatus = "awaiting_acceptance"
	TaskDone               TaskStatus = "done"
	TaskFailed             TaskStatus = "failed"
	TaskCancelled          TaskStatus = "cancelled"
)

type ReviewKind string

const (
	ReviewScenario ReviewKind = "scenario"
	ReviewCapacity ReviewKind = "capacity"
	ReviewRisk     ReviewKind = "risk"
	ReviewCost     ReviewKind = "cost"
)

var RequiredReviewKinds = []ReviewKind{
	ReviewScenario,
	ReviewCapacity,
	ReviewRisk,
	ReviewCost,
}

type ReviewDecision string

const (
	ReviewPending  ReviewDecision = "pending"
	ReviewApproved ReviewDecision = "approved"
	ReviewRejected ReviewDecision = "rejected"
)

type RequestFrame struct {
	RawRequest  string   `json:"raw_request"`
	Attachments []string `json:"attachments,omitempty"`
	Source      string   `json:"source,omitempty"`
}

type GoalSpec struct {
	Objective      string                `json:"objective"`
	NonGoals       []string              `json:"non_goals,omitempty"`
	Assumptions    []string              `json:"assumptions,omitempty"`
	SuccessTests   []string              `json:"success_tests,omitempty"`
	Alternatives   []DecisionAlternative `json:"alternatives,omitempty"`
	CostOfInaction []string              `json:"cost_of_inaction,omitempty"`
	Falsifiers     []TaskFalsifier       `json:"falsifiers,omitempty"`
	Predictions    []TaskPrediction      `json:"predictions,omitempty"`
	PreMortem      []string              `json:"pre_mortem,omitempty"`
}

type DecisionAlternative struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Summary   string   `json:"summary"`
	Tradeoffs []string `json:"tradeoffs,omitempty"`
	Selected  bool     `json:"selected"`
}

type TaskFalsifier struct {
	CriterionID      string `json:"criterion_id"`
	Condition        string `json:"condition"`
	EvidenceRequired string `json:"evidence_required"`
}

type TaskPrediction struct {
	ID              string  `json:"id"`
	Claim           string  `json:"claim"`
	ExpectedOutcome string  `json:"expected_outcome"`
	Horizon         string  `json:"horizon"`
	Confidence      float64 `json:"confidence"`
}

type PlanSpec struct {
	Summary    string      `json:"summary"`
	Milestones []Milestone `json:"milestones"`
}

type Milestone struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	WorkItems []WorkItem `json:"work_items"`
}

type WorkItemStatus string

const (
	WorkItemPending   WorkItemStatus = "pending"
	WorkItemRunning   WorkItemStatus = "running"
	WorkItemVerifying WorkItemStatus = "verifying"
	WorkItemDone      WorkItemStatus = "done"
	WorkItemBlocked   WorkItemStatus = "blocked"
)

type WorkItem struct {
	ID                   string             `json:"id"`
	Title                string             `json:"title"`
	Instructions         string             `json:"instructions"`
	DependsOn            []string           `json:"depends_on,omitempty"`
	Status               WorkItemStatus     `json:"status,omitempty"`
	AssigneeID           string             `json:"assignee_id,omitempty"`
	IssueIDs             []string           `json:"issue_ids,omitempty"`
	DocumentRefs         []string           `json:"document_refs,omitempty"`
	ScopePaths           []string           `json:"scope_paths,omitempty"`
	AcceptanceCriteria   []string           `json:"acceptance_criteria,omitempty"`
	CapabilityManifest   CapabilityManifest `json:"capability_manifest"`
	VerificationCommands []CommandSpec      `json:"verification_commands"`
}

type CapabilityManifest struct {
	Executor string   `json:"executor"`
	Model    string   `json:"model,omitempty"`
	Tools    []string `json:"tools"`
	Sandbox  string   `json:"sandbox"`
}

type CommandSpec struct {
	Name string   `json:"name"`
	Argv []string `json:"argv"`
}

type EvidencePlan struct {
	Required []string      `json:"required"`
	Commands []CommandSpec `json:"commands"`
}

type ScopePolicy struct {
	AllowedPaths       []string `json:"allowed_paths,omitempty"`
	DeniedPaths        []string `json:"denied_paths,omitempty"`
	MaxChangedFiles    int      `json:"max_changed_files"`
	MaxChangedLines    int      `json:"max_changed_lines"`
	AllowNewDependency bool     `json:"allow_new_dependency"`
}

type RiskPlan struct {
	Level          string              `json:"level"`
	Forbidden      []string            `json:"forbidden,omitempty"`
	Rollback       string              `json:"rollback"`
	HumanEscalates []string            `json:"human_escalates,omitempty"`
	KillConditions []TaskKillCondition `json:"kill_conditions,omitempty"`
	ReferenceClass TaskReferenceClass  `json:"reference_class,omitempty"`
}

type TaskKillCondition struct {
	ID        string `json:"id"`
	Condition string `json:"condition"`
	Metric    string `json:"metric"`
	Threshold string `json:"threshold"`
	Action    string `json:"action"`
}

type TaskReferenceClass struct {
	Basis              string  `json:"basis"`
	SampleSize         int     `json:"sample_size"`
	BaseFailureRate    float64 `json:"base_failure_rate"`
	P50DurationMinutes int     `json:"p50_duration_minutes,omitempty"`
	P90DurationMinutes int     `json:"p90_duration_minutes,omitempty"`
	P50InputTokens     int     `json:"p50_input_tokens,omitempty"`
	P90InputTokens     int     `json:"p90_input_tokens,omitempty"`
}

type CostPlan struct {
	MaxRepairAttempts int `json:"max_repair_attempts"`
	MaxInputTokens    int `json:"max_input_tokens,omitempty"`
	MaxOutputTokens   int `json:"max_output_tokens,omitempty"`
}

type DoneGateSpec struct {
	RequireChangedFiles      bool `json:"require_changed_files"`
	RequireAllVerifications  bool `json:"require_all_verifications"`
	RequirePolicyPass        bool `json:"require_policy_pass"`
	RequireIndependentReview bool `json:"require_independent_review"`
	RequireHumanAcceptance   bool `json:"require_human_acceptance"`
	RequireWorkItemTrace     bool `json:"require_work_item_traceability,omitempty"`
	RequirePolicyBundle      bool `json:"require_policy_bundle,omitempty"`
	RequireDocumentEvidence  bool `json:"require_document_evidence,omitempty"`
}

type ReviewRecord struct {
	Kind       ReviewKind                 `json:"kind"`
	Decision   ReviewDecision             `json:"decision"`
	Reviewer   string                     `json:"reviewer,omitempty"`
	Comment    string                     `json:"comment,omitempty"`
	Governance *governance.DecisionRecord `json:"governance,omitempty"`
	DecidedAt  *time.Time                 `json:"decided_at,omitempty"`
}

type CompileRecord struct {
	Revision            int       `json:"revision"`
	BaseRef             string    `json:"base_ref"`
	BaseCommit          string    `json:"base_commit,omitempty"`
	ExecutionBundleHash string    `json:"execution_bundle_hash,omitempty"`
	FrozenAt            time.Time `json:"frozen_at,omitempty"`
	FrozenBy            string    `json:"frozen_by,omitempty"`
}

// WaveBinding is the immutable repository-governance authority for a task.
// Registry and plan hashes are calculated over the exact blobs at the frozen
// Git base, never over the mutable control-plane checkout.
type WaveBinding struct {
	WaveID         string `json:"wave_id"`
	PlanRevision   int    `json:"plan_revision"`
	StepID         string `json:"step_id"`
	PlanPath       string `json:"plan_path"`
	RegistrySHA256 string `json:"registry_sha256"`
	PlanSHA256     string `json:"plan_sha256"`
}

type ChangeIntent struct {
	Reason      string `json:"reason"`
	Replacement *Task  `json:"replacement,omitempty"`
}

type Task struct {
	SchemaVersion       int                         `json:"schema_version"`
	ID                  string                      `json:"id"`
	TeamID              string                      `json:"team_id,omitempty"`
	ProjectID           string                      `json:"project_id"`
	RepositoryID        string                      `json:"repository_id,omitempty"`
	Module              string                      `json:"module,omitempty"`
	AssigneeID          string                      `json:"assignee_id,omitempty"`
	ParentTaskID        string                      `json:"parent_task_id,omitempty"`
	IssueIDs            []string                    `json:"issue_ids,omitempty"`
	SpecRefs            []string                    `json:"spec_refs,omitempty"`
	DocumentRefs        []string                    `json:"document_refs,omitempty"`
	PolicyBundleHash    string                      `json:"policy_bundle_hash,omitempty"`
	PolicyInstructions  []string                    `json:"policy_instructions,omitempty"`
	CorrelationID       string                      `json:"correlation_id,omitempty"`
	CreateRequestSHA256 string                      `json:"create_request_sha256,omitempty"`
	Wave                *WaveBinding                `json:"wave,omitempty"`
	Title               string                      `json:"title"`
	Status              TaskStatus                  `json:"status"`
	Request             RequestFrame                `json:"request"`
	Goal                GoalSpec                    `json:"goal"`
	Plan                PlanSpec                    `json:"plan"`
	EvidencePlan        EvidencePlan                `json:"evidence_plan"`
	Scope               ScopePolicy                 `json:"scope"`
	Risk                RiskPlan                    `json:"risk"`
	Cost                CostPlan                    `json:"cost"`
	DoneGate            DoneGateSpec                `json:"done_gate"`
	Reviews             map[ReviewKind]ReviewRecord `json:"reviews"`
	Compile             CompileRecord               `json:"compile"`
	RepoPath            string                      `json:"repo_path"`
	WorktreePath        string                      `json:"worktree_path,omitempty"`
	Branch              string                      `json:"branch,omitempty"`
	CurrentRunID        string                      `json:"current_run_id,omitempty"`
	CodexThreadID       string                      `json:"codex_thread_id,omitempty"`
	RepairCount         int                         `json:"repair_count"`
	RunCount            int                         `json:"run_count"`
	CumulativeUsage     CodexUsage                  `json:"cumulative_usage,omitempty"`
	LastGate            *DoneGateResult             `json:"last_gate,omitempty"`
	LastEvidence        string                      `json:"last_evidence,omitempty"`
	CommitSHA           string                      `json:"commit_sha,omitempty"`
	PullRequestURL      string                      `json:"pull_request_url,omitempty"`
	CreatedBy           string                      `json:"created_by"`
	RequestedBy         string                      `json:"requested_by,omitempty"`
	AcceptedBy          *governance.DecisionRecord  `json:"accepted_by,omitempty"`
	CreatedAt           time.Time                   `json:"created_at"`
	UpdatedAt           time.Time                   `json:"updated_at"`
}

type CreateRequest struct {
	ID                 string       `json:"id,omitempty"`
	TeamID             string       `json:"team_id,omitempty"`
	ProjectID          string       `json:"project_id"`
	RepositoryID       string       `json:"repository_id,omitempty"`
	Module             string       `json:"module,omitempty"`
	AssigneeID         string       `json:"assignee_id,omitempty"`
	ParentTaskID       string       `json:"parent_task_id,omitempty"`
	IssueIDs           []string     `json:"issue_ids,omitempty"`
	SpecRefs           []string     `json:"spec_refs,omitempty"`
	DocumentRefs       []string     `json:"document_refs,omitempty"`
	PolicyBundleHash   string       `json:"policy_bundle_hash,omitempty"`
	PolicyInstructions []string     `json:"policy_instructions,omitempty"`
	CorrelationID      string       `json:"correlation_id,omitempty"`
	Wave               *WaveBinding `json:"wave,omitempty"`
	Title              string       `json:"title"`
	RepoPath           string       `json:"repo_path"`
	BaseRef            string       `json:"base_ref,omitempty"`
	Request            RequestFrame `json:"request"`
	Goal               GoalSpec     `json:"goal"`
	Plan               PlanSpec     `json:"plan"`
	EvidencePlan       EvidencePlan `json:"evidence_plan"`
	Scope              ScopePolicy  `json:"scope"`
	Risk               RiskPlan     `json:"risk"`
	Cost               CostPlan     `json:"cost"`
	DoneGate           DoneGateSpec `json:"done_gate"`
	CreatedBy          string       `json:"created_by"`
	RequestedBy        string       `json:"requested_by,omitempty"`
}

type SessionEvent struct {
	SchemaVersion int             `json:"schema_version"`
	ID            string          `json:"id"`
	TaskID        string          `json:"task_id"`
	Sequence      int64           `json:"sequence"`
	Type          string          `json:"type"`
	Actor         string          `json:"actor"`
	CreatedAt     time.Time       `json:"created_at"`
	PreviousHash  string          `json:"previous_hash,omitempty"`
	Data          json.RawMessage `json:"data,omitempty"`
	Snapshot      Task            `json:"snapshot"`
	Hash          string          `json:"hash"`
}

type RepositorySnapshot struct {
	RepoPath   string    `json:"repo_path"`
	Worktree   string    `json:"worktree"`
	Branch     string    `json:"branch"`
	BaseCommit string    `json:"base_commit"`
	HeadCommit string    `json:"head_commit"`
	Status     string    `json:"status,omitempty"`
	CapturedAt time.Time `json:"captured_at"`
}

type CodexUsage struct {
	InputTokens           int `json:"input_tokens,omitempty"`
	CachedInputTokens     int `json:"cached_input_tokens,omitempty"`
	OutputTokens          int `json:"output_tokens,omitempty"`
	ReasoningOutputTokens int `json:"reasoning_output_tokens,omitempty"`
}

type HandResult struct {
	ThreadID  string     `json:"thread_id,omitempty"`
	FinalText string     `json:"final_text,omitempty"`
	Usage     CodexUsage `json:"usage,omitempty"`
	ExitCode  int        `json:"exit_code"`
	Stdout    string     `json:"stdout,omitempty"`
	Stderr    string     `json:"stderr,omitempty"`
}

type VerificationResult struct {
	Name       string   `json:"name"`
	Argv       []string `json:"argv"`
	ExitCode   int      `json:"exit_code"`
	Stdout     string   `json:"stdout,omitempty"`
	Stderr     string   `json:"stderr,omitempty"`
	DurationMS int64    `json:"duration_ms"`
	TimedOut   bool     `json:"timed_out,omitempty"`
	Passed     bool     `json:"passed"`
}

type FalsifierResult struct {
	CriterionID      string   `json:"criterion_id"`
	Condition        string   `json:"condition"`
	EvidenceRequired string   `json:"evidence_required"`
	Checked          bool     `json:"checked"`
	Triggered        bool     `json:"triggered"`
	EvidenceRefs     []string `json:"evidence_refs,omitempty"`
	Reason           string   `json:"reason"`
}

type PredictionCheck struct {
	PredictionID string   `json:"prediction_id"`
	Horizon      string   `json:"horizon"`
	Due          bool     `json:"due"`
	Checked      bool     `json:"checked"`
	Satisfied    bool     `json:"satisfied"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
	Observation  string   `json:"observation"`
}

type KillConditionCheck struct {
	ConditionID string  `json:"condition_id"`
	Metric      string  `json:"metric"`
	Observed    float64 `json:"observed"`
	Threshold   float64 `json:"threshold"`
	Evaluated   bool    `json:"evaluated"`
	Triggered   bool    `json:"triggered"`
	Action      string  `json:"action"`
	Reason      string  `json:"reason"`
}

type PolicyResult struct {
	Passed          bool     `json:"passed"`
	ChangedFiles    []string `json:"changed_files"`
	AddedLines      int      `json:"added_lines"`
	DeletedLines    int      `json:"deleted_lines"`
	NewDependencies []string `json:"new_dependencies,omitempty"`
	Violations      []string `json:"violations,omitempty"`
}

type IndependentReview struct {
	Passed      bool     `json:"passed"`
	Summary     string   `json:"summary"`
	Findings    []string `json:"findings,omitempty"`
	RequiredFix []string `json:"required_fixes,omitempty"`
	ThreadID    string   `json:"thread_id,omitempty"`
}

type DoneGateResult struct {
	Passed          bool      `json:"passed"`
	Verdict         string    `json:"verdict"`
	Reasons         []string  `json:"reasons,omitempty"`
	EvidencePath    string    `json:"evidence_path"`
	EvidenceSHA256  string    `json:"evidence_sha256,omitempty"`
	WorktreeTreeSHA string    `json:"worktree_tree_sha,omitempty"`
	EvaluatedAt     time.Time `json:"evaluated_at"`
	EvaluatedBy     string    `json:"evaluated_by"`
}

type EvidencePackage struct {
	SchemaVersion    int                        `json:"schema_version"`
	ProjectID        string                     `json:"project_id,omitempty"`
	RepositoryID     string                     `json:"repository_id,omitempty"`
	TaskID           string                     `json:"task_id"`
	WorkItemIDs      []string                   `json:"work_item_ids,omitempty"`
	IssueIDs         []string                   `json:"issue_ids,omitempty"`
	RunID            string                     `json:"run_id"`
	CorrelationID    string                     `json:"correlation_id,omitempty"`
	PolicyBundleHash string                     `json:"policy_bundle_hash,omitempty"`
	DocumentRefs     []string                   `json:"document_refs,omitempty"`
	TaskRevision     int                        `json:"task_revision"`
	Before           RepositorySnapshot         `json:"before"`
	After            RepositorySnapshot         `json:"after"`
	Hand             HandResult                 `json:"hand"`
	Policy           PolicyResult               `json:"policy"`
	Verification     []VerificationResult       `json:"verification"`
	Review           IndependentReview          `json:"independent_review"`
	Attribution      []ChangeAttribution        `json:"change_attribution,omitempty"`
	Unattributed     []string                   `json:"unattributed_files,omitempty"`
	Falsifiers       []FalsifierResult          `json:"falsifier_results,omitempty"`
	Predictions      []PredictionCheck          `json:"prediction_checks,omitempty"`
	KillChecks       []KillConditionCheck       `json:"kill_condition_checks,omitempty"`
	DiffPath         string                     `json:"diff_path"`
	Imported         *ImportedExecutionEvidence `json:"imported_execution,omitempty"`
	CreatedAt        time.Time                  `json:"created_at"`
}

type ChangeAttribution struct {
	WorkItemID string   `json:"work_item_id"`
	IssueIDs   []string `json:"issue_ids,omitempty"`
	Files      []string `json:"files"`
}

// ImportedEvidenceCheck is a deterministic check performed by an external
// execution runtime. The import path requires every frozen verification
// command to have a corresponding check; the DoneGate still decides whether
// failed checks can advance the task.
type ImportedEvidenceCheck struct {
	Name       string   `json:"name"`
	Argv       []string `json:"argv,omitempty"`
	Passed     bool     `json:"passed"`
	ExitCode   int      `json:"exit_code,omitempty"`
	Stdout     string   `json:"stdout,omitempty"`
	Stderr     string   `json:"stderr,omitempty"`
	Details    string   `json:"details,omitempty"`
	DurationMS int64    `json:"duration_ms,omitempty"`
	TimedOut   bool     `json:"timed_out,omitempty"`
	Artifacts  []string `json:"artifacts,omitempty"`
}

// ImportedEvidenceArtifact is a content-addressed artifact retained by the
// external runtime. URI is informational; SHA256 is the durable identity.
type ImportedEvidenceArtifact struct {
	Name      string `json:"name"`
	URI       string `json:"uri,omitempty"`
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"size_bytes,omitempty"`
}

// ImportedExecutionEvidence captures the signed execution receipt stored in
// an EvidencePackage. Signature verification happens at the execution-runtime
// trust boundary; VerifiedAt and VerifiedBy record that control-plane decision.
type ImportedExecutionEvidence struct {
	Source              string                     `json:"source"`
	ExecutionPackSHA256 string                     `json:"execution_pack_sha256"`
	RunnerID            string                     `json:"runner_id"`
	LeaseID             string                     `json:"lease_id"`
	Attempt             int                        `json:"attempt"`
	Outcome             string                     `json:"outcome"`
	Summary             string                     `json:"summary,omitempty"`
	StartedAt           time.Time                  `json:"started_at"`
	FinishedAt          time.Time                  `json:"finished_at"`
	BaseCommit          string                     `json:"base_commit"`
	HeadCommit          string                     `json:"head_commit"`
	CommitSHA           string                     `json:"commit_sha,omitempty"`
	Branch              string                     `json:"branch,omitempty"`
	ChangedFiles        []string                   `json:"changed_files,omitempty"`
	DiffSHA256          string                     `json:"diff_sha256,omitempty"`
	Checks              []ImportedEvidenceCheck    `json:"checks"`
	Artifacts           []ImportedEvidenceArtifact `json:"artifacts,omitempty"`
	TraceIDs            []string                   `json:"trace_ids,omitempty"`
	KeyID               string                     `json:"key_id"`
	SignatureAlgorithm  string                     `json:"signature_algorithm"`
	BundleSHA256        string                     `json:"bundle_sha256"`
	Signature           string                     `json:"signature"`
	VerifiedAt          time.Time                  `json:"verified_at"`
	VerifiedBy          string                     `json:"verified_by"`
}

// ImportExecutionEvidenceInput is intentionally independent of workstation
// package types. Adapters must map a previously verified signed bundle into
// this frozen-task contract.
type ImportExecutionEvidenceInput struct {
	TaskID              string                    `json:"task_id"`
	ProjectID           string                    `json:"project_id"`
	TaskRevision        int                       `json:"task_revision"`
	ExecutionBundleHash string                    `json:"execution_bundle_hash"`
	Evidence            ImportedExecutionEvidence `json:"evidence"`
	DiffPatch           string                    `json:"diff_patch,omitempty"`
}
