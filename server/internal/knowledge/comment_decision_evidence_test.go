package knowledge_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/knowledge"
	"github.com/multica-ai/multica/server/internal/knowledge/adapter/memory"
)

func TestCommentDecisionEvidenceCapturesProvenance(t *testing.T) {
	updatedAt := time.Date(2026, time.July, 31, 8, 30, 0, 123, time.UTC)
	evidence := knowledge.NewCommentDecisionEvidence(knowledge.CommentDecisionEvidenceDraft{
		WorkspaceID: "01000000-0000-0000-0000-000000000000",
		ProjectID:   "project-1",
		CommentID:   "04000000-0000-0000-0000-000000000000",
		Content:     "Use SQLite as the first knowledge adapter.\n\nKeep the port replaceable.",
		UpdatedAt:   updatedAt,
		ActorID:     "member-1",
	})

	if evidence.EventType != "comment.decision_proposed" {
		t.Fatalf("event type = %q", evidence.EventType)
	}
	if evidence.Kind != knowledge.KindDecision {
		t.Fatalf("kind = %q", evidence.Kind)
	}
	if evidence.ProjectID != "project-1" || evidence.ActorID != "member-1" {
		t.Fatalf("scope = project %q actor %q", evidence.ProjectID, evidence.ActorID)
	}
	if evidence.Title != "Decision: Use SQLite as the first knowledge adapter." {
		t.Fatalf("title = %q", evidence.Title)
	}
	if !strings.HasPrefix(evidence.SourceRevision, updatedAt.Format(time.RFC3339Nano)+"@sha256:") {
		t.Fatalf("source revision = %q", evidence.SourceRevision)
	}
	if evidence.Terminal {
		t.Fatal("comment decision evidence must require candidate review")
	}
	if len(evidence.SourceRefs) != 1 {
		t.Fatalf("source refs = %#v", evidence.SourceRefs)
	}
	ref := evidence.SourceRefs[0]
	if ref.Type != "comment" || ref.ID != "04000000-0000-0000-0000-000000000000" {
		t.Fatalf("source ref = %#v", ref)
	}
	if ref.Revision != evidence.SourceRevision || ref.URI != "multica://comments/"+ref.ID {
		t.Fatalf("source ref provenance = %#v", ref)
	}
	if ref.Checksum == "" || ref.Checksum != evidence.Checksum {
		t.Fatalf("source ref checksum = %q, evidence checksum = %q", ref.Checksum, evidence.Checksum)
	}
}

func TestCommentDecisionEvidenceIsRevisionedAndProjectsOneCandidate(t *testing.T) {
	draft := knowledge.CommentDecisionEvidenceDraft{
		WorkspaceID: "01000000-0000-0000-0000-000000000000",
		CommentID:   "04000000-0000-0000-0000-000000000000",
		Content:     "Keep the source adapter replaceable.",
		UpdatedAt:   time.Date(2026, time.July, 31, 8, 30, 0, 0, time.UTC),
		ActorID:     "member-1",
	}
	firstEvidence := knowledge.NewCommentDecisionEvidence(draft)
	repeatEvidence := knowledge.NewCommentDecisionEvidence(draft)
	draft.Content = "Keep the source adapter replaceable after edits."
	editedEvidence := knowledge.NewCommentDecisionEvidence(draft)

	if firstEvidence.IdempotencyKey != repeatEvidence.IdempotencyKey {
		t.Fatalf("same revision keys differ: %q != %q", firstEvidence.IdempotencyKey, repeatEvidence.IdempotencyKey)
	}
	if firstEvidence.IdempotencyKey == editedEvidence.IdempotencyKey {
		t.Fatalf("edited revision reused idempotency key %q", editedEvidence.IdempotencyKey)
	}
	if firstEvidence.SourceRevision == editedEvidence.SourceRevision {
		t.Fatalf("content edit reused source revision %q", editedEvidence.SourceRevision)
	}

	store := memory.New()
	service := knowledge.NewService(store, nil)
	first, err := service.IngestOutboxEvidence(context.Background(), firstEvidence)
	if err != nil {
		t.Fatal(err)
	}
	if first.Candidate == nil || first.Candidate.Kind != knowledge.KindDecision {
		t.Fatalf("first ingestion = %#v", first)
	}
	if len(first.Candidate.SourceRefs) != 1 || first.Candidate.SourceRefs[0].Type != "comment" {
		t.Fatalf("candidate provenance = %#v", first.Candidate.SourceRefs)
	}
	repeat, err := service.IngestOutboxEvidence(context.Background(), repeatEvidence)
	if err != nil {
		t.Fatal(err)
	}
	if !repeat.Duplicate || repeat.Candidate != nil {
		t.Fatalf("repeat ingestion = %#v", repeat)
	}
	page, err := store.ListCandidates(context.Background(), knowledge.CandidateQuery{
		WorkspaceID: draft.WorkspaceID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Candidates) != 1 {
		t.Fatalf("candidate count = %d, want 1", len(page.Candidates))
	}
}
