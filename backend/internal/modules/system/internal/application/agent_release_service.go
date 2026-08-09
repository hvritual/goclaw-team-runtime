// dddgen:service-implementation AgentReleaseService; method bodies are user-owned.
package application

import (
	"context"

	"github.com/hvritual/workspace/internal/modules/system/contract"
)

type AgentReleaseService struct{}

func NewAgentReleaseService() *AgentReleaseService { return &AgentReleaseService{} }

func (s *AgentReleaseService) PublishAgentRelease(ctx context.Context, request contract.PublishAgentReleaseRequest) (contract.PublishAgentReleaseResponse, error) {
	return contract.PublishAgentReleaseResponse{}, contract.ErrAgentReleaseNotImplemented
}

func (s *AgentReleaseService) ResolveAgentUpgrade(ctx context.Context, request contract.ResolveAgentUpgradeRequest) (contract.ResolveAgentUpgradeResponse, error) {
	return contract.ResolveAgentUpgradeResponse{}, contract.ErrAgentReleaseNotImplemented
}
