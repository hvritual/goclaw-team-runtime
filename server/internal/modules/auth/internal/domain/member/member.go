// Package member owns workspace membership role invariants.
package member

import "errors"

var (
	ErrInvalidRole               = errors.New("invalid member role")
	ErrInsufficientWorkspaceRole = errors.New("insufficient workspace role")
	ErrOwnerRoleRequiresOwner    = errors.New("owner role requires owner permission")
	ErrLastOwner                 = errors.New("workspace must retain an owner")
)

// Role is a workspace membership role.
type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)

// ParseRole converts the public role representation into a domain role.
func ParseRole(value string) (Role, error) {
	role := Role(value)
	switch role {
	case RoleOwner, RoleAdmin, RoleMember:
		return role, nil
	default:
		return "", ErrInvalidRole
	}
}

// Member is one user's workspace membership plus the profile projection needed
// by the existing member-management response.
type Member struct {
	ID          string
	WorkspaceID string
	UserID      string
	Role        Role
	CreatedAt   string
	Name        string
	Email       string
	AvatarURL   *string
}

// ValidateRoleChange enforces role-management and last-owner invariants.
func ValidateRoleChange(requester, target, next Role, ownerCount int) error {
	if requester != RoleOwner && requester != RoleAdmin {
		return ErrInsufficientWorkspaceRole
	}
	if (target == RoleOwner || next == RoleOwner) && requester != RoleOwner {
		return ErrOwnerRoleRequiresOwner
	}
	if target == RoleOwner && next != RoleOwner && ownerCount <= 1 {
		return ErrLastOwner
	}
	return nil
}
