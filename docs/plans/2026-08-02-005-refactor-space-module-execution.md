---
title: Space Module Incremental Migration - Execution
type: refactor
date: 2026-08-02
topic: space-module-migration
artifact_contract: multica-ddd-execution/v1
execution_status: in-progress
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

**S1a completed 2026-08-02; S1 remains in progress:** `/api/upload-file` now enters `server/modules/space/interfaces/http`, invokes the Space upload application service, computes a SHA-256 checksum, constructs the workspace-scoped Asset domain object, and persists through adapters for the existing sqlc attachment query and local/S3 storage providers. The Issue-side upload workflow owns its port/DTO, checks Auth-side workspace membership before Issue visibility, then stores Issue relation and Asset metadata in one `CreateAttachment` statement; the composition root maps the Issue contract to Space through an anti-corruption adapter. The installed-client fallback that returns a direct URL when metadata persistence fails remains explicit; S1 cannot be marked complete until a durable intent/cleanup retry path replaces that unrecoverable edge. Checksum persistence and Asset versions remain deferred because the current schema has no such fields.

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

## S1a Path and Verification Record

```text
Legacy: internal/handler.UploadFile -> sqlc + storage.Storage
Target: modules/space/interfaces/http -> internal/integration/uploadhttp
                                     -> internal/issueguard/UploadWorkflow (consumer-owned port/DTO)
                                         -> issueguard/adapter/spaceacl -> application/UploadService -> domain/Asset
                                                                                               -> dependency/postgres
                                                                                               -> dependency/objectstorage
                                         -> internal/auth/WorkspaceMemberships
                                         -> AttachmentReferences -> atomic Issue relation + metadata insert
Composition: cmd/server/router.go + cmd/server/space.go
```

- Preserved authenticated `POST /api/upload-file`, workspace header/slug resolution, membership checks, optional Issue validation, 100 MB request cap, media sniffing overrides, storage keys, response JSON, signed/stable download URLs and personal avatar uploads.
- Reused the existing `attachment` table and sqlc output; no migration, foreign key, generated code or new storage provider was introduced.
- Added domain, application, Issue workflow, PostgreSQL contract and HTTP adapter tests under `server/modules/space` and `server/internal/issueguard`, including checksum, authorization order, malformed/personal `issue_id` compatibility, cross-workspace Issue denial, single SQL relation insert and URL compatibility cases.
- Added Space/Issue depguard rules plus pinned local/CI lint gates for the full Space module, composition root and repository-wide changed Go code; verified three negative probes: domain-to-pgx, application-to-dependency and interface-to-another-context imports all fail.

## Remaining S1 Work

- Persist an upload intent before object storage and add retryable orphan cleanup/finalization so a crash or metadata failure cannot leave an untracked object.
- Add provider contract tests for the intent/finalization state machine.
- Persist the computed checksum when the Asset-owned schema is introduced; add Asset Version only with the versioning slice.
