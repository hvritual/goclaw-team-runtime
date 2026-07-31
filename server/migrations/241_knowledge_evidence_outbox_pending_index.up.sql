CREATE INDEX CONCURRENTLY knowledge_evidence_outbox_pending_idx ON knowledge_evidence_outbox(available_at, created_at, id) WHERE delivered_at IS NULL;
