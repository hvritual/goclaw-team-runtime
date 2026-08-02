// Package sqlite implements Space persistence with provider-native SQLite transactions.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/multica-ai/multica/server/internal/modules/space/contract"
	"github.com/multica-ai/multica/server/internal/modules/space/internal/application"
	"github.com/multica-ai/multica/server/internal/modules/space/internal/domain/asset"
)

type Config struct {
	DB              *sql.DB
	WorkspaceAccess contract.WorkspaceAccess
	Objects         contract.ObjectStore
	NewID           func() (string, error)
	Now             func() time.Time
}

func New(Config) contract.Service { return application.New() }

func NewAsset(config Config) (contract.AssetUploadService, error) {
	if config.DB == nil {
		return nil, errors.New("space sqlite database is required")
	}
	options := []application.AssetServiceOption{
		application.WithAssetRepository(&assetRepository{db: config.DB}),
		application.WithWorkspaceAccess(config.WorkspaceAccess),
		application.WithObjectStore(config.Objects),
	}
	if config.NewID != nil {
		options = append(options, application.WithAssetIDGenerator(config.NewID))
	}
	if config.Now != nil {
		options = append(options, application.WithAssetClock(config.Now))
	}
	return application.NewAssetService(options...), nil
}

type assetRepository struct {
	db *sql.DB
}

func (r *assetRepository) BeginUpload(ctx context.Context, intent asset.UploadIntent) error {
	timestamp := intent.CreatedAt.UTC().Format(time.RFC3339Nano)
	_, err := r.db.ExecContext(ctx, `INSERT INTO space_upload_intents(
		id, asset_id, version_id, workspace_id, uploader_type, uploader_id,
		filename, object_key, media_type, size_bytes, checksum, state,
		created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending', ?, ?)`,
		intent.ID, intent.AssetID, intent.VersionID, intent.WorkspaceID,
		intent.UploaderType, intent.UploaderID, intent.Filename, intent.ObjectKey,
		intent.MediaType, intent.SizeBytes, intent.Checksum, timestamp, timestamp,
	)
	if err != nil {
		return fmt.Errorf("insert upload intent: %w", err)
	}
	return nil
}

func (r *assetRepository) FinalizeUpload(
	ctx context.Context,
	intent asset.UploadIntent,
	rawURL string,
) (asset.Asset, error) {
	value, err := asset.Finalize(intent, rawURL)
	if err != nil {
		return asset.Asset{}, err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return asset.Asset{}, fmt.Errorf("begin asset finalization: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	timestamp := value.CreatedAt().Format(time.RFC3339Nano)
	if _, err := tx.ExecContext(ctx, `INSERT INTO space_assets(
		id, workspace_id, current_version_id, uploader_type, uploader_id, filename, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		value.ID(), value.WorkspaceID(), value.CurrentVersionID(), value.UploaderType(),
		value.UploaderID(), value.Filename(), timestamp,
	); err != nil {
		return asset.Asset{}, fmt.Errorf("insert asset: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO space_asset_versions(
		id, asset_id, object_key, object_url, media_type, size_bytes, checksum, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		value.CurrentVersionID(), value.ID(), value.ObjectKey(), value.URL(),
		value.MediaType(), value.SizeBytes(), value.Checksum(), timestamp,
	); err != nil {
		return asset.Asset{}, fmt.Errorf("insert asset version: %w", err)
	}
	result, err := tx.ExecContext(ctx, `UPDATE space_upload_intents
		SET state = 'finalized', last_error = '', updated_at = ?
		WHERE id = ? AND state IN ('pending', 'cleanup_pending')`, timestamp, intent.ID)
	if err != nil {
		return asset.Asset{}, fmt.Errorf("finalize upload intent: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return asset.Asset{}, fmt.Errorf("read finalized intent count: %w", err)
	}
	if affected != 1 {
		return asset.Asset{}, errors.New("upload intent is not pending")
	}
	if err := tx.Commit(); err != nil {
		return asset.Asset{}, fmt.Errorf("commit asset finalization: %w", err)
	}
	return value, nil
}

func (r *assetRepository) FindByID(ctx context.Context, assetID string) (asset.Asset, error) {
	var value asset.RehydrateParams
	var createdAt string
	err := r.db.QueryRowContext(ctx, `SELECT
		a.id, a.workspace_id, a.current_version_id, a.uploader_type, a.uploader_id,
		a.filename, v.object_key, v.object_url, v.media_type, v.size_bytes,
		v.checksum, a.created_at
	FROM space_assets a
	JOIN space_asset_versions v ON v.id = a.current_version_id
	WHERE a.id = ?`, assetID).Scan(
		&value.ID, &value.WorkspaceID, &value.CurrentVersionID, &value.UploaderType,
		&value.UploaderID, &value.Filename, &value.ObjectKey, &value.URL,
		&value.MediaType, &value.SizeBytes, &value.Checksum, &createdAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return asset.Asset{}, application.ErrAssetRecordNotFound
	}
	if err != nil {
		return asset.Asset{}, fmt.Errorf("find asset: %w", err)
	}
	value.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return asset.Asset{}, fmt.Errorf("parse asset created_at: %w", err)
	}
	return asset.Rehydrate(value)
}

func (r *assetRepository) ListPendingUploads(ctx context.Context) ([]asset.UploadIntent, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT
		id, asset_id, version_id, workspace_id, uploader_type, uploader_id,
		filename, object_key, media_type, size_bytes, checksum, created_at
	FROM space_upload_intents
	WHERE state IN ('pending', 'cleanup_pending')
	ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("list pending uploads: %w", err)
	}
	defer func() { _ = rows.Close() }()
	values := make([]asset.UploadIntent, 0)
	for rows.Next() {
		var value asset.UploadIntent
		var createdAt string
		if err := rows.Scan(
			&value.ID, &value.AssetID, &value.VersionID, &value.WorkspaceID,
			&value.UploaderType, &value.UploaderID, &value.Filename, &value.ObjectKey,
			&value.MediaType, &value.SizeBytes, &value.Checksum, &createdAt,
		); err != nil {
			return nil, fmt.Errorf("scan pending upload: %w", err)
		}
		value.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			return nil, fmt.Errorf("parse pending upload created_at: %w", err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pending uploads: %w", err)
	}
	return values, nil
}

func (r *assetRepository) MarkCleanupPending(ctx context.Context, intentID, reason string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE space_upload_intents
		SET state = 'cleanup_pending', last_error = ?, updated_at = ?
		WHERE id = ? AND state != 'finalized'`,
		reason, time.Now().UTC().Format(time.RFC3339Nano), intentID,
	)
	if err != nil {
		return fmt.Errorf("mark cleanup pending: %w", err)
	}
	return nil
}

func (r *assetRepository) MarkCleaned(ctx context.Context, intentID string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE space_upload_intents
		SET state = 'cleaned', updated_at = ?
		WHERE id = ? AND state = 'cleanup_pending'`,
		time.Now().UTC().Format(time.RFC3339Nano), intentID,
	)
	if err != nil {
		return fmt.Errorf("mark upload cleaned: %w", err)
	}
	return nil
}
