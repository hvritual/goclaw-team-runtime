package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	issueDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/issue"
	todoDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/todo"
)

type todoRepositoryStub struct {
	value          todoDomain.Todo
	values         []todoDomain.Todo
	query          TodoListQuery
	findErr        error
	createErr      error
	updateErr      error
	createCalls    int
	findCalls      int
	listCalls      int
	updateCalls    int
	reorderCalls   int
	reorderCommand TodoGovernanceCommand
}

func (s *todoRepositoryStub) Create(_ context.Context, value todoDomain.Todo) (todoDomain.Todo, error) {
	s.createCalls++
	s.value = value
	return value, s.createErr
}

func (s *todoRepositoryStub) FindByID(context.Context, string, string) (todoDomain.Todo, error) {
	s.findCalls++
	return s.value, s.findErr
}

func (s *todoRepositoryStub) List(_ context.Context, query TodoListQuery) ([]todoDomain.Todo, error) {
	s.listCalls++
	s.query = query
	return append([]todoDomain.Todo(nil), s.values...), nil
}

func (s *todoRepositoryStub) Update(_ context.Context, value todoDomain.Todo) error {
	s.updateCalls++
	s.value = value
	return s.updateErr
}

func (s *todoRepositoryStub) Reorder(ctx context.Context, _ string, updates []TodoPositionUpdate, now time.Time) ([]todoDomain.Todo, error) {
	s.reorderCalls++
	s.reorderCommand, _ = TodoGovernanceCommandFromContext(ctx)
	values := make([]todoDomain.Todo, 0, len(updates))
	for _, update := range updates {
		value := s.value
		value.ID = update.TodoID
		value.Position = update.Position
		value.Revision = update.ExpectedRevision + 1
		value.UpdatedAt = now
		values = append(values, value)
	}
	return values, nil
}

type todoIssueRepositoryStub struct {
	value issueDomain.Issue
	err   error
	calls int
}

func (s *todoIssueRepositoryStub) FindByIDOrIdentifier(context.Context, string, string) (issueDomain.Issue, error) {
	s.calls++
	return s.value, s.err
}

func (*todoIssueRepositoryStub) UpdateStatus(context.Context, issueDomain.Issue) error { return nil }

