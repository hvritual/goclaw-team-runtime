package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
	issueDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/issue"
	todoDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/todo"
)

type taskPromotionRepository struct {
	db                *sql.DB
	governance        *GovernanceRepository
	governanceService application.GovernanceService
}

func NewTaskPromotionRepository(config Config) (*taskPromotionRepository, error) {
	if config.DB == nil {
		return nil, errors.New("workspace sqlite database is required")
	}
	provider := application.TaskGovernancePolicyProvider{}
	governance, err := NewGovernanceRepository(config, WithGovernanceEventPolicies(provider))
	if err != nil {
		return nil, err
	}
	return &taskPromotionRepository{db: config.DB, governance: governance, governanceService: application.NewGovernanceService(provider)}, nil
}

func (r *taskPromotionRepository) FindTaskForPromotion(ctx context.Context, workspaceID, taskID string) (todoDomain.Todo, error) {
	return scanTodo(r.db.QueryRowContext(ctx, todoSelect+` WHERE workspace_id=? AND id=?`, workspaceID, taskID))
}

func (r *taskPromotionRepository) PromoteTask(ctx context.Context, command application.TaskPromotionCommand) (application.TaskPromotionResult, error) {
	workspaceID := strings.TrimSpace(command.Task.WorkspaceID)
	taskID := strings.TrimSpace(command.Task.ID)
	issueID := strings.TrimSpace(command.Issue.ID)
	idempotencyKey := strings.TrimSpace(command.IdempotencyKey)
	if workspaceID == "" || taskID == "" || issueID == "" || command.Issue.WorkspaceID != workspaceID || command.ExpectedRevision < 1 || idempotencyKey == "" || command.OccurredAt.IsZero() {
		return application.TaskPromotionResult{}, contract.ErrInvalidTaskPromotion
	}
	status := command.Task.Status
	if command.CompleteTask {
		status = todoDomain.StatusDone
	}
	responseFields := map[string]any{
		"id": taskID, "issue_id": issueID, "revision": command.ExpectedRevision + 1,
		"status": status, "complete_task": command.CompleteTask,
	}
	prepared, err := r.governanceService.PrepareContext(ctx, application.GovernanceRequest{
		Identity: contract.MutationIdentity{WorkspaceID: workspaceID, RequestID: uuid.NewString()},
		Command: contract.MutationCommand{
			Action: application.TaskActionPromote, ResourceKind: "task", ResourceID: taskID,
			ExpectedRevision: command.ExpectedRevision, IdempotencyKey: idempotencyKey,
		},
		RequestFields: map[string]any{
			"expected_revision": command.ExpectedRevision, "complete_task": command.CompleteTask,
		},
		ResponseStatus: 201,
		ResponseFields: responseFields,
		AuditID:        uuid.NewString(),
		OccurredAt:     command.OccurredAt.UTC(),
		AuditFields:    responseFields,
		Outbox: []application.OutboxDraft{{
			ID: uuid.NewString(), EventType: "task:updated",
			Fields: map[string]any{"id": taskID, "revision": command.ExpectedRevision + 1, "status": status},
		}},
	})
	if err != nil {
		return application.TaskPromotionResult{}, err
	}
	var promoted application.TaskPromotionResult
	mutation, err := r.governance.Execute(ctx, prepared, func(ctx context.Context, connection *sql.Conn) error {
		current, err := scanTodo(connection.QueryRowContext(ctx, todoSelect+` WHERE workspace_id=? AND id=?`, workspaceID, taskID))
		if errors.Is(err, application.ErrTodoRecordNotFound) {
			return contract.ErrTodoNotFound
		}
		if err != nil {
			return fmt.Errorf("load Task for promotion: %w", err)
		}
		if current.Revision != command.ExpectedRevision {
			return contract.RevisionConflictError{CurrentRevision: current.Revision}
		}
		var existingIssueID string
		err = connection.QueryRowContext(ctx, `SELECT issue_id FROM workspace_task_issue_promotions WHERE workspace_id=? AND task_id=?`, workspaceID, taskID).Scan(&existingIssueID)
		if err == nil || current.IssueID != nil {
			return contract.ErrTaskAlreadyLinked
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("inspect Task promotion link: %w", err)
		}
		patch := todoDomain.Patch{IssueID: todoDomain.StringChange{Set: true, Value: &issueID}}
		if command.CompleteTask {
			if current.Status != todoDomain.StatusInProgress {
				return contract.ErrTaskPromotionConflict
			}
			completed := todoDomain.StatusDone
			patch.Status = &completed
		}
		updated, err := current.Apply(patch, command.OccurredAt)
		if err != nil {
			return contract.ErrInvalidTaskPromotion
		}
		created, err := r.insertPromotedIssue(ctx, connection, command.Issue)
		if err != nil {
			return err
		}
		if _, err := connection.ExecContext(ctx, `INSERT INTO workspace_task_issue_promotions(workspace_id,task_id,issue_id,created_at) VALUES(?,?,?,?)`, workspaceID, taskID, created.ID, command.OccurredAt.UTC().Format(time.RFC3339Nano)); err != nil {
			return fmt.Errorf("insert immutable Task promotion link: %w", err)
		}
		if err := (&todoRepository{}).updateWith(ctx, connection, updated); err != nil {
			return err
		}
		promoted = application.TaskPromotionResult{Task: updated, Issue: created}
		return nil
	})
	if err != nil {
		return application.TaskPromotionResult{}, err
	}
	if !mutation.Replayed {
		return promoted, nil
	}
	return r.loadPromotion(ctx, workspaceID, taskID)
}

