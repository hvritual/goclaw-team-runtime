package application

import (
	"context"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	issueDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/issue"
	todoDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/todo"
)

func TestTaskPromotionUseCaseAuthorizesAndBuildsSnapshotCommand(t *testing.T) {
	now := time.Date(2026, 8, 18, 8, 0, 0, 0, time.UTC)
	task, err := todoDomain.Rehydrate(todoDomain.Todo{
		ID: "task-1", WorkspaceID: "workspace-1", Title: "Ship Release 1", Description: "Promotion snapshot",
		Status: todoDomain.StatusInProgress, Priority: todoDomain.PriorityHigh,
		CreatorType: "member", CreatorID: "member-1", Revision: 4, CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	issue, err := issueDomain.Rehydrate(issueDomain.Issue{
		ID: "issue-1", WorkspaceID: "workspace-1", Number: 7, Identifier: "ONE-7", Title: task.Title,
		Description: &task.Description, Status: issueDomain.StatusTodo, Priority: issueDomain.PriorityHigh,
		CreatorType: "member", CreatorID: "member-1", CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	repository := &taskPromotionRepositoryStub{task: task, result: TaskPromotionResult{Task: task, Issue: issue}}
	authorizer := &accessAuthorizerStub{}
	service, err := NewTaskPromotionUseCase(repository, repository, authorizer, &actorReaderStub{belongs: true}, func(context.Context) (string, error) {
		return "issue-1", nil
	}, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	ctx := contract.WithWorkspaceActor(context.Background(), "member", "member-1")
	response, err := service.PromoteTask(ctx, contract.PromoteTaskRequest{
		WorkspaceId: "workspace-1", TaskId: "task-1", ExpectedRevision: 4,
		CompleteTask: true, IdempotencyKey: "promote-task-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.Task == nil || response.Issue == nil || response.SourceTaskId != "task-1" {
		t.Fatalf("promotion response = %+v", response)
	}
	if repository.command.Task.ID != "task-1" || repository.command.Issue.ID != "issue-1" || !repository.command.CompleteTask || repository.command.ExpectedRevision != 4 || repository.command.IdempotencyKey != "promote-task-1" {
		t.Fatalf("promotion command = %+v", repository.command)
	}
	wantPermissions := []string{contract.PermissionTaskRead, contract.PermissionTaskUpdateOwn, PermissionIssueCreate}
	if len(authorizer.permissions) != len(wantPermissions) {
		t.Fatalf("permissions = %v", authorizer.permissions)
	}
	for index := range wantPermissions {
		if authorizer.permissions[index] != wantPermissions[index] {
			t.Fatalf("permissions = %v", authorizer.permissions)
		}
	}
}

type taskPromotionRepositoryStub struct {
	task    todoDomain.Todo
	command TaskPromotionCommand
	result  TaskPromotionResult
}

func (s *taskPromotionRepositoryStub) FindTaskForPromotion(context.Context, string, string) (todoDomain.Todo, error) {
	return s.task, nil
}

func (s *taskPromotionRepositoryStub) PromoteTask(_ context.Context, command TaskPromotionCommand) (TaskPromotionResult, error) {
	s.command = command
	return s.result, nil
}
