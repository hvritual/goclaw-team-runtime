// Package domain contains the business vocabulary and lifecycle rules for a
// versioned project requirement baseline.
package domain

import "errors"

// Status identifies the lifecycle state of a requirement baseline or revision.
type Status string

// StatusDraft, StatusInReview, StatusApproved, and StatusSuperseded are the
// supported requirement lifecycle states.
const (
	StatusDraft      Status = "draft"
	StatusInReview   Status = "in_review"
	StatusApproved   Status = "approved"
	StatusSuperseded Status = "superseded"
)

// ErrNotFound, ErrRevisionConflict, ErrInvalidTransition, and
// ErrInvalidTracking are stable project-requirements domain errors.
var (
	ErrNotFound          = errors.New("project requirement baseline not found")
	ErrRevisionConflict  = errors.New("project requirement revision conflict")
	ErrInvalidTransition = errors.New("invalid project requirement transition")
	ErrInvalidTracking   = errors.New("invalid project requirement tracking request")
)

// Item is one keyed requirement statement.
type Item struct {
	Key  string `json:"key"`
	Text string `json:"text"`
}

// Content is the versioned requirement document.
type Content struct {
	ProblemStatement   string `json:"problem_statement"`
	Goals              []Item `json:"goals"`
	InScope            []Item `json:"in_scope"`
	OutOfScope         []Item `json:"out_of_scope"`
	Constraints        []Item `json:"constraints"`
	AcceptanceCriteria []Item `json:"acceptance_criteria"`
	Dependencies       []Item `json:"dependencies"`
}

// Baseline identifies the lifecycle and current revision of one project's requirements.
type Baseline struct {
	ID               string  `json:"id"`
	WorkspaceID      string  `json:"workspace_id"`
	ProjectID        string  `json:"project_id"`
	Status           Status  `json:"status"`
	CurrentRevision  int     `json:"current_revision"`
	ApprovedRevision *int    `json:"approved_revision"`
	SubmittedBy      *string `json:"submitted_by"`
	SubmittedAt      *string `json:"submitted_at"`
	ApprovedBy       *string `json:"approved_by"`
	ApprovedAt       *string `json:"approved_at"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
}

// Revision is one immutable snapshot in a baseline's history.
type Revision struct {
	BaselineID    string  `json:"baseline_id"`
	Revision      int     `json:"revision"`
	Content       Content `json:"content"`
	ChangeSummary string  `json:"change_summary"`
	ActorID       string  `json:"actor_id"`
	CreatedAt     string  `json:"created_at"`
	State         Status  `json:"state"`
	SubmittedBy   *string `json:"submitted_by"`
	SubmittedAt   *string `json:"submitted_at"`
	ApprovedBy    *string `json:"approved_by"`
	ApprovedAt    *string `json:"approved_at"`
}

// Record combines a baseline with current, effective, and historic content.
type Record struct {
	Baseline         Baseline   `json:"baseline"`
	CurrentContent   Content    `json:"current_content"`
	EffectiveContent *Content   `json:"effective_content"`
	History          []Revision `json:"history"`
}

// Transition is a persistence-neutral lifecycle decision. Its audit flags
// describe the business fact; the adapter supplies actor and timestamp values.
type Transition struct {
	From              Status
	To                Status
	Revision          int
	ApprovedRevision  *int
	RecordsSubmission bool
	ClearsSubmission  bool
}

// ApprovesRevision reports whether the transition publishes the current revision.
func (t Transition) ApprovesRevision() bool { return t.ApprovedRevision != nil }

// PrepareTransition validates one lifecycle transition without depending on a
// transport, clock, or database implementation.
func (b Baseline) PrepareTransition(expectedRevision int, target Status) (Transition, error) {
	if b.CurrentRevision != expectedRevision {
		return Transition{}, ErrRevisionConflict
	}

	transition := Transition{From: b.Status, To: target, Revision: expectedRevision}
	switch {
	case b.Status == StatusDraft && target == StatusInReview:
		transition.RecordsSubmission = true
	case b.Status == StatusInReview && target == StatusApproved:
		revision := expectedRevision
		transition.ApprovedRevision = &revision
	case b.Status == StatusInReview && target == StatusDraft:
		transition.ClearsSubmission = true
	default:
		return Transition{}, ErrInvalidTransition
	}
	return transition, nil
}

// DraftPlan describes whether saving a draft updates the current revision or
// creates the successor of an approved baseline.
type DraftPlan struct {
	NextRevision        int
	Status              Status
	CreatesRevision     bool
	ClearsApprovalAudit bool
	ApprovedRevision    *int
}

// PrepareDraft validates and describes the next draft persistence decision.
func (b Baseline) PrepareDraft(expectedRevision int) (DraftPlan, error) {
	if b.CurrentRevision != expectedRevision {
		return DraftPlan{}, ErrRevisionConflict
	}
	if b.Status == StatusInReview {
		return DraftPlan{}, ErrInvalidTransition
	}
	if b.Status == StatusApproved {
		return DraftPlan{
			NextRevision:        b.CurrentRevision + 1,
			Status:              StatusDraft,
			CreatesRevision:     true,
			ClearsApprovalAudit: true,
			ApprovedRevision:    b.ApprovedRevision,
		}, nil
	}
	if b.Status != StatusDraft {
		return DraftPlan{}, ErrInvalidTransition
	}
	return DraftPlan{NextRevision: b.CurrentRevision, Status: StatusDraft}, nil
}
