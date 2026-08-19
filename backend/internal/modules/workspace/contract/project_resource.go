package contract

import "context"

type ProjectResourceRef struct {
	URL string `json:"url"`
	Ref string `json:"ref,omitempty"`
}

type ProjectResourceConnection struct {
	State          string `json:"state"`
	DiagnosticCode string `json:"diagnostic_code,omitempty"`
	CheckedAt      string `json:"checked_at,omitempty"`
}

type ProjectResource struct {
	ID           string                    `json:"id"`
	WorkspaceID  string                    `json:"workspace_id"`
	ProjectID    string                    `json:"project_id"`
	ResourceType string                    `json:"resource_type"`
	ResourceRef  ProjectResourceRef        `json:"resource_ref"`
	Label        string                    `json:"label,omitempty"`
	Position     int                       `json:"position"`
	Status       string                    `json:"status"`
	Revision     int64                     `json:"revision"`
	Connection   ProjectResourceConnection `json:"connection"`
	CreatedAt    string                    `json:"created_at"`
	CreatedBy    string                    `json:"created_by"`
	UpdatedAt    string                    `json:"updated_at"`
	UpdatedBy    string                    `json:"updated_by"`
	ArchivedAt   string                    `json:"archived_at,omitempty"`
	ArchivedBy   string                    `json:"archived_by,omitempty"`
}

type ProjectResourceList struct {
	Resources []ProjectResource `json:"resources"`
	Total     int               `json:"total"`
	Revision  int64             `json:"revision"`
}

type CreateProjectResourceRequest struct {
	ResourceType string             `json:"resource_type"`
	ResourceRef  ProjectResourceRef `json:"resource_ref"`
	Label        string             `json:"label,omitempty"`
}

type UpdateProjectResourceRequest struct {
	Action           string              `json:"action"`
	ExpectedRevision int64               `json:"expected_revision"`
	ResourceRef      *ProjectResourceRef `json:"resource_ref,omitempty"`
	Label            *string             `json:"label,omitempty"`
	BeforeResourceID *string             `json:"before_resource_id,omitempty"`
}

type ProjectResourceConnectionRequest struct {
	WorkspaceID  string
	ProjectID    string
	ResourceID   string
	ResourceType string
	ResourceRef  ProjectResourceRef
}

type ProjectResourceService interface {
	ListProjectResources(context.Context, string, string, bool) (ProjectResourceList, error)
	CreateProjectResource(context.Context, string, string, string, CreateProjectResourceRequest) (ProjectResource, error)
	UpdateProjectResource(context.Context, string, string, string, UpdateProjectResourceRequest) (ProjectResource, error)
	ArchiveProjectResource(context.Context, string, string, string, int64) error
}
