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
	authModule, err := auth.NewWithSqliteMemberServices(auth.SqlitePersistenceConfig{DB: db})
	if err != nil {
		return nil, nil, err
	}
	workspaceModule, err := workspace.NewWithSqliteWorkspaceChain(
		workspace.SqlitePersistenceConfig{DB: db},
		config.WorkspaceDependencies,
	)
	if err != nil {
		return nil, nil, err
	}
	failed = false
	return db, NewApplicationWithModules(workspaceModule, authModule, space.New(), systemmodule.New()), nil
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
