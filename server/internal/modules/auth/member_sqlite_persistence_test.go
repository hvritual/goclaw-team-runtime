package auth

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/modules/auth/contract"
	"github.com/multica-ai/multica/server/internal/modules/auth/internal/application"
	"github.com/multica-ai/multica/server/internal/modules/auth/internal/domain/member"
	authsqlite "github.com/multica-ai/multica/server/internal/modules/auth/internal/infrastructure/sqlite"
	workspacecontract "github.com/multica-ai/multica/server/internal/modules/workspace/contract"
	_ "modernc.org/sqlite"
)

type staticWorkspaceIdentityReader struct {
	identity workspacecontract.WorkspaceIdentity
}

func (r staticWorkspaceIdentityReader) FindIdentity(context.Context, string) (workspacecontract.WorkspaceIdentity, error) {
	return r.identity, nil
}

func TestSqliteMemberRolePersistence(t *testing.T) {
	db, err := sql.Open("sqlite", "file:auth-member?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	ctx := t.Context()
	if err := MigrateSqlite(ctx, db); err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(ctx, `
		INSERT INTO users(id, name, email, created_at, updated_at) VALUES
			('owner-user', 'Owner', 'owner@example.test', '2026-08-02T00:00:00Z', '2026-08-02T00:00:00Z'),
			('member-user', 'Member', 'member@example.test', '2026-08-02T00:00:00Z', '2026-08-02T00:00:00Z');
		INSERT INTO members(id, workspace_id, user_id, role, created_at) VALUES
			('owner-member', 'workspace', 'owner-user', 'owner', '2026-08-02T00:00:00Z'),
			('target-member', 'workspace', 'member-user', 'member', '2026-08-02T00:00:00Z');
	`)
	if err != nil {
		t.Fatal(err)
	}

	module, err := NewWithSqlitePersistence(SqlitePersistenceConfig{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	memberCtx := contract.WithMemberActor(ctx, "owner-user")
	result, err := module.MemberLocal().UpdateMemberRole(memberCtx, contract.Member_UpdateMemberRoleRequest{
		WorkspaceId: "workspace",
		MemberId:    "target-member",
		Role:        "admin",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Role != "admin" || result.Name != "Member" || result.Email != "member@example.test" {
		t.Fatalf("unexpected member result: %+v", result)
	}
	var role string
	if err := db.QueryRowContext(ctx, `SELECT role FROM members WHERE id = 'target-member'`).Scan(&role); err != nil {
		t.Fatal(err)
	}
	if role != "admin" {
		t.Fatalf("persisted role = %q", role)
	}
}

func TestNewWithSqlitePersistenceRejectsMissingDatabase(t *testing.T) {
	if _, err := NewWithSqlitePersistence(SqlitePersistenceConfig{}); err == nil {
		t.Fatal("NewWithSqlitePersistence() error = nil")
	}
}

func TestSqliteListMembersIsWorkspaceScopedAndOrdered(t *testing.T) {
	db, err := sql.Open("sqlite", "file:auth-member-list?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	ctx := t.Context()
	if err := MigrateSqlite(ctx, db); err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(ctx, `
		INSERT INTO users(id, name, email, created_at, updated_at) VALUES
			('owner-user', 'Owner', 'owner@example.test', '2026-08-02T00:00:00Z', '2026-08-02T00:00:00Z'),
			('member-user', 'Member', 'member@example.test', '2026-08-02T00:00:00Z', '2026-08-02T00:00:00Z'),
			('other-user', 'Other', 'other@example.test', '2026-08-02T00:00:00Z', '2026-08-02T00:00:00Z');
		INSERT INTO members(id, workspace_id, user_id, role, created_at) VALUES
			('owner-member', 'workspace', 'owner-user', 'owner', '2026-08-02T00:00:00Z'),
			('member-member', 'workspace', 'member-user', 'member', '2026-08-02T00:00:01Z'),
			('other-member', 'other-workspace', 'other-user', 'owner', '2026-08-02T00:00:00Z');
	`)
	if err != nil {
		t.Fatal(err)
	}
	module, err := NewWithSqlitePersistence(SqlitePersistenceConfig{DB: db})
	if err != nil {
		t.Fatal(err)
	}

	result, err := module.MemberLocal().ListMembers(
		contract.WithMemberActor(ctx, "owner-user"),
		contract.Member_ListMembersRequest{WorkspaceId: "workspace"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Members) != 2 || result.Members[0].Id != "owner-member" || result.Members[1].Id != "member-member" {
		t.Fatalf("unexpected ordered workspace members: %+v", result.Members)
	}
	if result.Members[1].Email != "member@example.test" {
		t.Fatalf("unexpected member projection: %+v", result.Members[1])
	}
}

func TestSqliteMemberRemovalAndLeavePersistence(t *testing.T) {
	db, err := sql.Open("sqlite", "file:auth-member-removal?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	ctx := t.Context()
	if err := MigrateSqlite(ctx, db); err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(ctx, `
		INSERT INTO users(id, name, email, created_at, updated_at) VALUES
			('owner-user', 'Owner', 'owner@example.test', '2026-08-02T00:00:00Z', '2026-08-02T00:00:00Z'),
			('admin-user', 'Admin', 'admin@example.test', '2026-08-02T00:00:00Z', '2026-08-02T00:00:00Z'),
			('member-user', 'Member', 'member@example.test', '2026-08-02T00:00:00Z', '2026-08-02T00:00:00Z');
		INSERT INTO members(id, workspace_id, user_id, role, created_at) VALUES
			('owner-member', 'workspace', 'owner-user', 'owner', '2026-08-02T00:00:00Z'),
			('admin-member', 'workspace', 'admin-user', 'admin', '2026-08-02T00:00:00Z'),
			('target-member', 'workspace', 'member-user', 'member', '2026-08-02T00:00:00Z');
	`)
	if err != nil {
		t.Fatal(err)
	}
	module, err := NewWithSqlitePersistence(SqlitePersistenceConfig{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	service := module.MemberLocal()

	ownerCtx := contract.WithMemberActor(ctx, "owner-user")
	if _, err := service.DeleteMember(ownerCtx, contract.Member_DeleteMemberRequest{
		WorkspaceId: "workspace",
		MemberId:    "target-member",
	}); err != nil {
		t.Fatal(err)
	}
	adminCtx := contract.WithMemberActor(ctx, "admin-user")
	if _, err := service.LeaveWorkspace(adminCtx, contract.Member_LeaveWorkspaceRequest{
		WorkspaceId: "workspace",
	}); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM members WHERE workspace_id = 'workspace'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("remaining membership count = %d, want 1", count)
	}
}

func TestSqliteInvitationRevocationIsWorkspaceScopedAndPendingOnly(t *testing.T) {
	db, err := sql.Open("sqlite", "file:auth-invitation-revoke?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	ctx := t.Context()
	if err := MigrateSqlite(ctx, db); err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(ctx, `
		INSERT INTO users(id, name, email, created_at, updated_at)
		VALUES ('owner-user', 'Owner', 'owner@example.test', '2026-08-02T00:00:00Z', '2026-08-02T00:00:00Z');
		INSERT INTO members(id, workspace_id, user_id, role, created_at) VALUES
			('owner-member', 'workspace', 'owner-user', 'owner', '2026-08-02T00:00:00Z'),
			('other-owner-member', 'other-workspace', 'owner-user', 'owner', '2026-08-02T00:00:00Z');
		INSERT INTO invitations(
			id, workspace_id, inviter_id, invitee_email, role, status, created_at, updated_at, expires_at
		) VALUES (
			'invitation', 'workspace', 'owner-user', 'invitee@example.test', 'member', 'pending',
			'2026-08-02T00:00:00Z', '2026-08-02T00:00:00Z', '2026-08-09T00:00:00Z'
		);
	`)
	if err != nil {
		t.Fatal(err)
	}
	module, err := NewWithSqlitePersistence(SqlitePersistenceConfig{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	service := module.MemberLocal()
	actorCtx := contract.WithMemberActor(ctx, "owner-user")

	_, err = service.RevokeInvitation(actorCtx, contract.Member_RevokeInvitationRequest{
		WorkspaceId: "other-workspace", InvitationId: "invitation",
	})
	if !errors.Is(err, contract.ErrInvitationNotFound) {
		t.Fatalf("cross-workspace RevokeInvitation() error = %v", err)
	}
	var status string
	if err := db.QueryRowContext(ctx, `SELECT status FROM invitations WHERE id = 'invitation'`).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "pending" {
		t.Fatalf("cross-workspace revoke changed status to %q", status)
	}

	_, err = service.RevokeInvitation(actorCtx, contract.Member_RevokeInvitationRequest{
		WorkspaceId: "workspace", InvitationId: "invitation",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `SELECT status FROM invitations WHERE id = 'invitation'`).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "revoked" {
		t.Fatalf("invitation status = %q, want revoked", status)
	}
	_, err = service.RevokeInvitation(actorCtx, contract.Member_RevokeInvitationRequest{
		WorkspaceId: "workspace", InvitationId: "invitation",
	})
	if !errors.Is(err, contract.ErrInvitationNotFound) {
		t.Fatalf("second RevokeInvitation() error = %v", err)
	}
}

func TestSqliteWorkspaceInvitationListExpiresAndScopesPendingInvitations(t *testing.T) {
	db, err := sql.Open("sqlite", "file:auth-invitation-list?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	ctx := t.Context()
	if err := MigrateSqlite(ctx, db); err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(ctx, `
		INSERT INTO users(id, name, email, created_at, updated_at) VALUES
			('owner-user', 'Owner', 'owner@example.test', '2026-08-02T00:00:00Z', '2026-08-02T00:00:00Z'),
			('member-user', 'Member', 'member@example.test', '2026-08-02T00:00:00Z', '2026-08-02T00:00:00Z');
		INSERT INTO members(id, workspace_id, user_id, role, created_at) VALUES
			('owner-member', 'workspace', 'owner-user', 'owner', '2026-08-02T00:00:00Z'),
			('member-member', 'workspace', 'member-user', 'member', '2026-08-02T00:00:00Z');
		INSERT INTO invitations(
			id, workspace_id, inviter_id, invitee_email, role, status, created_at, updated_at, expires_at
		) VALUES
			('expired', 'workspace', 'owner-user', 'old@example.test', 'member', 'pending', '2026-08-01T00:00:00Z', '2026-08-01T00:00:00Z', '2026-08-02T11:59:59Z'),
			('pending', 'workspace', 'owner-user', 'new@example.test', 'admin', 'pending', '2026-08-02T11:00:00Z', '2026-08-02T11:00:00Z', '2026-08-09T12:00:00Z'),
			('other', 'other-workspace', 'owner-user', 'other@example.test', 'member', 'pending', '2026-08-02T10:00:00Z', '2026-08-02T10:00:00Z', '2026-08-09T12:00:00Z');
	`)
	if err != nil {
		t.Fatal(err)
	}
	module, err := NewWithSqlitePersistence(SqlitePersistenceConfig{
		DB: db, WorkspaceIdentities: staticWorkspaceIdentityReader{identity: workspacecontract.WorkspaceIdentity{ID: "workspace", Name: "Acme"}},
		Now: func() time.Time { return time.Date(2026, time.August, 2, 12, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatal(err)
	}
	service := module.MemberLocal()

	result, err := service.ListWorkspaceInvitations(
		contract.WithMemberActor(ctx, "owner-user"),
		contract.Member_ListWorkspaceInvitationsRequest{WorkspaceId: "workspace"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Invitations) != 1 || result.Invitations[0].Id != "pending" {
		t.Fatalf("unexpected pending invitations: %+v", result.Invitations)
	}
	if result.Invitations[0].WorkspaceName != "Acme" || result.Invitations[0].InviterName != "Owner" {
		t.Fatalf("unexpected invitation projection: %+v", result.Invitations[0])
	}
	var status string
	if err := db.QueryRowContext(ctx, `SELECT status FROM invitations WHERE id = 'expired'`).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "expired" {
		t.Fatalf("elapsed invitation status = %q", status)
	}

	_, err = service.ListWorkspaceInvitations(
		contract.WithMemberActor(ctx, "member-user"),
		contract.Member_ListWorkspaceInvitationsRequest{WorkspaceId: "workspace"},
	)
	if !errors.Is(err, contract.ErrInsufficientWorkspaceRole) {
		t.Fatalf("member ListWorkspaceInvitations() error = %v", err)
	}
}

func TestSqliteMemberRoleTransactionRollsBack(t *testing.T) {
	db, err := sql.Open("sqlite", "file:auth-member-rollback?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	ctx := t.Context()
	if err := MigrateSqlite(ctx, db); err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(ctx, `
		INSERT INTO users(id, name, email, created_at, updated_at)
		VALUES ('member-user', 'Member', 'member@example.test', '2026-08-02T00:00:00Z', '2026-08-02T00:00:00Z');
		INSERT INTO members(id, workspace_id, user_id, role, created_at)
		VALUES ('target-member', 'workspace', 'member-user', 'member', '2026-08-02T00:00:00Z');
	`)
	if err != nil {
		t.Fatal(err)
	}

	rollbackErr := errors.New("force rollback")
	store := authsqlite.NewMemberStore(db)
	err = store.WithinTransaction(ctx, func(repository application.MemberRepository) error {
		if _, updateErr := repository.UpdateRole(ctx, "workspace", "target-member", member.RoleAdmin); updateErr != nil {
			return updateErr
		}
		return rollbackErr
	})
	if !errors.Is(err, rollbackErr) {
		t.Fatalf("WithinTransaction() error = %v", err)
	}
	var role string
	if err := db.QueryRowContext(ctx, `SELECT role FROM members WHERE id = 'target-member'`).Scan(&role); err != nil {
		t.Fatal(err)
	}
	if role != "member" {
		t.Fatalf("rolled-back role = %q", role)
	}
	err = store.WithinTransaction(ctx, func(repository application.MemberRepository) error {
		if deleteErr := repository.DeleteByIDAndWorkspace(ctx, "workspace", "target-member"); deleteErr != nil {
			return deleteErr
		}
		return rollbackErr
	})
	if !errors.Is(err, rollbackErr) {
		t.Fatalf("WithinTransaction(delete) error = %v", err)
	}
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM members WHERE id = 'target-member'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("rolled-back membership count = %d, want 1", count)
	}
}
