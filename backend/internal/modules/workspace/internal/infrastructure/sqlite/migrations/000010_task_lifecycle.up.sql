CREATE TABLE workspace_todos_v10 (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL
        CHECK (status IN ('todo', 'in_progress', 'done', 'cancelled', 'archived')),
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
    revision INTEGER NOT NULL DEFAULT 1 CHECK (revision >= 1),
    restore_status TEXT
        CHECK (restore_status IS NULL OR restore_status IN ('done', 'cancelled')),
    archived_at TEXT,
    CHECK (assignee_type IS NULL OR assignee_type IN ('member', 'agent')),
    CHECK ((assignee_type IS NULL) = (assignee_id IS NULL)),
    CHECK (
        (status = 'archived' AND restore_status IS NOT NULL AND archived_at IS NOT NULL)
        OR (status <> 'archived' AND restore_status IS NULL AND archived_at IS NULL)
    )
);

INSERT INTO workspace_todos_v10 (
    id, workspace_id, title, description, status, project_id, issue_id,
    assignee_type, assignee_id, created_at, updated_at, priority, creator_type,
    creator_id, position, start_date, due_date, completed_at, revision,
    restore_status, archived_at
)
SELECT
    id, workspace_id, title, description, status, project_id, issue_id,
    assignee_type, assignee_id, created_at, updated_at, priority, creator_type,
    creator_id, position, start_date, due_date, completed_at, 1, NULL, NULL
FROM workspace_todos;

DROP TABLE workspace_todos;
ALTER TABLE workspace_todos_v10 RENAME TO workspace_todos;
