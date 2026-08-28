package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hvritual/workspace/internal/modules/engineering/contract"
	"github.com/hvritual/workspace/internal/modules/engineering/internal/domain"
)

type Service struct {
	repository  domain.Repository
	memberships contract.WorkspaceRoleResolver
	now         func() time.Time
}

func New(repository domain.Repository, memberships contract.WorkspaceRoleResolver, now func() time.Time) (*Service, error) {
	if repository == nil || memberships == nil {
		return nil, contract.ErrUnavailable
	}
	if now == nil {
		now = time.Now
	}
	return &Service{repository: repository, memberships: memberships, now: now}, nil
}

func (s *Service) CreateEntity(ctx context.Context, actor contract.Actor, workspaceID string, request contract.CreateEntityRequest) (contract.Entity, error) {
	if err := s.authorize(ctx, actor, workspaceID, true); err != nil {
		return contract.Entity{}, err
	}
	if err := s.expectEntityAbsent(ctx, workspaceID, request.ID); err != nil {
		return contract.Entity{}, err
	}
	value, err := domain.NewEngineeringEntity(request.ID, workspaceID, domain.EntityType(request.Type), request.Name, domain.EntityStatus(request.Status), request.OwnerRef)
	if err != nil {
		return contract.Entity{}, invalid(err)
	}
	if err := s.repository.PutEntity(ctx, value); err != nil {
		return contract.Entity{}, repositoryError(err)
	}
	return toEntity(value), nil
}

func (s *Service) GetEntity(ctx context.Context, actor contract.Actor, workspaceID, id string) (contract.Entity, error) {
	if err := s.authorize(ctx, actor, workspaceID, false); err != nil {
		return contract.Entity{}, err
	}
	value, err := s.repository.GetEntity(ctx, strings.TrimSpace(workspaceID), strings.TrimSpace(id))
	if err != nil {
		return contract.Entity{}, repositoryError(err)
	}
	return toEntity(value), nil
}

func (s *Service) ListEntities(ctx context.Context, actor contract.Actor, workspaceID string) ([]contract.Entity, error) {
	if err := s.authorize(ctx, actor, workspaceID, false); err != nil {
		return nil, err
	}
	values, err := s.repository.ListEntities(ctx, strings.TrimSpace(workspaceID))
	if err != nil {
		return nil, repositoryError(err)
	}
	result := make([]contract.Entity, len(values))
	for index, value := range values {
		result[index] = toEntity(value)
	}
	return result, nil
}

func (s *Service) UpdateEntity(ctx context.Context, actor contract.Actor, workspaceID, id string, request contract.UpdateEntityRequest) (contract.Entity, error) {
	if err := s.authorize(ctx, actor, workspaceID, true); err != nil {
		return contract.Entity{}, err
	}
	workspaceID = strings.TrimSpace(workspaceID)
	id = strings.TrimSpace(id)
	current, err := s.repository.GetEntity(ctx, workspaceID, id)
	if err != nil {
		return contract.Entity{}, repositoryError(err)
	}
	name, status, ownerRef := current.Name(), string(current.Status()), current.OwnerRef()
	if request.Name != nil {
		name = *request.Name
	}
	if request.Status != nil {
		status = *request.Status
	}
	if request.OwnerRef != nil {
		ownerRef = *request.OwnerRef
	}
	updated, err := domain.NewEngineeringEntity(id, workspaceID, current.Type(), name, domain.EntityStatus(status), ownerRef)
	if err != nil {
		return contract.Entity{}, invalid(err)
	}
	if err := s.repository.PutEntity(ctx, updated); err != nil {
		return contract.Entity{}, repositoryError(err)
	}
	return toEntity(updated), nil
}

