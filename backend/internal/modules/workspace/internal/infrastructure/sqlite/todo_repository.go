package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
	todoDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/todo"
)

type todoRepository struct {
	db                *sql.DB
	governance        *GovernanceRepository
	governanceService application.GovernanceService
}

func NewTodoRepository(c Config) (application.TodoRepository, error) {
	if c.DB == nil {
		return nil, errors.New("workspace sqlite database is required")
	}
	return &todoRepository{db: c.DB}, nil
}

func NewGovernedTodoRepository(c Config) (application.TodoRepository, error) {
	if c.DB == nil {
		return nil, errors.New("workspace sqlite database is required")
	}
	provider := application.TaskGovernancePolicyProvider{}
	governance, err := NewGovernanceRepository(c, WithGovernanceEventPolicies(provider))
	if err != nil {
		return nil, err
	}
	return &todoRepository{db: c.DB, governance: governance, governanceService: application.NewGovernanceService(provider)}, nil
}

func (r *todoRepository) Create(ctx context.Context, value todoDomain.Todo) (todoDomain.Todo, error) {
	if r.governance == nil {
		return value, r.createWith(ctx, r.db, value)
	}
	result, err := r.executeGoverned(ctx, value, 0, httpStatusCreated, func(ctx context.Context, connection *sql.Conn) error {
		return r.createWith(ctx, connection, value)
	})
	if err != nil {
		return todoDomain.Todo{}, err
	}
	if !result.Replayed {
		return value, nil
	}
	var replay struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(result.ResponseBody, &replay); err != nil || strings.TrimSpace(replay.Data.ID) == "" {
		return todoDomain.Todo{}, fmt.Errorf("decode Task governance replay: %w", contract.ErrInvalidGovernanceMutation)
	}
	replayed, err := r.FindByID(ctx, value.WorkspaceID, replay.Data.ID)
	if err != nil {
		return todoDomain.Todo{}, fmt.Errorf("load replayed Task: %w", err)
	}
	return replayed, nil
}

type todoSQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func (r *todoRepository) createWith(ctx context.Context, executor todoSQLExecutor, value todoDomain.Todo) error {
	_, err := executor.ExecContext(ctx, `INSERT INTO workspace_todos(
		id,workspace_id,title,description,status,project_id,issue_id,assignee_type,assignee_id,
		created_at,updated_at,priority,creator_type,creator_id,position,start_date,due_date,completed_at,
		revision,restore_status,archived_at
	) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		value.ID, value.WorkspaceID, value.Title, value.Description, value.Status,
		nullable(value.ProjectID), nullable(value.IssueID), nullable(value.AssigneeType), nullable(value.AssigneeID),
		value.CreatedAt.Format(time.RFC3339Nano), value.UpdatedAt.Format(time.RFC3339Nano), value.Priority,
		value.CreatorType, value.CreatorID, value.Position, nullableTime(value.StartDate), nullableTime(value.DueDate), nullableTime(value.CompletedAt),
		value.Revision, nullableTodoString(value.RestoreStatus), nullableTime(value.ArchivedAt),
	)
	if err != nil {
		return fmt.Errorf("insert Workspace Todo: %w", err)
	}
	return nil
}

func (r *todoRepository) FindByID(ctx context.Context, workspaceID, todoID string) (todoDomain.Todo, error) {
	return scanTodo(r.db.QueryRowContext(ctx, todoSelect+` WHERE workspace_id=? AND id=?`, workspaceID, todoID))
}

func (r *todoRepository) List(ctx context.Context, query application.TodoListQuery) ([]todoDomain.Todo, error) {
	clauses := []string{"workspace_id=?"}
	args := []any{query.WorkspaceID}
	if query.ProjectID != nil {
		clauses = append(clauses, "project_id=?")
		args = append(args, *query.ProjectID)
	}
	if query.IssueID != nil {
		clauses = append(clauses, "issue_id=?")
		args = append(args, *query.IssueID)
	}
	if query.Status != "" {
		clauses = append(clauses, "status=?")
		args = append(args, query.Status)
	}
	if query.Limit < 1 || query.Limit > application.MaxTodoListLimit {
		return nil, fmt.Errorf("list Workspace Todos: invalid limit")
	}
	args = append(args, query.Limit)
	rows, err := r.db.QueryContext(ctx, todoSelect+` WHERE `+strings.Join(clauses, " AND ")+` ORDER BY position ASC, created_at DESC, id ASC LIMIT ?`, args...)
	if err != nil {
		return nil, fmt.Errorf("list Workspace Todos: %w", err)
	}
	defer rows.Close()
	values := make([]todoDomain.Todo, 0)
	for rows.Next() {
		value, scanErr := scanTodo(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Workspace Todos: %w", err)
	}
	return values, nil
}

func (r *todoRepository) Update(ctx context.Context, value todoDomain.Todo) error {
	if r.governance == nil {
		return r.updateWith(ctx, r.db, value)
	}
	_, err := r.executeGoverned(ctx, value, value.Revision-1, httpStatusOK, func(ctx context.Context, connection *sql.Conn) error {
		return r.updateWith(ctx, connection, value)
	})
	return err
}

func (r *todoRepository) updateWith(ctx context.Context, executor todoSQLExecutor, value todoDomain.Todo) error {
	result, err := executor.ExecContext(ctx, `UPDATE workspace_todos SET
		project_id=?,issue_id=?,title=?,description=?,status=?,priority=?,assignee_type=?,assignee_id=?,
		position=?,start_date=?,due_date=?,completed_at=?,updated_at=?,revision=?,restore_status=?,archived_at=?
		WHERE workspace_id=? AND id=? AND revision=?`,
		nullable(value.ProjectID), nullable(value.IssueID), value.Title, value.Description, value.Status, value.Priority,
		nullable(value.AssigneeType), nullable(value.AssigneeID), value.Position,
		nullableTime(value.StartDate), nullableTime(value.DueDate), nullableTime(value.CompletedAt),
		value.UpdatedAt.Format(time.RFC3339Nano), value.Revision, nullableTodoString(value.RestoreStatus), nullableTime(value.ArchivedAt), value.WorkspaceID, value.ID, value.Revision-1,
	)
	if err != nil {
		return fmt.Errorf("update Workspace Todo: %w", err)
	}
	return requireTodoAffected(result)
}

const (
	httpStatusOK      = 200
	httpStatusCreated = 201
)

func (r *todoRepository) executeGoverned(ctx context.Context, value todoDomain.Todo, expectedRevision int64, responseStatus int, apply DomainMutation) (contract.MutationResult, error) {
	commandContext, ok := application.TodoGovernanceCommandFromContext(ctx)
	if !ok {
		return contract.MutationResult{}, contract.ErrGovernanceUnavailable
	}
	action := commandContext.Action
	eventType, ok := todoEventForAction(action)
	if !ok {
		return contract.MutationResult{}, contract.ErrGovernanceUnavailable
	}
	status := value.Status
	if status == todoDomain.StatusArchived {
		status = "removed"
	}
	fields := map[string]any{"id": value.ID, "revision": value.Revision, "status": status}
	requestFields := fields
	if action == application.TaskActionCreate {
		requestFields = map[string]any{"fingerprint": commandContext.RequestFingerprint}
	}
	prepared, err := r.governanceService.PrepareContext(ctx, application.GovernanceRequest{
		Identity:       contract.MutationIdentity{WorkspaceID: value.WorkspaceID, RequestID: uuid.NewString()},
		Command:        contract.MutationCommand{Action: action, ResourceKind: "task", ResourceID: value.ID, ExpectedRevision: expectedRevision, IdempotencyKey: commandContext.IdempotencyKey},
		RequestFields:  requestFields,
		ResponseStatus: responseStatus,
		ResponseFields: fields,
		AuditID:        uuid.NewString(),
		OccurredAt:     value.UpdatedAt,
		AuditFields:    fields,
		Outbox: []application.OutboxDraft{{
			ID: uuid.NewString(), EventType: eventType, Fields: fields,
		}},
	})
	if err != nil {
		return contract.MutationResult{}, err
	}
	return r.governance.Execute(ctx, prepared, apply)
}

func todoEventForAction(action string) (string, bool) {
	switch action {
	case application.TaskActionCreate:
		return "task:created", true
	case application.TaskActionUpdate:
		return "task:updated", true
	case application.TaskActionArchive:
		return "task:updated", true
	case application.TaskActionRestore:
		return "task:updated", true
	case application.TaskActionReorder:
		return "task:updated", true
	default:
		return "", false
	}
}

func (r *todoRepository) Reorder(ctx context.Context, workspaceID string, updates []application.TodoPositionUpdate, now time.Time) (values []todoDomain.Todo, err error) {
	if r.governance != nil {
		commandContext, ok := application.TodoGovernanceCommandFromContext(ctx)
		if !ok || commandContext.Action != application.TaskActionReorder || commandContext.IdempotencyKey == "" || commandContext.RequestFingerprint == "" {
			return nil, contract.ErrGovernanceUnavailable
		}
		action := commandContext.Action
		batchID := uuid.NewString()
		fields := map[string]any{"id": batchID, "revision": int64(1), "status": "reordered"}
		prepared, prepareErr := r.governanceService.PrepareContext(ctx, application.GovernanceRequest{
			Identity:       contract.MutationIdentity{WorkspaceID: workspaceID, RequestID: uuid.NewString()},
			Command:        contract.MutationCommand{Action: action, ResourceKind: "task_order", ResourceID: batchID, ExpectedRevision: 0, IdempotencyKey: commandContext.IdempotencyKey},
			RequestFields:  map[string]any{"fingerprint": commandContext.RequestFingerprint},
			ResponseStatus: httpStatusOK,
			ResponseFields: fields,
			AuditID:        uuid.NewString(),
			OccurredAt:     now.UTC(),
			AuditFields:    fields,
			Outbox: []application.OutboxDraft{{
				ID: uuid.NewString(), EventType: "task:updated", Fields: fields,
			}},
		})
		if prepareErr != nil {
			return nil, prepareErr
		}
		result, executeErr := r.governance.Execute(ctx, prepared, func(ctx context.Context, connection *sql.Conn) error {
			values, err = r.reorderWithConnection(ctx, connection, workspaceID, updates, now)
			return err
		})
		if executeErr != nil || !result.Replayed {
			return values, executeErr
		}
		values = make([]todoDomain.Todo, 0, len(updates))
		for _, update := range updates {
			value, findErr := r.FindByID(ctx, workspaceID, update.TodoID)
			if findErr != nil {
				return nil, fmt.Errorf("load replayed reordered Task: %w", findErr)
			}
			values = append(values, value)
		}
		sort.Slice(values, func(i, j int) bool {
			if values[i].Position == values[j].Position {
				return values[i].ID < values[j].ID
			}
			return values[i].Position < values[j].Position
		})
		return values, nil
	}
	connection, err := r.db.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquire Todo reorder connection: %w", err)
	}
	defer connection.Close()
	if _, err = connection.ExecContext(ctx, `PRAGMA busy_timeout = 5000`); err != nil {
		return nil, fmt.Errorf("configure Todo reorder connection: %w", err)
	}
	if _, err = connection.ExecContext(ctx, `BEGIN IMMEDIATE`); err != nil {
		return nil, fmt.Errorf("begin Todo reorder: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = connection.ExecContext(context.WithoutCancel(ctx), `ROLLBACK`)
		}
	}()
	values, err = r.reorderWithConnection(ctx, connection, workspaceID, updates, now)
	if err != nil {
		return nil, err
	}
	if _, err = connection.ExecContext(ctx, `COMMIT`); err != nil {
		return nil, fmt.Errorf("commit Todo reorder: %w", err)
	}
	committed = true
	return values, nil
}

func (r *todoRepository) reorderWithConnection(ctx context.Context, connection *sql.Conn, workspaceID string, updates []application.TodoPositionUpdate, now time.Time) (values []todoDomain.Todo, err error) {
	updatedAt := now.UTC().Format(time.RFC3339Nano)
	for _, update := range updates {
		current, findErr := scanTodo(connection.QueryRowContext(ctx, todoSelect+` WHERE workspace_id=? AND id=?`, workspaceID, update.TodoID))
		if findErr != nil {
			return nil, findErr
		}
		if current.Revision != update.ExpectedRevision {
			return nil, contract.RevisionConflictError{CurrentRevision: current.Revision}
		}
		result, updateErr := connection.ExecContext(ctx, `UPDATE workspace_todos SET position=?,updated_at=?,revision=revision+1 WHERE workspace_id=? AND id=? AND revision=?`,
			update.Position, updatedAt, workspaceID, update.TodoID, update.ExpectedRevision)
		if updateErr != nil {
			return nil, fmt.Errorf("reorder Workspace Todo: %w", updateErr)
		}
		if updateErr = requireTodoAffected(result); updateErr != nil {
			return nil, updateErr
		}
		if r.governance != nil {
			if _, updateErr = connection.ExecContext(ctx, `INSERT INTO workspace_resource_revisions
				(workspace_id,resource_kind,resource_id,revision,updated_at) VALUES(?,?,?,?,?)
				ON CONFLICT(workspace_id,resource_kind,resource_id) DO UPDATE SET revision=excluded.revision,updated_at=excluded.updated_at`,
				workspaceID, "task", update.TodoID, update.ExpectedRevision+1, updatedAt); updateErr != nil {
				return nil, fmt.Errorf("persist reordered Task revision: %w", updateErr)
			}
		}
		current.Position = update.Position
		current.UpdatedAt = now.UTC()
		current.Revision++
		values = append(values, current)
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].Position == values[j].Position {
			return values[i].ID < values[j].ID
		}
		return values[i].Position < values[j].Position
	})
	return values, nil
}

const todoSelect = `SELECT
	id,workspace_id,title,description,status,project_id,issue_id,assignee_type,assignee_id,
	created_at,updated_at,priority,creator_type,creator_id,position,start_date,due_date,completed_at,
	revision,restore_status,archived_at
	FROM workspace_todos`

type todoScanner interface{ Scan(...any) error }

func scanTodo(scanner todoScanner) (todoDomain.Todo, error) {
	var value todoDomain.Todo
	var projectID, issueID, assigneeType, assigneeID sql.NullString
	var startDate, dueDate, completedAt, restoreStatus, archivedAt sql.NullString
	var createdAt, updatedAt string
	err := scanner.Scan(
		&value.ID, &value.WorkspaceID, &value.Title, &value.Description, &value.Status,
		&projectID, &issueID, &assigneeType, &assigneeID, &createdAt, &updatedAt,
		&value.Priority, &value.CreatorType, &value.CreatorID, &value.Position,
		&startDate, &dueDate, &completedAt, &value.Revision, &restoreStatus, &archivedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return todoDomain.Todo{}, application.ErrTodoRecordNotFound
	}
	if err != nil {
		return todoDomain.Todo{}, fmt.Errorf("scan Workspace Todo: %w", err)
	}
	value.ProjectID = pointer(projectID)
	value.IssueID = pointer(issueID)
	value.AssigneeType = pointer(assigneeType)
	value.AssigneeID = pointer(assigneeID)
	if restoreStatus.Valid {
		value.RestoreStatus = restoreStatus.String
	}
	if value.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt); err != nil {
		return todoDomain.Todo{}, fmt.Errorf("parse Todo created_at: %w", err)
	}
	if value.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt); err != nil {
		return todoDomain.Todo{}, fmt.Errorf("parse Todo updated_at: %w", err)
	}
	if value.StartDate, err = parseNullableTime(startDate); err != nil {
		return todoDomain.Todo{}, fmt.Errorf("parse Todo start_date: %w", err)
	}
	if value.DueDate, err = parseNullableTime(dueDate); err != nil {
		return todoDomain.Todo{}, fmt.Errorf("parse Todo due_date: %w", err)
	}
	if value.CompletedAt, err = parseNullableTime(completedAt); err != nil {
		return todoDomain.Todo{}, fmt.Errorf("parse Todo completed_at: %w", err)
	}
	if value.ArchivedAt, err = parseNullableTime(archivedAt); err != nil {
		return todoDomain.Todo{}, fmt.Errorf("parse Todo archived_at: %w", err)
	}
	value, err = todoDomain.Rehydrate(value)
	if err != nil {
		return todoDomain.Todo{}, fmt.Errorf("map Todo: %w", err)
	}
	return value, nil
}

func requireTodoAffected(result sql.Result) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read Todo affected rows: %w", err)
	}
	if affected == 0 {
		return application.ErrTodoRecordNotFound
	}
	return nil
}

func nullable(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableTodoString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func pointer(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	copied := value.String
	return &copied
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func parseNullableTime(value sql.NullString) (*time.Time, error) {
	if !value.Valid {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, value.String)
	if err != nil {
		return nil, err
	}
	parsed = parsed.UTC()
	return &parsed, nil
}
