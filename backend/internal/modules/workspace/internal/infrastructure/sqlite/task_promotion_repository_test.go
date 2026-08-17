package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
	issueDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/issue"
	todoDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/todo"
	_ "modernc.org/sqlite"
)

func TestTaskPromotionRepositoryCommitsAndReplaysOnePromotion(t *testing.T) {
	db := openTaskPromotionTestDB(t)
	now := time.Date(2026, 8, 18, 9, 0, 0, 0, time.UTC)
	for _, statement := range []string{
		`INSERT INTO workspaces(id,issue_prefix,next_issue_number,updated_at) VALUES('workspace-1','ONE',1,'2026-08-18T00:00:00Z')`,
		`INSERT INTO workspace_todos(id,workspace_id,title,description,status,priority,creator_type,creator_id,position,revision,created_at,updated_at) VALUES('task-1','workspace-1','Promote me','Snapshot','todo','high','member','member-1',10,1,'2026-08-18T00:00:00Z','2026-08-18T00:00:00Z')`,
		`INSERT INTO workspace_resource_revisions(workspace_id,resource_kind,resource_id,revision,updated_at) VALUES('workspace-1','task','task-1',1,'2026-08-18T00:00:00Z')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	task, err := todoDomain.Rehydrate(todoDomain.Todo{
		ID: "task-1", WorkspaceID: "workspace-1", Title: "Promote me", Description: "Snapshot",
		Status: todoDomain.StatusTodo, Priority: todoDomain.PriorityHigh, CreatorType: "member", CreatorID: "member-1",
		Position: 10, Revision: 1, CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	description := task.Description
	issue, err := issueDomain.New("issue-1", "workspace-1", task.Title, &description, issueDomain.StatusTodo, task.Priority, nil, nil, nil, nil, "member", "member-1", task.Position, nil, nil, nil, nil, now)
	if err != nil {
		t.Fatal(err)
	}
	repository, err := NewTaskPromotionRepository(Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	ctx := contract.WithWorkspaceActor(context.Background(), "member", "member-1")
	command := application.TaskPromotionCommand{
		Task: task, Issue: issue, ExpectedRevision: 1, IdempotencyKey: "promote-1", OccurredAt: now,
	}
	first, err := repository.PromoteTask(ctx, command)
	if err != nil {
		t.Fatal(err)
	}
	if first.Task.Revision != 2 || first.Task.IssueID == nil || *first.Task.IssueID != "issue-1" || first.Issue.Identifier != "ONE-1" {
		t.Fatalf("first promotion = %+v / %+v", first.Task, first.Issue)
	}
	replayed, err := repository.PromoteTask(ctx, command)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.Task.Revision != first.Task.Revision || replayed.Issue.ID != first.Issue.ID || replayed.Issue.Identifier != first.Issue.Identifier {
		t.Fatalf("replayed promotion = %+v / %+v", replayed.Task, replayed.Issue)
	}
	conflict := command
	conflict.CompleteTask = true
	if _, err := repository.PromoteTask(ctx, conflict); !errors.Is(err, contract.ErrIdempotencyConflict) {
		t.Fatalf("different-body replay error = %v", err)
	}
	replacementIssueID := "issue-2"
	relinked, err := first.Task.Apply(todoDomain.Patch{IssueID: todoDomain.StringChange{Set: true, Value: &replacementIssueID}}, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	tasks, err := NewGovernedTodoRepository(Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	if err := tasks.Update(application.WithTodoGovernanceAction(ctx, application.TaskActionUpdate), relinked); !errors.Is(err, contract.ErrTaskAlreadyLinked) {
		t.Fatalf("replace immutable promotion link error = %v", err)
	}
	retained, err := repository.FindTaskForPromotion(ctx, "workspace-1", "task-1")
	if err != nil || retained.IssueID == nil || *retained.IssueID != "issue-1" || retained.Revision != 2 {
		t.Fatalf("retained promotion link = %+v, %v", retained, err)
	}
	for table, want := range map[string]int{
		"workspace_issues":                1,
		"workspace_task_issue_promotions": 1,
		"workspace_mutation_idempotency":  1,
		"workspace_audit_entries":         1,
		"workspace_outbox_events":         1,
	} {
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil || count != want {
			t.Fatalf("%s count = %d, %v", table, count, err)
		}
	}
}

func openTaskPromotionTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(filepath.Join(t.TempDir(), "task-promotion.db")))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	base := `
CREATE TABLE workspaces(id TEXT PRIMARY KEY, issue_prefix TEXT NOT NULL, next_issue_number INTEGER NOT NULL, updated_at TEXT NOT NULL);
CREATE TABLE workspace_todos(id TEXT PRIMARY KEY, workspace_id TEXT NOT NULL, title TEXT NOT NULL, description TEXT NOT NULL, status TEXT NOT NULL, project_id TEXT, issue_id TEXT, assignee_type TEXT, assignee_id TEXT, created_at TEXT NOT NULL, updated_at TEXT NOT NULL, priority TEXT NOT NULL, creator_type TEXT NOT NULL, creator_id TEXT NOT NULL, position REAL NOT NULL, start_date TEXT, due_date TEXT, completed_at TEXT, revision INTEGER NOT NULL, restore_status TEXT, archived_at TEXT);
CREATE TABLE workspace_issues(id TEXT PRIMARY KEY, workspace_id TEXT NOT NULL, number INTEGER NOT NULL, identifier TEXT NOT NULL, title TEXT NOT NULL, description TEXT, status TEXT NOT NULL, priority TEXT NOT NULL, assignee_type TEXT, assignee_id TEXT, creator_type TEXT NOT NULL, creator_id TEXT NOT NULL, parent_issue_id TEXT, project_id TEXT, position REAL NOT NULL, stage INTEGER, start_date TEXT, due_date TEXT, metadata TEXT NOT NULL, properties TEXT NOT NULL, asset_ids TEXT NOT NULL, created_at TEXT NOT NULL, updated_at TEXT NOT NULL, UNIQUE(workspace_id,identifier));`
	if _, err := db.Exec(base); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"000009_workspace_governance.up.sql", "000012_task_issue_promotion.up.sql"} {
		schema, err := os.ReadFile(filepath.Join("migrations", name))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(string(schema)); err != nil {
			t.Fatal(err)
		}
	}
	return db
}
