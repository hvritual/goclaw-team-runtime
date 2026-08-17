DROP TABLE workspace_runtime_secrets;

DELETE FROM workspace_schema_migrations
WHERE version = '000011_task_cursor_signing.up.sql';
