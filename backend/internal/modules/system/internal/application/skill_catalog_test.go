package application

import (
	"context"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/system/contract"
)

type visibilityRepositoryStub struct {
	entries map[string]contract.SkillCatalogEntry
}

func (r visibilityRepositoryStub) Create(context.Context, contract.CreateSkillCatalogRequest, string, string, time.Time, contract.SkillCreateBinding) (contract.SkillCatalogEntry, error) {
	panic("not used")
}
func (r visibilityRepositoryStub) CreateVersion(context.Context, contract.SkillIdentity, string, contract.UpdateSkillCatalogRequest, string, time.Time) (contract.SkillCatalogEntry, error) {
	panic("not used")
}
func (r visibilityRepositoryStub) TransitionVersion(context.Context, contract.SkillIdentity, string, string, string, int64, time.Time) (contract.SkillCatalogEntry, error) {
	panic("not used")
}
func (r visibilityRepositoryStub) Archive(context.Context, contract.SkillIdentity, string, int64, time.Time) error {
	panic("not used")
}
func (r visibilityRepositoryStub) Restore(context.Context, contract.SkillIdentity, string, int64, time.Time) (contract.SkillCatalogEntry, error) {
	panic("not used")
}
func (r visibilityRepositoryStub) Get(context.Context, contract.SkillIdentity, string, string, bool) (contract.SkillCatalogEntry, error) {
	panic("management read not expected")
}
func (r visibilityRepositoryStub) List(context.Context, contract.SkillIdentity, bool) ([]contract.SkillCatalogEntry, error) {
	panic("management list not expected")
}
func (r visibilityRepositoryStub) GetReferenced(_ context.Context, skillID, versionID string) (contract.SkillCatalogEntry, error) {
	value, ok := r.entries[skillID+":"+versionID]
	if !ok {
		return contract.SkillCatalogEntry{}, contract.ErrSkillNotFound
	}
	return value, nil
}
func (r visibilityRepositoryStub) History(context.Context, contract.SkillIdentity, string) (contract.SkillHistory, error) {
	panic("not used")
}

func TestSkillCatalogAgentReadsOnlyExplicitExactBinding(t *testing.T) {
	repository := visibilityRepositoryStub{entries: map[string]contract.SkillCatalogEntry{
		"skill-1:version-1": {ID: "skill-1", VersionID: "version-1", Status: "published"},
	}}
	authorize := func(context.Context, contract.SkillIdentity, string) error { return contract.ErrSkillAccessDenied }
	resolve := func(_ context.Context, workspaceID, skillID string) (contract.SkillVisibilityReference, error) {
		return contract.SkillVisibilityReference{WorkspaceID: workspaceID, SkillID: skillID, VersionID: "version-1", Enabled: true, AgentIDs: []string{"agent-1"}}, nil
	}
	service := NewSkillCatalog(repository, authorize, nil, nil, resolve, nil)

	allowed, err := service.Get(context.Background(), contract.SkillIdentity{WorkspaceID: "workspace-1", ActorType: "agent", ActorID: "agent-1"}, "skill-1", "version-1")
	if err != nil || allowed.VersionID != "version-1" {
		t.Fatalf("explicit Agent binding = %#v, %v", allowed, err)
	}
	for _, request := range []struct{ actorID, versionID string }{
		{actorID: "agent-2", versionID: "version-1"},
		{actorID: "agent-1", versionID: "version-2"},
	} {
		if _, err := service.Get(context.Background(), contract.SkillIdentity{WorkspaceID: "workspace-1", ActorType: "agent", ActorID: request.actorID}, "skill-1", request.versionID); err != contract.ErrSkillAccessDenied {
			t.Fatalf("Agent %s version %s error = %v, want access denied", request.actorID, request.versionID, err)
		}
	}
}

func TestSkillCatalogMemberListUsesEnabledPublishedWorkspaceBindings(t *testing.T) {
	repository := visibilityRepositoryStub{entries: map[string]contract.SkillCatalogEntry{
		"skill-1:version-1": {ID: "skill-1", VersionID: "version-1", Status: "published"},
		"skill-2:version-2": {ID: "skill-2", VersionID: "version-2", Status: "draft"},
		"skill-4:version-4": {ID: "skill-4", VersionID: "version-4", Status: "archived", Archived: true},
	}}
	authorize := func(_ context.Context, identity contract.SkillIdentity, permission string) error {
		if identity.ActorType == "member" && permission == contract.PermissionSkillRead {
			return nil
		}
		return contract.ErrSkillAccessDenied
	}
	list := func(_ context.Context, workspaceID string) ([]contract.SkillVisibilityReference, error) {
		return []contract.SkillVisibilityReference{
			{WorkspaceID: workspaceID, SkillID: "skill-1", VersionID: "version-1", Enabled: true},
			{WorkspaceID: workspaceID, SkillID: "skill-2", VersionID: "version-2", Enabled: true},
			{WorkspaceID: workspaceID, SkillID: "skill-3", VersionID: "version-3", Enabled: false},
			{WorkspaceID: workspaceID, SkillID: "skill-4", VersionID: "version-4", Enabled: true},
		}, nil
	}
	service := NewSkillCatalog(repository, authorize, nil, nil, nil, list)

	values, err := service.List(context.Background(), contract.SkillIdentity{WorkspaceID: "workspace-2", ActorType: "member", ActorID: "member-1"})
	if err != nil || len(values) != 1 || values[0].ID != "skill-1" {
		t.Fatalf("member visible Skills = %#v, %v", values, err)
	}
}
