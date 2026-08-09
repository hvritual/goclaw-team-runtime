package issue

import (
	"testing"
	"time"
)

func TestIssueVocabularyAndDefaults(t *testing.T) {
	for _, status := range []string{"backlog", "todo", "in_progress", "in_review", "done", "blocked", "cancelled"} {
		if !ValidStatus(status) {
			t.Fatalf("status %q rejected", status)
		}
	}
	for _, priority := range []string{"urgent", "high", "medium", "low", "none"} {
		if !ValidPriority(priority) {
			t.Fatalf("priority %q rejected", priority)
		}
	}
	value, err := New("issue-1", "workspace-1", "  Ship  ", nil, "", "", nil, nil, nil, nil, "member", "member-1", 0, nil, nil, nil, nil, time.Now())
	if err != nil || value.Title != "Ship" || value.Status != StatusTodo || value.Priority != PriorityNone {
		t.Fatalf("New() = %+v, %v", value, err)
	}
	for _, invalid := range []string{"", "DONE", " todo "} {
		if invalid != "" && ValidStatus(invalid) {
			t.Fatalf("invalid status %q accepted", invalid)
		}
	}
}

func TestIssueIdentityPatchClearAndProjectionCopies(t *testing.T) {
	now := time.Date(2026, 8, 3, 1, 2, 3, 0, time.FixedZone("CST", 8*60*60))
	description, actorType, actorID := "details", "agent", "agent-1"
	projectID, parentID, startDate, dueDate := "project-1", "parent-1", "2026-08-04", "2026-08-05"
	stage := int32(2)
	value, err := New("issue-1", "workspace-1", "Ship", &description, "todo", "high", &actorType, &actorID, &parentID, &projectID, "member", "member-1", 3, &stage, &startDate, &dueDate, []string{"asset-1"}, now)
	if err != nil {
		t.Fatal(err)
	}
	value, err = value.AssignIdentity(7, "WSP")
	if err != nil || value.Identifier != "WSP-7" || value.CreatedAt.Location() != time.UTC {
		t.Fatalf("AssignIdentity() = %+v, %v", value, err)
	}
	empty := ""
	updated, err := value.Apply(Patch{
		AssigneeType: StringChange{Set: true, Value: &empty}, AssigneeID: StringChange{Set: true, Value: &empty},
		ParentIssueID: StringChange{Set: true}, ProjectID: StringChange{Set: true},
		Stage: StageChange{Set: true}, StartDate: StringChange{Set: true}, DueDate: StringChange{Set: true},
		AssetIDs: AssetsChange{Set: true},
	}, now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if updated.AssigneeID != nil || updated.ParentIssueID != nil || updated.ProjectID != nil || updated.Stage != nil || updated.StartDate != nil || updated.DueDate != nil || len(updated.AssetIDs) != 0 {
		t.Fatalf("clear patch = %+v", updated)
	}
	value.Metadata["nested"] = map[string]any{"key": "value"}
	rehydrated, err := Rehydrate(value)
	if err != nil {
		t.Fatal(err)
	}
	rehydrated.AssetIDs[0] = "changed"
	rehydrated.Metadata["nested"].(map[string]any)["key"] = "changed"
	if value.AssetIDs[0] != "asset-1" || value.Metadata["nested"].(map[string]any)["key"] != "value" {
		t.Fatal("rehydration leaked mutable projection state")
	}
}

func TestIssueRejectsInvalidPairsDatesAndCycles(t *testing.T) {
	actorType, self, badDate := "member", "issue-1", "2026-8-3"
	stage := int32(0)
	tests := []struct {
		name                     string
		assigneeType, assigneeID *string
		parentID, startDate      *string
		stage                    *int32
	}{
		{name: "half actor", assigneeType: &actorType},
		{name: "self parent", parentID: &self},
		{name: "zero stage", stage: &stage},
		{name: "bad date", startDate: &badDate},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := New("issue-1", "workspace-1", "Ship", nil, "todo", "none", test.assigneeType, test.assigneeID, test.parentID, nil, "member", "member-1", 0, test.stage, test.startDate, nil, nil, time.Now()); err == nil {
				t.Fatal("New() error = nil")
			}
		})
	}
}
