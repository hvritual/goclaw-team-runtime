package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/multica-ai/multica/server/internal/knowledge"
	knowledgeSqlite "github.com/multica-ai/multica/server/internal/knowledge/adapter/sqlite"
	"github.com/multica-ai/multica/server/internal/knowledge/outbox"
)

func TestOpenKnowledgeRuntimeCanBeDisabled(t *testing.T) {
	t.Setenv("MULTICA_KNOWLEDGE_ENABLED", "false")

	runtime := openKnowledgeRuntime(nil, nil)
	if runtime.enabled || runtime.store != nil || runtime.service != nil || runtime.unavailable != nil {
		t.Fatalf("disabled runtime = %#v", runtime)
	}
}

func TestOpenKnowledgeRuntimeDegradesWhenSQLiteIsUnavailable(t *testing.T) {
	t.Setenv("MULTICA_KNOWLEDGE_ENABLED", "true")
	t.Setenv("MULTICA_KNOWLEDGE_SQLITE_PATH", t.TempDir())

	runtime := openKnowledgeRuntime(nil, nil)
	if !runtime.enabled || runtime.store != nil || runtime.service != nil || runtime.unavailable == nil {
		t.Fatalf("degraded runtime = %#v", runtime)
	}
}

func TestPostgresOutboxDispatchesEvidenceToSQLite(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("integration test requires Postgres at DATABASE_URL")
	}
	ctx := context.Background()
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	config.MaxConns = 1
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if _, err := pool.Exec(ctx, `
		CREATE TEMP TABLE knowledge_evidence_outbox (
			id uuid NOT NULL DEFAULT gen_random_uuid(),
			workspace_id uuid NOT NULL,
			evidence_id text NOT NULL,
			idempotency_key text NOT NULL,
			payload_json jsonb NOT NULL,
			attempts integer NOT NULL DEFAULT 0,
			available_at timestamptz NOT NULL DEFAULT now(),
			delivered_at timestamptz,
			last_error text NOT NULL DEFAULT '',
			created_at timestamptz NOT NULL DEFAULT now(),
			updated_at timestamptz NOT NULL DEFAULT now()
		)`); err != nil {
		t.Fatal(err)
	}

	workspaceID := uuid.NewString()
	evidence := knowledge.Evidence{
		ID: uuid.NewString(), WorkspaceID: workspaceID,
		SourceType: "task", SourceID: uuid.NewString(), SourceRevision: "2",
		EventType: "task.completed", Kind: knowledge.KindReference,
		Title: "Task completed", Content: "The production outbox reached SQLite.",
		ActorID: uuid.NewString(), IdempotencyKey: uuid.NewString(),
		OccurredAt: time.Now().UTC(), Terminal: true, Validated: true, Confidence: 1,
		SourceRefs: []knowledge.SourceRef{{Type: "task", ID: "task-1", URI: "multica://tasks/task-1"}},
	}
	payload, err := json.Marshal(evidence)
	if err != nil {
		t.Fatal(err)
	}
	messageID := uuid.NewString()
	if _, err := pool.Exec(ctx, `
		INSERT INTO knowledge_evidence_outbox(
			id, workspace_id, evidence_id, idempotency_key, payload_json
		) VALUES ($1, $2, $3, $4, $5)`,
		messageID, workspaceID, evidence.ID, evidence.IdempotencyKey, payload,
	); err != nil {
		t.Fatal(err)
	}

	store, err := knowledgeSqlite.Open(filepath.Join(t.TempDir(), "knowledge.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	dispatcher := outbox.NewDispatcher(
		&postgresEvidenceOutbox{pool: pool},
		knowledge.NewService(store, nil),
	)
	report, err := dispatcher.Drain(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if report.Delivered != 1 {
		t.Fatalf("dispatch report = %#v", report)
	}
	var deliveredAt *time.Time
	if err := pool.QueryRow(ctx,
		"SELECT delivered_at FROM knowledge_evidence_outbox WHERE id = $1",
		messageID,
	).Scan(&deliveredAt); err != nil {
		t.Fatal(err)
	}
	if deliveredAt == nil {
		t.Fatal("outbox message was not marked delivered")
	}
	page, err := store.Search(ctx, knowledge.SearchQuery{
		WorkspaceID: workspaceID, Text: "production outbox", Limit: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Results) != 1 || page.Results[0].Entry.WorkspaceID != workspaceID {
		t.Fatalf("search page = %#v", page)
	}
}
