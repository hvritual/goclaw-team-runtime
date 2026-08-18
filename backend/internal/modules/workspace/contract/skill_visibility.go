package contract

import "context"

type BindInitialSkillRequest struct {
	WorkspaceID string
	SkillID     string
	VersionID   string
}

type SkillBindingExecutor interface {
	Execute(context.Context, string, ...any) error
}

type SkillVisibilityReference struct {
	WorkspaceID string
	SkillID     string
	VersionID   string
	Enabled     bool
	AgentIDs    []string
}

type SkillVisibilityService interface {
	AuthorizeInitialSkill(context.Context, BindInitialSkillRequest) error
	BindInitialSkill(context.Context, SkillBindingExecutor, BindInitialSkillRequest) error
	ResolveSkill(context.Context, string, string) (SkillVisibilityReference, error)
	ListSkills(context.Context, string) ([]SkillVisibilityReference, error)
}
