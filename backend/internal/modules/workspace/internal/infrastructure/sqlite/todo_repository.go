package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
	todoDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/todo"
)

type todoRepository struct{ db *sql.DB }

func NewTodoRepository(c Config) (application.TodoRepository, error) {
	if c.DB == nil {
		return nil, errors.New("workspace sqlite database is required")
	}
	return &todoRepository{c.DB}, nil
}

func (r *todoRepository) Create(ctx context.Context, value todoDomain.Todo) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO workspace_todos(
		id,workspace_id,title,description,status,project_id,issue_id,assignee_type,assignee_id,
		created_at,updated_at,priority,creator_type,creator_id,position,start_date,due_date,completed_at
	) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		value.ID, value.WorkspaceID, value.Title, value.Description, value.Status,
		nullable(value.ProjectID), nullable(value.IssueID), nullable(value.AssigneeType), nullable(value.AssigneeID),
		value.CreatedAt.Format(time.RFC3339Nano), value.UpdatedAt.Format(time.RFC3339Nano), value.Priority,
		value.CreatorType, value.CreatorID, value.Position, nullableTime(value.StartDate), nullableTime(value.DueDate), nullableTime(value.CompletedAt),
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
	rows, err := r.db.QueryContext(ctx, todoSelect+` WHERE `+strings.Join(clauses, " AND ")+` ORDER BY position ASC, created_at DESC, id ASC`, args...)
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
	result, err := r.db.ExecContext(ctx, `UPDATE workspace_todos SET
		project_id=?,issue_id=?,title=?,description=?,status=?,priority=?,assignee_type=?,assignee_id=?,
		position=?,start_date=?,due_date=?,completed_at=?,updated_at=?
		WHERE workspace_id=? AND id=?`,
		nullable(value.ProjectID), nullable(value.IssueID), value.Title, value.Description, value.Status, value.Priority,
		nullable(value.AssigneeType), nullable(value.AssigneeID), value.Position,
		nullableTime(value.StartDate), nullableTime(value.DueDate), nullableTime(value.CompletedAt),
		value.UpdatedAt.Format(time.RFC3339Nano), value.WorkspaceID, value.ID,
	)
	if err != nil {
		return fmt.Errorf("update Workspace Todo: %w", err)
	}
	return requireTodoAffected(result)
}

func (r *todoRepository) Delete(ctx context.Context, workspaceID, todoID string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM workspace_todos WHERE workspace_id=? AND id=?`, workspaceID, todoID)
	if err != nil {
		return fmt.Errorf("delete Workspace Todo: %w", err)
	}
	return requireTodoAffected(result)
}

const todoSelect = `SELECT
	id,workspace_id,title,description,status,project_id,issue_id,assignee_type,assignee_id,
	created_at,updated_at,priority,creator_type,creator_id,position,start_date,due_date,completed_at
	FROM workspace_todos`

type todoScanner interface{ Scan(...any) error }

func scanTodo(scanner todoScanner) (todoDomain.Todo, error) {
	var value todoDomain.Todo
	var projectID, issueID, assigneeType, assigneeID sql.NullString
	var startDate, dueDate, completedAt sql.NullString
	var createdAt, updatedAt string
	err := scanner.Scan(
		&value.ID, &value.WorkspaceID, &value.Title, &value.Description, &value.Status,
		&projectID, &issueID, &assigneeType, &assigneeID, &createdAt, &updatedAt,
		&value.Priority, &value.CreatorType, &value.CreatorID, &value.Position,
		&startDate, &dueDate, &completedAt,
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
