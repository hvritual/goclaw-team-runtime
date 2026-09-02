package domain

import (
	"errors"
	"strings"
	"time"
)

type ExecutionItemKind string

const (
	ExecutionItemKindRun         ExecutionItemKind = "run"
	ExecutionItemKindBuild       ExecutionItemKind = "build"
	ExecutionItemKindTest        ExecutionItemKind = "test"
	ExecutionItemKindRelease     ExecutionItemKind = "release"
	ExecutionItemKindDeployment  ExecutionItemKind = "deployment"
	ExecutionItemKindObservation ExecutionItemKind = "observation"
)

func (value ExecutionItemKind) Valid() bool {
	switch ExecutionItemKind(strings.ToLower(strings.TrimSpace(string(value)))) {
	case ExecutionItemKindRun,
		ExecutionItemKindBuild,
		ExecutionItemKindTest,
		ExecutionItemKindRelease,
		ExecutionItemKindDeployment,
		ExecutionItemKindObservation:
		return true
	default:
		return false
	}
}

var (
	ErrExecutionItemIDRequired            = errors.New("execution item id is required")
	ErrExecutionItemKindInvalid           = errors.New("invalid execution item kind")
	ErrExecutionItemSourceTypeRequired    = errors.New("execution item source type is required")
	ErrExecutionItemSourceIDRequired      = errors.New("execution item source id is required")
	ErrExecutionItemSourceLocatorInvalid  = errors.New("execution item source locator must be a canonical secret-free absolute uri")
	ErrExecutionItemCreatedAtRequired     = errors.New("execution item created at is required")
	ErrEvidenceAttachmentItemIDRequired   = errors.New("evidence attachment execution item id is required")
	ErrEvidenceAttachmentEvidenceRequired = errors.New("evidence attachment evidence id is required")
	ErrEvidenceAttachmentTimeRequired     = errors.New("evidence attachment time is required")
)

// ExecutionItem is an Engineering projection anchor for an execution record
// owned by Runtime, CI, release, deployment, or observability systems. It does
// not own native execution state and intentionally has no status in P3-S02.
type ExecutionItem struct {
	id            string
	workspaceID   string
	kind          ExecutionItemKind
	sourceType    string
	sourceID      string
	sourceLocator string
	createdAt     time.Time
}

func NewExecutionItem(
	id, workspaceID string,
	kind ExecutionItemKind,
	sourceType, sourceID, sourceLocator string,
	createdAt time.Time,
) (ExecutionItem, error) {
	id = strings.TrimSpace(id)
	workspaceID = strings.TrimSpace(workspaceID)
	kind = ExecutionItemKind(strings.ToLower(strings.TrimSpace(string(kind))))
	sourceType = strings.ToLower(strings.TrimSpace(sourceType))
	sourceID = strings.TrimSpace(sourceID)
	sourceLocator = strings.TrimSpace(sourceLocator)

	if id == "" {
		return ExecutionItem{}, ErrExecutionItemIDRequired
	}
	if workspaceID == "" {
		return ExecutionItem{}, ErrWorkspaceIDRequired
	}
	if !kind.Valid() {
		return ExecutionItem{}, ErrExecutionItemKindInvalid
	}
	if sourceType == "" {
		return ExecutionItem{}, ErrExecutionItemSourceTypeRequired
	}
	if sourceID == "" {
		return ExecutionItem{}, ErrExecutionItemSourceIDRequired
	}
	if !validEvidenceURI(sourceLocator) {
		return ExecutionItem{}, ErrExecutionItemSourceLocatorInvalid
	}
	if createdAt.IsZero() {
		return ExecutionItem{}, ErrExecutionItemCreatedAtRequired
	}
	return ExecutionItem{
		id:            id,
		workspaceID:   workspaceID,
		kind:          kind,
		sourceType:    sourceType,
		sourceID:      sourceID,
		sourceLocator: sourceLocator,
		createdAt:     createdAt.UTC(),
	}, nil
}

func (value ExecutionItem) ID() string                  { return value.id }
func (value ExecutionItem) WorkspaceID() string         { return value.workspaceID }
func (value ExecutionItem) Kind() ExecutionItemKind     { return value.kind }
func (value ExecutionItem) SourceType() string          { return value.sourceType }
func (value ExecutionItem) SourceID() string            { return value.sourceID }
func (value ExecutionItem) SourceLocator() string       { return value.sourceLocator }
func (value ExecutionItem) CreatedAt() time.Time        { return value.createdAt }

// EvidenceAttachment keeps the subject relationship outside EvidenceEnvelope.
// The repository validates that both identities exist in the same workspace.
type EvidenceAttachment struct {
	workspaceID     string
	executionItemID string
	evidenceID      string
	attachedAt      time.Time
}

func NewEvidenceAttachment(workspaceID, executionItemID, evidenceID string, attachedAt time.Time) (EvidenceAttachment, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	executionItemID = strings.TrimSpace(executionItemID)
	evidenceID = strings.TrimSpace(evidenceID)
	if workspaceID == "" {
		return EvidenceAttachment{}, ErrWorkspaceIDRequired
	}
	if executionItemID == "" {
		return EvidenceAttachment{}, ErrEvidenceAttachmentItemIDRequired
	}
	if evidenceID == "" {
		return EvidenceAttachment{}, ErrEvidenceAttachmentEvidenceRequired
	}
	if attachedAt.IsZero() {
		return EvidenceAttachment{}, ErrEvidenceAttachmentTimeRequired
	}
	return EvidenceAttachment{
		workspaceID:     workspaceID,
		executionItemID: executionItemID,
		evidenceID:      evidenceID,
		attachedAt:      attachedAt.UTC(),
	}, nil
}

func (value EvidenceAttachment) WorkspaceID() string     { return value.workspaceID }
func (value EvidenceAttachment) ExecutionItemID() string { return value.executionItemID }
func (value EvidenceAttachment) EvidenceID() string      { return value.evidenceID }
func (value EvidenceAttachment) AttachedAt() time.Time   { return value.attachedAt }
