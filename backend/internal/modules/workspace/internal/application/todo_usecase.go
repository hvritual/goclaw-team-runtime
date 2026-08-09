package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	todoDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/todo"
)

const (
	PermissionTodoCreate       = "workspace.todo.create"
	PermissionTodoGet          = "workspace.todo.get"
	PermissionTodoList         = "workspace.todo.list"
	PermissionTodoUpdate       = "workspace.todo.update"
	PermissionTodoUpdateStatus = "workspace.todo.update_status"
	PermissionTodoDelete       = "workspace.todo.delete"
)

var ErrTodoRecordNotFound = errors.New("todo record not found")

type TodoListQuery struct {
	WorkspaceID string
	ProjectID   *string
	IssueID     *string
	Status      string
}

type TodoRepository interface {
	Create(context.Context, todoDomain.Todo) error
	FindByID(context.Context, string, string) (todoDomain.Todo, error)
	List(context.Context, TodoListQuery) ([]todoDomain.Todo, error)
	Update(context.Context, todoDomain.Todo) error
	Delete(context.Context, string, string) error
}

type TodoUseCase struct {
	repository TodoRepository
	projects   ProjectRepository
	issues     IssueReferenceRepository
	authorizer contract.WorkspaceAccessAuthorizer
	actors     contract.WorkspaceActorReader
	newID      ProjectIDGenerator
	now        Clock
}

func NewTodoUseCase(repository TodoRepository, projects ProjectRepository, issues IssueReferenceRepository, authorizer contract.WorkspaceAccessAuthorizer, actors contract.WorkspaceActorReader, newID ProjectIDGenerator, now Clock) (*TodoUseCase, error) {
	if repository == nil || projects == nil || issues == nil || authorizer == nil || actors == nil || newID == nil || now == nil {
		return nil, errors.New("Todo dependencies are required")
	}
	return &TodoUseCase{repository: repository, projects: projects, issues: issues, authorizer: authorizer, actors: actors, newID: newID, now: now}, nil
}

func (s *TodoUseCase) CreateTodo(ctx context.Context, request contract.CreateTodoRequest) (contract.CreateTodoResponse, error) {
	workspaceID := strings.TrimSpace(request.WorkspaceId)
	if workspaceID == "" {
		return contract.CreateTodoResponse{}, fmt.Errorf("%w: workspace id is required", contract.ErrInvalidTodo)
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, workspaceID, PermissionTodoCreate); err != nil {
		return contract.CreateTodoResponse{}, err
	}
	actor, ok := contract.WorkspaceActorFromContext(ctx)
	if !ok {
		return contract.CreateTodoResponse{}, contract.ErrWorkspaceActorRequired
	}
	if err := s.requireWorkspaceActor(ctx, workspaceID, actor.Type, actor.ID); err != nil {
		return contract.CreateTodoResponse{}, err
	}
	projectID := cleanOptionalString(request.ProjectId)
	if err := s.validateProject(ctx, workspaceID, projectID); err != nil {
		return contract.CreateTodoResponse{}, err
	}
	issueID, err := s.canonicalIssueID(ctx, workspaceID, cleanOptionalString(request.IssueId))
	if err != nil {
		return contract.CreateTodoResponse{}, err
	}
	startDate, err := parseTodoTime(request.StartDate)
	if err != nil {
		return contract.CreateTodoResponse{}, fmt.Errorf("%w: invalid start date", contract.ErrInvalidTodo)
	}
	dueDate, err := parseTodoTime(request.DueDate)
	if err != nil {
		return contract.CreateTodoResponse{}, fmt.Errorf("%w: invalid due date", contract.ErrInvalidTodo)
	}
	id, err := s.newID(ctx)
	if err != nil {
		return contract.CreateTodoResponse{}, fmt.Errorf("generate Todo id: %w", err)
	}
	value, err := todoDomain.New(
		id, workspaceID, request.Title, request.Description, request.Status, request.Priority,
		projectID, issueID, request.AssigneeType, request.AssigneeId,
		actor.Type, actor.ID, request.Position, startDate, dueDate, s.now(),
	)
	if err != nil {
		return contract.CreateTodoResponse{}, fmt.Errorf("%w: %v", contract.ErrInvalidTodo, err)
	}
	if value.AssigneeID != nil {
		if err := s.requireWorkspaceActor(ctx, workspaceID, *value.AssigneeType, *value.AssigneeID); err != nil {
			return contract.CreateTodoResponse{}, err
		}
	}
	if err := s.repository.Create(ctx, value); err != nil {
		return contract.CreateTodoResponse{}, fmt.Errorf("create Todo: %w", err)
	}
	result := todoToContract(value)
	return contract.CreateTodoResponse{Todo: &result}, nil
}

func (s *TodoUseCase) GetTodo(ctx context.Context, request contract.GetTodoRequest) (contract.GetTodoResponse, error) {
	workspaceID, todoID, err := validateTodoIdentity(request.WorkspaceId, request.TodoId)
	if err != nil {
		return contract.GetTodoResponse{}, err
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, workspaceID, PermissionTodoGet); err != nil {
		return contract.GetTodoResponse{}, err
	}
	value, err := s.findTodo(ctx, workspaceID, todoID)
	if err != nil {
		return contract.GetTodoResponse{}, err
	}
	result := todoToContract(value)
	return contract.GetTodoResponse{Todo: &result}, nil
}

