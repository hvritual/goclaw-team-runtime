package teamcontrol

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"
	"time"
)

// BootstrapFirstUser atomically creates the first active user and its first
// personal access token. It is intended only for a local control-plane CLI;
// remote callers must never expose this unauthenticated operation.
func (s *Service) BootstrapFirstUser(
	input CreateUserInput,
	label, plaintext string,
) (User, AccessCredential, error) {
	id, err := normalizeID(input.ID, "usr")
	if err != nil {
		return User{}, AccessCredential{}, err
	}
	if isNonLoginPrincipal(id) {
		return User{}, AccessCredential{}, fmt.Errorf(
			"user id %q is reserved for a non-login service principal",
			id,
		)
	}
	name, err := requireText(input.DisplayName, "display_name", 200)
	if err != nil {
		return User{}, AccessCredential{}, err
	}
	email, err := validateEmail(input.Email)
	if err != nil {
		return User{}, AccessCredential{}, err
	}
	label, err = requireText(label, "label", 200)
	if err != nil {
		return User{}, AccessCredential{}, err
	}
	if len(plaintext) < 16 {
		return User{}, AccessCredential{}, fmt.Errorf(
			"access token must contain at least 16 bytes",
		)
	}
	if len(plaintext) > 4096 {
		return User{}, AccessCredential{}, fmt.Errorf(
			"access token exceeds 4096 bytes",
		)
	}
	var user User
	var credential AccessCredential
	err = s.store.update(func(st *state) error {
		if len(st.Users) != 0 || len(st.AccessCredentials) != 0 {
			return conflict("team control is already initialized")
		}
		now := time.Now().UTC()
		user = User{
			ID:          id,
			DisplayName: name,
			Email:       email,
			Status:      UserActive,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		credential = AccessCredential{
			ID:          newID("credential"),
			UserID:      id,
			Label:       label,
			TokenSHA256: hashAccessToken(plaintext),
			CreatedBy:   id,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		st.Users[id] = user
		st.AccessCredentials[credential.ID] = credential
		return nil
	})
	return user, credential, err
}

// RegisterAccessToken stores only a SHA-256 digest. The plaintext is never
// returned or persisted. Bootstrap is deliberately narrow: only the earliest
// active user may register the first credential for itself.
func (s *Service) RegisterAccessToken(
	actorID, userID, label, plaintext string,
	expiresAt *time.Time,
) (AccessCredential, error) {
	actorID, err := requireID(actorID, "actor_id")
	if err != nil {
		return AccessCredential{}, err
	}
	userID, err = requireID(userID, "user_id")
	if err != nil {
		return AccessCredential{}, err
	}
	if isNonLoginPrincipal(userID) {
		return AccessCredential{}, fmt.Errorf(
			"non-login service principal %q cannot receive an access token",
			userID,
		)
	}
	label, err = requireText(label, "label", 200)
	if err != nil {
		return AccessCredential{}, err
	}
	if len(plaintext) < 16 {
		return AccessCredential{}, fmt.Errorf("access token must contain at least 16 bytes")
	}
	if len(plaintext) > 4096 {
		return AccessCredential{}, fmt.Errorf("access token exceeds 4096 bytes")
	}
	var expiry *time.Time
	if expiresAt != nil {
		value := expiresAt.UTC()
		if !value.After(time.Now().UTC()) {
			return AccessCredential{}, fmt.Errorf("expires_at must be in the future")
		}
		expiry = &value
	}
	tokenHash := hashAccessToken(plaintext)
	var created AccessCredential
	err = s.store.update(func(st *state) error {
		if err := requireActiveUser(st, userID); err != nil {
			return err
		}
		bootstrap := len(st.AccessCredentials) == 0 &&
			actorID == userID &&
			firstActiveUserID(st) == userID
		if !bootstrap {
			if err := authorizeCredentialAdmin(st, actorID, userID); err != nil {
				return err
			}
		}
		for _, credential := range st.AccessCredentials {
			if subtle.ConstantTimeCompare(
				[]byte(credential.TokenSHA256),
				[]byte(tokenHash),
			) == 1 {
				return conflict("access token is already registered")
			}
		}
		now := time.Now().UTC()
		created = AccessCredential{
			ID:          newID("credential"),
			UserID:      userID,
			Label:       label,
			TokenSHA256: tokenHash,
			ExpiresAt:   expiry,
			CreatedBy:   actorID,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		st.AccessCredentials[created.ID] = created
		return nil
	})
	return created, err
}

func (s *Service) AuthenticateAccessToken(plaintext string) (User, error) {
	if plaintext == "" {
		return User{}, ErrAuthentication
	}
	tokenHash := hashAccessToken(plaintext)
	var authenticated User
	err := s.store.view(func(st state) error {
		now := time.Now().UTC()
		var matched *AccessCredential
		for _, credential := range st.AccessCredentials {
			if subtle.ConstantTimeCompare(
				[]byte(credential.TokenSHA256),
				[]byte(tokenHash),
			) == 1 {
				copy := credential
				matched = &copy
			}
		}
		if matched == nil || matched.RevokedAt != nil ||
			(matched.ExpiresAt != nil && !matched.ExpiresAt.After(now)) {
			return ErrAuthentication
		}
		user, ok := st.Users[matched.UserID]
		if !ok || user.Status != UserActive || isNonLoginPrincipal(user.ID) {
			return ErrAuthentication
		}
		authenticated = user
		return nil
	})
	return authenticated, err
}

func (s *Service) RevokeAccessToken(
	actorID, credentialID string,
) (AccessCredential, error) {
	actorID, err := requireID(actorID, "actor_id")
	if err != nil {
		return AccessCredential{}, err
	}
	credentialID, err = requireID(credentialID, "credential_id")
	if err != nil {
		return AccessCredential{}, err
	}
	var updated AccessCredential
	err = s.store.update(func(st *state) error {
		credential, ok := st.AccessCredentials[credentialID]
		if !ok {
			return entityNotFound("access credential", credentialID)
		}
		if err := authorizeCredentialAdmin(st, actorID, credential.UserID); err != nil {
			return err
		}
		if credential.RevokedAt != nil {
			return conflict("access credential %q is already revoked", credentialID)
		}
		now := time.Now().UTC()
		credential.RevokedAt = &now
		credential.UpdatedAt = now
		st.AccessCredentials[credentialID] = credential
		updated = credential
		return nil
	})
	return updated, err
}

func (s *Service) ListAccessCredentials(
	actorID, userID string,
) ([]AccessCredential, error) {
	actorID, err := requireID(actorID, "actor_id")
	if err != nil {
		return nil, err
	}
	userID, err = requireID(userID, "user_id")
	if err != nil {
		return nil, err
	}
	var result []AccessCredential
	err = s.store.view(func(st state) error {
		if err := authorizeCredentialAdmin(&st, actorID, userID); err != nil {
			return err
		}
		for _, credential := range st.AccessCredentials {
			if credential.UserID == userID {
				result = append(result, credential)
			}
		}
		slices.SortFunc(result, func(a, b AccessCredential) int {
			return strings.Compare(a.ID, b.ID)
		})
		return nil
	})
	return result, err
}

func authorizeCredentialAdmin(st *state, actorID, targetUserID string) error {
	if err := requireActiveUser(st, actorID); err != nil {
		return err
	}
	if err := requireActiveUser(st, targetUserID); err != nil {
		return err
	}
	targetTeams := make(map[string]struct{})
	for _, membership := range st.TeamMemberships {
		if membership.UserID == targetUserID && membership.Status == MembershipActive {
			targetTeams[membership.TeamID] = struct{}{}
		}
	}
	if len(targetTeams) == 0 {
		return fmt.Errorf(
			"%w: target user has no active team membership",
			ErrForbidden,
		)
	}
	for teamID := range targetTeams {
		authorized := false
		for _, membership := range st.TeamMemberships {
			if membership.TeamID == teamID &&
				membership.UserID == actorID &&
				membership.Status == MembershipActive &&
				(membership.Role == TeamOwner || membership.Role == TeamAdmin) {
				authorized = true
				break
			}
		}
		if !authorized {
			return fmt.Errorf(
				"%w: actor must be an owner or admin of every active target team",
				ErrForbidden,
			)
		}
	}
	return nil
}

// AuthorizeUserAdministration verifies that a process-global user mutation is
// controlled by an owner or administrator from every active team containing
// the target. Authority in one team must not affect the user's other teams.
func (s *Service) AuthorizeUserAdministration(
	actorID string,
	targetUserID string,
) error {
	return s.store.view(func(st state) error {
		return authorizeCredentialAdmin(&st, actorID, targetUserID)
	})
}

func firstActiveUserID(st *state) string {
	var users []User
	for _, user := range st.Users {
		if user.Status == UserActive {
			users = append(users, user)
		}
	}
	slices.SortFunc(users, func(a, b User) int {
		if a.CreatedAt.Equal(b.CreatedAt) {
			return strings.Compare(a.ID, b.ID)
		}
		if a.CreatedAt.Before(b.CreatedAt) {
			return -1
		}
		return 1
	})
	if len(users) == 0 {
		return ""
	}
	return users[0].ID
}

func hashAccessToken(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}
