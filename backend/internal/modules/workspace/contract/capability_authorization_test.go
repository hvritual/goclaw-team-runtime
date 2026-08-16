package contract

import (
	"strings"
	"testing"
)

func TestRoadmapCapabilityAuthorizationDefaults(t *testing.T) {
	tests := []struct {
		permission string
		roles      string
	}{
		{PermissionTaskRead, "owner,admin,member"},
		{PermissionTaskCreate, "owner,admin,member"},
		{PermissionTaskUpdateOwn, "owner,admin,member"},
		{PermissionTaskManageWorkspace, "owner,admin"},
		{PermissionSearchReadable, "owner,admin,member"},
		{PermissionPinReorder, "owner,admin,member"},
		{PermissionSkillReadPublished, "owner,admin,member"},
		{PermissionSkillCreate, "owner,admin"},
		{PermissionSkillImport, "owner,admin"},
		{PermissionSkillVersion, "owner,admin"},
		{PermissionSkillArchive, "owner,admin"},
		{PermissionKnowledgeQuery, "owner,admin,member"},
		{PermissionKnowledgePropose, "owner,admin,member"},
		{PermissionKnowledgeReview, "owner,admin"},
		{PermissionKnowledgeSelfReviewOverride, "owner"},
		{PermissionResourceRead, "owner,admin,member"},
		{PermissionResourceManage, "owner,admin"},
		{PermissionRequirementEditDraft, "owner,admin"},
		{PermissionRequirementApproveFreeze, "owner"},
		{PermissionRetrospectiveDraft, "owner,admin,member"},
		{PermissionRetrospectivePublish, "owner,admin"},
		{PermissionSimilarityCheck, "owner,admin,member"},
		{PermissionDuplicateMarkOverride, "owner,admin,member"},
		{PermissionNotificationReadUpdateOwn, "owner,admin,member"},
		{PermissionReminderReplayRepair, "owner"},
		{PermissionProjectPhaseTransition, "owner,admin"},
		{PermissionProjectPhaseProtectedTransition, "owner"},
		{PermissionOutlineEditReorderLink, "owner,admin"},
	}
	for _, test := range tests {
		t.Run(test.permission, func(t *testing.T) {
			for _, role := range []string{"owner", "admin", "member"} {
				want := roleListed(test.roles, role)
				if got := RoadmapCapabilityAllows(test.permission, "member", role); got != want {
					t.Errorf("member role %q = %v, want %v", role, got, want)
				}
			}
			if RoadmapCapabilityAllows(test.permission, "agent", "owner") {
				t.Error("agent received an implicit grant")
			}
		})
	}
	for _, test := range []struct{ permission, actorType, role string }{
		{PermissionTaskRead, "service", "owner"},
		{PermissionTaskRead, "member", "operator"},
		{"workspace.roadmap.unknown", "member", "owner"},
	} {
		if RoadmapCapabilityAllows(test.permission, test.actorType, test.role) {
			t.Errorf("unknown matrix input (%q, %q, %q) was allowed", test.permission, test.actorType, test.role)
		}
	}
}

func roleListed(roles, target string) bool {
	for _, role := range strings.Split(roles, ",") {
		if role == target {
			return true
		}
	}
	return false
}

func TestRoadmapCapabilityPermissionsAreNamedAndUnavailableUntilInstalled(t *testing.T) {
	permissions := RoadmapCapabilityPermissions()
	if len(permissions) == 0 {
		t.Fatal("RoadmapCapabilityPermissions() returned no permissions")
	}
	seen := make(map[string]struct{}, len(permissions))
	for _, permission := range permissions {
		if permission == "" {
			t.Fatal("catalog contains an empty permission")
		}
		if _, duplicate := seen[permission]; duplicate {
			t.Fatalf("catalog contains duplicate permission %q", permission)
		}
		seen[permission] = struct{}{}
		if RoadmapCapabilityInstalled(permission) {
			t.Errorf("permission %q is installed before its delivery story", permission)
		}
	}
	for _, permission := range []string{
		PermissionTaskRead,
		PermissionTaskCreate,
		PermissionTaskUpdateOwn,
		PermissionTaskManageWorkspace,
		PermissionSearchReadable,
		PermissionPinReorder,
		PermissionSkillReadPublished,
		PermissionSkillCreate,
		PermissionSkillImport,
		PermissionSkillVersion,
		PermissionSkillArchive,
		PermissionKnowledgeQuery,
		PermissionKnowledgePropose,
		PermissionKnowledgeReview,
		PermissionKnowledgeSelfReviewOverride,
		PermissionResourceRead,
		PermissionResourceManage,
		PermissionRequirementEditDraft,
		PermissionRequirementApproveFreeze,
		PermissionRetrospectiveDraft,
		PermissionRetrospectivePublish,
		PermissionSimilarityCheck,
		PermissionDuplicateMarkOverride,
		PermissionNotificationReadUpdateOwn,
		PermissionReminderReplayRepair,
		PermissionProjectPhaseTransition,
		PermissionProjectPhaseProtectedTransition,
		PermissionOutlineEditReorderLink,
	} {
		if _, ok := seen[permission]; !ok {
			t.Errorf("named permission %q is absent from the catalog", permission)
		}
	}
	provider := roadmapCapabilityProviderStub{PermissionTaskRead: true}
	if !RoadmapCapabilityInstalled(PermissionTaskRead, provider) {
		t.Error("injected provider did not install its named capability")
	}
	if RoadmapCapabilityInstalled(PermissionSkillImport, provider) {
		t.Error("provider installed an unreported capability")
	}
	if RoadmapCapabilityInstalled("workspace.roadmap.unknown", provider) {
		t.Error("provider installed an unknown capability")
	}
}

type roadmapCapabilityProviderStub map[string]bool

func (s roadmapCapabilityProviderStub) RoadmapCapabilityInstalled(permission string) bool {
	return s[permission]
}
