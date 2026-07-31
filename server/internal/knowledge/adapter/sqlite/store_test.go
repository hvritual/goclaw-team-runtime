package sqlite_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/knowledge"
	"github.com/multica-ai/multica/server/internal/knowledge/adapter/sqlite"
)

func TestOpenRejectsNewerSchemaWithoutChangingIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "knowledge.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		CREATE TABLE knowledge_schema_version (
			version INTEGER PRIMARY KEY,
			applied_at TEXT NOT NULL
		);
		INSERT INTO knowledge_schema_version(version, applied_at)
		VALUES (?, '2026-07-31T00:00:00Z')`, sqlite.SchemaVersion+1); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	if _, err := sqlite.Open(path); err == nil {
		t.Fatal("opening a newer knowledge schema must fail")
	}

	db, err = sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var version int
	if err := db.QueryRow(`SELECT MAX(version) FROM knowledge_schema_version`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != sqlite.SchemaVersion+1 {
		t.Fatalf("schema version changed to %d", version)
	}
}

func TestApprovedRevisionProposalSurvivesSQLiteRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "knowledge.db")
	store, err := sqlite.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	service := knowledge.NewService(store, nil)
	ctx := context.Background()
	baseline, err := service.Propose(ctx, knowledge.ProposalInput{
		WorkspaceID: "workspace-1",
		Kind:        knowledge.KindProcedure,
		Title:       "Backup procedure",
		Content:     "Stop writes before backup.",
		Reason:      "Establish a governed procedure.",
		ProposedBy:  "user-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, entry, err := service.Review(ctx, knowledge.ReviewInput{
		WorkspaceID:      "workspace-1",
		CandidateID:      baseline.ID,
		ExpectedRevision: 1,
		Action:           knowledge.ReviewApprove,
		ReviewerID:       "admin-1",
		Rationale:        "Baseline verified.",
	})
	if err != nil || entry == nil {
		t.Fatalf("publish baseline: entry=%#v err=%v", entry, err)
	}
	revision, err := service.Propose(ctx, knowledge.ProposalInput{
		WorkspaceID:   "workspace-1",
		TargetEntryID: entry.ID,
		Kind:          knowledge.KindProcedure,
		Title:         "Backup and restore procedure",
		Content:       "Stop writes, checkpoint WAL, then back up.",
		Reason:        "A restore drill identified the missing checkpoint.",
		ProposedBy:    "user-2",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, revised, err := service.Review(ctx, knowledge.ReviewInput{
		WorkspaceID:      "workspace-1",
		CandidateID:      revision.ID,
		ExpectedRevision: 1,
		Action:           knowledge.ReviewApprove,
		ReviewerID:       "admin-1",
		Rationale:        "Restore drill passed.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if revised == nil || revised.ID != entry.ID || revised.CurrentRevision != 2 {
		t.Fatalf("revised entry = %#v", revised)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := sqlite.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	persisted, err := reopened.GetEntry(ctx, "workspace-1", entry.ID)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.CurrentRevision != 2 || len(persisted.Revisions) != 2 {
		t.Fatalf("persisted revisions = %#v", persisted.Revisions)
	}
	if persisted.Revisions[1].SupersedesRevision != 1 {
		t.Fatalf("persisted supersedes revision = %d, want 1", persisted.Revisions[1].SupersedesRevision)
	}
}

func TestOpenProtectsFilesAndReportsSQLiteCapabilities(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "private", "knowledge")
	path := filepath.Join(directory, "knowledge.db")
	store, err := sqlite.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	directoryInfo, err := os.Stat(directory)
	if err != nil {
		t.Fatal(err)
	}
	if directoryInfo.Mode().Perm() != 0o700 {
		t.Fatalf("directory mode = %o, want 700", directoryInfo.Mode().Perm())
	}
	fileInfo, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fileInfo.Mode().Perm() != 0o600 {
		t.Fatalf("file mode = %o, want 600", fileInfo.Mode().Perm())
	}

	capabilities, err := store.Capabilities(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if capabilities.SchemaVersion != sqlite.SchemaVersion {
		t.Fatalf("schema version = %d, want %d", capabilities.SchemaVersion, sqlite.SchemaVersion)
	}
	if capabilities.JournalMode != "wal" {
		t.Fatalf("journal mode = %q, want wal", capabilities.JournalMode)
	}
	if capabilities.ForeignKeys {
		t.Fatal("knowledge SQLite must not enable foreign keys")
	}
}

func TestSQLiteWriteLockFailsCleanlyAndRecovers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "knowledge.db")
	store, err := sqlite.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	locker, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = locker.Close() })
	connection, err := locker.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = connection.Close() })
	if _, err := connection.ExecContext(context.Background(), `BEGIN IMMEDIATE`); err != nil {
		t.Fatal(err)
	}

	lockedContext, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, err = store.CreateCandidate(lockedContext, knowledge.Candidate{
		WorkspaceID: "workspace-1", Kind: knowledge.KindLesson,
		Title: "Locked write", Content: "Must fail without corrupting the store.",
		Reason: "Verify lock behavior.", Status: knowledge.StatusCandidate,
		Revision: 1, ProposedBy: "user-1",
	})
	if err == nil {
		t.Fatal("write under an exclusive SQLite lock unexpectedly succeeded")
	}
	if _, rollbackErr := connection.ExecContext(context.Background(), `ROLLBACK`); rollbackErr != nil {
		t.Fatal(rollbackErr)
	}
	if _, err := store.CreateCandidate(context.Background(), knowledge.Candidate{
		WorkspaceID: "workspace-1", Kind: knowledge.KindLesson,
		Title: "Recovered write", Content: "The adapter remains writable after lock release.",
		Reason: "Verify recovery.", Status: knowledge.StatusCandidate,
		Revision: 1, ProposedBy: "user-1",
	}); err != nil {
		t.Fatalf("write after lock release: %v", err)
	}
}

func TestBackupCreatesAConsistentRestorableDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source", "knowledge.db")
	store, err := sqlite.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	service := knowledge.NewService(store, nil)
	candidate, err := service.Propose(context.Background(), knowledge.ProposalInput{
		WorkspaceID: "workspace-1",
		Kind:        knowledge.KindProcedure,
		Title:       "Back up SQLite",
		Content:     "Use the adapter backup operation.",
		Reason:      "Copying an active WAL database is unsafe.",
		ProposedBy:  "user-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, entry, err := service.Review(context.Background(), knowledge.ReviewInput{
		WorkspaceID:      "workspace-1",
		CandidateID:      candidate.ID,
		ExpectedRevision: 1,
		Action:           knowledge.ReviewApprove,
		ReviewerID:       "admin-1",
		Rationale:        "The backup operation is covered by a restore test.",
	})
	if err != nil {
		t.Fatal(err)
	}

	backupPath := filepath.Join(t.TempDir(), "backup", "knowledge.db")
	if err := store.Backup(context.Background(), backupPath); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	restored, err := sqlite.Open(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = restored.Close() })
	if _, err := restored.GetEntry(context.Background(), "workspace-1", entry.ID); err != nil {
		t.Fatalf("read restored entry: %v", err)
	}
}
