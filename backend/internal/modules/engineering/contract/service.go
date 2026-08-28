package contract

import (
	"context"
	"time"
)

type Actor struct {
	UserID string
}

type WorkspaceRoleResolver interface {
	ResolveWorkspaceRole(ctx context.Context, userID, workspaceID string) (role string, found bool, err error)
}

type WorkspaceRoleResolverFunc func(ctx context.Context, userID, workspaceID string) (role string, found bool, err error)

func (f WorkspaceRoleResolverFunc) ResolveWorkspaceRole(ctx context.Context, userID, workspaceID string) (string, bool, error) {
	return f(ctx, userID, workspaceID)
}

type Provenance struct {
	SourceType string    `json:"source_type"`
	Locator    string    `json:"locator"`
	Revision   string    `json:"revision,omitempty"`
	ObservedAt time.Time `json:"observed_at"`
}

type NodeRef struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

type Entity struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspace_id"`
	Type        string `json:"type"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	OwnerRef    string `json:"owner_ref,omitempty"`
}

type CreateEntityRequest struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Status   string `json:"status,omitempty"`
	OwnerRef string `json:"owner_ref,omitempty"`
}

type UpdateEntityRequest struct {
	Name     *string `json:"name,omitempty"`
	Status   *string `json:"status,omitempty"`
	OwnerRef *string `json:"owner_ref,omitempty"`
}

type SourceBinding struct {
	ID          string     `json:"id"`
	WorkspaceID string     `json:"workspace_id"`
	EntityID    string     `json:"entity_id"`
	Provenance  Provenance `json:"provenance"`
	Authority   string     `json:"authority"`
}

type CreateSourceBindingRequest struct {
	ID         string     `json:"id"`
	EntityID   string     `json:"entity_id"`
	Provenance Provenance `json:"provenance"`
	Authority  string     `json:"authority"`
}

type ThreadEdge struct {
	ID          string     `json:"id"`
	WorkspaceID string     `json:"workspace_id"`
	From        NodeRef    `json:"from"`
	Relation    string     `json:"relation"`
	To          NodeRef    `json:"to"`
	Authority   string     `json:"authority"`
	Provenance  Provenance `json:"provenance"`
}

type CreateThreadEdgeRequest struct {
	ID         string     `json:"id"`
	From       NodeRef    `json:"from"`
	Relation   string     `json:"relation"`
	To         NodeRef    `json:"to"`
	Authority  string     `json:"authority"`
	Provenance Provenance `json:"provenance"`
}

type ArtifactRef struct {
	Kind     string `json:"kind"`
	Locator  string `json:"locator"`
	Revision string `json:"revision,omitempty"`
}

type Change struct {
	ID                string        `json:"id"`
	WorkspaceID       string        `json:"workspace_id"`
	ProjectID         string        `json:"project_id,omitempty"`
	RequirementID     string        `json:"requirement_id,omitempty"`
	WorkItem          *NodeRef      `json:"work_item,omitempty"`
	RunID             string        `json:"run_id,omitempty"`
	Summary           string        `json:"summary"`
	Status            string        `json:"status"`
	AffectedEntityIDs []string      `json:"affected_entity_ids"`
	Artifacts         []ArtifactRef `json:"artifacts,omitempty"`
	Provenance        Provenance    `json:"provenance"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`
	AcceptedAt        *time.Time    `json:"accepted_at,omitempty"`
}

type CreateChangeRequest struct {
	ID                string        `json:"id"`
	ProjectID         string        `json:"project_id,omitempty"`
	RequirementID     string        `json:"requirement_id,omitempty"`
	WorkItem          *NodeRef      `json:"work_item,omitempty"`
	RunID             string        `json:"run_id,omitempty"`
	Summary           string        `json:"summary"`
	AffectedEntityIDs []string      `json:"affected_entity_ids"`
	Artifacts         []ArtifactRef `json:"artifacts,omitempty"`
	Provenance        Provenance    `json:"provenance"`
}

type ContextReference struct {
	Kind     string `json:"kind"`
	ID       string `json:"id"`
	Revision string `json:"revision"`
	Checksum string `json:"checksum"`
}

type ContextPack struct {
	ID               string             `json:"id"`
	WorkspaceID      string             `json:"workspace_id"`
	WorkItem         NodeRef            `json:"work_item"`
	WorkItemRevision string             `json:"work_item_revision"`
	TargetEntityIDs  []string           `json:"target_entity_ids"`
	References       []ContextReference `json:"references"`
	PolicyVersion    string             `json:"policy_version"`
	Checksum         string             `json:"checksum"`
	CreatedAt        time.Time          `json:"created_at"`
}
