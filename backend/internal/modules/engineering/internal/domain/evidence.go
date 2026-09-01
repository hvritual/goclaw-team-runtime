package domain

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"strings"
	"time"
)

const (
	EvidenceEnvelopeSchemaV1 = "engineering.evidence/v1"
	MaxEvidencePayloadBytes  = 256 * 1024
)

type EvidenceKind string

const (
	EvidenceKindValidation EvidenceKind = "validation"
	EvidenceKindDeployment EvidenceKind = "deployment"
	EvidenceKindTrace      EvidenceKind = "trace"
)

func (value EvidenceKind) Valid() bool {
	switch EvidenceKind(strings.ToLower(strings.TrimSpace(string(value)))) {
	case EvidenceKindValidation, EvidenceKindDeployment, EvidenceKindTrace:
		return true
	default:
		return false
	}
}

type EvidenceOutcome string

const (
	EvidenceOutcomePassed     EvidenceOutcome = "passed"
	EvidenceOutcomeFailed     EvidenceOutcome = "failed"
	EvidenceOutcomeErrored    EvidenceOutcome = "errored"
	EvidenceOutcomeSucceeded  EvidenceOutcome = "succeeded"
	EvidenceOutcomeRolledBack EvidenceOutcome = "rolled_back"
	EvidenceOutcomeObserved   EvidenceOutcome = "observed"
)

var (
	ErrEvidenceIDRequired           = errors.New("evidence id is required")
	ErrEvidenceKindInvalid          = errors.New("invalid evidence kind")
	ErrEvidenceOutcomeInvalid       = errors.New("invalid evidence outcome for kind")
	ErrEvidenceSourceInvalid        = errors.New("invalid evidence source")
	ErrEvidenceSourceTypeRequired   = errors.New("evidence source type is required")
	ErrEvidenceSourceIDRequired     = errors.New("evidence source id is required")
	ErrEvidenceSourceIdentityWeak   = errors.New("evidence source requires an immutable revision or sha256 digest")
	ErrEvidenceSourceDigestInvalid  = errors.New("evidence source digest must be sha256")
	ErrEvidenceSourceLocatorInvalid = errors.New("evidence source locator must be a canonical secret-free absolute uri")
	ErrEvidenceProducerRequired     = errors.New("evidence producer id is required")
	ErrEvidenceArtifactInvalid      = errors.New("invalid evidence artifact")
	ErrEvidenceArtifactURIInvalid   = errors.New("evidence artifact uri must be a canonical secret-free absolute uri")
	ErrEvidenceArtifactDigestInvalid = errors.New("evidence artifact digest must be sha256")
	ErrEvidencePayloadInvalid       = errors.New("evidence payload must be valid json")
	ErrEvidencePayloadObjectRequired = errors.New("evidence payload must be a json object")
	ErrEvidencePayloadTooLarge      = errors.New("evidence payload exceeds 256 KiB")
	ErrEvidenceCapturedAtRequired   = errors.New("evidence captured at is required")
	ErrEvidenceSchemaVersionInvalid = errors.New("unsupported evidence envelope schema version")
	ErrEvidenceChecksumMismatch     = errors.New("evidence content checksum mismatch")
)

type EvidenceSource struct {
	sourceType string
	id         string
	locator    string
	revision   string
	digest     string
	observedAt time.Time
}

func NewEvidenceSource(sourceType, id, locator, revision, digest string, observedAt time.Time) (EvidenceSource, error) {
	sourceType = strings.ToLower(strings.TrimSpace(sourceType))
	id = strings.TrimSpace(id)
	locator = strings.TrimSpace(locator)
	revision = strings.TrimSpace(revision)
	digest = strings.ToLower(strings.TrimSpace(digest))

	if sourceType == "" {
		return EvidenceSource{}, ErrEvidenceSourceTypeRequired
	}
	if id == "" {
		return EvidenceSource{}, ErrEvidenceSourceIDRequired
	}
	if !validEvidenceURI(locator) {
		return EvidenceSource{}, ErrEvidenceSourceLocatorInvalid
	}
	if revision == "" && digest == "" {
		return EvidenceSource{}, ErrEvidenceSourceIdentityWeak
	}
	if digest != "" && !validSHA256Digest(digest) {
		return EvidenceSource{}, ErrEvidenceSourceDigestInvalid
	}
	if observedAt.IsZero() {
		return EvidenceSource{}, ErrObservedAtRequired
	}
	return EvidenceSource{
		sourceType: sourceType,
		id:         id,
		locator:    locator,
		revision:   revision,
		digest:     digest,
		observedAt: observedAt.UTC(),
	}, nil
}

