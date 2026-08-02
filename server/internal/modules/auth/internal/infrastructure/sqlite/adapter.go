// Package sqlite contains Sqlite adapters for the auth module.
// Implement domain or application ports here and keep driver types inside this package.
package sqlite

import (
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/multica-ai/multica/server/internal/modules/auth/contract"
	"github.com/multica-ai/multica/server/internal/modules/auth/internal/application"
	workspacecontract "github.com/multica-ai/multica/server/internal/modules/workspace/contract"
)

// Config is the provider-owned composition input. Add the native connection
// and provider settings here; never pass them into domain or application APIs.
type Config struct {
	DB                  *sql.DB
	WorkspaceIdentities workspacecontract.WorkspaceIdentityReader
	Now                 func() time.Time
	NewInvitationID     func() string
}

func New(Config) contract.Service {
	return application.New()
}

func NewMember(config Config) (contract.MemberService, error) {
	if config.DB == nil {
		return nil, errors.New("auth sqlite database is required")
	}
	store := NewMemberStore(config.DB)
	options := []application.MemberServiceOption{
		application.WithMemberUnitOfWork(store),
		application.WithInvitationUnitOfWork(store),
		application.WithWorkspaceIdentityReader(config.WorkspaceIdentities),
	}
	newInvitationID := config.NewInvitationID
	if newInvitationID == nil {
		newInvitationID = uuid.NewString
	}
	options = append(options, application.WithInvitationIDGenerator(newInvitationID))
	if config.Now != nil {
		options = append(options, application.WithInvitationClock(config.Now))
	}
	return application.NewMemberService(options...), nil
}
