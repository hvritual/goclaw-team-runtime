# P6-S2B1 Issue metadata KV migration

- Plan-ID: `server-function-migration`
- Version: `8`
- Status: `approved`
- Approval source: user instruction dated 2026-08-04 to create the next plan
  version and delegate P6-S2B1 implementation to a sidebar task
- Parent plan: `plan_v7.md`
- Active step: `P6-S2B1`
- Repository scope: `backend/` only

## Frozen task

- Task-ID: `P6-S2B1`
- Task-Revision: `1`
- Work-Item: `Issue metadata KV`
- Project: `goclaw`
- Project-ID:
  `c2xpbmdzaG90OmVudl9lXzZhMDg5ODhhYTc3NDgzMjQ5OTEwMTA4NWE2MDZiMzJiCi9Vc2Vycy9md29ybGQvSHZyaXR1YWwvZ29jbGF3`
- Repository: `/Users/fworld/Hvritual/goclaw`
- Assignee: one isolated sidebar implementation task; this control thread owns
  review and acceptance
- Git base: `0c4f848a8458e256dd4fe2ed51498af969aa3c59`
- Branch at freeze: `codex/multica-six-domain-baseline`
- Working-tree source: latest accepted P6-S2A baseline
- Product baseline digest for sorted files below `backend/api`, `backend/cmd`,
  `backend/internal`, `backend/rpc`, and `backend/tests`:
  `beb4a9b37bc07e78fbe9b89a3b09d82dc6499e50c30cc75776ed547cb2ab20bb`
- Policy bundle:
  `753606cfa0b1e99ecdeb7585ed8a541b19b1caed2f239f2f8a43f7b5ce7ab8cb`
  - root `AGENTS.md`:
    `3893c1ff20196c5ee3ee0ea4a0e5ba5d7430cdca218c56278816fc0aed83e933`
  - `backend/AGENTS.md`:
    `7e282916368103dd92973aef7310bb11978ae3dddf0f7db14ba626dbca2a4bb8`
  - root `CLAUDE.md`:
    `d58f641a5fa1942846045ca85634552508e59f1dc4c691142e77bf180ea35bc4`
- Generator source revision:
  `7266939f0e295648593739c694ab4e614b141546`
- Dependency: P6-S2A accepted in `journal.md`; no other Issue write task may
  run concurrently

The Git base does not contain the untracked backend migration tree. The content
digest and working-tree starting state are therefore part of the frozen task,
not optional diagnostic data.

## Goal

Migrate the installed server's Workspace-scoped, per-Issue metadata KV
behavior into the backend Workspace module as an independently verifiable
SQLite/local/gRPC slice. Mutations operate on exactly one key, preserve
concurrent writes to other keys, and never allow `UpdateIssue` to overwrite the
metadata bag.

## Current and target paths

```text
Current target backend:
IssueService -> IssueUseCase -> read full Issue -> SQLite Update rewrites
metadata/properties from the previously loaded projection

P6-S2B1 target:
IssueMetadata local/gRPC adapter -> application use case -> metadata value
rules -> Workspace-scoped atomic repository <- SQLite BEGIN IMMEDIATE
```

The installed HTTP adapter and PostgreSQL/sqlc implementation remain the
behavior source only. They are not modified or selected by this step.

## Scope

### Proto contract

Extend `backend/api/workspace/v1/issue.proto` additively with a second service,
`IssueMetadataService`, and exactly these RPCs:

1. `GetIssueMetadata`
2. `PutIssueMetadataKey`
3. `DeleteIssueMetadataKey`

The service uses Workspace module metadata and distinct access declarations:

- `workspace.issue.metadata.get`
- `workspace.issue.metadata.put`
- `workspace.issue.metadata.delete`

Mutation RPCs are audited. Do not add HTTP annotations in this step.

The request identity is `workspace_id` plus `issue_id`; `issue_id` accepts the
same UUID-or-Workspace-identifier form as Issue mainline. Put also carries a
key and `value_json`, where `value_json` is the UTF-8 representation of exactly
one JSON primitive. Using raw JSON text preserves number-versus-string typing
at storage boundaries. Delete carries only the key.

All three responses return an `IssueMetadataSnapshot` containing the canonical
Issue ID, the complete metadata object, and `updated_at`. A later compatibility
HTTP adapter may project this back to the installed `{\"metadata\": {...}}`
shape; P6-S2B1 does not implement that adapter.

Existing `Issue` fields retain their numbers, especially `metadata = 21`,
`properties = 22`, and `asset_ids = 23`. Existing services and RPCs are not
renamed or renumbered.

### Owned behavior

