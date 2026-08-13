package auth

import (
	"context"
	"testing"
	"time"
)

func TestWorkspaceMembershipReaderReturnsOnlyAuthenticatedUsersMemberships(t *testing.T) {
	db := openAuthTestDB(t)
	now := "2026-08-13T00:00:00Z"
	for _, statement := range []string{
		`INSERT INTO auth_users(id,name,email,created_at,updated_at) VALUES
			('user-1','Owner','owner@example.com','` + now + `','` + now + `'),
			('user-2','Other','other@example.com','` + now + `','` + now + `')`,
		`INSERT INTO auth_members(id,workspace_id,user_id,role,created_at) VALUES
			('member-owner','workspace-owner','user-1','owner','` + now + `'),
			('member-admin','workspace-admin','user-1','admin','` + now + `'),
			('member-member','workspace-member','user-1','member','` + now + `'),
			('member-foreign','workspace-foreign','user-2','owner','` + now + `')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	module, err := NewWithSqliteLocalAuth(SqlitePersistenceConfig{DB: db}, LocalAuthConfig{
		VerificationCode: "888888", SessionTTL: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}

	memberships, err := module.WorkspaceMemberships().ListForUser(context.Background(), "user-1")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"workspace-owner":  "owner",
		"workspace-admin":  "admin",
		"workspace-member": "member",
	}
	if len(memberships) != len(want) {
		t.Fatalf("memberships = %#v, want %d entries", memberships, len(want))
	}
	for _, membership := range memberships {
		if want[membership.WorkspaceID] != membership.Role {
			t.Fatalf("unexpected membership: %#v", membership)
		}
		delete(want, membership.WorkspaceID)
	}
	if len(want) != 0 {
		t.Fatalf("missing memberships: %#v", want)
	}

	empty, err := module.WorkspaceMemberships().ListForUser(context.Background(), "user-missing")
	if err != nil || len(empty) != 0 {
		t.Fatalf("missing user memberships = %#v, %v", empty, err)
	}
	if membership, ok, err := module.WorkspaceMemberships().FindForUserAndWorkspace(context.Background(), "user-1", "workspace-owner"); err != nil || !ok || membership.MemberID != "member-owner" {
		t.Fatalf("owner membership = %#v, %t, %v", membership, ok, err)
	}
	if membership, ok, err := module.WorkspaceMemberships().FindForUserAndWorkspace(context.Background(), "user-1", "workspace-foreign"); err != nil || ok || membership.MemberID != "" {
		t.Fatalf("foreign membership = %#v, %t, %v", membership, ok, err)
	}
}
