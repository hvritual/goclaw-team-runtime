package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/modules/auth/contract"
	"github.com/multica-ai/multica/server/internal/modules/auth/internal/domain/member"
)

type fakeInvitationRepository struct {
	workspaceID  string
	invitationID string
	updatedAt    time.Time
	revoked      bool
	err          error
}

func (r *fakeInvitationRepository) RevokePending(_ context.Context, workspaceID, invitationID string, updatedAt time.Time) error {
	r.workspaceID = workspaceID
	r.invitationID = invitationID
	r.updatedAt = updatedAt
	if r.err != nil {
		return r.err
	}
	r.revoked = true
	return nil
}

type fakeInvitationUnitOfWork struct {
	members     *fakeMemberRepository
	invitations *fakeInvitationRepository
	called      bool
}

func (u *fakeInvitationUnitOfWork) WithinInvitationTransaction(
	ctx context.Context,
	operation func(MemberRepository, InvitationRepository) error,
) error {
	u.called = true
	return operation(u.members, u.invitations)
}

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
	members    []member.Member
	ownerCount int
	updated    bool
	deleted    bool
	listed     bool
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

func (r *fakeMemberRepository) ListByWorkspace(context.Context, string) ([]member.Member, error) {
	r.listed = true
	return r.members, nil
}

func TestListMembersReturnsWorkspaceMemberships(t *testing.T) {
	repository := &fakeMemberRepository{
		requester: member.Member{ID: "requester", WorkspaceID: "workspace", UserID: "member-user", Role: member.RoleMember},
		members: []member.Member{
			{ID: "owner-member", WorkspaceID: "workspace", UserID: "owner-user", Role: member.RoleOwner, Name: "Owner"},
			{ID: "member", WorkspaceID: "workspace", UserID: "member-user", Role: member.RoleMember, Name: "Member"},
		},
	}
	service := NewMemberService(WithMemberUnitOfWork(&fakeMemberUnitOfWork{repository: repository}))
	ctx := contract.WithMemberActor(context.Background(), "member-user")

	result, err := service.ListMembers(ctx, contract.Member_ListMembersRequest{WorkspaceId: "workspace"})
	if err != nil {
		t.Fatal(err)
	}
	if !repository.listed || len(result.Members) != 2 {
		t.Fatalf("unexpected member list: %+v", result.Members)
	}
	if result.Members[0].Id != "owner-member" || result.Members[1].Name != "Member" {
		t.Fatalf("unexpected member projection: %+v", result.Members)
	}
}

func TestListMembersHidesWorkspaceFromOutsider(t *testing.T) {
	repository := &fakeMemberRepository{
		members: []member.Member{{ID: "private-member", WorkspaceID: "workspace"}},
	}
	service := NewMemberService(WithMemberUnitOfWork(&fakeMemberUnitOfWork{repository: repository}))
	ctx := contract.WithMemberActor(context.Background(), "outsider-user")

	_, err := service.ListMembers(ctx, contract.Member_ListMembersRequest{WorkspaceId: "workspace"})
	if !errors.Is(err, contract.ErrWorkspaceMembershipHidden) {
		t.Fatalf("ListMembers() error = %v", err)
	}
	if repository.listed {
		t.Fatal("workspace members were listed for an outsider")
	}
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

func TestRevokeInvitationWithdrawsPendingInvitation(t *testing.T) {
	fixedNow := time.Date(2026, time.August, 2, 12, 0, 0, 0, time.UTC)
	members := &fakeMemberRepository{requester: member.Member{
		ID: "requester", WorkspaceID: "workspace", UserID: "admin-user", Role: member.RoleAdmin,
	}}
	invitations := &fakeInvitationRepository{}
	unitOfWork := &fakeInvitationUnitOfWork{members: members, invitations: invitations}
	service := NewMemberService(
		WithInvitationUnitOfWork(unitOfWork),
		WithInvitationClock(func() time.Time { return fixedNow }),
	)
	ctx := contract.WithMemberActor(context.Background(), "admin-user")

	_, err := service.RevokeInvitation(ctx, contract.Member_RevokeInvitationRequest{
		WorkspaceId:  "workspace",
		InvitationId: "invitation",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !unitOfWork.called || !invitations.revoked {
		t.Fatal("invitation was not revoked inside the invitation transaction")
	}
	if invitations.workspaceID != "workspace" || invitations.invitationID != "invitation" || !invitations.updatedAt.Equal(fixedNow) {
		t.Fatalf("unexpected revoke call: %+v", invitations)
	}
}

func TestRevokeInvitationRequiresManagerBeforeLookingUpInvitation(t *testing.T) {
	members := &fakeMemberRepository{requester: member.Member{
		ID: "requester", WorkspaceID: "workspace", UserID: "member-user", Role: member.RoleMember,
	}}
	invitations := &fakeInvitationRepository{err: ErrInvitationNotFound}
	service := NewMemberService(WithInvitationUnitOfWork(&fakeInvitationUnitOfWork{
		members: members, invitations: invitations,
	}))
	ctx := contract.WithMemberActor(context.Background(), "member-user")

	_, err := service.RevokeInvitation(ctx, contract.Member_RevokeInvitationRequest{
		WorkspaceId: "workspace", InvitationId: "missing-or-cross-workspace",
	})
	if !errors.Is(err, contract.ErrInsufficientWorkspaceRole) {
		t.Fatalf("RevokeInvitation() error = %v", err)
	}
	if invitations.revoked || invitations.invitationID != "" {
		t.Fatal("invitation lookup ran before manager authorization")
	}
}

func TestRevokeInvitationReportsMissingPendingInvitation(t *testing.T) {
	members := &fakeMemberRepository{requester: member.Member{
		ID: "requester", WorkspaceID: "workspace", UserID: "owner-user", Role: member.RoleOwner,
	}}
	invitations := &fakeInvitationRepository{err: ErrInvitationNotFound}
	service := NewMemberService(WithInvitationUnitOfWork(&fakeInvitationUnitOfWork{
		members: members, invitations: invitations,
	}))
	ctx := contract.WithMemberActor(context.Background(), "owner-user")

	_, err := service.RevokeInvitation(ctx, contract.Member_RevokeInvitationRequest{
		WorkspaceId: "workspace", InvitationId: "missing",
	})
	if !errors.Is(err, contract.ErrInvitationNotFound) {
		t.Fatalf("RevokeInvitation() error = %v", err)
	}
}
