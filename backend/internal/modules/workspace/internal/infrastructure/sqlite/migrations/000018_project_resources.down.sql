CREATE TEMP TABLE project_resources_rollback_guard (
    safe INTEGER NOT NULL CHECK (safe = 1)
);

INSERT INTO project_resources_rollback_guard(safe)
SELECT CASE WHEN
    EXISTS (SELECT 1 FROM workspace_project_resources)
    OR EXISTS (SELECT 1 FROM workspace_project_resource_sets)
    OR EXISTS (
        SELECT 1 FROM workspace_audit_entries
        WHERE resource_kind = 'project_resource'
           OR action LIKE 'workspace.project.resource.%'
    )
    OR EXISTS (
        SELECT 1 FROM workspace_mutation_idempotency
        WHERE resource_kind = 'project_resource'
           OR action LIKE 'workspace.project.resource.%'
    )
THEN 0 ELSE 1 END;

DROP TABLE project_resources_rollback_guard;
DROP TABLE workspace_project_resources;
DROP TABLE workspace_project_resource_sets;

DELETE FROM workspace_schema_migrations
WHERE version = '000018_project_resources.up.sql';
