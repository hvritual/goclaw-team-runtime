// Package module provides process-level registration for generated module extensions.
package module

import (
	"sync"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"google.golang.org/grpc"
)

// Extension registers one generated service with the selected transports.
type Extension interface {
	RegisterHTTP(server *kratoshttp.Server)
	RegisterGRPC(registrar grpc.ServiceRegistrar)
}

// Factory constructs one isolated extension instance for a module.
type Factory func() Extension

// Registry stores generated extension factories until module assembly.
type Registry struct {
	mu        sync.RWMutex
	factories []Factory
}

// Register adds an extension factory to the module registry.
func (r *Registry) Register(factory Factory) {
	if factory == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories = append(r.factories, factory)
}

// Build creates a fresh extension set for one module instance.
func (r *Registry) Build() []Extension {
	r.mu.RLock()
	factories := append([]Factory(nil), r.factories...)
	r.mu.RUnlock()

	extensions := make([]Extension, 0, len(factories))
	for _, factory := range factories {
		if extension := factory(); extension != nil {
			extensions = append(extensions, extension)
		}
	}
	return extensions
}

// RegisterHTTP attaches all extensions to a Kratos HTTP server.
func RegisterHTTP(extensions []Extension, server *kratoshttp.Server) {
	for _, extension := range extensions {
		extension.RegisterHTTP(server)
	}
}

// RegisterGRPC attaches all extensions to a gRPC registrar.
func RegisterGRPC(extensions []Extension, registrar grpc.ServiceRegistrar) {
	for _, extension := range extensions {
		extension.RegisterGRPC(registrar)
	}
}
