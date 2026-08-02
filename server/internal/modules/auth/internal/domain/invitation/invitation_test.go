package invitation

import (
	"errors"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/modules/auth/internal/domain/member"
)

func TestNewPendingNormalizesEmailAndSetsLifecycle(t *testing.T) {
	now := time.Date(2026, time.August, 2, 10, 30, 0, 0, time.FixedZone("CST", 8*60*60))
	value, err := NewPending(
		"invitation", "workspace", "owner-user", "  Invitee@Example.TEST ",
		member.RoleMember, nil, now, 7*24*time.Hour,
	)
	if err != nil {
		t.Fatal(err)
	}
	if value.InviteeEmail != "invitee@example.test" || value.Status != StatusPending {
		t.Fatalf("unexpected invitation identity: %+v", value)
	}
	if !value.CreatedAt.Equal(now.UTC()) || !value.UpdatedAt.Equal(now.UTC()) {
		t.Fatalf("unexpected invitation timestamps: %+v", value)
	}
	if !value.ExpiresAt.Equal(now.UTC().Add(7 * 24 * time.Hour)) {
		t.Fatalf("expires at = %s", value.ExpiresAt)
	}
}

func TestNewPendingRejectsInvalidInviteeAndOwnerRole(t *testing.T) {
	tests := []struct {
		name  string
		email string
		role  member.Role
		want  error
	}{
		{name: "invalid email", email: "not-an-email", role: member.RoleMember, want: ErrInvalidEmail},
		{name: "owner role", email: "invitee@example.test", role: member.RoleOwner, want: ErrInvalidRole},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewPending(
				"invitation", "workspace", "owner-user", test.email,
				test.role, nil, time.Now(), 7*24*time.Hour,
			)
			if !errors.Is(err, test.want) {
				t.Fatalf("NewPending() error = %v", err)
			}
		})
	}
}
