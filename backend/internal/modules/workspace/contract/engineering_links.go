package contract

import (
	"context"
	"errors"
)

var (
	ErrEngineeringWorkLinkInvalid        = errors.New("invalid engineering work link")
	ErrEngineeringWorkNotFound           = errors.New("workspace work item not found")
	ErrEngineeringEntityReferenceMissing = errors.New("engineering entity not found")
	ErrEngineeringEntityArchived         = errors.New("engineering entity is archived")
	ErrEngineeringWorkLinkNotFound       = errors.New("engineering work link not found")
	ErrEngineeringWorkLinkUnavailable    = errors.New("engineering work link capability unavailable")
)

type EngineeringWorkKind string

const (
	EngineeringWorkProject     EngineeringWorkKind = "project"
	EngineeringWorkRequirement EngineeringWorkKind = "requirement"
	EngineeringWorkTask        EngineeringWorkKind = "task"
)

type EngineeringWorkLink struct {
	ID          string              `json:"id"`
	WorkspaceID string              `json:"workspace_id"`
	WorkKind    EngineeringWorkKind `json:"work_kind"`
	WorkID      string              `json:"work_id"`
	EntityID    string              `json:"entity_id"`
	Relation    string              `json:"relation"`
	Authority   string              `json:"authority"`
	Source      string              `json:"source"`
	Locator     string              `json:"locator"`
}

type EngineeringLinkGateway interface {
	PutEngineeringWorkLink(context.Context, string, EngineeringWorkKind, string, string) (EngineeringWorkLink, error)
	ListEngineeringWorkLinks(context.Context, string, EngineeringWorkKind, string) ([]EngineeringWorkLink, error)
	DeleteEngineeringWorkLink(context.Context, string, EngineeringWorkKind, string, string) error
}

type WorkEngineeringLinkService interface {
	LinkEngineeringEntity(context.Context, string, EngineeringWorkKind, string, string) (EngineeringWorkLink, error)
	ListEngineeringLinks(context.Context, string, EngineeringWorkKind, string) ([]EngineeringWorkLink, error)
	UnlinkEngineeringEntity(context.Context, string, EngineeringWorkKind, string, string) error
}
