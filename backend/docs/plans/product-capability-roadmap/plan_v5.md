# Product capability roadmap — execution plan v5

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Version: `5`
- Status: `approved-for-execution`
- Approved by: `Human Customer, 2026-08-17`
- Base commit: `0218ecbe5457f1afb716780ad44306e5b1b3b075`
- Active step: `PCR-S01B-5`
- Task revision: `r010`
- Supersedes for future execution: `plan_v4.md`
- Product contract: unchanged `PCR-CONTRACT-v1` and
  [s01b-foundation-design.md](s01b-foundation-design.md)

## 1. Objective

Repair every frozen-contract violation found by the independent S01B-4 review,
then re-run Release 0 evidence and independent review. The repair must complete
the existing authority and safety foundation without activating or implementing
any Release 1 capability.

## 2. Scope

### Included

- fail-closed server-side action/resource/actor policy resolution;
- application-derived versioned canonical request hashing;
- versioned, exact-schema, bounded, secret-free request/replay/audit/outbox
  envelopes;
- fail-closed handling for retained legacy unversioned replay/outbox rows;
- full outbox claim identity and transition-time lease checks;
- test-only empty-table guard for migration 000009 down execution;
- focused RED/GREEN tests, compatibility, full Backend, race, policy, scope,
  independent review, and Release 0 evidence.

### Excluded

- cleanup, rewriting, migration, replay, publication, or deletion of non-empty
  legacy governance rows;
- new tables, columns, indexes, triggers, up migrations, Proto/OpenAPI, public
  HTTP routes, frontend, Desktop, Control Plane, or Release 1 capability code;
- retrofitting existing Issue, project, pin, attachment, or realtime mutation
  repositories;
- changes below `server/**` or cleanup of unrelated dirty paths;
- deployment, push, merge, pull request, or production data operation.

## 3. Frozen design decisions

### 3.1 Authority and action policy

1. Governance preparation requires one installed server-side policy provider.
   Missing provider, unknown action, unknown resource kind, or a mismatched
   action/resource pair returns a fail-closed error.
2. The provider resolves the actor from trusted server context and Workspace
   membership/agent authority. Caller-supplied actor type or ID is never an
   authorization source.
3. Canonical member actor identity is the authenticated user ID; canonical agent
   identity is the server-resolved agent ID. Role is authorization input, not
   persisted actor identity.
4. Each installed action policy declares exactly one resource kind plus request,
   replay, audit, and event schemas. Merely naming a roadmap permission does not
   install its governance policy.
5. `PreparedGovernanceMutation` becomes opaque outside the application package;
   only successful policy resolution can construct a valid value. Repository
   execution revalidates its sealed invariants and cannot accept a forged field
   aggregate.

### 3.2 Canonical request hashing

1. Callers provide normalized request fields and an idempotency key, never a
   request digest.
2. Policy schemas allow only exact top-level scalar fields with explicit kinds:
   bounded identifier, fixed enum, boolean, non-negative integer, or lowercase
   SHA-256 content hash. Unknown fields, nested raw JSON, arrays, floats,
   duplicate keys, unconstrained strings, and invalid values fail closed.
3. Application code produces canonical JSON with sorted field names and stable
   scalar encoding. The hash preimage is the unambiguous JSON object:

   ```json
   {"version":"governance-request-v1","action":"<action>","request":{}}
   ```

4. The lowercase SHA-256 of the UTF-8 canonical bytes is the only persisted and
   compared request hash. Contract version, action, or result-affecting field
   changes therefore change the digest.

### 3.3 Safe persisted envelopes

1. Replay, audit, and outbox JSON use exactly two top-level keys:
   `version` and `data`. Versions are `governance-replay-v1`,
   `governance-audit-v1`, and `governance-outbox-v1` respectively.
2. `data` is validated against the installed action/event policy using the same
   strict scalar field kinds. No unrestricted free-text or raw body field is
   permitted.
3. A universal guard additionally rejects secret-bearing keys and values,
   including credential/password/secret/token/cookie/authorization/API-key
   names, bearer/cookie/basic authorization material, raw prompts, attachment
   bodies, and imported archives. Detection fails the mutation; it never
   silently drops or rewrites the value.
4. Repository replay and realtime publication revalidate the versioned envelope
   before returning or publishing it. Legacy unversioned or unknown-policy rows
   are retained but fail closed and are never published as successful delivery.
5. Existing 64 KiB replay/outbox and 16 KiB audit limits remain unchanged.

### 3.4 Outbox claim identity

1. Ack/fail transitions retain the complete claimed primary-key tuple:
   `state`, `available_at`, `workspace_id`, `id`, plus `claim_token` and the
   observed `lease_expires_at`.
2. The dispatcher reads the clock again after each publish attempt. That
   transition time, not claim time, validates `lease_expires_at > now` and is
   the base for retry scheduling.
