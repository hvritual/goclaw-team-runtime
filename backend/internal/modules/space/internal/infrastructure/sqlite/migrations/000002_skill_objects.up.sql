CREATE TABLE IF NOT EXISTS space_skill_objects (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    object_key TEXT NOT NULL UNIQUE,
    media_type TEXT NOT NULL,
    size_bytes INTEGER NOT NULL CHECK (size_bytes >= 0),
    checksum TEXT NOT NULL,
    content BLOB NOT NULL,
    state TEXT NOT NULL CHECK (state IN ('quarantined', 'committed')),
    created_at TEXT NOT NULL,
    committed_at TEXT
);