func (s *Service) CreateSourceBinding(ctx context.Context, actor contract.Actor, workspaceID string, request contract.CreateSourceBindingRequest) (contract.SourceBinding, error) {
	if err := s.authorize(ctx, actor, workspaceID, true); err != nil {
		return contract.SourceBinding{}, err
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if _, err := s.repository.GetEntity(ctx, workspaceID, strings.TrimSpace(request.EntityID)); err != nil {
		return contract.SourceBinding{}, repositoryError(err)
	}
	if err := s.expectSourceBindingAbsent(ctx, workspaceID, request.ID); err != nil {
		return contract.SourceBinding{}, err
	}
	provenance, err := s.newProvenance(request.Provenance)
	if err != nil {
		return contract.SourceBinding{}, err
	}
	value, err := domain.NewSourceBinding(request.ID, workspaceID, request.EntityID, provenance, domain.Authority(request.Authority))
	if err != nil {
		return contract.SourceBinding{}, invalid(err)
	}
	if err := s.repository.PutSourceBinding(ctx, value); err != nil {
		return contract.SourceBinding{}, repositoryError(err)
	}
	return toSourceBinding(value), nil
}

func (s *Service) GetSourceBinding(ctx context.Context, actor contract.Actor, workspaceID, id string) (contract.SourceBinding, error) {
	if err := s.authorize(ctx, actor, workspaceID, false); err != nil {
		return contract.SourceBinding{}, err
	}
	value, err := s.repository.GetSourceBinding(ctx, strings.TrimSpace(workspaceID), strings.TrimSpace(id))
	if err != nil {
		return contract.SourceBinding{}, repositoryError(err)
	}
	return toSourceBinding(value), nil
}

func (s *Service) ListSourceBindings(ctx context.Context, actor contract.Actor, workspaceID, entityID string) ([]contract.SourceBinding, error) {
	if err := s.authorize(ctx, actor, workspaceID, false); err != nil {
		return nil, err
	}
	workspaceID = strings.TrimSpace(workspaceID)
	entityID = strings.TrimSpace(entityID)
	if _, err := s.repository.GetEntity(ctx, workspaceID, entityID); err != nil {
		return nil, repositoryError(err)
	}
	values, err := s.repository.ListSourceBindings(ctx, workspaceID, entityID)
	if err != nil {
		return nil, repositoryError(err)
	}
	result := make([]contract.SourceBinding, len(values))
	for index, value := range values {
		result[index] = toSourceBinding(value)
	}
	return result, nil
}

func (s *Service) CreateThreadEdge(ctx context.Context, actor contract.Actor, workspaceID string, request contract.CreateThreadEdgeRequest) (contract.ThreadEdge, error) {
	if err := s.authorize(ctx, actor, workspaceID, true); err != nil {
		return contract.ThreadEdge{}, err
	}
	workspaceID = strings.TrimSpace(workspaceID)
	from, err := domain.NewNodeRef(domain.NodeKind(request.From.Kind), request.From.ID)
	if err != nil {
		return contract.ThreadEdge{}, invalid(err)
	}
	to, err := domain.NewNodeRef(domain.NodeKind(request.To.Kind), request.To.ID)
	if err != nil {
		return contract.ThreadEdge{}, invalid(err)
	}
	if err := s.requireEngineeringEntityNode(ctx, workspaceID, from); err != nil {
		return contract.ThreadEdge{}, err
	}
	if err := s.requireEngineeringEntityNode(ctx, workspaceID, to); err != nil {
		return contract.ThreadEdge{}, err
	}
	if err := s.expectThreadEdgeAbsent(ctx, workspaceID, request.ID); err != nil {
		return contract.ThreadEdge{}, err
	}
	provenance, err := s.newProvenance(request.Provenance)
	if err != nil {
		return contract.ThreadEdge{}, err
	}
	value, err := domain.NewThreadEdge(request.ID, workspaceID, from, domain.RelationType(request.Relation), to, domain.Authority(request.Authority), provenance)
	if err != nil {
		return contract.ThreadEdge{}, invalid(err)
	}
	if err := s.repository.PutThreadEdge(ctx, value); err != nil {
		return contract.ThreadEdge{}, repositoryError(err)
	}
	return toThreadEdge(value), nil
}

func (s *Service) ListThreadEdges(ctx context.Context, actor contract.Actor, workspaceID string, node contract.NodeRef) ([]contract.ThreadEdge, error) {
	if err := s.authorize(ctx, actor, workspaceID, false); err != nil {
		return nil, err
	}
	ref, err := domain.NewNodeRef(domain.NodeKind(node.Kind), node.ID)
	if err != nil {
		return nil, invalid(err)
	}
	if err := s.requireEngineeringEntityNode(ctx, strings.TrimSpace(workspaceID), ref); err != nil {
		return nil, err
	}
	values, err := s.repository.ListThreadEdges(ctx, strings.TrimSpace(workspaceID), ref)
	if err != nil {
		return nil, repositoryError(err)
	}
	result := make([]contract.ThreadEdge, len(values))
	for index, value := range values {
		result[index] = toThreadEdge(value)
	}
	return result, nil
}

func (s *Service) CreateChange(ctx context.Context, actor contract.Actor, workspaceID string, request contract.CreateChangeRequest) (contract.Change, error) {
	if err := s.authorize(ctx, actor, workspaceID, true); err != nil {
		return contract.Change{}, err
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if err := s.expectChangeAbsent(ctx, workspaceID, request.ID); err != nil {
		return contract.Change{}, err
	}
	for _, entityID := range request.AffectedEntityIDs {
		if _, err := s.repository.GetEntity(ctx, workspaceID, strings.TrimSpace(entityID)); err != nil {
			return contract.Change{}, repositoryError(err)
		}
	}
	var workItem *domain.NodeRef
	if request.WorkItem != nil {
		value, err := domain.NewNodeRef(domain.NodeKind(request.WorkItem.Kind), request.WorkItem.ID)
		if err != nil {
			return contract.Change{}, invalid(err)
		}
		workItem = &value
	}
	artifacts := make([]domain.ArtifactRef, len(request.Artifacts))
	for index, artifact := range request.Artifacts {
		value, err := domain.NewArtifactRef(artifact.Kind, artifact.Locator, artifact.Revision)
		if err != nil {
			return contract.Change{}, invalid(err)
		}
		artifacts[index] = value
	}
	provenance, err := s.newProvenance(request.Provenance)
	if err != nil {
		return contract.Change{}, err
	}
	value, err := domain.NewChange(request.ID, workspaceID, request.ProjectID, request.RequirementID, workItem, request.RunID, request.Summary, request.AffectedEntityIDs, artifacts, provenance, s.now().UTC())
	if err != nil {
		return contract.Change{}, invalid(err)
	}
	if err := s.repository.PutChange(ctx, value); err != nil {
		return contract.Change{}, repositoryError(err)
	}
	return toChange(value), nil
}

func (s *Service) GetChange(ctx context.Context, actor contract.Actor, workspaceID, id string) (contract.Change, error) {
	if err := s.authorize(ctx, actor, workspaceID, false); err != nil {
		return contract.Change{}, err
	}
	value, err := s.repository.GetChange(ctx, strings.TrimSpace(workspaceID), strings.TrimSpace(id))
	if err != nil {
		return contract.Change{}, repositoryError(err)
	}
	return toChange(value), nil
}

func (s *Service) ListChanges(ctx context.Context, actor contract.Actor, workspaceID, affectedEntityID string) ([]contract.Change, error) {
	if err := s.authorize(ctx, actor, workspaceID, false); err != nil {
		return nil, err
	}
	workspaceID = strings.TrimSpace(workspaceID)
	affectedEntityID = strings.TrimSpace(affectedEntityID)
	if affectedEntityID != "" {
		if _, err := s.repository.GetEntity(ctx, workspaceID, affectedEntityID); err != nil {
			return nil, repositoryError(err)
		}
	}
	values, err := s.repository.ListChanges(ctx, workspaceID, affectedEntityID)
	if err != nil {
		return nil, repositoryError(err)
	}
	result := make([]contract.Change, len(values))
	for index, value := range values {
		result[index] = toChange(value)
	}
	return result, nil
}

func (s *Service) GetContextPack(ctx context.Context, actor contract.Actor, workspaceID, id string) (contract.ContextPack, error) {
	if err := s.authorize(ctx, actor, workspaceID, false); err != nil {
		return contract.ContextPack{}, err
	}
	value, err := s.repository.GetContextPack(ctx, strings.TrimSpace(workspaceID), strings.TrimSpace(id))
	if err != nil {
		return contract.ContextPack{}, repositoryError(err)
	}
	return toContextPack(value), nil
}

func (s *Service) authorize(ctx context.Context, actor contract.Actor, workspaceID string, write bool) error {
	userID := strings.TrimSpace(actor.UserID)
	workspaceID = strings.TrimSpace(workspaceID)
	if userID == "" || workspaceID == "" {
		return contract.ErrInvalidArgument
	}
	role, found, err := s.memberships.ResolveWorkspaceRole(ctx, userID, workspaceID)
	if err != nil {
		return fmt.Errorf("%w: resolve workspace role", contract.ErrUnavailable)
	}
	if !found {
		return contract.ErrForbidden
	}
	switch strings.TrimSpace(role) {
	case "owner", "admin":
		return nil
	case "member":
		if write {
			return contract.ErrForbidden
		}
		return nil
	default:
		return contract.ErrForbidden
	}
}

func (s *Service) newProvenance(value contract.Provenance) (domain.Provenance, error) {
	observedAt := value.ObservedAt
	if observedAt.IsZero() {
		observedAt = s.now().UTC()
	}
	result, err := domain.NewProvenance(value.SourceType, value.Locator, value.Revision, observedAt)
	if err != nil {
		return domain.Provenance{}, invalid(err)
	}
	return result, nil
}

func (s *Service) requireEngineeringEntityNode(ctx context.Context, workspaceID string, node domain.NodeRef) error {
	if node.Kind() != domain.NodeKindEngineeringEntity {
		return nil
	}
	_, err := s.repository.GetEntity(ctx, workspaceID, node.ID())
	return repositoryError(err)
}

func (s *Service) expectEntityAbsent(ctx context.Context, workspaceID, id string) error {
	_, err := s.repository.GetEntity(ctx, strings.TrimSpace(workspaceID), strings.TrimSpace(id))
	return absentResult(err)
}

func (s *Service) expectSourceBindingAbsent(ctx context.Context, workspaceID, id string) error {
	_, err := s.repository.GetSourceBinding(ctx, workspaceID, strings.TrimSpace(id))
	return absentResult(err)
}

func (s *Service) expectThreadEdgeAbsent(ctx context.Context, workspaceID, id string) error {
	_, err := s.repository.GetThreadEdge(ctx, workspaceID, strings.TrimSpace(id))
	return absentResult(err)
}

func (s *Service) expectChangeAbsent(ctx context.Context, workspaceID, id string) error {
	_, err := s.repository.GetChange(ctx, workspaceID, strings.TrimSpace(id))
	return absentResult(err)
}

func absentResult(err error) error {
	if errors.Is(err, domain.ErrNotFound) {
		return nil
	}
	if err == nil {
		return contract.ErrConflict
	}
	return repositoryError(err)
}

func invalid(err error) error {
	return fmt.Errorf("%w: %v", contract.ErrInvalidArgument, err)
}

func repositoryError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return contract.ErrNotFound
	case errors.Is(err, domain.ErrConflict):
		return contract.ErrConflict
	default:
		return fmt.Errorf("%w: %v", contract.ErrUnavailable, err)
	}
}

