CREATE TABLE workspace_project_search_documents (
    project_id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    status TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX workspace_project_search_scope_idx
ON workspace_project_search_documents(workspace_id, status, updated_at DESC, project_id ASC);

CREATE INDEX workspace_project_search_title_idx
ON workspace_project_search_documents(workspace_id, title, project_id);

INSERT INTO workspace_project_search_documents(
    project_id, workspace_id, title, description, status, updated_at
)
SELECT id, workspace_id,
       goclaw_issue_search_normalize(name),
       goclaw_issue_search_normalize(COALESCE(description, '')),
       status, updated_at
FROM workspace_projects;

CREATE TRIGGER workspace_project_search_after_insert
AFTER INSERT ON workspace_projects
BEGIN
    INSERT INTO workspace_project_search_documents(
        project_id, workspace_id, title, description, status, updated_at
    ) VALUES (
        NEW.id, NEW.workspace_id,
        goclaw_issue_search_normalize(NEW.name),
        goclaw_issue_search_normalize(COALESCE(NEW.description, '')),
        NEW.status, NEW.updated_at
    );
END;

CREATE TRIGGER workspace_project_search_after_update
AFTER UPDATE OF workspace_id, name, description, status, updated_at
ON workspace_projects
BEGIN
    UPDATE workspace_project_search_documents
    SET workspace_id = NEW.workspace_id,
        title = goclaw_issue_search_normalize(NEW.name),
        description = goclaw_issue_search_normalize(COALESCE(NEW.description, '')),
        status = NEW.status,
        updated_at = NEW.updated_at
    WHERE project_id = OLD.id;
END;

CREATE TRIGGER workspace_project_search_after_delete
AFTER DELETE ON workspace_projects
BEGIN
    DELETE FROM workspace_project_search_documents WHERE project_id = OLD.id;
END;
