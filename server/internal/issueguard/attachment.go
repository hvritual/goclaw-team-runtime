// Package issueguard owns transitional Issue application policies that still
// compose the legacy sqlc persistence surface.
package issueguard

import (
	"context"
	"errors"

	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

// AttachmentReferences keeps Issue-owned attachment validation and persistence out
// of the Space domain while the installed attachment table remains shared.
type AttachmentReferences struct {
	queries *db.Queries
}

// NewAttachmentReferences creates the legacy Issue-side attachment adapter.
func NewAttachmentReferences(queries *db.Queries) *AttachmentReferences {
	return &AttachmentReferences{queries: queries}
}

// ExistsInWorkspace reports whether an Issue belongs to the upload workspace.
func (a *AttachmentReferences) ExistsInWorkspace(ctx context.Context, issueID, workspaceID string) bool {
	issueUUID, err := util.ParseUUID(issueID)
	if err != nil {
		return false
	}
	workspaceUUID, err := util.ParseUUID(workspaceID)
	if err != nil {
		return false
	}
	_, err = a.queries.GetIssueInWorkspace(ctx, db.GetIssueInWorkspaceParams{
		ID:          issueUUID,
		WorkspaceID: workspaceUUID,
	})
	return err == nil
}

// CreateForIssue atomically persists Asset metadata with its Issue relation.
func (a *AttachmentReferences) CreateForIssue(ctx context.Context, asset PreparedAsset, issueID string) (PreparedAsset, error) {
	assetUUID, err := util.ParseUUID(asset.ID)
	if err != nil {
		return PreparedAsset{}, err
	}
	issueUUID, err := util.ParseUUID(issueID)
	if err != nil {
		return PreparedAsset{}, err
	}
	workspaceUUID, err := util.ParseUUID(asset.WorkspaceID)
	if err != nil {
		return PreparedAsset{}, err
	}
	uploaderUUID, err := util.ParseUUID(asset.UploaderID)
	if err != nil {
		return PreparedAsset{}, err
	}
	row, err := a.queries.CreateAttachment(ctx, db.CreateAttachmentParams{
		ID:           assetUUID,
		WorkspaceID:  workspaceUUID,
		UploaderType: asset.UploaderType,
		UploaderID:   uploaderUUID,
		Filename:     asset.Filename,
		Url:          asset.URL,
		ContentType:  asset.ContentType,
		SizeBytes:    asset.SizeBytes,
		IssueID:      issueUUID,
	})
	if err != nil {
		return PreparedAsset{}, err
	}
	if !row.CreatedAt.Valid {
		return PreparedAsset{}, errors.New("attachment row has no creation time")
	}
	return PreparedAsset{
		ID:           util.UUIDToString(row.ID),
		WorkspaceID:  util.UUIDToString(row.WorkspaceID),
		UploaderType: row.UploaderType,
		UploaderID:   util.UUIDToString(row.UploaderID),
		Filename:     row.Filename,
		URL:          row.Url,
		ContentType:  row.ContentType,
		SizeBytes:    row.SizeBytes,
		Checksum:     asset.Checksum,
		CreatedAt:    row.CreatedAt.Time,
	}, nil
}
