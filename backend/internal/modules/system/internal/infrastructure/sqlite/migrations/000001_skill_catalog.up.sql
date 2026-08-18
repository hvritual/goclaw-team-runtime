CREATE TABLE IF NOT EXISTS system_skills (
    id TEXT PRIMARY KEY,
    origin_workspace_id TEXT NOT NULL,
    revision INTEGER NOT NULL CHECK (revision > 0),
    created_by TEXT NOT NULL,
    archived_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS system_skill_versions (
    id TEXT PRIMARY KEY,
    skill_id TEXT NOT NULL,
    version_number INTEGER NOT NULL CHECK (version_number > 0),
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    configuration TEXT NOT NULL DEFAULT '{}',
    status TEXT NOT NULL CHECK (status IN ('draft', 'published', 'deprecated', 'archived')),
    created_by TEXT NOT NULL,
    created_at TEXT NOT NULL,
    published_at TEXT,
    deprecated_at TEXT,
    UNIQUE (skill_id, version_number)
);

CREATE TABLE IF NOT EXISTS system_skill_audit (
    id TEXT PRIMARY KEY,
    skill_id TEXT NOT NULL,
    version_id TEXT,
    workspace_id TEXT NOT NULL,
    actor_type TEXT NOT NULL,
    actor_id TEXT NOT NULL,
    action TEXT NOT NULL,
    created_at TEXT NOT NULL
);
