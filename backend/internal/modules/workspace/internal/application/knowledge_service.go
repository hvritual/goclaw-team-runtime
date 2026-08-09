// dddgen:service-implementation KnowledgeService; method bodies are user-owned.
package application

import (
	"context"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

type KnowledgeService struct{}

func NewKnowledgeService() *KnowledgeService { return &KnowledgeService{} }

func (s *KnowledgeService) CreateKnowledge(ctx context.Context, request contract.CreateKnowledgeRequest) (contract.CreateKnowledgeResponse, error) {
	return contract.CreateKnowledgeResponse{}, contract.ErrKnowledgeNotImplemented
}

func (s *KnowledgeService) GetKnowledge(ctx context.Context, request contract.GetKnowledgeRequest) (contract.GetKnowledgeResponse, error) {
	return contract.GetKnowledgeResponse{}, contract.ErrKnowledgeNotImplemented
}
