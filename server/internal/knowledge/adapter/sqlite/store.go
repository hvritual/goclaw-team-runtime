package sqlite

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/multica-ai/multica/server/internal/knowledge"
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schema string

const SchemaVersion = 1

type migration struct {
	version int
	sql     string
}

var migrations = []migration{
	{version: 1, sql: schema},
}

type Store struct {
	db          *sql.DB
	path        string
	fts5Enabled bool
}

type Capabilities struct {
	SchemaVersion int
	JournalMode   string
	ForeignKeys   bool
	FTS5          bool
}

func Open(path string) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("knowledge sqlite path is required")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve knowledge sqlite path: %w", err)
	}
	directory := filepath.Dir(absolute)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return nil, fmt.Errorf("create knowledge sqlite directory: %w", err)
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		return nil, fmt.Errorf("protect knowledge sqlite directory: %w", err)
	}

	db, err := sql.Open("sqlite", absolute)
	if err != nil {
		return nil, fmt.Errorf("open knowledge sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	store := &Store{db: db, path: absolute}
	if err := store.initialize(); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := os.Chmod(absolute, 0o600); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("protect knowledge sqlite file: %w", err)
	}
	store.fts5Enabled = store.detectFTS5()
	if store.fts5Enabled {
		if err := store.ensureFTS(context.Background()); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	return store, nil
}

func (s *Store) initialize() error {
	for _, statement := range []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA busy_timeout = 5000",
		"PRAGMA foreign_keys = OFF",
	} {
		if _, err := s.db.Exec(statement); err != nil {
			return fmt.Errorf("configure knowledge sqlite: %w", err)
		}
	}
	if _, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS knowledge_schema_version (
			version INTEGER PRIMARY KEY,
			applied_at TEXT NOT NULL
		)`); err != nil {
		return fmt.Errorf("initialize knowledge schema version table: %w", err)
	}
	var version int
	if err := s.db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM knowledge_schema_version`).Scan(&version); err != nil {
		return fmt.Errorf("read knowledge sqlite schema version: %w", err)
	}
	if version > SchemaVersion {
		return fmt.Errorf("knowledge sqlite schema version %d is newer than supported version %d", version, SchemaVersion)
	}
	for _, migration := range migrations {
		if migration.version <= version {
			continue
		}
		if err := s.applyMigration(migration); err != nil {
			return err
		}
		version = migration.version
	}
	if version != SchemaVersion {
		return fmt.Errorf("knowledge sqlite schema version %d is unsupported, want %d", version, SchemaVersion)
	}
	return nil
}

