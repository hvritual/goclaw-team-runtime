package space

import (
	kratosgrpc "github.com/go-kratos/kratos/v3/transport/grpc"
	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/space/contract"
	"github.com/hvritual/workspace/internal/modules/space/internal/application"
	grpcadapter "github.com/hvritual/workspace/internal/modules/space/internal/interfaces/grpc"
	"github.com/hvritual/workspace/internal/modules/space/internal/interfaces/local"
	protoadapter "github.com/hvritual/workspace/internal/modules/space/internal/interfaces/proto"
	platformmodule "github.com/hvritual/workspace/internal/platform/module"
	spacev1 "github.com/hvritual/workspace/rpc/pb/space/v1"
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
	spacev1.RegisterSpaceServiceHTTPServer(server, m.server)
	platformmodule.RegisterHTTP(m.extensions, server)
}
func (m *Module) RegisterGRPC(server *kratosgrpc.Server) { m.RegisterGRPCService(server) }
func (m *Module) RegisterGRPCService(registrar stdgrpc.ServiceRegistrar) {
	spacev1.RegisterSpaceServiceServer(registrar, m.server)
	platformmodule.RegisterGRPC(m.extensions, registrar)
}
func NewGRPCClient(connection stdgrpc.ClientConnInterface) contract.Service {
	return grpcadapter.New(spacev1.NewSpaceServiceClient(connection))
}
