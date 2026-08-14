CREATE TABLE IF NOT EXISTS workspace_issue_comments (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    issue_id TEXT NOT NULL,
    author_type TEXT NOT NULL CHECK (author_type IN ('member', 'agent', 'system')),
    author_id TEXT NOT NULL,
    content TEXT NOT NULL,
    type TEXT NOT NULL DEFAULT 'comment' CHECK (type IN ('comment', 'status_change', 'progress_update', 'system')),
    parent_id TEXT,
    resolved_at TEXT,
    resolved_by_type TEXT CHECK (resolved_by_type IS NULL OR resolved_by_type IN ('member', 'agent')),
    resolved_by_id TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS workspace_comment_reactions (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    comment_id TEXT NOT NULL,
    actor_type TEXT NOT NULL CHECK (actor_type IN ('member', 'agent')),
    actor_id TEXT NOT NULL,
    emoji TEXT NOT NULL,
    created_at TEXT NOT NULL,
    UNIQUE (workspace_id, comment_id, actor_type, actor_id, emoji)
);

CREATE TABLE IF NOT EXISTS workspace_issue_reactions (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    issue_id TEXT NOT NULL,
    actor_type TEXT NOT NULL CHECK (actor_type IN ('member', 'agent')),
    actor_id TEXT NOT NULL,
    emoji TEXT NOT NULL,
    created_at TEXT NOT NULL,
    UNIQUE (workspace_id, issue_id, actor_type, actor_id, emoji)
);

CREATE TABLE IF NOT EXISTS workspace_issue_subscribers (
    workspace_id TEXT NOT NULL,
    issue_id TEXT NOT NULL,
    user_type TEXT NOT NULL CHECK (user_type IN ('member', 'agent')),
    user_id TEXT NOT NULL,
    reason TEXT NOT NULL CHECK (reason IN ('creator', 'assignee', 'commenter', 'mentioned', 'manual')),
    created_at TEXT NOT NULL,
    PRIMARY KEY (workspace_id, issue_id, user_type, user_id)
);

CREATE TABLE IF NOT EXISTS workspace_issue_activities (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    issue_id TEXT NOT NULL,
    actor_type TEXT NOT NULL CHECK (actor_type IN ('member', 'agent', 'system')),
    actor_id TEXT NOT NULL,
    action TEXT NOT NULL,
    details TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS workspace_comment_knowledge_proposals (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    comment_id TEXT NOT NULL,
    source_revision TEXT NOT NULL,
    content TEXT NOT NULL,
    actor_id TEXT NOT NULL,
    created_at TEXT NOT NULL,
    UNIQUE (workspace_id, comment_id, source_revision)
);
