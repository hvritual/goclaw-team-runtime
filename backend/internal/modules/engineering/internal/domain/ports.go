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

type ExecutionItemRepository interface {
	CreateExecutionItem(ctx context.Context, item ExecutionItem) error
	GetExecutionItem(ctx context.Context, workspaceID, id string) (ExecutionItem, error)
}

type EvidenceRepository interface {
	CreateEvidence(ctx context.Context, evidence EvidenceEnvelope) error
	GetEvidence(ctx context.Context, workspaceID, id string) (EvidenceEnvelope, error)
}

type EvidenceAttachmentRepository interface {
	AttachEvidence(ctx context.Context, attachment EvidenceAttachment) error
	ListEvidenceAttachments(ctx context.Context, workspaceID, executionItemID string) ([]EvidenceAttachment, error)
}

// ExecutionEvidenceRepository is deliberately separate from Repository so
// existing Engineering application services do not acquire unused Phase 3
// storage responsibilities. P3-S03 can depend on this narrower port directly.
type ExecutionEvidenceRepository interface {
	ExecutionItemRepository
	EvidenceRepository
	EvidenceAttachmentRepository
}

type Repository interface {
	EntityRepository
	SourceBindingRepository
	ThreadEdgeRepository
	ChangeRepository
	ContextPackRepository
}
