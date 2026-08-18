ALTER TABLE workspace_governed_knowledge RENAME TO workspace_governed_knowledge_v16;

CREATE TABLE workspace_governed_knowledge (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    project_id TEXT,
    candidate_id TEXT,
    kind TEXT NOT NULL CHECK (kind IN ('goal','decision','constraint','requirement','procedure','lesson','reference')),
    status TEXT NOT NULL CHECK (status IN ('published','superseded','quarantined','invalidated')),
    current_revision INTEGER NOT NULL CHECK (current_revision > 0),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

INSERT INTO workspace_governed_knowledge(id,workspace_id,project_id,candidate_id,kind,status,current_revision,created_at,updated_at)
SELECT id,workspace_id,project_id,candidate_id,kind,status,current_revision,created_at,updated_at
FROM workspace_governed_knowledge_v16;

DROP TABLE workspace_governed_knowledge_v16;

CREATE INDEX workspace_governed_knowledge_scope_idx
ON workspace_governed_knowledge(workspace_id, status, updated_at DESC, id ASC);

CREATE INDEX workspace_governed_knowledge_kind_idx
ON workspace_governed_knowledge(workspace_id, kind, updated_at DESC, id ASC);

CREATE TABLE workspace_knowledge_candidates (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    project_id TEXT,
    knowledge_id TEXT,
    target_revision INTEGER NOT NULL DEFAULT 0 CHECK (target_revision >= 0),
    kind TEXT NOT NULL CHECK (kind IN ('goal','decision','constraint','requirement','procedure','lesson','reference')),
    title TEXT NOT NULL CHECK (length(trim(title)) > 0),
    content TEXT NOT NULL CHECK (length(trim(content)) > 0),
    reason TEXT NOT NULL CHECK (length(trim(reason)) > 0),
    status TEXT NOT NULL CHECK (status IN ('candidate','in_review','published','rejected','quarantined')),
    revision INTEGER NOT NULL CHECK (revision > 0),
    proposed_by TEXT NOT NULL CHECK (length(trim(proposed_by)) > 0),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE workspace_knowledge_candidate_sources (
    candidate_id TEXT NOT NULL,
    ordinal INTEGER NOT NULL CHECK (ordinal >= 0),
    source_type TEXT NOT NULL,
    source_id TEXT NOT NULL,
    source_revision TEXT NOT NULL,
    citation TEXT NOT NULL,
    asset_id TEXT,
    asset_version_id TEXT,
    PRIMARY KEY (candidate_id, ordinal),
    CHECK ((asset_id IS NULL) = (asset_version_id IS NULL))
);

CREATE TABLE workspace_knowledge_review_events (
    candidate_id TEXT NOT NULL,
    candidate_revision INTEGER NOT NULL CHECK (candidate_revision > 0),
    action TEXT NOT NULL CHECK (action IN ('approve','reject','quarantine','return','publish','supersede','invalidate')),
    actor_id TEXT NOT NULL,
    rationale TEXT NOT NULL,
    emergency INTEGER NOT NULL CHECK (emergency IN (0,1)),
    occurred_at TEXT NOT NULL,
    PRIMARY KEY (candidate_id, candidate_revision)
);

CREATE TABLE workspace_knowledge_publications (
    candidate_id TEXT PRIMARY KEY,
    knowledge_id TEXT,
    target_knowledge_id TEXT,
    action TEXT NOT NULL CHECK (action IN ('publish','supersede','invalidate')),
    created_at TEXT NOT NULL
);

CREATE INDEX workspace_knowledge_candidate_queue_idx
ON workspace_knowledge_candidates(workspace_id, status, updated_at DESC, id ASC);

CREATE INDEX workspace_knowledge_candidate_target_idx
ON workspace_knowledge_candidates(workspace_id, knowledge_id, target_revision);