func (value EvidenceSource) Valid() bool {
	if value.sourceType == "" || value.sourceType != strings.ToLower(strings.TrimSpace(value.sourceType)) {
		return false
	}
	if strings.TrimSpace(value.id) == "" || strings.TrimSpace(value.id) != value.id {
		return false
	}
	if !validEvidenceURI(value.locator) || (value.revision == "" && value.digest == "") || value.observedAt.IsZero() {
		return false
	}
	return value.digest == "" || validSHA256Digest(value.digest)
}

func (value EvidenceSource) Type() string            { return value.sourceType }
func (value EvidenceSource) ID() string              { return value.id }
func (value EvidenceSource) Locator() string         { return value.locator }
func (value EvidenceSource) Revision() string        { return value.revision }
func (value EvidenceSource) Digest() string          { return value.digest }
func (value EvidenceSource) ObservedAt() time.Time   { return value.observedAt }

type EvidenceArtifact struct {
	uri    string
	digest string
}

func NewEvidenceArtifact(uri, digest string) (EvidenceArtifact, error) {
	uri = strings.TrimSpace(uri)
	digest = strings.ToLower(strings.TrimSpace(digest))
	if !validEvidenceURI(uri) {
		return EvidenceArtifact{}, ErrEvidenceArtifactURIInvalid
	}
	if !validSHA256Digest(digest) {
		return EvidenceArtifact{}, ErrEvidenceArtifactDigestInvalid
	}
	return EvidenceArtifact{uri: uri, digest: digest}, nil
}

func (value EvidenceArtifact) Valid() bool {
	return validEvidenceURI(value.uri) && validSHA256Digest(value.digest)
}

func (value EvidenceArtifact) URI() string    { return value.uri }
func (value EvidenceArtifact) Digest() string { return value.digest }

type EvidenceEnvelope struct {
	schemaVersion string
	id            string
	workspaceID   string
	kind          EvidenceKind
	outcome       EvidenceOutcome
	source        EvidenceSource
	producerID    string
	artifact      *EvidenceArtifact
	payloadJSON   string
	capturedAt    time.Time
	checksum      string
}

func NewEvidenceEnvelope(
	id, workspaceID string,
	kind EvidenceKind,
	outcome EvidenceOutcome,
	source EvidenceSource,
	producerID string,
	artifact *EvidenceArtifact,
	payload json.RawMessage,
	capturedAt time.Time,
) (EvidenceEnvelope, error) {
	id = strings.TrimSpace(id)
	workspaceID = strings.TrimSpace(workspaceID)
	kind = EvidenceKind(strings.ToLower(strings.TrimSpace(string(kind))))
	outcome = EvidenceOutcome(strings.ToLower(strings.TrimSpace(string(outcome))))
	producerID = strings.TrimSpace(producerID)

	if id == "" {
		return EvidenceEnvelope{}, ErrEvidenceIDRequired
	}
	if workspaceID == "" {
		return EvidenceEnvelope{}, ErrWorkspaceIDRequired
	}
	if !kind.Valid() {
		return EvidenceEnvelope{}, ErrEvidenceKindInvalid
	}
	if !evidenceOutcomeValidForKind(kind, outcome) {
		return EvidenceEnvelope{}, ErrEvidenceOutcomeInvalid
	}
	if !source.Valid() {
		return EvidenceEnvelope{}, ErrEvidenceSourceInvalid
	}
	if producerID == "" {
		return EvidenceEnvelope{}, ErrEvidenceProducerRequired
	}
	if artifact != nil && !artifact.Valid() {
		return EvidenceEnvelope{}, ErrEvidenceArtifactInvalid
	}
	payloadJSON, err := normalizeEvidencePayload(payload)
	if err != nil {
		return EvidenceEnvelope{}, err
	}
	if capturedAt.IsZero() {
		return EvidenceEnvelope{}, ErrEvidenceCapturedAtRequired
	}

	var artifactCopy *EvidenceArtifact
	if artifact != nil {
		copyValue := *artifact
		artifactCopy = &copyValue
	}
	value := EvidenceEnvelope{
		schemaVersion: EvidenceEnvelopeSchemaV1,
		id:            id,
		workspaceID:   workspaceID,
		kind:          kind,
		outcome:       outcome,
		source:        source,
		producerID:    producerID,
		artifact:      artifactCopy,
		payloadJSON:   payloadJSON,
		capturedAt:    capturedAt.UTC(),
	}
	value.checksum = evidenceContentChecksum(value)
	return value, nil
}

