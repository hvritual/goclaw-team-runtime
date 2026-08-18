package space

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"path/filepath"
	"testing"

	"github.com/hvritual/workspace/internal/modules/space/contract"
	_ "modernc.org/sqlite"
)

func TestSQLiteSkillObjectsStagePromoteOpenDiscardAndReconcile(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "skill-objects.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := MigrateSqlite(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	service, err := NewSQLiteSkillObjects(db)
	if err != nil {
		t.Fatal(err)
	}

	staged, err := service.Stage(t.Context(), contract.StageSkillObjectRequest{WorkspaceID: "workspace-1", MediaType: "text/markdown", Content: []byte("# Skill")})
	if err != nil {
		t.Fatal(err)
	}
	if staged.State != "quarantined" || staged.SizeBytes != 7 || staged.Checksum != "e2151f8490121dc5e6fd36c1d4e00b6da5593595e3eb8ece76c1d0ec3f310979" {
		t.Fatalf("staged = %#v", staged)
	}
	if _, _, err := service.Open(t.Context(), "workspace-1", staged.ID); !errors.Is(err, contract.ErrSkillObjectNotFound) {
		t.Fatalf("Open quarantined error = %v", err)
	}
	tx, err := db.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Promote(t.Context(), skillObjectTestExecutor{tx}, "workspace-1", staged.ID); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	opened, reader, err := service.Open(t.Context(), "workspace-1", staged.ID)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(reader)
	_ = reader.Close()
	if err != nil || string(body) != "# Skill" || opened.State != "committed" {
		t.Fatalf("opened = %#v body=%q err=%v", opened, body, err)
	}
	if _, _, err := service.Open(t.Context(), "workspace-2", staged.ID); !errors.Is(err, contract.ErrSkillObjectNotFound) {
		t.Fatalf("cross-workspace Open error = %v", err)
	}
	missingTx, err := db.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Promote(t.Context(), skillObjectTestExecutor{missingTx}, "workspace-2", "missing"); !errors.Is(err, contract.ErrSkillObjectNotFound) {
		_ = missingTx.Rollback()
		t.Fatalf("Promote missing error = %v", err)
	}
	_ = missingTx.Rollback()

	discarded, err := service.Stage(t.Context(), contract.StageSkillObjectRequest{WorkspaceID: "workspace-1", MediaType: "text/plain", Content: []byte("discard")})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Discard(t.Context(), "workspace-1", discarded.ID); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM space_skill_objects WHERE id=?`, discarded.ID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("discarded rows = %d, %v", count, err)
	}

	referenced, err := service.Stage(t.Context(), contract.StageSkillObjectRequest{WorkspaceID: "workspace-1", MediaType: "text/plain", Content: []byte("keep")})
	if err != nil {
		t.Fatal(err)
	}
	orphan, err := service.Stage(t.Context(), contract.StageSkillObjectRequest{WorkspaceID: "workspace-1", MediaType: "text/plain", Content: []byte("remove")})
	if err != nil {
		t.Fatal(err)
	}
	fresh, err := service.Stage(t.Context(), contract.StageSkillObjectRequest{WorkspaceID: "workspace-1", MediaType: "text/plain", Content: []byte("retain until expired")})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE space_skill_objects SET created_at='2000-01-01T00:00:00Z' WHERE id=?`, orphan.ID); err != nil {
		t.Fatal(err)
	}
	if err := service.Reconcile(t.Context(), []string{referenced.ID}); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM space_skill_objects WHERE id=? AND state='committed'`, referenced.ID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("referenced committed rows = %d, %v", count, err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM space_skill_objects WHERE id=?`, orphan.ID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("orphan rows = %d, %v", count, err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM space_skill_objects WHERE id=? AND state='quarantined'`, fresh.ID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("fresh orphan rows = %d, %v", count, err)
	}
}

type skillObjectTestExecutor struct{ tx *sql.Tx }

func (e skillObjectTestExecutor) Execute(ctx context.Context, statement string, arguments ...any) error {
	_, err := e.tx.ExecContext(ctx, statement, arguments...)
	return err
}

func (e skillObjectTestExecutor) ExecuteResult(ctx context.Context, statement string, arguments ...any) (int64, error) {
	result, err := e.tx.ExecContext(ctx, statement, arguments...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func TestSQLiteSkillObjectsHonorCanceledContext(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "skill-object-cancel.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := MigrateSqlite(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	service, err := NewSQLiteSkillObjects(db)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = service.Stage(ctx, contract.StageSkillObjectRequest{WorkspaceID: "workspace-1", MediaType: "text/plain", Content: []byte("no")})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Stage error = %v", err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM space_skill_objects`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("rows = %d, %v", count, err)
	}
}
