// Package sqlite contains Sqlite adapters for the auth module.
// Implement domain or application ports here and keep driver types inside this package.
package sqlite

import (
	"database/sql"
	"errors"

	"github.com/multica-ai/multica/server/internal/modules/auth/contract"
	"github.com/multica-ai/multica/server/internal/modules/auth/internal/application"
)

// Config is the provider-owned composition input. Add the native connection
// and provider settings here; never pass them into domain or application APIs.
type Config struct {
	DB *sql.DB
}

func New(Config) contract.Service {
	return application.New()
}

func NewMember(config Config) (contract.MemberService, error) {
	if config.DB == nil {
		return nil, errors.New("auth sqlite database is required")
	}
	return application.NewMemberService(application.WithMemberUnitOfWork(NewMemberStore(config.DB))), nil
}
