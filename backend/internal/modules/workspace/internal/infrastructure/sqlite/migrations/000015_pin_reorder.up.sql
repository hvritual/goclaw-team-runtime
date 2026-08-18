CREATE TABLE workspace_pin_order_revisions (
    workspace_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    revision INTEGER NOT NULL CHECK (revision >= 1),
    PRIMARY KEY (workspace_id, user_id)
);

INSERT INTO workspace_pin_order_revisions(workspace_id,user_id,revision)
SELECT DISTINCT workspace_id,user_id,1
FROM workspace_pins;

CREATE TRIGGER workspace_pins_order_revision_insert
AFTER INSERT ON workspace_pins
BEGIN
    INSERT INTO workspace_pin_order_revisions(workspace_id,user_id,revision)
    VALUES(NEW.workspace_id,NEW.user_id,1)
    ON CONFLICT(workspace_id,user_id)
    DO UPDATE SET revision=revision+1;
END;

CREATE TRIGGER workspace_pins_order_revision_delete
AFTER DELETE ON workspace_pins
BEGIN
    INSERT INTO workspace_pin_order_revisions(workspace_id,user_id,revision)
    VALUES(OLD.workspace_id,OLD.user_id,1)
    ON CONFLICT(workspace_id,user_id)
    DO UPDATE SET revision=revision+1;
END;
