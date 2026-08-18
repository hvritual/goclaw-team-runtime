package sqlite_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/system"
	"github.com/hvritual/workspace/internal/modules/system/contract"
	sqliteinfra "github.com/hvritual/workspace/internal/modules/system/internal/infrastructure/sqlite"
	_ "modernc.org/sqlite"
)

func TestCreateCancellationRollsBackCatalogAuditAndBinding(t *testing.T) {
	db := openSkillCatalogTestDatabase(t)
	ctx, cancel := context.WithCancel(t.Context())
	repository := sqliteinfra.NewSkillCatalogRepository(db)
	_, err := repository.Create(ctx, contract.CreateSkillCatalogRequest{
		WorkspaceID: "workspace-1",
		ActorType:   "member",
		ActorID:     "user-1",
		Name:        "Canceled Skill",
		Config:      map[string]any{},
	}, "skill-1", "version-1", time.Date(2026, 8, 18, 0, 0, 0, 0, time.UTC), func(ctx context.Context, executor contract.SkillCreateExecutor) error {
		cancel()
		if err := executor.Execute(ctx, `INSERT INTO workspace_skill_bindings(workspace_id,skill_id,skill_version_id) VALUES(?,?,?)`, "workspace-1", "skill-1", "version-1"); err != nil {
			return err
		}
		return ctx.Err()
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Create() error = %v, want context.Canceled", err)
	}
	assertSkillCreateTablesEmpty(t, db)
}

func TestCreateExpiredDeadlineLeavesNoPartialRows(t *testing.T) {
	db := openSkillCatalogTestDatabase(t)
	ctx, cancel := context.WithDeadline(t.Context(), time.Now().Add(-time.Second))
	defer cancel()
	repository := sqliteinfra.NewSkillCatalogRepository(db)
	_, err := repository.Create(ctx, contract.CreateSkillCatalogRequest{
		WorkspaceID: "workspace-1",
		ActorType:   "member",
		ActorID:     "user-1",
		Name:        "Expired Skill",
		Config:      map[string]any{},
	}, "skill-1", "version-1", time.Date(2026, 8, 18, 0, 0, 0, 0, time.UTC), func(context.Context, contract.SkillCreateExecutor) error {
		t.Fatal("binding callback ran after deadline")
		return nil
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Create() error = %v, want context.DeadlineExceeded", err)
	}
	assertSkillCreateTablesEmpty(t, db)
}

func openSkillCatalogTestDatabase(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "skill-cancellation.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := system.MigrateSqlite(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE workspace_skill_bindings(workspace_id TEXT NOT NULL, skill_id TEXT NOT NULL, skill_version_id TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	return db
}

func assertSkillCreateTablesEmpty(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, table := range []string{"system_skills", "system_skill_versions", "system_skill_audit", "workspace_skill_bindings"} {
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if count != 0 {
			t.Errorf("%s rows = %d, want 0", table, count)
		}
	}
}
