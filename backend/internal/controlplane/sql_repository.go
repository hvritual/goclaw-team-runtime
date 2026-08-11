package controlplane

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type sqlRepository struct {
	db      *sql.DB
	dialect Dialect
}

// OpenSQLite opens and migrates the local persistence implementation.
func OpenSQLite(ctx context.Context, dsn string) (Repository, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, invalid("open sqlite", "dsn", "is required")
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	repository, err := newSQLRepository(ctx, db, DialectSQLite)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return repository, nil
}

func newSQLRepository(ctx context.Context, db *sql.DB, dialect Dialect) (Repository, error) {
	if db == nil {
		return nil, invalid("new sql repository", "db", "is required")
	}
	if dialect != DialectSQLite && dialect != DialectPostgres {
		return nil, invalid("new sql repository", "dialect", "is unsupported")
	}
	r := &sqlRepository{db: db, dialect: dialect}
	if err := r.migrate(ctx); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *sqlRepository) Close() error { return r.db.Close() }

func (r *sqlRepository) migrate(ctx context.Context) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS cp_schema_version (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS cp_workspaces (
            id TEXT PRIMARY KEY, name TEXT NOT NULL, state TEXT NOT NULL, version INTEGER NOT NULL,
            created_at TEXT NOT NULL, updated_at TEXT NOT NULL
        )`,
		`CREATE TABLE IF NOT EXISTS cp_members (
            workspace_id TEXT NOT NULL, id TEXT NOT NULL, kind TEXT NOT NULL, role TEXT NOT NULL,
            state TEXT NOT NULL, version INTEGER NOT NULL, created_at TEXT NOT NULL, updated_at TEXT NOT NULL,
            PRIMARY KEY (workspace_id, id)
        )`,
		`CREATE TABLE IF NOT EXISTS cp_records (
            workspace_id TEXT NOT NULL, kind TEXT NOT NULL, id TEXT NOT NULL, project_id TEXT NOT NULL,
            state TEXT NOT NULL, version INTEGER NOT NULL, payload TEXT NOT NULL,
            created_at TEXT NOT NULL, updated_at TEXT NOT NULL,
            PRIMARY KEY (workspace_id, kind, id)
        )`,
		`CREATE TABLE IF NOT EXISTS cp_audit (
            id TEXT PRIMARY KEY, workspace_id TEXT NOT NULL, actor_id TEXT NOT NULL, action TEXT NOT NULL,
            resource TEXT NOT NULL, resource_id TEXT NOT NULL, metadata TEXT NOT NULL, occurred_at TEXT NOT NULL
        )`,
	}
	for index, statement := range statements {
		if _, err := r.db.ExecContext(ctx, r.bind(statement)); err != nil {
			return fmt.Errorf("migrate control plane statement %d: %w", index+1, err)
		}
	}
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS cp_members_workspace_idx ON cp_members (workspace_id, state)`,
		`CREATE INDEX IF NOT EXISTS cp_records_workspace_kind_idx ON cp_records (workspace_id, kind, updated_at)`,
		`CREATE INDEX IF NOT EXISTS cp_audit_workspace_idx ON cp_audit (workspace_id, occurred_at)`,
	}
	if r.dialect == DialectPostgres {
		indexes = []string{
			`CREATE INDEX CONCURRENTLY IF NOT EXISTS cp_members_workspace_idx ON cp_members (workspace_id, state)`,
			`CREATE INDEX CONCURRENTLY IF NOT EXISTS cp_records_workspace_kind_idx ON cp_records (workspace_id, kind, updated_at)`,
			`CREATE INDEX CONCURRENTLY IF NOT EXISTS cp_audit_workspace_idx ON cp_audit (workspace_id, occurred_at)`,
		}
	}
	for index, statement := range indexes {
		if _, err := r.db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("migrate control plane index %d: %w", index+1, err)
		}
	}
	if _, err := r.db.ExecContext(ctx, r.bind(`INSERT INTO cp_schema_version (version, applied_at)
        VALUES (1, ?) ON CONFLICT (version) DO NOTHING`), time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return fmt.Errorf("record control plane schema version: %w", err)
	}
	return nil
}

