package contract

import "context"

type WorkLinkKind string

const (
	WorkLinkProject     WorkLinkKind = "project"
	WorkLinkRequirement WorkLinkKind = "requirement"
	WorkLinkTask        WorkLinkKind = "task"
)

type WorkLink struct {
	ID          string       `json:"id"`
	WorkspaceID string       `json:"workspace_id"`
	WorkKind    WorkLinkKind `json:"work_kind"`
	WorkID      string       `json:"work_id"`
	EntityID    string       `json:"entity_id"`
	Relation    string       `json:"relation"`
	Authority   string       `json:"authority"`
	Provenance  Provenance   `json:"provenance"`
}

type PutWorkLinkRequest struct {
	WorkspaceID string
	WorkKind    WorkLinkKind
	WorkID      string
	EntityID    string
}

type ListWorkLinksRequest struct {
	WorkspaceID string
	WorkKind    WorkLinkKind
	WorkID      string
}

type DeleteWorkLinkRequest struct {
	WorkspaceID string
	WorkKind    WorkLinkKind
	WorkID      string
	EntityID    string
}

// WorkLinkProvider is an internal bounded-context capability. It owns
// canonical Engineering Thread edges but does not authorize Workspace users;
// the consumer must authorize its own work-plane mutation before calling it.
type WorkLinkProvider interface {
	PutWorkLink(context.Context, PutWorkLinkRequest) (WorkLink, error)
	ListWorkLinks(context.Context, ListWorkLinksRequest) ([]WorkLink, error)
	DeleteWorkLink(context.Context, DeleteWorkLinkRequest) error
}
