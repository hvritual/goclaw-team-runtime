package system

import (
	kratosgrpc "github.com/go-kratos/kratos/v3/transport/grpc"
	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	systemv1 "github.com/multica-ai/multica/server/gen/go/system/v1"
	"github.com/multica-ai/multica/server/internal/modules/system/contract"
	"github.com/multica-ai/multica/server/internal/modules/system/internal/application"
	grpcadapter "github.com/multica-ai/multica/server/internal/modules/system/internal/interfaces/grpc"
	"github.com/multica-ai/multica/server/internal/modules/system/internal/interfaces/local"
	protoadapter "github.com/multica-ai/multica/server/internal/modules/system/internal/interfaces/proto"
	platformmodule "github.com/multica-ai/multica/server/internal/platform/module"
	stdgrpc "google.golang.org/grpc"
)

var extensionRegistry platformmodule.Registry

func RegisterExtension(factory platformmodule.Factory) {
	extensionRegistry.Register(factory)
}

type Module struct {
	local      contract.Service
	server     *protoadapter.Server
	extensions []platformmodule.Extension
}

func New() *Module {
	client := local.New(application.New())
	extensions := extensionRegistry.Build()
	return &Module{local: client, server: protoadapter.New(client), extensions: extensions}
}
func (m *Module) Local() contract.Service { return m.local }
func (m *Module) RegisterHTTP(server *kratoshttp.Server) {
	systemv1.RegisterSystemServiceHTTPServer(server, m.server)
	platformmodule.RegisterHTTP(m.extensions, server)
}
func (m *Module) RegisterGRPC(server *kratosgrpc.Server) { m.RegisterGRPCService(server) }
func (m *Module) RegisterGRPCService(registrar stdgrpc.ServiceRegistrar) {
	systemv1.RegisterSystemServiceServer(registrar, m.server)
	platformmodule.RegisterGRPC(m.extensions, registrar)
}
func NewGRPCClient(connection stdgrpc.ClientConnInterface) contract.Service {
	return grpcadapter.New(systemv1.NewSystemServiceClient(connection))
}