3. The SQL predicate contains the complete tuple, token, observed lease, and
   current lease check. Zero or multiple affected rows return
   `outbox_claim_conflict`; no sibling row sharing an event ID is modified.
4. Replay of a dead-letter event also uses the complete persisted tuple; event
   ID alone is never write authority.

### 3.5 Empty-only down migration

1. Migration 000009 down remains test-only and is never called from normal
   migration, runtime close, worker stop, or operational rollback.
2. Its SQL creates a temporary guard, rejects execution if any of the four
   governance tables contains a row, then drops all four tables in reverse
   dependency order and removes only the matching migration-catalog entry.
3. The test harness executes the script transactionally. On guard failure all
   tables, evidence, and the catalog entry remain unchanged.
4. No data cleanup or down execution against a non-empty external database is
   authorized.

## 4. Dependencies and preflight

- committed S01B candidate `5062e84` and independent review evidence `0218ecb`;
- unchanged policy hashes for `CLAUDE.md`, `backend/AGENTS.md`, and plan
  snapshots v1-v4;
- Human Customer approval of `PRODUCT-CAPABILITY-ROADMAP-001 v5 / r010` on
  2026-08-17;
- existing Go 1.26.1 toolchain and process-local compatible Windows race
  compiler;
- current governance schema 000009 and retained S01B tests.

Before product edits, revalidate HEAD, hashes, branch, exact paths, unrelated
dirty exclusions, and empty `server/**` status. Read-only inspection may count
local governance rows; any non-empty external or user-owned database stops data
handling. The code repair itself never deletes or rewrites those rows.

## 5. Ordered TDD steps

### PCR-S01B-5.1 — Authority and opaque preparation

Write failing tests for missing/unknown/mismatched policy, forged/cross-workspace
actor, and direct prepared-value construction. Add the smallest provider seam,
exact action/resource policy resolution, trusted identity projection, and opaque
prepared mutation that makes them pass.

### PCR-S01B-5.2 — Canonical hash and safe envelopes

Write failing tests for caller digest forgery, semantic canonical equivalence,
different-body/action/version conflicts, unversioned/unknown/duplicate fields,
wrong scalar kinds, forbidden keys, and secret-bearing allowed values. Implement
canonical hashing, strict policy schemas, versioned envelopes, repository replay
validation, and sink revalidation. Retained legacy rows fail closed.

### PCR-S01B-5.3 — Full tuple and current lease

Write failing fake-clock and SQLite tests for publish beyond lease, expired
ack/fail without reclaim, same event ID on different tuples, stale observed
lease, and dead-letter replay identity. Change the internal port and SQL to use
the complete claim tuple and post-publish clock.

### PCR-S01B-5.4 — Empty-only test down

Write failing tests proving the current down SQL deletes evidence. Replace it
with the transactional empty-table guard, cover each non-empty table and partial
occupancy, then prove empty down and catalog consistency. Normal migration and
runtime close must retain evidence.

### PCR-S01B-5.5 — Integrated evidence and independent review

Run all focused S01A/S01B tests, negative security fixtures, migration upgrade/
restart, r008 attachment regressions, full Backend, race, Make, module, policy,
format, generated, diff, and scope gates on one candidate. Commit with required
trailers, then obtain fresh independent spec and code-quality review. Any BLOCK
stops closure and requires a new plan version.

### PCR-S01B-5.6 — Release 0 closure

Only after every deterministic and independent gate passes, synchronize the
roadmap records, preserve Release 1 inactive, commit indexed evidence, and apply
the Human Customer's continuing direction to complete Release 0. No technical
check silently substitutes for a recorded Customer acceptance decision.

## 6. Exact writable paths

Governance contract and application:

- `backend/internal/modules/workspace/contract/errors.go`
- `backend/internal/modules/workspace/contract/governance.go`
- `backend/internal/modules/workspace/contract/governance_test.go`
- `backend/internal/modules/workspace/internal/application/governance_policy.go` (new)
- `backend/internal/modules/workspace/internal/application/governance_policy_test.go` (new)
- `backend/internal/modules/workspace/internal/application/governance_service.go`
- `backend/internal/modules/workspace/internal/application/governance_service_test.go`
- `backend/internal/modules/workspace/internal/application/outbox_service.go`
- `backend/internal/modules/workspace/internal/application/outbox_service_test.go`

Persistence, composition, and compatibility:

- `backend/internal/modules/workspace/internal/infrastructure/sqlite/governance_repository.go`
- `backend/internal/modules/workspace/internal/infrastructure/sqlite/governance_repository_test.go`
- `backend/internal/modules/workspace/internal/infrastructure/sqlite/migrations/000009_workspace_governance.down.sql`
- `backend/internal/modules/workspace/governance.go`
- `backend/internal/modules/workspace/governance_test.go`
- `backend/internal/modules/workspace/governance_outbox.go`
- `backend/internal/modules/workspace/governance_outbox_test.go`
- `backend/internal/modules/workspace/sqlite_persistence_test.go`
- `backend/internal/bootstrap/sqlite.go`
- `backend/internal/bootstrap/sqlite_runtime_test.go`

