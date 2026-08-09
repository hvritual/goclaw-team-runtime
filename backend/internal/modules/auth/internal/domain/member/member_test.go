package member

import (
	"errors"
	"testing"
	"time"
)

func TestMemberRolePolicyMatrix(t *testing.T) {
	tests := []struct {
		name       string
		requester  Role
		target     Role
		next       Role
		ownerCount int
		wantErr    error
	}{
		{name: "owner promotes member", requester: RoleOwner, target: RoleMember, next: RoleOwner},
		{name: "admin updates member", requester: RoleAdmin, target: RoleMember, next: RoleAdmin},
		{name: "member cannot manage", requester: RoleMember, target: RoleMember, next: RoleAdmin, wantErr: ErrInsufficientWorkspaceRole},
		{name: "admin cannot promote owner", requester: RoleAdmin, target: RoleMember, next: RoleOwner, wantErr: ErrOwnerRoleRequiresOwner},
		{name: "admin cannot demote owner", requester: RoleAdmin, target: RoleOwner, next: RoleMember, ownerCount: 2, wantErr: ErrOwnerRoleRequiresOwner},
		{name: "last owner remains", requester: RoleOwner, target: RoleOwner, next: RoleAdmin, ownerCount: 1, wantErr: ErrLastOwner},
		{name: "one of two owners changes", requester: RoleOwner, target: RoleOwner, next: RoleAdmin, ownerCount: 2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := ValidateRoleChange(test.requester, test.target, test.next, test.ownerCount); !errors.Is(err, test.wantErr) {
				t.Fatalf("ValidateRoleChange() error = %v, want %v", err, test.wantErr)
			}
		})
	}
}

func TestInitialOwnerNormalizesIdentityAndProtectsProfileProjection(t *testing.T) {
	avatar := "avatar.png"
	user, err := RehydrateUser(" user-1 ", "Owner", "owner@example.test", &avatar)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 3, 12, 0, 0, 7, time.FixedZone("test", 8*60*60))
	owner, err := NewInitialOwner(" member-1 ", " workspace-1 ", user, now)
	if err != nil {
		t.Fatal(err)
	}
	if owner.ID() != "member-1" || owner.WorkspaceID() != "workspace-1" || owner.UserID() != "user-1" || owner.Role() != RoleOwner {
		t.Fatalf("initial owner = %q/%q/%q/%q", owner.ID(), owner.WorkspaceID(), owner.UserID(), owner.Role())
	}
	if owner.CreatedAt().Location() != time.UTC || owner.Name() != "Owner" || owner.Email() != "owner@example.test" {
		t.Fatalf("initial owner projection = %+v", owner)
	}
	returnedAvatar := owner.AvatarURL()
	*returnedAvatar = "changed"
	if *owner.AvatarURL() != "avatar.png" {
		t.Fatal("Member exposed mutable avatar projection")
	}
}

func TestParseRolePreservesLowercaseCompatibility(t *testing.T) {
	for _, role := range []Role{RoleOwner, RoleAdmin, RoleMember} {
		parsed, err := ParseRole(" " + string(role) + " ")
		if err != nil || parsed != role {
			t.Fatalf("ParseRole(%q) = %q, %v", role, parsed, err)
		}
	}
	for _, invalid := range []string{"", "viewer", "OWNER"} {
		if _, err := ParseRole(invalid); !errors.Is(err, ErrInvalidRole) {
			t.Fatalf("ParseRole(%q) error = %v", invalid, err)
		}
	}
}
