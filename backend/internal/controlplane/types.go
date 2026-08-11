package controlplane

import "time"

type WorkspaceState string

const (
	WorkspaceActive   WorkspaceState = "active"
	WorkspaceArchived WorkspaceState = "archived"
)

type ActorKind string

const (
	ActorHuman ActorKind = "human"
	ActorAgent ActorKind = "agent"
)

type Role string

const (
	RoleOwner    Role = "owner"
	RoleAdmin    Role = "admin"
	RoleMember   Role = "member"
	RoleReviewer Role = "reviewer"
	RoleViewer   Role = "viewer"
)

type MemberState string

const (
	MemberActive  MemberState = "active"
	MemberRemoved MemberState = "removed"
)

type Actor struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
	Kind        ActorKind `json:"kind"`
}

type Workspace struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	State     WorkspaceState `json:"state"`
	Version   int64          `json:"version"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type Member struct {
	WorkspaceID string      `json:"workspace_id"`
	ID          string      `json:"id"`
	Kind        ActorKind   `json:"kind"`
	Role        Role        `json:"role"`
	State       MemberState `json:"state"`
	Version     int64       `json:"version"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// Record is the persistence envelope for versioned domain aggregates.
type Record struct {
	WorkspaceID string    `json:"workspace_id"`
	Kind        string    `json:"kind"`
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id,omitempty"`
	State       string    `json:"state"`
	Version     int64     `json:"version"`
	Payload     []byte    `json:"payload"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type AuditEntry struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
	ActorID     string    `json:"actor_id"`
	Action      string    `json:"action"`
	Resource    string    `json:"resource"`
	ResourceID  string    `json:"resource_id"`
	Metadata    []byte    `json:"metadata,omitempty"`
	OccurredAt  time.Time `json:"occurred_at"`
}

type Page struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

func (p Page) normalized() Page {
	if p.Limit <= 0 {
		p.Limit = 50
	}
	if p.Limit > 200 {
		p.Limit = 200
	}
	if p.Offset < 0 {
		p.Offset = 0
	}
	return p
}
