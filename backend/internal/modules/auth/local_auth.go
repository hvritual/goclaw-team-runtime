package auth

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/google/uuid"
	"github.com/hvritual/workspace/internal/modules/auth/contract"
	"github.com/hvritual/workspace/internal/modules/auth/internal/application"
	persistence "github.com/hvritual/workspace/internal/modules/auth/internal/infrastructure/sqlite"
	authhttp "github.com/hvritual/workspace/internal/modules/auth/internal/interfaces/http"
	"google.golang.org/grpc"
)

type LocalAuthConfig struct {
	VerificationCode string
	SessionTTL       time.Duration
	Now              func() time.Time
	NewID            func(context.Context) (string, error)
}

type localAuthExtension struct{ handler *authhttp.LocalAuthHandler }

type memberHTTPListExtension struct{ handler *authhttp.MemberListHandler }

var verificationCodePattern = regexp.MustCompile(`^[0-9]{6}$`)

func (e *localAuthExtension) RegisterHTTP(server *kratoshttp.Server) { e.handler.Register(server) }
func (e *localAuthExtension) RegisterGRPC(grpc.ServiceRegistrar)     {}
func (e *memberHTTPListExtension) RegisterHTTP(server *kratoshttp.Server) {
	e.handler.Register(server)
}
func (e *memberHTTPListExtension) RegisterGRPC(grpc.ServiceRegistrar) {}

func NewWithSqliteLocalAuth(persistenceConfig SqlitePersistenceConfig, config LocalAuthConfig) (*Module, error) {
	config.VerificationCode = strings.TrimSpace(config.VerificationCode)
	if !verificationCodePattern.MatchString(config.VerificationCode) {
		return nil, errors.New("development verification code must contain exactly six digits")
	}
	if config.SessionTTL <= 0 {
		config.SessionTTL = 7 * 24 * time.Hour
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	if config.NewID == nil {
		config.NewID = func(context.Context) (string, error) { return uuid.NewString(), nil }
	}
	module, err := NewWithSqliteMemberServices(persistenceConfig)
	if err != nil {
		return nil, err
	}
	useCase := application.NewLocalAuthUseCase(persistence.NewLocalAuthStore(persistenceConfig.DB), strings.TrimSpace(config.VerificationCode), config.SessionTTL, config.Now, config.NewID)
	handler := authhttp.NewLocalAuthHandler(useCase)
	memberships, err := persistence.NewWorkspaceMembershipStore(persistenceConfig.DB)
	if err != nil {
		return nil, err
	}
	module.memberships = memberships
	module.httpUserID = handler.ResolveUserID
	module.extensions = append(module.extensions, &localAuthExtension{handler: handler})
	module.extensions = append(module.extensions, &memberHTTPListExtension{handler: authhttp.NewMemberListHandler(module.MemberLocal(), handler.ResolveUserID)})
	return module, nil
}

// ResolveHTTPUserID accepts only the S2 Bearer token or multica_auth cookie
// and validates the session before returning the Auth user ID.
func (m *Module) ResolveHTTPUserID(request *http.Request) (string, error) {
	if m.httpUserID == nil {
		return "", application.ErrInvalidToken
	}
	return m.httpUserID(request)
}

// AuthorizeHTTPMutation enforces the trusted local session's mutation policy.
func (m *Module) AuthorizeHTTPMutation(request *http.Request) error {
	if m.httpUserID == nil {
		return application.ErrInvalidToken
	}
	for _, extension := range m.extensions {
		if local, ok := extension.(*localAuthExtension); ok {
			return local.handler.AuthorizeMutation(request)
		}
	}
	return application.ErrInvalidToken
}

func NewSQLiteWorkspaceOwnerWriter() contract.SQLiteWorkspaceOwnerWriter {
	return persistence.NewWorkspaceOwnerWriter()
}
