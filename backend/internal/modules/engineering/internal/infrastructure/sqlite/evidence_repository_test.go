package sqlite_test

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	engineering "github.com/hvritual/workspace/internal/modules/engineering"
	"github.com/hvritual/workspace/internal/modules/engineering/internal/domain"
	persistence "github.com/hvritual/workspace/internal/modules/engineering/internal/infrastructure/sqlite"
)

func TestEvidenceRepositoryImmutabilityIsolationAndOrdering(t *testing.T) {
	ctx := context.Background()
	db := openDB(t, filepath.Join(t.TempDir(), "evidence.db"))
	if err := engineering.MigrateSqlite(ctx, db); err != nil {
		t.Fatal(err)
	}
	store, err := persistence.New(db)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 1, 15, 0, 0, 0, time.UTC)
	subject, err := domain.NewNodeRef(domain.NodeKindRun, "run-1")
	if err != nil {
		t.Fatal(err)
	}
	source, err := domain.NewEvidenceSource("runtime", "runtime://workspace-1/project-1/run-1/evidence-1", "event-17", strings.Repeat("a", 64), now)
	if err != nil {
		t.Fatal(err)
	}
	first, err := domain.NewEvidenceEnvelope("evidence-1", "workspace-1", domain.EvidenceKindExecution, subject, source, "runner-1", "artifact://runtime/550e8400-e29b-41d4-a716-446655440000", strings.Repeat("b", 64), now)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.PutEvidence(ctx, first); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.GetEvidence(ctx, "workspace-1", first.ID())
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ContentChecksum() != first.ContentChecksum() || loaded.Source().Revision() != "event-17" {
		t.Fatalf("loaded evidence = %#v", loaded)
	}

	replay, err := domain.NewEvidenceEnvelope(first.ID(), first.WorkspaceID(), first.Kind(), first.Subject(), first.Source(), first.ProducerID(), first.ArtifactURI(), first.ArtifactDigest(), now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.PutEvidence(ctx, replay); err != nil {
		t.Fatalf("idempotent replay: %v", err)
	}
	loadedAgain, err := store.GetEvidence(ctx, "workspace-1", first.ID())
	if err != nil {
		t.Fatal(err)
	}
	if !loadedAgain.CapturedAt().Equal(now) {
		t.Fatalf("idempotent replay mutated capture time: %v", loadedAgain.CapturedAt())
	}

	changedSource, err := domain.NewEvidenceSource("runtime", source.Locator(), "event-18", strings.Repeat("c", 64), now)
	if err != nil {
		t.Fatal(err)
	}
	conflict, err := domain.NewEvidenceEnvelope(first.ID(), first.WorkspaceID(), first.Kind(), first.Subject(), changedSource, first.ProducerID(), first.ArtifactURI(), first.ArtifactDigest(), now)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.PutEvidence(ctx, conflict); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("conflicting replay error = %v", err)
	}

	second, err := domain.NewEvidenceEnvelope("evidence-0", "workspace-1", domain.EvidenceKindExecution, subject, source, "runner-1", "", "", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.PutEvidence(ctx, second); err != nil {
		t.Fatal(err)
	}
	foreign, err := domain.NewEvidenceEnvelope("evidence-1", "workspace-2", domain.EvidenceKindExecution, subject, source, "runner-2", "", "", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.PutEvidence(ctx, foreign); err != nil {
		t.Fatal(err)
	}

	values, err := store.ListEvidence(ctx, "workspace-1", &subject)
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 2 || values[0].ID() != "evidence-0" || values[1].ID() != "evidence-1" {
		t.Fatalf("ordered evidence = %#v", values)
	}
	foreignValues, err := store.ListEvidence(ctx, "workspace-2", nil)
	if err != nil || len(foreignValues) != 1 || foreignValues[0].ProducerID() != "runner-2" {
		t.Fatalf("foreign evidence = %#v err=%v", foreignValues, err)
	}
	if _, err := store.GetEvidence(ctx, "workspace-1", "missing"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("missing evidence error = %v", err)
	}
}
