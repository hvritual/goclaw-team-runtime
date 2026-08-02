package bootstrap

import (
	"database/sql"
	"testing"

	authcontract "github.com/multica-ai/multica/server/internal/modules/auth/contract"
	_ "modernc.org/sqlite"
)

func TestApplicationRegistersFourAcceptedModules(t *testing.T) {
	application := NewApplication()
	if got := len(application.Modules()); got != 4 {
		t.Fatalf("module count = %d, want 4", got)
	}
}

func TestSQLiteApplicationComposesWorkspaceIdentityIntoAuthInvitations(t *testing.T) {
	db, err := sql.Open("sqlite", "file:bootstrap-invitation-list?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	application, err := NewSQLiteApplication(t.Context(), db)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO workspaces(id, name, slug, issue_prefix, created_at, updated_at)
		VALUES ('workspace', 'Acme', 'acme', 'ACM', '2026-08-02T00:00:00Z', '2026-08-02T00:00:00Z');
		INSERT INTO users(id, name, email, created_at, updated_at)
		VALUES ('owner-user', 'Owner', 'owner@example.test', '2026-08-02T00:00:00Z', '2026-08-02T00:00:00Z');
		INSERT INTO members(id, workspace_id, user_id, role, created_at)
		VALUES ('owner-member', 'workspace', 'owner-user', 'owner', '2026-08-02T00:00:00Z');
		INSERT INTO invitations(
			id, workspace_id, inviter_id, invitee_email, role, status, created_at, updated_at, expires_at
		) VALUES (
			'invitation', 'workspace', 'owner-user', 'invitee@example.test', 'member', 'pending',
			'2026-08-02T00:00:00Z', '2026-08-02T00:00:00Z', '2099-08-09T00:00:00Z'
		);
	`)
	if err != nil {
		t.Fatal(err)
	}

	result, err := application.AuthMembers().ListWorkspaceInvitations(
		authcontract.WithMemberActor(t.Context(), "owner-user"),
		authcontract.Member_ListWorkspaceInvitationsRequest{WorkspaceId: "workspace"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Invitations) != 1 || result.Invitations[0].WorkspaceName != "Acme" {
		t.Fatalf("unexpected composed invitation list: %+v", result.Invitations)
	}
}