Roadmap authority and evidence:

- `backend/docs/plans/product-capability-roadmap/plan.md`
- `backend/docs/plans/product-capability-roadmap/plan_v5.md`
- `backend/docs/plans/product-capability-roadmap/story-map.md`
- `backend/docs/plans/product-capability-roadmap/task-register.md`
- `backend/docs/plans/product-capability-roadmap/journal.md`

No other path may be modified or staged. A required path outside this list,
schema up change, or non-empty data repair stops execution and requires a new
plan version or task revision.

## 7. Acceptance criteria

1. Unknown/missing/mismatched policies and forged or foreign actors fail closed;
   repository execution cannot bypass successful policy preparation.
2. Persisted request hashes are application-derived canonical v1 SHA-256 values;
   same semantics replay and any result-affecting difference conflicts.
3. Replay, audit, and outbox data is versioned, exact-schema, bounded, and
   secret-free; forbidden fixtures never reach SQLite or realtime.
4. Retained unversioned/unknown-policy rows are not returned or published, and
   no repair deletes or rewrites them.
5. Expired or stale claims cannot ack/fail/replay; complete tuples isolate
   duplicate event IDs and every transition affects exactly one intended row.
6. Non-empty 000009 down fails atomically with all evidence/catalog intact;
   empty test down removes the four tables and its catalog entry only.
7. Existing atomic rollback, revision winner/conflict, retry/dead-letter,
   restart, diagnostics, readiness, attachment, and false feature-flag behavior
   remains green.
8. No up migration, public API, frontend, Control Plane, generated, unrelated
   dirty, or `server/**` path changes.
9. Full deterministic evidence and fresh independent spec/code-quality reviews
   report no BLOCK.
10. Closure records mark Release 0 complete and Release 1 inactive.

## 8. Deterministic verification

From `backend/` unless stated otherwise:

```text
go test ./internal/modules/workspace/contract -count=1
go test ./internal/modules/workspace/internal/application -count=1
go test ./internal/modules/workspace/internal/infrastructure/sqlite -count=1
go test ./internal/modules/workspace -count=1
go test ./internal/bootstrap -count=1
go test ./internal/modules/space/internal/infrastructure/sqlite -run 'AttachmentRepository.*(BusyWriteAcquisition|Cancellation|Serial)' -count=10
go test ./internal/bootstrap -run '^TestSQLiteRuntimeConcurrentAttachmentUploadsLoseNoReferencesOrFiles$' -count=10
go test ./... -count=1
go vet ./...
go mod verify
make fmt-check
make check
make test-race RACE_PACKAGES="./internal/modules/workspace/contract ./internal/modules/workspace/internal/application ./internal/modules/workspace/internal/infrastructure/sqlite ./internal/modules/workspace ./internal/bootstrap"
git diff --check
git diff --cached --check
git diff --name-only -- server
git diff --cached --name-only -- server
git status --porcelain -- server
```

Focused RED and GREEN commands are recorded per step in the journal. The final
candidate path set is compared exactly with Section 6 before commit and review.

## 9. Risks and controls

| Risk | Control |
| --- | --- |
| Policy provider becomes a caller-controlled bypass | server-owned injection, missing-provider denial, opaque prepared value |
| Canonicalization changes replay identity | freeze v1 wrapper/scalars, golden digest and equivalence tests |
| Secret scanner overclaims arbitrary detection | strict typed allowlists first, universal forbidden guard second, no free text |
| Legacy rows are accidentally trusted or destroyed | retain rows, validate on read/publish, fail closed, no cleanup authority |
| Tuple signature misses a PK component | typed claim key mirrors all four PK fields plus token/lease; duplicate-ID tests |
| Expired publish is acknowledged | post-publish clock and SQL lease predicate |
| Down script partially deletes tables | transactional guard and one-row-per-table negative tests |
| Repair expands into Release 1 | all feature flags false; no capability repository or UI retrofit |

## 10. Rollback

- Before commit, revert only v5 task paths through an explicit patch; never
  discard unrelated worktree changes.
- After commit, use a new revert commit if directed; never reset shared history.
- Code rollback disables the new governance provider/dispatcher behavior but
  retains all governance tables and evidence.
- Never run the down script against non-empty external data, delete legacy rows,
  mutate permanent host configuration, or touch `server/**`.

## 11. Approval record

The Human Customer explicitly approved `PRODUCT-CAPABILITY-ROADMAP-001 v5 /
r010` on 2026-08-17 after receiving the five frozen defaults, compatibility
boundary, exclusions, stop conditions, and verification/review requirement.
This approval authorizes only this plan and task; it does not authorize push,
merge, deployment, non-empty data handling, Release 1 activation, or scope
expansion.
