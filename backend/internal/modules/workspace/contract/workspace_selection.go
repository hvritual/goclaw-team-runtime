package contract

import "context"

type WorkspaceMembership struct {
	MemberID    string
	UserID      string
	WorkspaceID string
	Role        string
}

type WorkspaceMembershipReader interface {
	ListForUser(context.Context, string) ([]WorkspaceMembership, error)
	FindForUserAndWorkspace(context.Context, string, string) (WorkspaceMembership, bool, error)
	FindByMemberAndWorkspace(context.Context, string, string) (WorkspaceMembership, bool, error)
}

type WorkspaceRepo struct {
	URL         string  `json:"url"`
	Description *string `json:"description,omitempty"`
}

type WorkspaceSelection struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Slug        string          `json:"slug"`
	Description *string         `json:"description"`
	Context     *string         `json:"context"`
	Settings    map[string]any  `json:"settings"`
	Repos       []WorkspaceRepo `json:"repos"`
	IssuePrefix string          `json:"issue_prefix"`
	AvatarURL   *string         `json:"avatar_url"`
	CreatedAt   string          `json:"created_at"`
	UpdatedAt   string          `json:"updated_at"`
}

type WorkspaceSelectionService interface {
	List(context.Context, string) ([]WorkspaceSelection, error)
	ResolveSlug(context.Context, string, string) (string, error)
	MembershipForID(context.Context, string, string) (WorkspaceMembership, error)
}
