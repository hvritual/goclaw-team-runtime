package workspace

import (
	"context"
	"database/sql"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
)

type workspaceAssetCatalog struct {
	mu     sync.RWMutex
	assets map[string]bool
}

func (c *workspaceAssetCatalog) AssetBelongsToWorkspace(_ context.Context, workspaceID, assetID string) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.assets[workspaceID+"/"+assetID], nil
}

type skillReferenceCatalog struct {
	mu         sync.RWMutex
	references map[string]bool
}

func (c *skillReferenceCatalog) SkillReferenceExists(_ context.Context, skillID string, versionID *string) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	key := skillID
	if versionID != nil {
		key += "/" + *versionID
	}
	return c.references[key], nil
}

type chainIDSequence struct {
	mu     sync.Mutex
	values []string
}

func (s *chainIDSequence) next(context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.values) == 0 {
		return "", errors.New("test id sequence exhausted")
	}
	value := s.values[0]
	s.values = s.values[1:]
	return value, nil
}

func newWorkspaceChainTestModule(t *testing.T, db *sql.DB, actors *workspaceActorCatalog, assets *workspaceAssetCatalog, skills *skillReferenceCatalog) *Module {
	t.Helper()
	versionIDs := &chainIDSequence{values: []string{"requirement-version-1", "requirement-version-2", "requirement-version-grpc"}}
	todoIDs := &chainIDSequence{values: []string{"todo-1", "todo-2", "todo-grpc"}}
	module, err := NewWithSqliteWorkspaceChain(SqlitePersistenceConfig{DB: db}, WorkspaceServiceDependencies{
		Authorizer:   &workspaceAccessStub{},
		Actors:       actors,
		Assets:       assets,
		Skills:       skills,
		NewProjectID: func(context.Context) (string, error) { return "project-1", nil },
		NewTodoID:    todoIDs.next,
		NewKnowledgeID: func(context.Context) (string, error) {
			return "knowledge-1", nil
		},
		NewRequirementID:        func(context.Context) (string, error) { return "requirement-1", nil },
		NewRequirementVersionID: versionIDs.next,
		Now:                     func() time.Time { return time.Date(2026, 8, 3, 4, 5, 6, 7, time.UTC) },
	})
	if err != nil {
		t.Fatal(err)
	}
	return module
}

