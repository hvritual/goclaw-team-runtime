CREATE TABLE IF NOT EXISTS workspace_projects (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL CHECK (status IN ('planned', 'in_progress', 'paused', 'completed', 'cancelled')),
    asset_ids TEXT NOT NULL DEFAULT '[]',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS workspace_project_actor_relations (
    workspace_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    actor_type TEXT NOT NULL CHECK (actor_type IN ('member', 'agent')),
    actor_id TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('lead', 'member', 'agent')),
    PRIMARY KEY (workspace_id, project_id, actor_type, actor_id)
);

