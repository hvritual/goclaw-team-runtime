package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/hvritual/workspace/internal/modules/auth/contract"
	"github.com/hvritual/workspace/internal/modules/auth/internal/application"
)

type Config struct {
	DB          *sql.DB
	NewMemberID func(context.Context) (string, error)
	Now         func() time.Time
}

func New(Config) contract.Service { return application.New() }

func NewMember(config Config) (contract.MemberService, error) {
	if config.DB == nil {
		return nil, errors.New("auth sqlite database is required")
	}
	newMemberID := config.NewMemberID
	if newMemberID == nil {
		newMemberID = func(context.Context) (string, error) { return uuid.NewString(), nil }
	}
	now := config.Now
	if now == nil {
		now = time.Now
	}
	return application.NewMemberUseCase(
		NewMemberStore(config.DB),
		application.MemberIDGenerator(newMemberID),
		application.MemberClock(now),
	)
}
