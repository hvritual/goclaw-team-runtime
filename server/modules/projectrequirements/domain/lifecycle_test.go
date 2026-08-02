package domain

import (
	"errors"
	"testing"
)

func TestBaselinePrepareTransitionPreservesLifecycleInvariants(t *testing.T) {
	baseline := Baseline{Status: StatusInReview, CurrentRevision: 2}

	transition, err := baseline.PrepareTransition(2, StatusApproved)
	if err != nil {
		t.Fatalf("prepare approval transition: %v", err)
	}
	if transition.From != StatusInReview || transition.To != StatusApproved || !transition.ApprovesRevision() {
		t.Fatalf("approval transition = %#v", transition)
	}
	if transition.Revision != 2 || transition.ApprovedRevision == nil || *transition.ApprovedRevision != 2 {
		t.Fatalf("approval revision audit = %#v", transition)
	}

	if _, err := baseline.PrepareTransition(1, StatusApproved); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("stale revision error = %v, want %v", err, ErrRevisionConflict)
	}
	if _, err := (Baseline{Status: StatusDraft, CurrentRevision: 2}).PrepareTransition(2, StatusApproved); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("draft approval error = %v, want %v", err, ErrInvalidTransition)
	}
}

func TestBaselinePrepareDraftKeepsApprovedRevisionEffective(t *testing.T) {
	approvedRevision := 2
	baseline := Baseline{
		Status:           StatusApproved,
		CurrentRevision:  2,
		ApprovedRevision: &approvedRevision,
	}

	draft, err := baseline.PrepareDraft(2)
	if err != nil {
		t.Fatalf("prepare follow-up draft: %v", err)
	}
	if draft.NextRevision != 3 || draft.Status != StatusDraft || !draft.CreatesRevision || !draft.ClearsApprovalAudit {
		t.Fatalf("follow-up draft = %#v", draft)
	}
	if draft.ApprovedRevision == nil || *draft.ApprovedRevision != 2 {
		t.Fatalf("follow-up draft lost effective revision: %#v", draft)
	}

	if _, err := (Baseline{Status: StatusInReview, CurrentRevision: 2}).PrepareDraft(2); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("in-review edit error = %v, want %v", err, ErrInvalidTransition)
	}
}
