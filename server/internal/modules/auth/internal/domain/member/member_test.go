package member

import (
	"errors"
	"testing"
)

func TestParseRole(t *testing.T) {
	for _, role := range []Role{RoleOwner, RoleAdmin, RoleMember} {
		parsed, err := ParseRole(string(role))
		if err != nil || parsed != role {
			t.Fatalf("ParseRole(%q) = %q, %v", role, parsed, err)
		}
	}
	if _, err := ParseRole("viewer"); !errors.Is(err, ErrInvalidRole) {
		t.Fatalf("ParseRole(viewer) error = %v", err)
	}
}

func TestValidateRoleChange(t *testing.T) {
	tests := []struct {
		name       string
		requester  Role
		target     Role
		next       Role
		ownerCount int
		wantErr    error
	}{
		{name: "owner promotes member", requester: RoleOwner, target: RoleMember, next: RoleOwner, ownerCount: 1},
		{name: "admin changes member", requester: RoleAdmin, target: RoleMember, next: RoleAdmin, ownerCount: 1},
		{name: "member cannot manage roles", requester: RoleMember, target: RoleMember, next: RoleAdmin, ownerCount: 1, wantErr: ErrInsufficientWorkspaceRole},
		{name: "admin cannot promote owner", requester: RoleAdmin, target: RoleMember, next: RoleOwner, ownerCount: 1, wantErr: ErrOwnerRoleRequiresOwner},
		{name: "admin cannot demote owner", requester: RoleAdmin, target: RoleOwner, next: RoleMember, ownerCount: 2, wantErr: ErrOwnerRoleRequiresOwner},
		{name: "sole owner cannot be demoted", requester: RoleOwner, target: RoleOwner, next: RoleAdmin, ownerCount: 1, wantErr: ErrLastOwner},
		{name: "one of two owners can be demoted", requester: RoleOwner, target: RoleOwner, next: RoleAdmin, ownerCount: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRoleChange(tt.requester, tt.target, tt.next, tt.ownerCount)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ValidateRoleChange() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateRemoval(t *testing.T) {
	tests := []struct {
		name       string
		requester  Role
		target     Role
		ownerCount int
		want       error
	}{
		{name: "owner removes member", requester: RoleOwner, target: RoleMember},
		{name: "admin removes member", requester: RoleAdmin, target: RoleMember},
		{name: "member cannot remove", requester: RoleMember, target: RoleMember, want: ErrInsufficientWorkspaceRole},
		{name: "admin cannot remove owner", requester: RoleAdmin, target: RoleOwner, ownerCount: 2, want: ErrOwnerRemovalRequiresOwner},
		{name: "last owner remains", requester: RoleOwner, target: RoleOwner, ownerCount: 1, want: ErrLastOwner},
		{name: "owner removes another owner", requester: RoleOwner, target: RoleOwner, ownerCount: 2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := ValidateRemoval(test.requester, test.target, test.ownerCount); !errors.Is(err, test.want) {
				t.Fatalf("ValidateRemoval() error = %v, want %v", err, test.want)
			}
		})
	}
}
