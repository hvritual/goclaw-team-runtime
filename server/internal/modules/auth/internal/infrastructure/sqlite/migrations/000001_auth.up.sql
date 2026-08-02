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

CREATE TABLE IF NOT EXISTS members (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    role TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS members_workspace_user_idx
    ON members(workspace_id, user_id);
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
