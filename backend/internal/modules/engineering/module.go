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
	lifecycle contract.LifecycleService
	workLinks contract.WorkLinkProvider
	http      *httpinterface.Handler
	exitHTTP  *httpinterface.LifecycleHandler
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
		lifecycle: service,
		workLinks: service,
		http:      httpinterface.NewHandler(service, dependencies.HTTPUserIdentity),
		exitHTTP:  httpinterface.NewLifecycleHandler(service, dependencies.HTTPUserIdentity),
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
	if m == nil || server == nil {
		return
	}
	if m.http != nil {
		m.http.Register(server)
	}
	if m.exitHTTP != nil {
		m.exitHTTP.Register(server)
	}
}

func (*Module) RegisterGRPC(*kratosgrpc.Server) {}
