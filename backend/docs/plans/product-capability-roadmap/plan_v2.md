# Product capability roadmap — execution plan v2 (proposed)

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Version: `2`
- Status: `proposed; awaiting Human Customer approval`
- Base commit: `2589ce450aa8527a19e21ff23812e1659bfd0ddd`
- Proposed active step after approval: `PCR-S01B-1`
- Supersedes for future execution: `plan_v1.md`
- Product contract: `PCR-CONTRACT-v1` plus
  [s01b-foundation-design.md](s01b-foundation-design.md)

This snapshot is not executable until the Human Customer explicitly approves
v2 and `plan.md` is updated to name it as the approved version. Until then,
`PCR-001-S01B-R4` remains documentation-only.

## 1. Objective

Install a reusable Canonical Workspace foundation for optimistic revisions,
idempotent mutation replay, immutable audit, transactional outbox delivery, and
redacted operational diagnostics. Preserve all current behavior and leave every
roadmap product feature disabled.

## 2. Scope

### Included

- Workspace public governance values, ports, errors, and authorization action;
- additive SQLite governance tables and migration verification;
- one SQLite-native transaction helper used by later capability repositories;
- idempotency hashing/replay and revision-conflict behavior;
- immutable redacted audit and durable outbox storage;
- bounded outbox claim/delivery/retry/dead-letter worker;
- owner/admin-only governance diagnostics;
- Canonical runtime composition behind explicit provider validation;
- deterministic, concurrency, restart, rollback, security, and compatibility
  evidence.

### Excluded

- retrofitting existing Issue, project, pin, attachment, or realtime writes;
- Task, search, Skill, Knowledge, resource, Requirement, retrospective,
  similarity, notification, phase, or outline behavior;
- new frontend or desktop UI;
- Control Plane imports, table reads, writes, or process coupling;
- PostgreSQL implementation or schema;
- secondary SQLite indexes or any non-concurrent explicit index DDL;
- changes under `server/**`;
- cleanup of pre-existing dirty UI/local-runtime paths.

## 3. Invariants

1. Workspace identity and actor authority are resolved server-side.
2. Missing governance providers fail closed and do not change feature flags.
3. Domain, revision, audit, outbox, and idempotency replay commit once or all
   roll back.
4. One successful governed mutation advances its resource revision exactly one.
5. Same key/hash replays; same key/different hash conflicts without mutation.
6. Audit and outbox payloads are allowlisted, bounded, and secret-free.
7. Outbox delivery is at-least-once; claim tokens and leases prevent concurrent
   double ownership but not crash-window duplicate publication.
8. No foreign keys or cascading actions are added.
9. SQLite migration adds no explicit secondary or unique index DDL.
10. Existing synchronous realtime behavior is unchanged until a later vertical
    slice explicitly migrates it to the outbox.
11. Every current roadmap feature flag remains false.
12. `server/**` remains read-only without exception.

## 4. Dependencies

- Approved `PCR-CONTRACT-v1` and S01A capability authorization foundation.
- Human Customer S01A acceptance and explicit gate waiver recorded in J008.
- Canonical SQLite runtime and Workspace migration runner.
- Existing trusted HTTP identity, CSRF, membership, and owner/admin role
  resolution.
- Existing in-memory realtime Hub as an injectable test/runtime sink; it is not
  durable authority.
- Go standard library SHA-256 and canonical JSON implementation owned by S01B.

No external service, credential, package installation, network call, or Control
Plane runtime is required.

## 5. Ordered steps

Only one step may be active at a time. Each step requires a frozen task revision
with the exact base, policy hashes, paths, and acceptance before product edits.

### PCR-S01B-1 — Contract and migration

Add governance values/errors/ports and the additive version-9 Workspace SQLite
schema. Prove migration, isolation, bounds, and retained-data restart. Do not
compose runtime behavior yet.

Exact product paths:

