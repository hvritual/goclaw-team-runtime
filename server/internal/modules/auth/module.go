package auth

import (
	kratosgrpc "github.com/go-kratos/kratos/v3/transport/grpc"
	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	authv1 "github.com/multica-ai/multica/server/gen/go/auth/v1"
	"github.com/multica-ai/multica/server/internal/modules/auth/contract"
	"github.com/multica-ai/multica/server/internal/modules/auth/internal/application"
	grpcadapter "github.com/multica-ai/multica/server/internal/modules/auth/internal/interfaces/grpc"
	"github.com/multica-ai/multica/server/internal/modules/auth/internal/interfaces/local"
	protoadapter "github.com/multica-ai/multica/server/internal/modules/auth/internal/interfaces/proto"
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
	authv1.RegisterAuthServiceHTTPServer(server, m.server)
	platformmodule.RegisterHTTP(m.extensions, server)
}
func (m *Module) RegisterGRPC(server *kratosgrpc.Server) { m.RegisterGRPCService(server) }
func (m *Module) RegisterGRPCService(registrar stdgrpc.ServiceRegistrar) {
	authv1.RegisterAuthServiceServer(registrar, m.server)
	platformmodule.RegisterGRPC(m.extensions, registrar)
}
func NewGRPCClient(connection stdgrpc.ClientConnInterface) contract.Service {
	return grpcadapter.New(authv1.NewAuthServiceClient(connection))
}
