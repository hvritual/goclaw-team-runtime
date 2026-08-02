package uploadhttp

import (
	"errors"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/issueguard"
	spaceapp "github.com/multica-ai/multica/server/modules/space/application"
	spacehttp "github.com/multica-ai/multica/server/modules/space/interfaces/http"
)

func TestDomainAssetPreservesEveryField(t *testing.T) {
	createdAt := time.Date(2026, time.August, 2, 12, 0, 0, 0, time.UTC)
	reference := &issueguard.PreparedAsset{
		ID:           "asset-1",
		WorkspaceID:  "workspace-1",
		UploaderType: "member",
		UploaderID:   "member-1",
		Filename:     "notes.txt",
		URL:          "/uploads/notes.txt",
		ContentType:  "text/plain",
		SizeBytes:    5,
		Checksum:     "sha256:test",
		CreatedAt:    createdAt,
	}

	asset, err := domainAsset(reference)

	if err != nil {
		t.Fatalf("domainAsset: %v", err)
	}
	if asset == nil || asset.ID() != reference.ID || asset.WorkspaceID() != reference.WorkspaceID ||
		string(asset.UploaderType()) != reference.UploaderType || asset.UploaderID() != reference.UploaderID ||
		asset.Filename() != reference.Filename || asset.URL() != reference.URL ||
		asset.ContentType() != reference.ContentType || asset.SizeBytes() != reference.SizeBytes ||
		asset.Checksum() != reference.Checksum || !asset.CreatedAt().Equal(reference.CreatedAt) {
		t.Fatalf("asset = %#v", asset)
	}
}

func TestWorkflowErrorMappingPreservesHTTPContracts(t *testing.T) {
	tests := []struct {
		name string
		from error
		want error
	}{
		{name: "Issue visibility", from: issueguard.ErrIssueNotAccessible, want: spacehttp.ErrIssueNotAccessible},
		{name: "Issue syntax", from: issueguard.ErrInvalidIssueID, want: spacehttp.ErrInvalidIssueID},
		{name: "storage", from: issueguard.ErrStorageUnavailable, want: spaceapp.ErrStorageUnavailable},
		{name: "membership", from: issueguard.ErrNotWorkspaceMember, want: spaceapp.ErrNotWorkspaceMember},
		{name: "upload", from: issueguard.ErrUploadFailed, want: spaceapp.ErrUploadFailed},
		{name: "identity", from: issueguard.ErrGenerateID, want: spaceapp.ErrGenerateID},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mapped := mapWorkflowError(errors.Join(test.from, errors.New("cause")))
			if !errors.Is(mapped, test.want) || !errors.Is(mapped, test.from) {
				t.Fatalf("mapped error = %v", mapped)
			}
		})
	}
}

func TestWorkflowMetadataFallbackPreservesDirectLink(t *testing.T) {
	cause := errors.New("database unavailable")
	mapped := mapWorkflowError(&issueguard.MetadataPersistenceError{
		Result: issueguard.UploadResult{URL: "/uploads/notes.txt", Filename: "notes.txt"},
		Err:    cause,
	})
	var transportError *spaceapp.MetadataPersistenceError
	if !errors.As(mapped, &transportError) || !errors.Is(mapped, cause) ||
		transportError.Result.URL != "/uploads/notes.txt" || transportError.Result.Filename != "notes.txt" {
		t.Fatalf("metadata error = %#v", mapped)
	}
}
