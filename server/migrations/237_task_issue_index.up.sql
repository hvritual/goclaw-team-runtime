CREATE INDEX CONCURRENTLY task_issue_id_idx ON task (issue_id) WHERE issue_id IS NOT NULL;
