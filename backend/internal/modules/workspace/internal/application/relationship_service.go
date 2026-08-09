// dddgen:service-implementation RelationshipService; method bodies are user-owned.
package application

import (
	"context"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

type RelationshipService struct{}

func NewRelationshipService() *RelationshipService { return &RelationshipService{} }

func (s *RelationshipService) PutProjectActorRelation(ctx context.Context, request contract.PutProjectActorRelationRequest) (contract.PutProjectActorRelationResponse, error) {
	return contract.PutProjectActorRelationResponse{}, contract.ErrRelationshipNotImplemented
}

func (s *RelationshipService) ListProjectActorRelations(ctx context.Context, request contract.ListProjectActorRelationsRequest) (contract.ListProjectActorRelationsResponse, error) {
	return contract.ListProjectActorRelationsResponse{}, contract.ErrRelationshipNotImplemented
}

func (s *RelationshipService) DeleteProjectActorRelation(ctx context.Context, request contract.DeleteProjectActorRelationRequest) (contract.DeleteProjectActorRelationResponse, error) {
	return contract.DeleteProjectActorRelationResponse{}, contract.ErrRelationshipNotImplemented
}
