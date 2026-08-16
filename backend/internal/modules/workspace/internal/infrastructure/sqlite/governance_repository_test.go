package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
	_ "modernc.org/sqlite"
)

func TestGovernanceRepositoryReplaysCommittedMutationWithoutRepeatingEffects(t *testing.T) {
	db := openGovernanceTestDB(t)
	repository, err := NewGovernanceRepository(Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	prepared := prepareGovernanceTestMutation(t, "workspace-1", "workspace.task.create", "command-1", strings.Repeat("a", 64))
	applyCount := 0
	apply := func(ctx context.Context, connection *sql.Conn) error {
		applyCount++
		_, err := connection.ExecContext(ctx, `INSERT INTO test_domain_mutations(id) VALUES ('task-1')`)
		return err
	}

	first, err := repository.Execute(context.Background(), prepared, apply)
	if err != nil {
		t.Fatal(err)
	}
	second, err := repository.Execute(context.Background(), prepared, apply)
	if err != nil {
		t.Fatal(err)
	}
	if first.ResourceRevision != 1 || first.Replayed {
		t.Fatalf("first result = %+v", first)
	}
	if second.ResourceRevision != 1 || !second.Replayed || string(second.ResponseBody) != `{"id":"task-1"}` {
		t.Fatalf("replayed result = %+v", second)
	}
	if applyCount != 1 {
		t.Fatalf("domain apply count = %d, want 1", applyCount)
	}
	assertGovernanceRowCount(t, db, "test_domain_mutations", 1)
	assertGovernanceRowCount(t, db, "workspace_resource_revisions", 1)
	assertGovernanceRowCount(t, db, "workspace_audit_entries", 1)
	assertGovernanceRowCount(t, db, "workspace_outbox_events", 1)
	assertGovernanceRowCount(t, db, "workspace_mutation_idempotency", 1)
}

func TestGovernanceRepositoryRejectsSameKeyWithDifferentHash(t *testing.T) {
	db := openGovernanceTestDB(t)
	repository, err := NewGovernanceRepository(Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	first := prepareGovernanceTestMutation(t, "workspace-1", "workspace.task.create", "command-1", strings.Repeat("a", 64))
	if _, err := repository.Execute(context.Background(), first, insertTestDomainMutation("first")); err != nil {
		t.Fatal(err)
	}
	conflict := prepareGovernanceTestMutation(t, "workspace-1", "workspace.task.create", "command-1", strings.Repeat("b", 64))
	if _, err := repository.Execute(context.Background(), conflict, insertTestDomainMutation("second")); !errors.Is(err, contract.ErrIdempotencyConflict) {
		t.Fatalf("error = %v, want idempotency conflict", err)
	}
	assertGovernanceRowCount(t, db, "test_domain_mutations", 1)
}

func TestGovernanceRepositorySerializesConcurrentExpectedRevision(t *testing.T) {
	db := openGovernanceTestDB(t)
	db.SetMaxOpenConns(4)
	repository, err := NewGovernanceRepository(Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	mutations := []application.PreparedGovernanceMutation{
		prepareGovernanceTestMutation(t, "workspace-1", "workspace.task.update", "command-a", strings.Repeat("a", 64)),
		prepareGovernanceTestMutation(t, "workspace-1", "workspace.task.update", "command-b", strings.Repeat("b", 64)),
	}
	mutations[1].Audit.ID = "audit-2"
	mutations[1].Outbox[0].ID = "event-2"

	errorsByMutation := make([]error, len(mutations))
	var wait sync.WaitGroup
	for index := range mutations {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			_, errorsByMutation[index] = repository.Execute(context.Background(), mutations[index], insertTestDomainMutation("mutation-"+string(rune('a'+index))))
		}(index)
	}
	wait.Wait()

	successes, conflicts := 0, 0
	for _, err := range errorsByMutation {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, contract.ErrRevisionConflict):
			var conflict contract.RevisionConflictError
			if !errors.As(err, &conflict) || conflict.CurrentRevision != 1 {
				t.Fatalf("revision conflict = %v", err)
			}
			conflicts++
		default:
			t.Fatalf("unexpected concurrent error: %v", err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("successes=%d conflicts=%d", successes, conflicts)
	}
	assertGovernanceRowCount(t, db, "test_domain_mutations", 1)
	assertGovernanceRowCount(t, db, "workspace_audit_entries", 1)
	assertGovernanceRowCount(t, db, "workspace_outbox_events", 1)
}

func TestGovernanceRepositoryRollsBackEveryGovernancePhase(t *testing.T) {
	phases := []GovernancePhase{
		GovernanceAfterDomain,
		GovernanceAfterRevision,
		GovernanceAfterAudit,
		GovernanceAfterOutbox,
		GovernanceAfterReplay,
	}
	for _, phase := range phases {
		t.Run(string(phase), func(t *testing.T) {
			db := openGovernanceTestDB(t)
			repository, err := NewGovernanceRepository(Config{DB: db}, WithGovernanceFailureHook(func(observed GovernancePhase) error {
				if observed == phase {
					return errors.New("injected governance failure")
				}
				return nil
			}))
			if err != nil {
				t.Fatal(err)
			}
			prepared := prepareGovernanceTestMutation(t, "workspace-1", "workspace.task.create", "command-1", strings.Repeat("a", 64))
			if _, err := repository.Execute(context.Background(), prepared, insertTestDomainMutation("task-1")); err == nil {
				t.Fatal("expected injected failure")
			}
			for _, table := range []string{
				"test_domain_mutations",
				"workspace_resource_revisions",
				"workspace_audit_entries",
				"workspace_outbox_events",
				"workspace_mutation_idempotency",
			} {
				assertGovernanceRowCount(t, db, table, 0)
			}
		})
	}
}

func TestGovernanceRepositoryScopesIdempotencyByWorkspaceAndAction(t *testing.T) {
	db := openGovernanceTestDB(t)
	repository, err := NewGovernanceRepository(Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	workspaceOne := prepareGovernanceTestMutation(t, "workspace-1", "workspace.task.create", "shared-key", strings.Repeat("a", 64))
	workspaceTwo := prepareGovernanceTestMutation(t, "workspace-2", "workspace.task.create", "shared-key", strings.Repeat("a", 64))
	secondAction := prepareGovernanceTestMutation(t, "workspace-1", "workspace.task.update", "shared-key", strings.Repeat("a", 64))
	secondAction.Command.ExpectedRevision = 1
	secondAction.Result.ResourceRevision = 2
	secondAction.Audit.ID = "audit-update"
	secondAction.Audit.ResourceRevision = 2
	secondAction.Outbox[0].ID = "event-update"
	secondAction.Outbox[0].AggregateRevision = 2

	for index, mutation := range []application.PreparedGovernanceMutation{workspaceOne, workspaceTwo, secondAction} {
		if _, err := repository.Execute(context.Background(), mutation, insertTestDomainMutation("isolated-"+string(rune('a'+index)))); err != nil {
			t.Fatalf("mutation %d: %v", index, err)
		}
	}
	assertGovernanceRowCount(t, db, "test_domain_mutations", 3)
	assertGovernanceRowCount(t, db, "workspace_resource_revisions", 2)
	assertGovernanceRowCount(t, db, "workspace_mutation_idempotency", 3)
}

func TestGovernanceRepositoryRejectsCrossWorkspacePreparedEnvelope(t *testing.T) {
	db := openGovernanceTestDB(t)
	repository, err := NewGovernanceRepository(Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	prepared := prepareGovernanceTestMutation(t, "workspace-1", "workspace.task.create", "command-1", strings.Repeat("a", 64))
	prepared.Outbox[0].WorkspaceID = "workspace-2"
	if _, err := repository.Execute(context.Background(), prepared, insertTestDomainMutation("task-1")); !errors.Is(err, contract.ErrInvalidGovernanceMutation) {
		t.Fatalf("error = %v, want invalid governance mutation", err)
	}
	assertGovernanceRowCount(t, db, "test_domain_mutations", 0)
}

func openGovernanceTestDB(t *testing.T) *sql.DB {
	t.Helper()
	path := filepath.ToSlash(filepath.Join(t.TempDir(), "governance.db"))
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	schema, err := os.ReadFile("migrations/000009_workspace_governance.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(schema)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE test_domain_mutations (id TEXT PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	return db
}

func prepareGovernanceTestMutation(t *testing.T, workspaceID, action, key, hash string) application.PreparedGovernanceMutation {
	t.Helper()
	prepared, err := application.NewGovernanceService().Prepare(application.GovernanceRequest{
		Identity:       contract.MutationIdentity{WorkspaceID: workspaceID, ActorType: "member", ActorID: "member-1", RequestID: "request-1"},
		Command:        contract.MutationCommand{Action: action, ResourceKind: "task", ResourceID: "task-1", ExpectedRevision: 0, IdempotencyKey: key, RequestHash: hash},
		ResponseStatus: 201,
		ResponseBody:   json.RawMessage(`{"id":"task-1"}`),
		AuditID:        "audit-1",
		OccurredAt:     time.Unix(1, 0).UTC(),
		AuditMetadata:  map[string]string{"status": "todo"},
		Outbox:         []application.OutboxDraft{{ID: "event-1", EventType: "task:created", Payload: json.RawMessage(`{"id":"task-1"}`)}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return prepared
}

func assertGovernanceRowCount(t *testing.T, db *sql.DB, table string, want int) {
	t.Helper()
	var got int
	if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("%s count = %d, want %d", table, got, want)
	}
}

func insertTestDomainMutation(id string) DomainMutation {
	return func(ctx context.Context, connection *sql.Conn) error {
		_, err := connection.ExecContext(ctx, `INSERT INTO test_domain_mutations(id) VALUES (?)`, id)
		return err
	}
}