func (s *Store) applyMigration(migration migration) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin knowledge migration %d: %w", migration.version, err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(migration.sql); err != nil {
		return fmt.Errorf("apply knowledge migration %d: %w", migration.version, err)
	}
	if _, err := tx.Exec(`
		INSERT INTO knowledge_schema_version(version, applied_at)
		VALUES (?, ?)`,
		migration.version,
		formatTime(time.Now().UTC()),
	); err != nil {
		return fmt.Errorf("record knowledge migration %d: %w", migration.version, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit knowledge migration %d: %w", migration.version, err)
	}
	return nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Capabilities(ctx context.Context) (Capabilities, error) {
	var result Capabilities
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) FROM knowledge_schema_version`).
		Scan(&result.SchemaVersion); err != nil {
		return Capabilities{}, fmt.Errorf("read knowledge schema version: %w", err)
	}
	if err := s.db.QueryRowContext(ctx, `PRAGMA journal_mode`).Scan(&result.JournalMode); err != nil {
		return Capabilities{}, fmt.Errorf("read knowledge journal mode: %w", err)
	}
	var foreignKeys int
	if err := s.db.QueryRowContext(ctx, `PRAGMA foreign_keys`).Scan(&foreignKeys); err != nil {
		return Capabilities{}, fmt.Errorf("read knowledge foreign key mode: %w", err)
	}
	result.ForeignKeys = foreignKeys != 0
	result.FTS5 = s.fts5Enabled
	return result, nil
}

func (s *Store) detectFTS5() bool {
	if _, err := s.db.Exec(`CREATE VIRTUAL TABLE temp.knowledge_fts_probe USING fts5(content)`); err != nil {
		return false
	}
	_, _ = s.db.Exec(`DROP TABLE temp.knowledge_fts_probe`)
	return true
}

func (s *Store) ensureFTS(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `
		CREATE VIRTUAL TABLE IF NOT EXISTS knowledge_search_fts USING fts5(
			entry_id UNINDEXED,
			workspace_id UNINDEXED,
			project_id UNINDEXED,
			kind UNINDEXED,
			title,
			content,
			tokenize = 'unicode61'
		)`); err != nil {
		return fmt.Errorf("initialize knowledge search index: %w", err)
	}
	return s.Rebuild(ctx)
}

func (s *Store) Rebuild(ctx context.Context) error {
	if !s.fts5Enabled {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin knowledge search rebuild: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM knowledge_search_fts`); err != nil {
		return fmt.Errorf("clear knowledge search index: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO knowledge_search_fts(
			entry_id, workspace_id, project_id, kind, title, content
		)
		SELECT e.id, e.workspace_id, e.project_id, e.kind, r.title, r.content
		FROM knowledge_entry e
		JOIN knowledge_revision r
			ON r.entry_id = e.id AND r.revision = e.current_revision
		WHERE e.status = ?`,
		knowledge.StatusPublished,
	); err != nil {
		return fmt.Errorf("populate knowledge search index: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit knowledge search rebuild: %w", err)
	}
	return nil
}

func (s *Store) Backup(ctx context.Context, destination string) error {
	if strings.TrimSpace(destination) == "" {
		return errors.New("knowledge backup path is required")
	}
	absolute, err := filepath.Abs(destination)
	if err != nil {
		return fmt.Errorf("resolve knowledge backup path: %w", err)
	}
	if absolute == s.path {
		return errors.New("knowledge backup path must differ from the active database")
	}
	if _, err := os.Stat(absolute); err == nil {
		return errors.New("knowledge backup destination already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect knowledge backup destination: %w", err)
	}
	directory := filepath.Dir(absolute)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create knowledge backup directory: %w", err)
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		return fmt.Errorf("protect knowledge backup directory: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `PRAGMA wal_checkpoint(FULL)`); err != nil {
		return fmt.Errorf("checkpoint knowledge sqlite before backup: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `VACUUM INTO ?`, absolute); err != nil {
		_ = os.Remove(absolute)
		return fmt.Errorf("backup knowledge sqlite: %w", err)
	}
	if err := os.Chmod(absolute, 0o600); err != nil {
		return fmt.Errorf("protect knowledge backup file: %w", err)
	}
	return nil
}

func (s *Store) CreateCandidate(ctx context.Context, candidate knowledge.Candidate) (knowledge.Candidate, error) {
	currentTime := time.Now().UTC()
	if candidate.ID == "" {
		candidate.ID = uuid.NewString()
	}
	if candidate.CreatedAt.IsZero() {
		candidate.CreatedAt = currentTime
	}
	if candidate.UpdatedAt.IsZero() {
		candidate.UpdatedAt = currentTime
	}
	sourceRefs, err := json.Marshal(candidate.SourceRefs)
	if err != nil {
		return knowledge.Candidate{}, fmt.Errorf("encode candidate sources: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO knowledge_candidate(
			id, workspace_id, project_id, kind, title, content, reason, status,
			revision, proposed_by, source_refs_json, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		candidate.ID,
		candidate.WorkspaceID,
		candidate.ProjectID,
		candidate.Kind,
		candidate.Title,
		candidate.Content,
		candidate.Reason,
		candidate.Status,
		candidate.Revision,
		candidate.ProposedBy,
		string(sourceRefs),
		formatTime(candidate.CreatedAt),
		formatTime(candidate.UpdatedAt),
	)
	if err != nil {
		return knowledge.Candidate{}, fmt.Errorf("create knowledge candidate: %w", err)
	}
	return candidate, nil
}

func (s *Store) GetCandidate(ctx context.Context, id string) (knowledge.Candidate, error) {
	return scanCandidate(s.db.QueryRowContext(ctx, `
		SELECT id, workspace_id, project_id, kind, title, content, reason, status,
		       revision, proposed_by, source_refs_json, created_at, updated_at
		FROM knowledge_candidate WHERE id = ?`, id))
}

func (s *Store) GetEntry(ctx context.Context, workspaceID, id string) (knowledge.Entry, error) {
	entry, err := scanEntry(s.db.QueryRowContext(ctx, `
		SELECT id, workspace_id, project_id, candidate_id, kind, status,
		       current_revision, created_at, updated_at
		FROM knowledge_entry
		WHERE workspace_id = ? AND id = ?`, workspaceID, id))
	if err != nil {
		return knowledge.Entry{}, err
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT revision, title, content, created_by, created_at, source_refs_json
		FROM knowledge_revision
		WHERE entry_id = ?
		ORDER BY revision`, entry.ID)
	if err != nil {
		return knowledge.Entry{}, fmt.Errorf("list knowledge revisions: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		revision, err := scanRevision(rows)
		if err != nil {
			return knowledge.Entry{}, err
		}
		entry.Revisions = append(entry.Revisions, revision)
	}
	if err := rows.Err(); err != nil {
		return knowledge.Entry{}, fmt.Errorf("list knowledge revisions: %w", err)
	}
	return entry, nil
}

