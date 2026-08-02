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
