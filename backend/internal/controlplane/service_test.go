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
		{RoleOwner, "unknown.permission", false},
		{RoleAdmin, "unknown.permission", false},
	}
	for _, test := range tests {
		t.Run(string(test.role)+"/"+test.permission, func(t *testing.T) {
			if got := roleAllows(test.role, test.permission); got != test.allowed {
				t.Fatalf("roleAllows() = %v, want %v", got, test.allowed)
			}
		})
	}
}

func TestAgentCannotBootstrapWorkspace(t *testing.T) {
	service, repository := newTestService(t, filepath.Join(t.TempDir(), "controlplane.db"))
	defer repository.Close()
	agent := Actor{ID: "agent-1", Kind: ActorAgent}
	if _, err := service.CreateWorkspace(context.Background(), agent, "workspace-1", "Invalid"); !errors.Is(err, ErrDenied) {
		t.Fatalf("agent workspace bootstrap error = %v, want denied", err)
	}
}

func TestAuthorizePreservesInfrastructureFailure(t *testing.T) {
	service, repository := newTestService(t, filepath.Join(t.TempDir(), "controlplane.db"))
	owner := Actor{ID: "owner-1", Kind: ActorHuman}
	if _, err := service.CreateWorkspace(context.Background(), owner, "workspace-1", "Primary"); err != nil {
		t.Fatal(err)
	}
	owner.WorkspaceID = "workspace-1"
	if err := repository.Close(); err != nil {
		t.Fatal(err)
	}
	if err := service.Authorize(context.Background(), owner, PermissionRead); err == nil || errors.Is(err, ErrDenied) {
		t.Fatalf("Authorize error = %v, want preserved infrastructure error", err)
	}
}

func TestRepositorySerializesLastOwnerRemoval(t *testing.T) {
	ctx := context.Background()
	service, repository := newTestService(t, filepath.Join(t.TempDir(), "controlplane.db"))
	defer repository.Close()
	owner := Actor{ID: "owner-1", Kind: ActorHuman}
	if _, err := service.CreateWorkspace(ctx, owner, "workspace-1", "Primary"); err != nil {
		t.Fatal(err)
	}
	owner.WorkspaceID = "workspace-1"
	second, err := service.AddMember(ctx, owner, "owner-2", ActorHuman, RoleOwner)
	if err != nil {
		t.Fatal(err)
	}
	first, err := repository.GetMember(ctx, owner.WorkspaceID, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	first.State, first.Version = MemberRemoved, first.Version+1
	second.State, second.Version = MemberRemoved, second.Version+1
	now := time.Date(2026, 8, 11, 13, 0, 0, 0, time.UTC)
	results := make(chan error, 2)
	go func() {
		results <- repository.SaveMember(ctx, first, first.Version-1, AuditEntry{ID: "audit-1", WorkspaceID: owner.WorkspaceID, ActorID: owner.ID, Action: "test.remove", Resource: "member", ResourceID: first.ID, OccurredAt: now})
	}()
	go func() {
		results <- repository.SaveMember(ctx, second, second.Version-1, AuditEntry{ID: "audit-2", WorkspaceID: owner.WorkspaceID, ActorID: owner.ID, Action: "test.remove", Resource: "member", ResourceID: second.ID, OccurredAt: now})
	}()
	var successes, invariants int
	for range 2 {
		err := <-results
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrInvariant):
			invariants++
		default:
			t.Fatalf("unexpected owner mutation error: %v", err)
		}
	}
	if successes != 1 || invariants != 1 {
		t.Fatalf("successes=%d invariants=%d", successes, invariants)
	}
	members, err := repository.ListMembers(ctx, owner.WorkspaceID, false)
	if err != nil {
		t.Fatal(err)
	}
	var activeOwners int
	for _, member := range members {
		if member.Kind == ActorHuman && member.Role == RoleOwner && member.State == MemberActive {
			activeOwners++
		}
	}
	if activeOwners != 1 {
		t.Fatalf("active owners = %d, want 1", activeOwners)
	}
}

func TestReconcileTrustedSnapshotProjectsRolesAndRemovals(t *testing.T) {
	ctx := context.Background()
	service, repository := newTestService(t, filepath.Join(t.TempDir(), "identity.db"))
	defer repository.Close()
	owner := Actor{ID: "old-owner", Kind: ActorHuman}
	if _, err := service.CreateWorkspace(ctx, owner, "workspace-1", "Old name"); err != nil {
		t.Fatal(err)
	}
	owner.WorkspaceID = "workspace-1"
	if _, err := service.AddMember(ctx, owner, "removed-member", ActorHuman, RoleMember); err != nil {
		t.Fatal(err)
	}
	if _, err := service.AddMember(ctx, owner, "agent-1", ActorAgent, RoleMember); err != nil {
		t.Fatal(err)
	}

	snapshot := TrustedWorkspaceSnapshot{
		ID: "workspace-1", Name: "Trusted name", ActorID: "new-owner",
		Members: []TrustedMember{{ID: "new-owner", Role: RoleOwner}, {ID: "old-owner", Role: RoleAdmin}},
	}
	if err := service.reconcileTrustedSnapshot(ctx, snapshot); err != nil {
		t.Fatal(err)
	}
	workspace, err := repository.GetWorkspace(ctx, "workspace-1")
	if err != nil || workspace.Name != "Trusted name" {
		t.Fatalf("workspace = %#v, error = %v", workspace, err)
	}
	members, err := repository.ListMembers(ctx, "workspace-1", true)
	if err != nil {
		t.Fatal(err)
	}
	states := make(map[string]Member)
	for _, member := range members {
		states[member.ID] = member
	}
	if states["new-owner"].Role != RoleOwner || states["old-owner"].Role != RoleAdmin || states["removed-member"].State != MemberRemoved {
		t.Fatalf("projected members = %#v", states)
	}
	if states["agent-1"].State != MemberActive {
		t.Fatalf("agent should not be changed by human identity projection: %#v", states["agent-1"])
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
