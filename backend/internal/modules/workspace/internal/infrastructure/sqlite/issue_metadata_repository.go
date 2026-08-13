package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
	issueDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/issue"
)

type issueMetadataRepository struct{ db *sql.DB }

func NewIssueMetadataRepository(config Config) (application.IssueMetadataRepository, error) {
	if config.DB == nil {
		return nil, errors.New("workspace sqlite database is required")
	}
	return &issueMetadataRepository{db: config.DB}, nil
}

func (r *issueMetadataRepository) GetMetadata(ctx context.Context, workspaceID, issueID string) (string, map[string]any, time.Time, error) {
	var id, raw, updated string
	err := r.db.QueryRowContext(ctx, `SELECT id, metadata, updated_at FROM workspace_issues WHERE workspace_id=? AND (id=? OR identifier=?)`, workspaceID, issueID, issueID).Scan(&id, &raw, &updated)
	return decodeMetadataRecord(id, raw, updated, err)
}

func (r *issueMetadataRepository) PutMetadata(ctx context.Context, workspaceID, issueID, key string, value any, now time.Time) (string, map[string]any, time.Time, error) {
	return r.mutate(ctx, workspaceID, issueID, now, func(bag *issueDomain.MetadataBag) error { _, err := bag.Put(key, value); return err })
}

func (r *issueMetadataRepository) DeleteMetadata(ctx context.Context, workspaceID, issueID, key string, now time.Time) (string, map[string]any, time.Time, error) {
	return r.mutate(ctx, workspaceID, issueID, now, func(bag *issueDomain.MetadataBag) error { _, err := bag.Delete(key); return err })
}

func (r *issueMetadataRepository) mutate(ctx context.Context, workspaceID, issueID string, now time.Time, change func(*issueDomain.MetadataBag) error) (id string, values map[string]any, updated time.Time, err error) {
	connection, err := r.db.Conn(ctx)
	if err != nil {
		return "", nil, time.Time{}, fmt.Errorf("acquire Issue metadata connection: %w", err)
	}
	defer connection.Close()
	if _, err = connection.ExecContext(ctx, `PRAGMA busy_timeout = 5000`); err != nil {
		return "", nil, time.Time{}, err
	}
	if _, err = connection.ExecContext(ctx, `BEGIN IMMEDIATE`); err != nil {
		return "", nil, time.Time{}, fmt.Errorf("begin immediate Issue metadata mutation: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = connection.ExecContext(context.WithoutCancel(ctx), `ROLLBACK`)
		}
	}()
	var raw, currentUpdated string
	err = connection.QueryRowContext(ctx, `SELECT id, metadata, updated_at FROM workspace_issues WHERE workspace_id=? AND (id=? OR identifier=?)`, workspaceID, issueID, issueID).Scan(&id, &raw, &currentUpdated)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil, time.Time{}, application.ErrIssueRecordNotFound
	}
	if err != nil {
		return "", nil, time.Time{}, fmt.Errorf("load Issue metadata: %w", err)
	}
	values = decodeMetadata(raw)
	bag := issueDomain.NewMetadataBag(values)
	if err = change(&bag); err != nil {
		return "", nil, time.Time{}, err
	}
	values = bag.Snapshot()
	encoded, err := json.Marshal(values)
	if err != nil {
		return "", nil, time.Time{}, fmt.Errorf("encode Issue metadata: %w", err)
	}
	updated = now.UTC()
	result, err := connection.ExecContext(ctx, `UPDATE workspace_issues SET metadata=?, updated_at=? WHERE workspace_id=? AND id=?`, string(encoded), updated.Format(time.RFC3339Nano), workspaceID, id)
	if err != nil {
		return "", nil, time.Time{}, fmt.Errorf("update Issue metadata: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil || rows != 1 {
		return "", nil, time.Time{}, application.ErrIssueRecordNotFound
	}
	if _, err = connection.ExecContext(ctx, `COMMIT`); err != nil {
		return "", nil, time.Time{}, fmt.Errorf("commit Issue metadata: %w", err)
	}
	committed = true
	return id, values, updated, nil
}

func decodeMetadataRecord(id, raw, updated string, err error) (string, map[string]any, time.Time, error) {
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil, time.Time{}, application.ErrIssueRecordNotFound
	}
	if err != nil {
		return "", nil, time.Time{}, fmt.Errorf("get Issue metadata: %w", err)
	}
	timestamp, err := time.Parse(time.RFC3339Nano, updated)
	if err != nil {
		return "", nil, time.Time{}, fmt.Errorf("parse Issue metadata timestamp: %w", err)
	}
	return id, decodeMetadata(raw), timestamp, nil
}

func decodeMetadata(raw string) map[string]any {
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	var values map[string]any
	if err := decoder.Decode(&values); err != nil || values == nil {
		return map[string]any{}
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return map[string]any{}
	}
	for _, value := range values {
		encoded, _ := json.Marshal(value)
		if _, err := issueDomain.ParseMetadataValueJSON(string(encoded)); err != nil {
			return map[string]any{}
		}
	}
	return values
}
