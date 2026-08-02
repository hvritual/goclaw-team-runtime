package contract

import (
	"context"
	"errors"
)

var ErrWorkspaceNotFound = errors.New("workspace not found")

// WorkspaceIdentity is the stable cross-module projection of a Workspace.
type WorkspaceIdentity struct {
	ID   string
	Name string
}

// WorkspaceIdentityReader exposes Workspace-owned identity data without
// leaking Workspace persistence or transport details to consumers.
type WorkspaceIdentityReader interface {
	FindIdentity(context.Context, string) (WorkspaceIdentity, error)
}
