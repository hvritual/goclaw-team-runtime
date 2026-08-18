package contract

import "sort"

// Roadmap capability permissions are intentionally separate from broad roles.
// They name the frozen product actions before their providers are installed.
const (
	PermissionTaskRead                        = "workspace.task.read"
	PermissionTaskCreate                      = "workspace.task.create"
	PermissionTaskUpdateOwn                   = "workspace.task.update_own"
	PermissionTaskManageWorkspace             = "workspace.task.manage_workspace"
	PermissionSearchReadable                  = "workspace.search.readable"
	PermissionPinReorder                      = "workspace.pin.reorder"
	PermissionSkillReadPublished              = "workspace.skill.read_published"
	PermissionSkillCreate                     = "workspace.skill.create"
	PermissionSkillImport                     = "workspace.skill.import"
	PermissionSkillVersion                    = "workspace.skill.version"
	PermissionSkillArchive                    = "workspace.skill.archive"
	PermissionKnowledgeQuery                  = "workspace.knowledge.query"
	PermissionKnowledgePropose                = "workspace.knowledge.propose"
	PermissionKnowledgeReview                 = "workspace.knowledge.review"
	PermissionKnowledgeSelfReviewOverride     = "workspace.knowledge.self_review_override"
	PermissionResourceRead                    = "workspace.project.resource.read"
	PermissionResourceManage                  = "workspace.project.resource.manage"
	PermissionRequirementEditDraft            = "workspace.requirement.edit_draft"
	PermissionRequirementApproveFreeze        = "workspace.requirement.approve_freeze"
	PermissionRetrospectiveDraft              = "workspace.project.retrospective.draft"
	PermissionRetrospectivePublish            = "workspace.project.retrospective.publish"
	PermissionSimilarityCheck                 = "workspace.issue.similarity.check"
	PermissionDuplicateMarkOverride           = "workspace.issue.duplicate.mark_override"
	PermissionNotificationReadUpdateOwn       = "workspace.notification.read_update_own"
	PermissionReminderReplayRepair            = "workspace.notification.reminder.replay_repair"
	PermissionProjectPhaseTransition          = "workspace.project.phase.transition"
	PermissionProjectPhaseProtectedTransition = "workspace.project.phase.protected_transition"
	PermissionOutlineEditReorderLink          = "workspace.project.outline.edit_reorder_link"
)

type roadmapCapabilityPolicy struct {
	owner  bool
	admin  bool
	member bool
}

var roadmapCapabilityPolicies = map[string]roadmapCapabilityPolicy{
	PermissionTaskRead:                        {owner: true, admin: true, member: true},
	PermissionTaskCreate:                      {owner: true, admin: true, member: true},
	PermissionTaskUpdateOwn:                   {owner: true, admin: true, member: true},
	PermissionTaskManageWorkspace:             {owner: true, admin: true},
	PermissionSearchReadable:                  {owner: true, admin: true, member: true},
	PermissionPinReorder:                      {owner: true, admin: true, member: true},
	PermissionSkillReadPublished:              {owner: true, admin: true, member: true},
	PermissionSkillCreate:                     {owner: true, admin: true},
	PermissionSkillImport:                     {owner: true, admin: true},
	PermissionSkillVersion:                    {owner: true, admin: true},
	PermissionSkillArchive:                    {owner: true, admin: true},
	PermissionKnowledgeQuery:                  {owner: true, admin: true, member: true},
	PermissionKnowledgePropose:                {owner: true, admin: true, member: true},
	PermissionKnowledgeReview:                 {owner: true, admin: true},
	PermissionKnowledgeSelfReviewOverride:     {owner: true},
	PermissionResourceRead:                    {owner: true, admin: true, member: true},
	PermissionResourceManage:                  {owner: true, admin: true},
	PermissionRequirementEditDraft:            {owner: true, admin: true},
	PermissionRequirementApproveFreeze:        {owner: true},
	PermissionRetrospectiveDraft:              {owner: true, admin: true, member: true},
	PermissionRetrospectivePublish:            {owner: true, admin: true},
	PermissionSimilarityCheck:                 {owner: true, admin: true, member: true},
	PermissionDuplicateMarkOverride:           {owner: true, admin: true, member: true},
	PermissionNotificationReadUpdateOwn:       {owner: true, admin: true, member: true},
	PermissionReminderReplayRepair:            {owner: true},
	PermissionProjectPhaseTransition:          {owner: true, admin: true},
	PermissionProjectPhaseProtectedTransition: {owner: true},
	PermissionOutlineEditReorderLink:          {owner: true, admin: true},
}

// RoadmapCapabilityAllows returns only the frozen default role decision. It
// never grants agents implicitly, and unknown actors, roles, or actions deny.
// Runtime authorization must additionally require an installed provider.
func RoadmapCapabilityAllows(permission, actorType, role string) bool {
	if actorType != "member" {
		return false
	}
	policy, ok := roadmapCapabilityPolicies[permission]
	if !ok {
		return false
	}
	switch role {
	case "owner":
		return policy.owner
	case "admin":
		return policy.admin
	case "member":
		return policy.member
	default:
		return false
	}
}

// RoadmapCapabilityPermissions returns the complete, stable action catalog.
func RoadmapCapabilityPermissions() []string {
	permissions := make([]string, 0, len(roadmapCapabilityPolicies))
	for permission := range roadmapCapabilityPolicies {
		permissions = append(permissions, permission)
	}
	sort.Strings(permissions)
	return permissions
}

// RoadmapCapabilityProvider is the installation proof supplied by a completed
// delivery story. Merely naming or requesting a capability never enables it.
type RoadmapCapabilityProvider interface {
	RoadmapCapabilityInstalled(permission string) bool
}

// RoadmapFeatureProvider separates installed user-visible verticals from
// shared authorization actions. One readable-search grant must not imply that
// every search surface is installed.
type RoadmapFeatureProvider interface {
	RoadmapFeatureInstalled(feature string) bool
}

// RoadmapCapabilityInstalled remains false until the capability's delivery
// story injects a provider that explicitly reports the known permission.
func RoadmapCapabilityInstalled(permission string, providers ...RoadmapCapabilityProvider) bool {
	if _, known := roadmapCapabilityPolicies[permission]; !known || len(providers) != 1 || providers[0] == nil {
		return false
	}
	return providers[0].RoadmapCapabilityInstalled(permission)
}

func IsRoadmapCapabilityPermission(permission string) bool {
	_, ok := roadmapCapabilityPolicies[permission]
	return ok
}
