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
	canonicalrealtime "github.com/hvritual/workspace/internal/realtime"
	_ "modernc.org/sqlite"
)

func newSQLiteApplication(ctx context.Context, config Config) (*sql.DB, *Application, *canonicalrealtime.Hub, error) {
	path := strings.TrimSpace(config.SQLitePath)
	if path != ":memory:" && !strings.HasPrefix(path, "file:") {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, nil, nil, fmt.Errorf("create Canonical SQLite directory: %w", err)
		}
	}
	dataSource := path
	if path != ":memory:" {
		separator := "?"
		if strings.Contains(path, "?") {
			separator = "&"
		}
		dataSource += separator + "_pragma=busy_timeout(5000)"
	}
	db, err := sql.Open("sqlite", dataSource)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("open Canonical SQLite database: %w", err)
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
		return nil, nil, nil, fmt.Errorf("configure Canonical SQLite database: %w", err)
	}
	if err := workspace.MigrateSqlite(ctx, db); err != nil {
		return nil, nil, nil, err
	}
	if err := auth.MigrateSqlite(ctx, db); err != nil {
		return nil, nil, nil, err
	}
	if err := normalizeRetainedIssueMemberActors(ctx, db); err != nil {
		return nil, nil, nil, err
	}
	authModule, err := auth.NewWithSqliteLocalAuth(auth.SqlitePersistenceConfig{DB: db}, config.LocalAuth)
	if err != nil {
		return nil, nil, nil, err
	}
	memberships := authMembershipAdapter{reader: authModule.WorkspaceMemberships()}
	selection, err := workspace.NewSqliteWorkspaceSelection(workspace.SqlitePersistenceConfig{DB: db}, memberships)
	if err != nil {
		return nil, nil, nil, err
	}
	workspaceDependencies := config.WorkspaceDependencies
	workspaceDependencies.Authorizer = memberships
	workspaceDependencies.Actors = memberships
	workspaceDependencies.Selection = selection
	workspaceDependencies.WorkspaceMemberships = memberships
	workspaceDependencies.HTTPUserIdentity = authModule.ResolveHTTPUserID
	workspaceDependencies.HTTPIdentity = workspace.NewTrustedHTTPIdentityResolver(authModule.ResolveHTTPUserID, selection)
	workspaceDependencies.HTTPMutationAuthorizer = authModule.AuthorizeHTTPMutation
	workspaceDependencies.WorkspaceOwnerWriter = auth.NewSQLiteWorkspaceOwnerWriter()
	workspaceDependencies.IssueMetadataEnabled = config.IssueMetadataEnabled
	workspaceDependencies.IssueCreateEnabled = config.IssueCreateEnabled
	realtimeHub := canonicalrealtime.NewHub(canonicalrealtime.IdentityResolver(workspaceDependencies.HTTPIdentity))
	workspaceDependencies.Events = realtimeHub
	workspaceModule, err := workspace.NewWithSqliteWorkspaceChain(
		workspace.SqlitePersistenceConfig{DB: db},
		workspaceDependencies,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	failed = false
	return db, NewApplicationWithModules(workspaceModule, authModule, space.New(), systemmodule.New()), realtimeHub, nil
}

func normalizeRetainedIssueMemberActors(ctx context.Context, db *sql.DB) (err error) {
	connection, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("acquire retained Issue actor normalization connection: %w", err)
	}
	defer connection.Close()
	if _, err := connection.ExecContext(ctx, `PRAGMA busy_timeout = 5000`); err != nil {
		return fmt.Errorf("configure retained Issue actor normalization lock wait: %w", err)
	}
	if _, err := connection.ExecContext(ctx, `BEGIN IMMEDIATE`); err != nil {
		return fmt.Errorf("begin retained Issue actor normalization: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = connection.ExecContext(context.WithoutCancel(ctx), `ROLLBACK`)
		}
	}()
	for _, statement := range []string{
		`UPDATE workspace_issues AS issue
		 SET creator_id = (SELECT member.user_id FROM auth_members AS member WHERE member.id = issue.creator_id AND member.workspace_id = issue.workspace_id)
		 WHERE issue.creator_type = 'member'
		   AND EXISTS (SELECT 1 FROM auth_members AS member WHERE member.id = issue.creator_id AND member.workspace_id = issue.workspace_id)`,
		`UPDATE workspace_issues AS issue
		 SET assignee_id = (SELECT member.user_id FROM auth_members AS member WHERE member.id = issue.assignee_id AND member.workspace_id = issue.workspace_id)
		 WHERE issue.assignee_type = 'member'
		   AND EXISTS (SELECT 1 FROM auth_members AS member WHERE member.id = issue.assignee_id AND member.workspace_id = issue.workspace_id)`,
	} {
		if _, err := connection.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("normalize retained Issue member actors: %w", err)
		}
	}
	if _, err := connection.ExecContext(ctx, `COMMIT`); err != nil {
		return fmt.Errorf("commit retained Issue actor normalization: %w", err)
	}
	committed = true
	return nil
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
		result[index] = contract.WorkspaceMembership{MemberID: value.MemberID, UserID: value.UserID, WorkspaceID: value.WorkspaceID, Role: value.Role}
	}
	return result, nil
}

