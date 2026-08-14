package contract

import "context"

type CreateWorkspaceRequest struct {
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Description *string `json:"description,omitempty"`
	Context     *string `json:"context,omitempty"`
}

type WorkspaceCreationService interface {
	Create(context.Context, string, CreateWorkspaceRequest) (WorkspaceSelection, error)
}
