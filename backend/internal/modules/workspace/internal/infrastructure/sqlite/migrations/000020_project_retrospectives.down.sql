CREATE TEMP TABLE project_retrospective_rollback_guard (
    safe INTEGER NOT NULL CHECK (safe = 1)
);

INSERT INTO project_retrospective_rollback_guard(safe)
SELECT CASE WHEN
    EXISTS (SELECT 1 FROM workspace_project_retrospective_action_links)
    OR EXISTS (SELECT 1 FROM workspace_project_retrospective_participants)
    OR EXISTS (SELECT 1 FROM workspace_project_retrospective_revisions)
    OR EXISTS (SELECT 1 FROM workspace_project_retrospectives)
    OR EXISTS (
        SELECT 1 FROM workspace_resource_revisions
        WHERE resource_kind IN ('project_retrospective','retrospective_action_item')
    )
    OR EXISTS (
        SELECT 1 FROM workspace_audit_entries
        WHERE resource_kind IN ('project_retrospective','retrospective_action_item')
           OR action LIKE 'workspace.project.retrospective.%'
    )
    OR EXISTS (
        SELECT 1 FROM workspace_mutation_idempotency
        WHERE resource_kind IN ('project_retrospective','retrospective_action_item')
           OR action LIKE 'workspace.project.retrospective.%'
    )
    OR EXISTS (
        SELECT 1 FROM workspace_outbox_events
        WHERE aggregate_kind IN ('project_retrospective','retrospective_action_item')
           OR event_type LIKE 'retrospective:%'
    )
THEN 0 ELSE 1 END;

DROP TABLE project_retrospective_rollback_guard;
DROP TABLE workspace_project_retrospective_action_links;
DROP TABLE workspace_project_retrospective_participants;
DROP TABLE workspace_project_retrospective_revisions;
DROP TABLE workspace_project_retrospectives;

DELETE FROM workspace_schema_migrations
WHERE version = '000020_project_retrospectives.up.sql';
