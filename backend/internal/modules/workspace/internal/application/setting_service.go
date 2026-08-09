// dddgen:service-implementation SettingService; method bodies are user-owned.
package application

import (
	"context"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

type SettingService struct{}

func NewSettingService() *SettingService { return &SettingService{} }

func (s *SettingService) PutWorkspaceSetting(ctx context.Context, request contract.PutWorkspaceSettingRequest) (contract.PutWorkspaceSettingResponse, error) {
	return contract.PutWorkspaceSettingResponse{}, contract.ErrSettingNotImplemented
}

func (s *SettingService) PutWorkspaceSkillBinding(ctx context.Context, request contract.PutWorkspaceSkillBindingRequest) (contract.PutWorkspaceSkillBindingResponse, error) {
	return contract.PutWorkspaceSkillBindingResponse{}, contract.ErrSettingNotImplemented
}
