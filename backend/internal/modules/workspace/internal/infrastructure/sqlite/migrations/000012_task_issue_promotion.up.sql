CREATE TABLE workspace_task_issue_promotions (
    workspace_id TEXT NOT NULL CHECK (length(trim(workspace_id)) > 0),
    task_id TEXT NOT NULL CHECK (length(trim(task_id)) > 0),
    issue_id TEXT NOT NULL CHECK (length(trim(issue_id)) > 0),
    created_at TEXT NOT NULL CHECK (length(trim(created_at)) > 0),
    response_snapshot TEXT NOT NULL CHECK (json_valid(response_snapshot)),
    PRIMARY KEY (workspace_id, task_id),
    UNIQUE (workspace_id, issue_id)
);
