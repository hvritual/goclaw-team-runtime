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
	if value.CreatedAt.String() != now.UTC().Format(time.RFC3339Nano) || value.UpdatedAt.String() != now.UTC().Format(time.RFC3339Nano) {
		t.Fatalf("unexpected invitation timestamps: %+v", value)
	}
	expiresAt, err := value.ExpiresAt.Time()
	if err != nil {
		t.Fatal(err)
	}
	if !expiresAt.Equal(now.UTC().Add(7 * 24 * time.Hour)) {
		t.Fatalf("expires at = %s", value.ExpiresAt)
	}
}

func TestTimestampPreservesLegacyRepresentation(t *testing.T) {
	value := Timestamp("legacy-created-at")
	if value.String() != "legacy-created-at" {
		t.Fatalf("timestamp = %q", value.String())
	}
	if _, err := value.Time(); err == nil {
		t.Fatal("legacy timestamp unexpectedly parsed as RFC3339")
	}
}

func TestInvitationBelongsToResolvedUserOrEmail(t *testing.T) {
	userID := "invitee-user"
	value := Invitation{InviteeEmail: "invitee@example.test", InviteeUserID: &userID}
	if !value.BelongsTo("invitee-user", "changed@example.test") {
		t.Fatal("resolved invitee user did not own invitation")
	}
	if !value.BelongsTo("other-user", "invitee@example.test") {
		t.Fatal("matching invitee email did not own invitation")
	}
	if value.BelongsTo("other-user", "other@example.test") {
		t.Fatal("foreign user owned invitation")
	}
}

func TestInvitationDecisionValidation(t *testing.T) {
	now := time.Date(2026, time.August, 2, 12, 0, 0, 0, time.UTC)
	valid := Invitation{Status: StatusPending, ExpiresAt: NewTimestamp(now.Add(time.Hour))}
	if err := valid.ValidateAcceptance(now); err != nil {
		t.Fatalf("valid acceptance: %v", err)
	}
	if err := valid.ValidateDecline(); err != nil {
		t.Fatalf("valid decline: %v", err)
	}

	notPending := valid
	notPending.Status = StatusAccepted
	if !errors.Is(notPending.ValidateAcceptance(now), ErrNotPending) || !errors.Is(notPending.ValidateDecline(), ErrNotPending) {
		t.Fatal("non-pending invitation allowed a decision")
	}
	for _, expiresAt := range []Timestamp{NewTimestamp(now), Timestamp("legacy-invalid")} {
		expired := valid
		expired.ExpiresAt = expiresAt
		if !errors.Is(expired.ValidateAcceptance(now), ErrExpired) {
			t.Fatalf("expiry %q was accepted", expiresAt)
		}
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
