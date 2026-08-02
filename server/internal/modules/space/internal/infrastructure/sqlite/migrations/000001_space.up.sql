CREATE TABLE IF NOT EXISTS space_assets (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    current_version_id TEXT NOT NULL,
    uploader_type TEXT NOT NULL,
    uploader_id TEXT NOT NULL,
    filename TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS space_assets_workspace_idx
    ON space_assets(workspace_id, created_at);

CREATE TABLE IF NOT EXISTS space_asset_versions (
    id TEXT PRIMARY KEY,
    asset_id TEXT NOT NULL,
    object_key TEXT NOT NULL UNIQUE,
    object_url TEXT NOT NULL,
    media_type TEXT NOT NULL,
    size_bytes INTEGER NOT NULL,
    checksum TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS space_asset_versions_asset_idx
    ON space_asset_versions(asset_id, created_at);

CREATE TABLE IF NOT EXISTS space_upload_intents (
    id TEXT PRIMARY KEY,
    asset_id TEXT NOT NULL UNIQUE,
    version_id TEXT NOT NULL UNIQUE,
    workspace_id TEXT NOT NULL,
    uploader_type TEXT NOT NULL,
    uploader_id TEXT NOT NULL,
    filename TEXT NOT NULL,
    object_key TEXT NOT NULL UNIQUE,
    media_type TEXT NOT NULL,
    size_bytes INTEGER NOT NULL,
    checksum TEXT NOT NULL,
    state TEXT NOT NULL,
    last_error TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS space_upload_intents_state_idx
    ON space_upload_intents(state, updated_at);