func (s *TodoUseCase) ListTodos(ctx context.Context, request contract.ListTodosRequest) (contract.ListTodosResponse, error) {
	workspaceID := strings.TrimSpace(request.WorkspaceId)
	if workspaceID == "" {
		return contract.ListTodosResponse{}, fmt.Errorf("%w: workspace id is required", contract.ErrInvalidTodo)
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, workspaceID, PermissionTodoList); err != nil {
		return contract.ListTodosResponse{}, err
	}
	status := request.Status
	if status != "" && !todoDomain.ValidStatus(status) {
		return contract.ListTodosResponse{}, fmt.Errorf("%w: invalid status", contract.ErrInvalidTodo)
	}
	values, err := s.repository.List(ctx, TodoListQuery{
		WorkspaceID: workspaceID,
		ProjectID:   cleanOptionalString(request.ProjectId),
		IssueID:     cleanOptionalString(request.IssueId),
		Status:      status,
	})
	if err != nil {
		return contract.ListTodosResponse{}, fmt.Errorf("list Todos: %w", err)
	}
	result := make([]contract.Todo, len(values))
	for index, value := range values {
		result[index] = todoToContract(value)
	}
	return contract.ListTodosResponse{Todos: result, Total: int32(len(result))}, nil
}

func (s *TodoUseCase) UpdateTodo(ctx context.Context, request contract.UpdateTodoRequest) (contract.UpdateTodoResponse, error) {
	return s.updateTodo(ctx, request, PermissionTodoUpdate)
}

func (s *TodoUseCase) UpdateTodoStatus(ctx context.Context, request contract.UpdateTodoStatusRequest) (contract.UpdateTodoStatusResponse, error) {
	status := request.Status
	updated, err := s.updateTodo(ctx, contract.UpdateTodoRequest{
		WorkspaceId: request.WorkspaceId,
		TodoId:      request.TodoId,
		Status:      &status,
	}, PermissionTodoUpdateStatus)
	if err != nil {
		return contract.UpdateTodoStatusResponse{}, err
	}
	return contract.UpdateTodoStatusResponse{Todo: updated.Todo}, nil
}

func (s *TodoUseCase) DeleteTodo(ctx context.Context, request contract.DeleteTodoRequest) (contract.DeleteTodoResponse, error) {
	workspaceID, todoID, err := validateTodoIdentity(request.WorkspaceId, request.TodoId)
	if err != nil {
		return contract.DeleteTodoResponse{}, err
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, workspaceID, PermissionTodoDelete); err != nil {
		return contract.DeleteTodoResponse{}, err
	}
	if err := s.repository.Delete(ctx, workspaceID, todoID); errors.Is(err, ErrTodoRecordNotFound) {
		return contract.DeleteTodoResponse{}, contract.ErrTodoNotFound
	} else if err != nil {
		return contract.DeleteTodoResponse{}, fmt.Errorf("delete Todo: %w", err)
	}
	return contract.DeleteTodoResponse{}, nil
}

func (s *TodoUseCase) updateTodo(ctx context.Context, request contract.UpdateTodoRequest, permission string) (contract.UpdateTodoResponse, error) {
	workspaceID, todoID, err := validateTodoIdentity(request.WorkspaceId, request.TodoId)
	if err != nil {
		return contract.UpdateTodoResponse{}, err
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, workspaceID, permission); err != nil {
		return contract.UpdateTodoResponse{}, err
	}
	value, err := s.findTodo(ctx, workspaceID, todoID)
	if err != nil {
		return contract.UpdateTodoResponse{}, err
	}
	patch := todoDomain.Patch{
		Title: request.Title, Description: request.Description, Status: request.Status,
		Priority: request.Priority, Position: request.Position,
		ProjectID: todoStringChange(request.ProjectId), IssueID: todoStringChange(request.IssueId),
		AssigneeType: todoStringChange(request.AssigneeType), AssigneeID: todoStringChange(request.AssigneeId),
	}
	if request.ProjectId != nil {
		if err := s.validateProject(ctx, workspaceID, cleanOptionalString(request.ProjectId)); err != nil {
			return contract.UpdateTodoResponse{}, err
		}
	}
	if request.IssueId != nil {
		canonicalID, canonicalErr := s.canonicalIssueID(ctx, workspaceID, cleanOptionalString(request.IssueId))
		if canonicalErr != nil {
			return contract.UpdateTodoResponse{}, canonicalErr
		}
		patch.IssueID.Value = canonicalID
	}
	patch.StartDate, err = parseTodoTimeChange(request.StartDate)
	if err != nil {
		return contract.UpdateTodoResponse{}, fmt.Errorf("%w: invalid start date", contract.ErrInvalidTodo)
	}
	patch.DueDate, err = parseTodoTimeChange(request.DueDate)
	if err != nil {
		return contract.UpdateTodoResponse{}, fmt.Errorf("%w: invalid due date", contract.ErrInvalidTodo)
	}
	updated, err := value.Apply(patch, s.now())
	if err != nil {
		return contract.UpdateTodoResponse{}, fmt.Errorf("%w: %v", contract.ErrInvalidTodo, err)
	}
	if (request.AssigneeType != nil || request.AssigneeId != nil) && updated.AssigneeID != nil {
		if err := s.requireWorkspaceActor(ctx, workspaceID, *updated.AssigneeType, *updated.AssigneeID); err != nil {
			return contract.UpdateTodoResponse{}, err
		}
	}
	if err := s.repository.Update(ctx, updated); errors.Is(err, ErrTodoRecordNotFound) {
		return contract.UpdateTodoResponse{}, contract.ErrTodoNotFound
	} else if err != nil {
		return contract.UpdateTodoResponse{}, fmt.Errorf("update Todo: %w", err)
	}
	result := todoToContract(updated)
	return contract.UpdateTodoResponse{Todo: &result}, nil
}

