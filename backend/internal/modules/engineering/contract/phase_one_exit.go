package contract

import "context"

type FreezeContextPackRequest struct {
	ID               string             `json:"id"`
	WorkItem         NodeRef            `json:"work_item"`
	WorkItemRevision string             `json:"work_item_revision"`
	TargetEntityIDs  []string           `json:"target_entity_ids"`
	References       []ContextReference `json:"references,omitempty"`
	PolicyVersion    string             `json:"policy_version"`
}

type LifecycleService interface {
	AcceptChange(ctx context.Context, actor Actor, workspaceID, id string) (Change, error)
	FreezeContextPack(ctx context.Context, actor Actor, workspaceID string, request FreezeContextPackRequest) (ContextPack, error)
}
