package teamcontrol

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAccessCredentialBootstrapAuthenticationExpiryAndRevocation(t *testing.T) {
	root := t.TempDir()
	service, err := Open(root)
	require.NoError(t, err)
	alice, err := service.CreateUser(CreateUserInput{ID: "alice", DisplayName: "Alice"})
	require.NoError(t, err)
	bob, err := service.CreateUser(CreateUserInput{ID: "bob", DisplayName: "Bob"})
	require.NoError(t, err)

	const aliceToken = "alice-bootstrap-token-0123456789"
	aliceCredential, err := service.RegisterAccessToken(
		alice.ID, alice.ID, "bootstrap", aliceToken, nil,
	)
	require.NoError(t, err)
	require.NotEqual(t, aliceToken, aliceCredential.TokenSHA256)
	authenticated, err := service.AuthenticateAccessToken(aliceToken)
	require.NoError(t, err)
	require.Equal(t, alice.ID, authenticated.ID)

	team, err := service.CreateTeam(alice.ID, CreateTeamInput{ID: "team", Name: "Team"})
	require.NoError(t, err)
	_, err = service.AddTeamMember(alice.ID, team.ID, AddTeamMemberInput{
		UserID: bob.ID, Role: TeamRegularMember,
	})
	require.NoError(t, err)

	const bobToken = "bob-access-token-0123456789012"
	bobCredential, err := service.RegisterAccessToken(
		alice.ID, bob.ID, "bob laptop", bobToken, nil,
	)
	require.NoError(t, err)
	authenticated, err = service.AuthenticateAccessToken(bobToken)
	require.NoError(t, err)
	require.Equal(t, bob.ID, authenticated.ID)

	_, err = service.RegisterAccessToken(
		bob.ID, bob.ID, "self rotation", "bob-cannot-self-rotate-012345", nil,
	)
	require.ErrorIs(t, err, ErrForbidden)

	expiry := time.Now().UTC().Add(30 * time.Millisecond)
	const expiringToken = "bob-expiring-token-0123456789"
	_, err = service.RegisterAccessToken(
		alice.ID, bob.ID, "temporary", expiringToken, &expiry,
	)
	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)
	_, err = service.AuthenticateAccessToken(expiringToken)
	require.ErrorIs(t, err, ErrAuthentication)

	revoked, err := service.RevokeAccessToken(alice.ID, bobCredential.ID)
	require.NoError(t, err)
	require.NotNil(t, revoked.RevokedAt)
	_, err = service.AuthenticateAccessToken(bobToken)
	require.ErrorIs(t, err, ErrAuthentication)

	data, err := os.ReadFile(filepath.Join(root, stateFileName))
	require.NoError(t, err)
	require.False(t, strings.Contains(string(data), aliceToken))
	require.False(t, strings.Contains(string(data), bobToken))
}

func TestBootstrapFirstUserIsAtomicAndSingleUse(t *testing.T) {
	service, err := Open(t.TempDir())
	require.NoError(t, err)

	const token = "0123456789abcdef0123456789abcdef"
	user, credential, err := service.BootstrapFirstUser(
		CreateUserInput{
			ID:          "alice",
			DisplayName: "Alice",
			Email:       "alice@example.com",
		},
		"bootstrap",
		token,
	)
	require.NoError(t, err)
	require.Equal(t, "alice", user.ID)
	require.Equal(t, user.ID, credential.UserID)
	require.NotEmpty(t, credential.TokenSHA256)
	require.NotEqual(t, token, credential.TokenSHA256)

	_, _, err = service.BootstrapFirstUser(
		CreateUserInput{ID: "bob", DisplayName: "Bob"},
		"bootstrap",
		"abcdef0123456789abcdef0123456789",
	)
	require.Error(t, err)

	authenticated, err := service.AuthenticateAccessToken(token)
	require.NoError(t, err)
	require.Equal(t, user.ID, authenticated.ID)
}

func TestCredentialAdminMustControlEveryTargetTeam(t *testing.T) {
	service, err := Open(t.TempDir())
	require.NoError(t, err)
	alice, err := service.CreateUser(CreateUserInput{ID: "alice", DisplayName: "Alice"})
	require.NoError(t, err)
	bob, err := service.CreateUser(CreateUserInput{ID: "bob", DisplayName: "Bob"})
	require.NoError(t, err)
	carol, err := service.CreateUser(CreateUserInput{ID: "carol", DisplayName: "Carol"})
	require.NoError(t, err)

	_, err = service.RegisterAccessToken(
		alice.ID,
		alice.ID,
		"bootstrap",
		"alice-bootstrap-token-0123456789",
		nil,
	)
	require.NoError(t, err)
	teamA, err := service.CreateTeam(
		alice.ID,
		CreateTeamInput{ID: "team-a", Name: "Team A"},
	)
	require.NoError(t, err)
	_, err = service.AddTeamMember(alice.ID, teamA.ID, AddTeamMemberInput{
		UserID: bob.ID,
		Role:   TeamRegularMember,
	})
	require.NoError(t, err)
	teamB, err := service.CreateTeam(
		carol.ID,
		CreateTeamInput{ID: "team-b", Name: "Team B"},
	)
	require.NoError(t, err)
	_, err = service.AddTeamMember(carol.ID, teamB.ID, AddTeamMemberInput{
		UserID: bob.ID,
		Role:   TeamRegularMember,
	})
	require.NoError(t, err)

	_, err = service.RegisterAccessToken(
		alice.ID,
		bob.ID,
		"cross-team forbidden",
		"bob-cross-team-token-0123456789",
		nil,
	)
	require.ErrorIs(t, err, ErrForbidden)
	require.ErrorIs(
		t,
		service.AuthorizeUserAdministration(alice.ID, bob.ID),
		ErrForbidden,
	)
}
