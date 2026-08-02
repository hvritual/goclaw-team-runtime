// Package postgres adapts sqlc attachment persistence to the Space application ports.
package postgres

import (
	"context"
	"errors"

	"github.com/multica-ai/multica/server/internal/util"
	"github.com/multica-ai/multica/server/modules/space/domain"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

// Adapter implements Asset persistence with sqlc.
type Adapter struct {
	queries *db.Queries
}

// New creates a PostgreSQL/sqlc Space adapter.
func New(queries *db.Queries) *Adapter {
	return &Adapter{queries: queries}
}

// Create persists Asset metadata through the existing attachment sqlc query.
func (a *Adapter) Create(ctx context.Context, asset domain.Asset) (domain.Asset, error) {
	id, err := util.ParseUUID(asset.ID())
	if err != nil {
		return domain.Asset{}, err
	}
	workspaceID, err := util.ParseUUID(asset.WorkspaceID())
	if err != nil {
		return domain.Asset{}, err
	}
	uploaderID, err := util.ParseUUID(asset.UploaderID())
	if err != nil {
		return domain.Asset{}, err
	}
	row, err := a.queries.CreateAttachment(ctx, db.CreateAttachmentParams{
		ID:           id,
		WorkspaceID:  workspaceID,
		UploaderType: string(asset.UploaderType()),
		UploaderID:   uploaderID,
		Filename:     asset.Filename(),
		Url:          asset.URL(),
		ContentType:  asset.ContentType(),
		SizeBytes:    asset.SizeBytes(),
	})
	if err != nil {
		return domain.Asset{}, err
	}
	return assetFromRow(row, asset.Checksum())
}

func assetFromRow(row db.Attachment, checksum string) (domain.Asset, error) {
	if !row.CreatedAt.Valid {
		return domain.Asset{}, errors.New("attachment row has no creation time")
	}
	return domain.NewUploadedAsset(domain.UploadedAssetParams{
		ID:           util.UUIDToString(row.ID),
		WorkspaceID:  util.UUIDToString(row.WorkspaceID),
		UploaderType: domain.UploaderType(row.UploaderType),
		UploaderID:   util.UUIDToString(row.UploaderID),
		Filename:     row.Filename,
		URL:          row.Url,
		ContentType:  row.ContentType,
		SizeBytes:    row.SizeBytes,
		Checksum:     checksum,
		CreatedAt:    row.CreatedAt.Time,
	})
}
