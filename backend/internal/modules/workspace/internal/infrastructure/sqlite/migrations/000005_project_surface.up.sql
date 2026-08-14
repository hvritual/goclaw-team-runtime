ALTER TABLE workspace_projects ADD COLUMN icon TEXT;
ALTER TABLE workspace_projects ADD COLUMN priority TEXT NOT NULL DEFAULT 'none';
ALTER TABLE workspace_projects ADD COLUMN lead_type TEXT;
ALTER TABLE workspace_projects ADD COLUMN lead_id TEXT;
ALTER TABLE workspace_projects ADD COLUMN start_date TEXT;
ALTER TABLE workspace_projects ADD COLUMN due_date TEXT;

CREATE TABLE IF NOT EXISTS workspace_pins (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    item_type TEXT NOT NULL CHECK (item_type IN ('issue', 'project')),
    item_id TEXT NOT NULL,
    position REAL NOT NULL,
    created_at TEXT NOT NULL,
    UNIQUE (workspace_id, user_id, item_type, item_id)
);
