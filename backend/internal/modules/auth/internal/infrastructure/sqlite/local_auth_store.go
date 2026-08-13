package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hvritual/workspace/internal/modules/auth/internal/application"
)

type LocalAuthStore struct{ db *sql.DB }

func NewLocalAuthStore(db *sql.DB) *LocalAuthStore { return &LocalAuthStore{db: db} }

func (s *LocalAuthStore) FindOrCreateUser(ctx context.Context, email string, now time.Time) (application.LocalUser, error) {
	user, err := s.findUser(ctx, `SELECT id,name,email,avatar_url,onboarded_at,created_at,updated_at FROM auth_users WHERE email=?`, email)
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return application.LocalUser{}, err
	}
	name := strings.SplitN(email, "@", 2)[0]
	newUserID := uuid.NewString()
	timestamp := now.Format(time.RFC3339Nano)
	if _, err := s.db.ExecContext(ctx, `INSERT INTO auth_users(id,name,email,created_at,updated_at) VALUES(?,?,?,?,?) ON CONFLICT(email) DO NOTHING`, newUserID, name, email, timestamp, timestamp); err != nil {
		return application.LocalUser{}, err
	}
	return s.findUser(ctx, `SELECT id,name,email,avatar_url,onboarded_at,created_at,updated_at FROM auth_users WHERE email=?`, email)
}

func (s *LocalAuthStore) CreateSession(ctx context.Context, token, userID string, createdAt, expiresAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO auth_sessions(token_hash,user_id,created_at,expires_at_unix_nano) VALUES(?,?,?,?)`, tokenHash(token), userID, createdAt.Format(time.RFC3339Nano), expiresAt.UnixNano())
	return err
}

func (s *LocalAuthStore) FindSessionUser(ctx context.Context, token string, now time.Time) (application.LocalUser, error) {
	user, err := s.findUser(ctx, `SELECT u.id,u.name,u.email,u.avatar_url,u.onboarded_at,u.created_at,u.updated_at
		FROM auth_sessions s JOIN auth_users u ON u.id=s.user_id
		WHERE s.token_hash=? AND s.revoked_at IS NULL AND s.expires_at_unix_nano>?`, tokenHash(token), now.UnixNano())
	if errors.Is(err, sql.ErrNoRows) {
		return application.LocalUser{}, application.ErrInvalidToken
	}
	return user, err
}

func (s *LocalAuthStore) RevokeSession(ctx context.Context, token string, now time.Time) error {
	_, err := s.db.ExecContext(ctx, `UPDATE auth_sessions SET revoked_at=? WHERE token_hash=? AND revoked_at IS NULL`, now.Format(time.RFC3339Nano), tokenHash(token))
	return err
}

func (s *LocalAuthStore) findUser(ctx context.Context, query string, arguments ...any) (application.LocalUser, error) {
	var user application.LocalUser
	var avatar sql.NullString
	var onboardedAt sql.NullString
	err := s.db.QueryRowContext(ctx, query, arguments...).Scan(&user.ID, &user.Name, &user.Email, &avatar, &onboardedAt, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return application.LocalUser{}, err
	}
	if avatar.Valid {
		user.AvatarURL = &avatar.String
	}
	if onboardedAt.Valid {
		user.OnboardedAt = &onboardedAt.String
	}
	user.OnboardingQuestionnaire = map[string]any{}
	return user, nil
}

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
