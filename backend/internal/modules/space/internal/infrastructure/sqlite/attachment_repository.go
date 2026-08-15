package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/hvritual/workspace/internal/modules/space/contract"
	"github.com/hvritual/workspace/internal/modules/space/internal/application"
	sqlite3 "modernc.org/sqlite/lib"
)

type AttachmentRepository struct{ db *sql.DB }

func NewAttachmentRepository(db *sql.DB) (*AttachmentRepository, error) {
	if db == nil {
		return nil, errors.New("space sqlite database is required")
	}
	return &AttachmentRepository{db: db}, nil
}

func (r *AttachmentRepository) Create(ctx context.Context, value application.StoredAttachment, bind func(contract.AttachmentExecutor) error) (err error) {
	connection, err := r.writeConnection(ctx, "attachment creation")
	if err != nil {
		return err
	}
	defer connection.Close()
	committed := false
	defer rollback(ctx, connection, &committed)
	timestamp := value.CreatedAt.UTC().Format(time.RFC3339Nano)
	if _, err := connection.ExecContext(ctx, `INSERT INTO space_assets(id,workspace_id,current_version_id,uploader_type,uploader_id,filename,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?)`, value.ID, value.WorkspaceID, value.VersionID, value.UploaderType, value.UploaderID, value.Filename, timestamp, timestamp); err != nil {
		return fmt.Errorf("insert Space attachment: %w", err)
	}
	if _, err := connection.ExecContext(ctx, `INSERT INTO space_asset_versions(id,asset_id,object_key,media_type,size_bytes,checksum,created_at) VALUES(?,?,?,?,?,?,?)`, value.VersionID, value.ID, value.ObjectKey, value.ContentType, value.SizeBytes, value.Checksum, timestamp); err != nil {
		return fmt.Errorf("insert Space attachment version: %w", err)
	}
	if bind != nil {
		if err := bind(connection); err != nil {
			return fmt.Errorf("bind Workspace attachment relation: %w", err)
		}
	}
	if _, err := connection.ExecContext(ctx, `COMMIT`); err != nil {
		return fmt.Errorf("commit Space attachment creation: %w", err)
	}
	committed = true
	return nil
}

func (r *AttachmentRepository) FindByID(ctx context.Context, id string) (application.StoredAttachment, error) {
	return scanAttachment(r.db.QueryRowContext(ctx, attachmentSelect+` WHERE asset.id=?`, id))
}

