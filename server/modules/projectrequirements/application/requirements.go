// Package application contains use cases and ports for the versioned project
// requirement baseline.
package application

import (
	"context"
	"errors"

	"github.com/multica-ai/multica/server/modules/projectrequirements/domain"
)

type (
	// Status identifies the current lifecycle state of a baseline.
	Status = domain.Status
	// Item is one keyed statement in a requirement document.
	Item = domain.Item
	// Content is the versioned requirement document.
	Content = domain.Content
	// Baseline identifies a project's versioned requirement lifecycle.
	Baseline = domain.Baseline
	// Revision is one immutable snapshot in a baseline's history.
	Revision = domain.Revision
	// Record combines a baseline with its current, effective, and historic content.
	Record = domain.Record
)

// StatusDraft, StatusInReview, StatusApproved, and StatusSuperseded are the
// requirement-baseline lifecycle states.
const (
	StatusDraft      = domain.StatusDraft
	StatusInReview   = domain.StatusInReview
	StatusApproved   = domain.StatusApproved
	StatusSuperseded = domain.StatusSuperseded
)

// ErrNotFound, ErrRevisionConflict, ErrInvalidTransition, and
// ErrInvalidTracking are stable errors returned by this application boundary.
var (
	ErrNotFound          = domain.ErrNotFound
	ErrRevisionConflict  = domain.ErrRevisionConflict
	ErrInvalidTransition = domain.ErrInvalidTransition
	ErrInvalidTracking   = domain.ErrInvalidTracking
)

// SaveDraftInput is the command for creating or revising a draft baseline.
type SaveDraftInput struct {
	WorkspaceID      string
	ProjectID        string
	ActorID          string
	ExpectedRevision int
	Content          Content
	ChangeSummary    string
}

// TransitionInput identifies a lifecycle transition requested by an actor.
type TransitionInput struct {
	WorkspaceID      string
	ProjectID        string
	ActorID          string
	ExpectedRevision int
}

// Repository is an application-owned port. Implementations must preserve the
// compare-and-swap conditions supplied by the domain decisions so concurrent
// requests become ErrRevisionConflict rather than overwrite each other.
type Repository interface {
	Get(context.Context, string, string) (Record, error)
	CreateDraft(context.Context, SaveDraftInput) (Record, error)
	ApplyDraft(context.Context, SaveDraftInput, domain.DraftPlan) (Record, error)
	ApplyTransition(context.Context, TransitionInput, domain.Transition) (Record, error)
}

// Service executes project requirement-baseline use cases.
type Service struct{ repository Repository }

// NewService constructs the requirement-baseline use-case service.
func NewService(repository Repository) *Service { return &Service{repository: repository} }

// Get returns one project's baseline in the selected workspace.
func (s *Service) Get(ctx context.Context, workspaceID, projectID string) (Record, error) {
	return s.repository.Get(ctx, workspaceID, projectID)
}

// SaveDraft creates the first draft or applies the domain-approved draft plan.
func (s *Service) SaveDraft(ctx context.Context, input SaveDraftInput) (Record, error) {
	record, err := s.repository.Get(ctx, input.WorkspaceID, input.ProjectID)
	if errors.Is(err, ErrNotFound) {
		if input.ExpectedRevision != 0 {
			return Record{}, ErrRevisionConflict
		}
		return s.repository.CreateDraft(ctx, input)
	}
	if err != nil {
		return Record{}, err
	}
	plan, err := record.Baseline.PrepareDraft(input.ExpectedRevision)
	if err != nil {
		return Record{}, err
	}
	return s.repository.ApplyDraft(ctx, input, plan)
}

// SubmitReview moves a draft baseline into review.
func (s *Service) SubmitReview(ctx context.Context, input TransitionInput) (Record, error) {
	return s.transition(ctx, input, StatusInReview)
}

// Approve records approval of the current reviewed revision.
func (s *Service) Approve(ctx context.Context, input TransitionInput) (Record, error) {
	return s.transition(ctx, input, StatusApproved)
}

// Withdraw returns a reviewed baseline to draft.
func (s *Service) Withdraw(ctx context.Context, input TransitionInput) (Record, error) {
	return s.transition(ctx, input, StatusDraft)
}

func (s *Service) transition(ctx context.Context, input TransitionInput, target Status) (Record, error) {
	record, err := s.repository.Get(ctx, input.WorkspaceID, input.ProjectID)
	if err != nil {
		return Record{}, err
	}
	transition, err := record.Baseline.PrepareTransition(input.ExpectedRevision, target)
	if err != nil {
		return Record{}, err
	}
	return s.repository.ApplyTransition(ctx, input, transition)
}
