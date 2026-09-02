CREATE TABLE engineering_execution_items (
    workspace_id TEXT NOT NULL,
    id TEXT NOT NULL,
    kind TEXT NOT NULL,
    source_type TEXT NOT NULL,
    source_id TEXT NOT NULL,
    source_locator TEXT NOT NULL,
    created_at TEXT NOT NULL,
    PRIMARY KEY (workspace_id, id),
    UNIQUE (workspace_id, kind, source_type, source_id)
) WITHOUT ROWID;

CREATE TABLE engineering_evidence (
    workspace_id TEXT NOT NULL,
    id TEXT NOT NULL,
    schema_version TEXT NOT NULL,
    kind TEXT NOT NULL,
    outcome TEXT NOT NULL,
    source_type TEXT NOT NULL,
    source_id TEXT NOT NULL,
    source_locator TEXT NOT NULL,
    source_revision TEXT NOT NULL DEFAULT '',
    source_digest TEXT NOT NULL DEFAULT '',
    source_observed_at TEXT NOT NULL,
    producer_id TEXT NOT NULL,
    artifact_uri TEXT NOT NULL DEFAULT '',
    artifact_digest TEXT NOT NULL DEFAULT '',
    payload_json TEXT NOT NULL,
    captured_at TEXT NOT NULL,
    content_checksum TEXT NOT NULL,
    PRIMARY KEY (workspace_id, id)
) WITHOUT ROWID;

CREATE INDEX engineering_evidence_checksum_idx
    ON engineering_evidence(workspace_id, content_checksum);

CREATE TABLE engineering_execution_item_evidence (
    workspace_id TEXT NOT NULL,
    execution_item_id TEXT NOT NULL,
    evidence_id TEXT NOT NULL,
    attached_at TEXT NOT NULL,
    PRIMARY KEY (workspace_id, execution_item_id, evidence_id)
) WITHOUT ROWID;
