package workspace

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
)

const taskCursorSigningKeyBytes = 32

func loadOrCreateTaskCursorSigningKey(ctx context.Context, db *sql.DB) ([]byte, error) {
	if db == nil {
		return nil, errors.New("workspace sqlite database is required")
	}
	candidate := make([]byte, taskCursorSigningKeyBytes)
	if _, err := rand.Read(candidate); err != nil {
		return nil, fmt.Errorf("generate Task cursor signing key: %w", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT OR IGNORE INTO workspace_runtime_secrets(name,value) VALUES('task_cursor_hmac_v1',?)`, candidate); err != nil {
		return nil, fmt.Errorf("persist Task cursor signing key: %w", err)
	}
	var key []byte
	if err := db.QueryRowContext(ctx, `SELECT value FROM workspace_runtime_secrets WHERE name='task_cursor_hmac_v1'`).Scan(&key); err != nil {
		return nil, fmt.Errorf("load Task cursor signing key: %w", err)
	}
	if len(key) != taskCursorSigningKeyBytes {
		return nil, fmt.Errorf("Task cursor signing key has invalid length %d", len(key))
	}
	return append([]byte(nil), key...), nil
}
