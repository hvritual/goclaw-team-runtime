package contract

import "context"

type BindInitialSkillRequest struct {
	WorkspaceID string
	SkillID     string
	VersionID   string
}

type SkillVisibilityReference struct {
	WorkspaceID string
	SkillID     string
	VersionID   string
	Enabled     bool
	AgentIDs    []string
}

type SkillVisibilityService interface {
	BindInitialSkill(context.Context, BindInitialSkillRequest) error
	ResolveSkill(context.Context, string, string) (SkillVisibilityReference, error)
	ListSkills(context.Context, string) ([]SkillVisibilityReference, error)
}
