package application

import (
	"context"
	"errors"
	"testing"

	"github.com/multica-ai/multica/server/internal/modules/auth/contract"
	"github.com/multica-ai/multica/server/internal/modules/auth/internal/domain/member"
)

type fakeMemberUnitOfWork struct {
	repository *fakeMemberRepository
	called     bool
}

func (u *fakeMemberUnitOfWork) WithinTransaction(ctx context.Context, operation func(MemberRepository) error) error {
	u.called = true
	return operation(u.repository)
}

type fakeMemberRepository struct {
	requester  member.Member
	target     member.Member
	ownerCount int
	updated    bool
	deleted    bool
}

func (r *fakeMemberRepository) FindByUserAndWorkspace(context.Context, string, string) (member.Member, error) {
	if r.requester.ID == "" {
		return member.Member{}, ErrMembershipNotFound
	}
	return r.requester, nil
}

func (r *fakeMemberRepository) FindByIDAndWorkspace(context.Context, string, string) (member.Member, error) {
	if r.target.ID == "" {
		return member.Member{}, ErrMembershipNotFound
	}
	return r.target, nil
}

func (r *fakeMemberRepository) CountOwners(context.Context, string) (int, error) {
	return r.ownerCount, nil
}

func (r *fakeMemberRepository) UpdateRole(_ context.Context, _, _ string, role member.Role) (member.Member, error) {
	r.updated = true
	r.target.Role = role
	return r.target, nil
}

func (r *fakeMemberRepository) DeleteByIDAndWorkspace(context.Context, string, string) error {
	r.deleted = true
	return nil
}

func (r *fakeMemberRepository) DeleteByUserAndWorkspace(context.Context, string, string) error {
	r.deleted = true
	return nil
}

