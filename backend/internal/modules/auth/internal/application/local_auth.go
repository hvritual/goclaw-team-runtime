package application

import (
	"context"
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidEmail         = errors.New("valid email is required")
	ErrInvalidCode          = errors.New("invalid verification code")
	ErrInvalidToken         = errors.New("invalid token")
	ErrInvalidCSRF          = errors.New("invalid CSRF token")
	ErrWorkspaceUnavailable = errors.New("workspace is not available to the current user")
)

type LocalUser struct {
	ID                      string         `json:"id"`
	Name                    string         `json:"name"`
	Email                   string         `json:"email"`
	AvatarURL               *string        `json:"avatar_url"`
	OnboardedAt             *string        `json:"onboarded_at"`
	OnboardingQuestionnaire map[string]any `json:"onboarding_questionnaire"`
	StarterContentState     *string        `json:"starter_content_state"`
	Language                *string        `json:"language"`
	ProfileDescription      string         `json:"profile_description"`
	Timezone                *string        `json:"timezone"`
	CreatedAt               string         `json:"created_at"`
	UpdatedAt               string         `json:"updated_at"`
}

type LocalLogin struct {
	Token string    `json:"token"`
	User  LocalUser `json:"user"`
}

type LocalAuthRepository interface {
	FindOrCreateUser(context.Context, string, time.Time) (LocalUser, error)
	CreateSession(context.Context, string, string, time.Time, time.Time) error
	FindSessionUser(context.Context, string, time.Time) (LocalUser, error)
	RevokeSession(context.Context, string, time.Time) error
	CompleteOnboarding(context.Context, string, string, time.Time) (LocalUser, error)
}

type LocalAuthUseCase struct {
	repository       LocalAuthRepository
	verificationCode string
	sessionTTL       time.Duration
	now              func() time.Time
	newID            func(context.Context) (string, error)
}

func NewLocalAuthUseCase(repository LocalAuthRepository, verificationCode string, sessionTTL time.Duration, now func() time.Time, newID func(context.Context) (string, error)) *LocalAuthUseCase {
	return &LocalAuthUseCase{repository: repository, verificationCode: verificationCode, sessionTTL: sessionTTL, now: now, newID: newID}
}

func (u *LocalAuthUseCase) SendCode(email string) error {
	if !validEmail(email) {
		return ErrInvalidEmail
	}
	return nil
}

func (u *LocalAuthUseCase) VerifyCode(ctx context.Context, email, code string) (LocalLogin, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if code != u.verificationCode || !validEmail(email) {
		return LocalLogin{}, ErrInvalidCode
	}
	token, err := u.newID(ctx)
	if err != nil {
		return LocalLogin{}, err
	}
	current := u.now().UTC()
	user, err := u.repository.FindOrCreateUser(ctx, email, current)
	if err != nil {
		return LocalLogin{}, err
	}
	if err := u.repository.CreateSession(ctx, token, user.ID, current, current.Add(u.sessionTTL)); err != nil {
		return LocalLogin{}, err
	}
	return LocalLogin{Token: token, User: user}, nil
}

func (u *LocalAuthUseCase) Resolve(ctx context.Context, token string) (LocalUser, error) {
	if strings.TrimSpace(token) == "" {
		return LocalUser{}, ErrInvalidToken
	}
	return u.repository.FindSessionUser(ctx, token, u.now().UTC())
}

func (u *LocalAuthUseCase) Revoke(ctx context.Context, token string) error {
	if strings.TrimSpace(token) == "" {
		return nil
	}
	return u.repository.RevokeSession(ctx, token, u.now().UTC())
}

func (u *LocalAuthUseCase) CompleteOnboarding(ctx context.Context, userID, workspaceID string) (LocalUser, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return LocalUser{}, ErrInvalidToken
	}
	return u.repository.CompleteOnboarding(ctx, userID, strings.TrimSpace(workspaceID), u.now().UTC())
}

func (u *LocalAuthUseCase) SessionTTL() time.Duration { return u.sessionTTL }
func (u *LocalAuthUseCase) Now() time.Time            { return u.now().UTC() }

func validEmail(email string) bool {
	trimmed := strings.TrimSpace(email)
	parts := strings.Split(trimmed, "@")
	return len(parts) == 2 && parts[0] != "" && parts[1] != ""
}
