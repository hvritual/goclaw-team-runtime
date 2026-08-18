package space

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hvritual/workspace/internal/modules/space/contract"
)

type sqliteSkillObjects struct {
	db    *sql.DB
	newID func() string
	now   func() time.Time
}

const skillObjectQuarantineTTL = time.Hour

func NewSQLiteSkillObjects(db *sql.DB) (contract.SkillObjectService, error) {
	if db == nil {
		return nil, errors.New("Space Skill object database is required")
	}
	return &sqliteSkillObjects{db: db, newID: uuid.NewString, now: time.Now}, nil
}

func (s *sqliteSkillObjects) Stage(ctx context.Context, request contract.StageSkillObjectRequest) (contract.SkillObject, error) {
	if err := ctx.Err(); err != nil {
		return contract.SkillObject{}, err
	}
	request.WorkspaceID = strings.TrimSpace(request.WorkspaceID)
	request.MediaType = strings.TrimSpace(request.MediaType)
	if request.WorkspaceID == "" || request.MediaType == "" {
		return contract.SkillObject{}, errors.New("Workspace and media type are required")
	}
	id := s.newID()
	checksum := fmt.Sprintf("%x", sha256.Sum256(request.Content))
	value := contract.SkillObject{
		ID: id, WorkspaceID: request.WorkspaceID, ObjectKey: "skill/" + request.WorkspaceID + "/" + id,
		MediaType: request.MediaType, SizeBytes: int64(len(request.Content)), Checksum: checksum, State: "quarantined",
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO space_skill_objects(id,workspace_id,object_key,media_type,size_bytes,checksum,content,state,created_at) VALUES(?,?,?,?,?,?,?,?,?)`,
		value.ID, value.WorkspaceID, value.ObjectKey, value.MediaType, value.SizeBytes, value.Checksum, request.Content, value.State, s.now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return contract.SkillObject{}, fmt.Errorf("stage Space Skill object: %w", err)
	}
	return value, nil
}

func (s *sqliteSkillObjects) Promote(ctx context.Context, executor contract.SkillObjectExecutor, workspaceID, id string) error {
	if executor == nil {
		return errors.New("Skill object transaction executor is required")
	}
	rows, err := executor.ExecuteResult(ctx, `UPDATE space_skill_objects SET state='committed',committed_at=? WHERE workspace_id=? AND id=? AND state='quarantined'`, s.now().UTC().Format(time.RFC3339Nano), workspaceID, id)
	if err != nil {
		return fmt.Errorf("promote Space Skill object: %w", err)
	}
	if rows != 1 {
		return contract.ErrSkillObjectNotFound
	}
	return nil
}

func (s *sqliteSkillObjects) Discard(ctx context.Context, workspaceID, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM space_skill_objects WHERE workspace_id=? AND id=? AND state='quarantined'`, workspaceID, id)
	if err != nil {
		return fmt.Errorf("discard Space Skill object: %w", err)
	}
	return nil
}

func (s *sqliteSkillObjects) Open(ctx context.Context, workspaceID, id string) (contract.SkillObject, io.ReadCloser, error) {
	var value contract.SkillObject
	var content []byte
	err := s.db.QueryRowContext(ctx, `SELECT id,workspace_id,object_key,media_type,size_bytes,checksum,state,content FROM space_skill_objects WHERE workspace_id=? AND id=? AND state='committed'`, workspaceID, id).Scan(
		&value.ID, &value.WorkspaceID, &value.ObjectKey, &value.MediaType, &value.SizeBytes, &value.Checksum, &value.State, &content,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return contract.SkillObject{}, nil, contract.ErrSkillObjectNotFound
	}
	if err != nil {
		return contract.SkillObject{}, nil, fmt.Errorf("open Space Skill object: %w", err)
	}
	return value, io.NopCloser(bytes.NewReader(content)), nil
}

func (s *sqliteSkillObjects) Reconcile(ctx context.Context, referencedIDs []string) error {
	referenced := make(map[string]struct{}, len(referencedIDs))
	for _, id := range referencedIDs {
		if id = strings.TrimSpace(id); id != "" {
			referenced[id] = struct{}{}
		}
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin Space Skill object reconciliation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	rows, err := tx.QueryContext(ctx, `SELECT id,created_at FROM space_skill_objects WHERE state='quarantined' ORDER BY id`)
	if err != nil {
		return fmt.Errorf("list quarantined Space Skill objects: %w", err)
	}
	type quarantinedObject struct {
		id        string
		createdAt time.Time
	}
	var quarantined []quarantinedObject
	for rows.Next() {
		var id, createdAt string
		if err := rows.Scan(&id, &createdAt); err != nil {
			_ = rows.Close()
			return err
		}
		parsed, err := time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			continue
		}
		quarantined = append(quarantined, quarantinedObject{id: id, createdAt: parsed})
	}
	if err := rows.Close(); err != nil {
		return err
	}
	now := s.now().UTC()
	for _, object := range quarantined {
		if _, ok := referenced[object.id]; ok {
			if _, err := tx.ExecContext(ctx, `UPDATE space_skill_objects SET state='committed',committed_at=? WHERE id=?`, now.Format(time.RFC3339Nano), object.id); err != nil {
				return fmt.Errorf("restore referenced Space Skill object: %w", err)
			}
		} else if !object.createdAt.Add(skillObjectQuarantineTTL).After(now) {
			if _, err := tx.ExecContext(ctx, `DELETE FROM space_skill_objects WHERE id=? AND state='quarantined'`, object.id); err != nil {
				return fmt.Errorf("remove orphan Space Skill object: %w", err)
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit Space Skill object reconciliation: %w", err)
	}
	return nil
}

var _ contract.SkillObjectService = (*sqliteSkillObjects)(nil)