func (r *sqlRepository) bind(query string) string {
	if r.dialect != DialectPostgres {
		return query
	}
	var result strings.Builder
	position := 1
	for _, character := range query {
		if character == '?' {
			fmt.Fprintf(&result, "$%d", position)
			position++
			continue
		}
		result.WriteRune(character)
	}
	return result.String()
}

func (r *sqlRepository) CreateWorkspace(ctx context.Context, workspace Workspace, owner Member, audit AuditEntry) error {
	return r.transaction(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, r.bind(`INSERT INTO cp_workspaces
            (id, name, state, version, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`),
			workspace.ID, workspace.Name, workspace.State, workspace.Version, formatTime(workspace.CreatedAt), formatTime(workspace.UpdatedAt)); err != nil {
			return conflict("create workspace", "workspace already exists")
		}
		if err := r.insertMember(ctx, tx, owner); err != nil {
			return err
		}
		return r.insertAudit(ctx, tx, audit)
	})
}

func (r *sqlRepository) UpdateWorkspace(ctx context.Context, workspace Workspace, expectedVersion int64, audit AuditEntry) error {
	return r.transaction(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, r.bind(`UPDATE cp_workspaces SET name = ?, state = ?, version = ?, updated_at = ?
            WHERE id = ? AND version = ?`), workspace.Name, workspace.State, workspace.Version,
			formatTime(workspace.UpdatedAt), workspace.ID, expectedVersion)
		if err != nil {
			return fmt.Errorf("update workspace: %w", err)
		}
		if err := requireChanged(result, "update workspace"); err != nil {
			return err
		}
		return r.insertAudit(ctx, tx, audit)
	})
}

func (r *sqlRepository) GetWorkspace(ctx context.Context, id string) (Workspace, error) {
	row := r.db.QueryRowContext(ctx, r.bind(`SELECT id, name, state, version, created_at, updated_at
        FROM cp_workspaces WHERE id = ?`), id)
	var workspace Workspace
	var created, updated string
	if err := row.Scan(&workspace.ID, &workspace.Name, &workspace.State, &workspace.Version, &created, &updated); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Workspace{}, notFound("get workspace", id)
		}
		return Workspace{}, fmt.Errorf("get workspace: %w", err)
	}
	workspace.CreatedAt, workspace.UpdatedAt = parseTime(created), parseTime(updated)
	return workspace, nil
}

func (r *sqlRepository) GetMember(ctx context.Context, workspaceID, id string) (Member, error) {
	row := r.db.QueryRowContext(ctx, r.bind(`SELECT workspace_id, id, kind, role, state, version, created_at, updated_at
        FROM cp_members WHERE workspace_id = ? AND id = ?`), workspaceID, id)
	return scanMember(row, "get member")
}

