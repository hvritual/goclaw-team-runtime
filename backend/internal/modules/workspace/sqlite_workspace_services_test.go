package workspace

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

type workspaceAccessStub struct {
	mu          sync.Mutex
	permissions []string
}

func (s *workspaceAccessStub) AuthorizeWorkspace(_ context.Context, _ string, permission string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.permissions = append(s.permissions, permission)
	return nil
}

type workspaceActorCatalog struct {
	mu     sync.RWMutex
	actors map[string]bool
}

func (c *workspaceActorCatalog) ActorBelongsToWorkspace(_ context.Context, workspaceID, actorType, actorID string) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.actors[workspaceID+"/"+actorType+"/"+actorID], nil
}

func (c *workspaceActorCatalog) remove(workspaceID, actorType, actorID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.actors, workspaceID+"/"+actorType+"/"+actorID)
}

func openWorkspaceTestDB(t *testing.T) *sql.DB {
	t.Helper()
	name := strings.NewReplacer("/", "-", " ", "-").Replace(t.Name())
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", name))
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	if err := MigrateSqlite(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	return db
}

func seedWorkspace(t *testing.T, db *sql.DB, id, name, slug string) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO workspaces(
		id, name, slug, issue_prefix, created_at, updated_at
	) VALUES (?, ?, ?, 'WSP', '2026-08-03T00:00:00Z', '2026-08-03T00:00:00Z')`, id, name, slug)
	if err != nil {
		t.Fatal(err)
	}
}

func newWorkspaceServicesTestModule(t *testing.T, db *sql.DB, actors *workspaceActorCatalog, projectID string) *Module {
	t.Helper()
	module, err := NewWithSqliteWorkspaceServices(
		SqlitePersistenceConfig{DB: db},
		WorkspaceServiceDependencies{
			Authorizer: &workspaceAccessStub{},
			Actors:     actors,
			NewProjectID: func(context.Context) (string, error) {
				return projectID, nil
			},
			Now: func() time.Time {
				return time.Date(2026, 8, 3, 4, 5, 6, 7, time.UTC)
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return module
}

func TestSqliteMigrationsAreOrderedAtomicAndRepeatable(t *testing.T) {
	db := openWorkspaceTestDB(t)
	if err := MigrateSqlite(context.Background(), db); err != nil {
		t.Fatalf("second MigrateSqlite() error = %v", err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM workspace_schema_migrations`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 12 {
		t.Fatalf("migration count = %d, want 12", count)
	}
	for _, table := range []string{
		"workspaces",
		"workspace_projects",
		"workspace_project_actor_relations",
		"workspace_todos",
		"workspace_issues",
		"workspace_knowledge",
		"workspace_requirements",
		"workspace_requirement_versions",
		"workspace_settings",
		"workspace_skill_bindings",
		"workspace_pins",
		"workspace_issue_comments",
		"workspace_comment_reactions",
		"workspace_issue_reactions",
		"workspace_issue_subscribers",
		"workspace_issue_activities",
		"workspace_comment_knowledge_proposals",
		"workspace_issue_labels",
		"workspace_issue_label_assignments",
		"workspace_issue_property_definitions",
		"workspace_issue_acceptance_conclusions",
		"workspace_acceptance_knowledge_proposals",
		"workspace_resource_revisions",
		"workspace_mutation_idempotency",
		"workspace_audit_entries",
		"workspace_outbox_events",
	} {
		var found string
		if err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&found); err != nil {
			t.Fatalf("table %s: %v", table, err)
		}
	}
	if _, err := db.Exec(`INSERT INTO workspace_todos(
		id, workspace_id, title, status, created_at, updated_at
	) VALUES ('invalid-todo', 'workspace-1', 'Invalid', 'pending', '2026-08-03T00:00:00Z', '2026-08-03T00:00:00Z')`); err == nil {
		t.Fatal("invalid Todo status persisted")
	}
	if _, err := db.Exec(`INSERT INTO workspace_todos(
		id, workspace_id, title, status, priority, created_at, updated_at
	) VALUES ('invalid-todo-priority', 'workspace-1', 'Invalid', 'todo', 'highest',
		'2026-08-03T00:00:00Z', '2026-08-03T00:00:00Z')`); err == nil {
		t.Fatal("invalid Todo priority persisted")
	}
	if _, err := db.Exec(`INSERT INTO workspace_issues(
		id, workspace_id, number, identifier, title, status, priority,
		creator_type, creator_id, created_at, updated_at
	) VALUES ('invalid-issue', 'workspace-1', 1, 'WSP-0', 'Invalid', 'pending', 'medium',
		'member', 'member-1', '2026-08-03T00:00:00Z', '2026-08-03T00:00:00Z')`); err == nil {
		t.Fatal("invalid Issue status persisted")
	}
}

