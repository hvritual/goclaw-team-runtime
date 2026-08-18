package space

import (
	"database/sql"
	"io/fs"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestSpaceSQLiteMigrationsAreOrderedAtomicAndRepeatable(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "space-migrations.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for range 2 {
		if err := MigrateSqlite(t.Context(), db); err != nil {
			t.Fatal(err)
		}
	}
	var migrations int
	if err := db.QueryRow(`SELECT COUNT(*) FROM space_schema_migrations`).Scan(&migrations); err != nil || migrations != 2 {
		t.Fatalf("Space migrations = %d, %v", migrations, err)
	}
	for _, table := range []string{"space_assets", "space_asset_versions", "space_skill_objects"} {
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&count); err != nil || count != 1 {
			t.Fatalf("table %s = %d, %v", table, count, err)
		}
		rows, err := db.Query(`PRAGMA foreign_key_list(` + table + `)`)
		if err != nil {
			t.Fatal(err)
		}
		if rows.Next() {
			rows.Close()
			t.Fatalf("table %s unexpectedly has a foreign key", table)
		}
		if err := rows.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSkillObjectDownMigrationRefusesRetainedObjects(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "space-down.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := MigrateSqlite(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO space_skill_objects(id,workspace_id,object_key,media_type,size_bytes,checksum,content,state,created_at) VALUES('object-1','workspace-1','skill/workspace-1/object-1','text/plain',1,'checksum',X'78','committed','2026-08-18T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	down, err := fs.ReadFile(sqliteMigrationFiles, "internal/infrastructure/sqlite/migrations/000002_skill_objects.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(down)); err == nil {
		t.Fatal("down migration removed retained Skill objects")
	}
	var present int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='space_skill_objects'`).Scan(&present); err != nil || present != 1 {
		t.Fatalf("retained Skill object table present = %d, %v", present, err)
	}
}
