package workspace

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
)

type issueIDSequence struct {
	mu   sync.Mutex
	next int
}

func (s *issueIDSequence) generate(context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.next++
	return fmt.Sprintf("issue-generated-%03d", s.next), nil
}

func seedIssueProject(t *testing.T, db *sql.DB, workspaceID, projectID string) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO workspace_projects(
		id, workspace_id, name, description, status, asset_ids, created_at, updated_at
	) VALUES (?, ?, 'Project', '', 'planned', '[]', '2026-08-03T00:00:00Z', '2026-08-03T00:00:00Z')`, projectID, workspaceID)
	if err != nil {
		t.Fatal(err)
	}
}

func newIssueMainlineModule(t *testing.T, db *sql.DB, sequence *issueIDSequence, now func() time.Time) *Module {
	t.Helper()
	actors := &workspaceActorCatalog{actors: map[string]bool{
		"workspace-1/member/member-1": true,
		"workspace-1/agent/agent-1":   true,
		"workspace-2/member/member-2": true,
	}}
	assets := &workspaceAssetCatalog{assets: map[string]bool{
		"workspace-1/asset-1": true,
		"workspace-1/asset-2": true,
	}}
	module, err := NewWithSqliteWorkspaceChain(SqlitePersistenceConfig{DB: db}, WorkspaceServiceDependencies{
		Authorizer: &workspaceAccessStub{}, Actors: actors, Assets: assets,
		WorkspaceMemberships: selectionMemberships{},
		Skills:               &skillReferenceCatalog{references: map[string]bool{}}, NewIssueID: sequence.generate, Now: now,
		HTTPIdentity: func(request *http.Request) (contract.WorkspaceHTTPIdentity, error) {
			if request.Header.Get("Authorization") != "Bearer test-token" || request.Header.Get("X-Workspace-Slug") != "acme" {
				return contract.WorkspaceHTTPIdentity{}, contract.ErrWorkspaceActorRequired
			}
			return contract.WorkspaceHTTPIdentity{WorkspaceID: "workspace-1", ActorType: "member", ActorID: "member-1"}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return module
}

func TestIssueMainlineSQLiteLocalCRUDFilterOrderClearAndIsolation(t *testing.T) {
	db := openWorkspaceTestDB(t)
	seedWorkspace(t, db, "workspace-1", "Acme", "acme")
	seedWorkspace(t, db, "workspace-2", "Globex", "globex")
	seedIssueProject(t, db, "workspace-1", "project-1")
	sequence := &issueIDSequence{}
	now := time.Date(2026, 8, 3, 8, 0, 0, 0, time.UTC)
	module := newIssueMainlineModule(t, db, sequence, func() time.Time { return now })
	ctx := contract.WithWorkspaceActor(context.Background(), "member", "member-1")

	projectID, assigneeType, assigneeID := "project-1", "agent", "agent-1"
	startDate, dueDate, description := "2026-08-04", "2026-08-05", "details"
	stage := int32(2)
	parentResponse, err := module.IssueLocal().CreateIssue(ctx, contract.CreateIssueRequest{
		WorkspaceId: "workspace-1", Title: "Parent", Description: &description, Status: "backlog", Priority: "high",
		ProjectId: &projectID, AssigneeType: &assigneeType, AssigneeId: &assigneeID,
		Position: 2, Stage: &stage, StartDate: &startDate, DueDate: &dueDate, AssetIds: []string{"asset-1"},
	})
	if err != nil || parentResponse.Issue == nil || parentResponse.Issue.Identifier != "WSP-1" || parentResponse.Issue.CreatorId != "member-1" {
		t.Fatalf("CreateIssue(parent) = %+v, %v", parentResponse.Issue, err)
	}
	parentID := parentResponse.Issue.Id
	now = now.Add(time.Minute)
	childResponse, err := module.IssueLocal().CreateIssue(ctx, contract.CreateIssueRequest{
		WorkspaceId: "workspace-1", Title: "Child", ParentIssueId: &parentID, ProjectId: &projectID,
		Status: "todo", Priority: "medium", Position: 1,
	})
	if err != nil || childResponse.Issue == nil || childResponse.Issue.Identifier != "WSP-2" || childResponse.Issue.ParentIssueId == nil || *childResponse.Issue.ParentIssueId != parentID {
		t.Fatalf("CreateIssue(child) = %+v, %v", childResponse.Issue, err)
	}

	if _, err := db.Exec(`UPDATE workspace_issues SET metadata = '{"source":"test"}', properties = '{"property-1":"value"}' WHERE id = ?`, parentID); err != nil {
		t.Fatal(err)
	}
	got, err := module.IssueLocal().GetIssue(ctx, contract.GetIssueRequest{WorkspaceId: "workspace-1", IssueId: "WSP-1"})
	if err != nil || got.Issue == nil || got.Issue.Metadata["source"] != "test" || got.Issue.Properties["property-1"] != "value" || len(got.Issue.AssetIds) != 1 {
		t.Fatalf("GetIssue() = %+v, %v", got.Issue, err)
	}
	listed, err := module.IssueLocal().ListIssues(ctx, contract.ListIssuesRequest{WorkspaceId: "workspace-1", ProjectId: &projectID})
	if err != nil || listed.Total != 2 || listed.Issues[0].Id != childResponse.Issue.Id || listed.Issues[1].Id != parentID {
		t.Fatalf("ListIssues(order) = %+v, %v", listed, err)
	}
	filtered, err := module.IssueLocal().ListIssues(ctx, contract.ListIssuesRequest{WorkspaceId: "workspace-1", Status: "backlog", Priority: "high", AssigneeType: &assigneeType, AssigneeId: &assigneeID})
	if err != nil || filtered.Total != 1 || filtered.Issues[0].Id != parentID {
		t.Fatalf("ListIssues(filter) = %+v, %v", filtered, err)
	}

	empty, title, status, priority := "", "Updated Parent", "in_review", "urgent"
	zeroStage := int32(0)
	now = now.Add(time.Minute)
	updated, err := module.IssueLocal().UpdateIssue(ctx, contract.UpdateIssueRequest{
		WorkspaceId: "workspace-1", IssueId: "WSP-1", Title: &title, Status: &status, Priority: &priority,
		AssigneeType: &empty, AssigneeId: &empty, ProjectId: &empty, ParentIssueId: &empty,
		Stage: &zeroStage, StartDate: &empty, DueDate: &empty, AssetIds: &contract.IssueAssetIDs{Values: []string{"asset-2"}},
	})
	if err != nil || updated.Issue == nil || updated.Issue.AssigneeId != nil || updated.Issue.ProjectId != nil || updated.Issue.ParentIssueId != nil || updated.Issue.Stage != nil || updated.Issue.StartDate != nil || updated.Issue.DueDate != nil || len(updated.Issue.AssetIds) != 1 || updated.Issue.AssetIds[0] != "asset-2" {
		t.Fatalf("UpdateIssue(clear) = %+v, %v", updated.Issue, err)
	}
	if updated.Issue.Metadata["source"] != "test" || updated.Issue.Properties["property-1"] != "value" {
		t.Fatalf("UpdateIssue lost projections = %+v", updated.Issue)
	}
	statusResult, err := module.IssueLocal().UpdateIssueStatus(ctx, contract.UpdateIssueStatusRequest{WorkspaceId: "workspace-1", IssueId: parentID, Status: "done"})
	if err != nil || statusResult.Issue == nil || statusResult.Issue.Status != "done" {
		t.Fatalf("UpdateIssueStatus() = %+v, %v", statusResult.Issue, err)
	}
	if _, err := module.IssueLocal().GetIssue(ctx, contract.GetIssueRequest{WorkspaceId: "workspace-2", IssueId: parentID}); !errors.Is(err, contract.ErrIssueNotFound) {
		t.Fatalf("cross-Workspace GetIssue error = %v", err)
	}
	foreignList, err := module.IssueLocal().ListIssues(ctx, contract.ListIssuesRequest{WorkspaceId: "workspace-2"})
	if err != nil || foreignList.Total != 0 {
		t.Fatalf("cross-Workspace ListIssues = %+v, %v", foreignList, err)
	}
}

func TestIssueIdentifierAllocationConcurrentAndFailureRollback(t *testing.T) {
	db := openWorkspaceTestDB(t)
	db.SetMaxOpenConns(8)
	seedWorkspace(t, db, "workspace-1", "Acme", "acme")
	sequence := &issueIDSequence{}
	module := newIssueMainlineModule(t, db, sequence, func() time.Time { return time.Now().UTC() })
	ctx := contract.WithWorkspaceActor(context.Background(), "member", "member-1")

	const count = 24
	numbers := make(chan int32, count)
	errorsFound := make(chan error, count)
	var group sync.WaitGroup
	for index := 0; index < count; index++ {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			response, err := module.IssueLocal().CreateIssue(ctx, contract.CreateIssueRequest{WorkspaceId: "workspace-1", Title: fmt.Sprintf("Issue %d", index)})
			if err != nil {
				errorsFound <- err
				return
			}
			numbers <- response.Issue.Number
		}(index)
	}
	group.Wait()
	close(numbers)
	close(errorsFound)
	for err := range errorsFound {
		t.Fatalf("concurrent CreateIssue() error = %v", err)
	}
	allocated := make([]int, 0, count)
	for number := range numbers {
		allocated = append(allocated, int(number))
	}
	sort.Ints(allocated)
	if len(allocated) != count {
		t.Fatalf("allocated count = %d", len(allocated))
	}
	for index, number := range allocated {
		if number != index+1 {
			t.Fatalf("allocated numbers = %v", allocated)
		}
	}

	if _, err := db.Exec(`CREATE TRIGGER fail_issue_mainline_create BEFORE INSERT ON workspace_issues
		WHEN NEW.title = 'Fail' BEGIN SELECT RAISE(ABORT, 'forced issue failure'); END`); err != nil {
		t.Fatal(err)
	}
	var counterBefore int
	if err := db.QueryRow(`SELECT next_issue_number FROM workspaces WHERE id = 'workspace-1'`).Scan(&counterBefore); err != nil {
		t.Fatal(err)
	}
	if _, err := module.IssueLocal().CreateIssue(ctx, contract.CreateIssueRequest{WorkspaceId: "workspace-1", Title: "Fail"}); err == nil {
		t.Fatal("forced CreateIssue() error = nil")
	}
	var counterAfter, issueCount int
	if err := db.QueryRow(`SELECT next_issue_number FROM workspaces WHERE id = 'workspace-1'`).Scan(&counterAfter); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM workspace_issues`).Scan(&issueCount); err != nil {
		t.Fatal(err)
	}
	if counterAfter != counterBefore || issueCount != count {
		t.Fatalf("failed create leaked state: counter %d/%d issues %d/%d", counterAfter, counterBefore, issueCount, count)
	}
}

