CREATE TABLE IF NOT EXISTS auth_users (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    avatar_url TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS auth_members (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('owner', 'admin', 'member')),
    created_at TEXT NOT NULL,
    UNIQUE (workspace_id, user_id)
);

CREATE TABLE IF NOT EXISTS auth_workspace_membership_roots (
    workspace_id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    member_id TEXT NOT NULL,
    created_at TEXT NOT NULL
);