- Metadata is an Issue-owned, flat JSON object scoped by Workspace.
- Keys must match `^[a-zA-Z_][a-zA-Z0-9_.-]{0,63}$` and are 1-64 bytes in the
  accepted ASCII vocabulary.
- Values may be JSON string, number, or bool only.
- Empty input, invalid JSON, `null`, arrays, and objects are rejected with
  stable application/domain error classes.
- Each Issue may contain at most 50 keys. Overwriting an existing key at the
  limit succeeds; adding a 51st key fails.
- The persisted metadata representation is bounded to 8 KiB. SQLite measures
  the compact UTF-8 JSON representation. The later PostgreSQL provider must
  enforce its native `pg_column_size(jsonb) <= 8192` rule; exact representation
  boundary parity remains an explicit P6-S6/P6-S7 provider test concern.
- Put replaces exactly one key and returns the complete authoritative snapshot.
- Delete removes exactly one key, is a successful no-op when the key is absent,
  still refreshes `updated_at`, and returns the complete snapshot.
- Missing and foreign-Workspace Issues map to the same public not-found error.
- Malformed historical metadata degrades to an empty object on Get. A mutation
  starts from that empty object and writes a valid bounded object.
- Metadata maps returned across boundaries are defensive copies.

### Authorization, identity, and transaction rules

- Validate boundary syntax without accessing persistence, then authorize the
  Workspace before any repository call.
- Get requires Workspace access.
- Put/Delete additionally require a trusted Workspace Actor context and verify
  that the Member or Agent belongs to the same Workspace through the existing
  consumer-owned Auth port.
- Every read and write predicate includes `workspace_id`.
- Put/Delete use one SQLite-native `BEGIN IMMEDIATE` transaction to load the
  current bag, apply one mutation, enforce count and size, update metadata and
  `updated_at`, and read the canonical snapshot.
- Failure before commit rolls back metadata and timestamp together.
- Two concurrent writes to distinct keys must serialize without losing either
  key. Same-key writes are last-committer-wins according to transaction order.
- No event is published in this slice. The response contains canonical Issue,
  actor context remains available, and the complete snapshot is sufficient for
  the later durable realtime/outbox step.

### Mainline concurrency correction

Change the existing SQLite Issue mainline update so `UpdateIssue` no longer
writes the `metadata` or `properties` columns from a previously loaded Issue
projection. Neither field is mutable through `UpdateIssue`; retaining those
assignments would allow a concurrent metadata/property mutation to be lost.

`UpdateIssue` may continue reading both fields for its response, and asset
association behavior remains unchanged. Add a regression test that overlaps a
metadata Put with a mainline Issue update and proves both changes survive.

### Layers and composition

Implement only the necessary user-owned domain/application/provider files and
the dddgen-generated local/gRPC/Proto boundaries for
`workspace.v1.IssueMetadataService`.

- Domain metadata rules stay below
  `internal/modules/workspace/internal/domain/issue`.
- Application owns authorization, Actor validation, stable error mapping, and
  the transaction-capability port.
- SQLite infrastructure owns JSON persistence, `BEGIN IMMEDIATE`, row mapping,
  and rollback.
- Workspace composition replaces the generated metadata extension only in
  `NewWithSqliteWorkspaceChain`.
- `workspace.New()` and `internal/bootstrap.NewApplication()` remain generated
  stub composition.

The existing `workspace_issues.metadata` column is sufficient. No migration or
new table is expected. If implementation discovers that a schema change is
required, stop and return to the control thread for a new plan version.

## Non-goals

- Labels, label catalogs, label assignments, custom-property definitions,
  options, property values, or Skill labels.
- Metadata filters in List/Query/Table, search, grouping, facets, or indexes.
- Whole-bag metadata replacement through CreateIssue or UpdateIssue.
- DeleteIssue cleanup, batch, hierarchy, move, collaboration objects, assets,
  or acceptance conclusions.
- HTTP compatibility routes or JSON adapters.
- PostgreSQL provider/sqlc, realtime publishing, outbox, cache invalidation, or
  runtime cutover.
- Changes outside `backend/`, changes to installed `server/`, foreign keys,
  cascade actions, or production data.

## Ordered execution

### P6-S2B1-1 — Revalidate frozen task

Reread policies, this plan, and the journal; verify Git base, product baseline
digest, policy hashes, generator revision, and that no other Issue writer is
active. Stop on drift before writing product code.

### P6-S2B1-2 — Characterize contract and domain

Add focused tests for key syntax, primitive JSON typing, 50-key behavior, size
limit, defensive copies, stable errors, and malformed historical data. Extend
the Proto contract additively and run the established generation pipeline.

### P6-S2B1-3 — Application and SQLite transaction