- `backend/internal/modules/workspace/contract/governance.go` (new)
- `backend/internal/modules/workspace/contract/governance_test.go` (new)
- `backend/internal/modules/workspace/contract/errors.go`
- `backend/internal/modules/workspace/internal/infrastructure/sqlite/migrations/000009_workspace_governance.up.sql` (new)
- `backend/internal/modules/workspace/internal/infrastructure/sqlite/migrations/000009_workspace_governance.down.sql` (new)
- `backend/internal/modules/workspace/sqlite_workspace_services_test.go`
- `backend/internal/modules/workspace/sqlite_persistence_test.go`
- roadmap task register and journal.

Acceptance:

- all contract values reject empty/untrusted identities, invalid revisions,
  oversized replay/audit/outbox payloads, and unknown states;
- migration 9 contains no FK/cascade/trigger/explicit index DDL;
- fresh, version-8 upgrade, second run, failed migration rollback, and restart
  tests pass;
- no runtime route, worker, event, or feature flag changes.

### PCR-S01B-2 — SQLite mutation governance

Implement the SQLite-native transaction helper and application service for
revision checks, replay, audit, and outbox enqueue on one connection.

Exact product paths:

- `backend/internal/modules/workspace/internal/application/governance_service.go` (new)
- `backend/internal/modules/workspace/internal/application/governance_service_test.go` (new)
- `backend/internal/modules/workspace/internal/infrastructure/sqlite/governance_repository.go` (new)
- `backend/internal/modules/workspace/internal/infrastructure/sqlite/governance_repository_test.go` (new)
- `backend/internal/modules/workspace/governance.go` (new)
- `backend/internal/modules/workspace/governance_test.go` (new)
- roadmap task register and journal.

Acceptance:

- concurrent same-revision writes produce one winner and one conflict;
- same-key replay and different-body conflict are deterministic;
- test aborts after every phase leave no partial domain/governance state;
- workspace/action isolation and audit redaction pass;
- no existing repository is retrofitted.

### PCR-S01B-3 — Outbox delivery and diagnostics

Add bounded claim/delivery state transitions, the dispatcher, diagnostics, and
explicit Canonical composition. The worker starts only when repository and sink
providers are both installed.

Exact product paths:

- `backend/internal/modules/workspace/internal/application/outbox_service.go` (new)
- `backend/internal/modules/workspace/internal/application/outbox_service_test.go` (new)
- `backend/internal/modules/workspace/internal/infrastructure/sqlite/governance_repository.go`
- `backend/internal/modules/workspace/internal/infrastructure/sqlite/governance_repository_test.go`
- `backend/internal/modules/workspace/internal/interfaces/http/governance.go` (new)
- `backend/internal/modules/workspace/internal/interfaces/http/governance_test.go` (new)
- `backend/internal/modules/workspace/governance.go`
- `backend/internal/modules/workspace/governance_outbox.go` (new)
- `backend/internal/modules/workspace/governance_outbox_test.go` (new)
- `backend/internal/modules/workspace/sqlite_workspace_chain.go`
- `backend/internal/modules/workspace/sqlite_workspace_chain_test.go`
- `backend/internal/modules/workspace/sqlite_workspace_services.go`
- `backend/internal/bootstrap/sqlite.go`
- `backend/internal/bootstrap/sqlite_runtime_test.go`
- `backend/internal/bootstrap/runtime.go`
- `backend/internal/bootstrap/runtime_test.go`
- roadmap task register and journal.

Acceptance:

- claim batches, 60-second leases, stale tokens, initial plus three attempts,
  deterministic jitter, dead-letter, replay, and restart pass;
- crash-window duplicate delivery retains one stable event ID/revision;
- `/api/operations/governance` is workspace-scoped and owner/admin-only;
- diagnostics contain counts/timestamps only and report backlog degradation;
- missing repository/sink keeps governance unavailable and readiness honest;
- all product feature flags remain false.

### PCR-S01B-4 — Integrated evidence and independent review

Run deterministic, race-capable, clean-candidate, restart, security, and current
behavior regressions. Index evidence and obtain independent review. No product
behavior is added in this step.

