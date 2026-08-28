package domain

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("engineering thread record not found")

type EntityRepository interface {
	PutEntity(context.Context, EngineeringEntity) error
	GetEntity(context.Context, string, string) (EngineeringEntity, error)
	ListEntities(context.Context, string) ([]EngineeringEntity, error)
}

type SourceBindingRepository interface {
	PutSourceBinding(context.Context, SourceBinding) error
	GetSourceBinding(context.Context, string, string) (SourceBinding, error)
	ListSourceBindings(context.Context, string, string) ([]SourceBinding, error)
}

type ThreadEdgeRepository interface {
	PutThreadEdge(context.Context, ThreadEdge) error
	GetThreadEdge(context.Context, string, string) (ThreadEdge, error)
	ListThreadEdges(context.Context, string, NodeRef) ([]ThreadEdge, error)
}

type ChangeRepository interface {
	PutChange(context.Context, Change) error
	GetChange(context.Context, string, string) (Change, error)
	ListChanges(context.Context, string, string) ([]Change, error)
}

type ContextPackRepository interface {
	PutContextPack(context.Context, ContextPack) error
	GetContextPack(context.Context, string, string) (ContextPack, error)
}

type Repository interface {
	EntityRepository
	SourceBindingRepository
	ThreadEdgeRepository
	ChangeRepository
	ContextPackRepository
}
