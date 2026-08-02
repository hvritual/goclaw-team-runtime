package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/modules/auth/contract"
	"github.com/multica-ai/multica/server/internal/modules/auth/internal/domain/invitation"
	"github.com/multica-ai/multica/server/internal/modules/auth/internal/domain/member"
	workspacecontract "github.com/multica-ai/multica/server/internal/modules/workspace/contract"
)

type fakeInvitationRepository struct {
	workspaceID   string
	invitationID  string
	inviteeUserID string
	email         string
	updatedAt     time.Time
	revoked       bool
	expired       bool
	listed        bool
	pending       bool
	created       *invitation.Invitation
	accepted      bool
	declined      bool
	values        []invitation.Invitation
	found         invitation.Invitation
	err           error
}

func (r *fakeInvitationRepository) ExpirePendingByWorkspace(_ context.Context, workspaceID string, expiredAt time.Time) error {
	r.workspaceID = workspaceID
	r.updatedAt = expiredAt
	r.expired = true
	return r.err
}

func (r *fakeInvitationRepository) ListPendingByWorkspace(_ context.Context, workspaceID string) ([]invitation.Invitation, error) {
	r.workspaceID = workspaceID
	r.listed = true
	return r.values, r.err
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

func (r *fakeInvitationRepository) ExpirePendingByWorkspaceAndEmail(_ context.Context, workspaceID, email string, expiredAt time.Time) error {
	r.workspaceID = workspaceID
	r.email = email
	r.updatedAt = expiredAt
	r.expired = true
	return r.err
}

func (r *fakeInvitationRepository) PendingExistsByWorkspaceAndEmail(_ context.Context, workspaceID, email string) (bool, error) {
	r.workspaceID = workspaceID
	r.email = email
	return r.pending, r.err
}

func (r *fakeInvitationRepository) Create(_ context.Context, value invitation.Invitation) error {
	if r.err != nil {
		return r.err
	}
	r.created = &value
	return nil
}

func (r *fakeInvitationRepository) ExpirePendingByInvitee(
	_ context.Context,
	invitee member.UserIdentity,
	expiredAt time.Time,
) error {
	r.inviteeUserID = invitee.ID
	r.email = invitee.Email
	r.updatedAt = expiredAt
	r.expired = true
	return r.err
}

func (r *fakeInvitationRepository) ListPendingByInvitee(
	_ context.Context,
	invitee member.UserIdentity,
) ([]invitation.Invitation, error) {
	r.inviteeUserID = invitee.ID
	r.email = invitee.Email
	r.listed = true
	return r.values, r.err
}

func (r *fakeInvitationRepository) FindByID(_ context.Context, invitationID string) (invitation.Invitation, error) {
	r.invitationID = invitationID
	if r.err != nil {
		return invitation.Invitation{}, r.err
	}
	if r.found.ID == "" {
		return invitation.Invitation{}, ErrInvitationNotFound
	}
	return r.found, nil
}

func (r *fakeInvitationRepository) AcceptPending(_ context.Context, invitationID, inviteeUserID string, updatedAt time.Time) error {
	r.invitationID = invitationID
	r.inviteeUserID = inviteeUserID
	r.updatedAt = updatedAt
	if r.err != nil {
		return r.err
	}
	r.accepted = true
	return nil
}

func (r *fakeInvitationRepository) DeclinePending(_ context.Context, invitationID, inviteeUserID string, updatedAt time.Time) error {
	r.invitationID = invitationID
	r.inviteeUserID = inviteeUserID
	r.updatedAt = updatedAt
	if r.err != nil {
		return r.err
	}
	r.declined = true
	return nil
}

func (r *fakeInvitationRepository) ExpirePendingByID(_ context.Context, invitationID string, updatedAt time.Time) error {
	r.invitationID = invitationID
	r.updatedAt = updatedAt
	if r.err != nil {
		return r.err
	}
	r.expired = true
	return nil
}

type fakeWorkspaceIdentityReader struct {
	identity   workspacecontract.WorkspaceIdentity
	identities map[string]workspacecontract.WorkspaceIdentity
	called     bool
	err        error
}

func (r *fakeWorkspaceIdentityReader) FindIdentity(_ context.Context, workspaceID string) (workspacecontract.WorkspaceIdentity, error) {
	r.called = true
	if r.identities != nil {
		identity, ok := r.identities[workspaceID]
		if !ok {
			return workspacecontract.WorkspaceIdentity{}, workspacecontract.ErrWorkspaceNotFound
		}
		return identity, nil
	}
	return r.identity, r.err
}

type fakeInvitationUnitOfWork struct {
	members     *fakeMemberRepository
	invitations *fakeInvitationRepository
	called      bool
}

func (u *fakeInvitationUnitOfWork) WithinInvitationTransaction(
	ctx context.Context,
	operation func(MemberRepository, InvitationRepository, InvitationDecisionRepository) error,
) error {
	u.called = true
	return operation(u.members, u.invitations, &fakeInvitationDecisionRepository{
		fakeMemberRepository:     u.members,
		fakeInvitationRepository: u.invitations,
	})
}

type fakeInvitationDecisionRepository struct {
	*fakeMemberRepository
	*fakeInvitationRepository
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
	requester       member.Member
	target          member.Member
	members         []member.Member
	ownerCount      int
	memberByEmail   bool
	inviteeUserID   *string
	updated         bool
	deleted         bool
	listed          bool
	resolvedInvitee bool
	created         *member.Member
	onboarded       bool
	currentUser     member.UserIdentity
	findUserErr     error
	createErr       error
	onboardingErr   error
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

func (r *fakeMemberRepository) ExistsByEmail(context.Context, string, string) (bool, error) {
	return r.memberByEmail, nil
}

func (r *fakeMemberRepository) FindUserIDByEmail(context.Context, string) (*string, error) {
	r.resolvedInvitee = true
	return r.inviteeUserID, nil
}

func (r *fakeMemberRepository) FindUserByID(context.Context, string) (member.UserIdentity, error) {
	if r.findUserErr != nil {
		return member.UserIdentity{}, r.findUserErr
	}
	if r.currentUser.ID == "" {
		return member.UserIdentity{}, ErrAuthUserNotFound
	}
	return r.currentUser, nil
}

func (r *fakeMemberRepository) CreateMember(_ context.Context, value member.Member) error {
	if r.createErr != nil {
		return r.createErr
	}
	r.created = &value
	return nil
}

func (r *fakeMemberRepository) CompleteOnboarding(context.Context, string, time.Time) error {
	if r.onboardingErr != nil {
		return r.onboardingErr
	}
	r.onboarded = true
	return nil
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

func TestListWorkspaceInvitationsExpiresAndReturnsPendingInvitations(t *testing.T) {
	fixedNow := time.Date(2026, time.August, 2, 12, 0, 0, 0, time.UTC)
	inviteeUserID := "invitee-user"
	members := &fakeMemberRepository{requester: member.Member{
		ID: "requester", WorkspaceID: "workspace", UserID: "admin-user", Role: member.RoleAdmin,
	}}
	invitations := &fakeInvitationRepository{values: []invitation.Invitation{{
		ID: "invitation", WorkspaceID: "workspace", InviterID: "owner-user",
		InviteeEmail: "invitee@example.test", InviteeUserID: &inviteeUserID,
		Role: member.RoleMember, Status: invitation.StatusPending,
		CreatedAt: invitation.NewTimestamp(fixedNow.Add(-time.Hour)), UpdatedAt: invitation.NewTimestamp(fixedNow.Add(-time.Hour)), ExpiresAt: invitation.NewTimestamp(fixedNow.Add(7 * 24 * time.Hour)),
		InviterName: "Owner", InviterEmail: "owner@example.test",
	}}}
	identities := &fakeWorkspaceIdentityReader{identity: workspacecontract.WorkspaceIdentity{ID: "workspace", Name: "Acme"}}
	service := NewMemberService(
		WithInvitationUnitOfWork(&fakeInvitationUnitOfWork{members: members, invitations: invitations}),
		WithWorkspaceIdentityReader(identities),
		WithInvitationClock(func() time.Time { return fixedNow }),
	)

	result, err := service.ListWorkspaceInvitations(
		contract.WithMemberActor(context.Background(), "admin-user"),
		contract.Member_ListWorkspaceInvitationsRequest{WorkspaceId: "workspace"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !invitations.expired || !invitations.listed || !invitations.updatedAt.Equal(fixedNow) {
		t.Fatalf("pending invitation lifecycle not applied: %+v", invitations)
	}
	if !identities.called || len(result.Invitations) != 1 {
		t.Fatalf("unexpected invitation list: %+v", result.Invitations)
	}
	got := result.Invitations[0]
	if got.Id != "invitation" || got.WorkspaceName != "Acme" || got.InviterName != "Owner" || got.InviteeUserId == nil || *got.InviteeUserId != inviteeUserID {
		t.Fatalf("unexpected invitation projection: %+v", got)
	}
	if got.CreatedAt != "2026-08-02T11:00:00Z" || got.ExpiresAt != "2026-08-09T12:00:00Z" {
		t.Fatalf("unexpected invitation timestamps: %+v", got)
	}
}

func TestListWorkspaceInvitationsRejectsMemberBeforeInvitationLookup(t *testing.T) {
	members := &fakeMemberRepository{requester: member.Member{
		ID: "requester", WorkspaceID: "workspace", UserID: "member-user", Role: member.RoleMember,
	}}
	invitations := &fakeInvitationRepository{}
	identities := &fakeWorkspaceIdentityReader{}
	service := NewMemberService(
		WithInvitationUnitOfWork(&fakeInvitationUnitOfWork{members: members, invitations: invitations}),
		WithWorkspaceIdentityReader(identities),
	)

	_, err := service.ListWorkspaceInvitations(
		contract.WithMemberActor(context.Background(), "member-user"),
		contract.Member_ListWorkspaceInvitationsRequest{WorkspaceId: "workspace"},
	)
	if !errors.Is(err, contract.ErrInsufficientWorkspaceRole) {
		t.Fatalf("ListWorkspaceInvitations() error = %v", err)
	}
	if invitations.expired || invitations.listed || identities.called {
		t.Fatal("unauthorized member observed invitation or workspace identity data")
	}
}

func TestListWorkspaceInvitationsHidesMissingWorkspaceIdentity(t *testing.T) {
	members := &fakeMemberRepository{requester: member.Member{
		ID: "requester", WorkspaceID: "workspace", UserID: "owner-user", Role: member.RoleOwner,
	}}
	service := NewMemberService(
		WithInvitationUnitOfWork(&fakeInvitationUnitOfWork{
			members: members, invitations: &fakeInvitationRepository{},
		}),
		WithWorkspaceIdentityReader(&fakeWorkspaceIdentityReader{err: workspacecontract.ErrWorkspaceNotFound}),
	)

	_, err := service.ListWorkspaceInvitations(
		contract.WithMemberActor(context.Background(), "owner-user"),
		contract.Member_ListWorkspaceInvitationsRequest{WorkspaceId: "workspace"},
	)
	if !errors.Is(err, contract.ErrWorkspaceMembershipHidden) {
		t.Fatalf("ListWorkspaceInvitations() error = %v", err)
	}
}

func TestCreateInvitationPersistsNormalizedPendingInvitation(t *testing.T) {
	fixedNow := time.Date(2026, time.August, 2, 12, 0, 0, 0, time.UTC)
	members := &fakeMemberRepository{requester: member.Member{
		ID: "owner-member", WorkspaceID: "workspace", UserID: "owner-user",
		Role: member.RoleOwner, Name: "Owner", Email: "owner@example.test",
	}}
	invitations := &fakeInvitationRepository{}
	service := NewMemberService(
		WithInvitationUnitOfWork(&fakeInvitationUnitOfWork{members: members, invitations: invitations}),
		WithWorkspaceIdentityReader(&fakeWorkspaceIdentityReader{identity: workspacecontract.WorkspaceIdentity{ID: "workspace", Name: "Acme"}}),
		WithInvitationClock(func() time.Time { return fixedNow }),
		WithInvitationIDGenerator(func() string { return "invitation" }),
		WithInvitationLifetime(7*24*time.Hour),
	)

	result, err := service.CreateInvitation(
		contract.WithMemberActor(context.Background(), "owner-user"),
		contract.Member_CreateInvitationRequest{
			WorkspaceId: "workspace", Email: "  Invitee@Example.TEST ",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if invitations.created == nil {
		t.Fatal("pending invitation was not persisted")
	}
	created := *invitations.created
	if created.ID != "invitation" || created.InviteeEmail != "invitee@example.test" || created.Role != member.RoleMember {
		t.Fatalf("unexpected persisted invitation: %+v", created)
	}
	if !invitations.expired || invitations.email != "invitee@example.test" || !members.resolvedInvitee {
		t.Fatalf("duplicate preparation was not performed: invitation=%+v members=%+v", invitations, members)
	}
	if result.Id != "invitation" || result.WorkspaceName != "Acme" || result.InviterName != "Owner" || result.Status != "pending" {
		t.Fatalf("unexpected invitation response: %+v", result)
	}
}

func TestCreateInvitationAuthorizesBeforeValidatingInput(t *testing.T) {
	invitations := &fakeInvitationRepository{}
	service := NewMemberService(
		WithInvitationUnitOfWork(&fakeInvitationUnitOfWork{
			members:     &fakeMemberRepository{requester: member.Member{ID: "member", Role: member.RoleMember}},
			invitations: invitations,
		}),
		WithWorkspaceIdentityReader(&fakeWorkspaceIdentityReader{}),
		WithInvitationIDGenerator(func() string { return "invitation" }),
	)

	_, err := service.CreateInvitation(
		contract.WithMemberActor(context.Background(), "member-user"),
		contract.Member_CreateInvitationRequest{WorkspaceId: "workspace", Email: "invalid", Role: "owner"},
	)
	if !errors.Is(err, contract.ErrInsufficientWorkspaceRole) {
		t.Fatalf("CreateInvitation() error = %v", err)
	}
	if invitations.expired || invitations.created != nil {
		t.Fatal("unauthorized invitation request reached invitation persistence")
	}
}

func TestAuthorizeCreateInvitationRequiresWorkspaceManager(t *testing.T) {
	tests := []struct {
		name      string
		requester member.Member
		want      error
	}{
		{name: "owner", requester: member.Member{ID: "owner", Role: member.RoleOwner}},
		{name: "admin", requester: member.Member{ID: "admin", Role: member.RoleAdmin}},
		{name: "member", requester: member.Member{ID: "member", Role: member.RoleMember}, want: contract.ErrInsufficientWorkspaceRole},
		{name: "outsider", want: contract.ErrWorkspaceMembershipHidden},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := NewMemberService(WithInvitationUnitOfWork(&fakeInvitationUnitOfWork{
				members: &fakeMemberRepository{requester: test.requester}, invitations: &fakeInvitationRepository{},
			}))
			err := service.AuthorizeCreateInvitation(
				contract.WithMemberActor(context.Background(), "actor-user"),
				contract.Member_CreateInvitationRequest{WorkspaceId: "workspace"},
			)
			if !errors.Is(err, test.want) {
				t.Fatalf("AuthorizeCreateInvitation() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestCreateInvitationRejectsMemberAndPendingInvitationConflicts(t *testing.T) {
	tests := []struct {
		name         string
		memberExists bool
		pending      bool
		want         error
	}{
		{name: "existing member", memberExists: true, want: contract.ErrInviteeAlreadyMember},
		{name: "pending invitation", pending: true, want: contract.ErrInvitationAlreadyPending},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			members := &fakeMemberRepository{
				requester:     member.Member{ID: "owner", UserID: "owner-user", Role: member.RoleOwner},
				memberByEmail: test.memberExists,
			}
			invitations := &fakeInvitationRepository{pending: test.pending}
			service := NewMemberService(
				WithInvitationUnitOfWork(&fakeInvitationUnitOfWork{members: members, invitations: invitations}),
				WithWorkspaceIdentityReader(&fakeWorkspaceIdentityReader{}),
				WithInvitationIDGenerator(func() string { return "invitation" }),
			)

			_, err := service.CreateInvitation(
				contract.WithMemberActor(context.Background(), "owner-user"),
				contract.Member_CreateInvitationRequest{WorkspaceId: "workspace", Email: "invitee@example.test"},
			)
			if !errors.Is(err, test.want) {
				t.Fatalf("CreateInvitation() error = %v", err)
			}
			if invitations.created != nil {
				t.Fatal("conflicting invitation was persisted")
			}
		})
	}
}

func TestListMyInvitationsExpiresAndReturnsOwnedPendingInvitations(t *testing.T) {
	now := time.Date(2026, time.August, 2, 13, 0, 0, 0, time.UTC)
	members := &fakeMemberRepository{currentUser: member.UserIdentity{
		ID: "invitee-user", Email: "invitee@example.test",
	}}
	invitations := &fakeInvitationRepository{values: []invitation.Invitation{
		{
			ID: "invitation-a", WorkspaceID: "workspace-a", InviterID: "owner-a",
			InviteeEmail: "invitee@example.test", Role: member.RoleMember, Status: invitation.StatusPending,
			CreatedAt: invitation.NewTimestamp(now.Add(-time.Hour)), UpdatedAt: invitation.NewTimestamp(now.Add(-time.Hour)), ExpiresAt: invitation.NewTimestamp(now.Add(time.Hour)),
			InviterName: "Owner A", InviterEmail: "owner-a@example.test",
		},
		{
			ID: "invitation-b", WorkspaceID: "workspace-b", InviterID: "owner-b",
			InviteeEmail: "old@example.test", InviteeUserID: stringPointer("invitee-user"),
			Role: member.RoleAdmin, Status: invitation.StatusPending,
			CreatedAt: invitation.NewTimestamp(now), UpdatedAt: invitation.NewTimestamp(now), ExpiresAt: invitation.NewTimestamp(now.Add(2 * time.Hour)),
			InviterName: "Owner B", InviterEmail: "owner-b@example.test",
		},
	}}
	identities := &fakeWorkspaceIdentityReader{identities: map[string]workspacecontract.WorkspaceIdentity{
		"workspace-a": {ID: "workspace-a", Name: "Workspace A"},
		"workspace-b": {ID: "workspace-b", Name: "Workspace B"},
	}}
	service := NewMemberService(
		WithInvitationUnitOfWork(&fakeInvitationUnitOfWork{members: members, invitations: invitations}),
		WithWorkspaceIdentityReader(identities),
		WithInvitationClock(func() time.Time { return now }),
	)

	result, err := service.ListMyInvitations(
		contract.WithMemberActor(context.Background(), "invitee-user"),
		contract.Member_ListMyInvitationsRequest{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !invitations.expired || !invitations.listed || invitations.inviteeUserID != "invitee-user" || invitations.email != "invitee@example.test" {
		t.Fatalf("personal invitation repository calls were incomplete: %+v", invitations)
	}
	if !invitations.updatedAt.Equal(now) {
		t.Fatalf("expiration clock = %s, want %s", invitations.updatedAt, now)
	}
	if len(result.Invitations) != 2 || result.Invitations[0].WorkspaceName != "Workspace A" || result.Invitations[1].WorkspaceName != "Workspace B" {
		t.Fatalf("unexpected personal invitations: %+v", result.Invitations)
	}
}

func TestListMyInvitationsReportsMissingAuthenticatedUser(t *testing.T) {
	members := &fakeMemberRepository{}
	service := NewMemberService(
		WithInvitationUnitOfWork(&fakeInvitationUnitOfWork{members: members, invitations: &fakeInvitationRepository{}}),
		WithWorkspaceIdentityReader(&fakeWorkspaceIdentityReader{}),
	)

	_, err := service.ListMyInvitations(
		contract.WithMemberActor(context.Background(), "missing-user"),
		contract.Member_ListMyInvitationsRequest{},
	)
	if !errors.Is(err, contract.ErrAuthUserNotFound) {
		t.Fatalf("ListMyInvitations() error = %v", err)
	}
}

func TestGetMyInvitationReturnsOwnedInvitationInAnyState(t *testing.T) {
	now := time.Date(2026, time.August, 2, 13, 0, 0, 0, time.UTC)
	members := &fakeMemberRepository{currentUser: member.UserIdentity{
		ID: "invitee-user", Email: "invitee@example.test",
	}}
	invitations := &fakeInvitationRepository{found: invitation.Invitation{
		ID: "invitation", WorkspaceID: "workspace", InviterID: "owner-user",
		InviteeEmail: "invitee@example.test", Role: member.RoleMember, Status: invitation.StatusDeclined,
		CreatedAt: invitation.NewTimestamp(now.Add(-time.Hour)), UpdatedAt: invitation.NewTimestamp(now), ExpiresAt: invitation.NewTimestamp(now.Add(time.Hour)),
		InviterName: "Owner", InviterEmail: "owner@example.test",
	}}
	service := NewMemberService(
		WithInvitationUnitOfWork(&fakeInvitationUnitOfWork{members: members, invitations: invitations}),
		WithWorkspaceIdentityReader(&fakeWorkspaceIdentityReader{identity: workspacecontract.WorkspaceIdentity{
			ID: "workspace", Name: "Workspace",
		}}),
	)

	result, err := service.GetMyInvitation(
		contract.WithMemberActor(context.Background(), "invitee-user"),
		contract.Member_GetMyInvitationRequest{InvitationId: "invitation"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.Invitation == nil || result.Invitation.Status != "declined" || result.Invitation.WorkspaceName != "Workspace" {
		t.Fatalf("unexpected invitation: %+v", result.Invitation)
	}
}

func TestGetMyInvitationAllowsResolvedUserOwnership(t *testing.T) {
	members := &fakeMemberRepository{currentUser: member.UserIdentity{
		ID: "invitee-user", Email: "new@example.test",
	}}
	invitations := &fakeInvitationRepository{found: invitation.Invitation{
		ID: "invitation", WorkspaceID: "workspace", InviteeEmail: "old@example.test",
		InviteeUserID: stringPointer("invitee-user"), Role: member.RoleMember, Status: invitation.StatusPending,
	}}
	service := NewMemberService(
		WithInvitationUnitOfWork(&fakeInvitationUnitOfWork{members: members, invitations: invitations}),
		WithWorkspaceIdentityReader(&fakeWorkspaceIdentityReader{identity: workspacecontract.WorkspaceIdentity{ID: "workspace"}}),
	)

	_, err := service.GetMyInvitation(
		contract.WithMemberActor(context.Background(), "invitee-user"),
		contract.Member_GetMyInvitationRequest{InvitationId: "invitation"},
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetMyInvitationRejectsForeignInvitation(t *testing.T) {
	members := &fakeMemberRepository{currentUser: member.UserIdentity{
		ID: "actor-user", Email: "actor@example.test",
	}}
	invitations := &fakeInvitationRepository{found: invitation.Invitation{
		ID: "invitation", WorkspaceID: "workspace", InviteeEmail: "invitee@example.test",
		InviteeUserID: stringPointer("invitee-user"), Role: member.RoleMember, Status: invitation.StatusPending,
	}}
	identities := &fakeWorkspaceIdentityReader{identity: workspacecontract.WorkspaceIdentity{ID: "workspace"}}
	service := NewMemberService(
		WithInvitationUnitOfWork(&fakeInvitationUnitOfWork{members: members, invitations: invitations}),
		WithWorkspaceIdentityReader(identities),
	)

	_, err := service.GetMyInvitation(
		contract.WithMemberActor(context.Background(), "actor-user"),
		contract.Member_GetMyInvitationRequest{InvitationId: "invitation"},
	)
	if !errors.Is(err, contract.ErrInvitationForbidden) {
		t.Fatalf("GetMyInvitation() error = %v", err)
	}
	if identities.called {
		t.Fatal("foreign invitation resolved workspace identity")
	}
}

func TestAcceptInvitationPreservesOnboardingFailureContract(t *testing.T) {
	now := time.Date(2026, time.August, 2, 12, 0, 0, 0, time.UTC)
	members := &fakeMemberRepository{
		currentUser:   member.UserIdentity{ID: "invitee-user", Email: "invitee@example.test"},
		onboardingErr: errors.New("database unavailable"),
	}
	invitations := &fakeInvitationRepository{found: invitation.Invitation{
		ID: "invitation", WorkspaceID: "workspace", InviteeEmail: "invitee@example.test",
		Role: member.RoleMember, Status: invitation.StatusPending,
		ExpiresAt: invitation.NewTimestamp(now.Add(time.Hour)),
	}}
	service := NewMemberService(
		WithInvitationUnitOfWork(&fakeInvitationUnitOfWork{members: members, invitations: invitations}),
		WithWorkspaceIdentityReader(&fakeWorkspaceIdentityReader{identity: workspacecontract.WorkspaceIdentity{ID: "workspace"}}),
		WithInvitationClock(func() time.Time { return now }),
		WithMemberIDGenerator(func() string { return "member" }),
	)

	_, err := service.AcceptInvitation(
		contract.WithMemberActor(context.Background(), "invitee-user"),
		contract.Member_AcceptInvitationRequest{InvitationId: "invitation"},
	)
	if !errors.Is(err, contract.ErrInvitationOnboarding) {
		t.Fatalf("AcceptInvitation() error = %v", err)
	}
}

func stringPointer(value string) *string {
	return &value
}
