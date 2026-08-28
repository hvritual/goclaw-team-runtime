package workspace

import (
	"database/sql"
	"errors"
	"net/http"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
	workspacesqlite "github.com/hvritual/workspace/internal/modules/workspace/internal/infrastructure/sqlite"
	workspacehttp "github.com/hvritual/workspace/internal/modules/workspace/internal/interfaces/http"
	"google.golang.org/grpc"
)

type workEngineeringLinkExtension struct {
	handler *workspacehttp.WorkEngineeringLinkHandler
}

func (e *workEngineeringLinkExtension) RegisterHTTP(server *kratoshttp.Server) {
	if e != nil && e.handler != nil {
		e.handler.Register(server)
	}
}

func (*workEngineeringLinkExtension) RegisterGRPC(grpc.ServiceRegistrar) {}

func NewSQLiteWorkEngineeringLinkService(db *sql.DB, memberships contract.WorkspaceMembershipReader, engineering contract.EngineeringLinkGateway) (contract.WorkEngineeringLinkService, error) {
	reader, err := workspacesqlite.NewEngineeringWorkReferenceReader(db)
	if err != nil {
		return nil, err
	}
	return application.NewWorkEngineeringLinkUseCase(reader, memberships, engineering)
}

func (m *Module) InstallWorkEngineeringLinks(service contract.WorkEngineeringLinkService, identity contract.WorkspaceHTTPIdentityResolver, authenticate func(*http.Request) (string, error), mutation func(*http.Request) error) error {
	if m == nil || service == nil || identity == nil || authenticate == nil || mutation == nil {
		return errors.New("Workspace Engineering work-link dependencies are required")
	}
	m.extensions = append(m.extensions, &workEngineeringLinkExtension{
		handler: workspacehttp.NewWorkEngineeringLinkHandler(service, identity, authenticate, mutation),
	})
	return nil
}
