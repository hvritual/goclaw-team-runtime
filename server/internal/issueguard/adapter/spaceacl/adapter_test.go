package spaceacl

import (
	"errors"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/issueguard"
	spaceapp "github.com/multica-ai/multica/server/modules/space/application"
	"github.com/multica-ai/multica/server/modules/space/domain"
)

func TestAssetReferencePreservesEveryField(t *testing.T) {
	createdAt := time.Date(2026, time.August, 2, 12, 0, 0, 0, time.UTC)
	asset, err := domain.NewUploadedAsset(domain.UploadedAssetParams{
		ID:           "asset-1",
		WorkspaceID:  "workspace-1",
		UploaderType: domain.UploaderMember,
		UploaderID:   "member-1",
		Filename:     "notes.txt",
		URL:          "/uploads/notes.txt",
		ContentType:  "text/plain",
		SizeBytes:    5,
		Checksum:     "sha256:test",
		CreatedAt:    createdAt,
	})
	if err != nil {
		t.Fatalf("NewUploadedAsset: %v", err)
	}

	reference := assetReference(&asset)

	if reference == nil || reference.ID != asset.ID() || reference.WorkspaceID != asset.WorkspaceID() ||
		reference.UploaderType != string(asset.UploaderType()) || reference.UploaderID != asset.UploaderID() ||
		reference.Filename != asset.Filename() || reference.URL != asset.URL() ||
		reference.ContentType != asset.ContentType() || reference.SizeBytes != asset.SizeBytes() ||
		reference.Checksum != asset.Checksum() || !reference.CreatedAt.Equal(asset.CreatedAt()) {
		t.Fatalf("reference = %#v", reference)
	}
}

func TestProviderErrorMappingPreservesConsumerSentinels(t *testing.T) {
	tests := []struct {
		name string
		from error
		want error
	}{
		{name: "storage", from: spaceapp.ErrStorageUnavailable, want: issueguard.ErrStorageUnavailable},
		{name: "membership", from: spaceapp.ErrNotWorkspaceMember, want: issueguard.ErrNotWorkspaceMember},
		{name: "upload", from: spaceapp.ErrUploadFailed, want: issueguard.ErrUploadFailed},
		{name: "identity", from: spaceapp.ErrGenerateID, want: issueguard.ErrGenerateID},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mapped := mapProviderError(errors.Join(test.from, errors.New("cause")))
			if !errors.Is(mapped, test.want) || !errors.Is(mapped, test.from) {
				t.Fatalf("mapped error = %v", mapped)
			}
		})
	}
}

func TestProviderMetadataFallbackPreservesDirectLink(t *testing.T) {
	cause := errors.New("database unavailable")
	mapped := mapProviderError(&spaceapp.MetadataPersistenceError{
		Result: spaceapp.UploadResult{URL: "/uploads/notes.txt", Filename: "notes.txt"},
		Err:    cause,
	})
	var consumerError *issueguard.MetadataPersistenceError
	if !errors.As(mapped, &consumerError) || !errors.Is(mapped, cause) ||
		consumerError.Result.URL != "/uploads/notes.txt" || consumerError.Result.Filename != "notes.txt" {
		t.Fatalf("metadata error = %#v", mapped)
	}
}
