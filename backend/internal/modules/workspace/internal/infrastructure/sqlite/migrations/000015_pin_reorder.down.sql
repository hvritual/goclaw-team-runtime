CREATE TEMP TABLE pin_reorder_rollback_guard (
    safe INTEGER NOT NULL CHECK (safe = 1)
);

INSERT INTO pin_reorder_rollback_guard(safe)
SELECT CASE WHEN EXISTS (
    SELECT 1 FROM workspace_pin_order_revisions WHERE revision > 1
) THEN 0 ELSE 1 END;

DROP TABLE pin_reorder_rollback_guard;
DROP TRIGGER workspace_pins_order_revision_delete;
DROP TRIGGER workspace_pins_order_revision_insert;
DROP TABLE workspace_pin_order_revisions;

DELETE FROM workspace_schema_migrations
WHERE version = '000015_pin_reorder.up.sql';
