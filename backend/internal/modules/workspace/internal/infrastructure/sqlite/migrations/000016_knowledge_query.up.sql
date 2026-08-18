CREATE TABLE workspace_governed_knowledge (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    project_id TEXT,
    candidate_id TEXT,
    kind TEXT NOT NULL CHECK (kind IN ('goal','decision','constraint','requirement','procedure','lesson','reference')),
    status TEXT NOT NULL CHECK (status IN ('published','superseded','quarantined')),
    current_revision INTEGER NOT NULL CHECK (current_revision > 0),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE workspace_knowledge_revisions (
    knowledge_id TEXT NOT NULL,
    revision INTEGER NOT NULL CHECK (revision > 0),
    supersedes_revision INTEGER NOT NULL DEFAULT 0 CHECK (supersedes_revision >= 0),
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    created_by TEXT NOT NULL,
    created_at TEXT NOT NULL,
    PRIMARY KEY (knowledge_id, revision)
);

CREATE TABLE workspace_knowledge_source_refs (
    knowledge_id TEXT NOT NULL,
    revision INTEGER NOT NULL,
    ordinal INTEGER NOT NULL CHECK (ordinal >= 0),
    source_type TEXT NOT NULL,
    source_id TEXT NOT NULL,
    source_revision TEXT NOT NULL,
    citation TEXT NOT NULL,
    asset_id TEXT,
    asset_version_id TEXT,
    PRIMARY KEY (knowledge_id, revision, ordinal),
    CHECK ((asset_id IS NULL) = (asset_version_id IS NULL))
);

CREATE INDEX workspace_governed_knowledge_scope_idx
ON workspace_governed_knowledge(workspace_id, status, updated_at DESC, id ASC);

CREATE INDEX workspace_governed_knowledge_kind_idx
ON workspace_governed_knowledge(workspace_id, kind, updated_at DESC, id ASC);

CREATE INDEX workspace_knowledge_source_lookup_idx
ON workspace_knowledge_source_refs(source_type, source_id, source_revision, knowledge_id, revision);