Exact writable paths are roadmap journal/task evidence only unless a reproduced
defect requires a new plan version and task revision.

Acceptance:

- all frozen commands pass on the same candidate, except an explicitly recorded
  environment limitation that the Customer separately accepts;
- no `server/**` or unrelated dirty path is present in the candidate diff;
- independent reviewer confirms transaction atomicity, isolation, redaction,
  lease safety, migration policy, and scope;
- Customer Acceptance remains a separate explicit decision.

## 6. Deterministic verification

Narrow iteration:

```text
cd backend && go test ./internal/modules/workspace/contract
cd backend && go test ./internal/modules/workspace/internal/application
cd backend && go test ./internal/modules/workspace/internal/infrastructure/sqlite
cd backend && go test ./internal/modules/workspace ./internal/bootstrap
```

Step and final gates:

```text
cd backend && go test ./internal/modules/workspace/contract ./internal/modules/workspace/internal/application ./internal/modules/workspace/internal/infrastructure/sqlite ./internal/modules/workspace ./internal/bootstrap
cd backend && go test -race ./internal/modules/workspace/contract ./internal/modules/workspace/internal/application ./internal/modules/workspace/internal/infrastructure/sqlite ./internal/modules/workspace ./internal/bootstrap
cd backend && make check
git diff --check
git diff --name-only -- server
```

On Windows, `0xc0000139` remains an environment limitation rather than a race
PASS. Full-suite attachment concurrency failures remain separate evidence; a
new S01B change may not hide, weaken, or silently waive them.

## 7. Acceptance criteria

S01B is technically complete only when:

1. every ordered step is committed on its frozen base and scope;
2. revision, replay, audit, and outbox atomicity are proven under failure and
   concurrency;
3. fresh/upgrade/restart migration paths pass without explicit index DDL;
4. worker lease/retry/dead-letter behavior meets frozen budgets;
5. diagnostics are authorized, redacted, and operationally useful;
6. existing runtime behavior and all false roadmap flags remain unchanged;
7. deterministic checks and an independent review pass;
8. the Human Customer separately accepts S01B.

## 8. Risks and controls

| Risk | Control |
| --- | --- |
| Nested transactions split atomicity | Capability repository owns one connection and transaction; governance helper never starts a nested transaction |
| Idempotency key leaks across tenants/actions | Composite key includes workspace and action; negative isolation tests |
| Replay body stores sensitive content | Versioned allowlisted envelope, 64 KiB cap, forbidden-value tests |
| Audit captures secrets | Action-specific allowlist and redaction tests |
| Crash duplicates event delivery | Stable event ID and aggregate revision; at-least-once contract |
| Worker double claims | `BEGIN IMMEDIATE`, claim token, lease expiry, stale-token rejection |
| Outbox grows without visibility | ready age, dead-letter, attempts, and last-success diagnostics |
| SQLite query needs secondary index | Stop and propose a new plan version; never add non-concurrent index DDL |
| Existing realtime semantics drift | No retrofit in S01B; compatibility tests remain mandatory |
| Control Plane becomes second authority | No imports, reads, calls, or dual writes; architecture test and review |

## 9. Rollback

- Before composition: revert the current step's Go code and retain additive
  empty tables.
- After composition: disable governance provider/dispatcher composition, stop
  the owned worker, and retain all governance evidence.
- Do not run destructive down migrations against populated tables in normal
  rollback.
- Removing audit, idempotency, or outbox data requires separate Customer
  authority and a retention/legal impact review.
- A rollback never changes `server/**` or adopts Control Plane storage.

## 10. Approval gate

Approval must explicitly name `PRODUCT-CAPABILITY-ROADMAP-001 v2`. After
approval, update `plan.md`, close `PCR-001-S01B-R4`, and freeze a new task for
`PCR-S01B-1` on the then-current commit and policy hashes. Approval does not
activate later S01B steps or any product capability.
