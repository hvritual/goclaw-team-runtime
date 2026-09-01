package contract

import (
	"encoding/json"
	"time"
)

const EvidenceEnvelopeSchemaV1 = "engineering.evidence/v1"

type EvidenceKind string

const (
	EvidenceKindValidation EvidenceKind = "validation"
	EvidenceKindDeployment EvidenceKind = "deployment"
	EvidenceKindTrace      EvidenceKind = "trace"
)

type EvidenceOutcome string

const (
	EvidenceOutcomePassed     EvidenceOutcome = "passed"
	EvidenceOutcomeFailed     EvidenceOutcome = "failed"
	EvidenceOutcomeErrored    EvidenceOutcome = "errored"
	EvidenceOutcomeSucceeded  EvidenceOutcome = "succeeded"
	EvidenceOutcomeRolledBack EvidenceOutcome = "rolled_back"
	EvidenceOutcomeObserved   EvidenceOutcome = "observed"
)

type EvidenceSource struct {
	Type       string    `json:"type"`
	ID         string    `json:"id"`
	Locator    string    `json:"locator"`
	Revision   string    `json:"revision,omitempty"`
	Digest     string    `json:"digest,omitempty"`
	ObservedAt time.Time `json:"observed_at"`
}

type EvidenceArtifact struct {
	URI    string `json:"uri"`
	Digest string `json:"digest"`
}

type EvidenceEnvelope struct {
	SchemaVersion   string            `json:"schema_version"`
	ID              string            `json:"id"`
	WorkspaceID     string            `json:"workspace_id"`
	Kind            EvidenceKind      `json:"kind"`
	Outcome         EvidenceOutcome   `json:"outcome"`
	Source          EvidenceSource    `json:"source"`
	ProducerID      string            `json:"producer_id"`
	Artifact        *EvidenceArtifact `json:"artifact,omitempty"`
	Payload         json.RawMessage   `json:"payload"`
	CapturedAt      time.Time         `json:"captured_at"`
	ContentChecksum string            `json:"content_checksum"`
}

// NormalizeEvidenceRequest is the transport-neutral input contract for creating
// a normalized immutable envelope. It intentionally has no execution-item
// subject: evidence attachment is owned by later Phase 3 slices.
type NormalizeEvidenceRequest struct {
	ID          string            `json:"id"`
	WorkspaceID string            `json:"workspace_id"`
	Kind        EvidenceKind      `json:"kind"`
	Outcome     EvidenceOutcome   `json:"outcome"`
	Source      EvidenceSource    `json:"source"`
	ProducerID  string            `json:"producer_id"`
	Artifact    *EvidenceArtifact `json:"artifact,omitempty"`
	Payload     json.RawMessage   `json:"payload"`
	CapturedAt  time.Time         `json:"captured_at"`
}
