package requirement

import (
	"errors"
	"testing"
	"time"
)

func TestBaselineLifecyclePreservesEffectiveRevisionAcrossMaterialRereview(t *testing.T) {
	now := time.Date(2026, 8, 19, 8, 0, 0, 0, time.UTC)
	baseline, revision, err := NewBaseline(
		"baseline-1",
		"workspace-1",
		"project-1",
		testBaselineContent("Ship the governed Requirement"),
		"Initial baseline",
		"author-1",
		now,
	)
	if err != nil {
		t.Fatalf("NewBaseline() error = %v", err)
	}
	assertBaselineSnapshot(t, baseline, revision, StatusDraft, 1, nil, ActionCreate)

	baseline, revision, err = baseline.SubmitReview(1, "author-1", now.Add(time.Minute))
	if err != nil {
		t.Fatalf("SubmitReview() error = %v", err)
	}
	assertBaselineSnapshot(t, baseline, revision, StatusInReview, 2, nil, ActionSubmitReview)

	baseline, revision, err = baseline.Approve(2, "approver-1", now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("Approve() error = %v", err)
	}
	assertBaselineSnapshot(t, baseline, revision, StatusApproved, 3, nil, ActionApprove)

	baseline, revision, err = baseline.Freeze(3, "approver-1", now.Add(3*time.Minute))
	if err != nil {
		t.Fatalf("Freeze() error = %v", err)
	}
	effective := int64(4)
	assertBaselineSnapshot(t, baseline, revision, StatusFrozen, 4, &effective, ActionFreeze)

	changed := testBaselineContent("Ship the governed Requirement safely")
	baseline, revision, err = baseline.SaveDraft(4, changed, "Material safety change", "author-1", true, now.Add(4*time.Minute))
	if err != nil {
		t.Fatalf("SaveDraft(material) error = %v", err)
	}
	assertBaselineSnapshot(t, baseline, revision, StatusChanged, 5, &effective, ActionMaterialChange)

	baseline, _, err = baseline.SubmitReview(5, "author-1", now.Add(5*time.Minute))
	if err != nil {
		t.Fatalf("SubmitReview(changed) error = %v", err)
	}
	baseline, _, err = baseline.WithdrawReview(6, "author-1", now.Add(6*time.Minute))
	if err != nil {
		t.Fatalf("WithdrawReview() error = %v", err)
	}
	if baseline.Status != StatusChanged {
		t.Fatalf("WithdrawReview() status = %q, want %q", baseline.Status, StatusChanged)
	}

	baseline, _, err = baseline.SubmitReview(7, "author-1", now.Add(7*time.Minute))
	if err != nil {
		t.Fatalf("SubmitReview(second) error = %v", err)
	}
	baseline, _, err = baseline.Approve(8, "approver-1", now.Add(8*time.Minute))
	if err != nil {
		t.Fatalf("Approve(second) error = %v", err)
	}
	baseline, revision, err = baseline.Freeze(9, "approver-1", now.Add(9*time.Minute))
	if err != nil {
		t.Fatalf("Freeze(second) error = %v", err)
	}
	effective = 10
	assertBaselineSnapshot(t, baseline, revision, StatusFrozen, 10, &effective, ActionFreeze)

	baseline, revision, err = baseline.Retire(10, "approver-1", now.Add(10*time.Minute))
	if err != nil {
		t.Fatalf("Retire() error = %v", err)
	}
	assertBaselineSnapshot(t, baseline, revision, StatusRetired, 11, &effective, ActionRetire)
}

func TestBaselineRejectsStaleInvalidAndSelfApprovedMutations(t *testing.T) {
	now := time.Date(2026, 8, 19, 8, 30, 0, 0, time.UTC)
	baseline, _, err := NewBaseline("baseline-1", "workspace-1", "project-1", testBaselineContent("Initial"), "Initial", "author-1", now)
	if err != nil {
		t.Fatalf("NewBaseline() error = %v", err)
	}
	if _, _, err = baseline.SubmitReview(0, "author-1", now); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("SubmitReview(stale) error = %v, want ErrRevisionConflict", err)
	}
	baseline, _, err = baseline.SubmitReview(1, "author-1", now)
	if err != nil {
		t.Fatalf("SubmitReview() error = %v", err)
	}
	if _, _, err = baseline.Approve(2, "author-1", now); !errors.Is(err, ErrIndependentApprovalRequired) {
		t.Fatalf("Approve(self) error = %v, want ErrIndependentApprovalRequired", err)
	}
	baseline, _, err = baseline.Approve(2, "approver-1", now)
	if err != nil {
		t.Fatalf("Approve() error = %v", err)
	}
	baseline, _, err = baseline.Freeze(3, "approver-1", now)
	if err != nil {
		t.Fatalf("Freeze() error = %v", err)
	}
	if _, _, err = baseline.SaveDraft(4, testBaselineContent("Changed"), "Plain edit", "author-1", false, now); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("SaveDraft(frozen plain) error = %v, want ErrInvalidTransition", err)
	}
	if _, _, err = baseline.SaveDraft(4, baseline.Content, "No-op material edit", "author-1", true, now); !errors.Is(err, ErrMaterialChangeRequired) {
		t.Fatalf("SaveDraft(material no-op) error = %v, want ErrMaterialChangeRequired", err)
	}
}

