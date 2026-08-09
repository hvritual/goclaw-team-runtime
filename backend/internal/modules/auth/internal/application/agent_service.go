// dddgen:service-implementation AgentService; method bodies are user-owned.
package application

import (
	"context"

	"github.com/hvritual/workspace/internal/modules/auth/contract"
)

type AgentService struct{}

func NewAgentService() *AgentService { return &AgentService{} }

func (s *AgentService) RegisterAgent(ctx context.Context, request contract.RegisterAgentRequest) (contract.RegisterAgentResponse, error) {
	return contract.RegisterAgentResponse{}, contract.ErrAgentNotImplemented
}

func (s *AgentService) GetAgent(ctx context.Context, request contract.GetAgentRequest) (contract.GetAgentResponse, error) {
	return contract.GetAgentResponse{}, contract.ErrAgentNotImplemented
}
