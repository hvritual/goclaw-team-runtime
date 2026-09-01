package domain

import (
	"context"
	"errors"
)

var (
	ErrNotFound = errors.New("engineering thread record not found")
	ErrConflict = errors.New("engineering thread record conflict")
)

type EntityRepository interface {
	PutEntity(ctx context.Context, entity EngineeringEntity) error
	GetEntity(ctx context.Context, workspaceID, id string) (EngineeringEntity, error)
	ListEntities(ctx context.Context, workspaceID string) ([]EngineeringEntity, error)
}

type SourceBindingRepository interface {
	PutSourceBinding(ctx context.Context, binding SourceBinding) error
	GetSourceBinding(ctx context.Context, workspaceID, id string) (SourceBinding, error)
	ListSourceBindings(ctx context.Context, workspaceID, entityID string) ([]SourceBinding, error)
}

type ThreadEdgeRepository interface {
	PutThreadEdge(ctx context.Context, edge ThreadEdge) error
	GetThreadEdge(ctx context.Context, workspaceID, id string) (ThreadEdge, error)
	ListThreadEdges(ctx context.Context, workspaceID string, node NodeRef) ([]ThreadEdge, error)
}

type ChangeRepository interface {
	PutChange(ctx context.Context, change Change) error
	GetChange(ctx context.Context, workspaceID, id string) (Change, error)
	ListChanges(ctx context.Context, workspaceID, affectedEntityID string) ([]Change, error)
}

type ContextPackRepository interface {
	PutContextPack(ctx context.Context, pack ContextPack) error
	GetContextPack(ctx context.Context, workspaceID, id string) (ContextPack, error)
}

type EvidenceRepository interface {
	PutEvidence(ctx context.Context, evidence EvidenceEnvelope) error
	GetEvidence(ctx context.Context, workspaceID, id string) (EvidenceEnvelope, error)
	ListEvidence(ctx context.Context, workspaceID string, subject *NodeRef) ([]EvidenceEnvelope, error)
}

type Repository interface {
	EntityRepository
	SourceBindingRepository
	ThreadEdgeRepository
	ChangeRepository
	ContextPackRepository
	EvidenceRepository
}
