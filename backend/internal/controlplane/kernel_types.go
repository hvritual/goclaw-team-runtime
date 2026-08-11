package controlplane

import (
	"context"
	"encoding/json"
	"time"
)

const kernelSchemaVersion = 1

const (
	EventWorkNodeUpserted   = "work.node.upserted.v1"
	EventWorkEdgeAdded      = "work.edge.added.v1"
	EventEvidenceAttached   = "evidence.attached.v1"
	EventCheckRecorded      = "check.recorded.v1"
	EventAcceptanceRecorded = "acceptance.recorded.v1"
)

type SessionEvent struct {
	SchemaVersion int             `json:"schema_version"`
	WorkspaceID   string          `json:"workspace_id"`
	ProjectID     string          `json:"project_id"`
	Sequence      int64           `json:"sequence"`
	EventID       string          `json:"event_id"`
	CommandID     string          `json:"command_id"`
	Type          string          `json:"type"`
	ActorID       string          `json:"actor_id"`
	ActorKind     ActorKind       `json:"actor_kind"`
	Payload       json.RawMessage `json:"payload"`
	PreviousHash  string          `json:"previous_hash"`
	Hash          string          `json:"hash"`
	OccurredAt    time.Time       `json:"occurred_at"`
}

type CommandEnvelope struct {
	WorkspaceID  string
	ProjectID    string
	CommandID    string
	Name         string
	Actor        Actor
	ExpectedHead int64
	Request      json.RawMessage
}

type ProposedEvent struct {
	Type       string
	Payload    json.RawMessage
	OccurredAt time.Time
}

type AppendResult struct {
	Events   []SessionEvent `json:"events"`
	Head     int64          `json:"head"`
	HeadHash string         `json:"head_hash"`
	Replayed bool           `json:"replayed"`
}

type KernelStore interface {
	AppendCommand(context.Context, CommandEnvelope, []ProposedEvent) (AppendResult, error)
	ListSessionEvents(context.Context, string, string) ([]SessionEvent, error)
	ProjectHead(context.Context, string, string) (int64, string, error)
}

type WorkNode struct {
	ID          string          `json:"id"`
	Kind        string          `json:"kind"`
	Revision    int64           `json:"revision"`
	State       string          `json:"state"`
	CreatorID   string          `json:"creator_id"`
	AssigneeIDs []string        `json:"assignee_ids"`
	ExecutorIDs []string        `json:"executor_ids"`
	Data        json.RawMessage `json:"data,omitempty"`
}

type WorkEdge struct {
	ID   string `json:"id"`
	From string `json:"from"`
	To   string `json:"to"`
	Kind string `json:"kind"`
}

type EvidenceRef struct {
	ID         string    `json:"id"`
	SubjectID  string    `json:"subject_id"`
	Kind       string    `json:"kind"`
	URI        string    `json:"uri"`
	SHA256     string    `json:"sha256"`
	Size       int64     `json:"size"`
	MediaType  string    `json:"media_type"`
	ProducedBy string    `json:"produced_by"`
	RunID      string    `json:"run_id,omitempty"`
	Sanitized  bool      `json:"sanitized"`
	CapturedAt time.Time `json:"captured_at"`
}

type CheckOutcome string

const (
	CheckPassed CheckOutcome = "passed"
	CheckFailed CheckOutcome = "failed"
)

type CheckResult struct {
	ID            string       `json:"id"`
	PolicyID      string       `json:"policy_id"`
	SubjectID     string       `json:"subject_id"`
	Revision      int64        `json:"revision"`
	Outcome       CheckOutcome `json:"outcome"`
	EvidenceIDs   []string     `json:"evidence_ids"`
	CheckerID     string       `json:"checker_id"`
	Deterministic bool         `json:"deterministic"`
	RecordedAt    time.Time    `json:"recorded_at"`
}

type Acceptance struct {
	SubjectID  string    `json:"subject_id"`
	Revision   int64     `json:"revision"`
	AcceptorID string    `json:"acceptor_id"`
	PolicyIDs  []string  `json:"policy_ids"`
	AcceptedAt time.Time `json:"accepted_at"`
}

type ProjectProjection struct {
	WorkspaceID string                   `json:"workspace_id"`
	ProjectID   string                   `json:"project_id"`
	Head        int64                    `json:"head"`
	HeadHash    string                   `json:"head_hash"`
	Nodes       map[string]WorkNode      `json:"nodes"`
	Edges       map[string]WorkEdge      `json:"edges"`
	Evidence    map[string]EvidenceRef   `json:"evidence"`
	Checks      map[string][]CheckResult `json:"checks"`
	Acceptances map[string]Acceptance    `json:"acceptances"`
}