func newTodoApplicationValue(t *testing.T) todoDomain.Todo {
	t.Helper()
	value, err := todoDomain.New(
		"todo-1", "workspace-1", "Todo", "details", todoDomain.StatusTodo, todoDomain.PriorityNone,
		nil, nil, nil, nil, "member", "member-1", 0, nil, nil,
		time.Date(2026, 8, 3, 1, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func newTodoApplicationService(t *testing.T, repository *todoRepositoryStub, authorizer *accessAuthorizerStub, actors *actorReaderStub, issues *todoIssueRepositoryStub, now time.Time) *TodoUseCase {
	t.Helper()
	service, err := NewTodoUseCase(
		repository,
		&projectRepositoryStub{},
		issues,
		authorizer,
		actors,
		func(context.Context) (string, error) { return "todo-new", nil },
		func() time.Time { return now },
	)
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func TestTodoUseCaseCreatesWithTrustedActorAndCanonicalIssue(t *testing.T) {
	now := time.Date(2026, 8, 3, 2, 0, 0, 0, time.UTC)
	repository := &todoRepositoryStub{}
	actors := &actorReaderStub{belongs: true}
	issues := &todoIssueRepositoryStub{value: issueDomain.Issue{ID: "issue-uuid"}}
	service := newTodoApplicationService(t, repository, &accessAuthorizerStub{}, actors, issues, now)
	ctx := contract.WithWorkspaceActor(context.Background(), "member", "member-1")
	issueIdentifier, assigneeType, assigneeID := "WSP-1", "agent", "agent-1"
	response, err := service.CreateTodo(ctx, contract.CreateTodoRequest{
		WorkspaceId: "workspace-1", Title: "  Ship  ", IssueId: &issueIdentifier,
		AssigneeType: &assigneeType, AssigneeId: &assigneeID, Priority: "high",
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.Todo == nil || response.Todo.CreatorId != "member-1" || response.Todo.Priority != "high" || response.Todo.IssueId == nil || *response.Todo.IssueId != "issue-uuid" {
		t.Fatalf("CreateTodo() = %+v", response.Todo)
	}
	if repository.createCalls != 1 || actors.calls != 2 || issues.calls != 1 {
		t.Fatalf("calls = repo:%d actors:%d issues:%d", repository.createCalls, actors.calls, issues.calls)
	}
}

func TestTodoUseCaseRequiresTrustedCreatorAndWorkspaceMembership(t *testing.T) {
	service := newTodoApplicationService(t, &todoRepositoryStub{}, &accessAuthorizerStub{}, &actorReaderStub{belongs: true}, &todoIssueRepositoryStub{}, time.Now())
	_, err := service.CreateTodo(context.Background(), contract.CreateTodoRequest{WorkspaceId: "workspace-1", Title: "Todo"})
	if !errors.Is(err, contract.ErrWorkspaceActorRequired) {
		t.Fatalf("missing actor error = %v", err)
	}

	repository := &todoRepositoryStub{}
	service = newTodoApplicationService(t, repository, &accessAuthorizerStub{}, &actorReaderStub{belongs: false}, &todoIssueRepositoryStub{}, time.Now())
	ctx := contract.WithWorkspaceActor(context.Background(), "member", "outsider")
	_, err = service.CreateTodo(ctx, contract.CreateTodoRequest{WorkspaceId: "workspace-1", Title: "Todo"})
	if !errors.Is(err, contract.ErrActorOutsideWorkspace) || repository.createCalls != 0 {
		t.Fatalf("outsider error/calls = %v/%d", err, repository.createCalls)
	}
}

func TestTodoUseCaseGetsListsAndHidesWorkspaceMisses(t *testing.T) {
	value := newTodoApplicationValue(t)
	repository := &todoRepositoryStub{value: value, values: []todoDomain.Todo{value}}
	authorizer := &accessAuthorizerStub{}
	service := newTodoApplicationService(t, repository, authorizer, &actorReaderStub{belongs: true}, &todoIssueRepositoryStub{}, time.Now())

	got, err := service.GetTodo(context.Background(), contract.GetTodoRequest{WorkspaceId: "workspace-1", TodoId: "todo-1"})
	if err != nil || got.Todo == nil || got.Todo.Id != "todo-1" {
		t.Fatalf("GetTodo() = %+v, %v", got.Todo, err)
	}
	projectID := "project-1"
	listed, err := service.ListTodos(context.Background(), contract.ListTodosRequest{WorkspaceId: "workspace-1", ProjectId: &projectID, Status: "todo"})
	if err != nil || listed.Total != 1 || len(listed.Todos) != 1 {
		t.Fatalf("ListTodos() = %+v, %v", listed, err)
	}
	if repository.query.WorkspaceID != "workspace-1" || repository.query.ProjectID == nil || *repository.query.ProjectID != "project-1" || repository.query.Status != "todo" || repository.query.Limit != 50 {
		t.Fatalf("list query = %+v", repository.query)
	}
	if _, err := service.ListTodos(context.Background(), contract.ListTodosRequest{WorkspaceId: "workspace-1", Limit: 250}); err != nil || repository.query.Limit != 100 {
		t.Fatalf("bounded list query/error = %+v/%v", repository.query, err)
	}
	if _, err := service.ListTodos(context.Background(), contract.ListTodosRequest{WorkspaceId: "workspace-1", Limit: -1}); !errors.Is(err, contract.ErrInvalidTodo) {
		t.Fatalf("negative limit error = %v", err)
	}
	if _, err := service.ListTodos(context.Background(), contract.ListTodosRequest{WorkspaceId: "workspace-1", Status: " todo "}); !errors.Is(err, contract.ErrInvalidTodo) {
		t.Fatalf("space-padded status error = %v", err)
	}
	repository.findErr = ErrTodoRecordNotFound
	_, err = service.GetTodo(context.Background(), contract.GetTodoRequest{WorkspaceId: "workspace-2", TodoId: "todo-1"})
	if !errors.Is(err, contract.ErrTodoNotFound) {
		t.Fatalf("foreign GetTodo() error = %v", err)
	}
	if len(authorizer.permissions) != 6 || authorizer.permissions[0] != contract.PermissionTaskRead || authorizer.permissions[1] != contract.PermissionTaskRead || authorizer.permissions[2] != contract.PermissionTaskRead || authorizer.permissions[3] != contract.PermissionTaskRead || authorizer.permissions[4] != contract.PermissionTaskRead || authorizer.permissions[5] != contract.PermissionTaskRead {
		t.Fatalf("permissions = %v", authorizer.permissions)
	}
}

func TestTodoUseCaseFullUpdateClearAndStatusCompatibility(t *testing.T) {
	now := time.Date(2026, 8, 3, 4, 0, 0, 0, time.UTC)
	repository := &todoRepositoryStub{value: newTodoApplicationValue(t)}
	authorizer := &accessAuthorizerStub{}
	actors := &actorReaderStub{belongs: true}
	service := newTodoApplicationService(t, repository, authorizer, actors, &todoIssueRepositoryStub{}, now)
	empty, title, status := "", " Updated ", "in_progress"
	ctx := contract.WithWorkspaceActor(context.Background(), "member", "member-1")
	response, err := service.UpdateTodo(ctx, contract.UpdateTodoRequest{
		WorkspaceId: "workspace-1", TodoId: "todo-1", Title: &title, Status: &status,
		ProjectId: &empty, AssigneeId: &empty, StartDate: &empty, ExpectedRevision: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.Todo == nil || response.Todo.Title != "Updated" || response.Todo.CompletedAt != nil || response.Todo.ProjectId != nil || response.Todo.AssigneeId != nil {
		t.Fatalf("UpdateTodo() = %+v", response.Todo)
	}
	status = "done"
	compatible, err := service.UpdateTodoStatus(ctx, contract.UpdateTodoStatusRequest{WorkspaceId: "workspace-1", TodoId: "todo-1", Status: status, ExpectedRevision: 2})
	if err != nil || compatible.Todo == nil || compatible.Todo.Status != status || compatible.Todo.CompletedAt == nil {
		t.Fatalf("UpdateTodoStatus() = %+v, %v", compatible.Todo, err)
	}
	if repository.updateCalls != 2 || len(authorizer.permissions) != 4 || authorizer.permissions[0] != contract.PermissionTaskRead || authorizer.permissions[1] != contract.PermissionTaskUpdateOwn || authorizer.permissions[2] != contract.PermissionTaskRead || authorizer.permissions[3] != contract.PermissionTaskUpdateOwn {
		t.Fatalf("update calls/permissions = %d/%v", repository.updateCalls, authorizer.permissions)
	}
}

func TestTodoUseCaseRejectsStaleRevisionBeforeUpdate(t *testing.T) {
	repository := &todoRepositoryStub{value: newTodoApplicationValue(t)}
	service := newTodoApplicationService(t, repository, &accessAuthorizerStub{}, &actorReaderStub{belongs: true}, &todoIssueRepositoryStub{}, time.Now())
	title := "stale"

	ctx := contract.WithWorkspaceActor(context.Background(), "member", "member-1")
	_, err := service.UpdateTodo(ctx, contract.UpdateTodoRequest{
		WorkspaceId: "workspace-1", TodoId: "todo-1", Title: &title, ExpectedRevision: 0,
	})
	var conflict contract.RevisionConflictError
	if !errors.As(err, &conflict) || conflict.CurrentRevision != 1 {
		t.Fatalf("stale update error = %v, want current revision 1", err)
	}
	if repository.updateCalls != 0 {
		t.Fatalf("stale update wrote repository %d times", repository.updateCalls)
	}
}

func TestTodoUseCaseReordersWorkspaceTasksWithRevisionChecks(t *testing.T) {
	value := newTodoApplicationValue(t)
	repository := &todoRepositoryStub{value: value}
	authorizer := &accessAuthorizerStub{}
	service := newTodoApplicationService(t, repository, authorizer, &actorReaderStub{belongs: true}, &todoIssueRepositoryStub{}, time.Now())
	ctx := contract.WithWorkspaceActor(context.Background(), "member", "member-1")

	result, err := service.ReorderTodos(ctx, contract.ReorderTodosRequest{
		WorkspaceId: "workspace-1", IdempotencyKey: "reorder-1",
		Items: []contract.ReorderTodoItem{{TodoId: "todo-1", Position: 20, ExpectedRevision: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if repository.reorderCalls != 1 || len(result.Todos) != 1 || result.Todos[0].Position != 20 || result.Todos[0].Revision != 2 {
		t.Fatalf("reorder result/calls = %+v/%d", result, repository.reorderCalls)
	}
	if len(authorizer.permissions) != 2 || authorizer.permissions[0] != contract.PermissionTaskRead || authorizer.permissions[1] != contract.PermissionTaskUpdateOwn {
		t.Fatalf("reorder permissions = %v", authorizer.permissions)
	}
	if repository.reorderCommand.IdempotencyKey != "reorder-1" || len(repository.reorderCommand.RequestFingerprint) != 64 {
		t.Fatalf("reorder governance command = %+v", repository.reorderCommand)
	}
	if _, err := service.ReorderTodos(ctx, contract.ReorderTodosRequest{WorkspaceId: "workspace-1", Items: []contract.ReorderTodoItem{{TodoId: "todo-1", Position: 20, ExpectedRevision: 1}}}); !errors.Is(err, contract.ErrInvalidTodo) {
		t.Fatalf("missing idempotency key error = %v", err)
	}
}

func TestTodoUseCaseArchiveMapsWorkspaceScopedMiss(t *testing.T) {
	repository := &todoRepositoryStub{findErr: ErrTodoRecordNotFound}
	service := newTodoApplicationService(t, repository, &accessAuthorizerStub{}, &actorReaderStub{belongs: true}, &todoIssueRepositoryStub{}, time.Now())
	ctx := contract.WithWorkspaceActor(context.Background(), "member", "member-1")
	_, err := service.DeleteTodo(ctx, contract.DeleteTodoRequest{WorkspaceId: "workspace-1", TodoId: "todo-1", ExpectedRevision: 1})
	if !errors.Is(err, contract.ErrTodoNotFound) || repository.findCalls != 1 || repository.updateCalls != 0 {
		t.Fatalf("DeleteTodo() error/find/update calls = %v/%d/%d", err, repository.findCalls, repository.updateCalls)
	}
}

func TestTodoUseCaseArchivesAndRestoresTerminalTask(t *testing.T) {
	now := time.Date(2026, 8, 18, 3, 0, 0, 0, time.UTC)
	value, err := todoDomain.New(
		"todo-1", "workspace-1", "Todo", "", todoDomain.StatusCancelled, todoDomain.PriorityNone,
		nil, nil, nil, nil, "member", "member-1", 0, nil, nil, now,
	)
	if err != nil {
		t.Fatal(err)
	}
	repository := &todoRepositoryStub{value: value}
	service := newTodoApplicationService(t, repository, &accessAuthorizerStub{}, &actorReaderStub{belongs: true}, &todoIssueRepositoryStub{}, now.Add(time.Minute))

	ctx := contract.WithWorkspaceActor(context.Background(), "member", "member-1")
	if _, err := service.DeleteTodo(ctx, contract.DeleteTodoRequest{
		WorkspaceId: "workspace-1", TodoId: "todo-1", ExpectedRevision: 1,
	}); err != nil {
		t.Fatal(err)
	}
	if repository.value.Status != todoDomain.StatusArchived || repository.value.Revision != 2 {
		t.Fatalf("archived repository value = %+v", repository.value)
	}

	restored, err := service.RestoreTodo(ctx, contract.RestoreTodoRequest{
		WorkspaceId: "workspace-1", TodoId: "todo-1", ExpectedRevision: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if restored.Todo == nil || restored.Todo.Status != todoDomain.StatusCancelled || restored.Todo.Revision != 3 {
		t.Fatalf("restored task = %+v", restored.Todo)
	}
}

func TestTodoUseCaseAuthorizesEveryOperationBeforePersistence(t *testing.T) {
	denied := errors.New("denied")
	repository := &todoRepositoryStub{}
	service := newTodoApplicationService(t, repository, &accessAuthorizerStub{err: denied}, &actorReaderStub{belongs: true}, &todoIssueRepositoryStub{}, time.Now())
	ctx := contract.WithWorkspaceActor(context.Background(), "member", "member-1")
	operations := []func() error{
		func() error {
			_, err := service.CreateTodo(ctx, contract.CreateTodoRequest{WorkspaceId: "workspace-1", Title: "Todo"})
			return err
		},
		func() error {
			_, err := service.GetTodo(ctx, contract.GetTodoRequest{WorkspaceId: "workspace-1", TodoId: "todo-1"})
			return err
		},
		func() error {
			_, err := service.ListTodos(ctx, contract.ListTodosRequest{WorkspaceId: "workspace-1"})
			return err
		},
		func() error {
			_, err := service.UpdateTodo(ctx, contract.UpdateTodoRequest{WorkspaceId: "workspace-1", TodoId: "todo-1"})
			return err
		},
		func() error {
			_, err := service.UpdateTodoStatus(ctx, contract.UpdateTodoStatusRequest{WorkspaceId: "workspace-1", TodoId: "todo-1", Status: "done"})
			return err
		},
		func() error {
			_, err := service.DeleteTodo(ctx, contract.DeleteTodoRequest{WorkspaceId: "workspace-1", TodoId: "todo-1"})
			return err
		},
		func() error {
			_, err := service.RestoreTodo(ctx, contract.RestoreTodoRequest{WorkspaceId: "workspace-1", TodoId: "todo-1"})
			return err
		},
	}
	for index, operation := range operations {
		if err := operation(); !errors.Is(err, denied) {
			t.Fatalf("operation %d error = %v", index, err)
		}
	}
	if repository.createCalls+repository.findCalls+repository.listCalls+repository.updateCalls != 0 {
		t.Fatalf("repository accessed before authorization: %+v", repository)
	}
}
