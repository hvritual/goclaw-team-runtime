package domain

import (
	"errors"
	"sort"
	"strings"
	"time"
)

type ChangeStatus string

const (
	ChangeStatusProposed   ChangeStatus = "proposed"
	ChangeStatusAccepted   ChangeStatus = "accepted"
	ChangeStatusRejected   ChangeStatus = "rejected"
	ChangeStatusSuperseded ChangeStatus = "superseded"
)

var validChangeStatuses = map[ChangeStatus]struct{}{
	ChangeStatusProposed: {}, ChangeStatusAccepted: {}, ChangeStatusRejected: {}, ChangeStatusSuperseded: {},
}

func (value ChangeStatus) Valid() bool {
	_, ok := validChangeStatuses[ChangeStatus(strings.TrimSpace(string(value)))]
	return ok
}

var (
	ErrChangeIDRequired        = errors.New("change id is required")
	ErrChangeSummaryRequired   = errors.New("change summary is required")
	ErrAffectedEntityRequired  = errors.New("change requires at least one affected engineering entity")
	ErrArtifactKindRequired    = errors.New("artifact kind is required")
	ErrArtifactLocatorRequired = errors.New("artifact locator is required")
	ErrChangeStatusInvalid     = errors.New("invalid change status")
	ErrChangeTransitionInvalid = errors.New("invalid change status transition")
	ErrTimestampRequired       = errors.New("timestamp is required")
)

type ArtifactRef struct {
	kind     string
	locator  string
	revision string
}

func NewArtifactRef(kind, locator, revision string) (ArtifactRef, error) {
	kind = strings.TrimSpace(kind)
	locator = strings.TrimSpace(locator)
	revision = strings.TrimSpace(revision)
	if kind == "" {
		return ArtifactRef{}, ErrArtifactKindRequired
	}
	if locator == "" {
		return ArtifactRef{}, ErrArtifactLocatorRequired
	}
	return ArtifactRef{kind: kind, locator: locator, revision: revision}, nil
}

func (value ArtifactRef) Kind() string     { return value.kind }
func (value ArtifactRef) Locator() string  { return value.locator }
func (value ArtifactRef) Revision() string { return value.revision }

type Change struct {
	id                string
	workspaceID       string
	projectID         string
	requirementID     string
	workItem          *NodeRef
	runID             string
	summary           string
	status            ChangeStatus
	affectedEntityIDs []string
	artifacts         []ArtifactRef
	provenance        Provenance
	createdAt         time.Time
	updatedAt         time.Time
	acceptedAt        *time.Time
}

func NewChange(id, workspaceID, projectID, requirementID string, workItem *NodeRef, runID, summary string, affectedEntityIDs []string, artifacts []ArtifactRef, provenance Provenance, now time.Time) (Change, error) {
	id = strings.TrimSpace(id)
	workspaceID = strings.TrimSpace(workspaceID)
	projectID = strings.TrimSpace(projectID)
	requirementID = strings.TrimSpace(requirementID)
	runID = strings.TrimSpace(runID)
	summary = strings.TrimSpace(summary)
	if id == "" {
		return Change{}, ErrChangeIDRequired
	}
	if workspaceID == "" {
		return Change{}, ErrWorkspaceIDRequired
	}
	if summary == "" {
		return Change{}, ErrChangeSummaryRequired
	}
	if workItem != nil && !isWorkItemKind(workItem.Kind()) {
		return Change{}, ErrNodeKindInvalid
	}
	normalizedAffected, err := normalizeIDs(affectedEntityIDs)
	if err != nil || len(normalizedAffected) == 0 {
		return Change{}, ErrAffectedEntityRequired
	}
	if !provenance.Valid() {
		return Change{}, ErrProvenanceRequired
	}
	if now.IsZero() {
		return Change{}, ErrTimestampRequired
	}
	now = now.UTC()
	return Change{
		id: id, workspaceID: workspaceID, projectID: projectID, requirementID: requirementID,
		workItem: copyNodeRef(workItem), runID: runID, summary: summary, status: ChangeStatusProposed,
		affectedEntityIDs: normalizedAffected, artifacts: append([]ArtifactRef(nil), artifacts...), provenance: provenance,
		createdAt: now, updatedAt: now,
	}, nil
}

