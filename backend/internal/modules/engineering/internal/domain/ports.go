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
	PutEntity(context.Context, EngineeringEntity) error
	GetEntity(context.Context, workspaceID, id string) (EngineeringEntity, error)
	ListEntities(context.Context, workspaceID string) ([]EngineeringEntity, error)
}

type SourceBindingRepository interface {
	PutSourceBinding(context.Context, SourceBinding) error
	GetSourceBinding(context.Context, workspaceID, id string) (SourceBinding, error)
	ListSourceBindings(context.Context, workspaceID, entityID string) ([]SourceBinding, error)
}

type ThreadEdgeRepository interface {
	PutThreadEdge(context.Context, ThreadEdge) error
	GetThreadEdge(context.Context, workspaceID, id string) (ThreadEdge, error)
	ListThreadEdges(context.Context, workspaceID string, node NodeRef) ([]ThreadEdge, error)
}

type ChangeRepository interface {
	PutChange(context.Context, Change) error
	GetChange(context.Context, workspaceID, id string) (Change, error)
	ListChanges(context.Context, workspaceID, affectedEntityID string) ([]Change, error)
}

type ContextPackRepository interface {
	PutContextPack(context.Context, ContextPack) error
	GetContextPack(context.Context, workspaceID, id string) (ContextPack, error)
}

type Repository interface {
	EntityRepository
	SourceBindingRepository
	ThreadEdgeRepository
	ChangeRepository
	ContextPackRepository
}
