package knowledge_test

import (
	"context"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/knowledge"
	"github.com/multica-ai/multica/server/internal/knowledge/adapter/memory"
)

func TestManualProposalRequiresReason(t *testing.T) {
	service := knowledge.NewService(nil, nil)

	_, err := service.Propose(context.Background(), knowledge.ProposalInput{
		WorkspaceID: "workspace-1",
		Kind:        knowledge.KindDecision,
		Title:       "Choose SQLite first",
		Content:     "SQLite is the first durable knowledge adapter.",
		ProposedBy:  "user-1",
	})
	if err == nil {
		t.Fatal("expected a proposal without a reason to be rejected")
	}
}

func TestManualProposalRequiresWorkspaceContentAndActor(t *testing.T) {
	service := knowledge.NewService(memory.New(), nil)
	ctx := context.Background()
	tests := []knowledge.ProposalInput{
		{
			Kind:       knowledge.KindDecision,
			Title:      "Missing workspace",
			Content:    "content",
			Reason:     "reason",
			ProposedBy: "user-1",
		},
		{
			WorkspaceID: "workspace-1",
			Kind:        knowledge.KindDecision,
			Title:       "Missing content",
			Reason:      "reason",
			ProposedBy:  "user-1",
		},
		{
			WorkspaceID: "workspace-1",
			Kind:        knowledge.KindDecision,
			Title:       "Missing actor",
			Content:     "content",
			Reason:      "reason",
		},
	}
	for _, input := range tests {
		if _, err := service.Propose(ctx, input); err != knowledge.ErrInvalidProposal {
			t.Fatalf("proposal %q error = %v, want %v", input.Title, err, knowledge.ErrInvalidProposal)
		}
	}
}

func TestManualProposalCreatesCandidate(t *testing.T) {
	store := memory.New()
	service := knowledge.NewService(store, nil)

	candidate, err := service.Propose(context.Background(), knowledge.ProposalInput{
		WorkspaceID: "workspace-1",
		ProjectID:   "project-1",
		Kind:        knowledge.KindDecision,
		Title:       "Choose SQLite first",
		Content:     "SQLite is the first durable knowledge adapter.",
		Reason:      "Keep the first deployment self-contained.",
		ProposedBy:  "user-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if candidate.ID == "" {
		t.Fatal("expected a stable candidate ID")
	}
	if candidate.Status != knowledge.StatusCandidate {
		t.Fatalf("status = %q, want %q", candidate.Status, knowledge.StatusCandidate)
	}
	if candidate.Revision != 1 {
		t.Fatalf("revision = %d, want 1", candidate.Revision)
	}
}

func TestApprovedCandidatePublishesAnImmutableFirstRevision(t *testing.T) {
	store := memory.New()
	service := knowledge.NewService(store, nil)
	ctx := context.Background()
	candidate, err := service.Propose(ctx, knowledge.ProposalInput{
		WorkspaceID: "workspace-1",
		ProjectID:   "project-1",
		Kind:        knowledge.KindProcedure,
		Title:       "Restore the knowledge database",
		Content:     "Checkpoint the database before copying it.",
		Reason:      "Operators need a safe recovery procedure.",
		ProposedBy:  "user-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	reviewed, entry, err := service.Review(ctx, knowledge.ReviewInput{
		WorkspaceID:      "workspace-1",
		CandidateID:      candidate.ID,
		ExpectedRevision: 1,
		Action:           knowledge.ReviewApprove,
		ReviewerID:       "admin-1",
		Rationale:        "The procedure was verified during recovery testing.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if reviewed.Status != knowledge.StatusPublished {
		t.Fatalf("candidate status = %q, want %q", reviewed.Status, knowledge.StatusPublished)
	}
	if entry == nil || entry.Status != knowledge.StatusPublished {
		t.Fatalf("published entry = %#v", entry)
	}
	if entry.CurrentRevision != 1 || len(entry.Revisions) != 1 {
		t.Fatalf("entry revisions = %#v", entry.Revisions)
	}
	if entry.Revisions[0].Content != candidate.Content {
		t.Fatalf("revision content = %q, want %q", entry.Revisions[0].Content, candidate.Content)
	}
}

func TestEvidenceIngestionIsIdempotent(t *testing.T) {
	store := memory.New()
	service := knowledge.NewService(store, knowledge.DefaultPromotionPolicy{})
	ctx := context.Background()
	evidence := knowledge.Evidence{
		ID:             "evidence-1",
		WorkspaceID:    "workspace-1",
		ProjectID:      "project-1",
		SourceType:     "issue",
		SourceID:       "issue-1",
		SourceRevision: "4",
		EventType:      "acceptance.completed",
		Kind:           knowledge.KindRequirement,
		Title:          "Recovery acceptance",
		Content:        "Recovery completes without deleting evidence.",
		ActorID:        "user-1",
		IdempotencyKey: "issue-1:4:acceptance.completed",
		Checksum:       "sha256:example",
		OccurredAt:     candidateTime(),
		Terminal:       true,
		Validated:      true,
		Confidence:     1,
		SourceRefs: []knowledge.SourceRef{{
			Type:     "issue",
			ID:       "issue-1",
			Revision: "4",
			URI:      "multica://issues/issue-1",
		}},
	}

	first, err := service.IngestEvidence(ctx, evidence)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.IngestEvidence(ctx, evidence)
	if err != nil {
		t.Fatal(err)
	}
	if first.Duplicate {
		t.Fatal("first ingestion was unexpectedly marked duplicate")
	}
	if first.Candidate == nil || first.Candidate.Status != knowledge.StatusCandidate {
		t.Fatalf("first ingestion candidate = %#v", first.Candidate)
	}
	if !second.Duplicate || second.Candidate != nil || second.Entry != nil {
		t.Fatalf("second ingestion = %#v", second)
	}
}

func candidateTime() time.Time {
	return time.Date(2026, time.July, 31, 10, 0, 0, 0, time.UTC)
}
