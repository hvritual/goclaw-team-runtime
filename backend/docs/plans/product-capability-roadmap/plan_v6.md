# Product capability roadmap — execution plan v6

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Version: `6`
- Status: `approved-for-execution`
- Approved by: `Human Customer, 2026-08-17`
- Base commit: `f93eca77c3450109b7328441812d63710f179521`
- Active step: `PCR-S01B-6`
- Task revision: `r011`
- Supersedes for future execution: `plan_v5.md`
- Product contract: unchanged `PCR-CONTRACT-v1` and
  [s01b-foundation-design.md](s01b-foundation-design.md)

## 1. Objective

Repair every v5 independent-review block without weakening the already passing
Release 0 authority, canonical hash, opaque preparation, outbox tuple/lease, or
empty-only migration guarantees. Re-run deterministic and independent evidence
and close Release 0 only if no BLOCK remains.

## 2. Included scope

- remove deprecated unrestricted raw response, audit, and outbox draft inputs;
- detect Basic authorization material in the universal secret guard;
- validate the installed exact event policy before any outbox claim rewrite;
- validate persisted envelope and exact event policy before dead-letter replay;
- freeze a defensive deep copy of each resolved action policy inside prepared
  mutations;
- RED/GREEN tests, compatibility checks, full Backend/race evidence, exact
  scope audit, independent review, and Release 0 records.

## 3. Excluded scope

- deletion, cleanup, quarantine, repair, migration, backfill, or rewrite of any
  retained invalid/legacy/unknown-policy row;
- new schema, table, column, index, trigger, up/down migration, Proto/OpenAPI,
  public route, frontend, Desktop, Control Plane, or Release 1 behavior;
- retrofit of existing Issue/project/attachment/realtime mutations;
- `server/**`, unrelated dirty paths, push, merge, deployment, or external data.

## 4. Frozen design

### 4.1 Typed-only request boundary

`GovernanceRequest` exposes only normalized `RequestFields`, `ResponseFields`,
and `AuditFields`; `OutboxDraft` exposes only `Fields`. Deprecated
`ResponseBody`, `AuditMetadata`, and `Payload` fields are removed rather than
ignored or filtered. Compile-time reflection tests prove no unrestricted raw
input remains.

### 4.2 Basic authorization detection

The case-insensitive universal guard rejects `basic ` and `basic:` material in
keys or string values in addition to the v5 forbidden set. It does not reject
an unrelated identifier merely containing the word `basic` without an
authorization separator. Exact typed schemas remain the first boundary.

### 4.3 Immutable policy snapshot

Successful server-side resolution produces a deep copied policy snapshot:
request/replay/audit schemas, event-schema maps, field rules, and enum slices.
Later mutation of provider-owned maps or slices cannot alter a prepared value,
its validation, or its persisted envelopes.

### 4.4 Validate-before-claim

The SQLite governance repository receives the same installed
`GovernanceEventPolicyProvider` as the dispatcher. Within the existing
`BEGIN IMMEDIATE` claim transaction it scans the candidate batch, validates
each generic v1 envelope and exact event/aggregate policy, and only then updates
any tuple to `inflight`. Missing/unknown policy or invalid/legacy payload aborts
and rolls back the whole batch; every selected row retains its exact tuple and
contents. Safety takes precedence over claiming later valid rows in that batch.

### 4.5 Validate-before-replay

Dead-letter replay uses `BEGIN IMMEDIATE`, loads the exact persisted tuple,
validates the full event plus installed exact event policy, and only then moves
that one tuple to `ready`. Missing, unversioned, secret-bearing, mismatched, or
unknown-policy rows return a fail-closed error and remain byte-for-byte and
tuple-for-tuple unchanged.

## 5. Ordered TDD steps

### PCR-S01B-6.1 — Raw-input and Basic guard

Write failing reflection and secret fixtures proving unrestricted raw inputs
remain and `Basic:dXNlcjpwYXNz` passes. Remove the raw fields and minimally
extend the universal guard; retain a legal `basic-plan` negative-control test.

### PCR-S01B-6.2 — Frozen policy snapshot

Write a failing test that mutates provider schema maps and enum slices after
preparation and observes prepared validation drift. Deep copy the resolved
policy and keep all authority/hash/envelope tests green.

### PCR-S01B-6.3 — Validate-before-claim

Write real SQLite failing tests for unversioned, secret-bearing, and
unknown-policy ready rows. Prove ClaimOutbox currently rewrites them, then
inject the provider and validate the full candidate batch before the first
UPDATE. Prove invalid and valid sibling rows both retain their original tuples
when the batch fails.

### PCR-S01B-6.4 — Validate-before-replay

Write real SQLite failing tests for unversioned and unknown-policy dead letters.
Move read, exact-policy validation, and tuple update into one immediate
transaction; prove failure preserves tuple, attempts, error code, and payload.

### PCR-S01B-6.5 — Candidate evidence and independent review

Commit the exact candidate with machine-readable trailers. Run all focused
S01A/S01B, negative security, attachment contention, full Backend, Make, module,
race, policy, diff, and `server/**` gates on that commit. Then obtain a fresh
independent spec/code-quality review. Any BLOCK stops closure and requires a new
approved plan version.

