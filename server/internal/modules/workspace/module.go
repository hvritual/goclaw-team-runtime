package workspace

import (
	kratosgrpc "github.com/go-kratos/kratos/v3/transport/grpc"
	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	workspacev1 "github.com/multica-ai/multica/server/gen/go/workspace/v1"
	"github.com/multica-ai/multica/server/internal/modules/workspace/contract"
	"github.com/multica-ai/multica/server/internal/modules/workspace/internal/application"
	grpcadapter "github.com/multica-ai/multica/server/internal/modules/workspace/internal/interfaces/grpc"
	"github.com/multica-ai/multica/server/internal/modules/workspace/internal/interfaces/local"
	protoadapter "github.com/multica-ai/multica/server/internal/modules/workspace/internal/interfaces/proto"
	platformmodule "github.com/multica-ai/multica/server/internal/platform/module"
	stdgrpc "google.golang.org/grpc"
)

var extensionRegistry platformmodule.Registry

func RegisterExtension(factory platformmodule.Factory) {
	extensionRegistry.Register(factory)
}

type Module struct {
	local      contract.Service
	identity   contract.WorkspaceIdentityReader
	server     *protoadapter.Server
	extensions []platformmodule.Extension
}

func New() *Module {
	client := local.New(application.New())
	extensions := extensionRegistry.Build()
	return &Module{local: client, server: protoadapter.New(client), extensions: extensions}
}
func (m *Module) Local() contract.Service                         { return m.local }
func (m *Module) IdentityLocal() contract.WorkspaceIdentityReader { return m.identity }
func (m *Module) RegisterHTTP(server *kratoshttp.Server) {
	workspacev1.RegisterWorkspaceServiceHTTPServer(server, m.server)
	platformmodule.RegisterHTTP(m.extensions, server)
}
func (m *Module) RegisterGRPC(server *kratosgrpc.Server) { m.RegisterGRPCService(server) }
func (m *Module) RegisterGRPCService(registrar stdgrpc.ServiceRegistrar) {
	workspacev1.RegisterWorkspaceServiceServer(registrar, m.server)
	platformmodule.RegisterGRPC(m.extensions, registrar)
}
func NewGRPCClient(connection stdgrpc.ClientConnInterface) contract.Service {
	return grpcadapter.New(workspacev1.NewWorkspaceServiceClient(connection))
}