func (s *Store) ListCandidates(
	ctx context.Context,
	query knowledge.CandidateQuery,
) (knowledge.CandidatePage, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset, _ := strconv.Atoi(query.Cursor)
	where := []string{"workspace_id = ?"}
	arguments := []any{query.WorkspaceID}
	if query.ProjectID != "" {
		where = append(where, "project_id = ?")
		arguments = append(arguments, query.ProjectID)
	}
	if len(query.Statuses) > 0 {
		placeholders := make([]string, 0, len(query.Statuses))
		for _, status := range query.Statuses {
			placeholders = append(placeholders, "?")
			arguments = append(arguments, status)
		}
		where = append(where, "status IN ("+strings.Join(placeholders, ",")+")")
	}
	if len(query.Kinds) > 0 {
		placeholders := make([]string, 0, len(query.Kinds))
		for _, kind := range query.Kinds {
			placeholders = append(placeholders, "?")
			arguments = append(arguments, kind)
		}
		where = append(where, "kind IN ("+strings.Join(placeholders, ",")+")")
	}
	arguments = append(arguments, limit+1, offset)
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, workspace_id, project_id, kind, title, content, reason, status,
		       revision, proposed_by, source_refs_json, created_at, updated_at
		FROM knowledge_candidate
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY updated_at DESC, id
		LIMIT ? OFFSET ?`, arguments...)
	if err != nil {
		return knowledge.CandidatePage{}, fmt.Errorf("list knowledge candidates: %w", err)
	}
	defer rows.Close()
	candidates := make([]knowledge.Candidate, 0, limit+1)
	for rows.Next() {
		candidate, err := scanCandidate(rows)
		if err != nil {
			return knowledge.CandidatePage{}, err
		}
		candidates = append(candidates, candidate)
	}
	if err := rows.Err(); err != nil {
		return knowledge.CandidatePage{}, fmt.Errorf("list knowledge candidates: %w", err)
	}
	nextCursor := ""
	if len(candidates) > limit {
		candidates = candidates[:limit]
		nextCursor = strconv.Itoa(offset + limit)
	}
	return knowledge.CandidatePage{Candidates: candidates, NextCursor: nextCursor}, nil
}

func (s *Store) ReviewCandidate(
	ctx context.Context,
	command knowledge.ReviewCommand,
) (knowledge.Candidate, *knowledge.Entry, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return knowledge.Candidate{}, nil, fmt.Errorf("begin knowledge review: %w", err)
	}
	defer tx.Rollback()

	candidate, err := scanCandidate(tx.QueryRowContext(ctx, `
		SELECT id, workspace_id, project_id, kind, title, content, reason, status,
		       revision, proposed_by, source_refs_json, created_at, updated_at
		FROM knowledge_candidate WHERE id = ?`, command.CandidateID))
	if err != nil {
		return knowledge.Candidate{}, nil, err
	}
	if candidate.WorkspaceID != command.WorkspaceID {
		return knowledge.Candidate{}, nil, knowledge.ErrWorkspaceMismatch
	}
	if candidate.Revision != command.ExpectedRevision {
		return knowledge.Candidate{}, nil, knowledge.ErrRevisionConflict
	}
	if candidate.Status != knowledge.StatusCandidate && candidate.Status != knowledge.StatusInReview {
		return knowledge.Candidate{}, nil, errors.New("knowledge candidate is already reviewed")
	}
	result, err := tx.ExecContext(ctx, `
		UPDATE knowledge_candidate
		SET status = ?, revision = revision + 1, updated_at = ?
		WHERE id = ? AND workspace_id = ? AND revision = ?`,
		command.NewStatus,
		formatTime(command.Review.ReviewedAt),
		command.CandidateID,
		command.WorkspaceID,
		command.ExpectedRevision,
	)
	if err != nil {
		return knowledge.Candidate{}, nil, fmt.Errorf("update knowledge candidate: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return knowledge.Candidate{}, nil, fmt.Errorf("inspect knowledge review update: %w", err)
	}
	if affected != 1 {
		return knowledge.Candidate{}, nil, knowledge.ErrRevisionConflict
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO knowledge_review(
			id, candidate_id, workspace_id, action, reviewer_id, rationale,
			reviewed_at, old_revision, new_revision
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		uuid.NewString(),
		command.CandidateID,
		command.WorkspaceID,
		command.Review.Action,
		command.Review.ReviewerID,
		command.Review.Rationale,
		formatTime(command.Review.ReviewedAt),
		command.Review.OldRevision,
		command.Review.NewRevision,
	); err != nil {
		return knowledge.Candidate{}, nil, fmt.Errorf("record knowledge review: %w", err)
	}
	if command.Entry != nil {
		if err := insertEntry(ctx, tx, *command.Entry); err != nil {
			return knowledge.Candidate{}, nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return knowledge.Candidate{}, nil, fmt.Errorf("commit knowledge review: %w", err)
	}
	if command.Entry != nil {
		_ = s.Rebuild(ctx)
	}
	candidate.Status = command.NewStatus
	candidate.Revision++
	candidate.UpdatedAt = command.Review.ReviewedAt
	return candidate, command.Entry, nil
}

func (s *Store) IngestEvidence(
	ctx context.Context,
	command knowledge.IngestionCommand,
) (knowledge.IngestionResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return knowledge.IngestionResult{}, fmt.Errorf("begin knowledge ingestion: %w", err)
	}
	defer tx.Rollback()

	var existingID string
	err = tx.QueryRowContext(ctx, `
		SELECT id FROM knowledge_evidence
		WHERE id = ? OR idempotency_key = ?
		LIMIT 1`, command.Evidence.ID, command.Evidence.IdempotencyKey).Scan(&existingID)
	if err == nil {
		return knowledge.IngestionResult{Duplicate: true}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return knowledge.IngestionResult{}, fmt.Errorf("check knowledge evidence idempotency: %w", err)
	}

	sourceRefs, err := json.Marshal(command.Evidence.SourceRefs)
	if err != nil {
		return knowledge.IngestionResult{}, fmt.Errorf("encode evidence sources: %w", err)
	}
	metadata, err := json.Marshal(command.Evidence.Metadata)
	if err != nil {
		return knowledge.IngestionResult{}, fmt.Errorf("encode evidence metadata: %w", err)
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO knowledge_evidence(
			id, workspace_id, project_id, source_type, source_id, source_revision,
			event_type, kind, title, content, actor_id, idempotency_key,
			provenance_uri, checksum, occurred_at, terminal, validated,
			has_conflict, confidence, source_refs_json, metadata_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		command.Evidence.ID,
		command.Evidence.WorkspaceID,
		command.Evidence.ProjectID,
		command.Evidence.SourceType,
		command.Evidence.SourceID,
		command.Evidence.SourceRevision,
		command.Evidence.EventType,
		command.Evidence.Kind,
		command.Evidence.Title,
		command.Evidence.Content,
		command.Evidence.ActorID,
		command.Evidence.IdempotencyKey,
		command.Evidence.ProvenanceURI,
		command.Evidence.Checksum,
		formatTime(command.Evidence.OccurredAt),
		boolInt(command.Evidence.Terminal),
		boolInt(command.Evidence.Validated),
		boolInt(command.Evidence.HasConflict),
		command.Evidence.Confidence,
		string(sourceRefs),
		string(metadata),
	)
	if err != nil {
		return knowledge.IngestionResult{}, fmt.Errorf("store knowledge evidence: %w", err)
	}

	result := knowledge.IngestionResult{}
	if command.Candidate != nil {
		candidate := *command.Candidate
		if err := insertCandidate(ctx, tx, candidate); err != nil {
			return knowledge.IngestionResult{}, err
		}
		result.Candidate = &candidate
	}
	if command.Entry != nil {
		entry := *command.Entry
		if err := insertEntry(ctx, tx, entry); err != nil {
			return knowledge.IngestionResult{}, err
		}
		result.Entry = &entry
	}
	if err := tx.Commit(); err != nil {
		return knowledge.IngestionResult{}, fmt.Errorf("commit knowledge ingestion: %w", err)
	}
	if command.Entry != nil {
		_ = s.Rebuild(ctx)
	}
	return result, nil
}

func insertCandidate(ctx context.Context, tx *sql.Tx, candidate knowledge.Candidate) error {
	sourceRefs, err := json.Marshal(candidate.SourceRefs)
	if err != nil {
		return fmt.Errorf("encode candidate sources: %w", err)
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO knowledge_candidate(
			id, workspace_id, project_id, kind, title, content, reason, status,
			revision, proposed_by, source_refs_json, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		candidate.ID,
		candidate.WorkspaceID,
		candidate.ProjectID,
		candidate.Kind,
		candidate.Title,
		candidate.Content,
		candidate.Reason,
		candidate.Status,
		candidate.Revision,
		candidate.ProposedBy,
		string(sourceRefs),
		formatTime(candidate.CreatedAt),
		formatTime(candidate.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("create knowledge candidate: %w", err)
	}
	return nil
}

func insertEntry(ctx context.Context, tx *sql.Tx, entry knowledge.Entry) error {
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO knowledge_entry(
			id, workspace_id, project_id, candidate_id, kind, status,
			current_revision, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.ID,
		entry.WorkspaceID,
		entry.ProjectID,
		entry.CandidateID,
		entry.Kind,
		entry.Status,
		entry.CurrentRevision,
		formatTime(entry.CreatedAt),
		formatTime(entry.UpdatedAt),
	); err != nil {
		return fmt.Errorf("create knowledge entry: %w", err)
	}
	for _, revision := range entry.Revisions {
		sourceRefs, err := json.Marshal(revision.SourceRefs)
		if err != nil {
			return fmt.Errorf("encode knowledge revision sources: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO knowledge_revision(
				entry_id, revision, title, content, created_by, created_at, source_refs_json
			) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			entry.ID,
			revision.Number,
			revision.Title,
			revision.Content,
			revision.CreatedBy,
			formatTime(revision.CreatedAt),
			string(sourceRefs),
		); err != nil {
			return fmt.Errorf("create knowledge revision: %w", err)
		}
	}
	return nil
}

type rowScanner interface {
	Scan(...any) error
}

func scanCandidate(row rowScanner) (knowledge.Candidate, error) {
	var (
		candidate      knowledge.Candidate
		sourceRefsJSON string
		createdAt      string
		updatedAt      string
	)
	err := row.Scan(
		&candidate.ID,
		&candidate.WorkspaceID,
		&candidate.ProjectID,
		&candidate.Kind,
		&candidate.Title,
		&candidate.Content,
		&candidate.Reason,
		&candidate.Status,
		&candidate.Revision,
		&candidate.ProposedBy,
		&sourceRefsJSON,
		&createdAt,
		&updatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return knowledge.Candidate{}, knowledge.ErrNotFound
	}
	if err != nil {
		return knowledge.Candidate{}, fmt.Errorf("read knowledge candidate: %w", err)
	}
	if err := json.Unmarshal([]byte(sourceRefsJSON), &candidate.SourceRefs); err != nil {
		return knowledge.Candidate{}, fmt.Errorf("decode candidate sources: %w", err)
	}
	candidate.CreatedAt, err = parseTime(createdAt)
	if err != nil {
		return knowledge.Candidate{}, err
	}
	candidate.UpdatedAt, err = parseTime(updatedAt)
	if err != nil {
		return knowledge.Candidate{}, err
	}
	return candidate, nil
}

func scanEntry(row rowScanner) (knowledge.Entry, error) {
	var (
		entry     knowledge.Entry
		createdAt string
		updatedAt string
	)
	err := row.Scan(
		&entry.ID,
		&entry.WorkspaceID,
		&entry.ProjectID,
		&entry.CandidateID,
		&entry.Kind,
		&entry.Status,
		&entry.CurrentRevision,
		&createdAt,
		&updatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return knowledge.Entry{}, knowledge.ErrNotFound
	}
	if err != nil {
		return knowledge.Entry{}, fmt.Errorf("read knowledge entry: %w", err)
	}
	entry.CreatedAt, err = parseTime(createdAt)
	if err != nil {
		return knowledge.Entry{}, err
	}
	entry.UpdatedAt, err = parseTime(updatedAt)
	if err != nil {
		return knowledge.Entry{}, err
	}
	return entry, nil
}

func scanRevision(row rowScanner) (knowledge.Revision, error) {
	var (
		revision       knowledge.Revision
		createdAt      string
		sourceRefsJSON string
	)
	if err := row.Scan(
		&revision.Number,
		&revision.Title,
		&revision.Content,
		&revision.CreatedBy,
		&createdAt,
		&sourceRefsJSON,
	); err != nil {
		return knowledge.Revision{}, fmt.Errorf("read knowledge revision: %w", err)
	}
	var err error
	revision.CreatedAt, err = parseTime(createdAt)
	if err != nil {
		return knowledge.Revision{}, err
	}
	if err := json.Unmarshal([]byte(sourceRefsJSON), &revision.SourceRefs); err != nil {
		return knowledge.Revision{}, fmt.Errorf("decode knowledge revision sources: %w", err)
	}
	return revision, nil
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func parseTime(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse knowledge time: %w", err)
	}
	return parsed, nil
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func (s *Store) Search(ctx context.Context, query knowledge.SearchQuery) (knowledge.SearchPage, error) {
	if s.fts5Enabled && strings.TrimSpace(query.Text) != "" {
		return s.searchFTS(ctx, query)
	}
	return s.searchPortable(ctx, query)
}

func (s *Store) searchPortable(ctx context.Context, query knowledge.SearchQuery) (knowledge.SearchPage, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset, _ := strconv.Atoi(query.Cursor)
	arguments := []any{query.WorkspaceID, knowledge.StatusPublished}
	where := []string{
		"e.workspace_id = ?",
		"e.status = ?",
		"r.revision = e.current_revision",
	}
	if query.ProjectID != "" {
		where = append(where, "e.project_id = ?")
		arguments = append(arguments, query.ProjectID)
	}
	if len(query.Kinds) > 0 {
		placeholders := make([]string, 0, len(query.Kinds))
		for _, kind := range query.Kinds {
			placeholders = append(placeholders, "?")
			arguments = append(arguments, kind)
		}
		where = append(where, "e.kind IN ("+strings.Join(placeholders, ",")+")")
	}
	textQuery := strings.TrimSpace(query.Text)
	if textQuery != "" {
		where = append(where, "(LOWER(r.title) LIKE LOWER(?) OR LOWER(r.content) LIKE LOWER(?))")
		pattern := "%" + escapeLike(textQuery) + "%"
		arguments = append(arguments, pattern, pattern)
	}
	arguments = append(arguments, limit+1, offset)
	rows, err := s.db.QueryContext(ctx, `
		SELECT e.id, e.workspace_id, e.project_id, e.candidate_id, e.kind, e.status,
		       e.current_revision, e.created_at, e.updated_at,
		       r.revision, r.title, r.content, r.created_by, r.created_at,
		       r.source_refs_json
		FROM knowledge_entry e
		JOIN knowledge_revision r ON r.entry_id = e.id
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY e.updated_at DESC, e.id
		LIMIT ? OFFSET ?`, arguments...)
	if err != nil {
		return knowledge.SearchPage{}, fmt.Errorf("search knowledge: %w", err)
	}
	defer rows.Close()
	results := make([]knowledge.SearchResult, 0, limit+1)
	for rows.Next() {
		var (
			entry          knowledge.Entry
			entryCreatedAt string
			entryUpdatedAt string
			revision       knowledge.Revision
			revisionTime   string
			sourceRefsJSON string
		)
		if err := rows.Scan(
			&entry.ID,
			&entry.WorkspaceID,
			&entry.ProjectID,
			&entry.CandidateID,
			&entry.Kind,
			&entry.Status,
			&entry.CurrentRevision,
			&entryCreatedAt,
			&entryUpdatedAt,
			&revision.Number,
			&revision.Title,
			&revision.Content,
			&revision.CreatedBy,
			&revisionTime,
			&sourceRefsJSON,
		); err != nil {
			return knowledge.SearchPage{}, fmt.Errorf("read knowledge search result: %w", err)
		}
		entry.CreatedAt, err = parseTime(entryCreatedAt)
		if err != nil {
			return knowledge.SearchPage{}, err
		}
		entry.UpdatedAt, err = parseTime(entryUpdatedAt)
		if err != nil {
			return knowledge.SearchPage{}, err
		}
		revision.CreatedAt, err = parseTime(revisionTime)
		if err != nil {
			return knowledge.SearchPage{}, err
		}
		if err := json.Unmarshal([]byte(sourceRefsJSON), &revision.SourceRefs); err != nil {
			return knowledge.SearchPage{}, fmt.Errorf("decode search result sources: %w", err)
		}
		entry.Revisions = []knowledge.Revision{revision}
		results = append(results, knowledge.SearchResult{
			Entry:     entry,
			Score:     1,
			MatchedBy: []string{"title", "content"},
			Citation:  "knowledge://" + entry.WorkspaceID + "/entries/" + entry.ID,
		})
	}
	if err := rows.Err(); err != nil {
		return knowledge.SearchPage{}, fmt.Errorf("search knowledge: %w", err)
	}
	nextCursor := ""
	if len(results) > limit {
		results = results[:limit]
		nextCursor = strconv.Itoa(offset + limit)
	}
	return knowledge.SearchPage{Results: results, NextCursor: nextCursor}, nil
}

func (s *Store) searchFTS(ctx context.Context, query knowledge.SearchQuery) (knowledge.SearchPage, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset, _ := strconv.Atoi(query.Cursor)
	arguments := []any{
		query.WorkspaceID,
		knowledge.StatusPublished,
		ftsPhrase(query.Text),
	}
	where := []string{
		"e.workspace_id = ?",
		"e.status = ?",
		"knowledge_search_fts MATCH ?",
	}
	if query.ProjectID != "" {
		where = append(where, "e.project_id = ?")
		arguments = append(arguments, query.ProjectID)
	}
	if len(query.Kinds) > 0 {
		placeholders := make([]string, 0, len(query.Kinds))
		for _, kind := range query.Kinds {
			placeholders = append(placeholders, "?")
			arguments = append(arguments, kind)
		}
		where = append(where, "e.kind IN ("+strings.Join(placeholders, ",")+")")
	}
	arguments = append(arguments, limit+1, offset)
	rows, err := s.db.QueryContext(ctx, `
		SELECT e.id, e.workspace_id, e.project_id, e.candidate_id, e.kind, e.status,
		       e.current_revision, e.created_at, e.updated_at,
		       r.revision, r.title, r.content, r.created_by, r.created_at,
		       r.source_refs_json
		FROM knowledge_search_fts
		JOIN knowledge_entry e ON e.id = knowledge_search_fts.entry_id
		JOIN knowledge_revision r
			ON r.entry_id = e.id AND r.revision = e.current_revision
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY bm25(knowledge_search_fts), e.updated_at DESC, e.id
		LIMIT ? OFFSET ?`, arguments...)
	if err != nil {
		return knowledge.SearchPage{}, fmt.Errorf("search knowledge FTS5 index: %w", err)
	}
	defer rows.Close()
	return scanSearchPage(rows, limit, offset)
}

func scanSearchPage(rows *sql.Rows, limit, offset int) (knowledge.SearchPage, error) {
	results := make([]knowledge.SearchResult, 0, limit+1)
	for rows.Next() {
		var (
			entry          knowledge.Entry
			entryCreatedAt string
			entryUpdatedAt string
			revision       knowledge.Revision
			revisionTime   string
			sourceRefsJSON string
		)
		if err := rows.Scan(
			&entry.ID,
			&entry.WorkspaceID,
			&entry.ProjectID,
			&entry.CandidateID,
			&entry.Kind,
			&entry.Status,
			&entry.CurrentRevision,
			&entryCreatedAt,
			&entryUpdatedAt,
			&revision.Number,
			&revision.Title,
			&revision.Content,
			&revision.CreatedBy,
			&revisionTime,
			&sourceRefsJSON,
		); err != nil {
			return knowledge.SearchPage{}, fmt.Errorf("read knowledge search result: %w", err)
		}
		var err error
		entry.CreatedAt, err = parseTime(entryCreatedAt)
		if err != nil {
			return knowledge.SearchPage{}, err
		}
		entry.UpdatedAt, err = parseTime(entryUpdatedAt)
		if err != nil {
			return knowledge.SearchPage{}, err
		}
		revision.CreatedAt, err = parseTime(revisionTime)
		if err != nil {
			return knowledge.SearchPage{}, err
		}
		if err := json.Unmarshal([]byte(sourceRefsJSON), &revision.SourceRefs); err != nil {
			return knowledge.SearchPage{}, fmt.Errorf("decode search result sources: %w", err)
		}
		entry.Revisions = []knowledge.Revision{revision}
		results = append(results, knowledge.SearchResult{
			Entry:     entry,
			Score:     1,
			MatchedBy: []string{"title", "content"},
			Citation:  "knowledge://" + entry.WorkspaceID + "/entries/" + entry.ID,
		})
	}
	if err := rows.Err(); err != nil {
		return knowledge.SearchPage{}, fmt.Errorf("search knowledge: %w", err)
	}
	nextCursor := ""
	if len(results) > limit {
		results = results[:limit]
		nextCursor = strconv.Itoa(offset + limit)
	}
	return knowledge.SearchPage{Results: results, NextCursor: nextCursor}, nil
}

func ftsPhrase(value string) string {
	return `"` + strings.ReplaceAll(strings.TrimSpace(value), `"`, `""`) + `"`
}

func escapeLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	return strings.ReplaceAll(value, `_`, `\_`)
}
