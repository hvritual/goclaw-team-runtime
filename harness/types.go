package harness

import (
	"time"

	"github.com/smallnest/goclaw/governance"
)

const SchemaVersion = 1

// Config controls the Better-Harness runtime. Runtime state must live outside
// the governed knowledge root so active JSONL files and locks are never
// synchronized or committed with human knowledge assets.
type Config struct {
	Enabled          bool   `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	Root             string `mapstructure:"root" json:"root" yaml:"root"`
	ProjectID        string `mapstructure:"project_id" json:"project_id" yaml:"project_id"`
	KnowledgeRoot    string `mapstructure:"knowledge_root" json:"knowledge_root,omitempty" yaml:"knowledge_root,omitempty"`
	KnowledgeBackend string `mapstructure:"knowledge_backend" json:"knowledge_backend,omitempty" yaml:"knowledge_backend,omitempty"`
	// VaultPath is a deprecated compatibility alias for KnowledgeRoot.
	VaultPath     string `mapstructure:"vault_path" json:"vault_path,omitempty" yaml:"vault_path,omitempty"`
	TraceEnabled  bool   `mapstructure:"trace_enabled" json:"trace_enabled" yaml:"trace_enabled"`
	ActiveVersion string `mapstructure:"active_version" json:"active_version" yaml:"active_version"`
	// Routes maps channel/chat identities to a project. Supported keys, from
	// most to least specific: channel:account:chat, channel:chat, channel.
	Routes map[string]string `mapstructure:"routes" json:"routes,omitempty" yaml:"routes,omitempty"`
}

// Manifest is an immutable, versioned description of a harness.
type Manifest struct {
	SchemaVersion   int               `json:"schema_version" yaml:"schema_version"`
	Version         string            `json:"version" yaml:"version"`
	Name            string            `json:"name" yaml:"name"`
	Description     string            `json:"description,omitempty" yaml:"description,omitempty"`
	ModelProfile    string            `json:"model_profile,omitempty" yaml:"model_profile,omitempty"`
	ProjectID       string            `json:"project_id,omitempty" yaml:"project_id,omitempty"`
	Components      map[string]string `json:"components" yaml:"components"`
	ProtectedPaths  []string          `json:"protected_paths,omitempty" yaml:"protected_paths,omitempty"`
	CreatedAt       time.Time         `json:"created_at" yaml:"created_at"`
	CreatedBy       string            `json:"created_by,omitempty" yaml:"created_by,omitempty"`
	ParentVersion   string            `json:"parent_version,omitempty" yaml:"parent_version,omitempty"`
	ChangeManifest  string            `json:"change_manifest,omitempty" yaml:"change_manifest,omitempty"`
	MinimumGolden   float64           `json:"minimum_golden" yaml:"minimum_golden"`
	MinimumHoldout  float64           `json:"minimum_holdout" yaml:"minimum_holdout"`
	MaxTokenDelta   float64           `json:"max_token_delta" yaml:"max_token_delta"`
	MaxLatencyDelta float64           `json:"max_latency_delta" yaml:"max_latency_delta"`
}

// ActiveState points at the production harness and its immediate predecessor.
type ActiveState struct {
	Version         string                     `json:"version"`
	PreviousVersion string                     `json:"previous_version,omitempty"`
	ActivatedAt     time.Time                  `json:"activated_at"`
	ActivatedBy     string                     `json:"activated_by"`
	ExperimentID    string                     `json:"experiment_id,omitempty"`
	Decision        *governance.DecisionRecord `json:"decision,omitempty"`
}

// Trace is the durable envelope written for every GoClaw run.
type Trace struct {
	SchemaVersion    int             `json:"schema_version"`
	ID               string          `json:"id"`
	HarnessVersion   string          `json:"harness_version"`
	ProjectID        string          `json:"project_id"`
	RepositoryID     string          `json:"repository_id,omitempty"`
	TaskID           string          `json:"task_id,omitempty"`
	WorkItemID       string          `json:"work_item_id,omitempty"`
	IssueID          string          `json:"issue_id,omitempty"`
	RunID            string          `json:"run_id,omitempty"`
	CorrelationID    string          `json:"correlation_id,omitempty"`
	CommitSHA        string          `json:"commit_sha,omitempty"`
	PullRequestURL   string          `json:"pull_request_url,omitempty"`
	PolicyBundleHash string          `json:"policy_bundle_hash,omitempty"`
	DocumentRefs     []string        `json:"document_refs,omitempty"`
	TopicID          string          `json:"topic_id"`
	ConversationID   string          `json:"conversation_id"`
	SessionID        string          `json:"session_id"`
	MessageID        string          `json:"message_id,omitempty"`
	Channel          string          `json:"channel"`
	AccountID        string          `json:"account_id,omitempty"`
	ActorID          string          `json:"actor_id,omitempty"`
	Model            string          `json:"model,omitempty"`
	Status           string          `json:"status"`
	Input            string          `json:"input,omitempty"`
	Output           string          `json:"output,omitempty"`
	Error            string          `json:"error,omitempty"`
	ToolCalls        []ToolCallTrace `json:"tool_calls,omitempty"`
	Context          ContextManifest `json:"context_manifest"`
	Metadata         map[string]any  `json:"metadata,omitempty"`
	StartedAt        time.Time       `json:"started_at"`
	FinishedAt       time.Time       `json:"finished_at"`
	DurationMS       int64           `json:"duration_ms"`
	TokenUsage       TokenUsage      `json:"token_usage,omitempty"`
	HumanFeedback    *HumanFeedback  `json:"human_feedback,omitempty"`
}

type ToolCallTrace struct {
	ID      string         `json:"id,omitempty"`
	Name    string         `json:"name"`
	Params  map[string]any `json:"params,omitempty"`
	Status  string         `json:"status,omitempty"`
	Result  string         `json:"result,omitempty"`
	Error   string         `json:"error,omitempty"`
	Details map[string]any `json:"details,omitempty"`
}

type ContextManifest struct {
	KnowledgeRevision string   `json:"knowledge_revision,omitempty"`
	LoadedFiles       []string `json:"loaded_files,omitempty"`
	MemoryIDs         []string `json:"memory_ids,omitempty"`
	RecentMessages    int      `json:"recent_messages"`
	EstimatedTokens   int      `json:"estimated_tokens,omitempty"`
}

type TokenUsage struct {
	Input  int `json:"input,omitempty"`
	Output int `json:"output,omitempty"`
	Total  int `json:"total,omitempty"`
}

type HumanFeedback struct {
	Rating    string    `json:"rating"`
	Comment   string    `json:"comment,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type KnowledgeProposalStatus string

const (
	KnowledgeProposalPending  KnowledgeProposalStatus = "pending"
	KnowledgeProposalApproved KnowledgeProposalStatus = "approved"
	KnowledgeProposalRejected KnowledgeProposalStatus = "rejected"
)

// KnowledgeProposal is an approval-gated full-content change to a governed
// Markdown file. BaseSHA256 protects the file while BaseRevision protects the
// containing Git-backed knowledge snapshot when configured.
type KnowledgeProposal struct {
	SchemaVersion   int                        `json:"schema_version"`
	ID              string                     `json:"id"`
	ProjectID       string                     `json:"project_id"`
	TargetPath      string                     `json:"target_path"`
	BaseSHA256      string                     `json:"base_sha256,omitempty"`
	BaseRevision    string                     `json:"base_revision,omitempty"`
	SourceURI       string                     `json:"source_uri,omitempty"`
	StoreKind       string                     `json:"store_kind,omitempty"`
	ProposedContent string                     `json:"proposed_content"`
	Reason          string                     `json:"reason"`
	EvidenceTraceID string                     `json:"evidence_trace_id,omitempty"`
	Status          KnowledgeProposalStatus    `json:"status"`
	CreatedBy       string                     `json:"created_by"`
	CreatedAt       time.Time                  `json:"created_at"`
	ReviewedBy      string                     `json:"reviewed_by,omitempty"`
	ReviewComment   string                     `json:"review_comment,omitempty"`
	ReviewedAt      *time.Time                 `json:"reviewed_at,omitempty"`
	Review          *governance.DecisionRecord `json:"review,omitempty"`
}

type KnowledgeSearchResult struct {
	Path    string `json:"path"`
	Excerpt string `json:"excerpt"`
}

type EvalSplit string

const (
	SplitOptimization EvalSplit = "optimization"
	SplitHoldout      EvalSplit = "holdout"
	SplitGolden       EvalSplit = "golden"
)

// EvalCase is intentionally model-independent. A runner may execute an actual
// GoClaw command, or the case may grade a previously captured trace fixture.
type EvalCase struct {
	SchemaVersion int               `json:"schema_version" yaml:"schema_version"`
	ID            string            `json:"id" yaml:"id"`
	Description   string            `json:"description" yaml:"description"`
	Tags          []string          `json:"tags" yaml:"tags"`
	Split         EvalSplit         `json:"split" yaml:"split"`
	Critical      bool              `json:"critical" yaml:"critical"`
	Input         map[string]any    `json:"input,omitempty" yaml:"input,omitempty"`
	Runner        RunnerSpec        `json:"runner,omitempty" yaml:"runner,omitempty"`
	TraceFixture  string            `json:"trace_fixture,omitempty" yaml:"trace_fixture,omitempty"`
	Expected      ExpectedBehavior  `json:"expected" yaml:"expected"`
	Metadata      map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

type RunnerSpec struct {
	Command        []string          `json:"command,omitempty" yaml:"command,omitempty"`
	WorkingDir     string            `json:"working_dir,omitempty" yaml:"working_dir,omitempty"`
	Environment    map[string]string `json:"environment,omitempty" yaml:"environment,omitempty"`
	TimeoutSeconds int               `json:"timeout_seconds,omitempty" yaml:"timeout_seconds,omitempty"`
}

type ExpectedBehavior struct {
	ExitCode             *int     `json:"exit_code,omitempty" yaml:"exit_code,omitempty"`
	OutputContains       []string `json:"output_contains,omitempty" yaml:"output_contains,omitempty"`
	OutputNotContains    []string `json:"output_not_contains,omitempty" yaml:"output_not_contains,omitempty"`
	ExpectedProjectID    string   `json:"expected_project_id,omitempty" yaml:"expected_project_id,omitempty"`
	RequiredContextFiles []string `json:"required_context_files,omitempty" yaml:"required_context_files,omitempty"`
	RequiredToolCalls    []string `json:"required_tool_calls,omitempty" yaml:"required_tool_calls,omitempty"`
	ForbiddenToolCalls   []string `json:"forbidden_tool_calls,omitempty" yaml:"forbidden_tool_calls,omitempty"`
	ForbiddenWrites      []string `json:"forbidden_writes,omitempty" yaml:"forbidden_writes,omitempty"`
	RequiredWrites       []string `json:"required_writes,omitempty" yaml:"required_writes,omitempty"`
	RequireEvidence      bool     `json:"require_evidence,omitempty" yaml:"require_evidence,omitempty"`
	MaxDurationMS        int64    `json:"max_duration_ms,omitempty" yaml:"max_duration_ms,omitempty"`
	MaxTokens            int      `json:"max_tokens,omitempty" yaml:"max_tokens,omitempty"`
	MaxToolCalls         int      `json:"max_tool_calls,omitempty" yaml:"max_tool_calls,omitempty"`
}

type EvalResult struct {
	CaseID           string        `json:"case_id"`
	Split            EvalSplit     `json:"split"`
	Tags             []string      `json:"tags"`
	Critical         bool          `json:"critical"`
	Passed           bool          `json:"passed"`
	Failures         []string      `json:"failures,omitempty"`
	ExitCode         int           `json:"exit_code"`
	Stdout           string        `json:"stdout,omitempty"`
	Stderr           string        `json:"stderr,omitempty"`
	ChangedFiles     []string      `json:"changed_files,omitempty"`
	Trace            *Trace        `json:"trace,omitempty"`
	Duration         time.Duration `json:"duration"`
	MetricTokens     int           `json:"metric_tokens,omitempty"`
	MetricDurationMS int64         `json:"metric_duration_ms,omitempty"`
	HarnessVersion   string        `json:"harness_version"`
}

type SplitScore struct {
	Total    int     `json:"total"`
	Passed   int     `json:"passed"`
	Critical int     `json:"critical_failures"`
	Rate     float64 `json:"rate"`
}

type ValidationReport struct {
	SchemaVersion       int                      `json:"schema_version"`
	ExperimentID        string                   `json:"experiment_id"`
	BaselineVersion     string                   `json:"baseline_version"`
	HarnessVersion      string                   `json:"harness_version"`
	CreatedAt           time.Time                `json:"created_at"`
	BaselineResults     []EvalResult             `json:"baseline_results"`
	Results             []EvalResult             `json:"results"`
	BaselineScores      map[EvalSplit]SplitScore `json:"baseline_scores"`
	Scores              map[EvalSplit]SplitScore `json:"scores"`
	CandidateChanges    []string                 `json:"candidate_changes,omitempty"`
	Regressions         []EvalRegression         `json:"regressions,omitempty"`
	BaselineTokens      int                      `json:"baseline_tokens,omitempty"`
	CandidateTokens     int                      `json:"candidate_tokens,omitempty"`
	TokenDelta          float64                  `json:"token_delta,omitempty"`
	BaselineDurationMS  int64                    `json:"baseline_duration_ms,omitempty"`
	CandidateDurationMS int64                    `json:"candidate_duration_ms,omitempty"`
	LatencyDelta        float64                  `json:"latency_delta,omitempty"`
	Accepted            bool                     `json:"accepted"`
	Rejection           []string                 `json:"rejection_reasons,omitempty"`
}

type EvalRegression struct {
	CaseID            string   `json:"case_id"`
	BaselinePassed    bool     `json:"baseline_passed"`
	CandidatePassed   bool     `json:"candidate_passed"`
	CandidateFailures []string `json:"candidate_failures,omitempty"`
}

type ExperimentStatus string

const (
	ExperimentDraft         ExperimentStatus = "draft"
	ExperimentValidated     ExperimentStatus = "validated"
	ExperimentHumanApproved ExperimentStatus = "human_approved"
	ExperimentActive        ExperimentStatus = "active"
	ExperimentRejected      ExperimentStatus = "rejected"
	ExperimentRolledBack    ExperimentStatus = "rolled_back"
)

type Experiment struct {
	SchemaVersion    int                         `json:"schema_version"`
	ID               string                      `json:"id"`
	BaseVersion      string                      `json:"base_version"`
	CandidateVersion string                      `json:"candidate_version"`
	CandidatePath    string                      `json:"candidate_path"`
	Status           ExperimentStatus            `json:"status"`
	ChangeManifest   ChangeManifest              `json:"change_manifest"`
	ValidationReport string                      `json:"validation_report,omitempty"`
	CreatedAt        time.Time                   `json:"created_at"`
	UpdatedAt        time.Time                   `json:"updated_at"`
	ReviewedBy       string                      `json:"reviewed_by,omitempty"`
	ReviewComment    string                      `json:"review_comment,omitempty"`
	CreatedBy        string                      `json:"created_by"`
	Approvals        []governance.DecisionRecord `json:"approvals,omitempty"`
	Promotion        *governance.DecisionRecord  `json:"promotion,omitempty"`
	Rollback         *governance.DecisionRecord  `json:"rollback,omitempty"`
}

type ChangeManifest struct {
	TargetComponents    []string `json:"target_components"`
	EvidenceTraceIDs    []string `json:"evidence_trace_ids,omitempty"`
	RootCause           string   `json:"root_cause"`
	ChangeSummary       string   `json:"change_summary"`
	ExpectedFixTags     []string `json:"expected_fix_tags,omitempty"`
	PossibleRegressions []string `json:"possible_regressions,omitempty"`
}
