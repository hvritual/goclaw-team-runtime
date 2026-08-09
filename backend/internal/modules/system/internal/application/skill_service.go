// dddgen:service-implementation SkillService; method bodies are user-owned.
package application

import (
	"context"

	"github.com/hvritual/workspace/internal/modules/system/contract"
)

type SkillService struct{}

func NewSkillService() *SkillService { return &SkillService{} }

func (s *SkillService) PublishSkillVersion(ctx context.Context, request contract.PublishSkillVersionRequest) (contract.PublishSkillVersionResponse, error) {
	return contract.PublishSkillVersionResponse{}, contract.ErrSkillNotImplemented
}

func (s *SkillService) GetSkill(ctx context.Context, request contract.GetSkillRequest) (contract.GetSkillResponse, error) {
	return contract.GetSkillResponse{}, contract.ErrSkillNotImplemented
}
