package sqlite_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/system"
	"github.com/hvritual/workspace/internal/modules/system/contract"
	sqliteinfra "github.com/hvritual/workspace/internal/modules/system/internal/infrastructure/sqlite"
	_ "modernc.org/sqlite"
)

func TestCreateFileVersionCopiesManifestWithoutMutatingHistory(t *testing.T) {
	db, err := sql.Open("sqlite", t.TempDir()+"/skill-files.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := system.MigrateSqlite(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	seedSkillManifest(t, db)
	repository := sqliteinfra.NewSkillCatalogRepository(db)
	identity := contract.SkillIdentity{WorkspaceID: "workspace-1", ActorType: "member", ActorID: "user-1"}
	entry, err := repository.CreateFileVersion(t.Context(), identity, "skill-1", contract.SkillFileMutation{
		Path: "scripts/run.py", ExpectedRevision: 1,
		Object: &contract.SkillFileManifest{ID: "manifest-2", SpaceObjectID: "object-2", MediaType: "text/x-python", SizeBytes: 12, Checksum: "checksum-2"},
	}, "version-2", time.Date(2026, 8, 18, 1, 0, 0, 0, time.UTC), func(ctx context.Context, executor contract.SkillCreateExecutor, objectID string) error {
		if objectID != "object-2" {
			t.Fatalf("promoted object = %q", objectID)
		}
		return executor.Execute(ctx, `INSERT INTO promotion_evidence(object_id) VALUES(?)`, objectID)
	})
	if err != nil {
		t.Fatal(err)
	}
	if entry.VersionID != "version-2" || entry.Version != "2" || entry.Revision != 2 || entry.Status != "draft" {
		t.Fatalf("entry = %#v", entry)
	}
	oldFiles, err := repository.ListFiles(t.Context(), identity, "skill-1", "version-1")
	if err != nil {
		t.Fatal(err)
	}
	newFiles, err := repository.ListFiles(t.Context(), identity, "skill-1", "version-2")
	if err != nil {
		t.Fatal(err)
	}
	if len(oldFiles) != 1 || oldFiles[0].SpaceObjectID != "object-1" || len(newFiles) != 2 || newFiles[1].Path != "scripts/run.py" {
		t.Fatalf("old=%#v new=%#v", oldFiles, newFiles)
	}
	var promoted int
	if err := db.QueryRow(`SELECT COUNT(*) FROM promotion_evidence WHERE object_id='object-2'`).Scan(&promoted); err != nil || promoted != 1 {
		t.Fatalf("promotion = %d, %v", promoted, err)
	}
}

func TestListFilesRetainsExactDeprecatedArchivedVersion(t *testing.T) {
	db, err := sql.Open("sqlite", t.TempDir()+"/skill-files-retained.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := system.MigrateSqlite(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	seedSkillManifest(t, db)
	if _, err := db.Exec(`UPDATE system_skill_versions SET status='deprecated'; UPDATE system_skills SET archived_at='2026-08-18T02:00:00Z'`); err != nil {
		t.Fatal(err)
	}
	repository := sqliteinfra.NewSkillCatalogRepository(db)
	identity := contract.SkillIdentity{WorkspaceID: "workspace-1", ActorType: "agent", ActorID: "agent-1"}
	files, err := repository.ListFiles(t.Context(), identity, "skill-1", "version-1")
	if err != nil || len(files) != 1 || files[0].Path != "SKILL.md" {
		t.Fatalf("retained files = %#v, %v", files, err)
	}
}

func TestCreateFileVersionRollsBackManifestAndRevisionWhenPromotionFails(t *testing.T) {
	db, err := sql.Open("sqlite", t.TempDir()+"/skill-files-rollback.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := system.MigrateSqlite(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	seedSkillManifest(t, db)
	repository := sqliteinfra.NewSkillCatalogRepository(db)
	identity := contract.SkillIdentity{WorkspaceID: "workspace-1", ActorType: "member", ActorID: "user-1"}
	_, err = repository.CreateFileVersion(t.Context(), identity, "skill-1", contract.SkillFileMutation{
		Path: "scripts/run.py", ExpectedRevision: 1,
		Object: &contract.SkillFileManifest{ID: "manifest-2", SpaceObjectID: "object-2", MediaType: "text/plain", SizeBytes: 1, Checksum: "checksum-2"},
	}, "version-2", time.Now(), func(context.Context, contract.SkillCreateExecutor, string) error { return context.Canceled })
	if err == nil {
		t.Fatal("CreateFileVersion() error = nil")
	}
	var versions, manifests, revision int
	_ = db.QueryRow(`SELECT COUNT(*) FROM system_skill_versions`).Scan(&versions)
	_ = db.QueryRow(`SELECT COUNT(*) FROM system_skill_file_manifests`).Scan(&manifests)
	_ = db.QueryRow(`SELECT revision FROM system_skills WHERE id='skill-1'`).Scan(&revision)
	if versions != 1 || manifests != 1 || revision != 1 {
		t.Fatalf("versions=%d manifests=%d revision=%d", versions, manifests, revision)
	}
}

func seedSkillManifest(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`
		CREATE TABLE promotion_evidence(object_id TEXT PRIMARY KEY);
		INSERT INTO system_skills(id,origin_workspace_id,revision,created_by,created_at,updated_at) VALUES('skill-1','workspace-1',1,'user-1','2026-08-18T00:00:00Z','2026-08-18T00:00:00Z');
		INSERT INTO system_skill_versions(id,skill_id,version_number,name,description,configuration,status,created_by,created_at) VALUES('version-1','skill-1',1,'Demo','','{}','draft','user-1','2026-08-18T00:00:00Z');
		INSERT INTO system_skill_file_manifests(id,skill_id,version_id,path,space_object_id,media_type,size_bytes,checksum,created_at) VALUES('manifest-1','skill-1','version-1','SKILL.md','object-1','text/markdown',6,'checksum-1','2026-08-18T00:00:00Z');
	`)
	if err != nil {
		t.Fatal(err)
	}
}
