package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hvritual/workspace/internal/modules/system/contract"
)

type SkillCatalogRepository struct{ db *sql.DB }

func NewSkillCatalogRepository(db *sql.DB) *SkillCatalogRepository {
	return &SkillCatalogRepository{db: db}
}

func (r *SkillCatalogRepository) Create(ctx context.Context, request contract.CreateSkillCatalogRequest, skillID, versionID string, now time.Time) (contract.SkillCatalogEntry, error) {
	config, err := json.Marshal(request.Config)
	if err != nil {
		return contract.SkillCatalogEntry{}, fmt.Errorf("encode Skill config: %w", err)
	}
	timestamp := now.Format(time.RFC3339Nano)
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return contract.SkillCatalogEntry{}, fmt.Errorf("begin Skill create: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `INSERT INTO system_skills(id,origin_workspace_id,revision,created_by,created_at,updated_at) VALUES(?,?,?,?,?,?)`, skillID, request.WorkspaceID, 1, request.ActorID, timestamp, timestamp); err != nil {
		if isUniqueViolation(err) {
			return contract.SkillCatalogEntry{}, contract.ErrSkillAlreadyExists
		}
		return contract.SkillCatalogEntry{}, fmt.Errorf("insert Skill: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO system_skill_versions(id,skill_id,version_number,name,description,configuration,status,created_by,created_at) VALUES(?,?,?,?,?,?,?,?,?)`, versionID, skillID, 1, request.Name, request.Description, string(config), "draft", request.ActorID, timestamp); err != nil {
		return contract.SkillCatalogEntry{}, fmt.Errorf("insert Skill version: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO system_skill_audit(id,skill_id,version_id,workspace_id,actor_type,actor_id,action,created_at) VALUES(?,?,?,?,?,?,?,?)`, uuid.NewString(), skillID, versionID, request.WorkspaceID, request.ActorType, request.ActorID, "skill.created", timestamp); err != nil {
		return contract.SkillCatalogEntry{}, fmt.Errorf("insert Skill audit: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return contract.SkillCatalogEntry{}, fmt.Errorf("commit Skill create: %w", err)
	}
	return contract.SkillCatalogEntry{
		ID: skillID, WorkspaceID: request.WorkspaceID, VersionID: versionID, Version: "1",
		Name: request.Name, Description: request.Description, Config: request.Config,
		Status: "draft", Revision: 1, CreatedBy: request.ActorID, CreatedAt: timestamp, UpdatedAt: timestamp,
	}, nil
}

func (r *SkillCatalogRepository) DeleteCreated(ctx context.Context, skillID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin Skill create compensation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, statement := range []string{
		`DELETE FROM system_skill_audit WHERE skill_id=?`,
		`DELETE FROM system_skill_versions WHERE skill_id=?`,
		`DELETE FROM system_skills WHERE id=?`,
	} {
		if _, err := tx.ExecContext(ctx, statement, skillID); err != nil {
			return fmt.Errorf("compensate Skill create: %w", err)
		}
	}
	return tx.Commit()
}

func (r *SkillCatalogRepository) CreateVersion(ctx context.Context, identity contract.SkillIdentity, skillID string, request contract.UpdateSkillCatalogRequest, versionID string, now time.Time) (contract.SkillCatalogEntry, error) {
	connection, err := beginImmediate(ctx, r.db)
	if err != nil {
		return contract.SkillCatalogEntry{}, fmt.Errorf("begin Skill version: %w", err)
	}
	defer connection.Close()
	defer rollbackImmediate(connection)()
	current, err := getSkillEntry(ctx, connection, identity.WorkspaceID, skillID, "")
	if err != nil {
		return contract.SkillCatalogEntry{}, err
	}
	if current.Revision != request.ExpectedRevision {
		return contract.SkillCatalogEntry{}, contract.SkillRevisionConflict{CurrentRevision: current.Revision}
	}
	if current.Archived {
		return contract.SkillCatalogEntry{}, contract.ErrSkillTransition
	}
	if request.Name != nil {
		current.Name = *request.Name
	}
	if request.Description != nil {
		current.Description = *request.Description
	}
	if request.ConfigPresent {
		current.Config = request.Config
	}
	config, err := json.Marshal(current.Config)
	if err != nil {
		return contract.SkillCatalogEntry{}, fmt.Errorf("encode Skill config: %w", err)
	}
	nextVersion := parseVersion(current.Version) + 1
	timestamp := now.Format(time.RFC3339Nano)
	if _, err := connection.ExecContext(ctx, `INSERT INTO system_skill_versions(id,skill_id,version_number,name,description,configuration,status,created_by,created_at) VALUES(?,?,?,?,?,?,?,?,?)`, versionID, skillID, nextVersion, current.Name, current.Description, string(config), "draft", identity.ActorID, timestamp); err != nil {
		return contract.SkillCatalogEntry{}, fmt.Errorf("insert Skill version: %w", err)
	}
	if _, err := connection.ExecContext(ctx, `UPDATE system_skills SET revision=revision+1,updated_at=? WHERE id=? AND revision=?`, timestamp, skillID, request.ExpectedRevision); err != nil {
		return contract.SkillCatalogEntry{}, fmt.Errorf("advance Skill revision: %w", err)
	}
	if err := insertAudit(ctx, connection, identity, skillID, versionID, "skill.version_created", timestamp); err != nil {
		return contract.SkillCatalogEntry{}, err
	}
	if _, err := connection.ExecContext(ctx, `COMMIT`); err != nil {
		return contract.SkillCatalogEntry{}, fmt.Errorf("commit Skill version: %w", err)
	}
	return r.Get(ctx, identity, skillID, versionID, true)
}

func (r *SkillCatalogRepository) TransitionVersion(ctx context.Context, identity contract.SkillIdentity, skillID, versionID, transition string, expectedRevision int64, now time.Time) (contract.SkillCatalogEntry, error) {
	connection, err := beginImmediate(ctx, r.db)
	if err != nil {
		return contract.SkillCatalogEntry{}, fmt.Errorf("begin Skill transition: %w", err)
	}
	defer connection.Close()
	defer rollbackImmediate(connection)()
	current, err := getSkillEntry(ctx, connection, identity.WorkspaceID, skillID, versionID)
	if err != nil {
		return contract.SkillCatalogEntry{}, err
	}
	if current.Revision != expectedRevision {
		return contract.SkillCatalogEntry{}, contract.SkillRevisionConflict{CurrentRevision: current.Revision}
	}
	nextStatus, timestampColumn := "", ""
	switch {
	case transition == "publish" && current.Status == "draft":
		nextStatus, timestampColumn = "published", "published_at"
	case transition == "deprecate" && current.Status == "published":
		nextStatus, timestampColumn = "deprecated", "deprecated_at"
	default:
		return contract.SkillCatalogEntry{}, contract.ErrSkillTransition
	}
	timestamp := now.Format(time.RFC3339Nano)
	if _, err := connection.ExecContext(ctx, `UPDATE system_skill_versions SET status=?,`+timestampColumn+`=? WHERE id=? AND skill_id=?`, nextStatus, timestamp, versionID, skillID); err != nil {
		return contract.SkillCatalogEntry{}, fmt.Errorf("transition Skill version: %w", err)
	}
	if _, err := connection.ExecContext(ctx, `UPDATE system_skills SET revision=revision+1,updated_at=? WHERE id=? AND revision=?`, timestamp, skillID, expectedRevision); err != nil {
		return contract.SkillCatalogEntry{}, fmt.Errorf("advance Skill transition revision: %w", err)
	}
	if err := insertAudit(ctx, connection, identity, skillID, versionID, "skill."+nextStatus, timestamp); err != nil {
		return contract.SkillCatalogEntry{}, err
	}
	if _, err := connection.ExecContext(ctx, `COMMIT`); err != nil {
		return contract.SkillCatalogEntry{}, fmt.Errorf("commit Skill transition: %w", err)
	}
	return r.Get(ctx, identity, skillID, versionID, true)
}

func (r *SkillCatalogRepository) Archive(ctx context.Context, identity contract.SkillIdentity, skillID string, expectedRevision int64, now time.Time) error {
	connection, err := beginImmediate(ctx, r.db)
	if err != nil {
		return fmt.Errorf("begin Skill archive: %w", err)
	}
	defer connection.Close()
	defer rollbackImmediate(connection)()
	current, err := getSkillEntry(ctx, connection, identity.WorkspaceID, skillID, "")
	if err != nil {
		return err
	}
	if current.Revision != expectedRevision {
		return contract.SkillRevisionConflict{CurrentRevision: current.Revision}
	}
	if current.Archived {
		return contract.ErrSkillTransition
	}
	timestamp := now.Format(time.RFC3339Nano)
	if _, err := connection.ExecContext(ctx, `UPDATE system_skills SET revision=revision+1,archived_at=?,updated_at=? WHERE id=? AND revision=?`, timestamp, timestamp, skillID, expectedRevision); err != nil {
		return fmt.Errorf("archive Skill: %w", err)
	}
	if err := insertAudit(ctx, connection, identity, skillID, current.VersionID, "skill.archived", timestamp); err != nil {
		return err
	}
	_, err = connection.ExecContext(ctx, `COMMIT`)
	return err
}

func (r *SkillCatalogRepository) Restore(ctx context.Context, identity contract.SkillIdentity, skillID string, expectedRevision int64, now time.Time) (contract.SkillCatalogEntry, error) {
	connection, err := beginImmediate(ctx, r.db)
	if err != nil {
		return contract.SkillCatalogEntry{}, fmt.Errorf("begin Skill restore: %w", err)
	}
	defer connection.Close()
	defer rollbackImmediate(connection)()
	current, err := getSkillEntry(ctx, connection, identity.WorkspaceID, skillID, "")
	if err != nil {
		return contract.SkillCatalogEntry{}, err
	}
	if current.Revision != expectedRevision {
		return contract.SkillCatalogEntry{}, contract.SkillRevisionConflict{CurrentRevision: current.Revision}
	}
	if !current.Archived {
		return contract.SkillCatalogEntry{}, contract.ErrSkillTransition
	}
	timestamp := now.Format(time.RFC3339Nano)
	if _, err := connection.ExecContext(ctx, `UPDATE system_skills SET revision=revision+1,archived_at=NULL,updated_at=? WHERE id=? AND revision=?`, timestamp, skillID, expectedRevision); err != nil {
		return contract.SkillCatalogEntry{}, fmt.Errorf("restore Skill: %w", err)
	}
	if err := insertAudit(ctx, connection, identity, skillID, current.VersionID, "skill.restored", timestamp); err != nil {
		return contract.SkillCatalogEntry{}, err
	}
	if _, err := connection.ExecContext(ctx, `COMMIT`); err != nil {
		return contract.SkillCatalogEntry{}, fmt.Errorf("commit Skill restore: %w", err)
	}
	return r.Get(ctx, identity, skillID, "", true)
}

func (r *SkillCatalogRepository) Get(ctx context.Context, identity contract.SkillIdentity, skillID, versionID string, includeDrafts bool) (contract.SkillCatalogEntry, error) {
	return getSkillEntry(ctx, r.db, identity.WorkspaceID, skillID, versionID, includeDrafts)
}

func (r *SkillCatalogRepository) List(ctx context.Context, identity contract.SkillIdentity, includeDrafts bool) ([]contract.SkillCatalogEntry, error) {
	statement := `SELECT s.id,s.origin_workspace_id,v.id,v.version_number,v.name,v.description,v.configuration,v.status,s.revision,v.created_by,v.created_at,s.updated_at,s.archived_at IS NOT NULL
		FROM system_skills s JOIN system_skill_versions v ON v.skill_id=s.id
		WHERE s.origin_workspace_id=? AND s.archived_at IS NULL AND v.version_number=(SELECT MAX(v2.version_number) FROM system_skill_versions v2 WHERE v2.skill_id=s.id)
		`
	if !includeDrafts {
		statement += ` AND v.status='published'`
	}
	statement += ` ORDER BY s.updated_at DESC,s.id ASC`
	rows, err := r.db.QueryContext(ctx, statement, identity.WorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("list Skills: %w", err)
	}
	defer rows.Close()
	var values []contract.SkillCatalogEntry
	for rows.Next() {
		value, err := scanSkillEntry(rows)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func (r *SkillCatalogRepository) GetReferenced(ctx context.Context, skillID, versionID string) (contract.SkillCatalogEntry, error) {
	statement := `SELECT s.id,s.origin_workspace_id,v.id,v.version_number,v.name,v.description,v.configuration,v.status,s.revision,v.created_by,v.created_at,s.updated_at,s.archived_at IS NOT NULL
		FROM system_skills s JOIN system_skill_versions v ON v.skill_id=s.id WHERE s.id=? AND v.id=?`
	return scanSkillEntry(r.db.QueryRowContext(ctx, statement, skillID, versionID))
}

type skillRow interface{ Scan(...any) error }

func getSkillEntry(ctx context.Context, query interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, workspaceID, skillID, versionID string, includeDrafts ...bool) (contract.SkillCatalogEntry, error) {
	statement := `SELECT s.id,s.origin_workspace_id,v.id,v.version_number,v.name,v.description,v.configuration,v.status,s.revision,v.created_by,v.created_at,s.updated_at,s.archived_at IS NOT NULL
		FROM system_skills s JOIN system_skill_versions v ON v.skill_id=s.id WHERE s.origin_workspace_id=? AND s.id=?`
	args := []any{workspaceID, skillID}
	if versionID == "" {
		statement += ` AND v.version_number=(SELECT MAX(v2.version_number) FROM system_skill_versions v2 WHERE v2.skill_id=s.id)`
	} else {
		statement += ` AND v.id=?`
		args = append(args, versionID)
	}
	if len(includeDrafts) > 0 && !includeDrafts[0] {
		statement += ` AND v.status='published'`
	}
	return scanSkillEntry(query.QueryRowContext(ctx, statement, args...))
}

func scanSkillEntry(row skillRow) (contract.SkillCatalogEntry, error) {
	var value contract.SkillCatalogEntry
	var version int
	var config string
	if err := row.Scan(&value.ID, &value.WorkspaceID, &value.VersionID, &version, &value.Name, &value.Description, &config, &value.Status, &value.Revision, &value.CreatedBy, &value.CreatedAt, &value.UpdatedAt, &value.Archived); errors.Is(err, sql.ErrNoRows) {
		return contract.SkillCatalogEntry{}, contract.ErrSkillNotFound
	} else if err != nil {
		return contract.SkillCatalogEntry{}, fmt.Errorf("read Skill: %w", err)
	}
	value.Version = fmt.Sprint(version)
	if err := json.Unmarshal([]byte(config), &value.Config); err != nil {
		return contract.SkillCatalogEntry{}, fmt.Errorf("decode Skill config: %w", err)
	}
	return value, nil
}

type skillExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func insertAudit(ctx context.Context, executor skillExecutor, identity contract.SkillIdentity, skillID, versionID, action, timestamp string) error {
	if _, err := executor.ExecContext(ctx, `INSERT INTO system_skill_audit(id,skill_id,version_id,workspace_id,actor_type,actor_id,action,created_at) VALUES(?,?,?,?,?,?,?,?)`, uuid.NewString(), skillID, versionID, identity.WorkspaceID, identity.ActorType, identity.ActorID, action, timestamp); err != nil {
		return fmt.Errorf("insert Skill audit: %w", err)
	}
	return nil
}

func beginImmediate(ctx context.Context, db *sql.DB) (*sql.Conn, error) {
	connection, err := db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := connection.ExecContext(ctx, `BEGIN IMMEDIATE`); err != nil {
		_ = connection.Close()
		return nil, err
	}
	return connection, nil
}

func rollbackImmediate(connection *sql.Conn) func() {
	return func() {
		_, _ = connection.ExecContext(context.Background(), `ROLLBACK`)
	}
}

func parseVersion(value string) int {
	var version int
	_, _ = fmt.Sscan(value, &version)
	return version
}

func isUniqueViolation(err error) bool {
	return err != nil && !errors.Is(err, context.Canceled) && (contains(err.Error(), "UNIQUE constraint failed") || contains(err.Error(), "constraint failed"))
}

func contains(value, fragment string) bool {
	for index := 0; index+len(fragment) <= len(value); index++ {
		if value[index:index+len(fragment)] == fragment {
			return true
		}
	}
	return false
}

var _ contract.SkillCatalogRepository = (*SkillCatalogRepository)(nil)
