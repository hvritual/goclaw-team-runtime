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

func (r *SkillCatalogRepository) SavePreview(ctx context.Context, value contract.SkillImportPreviewRecord) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO system_skill_import_previews(token_hash,workspace_id,actor_id,validator_version,source_checksum,expires_at,created_at) VALUES(?,?,?,?,?,?,?)`,
		value.TokenHash, value.WorkspaceID, value.ActorID, value.ValidatorVersion, value.SourceChecksum, value.ExpiresAt.UTC().Format(time.RFC3339Nano), value.CreatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("save Skill import preview: %w", err)
	}
	return nil
}

func (r *SkillCatalogRepository) GetPreview(ctx context.Context, tokenHash string) (contract.SkillImportPreviewRecord, error) {
	var value contract.SkillImportPreviewRecord
	var expiresAt, createdAt string
	err := r.db.QueryRowContext(ctx, `SELECT token_hash,workspace_id,actor_id,validator_version,source_checksum,expires_at,created_at FROM system_skill_import_previews WHERE token_hash=?`, tokenHash).Scan(
		&value.TokenHash, &value.WorkspaceID, &value.ActorID, &value.ValidatorVersion, &value.SourceChecksum, &expiresAt, &createdAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return contract.SkillImportPreviewRecord{}, contract.ErrInvalidSkill
	}
	if err != nil {
		return contract.SkillImportPreviewRecord{}, fmt.Errorf("read Skill import preview: %w", err)
	}
	value.ExpiresAt, err = time.Parse(time.RFC3339Nano, expiresAt)
	if err != nil {
		return contract.SkillImportPreviewRecord{}, fmt.Errorf("decode Skill import expiry: %w", err)
	}
	value.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	return value, err
}

func (r *SkillCatalogRepository) DiscardPreview(ctx context.Context, tokenHash string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM system_skill_import_previews WHERE token_hash=?`, tokenHash)
	if err != nil {
		return fmt.Errorf("discard Skill import preview: %w", err)
	}
	return nil
}

func (r *SkillCatalogRepository) FindImportResult(ctx context.Context, workspaceID, idempotencyKey, requestHash string) (contract.SkillCatalogEntry, bool, error) {
	var storedHash, response string
	err := r.db.QueryRowContext(ctx, `SELECT request_hash,response_json FROM system_skill_import_idempotency WHERE workspace_id=? AND idempotency_key=?`, workspaceID, idempotencyKey).Scan(&storedHash, &response)
	if errors.Is(err, sql.ErrNoRows) {
		return contract.SkillCatalogEntry{}, false, nil
	}
	if err != nil {
		return contract.SkillCatalogEntry{}, false, fmt.Errorf("read Skill import idempotency: %w", err)
	}
	if storedHash != requestHash {
		return contract.SkillCatalogEntry{}, false, contract.ErrSkillImportConflict
	}
	var value contract.SkillCatalogEntry
	if err := json.Unmarshal([]byte(response), &value); err != nil {
		return contract.SkillCatalogEntry{}, false, fmt.Errorf("decode Skill import replay: %w", err)
	}
	return value, true, nil
}

