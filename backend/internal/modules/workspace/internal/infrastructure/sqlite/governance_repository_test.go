package sqlite

import (
	"context"
	"database/sql"
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
	if second.ResourceRevision != 1 || !second.Replayed || string(second.ResponseBody) != `{"version":"governance-replay-v1","data":{"id":"task-1"}}` {
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

func TestGovernanceRepositoryRejectsLegacyReplayEnvelopeWithoutRepeatingMutation(t *testing.T) {
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
	if _, err := repository.Execute(context.Background(), prepared, apply); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE workspace_mutation_idempotency SET response_body='{"id":"task-1"}'`); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.Execute(context.Background(), prepared, apply); !errors.Is(err, contract.ErrInvalidGovernanceMutation) {
		t.Fatalf("legacy replay error = %v", err)
	}
	if applyCount != 1 {
		t.Fatalf("domain apply count = %d, want 1", applyCount)
	}
}

func TestGovernanceRepositorySerializesConcurrentExpectedRevision(t *testing.T) {
	db := openGovernanceTestDB(t)
	db.SetMaxOpenConns(4)
	repository, err := NewGovernanceRepository(Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	mutations := []application.PreparedGovernanceMutation{
		prepareGovernanceTestMutationWithOptions(t, "workspace-1", "workspace.task.update", "command-a", strings.Repeat("a", 64), governanceMutationOptions{AuditID: "audit-1", EventID: "event-1"}),
		prepareGovernanceTestMutationWithOptions(t, "workspace-1", "workspace.task.update", "command-b", strings.Repeat("b", 64), governanceMutationOptions{AuditID: "audit-2", EventID: "event-2"}),
	}

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
	secondAction := prepareGovernanceTestMutationWithOptions(t, "workspace-1", "workspace.task.update", "shared-key", strings.Repeat("a", 64), governanceMutationOptions{
		ExpectedRevision: 1,
		AuditID:          "audit-update",
		EventID:          "event-update",
	})

	for index, mutation := range []application.PreparedGovernanceMutation{workspaceOne, workspaceTwo, secondAction} {
		if _, err := repository.Execute(context.Background(), mutation, insertTestDomainMutation("isolated-"+string(rune('a'+index)))); err != nil {
			t.Fatalf("mutation %d: %v", index, err)
		}
	}
	assertGovernanceRowCount(t, db, "test_domain_mutations", 3)
	assertGovernanceRowCount(t, db, "workspace_resource_revisions", 2)
	assertGovernanceRowCount(t, db, "workspace_mutation_idempotency", 3)
}

func TestGovernanceRepositoryRejectsMutationWithoutPolicyPreparation(t *testing.T) {
	db := openGovernanceTestDB(t)
	repository, err := NewGovernanceRepository(Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repository.Execute(context.Background(), application.PreparedGovernanceMutation{}, insertTestDomainMutation("task-1")); !errors.Is(err, contract.ErrInvalidGovernanceMutation) {
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
	staleIdentity, err := event.ClaimIdentity()
	if err != nil {
		t.Fatal(err)
	}
	currentIdentity, err := reclaimed[0].ClaimIdentity()
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.MarkOutboxDelivered(context.Background(), staleIdentity, now.Add(62*time.Second)); !errors.Is(err, contract.ErrOutboxClaimConflict) {
		t.Fatalf("stale delivery error = %v", err)
	}
	if err := repository.MarkOutboxDelivered(context.Background(), currentIdentity, now.Add(62*time.Second)); err != nil {
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
	firstClaim, err := claimed[0].ClaimIdentity()
	if err != nil {
		t.Fatal(err)
	}
	retryAt := now.Add(time.Minute)
	if err := repository.MarkOutboxFailed(context.Background(), firstClaim, now, retryAt, "publish_failed", false); err != nil {
		t.Fatal(err)
	}
	if err := repository.MarkOutboxFailed(context.Background(), firstClaim, now, retryAt, "publish_failed", false); !errors.Is(err, contract.ErrOutboxClaimConflict) {
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
	secondClaim, err := reclaimed[0].ClaimIdentity()
	if err != nil {
		t.Fatal(err)
	}
	if err := restarted.MarkOutboxFailed(context.Background(), secondClaim, retryAt, retryAt, "publish_failed", true); err != nil {
		t.Fatal(err)
	}
	diagnostics, err = restarted.ReadOutboxDiagnostics(context.Background(), "workspace-1", retryAt)
	if err != nil || diagnostics.DeadLetterCount != 1 {
		t.Fatalf("dead-letter diagnostics = %+v, %v", diagnostics, err)
	}
	deadLetterIdentity := contract.OutboxRowIdentity{State: contract.OutboxDeadLetter, AvailableAt: retryAt, WorkspaceID: "workspace-1", ID: "event-1"}
	if err := restarted.ReplayOutbox(context.Background(), deadLetterIdentity, retryAt.Add(time.Minute)); err != nil {
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

func TestGovernanceRepositoryRejectsStaleObservedLeaseWithoutChangingRow(t *testing.T) {
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
	claimed, err := repository.ClaimOutbox(context.Background(), now, 1, time.Minute, "claim-1")
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim = %+v, %v", claimed, err)
	}
	identity, err := claimed[0].ClaimIdentity()
	if err != nil {
		t.Fatal(err)
	}
	identity.LeaseExpiresAt = identity.LeaseExpiresAt.Add(time.Second)
	if err := repository.MarkOutboxDelivered(context.Background(), identity, now.Add(time.Second)); !errors.Is(err, contract.ErrOutboxClaimConflict) {
		t.Fatalf("stale observed lease error = %v", err)
	}
	var state string
	if err := db.QueryRow(`SELECT state FROM workspace_outbox_events WHERE workspace_id='workspace-1' AND id='event-1'`).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != string(contract.OutboxInflight) {
		t.Fatalf("state = %s, want inflight", state)
	}
}

func TestGovernanceRepositoryCompleteTupleDoesNotModifySiblingWithSameEventID(t *testing.T) {
	db := openGovernanceTestDB(t)
	repository, err := NewGovernanceRepository(Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	availableOne := time.Unix(10, 0).UTC()
	availableTwo := availableOne.Add(time.Second)
	lease := availableOne.Add(time.Minute)
	insertInflightOutboxTestRow(t, db, availableOne, "event-1", "claim-1", lease)
	insertInflightOutboxTestRow(t, db, availableTwo, "event-1", "claim-1", lease)
	identity := contract.OutboxClaimIdentity{
		OutboxRowIdentity: contract.OutboxRowIdentity{State: contract.OutboxInflight, AvailableAt: availableOne, WorkspaceID: "workspace-1", ID: "event-1"},
		ClaimToken:        "claim-1", LeaseExpiresAt: lease,
	}
	if err := repository.MarkOutboxDelivered(context.Background(), identity, availableOne.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	var firstState, secondState string
	if err := db.QueryRow(`SELECT state FROM workspace_outbox_events WHERE workspace_id=? AND id=? AND available_at=?`, "workspace-1", "event-1", availableOne.Format(time.RFC3339Nano)).Scan(&firstState); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT state FROM workspace_outbox_events WHERE workspace_id=? AND id=? AND available_at=?`, "workspace-1", "event-1", availableTwo.Format(time.RFC3339Nano)).Scan(&secondState); err != nil {
		t.Fatal(err)
	}
	if firstState != string(contract.OutboxDelivered) || secondState != string(contract.OutboxInflight) {
		t.Fatalf("states = first:%s second:%s", firstState, secondState)
	}
}

func TestGovernanceRepositoryDeadLetterReplayUsesCompleteTuple(t *testing.T) {
	db := openGovernanceTestDB(t)
	repository, err := NewGovernanceRepository(Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	availableOne := time.Unix(10, 0).UTC()
	availableTwo := availableOne.Add(time.Second)
	insertDeadOutboxTestRow(t, db, availableOne, "event-1")
	insertDeadOutboxTestRow(t, db, availableTwo, "event-1")
	identity := contract.OutboxRowIdentity{State: contract.OutboxDeadLetter, AvailableAt: availableOne, WorkspaceID: "workspace-1", ID: "event-1"}
	if err := repository.ReplayOutbox(context.Background(), identity, availableOne.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	var readyCount, deadCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM workspace_outbox_events WHERE state='ready'`).Scan(&readyCount); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM workspace_outbox_events WHERE state='dead_letter'`).Scan(&deadCount); err != nil {
		t.Fatal(err)
	}
	if readyCount != 1 || deadCount != 1 {
		t.Fatalf("ready=%d dead=%d", readyCount, deadCount)
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
	return prepareGovernanceTestMutationWithOptions(t, workspaceID, action, key, hash, governanceMutationOptions{
		AuditID: "audit-1",
		EventID: "event-1",
	})
}

type governanceMutationOptions struct {
	ExpectedRevision int64
	AuditID          string
	EventID          string
}

func prepareGovernanceTestMutationWithOptions(t *testing.T, workspaceID, action, key, hash string, options governanceMutationOptions) application.PreparedGovernanceMutation {
	t.Helper()
	service := application.NewGovernanceService(repositoryGovernanceProvider{})
	prepared, err := service.PrepareContext(context.Background(), application.GovernanceRequest{
		Identity:       contract.MutationIdentity{WorkspaceID: workspaceID, RequestID: "request-1"},
		Command:        contract.MutationCommand{Action: action, ResourceKind: "task", ResourceID: "task-1", ExpectedRevision: options.ExpectedRevision, IdempotencyKey: key},
		RequestFields:  map[string]any{"content_hash": hash},
		ResponseStatus: 201,
		ResponseFields: map[string]any{"id": "task-1"},
		AuditID:        options.AuditID,
		OccurredAt:     time.Unix(1, 0).UTC(),
		AuditFields:    map[string]any{"status": "todo"},
		Outbox:         []application.OutboxDraft{{ID: options.EventID, EventType: "task:created", Fields: map[string]any{"id": "task-1"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return prepared
}

type repositoryGovernanceProvider struct{}

func (repositoryGovernanceProvider) ResolveGovernancePolicy(_ context.Context, workspaceID, requestID, action, resourceKind string) (contract.MutationIdentity, application.GovernanceActionPolicy, error) {
	return contract.MutationIdentity{WorkspaceID: workspaceID, ActorType: "member", ActorID: "user-1", RequestID: requestID}, application.GovernanceActionPolicy{
		Action: action, ResourceKind: resourceKind,
		RequestSchema: application.EnvelopeSchema{"content_hash": {Kind: application.SafeSHA256, Required: true}},
		ReplaySchema:  application.EnvelopeSchema{"id": {Kind: application.SafeIdentifier, MaxLength: 64, Required: true}},
		AuditSchema:   application.EnvelopeSchema{"status": {Kind: application.SafeEnum, EnumValues: []string{"todo", "done"}, Required: true}},
		EventSchemas: map[string]application.EnvelopeSchema{
			"task:created": {"id": {Kind: application.SafeIdentifier, MaxLength: 64, Required: true}},
		},
	}, nil
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

func insertInflightOutboxTestRow(t *testing.T, db *sql.DB, availableAt time.Time, eventID, claimToken string, leaseExpiresAt time.Time) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO workspace_outbox_events(
		state, available_at, workspace_id, id, event_type, aggregate_kind, aggregate_id, aggregate_revision,
		payload_json, actor_type, actor_id, attempt_count, claim_token, lease_expires_at, created_at)
		VALUES ('inflight', ?, 'workspace-1', ?, 'task:created', 'task', 'task-1', 1,
		'{"version":"governance-outbox-v1","data":{"id":"task-1"}}', 'member', 'user-1', 1, ?, ?, ?)`,
		availableAt.Format(time.RFC3339Nano), eventID, claimToken, leaseExpiresAt.Format(time.RFC3339Nano), availableAt.Format(time.RFC3339Nano))
	if err != nil {
		t.Fatal(err)
	}
}

func insertDeadOutboxTestRow(t *testing.T, db *sql.DB, availableAt time.Time, eventID string) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO workspace_outbox_events(
		state, available_at, workspace_id, id, event_type, aggregate_kind, aggregate_id, aggregate_revision,
		payload_json, actor_type, actor_id, attempt_count, last_error_code, created_at)
		VALUES ('dead_letter', ?, 'workspace-1', ?, 'task:created', 'task', 'task-1', 1,
		'{"version":"governance-outbox-v1","data":{"id":"task-1"}}', 'member', 'user-1', 4, 'publish_failed', ?)`,
		availableAt.Format(time.RFC3339Nano), eventID, availableAt.Format(time.RFC3339Nano))
	if err != nil {
		t.Fatal(err)
	}
}