func (r *sqlRepository) ListMembers(ctx context.Context, workspaceID string, includeRemoved bool) ([]Member, error) {
	query := `SELECT workspace_id, id, kind, role, state, version, created_at, updated_at
        FROM cp_members WHERE workspace_id = ?`
	if !includeRemoved {
		query += ` AND state = 'active'`
	}
	query += ` ORDER BY created_at, id`
	rows, err := r.db.QueryContext(ctx, r.bind(query), workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	defer rows.Close()
	var members []Member
	for rows.Next() {
		member, err := scanMember(rows, "list members")
		if err != nil {
			return nil, err
		}
		members = append(members, member)
	}
	return members, rows.Err()
}

func (r *sqlRepository) SaveMember(ctx context.Context, member Member, expectedVersion int64, audit AuditEntry) error {
	return r.transaction(ctx, func(tx *sql.Tx) error {
		if err := r.lockWorkspaceMembers(ctx, tx, member.WorkspaceID); err != nil {
			return err
		}
		if expectedVersion == 0 {
			if err := r.insertMember(ctx, tx, member); err != nil {
				return err
			}
		} else {
			result, err := tx.ExecContext(ctx, r.bind(`UPDATE cp_members SET kind = ?, role = ?, state = ?, version = ?, updated_at = ?
                WHERE workspace_id = ? AND id = ? AND version = ?`), member.Kind, member.Role, member.State,
				member.Version, formatTime(member.UpdatedAt), member.WorkspaceID, member.ID, expectedVersion)
			if err != nil {
				return fmt.Errorf("save member: %w", err)
			}
			if err := requireChanged(result, "save member"); err != nil {
				return err
			}
		}
		if err := r.requireActiveHumanOwner(ctx, tx, member.WorkspaceID); err != nil {
			return err
		}
		return r.insertAudit(ctx, tx, audit)
	})
}

func (r *sqlRepository) lockWorkspaceMembers(ctx context.Context, tx *sql.Tx, workspaceID string) error {
	if r.dialect != DialectPostgres {
		return nil
	}
	if _, err := tx.ExecContext(ctx, r.bind(`SELECT pg_advisory_xact_lock(hashtextextended(?, 0))`), workspaceID); err != nil {
		return fmt.Errorf("lock workspace members: %w", err)
	}
	return nil
}

func (r *sqlRepository) requireActiveHumanOwner(ctx context.Context, tx *sql.Tx, workspaceID string) error {
	row := tx.QueryRowContext(ctx, r.bind(`SELECT COUNT(*) FROM cp_members
        WHERE workspace_id = ? AND kind = 'human' AND role = 'owner' AND state = 'active'`), workspaceID)
	var count int
	if err := row.Scan(&count); err != nil {
		return fmt.Errorf("count active owners: %w", err)
	}
	if count < 1 {
		return invariant("save member", "workspace must retain at least one active human owner")
	}
	return nil
}

func (r *sqlRepository) insertMember(ctx context.Context, tx *sql.Tx, member Member) error {
	_, err := tx.ExecContext(ctx, r.bind(`INSERT INTO cp_members
        (workspace_id, id, kind, role, state, version, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`),
		member.WorkspaceID, member.ID, member.Kind, member.Role, member.State, member.Version,
		formatTime(member.CreatedAt), formatTime(member.UpdatedAt))
	if err != nil {
		return conflict("save member", "member already exists")
	}
	return nil
}

func (r *sqlRepository) SaveRecord(ctx context.Context, record Record, expectedVersion int64, audit AuditEntry) (Record, error) {
	err := r.transaction(ctx, func(tx *sql.Tx) error {
		if expectedVersion == 0 {
			_, err := tx.ExecContext(ctx, r.bind(`INSERT INTO cp_records
                (workspace_id, kind, id, project_id, state, version, payload, created_at, updated_at)
                VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`), record.WorkspaceID, record.Kind, record.ID,
				record.ProjectID, record.State, record.Version, string(record.Payload), formatTime(record.CreatedAt), formatTime(record.UpdatedAt))
			if err != nil {
				return conflict("save record", "record already exists")
			}
		} else {
			result, err := tx.ExecContext(ctx, r.bind(`UPDATE cp_records SET project_id = ?, state = ?, version = ?, payload = ?, updated_at = ?
                WHERE workspace_id = ? AND kind = ? AND id = ? AND version = ?`), record.ProjectID, record.State,
				record.Version, string(record.Payload), formatTime(record.UpdatedAt), record.WorkspaceID, record.Kind, record.ID, expectedVersion)
			if err != nil {
				return fmt.Errorf("save record: %w", err)
			}
			if err := requireChanged(result, "save record"); err != nil {
				return err
			}
		}
		return r.insertAudit(ctx, tx, audit)
	})
	return record, err
}

func (r *sqlRepository) GetRecord(ctx context.Context, workspaceID, kind, id string) (Record, error) {
	row := r.db.QueryRowContext(ctx, r.bind(`SELECT workspace_id, kind, id, project_id, state, version, payload, created_at, updated_at
        FROM cp_records WHERE workspace_id = ? AND kind = ? AND id = ?`), workspaceID, kind, id)
	var record Record
	var payload, created, updated string
	if err := row.Scan(&record.WorkspaceID, &record.Kind, &record.ID, &record.ProjectID, &record.State,
		&record.Version, &payload, &created, &updated); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Record{}, notFound("get record", kind+"/"+id)
		}
		return Record{}, fmt.Errorf("get record: %w", err)
	}
	record.Payload = []byte(payload)
	record.CreatedAt, record.UpdatedAt = parseTime(created), parseTime(updated)
	return record, nil
}

