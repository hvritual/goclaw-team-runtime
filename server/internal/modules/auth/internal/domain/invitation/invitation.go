package invitation

import (
	"errors"
	"strings"
	"time"

	"github.com/multica-ai/multica/server/internal/modules/auth/internal/domain/member"
)

var (
	ErrInvalidStatus = errors.New("invalid invitation status")
	ErrInvalidEmail  = errors.New("invalid invitation email")
	ErrInvalidRole   = errors.New("invalid invitation role")
)

type Status string

const (
	StatusPending  Status = "pending"
	StatusAccepted Status = "accepted"
	StatusDeclined Status = "declined"
	StatusExpired  Status = "expired"
	StatusRevoked  Status = "revoked"
)

func ParseStatus(value string) (Status, error) {
	status := Status(value)
	switch status {
	case StatusPending, StatusAccepted, StatusDeclined, StatusExpired, StatusRevoked:
		return status, nil
	default:
		return "", ErrInvalidStatus
	}
}

// Invitation is the aggregate root and its Auth-owned inviter projection.
type Invitation struct {
	ID            string
	WorkspaceID   string
	InviterID     string
	InviteeEmail  string
	InviteeUserID *string
	Role          member.Role
	Status        Status
	CreatedAt     time.Time
	UpdatedAt     time.Time
	ExpiresAt     time.Time
	InviterName   string
	InviterEmail  string
}

// NewPending creates the initial Invitation state. Workspace authorization,
// duplicate checks and persistence remain application concerns.
func NewPending(
	id string,
	workspaceID string,
	inviterID string,
	inviteeEmail string,
	role member.Role,
	inviteeUserID *string,
	now time.Time,
	lifetime time.Duration,
) (Invitation, error) {
	email := strings.ToLower(strings.TrimSpace(inviteeEmail))
	if !strings.Contains(email, "@") {
		return Invitation{}, ErrInvalidEmail
	}
	if role != member.RoleAdmin && role != member.RoleMember {
		return Invitation{}, ErrInvalidRole
	}
	timestamp := now.UTC()
	return Invitation{
		ID:            id,
		WorkspaceID:   workspaceID,
		InviterID:     inviterID,
		InviteeEmail:  email,
		InviteeUserID: inviteeUserID,
		Role:          role,
		Status:        StatusPending,
		CreatedAt:     timestamp,
		UpdatedAt:     timestamp,
		ExpiresAt:     timestamp.Add(lifetime),
	}, nil
}
