package workstation

import (
	"encoding/json"
	"time"
)

const SchemaVersion = 1

// Config controls the single-writer workstation queue. Root must be located on
// the control-plane host, not inside a synchronized Vault or shared worktree.
type Config struct {
	Enabled                bool   `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	Root                   string `mapstructure:"root" json:"root" yaml:"root"`
	LeaseDurationSeconds   int    `mapstructure:"lease_duration_seconds" json:"lease_duration_seconds" yaml:"lease_duration_seconds"`
	RunnerOfflineSeconds   int    `mapstructure:"runner_offline_seconds" json:"runner_offline_seconds" yaml:"runner_offline_seconds"`
	DefaultMaxAttempts     int    `mapstructure:"default_max_attempts" json:"default_max_attempts" yaml:"default_max_attempts"`
	MaxIdempotencyReceipts int    `mapstructure:"max_idempotency_receipts" json:"max_idempotency_receipts" yaml:"max_idempotency_receipts"`
}

type RunnerStatus string

const (
	RunnerOnline   RunnerStatus = "online"
	RunnerOffline  RunnerStatus = "offline"
	RunnerDisabled RunnerStatus = "disabled"
)

// Runner is the public workstation registration. Device keys are deliberately
// absent: the control plane stores them in a separate 0600 credential file.
type Runner struct {
	SchemaVersion   int               `json:"schema_version"`
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	OwnerUserID     string            `json:"owner_user_id"`
	Status          RunnerStatus      `json:"status"`
	Capabilities    []string          `json:"capabilities"`
	Projects        []string          `json:"projects"`
	KeyID           string            `json:"key_id"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	RegisteredAt    time.Time         `json:"registered_at"`
	LastHeartbeatAt time.Time         `json:"last_heartbeat_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

type RegisterRunnerRequest struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	OwnerUserID  string            `json:"owner_user_id"`
	Capabilities []string          `json:"capabilities"`
	Projects     []string          `json:"projects"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

type UpdateRunnerRequest struct {
	Name         string            `json:"name,omitempty"`
	Capabilities []string          `json:"capabilities,omitempty"`
	Projects     []string          `json:"projects,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	Disabled     *bool             `json:"disabled,omitempty"`
}

type TaskStatus string

const (
	TaskQueued    TaskStatus = "queued"
	TaskLeased    TaskStatus = "leased"
	TaskCompleted TaskStatus = "completed"
	TaskFailed    TaskStatus = "failed"
	TaskCancelled TaskStatus = "cancelled"
)

// CommandSpec contains only deterministic argv. Runtime credentials and secret
// environment values are resolved locally by the authorized runner.
type CommandSpec struct {
	Name string   `json:"name"`
	Argv []string `json:"argv"`
}

// ExecutionPack is the immutable, secret-free contract delivered to a runner.
// Its canonical JSON is SHA-256 hashed when the task is enqueued.
type ExecutionPack struct {
	SchemaVersion     int               `json:"schema_version"`
	ExecutionProfile  ExecutionProfile  `json:"execution_profile,omitempty"`
	TaskID            string            `json:"task_id"`
	TaskRevision      int               `json:"task_revision"`
	ProjectID         string            `json:"project_id"`
	CorrelationID     string            `json:"correlation_id,omitempty"`
	IssueIDs          []string          `json:"issue_ids,omitempty"`
	SpecHash          string            `json:"spec_hash,omitempty"`
	WorkItemIDs       []string          `json:"work_item_ids,omitempty"`
	RepositoryID      string            `json:"repository_id"`
	RepositoryURL     string            `json:"repository_url,omitempty"`
	BaseRef           string            `json:"base_ref,omitempty"`
	BaseCommit        string            `json:"base_commit"`
	Branch            string            `json:"branch,omitempty"`
	Prompt            string            `json:"prompt"`
	AllowedPaths      []string          `json:"allowed_paths,omitempty"`
	DeniedPaths       []string          `json:"denied_paths,omitempty"`
	Verification      []CommandSpec     `json:"verification"`
	HarnessVersion    string            `json:"harness_version,omitempty"`
	PolicyPackVersion string            `json:"policy_pack_version,omitempty"`
	PolicyBundleHash  string            `json:"policy_bundle_hash,omitempty"`
	Metadata          map[string]string `json:"metadata,omitempty"`
	Payload           json.RawMessage   `json:"payload,omitempty"`
}

type EnqueueRequest struct {
	ID                   string        `json:"id,omitempty"`
	IdempotencyKey       string        `json:"idempotency_key"`
	ProjectID            string        `json:"project_id"`
	Priority             int           `json:"priority,omitempty"`
	RequiredCapabilities []string      `json:"required_capabilities"`
	MaxAttempts          int           `json:"max_attempts,omitempty"`
	ExecutionPack        ExecutionPack `json:"execution_pack"`
}

type Lease struct {
	ID             string    `json:"id"`
	RunnerID       string    `json:"runner_id"`
	Attempt        int       `json:"attempt"`
	ClaimedAt      time.Time `json:"claimed_at"`
	HeartbeatAt    time.Time `json:"heartbeat_at"`
	ExpiresAt      time.Time `json:"expires_at"`
	IdempotencyKey string    `json:"idempotency_key"`
}

type EvidenceCheck struct {
	Name       string   `json:"name"`
	Passed     bool     `json:"passed"`
	ExitCode   int      `json:"exit_code,omitempty"`
	DurationMS int64    `json:"duration_ms,omitempty"`
	Details    string   `json:"details,omitempty"`
	Artifacts  []string `json:"artifacts,omitempty"`
}

type EvidenceArtifact struct {
	Name      string `json:"name"`
	URI       string `json:"uri,omitempty"`
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"size_bytes,omitempty"`
}

// EvidenceBundle is signed on the workstation using its device key. Signature
// is HMAC-SHA256 over canonical JSON with BundleSHA256 and Signature cleared.
type EvidenceBundle struct {
	SchemaVersion       int                `json:"schema_version"`
	TaskID              string             `json:"task_id"`
	ProjectID           string             `json:"project_id"`
	ExecutionPackSHA256 string             `json:"execution_pack_sha256"`
	RunnerID            string             `json:"runner_id"`
	LeaseID             string             `json:"lease_id"`
	Attempt             int                `json:"attempt"`
	Outcome             string             `json:"outcome"`
	StartedAt           time.Time          `json:"started_at"`
	FinishedAt          time.Time          `json:"finished_at"`
	BaseCommit          string             `json:"base_commit"`
	HeadCommit          string             `json:"head_commit,omitempty"`
	CommitSHA           string             `json:"commit_sha,omitempty"`
	Branch              string             `json:"branch,omitempty"`
	ChangedFiles        []string           `json:"changed_files,omitempty"`
	DiffSHA256          string             `json:"diff_sha256,omitempty"`
	DiffPatch           string             `json:"diff_patch,omitempty"`
	Checks              []EvidenceCheck    `json:"checks,omitempty"`
	Artifacts           []EvidenceArtifact `json:"artifacts,omitempty"`
	TraceIDs            []string           `json:"trace_ids,omitempty"`
	Metadata            map[string]string  `json:"metadata,omitempty"`
	KeyID               string             `json:"key_id"`
	SignatureAlgorithm  string             `json:"signature_algorithm"`
	BundleSHA256        string             `json:"bundle_sha256"`
	Signature           string             `json:"signature"`
}

type EvidenceReference struct {
	BundleSHA256       string    `json:"bundle_sha256"`
	Signature          string    `json:"signature"`
	KeyID              string    `json:"key_id"`
	SignatureAlgorithm string    `json:"signature_algorithm"`
	Path               string    `json:"path"`
	VerifiedAt         time.Time `json:"verified_at"`
}

type TaskResult struct {
	RunnerID    string            `json:"runner_id"`
	LeaseID     string            `json:"lease_id"`
	Attempt     int               `json:"attempt"`
	Summary     string            `json:"summary,omitempty"`
	Evidence    EvidenceReference `json:"evidence"`
	CompletedAt time.Time         `json:"completed_at"`
}

type TaskFailure struct {
	RunnerID string             `json:"runner_id"`
	LeaseID  string             `json:"lease_id"`
	Attempt  int                `json:"attempt"`
	Error    string             `json:"error"`
	Evidence *EvidenceReference `json:"evidence,omitempty"`
	FailedAt time.Time          `json:"failed_at"`
}

type IdempotencyReceipt struct {
	Operation     string     `json:"operation"`
	Key           string     `json:"key"`
	RequestSHA256 string     `json:"request_sha256"`
	ResultStatus  TaskStatus `json:"result_status"`
	Lease         *Lease     `json:"lease,omitempty"`
	RecordedAt    time.Time  `json:"recorded_at"`
}

type TaskEvent struct {
	Type      string    `json:"type"`
	Actor     string    `json:"actor"`
	Attempt   int       `json:"attempt,omitempty"`
	Message   string    `json:"message,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Task is the durable queue record. It contains hashes and signed evidence
// references, never the workstation device key.
type Task struct {
	SchemaVersion        int                  `json:"schema_version"`
	ID                   string               `json:"id"`
	ProjectID            string               `json:"project_id"`
	IdempotencyKey       string               `json:"idempotency_key"`
	RequestSHA256        string               `json:"request_sha256"`
	Status               TaskStatus           `json:"status"`
	Priority             int                  `json:"priority"`
	RequiredCapabilities []string             `json:"required_capabilities"`
	ExecutionPack        ExecutionPack        `json:"execution_pack"`
	ExecutionPackSHA256  string               `json:"execution_pack_sha256"`
	Attempt              int                  `json:"attempt"`
	MaxAttempts          int                  `json:"max_attempts"`
	Lease                *Lease               `json:"lease,omitempty"`
	Result               *TaskResult          `json:"result,omitempty"`
	LastFailure          *TaskFailure         `json:"last_failure,omitempty"`
	Receipts             []IdempotencyReceipt `json:"idempotency_receipts,omitempty"`
	History              []TaskEvent          `json:"history,omitempty"`
	CreatedAt            time.Time            `json:"created_at"`
	UpdatedAt            time.Time            `json:"updated_at"`
	CompletedAt          *time.Time           `json:"completed_at,omitempty"`
}

type ClaimRequest struct {
	RunnerID       string `json:"runner_id"`
	ProjectID      string `json:"project_id,omitempty"`
	IdempotencyKey string `json:"idempotency_key"`
}

type ClaimResult struct {
	Task  Task  `json:"task"`
	Lease Lease `json:"lease"`
}

type HeartbeatRequest struct {
	RunnerID       string `json:"runner_id"`
	TaskID         string `json:"task_id"`
	LeaseID        string `json:"lease_id"`
	IdempotencyKey string `json:"idempotency_key"`
}

type CompleteRequest struct {
	RunnerID       string         `json:"runner_id"`
	TaskID         string         `json:"task_id"`
	LeaseID        string         `json:"lease_id"`
	IdempotencyKey string         `json:"idempotency_key"`
	Summary        string         `json:"summary,omitempty"`
	Evidence       EvidenceBundle `json:"evidence"`
}

type FailRequest struct {
	RunnerID       string          `json:"runner_id"`
	TaskID         string          `json:"task_id"`
	LeaseID        string          `json:"lease_id"`
	IdempotencyKey string          `json:"idempotency_key"`
	Error          string          `json:"error"`
	Evidence       *EvidenceBundle `json:"evidence,omitempty"`
}

type RequeueRequest struct {
	TaskID         string `json:"task_id"`
	Actor          string `json:"actor"`
	Reason         string `json:"reason"`
	IdempotencyKey string `json:"idempotency_key"`
	Force          bool   `json:"force,omitempty"`
}

type CancelRequest struct {
	TaskID         string `json:"task_id"`
	Actor          string `json:"actor"`
	Reason         string `json:"reason"`
	IdempotencyKey string `json:"idempotency_key"`
}

type TaskFilter struct {
	ProjectID string
	Status    TaskStatus
	RunnerID  string
}

type QueueStatus struct {
	SchemaVersion  int                  `json:"schema_version"`
	ProjectID      string               `json:"project_id,omitempty"`
	TaskCounts     map[TaskStatus]int   `json:"task_counts"`
	RunnerCounts   map[RunnerStatus]int `json:"runner_counts"`
	Leased         []LeaseStatus        `json:"leased,omitempty"`
	OldestQueuedAt *time.Time           `json:"oldest_queued_at,omitempty"`
	UpdatedAt      time.Time            `json:"updated_at"`
}

type LeaseStatus struct {
	TaskID    string    `json:"task_id"`
	ProjectID string    `json:"project_id"`
	RunnerID  string    `json:"runner_id"`
	Attempt   int       `json:"attempt"`
	ExpiresAt time.Time `json:"expires_at"`
}

type RecoveryReport struct {
	ExpiredTaskIDs   []string  `json:"expired_task_ids,omitempty"`
	RequeuedTaskIDs  []string  `json:"requeued_task_ids,omitempty"`
	FailedTaskIDs    []string  `json:"failed_task_ids,omitempty"`
	OfflineRunnerIDs []string  `json:"offline_runner_ids,omitempty"`
	RecoveredAt      time.Time `json:"recovered_at"`
}