func (r *sqlRepository) ListRecords(ctx context.Context, workspaceID, kind string, page Page) ([]Record, error) {
	page = page.normalized()
	rows, err := r.db.QueryContext(ctx, r.bind(`SELECT workspace_id, kind, id, project_id, state, version, payload, created_at, updated_at
        FROM cp_records WHERE workspace_id = ? AND kind = ? ORDER BY updated_at DESC, id LIMIT ? OFFSET ?`),
		workspaceID, kind, page.Limit, page.Offset)
	if err != nil {
		return nil, fmt.Errorf("list records: %w", err)
	}
	defer rows.Close()
	var records []Record
	for rows.Next() {
		var record Record
		var payload, created, updated string
		if err := rows.Scan(&record.WorkspaceID, &record.Kind, &record.ID, &record.ProjectID, &record.State,
			&record.Version, &payload, &created, &updated); err != nil {
			return nil, fmt.Errorf("list records: %w", err)
		}
		record.Payload = []byte(payload)
		record.CreatedAt, record.UpdatedAt = parseTime(created), parseTime(updated)
		records = append(records, record)
	}
	return records, rows.Err()
}

func (r *sqlRepository) ListAudit(ctx context.Context, workspaceID string, page Page) ([]AuditEntry, error) {
	page = page.normalized()
	rows, err := r.db.QueryContext(ctx, r.bind(`SELECT id, workspace_id, actor_id, action, resource, resource_id, metadata, occurred_at
        FROM cp_audit WHERE workspace_id = ? ORDER BY occurred_at DESC, id LIMIT ? OFFSET ?`), workspaceID, page.Limit, page.Offset)
	if err != nil {
		return nil, fmt.Errorf("list audit: %w", err)
	}
	defer rows.Close()
	var entries []AuditEntry
	for rows.Next() {
		var entry AuditEntry
		var metadata, occurred string
		if err := rows.Scan(&entry.ID, &entry.WorkspaceID, &entry.ActorID, &entry.Action, &entry.Resource,
			&entry.ResourceID, &metadata, &occurred); err != nil {
			return nil, fmt.Errorf("list audit: %w", err)
		}
		entry.Metadata, entry.OccurredAt = []byte(metadata), parseTime(occurred)
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func (r *sqlRepository) insertAudit(ctx context.Context, tx *sql.Tx, audit AuditEntry) error {
	_, err := tx.ExecContext(ctx, r.bind(`INSERT INTO cp_audit
        (id, workspace_id, actor_id, action, resource, resource_id, metadata, occurred_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?)`), audit.ID, audit.WorkspaceID, audit.ActorID, audit.Action,
		audit.Resource, audit.ResourceID, string(audit.Metadata), formatTime(audit.OccurredAt))
	if err != nil {
		return fmt.Errorf("append audit: %w", err)
	}
	return nil
}

func (r *sqlRepository) transaction(ctx context.Context, operation func(*sql.Tx) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	if err := operation(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

type scanner interface{ Scan(...any) error }

func scanMember(row scanner, operation string) (Member, error) {
	var member Member
	var created, updated string
	if err := row.Scan(&member.WorkspaceID, &member.ID, &member.Kind, &member.Role, &member.State,
		&member.Version, &created, &updated); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Member{}, notFound(operation, "member")
		}
		return Member{}, fmt.Errorf("%s: %w", operation, err)
	}
	member.CreatedAt, member.UpdatedAt = parseTime(created), parseTime(updated)
	return member, nil
}

func requireChanged(result sql.Result, operation string) error {
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s rows affected: %w", operation, err)
	}
	if count != 1 {
		return conflict(operation, "version changed or resource does not exist")
	}
	return nil
}

func formatTime(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }

func parseTime(value string) time.Time {
	parsed, _ := time.Parse(time.RFC3339Nano, value)
	return parsed
}
