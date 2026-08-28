package contract

import "context"

type Service interface {
	CreateEntity(ctx context.Context, actor Actor, workspaceID string, request CreateEntityRequest) (Entity, error)
	GetEntity(ctx context.Context, actor Actor, workspaceID, id string) (Entity, error)
	ListEntities(ctx context.Context, actor Actor, workspaceID string) ([]Entity, error)
	UpdateEntity(ctx context.Context, actor Actor, workspaceID, id string, request UpdateEntityRequest) (Entity, error)

	CreateSourceBinding(ctx context.Context, actor Actor, workspaceID string, request CreateSourceBindingRequest) (SourceBinding, error)
	GetSourceBinding(ctx context.Context, actor Actor, workspaceID, id string) (SourceBinding, error)
	ListSourceBindings(ctx context.Context, actor Actor, workspaceID, entityID string) ([]SourceBinding, error)

	CreateThreadEdge(ctx context.Context, actor Actor, workspaceID string, request CreateThreadEdgeRequest) (ThreadEdge, error)
	ListThreadEdges(ctx context.Context, actor Actor, workspaceID string, node NodeRef) ([]ThreadEdge, error)

	CreateChange(ctx context.Context, actor Actor, workspaceID string, request CreateChangeRequest) (Change, error)
	GetChange(ctx context.Context, actor Actor, workspaceID, id string) (Change, error)
	ListChanges(ctx context.Context, actor Actor, workspaceID, affectedEntityID string) ([]Change, error)

	GetContextPack(ctx context.Context, actor Actor, workspaceID, id string) (ContextPack, error)
}