func seedWorkspaceIssue(t *testing.T, db *sql.DB, id, workspaceID, identifier, projectID string) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO workspace_issues(
		id, workspace_id, number, identifier, title, status, priority,
		creator_type, creator_id, project_id, metadata, properties, asset_ids,
		created_at, updated_at
	) VALUES (?, ?, 1, ?, 'Seeded Issue', 'backlog', 'medium',
		'member', 'member-1', ?, '{}', '{}', '[]',
		'2026-08-03T00:00:00Z', '2026-08-03T00:00:00Z')`, id, workspaceID, identifier, projectID)
	if err != nil {
		t.Fatal(err)
	}
}

func TestNewWithSqliteWorkspaceChainRequiresBoundaryDependencies(t *testing.T) {
	db := openWorkspaceTestDB(t)
	authorizer := &workspaceAccessStub{}
	actors := &workspaceActorCatalog{}
	assets := &workspaceAssetCatalog{}
	skills := &skillReferenceCatalog{}
	tests := []struct {
		name string
		deps WorkspaceServiceDependencies
	}{
		{name: "authorizer", deps: WorkspaceServiceDependencies{Actors: actors, Assets: assets, Skills: skills}},
		{name: "actors", deps: WorkspaceServiceDependencies{Authorizer: authorizer, Assets: assets, Skills: skills}},
		{name: "assets", deps: WorkspaceServiceDependencies{Authorizer: authorizer, Actors: actors, Skills: skills}},
		{name: "skills", deps: WorkspaceServiceDependencies{Authorizer: authorizer, Actors: actors, Assets: assets}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewWithSqliteWorkspaceChain(SqlitePersistenceConfig{DB: db}, tt.deps); err == nil {
				t.Fatal("NewWithSqliteWorkspaceChain() error = nil")
			}
		})
	}
}

func TestSqliteWorkspaceChainLocalContracts(t *testing.T) {
	ctx := contract.WithWorkspaceActor(context.Background(), "member", "member-1")
	db := openWorkspaceTestDB(t)
	seedWorkspace(t, db, "workspace-1", "Acme", "acme")
	seedWorkspace(t, db, "workspace-2", "Globex", "globex")
	actors := &workspaceActorCatalog{actors: map[string]bool{
		"workspace-1/member/member-1": true,
		"workspace-1/agent/agent-1":   true,
	}}
	assets := &workspaceAssetCatalog{assets: map[string]bool{"workspace-1/asset-1": true}}
	skills := &skillReferenceCatalog{references: map[string]bool{"skill-1/version-1": true}}
	module := newWorkspaceChainTestModule(t, db, actors, assets, skills)

	projectResponse, err := module.ProjectLocal().CreateProject(ctx, contract.CreateProjectRequest{WorkspaceId: "workspace-1", Name: "Delivery"})
	if err != nil || projectResponse.Project == nil {
		t.Fatalf("CreateProject() = %+v, %v", projectResponse.Project, err)
	}
	seedWorkspaceIssue(t, db, "issue-1", "workspace-1", "WSP-1", "project-1")
	seedWorkspaceIssue(t, db, "issue-other-project", "workspace-1", "WSP-2", "project-other")
	seedWorkspaceIssue(t, db, "issue-foreign", "workspace-2", "WSP-3", "project-foreign")

	projectID, issueID := "project-1", "WSP-1"
	assigneeType, assigneeID := "member", "member-1"
	createdTodo, err := module.TodoLocal().CreateTodo(ctx, contract.CreateTodoRequest{
		WorkspaceId: "workspace-1", Title: "  Ship migration  ", ProjectId: &projectID, IssueId: &issueID,
		AssigneeType: &assigneeType, AssigneeId: &assigneeID,
	})
	if err != nil || createdTodo.Todo == nil || createdTodo.Todo.Status != "todo" || createdTodo.Todo.Title != "Ship migration" {
		t.Fatalf("CreateTodo() = %+v, %v", createdTodo.Todo, err)
	}
	if createdTodo.Todo.IssueId == nil || *createdTodo.Todo.IssueId != "issue-1" || createdTodo.Todo.CreatorId != "member-1" {
		t.Fatalf("Todo references/creator = %+v", createdTodo.Todo)
	}
	gotTodo, err := module.TodoLocal().GetTodo(ctx, contract.GetTodoRequest{WorkspaceId: "workspace-1", TodoId: "todo-1"})
	if err != nil || gotTodo.Todo == nil || gotTodo.Todo.Id != "todo-1" {
		t.Fatalf("GetTodo() = %+v, %v", gotTodo.Todo, err)
	}
	secondTodo, err := module.TodoLocal().CreateTodo(ctx, contract.CreateTodoRequest{
		WorkspaceId: "workspace-1", Title: "First in list", ProjectId: &projectID,
		Status: "in_progress", Priority: "urgent", Position: -1,
	})
	if err != nil || secondTodo.Todo == nil || secondTodo.Todo.Id != "todo-2" {
		t.Fatalf("CreateTodo(second) = %+v, %v", secondTodo.Todo, err)
	}
	listedTodos, err := module.TodoLocal().ListTodos(ctx, contract.ListTodosRequest{WorkspaceId: "workspace-1", ProjectId: &projectID})
	if err != nil || listedTodos.Total != 2 || len(listedTodos.Todos) != 2 || listedTodos.Todos[0].Id != "todo-2" {
		t.Fatalf("ListTodos() = %+v, %v", listedTodos, err)
	}
	updatedTitle, done, high, empty := "Updated Todo", "done", "high", ""
	startDate := "2026-08-04T01:02:03+08:00"
	updatedTodo, err := module.TodoLocal().UpdateTodo(ctx, contract.UpdateTodoRequest{
		WorkspaceId: "workspace-1", TodoId: "todo-1", Title: &updatedTitle,
		Status: &done, Priority: &high, ProjectId: &empty, IssueId: &empty,
		AssigneeId: &empty, StartDate: &startDate,
	})
	if err != nil || updatedTodo.Todo == nil || updatedTodo.Todo.Status != "done" || updatedTodo.Todo.CompletedAt == nil || updatedTodo.Todo.ProjectId != nil || updatedTodo.Todo.AssigneeId != nil {
		t.Fatalf("UpdateTodo() = %+v, %v", updatedTodo.Todo, err)
	}
	updatedTodoStatus, err := module.TodoLocal().UpdateTodoStatus(ctx, contract.UpdateTodoStatusRequest{WorkspaceId: "workspace-1", TodoId: "todo-1", Status: "in_progress"})
	if err != nil || updatedTodoStatus.Todo == nil || updatedTodoStatus.Todo.Status != "in_progress" || updatedTodoStatus.Todo.CompletedAt != nil {
		t.Fatalf("UpdateTodoStatus() = %+v, %v", updatedTodoStatus.Todo, err)
	}
	if _, err := module.TodoLocal().GetTodo(ctx, contract.GetTodoRequest{WorkspaceId: "workspace-2", TodoId: "todo-1"}); !errors.Is(err, contract.ErrTodoNotFound) {
		t.Fatalf("cross-workspace GetTodo error = %v", err)
	}
	if _, err := module.TodoLocal().DeleteTodo(ctx, contract.DeleteTodoRequest{WorkspaceId: "workspace-1", TodoId: "todo-2"}); err != nil {
		t.Fatalf("DeleteTodo() error = %v", err)
	}
	if _, err := module.TodoLocal().DeleteTodo(ctx, contract.DeleteTodoRequest{WorkspaceId: "workspace-1", TodoId: "todo-2"}); !errors.Is(err, contract.ErrTodoNotFound) {
		t.Fatalf("repeated DeleteTodo error = %v", err)
	}
	foreignIssue := "issue-foreign"
	if _, err := module.TodoLocal().CreateTodo(ctx, contract.CreateTodoRequest{WorkspaceId: "workspace-1", Title: "bad", IssueId: &foreignIssue}); !errors.Is(err, contract.ErrIssueNotFound) {
		t.Fatalf("foreign Todo Issue error = %v", err)
	}
	foreignProject := "project-foreign"
	if _, err := module.TodoLocal().CreateTodo(ctx, contract.CreateTodoRequest{WorkspaceId: "workspace-1", Title: "bad", ProjectId: &foreignProject}); !errors.Is(err, contract.ErrProjectNotFound) {
		t.Fatalf("foreign Todo Project error = %v", err)
	}
	foreignAssignee := "foreign-member"
	if _, err := module.TodoLocal().CreateTodo(ctx, contract.CreateTodoRequest{WorkspaceId: "workspace-1", Title: "bad", AssigneeType: &assigneeType, AssigneeId: &foreignAssignee}); !errors.Is(err, contract.ErrActorOutsideWorkspace) {
		t.Fatalf("foreign Todo Actor error = %v", err)
	}

	updatedIssue, err := module.IssueLocal().UpdateIssueStatus(ctx, contract.UpdateIssueStatusRequest{WorkspaceId: "workspace-1", IssueId: "WSP-1", Status: "in_progress"})
	if err != nil || updatedIssue.Issue == nil || updatedIssue.Issue.Id != "issue-1" || updatedIssue.Issue.Status != "in_progress" {
		t.Fatalf("UpdateIssueStatus() = %+v, %v", updatedIssue.Issue, err)
	}
	if _, err := module.IssueLocal().UpdateIssueStatus(ctx, contract.UpdateIssueStatusRequest{WorkspaceId: "workspace-1", IssueId: "issue-foreign", Status: "done"}); !errors.Is(err, contract.ErrIssueNotFound) {
		t.Fatalf("cross-workspace Issue error = %v", err)
	}
	if _, err := module.IssueLocal().UpdateIssueStatus(ctx, contract.UpdateIssueStatusRequest{WorkspaceId: "workspace-1", IssueId: "issue-1", Status: "DONE"}); !errors.Is(err, contract.ErrInvalidIssue) {
		t.Fatalf("invalid Issue status error = %v", err)
	}

	createdKnowledge, err := module.KnowledgeLocal().CreateKnowledge(ctx, contract.CreateKnowledgeRequest{WorkspaceId: "workspace-1", Title: "Runbook", AssetIds: []string{"asset-1"}})
	if err != nil || createdKnowledge.Knowledge == nil || createdKnowledge.Knowledge.Status != "candidate" {
		t.Fatalf("CreateKnowledge() = %+v, %v", createdKnowledge.Knowledge, err)
	}
	gotKnowledge, err := module.KnowledgeLocal().GetKnowledge(ctx, contract.GetKnowledgeRequest{WorkspaceId: "workspace-1", KnowledgeId: "knowledge-1"})
	if err != nil || gotKnowledge.Knowledge == nil || gotKnowledge.Knowledge.AssetIds[0] != "asset-1" {
		t.Fatalf("GetKnowledge() = %+v, %v", gotKnowledge.Knowledge, err)
	}
	if _, err := module.KnowledgeLocal().CreateKnowledge(ctx, contract.CreateKnowledgeRequest{WorkspaceId: "workspace-1", Title: "bad", AssetIds: []string{"asset-foreign"}}); !errors.Is(err, contract.ErrAssetOutsideWorkspace) {
		t.Fatalf("foreign Knowledge Asset error = %v", err)
	}
	if _, err := module.KnowledgeLocal().GetKnowledge(ctx, contract.GetKnowledgeRequest{WorkspaceId: "workspace-2", KnowledgeId: "knowledge-1"}); !errors.Is(err, contract.ErrKnowledgeNotFound) {
		t.Fatalf("cross-workspace Knowledge error = %v", err)
	}

	savedRequirement, err := module.RequirementLocal().SaveRequirementVersion(ctx, contract.SaveRequirementVersionRequest{
		WorkspaceId: "workspace-1", ProjectId: "project-1", Title: "First", Content: "v1", IssueIds: []string{"issue-1", "issue-1"},
	})
	if err != nil || savedRequirement.Requirement == nil || savedRequirement.Version == nil || savedRequirement.Requirement.CurrentVersion != 1 || len(savedRequirement.Requirement.IssueIds) != 1 {
		t.Fatalf("SaveRequirementVersion(v1) = %+v / %+v, %v", savedRequirement.Requirement, savedRequirement.Version, err)
	}
	requirementID := savedRequirement.Requirement.Id
	savedRequirement, err = module.RequirementLocal().SaveRequirementVersion(ctx, contract.SaveRequirementVersionRequest{
		WorkspaceId: "workspace-1", ProjectId: "project-1", RequirementId: &requirementID, Title: "Second", Content: "v2", IssueIds: []string{"WSP-1"},
	})
	if err != nil || savedRequirement.Requirement.CurrentVersion != 2 || savedRequirement.Version.Version != 2 {
		t.Fatalf("SaveRequirementVersion(v2) = %+v / %+v, %v", savedRequirement.Requirement, savedRequirement.Version, err)
	}
	gotRequirement, err := module.RequirementLocal().GetRequirement(ctx, contract.GetRequirementRequest{WorkspaceId: "workspace-1", RequirementId: requirementID})
	if err != nil || gotRequirement.CurrentVersion == nil || gotRequirement.CurrentVersion.Content != "v2" {
		t.Fatalf("GetRequirement() = %+v, %v", gotRequirement, err)
	}
	if _, err := module.RequirementLocal().SaveRequirementVersion(ctx, contract.SaveRequirementVersionRequest{WorkspaceId: "workspace-1", ProjectId: "project-1", Title: "bad", Content: "bad", IssueIds: []string{"issue-other-project"}}); !errors.Is(err, contract.ErrInvalidRequirement) {
		t.Fatalf("cross-project Requirement Issue error = %v", err)
	}
	if _, err := module.RequirementLocal().SaveRequirementVersion(ctx, contract.SaveRequirementVersionRequest{WorkspaceId: "workspace-1", ProjectId: "project-1", Title: "bad", Content: "bad", IssueIds: []string{"issue-foreign"}}); !errors.Is(err, contract.ErrIssueNotFound) {
		t.Fatalf("foreign Requirement Issue error = %v", err)
	}

	settingResponse, err := module.SettingLocal().PutWorkspaceSetting(ctx, contract.PutWorkspaceSettingRequest{WorkspaceId: "workspace-1", Key: "notifications", Value: map[string]any{"email": true}})
	if err != nil || settingResponse.Setting == nil || settingResponse.Setting.Value["email"] != true {
		t.Fatalf("PutWorkspaceSetting() = %+v, %v", settingResponse.Setting, err)
	}
	versionID := "version-1"
	bindingResponse, err := module.SettingLocal().PutWorkspaceSkillBinding(ctx, contract.PutWorkspaceSkillBindingRequest{
		WorkspaceId: "workspace-1", SkillId: "skill-1", SkillVersionId: &versionID, Enabled: true,
		Configuration: map[string]any{"mode": "strict"}, AgentIds: []string{"agent-1", "agent-1"},
	})
	if err != nil || bindingResponse.Binding == nil || len(bindingResponse.Binding.AgentIds) != 1 {
		t.Fatalf("PutWorkspaceSkillBinding() = %+v, %v", bindingResponse.Binding, err)
	}
	if _, err := module.SettingLocal().PutWorkspaceSkillBinding(ctx, contract.PutWorkspaceSkillBindingRequest{WorkspaceId: "workspace-1", SkillId: "missing"}); !errors.Is(err, contract.ErrSkillReferenceNotFound) {
		t.Fatalf("missing Skill error = %v", err)
	}
	if _, err := module.SettingLocal().PutWorkspaceSkillBinding(ctx, contract.PutWorkspaceSkillBindingRequest{WorkspaceId: "workspace-1", SkillId: "skill-1", SkillVersionId: &versionID, AgentIds: []string{"foreign-agent"}}); !errors.Is(err, contract.ErrActorOutsideWorkspace) {
		t.Fatalf("foreign Skill Agent error = %v", err)
	}

	var todoStatus, settingValue, agentIDs string
	if err := db.QueryRow(`SELECT status FROM workspace_todos WHERE id = 'todo-1'`).Scan(&todoStatus); err != nil || todoStatus != "in_progress" {
		t.Fatalf("persisted Todo status = %q, %v", todoStatus, err)
	}
	if err := db.QueryRow(`SELECT value FROM workspace_settings WHERE workspace_id = 'workspace-1' AND key = 'notifications'`).Scan(&settingValue); err != nil || settingValue != `{"email":true}` {
		t.Fatalf("persisted setting = %q, %v", settingValue, err)
	}
	if err := db.QueryRow(`SELECT agent_ids FROM workspace_skill_bindings WHERE workspace_id = 'workspace-1' AND skill_id = 'skill-1'`).Scan(&agentIDs); err != nil || agentIDs != `["agent-1"]` {
		t.Fatalf("persisted Agent ids = %q, %v", agentIDs, err)
	}
	var versionCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM workspace_requirement_versions WHERE requirement_id = ?`, requirementID).Scan(&versionCount); err != nil || versionCount != 2 {
		t.Fatalf("Requirement version count = %d, %v", versionCount, err)
	}
}

