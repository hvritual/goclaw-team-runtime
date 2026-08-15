package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/space/contract"
	"github.com/hvritual/workspace/internal/modules/space/internal/application"
	_ "modernc.org/sqlite"
)

func TestAttachmentRepositoryRetriesBusyWriteAcquisition(t *testing.T) {
	database, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "attachment-contention.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	database.SetMaxOpenConns(16)
	if _, err := database.Exec(`CREATE TABLE space_assets(id TEXT PRIMARY KEY,workspace_id TEXT NOT NULL,current_version_id TEXT NOT NULL,uploader_type TEXT NOT NULL,uploader_id TEXT NOT NULL,filename TEXT NOT NULL,created_at TEXT NOT NULL,updated_at TEXT NOT NULL); CREATE TABLE space_asset_versions(id TEXT PRIMARY KEY,asset_id TEXT NOT NULL,object_key TEXT NOT NULL,media_type TEXT NOT NULL,size_bytes INTEGER NOT NULL,checksum TEXT NOT NULL,created_at TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	repository, err := NewAttachmentRepository(database)
	if err != nil {
		t.Fatal(err)
	}

	const writers = 12
	start := make(chan struct{})
	results := make(chan error, writers)
	var wait sync.WaitGroup
	for index := 0; index < writers; index++ {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			<-start
			results <- repository.Create(context.Background(), application.StoredAttachment{
				ID:           fmt.Sprintf("asset-%02d", index),
				VersionID:    fmt.Sprintf("version-%02d", index),
				WorkspaceID:  "workspace-1",
				UploaderType: "member",
				UploaderID:   "member-1",
				Filename:     fmt.Sprintf("attachment-%02d.txt", index),
				ObjectKey:    fmt.Sprintf("workspace-1/asset-%02d/version-%02d.blob", index, index),
				ContentType:  "text/plain",
				SizeBytes:    1,
				Checksum:     "checksum",
				CreatedAt:    time.Now().UTC(),
			}, func(contract.AttachmentExecutor) error {
				// Keep one writer active long enough that later writers must survive
				// at least one SQLite busy acquisition rather than relying on a
				// single 5-second wait window.
				time.Sleep(550 * time.Millisecond)
				return nil
			})
		}(index)
	}
	started := time.Now()
	close(start)
	wait.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatalf("concurrent attachment creation = %v", err)
		}
	}
	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM space_assets`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != writers {
		t.Fatalf("persisted attachment rows = %d, want %d", count, writers)
	}
	if elapsed := time.Since(started); elapsed <= 5*time.Second {
		t.Fatalf("contention window = %s, want to exceed the original 5s SQLite wait", elapsed)
	}
}
