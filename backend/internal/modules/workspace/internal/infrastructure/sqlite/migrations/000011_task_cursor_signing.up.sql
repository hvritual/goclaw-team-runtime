CREATE TABLE workspace_runtime_secrets (
    name TEXT PRIMARY KEY,
    value BLOB NOT NULL CHECK (length(value) = 32)
);
