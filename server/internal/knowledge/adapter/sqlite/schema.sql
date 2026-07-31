CREATE TABLE IF NOT EXISTS knowledge_evidence (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    project_id TEXT NOT NULL DEFAULT '',
    source_type TEXT NOT NULL,
    source_id TEXT NOT NULL,
    source_revision TEXT NOT NULL DEFAULT '',
    event_type TEXT NOT NULL,
    kind TEXT NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL,
    actor_id TEXT NOT NULL DEFAULT '',
    idempotency_key TEXT NOT NULL UNIQUE,
    provenance_uri TEXT NOT NULL DEFAULT '',
    checksum TEXT NOT NULL DEFAULT '',
    occurred_at TEXT NOT NULL,
    terminal INTEGER NOT NULL DEFAULT 0,
    validated INTEGER NOT NULL DEFAULT 0,
    has_conflict INTEGER NOT NULL DEFAULT 0,
    confidence REAL NOT NULL DEFAULT 0,
    source_refs_json TEXT NOT NULL DEFAULT '[]',
    metadata_json TEXT NOT NULL DEFAULT '{}'
);
CREATE INDEX IF NOT EXISTS knowledge_evidence_workspace_source_idx
    ON knowledge_evidence(workspace_id, source_type, source_id, occurred_at);

CREATE TABLE IF NOT EXISTS knowledge_candidate (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    project_id TEXT NOT NULL DEFAULT '',
    kind TEXT NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    reason TEXT NOT NULL,
    status TEXT NOT NULL,
    revision INTEGER NOT NULL,
    proposed_by TEXT NOT NULL,
    source_refs_json TEXT NOT NULL DEFAULT '[]',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS knowledge_candidate_workspace_status_idx
    ON knowledge_candidate(workspace_id, status, updated_at);

CREATE TABLE IF NOT EXISTS knowledge_entry (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    project_id TEXT NOT NULL DEFAULT '',
    candidate_id TEXT NOT NULL DEFAULT '',
    kind TEXT NOT NULL,
    status TEXT NOT NULL,
    current_revision INTEGER NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS knowledge_entry_workspace_status_idx
    ON knowledge_entry(workspace_id, status, updated_at);
CREATE INDEX IF NOT EXISTS knowledge_entry_project_idx
    ON knowledge_entry(workspace_id, project_id, updated_at);

CREATE TABLE IF NOT EXISTS knowledge_revision (
    entry_id TEXT NOT NULL,
    revision INTEGER NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    created_by TEXT NOT NULL,
    created_at TEXT NOT NULL,
    source_refs_json TEXT NOT NULL DEFAULT '[]',
    PRIMARY KEY(entry_id, revision)
);

CREATE TABLE IF NOT EXISTS knowledge_review (
    id TEXT PRIMARY KEY,
    candidate_id TEXT NOT NULL,
    workspace_id TEXT NOT NULL,
    action TEXT NOT NULL,
    reviewer_id TEXT NOT NULL,
    rationale TEXT NOT NULL,
    reviewed_at TEXT NOT NULL,
    old_revision INTEGER NOT NULL,
    new_revision INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS knowledge_review_candidate_idx
    ON knowledge_review(candidate_id, reviewed_at);