func RehydrateChange(id, workspaceID, projectID, requirementID string, workItem *NodeRef, runID, summary string, status ChangeStatus, affectedEntityIDs []string, artifacts []ArtifactRef, provenance Provenance, createdAt, updatedAt time.Time, acceptedAt *time.Time) (Change, error) {
	value, err := NewChange(id, workspaceID, projectID, requirementID, workItem, runID, summary, affectedEntityIDs, artifacts, provenance, createdAt)
	if err != nil {
		return Change{}, err
	}
	if !status.Valid() {
		return Change{}, ErrChangeStatusInvalid
	}
	if updatedAt.IsZero() {
		return Change{}, ErrTimestampRequired
	}
	if (status == ChangeStatusAccepted || status == ChangeStatusSuperseded) != (acceptedAt != nil) {
		return Change{}, ErrChangeTransitionInvalid
	}
	value.status = status
	value.updatedAt = updatedAt.UTC()
	if acceptedAt != nil {
		normalized := acceptedAt.UTC()
		value.acceptedAt = &normalized
	}
	return value, nil
}

func normalizeIDs(values []string) ([]string, error) {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, ErrAffectedEntityRequired
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result, nil
}

func copyNodeRef(value *NodeRef) *NodeRef {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}

func isWorkItemKind(kind NodeKind) bool {
	switch kind {
	case NodeKindProject, NodeKindRequirement, NodeKindIssue, NodeKindTodo, NodeKindTask:
		return true
	default:
		return false
	}
}

func (value Change) Accept(now time.Time) (Change, error) {
	if value.status != ChangeStatusProposed || now.IsZero() {
		return Change{}, ErrChangeTransitionInvalid
	}
	acceptedAt := now.UTC()
	value.status = ChangeStatusAccepted
	value.updatedAt = acceptedAt
	value.acceptedAt = &acceptedAt
	return value, nil
}

func (value Change) Reject(now time.Time) (Change, error) {
	if value.status != ChangeStatusProposed || now.IsZero() {
		return Change{}, ErrChangeTransitionInvalid
	}
	value.status = ChangeStatusRejected
	value.updatedAt = now.UTC()
	return value, nil
}

func (value Change) Supersede(now time.Time) (Change, error) {
	if value.status != ChangeStatusAccepted || now.IsZero() {
		return Change{}, ErrChangeTransitionInvalid
	}
	value.status = ChangeStatusSuperseded
	value.updatedAt = now.UTC()
	return value, nil
}

func (value Change) ID() string            { return value.id }
func (value Change) WorkspaceID() string   { return value.workspaceID }
func (value Change) ProjectID() string     { return value.projectID }
func (value Change) RequirementID() string { return value.requirementID }
func (value Change) WorkItem() *NodeRef    { return copyNodeRef(value.workItem) }
func (value Change) RunID() string         { return value.runID }
func (value Change) Summary() string       { return value.summary }
func (value Change) Status() ChangeStatus  { return value.status }
func (value Change) AffectedEntityIDs() []string {
	return append([]string(nil), value.affectedEntityIDs...)
}
func (value Change) Artifacts() []ArtifactRef { return append([]ArtifactRef(nil), value.artifacts...) }
func (value Change) Provenance() Provenance   { return value.provenance }
func (value Change) CreatedAt() time.Time     { return value.createdAt }
func (value Change) UpdatedAt() time.Time     { return value.updatedAt }
func (value Change) AcceptedAt() *time.Time {
	if value.acceptedAt == nil {
		return nil
	}
	copyValue := *value.acceptedAt
	return &copyValue
}
