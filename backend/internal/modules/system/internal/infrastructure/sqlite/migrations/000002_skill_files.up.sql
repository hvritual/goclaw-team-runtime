CREATE TABLE IF NOT EXISTS system_skill_file_manifests (
    id TEXT PRIMARY KEY,
    skill_id TEXT NOT NULL,
    version_id TEXT NOT NULL,
    path TEXT NOT NULL,
    space_object_id TEXT NOT NULL,
    media_type TEXT NOT NULL,
    size_bytes INTEGER NOT NULL CHECK (size_bytes >= 0),
    checksum TEXT NOT NULL,
    created_at TEXT NOT NULL,
    UNIQUE(version_id, path)
);

CREATE TABLE IF NOT EXISTS system_skill_import_previews (
    token_hash TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    actor_id TEXT NOT NULL,
    validator_version TEXT NOT NULL,
    source_checksum TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS system_skill_import_idempotency (
    workspace_id TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    request_hash TEXT NOT NULL,
    response_json TEXT NOT NULL,
    created_at TEXT NOT NULL,
    PRIMARY KEY(workspace_id, idempotency_key)
);