func TestUpdateMemberRole(t *testing.T) {
	avatarURL := "https://example.test/avatar.png"
	repository := &fakeMemberRepository{
		requester:  member.Member{ID: "requester", WorkspaceID: "workspace", UserID: "owner-user", Role: member.RoleOwner},
		target:     member.Member{ID: "target", WorkspaceID: "workspace", UserID: "member-user", Role: member.RoleMember, Name: "Member", Email: "member@example.test", AvatarURL: &avatarURL},
		ownerCount: 1,
	}
	unitOfWork := &fakeMemberUnitOfWork{repository: repository}
	service := NewMemberService(WithMemberUnitOfWork(unitOfWork))
	ctx := contract.WithMemberActor(context.Background(), "owner-user")

	result, err := service.UpdateMemberRole(ctx, contract.Member_UpdateMemberRoleRequest{
		WorkspaceId: "workspace",
		MemberId:    "target",
		Role:        "admin",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !unitOfWork.called || !repository.updated {
		t.Fatal("role update did not run inside the unit of work")
	}
	if result.Id != "target" || result.Role != "admin" || result.AvatarUrl == nil || *result.AvatarUrl != avatarURL {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestUpdateMemberRoleProtectsLastOwner(t *testing.T) {
	repository := &fakeMemberRepository{
		requester:  member.Member{ID: "requester", WorkspaceID: "workspace", UserID: "owner-user", Role: member.RoleOwner},
		target:     member.Member{ID: "target", WorkspaceID: "workspace", UserID: "target-user", Role: member.RoleOwner},
		ownerCount: 1,
	}
	service := NewMemberService(WithMemberUnitOfWork(&fakeMemberUnitOfWork{repository: repository}))
	ctx := contract.WithMemberActor(context.Background(), "owner-user")

	_, err := service.UpdateMemberRole(ctx, contract.Member_UpdateMemberRoleRequest{
		WorkspaceId: "workspace",
		MemberId:    "target",
		Role:        "member",
	})
	if !errors.Is(err, contract.ErrLastWorkspaceOwner) {
		t.Fatalf("UpdateMemberRole() error = %v", err)
	}
	if repository.updated {
		t.Fatal("last owner was updated")
	}
}

func TestUpdateMemberRoleRejectsInvalidInputBeforeTransaction(t *testing.T) {
	unitOfWork := &fakeMemberUnitOfWork{repository: &fakeMemberRepository{}}
	service := NewMemberService(WithMemberUnitOfWork(unitOfWork))
	ctx := contract.WithMemberActor(context.Background(), "owner-user")

	_, err := service.UpdateMemberRole(ctx, contract.Member_UpdateMemberRoleRequest{Role: "viewer"})
	if !errors.Is(err, contract.ErrInvalidMemberRole) {
		t.Fatalf("UpdateMemberRole() error = %v", err)
	}
	if unitOfWork.called {
		t.Fatal("invalid input opened a transaction")
	}
}

func TestUpdateMemberRoleHidesMissingRequesterMembership(t *testing.T) {
	repository := &fakeMemberRepository{
		target: member.Member{ID: "target", WorkspaceID: "workspace", Role: member.RoleMember},
	}
	service := NewMemberService(WithMemberUnitOfWork(&fakeMemberUnitOfWork{repository: repository}))
	ctx := contract.WithMemberActor(context.Background(), "outsider-user")

	_, err := service.UpdateMemberRole(ctx, contract.Member_UpdateMemberRoleRequest{
		WorkspaceId: "workspace",
		MemberId:    "target",
		Role:        "admin",
	})
	if !errors.Is(err, contract.ErrWorkspaceMembershipHidden) {
		t.Fatalf("UpdateMemberRole() error = %v", err)
	}
}

func TestUpdateMemberRoleReportsMissingTargetMember(t *testing.T) {
	repository := &fakeMemberRepository{
		requester: member.Member{ID: "requester", WorkspaceID: "workspace", UserID: "owner-user", Role: member.RoleOwner},
	}
	service := NewMemberService(WithMemberUnitOfWork(&fakeMemberUnitOfWork{repository: repository}))
	ctx := contract.WithMemberActor(context.Background(), "owner-user")

	_, err := service.UpdateMemberRole(ctx, contract.Member_UpdateMemberRoleRequest{
		WorkspaceId: "workspace",
		MemberId:    "missing",
		Role:        "admin",
	})
	if !errors.Is(err, contract.ErrMemberNotFound) {
		t.Fatalf("UpdateMemberRole() error = %v", err)
	}
}

func TestDeleteMemberRemovesMembership(t *testing.T) {
	repository := &fakeMemberRepository{
		requester: member.Member{ID: "requester", WorkspaceID: "workspace", UserID: "admin-user", Role: member.RoleAdmin},
		target:    member.Member{ID: "target", WorkspaceID: "workspace", UserID: "member-user", Role: member.RoleMember},
	}
	service := NewMemberService(WithMemberUnitOfWork(&fakeMemberUnitOfWork{repository: repository}))
	ctx := contract.WithMemberActor(context.Background(), "admin-user")

	_, err := service.DeleteMember(ctx, contract.Member_DeleteMemberRequest{
		WorkspaceId: "workspace",
		MemberId:    "target",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !repository.deleted {
		t.Fatal("membership was not deleted")
	}
}

func TestDeleteMemberRejectsAdminRemovingOwner(t *testing.T) {
	repository := &fakeMemberRepository{
		requester:  member.Member{ID: "requester", WorkspaceID: "workspace", UserID: "admin-user", Role: member.RoleAdmin},
		target:     member.Member{ID: "target", WorkspaceID: "workspace", UserID: "owner-user", Role: member.RoleOwner},
		ownerCount: 2,
	}
	service := NewMemberService(WithMemberUnitOfWork(&fakeMemberUnitOfWork{repository: repository}))
	ctx := contract.WithMemberActor(context.Background(), "admin-user")

	_, err := service.DeleteMember(ctx, contract.Member_DeleteMemberRequest{WorkspaceId: "workspace", MemberId: "target"})
	if !errors.Is(err, contract.ErrOwnerRemovalRequiresOwner) {
		t.Fatalf("DeleteMember() error = %v", err)
	}
	if repository.deleted {
		t.Fatal("owner membership was deleted")
	}
}

func TestDeleteMemberChecksManagerRoleBeforeTargetExistence(t *testing.T) {
	repository := &fakeMemberRepository{
		requester: member.Member{ID: "requester", WorkspaceID: "workspace", UserID: "member-user", Role: member.RoleMember},
	}
	service := NewMemberService(WithMemberUnitOfWork(&fakeMemberUnitOfWork{repository: repository}))
	ctx := contract.WithMemberActor(context.Background(), "member-user")

	_, err := service.DeleteMember(ctx, contract.Member_DeleteMemberRequest{
		WorkspaceId: "workspace",
		MemberId:    "missing-or-cross-workspace",
	})
	if !errors.Is(err, contract.ErrInsufficientWorkspaceRole) {
		t.Fatalf("DeleteMember() error = %v", err)
	}
}

func TestLeaveWorkspaceProtectsLastOwner(t *testing.T) {
	repository := &fakeMemberRepository{
		requester:  member.Member{ID: "requester", WorkspaceID: "workspace", UserID: "owner-user", Role: member.RoleOwner},
		ownerCount: 1,
	}
	service := NewMemberService(WithMemberUnitOfWork(&fakeMemberUnitOfWork{repository: repository}))
	ctx := contract.WithMemberActor(context.Background(), "owner-user")

	_, err := service.LeaveWorkspace(ctx, contract.Member_LeaveWorkspaceRequest{WorkspaceId: "workspace"})
	if !errors.Is(err, contract.ErrLastWorkspaceOwner) {
		t.Fatalf("LeaveWorkspace() error = %v", err)
	}
	if repository.deleted {
		t.Fatal("last owner membership was deleted")
	}
}

func TestLeaveWorkspaceRemovesNonOwnerMembership(t *testing.T) {
	repository := &fakeMemberRepository{
		requester: member.Member{ID: "requester", WorkspaceID: "workspace", UserID: "member-user", Role: member.RoleMember},
	}
	service := NewMemberService(WithMemberUnitOfWork(&fakeMemberUnitOfWork{repository: repository}))
	ctx := contract.WithMemberActor(context.Background(), "member-user")

	_, err := service.LeaveWorkspace(ctx, contract.Member_LeaveWorkspaceRequest{WorkspaceId: "workspace"})
	if err != nil {
		t.Fatal(err)
	}
	if !repository.deleted {
		t.Fatal("membership was not deleted")
	}
}
