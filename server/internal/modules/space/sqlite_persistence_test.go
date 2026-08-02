package space

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/modules/space/contract"
	_ "modernc.org/sqlite"
)

type testWorkspaceAccess map[string]bool

func (a testWorkspaceAccess) IsMember(_ context.Context, userID, workspaceID string) (bool, error) {
	return a[userID+"/"+workspaceID], nil
}

type testObjectStore struct {
	objects map[string][]byte
	deleted []string
}

func (s *testObjectStore) Available() bool { return true }
func (s *testObjectStore) Upload(_ context.Context, key string, data []byte, _, _ string) (string, error) {
	if s.objects == nil {
		s.objects = make(map[string][]byte)
	}
	s.objects[key] = append([]byte(nil), data...)
	return "/uploads/" + key, nil
}
func (s *testObjectStore) DeleteObject(_ context.Context, key string) error {
	s.deleted = append(s.deleted, key)
	delete(s.objects, key)
	return nil
}
func (s *testObjectStore) KeyFromURL(rawURL string) string {
	return rawURL[len("/uploads/"):]
}
func (s *testObjectStore) GetReader(_ context.Context, key string) (io.ReadCloser, error) {
	value, ok := s.objects[key]
	if !ok {
		return nil, errors.New("object not found")
	}
	return io.NopCloser(bytes.NewReader(value)), nil
}

func TestSqliteAssetUploadPersistsVersionAndHidesForeignReads(t *testing.T) {
	db := openSpaceTestDB(t, "asset-lifecycle")
	objects := &testObjectStore{}
	ids := []string{"intent", "asset", "version"}
	module, err := NewWithSqlitePersistence(SqlitePersistenceConfig{
		DB: db, Objects: objects,
		WorkspaceAccess: testWorkspaceAccess{"member/workspace": true},
		NewID:           func() (string, error) { value := ids[0]; ids = ids[1:]; return value, nil },
		Now:             func() time.Time { return time.Date(2026, 8, 2, 15, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatal(err)
	}
	service := module.AssetUploads()
	result, err := service.UploadAsset(
		contract.WithAssetActor(t.Context(), "member"),
		contract.Asset_UploadAssetRequest{WorkspaceId: "workspace", Filename: "notes.txt", MediaType: "text/plain", Content: []byte("hello")},
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.Asset == nil || result.Asset.Id != "asset" || result.Asset.CurrentVersionId != "version" {
		t.Fatalf("unexpected asset: %+v", result.Asset)
	}
	var state, checksum string
	var size int64
	if err := db.QueryRowContext(t.Context(), `SELECT i.state, v.checksum, v.size_bytes
		FROM space_upload_intents i JOIN space_asset_versions v ON v.asset_id = i.asset_id
		WHERE i.id = 'intent'`).Scan(&state, &checksum, &size); err != nil {
		t.Fatal(err)
	}
	if state != "finalized" || checksum != result.Asset.Checksum || size != 5 {
		t.Fatalf("state=%q checksum=%q size=%d", state, checksum, size)
	}
	read, err := service.GetAsset(
		contract.WithAssetActor(t.Context(), "member"),
		contract.Asset_GetAssetRequest{AssetId: "asset"},
	)
	if err != nil || read.Asset == nil || read.Asset.ObjectKey == "" {
		t.Fatalf("GetAsset() = %+v, %v", read, err)
	}
	_, err = service.GetAsset(
		contract.WithAssetActor(t.Context(), "outsider"),
		contract.Asset_GetAssetRequest{AssetId: "asset"},
	)
	if !errors.Is(err, contract.ErrAssetNotFound) {
		t.Fatalf("foreign GetAsset() error = %v", err)
	}
}

func TestSqliteAssetFinalizeRollbackKeepsCleanupIntent(t *testing.T) {
	db := openSpaceTestDB(t, "asset-rollback")
	if _, err := db.ExecContext(t.Context(), `CREATE TRIGGER reject_space_version BEFORE INSERT ON space_asset_versions
		BEGIN SELECT RAISE(FAIL, 'reject version'); END`); err != nil {
		t.Fatal(err)
	}
	objects := &testObjectStore{}
	ids := []string{"intent", "asset", "version"}
	module, err := NewWithSqlitePersistence(SqlitePersistenceConfig{
		DB: db, Objects: objects,
		WorkspaceAccess: testWorkspaceAccess{"member/workspace": true},
		NewID:           func() (string, error) { value := ids[0]; ids = ids[1:]; return value, nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = module.AssetUploads().UploadAsset(
		contract.WithAssetActor(t.Context(), "member"),
		contract.Asset_UploadAssetRequest{WorkspaceId: "workspace", Filename: "x.txt", MediaType: "text/plain", Content: []byte("x")},
	)
	if !errors.Is(err, contract.ErrAssetFinalizeFailed) {
		t.Fatalf("UploadAsset() error = %v", err)
	}
	var assetCount int
	if err := db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM space_assets`).Scan(&assetCount); err != nil {
		t.Fatal(err)
	}
	var state string
	if err := db.QueryRowContext(t.Context(), `SELECT state FROM space_upload_intents WHERE id = 'intent'`).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if assetCount != 0 || state != "cleaned" || len(objects.deleted) != 1 {
		t.Fatalf("assetCount=%d state=%q deleted=%v", assetCount, state, objects.deleted)
	}
}

func TestSqliteAssetReconcilesCrashLeftoverIntent(t *testing.T) {
	db := openSpaceTestDB(t, "asset-reconcile")
	createdAt := "2026-08-02T15:00:00Z"
	if _, err := db.ExecContext(t.Context(), `INSERT INTO space_upload_intents(
		id, asset_id, version_id, workspace_id, uploader_type, uploader_id,
		filename, object_key, media_type, size_bytes, checksum, state, created_at, updated_at
	) VALUES ('intent', 'asset', 'version', 'workspace', 'member', 'member',
		'x.txt', 'workspaces/workspace/asset.txt', 'text/plain', 1, 'sha256:x', 'pending', ?, ?)`, createdAt, createdAt); err != nil {
		t.Fatal(err)
	}
	objects := &testObjectStore{objects: map[string][]byte{"workspaces/workspace/asset.txt": []byte("x")}}
	module, err := NewWithSqlitePersistence(SqlitePersistenceConfig{
		DB: db, Objects: objects, WorkspaceAccess: testWorkspaceAccess{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := module.AssetUploads().ReconcilePendingUploads(t.Context()); err != nil {
		t.Fatal(err)
	}
	var state string
	if err := db.QueryRowContext(t.Context(), `SELECT state FROM space_upload_intents WHERE id = 'intent'`).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != "cleaned" || len(objects.objects) != 0 {
		t.Fatalf("state=%q objects=%v", state, objects.objects)
	}
}

func openSpaceTestDB(t *testing.T, name string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+name+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	if err := MigrateSqlite(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	return db
}
