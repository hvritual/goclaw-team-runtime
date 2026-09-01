package contract

import "time"

type EvidenceSource struct {
	SourceType string    `json:"source_type"`
	Locator    string    `json:"locator"`
	Revision   string    `json:"revision,omitempty"`
	Digest     string    `json:"digest,omitempty"`
	ObservedAt time.Time `json:"observed_at"`
}

type Evidence struct {
	ID              string         `json:"id"`
	WorkspaceID     string         `json:"workspace_id"`
	Kind            string         `json:"kind"`
	Subject         NodeRef        `json:"subject"`
	Source          EvidenceSource `json:"source"`
	ProducerID      string         `json:"producer_id"`
	ArtifactURI     string         `json:"artifact_uri,omitempty"`
	ArtifactDigest  string         `json:"artifact_digest,omitempty"`
	CapturedAt      time.Time      `json:"captured_at"`
	ContentChecksum string         `json:"content_checksum"`
}

type RecordEvidenceRequest struct {
	ID             string         `json:"id"`
	Kind           string         `json:"kind"`
	Subject        NodeRef        `json:"subject"`
	Source         EvidenceSource `json:"source"`
	ProducerID     string         `json:"producer_id"`
	ArtifactURI    string         `json:"artifact_uri,omitempty"`
	ArtifactDigest string         `json:"artifact_digest,omitempty"`
}
