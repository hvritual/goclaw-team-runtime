package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/auth/contract"
	memberDomain "github.com/hvritual/workspace/internal/modules/auth/internal/domain/member"
)

type memberRepositoryStub struct {
	root       *memberDomain.WorkspaceRoot
	users      map[string]memberDomain.User
	members    map[string]memberDomain.Member
	listed     []memberDomain.Member
	ownerCount int
	updated    []memberDomain.Member
	findCalls  int
	listCalls  int
}

func (r *memberRepositoryStub) CreateWorkspaceRoot(_ context.Context, value memberDomain.WorkspaceRoot) error {
	r.root = &value
	return nil
}

func (r *memberRepositoryStub) FindWorkspaceRoot(context.Context, string) (memberDomain.WorkspaceRoot, error) {
	if r.root == nil {
		return memberDomain.WorkspaceRoot{}, ErrWorkspaceRootNotFound
	}
	return *r.root, nil
}

func (r *memberRepositoryStub) FindUserByID(_ context.Context, userID string) (memberDomain.User, error) {
	value, ok := r.users[userID]
	if !ok {
		return memberDomain.User{}, ErrAuthUserRecordNotFound
	}
	return value, nil
}

func (r *memberRepositoryStub) Create(_ context.Context, value memberDomain.Member) error {
	if r.members == nil {
		r.members = make(map[string]memberDomain.Member)
	}
	r.members[value.ID()] = value
	return nil
}

func (r *memberRepositoryStub) FindByUserAndWorkspace(_ context.Context, userID, workspaceID string) (memberDomain.Member, error) {
	r.findCalls++
	for _, value := range r.members {
		if value.UserID() == userID && value.WorkspaceID() == workspaceID {
			return value, nil
		}
	}
	return memberDomain.Member{}, ErrMemberRecordNotFound
}

func (r *memberRepositoryStub) FindByIDAndWorkspace(_ context.Context, memberID, workspaceID string) (memberDomain.Member, error) {
	r.findCalls++
	value, ok := r.members[memberID]
	if !ok || value.WorkspaceID() != workspaceID {
		return memberDomain.Member{}, ErrMemberRecordNotFound
	}
	return value, nil
}

func (r *memberRepositoryStub) ListByWorkspace(context.Context, string) ([]memberDomain.Member, error) {
	r.listCalls++
	return append([]memberDomain.Member(nil), r.listed...), nil
}

func (r *memberRepositoryStub) CountOwners(context.Context, string) (int, error) {
	return r.ownerCount, nil
}

func (r *memberRepositoryStub) UpdateRole(_ context.Context, value memberDomain.Member) error {
	r.updated = append(r.updated, value)
	r.members[value.ID()] = value
	return nil
}

type memberUnitOfWorkStub struct {
	repository *memberRepositoryStub
	calls      int
}

func (u *memberUnitOfWorkStub) WithinTransaction(ctx context.Context, operation func(MemberRepository) error) error {
	u.calls++
	return operation(u.repository)
}

