package invitation

import (
	"errors"
	"time"

	"github.com/multica-ai/multica/server/internal/modules/auth/internal/domain/member"
)

var ErrInvalidStatus = errors.New("invalid invitation status")

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