func (a authMembershipAdapter) FindForUserAndWorkspace(ctx context.Context, userID, workspaceID string) (contract.WorkspaceMembership, bool, error) {
	value, ok, err := a.reader.FindForUserAndWorkspace(ctx, userID, workspaceID)
	return contract.WorkspaceMembership{MemberID: value.MemberID, UserID: value.UserID, WorkspaceID: value.WorkspaceID, Role: value.Role}, ok, err
}

func (a authMembershipAdapter) FindByMemberAndWorkspace(ctx context.Context, memberID, workspaceID string) (contract.WorkspaceMembership, bool, error) {
	value, ok, err := a.reader.FindByMemberAndWorkspace(ctx, memberID, workspaceID)
	return contract.WorkspaceMembership{MemberID: value.MemberID, UserID: value.UserID, WorkspaceID: value.WorkspaceID, Role: value.Role}, ok, err
}

func (a authMembershipAdapter) AuthorizeWorkspace(ctx context.Context, workspaceID, permission string) error {
	switch permission {
	case "workspace.issue.create", "workspace.issue.get", "workspace.issue.list", "workspace.issue.update", "workspace.issue.update_status", "workspace.issue.delete",
		"workspace.issue.metadata.get", "workspace.issue.metadata.put", "workspace.issue.metadata.delete",
		"workspace.issue.timeline.list", "workspace.issue.timeline.record",
		"workspace.issue.comment.get", "workspace.issue.comment.list", "workspace.issue.comment.create", "workspace.issue.comment.update", "workspace.issue.comment.delete", "workspace.issue.comment.resolve", "workspace.issue.comment.knowledge", "workspace.issue.comment.react",
		"workspace.issue.reaction.list", "workspace.issue.reaction.put", "workspace.issue.reaction.delete",
		"workspace.issue.subscriber.list", "workspace.issue.subscriber.put", "workspace.issue.subscriber.delete",
		"workspace.project.create", "workspace.project.get", "workspace.project.list", "workspace.project.search", "workspace.project.update", "workspace.project.delete",
		"workspace.pin.list", "workspace.pin.create", "workspace.pin.delete":
	default:
		return contract.ErrWorkspaceActorRequired
	}
	actor, ok := contract.WorkspaceActorFromContext(ctx)
	if !ok || actor.Type != "member" {
		return contract.ErrWorkspaceActorRequired
	}
	membership, found, err := a.FindForUserAndWorkspace(ctx, actor.ID, workspaceID)
	if err != nil {
		return err
	}
	if !found {
		membership, found, err = a.FindByMemberAndWorkspace(ctx, actor.ID, workspaceID)
		if err != nil {
			return err
		}
	}
	if !found {
		return contract.ErrActorOutsideWorkspace
	}
	if permission == "workspace.project.delete" && membership.Role != "owner" && membership.Role != "admin" {
		return contract.ErrWorkspacePermissionDenied
	}
	return nil
}

func (a authMembershipAdapter) ActorBelongsToWorkspace(ctx context.Context, workspaceID, actorType, actorID string) (bool, error) {
	if actorType != "member" {
		return false, nil
	}
	_, found, err := a.FindForUserAndWorkspace(ctx, actorID, workspaceID)
	if err == nil && !found {
		_, found, err = a.FindByMemberAndWorkspace(ctx, actorID, workspaceID)
	}
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
