package application

import (
	"context"
	"errors"
	"testing"

	"github.com/multica-ai/multica/server/modules/space/domain"
)

type fakeAssetRepository struct {
	created []domain.Asset
	err     error
}

func (f *fakeAssetRepository) Create(_ context.Context, asset domain.Asset) (domain.Asset, error) {
	f.created = append(f.created, asset)
	return asset, f.err
}

type fakeWorkspaceAccess struct {
	allowed bool
	calls   int
}

func (f *fakeWorkspaceAccess) IsMember(_ context.Context, _, _ string) bool {
	f.calls++
	return f.allowed
}

type fakeObjectStore struct {
	available bool
	keys      []string
	url       string
	err       error
}

func (f *fakeObjectStore) Available() bool { return f.available }

func (f *fakeObjectStore) Upload(_ context.Context, key string, _ []byte, _, _ string) (string, error) {
	f.keys = append(f.keys, key)
	return f.url, f.err
}

type fixedIDGenerator struct {
	id  string
	err error
}

func (f fixedIDGenerator) NewID() (string, error) { return f.id, f.err }

type fixedChecksummer struct{ value string }

func (f fixedChecksummer) Sum(_ []byte) string { return f.value }

func TestUploadWorkspaceAssetPersistsAfterAuthorizationAndStorage(t *testing.T) {
	repo := &fakeAssetRepository{}
	members := &fakeWorkspaceAccess{allowed: true}
	objects := &fakeObjectStore{available: true, url: "/uploads/workspaces/ws-1/asset-1.txt"}
	service := NewUploadService(repo, members, objects, fixedIDGenerator{id: "asset-1"}, fixedChecksummer{value: "sha256:test"})

	result, err := service.Upload(context.Background(), UploadCommand{
		UserID:      "member-1",
		WorkspaceID: "ws-1",
		Filename:    "notes.txt",
		ContentType: "text/plain",
		Content:     []byte("hello"),
	})
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if result.Asset == nil || result.Asset.ID() != "asset-1" {
		t.Fatalf("persisted asset = %#v", result.Asset)
	}
	if len(objects.keys) != 1 || objects.keys[0] != "workspaces/ws-1/asset-1.txt" {
		t.Fatalf("object keys = %v", objects.keys)
	}
	if len(repo.created) != 1 || repo.created[0].WorkspaceID() != "ws-1" {
		t.Fatalf("created assets = %#v", repo.created)
	}
	if repo.created[0].Checksum() != "sha256:test" {
		t.Fatalf("checksum = %q", repo.created[0].Checksum())
	}
}

func TestUploadRejectsNonMemberBeforeWritingObject(t *testing.T) {
	repo := &fakeAssetRepository{}
	objects := &fakeObjectStore{available: true}
	service := NewUploadService(
		repo,
		&fakeWorkspaceAccess{allowed: false},
		objects,
		fixedIDGenerator{id: "asset-1"},
		fixedChecksummer{value: "sha256:test"},
	)

	_, err := service.Upload(context.Background(), UploadCommand{
		UserID: "member-1", WorkspaceID: "ws-1", Filename: "notes.txt", Content: []byte("hello"),
	})
	if !errors.Is(err, ErrNotWorkspaceMember) {
		t.Fatalf("Upload error = %v, want ErrNotWorkspaceMember", err)
	}
	if len(objects.keys) != 0 || len(repo.created) != 0 {
		t.Fatalf("denied upload wrote object=%v metadata=%d", objects.keys, len(repo.created))
	}
}

func TestUploadPreservesLegacyDirectLinkWhenMetadataInsertFails(t *testing.T) {
	repoErr := errors.New("database unavailable")
	service := NewUploadService(
		&fakeAssetRepository{err: repoErr},
		&fakeWorkspaceAccess{allowed: true},
		&fakeObjectStore{available: true, url: "/uploads/workspaces/ws-1/asset-1.txt"},
		fixedIDGenerator{id: "asset-1"},
		fixedChecksummer{value: "sha256:test"},
	)

	result, err := service.Upload(context.Background(), UploadCommand{
		UserID: "member-1", WorkspaceID: "ws-1", Filename: "notes.txt", Content: []byte("hello"),
	})
	var metadataErr *MetadataPersistenceError
	if !errors.As(err, &metadataErr) {
		t.Fatalf("Upload error = %v, want MetadataPersistenceError", err)
	}
	if result.Asset != nil || metadataErr.Result.ID != "" || metadataErr.Result.URL == "" {
		t.Fatalf("legacy fallback result = %#v", metadataErr.Result)
	}
	if !errors.Is(metadataErr, repoErr) {
		t.Fatalf("metadata error = %v, want %v", metadataErr, repoErr)
	}
}

func TestUploadWithoutWorkspaceKeepsPersonalObjectBehavior(t *testing.T) {
	repo := &fakeAssetRepository{}
	members := &fakeWorkspaceAccess{allowed: false}
	objects := &fakeObjectStore{available: true, url: "/uploads/users/user-1/asset-1.png"}
	service := NewUploadService(repo, members, objects, fixedIDGenerator{id: "asset-1"}, fixedChecksummer{value: "sha256:test"})

	result, err := service.Upload(context.Background(), UploadCommand{
		UserID: "user-1", Filename: "avatar.png", ContentType: "image/png", Content: []byte("png"),
	})
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if result.ID != "asset-1" || result.URL != objects.url || result.Asset != nil {
		t.Fatalf("personal upload result = %#v", result)
	}
	if members.calls != 0 || len(repo.created) != 0 {
		t.Fatalf("personal upload checked members=%d or persisted=%d", members.calls, len(repo.created))
	}
}
