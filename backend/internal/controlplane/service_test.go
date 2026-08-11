package controlplane

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestWorkspaceMembershipAndIsolation(t *testing.T) {
	ctx := context.Background()
	service, repository := newTestService(t, filepath.Join(t.TempDir(), "controlplane.db"))
	defer repository.Close()

	owner := Actor{ID: "owner-1", Kind: ActorHuman}
	workspace, err := service.CreateWorkspace(ctx, owner, "workspace-1", "Primary")
	if err != nil {
		t.Fatal(err)
	}
	owner.WorkspaceID = workspace.ID

	member, err := service.AddMember(ctx, owner, "member-1", ActorHuman, RoleMember)
	if err != nil {
		t.Fatal(err)
	}
	if member.Role != RoleMember || member.Version != 1 {
		t.Fatalf("member = %#v", member)
	}

	outsider := Actor{ID: "member-1", WorkspaceID: "workspace-2", Kind: ActorHuman}
	if _, err := service.GetWorkspace(ctx, outsider); !errors.Is(err, ErrDenied) {
		t.Fatalf("cross-workspace GetWorkspace error = %v, want denied", err)
	}

	if _, err := service.ChangeMemberRole(ctx, owner, owner.ID, RoleAdmin, 1); !errors.Is(err, ErrInvariant) {
		t.Fatalf("last-owner role change error = %v, want invariant", err)
	}
}

func TestSQLitePersistsAndUsesCAS(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "controlplane.db")
	service, repository := newTestService(t, path)
	owner := Actor{ID: "owner-1", Kind: ActorHuman}
	if _, err := service.CreateWorkspace(ctx, owner, "workspace-1", "Primary"); err != nil {
		t.Fatal(err)
	}
	owner.WorkspaceID = "workspace-1"
	if err := repository.Close(); err != nil {
		t.Fatal(err)
	}

	service, repository = newTestService(t, path)
	defer repository.Close()
	workspace, err := service.GetWorkspace(ctx, owner)
	if err != nil {
		t.Fatal(err)
	}
	if workspace.Name != "Primary" {
		t.Fatalf("workspace name = %q", workspace.Name)
	}
	if _, err := service.UpdateWorkspace(ctx, owner, "Changed", WorkspaceActive, 99); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale update error = %v, want conflict", err)
	}
}

func TestRoleAuthorizationMatrix(t *testing.T) {
	tests := []struct {
		role       Role
		permission string
		allowed    bool
	}{
		{RoleOwner, PermissionAccept, true},
		{RoleAdmin, PermissionManageMembers, true},
		{RoleAdmin, PermissionAccept, false},
		{RoleMember, PermissionRun, true},
		{RoleMember, PermissionReview, false},
		{RoleReviewer, PermissionAccept, true},
		{RoleViewer, PermissionWrite, false},
	}
	for _, test := range tests {
		t.Run(string(test.role)+"/"+test.permission, func(t *testing.T) {
			if got := roleAllows(test.role, test.permission); got != test.allowed {
				t.Fatalf("roleAllows() = %v, want %v", got, test.allowed)
			}
		})
	}
}

func newTestService(t *testing.T, path string) (*Service, Repository) {
	t.Helper()
	repository, err := OpenSQLite(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	fixed := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	service, err := NewService(repository, func() time.Time { return fixed })
	if err != nil {
		t.Fatal(err)
	}
	return service, repository
}
