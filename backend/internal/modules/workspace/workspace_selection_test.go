package workspace

import (
	"context"
	"errors"
	"testing"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

type selectionMemberships map[string][]contract.WorkspaceMembership

func (m selectionMemberships) ListForUser(_ context.Context, userID string) ([]contract.WorkspaceMembership, error) {
	return append([]contract.WorkspaceMembership(nil), m[userID]...), nil
}

func (m selectionMemberships) FindForUserAndWorkspace(_ context.Context, userID, workspaceID string) (contract.WorkspaceMembership, bool, error) {
	for _, membership := range m[userID] {
		if membership.WorkspaceID == workspaceID {
			return membership, true, nil
		}
	}
	return contract.WorkspaceMembership{}, false, nil
}

func (m selectionMemberships) FindByMemberAndWorkspace(_ context.Context, memberID, workspaceID string) (contract.WorkspaceMembership, bool, error) {
	for _, memberships := range m {
		for _, membership := range memberships {
			if membership.MemberID == memberID && membership.WorkspaceID == workspaceID {
				return membership, true, nil
			}
		}
	}
	return contract.WorkspaceMembership{}, false, nil
}

func TestSqliteWorkspaceSelectionFiltersBeforeReadingWorkspaceData(t *testing.T) {
	db := openWorkspaceTestDB(t)
	for _, row := range []struct{ id, name, slug, prefix string }{
		{"workspace-owner", "Owner Space", "owner-space", "OWN"},
		{"workspace-admin", "Admin Space", "admin-space", "ADM"},
		{"workspace-member", "Member Space", "member-space", "MEM"},
		{"workspace-foreign", "Foreign Space", "foreign-space", "FOR"},
	} {
		if _, err := db.Exec(`INSERT INTO workspaces(
			id,name,slug,description,context,settings,repos,issue_prefix,avatar_url,created_at,updated_at
		) VALUES(?,?,?,NULL,NULL,'{}','[]',?,NULL,'2026-08-13T00:00:00Z','2026-08-13T00:00:00Z')`, row.id, row.name, row.slug, row.prefix); err != nil {
			t.Fatal(err)
		}
	}
	selection, err := NewSqliteWorkspaceSelection(SqlitePersistenceConfig{DB: db}, selectionMemberships{
		"user-1": {
			{WorkspaceID: "workspace-owner", Role: "owner"},
			{WorkspaceID: "workspace-admin", Role: "admin"},
			{WorkspaceID: "workspace-member", Role: "member"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	list, err := selection.List(context.Background(), "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 3 || list[0].ID != "workspace-admin" || list[1].ID != "workspace-member" || list[2].ID != "workspace-owner" {
		t.Fatalf("authorized list = %#v", list)
	}
	empty, err := selection.List(context.Background(), "user-empty")
	if err != nil || len(empty) != 0 {
		t.Fatalf("empty list = %#v, %v", empty, err)
	}
	resolved, err := selection.ResolveSlug(context.Background(), "user-1", "member-space")
	if err != nil || resolved != "workspace-member" {
		t.Fatalf("member slug = %q, %v", resolved, err)
	}
	if _, err := selection.ResolveSlug(context.Background(), "user-1", "foreign-space"); !errors.Is(err, contract.ErrWorkspaceNotFound) {
		t.Fatalf("foreign slug error = %v", err)
	}
}