func (r *taskPromotionRepository) insertPromotedIssue(ctx context.Context, connection *sql.Conn, value issueDomain.Issue) (issueDomain.Issue, error) {
	var prefix string
	var number int32
	now := value.UpdatedAt.UTC().Format(time.RFC3339Nano)
	if err := connection.QueryRowContext(ctx, `UPDATE workspaces SET next_issue_number=next_issue_number+1,updated_at=? WHERE id=? RETURNING issue_prefix,next_issue_number-1`, now, value.WorkspaceID).Scan(&prefix, &number); errors.Is(err, sql.ErrNoRows) {
		return issueDomain.Issue{}, application.ErrWorkspaceNotFound
	} else if err != nil {
		return issueDomain.Issue{}, fmt.Errorf("allocate promoted Issue number: %w", err)
	}
	created, err := value.AssignIdentity(number, prefix)
	if err != nil {
		return issueDomain.Issue{}, fmt.Errorf("assign promoted Issue identifier: %w", err)
	}
	metadata, properties, assets, err := encodeIssueJSON(created)
	if err != nil {
		return issueDomain.Issue{}, err
	}
	_, err = connection.ExecContext(ctx, `INSERT INTO workspace_issues(
		id,workspace_id,number,identifier,title,description,status,priority,
		assignee_type,assignee_id,creator_type,creator_id,parent_issue_id,project_id,
		position,stage,start_date,due_date,metadata,properties,asset_ids,created_at,updated_at
	) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		created.ID, created.WorkspaceID, created.Number, created.Identifier, created.Title, nullableString(created.Description),
		created.Status, created.Priority, nullableString(created.AssigneeType), nullableString(created.AssigneeID),
		created.CreatorType, created.CreatorID, nullableString(created.ParentIssueID), nullableString(created.ProjectID),
		created.Position, nullableInt32(created.Stage), nullableString(created.StartDate), nullableString(created.DueDate),
		metadata, properties, assets, created.CreatedAt.UTC().Format(time.RFC3339Nano), created.UpdatedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return issueDomain.Issue{}, fmt.Errorf("insert promoted Issue: %w", err)
	}
	return created, nil
}

func (r *taskPromotionRepository) loadPromotion(ctx context.Context, workspaceID, taskID string) (application.TaskPromotionResult, error) {
	var issueID string
	if err := r.db.QueryRowContext(ctx, `SELECT issue_id FROM workspace_task_issue_promotions WHERE workspace_id=? AND task_id=?`, workspaceID, taskID).Scan(&issueID); err != nil {
		return application.TaskPromotionResult{}, fmt.Errorf("load replayed Task promotion link: %w", err)
	}
	task, err := scanTodo(r.db.QueryRowContext(ctx, todoSelect+` WHERE workspace_id=? AND id=?`, workspaceID, taskID))
	if err != nil {
		return application.TaskPromotionResult{}, fmt.Errorf("load replayed promoted Task: %w", err)
	}
	issue, err := scanIssue(r.db.QueryRowContext(ctx, `SELECT `+issueColumns+` FROM workspace_issues WHERE workspace_id=? AND id=?`, workspaceID, issueID))
	if err != nil {
		return application.TaskPromotionResult{}, fmt.Errorf("load replayed promoted Issue: %w", err)
	}
	return application.TaskPromotionResult{Task: task, Issue: issue}, nil
}

var _ application.TaskPromotionReader = (*taskPromotionRepository)(nil)
var _ application.TaskPromotionRepository = (*taskPromotionRepository)(nil)
