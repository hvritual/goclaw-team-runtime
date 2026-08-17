CREATE TEMP TABLE task_lifecycle_rollback_guard (
    safe INTEGER NOT NULL CHECK (safe = 1)
);

INSERT INTO task_lifecycle_rollback_guard(safe)
SELECT CASE WHEN EXISTS (
    SELECT 1
    FROM workspace_todos
    WHERE revision <> 1 OR status = 'archived'
        OR restore_status IS NOT NULL OR archived_at IS NOT NULL
) THEN 0 ELSE 1 END;

DROP TABLE task_lifecycle_rollback_guard;

CREATE TABLE workspace_todos_v9 (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL
        CHECK (status IN ('todo', 'in_progress', 'done', 'cancelled')),
    project_id TEXT,
    issue_id TEXT,
    assignee_type TEXT,
    assignee_id TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    priority TEXT NOT NULL DEFAULT 'none'
        CHECK (priority IN ('urgent', 'high', 'medium', 'low', 'none')),
    creator_type TEXT NOT NULL DEFAULT 'member'
        CHECK (creator_type IN ('member', 'agent')),
    creator_id TEXT NOT NULL DEFAULT '',
    position REAL NOT NULL DEFAULT 0,
    start_date TEXT,
    due_date TEXT,
    completed_at TEXT,
    CHECK (assignee_type IS NULL OR assignee_type IN ('member', 'agent')),
    CHECK ((assignee_type IS NULL) = (assignee_id IS NULL))
);

INSERT INTO workspace_todos_v9 (
    id, workspace_id, title, description, status, project_id, issue_id,
    assignee_type, assignee_id, created_at, updated_at, priority, creator_type,
    creator_id, position, start_date, due_date, completed_at
)
SELECT
    id, workspace_id, title, description, status, project_id, issue_id,
    assignee_type, assignee_id, created_at, updated_at, priority, creator_type,
    creator_id, position, start_date, due_date, completed_at
FROM workspace_todos;

DROP TABLE workspace_todos;
ALTER TABLE workspace_todos_v9 RENAME TO workspace_todos;

DELETE FROM workspace_schema_migrations
WHERE version = '000010_task_lifecycle.up.sql';