func (s *TodoUseCase) findTodo(ctx context.Context, workspaceID, todoID string) (todoDomain.Todo, error) {
	value, err := s.repository.FindByID(ctx, workspaceID, todoID)
	if errors.Is(err, ErrTodoRecordNotFound) {
		return todoDomain.Todo{}, contract.ErrTodoNotFound
	}
	if err != nil {
		return todoDomain.Todo{}, fmt.Errorf("get Todo: %w", err)
	}
	return value, nil
}

func (s *TodoUseCase) validateProject(ctx context.Context, workspaceID string, projectID *string) error {
	if projectID == nil {
		return nil
	}
	if _, err := s.projects.FindByID(ctx, workspaceID, *projectID); errors.Is(err, ErrProjectRecordNotFound) {
		return contract.ErrProjectNotFound
	} else if err != nil {
		return fmt.Errorf("validate Todo Project: %w", err)
	}
	return nil
}

func (s *TodoUseCase) canonicalIssueID(ctx context.Context, workspaceID string, issueID *string) (*string, error) {
	if issueID == nil {
		return nil, nil
	}
	issue, err := s.issues.FindByIDOrIdentifier(ctx, workspaceID, *issueID)
	if errors.Is(err, ErrIssueRecordNotFound) {
		return nil, contract.ErrIssueNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("validate Todo Issue: %w", err)
	}
	canonical := issue.ID
	return &canonical, nil
}

func (s *TodoUseCase) requireWorkspaceActor(ctx context.Context, workspaceID, actorType, actorID string) error {
	belongs, err := s.actors.ActorBelongsToWorkspace(ctx, workspaceID, actorType, actorID)
	if err != nil {
		return fmt.Errorf("validate Todo actor: %w", err)
	}
	if !belongs {
		return contract.ErrActorOutsideWorkspace
	}
	return nil
}

func validateTodoIdentity(workspaceID, todoID string) (string, string, error) {
	workspaceID, todoID = strings.TrimSpace(workspaceID), strings.TrimSpace(todoID)
	if workspaceID == "" || todoID == "" {
		return "", "", fmt.Errorf("%w: workspace id and Todo id are required", contract.ErrInvalidTodo)
	}
	return workspaceID, todoID, nil
}

func todoStringChange(value *string) todoDomain.StringChange {
	if value == nil {
		return todoDomain.StringChange{}
	}
	return todoDomain.StringChange{Set: true, Value: cleanOptionalString(value)}
}

func cleanOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func parseTodoTime(value *string) (*time.Time, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, *value)
	if err != nil {
		return nil, err
	}
	parsed = parsed.UTC()
	return &parsed, nil
}

func parseTodoTimeChange(value *string) (todoDomain.TimeChange, error) {
	if value == nil {
		return todoDomain.TimeChange{}, nil
	}
	parsed, err := parseTodoTime(value)
	if err != nil {
		return todoDomain.TimeChange{}, err
	}
	return todoDomain.TimeChange{Set: true, Value: parsed}, nil
}

func todoToContract(value todoDomain.Todo) contract.Todo {
	return contract.Todo{
		Id: value.ID, WorkspaceId: value.WorkspaceID, Title: value.Title,
		Description: value.Description, Status: value.Status, Priority: value.Priority,
		ProjectId: copyTodoString(value.ProjectID), IssueId: copyTodoString(value.IssueID),
		AssigneeType: copyTodoString(value.AssigneeType), AssigneeId: copyTodoString(value.AssigneeID),
		CreatorType: value.CreatorType, CreatorId: value.CreatorID, Position: value.Position,
		StartDate: formatTodoTime(value.StartDate), DueDate: formatTodoTime(value.DueDate),
		CompletedAt: formatTodoTime(value.CompletedAt),
		CreatedAt:   value.CreatedAt.Format(time.RFC3339Nano), UpdatedAt: value.UpdatedAt.Format(time.RFC3339Nano),
	}
}

func copyTodoString(value *string) *string {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func formatTodoTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339Nano)
	return &formatted
}

var _ contract.TodoService = (*TodoUseCase)(nil)
