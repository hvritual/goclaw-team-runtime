DROP TRIGGER IF EXISTS workspace_project_search_after_delete;
DROP TRIGGER IF EXISTS workspace_project_search_after_update;
DROP TRIGGER IF EXISTS workspace_project_search_after_insert;
DROP TABLE workspace_project_search_documents;

DELETE FROM workspace_schema_migrations
WHERE version = '000014_project_search.up.sql';
