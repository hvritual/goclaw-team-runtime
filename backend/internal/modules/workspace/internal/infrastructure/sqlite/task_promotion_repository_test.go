package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"sync"
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
	secondPromotion := command
	secondPromotion.Task = first.Task
	secondPromotion.ExpectedRevision = first.Task.Revision
	secondPromotion.IdempotencyKey = "promote-2"
	secondPromotion.Issue = newTaskPromotionIssue(t, task, "issue-2", now.Add(time.Second))
	if _, err := repository.PromoteTask(ctx, secondPromotion); !errors.Is(err, contract.ErrTaskAlreadyLinked) {
		t.Fatalf("second promotion error = %v", err)
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
	renamed, err := first.Task.Apply(todoDomain.Patch{Title: stringPointer("Renamed Task only")}, now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if err := tasks.Update(application.WithTodoGovernanceAction(ctx, application.TaskActionUpdate), renamed); err != nil {
		t.Fatalf("unrelated promoted Task edit: %v", err)
	}
	if _, err := db.Exec(`UPDATE workspace_issues SET title='Renamed Issue only' WHERE id='issue-1'`); err != nil {
		t.Fatal(err)
	}
	retained, err := repository.FindTaskForPromotion(ctx, "workspace-1", "task-1")
	if err != nil || retained.Title != "Renamed Task only" || retained.IssueID == nil || *retained.IssueID != "issue-1" {
		t.Fatalf("independent later edits = %+v, %v", retained, err)
	}
	for table, want := range map[string]int{
		"workspace_issues":                1,
		"workspace_task_issue_promotions": 1,
		"workspace_mutation_idempotency":  1,
		"workspace_audit_entries":         2,
		"workspace_outbox_events":         2,
	} {
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil || count != want {
			t.Fatalf("%s count = %d, %v", table, count, err)
		}
	}
}

func TestTaskPromotionRepositorySerializesConcurrentAttempts(t *testing.T) {
	db := openTaskPromotionTestDB(t)
	now := time.Date(2026, 8, 18, 10, 0, 0, 0, time.UTC)
	task := seedTaskPromotion(t, db, now)
	repository, err := NewTaskPromotionRepository(Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	ctx := contract.WithWorkspaceActor(context.Background(), "member", "member-1")
	commands := []application.TaskPromotionCommand{
		newTaskPromotionCommand(t, task, "issue-a", "concurrent-a", now),
		newTaskPromotionCommand(t, task, "issue-b", "concurrent-b", now),
	}
	errs := make(chan error, len(commands))
	var wait sync.WaitGroup
	for _, command := range commands {
		wait.Add(1)
		go func(command application.TaskPromotionCommand) {
			defer wait.Done()
			_, err := repository.PromoteTask(ctx, command)
			errs <- err
		}(command)
	}
	wait.Wait()
	close(errs)
	successes, conflicts := 0, 0
	for err := range errs {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, contract.ErrTaskAlreadyLinked):
			conflicts++
		default:
			var revisionConflict contract.RevisionConflictError
			if errors.As(err, &revisionConflict) {
				conflicts++
			} else {
				t.Fatalf("unexpected concurrent promotion error: %v", err)
			}
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("successes=%d conflicts=%d", successes, conflicts)
	}
	assertTaskPromotionCounts(t, db, 1, 1, 1)
}

func TestTaskPromotionRepositoryRollsBackEveryGovernancePhase(t *testing.T) {
	for _, phase := range []GovernancePhase{GovernanceAfterDomain, GovernanceAfterRevision, GovernanceAfterAudit, GovernanceAfterOutbox, GovernanceAfterReplay} {
		t.Run(string(phase), func(t *testing.T) {
			db := openTaskPromotionTestDB(t)
			now := time.Date(2026, 8, 18, 11, 0, 0, 0, time.UTC)
			task := seedTaskPromotion(t, db, now)
			repository, err := NewTaskPromotionRepository(Config{DB: db})
			if err != nil {
				t.Fatal(err)
			}
			repository.governance, err = NewGovernanceRepository(
				Config{DB: db},
				WithGovernanceEventPolicies(application.TaskGovernancePolicyProvider{}),
				WithGovernanceFailureHook(func(observed GovernancePhase) error {
					if observed == phase {
						return errors.New("injected promotion failure")
					}
					return nil
				}),
			)
			if err != nil {
				t.Fatal(err)
			}
			ctx := contract.WithWorkspaceActor(context.Background(), "member", "member-1")
			if _, err := repository.PromoteTask(ctx, newTaskPromotionCommand(t, task, "issue-rollback", "rollback", now)); err == nil {
				t.Fatal("expected injected failure")
			}
			assertTaskPromotionCounts(t, db, 0, 0, 1)
			var issueID sql.NullString
			var revision, nextNumber int
			if err := db.QueryRow(`SELECT issue_id,revision FROM workspace_todos WHERE id='task-1'`).Scan(&issueID, &revision); err != nil {
				t.Fatal(err)
			}
			if err := db.QueryRow(`SELECT next_issue_number FROM workspaces WHERE id='workspace-1'`).Scan(&nextNumber); err != nil {
				t.Fatal(err)
			}
			if issueID.Valid || revision != 1 || nextNumber != 1 {
				t.Fatalf("rollback task issue=%v revision=%d next=%d", issueID, revision, nextNumber)
			}
		})
	}
}

func TestTaskPromotionRepositoryReplaysAfterDatabaseRestart(t *testing.T) {
	db := openTaskPromotionTestDB(t)
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	task := seedTaskPromotion(t, db, now)
	repository, err := NewTaskPromotionRepository(Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	ctx := contract.WithWorkspaceActor(context.Background(), "member", "member-1")
	command := newTaskPromotionCommand(t, task, "issue-restart", "restart", now)
	first, err := repository.PromoteTask(ctx, command)
	if err != nil {
		t.Fatal(err)
	}
	var sequence int
	var databaseName, databasePath string
	if err := db.QueryRow(`PRAGMA database_list`).Scan(&sequence, &databaseName, &databasePath); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := sql.Open("sqlite", "file:"+filepath.ToSlash(databasePath))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	restarted, err := NewTaskPromotionRepository(Config{DB: reopened})
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := restarted.PromoteTask(ctx, command)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.Task.Revision != first.Task.Revision || replayed.Issue.Identifier != first.Issue.Identifier {
		t.Fatalf("restart replay = %+v / %+v", replayed.Task, replayed.Issue)
	}
	assertTaskPromotionCounts(t, reopened, 1, 1, 1)
}

func seedTaskPromotion(t *testing.T, db *sql.DB, now time.Time) todoDomain.Todo {
	t.Helper()
	for _, statement := range []string{
		`INSERT INTO workspaces(id,issue_prefix,next_issue_number,updated_at) VALUES('workspace-1','ONE',1,'2026-08-18T00:00:00Z')`,
		`INSERT INTO workspace_todos(id,workspace_id,title,description,status,priority,creator_type,creator_id,position,revision,created_at,updated_at) VALUES('task-1','workspace-1','Promote me','Snapshot','todo','high','member','member-1',10,1,'2026-08-18T00:00:00Z','2026-08-18T00:00:00Z')`,
		`INSERT INTO workspace_resource_revisions(workspace_id,resource_kind,resource_id,revision,updated_at) VALUES('workspace-1','task','task-1',1,'2026-08-18T00:00:00Z')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	task, err := todoDomain.Rehydrate(todoDomain.Todo{ID: "task-1", WorkspaceID: "workspace-1", Title: "Promote me", Description: "Snapshot", Status: todoDomain.StatusTodo, Priority: todoDomain.PriorityHigh, CreatorType: "member", CreatorID: "member-1", Position: 10, Revision: 1, CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	return task
}

func newTaskPromotionIssue(t *testing.T, task todoDomain.Todo, issueID string, now time.Time) issueDomain.Issue {
	t.Helper()
	description := task.Description
	issue, err := issueDomain.New(issueID, task.WorkspaceID, task.Title, &description, issueDomain.StatusTodo, task.Priority, nil, nil, nil, nil, "member", "member-1", task.Position, nil, nil, nil, nil, now)
	if err != nil {
		t.Fatal(err)
	}
	return issue
}

func newTaskPromotionCommand(t *testing.T, task todoDomain.Todo, issueID, key string, now time.Time) application.TaskPromotionCommand {
	t.Helper()
	return application.TaskPromotionCommand{Task: task, Issue: newTaskPromotionIssue(t, task, issueID, now), ExpectedRevision: task.Revision, IdempotencyKey: key, OccurredAt: now}
}

func stringPointer(value string) *string { return &value }

func assertTaskPromotionCounts(t *testing.T, db *sql.DB, promotions, mutations, revisions int) {
	t.Helper()
	for table, want := range map[string]int{
		"workspace_issues": promotions, "workspace_task_issue_promotions": promotions,
		"workspace_mutation_idempotency": mutations, "workspace_audit_entries": mutations,
		"workspace_outbox_events": mutations, "workspace_resource_revisions": revisions,
	} {
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil || count != want {
			t.Fatalf("%s count=%d want=%d err=%v", table, count, want, err)
		}
	}
	if promotions == 1 {
		var revision, nextNumber int
		if err := db.QueryRow(`SELECT revision FROM workspace_resource_revisions WHERE workspace_id='workspace-1' AND resource_kind='task' AND resource_id='task-1'`).Scan(&revision); err != nil || revision != 2 {
			t.Fatalf("resource revision=%d want=2 err=%v", revision, err)
		}
		if err := db.QueryRow(`SELECT next_issue_number FROM workspaces WHERE id='workspace-1'`).Scan(&nextNumber); err != nil || nextNumber != 2 {
			t.Fatalf("next Issue number=%d want=2 err=%v", nextNumber, err)
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
