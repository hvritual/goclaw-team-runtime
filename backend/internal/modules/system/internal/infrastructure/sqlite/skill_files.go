package sqlite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hvritual/workspace/internal/modules/system/contract"
)

func (r *SkillCatalogRepository) ListFiles(ctx context.Context, identity contract.SkillIdentity, skillID, versionID string) ([]contract.SkillFileManifest, error) {
	statement := `SELECT f.id,f.skill_id,f.version_id,f.path,f.space_object_id,f.media_type,f.size_bytes,f.checksum,f.created_at
		FROM system_skill_file_manifests f
		JOIN system_skills s ON s.id=f.skill_id
		JOIN system_skill_versions v ON v.id=f.version_id AND v.skill_id=f.skill_id
		WHERE s.origin_workspace_id=? AND s.id=?`
	arguments := []any{identity.WorkspaceID, skillID}
	if versionID == "" {
		statement += ` AND v.version_number=(SELECT MAX(v2.version_number) FROM system_skill_versions v2 WHERE v2.skill_id=s.id)`
	} else {
		statement += ` AND v.id=?`
		arguments = append(arguments, versionID)
	}
	statement += ` ORDER BY f.path`
	rows, err := r.db.QueryContext(ctx, statement, arguments...)
	if err != nil {
		return nil, fmt.Errorf("list Skill files: %w", err)
	}
	defer rows.Close()
	values := make([]contract.SkillFileManifest, 0)
	for rows.Next() {
		var value contract.SkillFileManifest
		if err := rows.Scan(&value.ID, &value.SkillID, &value.VersionID, &value.Path, &value.SpaceObjectID, &value.MediaType, &value.SizeBytes, &value.Checksum, &value.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan Skill file manifest: %w", err)
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func (r *SkillCatalogRepository) CreateFileVersion(ctx context.Context, identity contract.SkillIdentity, skillID string, mutation contract.SkillFileMutation, versionID string, now time.Time, promote contract.SkillObjectPromoter) (contract.SkillCatalogEntry, error) {
	connection, err := beginImmediate(ctx, r.db)
	if err != nil {
		return contract.SkillCatalogEntry{}, fmt.Errorf("begin Skill file version: %w", err)
	}
	defer connection.Close()
	defer rollbackImmediate(connection)()
	current, err := getSkillEntry(ctx, connection, identity.WorkspaceID, skillID, "")
	if err != nil {
		return contract.SkillCatalogEntry{}, err
	}
	if current.Revision != mutation.ExpectedRevision {
		return contract.SkillCatalogEntry{}, contract.SkillRevisionConflict{CurrentRevision: current.Revision}
	}
	if current.Archived {
		return contract.SkillCatalogEntry{}, contract.ErrSkillTransition
	}
	if mutation.Delete && mutation.Path == "SKILL.md" {
		return contract.SkillCatalogEntry{}, errors.New("SKILL.md cannot be deleted")
	}
	timestamp := now.UTC().Format(time.RFC3339Nano)
	nextVersion := parseVersion(current.Version) + 1
	config, err := jsonConfig(current.Config)
	if err != nil {
		return contract.SkillCatalogEntry{}, err
	}
	if _, err := connection.ExecContext(ctx, `INSERT INTO system_skill_versions(id,skill_id,version_number,name,description,configuration,status,created_by,created_at) VALUES(?,?,?,?,?,?,?,?,?)`, versionID, skillID, nextVersion, current.Name, current.Description, config, "draft", identity.ActorID, timestamp); err != nil {
		return contract.SkillCatalogEntry{}, fmt.Errorf("insert Skill file version: %w", err)
	}
	if _, err := connection.ExecContext(ctx, `INSERT INTO system_skill_file_manifests(id,skill_id,version_id,path,space_object_id,media_type,size_bytes,checksum,created_at)
		SELECT lower(hex(randomblob(16))),skill_id,?,path,space_object_id,media_type,size_bytes,checksum,? FROM system_skill_file_manifests WHERE skill_id=? AND version_id=?`, versionID, timestamp, skillID, current.VersionID); err != nil {
		return contract.SkillCatalogEntry{}, fmt.Errorf("copy Skill file manifest: %w", err)
	}
	if _, err := connection.ExecContext(ctx, `DELETE FROM system_skill_file_manifests WHERE skill_id=? AND version_id=? AND path=?`, skillID, versionID, mutation.Path); err != nil {
		return contract.SkillCatalogEntry{}, fmt.Errorf("replace Skill file manifest: %w", err)
	}
	if !mutation.Delete {
		if mutation.Object == nil || mutation.Object.SpaceObjectID == "" {
			return contract.SkillCatalogEntry{}, errors.New("Skill file object is required")
		}
		manifestID := mutation.Object.ID
		if manifestID == "" {
			manifestID = uuid.NewString()
		}
		if _, err := connection.ExecContext(ctx, `INSERT INTO system_skill_file_manifests(id,skill_id,version_id,path,space_object_id,media_type,size_bytes,checksum,created_at) VALUES(?,?,?,?,?,?,?,?,?)`, manifestID, skillID, versionID, mutation.Path, mutation.Object.SpaceObjectID, mutation.Object.MediaType, mutation.Object.SizeBytes, mutation.Object.Checksum, timestamp); err != nil {
			return contract.SkillCatalogEntry{}, fmt.Errorf("insert Skill file manifest: %w", err)
		}
		if promote == nil {
			return contract.SkillCatalogEntry{}, errors.New("Skill object promoter is required")
		}
		if err := promote(ctx, skillCreateExecutor{executor: connection}, mutation.Object.SpaceObjectID); err != nil {
			return contract.SkillCatalogEntry{}, fmt.Errorf("promote Skill file object: %w", err)
		}
	}
	if _, err := connection.ExecContext(ctx, `UPDATE system_skills SET revision=revision+1,updated_at=? WHERE id=? AND revision=?`, timestamp, skillID, mutation.ExpectedRevision); err != nil {
		return contract.SkillCatalogEntry{}, fmt.Errorf("advance Skill file revision: %w", err)
	}
	if err := insertAudit(ctx, connection, identity, skillID, versionID, "skill.files_versioned", timestamp); err != nil {
		return contract.SkillCatalogEntry{}, err
	}
	if _, err := connection.ExecContext(ctx, `COMMIT`); err != nil {
		return contract.SkillCatalogEntry{}, fmt.Errorf("commit Skill file version: %w", err)
	}
	return r.Get(ctx, identity, skillID, versionID, true)
}

func jsonConfig(value map[string]any) (string, error) {
	body, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode Skill config: %w", err)
	}
	return string(body), nil
}

var _ contract.SkillFileRepository = (*SkillCatalogRepository)(nil)
