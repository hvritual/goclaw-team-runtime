package space

import (
	"database/sql"
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
	if err := db.QueryRow(`SELECT COUNT(*) FROM space_schema_migrations`).Scan(&migrations); err != nil || migrations != 1 {
		t.Fatalf("Space migrations = %d, %v", migrations, err)
	}
	for _, table := range []string{"space_assets", "space_asset_versions"} {
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
