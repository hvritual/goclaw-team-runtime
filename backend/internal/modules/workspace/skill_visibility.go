package workspace

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

type sqliteSkillVisibilityService struct {
	db         *sql.DB
	authorizer contract.WorkspaceAccessAuthorizer
	now        func() time.Time
}

func NewSQLiteSkillVisibilityService(db *sql.DB, authorizer contract.WorkspaceAccessAuthorizer) (contract.SkillVisibilityService, error) {
	if db == nil || authorizer == nil {
		return nil, errors.New("Skill visibility dependencies are required")
	}
	return &sqliteSkillVisibilityService{db: db, authorizer: authorizer, now: time.Now}, nil
}

func (s *sqliteSkillVisibilityService) AuthorizeInitialSkill(ctx context.Context, request contract.BindInitialSkillRequest) error {
	if err := validateInitialSkillBinding(&request); err != nil {
		return err
	}
	return s.authorizer.AuthorizeWorkspace(ctx, request.WorkspaceID, contract.PermissionSkillCreate)
}

func (s *sqliteSkillVisibilityService) BindInitialSkill(ctx context.Context, executor contract.SkillBindingExecutor, request contract.BindInitialSkillRequest) error {
	if err := validateInitialSkillBinding(&request); err != nil {
		return err
	}
	if executor == nil {
		return errors.New("Skill visibility binding executor is required")
	}
	if err := executor.Execute(ctx, `INSERT INTO workspace_skill_bindings(workspace_id,skill_id,skill_version_id,enabled,configuration,agent_ids,updated_at) VALUES(?,?,?,?,?,?,?)`, request.WorkspaceID, request.SkillID, request.VersionID, true, `{}`, `[]`, s.now().UTC().Format(time.RFC3339Nano)); err != nil {
		return fmt.Errorf("bind initial Workspace Skill: %w", err)
	}
	return nil
}

func validateInitialSkillBinding(request *contract.BindInitialSkillRequest) error {
	request.WorkspaceID = strings.TrimSpace(request.WorkspaceID)
	request.SkillID = strings.TrimSpace(request.SkillID)
	request.VersionID = strings.TrimSpace(request.VersionID)
	if request.WorkspaceID == "" || request.SkillID == "" || request.VersionID == "" {
		return errors.New("invalid Skill visibility binding")
	}
	return nil
}

func (s *sqliteSkillVisibilityService) ResolveSkill(ctx context.Context, workspaceID, skillID string) (contract.SkillVisibilityReference, error) {
	workspaceID, skillID = strings.TrimSpace(workspaceID), strings.TrimSpace(skillID)
	if workspaceID == "" || skillID == "" {
		return contract.SkillVisibilityReference{}, errors.New("invalid Skill visibility reference")
	}
	var value contract.SkillVisibilityReference
	var agentIDs string
	err := s.db.QueryRowContext(ctx, `SELECT workspace_id,skill_id,COALESCE(skill_version_id,''),enabled,agent_ids FROM workspace_skill_bindings WHERE workspace_id=? AND skill_id=?`, workspaceID, skillID).Scan(&value.WorkspaceID, &value.SkillID, &value.VersionID, &value.Enabled, &agentIDs)
	if err != nil {
		return contract.SkillVisibilityReference{}, fmt.Errorf("resolve Workspace Skill visibility: %w", err)
	}
	if err := json.Unmarshal([]byte(agentIDs), &value.AgentIDs); err != nil {
		return contract.SkillVisibilityReference{}, fmt.Errorf("decode Workspace Skill agents: %w", err)
	}
	return value, nil
}

func (s *sqliteSkillVisibilityService) ListSkills(ctx context.Context, workspaceID string) ([]contract.SkillVisibilityReference, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, errors.New("invalid Workspace Skill visibility list")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT workspace_id,skill_id,COALESCE(skill_version_id,''),enabled,agent_ids FROM workspace_skill_bindings WHERE workspace_id=? ORDER BY skill_id`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list Workspace Skill visibility: %w", err)
	}
	defer rows.Close()
	values := make([]contract.SkillVisibilityReference, 0)
	for rows.Next() {
		var value contract.SkillVisibilityReference
		var agentIDs string
		if err := rows.Scan(&value.WorkspaceID, &value.SkillID, &value.VersionID, &value.Enabled, &agentIDs); err != nil {
			return nil, fmt.Errorf("scan Workspace Skill visibility: %w", err)
		}
		if err := json.Unmarshal([]byte(agentIDs), &value.AgentIDs); err != nil {
			return nil, fmt.Errorf("decode Workspace Skill agents: %w", err)
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

var _ contract.SkillVisibilityService = (*sqliteSkillVisibilityService)(nil)