Implement authorization-first orchestration and the atomic SQLite metadata
repository. Remove metadata/properties assignments from mainline Issue update.
Cover tenant hiding, Actor validation, rollback, two-key concurrency, same-key
ordering, delete no-op, timestamp updates, and UpdateIssue concurrency.

### P6-S2B1-4 — Local/gRPC composition and verification

Wire the generated metadata extension only into opt-in SQLite composition. Add
local and bufconn coverage for all three RPCs, rerun generation for content
idempotence, and execute every acceptance gate below.

### P6-S2B1-5 — Control-thread review

The sidebar task reports changed files, decisions, generated digests, commands,
and unresolved risks. It does not self-accept. This control thread independently
reviews and reruns deterministic gates before advancing to P6-S2B2.

## Acceptance criteria

1. Actual changes stay inside `backend/` and P6-S2B1 scope.
2. Proto evolution is additive, existing field/RPC numbers are unchanged, and
   generated files are changed only by dddgen/Buf plus the approved import
   postprocessor and gofmt pipeline.
3. Get/Put/Delete work through domain, application, SQLite, local, and gRPC
   boundaries with stable Workspace-scoped not-found behavior.
4. Key, value, 50-key, and 8-KiB rules match this frozen contract; invalid
   mutations change neither metadata nor `updated_at`.
5. Authorization precedes repository access; mutation Actor identity is trusted
   and same-Workspace validated.
6. SQLite mutations are atomic under failure and concurrent different-key
   writes never lose data.
7. `UpdateIssue` cannot overwrite concurrent metadata or properties, and its
   existing mainline fields retain accepted behavior.
8. No schema migration, FK, cascade, cross-module table read, HTTP,
   PostgreSQL, realtime, or default-runtime selection appears.
9. Domain/application/provider/local/bufconn tests cover the declared matrix.
10. Full deterministic verification passes and the running default backend's
    health/Ping behavior remains unchanged.

## Deterministic verification

From `backend/`, with the frozen dddgen source and exact temporary toolchain:

```sh
buf format --diff --exit-code
buf lint
dddgen -root . proto-service workspace.v1.IssueMetadataService
go run ./cmd/postprocess-dddgen-imports
buf generate
gofmt -w <changed-go-files>
# Repeat the preceding generation pipeline and require identical content.
go test -race ./internal/modules/workspace/... -count=1
go test ./tests/contract/... -count=1
go test ./... -count=1
go vet ./...
go mod verify
```

Also require deterministic static checks for:

- gofmt cleanliness and `git diff --check`;
- no old `github.com/hvritual/workspace/gen/go` imports in live code;
- no forbidden domain/application or cross-module internal imports;
- no FK/cascade SQL;
- exactly scoped Proto RPCs and generated outputs;
- unchanged default bootstrap selection;
- generated file count/digest and complete-pipeline idempotence;
- HTTP health/readiness, live gRPC health, and Workspace Ping when the shared
  backend process is available.

## Risks and mitigations

- **Untracked baseline:** freeze working-tree content digest and create the
  sidebar task from `working-tree`, not Git base alone.
- **Absolute-path leakage:** the sidebar task must treat its assigned cwd as the
  repository root and must never write the main checkout's absolute path.
- **Read-modify-write loss:** require `BEGIN IMMEDIATE` and explicit concurrent
  two-key plus UpdateIssue tests.
- **Numeric fidelity:** accept raw `value_json`, validate with `json.Number`, and
  do not normalize a number into a quoted string. Existing Struct projections
  remain a transport limitation to revisit with HTTP/PostgreSQL parity.
- **8-KiB dialect difference:** freeze SQLite compact-JSON measurement now and
  keep PostgreSQL native-size parity as an explicit later provider gate.
- **Generated shared files:** no other Issue write task may run concurrently;
  reconcile only `IssueMetadataService` and inspect module-wide messages.
- **Runtime safety:** opt-in composition only; default stubs make rollback a
  code-only operation with no production data impact.

## Rollback and stop conditions

Rollback removes the additive metadata service/generated boundaries and
user-owned metadata behavior, then restores the accepted P6-S2A product
baseline. No data or schema rollback is needed because this step has no
migration and no runtime cutover.

Stop and return to the control thread if:

- frozen policy, baseline digest, generator revision, or active-writer state
  differs;
- installed ownership or key/value semantics cannot be reconciled;
- a schema migration, PostgreSQL/HTTP/realtime change, or cross-module table
  access appears necessary;
- generated reconciliation changes unrelated services or loses user-owned
  application bodies;
- concurrent mutation cannot be made atomic without expanding the transaction
  boundary.

