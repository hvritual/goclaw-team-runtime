package application

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/modules/space/contract"
	"github.com/multica-ai/multica/server/internal/modules/space/internal/domain/asset"
)

type fakeAssetRepository struct {
	intents        []asset.UploadIntent
	pending        []asset.UploadIntent
	stored         asset.Asset
	beginErr       error
	finalizeErr    error
	findErr        error
	cleanupPending bool
	cleaned        bool
}

func (r *fakeAssetRepository) BeginUpload(_ context.Context, intent asset.UploadIntent) error {
	r.intents = append(r.intents, intent)
	return r.beginErr
}
func (r *fakeAssetRepository) FinalizeUpload(_ context.Context, intent asset.UploadIntent, rawURL string) (asset.Asset, error) {
	if r.finalizeErr != nil {
		return asset.Asset{}, r.finalizeErr
	}
	value, err := asset.Finalize(intent, rawURL)
	r.stored = value
	return value, err
}
func (r *fakeAssetRepository) FindByID(context.Context, string) (asset.Asset, error) {
	return r.stored, r.findErr
}
func (r *fakeAssetRepository) ListPendingUploads(context.Context) ([]asset.UploadIntent, error) {
	return append([]asset.UploadIntent(nil), r.pending...), nil
}
func (r *fakeAssetRepository) MarkCleanupPending(context.Context, string, string) error {
	r.cleanupPending = true
	return nil
}
func (r *fakeAssetRepository) MarkCleaned(context.Context, string) error {
	r.cleaned = true
	return nil
}

type fakeWorkspaceAccess struct {
	allowed bool
	err     error
}

func (a fakeWorkspaceAccess) IsMember(context.Context, string, string) (bool, error) {
	return a.allowed, a.err
}

type fakeObjectStore struct {
	available bool
	uploaded  bool
	deleted   []string
	uploadErr error
	deleteErr error
}

func (s *fakeObjectStore) Available() bool { return s.available }
func (s *fakeObjectStore) Upload(_ context.Context, key string, _ []byte, _, _ string) (string, error) {
	s.uploaded = true
	return "/uploads/" + key, s.uploadErr
}
func (s *fakeObjectStore) DeleteObject(_ context.Context, key string) error {
	s.deleted = append(s.deleted, key)
	return s.deleteErr
}
func (s *fakeObjectStore) KeyFromURL(value string) string { return value }
func (s *fakeObjectStore) GetReader(context.Context, string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(nil)), nil
}

func TestUploadAssetPersistsIntentBeforeFinalizedAsset(t *testing.T) {
	repository := &fakeAssetRepository{}
	objects := &fakeObjectStore{available: true}
	ids := []string{"intent", "asset", "version"}
	service := NewAssetService(
		WithAssetRepository(repository), WithWorkspaceAccess(fakeWorkspaceAccess{allowed: true}),
		WithObjectStore(objects), WithAssetClock(func() time.Time { return time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC) }),
		WithAssetIDGenerator(func() (string, error) { value := ids[0]; ids = ids[1:]; return value, nil }),
	)

	result, err := service.UploadAsset(
		contract.WithAssetActor(context.Background(), "user"),
		contract.Asset_UploadAssetRequest{WorkspaceId: "workspace", Filename: "notes.txt", MediaType: "text/plain", Content: []byte("hello")},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(repository.intents) != 1 || !objects.uploaded || result.Asset == nil {
		t.Fatalf("upload was not finalized: intents=%d uploaded=%v result=%+v", len(repository.intents), objects.uploaded, result)
	}
	if result.Asset.Id != "asset" || result.Asset.CurrentVersionId != "version" || result.Asset.SizeBytes != "5" {
		t.Fatalf("unexpected asset: %+v", result.Asset)
	}
	if result.Asset.Checksum != "sha256:2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824" {
		t.Fatalf("checksum = %q", result.Asset.Checksum)
	}
}

