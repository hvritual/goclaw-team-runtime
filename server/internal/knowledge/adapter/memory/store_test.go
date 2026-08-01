package memory

import (
	"context"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/knowledge"
)

func TestSourceRefMetadataIsolatedAcrossMemoryStoreBoundaries(t *testing.T) {
	store := New()
	sourceRefs := []knowledge.SourceRef{{Type: "source", ID: "candidate-1", Metadata: map[string]string{"state": "original"}}}
	candidate, err := store.CreateCandidate(context.Background(), knowledge.Candidate{
		WorkspaceID: "workspace-1", Kind: knowledge.KindRequirement, Status: knowledge.StatusCandidate,
		Revision: 1, SourceRefs: sourceRefs,
	})
	if err != nil {
		t.Fatal(err)
	}
	sourceRefs[0].Metadata["state"] = "input-mutated"
	candidate.SourceRefs[0].Metadata["state"] = "result-mutated"
	assertCandidateMetadata(t, store, candidate.ID, "original")
	listed, err := store.ListCandidates(context.Background(), knowledge.CandidateQuery{WorkspaceID: "workspace-1"})
	if err != nil {
		t.Fatal(err)
	}
	listed.Candidates[0].SourceRefs[0].Metadata["state"] = "list-mutated"
	assertCandidateMetadata(t, store, candidate.ID, "original")

	evidenceSources := []knowledge.SourceRef{{Type: "source", ID: "evidence-1", Metadata: map[string]string{"state": "original"}}}
	entrySources := []knowledge.SourceRef{{Type: "source", ID: "entry-1", Metadata: map[string]string{"state": "original"}}}
	entry := knowledge.Entry{
		ID: "entry-1", WorkspaceID: "workspace-1", Kind: knowledge.KindRequirement,
		Status: knowledge.StatusPublished, CurrentRevision: 1,
		Revisions: []knowledge.Revision{{Number: 1, SourceRefs: entrySources}},
	}
	result, err := store.IngestEvidence(context.Background(), knowledge.IngestionCommand{
		Evidence:  knowledge.Evidence{ID: "evidence-1", IdempotencyKey: "evidence-1", SourceRefs: evidenceSources, Metadata: map[string]string{"state": "original"}},
		Candidate: &knowledge.Candidate{ID: "candidate-2", WorkspaceID: "workspace-1", TargetEntryID: "entry-1", TargetRevision: 1, Kind: knowledge.KindRequirement, Status: knowledge.StatusCandidate, Revision: 1, SourceRefs: evidenceSources},
		Entry:     &entry,
	})
	if err != nil {
		t.Fatal(err)
	}
	evidenceSources[0].Metadata["state"] = "input-mutated"
	entrySources[0].Metadata["state"] = "input-mutated"
	result.Candidate.SourceRefs[0].Metadata["state"] = "result-mutated"
	result.Entry.Revisions[0].SourceRefs[0].Metadata["state"] = "result-mutated"
	if store.evidence["evidence-1"].SourceRefs[0].Metadata["state"] != "original" {
		t.Fatalf("evidence source metadata leaked: %#v", store.evidence["evidence-1"])
	}
	assertCandidateMetadata(t, store, "candidate-2", "original")
	storedEntry, err := store.GetEntry(context.Background(), "workspace-1", entry.ID)
	if err != nil {
		t.Fatal(err)
	}
	if storedEntry.Revisions[0].SourceRefs[0].Metadata["state"] != "original" {
		t.Fatalf("entry source metadata leaked: %#v", storedEntry)
	}

	appendRevision := knowledge.Revision{Number: 2, CreatedAt: time.Now(), SourceRefs: []knowledge.SourceRef{{Type: "source", ID: "entry-1", Metadata: map[string]string{"state": "original"}}}}
	_, revised, err := store.ReviewCandidate(context.Background(), knowledge.ReviewCommand{
		CandidateID: "candidate-2", WorkspaceID: "workspace-1", ExpectedRevision: 1,
		ExpectedEntryRevision: 1, NewStatus: knowledge.StatusPublished,
		Review: knowledge.Review{ReviewedAt: time.Now()}, AppendRevision: &appendRevision,
	})
	if err != nil {
		t.Fatal(err)
	}
	appendRevision.SourceRefs[0].Metadata["state"] = "input-mutated"
	revised.Revisions[1].SourceRefs[0].Metadata["state"] = "result-mutated"
	storedEntry, err = store.GetEntry(context.Background(), "workspace-1", entry.ID)
	if err != nil {
		t.Fatal(err)
	}
	if storedEntry.Revisions[1].SourceRefs[0].Metadata["state"] != "original" {
		t.Fatalf("appended revision source metadata leaked: %#v", storedEntry)
	}
}

func assertCandidateMetadata(t *testing.T, store *Store, id, want string) {
	t.Helper()
	candidate, err := store.GetCandidate(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if candidate.SourceRefs[0].Metadata["state"] != want {
		t.Fatalf("candidate source metadata = %#v, want %q", candidate.SourceRefs, want)
	}
}
