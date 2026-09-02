package sqlite_test

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	engineering "github.com/hvritual/workspace/internal/modules/engineering"
	"github.com/hvritual/workspace/internal/modules/engineering/internal/domain"
	persistence "github.com/hvritual/workspace/internal/modules/engineering/internal/infrastructure/sqlite"
)

func TestExecutionEvidenceRepositoryRoundTripAndIsolation(t *testing.T) {
	ctx := context.Background()
	db := openDB(t, filepath.Join(t.TempDir(), "execution-evidence.db"))
	if err := engineering.MigrateSqlite(ctx, db); err != nil {
		t.Fatal(err)
	}
	store, err := persistence.New(db)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 2, 4, 30, 0, 0, time.UTC)

	item, err := domain.NewExecutionItem(
		"execution-1", "workspace-1", domain.ExecutionItemKindBuild,
		"github_actions", "build-17", "https://ci.example/builds/17", now,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateExecutionItem(ctx, item); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateExecutionItem(ctx, item); err != nil {
		t.Fatalf("idempotent execution item create: %v", err)
	}
	conflictingID, _ := domain.NewExecutionItem(
		item.ID(), item.WorkspaceID(), item.Kind(), item.SourceType(), item.SourceID(), "https://ci.example/builds/17-rebound", now,
	)
	if err := store.CreateExecutionItem(ctx, conflictingID); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("execution item id conflict = %v", err)
	}
	conflictingSource, _ := domain.NewExecutionItem(
		"execution-2", item.WorkspaceID(), item.Kind(), item.SourceType(), item.SourceID(), item.SourceLocator(), now,
	)
	if err := store.CreateExecutionItem(ctx, conflictingSource); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("execution item source conflict = %v", err)
	}
	loadedItem, err := store.GetExecutionItem(ctx, item.WorkspaceID(), item.ID())
	if err != nil {
		t.Fatal(err)
	}
	if loadedItem.Kind() != item.Kind() || loadedItem.SourceID() != item.SourceID() || !loadedItem.CreatedAt().Equal(item.CreatedAt()) {
		t.Fatalf("execution item round trip = %#v", loadedItem)
	}
	if _, err := store.GetExecutionItem(ctx, "workspace-missing", item.ID()); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("execution item workspace isolation = %v", err)
	}

	evidence := newRepositoryEvidence(t, "workspace-1", "evidence-1", domain.EvidenceOutcomePassed, now)
	if err := store.CreateEvidence(ctx, evidence); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateEvidence(ctx, evidence); err != nil {
		t.Fatalf("idempotent evidence create: %v", err)
	}
	conflictingEvidence := newRepositoryEvidence(t, "workspace-1", "evidence-1", domain.EvidenceOutcomeFailed, now)
	if err := store.CreateEvidence(ctx, conflictingEvidence); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("evidence id conflict = %v", err)
	}
	loadedEvidence, err := store.GetEvidence(ctx, evidence.WorkspaceID(), evidence.ID())
	if err != nil {
		t.Fatal(err)
	}
	if loadedEvidence.ContentChecksum() != evidence.ContentChecksum() || string(loadedEvidence.Payload()) != string(evidence.Payload()) {
		t.Fatalf("evidence round trip checksum=%q payload=%s", loadedEvidence.ContentChecksum(), loadedEvidence.Payload())
	}
	if _, err := store.GetEvidence(ctx, "workspace-missing", evidence.ID()); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("evidence workspace isolation = %v", err)
	}

	foreignEvidence := newRepositoryEvidence(t, "workspace-2", "foreign-evidence", domain.EvidenceOutcomePassed, now)
	if err := store.CreateEvidence(ctx, foreignEvidence); err != nil {
		t.Fatal(err)
	}
	crossWorkspace, _ := domain.NewEvidenceAttachment("workspace-1", item.ID(), foreignEvidence.ID(), now.Add(time.Minute))
	if err := store.AttachEvidence(ctx, crossWorkspace); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("cross-workspace attachment = %v", err)
	}
	missingItem, _ := domain.NewEvidenceAttachment("workspace-1", "missing-item", evidence.ID(), now.Add(time.Minute))
	if err := store.AttachEvidence(ctx, missingItem); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("missing execution item attachment = %v", err)
	}

	firstAttachedAt := now.Add(time.Minute)
	attachment, _ := domain.NewEvidenceAttachment("workspace-1", item.ID(), evidence.ID(), firstAttachedAt)
	if err := store.AttachEvidence(ctx, attachment); err != nil {
		t.Fatal(err)
	}
	repeated, _ := domain.NewEvidenceAttachment("workspace-1", item.ID(), evidence.ID(), now.Add(2*time.Minute))
	if err := store.AttachEvidence(ctx, repeated); err != nil {
		t.Fatalf("idempotent attachment = %v", err)
	}
	attachments, err := store.ListEvidenceAttachments(ctx, "workspace-1", item.ID())
	if err != nil {
		t.Fatal(err)
	}
	if len(attachments) != 1 || attachments[0].EvidenceID() != evidence.ID() || !attachments[0].AttachedAt().Equal(firstAttachedAt) {
		t.Fatalf("attachments = %#v", attachments)
	}
	if _, err := store.ListEvidenceAttachments(ctx, "workspace-missing", item.ID()); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("attachment list workspace isolation = %v", err)
	}

	if _, err := db.ExecContext(ctx, `UPDATE engineering_evidence SET content_checksum=? WHERE workspace_id=? AND id=?`,
		strings.Repeat("f", 64), evidence.WorkspaceID(), evidence.ID()); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetEvidence(ctx, evidence.WorkspaceID(), evidence.ID()); !errors.Is(err, domain.ErrEvidenceChecksumMismatch) {
		t.Fatalf("corrupted evidence checksum = %v", err)
	}
}

func newRepositoryEvidence(t *testing.T, workspaceID, id string, outcome domain.EvidenceOutcome, now time.Time) domain.EvidenceEnvelope {
	t.Helper()
	source, err := domain.NewEvidenceSource(
		"ci", "validation-17", "https://ci.example/validations/17", "attempt-1", strings.Repeat("a", 64), now,
	)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := domain.NewEvidenceArtifact("artifact://ci/builds/17", strings.Repeat("b", 64))
	if err != nil {
		t.Fatal(err)
	}
	value, err := domain.NewEvidenceEnvelope(
		id, workspaceID, domain.EvidenceKindValidation, outcome, source, "validator-1", &artifact,
		json.RawMessage(`{"suite":"go test","packages":42}`), now.Add(30*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	return value
}
