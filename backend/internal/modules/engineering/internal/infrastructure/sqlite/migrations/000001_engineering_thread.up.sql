CREATE TABLE engineering_entities (
    workspace_id TEXT NOT NULL,
    id TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    name TEXT NOT NULL,
    status TEXT NOT NULL,
    owner_ref TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (workspace_id, id)
) WITHOUT ROWID;

CREATE TABLE engineering_source_bindings (
    workspace_id TEXT NOT NULL,
    id TEXT NOT NULL,
    entity_id TEXT NOT NULL,
    source_type TEXT NOT NULL,
    source_locator TEXT NOT NULL,
    source_revision TEXT NOT NULL DEFAULT '',
    observed_at TEXT NOT NULL,
    authority TEXT NOT NULL,
    PRIMARY KEY (workspace_id, id)
) WITHOUT ROWID;

CREATE TABLE engineering_thread_edges (
    workspace_id TEXT NOT NULL,
    id TEXT NOT NULL,
    from_kind TEXT NOT NULL,
    from_id TEXT NOT NULL,
    relation_type TEXT NOT NULL,
    to_kind TEXT NOT NULL,
    to_id TEXT NOT NULL,
    authority TEXT NOT NULL,
    source_type TEXT NOT NULL,
    source_locator TEXT NOT NULL,
    source_revision TEXT NOT NULL DEFAULT '',
    observed_at TEXT NOT NULL,
    PRIMARY KEY (workspace_id, id)
) WITHOUT ROWID;

CREATE TABLE engineering_changes (
    workspace_id TEXT NOT NULL,
    id TEXT NOT NULL,
    project_id TEXT NOT NULL DEFAULT '',
    requirement_id TEXT NOT NULL DEFAULT '',
    work_item_kind TEXT,
    work_item_id TEXT,
    run_id TEXT NOT NULL DEFAULT '',
    summary TEXT NOT NULL,
    status TEXT NOT NULL,
    source_type TEXT NOT NULL,
    source_locator TEXT NOT NULL,
    source_revision TEXT NOT NULL DEFAULT '',
    observed_at TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    accepted_at TEXT,
    PRIMARY KEY (workspace_id, id)
) WITHOUT ROWID;

CREATE TABLE engineering_change_entities (
    workspace_id TEXT NOT NULL,
    change_id TEXT NOT NULL,
    ordinal INTEGER NOT NULL,
    entity_id TEXT NOT NULL,
    PRIMARY KEY (workspace_id, change_id, ordinal)
) WITHOUT ROWID;

CREATE TABLE engineering_change_artifacts (
    workspace_id TEXT NOT NULL,
    change_id TEXT NOT NULL,
    ordinal INTEGER NOT NULL,
    artifact_kind TEXT NOT NULL,
    locator TEXT NOT NULL,
    revision TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (workspace_id, change_id, ordinal)
) WITHOUT ROWID;

CREATE TABLE engineering_context_packs (
    workspace_id TEXT NOT NULL,
    id TEXT NOT NULL,
    work_item_kind TEXT NOT NULL,
    work_item_id TEXT NOT NULL,
    work_item_revision TEXT NOT NULL,
    policy_version TEXT NOT NULL,
    checksum TEXT NOT NULL,
    created_at TEXT NOT NULL,
    PRIMARY KEY (workspace_id, id)
) WITHOUT ROWID;

CREATE TABLE engineering_context_pack_targets (
    workspace_id TEXT NOT NULL,
    context_pack_id TEXT NOT NULL,
    ordinal INTEGER NOT NULL,
    entity_id TEXT NOT NULL,
    PRIMARY KEY (workspace_id, context_pack_id, ordinal)
) WITHOUT ROWID;

CREATE TABLE engineering_context_pack_references (
    workspace_id TEXT NOT NULL,
    context_pack_id TEXT NOT NULL,
    ordinal INTEGER NOT NULL,
    context_kind TEXT NOT NULL,
    reference_id TEXT NOT NULL,
    revision TEXT NOT NULL,
    checksum TEXT NOT NULL,
    PRIMARY KEY (workspace_id, context_pack_id, ordinal)
) WITHOUT ROWID;
