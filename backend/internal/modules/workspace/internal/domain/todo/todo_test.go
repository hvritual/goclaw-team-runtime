package todo

import (
	"errors"
	"testing"
	"time"
)

func TestTodoStatusAndPriorityCompatibility(t *testing.T) {
	now := time.Date(2026, 8, 3, 1, 2, 3, 4, time.FixedZone("offset", 8*60*60))
	for _, status := range []string{"", StatusTodo, StatusInProgress, StatusDone, StatusCancelled} {
		for _, priority := range []string{"", PriorityUrgent, PriorityHigh, PriorityMedium, PriorityLow, PriorityNone} {
			value, err := New("t", "w", " Todo ", "details", status, priority, nil, nil, nil, nil, "member", "member-1", 2, nil, nil, now)
			if err != nil {
				t.Fatalf("status/priority %q/%q: %v", status, priority, err)
			}
			if value.Title != "Todo" || value.CreatedAt.Location() != time.UTC || value.CompletedAt != nil {
				t.Fatalf("unexpected Todo: %+v", value)
			}
		}
	}
	if _, err := New("t", "w", "Todo", "", "pending", PriorityNone, nil, nil, nil, nil, "member", "member-1", 0, nil, nil, now); !errors.Is(err, ErrInvalid) {
		t.Fatalf("invalid status error = %v", err)
	}
	if _, err := New("t", "w", "Todo", "", StatusTodo, "highest", nil, nil, nil, nil, "member", "member-1", 0, nil, nil, now); !errors.Is(err, ErrInvalid) {
		t.Fatalf("invalid priority error = %v", err)
	}
	if _, err := New("t", "w", "Todo", "", " done ", PriorityNone, nil, nil, nil, nil, "member", "member-1", 0, nil, nil, now); !errors.Is(err, ErrInvalid) {
		t.Fatalf("space-padded status error = %v", err)
	}
}