### PCR-S01B-6.6 — Release 0 closure

Only after deterministic and independent PASS, synchronize roadmap records,
mark Release 0 complete, keep Release 1 inactive, and commit indexed evidence.

## 6. Exact writable paths

- `backend/internal/modules/workspace/contract/governance.go`
- `backend/internal/modules/workspace/contract/governance_test.go`
- `backend/internal/modules/workspace/internal/application/governance_policy.go`
- `backend/internal/modules/workspace/internal/application/governance_policy_test.go`
- `backend/internal/modules/workspace/internal/application/governance_service.go`
- `backend/internal/modules/workspace/internal/application/governance_service_test.go`
- `backend/internal/modules/workspace/internal/application/outbox_service.go`
- `backend/internal/modules/workspace/internal/application/outbox_service_test.go`
- `backend/internal/modules/workspace/internal/infrastructure/sqlite/governance_repository.go`
- `backend/internal/modules/workspace/internal/infrastructure/sqlite/governance_repository_test.go`
- `backend/internal/modules/workspace/governance_outbox.go`
- `backend/internal/modules/workspace/governance_outbox_test.go`
- `backend/internal/bootstrap/sqlite.go`
- `backend/internal/bootstrap/sqlite_runtime_test.go`
- `backend/docs/plans/product-capability-roadmap/plan.md`
- `backend/docs/plans/product-capability-roadmap/plan_v6.md`
- `backend/docs/plans/product-capability-roadmap/story-map.md`
- `backend/docs/plans/product-capability-roadmap/task-register.md`
- `backend/docs/plans/product-capability-roadmap/journal.md`

No other path may be modified or staged. A required path outside this list,
schema change, non-empty data action, or new behavior stops execution and
requires another plan/task revision.

## 7. Acceptance criteria

1. No caller-writable unrestricted raw response/audit/outbox draft input exists.
2. Basic authorization fixtures fail while legal non-authorization identifiers
   remain accepted.
3. Prepared policy snapshots cannot drift after provider-owned data changes.
4. Invalid, unversioned, secret-bearing, or unknown-policy claim candidates are
   not published, returned as successful delivery, or modified in SQLite.
5. Invalid or unknown-policy dead letters cannot replay and remain unchanged.
6. All v5 PASS dimensions remain green, including canonical hashes, authority,
   opaque preparation, full tuple/current lease, and empty-only down.
7. Attachment concurrency, restart/readiness, diagnostics, policy, generated,
   full Backend, and race evidence passes on one fixed candidate.
8. Candidate contains only Section 6 paths and no `server/**` changes.
9. Fresh independent review reports no BLOCK.
10. Closure records mark Release 0 complete and Release 1 inactive.

## 8. Deterministic verification

From `backend/` unless stated otherwise:

```text
go test ./internal/modules/workspace/contract -count=1
go test ./internal/modules/workspace/internal/application -count=1
go test ./internal/modules/workspace/internal/infrastructure/sqlite -count=1
go test ./internal/modules/workspace -count=1
go test ./internal/bootstrap -count=1
go test ./internal/modules/space/internal/infrastructure/sqlite -run '^(TestAttachmentRepositoryRetriesBusyWriteAcquisition|TestAttachmentRepositorySerializesOwnedWritersBeforeAcquiringSQLiteConnections|TestAttachmentRepositoryCancelsWhileWaitingForOwnedWriter)$' -count=10
go test ./internal/bootstrap -run '^TestSQLiteRuntimeConcurrentAttachmentUploadsLoseNoReferencesOrFiles$' -count=10
go test ./... -count=1
go vet ./...
go mod verify
make check
make test-race
git diff --check
git diff --cached --check
git diff --name-only -- server
git diff --cached --name-only -- server
git status --porcelain -- server
```

Focused RED/GREEN evidence is recorded in the journal. Candidate paths are
compared exactly with Section 6 before commit and review.

## 9. Risks and controls

| Risk | Control |
| --- | --- |
| Validation occurs after a write | validate every candidate in the same transaction before the first UPDATE |
| Invalid head row starves valid rows | explicit safety-first batch failure; no silent skip or mutation |
| Provider data changes after prepare | deep-copy all policy maps, rules, and enum slices |
| Basic detector causes broad false positives | match authorization separators and retain a legal negative control |
| Replay races with another transition | immediate transaction plus complete persisted tuple |
| Repair expands into Release 1 | deny-all runtime catalog and exact path allowlist |

## 10. Rollback

- Before commit, revert only v6 paths through explicit patches.
- After commit, use a new revert commit if directed; never reset shared history.
- Never mutate retained invalid rows, run governance down, change external data,
  modify permanent host configuration, or touch `server/**`.

## 11. Approval record

The Human Customer explicitly approved `PRODUCT-CAPABILITY-ROADMAP-001 v6 /
r011` on 2026-08-17 after receiving the exact raw-input, Basic guard,
validate-before-claim/replay, immutable-policy, verification, exclusion, and
stop boundaries. This does not authorize push, merge, deployment, Release 1,
schema changes, external data handling, or scope expansion.
