package domain

import (
	"testing"
	"time"
)

func TestNewUploadedAssetRequiresWorkspaceBoundary(t *testing.T) {
	_, err := NewUploadedAsset(UploadedAssetParams{
		ID:           "asset-1",
		UploaderType: UploaderMember,
		UploaderID:   "member-1",
		Filename:     "notes.txt",
		URL:          "/uploads/notes.txt",
		ContentType:  "text/plain",
		SizeBytes:    5,
		Checksum:     "sha256:example",
	})
	if err == nil {
		t.Fatal("NewUploadedAsset accepted an asset without a workspace")
	}
}

func TestNewUploadedAssetCapturesImmutableUploadMetadata(t *testing.T) {
	createdAt := time.Date(2026, time.August, 2, 10, 30, 0, 0, time.UTC)
	asset, err := NewUploadedAsset(UploadedAssetParams{
		ID:           "asset-1",
		WorkspaceID:  "workspace-1",
		UploaderType: UploaderMember,
		UploaderID:   "member-1",
		Filename:     "notes.txt",
		URL:          "/uploads/workspaces/workspace-1/asset-1.txt",
		ContentType:  "text/plain; charset=utf-8",
		SizeBytes:    5,
		Checksum:     "sha256:example",
		CreatedAt:    createdAt,
	})
	if err != nil {
		t.Fatalf("NewUploadedAsset: %v", err)
	}

	if asset.ID() != "asset-1" || asset.WorkspaceID() != "workspace-1" {
		t.Fatalf("unexpected identity: id=%q workspace=%q", asset.ID(), asset.WorkspaceID())
	}
	if asset.Filename() != "notes.txt" || asset.SizeBytes() != 5 {
		t.Fatalf("unexpected metadata: filename=%q size=%d", asset.Filename(), asset.SizeBytes())
	}
	if asset.Checksum() != "sha256:example" {
		t.Fatalf("checksum = %q", asset.Checksum())
	}
	if !asset.CreatedAt().Equal(createdAt) {
		t.Fatalf("created at = %s, want %s", asset.CreatedAt(), createdAt)
	}
}
