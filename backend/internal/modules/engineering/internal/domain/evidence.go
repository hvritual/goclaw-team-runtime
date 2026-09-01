package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"time"
)

type EvidenceKind string

const (
	EvidenceKindExecution    EvidenceKind = "execution"
	EvidenceKindSourceChange EvidenceKind = "source_change"
	EvidenceKindBuild        EvidenceKind = "build"
	EvidenceKindTest         EvidenceKind = "test"
	EvidenceKindRelease      EvidenceKind = "release"
	EvidenceKindDeployment   EvidenceKind = "deployment"
	EvidenceKindObservation  EvidenceKind = "observation"
	EvidenceKindIncident     EvidenceKind = "incident"
)

var validEvidenceKinds = map[EvidenceKind]struct{}{
	EvidenceKindExecution: {}, EvidenceKindSourceChange: {}, EvidenceKindBuild: {}, EvidenceKindTest: {},
	EvidenceKindRelease: {}, EvidenceKindDeployment: {}, EvidenceKindObservation: {}, EvidenceKindIncident: {},
}

func (value EvidenceKind) Valid() bool {
	_, ok := validEvidenceKinds[EvidenceKind(strings.TrimSpace(string(value)))]
	return ok
}

var (
	ErrEvidenceIDRequired           = errors.New("evidence id is required")
	ErrEvidenceKindInvalid          = errors.New("invalid evidence kind")
	ErrEvidenceSourceIdentityWeak   = errors.New("evidence source requires an immutable revision or sha256 digest")
	ErrEvidenceSourceDigestInvalid  = errors.New("evidence source digest must be sha256")
	ErrEvidenceSourceLocatorInvalid = errors.New("evidence source locator must be a normalized secret-free uri")
	ErrEvidenceProducerRequired     = errors.New("evidence producer id is required")
	ErrEvidenceArtifactInvalid      = errors.New("evidence artifact uri and sha256 digest must be supplied together")
	ErrEvidenceArtifactURIInvalid   = errors.New("evidence artifact uri must be a normalized secret-free uri")
	ErrEvidenceChecksumMismatch     = errors.New("evidence content checksum mismatch")
	ErrEvidenceCapturedAtRequired   = errors.New("evidence captured at is required")
)

type EvidenceSource struct {
	sourceType string
	locator    string
	revision   string
	digest     string
	observedAt time.Time
}

