package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
	settingDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/setting"
)

type settingRepository struct{ db *sql.DB }

func NewSettingRepository(c Config) (application.SettingRepository, error) {
	if c.DB == nil {
		return nil, errors.New("workspace sqlite database is required")
	}
	return &settingRepository{c.DB}, nil
}
func (r *settingRepository) PutSetting(ctx context.Context, v settingDomain.Setting) error {
	raw, err := json.Marshal(v.Value)
	if err != nil {
		return fmt.Errorf("encode Workspace setting: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO workspace_settings(workspace_id,key,value,updated_at) VALUES(?,?,?,?) ON CONFLICT(workspace_id,key) DO UPDATE SET value=excluded.value,updated_at=excluded.updated_at`, v.WorkspaceID, v.Key, string(raw), v.UpdatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("upsert Workspace setting: %w", err)
	}
	return nil
}
func (r *settingRepository) PutSkillBinding(ctx context.Context, v settingDomain.SkillBinding) error {
	config, err := json.Marshal(v.Configuration)
	if err != nil {
		return err
	}
	agents, err := json.Marshal(v.AgentIDs)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO workspace_skill_bindings(workspace_id,skill_id,skill_version_id,enabled,configuration,agent_ids,updated_at) VALUES(?,?,?,?,?,?,?) ON CONFLICT(workspace_id,skill_id) DO UPDATE SET skill_version_id=excluded.skill_version_id,enabled=excluded.enabled,configuration=excluded.configuration,agent_ids=excluded.agent_ids,updated_at=excluded.updated_at`, v.WorkspaceID, v.SkillID, nullable(v.SkillVersionID), v.Enabled, string(config), string(agents), v.UpdatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("upsert Workspace Skill binding: %w", err)
	}
	return nil
}
