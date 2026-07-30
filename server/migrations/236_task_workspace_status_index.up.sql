CREATE INDEX CONCURRENTLY task_workspace_status_position_idx ON task (workspace_id, status, position, created_at DESC);
