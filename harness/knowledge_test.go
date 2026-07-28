package harness

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestKnowledgeProposalApprovalAndConflict(t *testing.T) {
	root := t.TempDir()
	vault := filepath.Join(root, "vault")
	if err := os.MkdirAll(filepath.Join(vault, "02-decisions"), 0o755); err != nil {
		t.Fatal(err)
	}
	service, err := NewService(Config{
		Root:         filepath.Join(root, "runtime"),
		ProjectID:    "alpha",
		VaultPath:    vault,
		TraceEnabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	proposal, err := service.CreateKnowledgeProposal(
		"02-decisions/ADR-0001.md",
		"# ADR-0001\n\nApproved decision.\n",
		"Capture the accepted architecture.",
		"trace-1",
		"agent",
	)
	if err != nil {
		t.Fatal(err)
	}
	if proposal.Status != KnowledgeProposalPending {
		t.Fatalf("unexpected proposal status: %s", proposal.Status)
	}
	if _, err := os.Stat(filepath.Join(vault, "08-reviews", "inbox", proposal.ID+".md")); err != nil {
		t.Fatalf("missing vault projection: %v", err)
	}
	approved, err := service.ApproveKnowledgeProposal(proposal.ID, "human", "knowledge evidence accepted")
	if err != nil {
		t.Fatal(err)
	}
	if approved.Status != KnowledgeProposalApproved {
		t.Fatalf("unexpected approved status: %s", approved.Status)
	}
	data, err := os.ReadFile(filepath.Join(vault, "02-decisions", "ADR-0001.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Approved decision") {
		t.Fatalf("unexpected target content: %s", data)
	}
	results, err := service.SearchKnowledge("approved decision", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Path != "02-decisions/ADR-0001.md" {
		t.Fatalf("unexpected knowledge search results: %+v", results)
	}
	content, err := service.ReadKnowledge("02-decisions/ADR-0001.md")
	if err != nil || !strings.Contains(content, "Approved decision") {
		t.Fatalf("unexpected read result: %q, %v", content, err)
	}

	conflicting, err := service.CreateKnowledgeProposal(
		"02-decisions/ADR-0001.md",
		"# ADR-0001\n\nSecond change.\n",
		"Test synchronization conflict.",
		"",
		"agent",
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vault, "02-decisions", "ADR-0001.md"), []byte("changed elsewhere"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ApproveKnowledgeProposal(conflicting.ID, "human", "stale synchronized proposal"); err == nil {
		t.Fatal("expected synchronized-vault conflict")
	}
}

func TestGitKnowledgeProposalRejectsChangedRevision(t *testing.T) {
	root := t.TempDir()
	knowledgeRoot := filepath.Join(root, "knowledge")
	if err := os.MkdirAll(filepath.Join(knowledgeRoot, "02-decisions"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"init"},
		{"config", "user.name", "GoClaw Test"},
		{"config", "user.email", "goclaw-test@example.invalid"},
	} {
		if output, err := exec.Command("git", append([]string{"-C", knowledgeRoot}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, output)
		}
	}
	firstPath := filepath.Join(knowledgeRoot, "02-decisions", "ADR-0001.md")
	if err := os.WriteFile(firstPath, []byte("# Initial\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "."}, {"commit", "-m", "initial"}} {
		if output, err := exec.Command("git", append([]string{"-C", knowledgeRoot}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, output)
		}
	}
	service, err := NewService(Config{
		Root:             filepath.Join(root, "runtime"),
		ProjectID:        "alpha",
		KnowledgeRoot:    knowledgeRoot,
		KnowledgeBackend: "git",
	})
	if err != nil {
		t.Fatal(err)
	}
	proposal, err := service.CreateKnowledgeProposal(
		"02-decisions/ADR-0001.md",
		"# Approved\n",
		"Update the decision.",
		"trace-1",
		"agent",
	)
	if err != nil {
		t.Fatal(err)
	}
	if proposal.BaseRevision == "" || proposal.StoreKind != "git" ||
		!strings.HasPrefix(proposal.SourceURI, "git+markdown://alpha/") {
		t.Fatalf("unexpected Git proposal metadata: %+v", proposal)
	}
	secondPath := filepath.Join(knowledgeRoot, "02-decisions", "ADR-0002.md")
	if err := os.WriteFile(secondPath, []byte("# Concurrent\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "."}, {"commit", "-m", "concurrent"}} {
		if output, err := exec.Command("git", append([]string{"-C", knowledgeRoot}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, output)
		}
	}
	if _, err := service.ApproveKnowledgeProposal(
		proposal.ID,
		"human",
		"Review evidence accepted.",
	); err == nil || !strings.Contains(err.Error(), "base revision changed") {
		t.Fatalf("expected Git revision conflict, got %v", err)
	}
}

func TestKnowledgeProposalRejectsUnsafeTarget(t *testing.T) {
	service, err := NewService(Config{Root: filepath.Join(t.TempDir(), "runtime"), VaultPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateKnowledgeProposal("../secret.md", "x", "unsafe", "", "agent"); err == nil {
		t.Fatal("expected unsafe target rejection")
	}
	if _, err := service.CreateKnowledgeProposal("08-reviews/inbox/bypass.md", "x", "unsafe", "", "agent"); err == nil {
		t.Fatal("expected non-governed target rejection")
	}
}
