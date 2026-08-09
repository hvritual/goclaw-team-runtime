package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	settingDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/setting"
)

const (
	PermissionSettingPut      = "workspace.setting.put"
	PermissionSkillBindingPut = "workspace.skill_binding.put"
)

type SettingRepository interface {
	PutSetting(context.Context, settingDomain.Setting) error
	PutSkillBinding(context.Context, settingDomain.SkillBinding) error
}
type SettingUseCase struct {
	repository SettingRepository
	authorizer contract.WorkspaceAccessAuthorizer
	actors     contract.WorkspaceActorReader
	skills     contract.SkillReferenceReader
	now        Clock
}

func NewSettingUseCase(repository SettingRepository, authorizer contract.WorkspaceAccessAuthorizer, actors contract.WorkspaceActorReader, skills contract.SkillReferenceReader, now Clock) (*SettingUseCase, error) {
	if repository == nil || authorizer == nil || actors == nil || skills == nil || now == nil {
		return nil, errors.New("Setting dependencies are required")
	}
	return &SettingUseCase{repository: repository, authorizer: authorizer, actors: actors, skills: skills, now: now}, nil
}
func (s *SettingUseCase) PutWorkspaceSetting(ctx context.Context, r contract.PutWorkspaceSettingRequest) (contract.PutWorkspaceSettingResponse, error) {
	w := strings.TrimSpace(r.WorkspaceId)
	if w == "" {
		return contract.PutWorkspaceSettingResponse{}, fmt.Errorf("%w: workspace id is required", contract.ErrInvalidWorkspaceSetting)
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, w, PermissionSettingPut); err != nil {
		return contract.PutWorkspaceSettingResponse{}, err
	}
	v, err := settingDomain.New(w, r.Key, r.Value, s.now())
	if err != nil {
		return contract.PutWorkspaceSettingResponse{}, fmt.Errorf("%w: %v", contract.ErrInvalidWorkspaceSetting, err)
	}
	if err := s.repository.PutSetting(ctx, v); err != nil {
		return contract.PutWorkspaceSettingResponse{}, fmt.Errorf("put Workspace setting: %w", err)
	}
	out := contract.WorkspaceSetting{WorkspaceId: v.WorkspaceID, Key: v.Key, Value: v.Value, UpdatedAt: v.UpdatedAt.Format(time.RFC3339Nano)}
	return contract.PutWorkspaceSettingResponse{Setting: &out}, nil
}
func (s *SettingUseCase) PutWorkspaceSkillBinding(ctx context.Context, r contract.PutWorkspaceSkillBindingRequest) (contract.PutWorkspaceSkillBindingResponse, error) {
	w := strings.TrimSpace(r.WorkspaceId)
	if w == "" {
		return contract.PutWorkspaceSkillBindingResponse{}, fmt.Errorf("%w: workspace id is required", contract.ErrInvalidWorkspaceSkillBinding)
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, w, PermissionSkillBindingPut); err != nil {
		return contract.PutWorkspaceSkillBindingResponse{}, err
	}
	v, err := settingDomain.NewSkillBinding(w, r.SkillId, r.SkillVersionId, r.Enabled, r.Configuration, r.AgentIds, s.now())
	if err != nil {
		return contract.PutWorkspaceSkillBindingResponse{}, fmt.Errorf("%w: %v", contract.ErrInvalidWorkspaceSkillBinding, err)
	}
	exists, err := s.skills.SkillReferenceExists(ctx, v.SkillID, v.SkillVersionID)
	if err != nil {
		return contract.PutWorkspaceSkillBindingResponse{}, fmt.Errorf("validate Skill reference: %w", err)
	}
	if !exists {
		return contract.PutWorkspaceSkillBindingResponse{}, contract.ErrSkillReferenceNotFound
	}
	for _, agentID := range v.AgentIDs {
		belongs, err := s.actors.ActorBelongsToWorkspace(ctx, w, "agent", agentID)
		if err != nil {
			return contract.PutWorkspaceSkillBindingResponse{}, fmt.Errorf("validate Skill Agent: %w", err)
		}
		if !belongs {
			return contract.PutWorkspaceSkillBindingResponse{}, contract.ErrActorOutsideWorkspace
		}
	}
	if err := s.repository.PutSkillBinding(ctx, v); err != nil {
		return contract.PutWorkspaceSkillBindingResponse{}, fmt.Errorf("put Workspace Skill binding: %w", err)
	}
	out := contract.WorkspaceSkillBinding{WorkspaceId: v.WorkspaceID, SkillId: v.SkillID, SkillVersionId: v.SkillVersionID, Enabled: v.Enabled, Configuration: v.Configuration, AgentIds: append([]string(nil), v.AgentIDs...), UpdatedAt: v.UpdatedAt.Format(time.RFC3339Nano)}
	return contract.PutWorkspaceSkillBindingResponse{Binding: &out}, nil
}

var _ contract.SettingService = (*SettingUseCase)(nil)
