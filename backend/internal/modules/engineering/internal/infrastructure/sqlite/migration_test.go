package sqlite_test

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	engineering "github.com/hvritual/workspace/internal/modules/engineering"
	"github.com/hvritual/workspace/internal/modules/engineering/internal/domain"
	persistence "github.com/hvritual/workspace/internal/modules/engineering/internal/infrastructure/sqlite"
	_ "modernc.org/sqlite"
)

func TestMigrationReopenRollbackAndReapply(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "reopen.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if err := engineering.MigrateSqlite(ctx, db); err != nil {
		t.Fatal(err)
	}
	store, err := persistence.New(db)
	if err != nil {
		t.Fatal(err)
	}
	entity, _ := domain.NewEngineeringEntity("service:ota", "workspace-1", domain.EntityTypeService, "OTA", domain.EntityStatusActive, "team:iot")
	if err := store.PutEntity(ctx, entity); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	db, err = sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := engineering.MigrateSqlite(ctx, db); err != nil {
		t.Fatal(err)
	}
	store, _ = persistence.New(db)
	if _, err := store.GetEntity(ctx, "workspace-1", entity.ID()); err != nil {
		t.Fatalf("reopen get: %v", err)
	}
	if err := engineering.RollbackSqliteLatest(ctx, db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `SELECT 1 FROM engineering_evidence_envelopes LIMIT 1`); err == nil {
		t.Fatal("engineering_evidence_envelopes must be removed after latest rollback")
	}
	if _, err := db.ExecContext(ctx, `SELECT 1 FROM engineering_entities LIMIT 1`); err != nil {
		t.Fatalf("base engineering schema must survive evidence rollback: %v", err)
	}
	if err := engineering.RollbackSqliteLatest(ctx, db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `SELECT 1 FROM engineering_entities LIMIT 1`); err == nil {
		t.Fatal("engineering_entities must be removed after base rollback")
	}
	if err := engineering.MigrateSqlite(ctx, db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `SELECT 1 FROM engineering_entities LIMIT 1`); err != nil {
		t.Fatalf("reapplied base schema: %v", err)
	}
	if _, err := db.ExecContext(ctx, `SELECT 1 FROM engineering_evidence_envelopes LIMIT 1`); err != nil {
		t.Fatalf("reapplied evidence schema: %v", err)
	}
}

func TestSchemaHasNoForeignKeys(t *testing.T) {
	ctx := context.Background()
	db := openDB(t, filepath.Join(t.TempDir(), "no-fk.db"))
	if err := engineering.MigrateSqlite(ctx, db); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{
		"engineering_entities", "engineering_source_bindings", "engineering_thread_edges", "engineering_changes",
		"engineering_change_entities", "engineering_change_artifacts", "engineering_context_packs",
		"engineering_context_pack_targets", "engineering_context_pack_references", "engineering_evidence_envelopes",
	} {
		rows, err := db.QueryContext(ctx, `PRAGMA foreign_key_list(`+table+`)`)
		if err != nil {
			t.Fatal(err)
		}
		if rows.Next() {
			_ = rows.Close()
			t.Fatalf("%s unexpectedly has a foreign key", table)
		}
		if err := rows.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestConcurrentEntityWriters(t *testing.T) {
	ctx := context.Background()
	db := openDB(t, filepath.Join(t.TempDir(), "concurrent.db"))
	if err := engineering.MigrateSqlite(ctx, db); err != nil {
		t.Fatal(err)
	}
	store, err := persistence.New(db)
	if err != nil {
		t.Fatal(err)
	}
	const writers = 24
	var wg sync.WaitGroup
	errs := make(chan error, writers)
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			id := fmt.Sprintf("service:%02d", index)
			entity, err := domain.NewEngineeringEntity(id, "workspace-1", domain.EntityTypeService, id, domain.EntityStatusActive, "team:iot")
			if err == nil {
				err = store.PutEntity(ctx, entity)
			}
			errs <- err
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	values, err := store.ListEntities(ctx, "workspace-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != writers {
		t.Fatalf("entities=%d want=%d", len(values), writers)
	}
}
