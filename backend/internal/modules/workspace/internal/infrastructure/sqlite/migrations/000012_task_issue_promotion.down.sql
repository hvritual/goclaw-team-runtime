CREATE TEMP TABLE task_issue_promotion_rollback_guard (
    safe INTEGER NOT NULL CHECK (safe = 1)
);

INSERT INTO task_issue_promotion_rollback_guard(safe)
SELECT CASE WHEN EXISTS (
    SELECT 1 FROM workspace_task_issue_promotions
) THEN 0 ELSE 1 END;

DROP TABLE task_issue_promotion_rollback_guard;
DROP TABLE workspace_task_issue_promotions;

DELETE FROM workspace_schema_migrations
WHERE version = '000012_task_issue_promotion.up.sql';