func TestUploadAssetRejectsForeignWorkspaceBeforeIntentOrObject(t *testing.T) {
	repository := &fakeAssetRepository{}
	objects := &fakeObjectStore{available: true}
	service := NewAssetService(
		WithAssetRepository(repository), WithWorkspaceAccess(fakeWorkspaceAccess{}), WithObjectStore(objects),
	)
	_, err := service.UploadAsset(
		contract.WithAssetActor(context.Background(), "user"),
		contract.Asset_UploadAssetRequest{WorkspaceId: "workspace", Filename: "x.txt", MediaType: "text/plain"},
	)
	if !errors.Is(err, contract.ErrAssetWorkspaceForbidden) {
		t.Fatalf("UploadAsset() error = %v", err)
	}
	if len(repository.intents) != 0 || objects.uploaded {
		t.Fatal("denied upload changed persistence")
	}
}

func TestUploadAssetFinalizationFailureDeletesObjectAndClosesIntent(t *testing.T) {
	repository := &fakeAssetRepository{finalizeErr: errors.New("disk full")}
	objects := &fakeObjectStore{available: true}
	ids := []string{"intent", "asset", "version"}
	service := NewAssetService(
		WithAssetRepository(repository), WithWorkspaceAccess(fakeWorkspaceAccess{allowed: true}), WithObjectStore(objects),
		WithAssetIDGenerator(func() (string, error) { value := ids[0]; ids = ids[1:]; return value, nil }),
	)
	_, err := service.UploadAsset(
		contract.WithAssetActor(context.Background(), "user"),
		contract.Asset_UploadAssetRequest{WorkspaceId: "workspace", Filename: "x.txt", MediaType: "text/plain"},
	)
	if !errors.Is(err, contract.ErrAssetFinalizeFailed) {
		t.Fatalf("UploadAsset() error = %v", err)
	}
	if !repository.cleanupPending || !repository.cleaned || len(objects.deleted) != 1 {
		t.Fatalf("cleanup state pending=%v cleaned=%v deleted=%v", repository.cleanupPending, repository.cleaned, objects.deleted)
	}
}

func TestReconcilePendingUploadsRetainsIntentUntilObjectDeleteSucceeds(t *testing.T) {
	intent, err := asset.NewUploadIntent(asset.UploadIntent{
		ID: "intent", AssetID: "asset", VersionID: "version", WorkspaceID: "workspace",
		UploaderType: asset.UploaderMember, UploaderID: "user", Filename: "x.txt",
		ObjectKey: "workspaces/workspace/asset.txt", MediaType: "text/plain",
		Checksum: "sha256:x", CreatedAt: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	repository := &fakeAssetRepository{pending: []asset.UploadIntent{intent}}
	objects := &fakeObjectStore{available: true, deleteErr: errors.New("offline")}
	service := NewAssetService(WithAssetRepository(repository), WithObjectStore(objects))
	if err := service.ReconcilePendingUploads(context.Background()); err == nil {
		t.Fatal("ReconcilePendingUploads() error = nil")
	}
	if repository.cleaned {
		t.Fatal("failed object deletion marked intent clean")
	}
	objects.deleteErr = nil
	if err := service.ReconcilePendingUploads(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !repository.cleaned {
		t.Fatal("successful retry did not close intent")
	}
}

func TestPersonalUploadReturnsDirectObjectWithoutAssetMetadata(t *testing.T) {
	repository := &fakeAssetRepository{}
	objects := &fakeObjectStore{available: true}
	service := NewAssetService(
		WithAssetRepository(repository), WithWorkspaceAccess(fakeWorkspaceAccess{allowed: true}), WithObjectStore(objects),
		WithAssetIDGenerator(func() (string, error) { return "object", nil }),
	)
	result, err := service.UploadAsset(
		contract.WithAssetActor(context.Background(), "user"),
		contract.Asset_UploadAssetRequest{Filename: "avatar.png", MediaType: "image/png", Content: []byte("png")},
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.Asset != nil || result.DirectObjectId != "object" || len(repository.intents) != 0 {
		t.Fatalf("unexpected direct result: %+v", result)
	}
}