func (r *SkillCatalogRepository) Import(ctx context.Context, request contract.ImportSkillRequest, now time.Time, bind contract.SkillCreateBinding, promote contract.SkillObjectPromoter) (contract.SkillCatalogEntry, error) {
	connection, err := beginImmediate(ctx, r.db)
	if err != nil {
		return contract.SkillCatalogEntry{}, fmt.Errorf("begin Skill import: %w", err)
	}
	defer connection.Close()
	defer rollbackImmediate(connection)()
	var storedHash, storedResponse string
	err = connection.QueryRowContext(ctx, `SELECT request_hash,response_json FROM system_skill_import_idempotency WHERE workspace_id=? AND idempotency_key=?`, request.Identity.WorkspaceID, request.IdempotencyKey).Scan(&storedHash, &storedResponse)
	if err == nil {
		if storedHash != request.RequestHash {
			return contract.SkillCatalogEntry{}, contract.ErrSkillImportConflict
		}
		var replay contract.SkillCatalogEntry
		if err := json.Unmarshal([]byte(storedResponse), &replay); err != nil {
			return contract.SkillCatalogEntry{}, err
		}
		return replay, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return contract.SkillCatalogEntry{}, err
	}
	var previewWorkspace, previewActor, validatorVersion, sourceChecksum, expiresAt string
	err = connection.QueryRowContext(ctx, `SELECT workspace_id,actor_id,validator_version,source_checksum,expires_at FROM system_skill_import_previews WHERE token_hash=?`, request.PreviewTokenHash).Scan(&previewWorkspace, &previewActor, &validatorVersion, &sourceChecksum, &expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return contract.SkillCatalogEntry{}, contract.ErrInvalidSkill
	}
	if err != nil {
		return contract.SkillCatalogEntry{}, err
	}
	expiry, err := time.Parse(time.RFC3339Nano, expiresAt)
	if err != nil || previewWorkspace != request.Identity.WorkspaceID || previewActor != request.Identity.ActorID || validatorVersion != "skill-import-v1" || sourceChecksum != request.SourceChecksum || !now.Before(expiry) {
		return contract.SkillCatalogEntry{}, contract.ErrInvalidSkill
	}
	timestamp := now.UTC().Format(time.RFC3339Nano)
	var existingID string
	err = connection.QueryRowContext(ctx, `SELECT s.id FROM system_skills s JOIN system_skill_versions v ON v.skill_id=s.id WHERE s.origin_workspace_id=? AND s.archived_at IS NULL AND v.version_number=(SELECT MAX(v2.version_number) FROM system_skill_versions v2 WHERE v2.skill_id=s.id) AND v.name=? ORDER BY s.id LIMIT 1`, request.Identity.WorkspaceID, request.Name).Scan(&existingID)
	newDefinition := errors.Is(err, sql.ErrNoRows)
	if err != nil && !newDefinition {
		return contract.SkillCatalogEntry{}, err
	}
	entry := contract.SkillCatalogEntry{WorkspaceID: request.Identity.WorkspaceID, VersionID: request.VersionID, Name: request.Name, Description: request.Description, Config: map[string]any{}, Status: "draft", CreatedBy: request.Identity.ActorID, CreatedAt: timestamp, UpdatedAt: timestamp}
	if newDefinition {
		entry.ID, entry.Version, entry.Revision = request.SkillID, "1", 1
		if _, err := connection.ExecContext(ctx, `INSERT INTO system_skills(id,origin_workspace_id,revision,created_by,created_at,updated_at) VALUES(?,?,?,?,?,?)`, entry.ID, entry.WorkspaceID, entry.Revision, entry.CreatedBy, timestamp, timestamp); err != nil {
			return contract.SkillCatalogEntry{}, fmt.Errorf("insert imported Skill: %w", err)
		}
		if bind == nil {
			return contract.SkillCatalogEntry{}, contract.ErrSkillAccessDenied
		}
	} else {
		current, err := getSkillEntry(ctx, connection, request.Identity.WorkspaceID, existingID, "")
		if err != nil {
			return contract.SkillCatalogEntry{}, err
		}
		if request.ConflictMode == "replace" {
			if request.ExpectedRevision != current.Revision {
				return contract.SkillCatalogEntry{}, contract.SkillRevisionConflict{CurrentRevision: current.Revision}
			}
			if current.Status != "draft" {
				return contract.SkillCatalogEntry{}, contract.ErrSkillTransition
			}
			if _, err := connection.ExecContext(ctx, `UPDATE system_skill_versions SET status='archived' WHERE id=? AND status='draft'`, current.VersionID); err != nil {
				return contract.SkillCatalogEntry{}, err
			}
		}
		entry.ID, entry.Version, entry.Revision = existingID, fmt.Sprint(parseVersion(current.Version)+1), current.Revision+1
		if _, err := connection.ExecContext(ctx, `UPDATE system_skills SET revision=?,updated_at=? WHERE id=? AND revision=?`, entry.Revision, timestamp, entry.ID, current.Revision); err != nil {
			return contract.SkillCatalogEntry{}, fmt.Errorf("advance imported Skill revision: %w", err)
		}
	}
	if _, err := connection.ExecContext(ctx, `INSERT INTO system_skill_versions(id,skill_id,version_number,name,description,configuration,status,created_by,created_at) VALUES(?,?,?,?,?,'{}','draft',?,?)`, entry.VersionID, entry.ID, parseVersion(entry.Version), entry.Name, entry.Description, entry.CreatedBy, timestamp); err != nil {
		return contract.SkillCatalogEntry{}, fmt.Errorf("insert imported Skill version: %w", err)
	}
	for _, file := range request.Files {
		manifestID := file.ID
		if manifestID == "" {
			manifestID = uuid.NewString()
		}
		if _, err := connection.ExecContext(ctx, `INSERT INTO system_skill_file_manifests(id,skill_id,version_id,path,space_object_id,media_type,size_bytes,checksum,created_at) VALUES(?,?,?,?,?,?,?,?,?)`, manifestID, entry.ID, entry.VersionID, file.Path, file.SpaceObjectID, file.MediaType, file.SizeBytes, file.Checksum, timestamp); err != nil {
			return contract.SkillCatalogEntry{}, fmt.Errorf("insert imported Skill manifest: %w", err)
		}
		if promote == nil {
			return contract.SkillCatalogEntry{}, errors.New("promote imported Skill object: promoter is required")
		}
		if err := promote(ctx, skillCreateExecutor{executor: connection}, file.SpaceObjectID); err != nil {
			return contract.SkillCatalogEntry{}, fmt.Errorf("promote imported Skill object: %w", err)
		}
	}
	if newDefinition {
		if err := bind(ctx, skillCreateExecutor{executor: connection}); err != nil {
			return contract.SkillCatalogEntry{}, fmt.Errorf("bind imported Skill: %w", err)
		}
	}
	if err := insertAudit(ctx, connection, request.Identity, entry.ID, entry.VersionID, "skill.imported", timestamp); err != nil {
		return contract.SkillCatalogEntry{}, err
	}
	if _, err := connection.ExecContext(ctx, `DELETE FROM system_skill_import_previews WHERE token_hash=?`, request.PreviewTokenHash); err != nil {
		return contract.SkillCatalogEntry{}, err
	}
	response, err := json.Marshal(entry)
	if err != nil {
		return contract.SkillCatalogEntry{}, err
	}
	if _, err := connection.ExecContext(ctx, `INSERT INTO system_skill_import_idempotency(workspace_id,idempotency_key,request_hash,response_json,created_at) VALUES(?,?,?,?,?)`, entry.WorkspaceID, request.IdempotencyKey, request.RequestHash, string(response), timestamp); err != nil {
		return contract.SkillCatalogEntry{}, fmt.Errorf("store Skill import idempotency: %w", err)
	}
	if _, err := connection.ExecContext(ctx, `COMMIT`); err != nil {
		return contract.SkillCatalogEntry{}, fmt.Errorf("commit Skill import: %w", err)
	}
	return entry, nil
}

var _ contract.SkillImportRepository = (*SkillCatalogRepository)(nil)
