package http

import "testing"

func TestSortIssuesUsesCanonicalRanksNullsAndTieBreakers(t *testing.T) {
	status := []publicIssue{{ID: "blocked", Status: "blocked"}, {ID: "done", Status: "done"}, {ID: "todo", Status: "todo"}, {ID: "backlog", Status: "backlog"}, {ID: "cancelled", Status: "cancelled"}, {ID: "review", Status: "in_review"}, {ID: "progress", Status: "in_progress"}}
	sortIssues(status, "status", "asc")
	assertIssueOrder(t, status, "backlog", "todo", "progress", "review", "done", "blocked", "cancelled")
	priority := []publicIssue{{ID: "none", Priority: "none"}, {ID: "low", Priority: "low"}, {ID: "urgent", Priority: "urgent"}, {ID: "medium", Priority: "medium"}, {ID: "high", Priority: "high"}}
	sortIssues(priority, "priority", "asc")
	assertIssueOrder(t, priority, "urgent", "high", "medium", "low", "none")
	a, b := "2026-01-01", "2026-01-02"
	dates := []publicIssue{{ID: "nil", CreatedAt: "2026-01-03"}, {ID: "a", StartDate: &a, CreatedAt: "2026-01-01"}, {ID: "b", StartDate: &b, CreatedAt: "2026-01-02"}}
	sortIssues(dates, "start_date", "asc")
	assertIssueOrder(t, dates, "a", "b", "nil")
	sortIssues(dates, "start_date", "desc")
	assertIssueOrder(t, dates, "b", "a", "nil")
	ties := []publicIssue{{ID: "a", Title: "same", CreatedAt: "2026-01-02"}, {ID: "c", Title: "same", CreatedAt: "2026-01-02"}, {ID: "b", Title: "same", CreatedAt: "2026-01-03"}}
	sortIssues(ties, "title", "asc")
	assertIssueOrder(t, ties, "b", "c", "a")
	positions := []publicIssue{{ID: "two", Position: 2}, {ID: "one", Position: 1}}
	sortIssues(positions, "position", "desc")
	assertIssueOrder(t, positions, "one", "two")
}

func assertIssueOrder(t *testing.T, issues []publicIssue, ids ...string) {
	t.Helper()
	for index, id := range ids {
		if issues[index].ID != id {
			t.Fatalf("index %d = %s, want %s", index, issues[index].ID, id)
		}
	}
}
