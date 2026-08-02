// Package bootstrap assembles generated DDD modules without binding the legacy Chi runtime.
package bootstrap

import (
	"context"
	"database/sql"
	"fmt"

	kratosgrpc "github.com/go-kratos/kratos/v3/transport/grpc"
	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/multica-ai/multica/server/internal/modules/auth"
	authcontract "github.com/multica-ai/multica/server/internal/modules/auth/contract"
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
	modules     []Module
	authMembers authcontract.MemberService
}

// NewApplication registers every accepted top-level bounded context.
func NewApplication() *Application {
	return assemble(auth.New())
}

// NewSQLiteApplication assembles native modules after installing provider-owned schemas.
func NewSQLiteApplication(ctx context.Context, db *sql.DB) (*Application, error) {
	if err := auth.MigrateSqlite(ctx, db); err != nil {
		return nil, fmt.Errorf("migrate Auth module: %w", err)
	}
	authModule, err := auth.NewWithSqlitePersistence(auth.SqlitePersistenceConfig{DB: db})
	if err != nil {
		return nil, fmt.Errorf("assemble Auth module: %w", err)
	}
	return assemble(authModule), nil
}

func assemble(authModule *auth.Module) *Application {
	return &Application{modules: []Module{
		workspace.New(),
		authModule,
		space.New(),
		system.New(),
	}, authMembers: authModule.MemberLocal()}
}

// AuthMembers exposes the Auth member contract to the current SQLite HTTP adapter.
func (a *Application) AuthMembers() authcontract.MemberService { return a.authMembers }

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
