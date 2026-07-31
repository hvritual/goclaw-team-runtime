package handler

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/multica-ai/multica/server/internal/knowledge"
)

type evidenceExecutorStub struct {
	tag   pgconn.CommandTag
	query string
	args  []any
}

func (s *evidenceExecutorStub) Exec(_ context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	s.query = query
	s.args = args
	return s.tag, nil
}

func TestAppendKnowledgeEvidenceReportsIdempotentInsert(t *testing.T) {
	h := &Handler{knowledgeEvidenceEnabled: true}
	evidence := knowledge.NewEvidence(knowledge.EvidenceDraft{
		WorkspaceID:    "01000000-0000-0000-0000-000000000000",
		SourceType:     "comment",
		SourceID:       "04000000-0000-0000-0000-000000000000",
		SourceRevision: "2026-07-31T08:30:00Z",
		EventType:      "comment.decision_proposed",
		Kind:           knowledge.KindDecision,
		Title:          "Decision: Keep immutable evidence",
		Content:        "Keep immutable evidence.",
		ActorID:        "02000000-0000-0000-0000-000000000000",
		OccurredAt:     time.Date(2026, time.July, 31, 8, 30, 0, 0, time.UTC),
	})

	insertedExecutor := &evidenceExecutorStub{tag: pgconn.NewCommandTag("INSERT 0 1")}
	inserted, err := h.appendKnowledgeEvidence(context.Background(), insertedExecutor, evidence)
	if err != nil {
		t.Fatal(err)
	}
	if !inserted {
		t.Fatal("new outbox evidence was not reported as inserted")
	}
	if !strings.Contains(insertedExecutor.query, "ON CONFLICT (idempotency_key) DO NOTHING") {
		t.Fatalf("outbox query is not idempotent: %s", insertedExecutor.query)
	}

	duplicateExecutor := &evidenceExecutorStub{tag: pgconn.NewCommandTag("INSERT 0 0")}
	inserted, err = h.appendKnowledgeEvidence(context.Background(), duplicateExecutor, evidence)
	if err != nil {
		t.Fatal(err)
	}
	if inserted {
		t.Fatal("duplicate outbox evidence was reported as inserted")
	}
}