func TestTodoAssigneeAndCreatorIdentity(t *testing.T) {
	typeMember, id := "member", "member-1"
	if _, err := New("t", "w", "Todo", "", "", "", nil, nil, &typeMember, &id, "agent", "agent-1", 0, nil, nil, time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := New("t", "w", "Todo", "", "", "", nil, nil, &typeMember, nil, "member", "member-1", 0, nil, nil, time.Now()); !errors.Is(err, ErrInvalid) {
		t.Fatalf("assignee pair error = %v", err)
	}
	if _, err := New("t", "w", "Todo", "", "", "", nil, nil, nil, nil, "service", "service-1", 0, nil, nil, time.Now()); !errors.Is(err, ErrInvalid) {
		t.Fatalf("creator type error = %v", err)
	}
}

func TestTodoPatchPreservesClearsAndCompletes(t *testing.T) {
	createdAt := time.Date(2026, 8, 3, 1, 0, 0, 0, time.UTC)
	projectID, issueID, actorType, actorID := "project-1", "issue-1", "member", "member-2"
	start := createdAt.Add(time.Hour)
	value, err := New("todo-1", "workspace-1", "Title", "details", StatusTodo, PriorityNone,
		&projectID, &issueID, &actorType, &actorID, "member", "member-1", 1, &start, nil, createdAt)
	if err != nil {
		t.Fatal(err)
	}
	inProgress := StatusInProgress
	value, err = value.Apply(Patch{Status: &inProgress}, createdAt.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	now := createdAt.Add(2 * time.Hour)
	done := StatusDone
	empty := ""
	updated, err := value.Apply(Patch{
		Status:     &done,
		ProjectID:  StringChange{Set: true},
		IssueID:    StringChange{Set: true},
		AssigneeID: StringChange{Set: true},
		StartDate:  TimeChange{Set: true},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if updated.ProjectID != nil || updated.IssueID != nil || updated.AssigneeType != nil || updated.AssigneeID != nil || updated.StartDate != nil {
		t.Fatalf("clear patch failed: %+v", updated)
	}
	if updated.CompletedAt == nil || !updated.CompletedAt.Equal(now) {
		t.Fatalf("completed_at = %v", updated.CompletedAt)
	}
	later := now.Add(time.Hour)
	updatedAgain, err := updated.Apply(Patch{Status: &done}, later)
	if err != nil || updatedAgain.CompletedAt == nil || !updatedAgain.CompletedAt.Equal(now) {
		t.Fatalf("same done transition = %+v, %v", updatedAgain, err)
	}
	cancelled := StatusCancelled
	if _, err := updatedAgain.Apply(Patch{Status: &cancelled}, later); !errors.Is(err, ErrInvalid) {
		t.Fatalf("done -> cancelled error = %v, want %v", err, ErrInvalid)
	}
	cancelCandidate, err := New("todo-2", "workspace-1", "Cancel", "", StatusInProgress, PriorityNone,
		&projectID, nil, nil, nil, "member", "member-1", 2, nil, nil, createdAt)
	if err != nil {
		t.Fatal(err)
	}
	cancelledValue, err := cancelCandidate.Apply(Patch{Status: &cancelled, ProjectID: StringChange{Set: true, Value: &empty}}, later)
	if err != nil || cancelledValue.CompletedAt != nil {
		t.Fatalf("non-done status must clear completion: %+v, %v", cancelledValue, err)
	}
}

func TestTodoPatchRejectsInvalidPartialValuesAndCopiesPointers(t *testing.T) {
	createdAt := time.Date(2026, 8, 3, 1, 0, 0, 0, time.UTC)
	value, err := New("todo-1", "workspace-1", "Title", "", StatusTodo, PriorityNone, nil, nil, nil, nil, "member", "member-1", 0, nil, nil, createdAt)
	if err != nil {
		t.Fatal(err)
	}
	blank := "   "
	if _, err := value.Apply(Patch{Title: &blank}, createdAt); !errors.Is(err, ErrInvalid) {
		t.Fatalf("blank title error = %v", err)
	}
	assigneeType := "agent"
	updated, err := value.Apply(Patch{
		AssigneeType: StringChange{Set: true, Value: &assigneeType},
		AssigneeID:   StringChange{Set: true, Value: stringPointer("agent-1")},
	}, createdAt)
	if err != nil {
		t.Fatal(err)
	}
	assigneeType = "member"
	if updated.AssigneeType == nil || *updated.AssigneeType != "agent" {
		t.Fatalf("assignee pointer leaked: %+v", updated.AssigneeType)
	}
}

func TestTodoRejectsLifecycleJumps(t *testing.T) {
	createdAt := time.Date(2026, 8, 18, 2, 0, 0, 0, time.UTC)
	value, err := New("todo-1", "workspace-1", "Title", "", StatusTodo, PriorityNone, nil, nil, nil, nil, "member", "member-1", 0, nil, nil, createdAt)
	if err != nil {
		t.Fatal(err)
	}

	done := StatusDone
	if _, err := value.Apply(Patch{Status: &done}, createdAt.Add(time.Minute)); !errors.Is(err, ErrInvalid) {
		t.Fatalf("todo -> done error = %v, want %v", err, ErrInvalid)
	}
}

func TestTodoRevisionAdvancesOnChange(t *testing.T) {
	createdAt := time.Date(2026, 8, 18, 2, 0, 0, 0, time.UTC)
	value, err := New("todo-1", "workspace-1", "Title", "", StatusTodo, PriorityNone, nil, nil, nil, nil, "member", "member-1", 0, nil, nil, createdAt)
	if err != nil {
		t.Fatal(err)
	}
	if value.Revision != 1 {
		t.Fatalf("created revision = %d, want 1", value.Revision)
	}

	title := "Updated"
	updated, err := value.Apply(Patch{Title: &title}, createdAt.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if updated.Revision != 2 {
		t.Fatalf("updated revision = %d, want 2", updated.Revision)
	}
}

func TestTodoArchiveRestorePreservesTerminalState(t *testing.T) {
	createdAt := time.Date(2026, 8, 18, 2, 0, 0, 0, time.UTC)
	value, err := New("todo-1", "workspace-1", "Title", "", StatusInProgress, PriorityNone, nil, nil, nil, nil, "member", "member-1", 0, nil, nil, createdAt)
	if err != nil {
		t.Fatal(err)
	}
	done := StatusDone
	value, err = value.Apply(Patch{Status: &done}, createdAt.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	completedAt := value.CompletedAt

	archived, err := value.Archive(createdAt.Add(2 * time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if archived.Status != StatusArchived || archived.RestoreStatus != StatusDone || archived.ArchivedAt == nil || archived.Revision != value.Revision+1 {
		t.Fatalf("archived task = %+v", archived)
	}

	restored, err := archived.Restore(createdAt.Add(3 * time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if restored.Status != StatusDone || restored.RestoreStatus != "" || restored.ArchivedAt != nil || restored.Revision != archived.Revision+1 {
		t.Fatalf("restored task = %+v", restored)
	}
	if restored.CompletedAt == nil || completedAt == nil || !restored.CompletedAt.Equal(*completedAt) {
		t.Fatalf("completed_at changed during archive/restore: before=%v after=%v", completedAt, restored.CompletedAt)
	}
}

func stringPointer(value string) *string { return &value }
