package engineering

import (
	"database/sql"
	"net/http"
	"time"

	kratosgrpc "github.com/go-kratos/kratos/v3/transport/grpc"
	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/engineering/contract"
	"github.com/hvritual/workspace/internal/modules/engineering/internal/application"
	persistence "github.com/hvritual/workspace/internal/modules/engineering/internal/infrastructure/sqlite"
	httpinterface "github.com/hvritual/workspace/internal/modules/engineering/internal/interfaces/http"
)

type Dependencies struct {
	Memberships      contract.WorkspaceRoleResolver
	HTTPUserIdentity func(*http.Request) (string, error)
	Now              func() time.Time
}

type Module struct {
	service   contract.Service
	workLinks contract.WorkLinkProvider
	http      *httpinterface.Handler
}

func NewWithSQLite(db *sql.DB, dependencies Dependencies) (*Module, error) {
	repository, err := persistence.New(db)
	if err != nil {
		return nil, err
	}
	service, err := application.New(repository, dependencies.Memberships, dependencies.Now)
	if err != nil {
		return nil, err
	}
	return &Module{
		service:   service,
		workLinks: service,
		http:      httpinterface.NewHandler(service, dependencies.HTTPUserIdentity),
	}, nil
}

func (m *Module) Service() contract.Service {
	if m == nil {
		return nil
	}
	return m.service
}

func (m *Module) WorkLinks() contract.WorkLinkProvider {
	if m == nil {
		return nil
	}
	return m.workLinks
}

func (m *Module) RegisterHTTP(server *kratoshttp.Server) {
	if m == nil || m.http == nil || server == nil {
		return
	}
	m.http.Register(server)
}

func (*Module) RegisterGRPC(*kratosgrpc.Server) {}
