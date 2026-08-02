// Package member owns workspace membership role invariants.
package member

import "errors"

var (
	ErrInvalidRole               = errors.New("invalid member role")
	ErrInsufficientWorkspaceRole = errors.New("insufficient workspace role")
	ErrOwnerRoleRequiresOwner    = errors.New("owner role requires owner permission")
	ErrOwnerRemovalRequiresOwner = errors.New("owner removal requires owner permission")
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
	if err := ValidateManager(requester); err != nil {
		return err
	}
	if (target == RoleOwner || next == RoleOwner) && requester != RoleOwner {
		return ErrOwnerRoleRequiresOwner
	}
	if target == RoleOwner && next != RoleOwner && ownerCount <= 1 {
		return ErrLastOwner
	}
	return nil
}

// ValidateRemoval enforces membership-management and last-owner invariants.
func ValidateRemoval(requester, target Role, ownerCount int) error {
	if err := ValidateManager(requester); err != nil {
		return err
	}
	if target == RoleOwner && requester != RoleOwner {
		return ErrOwnerRemovalRequiresOwner
	}
	return ValidateDeparture(target, ownerCount)
}

// ValidateManager requires the fixed Owner or Admin workspace role.
func ValidateManager(role Role) error {
	if role != RoleOwner && role != RoleAdmin {
		return ErrInsufficientWorkspaceRole
	}
	return nil
}

// ValidateDeparture prevents the final Owner from leaving a workspace.
func ValidateDeparture(role Role, ownerCount int) error {
	if role == RoleOwner && ownerCount <= 1 {
		return ErrLastOwner
	}
	return nil
}
