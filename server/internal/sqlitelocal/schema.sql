PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA busy_timeout = 5000;
PRAGMA foreign_keys = OFF;

CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    applied_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    avatar_url TEXT,
    language TEXT,
    timezone TEXT,
    onboarded_at TEXT,
    onboarding_questionnaire TEXT NOT NULL DEFAULT '{}',
    starter_content_state TEXT,
    profile_description TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS auth_tokens (
    token TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS workspaces (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    description TEXT,
    context TEXT,
    settings TEXT NOT NULL DEFAULT '{}',
    repos TEXT NOT NULL DEFAULT '[]',
    issue_prefix TEXT NOT NULL,
    avatar_url TEXT,
    next_issue_number INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS members (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    role TEXT NOT NULL,
    created_at TEXT NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS members_workspace_user_idx ON members(workspace_id, user_id);
CREATE INDEX IF NOT EXISTS members_user_idx ON members(user_id);

CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    icon TEXT,
    status TEXT NOT NULL,
    priority TEXT NOT NULL,
    lead_type TEXT,
    lead_id TEXT,
    start_date TEXT,
    due_date TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS projects_workspace_idx ON projects(workspace_id, updated_at);

CREATE TABLE IF NOT EXISTS issues (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    number INTEGER NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL,
    priority TEXT NOT NULL,
    assignee_type TEXT,
    assignee_id TEXT,
    creator_type TEXT NOT NULL,
    creator_id TEXT NOT NULL,
    parent_issue_id TEXT,
    project_id TEXT,
    position REAL NOT NULL DEFAULT 0,
    stage INTEGER,
    start_date TEXT,
    due_date TEXT,
    metadata TEXT NOT NULL DEFAULT '{}',
    properties TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS issues_workspace_number_idx ON issues(workspace_id, number);
CREATE INDEX IF NOT EXISTS issues_workspace_updated_idx ON issues(workspace_id, updated_at);
CREATE INDEX IF NOT EXISTS issues_project_idx ON issues(project_id);

CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    issue_id TEXT,
    agent_id TEXT NOT NULL DEFAULT '',
    runtime_id TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    priority INTEGER NOT NULL DEFAULT 0,
    dispatched_at TEXT,
    started_at TEXT,
    completed_at TEXT,
    result TEXT,
    error TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS tasks_workspace_created_idx ON tasks(workspace_id, created_at);
CREATE INDEX IF NOT EXISTS tasks_issue_created_idx ON tasks(issue_id, created_at);

CREATE TABLE IF NOT EXISTS task_messages (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    role TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS task_messages_task_idx ON task_messages(task_id, created_at);

CREATE TABLE IF NOT EXISTS skills (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL DEFAULT '',
    config TEXT NOT NULL DEFAULT '{}',
    created_by TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS skills_workspace_name_idx ON skills(workspace_id, name);

CREATE TABLE IF NOT EXISTS skill_files (
    id TEXT PRIMARY KEY,
    skill_id TEXT NOT NULL,
    path TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS skill_files_skill_path_idx ON skill_files(skill_id, path);

INSERT OR IGNORE INTO schema_migrations(version, applied_at)
VALUES (1, strftime('%Y-%m-%dT%H:%M:%fZ', 'now'));