func TestSqliteWorkspaceChainGRPCContracts(t *testing.T) {
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs(
		"x-workspace-actor-type", "agent",
		"x-workspace-actor-id", "agent-1",
	))
	db := openWorkspaceTestDB(t)
	seedWorkspace(t, db, "workspace-1", "Acme", "acme")
	actors := &workspaceActorCatalog{actors: map[string]bool{"workspace-1/agent/agent-1": true}}
	assets := &workspaceAssetCatalog{assets: map[string]bool{"workspace-1/asset-1": true}}
	skills := &skillReferenceCatalog{references: map[string]bool{"skill-1": true}}
	module := newWorkspaceChainTestModule(t, db, actors, assets, skills)
	if _, err := module.ProjectLocal().CreateProject(ctx, contract.CreateProjectRequest{WorkspaceId: "workspace-1", Name: "Delivery"}); err != nil {
		t.Fatal(err)
	}
	seedWorkspaceIssue(t, db, "issue-1", "workspace-1", "WSP-1", "project-1")

	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer(grpc.UnaryInterceptor(workspaceActorTestInterceptor))
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

	projectID, issueID := "project-1", "issue-1"
	createdTodo, err := NewTodoGRPCClient(connection).CreateTodo(ctx, contract.CreateTodoRequest{WorkspaceId: "workspace-1", Title: "gRPC Todo", ProjectId: &projectID, IssueId: &issueID})
	if err != nil || createdTodo.Todo == nil || createdTodo.Todo.Status != "todo" {
		t.Fatalf("gRPC CreateTodo() = %+v, %v", createdTodo.Todo, err)
	}
	todos := NewTodoGRPCClient(connection)
	gotTodo, err := todos.GetTodo(ctx, contract.GetTodoRequest{WorkspaceId: "workspace-1", TodoId: createdTodo.Todo.Id})
	if err != nil || gotTodo.Todo == nil || gotTodo.Todo.CreatorId != "agent-1" {
		t.Fatalf("gRPC GetTodo() = %+v, %v", gotTodo.Todo, err)
	}
	listedTodos, err := todos.ListTodos(ctx, contract.ListTodosRequest{WorkspaceId: "workspace-1", Status: "todo"})
	if err != nil || listedTodos.Total != 1 {
		t.Fatalf("gRPC ListTodos() = %+v, %v", listedTodos, err)
	}
	done := "done"
	updatedTodo, err := todos.UpdateTodo(ctx, contract.UpdateTodoRequest{WorkspaceId: "workspace-1", TodoId: createdTodo.Todo.Id, Status: &done})
	if err != nil || updatedTodo.Todo == nil || updatedTodo.Todo.CompletedAt == nil {
		t.Fatalf("gRPC UpdateTodo() = %+v, %v", updatedTodo.Todo, err)
	}
	if _, err := todos.DeleteTodo(ctx, contract.DeleteTodoRequest{WorkspaceId: "workspace-1", TodoId: createdTodo.Todo.Id}); err != nil {
		t.Fatalf("gRPC DeleteTodo() error = %v", err)
	}
	updatedIssue, err := NewIssueGRPCClient(connection).UpdateIssueStatus(ctx, contract.UpdateIssueStatusRequest{WorkspaceId: "workspace-1", IssueId: "WSP-1", Status: "done"})
	if err != nil || updatedIssue.Issue == nil || updatedIssue.Issue.Status != "done" {
		t.Fatalf("gRPC UpdateIssueStatus() = %+v, %v", updatedIssue.Issue, err)
	}
	createdKnowledge, err := NewKnowledgeGRPCClient(connection).CreateKnowledge(ctx, contract.CreateKnowledgeRequest{WorkspaceId: "workspace-1", Title: "gRPC Knowledge", AssetIds: []string{"asset-1"}})
	if err != nil || createdKnowledge.Knowledge == nil || createdKnowledge.Knowledge.Status != "candidate" {
		t.Fatalf("gRPC CreateKnowledge() = %+v, %v", createdKnowledge.Knowledge, err)
	}
	savedRequirement, err := NewRequirementGRPCClient(connection).SaveRequirementVersion(ctx, contract.SaveRequirementVersionRequest{WorkspaceId: "workspace-1", ProjectId: "project-1", Title: "gRPC Requirement", Content: "v1", IssueIds: []string{"issue-1"}})
	if err != nil || savedRequirement.Requirement == nil || savedRequirement.Requirement.CoverageStatus != "covered" {
		t.Fatalf("gRPC SaveRequirementVersion() = %+v, %v", savedRequirement.Requirement, err)
	}
	binding, err := NewSettingGRPCClient(connection).PutWorkspaceSkillBinding(ctx, contract.PutWorkspaceSkillBindingRequest{WorkspaceId: "workspace-1", SkillId: "skill-1", Enabled: true, AgentIds: []string{"agent-1"}})
	if err != nil || binding.Binding == nil || !binding.Binding.Enabled {
		t.Fatalf("gRPC PutWorkspaceSkillBinding() = %+v, %v", binding.Binding, err)
	}
}

func workspaceActorTestInterceptor(ctx context.Context, request any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	metadataValues, _ := metadata.FromIncomingContext(ctx)
	actorTypes, actorIDs := metadataValues.Get("x-workspace-actor-type"), metadataValues.Get("x-workspace-actor-id")
	if len(actorTypes) != 0 && len(actorIDs) != 0 {
		ctx = contract.WithWorkspaceActor(ctx, actorTypes[0], actorIDs[0])
	}
	return handler(ctx, request)
}