func (r *AttachmentRepository) FindMany(ctx context.Context, ids []string) ([]application.StoredAttachment, error) {
	values := make([]application.StoredAttachment, 0, len(ids))
	for _, id := range ids {
		value, err := r.FindByID(ctx, id)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func (r *AttachmentRepository) Delete(ctx context.Context, value application.StoredAttachment, unbind func(contract.AttachmentExecutor) error) (err error) {
	connection, err := r.writeConnection(ctx, "attachment deletion")
	if err != nil {
		return err
	}
	defer connection.Close()
	committed := false
	defer rollback(ctx, connection, &committed)
	if unbind != nil {
		if err := unbind(connection); err != nil {
			return fmt.Errorf("unbind Workspace attachment relation: %w", err)
		}
	}
	result, err := connection.ExecContext(ctx, `DELETE FROM space_asset_versions WHERE asset_id=?`, value.ID)
	if err != nil {
		return fmt.Errorf("delete Space attachment versions: %w", err)
	}
	if rows, rowsErr := result.RowsAffected(); rowsErr != nil || rows < 1 {
		if rowsErr != nil {
			return rowsErr
		}
		return contract.ErrAttachmentNotFound
	}
	result, err = connection.ExecContext(ctx, `DELETE FROM space_assets WHERE workspace_id=? AND id=?`, value.WorkspaceID, value.ID)
	if err != nil {
		return fmt.Errorf("delete Space attachment: %w", err)
	}
	if rows, rowsErr := result.RowsAffected(); rowsErr != nil || rows != 1 {
		if rowsErr != nil {
			return rowsErr
		}
		return contract.ErrAttachmentNotFound
	}
	if _, err := connection.ExecContext(ctx, `COMMIT`); err != nil {
		return fmt.Errorf("commit Space attachment deletion: %w", err)
	}
	committed = true
	return nil
}

func (r *AttachmentRepository) ObjectKeys(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT object_key FROM space_asset_versions ORDER BY object_key`)
	if err != nil {
		return nil, fmt.Errorf("list Space attachment object keys: %w", err)
	}
	defer rows.Close()
	result := []string{}
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		result = append(result, key)
	}
	return result, rows.Err()
}

func (r *AttachmentRepository) FindForCleanup(ctx context.Context, executor contract.AttachmentExecutor, workspaceID string, ids []string) ([]application.StoredAttachment, error) {
	values := make([]application.StoredAttachment, 0, len(ids))
	for _, id := range ids {
		value, err := scanAttachment(executor.QueryRowContext(ctx, attachmentSelect+` WHERE asset.workspace_id=? AND asset.id=?`, workspaceID, id))
		if errors.Is(err, contract.ErrAttachmentNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func (r *AttachmentRepository) DeleteForCleanup(ctx context.Context, executor contract.AttachmentExecutor, workspaceID string, values []application.StoredAttachment) error {
	for _, value := range values {
		result, err := executor.ExecContext(ctx, `DELETE FROM space_asset_versions WHERE asset_id=?`, value.ID)
		if err != nil {
			return fmt.Errorf("delete dependent Space attachment versions: %w", err)
		}
		if rows, rowsErr := result.RowsAffected(); rowsErr != nil || rows != 1 {
			if rowsErr != nil {
				return rowsErr
			}
			return fmt.Errorf("delete dependent Space attachment versions: expected 1 row, got %d", rows)
		}
		result, err = executor.ExecContext(ctx, `DELETE FROM space_assets WHERE workspace_id=? AND id=?`, workspaceID, value.ID)
		if err != nil {
			return fmt.Errorf("delete dependent Space attachment: %w", err)
		}
		if rows, rowsErr := result.RowsAffected(); rowsErr != nil || rows != 1 {
			if rowsErr != nil {
				return rowsErr
			}
			return fmt.Errorf("delete dependent Space attachment: expected 1 row, got %d", rows)
		}
	}
	return nil
}

const attachmentSelect = `SELECT asset.id,asset.workspace_id,asset.current_version_id,asset.uploader_type,asset.uploader_id,asset.filename,version.object_key,version.media_type,version.size_bytes,version.checksum,asset.created_at FROM space_assets asset JOIN space_asset_versions version ON version.id=asset.current_version_id`

type rowScanner interface{ Scan(...any) error }

func scanAttachment(row rowScanner) (application.StoredAttachment, error) {
	var value application.StoredAttachment
	var createdAt string
	if err := row.Scan(&value.ID, &value.WorkspaceID, &value.VersionID, &value.UploaderType, &value.UploaderID, &value.Filename, &value.ObjectKey, &value.ContentType, &value.SizeBytes, &value.Checksum, &createdAt); errors.Is(err, sql.ErrNoRows) {
		return application.StoredAttachment{}, contract.ErrAttachmentNotFound
	} else if err != nil {
		return application.StoredAttachment{}, fmt.Errorf("read Space attachment: %w", err)
	}
	parsed, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return application.StoredAttachment{}, fmt.Errorf("parse Space attachment timestamp: %w", err)
	}
	value.CreatedAt = parsed
	return value, nil
}

const (
	attachmentWriteAcquireBudget  = 8 * time.Second
	attachmentWriteAcquireRetries = 2
)

func (r *AttachmentRepository) writeConnection(ctx context.Context, label string) (*sql.Conn, error) {
	acquireContext, cancel := context.WithTimeout(ctx, attachmentWriteAcquireBudget)
	defer cancel()
	for attempt := 0; ; attempt++ {
		connection, err := r.db.Conn(acquireContext)
		if err != nil {
			return nil, fmt.Errorf("acquire %s connection: %w", label, err)
		}
		if _, err := connection.ExecContext(acquireContext, `PRAGMA busy_timeout = 5000`); err != nil {
			connection.Close()
			return nil, fmt.Errorf("configure %s lock wait: %w", label, err)
		}
		if _, err := connection.ExecContext(acquireContext, `BEGIN IMMEDIATE`); err == nil {
			return connection, nil
		} else {
			connection.Close()
			if !isSQLiteWriteContention(err) || attempt >= attachmentWriteAcquireRetries || acquireContext.Err() != nil {
				return nil, fmt.Errorf("begin %s: %w", label, err)
			}
			if err := waitForAttachmentWriteRetry(acquireContext, attempt); err != nil {
				return nil, fmt.Errorf("begin %s: %w", label, err)
			}
		}
	}
}

func isSQLiteWriteContention(err error) bool {
	var coded interface{ Code() int }
	if !errors.As(err, &coded) {
		return false
	}
	code := coded.Code() & 0xff
	return code == sqlite3.SQLITE_BUSY || code == sqlite3.SQLITE_LOCKED
}

func waitForAttachmentWriteRetry(ctx context.Context, attempt int) error {
	delay := 20 * time.Millisecond * time.Duration(attempt+1)
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func rollback(ctx context.Context, connection *sql.Conn, committed *bool) {
	if !*committed {
		_, _ = connection.ExecContext(context.WithoutCancel(ctx), `ROLLBACK`)
	}
}
