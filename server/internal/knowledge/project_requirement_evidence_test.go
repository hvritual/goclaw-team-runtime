package knowledge_test

import (
	"context"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/knowledge"
	"github.com/multica-ai/multica/server/internal/knowledge/adapter/memory"
)

func TestProjectRequirementEvidenceRetainsStableApprovedItemProvenance(t *testing.T) {
	approvedAt := time.Date(2026, 8, 1, 9, 0, 0, 0, time.UTC)
	evidence := knowledge.NewProjectRequirementEvidence(knowledge.ProjectRequirementEvidenceDraft{
		WorkspaceID: "workspace-1", ProjectID: "project-1", BaselineID: "baseline-1",
		ApprovedRevision: 2, RequirementKey: "scope-1", Section: "in_scope",
		Content: "Keep published history", ActorID: "member-1", ApprovedAt: approvedAt,
		LinkedIssueIDs: []string{"issue-1", "issue-2"},
	}, knowledge.KindRequirement)

	if evidence.SourceID != "baseline-1:scope-1" || evidence.SourceRevision != "2" || evidence.EventType != "project_requirement.baseline_approved" {
		t.Fatalf("source identity = %#v", evidence)
	}
	if evidence.Metadata["baseline_id"] != "baseline-1" || evidence.Metadata["requirement_key"] != "scope-1" || evidence.Metadata["section"] != "in_scope" || evidence.Metadata["approved_revision"] != "2" {
		t.Fatalf("evidence metadata = %#v", evidence.Metadata)
	}
	if len(evidence.SourceRefs) != 3 {
		t.Fatalf("source refs = %#v", evidence.SourceRefs)
	}
	item := evidence.SourceRefs[0]
	if item.Type != "project_requirement_item" || item.ID != "baseline-1:scope-1" || item.Revision != "2" || item.Checksum != evidence.Checksum {
		t.Fatalf("item source ref = %#v", item)
	}
	if item.Metadata["section"] != "in_scope" || item.Metadata["requirement_key"] != "scope-1" {
		t.Fatalf("item source metadata = %#v", item.Metadata)
	}
	if evidence.SourceRefs[1].ID != "issue-1" || evidence.SourceRefs[2].ID != "issue-2" {
		t.Fatalf("issue source refs = %#v", evidence.SourceRefs)
	}
}

func TestProjectRequirementLaterApprovalTargetsPublishedEntry(t *testing.T) {
	service := knowledge.NewService(memory.New(), knowledge.DefaultPromotionPolicy{})
	first := knowledge.NewProjectRequirementEvidence(knowledge.ProjectRequirementEvidenceDraft{
		WorkspaceID: "workspace-1", ProjectID: "project-1", BaselineID: "baseline-1",
		ApprovedRevision: 1, RequirementKey: "goal-1", Section: "goals", Content: "Ship safely",
		ActorID: "member-1", ApprovedAt: time.Now(),
	}, knowledge.KindGoal)
	result, err := service.IngestOutboxEvidence(context.Background(), first)
	if err != nil || result.Candidate == nil {
		t.Fatalf("first evidence result = %#v, err = %v", result, err)
	}
	_, entry, err := service.Review(context.Background(), knowledge.ReviewInput{
		WorkspaceID: "workspace-1", CandidateID: result.Candidate.ID, ExpectedRevision: 1,
		Action: knowledge.ReviewApprove, ReviewerID: "reviewer-1", Rationale: "Approved baseline.",
	})
	if err != nil || entry == nil {
		t.Fatalf("publish first approval: entry=%#v err=%v", entry, err)
	}
	second := knowledge.NewProjectRequirementEvidence(knowledge.ProjectRequirementEvidenceDraft{
		WorkspaceID: "workspace-1", ProjectID: "project-1", BaselineID: "baseline-1",
		ApprovedRevision: 2, RequirementKey: "goal-1", Section: "goals", Content: "Ship safely with rollback",
		ActorID: "member-1", ApprovedAt: time.Now(),
	}, knowledge.KindGoal)
	result, err = service.IngestOutboxEvidence(context.Background(), second)
	if err != nil || result.Candidate == nil {
		t.Fatalf("second evidence result = %#v, err = %v", result, err)
	}
	if result.Candidate.TargetEntryID != entry.ID || result.Candidate.TargetRevision != 1 {
		t.Fatalf("revision candidate = %#v, published entry = %#v", result.Candidate, entry)
	}
}
