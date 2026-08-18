package system

import (
	"context"
	"database/sql"
	"io/fs"
	"testing"

	_ "modernc.org/sqlite"
)

func TestSkillCatalogDownMigrationOnlyRunsWhenCatalogIsEmpty(t *testing.T) {
	for _, test := range []struct {
		name      string
		seed      bool
		wantError bool
	}{
		{name: "empty catalog"},
		{name: "retained catalog", seed: true, wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			db, err := sql.Open("sqlite", ":memory:")
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if err := MigrateSqlite(context.Background(), db); err != nil {
				t.Fatal(err)
			}
			if test.seed {
				if _, err := db.Exec(`INSERT INTO system_skills(id,origin_workspace_id,revision,created_by,created_at,updated_at) VALUES('skill-1','workspace-1',1,'user-1','2026-08-18T00:00:00Z','2026-08-18T00:00:00Z')`); err != nil {
					t.Fatal(err)
				}
			}
			down, err := fs.ReadFile(sqliteMigrationFiles, "internal/infrastructure/sqlite/migrations/000001_skill_catalog.down.sql")
			if err != nil {
				t.Fatal(err)
			}
			_, err = db.Exec(string(down))
			if (err != nil) != test.wantError {
				t.Fatalf("down migration error = %v, wantError %v", err, test.wantError)
			}
			var present int
			err = db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='system_skills'`).Scan(&present)
			if err != nil {
				t.Fatal(err)
			}
			if test.wantError && present != 1 {
				t.Fatalf("retained Skill table present = %d, want 1", present)
			}
			if !test.wantError && present != 0 {
				t.Fatalf("empty Skill table present = %d, want 0", present)
			}
		})
	}
}
