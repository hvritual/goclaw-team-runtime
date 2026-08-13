package bootstrap

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/hvritual/workspace/internal/modules/auth"
	authcontract "github.com/hvritual/workspace/internal/modules/auth/contract"
	"github.com/hvritual/workspace/internal/modules/space"
	systemmodule "github.com/hvritual/workspace/internal/modules/system"
	"github.com/hvritual/workspace/internal/modules/workspace"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	_ "modernc.org/sqlite"
)

func newSQLiteApplication(ctx context.Context, config Config) (*sql.DB, *Application, error) {
	path := strings.TrimSpace(config.SQLitePath)
	if path != ":memory:" && !strings.HasPrefix(path, "file:") {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, nil, fmt.Errorf("create Canonical SQLite directory: %w", err)
		}
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, nil, fmt.Errorf("open Canonical SQLite database: %w", err)
	}
	if path == ":memory:" {
		db.SetMaxOpenConns(1)
	} else {
		db.SetMaxOpenConns(8)
	}
	failed := true
	defer func() {
		if failed {
			_ = db.Close()
		}
	}()
	if _, err := db.ExecContext(ctx, `PRAGMA busy_timeout = 5000`); err != nil {
		return nil, nil, fmt.Errorf("configure Canonical SQLite database: %w", err)
	}
	if err := workspace.MigrateSqlite(ctx, db); err != nil {
		return nil, nil, err
	}
	if err := auth.MigrateSqlite(ctx, db); err != nil {
		return nil, nil, err
	}
	authModule, err := auth.NewWithSqliteLocalAuth(auth.SqlitePersistenceConfig{DB: db}, config.LocalAuth)
	if err != nil {
		return nil, nil, err
	}
	memberships := authMembershipAdapter{reader: authModule.WorkspaceMemberships()}
	selection, err := workspace.NewSqliteWorkspaceSelection(workspace.SqlitePersistenceConfig{DB: db}, memberships)
	if err != nil {
		return nil, nil, err
	}
	workspaceDependencies := config.WorkspaceDependencies
	workspaceDependencies.Authorizer = memberships
	workspaceDependencies.Actors = memberships
	workspaceDependencies.Selection = selection
	workspaceDependencies.HTTPUserIdentity = authModule.ResolveHTTPUserID
	workspaceDependencies.HTTPIdentity = workspace.NewTrustedHTTPIdentityResolver(authModule.ResolveHTTPUserID, selection)
	workspaceModule, err := workspace.NewWithSqliteWorkspaceChain(
		workspace.SqlitePersistenceConfig{DB: db},
		workspaceDependencies,
	)
	if err != nil {
		return nil, nil, err
	}
	failed = false
	return db, NewApplicationWithModules(workspaceModule, authModule, space.New(), systemmodule.New()), nil
}

type authMembershipAdapter struct {
	reader authcontract.WorkspaceMembershipReader
}

func (a authMembershipAdapter) ListForUser(ctx context.Context, userID string) ([]contract.WorkspaceMembership, error) {
	values, err := a.reader.ListForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make([]contract.WorkspaceMembership, len(values))
	for index, value := range values {
		result[index] = contract.WorkspaceMembership{MemberID: value.MemberID, WorkspaceID: value.WorkspaceID, Role: value.Role}
	}
	return result, nil
}

func (a authMembershipAdapter) FindForUserAndWorkspace(ctx context.Context, userID, workspaceID string) (contract.WorkspaceMembership, bool, error) {
	value, ok, err := a.reader.FindForUserAndWorkspace(ctx, userID, workspaceID)
	return contract.WorkspaceMembership{MemberID: value.MemberID, WorkspaceID: value.WorkspaceID, Role: value.Role}, ok, err
}

func (a authMembershipAdapter) FindByMemberAndWorkspace(ctx context.Context, memberID, workspaceID string) (contract.WorkspaceMembership, bool, error) {
	value, ok, err := a.reader.FindByMemberAndWorkspace(ctx, memberID, workspaceID)
	return contract.WorkspaceMembership{MemberID: value.MemberID, WorkspaceID: value.WorkspaceID, Role: value.Role}, ok, err
}

func (a authMembershipAdapter) AuthorizeWorkspace(ctx context.Context, workspaceID, permission string) error {
	if permission != "workspace.issue.get" && permission != "workspace.issue.list" {
		return contract.ErrWorkspaceActorRequired
	}
	actor, ok := contract.WorkspaceActorFromContext(ctx)
	if !ok || actor.Type != "member" {
		return contract.ErrWorkspaceActorRequired
	}
	_, found, err := a.FindByMemberAndWorkspace(ctx, actor.ID, workspaceID)
	if err != nil {
		return err
	}
	if !found {
		return contract.ErrActorOutsideWorkspace
	}
	return nil
}

func (a authMembershipAdapter) ActorBelongsToWorkspace(ctx context.Context, workspaceID, actorType, actorID string) (bool, error) {
	if actorType != "member" {
		return false, nil
	}
	_, found, err := a.FindByMemberAndWorkspace(ctx, actorID, workspaceID)
	return found, err
}

type failClosedWorkspaceBoundaries struct{}

func (failClosedWorkspaceBoundaries) AuthorizeWorkspace(context.Context, string, string) error {
	return errors.New("Canonical Workspace authorization is not active")
}
func (failClosedWorkspaceBoundaries) ActorBelongsToWorkspace(context.Context, string, string, string) (bool, error) {
	return false, nil
}
func (failClosedWorkspaceBoundaries) AssetBelongsToWorkspace(context.Context, string, string) (bool, error) {
	return false, nil
}
func (failClosedWorkspaceBoundaries) SkillReferenceExists(context.Context, string, *string) (bool, error) {
	return false, nil
}

// FailClosedWorkspaceDependencies selects real SQLite providers without
// granting authorization that belongs to later milestone stories.
func FailClosedWorkspaceDependencies() workspace.WorkspaceServiceDependencies {
	boundaries := failClosedWorkspaceBoundaries{}
	return workspace.WorkspaceServiceDependencies{
		Authorizer: boundaries,
		Actors:     boundaries,
		Assets:     boundaries,
		Skills:     boundaries,
		HTTPIdentity: func(*http.Request) (contract.WorkspaceHTTPIdentity, error) {
			return contract.WorkspaceHTTPIdentity{}, contract.ErrWorkspaceActorRequired
		},
	}
}
