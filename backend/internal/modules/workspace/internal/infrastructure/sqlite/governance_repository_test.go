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

func TestGovernanceRepositoryClaimsWithLeaseAndRejectsStaleToken(t *testing.T) {
	db := openGovernanceTestDB(t)
	repository, err := NewGovernanceRepository(Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	prepared := prepareGovernanceTestMutation(t, "workspace-1", "workspace.task.create", "command-1", strings.Repeat("a", 64))
	if _, err := repository.Execute(context.Background(), prepared, insertTestDomainMutation("task-1")); err != nil {
		t.Fatal(err)
	}
	now := time.Unix(2, 0).UTC()
	claimed, err := repository.ClaimOutbox(context.Background(), now, 100, time.Minute, "claim-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(claimed) != 1 {
		t.Fatalf("claimed = %d, want 1", len(claimed))
	}
	event := claimed[0]
	if event.ID != "event-1" || event.AggregateRevision != 1 || event.State != contract.OutboxInflight || event.ClaimToken != "claim-1" || event.AttemptCount != 1 {
		t.Fatalf("claimed event = %+v", event)
	}
	if event.LeaseExpiresAt == nil || !event.LeaseExpiresAt.Equal(now.Add(time.Minute)) {
		t.Fatalf("lease = %v", event.LeaseExpiresAt)
	}
	second, err := repository.ClaimOutbox(context.Background(), now.Add(30*time.Second), 100, time.Minute, "claim-2")
	if err != nil || len(second) != 0 {
		t.Fatalf("claim before expiry = %+v, %v", second, err)
	}
	reclaimed, err := repository.ClaimOutbox(context.Background(), now.Add(61*time.Second), 100, time.Minute, "claim-2")
	if err != nil || len(reclaimed) != 1 {
		t.Fatalf("claim after expiry = %+v, %v", reclaimed, err)
	}
	if reclaimed[0].ID != event.ID || reclaimed[0].AggregateRevision != event.AggregateRevision || reclaimed[0].AttemptCount != 2 {
		t.Fatalf("reclaimed event = %+v", reclaimed[0])
	}
	if err := repository.MarkOutboxDelivered(context.Background(), "workspace-1", "event-1", "claim-1", now.Add(62*time.Second)); !errors.Is(err, contract.ErrOutboxClaimConflict) {
		t.Fatalf("stale delivery error = %v", err)
	}
	if err := repository.MarkOutboxDelivered(context.Background(), "workspace-1", "event-1", "claim-2", now.Add(62*time.Second)); err != nil {
		t.Fatal(err)
	}
	var state, claim sql.NullString
	var delivered sql.NullString
	if err := db.QueryRow(`SELECT state, claim_token, delivered_at FROM workspace_outbox_events WHERE workspace_id='workspace-1' AND id='event-1'`).Scan(&state, &claim, &delivered); err != nil {
		t.Fatal(err)
	}
	if state.String != string(contract.OutboxDelivered) || claim.Valid || !delivered.Valid {
		t.Fatalf("stored delivery = state:%v claim:%v delivered:%v", state, claim, delivered)
	}
}

func TestGovernanceRepositoryRetriesDeadLettersReplaysAndSurvivesRestart(t *testing.T) {
	db := openGovernanceTestDB(t)
	repository, err := NewGovernanceRepository(Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	prepared := prepareGovernanceTestMutation(t, "workspace-1", "workspace.task.create", "command-1", strings.Repeat("a", 64))
	if _, err := repository.Execute(context.Background(), prepared, insertTestDomainMutation("task-1")); err != nil {
		t.Fatal(err)
	}
	now := time.Unix(2, 0).UTC()
	claimed, err := repository.ClaimOutbox(context.Background(), now, 100, time.Minute, "claim-1")
	if err != nil || len(claimed) != 1 {
		t.Fatalf("first claim = %+v, %v", claimed, err)
	}
	retryAt := now.Add(time.Minute)
	if err := repository.MarkOutboxFailed(context.Background(), "workspace-1", "event-1", "claim-1", now, retryAt, "publish_failed", false); err != nil {
		t.Fatal(err)
	}
	if err := repository.MarkOutboxFailed(context.Background(), "workspace-1", "event-1", "claim-1", now, retryAt, "publish_failed", false); !errors.Is(err, contract.ErrOutboxClaimConflict) {
		t.Fatalf("stale retry error = %v", err)
	}
	diagnostics, err := repository.ReadOutboxDiagnostics(context.Background(), "workspace-1", retryAt)
	if err != nil {
		t.Fatal(err)
	}
	if diagnostics.RetryWaitCount != 1 || diagnostics.SchemaVersion == "" {
		t.Fatalf("retry diagnostics = %+v", diagnostics)
	}

	restarted, err := NewGovernanceRepository(Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	reclaimed, err := restarted.ClaimOutbox(context.Background(), retryAt, 100, time.Minute, "claim-2")
	if err != nil || len(reclaimed) != 1 || reclaimed[0].ID != "event-1" || reclaimed[0].AggregateRevision != 1 {
		t.Fatalf("restart claim = %+v, %v", reclaimed, err)
	}
	if err := restarted.MarkOutboxFailed(context.Background(), "workspace-1", "event-1", "claim-2", retryAt, retryAt, "publish_failed", true); err != nil {
		t.Fatal(err)
	}
	diagnostics, err = restarted.ReadOutboxDiagnostics(context.Background(), "workspace-1", retryAt)
	if err != nil || diagnostics.DeadLetterCount != 1 {
		t.Fatalf("dead-letter diagnostics = %+v, %v", diagnostics, err)
	}
	if err := restarted.ReplayOutbox(context.Background(), "workspace-1", "event-1", retryAt.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	var state string
	var attempt int
	var eventID string
	var revision int64
	if err := db.QueryRow(`SELECT state, attempt_count, id, aggregate_revision FROM workspace_outbox_events WHERE workspace_id='workspace-1'`).Scan(&state, &attempt, &eventID, &revision); err != nil {
		t.Fatal(err)
	}
	if state != string(contract.OutboxReady) || attempt != 0 || eventID != "event-1" || revision != 1 {
		t.Fatalf("replayed row = state:%s attempt:%d id:%s revision:%d", state, attempt, eventID, revision)
	}
}

func TestGovernanceRepositoryEmptyClaimDoesNotAcquireWriteLock(t *testing.T) {
	db := openGovernanceTestDB(t)
	db.SetMaxOpenConns(2)
	repository, err := NewGovernanceRepository(Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	writer, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	if _, err := writer.ExecContext(context.Background(), `BEGIN IMMEDIATE`); err != nil {
		t.Fatal(err)
	}
	defer writer.ExecContext(context.Background(), `ROLLBACK`)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	events, err := repository.ClaimOutbox(ctx, time.Unix(2, 0).UTC(), 100, time.Minute, "claim-1")
	if err != nil || len(events) != 0 {
		t.Fatalf("empty claim = %+v, %v", events, err)
	}
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
