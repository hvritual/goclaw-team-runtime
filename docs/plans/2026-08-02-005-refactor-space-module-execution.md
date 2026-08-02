---
title: Space Module Incremental Migration - Execution
type: refactor
date: 2026-08-02
topic: space-module-migration
artifact_contract: multica-ddd-execution/v1
execution_status: planned
depends_on:
  - four-module-proto-foundation
---

# Space Module Incremental Migration - Execution

## Outcome

Create a workspace-isolated Asset lifecycle without moving Project, Issue or Knowledge business relationships into Space.

## Boundary

- **Owned:** Asset ID, Workspace ID, content/media metadata, checksum, size, storage object key, versions, upload/finalization state and access lifecycle.
- **Not owned:** Issue attachment meaning, comment attachment meaning, Project resource meaning, Knowledge evidence/revision meaning, avatar ownership rules.
- **Contract:** consumers receive stable Asset IDs and safe access results; they never consume storage-driver rows.

## Tracer-Slice Order

### S1. Upload and finalize one Asset

- Characterize current upload authorization, storage adapter, size/media checks and failure cleanup.
- Introduce an application use case with storage, identity, checksum and repository ports.
- Return an Asset ID only after durable metadata and storage state meet the existing safety rule.

### S2. Read and download authorization

- Centralize stable metadata and storage URL/proxy decisions without moving consumer-specific visibility rules.
- Require the consumer context to prove the caller may access the referenced business object.

### S3. Issue/comment attachment adapter

- Keep the attachment relation in Workspace/Issue.
- Replace copied storage metadata with an Asset reference only after response parity and cleanup behavior are proven.

### S4. Project resource and Knowledge source adapters

- Project keeps resource type, label, position and business reference semantics.
- Knowledge keeps evidence provenance and revisions.
- Space supplies only Asset lifecycle operations for file-backed resources.

## Data and Consistency Rules

- Every Asset is scoped by `workspace_id`.
- No cross-context foreign keys or cascade actions.
- Consumer deletion and Asset reclamation require an explicit application workflow; an unreferenced Asset is not deleted by database cascade.
- Object-store cleanup must have a retryable safe path; do not claim atomicity across PostgreSQL/SQLite and external storage.

## Verification

- Upload limits, media validation, checksum and unsafe-key tests.
- Workspace authorization and cross-workspace denial tests.
- Storage adapter contract tests and failure cleanup/retry tests.
- Existing attachment URL/download behavior tests.
- No Project/Issue/Knowledge implementation imports inside Space domain/application.

## Stop Conditions

- Asset ownership would require Space to decide Project, Issue or Knowledge business rules.
- Storage cleanup could lose an object or metadata without a recoverable state.
- Stable API response behavior cannot be preserved.
