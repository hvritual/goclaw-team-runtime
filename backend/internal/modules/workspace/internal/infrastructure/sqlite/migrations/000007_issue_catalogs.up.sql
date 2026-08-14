CREATE TABLE IF NOT EXISTS workspace_issue_labels (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    resource_type TEXT NOT NULL DEFAULT 'issue' CHECK (resource_type = 'issue'),
    name TEXT NOT NULL COLLATE NOCASE,
    description TEXT NOT NULL DEFAULT '',
    color TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE (workspace_id, resource_type, name)
);

CREATE TABLE IF NOT EXISTS workspace_issue_label_assignments (
    workspace_id TEXT NOT NULL,
    issue_id TEXT NOT NULL,
    label_id TEXT NOT NULL,
    created_at TEXT NOT NULL,
    PRIMARY KEY (workspace_id, issue_id, label_id)
);

CREATE TABLE IF NOT EXISTS workspace_issue_property_definitions (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    name TEXT NOT NULL COLLATE NOCASE,
    type TEXT NOT NULL CHECK (type IN ('text','number','select','multi_select','date','checkbox','url')),
    description TEXT NOT NULL DEFAULT '',
    icon TEXT NOT NULL DEFAULT '',
    config TEXT NOT NULL DEFAULT '{}' CHECK (json_valid(config)),
    position REAL NOT NULL DEFAULT 0,
    archived_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE (workspace_id, name)
);

CREATE TABLE IF NOT EXISTS workspace_issue_acceptance_conclusions (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    issue_id TEXT NOT NULL,
    result TEXT NOT NULL CHECK (result IN ('accepted','conditional','rejected')),
    rationale TEXT NOT NULL,
    evidence_refs TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(evidence_refs)),
    actor_id TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS workspace_acceptance_knowledge_proposals (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    issue_id TEXT NOT NULL,
    conclusion_id TEXT NOT NULL,
    source_revision TEXT NOT NULL,
    content TEXT NOT NULL,
    actor_id TEXT NOT NULL,
    created_at TEXT NOT NULL,
    UNIQUE (workspace_id, conclusion_id)
);
