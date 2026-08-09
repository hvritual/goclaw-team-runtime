package application

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	relationDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/relationship"
)

const (
	PermissionRelationshipPut    = "workspace.relationship.put"
	PermissionRelationshipList   = "workspace.relationship.list"
	PermissionRelationshipDelete = "workspace.relationship.delete"
)

type ProjectActorRelationRepository interface {
	Put(context.Context, relationDomain.Relation) error
	List(context.Context, string, string) ([]relationDomain.Relation, error)
	Delete(context.Context, string, string, string, string) error
}

type RelationshipUseCase struct {
	projects   ProjectRepository
	relations  ProjectActorRelationRepository
	authorizer contract.WorkspaceAccessAuthorizer
	actors     contract.WorkspaceActorReader
}

func NewRelationshipUseCase(projects ProjectRepository, relations ProjectActorRelationRepository, authorizer contract.WorkspaceAccessAuthorizer, actors contract.WorkspaceActorReader) (*RelationshipUseCase, error) {
	if projects == nil {
		return nil, errors.New("project repository is required")
	}
	if relations == nil {
		return nil, errors.New("project actor relation repository is required")
	}
	if authorizer == nil {
		return nil, errors.New("workspace access authorizer is required")
	}
	if actors == nil {
		return nil, errors.New("workspace actor reader is required")
	}
	return &RelationshipUseCase{projects: projects, relations: relations, authorizer: authorizer, actors: actors}, nil
}

func (s *RelationshipUseCase) PutProjectActorRelation(ctx context.Context, request contract.PutProjectActorRelationRequest) (contract.PutProjectActorRelationResponse, error) {
	if request.Relation == nil {
		return contract.PutProjectActorRelationResponse{}, fmt.Errorf("%w: relation is required", contract.ErrInvalidProjectActorRelation)
	}
	value, err := relationDomain.New(
		request.Relation.WorkspaceId, request.Relation.ProjectId,
		request.Relation.ActorType, request.Relation.ActorId, request.Relation.Role,
	)
	if err != nil {
		return contract.PutProjectActorRelationResponse{}, fmt.Errorf("%w: %v", contract.ErrInvalidProjectActorRelation, err)
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, value.WorkspaceID(), PermissionRelationshipPut); err != nil {
		return contract.PutProjectActorRelationResponse{}, err
	}
	if err := s.ensureProject(ctx, value.WorkspaceID(), value.ProjectID()); err != nil {
		return contract.PutProjectActorRelationResponse{}, err
	}
	belongs, err := s.actors.ActorBelongsToWorkspace(ctx, value.WorkspaceID(), value.ActorType(), value.ActorID())
	if err != nil {
		return contract.PutProjectActorRelationResponse{}, fmt.Errorf("verify project actor workspace: %w", err)
	}
	if !belongs {
		return contract.PutProjectActorRelationResponse{}, contract.ErrActorOutsideWorkspace
	}
	if err := s.relations.Put(ctx, value); err != nil {
		return contract.PutProjectActorRelationResponse{}, fmt.Errorf("put project actor relation: %w", err)
	}
	result := relationToContract(value)
	return contract.PutProjectActorRelationResponse{Relation: &result}, nil
}

func (s *RelationshipUseCase) ListProjectActorRelations(ctx context.Context, request contract.ListProjectActorRelationsRequest) (contract.ListProjectActorRelationsResponse, error) {
	workspaceID := strings.TrimSpace(request.WorkspaceId)
	projectID := strings.TrimSpace(request.ProjectId)
	if workspaceID == "" || projectID == "" {
		return contract.ListProjectActorRelationsResponse{}, fmt.Errorf("%w: workspace id and project id are required", contract.ErrInvalidProjectActorRelation)
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, workspaceID, PermissionRelationshipList); err != nil {
		return contract.ListProjectActorRelationsResponse{}, err
	}
	if err := s.ensureProject(ctx, workspaceID, projectID); err != nil {
		return contract.ListProjectActorRelationsResponse{}, err
	}
	values, err := s.relations.List(ctx, workspaceID, projectID)
	if err != nil {
		return contract.ListProjectActorRelationsResponse{}, fmt.Errorf("list project actor relations: %w", err)
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].Role() != values[j].Role() {
			return values[i].Role() < values[j].Role()
		}
		if values[i].ActorType() != values[j].ActorType() {
			return values[i].ActorType() < values[j].ActorType()
		}
		return values[i].ActorID() < values[j].ActorID()
	})
	result := make([]contract.ProjectActorRelation, len(values))
	for index, value := range values {
		result[index] = relationToContract(value)
	}
	return contract.ListProjectActorRelationsResponse{Relations: result}, nil
}

func (s *RelationshipUseCase) DeleteProjectActorRelation(ctx context.Context, request contract.DeleteProjectActorRelationRequest) (contract.DeleteProjectActorRelationResponse, error) {
	workspaceID := strings.TrimSpace(request.WorkspaceId)
	projectID := strings.TrimSpace(request.ProjectId)
	actorType := strings.TrimSpace(request.ActorType)
	actorID := strings.TrimSpace(request.ActorId)
	if err := relationDomain.ValidateReference(workspaceID, projectID, actorType, actorID); err != nil {
		return contract.DeleteProjectActorRelationResponse{}, fmt.Errorf("%w: %v", contract.ErrInvalidProjectActorRelation, err)
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, workspaceID, PermissionRelationshipDelete); err != nil {
		return contract.DeleteProjectActorRelationResponse{}, err
	}
	if err := s.ensureProject(ctx, workspaceID, projectID); err != nil {
		return contract.DeleteProjectActorRelationResponse{}, err
	}
	if err := s.relations.Delete(ctx, workspaceID, projectID, actorType, actorID); err != nil {
		return contract.DeleteProjectActorRelationResponse{}, fmt.Errorf("delete project actor relation: %w", err)
	}
	return contract.DeleteProjectActorRelationResponse{}, nil
}

func (s *RelationshipUseCase) ensureProject(ctx context.Context, workspaceID, projectID string) error {
	_, err := s.projects.FindByID(ctx, workspaceID, projectID)
	if errors.Is(err, ErrProjectRecordNotFound) {
		return contract.ErrProjectNotFound
	}
	if err != nil {
		return fmt.Errorf("get relation project: %w", err)
	}
	return nil
}

func relationToContract(value relationDomain.Relation) contract.ProjectActorRelation {
	return contract.ProjectActorRelation{
		WorkspaceId: value.WorkspaceID(), ProjectId: value.ProjectID(),
		ActorType: value.ActorType(), ActorId: value.ActorID(), Role: value.Role(),
	}
}

var _ contract.RelationshipService = (*RelationshipUseCase)(nil)