func newMemberUseCaseForTest(t *testing.T, repository *memberRepositoryStub, ids ...string) *MemberUseCase {
	t.Helper()
	index := 0
	service, err := NewMemberUseCase(
		&memberUnitOfWorkStub{repository: repository},
		func(context.Context) (string, error) {
			if index >= len(ids) {
				return "member-unused", nil
			}
			value := ids[index]
			index++
			return value, nil
		},
		func() time.Time { return time.Date(2026, 8, 3, 12, 0, 0, 7, time.UTC) },
	)
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func TestProvisionWorkspaceOwnerIsIdempotentForSameUser(t *testing.T) {
	user, _ := memberDomain.RehydrateUser("user-1", "Owner", "owner@example.test", nil)
	repository := &memberRepositoryStub{users: map[string]memberDomain.User{"user-1": user}, members: make(map[string]memberDomain.Member)}
	service := newMemberUseCaseForTest(t, repository, "member-1", "member-replay")

	first, err := service.ProvisionWorkspaceOwner(context.Background(), contract.ProvisionWorkspaceOwnerRequest{WorkspaceId: " workspace-1 ", UserId: " user-1 "})
	if err != nil || first.Member == nil || !first.Created || first.Member.Id != "member-1" || first.Member.Role != "owner" {
		t.Fatalf("first ProvisionWorkspaceOwner() = %+v, %v", first, err)
	}
	replay, err := service.ProvisionWorkspaceOwner(context.Background(), contract.ProvisionWorkspaceOwnerRequest{WorkspaceId: "workspace-1", UserId: "user-1"})
	if err != nil || replay.Member == nil || replay.Created || replay.Member.Id != "member-1" {
		t.Fatalf("replayed ProvisionWorkspaceOwner() = %+v, %v", replay, err)
	}
	if len(repository.members) != 1 {
		t.Fatalf("member count = %d", len(repository.members))
	}
	replayWithoutGenerator, err := NewMemberUseCase(
		&memberUnitOfWorkStub{repository: repository},
		func(context.Context) (string, error) { return "", errors.New("generator unavailable") },
		time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	if response, err := replayWithoutGenerator.ProvisionWorkspaceOwner(context.Background(), contract.ProvisionWorkspaceOwnerRequest{WorkspaceId: "workspace-1", UserId: "user-1"}); err != nil || response.Member == nil || response.Created {
		t.Fatalf("generator-independent replay = %+v, %v", response, err)
	}
}

func TestProvisionWorkspaceOwnerRejectsDifferentInitializerAndMissingUser(t *testing.T) {
	user, _ := memberDomain.RehydrateUser("user-1", "Owner", "owner@example.test", nil)
	repository := &memberRepositoryStub{users: map[string]memberDomain.User{"user-1": user}, members: make(map[string]memberDomain.Member)}
	service := newMemberUseCaseForTest(t, repository, "member-1", "member-2")
	if _, err := service.ProvisionWorkspaceOwner(context.Background(), contract.ProvisionWorkspaceOwnerRequest{WorkspaceId: "workspace-1", UserId: "user-1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ProvisionWorkspaceOwner(context.Background(), contract.ProvisionWorkspaceOwnerRequest{WorkspaceId: "workspace-1", UserId: "user-2"}); !errors.Is(err, contract.ErrWorkspaceMembershipInitialized) {
		t.Fatalf("different initializer error = %v", err)
	}
	missingRepository := &memberRepositoryStub{users: map[string]memberDomain.User{}, members: make(map[string]memberDomain.Member)}
	missingService := newMemberUseCaseForTest(t, missingRepository, "member-missing")
	if _, err := missingService.ProvisionWorkspaceOwner(context.Background(), contract.ProvisionWorkspaceOwnerRequest{WorkspaceId: "workspace-2", UserId: "missing"}); !errors.Is(err, contract.ErrAuthUserNotFound) {
		t.Fatalf("missing user error = %v", err)
	}
}

func TestListMembersRequiresWorkspaceMembership(t *testing.T) {
	owner, _ := memberDomain.Rehydrate("owner-member", "workspace-1", "owner-user", "owner", time.Now(), "Owner", "owner@example.test", nil)
	member, _ := memberDomain.Rehydrate("target-member", "workspace-1", "member-user", "member", time.Now(), "Member", "member@example.test", nil)
	repository := &memberRepositoryStub{members: map[string]memberDomain.Member{owner.ID(): owner, member.ID(): member}, listed: []memberDomain.Member{owner, member}}
	service := newMemberUseCaseForTest(t, repository)

	listed, err := service.ListMembers(contract.WithMemberActor(context.Background(), "owner-user"), contract.ListMembersRequest{WorkspaceId: "workspace-1"})
	if err != nil || len(listed.Members) != 2 || listed.Members[1].Email != "member@example.test" {
		t.Fatalf("ListMembers() = %+v, %v", listed, err)
	}
	if _, err := service.ListMembers(contract.WithMemberActor(context.Background(), "outsider"), contract.ListMembersRequest{WorkspaceId: "workspace-1"}); !errors.Is(err, contract.ErrWorkspaceMembershipHidden) {
		t.Fatalf("outsider ListMembers() error = %v", err)
	}
	if _, err := service.ListMembers(context.Background(), contract.ListMembersRequest{WorkspaceId: "workspace-1"}); !errors.Is(err, contract.ErrMemberActorRequired) {
		t.Fatalf("anonymous ListMembers() error = %v", err)
	}
}

func TestUpdateMemberRoleEnforcesManagerAndLastOwnerPolicies(t *testing.T) {
	tests := []struct {
		name       string
		actorRole  string
		targetRole string
		nextRole   string
		ownerCount int
		wantErr    error
	}{
		{name: "owner promotes member", actorRole: "owner", targetRole: "member", nextRole: "owner"},
		{name: "admin updates member", actorRole: "admin", targetRole: "member", nextRole: "admin"},
		{name: "member cannot manage", actorRole: "member", targetRole: "member", nextRole: "admin", wantErr: contract.ErrInsufficientWorkspaceRole},
		{name: "admin cannot manage owner role", actorRole: "admin", targetRole: "owner", nextRole: "member", ownerCount: 2, wantErr: contract.ErrOwnerRoleRequiresOwner},
		{name: "last owner remains", actorRole: "owner", targetRole: "owner", nextRole: "admin", ownerCount: 1, wantErr: contract.ErrLastWorkspaceOwner},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actor, _ := memberDomain.Rehydrate("actor-member", "workspace-1", "actor-user", test.actorRole, time.Now(), "Actor", "actor@example.test", nil)
			target, _ := memberDomain.Rehydrate("target-member", "workspace-1", "target-user", test.targetRole, time.Now(), "Target", "target@example.test", nil)
			repository := &memberRepositoryStub{
				members: map[string]memberDomain.Member{actor.ID(): actor, target.ID(): target}, ownerCount: test.ownerCount,
			}
			service := newMemberUseCaseForTest(t, repository)
			response, err := service.UpdateMemberRole(
				contract.WithMemberActor(context.Background(), "actor-user"),
				contract.UpdateMemberRoleRequest{WorkspaceId: "workspace-1", MemberId: "target-member", Role: test.nextRole},
			)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("UpdateMemberRole() error = %v, want %v", err, test.wantErr)
			}
			if test.wantErr == nil && (response.Member == nil || response.Member.Role != test.nextRole || len(repository.updated) != 1) {
				t.Fatalf("UpdateMemberRole() = %+v, writes=%d", response.Member, len(repository.updated))
			}
			if test.wantErr != nil && len(repository.updated) != 0 {
				t.Fatalf("rejected update writes = %d", len(repository.updated))
			}
		})
	}
}
