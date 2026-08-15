CREATE TABLE IF NOT EXISTS space_assets (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    current_version_id TEXT NOT NULL UNIQUE,
    uploader_type TEXT NOT NULL CHECK (uploader_type IN ('member', 'agent')),
    uploader_id TEXT NOT NULL,
    filename TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS space_asset_versions (
    id TEXT PRIMARY KEY,
    asset_id TEXT NOT NULL UNIQUE,
    object_key TEXT NOT NULL UNIQUE,
    media_type TEXT NOT NULL,
    size_bytes INTEGER NOT NULL CHECK (size_bytes >= 0),
    checksum TEXT NOT NULL,
    created_at TEXT NOT NULL
);
