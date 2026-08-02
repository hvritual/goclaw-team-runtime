// Package bootstrap assembles generated DDD modules without binding the legacy Chi runtime.
package bootstrap

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	kratosgrpc "github.com/go-kratos/kratos/v3/transport/grpc"
	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/multica-ai/multica/server/internal/modules/auth"
	authcontract "github.com/multica-ai/multica/server/internal/modules/auth/contract"
	"github.com/multica-ai/multica/server/internal/modules/space"
	spacecontract "github.com/multica-ai/multica/server/internal/modules/space/contract"
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
	spaceAssets spacecontract.AssetUploadService
}

// NewApplication registers every accepted top-level bounded context.
func NewApplication() *Application {
	return assemble(workspace.New(), auth.New(), space.New())
}

// SQLiteOptions supplies provider adapters that are not database connections.
type SQLiteOptions struct {
	SpaceObjects spacecontract.ObjectStore
}

// NewSQLiteApplication assembles native modules after installing provider-owned schemas.
func NewSQLiteApplication(ctx context.Context, db *sql.DB, optionValues ...SQLiteOptions) (*Application, error) {
	var options SQLiteOptions
	if len(optionValues) > 0 {
		options = optionValues[0]
	}
	if err := workspace.MigrateSqlite(ctx, db); err != nil {
		return nil, fmt.Errorf("migrate Workspace module: %w", err)
	}
	if err := auth.MigrateSqlite(ctx, db); err != nil {
		return nil, fmt.Errorf("migrate Auth module: %w", err)
	}
	if err := space.MigrateSqlite(ctx, db); err != nil {
		return nil, fmt.Errorf("migrate Space module: %w", err)
	}
	workspaceModule, err := workspace.NewWithSqlitePersistence(workspace.SqlitePersistenceConfig{DB: db})
	if err != nil {
		return nil, fmt.Errorf("assemble Workspace module: %w", err)
	}
	authModule, err := auth.NewWithSqlitePersistence(auth.SqlitePersistenceConfig{
		DB: db, WorkspaceIdentities: workspaceModule.IdentityLocal(),
	})
	if err != nil {
		return nil, fmt.Errorf("assemble Auth module: %w", err)
	}
	spaceModule, err := space.NewWithSqlitePersistence(space.SqlitePersistenceConfig{
		DB: db, WorkspaceAccess: authWorkspaceAccess{members: authModule.MemberLocal()},
		Objects: options.SpaceObjects,
	})
	if err != nil {
		return nil, fmt.Errorf("assemble Space module: %w", err)
	}
	return assemble(workspaceModule, authModule, spaceModule), nil
}

func assemble(workspaceModule *workspace.Module, authModule *auth.Module, spaceModule *space.Module) *Application {
	return &Application{modules: []Module{
		workspaceModule,
		authModule,
		spaceModule,
		system.New(),
	}, authMembers: authModule.MemberLocal(), spaceAssets: spaceModule.AssetUploads()}
}

// AuthMembers exposes the Auth member contract to the current SQLite HTTP adapter.
func (a *Application) AuthMembers() authcontract.MemberService { return a.authMembers }

// SpaceAssets exposes the Space upload lifecycle to the current SQLite HTTP adapter.
func (a *Application) SpaceAssets() spacecontract.AssetUploadService { return a.spaceAssets }

type authWorkspaceAccess struct {
	members authcontract.MemberService
}

func (a authWorkspaceAccess) IsMember(ctx context.Context, userID, workspaceID string) (bool, error) {
	if a.members == nil {
		return false, nil
	}
	_, err := a.members.ListMembers(
		authcontract.WithMemberActor(ctx, userID),
		authcontract.Member_ListMembersRequest{WorkspaceId: workspaceID},
	)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, authcontract.ErrWorkspaceMembershipHidden) || errors.Is(err, authcontract.ErrAuthUserNotFound) {
		return false, nil
	}
	return false, err
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
