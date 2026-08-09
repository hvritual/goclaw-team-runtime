CREATE TABLE IF NOT EXISTS workspace_todos (
    id TEXT PRIMARY KEY, workspace_id TEXT NOT NULL, title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL CHECK (status IN ('todo', 'in_progress', 'done', 'cancelled')),
    project_id TEXT, issue_id TEXT, assignee_type TEXT, assignee_id TEXT,
    created_at TEXT NOT NULL, updated_at TEXT NOT NULL,
    CHECK (assignee_type IS NULL OR assignee_type IN ('member', 'agent')),
    CHECK ((assignee_type IS NULL) = (assignee_id IS NULL))
);
CREATE TABLE IF NOT EXISTS workspace_issues (
    id TEXT PRIMARY KEY, workspace_id TEXT NOT NULL, number INTEGER NOT NULL,
    identifier TEXT NOT NULL, title TEXT NOT NULL, description TEXT,
    status TEXT NOT NULL CHECK (status IN ('backlog', 'todo', 'in_progress', 'in_review', 'done', 'blocked', 'cancelled')),
    priority TEXT NOT NULL, assignee_type TEXT,
    assignee_id TEXT, creator_type TEXT NOT NULL, creator_id TEXT NOT NULL,
    parent_issue_id TEXT, project_id TEXT, position REAL NOT NULL DEFAULT 0,
    stage INTEGER, start_date TEXT, due_date TEXT, metadata TEXT NOT NULL DEFAULT '{}',
    properties TEXT NOT NULL DEFAULT '{}', asset_ids TEXT NOT NULL DEFAULT '[]',
    created_at TEXT NOT NULL, updated_at TEXT NOT NULL,
    UNIQUE (workspace_id, identifier)
);
CREATE TABLE IF NOT EXISTS workspace_knowledge (
    id TEXT PRIMARY KEY, workspace_id TEXT NOT NULL, title TEXT NOT NULL,
    summary TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL CHECK (status IN ('candidate', 'in_review', 'published', 'quarantined')),
    asset_ids TEXT NOT NULL DEFAULT '[]', created_at TEXT NOT NULL, updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS workspace_requirements (
    id TEXT PRIMARY KEY, workspace_id TEXT NOT NULL, project_id TEXT NOT NULL,
    title TEXT NOT NULL, current_version INTEGER NOT NULL,
    approval_status TEXT NOT NULL CHECK (approval_status = 'draft'),
    coverage_status TEXT NOT NULL CHECK (coverage_status IN ('uncovered', 'covered')),
    issue_ids TEXT NOT NULL DEFAULT '[]', created_at TEXT NOT NULL, updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS workspace_requirement_versions (
    id TEXT PRIMARY KEY, requirement_id TEXT NOT NULL, version INTEGER NOT NULL,
    content TEXT NOT NULL, created_at TEXT NOT NULL,
    UNIQUE (requirement_id, version)
);
CREATE TABLE IF NOT EXISTS workspace_settings (
    workspace_id TEXT NOT NULL, key TEXT NOT NULL, value TEXT NOT NULL,
    updated_at TEXT NOT NULL, PRIMARY KEY (workspace_id, key)
);
CREATE TABLE IF NOT EXISTS workspace_skill_bindings (
    workspace_id TEXT NOT NULL, skill_id TEXT NOT NULL, skill_version_id TEXT,
    enabled INTEGER NOT NULL CHECK (enabled IN (0, 1)), configuration TEXT NOT NULL DEFAULT '{}',
    agent_ids TEXT NOT NULL DEFAULT '[]', updated_at TEXT NOT NULL,
    PRIMARY KEY (workspace_id, skill_id)
);
