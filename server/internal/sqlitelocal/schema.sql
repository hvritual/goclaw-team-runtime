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

CREATE TABLE IF NOT EXISTS invitations (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    inviter_id TEXT NOT NULL,
    invitee_email TEXT NOT NULL,
    invitee_user_id TEXT,
    role TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    expires_at TEXT NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS invitations_pending_workspace_email_idx
    ON invitations(workspace_id, invitee_email) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS invitations_invitee_idx
    ON invitations(invitee_email, invitee_user_id, status, expires_at);
CREATE INDEX IF NOT EXISTS invitations_workspace_idx
    ON invitations(workspace_id, status, created_at);

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

CREATE TABLE IF NOT EXISTS issue_acceptance_conclusion (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    issue_id TEXT NOT NULL,
    result TEXT NOT NULL,
    rationale TEXT NOT NULL,
    evidence_refs TEXT NOT NULL DEFAULT '[]',
    actor_id TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS issue_acceptance_conclusion_issue_idx
    ON issue_acceptance_conclusion(issue_id, created_at);

CREATE TABLE IF NOT EXISTS project_retrospective (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    summary TEXT NOT NULL,
    successes TEXT NOT NULL DEFAULT '[]',
    problems TEXT NOT NULL DEFAULT '[]',
    lessons TEXT NOT NULL DEFAULT '[]',
    follow_up_refs TEXT NOT NULL DEFAULT '[]',
    actor_id TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS project_retrospective_project_idx
    ON project_retrospective(project_id, created_at);

CREATE TABLE IF NOT EXISTS comments (
    id TEXT PRIMARY KEY,
    issue_id TEXT NOT NULL,
    workspace_id TEXT NOT NULL,
    author_type TEXT NOT NULL,
    author_id TEXT NOT NULL,
    content TEXT NOT NULL,
    type TEXT NOT NULL DEFAULT 'comment',
    parent_id TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS comments_issue_created_idx ON comments(issue_id, created_at);
CREATE INDEX IF NOT EXISTS comments_workspace_idx ON comments(workspace_id, updated_at);

CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    project_id TEXT,
    issue_id TEXT,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'todo',
    priority TEXT NOT NULL DEFAULT 'none',
    assignee_id TEXT,
    creator_id TEXT NOT NULL,
    position REAL NOT NULL DEFAULT 0,
    start_date TEXT,
    due_date TEXT,
    completed_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS tasks_workspace_created_idx ON tasks(workspace_id, created_at);
CREATE INDEX IF NOT EXISTS tasks_project_created_idx ON tasks(project_id, created_at);
CREATE INDEX IF NOT EXISTS tasks_issue_created_idx ON tasks(issue_id, created_at);

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

CREATE TABLE IF NOT EXISTS knowledge_evidence_outbox (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    evidence_id TEXT NOT NULL UNIQUE,
    idempotency_key TEXT NOT NULL UNIQUE,
    payload_json TEXT NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0,
    available_at TEXT NOT NULL,
    delivered_at TEXT,
    last_error TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS knowledge_evidence_outbox_pending_idx
    ON knowledge_evidence_outbox(delivered_at, available_at, created_at);

INSERT OR IGNORE INTO schema_migrations(version, applied_at)
VALUES (1, strftime('%Y-%m-%dT%H:%M:%fZ', 'now'));

INSERT OR IGNORE INTO schema_migrations(version, applied_at)
VALUES (2, strftime('%Y-%m-%dT%H:%M:%fZ', 'now'));

INSERT OR IGNORE INTO schema_migrations(version, applied_at)
VALUES (3, strftime('%Y-%m-%dT%H:%M:%fZ', 'now'));

INSERT OR IGNORE INTO schema_migrations(version, applied_at)
VALUES (4, strftime('%Y-%m-%dT%H:%M:%fZ', 'now'));

INSERT OR IGNORE INTO schema_migrations(version, applied_at)
VALUES (5, strftime('%Y-%m-%dT%H:%M:%fZ', 'now'));
