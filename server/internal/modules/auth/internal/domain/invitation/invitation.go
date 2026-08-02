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

// Timestamp preserves the stored public timestamp representation. New domain
// values are canonical RFC3339Nano, while reads remain compatible with legacy
// rows that predate timestamp validation.
type Timestamp string

func NewTimestamp(value time.Time) Timestamp {
	return Timestamp(value.UTC().Format(time.RFC3339Nano))
}

func (t Timestamp) String() string { return string(t) }

func (t Timestamp) Time() (time.Time, error) {
	return time.Parse(time.RFC3339Nano, string(t))
}

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
	CreatedAt     Timestamp
	UpdatedAt     Timestamp
	ExpiresAt     Timestamp
	InviterName   string
	InviterEmail  string
}

// BelongsTo accepts either a resolved invitee identity or the invited email,
// preserving invitations created before the user account existed.
func (i Invitation) BelongsTo(userID, email string) bool {
	if i.InviteeUserID != nil && *i.InviteeUserID == userID {
		return true
	}
	return i.InviteeEmail == email
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
		CreatedAt:     NewTimestamp(timestamp),
		UpdatedAt:     NewTimestamp(timestamp),
		ExpiresAt:     NewTimestamp(timestamp.Add(lifetime)),
	}, nil
}
