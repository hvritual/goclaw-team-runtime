package domain

import (
	"errors"
	"testing"
	"time"
)

func TestExecutionItemNormalizesStableProjectionIdentity(t *testing.T) {
	now := time.Date(2026, 9, 2, 6, 30, 0, 0, time.FixedZone("UTC+2", 2*60*60))
	item, err := NewExecutionItem(
		" execution-1 ",
		" workspace-1 ",
		ExecutionItemKind(" BUILD "),
		" GitHub_Actions ",
		" run-17 ",
		"https://ci.example/runs/17",
		now,
	)
	if err != nil {
		t.Fatal(err)
	}
	if item.ID() != "execution-1" || item.WorkspaceID() != "workspace-1" {
		t.Fatalf("identity = %q/%q", item.WorkspaceID(), item.ID())
	}
	if item.Kind() != ExecutionItemKindBuild || item.SourceType() != "github_actions" || item.SourceID() != "run-17" {
		t.Fatalf("source identity = %q/%q/%q", item.Kind(), item.SourceType(), item.SourceID())
	}
	if item.SourceLocator() != "https://ci.example/runs/17" {
		t.Fatalf("source locator = %q", item.SourceLocator())
	}
	if item.CreatedAt().Location() != time.UTC || !item.CreatedAt().Equal(now) {
		t.Fatalf("created_at = %v", item.CreatedAt())
	}
}

func TestExecutionItemKindVocabulary(t *testing.T) {
	valid := []ExecutionItemKind{
		ExecutionItemKindRun,
		ExecutionItemKindBuild,
		ExecutionItemKindTest,
		ExecutionItemKindRelease,
		ExecutionItemKindDeployment,
		ExecutionItemKindObservation,
	}
	for _, kind := range valid {
		if !kind.Valid() {
			t.Fatalf("valid kind rejected: %q", kind)
		}
	}
	if ExecutionItemKind("task").Valid() {
		t.Fatal("workspace task must not become an execution item kind")
	}
}

func TestExecutionItemRejectsInvalidIdentityAndUnsafeLocator(t *testing.T) {
	now := time.Date(2026, 9, 2, 4, 30, 0, 0, time.UTC)
	tests := []struct {
		name          string
		id            string
		workspaceID   string
		kind          ExecutionItemKind
		sourceType    string
		sourceID      string
		sourceLocator string
		createdAt     time.Time
		want          error
	}{
		{"id", "", "workspace-1", ExecutionItemKindRun, "runtime", "run-1", "runtime://workspace-1/runs/run-1", now, ErrExecutionItemIDRequired},
		{"workspace", "execution-1", "", ExecutionItemKindRun, "runtime", "run-1", "runtime://workspace-1/runs/run-1", now, ErrWorkspaceIDRequired},
		{"kind", "execution-1", "workspace-1", "task", "runtime", "run-1", "runtime://workspace-1/runs/run-1", now, ErrExecutionItemKindInvalid},
		{"source type", "execution-1", "workspace-1", ExecutionItemKindRun, "", "run-1", "runtime://workspace-1/runs/run-1", now, ErrExecutionItemSourceTypeRequired},
		{"source id", "execution-1", "workspace-1", ExecutionItemKindRun, "runtime", "", "runtime://workspace-1/runs/run-1", now, ErrExecutionItemSourceIDRequired},
		{"credentials", "execution-1", "workspace-1", ExecutionItemKindBuild, "ci", "run-1", "https://token@example.com/runs/1", now, ErrExecutionItemSourceLocatorInvalid},
		{"query", "execution-1", "workspace-1", ExecutionItemKindBuild, "ci", "run-1", "https://ci.example/runs/1?token=secret", now, ErrExecutionItemSourceLocatorInvalid},
		{"created", "execution-1", "workspace-1", ExecutionItemKindRun, "runtime", "run-1", "runtime://workspace-1/runs/run-1", time.Time{}, ErrExecutionItemCreatedAtRequired},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewExecutionItem(tc.id, tc.workspaceID, tc.kind, tc.sourceType, tc.sourceID, tc.sourceLocator, tc.createdAt)
			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v want %v", err, tc.want)
			}
		})
	}
}

func TestEvidenceAttachmentIsSeparateWorkspaceScopedRelation(t *testing.T) {
	now := time.Date(2026, 9, 2, 4, 30, 0, 0, time.UTC)
	value, err := NewEvidenceAttachment(" workspace-1 ", " execution-1 ", " evidence-1 ", now)
	if err != nil {
		t.Fatal(err)
	}
	if value.WorkspaceID() != "workspace-1" || value.ExecutionItemID() != "execution-1" || value.EvidenceID() != "evidence-1" {
		t.Fatalf("attachment identity = %#v", value)
	}
	if !value.AttachedAt().Equal(now) || value.AttachedAt().Location() != time.UTC {
		t.Fatalf("attached_at = %v", value.AttachedAt())
	}

	if _, err := NewEvidenceAttachment("", "execution-1", "evidence-1", now); !errors.Is(err, ErrWorkspaceIDRequired) {
		t.Fatalf("workspace err = %v", err)
	}
	if _, err := NewEvidenceAttachment("workspace-1", "", "evidence-1", now); !errors.Is(err, ErrEvidenceAttachmentItemIDRequired) {
		t.Fatalf("item err = %v", err)
	}
	if _, err := NewEvidenceAttachment("workspace-1", "execution-1", "", now); !errors.Is(err, ErrEvidenceAttachmentEvidenceRequired) {
		t.Fatalf("evidence err = %v", err)
	}
	if _, err := NewEvidenceAttachment("workspace-1", "execution-1", "evidence-1", time.Time{}); !errors.Is(err, ErrEvidenceAttachmentTimeRequired) {
		t.Fatalf("time err = %v", err)
	}
}
