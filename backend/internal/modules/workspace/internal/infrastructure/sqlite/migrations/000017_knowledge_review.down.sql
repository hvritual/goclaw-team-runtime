CREATE TEMP TABLE knowledge_review_rollback_guard (
    safe INTEGER NOT NULL CHECK (safe = 1)
);

INSERT INTO knowledge_review_rollback_guard(safe)
SELECT CASE WHEN
    EXISTS (SELECT 1 FROM workspace_knowledge_candidates)
    OR EXISTS (SELECT 1 FROM workspace_knowledge_candidate_sources)
    OR EXISTS (SELECT 1 FROM workspace_knowledge_review_events)
    OR EXISTS (SELECT 1 FROM workspace_knowledge_publications)
    OR EXISTS (SELECT 1 FROM workspace_governed_knowledge WHERE status = 'invalidated')
    OR EXISTS (SELECT 1 FROM workspace_audit_entries WHERE action LIKE 'workspace.knowledge.%')
    OR EXISTS (SELECT 1 FROM workspace_mutation_idempotency WHERE action = 'workspace.knowledge.propose')
THEN 0 ELSE 1 END;

DROP TABLE knowledge_review_rollback_guard;
DROP INDEX workspace_knowledge_candidate_target_idx;
DROP INDEX workspace_knowledge_candidate_queue_idx;
DROP TABLE workspace_knowledge_publications;
DROP TABLE workspace_knowledge_review_events;
DROP TABLE workspace_knowledge_candidate_sources;
DROP TABLE workspace_knowledge_candidates;

ALTER TABLE workspace_governed_knowledge RENAME TO workspace_governed_knowledge_v17;

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

INSERT INTO workspace_governed_knowledge(id,workspace_id,project_id,candidate_id,kind,status,current_revision,created_at,updated_at)
SELECT id,workspace_id,project_id,candidate_id,kind,status,current_revision,created_at,updated_at
FROM workspace_governed_knowledge_v17;

DROP TABLE workspace_governed_knowledge_v17;

CREATE INDEX workspace_governed_knowledge_scope_idx
ON workspace_governed_knowledge(workspace_id, status, updated_at DESC, id ASC);

CREATE INDEX workspace_governed_knowledge_kind_idx
ON workspace_governed_knowledge(workspace_id, kind, updated_at DESC, id ASC);

DELETE FROM workspace_schema_migrations
WHERE version = '000017_knowledge_review.up.sql';