func TestIssueMainlineBufconnAllRPCs(t *testing.T) {
	db := openWorkspaceTestDB(t)
	seedWorkspace(t, db, "workspace-1", "Acme", "acme")
	sequence := &issueIDSequence{}
	module := newIssueMainlineModule(t, db, sequence, func() time.Time { return time.Date(2026, 8, 3, 9, 0, 0, 0, time.UTC) })
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
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("x-workspace-actor-type", "agent", "x-workspace-actor-id", "agent-1"))
	client := NewIssueGRPCClient(connection)
	created, err := client.CreateIssue(ctx, contract.CreateIssueRequest{WorkspaceId: "workspace-1", Title: "gRPC Issue", AssetIds: []string{"asset-1"}})
	if err != nil || created.Issue == nil || created.Issue.Identifier != "WSP-1" || created.Issue.CreatorId != "agent-1" {
		t.Fatalf("gRPC CreateIssue() = %+v, %v", created.Issue, err)
	}
	got, err := client.GetIssue(ctx, contract.GetIssueRequest{WorkspaceId: "workspace-1", IssueId: "WSP-1"})
	if err != nil || got.Issue == nil || got.Issue.Id != created.Issue.Id {
		t.Fatalf("gRPC GetIssue() = %+v, %v", got.Issue, err)
	}
	listed, err := client.ListIssues(ctx, contract.ListIssuesRequest{WorkspaceId: "workspace-1", Status: "todo"})
	if err != nil || listed.Total != 1 {
		t.Fatalf("gRPC ListIssues() = %+v, %v", listed, err)
	}
	title := "Updated over gRPC"
	updated, err := client.UpdateIssue(ctx, contract.UpdateIssueRequest{WorkspaceId: "workspace-1", IssueId: created.Issue.Id, Title: &title})
	if err != nil || updated.Issue == nil || updated.Issue.Title != title {
		t.Fatalf("gRPC UpdateIssue() = %+v, %v", updated.Issue, err)
	}
	status, err := client.UpdateIssueStatus(ctx, contract.UpdateIssueStatusRequest{WorkspaceId: "workspace-1", IssueId: "WSP-1", Status: "blocked"})
	if err != nil || status.Issue == nil || status.Issue.Status != "blocked" {
		t.Fatalf("gRPC UpdateIssueStatus() = %+v, %v", status.Issue, err)
	}
}