func TestWorkspaceGovernanceMigrationEnforcesIdentityStateAndIsolation(t *testing.T) {
	db := openWorkspaceTestDB(t)
	mustReject := func(name, statement string, args ...any) {
		t.Helper()
		if _, err := db.Exec(statement, args...); err == nil {
			t.Errorf("%s persisted", name)
		}
	}
	mustReject("empty revision workspace", `INSERT INTO workspace_resource_revisions(workspace_id,resource_kind,resource_id,revision,updated_at) VALUES('','task','task-1',1,'2026-08-16T00:00:00Z')`)
	mustReject("empty idempotency action", `INSERT INTO workspace_mutation_idempotency(workspace_id,action,idempotency_key,request_hash,resource_kind,resource_id,resource_revision,response_status,response_body,created_at) VALUES('workspace-1','','key-1','hash','task','task-1',1,201,'{}','2026-08-16T00:00:00Z')`)
	mustReject("invalid replay envelope", `INSERT INTO workspace_mutation_idempotency(workspace_id,action,idempotency_key,request_hash,resource_kind,resource_id,resource_revision,response_status,response_body,created_at) VALUES('workspace-1','workspace.task.create','key-1','aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa','task','task-1',1,201,'{','2026-08-16T00:00:00Z')`)
	mustReject("ready event with claim", `INSERT INTO workspace_outbox_events(state,available_at,workspace_id,id,event_type,aggregate_kind,aggregate_id,aggregate_revision,payload_json,actor_type,actor_id,claim_token,lease_expires_at,created_at) VALUES('ready','2026-08-16T00:00:00Z','workspace-1','event-1','task:created','task','task-1',1,'{}','member','member-1','claim-1','2026-08-16T00:01:00Z','2026-08-16T00:00:00Z')`)

	for _, workspaceID := range []string{"workspace-1", "workspace-2"} {
		if _, err := db.Exec(`INSERT INTO workspace_mutation_idempotency(workspace_id,action,idempotency_key,request_hash,resource_kind,resource_id,resource_revision,response_status,response_body,created_at) VALUES(?,?,?,?,?,?,1,201,'{}','2026-08-16T00:00:00Z')`, workspaceID, "workspace.task.create", "same-key", strings.Repeat("a", 64), "task", "task-1"); err != nil {
			t.Fatalf("insert isolated idempotency row for %s: %v", workspaceID, err)
		}
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM workspace_mutation_idempotency WHERE action='workspace.task.create' AND idempotency_key='same-key'`).Scan(&count); err != nil || count != 2 {
		t.Fatalf("isolated idempotency rows = %d, %v", count, err)
	}
}

func TestNewWithSqliteWorkspaceServicesRequiresSecurityDependencies(t *testing.T) {
	db := openWorkspaceTestDB(t)
	tests := []struct {
		name string
		deps WorkspaceServiceDependencies
	}{
		{name: "authorizer", deps: WorkspaceServiceDependencies{Actors: &workspaceActorCatalog{}}},
		{name: "actor reader", deps: WorkspaceServiceDependencies{Authorizer: &workspaceAccessStub{}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewWithSqliteWorkspaceServices(SqlitePersistenceConfig{DB: db}, tt.deps); err == nil {
				t.Fatal("NewWithSqliteWorkspaceServices() error = nil")
			}
		})
	}
}

func TestSqliteProjectAndRelationshipLocalContracts(t *testing.T) {
	ctx := context.Background()
	db := openWorkspaceTestDB(t)
	seedWorkspace(t, db, "workspace-1", "Acme", "acme")
	seedWorkspace(t, db, "workspace-2", "Globex", "globex")
	actors := &workspaceActorCatalog{actors: map[string]bool{
		"workspace-1/member/member-1": true,
		"workspace-1/agent/agent-1":   true,
	}}
	module := newWorkspaceServicesTestModule(t, db, actors, "project-1")

	created, err := module.ProjectLocal().CreateProject(ctx, contract.CreateProjectRequest{
		WorkspaceId: "workspace-1", Name: "  Delivery  ", Description: "Ship it", AssetIds: []string{"asset-1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Project == nil || created.Project.Id != "project-1" || created.Project.Status != "planned" || created.Project.Name != "Delivery" {
		t.Fatalf("created Project = %+v", created.Project)
	}
	got, err := module.ProjectLocal().GetProject(ctx, contract.GetProjectRequest{WorkspaceId: "workspace-1", ProjectId: "project-1"})
	if err != nil || got.Project == nil || len(got.Project.AssetIds) != 1 || got.Project.AssetIds[0] != "asset-1" {
		t.Fatalf("GetProject() = %+v, %v", got.Project, err)
	}
	if _, err := module.ProjectLocal().GetProject(ctx, contract.GetProjectRequest{WorkspaceId: "workspace-2", ProjectId: "project-1"}); !errors.Is(err, contract.ErrProjectNotFound) {
		t.Fatalf("cross-workspace GetProject() error = %v", err)
	}

	for _, relation := range []contract.ProjectActorRelation{
		{WorkspaceId: "workspace-1", ProjectId: "project-1", ActorType: "member", ActorId: "member-1", Role: "lead"},
		{WorkspaceId: "workspace-1", ProjectId: "project-1", ActorType: "agent", ActorId: "agent-1", Role: "agent"},
	} {
		if _, err := module.RelationshipLocal().PutProjectActorRelation(ctx, contract.PutProjectActorRelationRequest{Relation: &relation}); err != nil {
			t.Fatal(err)
		}
	}
	foreign := contract.ProjectActorRelation{WorkspaceId: "workspace-1", ProjectId: "project-1", ActorType: "member", ActorId: "foreign-member", Role: "member"}
	if _, err := module.RelationshipLocal().PutProjectActorRelation(ctx, contract.PutProjectActorRelationRequest{Relation: &foreign}); !errors.Is(err, contract.ErrActorOutsideWorkspace) {
		t.Fatalf("foreign actor error = %v", err)
	}
	listed, err := module.RelationshipLocal().ListProjectActorRelations(ctx, contract.ListProjectActorRelationsRequest{WorkspaceId: "workspace-1", ProjectId: "project-1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(listed.Relations) != 2 || listed.Relations[0].Role != "agent" || listed.Relations[1].Role != "lead" {
		t.Fatalf("relations = %+v", listed.Relations)
	}
	actors.remove("workspace-1", "member", "member-1")
	deleteRequest := contract.DeleteProjectActorRelationRequest{WorkspaceId: "workspace-1", ProjectId: "project-1", ActorType: "member", ActorId: "member-1"}
	if _, err := module.RelationshipLocal().DeleteProjectActorRelation(ctx, deleteRequest); err != nil {
		t.Fatal(err)
	}
	if _, err := module.RelationshipLocal().DeleteProjectActorRelation(ctx, deleteRequest); err != nil {
		t.Fatalf("idempotent delete error = %v", err)
	}
	listed, err = module.RelationshipLocal().ListProjectActorRelations(ctx, contract.ListProjectActorRelationsRequest{WorkspaceId: "workspace-1", ProjectId: "project-1"})
	if err != nil || len(listed.Relations) != 1 || listed.Relations[0].ActorId != "agent-1" {
		t.Fatalf("relations after delete = %+v, %v", listed.Relations, err)
	}
}

func TestSqliteProjectAndRelationshipGRPCContracts(t *testing.T) {
	ctx := context.Background()
	db := openWorkspaceTestDB(t)
	seedWorkspace(t, db, "workspace-1", "Acme", "acme")
	actors := &workspaceActorCatalog{actors: map[string]bool{"workspace-1/member/member-1": true}}
	module := newWorkspaceServicesTestModule(t, db, actors, "project-grpc")
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	module.RegisterGRPCService(server)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)
	connection, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = connection.Close() })
	projects := NewProjectGRPCClient(connection)
	relations := NewRelationshipGRPCClient(connection)
	created, err := projects.CreateProject(ctx, contract.CreateProjectRequest{WorkspaceId: "workspace-1", Name: "gRPC Project"})
	if err != nil || created.Project == nil || created.Project.Id != "project-grpc" {
		t.Fatalf("CreateProject() = %+v, %v", created.Project, err)
	}
	got, err := projects.GetProject(ctx, contract.GetProjectRequest{WorkspaceId: "workspace-1", ProjectId: "project-grpc"})
	if err != nil || got.Project == nil || got.Project.Name != "gRPC Project" {
		t.Fatalf("GetProject() = %+v, %v", got.Project, err)
	}
	listedProjects, err := projects.ListProjects(ctx, contract.ListProjectsRequest{WorkspaceId: "workspace-1", Status: "planned"})
	if err != nil || listedProjects.Total != 1 || len(listedProjects.Projects) != 1 {
		t.Fatalf("ListProjects() = %+v, %v", listedProjects, err)
	}
	searchedProjects, err := projects.SearchProjects(ctx, contract.SearchProjectsRequest{WorkspaceId: "workspace-1", Query: "grpc"})
	if err != nil || searchedProjects.Total != 1 || len(searchedProjects.Hits) != 1 {
		t.Fatalf("SearchProjects() = %+v, %v", searchedProjects, err)
	}
	updatedName := "Updated gRPC Project"
	updatedProject, err := projects.UpdateProject(ctx, contract.UpdateProjectRequest{WorkspaceId: "workspace-1", ProjectId: "project-grpc", Name: &updatedName})
	if err != nil || updatedProject.Project == nil || updatedProject.Project.Name != updatedName {
		t.Fatalf("UpdateProject() = %+v, %v", updatedProject.Project, err)
	}
	relation := contract.ProjectActorRelation{WorkspaceId: "workspace-1", ProjectId: "project-grpc", ActorType: "member", ActorId: "member-1", Role: "lead"}
	if _, err := relations.PutProjectActorRelation(ctx, contract.PutProjectActorRelationRequest{Relation: &relation}); err != nil {
		t.Fatal(err)
	}
	listed, err := relations.ListProjectActorRelations(ctx, contract.ListProjectActorRelationsRequest{WorkspaceId: "workspace-1", ProjectId: "project-grpc"})
	if err != nil || len(listed.Relations) != 1 || listed.Relations[0].ActorId != "member-1" {
		t.Fatalf("ListProjectActorRelations() = %+v, %v", listed.Relations, err)
	}
	if _, err := relations.DeleteProjectActorRelation(ctx, contract.DeleteProjectActorRelationRequest{WorkspaceId: "workspace-1", ProjectId: "project-grpc", ActorType: "member", ActorId: "member-1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := projects.DeleteProject(ctx, contract.DeleteProjectRequest{WorkspaceId: "workspace-1", ProjectId: "project-grpc"}); err != nil {
		t.Fatal(err)
	}
}
