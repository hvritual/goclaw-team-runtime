CREATE TEMP TABLE workspace_governance_down_guard (
    allowed INTEGER NOT NULL CHECK (allowed = 1)
);

INSERT INTO workspace_governance_down_guard(allowed)
SELECT CASE WHEN
    NOT EXISTS (SELECT 1 FROM workspace_outbox_events) AND
    NOT EXISTS (SELECT 1 FROM workspace_audit_entries) AND
    NOT EXISTS (SELECT 1 FROM workspace_mutation_idempotency) AND
    NOT EXISTS (SELECT 1 FROM workspace_resource_revisions)
THEN 1 ELSE 0 END;

DROP TABLE workspace_outbox_events;
DROP TABLE workspace_audit_entries;
DROP TABLE workspace_mutation_idempotency;
DROP TABLE workspace_resource_revisions;

DELETE FROM workspace_schema_migrations
WHERE version = '000009_workspace_governance.up.sql';

DROP TABLE workspace_governance_down_guard;
