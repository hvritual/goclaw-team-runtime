// dddgen:service-implementation RequirementService; method bodies are user-owned.
package application

import (
	"context"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

type RequirementService struct{}

func NewRequirementService() *RequirementService { return &RequirementService{} }

func (s *RequirementService) SaveRequirementVersion(ctx context.Context, request contract.SaveRequirementVersionRequest) (contract.SaveRequirementVersionResponse, error) {
	return contract.SaveRequirementVersionResponse{}, contract.ErrRequirementNotImplemented
}

func (s *RequirementService) GetRequirement(ctx context.Context, request contract.GetRequirementRequest) (contract.GetRequirementResponse, error) {
	return contract.GetRequirementResponse{}, contract.ErrRequirementNotImplemented
}