func TestBaselineTraceabilityMutationAdvancesFrozenEffectiveRevision(t *testing.T) {
	now := time.Date(2026, 8, 19, 9, 0, 0, 0, time.UTC)
	baseline, _, err := NewBaseline("baseline-1", "workspace-1", "project-1", testBaselineContent("Initial"), "Initial", "author-1", now)
	if err != nil {
		t.Fatalf("NewBaseline() error = %v", err)
	}
	baseline, _, _ = baseline.SubmitReview(1, "author-1", now)
	baseline, _, _ = baseline.Approve(2, "approver-1", now)
	baseline, _, _ = baseline.Freeze(3, "approver-1", now)

	baseline, revision, err := baseline.RecordTraceabilityMutation(4, ActionLinkIssue, "editor-1", now)
	if err != nil {
		t.Fatalf("RecordTraceabilityMutation() error = %v", err)
	}
	effective := int64(5)
	assertBaselineSnapshot(t, baseline, revision, StatusFrozen, 5, &effective, ActionLinkIssue)
}

func TestBaselineAllowsOnlySystemIssueCleanupAfterRetirement(t *testing.T) {
	now := time.Date(2026, 8, 19, 8, 45, 0, 0, time.UTC)
	baseline, _, err := NewBaseline("baseline-1", "workspace-1", "project-1", testBaselineContent("Initial"), "Initial", "author-1", now)
	if err != nil {
		t.Fatal(err)
	}
	baseline, _, err = baseline.Retire(1, "approver-1", now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = baseline.RecordTraceabilityMutation(2, ActionLinkIssue, "editor-1", now.Add(2*time.Minute)); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("RecordTraceabilityMutation(retired user link) error = %v", err)
	}
	cleaned, revision, err := baseline.RecordTraceabilityMutation(2, ActionIssueDeleted, "system:issue-deletion", now.Add(3*time.Minute))
	if err != nil {
		t.Fatalf("RecordTraceabilityMutation(retired cleanup) error = %v", err)
	}
	if cleaned.Status != StatusRetired || cleaned.CurrentRevision != 3 || revision.Action != ActionIssueDeleted || revision.ActorID != "system:issue-deletion" {
		t.Fatalf("retired cleanup = baseline %#v revision %#v", cleaned, revision)
	}
}

func testBaselineContent(problem string) Content {
	return Content{
		ProblemStatement: problem,
		Goals:            []Item{{Key: "goal-1", Text: "Deliver the project"}},
		InScope:          []Item{{Key: "scope-1", Text: "Canonical Workspace"}},
		OutOfScope:       []Item{{Key: "out-1", Text: "Legacy server"}},
		Constraints:      []Item{{Key: "constraint-1", Text: "No dual write"}},
		AcceptanceCriteria: []Item{{
			Key:  "acceptance-1",
			Text: "Independent approval passes",
		}},
		Dependencies: []Item{{Key: "dependency-1", Text: "Project Registry"}},
	}
}

func assertBaselineSnapshot(t *testing.T, baseline Baseline, revision Revision, status Status, current int64, effective *int64, action Action) {
	t.Helper()
	if baseline.Status != status || revision.Status != status {
		t.Fatalf("status = baseline %q revision %q, want %q", baseline.Status, revision.Status, status)
	}
	if baseline.CurrentRevision != current || revision.Revision != current {
		t.Fatalf("revision = baseline %d snapshot %d, want %d", baseline.CurrentRevision, revision.Revision, current)
	}
	if !equalOptionalRevision(baseline.EffectiveRevision, effective) {
		t.Fatalf("effective revision = %v, want %v", baseline.EffectiveRevision, effective)
	}
	if revision.Action != action {
		t.Fatalf("action = %q, want %q", revision.Action, action)
	}
}

func equalOptionalRevision(left, right *int64) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}
