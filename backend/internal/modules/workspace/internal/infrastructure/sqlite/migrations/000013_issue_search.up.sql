CREATE TABLE workspace_issue_search_documents (
    issue_id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    identifier TEXT NOT NULL,
    number INTEGER NOT NULL,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    status TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX workspace_issue_search_scope_idx
ON workspace_issue_search_documents(workspace_id, status, updated_at DESC, issue_id ASC);

CREATE INDEX workspace_issue_search_identifier_idx
ON workspace_issue_search_documents(workspace_id, identifier, issue_id);

INSERT INTO workspace_issue_search_documents(
    issue_id, workspace_id, identifier, number, title, description, status, updated_at
)
SELECT id, workspace_id,
       goclaw_issue_search_normalize(identifier), number,
       goclaw_issue_search_normalize(title),
       goclaw_issue_search_normalize(COALESCE(description, '')),
       status, updated_at
FROM workspace_issues;

CREATE TRIGGER workspace_issue_search_after_insert
AFTER INSERT ON workspace_issues
BEGIN
    INSERT INTO workspace_issue_search_documents(
        issue_id, workspace_id, identifier, number, title, description, status, updated_at
    ) VALUES (
        NEW.id, NEW.workspace_id,
        goclaw_issue_search_normalize(NEW.identifier), NEW.number,
        goclaw_issue_search_normalize(NEW.title),
        goclaw_issue_search_normalize(COALESCE(NEW.description, '')),
        NEW.status, NEW.updated_at
    );
END;

CREATE TRIGGER workspace_issue_search_after_update
AFTER UPDATE OF workspace_id, identifier, number, title, description, status, updated_at
ON workspace_issues
BEGIN
    UPDATE workspace_issue_search_documents
    SET workspace_id = NEW.workspace_id,
        identifier = goclaw_issue_search_normalize(NEW.identifier),
        number = NEW.number,
        title = goclaw_issue_search_normalize(NEW.title),
        description = goclaw_issue_search_normalize(COALESCE(NEW.description, '')),
        status = NEW.status,
        updated_at = NEW.updated_at
    WHERE issue_id = OLD.id;
END;

CREATE TRIGGER workspace_issue_search_after_delete
AFTER DELETE ON workspace_issues
BEGIN
    DELETE FROM workspace_issue_search_documents WHERE issue_id = OLD.id;
END;
