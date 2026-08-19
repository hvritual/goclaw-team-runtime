CREATE TABLE workspace_project_resource_sets (
    workspace_id TEXT NOT NULL CHECK (length(trim(workspace_id)) > 0),
    project_id TEXT NOT NULL CHECK (length(trim(project_id)) > 0),
    revision INTEGER NOT NULL DEFAULT 0 CHECK (revision >= 0),
    updated_at TEXT NOT NULL CHECK (length(trim(updated_at)) > 0),
    PRIMARY KEY (workspace_id, project_id)
) WITHOUT ROWID;

CREATE TABLE workspace_project_resources (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL CHECK (length(trim(workspace_id)) > 0),
    project_id TEXT NOT NULL CHECK (length(trim(project_id)) > 0),
    resource_type TEXT NOT NULL CHECK (resource_type IN ('github_repo','url')),
    canonical_url TEXT NOT NULL CHECK (
        length(trim(canonical_url)) BETWEEN 1 AND 2048
        AND instr(canonical_url, char(10)) = 0
        AND instr(canonical_url, char(13)) = 0
    ),
    resource_ref TEXT NOT NULL DEFAULT '' CHECK (length(resource_ref) <= 255),
    fingerprint TEXT NOT NULL CHECK (length(fingerprint) = 64),
    label TEXT NOT NULL DEFAULT '' CHECK (length(label) <= 120),
    position INTEGER NOT NULL CHECK (position >= 0),
    status TEXT NOT NULL CHECK (status IN ('active','archived')),
    revision INTEGER NOT NULL CHECK (revision >= 1),
    connection_state TEXT NOT NULL DEFAULT 'unchecked'
        CHECK (connection_state IN ('unchecked','available','degraded','unavailable')),
    connection_diagnostic_code TEXT NOT NULL DEFAULT ''
        CHECK (length(connection_diagnostic_code) <= 64),
    connection_checked_at TEXT,
    created_at TEXT NOT NULL CHECK (length(trim(created_at)) > 0),
    created_by TEXT NOT NULL CHECK (length(trim(created_by)) > 0),
    updated_at TEXT NOT NULL CHECK (length(trim(updated_at)) > 0),
    updated_by TEXT NOT NULL CHECK (length(trim(updated_by)) > 0),
    archived_at TEXT,
    archived_by TEXT
);
