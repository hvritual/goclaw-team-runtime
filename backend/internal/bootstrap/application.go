// Package bootstrap assembles the standalone backend application.
package bootstrap

import (
	kratosgrpc "github.com/go-kratos/kratos/v3/transport/grpc"
	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/auth"
	"github.com/hvritual/workspace/internal/modules/space"
	systemmodule "github.com/hvritual/workspace/internal/modules/system"
	"github.com/hvritual/workspace/internal/modules/workspace"
)

// Module is the transport registration contract implemented by generated modules.
type Module interface {
	RegisterHTTP(server *kratoshttp.Server)
	RegisterGRPC(server *kratosgrpc.Server)
}

// Application owns the four top-level bounded contexts.
type Application struct {
	modules []Module
}

// NewApplication assembles all bounded contexts without selecting persistence.
func NewApplication() *Application {
	return NewApplicationWithModules(
		workspace.New(),
		auth.New(),
		space.New(),
		systemmodule.New(),
	)
}

// NewApplicationWithModules assembles an application from an explicitly
// selected provider graph.
func NewApplicationWithModules(modules ...Module) *Application {
	return &Application{modules: append([]Module(nil), modules...)}
}

// Modules returns a defensive copy of the assembled module set.
func (a *Application) Modules() []Module {
	return append([]Module(nil), a.modules...)
}

// RegisterHTTP registers every module with the shared HTTP server.
func (a *Application) RegisterHTTP(server *kratoshttp.Server) {
	for _, module := range a.modules {
		module.RegisterHTTP(server)
	}
}

// RegisterGRPC registers every module with the shared gRPC server.
func (a *Application) RegisterGRPC(server *kratosgrpc.Server) {
	for _, module := range a.modules {
		module.RegisterGRPC(server)
	}
}
