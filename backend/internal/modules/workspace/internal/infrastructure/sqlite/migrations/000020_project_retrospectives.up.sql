CREATE TABLE workspace_project_retrospectives (
    workspace_id TEXT NOT NULL CHECK (length(trim(workspace_id)) > 0),
    project_id TEXT NOT NULL CHECK (length(trim(project_id)) > 0),
    id TEXT NOT NULL CHECK (length(trim(id)) > 0),
    status TEXT NOT NULL CHECK (status IN ('draft','published','archived')),
    current_revision INTEGER NOT NULL CHECK (current_revision >= 1),
    published_revision INTEGER CHECK (
        published_revision IS NULL OR
        (published_revision >= 1 AND published_revision <= current_revision)
    ),
    created_by TEXT NOT NULL CHECK (length(trim(created_by)) > 0),
    created_at TEXT NOT NULL CHECK (length(trim(created_at)) > 0),
    updated_at TEXT NOT NULL CHECK (length(trim(updated_at)) > 0),
    PRIMARY KEY (workspace_id, project_id, id),
    CHECK (status <> 'published' OR published_revision IS NOT NULL)
) WITHOUT ROWID;

CREATE TABLE workspace_project_retrospective_revisions (
    workspace_id TEXT NOT NULL CHECK (length(trim(workspace_id)) > 0),
    project_id TEXT NOT NULL CHECK (length(trim(project_id)) > 0),
    retrospective_id TEXT NOT NULL CHECK (length(trim(retrospective_id)) > 0),
    revision INTEGER NOT NULL CHECK (revision >= 1),
    lifecycle_status TEXT NOT NULL CHECK (lifecycle_status IN ('draft','published','archived')),
    action TEXT NOT NULL CHECK (action IN ('create','save_draft','publish','publish_revision','archive')),
    content_json TEXT NOT NULL CHECK (json_valid(content_json) AND length(CAST(content_json AS BLOB)) <= 131072),
    actor_id TEXT NOT NULL CHECK (length(trim(actor_id)) > 0),
    created_at TEXT NOT NULL CHECK (length(trim(created_at)) > 0),
    PRIMARY KEY (workspace_id, project_id, retrospective_id, revision)
) WITHOUT ROWID;

CREATE TABLE workspace_project_retrospective_participants (
    workspace_id TEXT NOT NULL CHECK (length(trim(workspace_id)) > 0),
    project_id TEXT NOT NULL CHECK (length(trim(project_id)) > 0),
    retrospective_id TEXT NOT NULL CHECK (length(trim(retrospective_id)) > 0),
    revision INTEGER NOT NULL CHECK (revision >= 1),
    member_id TEXT NOT NULL CHECK (length(trim(member_id)) > 0),
    role TEXT NOT NULL CHECK (role IN ('participant','facilitator')),
    PRIMARY KEY (workspace_id, project_id, retrospective_id, revision, member_id)
) WITHOUT ROWID;

CREATE TABLE workspace_project_retrospective_action_links (
    workspace_id TEXT NOT NULL CHECK (length(trim(workspace_id)) > 0),
    project_id TEXT NOT NULL CHECK (length(trim(project_id)) > 0),
    retrospective_id TEXT NOT NULL CHECK (length(trim(retrospective_id)) > 0),
    action_item_id TEXT NOT NULL CHECK (length(trim(action_item_id)) BETWEEN 1 AND 64),
    source_revision INTEGER NOT NULL CHECK (source_revision >= 1),
    state TEXT NOT NULL CHECK (state IN ('pending','linked')),
    target_kind TEXT NOT NULL CHECK (target_kind IN ('task','issue')),
    target_id TEXT,
    request_hash TEXT NOT NULL CHECK (length(request_hash) = 64),
    claimed_by TEXT NOT NULL CHECK (length(trim(claimed_by)) > 0),
    claimed_at TEXT NOT NULL CHECK (length(trim(claimed_at)) > 0),
    linked_by TEXT,
    linked_at TEXT,
    PRIMARY KEY (workspace_id, project_id, retrospective_id, action_item_id),
    CHECK (
        (state = 'pending' AND target_id IS NULL AND linked_by IS NULL AND linked_at IS NULL) OR
        (state = 'linked' AND length(trim(target_id)) > 0 AND length(trim(linked_by)) > 0 AND length(trim(linked_at)) > 0)
    )
) WITHOUT ROWID;
