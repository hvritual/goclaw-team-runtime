// Package bootstrap assembles generated DDD modules without binding the legacy Chi runtime.
package bootstrap

import (
	kratosgrpc "github.com/go-kratos/kratos/v3/transport/grpc"
	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/multica-ai/multica/server/internal/modules/auth"
	"github.com/multica-ai/multica/server/internal/modules/space"
	"github.com/multica-ai/multica/server/internal/modules/system"
	"github.com/multica-ai/multica/server/internal/modules/workspace"
)

// Module is the transport registration contract implemented by generated modules.
type Module interface {
	RegisterHTTP(server *kratoshttp.Server)
	RegisterGRPC(server *kratosgrpc.Server)
}

// Application owns the generated module graph.
type Application struct {
	modules []Module
}

// NewApplication registers every accepted top-level bounded context.
func NewApplication() *Application {
	return &Application{modules: []Module{
		workspace.New(),
		auth.New(),
		space.New(),
		system.New(),
	}}
}

// Modules returns a defensive copy for alternative composition and tests.
func (a *Application) Modules() []Module {
	return append([]Module(nil), a.modules...)
}

// RegisterHTTP registers all generated module services with Kratos HTTP.
func (a *Application) RegisterHTTP(server *kratoshttp.Server) {
	for _, module := range a.modules {
		module.RegisterHTTP(server)
	}
}

// RegisterGRPC registers all generated module services with Kratos gRPC.
func (a *Application) RegisterGRPC(server *kratosgrpc.Server) {
	for _, module := range a.modules {
		module.RegisterGRPC(server)
	}
}
