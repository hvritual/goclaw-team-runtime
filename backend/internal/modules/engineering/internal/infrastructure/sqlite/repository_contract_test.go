package sqlite_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	engineering "github.com/hvritual/workspace/internal/modules/engineering"
	"github.com/hvritual/workspace/internal/modules/engineering/internal/domain"
	persistence "github.com/hvritual/workspace/internal/modules/engineering/internal/infrastructure/sqlite"
	_ "modernc.org/sqlite"
)

func TestRepositoryContract(t *testing.T) {
	ctx := context.Background()
	db := openDB(t, filepath.Join(t.TempDir(), "engineering.db"))
	if err := engineering.MigrateSqlite(ctx, db); err != nil {
		t.Fatal(err)
	}
	store, err := persistence.New(db)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 28, 10, 0, 0, 0, time.UTC)
	provenance, err := domain.NewProvenance("github", "github://hvritual/iot/device-gateway", "abc123", now)
	if err != nil {
		t.Fatal(err)
	}

	entity, err := domain.NewEngineeringEntity("service:device-gateway", "workspace-1", domain.EntityTypeService, "Device Gateway", domain.EntityStatusActive, "team:iot")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.PutEntity(ctx, entity); err != nil {
		t.Fatal(err)
	}
	foreign, _ := domain.NewEngineeringEntity(entity.ID(), "workspace-2", domain.EntityTypeService, "Foreign", domain.EntityStatusActive, "team:other")
	if err := store.PutEntity(ctx, foreign); err != nil {
		t.Fatal(err)
	}
	loadedEntity, err := store.GetEntity(ctx, "workspace-1", entity.ID())
	if err != nil || loadedEntity.Name() != "Device Gateway" {
		t.Fatalf("entity=%#v err=%v", loadedEntity, err)
	}
	entities, err := store.ListEntities(ctx, "workspace-1")
	if err != nil || len(entities) != 1 {
		t.Fatalf("entities=%d err=%v", len(entities), err)
	}

	binding, err := domain.NewSourceBinding("binding-1", "workspace-1", entity.ID(), provenance, domain.AuthorityAuthoritative)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.PutSourceBinding(ctx, binding); err != nil {
		t.Fatal(err)
	}
	bindings, err := store.ListSourceBindings(ctx, "workspace-1", entity.ID())
	if err != nil || len(bindings) != 1 || bindings[0].Provenance().Revision() != "abc123" {
		t.Fatalf("bindings=%#v err=%v", bindings, err)
	}

	projectRef, _ := domain.NewNodeRef(domain.NodeKindProject, "project-1")
	entityRef, _ := domain.NewNodeRef(domain.NodeKindEngineeringEntity, entity.ID())
	edge, err := domain.NewThreadEdge("edge-1", "workspace-1", projectRef, domain.RelationChanges, entityRef, domain.AuthorityAuthoritative, provenance)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.PutThreadEdge(ctx, edge); err != nil {
		t.Fatal(err)
	}
	edges, err := store.ListThreadEdges(ctx, "workspace-1", entityRef)
	if err != nil || len(edges) != 1 || edges[0].From().ID() != "project-1" {
		t.Fatalf("edges=%#v err=%v", edges, err)
	}

	taskRef, _ := domain.NewNodeRef(domain.NodeKindTask, "task-1")
	artifact, _ := domain.NewArtifactRef("pull_request", "github://hvritual/iot/pull/42", "merge-sha")
	change, err := domain.NewChange("change-1", "workspace-1", "project-1", "requirement-1", &taskRef, "run-1", "Improve reconnect strategy", []string{entity.ID()}, []domain.ArtifactRef{artifact}, provenance, now)
	if err != nil {
		t.Fatal(err)
	}
	change, err = change.Accept(now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.PutChange(ctx, change); err != nil {
		t.Fatal(err)
	}
	loadedChange, err := store.GetChange(ctx, "workspace-1", change.ID())
	if err != nil || loadedChange.Status() != domain.ChangeStatusAccepted || len(loadedChange.Artifacts()) != 1 || loadedChange.Artifacts()[0].Revision() != "merge-sha" {
		t.Fatalf("change=%#v err=%v", loadedChange, err)
	}
	changes, err := store.ListChanges(ctx, "workspace-1", entity.ID())
	if err != nil || len(changes) != 1 {
		t.Fatalf("changes=%d err=%v", len(changes), err)
	}

	reference, _ := domain.NewContextReference(domain.ContextKindArchitecture, "ARCH-001", "r3", "sha256:arch")
	pack, err := domain.NewContextPack("context-1", "workspace-1", taskRef, "r7", []string{entity.ID()}, []domain.ContextReference{reference}, "policy-v1", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.PutContextPack(ctx, pack); err != nil {
		t.Fatal(err)
	}
	if err := store.PutContextPack(ctx, pack); err != nil {
		t.Fatalf("idempotent context pack write: %v", err)
	}
	loadedPack, err := store.GetContextPack(ctx, "workspace-1", pack.ID())
	if err != nil || loadedPack.Checksum() != pack.Checksum() || len(loadedPack.References()) != 1 {
		t.Fatalf("context pack=%#v err=%v", loadedPack, err)
	}
	conflict, _ := domain.NewContextPack(pack.ID(), "workspace-1", taskRef, "r7", []string{entity.ID()}, []domain.ContextReference{reference}, "policy-v1", now.Add(time.Second))
	if err := store.PutContextPack(ctx, conflict); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("conflict err=%v", err)
	}

	if _, err := store.GetEntity(ctx, "workspace-1", "missing"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("not found err=%v", err)
	}
}

func openDB(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}
