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

**S1 completed for the SQLite-native runtime on 2026-08-02:** the native
`internal/modules/space` implementation now owns upload intent, Asset and Asset
Version state. It persists the intent before object I/O, stores SHA-256 and
immutable version facts, finalizes metadata transactionally, and retains a
retryable cleanup state when object deletion fails. Startup and each subsequent
workspace upload reconcile incomplete intents. PostgreSQL still uses the
earlier compatibility module and remains a separate migration item.

### S2. Read and download authorization

- Centralize stable metadata and storage URL/proxy decisions without moving consumer-specific visibility rules.
- Require the consumer context to prove the caller may access the referenced business object.

**Completed for SQLite workspace Assets on 2026-08-02:** metadata and download
requests call the Space contract with an authenticated actor; Space asks the
Auth-supplied `WorkspaceAccess` port and hides missing and foreign Assets behind
the same not-found response. Raw workspace storage keys are never exposed by
the public object route. Personal direct objects retain the installed avatar
contract and are readable only under generated `users/{userUUID}/{objectUUID}`
keys.

### S3. Issue/comment attachment adapter

- Keep the attachment relation in Workspace/Issue.
- Replace copied storage metadata with an Asset reference only after response parity and cleanup behavior are proven.

**Issue upload/list slice completed for SQLite on 2026-08-02:** the consumer-owned
`issue_asset_refs` table stores only Asset identity and Issue relation facts.
Issue visibility and workspace membership are checked before relation creation,
and list responses resolve metadata only through the Space public contract.
Comment binding, deletion and unreferenced-Asset reclamation remain pending.

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

- Port the native intent/version/checksum lifecycle to the PostgreSQL runtime.
- Add explicit consumer deletion and unreferenced-Asset reclamation workflows;
  relation deletion must not rely on foreign keys or database cascades.
- Migrate comment, Project resource and Knowledge file relations to stable Asset
  IDs without moving their business meaning into Space.
- Move the custom multipart/download transport from the SQLite Chi compatibility
  router to the final Kratos runtime during transport cutover.

## SQLite-Native Composition and Evidence

```text
POST /api/upload-file
  -> internal/modules/space/interfaces/http.UploadHandler
  -> sqlitelocal.sqliteSpaceUploader (Issue consumer relation adapter)
  -> internal/modules/space/contract.AssetUploadService
  -> internal/modules/space/internal/application.AssetService
       -> Auth-backed WorkspaceAccess contract
       -> SQLite AssetRepository
       -> storage-backed ObjectStore

GET /api/attachments/{id}[/download]
  -> AssetHandler -> AssetService -> WorkspaceAccess + SQLite repository

GET /uploads/users/{userUUID}/{objectUUID.ext}
  -> PublicObjectHandler -> ObjectStore
```

- The module owns `space_assets`, `space_asset_versions` and
  `space_upload_intents`; the Issue consumer owns `issue_asset_refs`.
- No cross-context foreign key or cascade was introduced.
- Real SQLite tests cover successful finalization, rollback after finalization
  failure, deletion retry, crash-leftover reconciliation, workspace isolation,
  Issue relation/list behavior, authenticated download and personal avatar
  retrieval.
- The frontend upload boundary validates both workspace Asset and personal
  direct-object responses and rejects malformed 200 responses instead of
  reporting a false success.
