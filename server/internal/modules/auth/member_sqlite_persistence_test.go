package auth

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/multica-ai/multica/server/internal/modules/auth/contract"
	"github.com/multica-ai/multica/server/internal/modules/auth/internal/application"
	"github.com/multica-ai/multica/server/internal/modules/auth/internal/domain/member"
	authsqlite "github.com/multica-ai/multica/server/internal/modules/auth/internal/infrastructure/sqlite"
	_ "modernc.org/sqlite"
)

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
