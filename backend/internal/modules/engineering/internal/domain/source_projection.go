package domain

import (
	"context"
	"errors"
	"strings"
)

var (
	ErrSourceProjectionInvalid = errors.New("invalid engineering source projection")
)

type SourceProjection struct {
	entity        EngineeringEntity
	binding       SourceBinding
	upsertEdges   []ThreadEdge
	deleteEdgeIDs []string
}

func NewSourceProjection(entity EngineeringEntity, binding SourceBinding, upsertEdges []ThreadEdge, deleteEdgeIDs []string) (SourceProjection, error) {
	workspaceID := entity.WorkspaceID()
	if workspaceID == "" || binding.WorkspaceID() != workspaceID || binding.EntityID() != entity.ID() {
		return SourceProjection{}, ErrSourceProjectionInvalid
	}
	upsertIDs := make(map[string]struct{}, len(upsertEdges))
	for _, edge := range upsertEdges {
		if edge.WorkspaceID() != workspaceID || edge.From().Kind() != NodeKindEngineeringEntity || edge.From().ID() != entity.ID() {
			return SourceProjection{}, ErrSourceProjectionInvalid
		}
		if _, exists := upsertIDs[edge.ID()]; exists {
			return SourceProjection{}, ErrSourceProjectionInvalid
		}
		upsertIDs[edge.ID()] = struct{}{}
	}
	deleteIDs := make(map[string]struct{}, len(deleteEdgeIDs))
	cleanDeletes := make([]string, 0, len(deleteEdgeIDs))
	for _, raw := range deleteEdgeIDs {
		id := strings.TrimSpace(raw)
		if id == "" {
			return SourceProjection{}, ErrSourceProjectionInvalid
		}
		if _, upserting := upsertIDs[id]; upserting {
			return SourceProjection{}, ErrSourceProjectionInvalid
		}
		if _, exists := deleteIDs[id]; exists {
			continue
		}
		deleteIDs[id] = struct{}{}
		cleanDeletes = append(cleanDeletes, id)
	}
	return SourceProjection{
		entity: entity, binding: binding,
		upsertEdges: append([]ThreadEdge(nil), upsertEdges...),
		deleteEdgeIDs: cleanDeletes,
	}, nil
}

func (value SourceProjection) Entity() EngineeringEntity { return value.entity }
func (value SourceProjection) Binding() SourceBinding    { return value.binding }
func (value SourceProjection) UpsertEdges() []ThreadEdge {
	return append([]ThreadEdge(nil), value.upsertEdges...)
}
func (value SourceProjection) DeleteEdgeIDs() []string {
	return append([]string(nil), value.deleteEdgeIDs...)
}

type SourceProjectionRepository interface {
	ApplySourceProjection(context.Context, SourceProjection) error
}
