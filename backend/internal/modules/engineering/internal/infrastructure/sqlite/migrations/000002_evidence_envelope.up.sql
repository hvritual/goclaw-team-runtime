CREATE TABLE engineering_evidence_envelopes (
    workspace_id TEXT NOT NULL,
    id TEXT NOT NULL,
    kind TEXT NOT NULL,
    subject_kind TEXT NOT NULL,
    subject_id TEXT NOT NULL,
    source_type TEXT NOT NULL,
    source_locator TEXT NOT NULL,
    source_revision TEXT NOT NULL DEFAULT '',
    source_digest TEXT NOT NULL DEFAULT '',
    source_observed_at TEXT NOT NULL,
    producer_id TEXT NOT NULL,
    artifact_uri TEXT NOT NULL DEFAULT '',
    artifact_digest TEXT NOT NULL DEFAULT '',
    captured_at TEXT NOT NULL,
    content_checksum TEXT NOT NULL,
    PRIMARY KEY (workspace_id, id)
);

CREATE INDEX engineering_evidence_subject_idx
    ON engineering_evidence_envelopes(workspace_id, subject_kind, subject_id, id);

CREATE INDEX engineering_evidence_source_idx
    ON engineering_evidence_envelopes(workspace_id, source_type, source_locator, source_revision, id);
