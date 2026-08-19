package retrospective

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestNormalizeContentProducesCanonicalSnapshot(t *testing.T) {
	content, err := NormalizeContent(Content{
		Summary:   "  Release learning  ",
		Successes: []string{"  Small batch ", "", "Small batch"},
		Problems:  []string{" Late review "},
		Lessons:   []string{"  Review before release "},
		ActionItems: []ActionItem{{
			ID: "action-1", Title: "  Schedule review  ", Description: "  Before freeze ",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if content.Summary != "Release learning" || len(content.Successes) != 1 || content.Successes[0] != "Small batch" {
		t.Fatalf("canonical content = %#v", content)
	}
	if content.ActionItems[0].Title != "Schedule review" || content.ActionItems[0].Description != "Before freeze" {
		t.Fatalf("canonical action = %#v", content.ActionItems[0])
	}
}

func TestNormalizeContentRejectsInvalidSnapshots(t *testing.T) {
	valid := Content{Summary: "Summary", Lessons: []string{"Lesson"}}
	for _, test := range []struct {
		name    string
		content Content
	}{
		{name: "missing summary", content: Content{Lessons: []string{"Lesson"}}},
		{name: "missing lesson", content: Content{Summary: "Summary"}},
		{name: "duplicate action id", content: Content{Summary: "Summary", Lessons: []string{"Lesson"}, ActionItems: []ActionItem{{ID: "a", Title: "One"}, {ID: "a", Title: "Two"}}}},
		{name: "invalid action id", content: Content{Summary: "Summary", Lessons: []string{"Lesson"}, ActionItems: []ActionItem{{ID: "bad id", Title: "One"}}}},
		{name: "invalid due date", content: Content{Summary: "Summary", Lessons: []string{"Lesson"}, ActionItems: []ActionItem{{ID: "action-1", Title: "One", DueDate: "2026-02-30"}}}},
		{name: "oversized summary", content: Content{Summary: strings.Repeat("x", MaxSummaryRunes+1), Lessons: []string{"Lesson"}}},
		{name: "too many actions", content: Content{Summary: "Summary", Lessons: []string{"Lesson"}, ActionItems: make([]ActionItem, MaxActionItems+1)}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NormalizeContent(test.content); !errors.Is(err, ErrInvalidContent) {
				t.Fatalf("NormalizeContent() error = %v, want %v", err, ErrInvalidContent)
			}
		})
	}
	if _, err := NormalizeContent(valid); err != nil {
		t.Fatalf("valid content error = %v", err)
	}
}

func TestNormalizeParticipantsNeverProjectsBeyondSnapshotLimit(t *testing.T) {
	participants := make([]Participant, maxParticipants)
	for index := range participants {
		participants[index] = Participant{MemberID: fmt.Sprintf("member-%03d", index), Role: RoleParticipant}
	}
	if projected, err := NormalizeParticipants(participants, "creator-member"); !errors.Is(err, ErrInvalidParticipants) || projected != nil {
		t.Fatalf("projected overflow = %#v, error = %v", projected, err)
	}

	participants[0].MemberID = "creator-member"
	projected, err := NormalizeParticipants(participants, "creator-member")
	if err != nil || len(projected) != maxParticipants {
		t.Fatalf("bounded projection length = %d, error = %v", len(projected), err)
	}
}

func TestNormalizeParticipantsProjectsCreatorAndRejectsInvalidRoles(t *testing.T) {
	participants, err := NormalizeParticipants([]Participant{{MemberID: " member-2 ", Role: RoleFacilitator}}, "member-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(participants) != 2 || participants[0] != (Participant{MemberID: "member-1", Role: RoleParticipant}) || participants[1] != (Participant{MemberID: "member-2", Role: RoleFacilitator}) {
		t.Fatalf("participants = %#v", participants)
	}
	if _, err := NormalizeParticipants([]Participant{{MemberID: "member-1", Role: "owner"}}, "member-1"); !errors.Is(err, ErrInvalidParticipants) {
		t.Fatalf("invalid role error = %v", err)
	}
	if _, err := NormalizeParticipants([]Participant{{MemberID: "member-1", Role: RoleParticipant}, {MemberID: "member-1", Role: RoleParticipant}}, "member-1"); !errors.Is(err, ErrInvalidParticipants) {
		t.Fatalf("duplicate member error = %v", err)
	}
}

func TestNextStatusEnforcesRetrospectiveLifecycle(t *testing.T) {
	for _, test := range []struct{ current, action, want string }{
		{StatusDraft, ActionSaveDraft, StatusDraft},
		{StatusDraft, ActionPublish, StatusPublished},
		{StatusPublished, ActionPublishRevision, StatusPublished},
		{StatusDraft, ActionArchive, StatusArchived},
		{StatusPublished, ActionArchive, StatusArchived},
	} {
		got, err := NextStatus(test.current, test.action)
		if err != nil || got != test.want {
			t.Fatalf("NextStatus(%q,%q) = %q,%v, want %q", test.current, test.action, got, err, test.want)
		}
	}
	for _, test := range [][2]string{{StatusPublished, ActionSaveDraft}, {StatusDraft, ActionPublishRevision}, {StatusArchived, ActionPublish}, {StatusDraft, "unknown"}} {
		if _, err := NextStatus(test[0], test[1]); !errors.Is(err, ErrInvalidTransition) {
			t.Fatalf("NextStatus(%q,%q) error = %v", test[0], test[1], err)
		}
	}
}

func TestValidateLinkedActionItemsUnchanged(t *testing.T) {
	previous := Content{Summary: "One", Lessons: []string{"Lesson"}, ActionItems: []ActionItem{{ID: "linked", Title: "Ship", Description: "Exact", AssigneeID: "member-1", DueDate: "2026-08-20"}, {ID: "free", Title: "Free"}}}
	next := previous
	next.Summary = "Two"
	next.ActionItems = append([]ActionItem(nil), previous.ActionItems...)
	next.ActionItems[1].Title = "Changed free item"
	if err := ValidateLinkedActionItemsUnchanged(previous, next, map[string]struct{}{"linked": {}}); err != nil {
		t.Fatalf("unlinked edit rejected: %v", err)
	}
	next.ActionItems[0].Title = "Changed linked item"
	if err := ValidateLinkedActionItemsUnchanged(previous, next, map[string]struct{}{"linked": {}}); !errors.Is(err, ErrLinkedActionItemChanged) {
		t.Fatalf("linked edit error = %v", err)
	}
	next.ActionItems = next.ActionItems[1:]
	if err := ValidateLinkedActionItemsUnchanged(previous, next, map[string]struct{}{"linked": {}}); !errors.Is(err, ErrLinkedActionItemChanged) {
		t.Fatalf("linked removal error = %v", err)
	}
}
