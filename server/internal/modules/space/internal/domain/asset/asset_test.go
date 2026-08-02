package asset

import (
	"errors"
	"testing"
	"time"
)

func TestUploadIntentAndFinalizePreserveImmutableFacts(t *testing.T) {
	createdAt := time.Date(2026, time.August, 2, 14, 0, 0, 0, time.FixedZone("test", 8*60*60))
	intent, err := NewUploadIntent(UploadIntent{
		ID: "intent", AssetID: "asset", VersionID: "version", WorkspaceID: "workspace",
		UploaderType: UploaderMember, UploaderID: "user", Filename: "notes.txt",
		ObjectKey: "workspaces/workspace/asset.txt", MediaType: "text/plain",
		SizeBytes: 5, Checksum: "sha256:abc", CreatedAt: createdAt,
	})
	if err != nil {
		t.Fatal(err)
	}
	value, err := Finalize(intent, "/uploads/workspaces/workspace/asset.txt")
	if err != nil {
		t.Fatal(err)
	}
	if value.ID() != "asset" || value.CurrentVersionID() != "version" || value.ObjectKey() != intent.ObjectKey {
		t.Fatalf("unexpected identity: %+v", value)
	}
	if value.Checksum() != "sha256:abc" || value.SizeBytes() != 5 || !value.CreatedAt().Equal(createdAt) {
		t.Fatalf("unexpected immutable facts: %+v", value)
	}
}

func TestUploadIntentRejectsMissingRecoveryFacts(t *testing.T) {
	_, err := NewUploadIntent(UploadIntent{ID: "intent"})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("NewUploadIntent() error = %v", err)
	}
}
