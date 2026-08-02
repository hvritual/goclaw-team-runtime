package application

import (
	"context"
	"errors"
	"testing"

	"github.com/multica-ai/multica/server/modules/projectrequirements/domain"
)

func TestServiceApprovesOnlyAReviewedCurrentRevision(t *testing.T) {
	t.Run("rejects an approval from draft without asking persistence to apply it", func(t *testing.T) {
		repository := &fakeRepository{
			record:             Record{Baseline: Baseline{Status: StatusDraft, CurrentRevision: 2}},
			unexpectedApplyErr: errors.New("persistence must not receive an invalid transition"),
		}

		_, err := NewService(repository).Approve(context.Background(), TransitionInput{
			WorkspaceID: "workspace-1", ProjectID: "project-1", ActorID: "lead-1", ExpectedRevision: 2,
		})
		if !errors.Is(err, ErrInvalidTransition) {
			t.Fatalf("approve draft error = %v, want %v", err, ErrInvalidTransition)
		}
	})

	t.Run("sends the approved revision decision to the persistence port", func(t *testing.T) {
		expected := Record{Baseline: Baseline{Status: StatusApproved, CurrentRevision: 2}}
		repository := &fakeRepository{
			record:                Record{Baseline: Baseline{Status: StatusInReview, CurrentRevision: 2}},
			applyTransitionResult: expected,
		}

		got, err := NewService(repository).Approve(context.Background(), TransitionInput{
			WorkspaceID: "workspace-1", ProjectID: "project-1", ActorID: "lead-1", ExpectedRevision: 2,
		})
		if err != nil {
			t.Fatalf("approve reviewed baseline: %v", err)
		}
		if got.Baseline.Status != StatusApproved {
			t.Fatalf("approved record = %#v", got)
		}
		if repository.transition.From != StatusInReview || repository.transition.To != StatusApproved || !repository.transition.ApprovesRevision() {
			t.Fatalf("persistence transition = %#v", repository.transition)
		}
	})
}

type fakeRepository struct {
	record                Record
	getErr                error
	applyTransitionResult Record
	unexpectedApplyErr    error
	transition            domain.Transition
}

func (r *fakeRepository) Get(context.Context, string, string) (Record, error) {
	return r.record, r.getErr
}

func (*fakeRepository) CreateDraft(context.Context, SaveDraftInput) (Record, error) {
	return Record{}, errors.New("unexpected draft creation")
}

func (*fakeRepository) ApplyDraft(context.Context, SaveDraftInput, domain.DraftPlan) (Record, error) {
	return Record{}, errors.New("unexpected draft update")
}

func (r *fakeRepository) ApplyTransition(_ context.Context, _ TransitionInput, transition domain.Transition) (Record, error) {
	r.transition = transition
	if r.unexpectedApplyErr != nil {
		return Record{}, r.unexpectedApplyErr
	}
	if r.applyTransitionResult.Baseline.Status == "" {
		return Record{}, errors.New("persistence must not receive an invalid transition")
	}
	return r.applyTransitionResult, nil
}