func toProvenance(value domain.Provenance) contract.Provenance {
	return contract.Provenance{SourceType: value.SourceType(), Locator: value.Locator(), Revision: value.Revision(), ObservedAt: value.ObservedAt()}
}

func toNodeRef(value domain.NodeRef) contract.NodeRef {
	return contract.NodeRef{Kind: string(value.Kind()), ID: value.ID()}
}

func toEntity(value domain.EngineeringEntity) contract.Entity {
	return contract.Entity{ID: value.ID(), WorkspaceID: value.WorkspaceID(), Type: string(value.Type()), Name: value.Name(), Status: string(value.Status()), OwnerRef: value.OwnerRef()}
}

func toSourceBinding(value domain.SourceBinding) contract.SourceBinding {
	return contract.SourceBinding{ID: value.ID(), WorkspaceID: value.WorkspaceID(), EntityID: value.EntityID(), Provenance: toProvenance(value.Provenance()), Authority: string(value.Authority())}
}

func toThreadEdge(value domain.ThreadEdge) contract.ThreadEdge {
	return contract.ThreadEdge{ID: value.ID(), WorkspaceID: value.WorkspaceID(), From: toNodeRef(value.From()), Relation: string(value.Relation()), To: toNodeRef(value.To()), Authority: string(value.Authority()), Provenance: toProvenance(value.Provenance())}
}

