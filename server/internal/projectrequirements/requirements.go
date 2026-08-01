// Package projectrequirements owns the versioned project requirement baseline
// contract. Storage adapters implement Repository without exposing database
// values through this application boundary.
package projectrequirements

import (
	"context"
	"errors"
)

type Status string

const (
	StatusDraft      Status = "draft"
	StatusInReview   Status = "in_review"
	StatusApproved   Status = "approved"
	StatusSuperseded Status = "superseded"
)

var (
	ErrNotFound          = errors.New("project requirement baseline not found")
	ErrRevisionConflict  = errors.New("project requirement revision conflict")
	ErrInvalidTransition = errors.New("invalid project requirement transition")
	ErrInvalidTracking   = errors.New("invalid project requirement tracking request")
)

type Item struct {
	Key  string `json:"key"`
	Text string `json:"text"`
}

type Content struct {
	ProblemStatement   string `json:"problem_statement"`
	Goals              []Item `json:"goals"`
	InScope            []Item `json:"in_scope"`
	OutOfScope         []Item `json:"out_of_scope"`
	Constraints        []Item `json:"constraints"`
	AcceptanceCriteria []Item `json:"acceptance_criteria"`
	Dependencies       []Item `json:"dependencies"`
}

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

type Record struct {
	Baseline         Baseline   `json:"baseline"`
	CurrentContent   Content    `json:"current_content"`
	EffectiveContent *Content   `json:"effective_content"`
	History          []Revision `json:"history"`
}

type SaveDraftInput struct {
	WorkspaceID      string
	ProjectID        string
	ActorID          string
	ExpectedRevision int
	Content          Content
	ChangeSummary    string
}

type TransitionInput struct {
	WorkspaceID      string
	ProjectID        string
	ActorID          string
	ExpectedRevision int
}

type Repository interface {
	Get(context.Context, string, string) (Record, error)
	SaveDraft(context.Context, SaveDraftInput) (Record, error)
	SubmitReview(context.Context, TransitionInput) (Record, error)
	Approve(context.Context, TransitionInput) (Record, error)
	Withdraw(context.Context, TransitionInput) (Record, error)
}

type Service struct{ repository Repository }

func NewService(repository Repository) *Service { return &Service{repository: repository} }

func (s *Service) Get(ctx context.Context, workspaceID, projectID string) (Record, error) {
	return s.repository.Get(ctx, workspaceID, projectID)
}

func (s *Service) SaveDraft(ctx context.Context, input SaveDraftInput) (Record, error) {
	return s.repository.SaveDraft(ctx, input)
}

func (s *Service) SubmitReview(ctx context.Context, input TransitionInput) (Record, error) {
	return s.repository.SubmitReview(ctx, input)
}

func (s *Service) Approve(ctx context.Context, input TransitionInput) (Record, error) {
	return s.repository.Approve(ctx, input)
}

func (s *Service) Withdraw(ctx context.Context, input TransitionInput) (Record, error) {
	return s.repository.Withdraw(ctx, input)
}
