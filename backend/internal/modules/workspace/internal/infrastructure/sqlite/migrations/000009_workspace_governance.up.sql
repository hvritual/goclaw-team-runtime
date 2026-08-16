CREATE TABLE workspace_resource_revisions (
    workspace_id TEXT NOT NULL CHECK (length(trim(workspace_id)) > 0),
    resource_kind TEXT NOT NULL CHECK (length(trim(resource_kind)) > 0),
    resource_id TEXT NOT NULL CHECK (length(trim(resource_id)) > 0),
    revision INTEGER NOT NULL CHECK (revision >= 0),
    updated_at TEXT NOT NULL CHECK (length(trim(updated_at)) > 0),
    PRIMARY KEY (workspace_id, resource_kind, resource_id)
) WITHOUT ROWID;

CREATE TABLE workspace_mutation_idempotency (
    workspace_id TEXT NOT NULL CHECK (length(trim(workspace_id)) > 0),
    action TEXT NOT NULL CHECK (length(trim(action)) > 0),
    idempotency_key TEXT NOT NULL CHECK (length(trim(idempotency_key)) BETWEEN 1 AND 200),
    request_hash TEXT NOT NULL CHECK (length(request_hash) = 64),
    resource_kind TEXT NOT NULL CHECK (length(trim(resource_kind)) > 0),
    resource_id TEXT NOT NULL CHECK (length(trim(resource_id)) > 0),
    resource_revision INTEGER NOT NULL CHECK (resource_revision >= 1),
    response_status INTEGER NOT NULL CHECK (response_status BETWEEN 100 AND 599),
    response_body TEXT NOT NULL
        CHECK (json_valid(response_body) AND length(CAST(response_body AS BLOB)) <= 65536),
    created_at TEXT NOT NULL CHECK (length(trim(created_at)) > 0),
    expires_at TEXT,
    PRIMARY KEY (workspace_id, action, idempotency_key)
) WITHOUT ROWID;

CREATE TABLE workspace_audit_entries (
    workspace_id TEXT NOT NULL CHECK (length(trim(workspace_id)) > 0),
    occurred_at TEXT NOT NULL CHECK (length(trim(occurred_at)) > 0),
    id TEXT NOT NULL CHECK (length(trim(id)) > 0),
    actor_type TEXT NOT NULL CHECK (actor_type IN ('member','agent')),
    actor_id TEXT NOT NULL CHECK (length(trim(actor_id)) > 0),
    action TEXT NOT NULL CHECK (length(trim(action)) > 0),
    resource_kind TEXT NOT NULL CHECK (length(trim(resource_kind)) > 0),
    resource_id TEXT NOT NULL CHECK (length(trim(resource_id)) > 0),
    resource_revision INTEGER NOT NULL CHECK (resource_revision >= 1),
    request_id TEXT NOT NULL CHECK (length(trim(request_id)) > 0),
    metadata_json TEXT NOT NULL DEFAULT '{}'
        CHECK (json_valid(metadata_json) AND length(CAST(metadata_json AS BLOB)) <= 16384),
    PRIMARY KEY (workspace_id, occurred_at, id)
) WITHOUT ROWID;

CREATE TABLE workspace_outbox_events (
    state TEXT NOT NULL CHECK (state IN ('ready','inflight','retry_wait','delivered','dead_letter')),
    available_at TEXT NOT NULL CHECK (length(trim(available_at)) > 0),
    workspace_id TEXT NOT NULL CHECK (length(trim(workspace_id)) > 0),
    id TEXT NOT NULL CHECK (length(trim(id)) > 0),
    event_type TEXT NOT NULL CHECK (length(trim(event_type)) > 0),
    aggregate_kind TEXT NOT NULL CHECK (length(trim(aggregate_kind)) > 0),
    aggregate_id TEXT NOT NULL CHECK (length(trim(aggregate_id)) > 0),
    aggregate_revision INTEGER NOT NULL CHECK (aggregate_revision >= 1),
    payload_json TEXT NOT NULL
        CHECK (json_valid(payload_json) AND length(CAST(payload_json AS BLOB)) <= 65536),
    actor_type TEXT NOT NULL CHECK (actor_type IN ('member','agent')),
    actor_id TEXT NOT NULL CHECK (length(trim(actor_id)) > 0),
    attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
    claim_token TEXT,
    lease_expires_at TEXT,
    last_error_code TEXT,
    created_at TEXT NOT NULL CHECK (length(trim(created_at)) > 0),
    delivered_at TEXT,
    PRIMARY KEY (state, available_at, workspace_id, id),
    CHECK (
        (state = 'inflight' AND length(trim(claim_token)) > 0 AND lease_expires_at IS NOT NULL)
        OR (state <> 'inflight' AND claim_token IS NULL AND lease_expires_at IS NULL)
    )
) WITHOUT ROWID;