func toChange(value domain.Change) contract.Change {
	var workItem *contract.NodeRef
	if value.WorkItem() != nil {
		converted := toNodeRef(*value.WorkItem())
		workItem = &converted
	}
	artifacts := make([]contract.ArtifactRef, len(value.Artifacts()))
	for index, artifact := range value.Artifacts() {
		artifacts[index] = contract.ArtifactRef{Kind: artifact.Kind(), Locator: artifact.Locator(), Revision: artifact.Revision()}
	}
	return contract.Change{
		ID: value.ID(), WorkspaceID: value.WorkspaceID(), ProjectID: value.ProjectID(), RequirementID: value.RequirementID(),
		WorkItem: workItem, RunID: value.RunID(), Summary: value.Summary(), Status: string(value.Status()),
		AffectedEntityIDs: value.AffectedEntityIDs(), Artifacts: artifacts, Provenance: toProvenance(value.Provenance()),
		CreatedAt: value.CreatedAt(), UpdatedAt: value.UpdatedAt(), AcceptedAt: value.AcceptedAt(),
	}
}

func toContextPack(value domain.ContextPack) contract.ContextPack {
	references := make([]contract.ContextReference, len(value.References()))
	for index, reference := range value.References() {
		references[index] = contract.ContextReference{Kind: string(reference.Kind()), ID: reference.ID(), Revision: reference.Revision(), Checksum: reference.Checksum()}
	}
	return contract.ContextPack{
		ID: value.ID(), WorkspaceID: value.WorkspaceID(), WorkItem: toNodeRef(value.WorkItem()), WorkItemRevision: value.WorkItemRevision(),
		TargetEntityIDs: value.TargetEntityIDs(), References: references, PolicyVersion: value.PolicyVersion(), Checksum: value.Checksum(), CreatedAt: value.CreatedAt(),
	}
}

var _ contract.Service = (*Service)(nil)