func NewEvidenceSource(sourceType, locator, revision, digest string, observedAt time.Time) (EvidenceSource, error) {
	sourceType = strings.TrimSpace(sourceType)
	locator = strings.TrimSpace(locator)
	revision = strings.TrimSpace(revision)
	digest = strings.ToLower(strings.TrimSpace(digest))
	if sourceType == "" {
		return EvidenceSource{}, ErrSourceTypeRequired
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
	return EvidenceSource{sourceType: sourceType, locator: locator, revision: revision, digest: digest, observedAt: observedAt.UTC()}, nil
}

func (value EvidenceSource) Valid() bool {
	return value.sourceType != "" && validEvidenceURI(value.locator) && (value.revision != "" || validSHA256Digest(value.digest)) && !value.observedAt.IsZero()
}
func (value EvidenceSource) SourceType() string    { return value.sourceType }
func (value EvidenceSource) Locator() string       { return value.locator }
func (value EvidenceSource) Revision() string      { return value.revision }
func (value EvidenceSource) Digest() string        { return value.digest }
func (value EvidenceSource) ObservedAt() time.Time { return value.observedAt }

type EvidenceEnvelope struct {
	id             string
	workspaceID    string
	kind           EvidenceKind
	subject        NodeRef
	source         EvidenceSource
	producerID     string
	artifactURI    string
	artifactDigest string
	capturedAt     time.Time
	checksum       string
}

func NewEvidenceEnvelope(id, workspaceID string, kind EvidenceKind, subject NodeRef, source EvidenceSource, producerID, artifactURI, artifactDigest string, capturedAt time.Time) (EvidenceEnvelope, error) {
	id = strings.TrimSpace(id)
	workspaceID = strings.TrimSpace(workspaceID)
	producerID = strings.TrimSpace(producerID)
	artifactURI = strings.TrimSpace(artifactURI)
	artifactDigest = strings.ToLower(strings.TrimSpace(artifactDigest))
	if id == "" {
		return EvidenceEnvelope{}, ErrEvidenceIDRequired
	}
	if workspaceID == "" {
		return EvidenceEnvelope{}, ErrWorkspaceIDRequired
	}
	if !kind.Valid() {
		return EvidenceEnvelope{}, ErrEvidenceKindInvalid
	}
	if !subject.Kind().Valid() || strings.TrimSpace(subject.ID()) == "" {
		return EvidenceEnvelope{}, ErrNodeKindInvalid
	}
	if !source.Valid() {
		return EvidenceEnvelope{}, ErrProvenanceRequired
	}
	if producerID == "" {
		return EvidenceEnvelope{}, ErrEvidenceProducerRequired
	}
	if (artifactURI == "") != (artifactDigest == "") {
		return EvidenceEnvelope{}, ErrEvidenceArtifactInvalid
	}
	if artifactURI != "" && !validEvidenceURI(artifactURI) {
		return EvidenceEnvelope{}, ErrEvidenceArtifactURIInvalid
	}
	if artifactDigest != "" && !validSHA256Digest(artifactDigest) {
		return EvidenceEnvelope{}, ErrEvidenceArtifactInvalid
	}
	if capturedAt.IsZero() {
		return EvidenceEnvelope{}, ErrEvidenceCapturedAtRequired
	}
	value := EvidenceEnvelope{
		id: id, workspaceID: workspaceID, kind: kind, subject: subject, source: source, producerID: producerID,
		artifactURI: artifactURI, artifactDigest: artifactDigest, capturedAt: capturedAt.UTC(),
	}
	value.checksum = evidenceContentChecksum(value)
	return value, nil
}

func RehydrateEvidenceEnvelope(id, workspaceID string, kind EvidenceKind, subject NodeRef, source EvidenceSource, producerID, artifactURI, artifactDigest string, capturedAt time.Time, checksum string) (EvidenceEnvelope, error) {
	value, err := NewEvidenceEnvelope(id, workspaceID, kind, subject, source, producerID, artifactURI, artifactDigest, capturedAt)
	if err != nil {
		return EvidenceEnvelope{}, err
	}
	checksum = strings.ToLower(strings.TrimSpace(checksum))
	if !validSHA256Digest(checksum) || checksum != value.checksum {
		return EvidenceEnvelope{}, ErrEvidenceChecksumMismatch
	}
	return value, nil
}

func (value EvidenceEnvelope) ID() string               { return value.id }
func (value EvidenceEnvelope) WorkspaceID() string      { return value.workspaceID }
func (value EvidenceEnvelope) Kind() EvidenceKind       { return value.kind }
func (value EvidenceEnvelope) Subject() NodeRef          { return value.subject }
func (value EvidenceEnvelope) Source() EvidenceSource    { return value.source }
func (value EvidenceEnvelope) ProducerID() string        { return value.producerID }
func (value EvidenceEnvelope) ArtifactURI() string       { return value.artifactURI }
func (value EvidenceEnvelope) ArtifactDigest() string    { return value.artifactDigest }
func (value EvidenceEnvelope) CapturedAt() time.Time     { return value.capturedAt }
func (value EvidenceEnvelope) ContentChecksum() string   { return value.checksum }

func evidenceContentChecksum(value EvidenceEnvelope) string {
	canonical := struct {
		WorkspaceID    string `json:"workspace_id"`
		Kind           string `json:"kind"`
		SubjectKind    string `json:"subject_kind"`
		SubjectID      string `json:"subject_id"`
		SourceType     string `json:"source_type"`
		SourceLocator  string `json:"source_locator"`
		SourceRevision string `json:"source_revision"`
		SourceDigest   string `json:"source_digest"`
		ProducerID     string `json:"producer_id"`
		ArtifactURI    string `json:"artifact_uri"`
		ArtifactDigest string `json:"artifact_digest"`
	}{
		WorkspaceID: value.workspaceID, Kind: string(value.kind), SubjectKind: string(value.subject.Kind()), SubjectID: value.subject.ID(),
		SourceType: value.source.SourceType(), SourceLocator: value.source.Locator(), SourceRevision: value.source.Revision(), SourceDigest: value.source.Digest(),
		ProducerID: value.producerID, ArtifactURI: value.artifactURI, ArtifactDigest: value.artifactDigest,
	}
	encoded, _ := json.Marshal(canonical)
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

func validSHA256Digest(value string) bool {
	if len(value) != 64 {
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
	if err != nil || parsed.Scheme == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return false
	}
	return parsed.String() == value
}