func RehydrateEvidenceEnvelope(
	schemaVersion, id, workspaceID string,
	kind EvidenceKind,
	outcome EvidenceOutcome,
	source EvidenceSource,
	producerID string,
	artifact *EvidenceArtifact,
	payload json.RawMessage,
	capturedAt time.Time,
	checksum string,
) (EvidenceEnvelope, error) {
	if strings.TrimSpace(schemaVersion) != EvidenceEnvelopeSchemaV1 {
		return EvidenceEnvelope{}, ErrEvidenceSchemaVersionInvalid
	}
	value, err := NewEvidenceEnvelope(id, workspaceID, kind, outcome, source, producerID, artifact, payload, capturedAt)
	if err != nil {
		return EvidenceEnvelope{}, err
	}
	checksum = strings.ToLower(strings.TrimSpace(checksum))
	if !validSHA256Digest(checksum) || checksum != value.checksum {
		return EvidenceEnvelope{}, ErrEvidenceChecksumMismatch
	}
	return value, nil
}

func (value EvidenceEnvelope) SchemaVersion() string  { return value.schemaVersion }
func (value EvidenceEnvelope) ID() string             { return value.id }
func (value EvidenceEnvelope) WorkspaceID() string    { return value.workspaceID }
func (value EvidenceEnvelope) Kind() EvidenceKind     { return value.kind }
func (value EvidenceEnvelope) Outcome() EvidenceOutcome { return value.outcome }
func (value EvidenceEnvelope) Source() EvidenceSource { return value.source }
func (value EvidenceEnvelope) ProducerID() string     { return value.producerID }
func (value EvidenceEnvelope) CapturedAt() time.Time  { return value.capturedAt }
func (value EvidenceEnvelope) ContentChecksum() string { return value.checksum }

func (value EvidenceEnvelope) Artifact() *EvidenceArtifact {
	if value.artifact == nil {
		return nil
	}
	copyValue := *value.artifact
	return &copyValue
}

func (value EvidenceEnvelope) Payload() json.RawMessage {
	return append(json.RawMessage(nil), []byte(value.payloadJSON)...)
}

func evidenceOutcomeValidForKind(kind EvidenceKind, outcome EvidenceOutcome) bool {
	switch kind {
	case EvidenceKindValidation:
		return outcome == EvidenceOutcomePassed || outcome == EvidenceOutcomeFailed || outcome == EvidenceOutcomeErrored
	case EvidenceKindDeployment:
		return outcome == EvidenceOutcomeSucceeded || outcome == EvidenceOutcomeFailed || outcome == EvidenceOutcomeRolledBack
	case EvidenceKindTrace:
		return outcome == EvidenceOutcomeObserved
	default:
		return false
	}
}

func normalizeEvidencePayload(raw json.RawMessage) (string, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return "{}", nil
	}

	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.UseNumber()
	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		return "", ErrEvidencePayloadInvalid
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return "", ErrEvidencePayloadInvalid
	}
	object, ok := decoded.(map[string]any)
	if !ok || object == nil {
		return "", ErrEvidencePayloadObjectRequired
	}
	canonical, err := json.Marshal(object)
	if err != nil {
		return "", ErrEvidencePayloadInvalid
	}
	if len(canonical) > MaxEvidencePayloadBytes {
		return "", ErrEvidencePayloadTooLarge
	}
	return string(canonical), nil
}

func evidenceContentChecksum(value EvidenceEnvelope) string {
	canonical := struct {
		SchemaVersion  string          `json:"schema_version"`
		WorkspaceID    string          `json:"workspace_id"`
		Kind           EvidenceKind    `json:"kind"`
		Outcome        EvidenceOutcome `json:"outcome"`
		SourceType     string          `json:"source_type"`
		SourceID       string          `json:"source_id"`
		SourceLocator  string          `json:"source_locator"`
		SourceRevision string          `json:"source_revision"`
		SourceDigest   string          `json:"source_digest"`
		ProducerID     string          `json:"producer_id"`
		ArtifactURI    string          `json:"artifact_uri"`
		ArtifactDigest string          `json:"artifact_digest"`
		Payload        json.RawMessage `json:"payload"`
	}{
		SchemaVersion:  value.schemaVersion,
		WorkspaceID:    value.workspaceID,
		Kind:           value.kind,
		Outcome:        value.outcome,
		SourceType:     value.source.Type(),
		SourceID:       value.source.ID(),
		SourceLocator:  value.source.Locator(),
		SourceRevision: value.source.Revision(),
		SourceDigest:   value.source.Digest(),
		ProducerID:     value.producerID,
		Payload:        json.RawMessage(value.payloadJSON),
	}
	if value.artifact != nil {
		canonical.ArtifactURI = value.artifact.URI()
		canonical.ArtifactDigest = value.artifact.Digest()
	}
	encoded, _ := json.Marshal(canonical)
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

func validSHA256Digest(value string) bool {
	if len(value) != sha256.Size*2 || value != strings.ToLower(value) {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size
}

func validEvidenceURI(value string) bool {
	if value == "" || len(value) > 1024 || strings.TrimSpace(value) != value {
		return false
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Opaque != "" {
		return false
	}
	return parsed.String() == value
}
