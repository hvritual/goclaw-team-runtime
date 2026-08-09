package relationship

import (
	"errors"
	"strings"
)

const (
	ActorTypeMember = "member"
	ActorTypeAgent  = "agent"
	RoleLead        = "lead"
	RoleMember      = "member"
	RoleAgent       = "agent"
)

var (
	ErrWorkspaceRequired = errors.New("workspace id is required")
	ErrProjectRequired   = errors.New("project id is required")
	ErrActorTypeInvalid  = errors.New("invalid actor type")
	ErrActorIDRequired   = errors.New("actor id is required")
	ErrRoleInvalid       = errors.New("invalid project actor role")
)

type Relation struct {
	workspaceID string
	projectID   string
	actorType   string
	actorID     string
	role        string
}

func New(workspaceID, projectID, actorType, actorID, role string) (Relation, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	projectID = strings.TrimSpace(projectID)
	actorType = strings.TrimSpace(actorType)
	actorID = strings.TrimSpace(actorID)
	role = strings.TrimSpace(role)
	if workspaceID == "" {
		return Relation{}, ErrWorkspaceRequired
	}
	if projectID == "" {
		return Relation{}, ErrProjectRequired
	}
	if actorID == "" {
		return Relation{}, ErrActorIDRequired
	}
	switch actorType {
	case ActorTypeMember:
		if role != RoleLead && role != RoleMember {
			return Relation{}, ErrRoleInvalid
		}
	case ActorTypeAgent:
		if role != RoleAgent {
			return Relation{}, ErrRoleInvalid
		}
	default:
		return Relation{}, ErrActorTypeInvalid
	}
	return Relation{workspaceID: workspaceID, projectID: projectID, actorType: actorType, actorID: actorID, role: role}, nil
}

func ValidateReference(workspaceID, projectID, actorType, actorID string) error {
	workspaceID = strings.TrimSpace(workspaceID)
	projectID = strings.TrimSpace(projectID)
	actorType = strings.TrimSpace(actorType)
	actorID = strings.TrimSpace(actorID)
	if workspaceID == "" {
		return ErrWorkspaceRequired
	}
	if projectID == "" {
		return ErrProjectRequired
	}
	if actorID == "" {
		return ErrActorIDRequired
	}
	if actorType != ActorTypeMember && actorType != ActorTypeAgent {
		return ErrActorTypeInvalid
	}
	return nil
}

func (r Relation) WorkspaceID() string { return r.workspaceID }
func (r Relation) ProjectID() string   { return r.projectID }
func (r Relation) ActorType() string   { return r.actorType }
func (r Relation) ActorID() string     { return r.actorID }
func (r Relation) Role() string        { return r.role }
