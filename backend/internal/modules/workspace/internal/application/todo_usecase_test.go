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
	value       todoDomain.Todo
	values      []todoDomain.Todo
	query       TodoListQuery
	findErr     error
	createErr   error
	updateErr   error
	deleteErr   error
	createCalls int
	findCalls   int
	listCalls   int
	updateCalls int
	deleteCalls int
}

func (s *todoRepositoryStub) Create(_ context.Context, value todoDomain.Todo) error {
	s.createCalls++
	s.value = value
	return s.createErr
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

func (s *todoRepositoryStub) Delete(context.Context, string, string) error {
	s.deleteCalls++
	return s.deleteErr
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
	if repository.query.WorkspaceID != "workspace-1" || repository.query.ProjectID == nil || *repository.query.ProjectID != "project-1" || repository.query.Status != "todo" {
		t.Fatalf("list query = %+v", repository.query)
	}
	if _, err := service.ListTodos(context.Background(), contract.ListTodosRequest{WorkspaceId: "workspace-1", Status: " todo "}); !errors.Is(err, contract.ErrInvalidTodo) {
		t.Fatalf("space-padded status error = %v", err)
	}
	repository.findErr = ErrTodoRecordNotFound
	_, err = service.GetTodo(context.Background(), contract.GetTodoRequest{WorkspaceId: "workspace-2", TodoId: "todo-1"})
	if !errors.Is(err, contract.ErrTodoNotFound) {
		t.Fatalf("foreign GetTodo() error = %v", err)
	}
	if len(authorizer.permissions) != 4 || authorizer.permissions[0] != PermissionTodoGet || authorizer.permissions[1] != PermissionTodoList || authorizer.permissions[2] != PermissionTodoList || authorizer.permissions[3] != PermissionTodoGet {
		t.Fatalf("permissions = %v", authorizer.permissions)
	}
}

func TestTodoUseCaseFullUpdateClearAndStatusCompatibility(t *testing.T) {
	now := time.Date(2026, 8, 3, 4, 0, 0, 0, time.UTC)
	repository := &todoRepositoryStub{value: newTodoApplicationValue(t)}
	authorizer := &accessAuthorizerStub{}
	actors := &actorReaderStub{belongs: true}
	service := newTodoApplicationService(t, repository, authorizer, actors, &todoIssueRepositoryStub{}, now)
	empty, title, status := "", " Updated ", "done"
	response, err := service.UpdateTodo(context.Background(), contract.UpdateTodoRequest{
		WorkspaceId: "workspace-1", TodoId: "todo-1", Title: &title, Status: &status,
		ProjectId: &empty, AssigneeId: &empty, StartDate: &empty,
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.Todo == nil || response.Todo.Title != "Updated" || response.Todo.CompletedAt == nil || response.Todo.ProjectId != nil || response.Todo.AssigneeId != nil {
		t.Fatalf("UpdateTodo() = %+v", response.Todo)
	}
	status = "in_progress"
	compatible, err := service.UpdateTodoStatus(context.Background(), contract.UpdateTodoStatusRequest{WorkspaceId: "workspace-1", TodoId: "todo-1", Status: status})
	if err != nil || compatible.Todo == nil || compatible.Todo.Status != status || compatible.Todo.CompletedAt != nil {
		t.Fatalf("UpdateTodoStatus() = %+v, %v", compatible.Todo, err)
	}
	if repository.updateCalls != 2 || authorizer.permissions[0] != PermissionTodoUpdate || authorizer.permissions[1] != PermissionTodoUpdateStatus {
		t.Fatalf("update calls/permissions = %d/%v", repository.updateCalls, authorizer.permissions)
	}
}

func TestTodoUseCaseDeleteMapsWorkspaceScopedMiss(t *testing.T) {
	repository := &todoRepositoryStub{deleteErr: ErrTodoRecordNotFound}
	service := newTodoApplicationService(t, repository, &accessAuthorizerStub{}, &actorReaderStub{belongs: true}, &todoIssueRepositoryStub{}, time.Now())
	_, err := service.DeleteTodo(context.Background(), contract.DeleteTodoRequest{WorkspaceId: "workspace-1", TodoId: "todo-1"})
	if !errors.Is(err, contract.ErrTodoNotFound) || repository.deleteCalls != 1 {
		t.Fatalf("DeleteTodo() error/calls = %v/%d", err, repository.deleteCalls)
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
	}
	for index, operation := range operations {
		if err := operation(); !errors.Is(err, denied) {
			t.Fatalf("operation %d error = %v", index, err)
		}
	}
	if repository.createCalls+repository.findCalls+repository.listCalls+repository.updateCalls+repository.deleteCalls != 0 {
		t.Fatalf("repository accessed before authorization: %+v", repository)
	}
}
