package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	issueDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/issue"
	todoDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/todo"
)

const TaskActionPromote = "workspace.task.promote"

type TaskPromotionCommand struct {
	Task             todoDomain.Todo
	Issue            issueDomain.Issue
	ExpectedRevision int64
	CompleteTask     bool
	IdempotencyKey   string
	OccurredAt       time.Time
}

type TaskPromotionResult struct {
	Task  todoDomain.Todo
	Issue issueDomain.Issue
}

type TaskPromotionReader interface {
	FindTaskForPromotion(context.Context, string, string) (todoDomain.Todo, error)
}

type TaskPromotionRepository interface {
	PromoteTask(context.Context, TaskPromotionCommand) (TaskPromotionResult, error)
}

type TaskPromotionUseCase struct {
	reader     TaskPromotionReader
	repository TaskPromotionRepository
	authorizer contract.WorkspaceAccessAuthorizer
	actors     contract.WorkspaceActorReader
	newIssueID ProjectIDGenerator
	now        Clock
}

func NewTaskPromotionUseCase(reader TaskPromotionReader, repository TaskPromotionRepository, authorizer contract.WorkspaceAccessAuthorizer, actors contract.WorkspaceActorReader, newIssueID ProjectIDGenerator, now Clock) (*TaskPromotionUseCase, error) {
	if reader == nil || repository == nil || authorizer == nil || actors == nil || newIssueID == nil || now == nil {
		return nil, errors.New("Task promotion dependencies are required")
	}
	return &TaskPromotionUseCase{reader: reader, repository: repository, authorizer: authorizer, actors: actors, newIssueID: newIssueID, now: now}, nil
}

func (s *TaskPromotionUseCase) PromoteTask(ctx context.Context, request contract.PromoteTaskRequest) (contract.PromoteTaskResponse, error) {
	workspaceID := strings.TrimSpace(request.WorkspaceId)
	taskID := strings.TrimSpace(request.TaskId)
	idempotencyKey := strings.TrimSpace(request.IdempotencyKey)
	if workspaceID == "" || taskID == "" || request.ExpectedRevision < 1 || idempotencyKey == "" {
		return contract.PromoteTaskResponse{}, contract.ErrInvalidTaskPromotion
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, workspaceID, contract.PermissionTaskRead); err != nil {
		return contract.PromoteTaskResponse{}, err
	}
	task, err := s.reader.FindTaskForPromotion(ctx, workspaceID, taskID)
	if errors.Is(err, ErrTodoRecordNotFound) {
		return contract.PromoteTaskResponse{}, contract.ErrTodoNotFound
	}
	if err != nil {
		return contract.PromoteTaskResponse{}, fmt.Errorf("find Task for promotion: %w", err)
	}
	actor, ok := contract.WorkspaceActorFromContext(ctx)
	if !ok {
		return contract.PromoteTaskResponse{}, contract.ErrWorkspaceActorRequired
	}
	belongs, err := s.actors.ActorBelongsToWorkspace(ctx, workspaceID, actor.Type, actor.ID)
	if err != nil {
		return contract.PromoteTaskResponse{}, fmt.Errorf("validate Task promotion actor: %w", err)
	}
	if !belongs {
		return contract.PromoteTaskResponse{}, contract.ErrActorOutsideWorkspace
	}
	permission := contract.PermissionTaskManageWorkspace
	if actor.Type == task.CreatorType && actor.ID == task.CreatorID {
		permission = contract.PermissionTaskUpdateOwn
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, workspaceID, permission); err != nil {
		return contract.PromoteTaskResponse{}, err
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, workspaceID, PermissionIssueCreate); err != nil {
		return contract.PromoteTaskResponse{}, err
	}
	issueID, err := s.newIssueID(ctx)
	if err != nil {
		return contract.PromoteTaskResponse{}, fmt.Errorf("generate promoted Issue id: %w", err)
	}
	occurredAt := s.now().UTC()
	description := task.Description
	issue, err := issueDomain.New(
		issueID, workspaceID, task.Title, &description, issueDomain.StatusTodo, task.Priority,
		task.AssigneeType, task.AssigneeID, nil, task.ProjectID, actor.Type, actor.ID,
		task.Position, nil, formatPromotionDate(task.StartDate), formatPromotionDate(task.DueDate), nil, occurredAt,
	)
	if err != nil {
		return contract.PromoteTaskResponse{}, fmt.Errorf("%w: %v", contract.ErrInvalidTaskPromotion, err)
	}
	result, err := s.repository.PromoteTask(ctx, TaskPromotionCommand{
		Task: task, Issue: issue, ExpectedRevision: request.ExpectedRevision,
		CompleteTask: request.CompleteTask, IdempotencyKey: idempotencyKey, OccurredAt: occurredAt,
	})
	if err != nil {
		return contract.PromoteTaskResponse{}, err
	}
	taskResult := todoToContract(result.Task)
	issueResult := issueToContract(result.Issue)
	return contract.PromoteTaskResponse{Task: &taskResult, Issue: &issueResult, SourceTaskId: task.ID}, nil
}

func formatPromotionDate(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339Nano)
	return &formatted
}

var _ contract.TaskPromotionService = (*TaskPromotionUseCase)(nil)
