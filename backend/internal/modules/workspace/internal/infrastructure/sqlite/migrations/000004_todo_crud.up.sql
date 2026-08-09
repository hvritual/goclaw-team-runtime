ALTER TABLE workspace_todos ADD COLUMN priority TEXT NOT NULL DEFAULT 'none'
    CHECK (priority IN ('urgent', 'high', 'medium', 'low', 'none'));
ALTER TABLE workspace_todos ADD COLUMN creator_type TEXT NOT NULL DEFAULT 'member'
    CHECK (creator_type IN ('member', 'agent'));
ALTER TABLE workspace_todos ADD COLUMN creator_id TEXT NOT NULL DEFAULT '';
ALTER TABLE workspace_todos ADD COLUMN position REAL NOT NULL DEFAULT 0;
ALTER TABLE workspace_todos ADD COLUMN start_date TEXT;
ALTER TABLE workspace_todos ADD COLUMN due_date TEXT;
ALTER TABLE workspace_todos ADD COLUMN completed_at TEXT;
