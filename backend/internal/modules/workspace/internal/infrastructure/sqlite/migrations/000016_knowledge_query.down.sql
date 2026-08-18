CREATE TEMP TABLE knowledge_query_rollback_guard (
    safe INTEGER NOT NULL CHECK (safe = 1)
);

INSERT INTO knowledge_query_rollback_guard(safe)
SELECT CASE WHEN
    EXISTS (SELECT 1 FROM workspace_governed_knowledge)
    OR EXISTS (SELECT 1 FROM workspace_knowledge_revisions)
    OR EXISTS (SELECT 1 FROM workspace_knowledge_source_refs)
THEN 0 ELSE 1 END;

DROP TABLE knowledge_query_rollback_guard;
DROP INDEX workspace_knowledge_source_lookup_idx;
DROP INDEX workspace_governed_knowledge_kind_idx;
DROP INDEX workspace_governed_knowledge_scope_idx;
DROP TABLE workspace_knowledge_source_refs;
DROP TABLE workspace_knowledge_revisions;
DROP TABLE workspace_governed_knowledge;

DELETE FROM workspace_schema_migrations
WHERE version = '000016_knowledge_query.up.sql';
