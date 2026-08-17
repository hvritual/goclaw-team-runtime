# Product Capability Roadmap Task Register

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Plan-Version: `v2`
- Registry status: `PCR-S01B-3 candidate committed; verification blocked`
- Registry revision: `r007`
- Updated: `2026-08-17`

## Frozen policy bundle

| Policy source | SHA256 |
| --- | --- |
| `CLAUDE.md` | `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646` |
| `backend/AGENTS.md` | `fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209` |
| `plan_v1.md` | `4351025652e88083b8ceb25a72488e9d568a445a5f0e3a8793650393086ad0b5` |
| `plan_v2.md` | `af0486c7625067b2b25de163271ba6f4be153c74331488eb57d9c42ab69ffd30` |

Changing any policy hash invalidates a queued product task until it is reviewed
and re-frozen. `server/**` remains excluded regardless of hash changes.

## PCR-001-S00-R1

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-S00-R1`
- Task-Revision: `r001`
- Work-Item: `PCR-S00`
- Status: `complete-by-customer-direction`
- Assignee: `Codex primary agent`
- Independent reviewer: `not assigned; retained as program acceptance debt`
- Base commit: `45213820fade7f61294d2287e063bf19fbd015ee`
- Write boundary: `backend/docs/plans/product-capability-roadmap/**` only
- Product-code authority: none
- Acceptance source: `plan_v1.md` PCR-S00
- Verification:
  - Markdown relative links resolve.
  - `git diff --check -- backend/docs/plans/product-capability-roadmap` passes.
  - HEAD and policy hashes match the frozen values.
  - Worktree exclusions remain present and unmodified.

Expected outputs:

- approved plan pointer and immutable v1 snapshot;
- `contract-freeze_v1.md`;
- this task register;
- append-only approval, drift, and gate evidence in `journal.md`.

## PCR-001-S01A-R3

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-S01A-R3`
- Task-Revision: `r003`
- Work-Item: `PCR-S01A`
- Title: `Canonical capability authorization and accurate feature flags`
- Status: `customer-accepted-with-explicit-gate-waiver`
- Assignee: `Codex primary agent`
- Independent reviewer: `waived by explicit Human Customer acceptance`
- Base commit: `144997ab5fcd04544f8ffa40a1a75fc79fdb5904`
- Prior blocking gate: cleared by the Human Customer statement that C9 passed
  and the explicit direction to return to PCR-S01A.
- Evidence qualification: the local v11 dependency/browser rerun remained
  incomplete and is not represented as a technical PASS.
- Overlap audit: v11 changed Issue HTTP filtering and Canonical plan evidence;
  PCR-S01A r002 does not authorize Issue HTTP paths and has no product overlap.

### Exact allowed product paths after resume

Only the following files are authorized for task revision r003. If architecture
discovery proves another path is necessary, stop and create r004 before editing.

- `backend/internal/modules/workspace/contract/capability_authorization.go`
- `backend/internal/modules/workspace/contract/capability_authorization_test.go`
- `backend/internal/bootstrap/sqlite.go`
- `backend/internal/bootstrap/sqlite_runtime_test.go`
- `backend/internal/bootstrap/runtime.go`
- `backend/internal/bootstrap/runtime_test.go`
- `backend/docs/plans/product-capability-roadmap/journal.md`
- `backend/docs/plans/product-capability-roadmap/task-register.md`

The task may add the two new contract files above. It may make narrow edits to
the named bootstrap/runtime files. No migration, API, frontend, Control Plane,
legacy server, or unrelated refactor is authorized by S01A r003.

### Frozen acceptance

1. Every roadmap action has a named authorization constant or is explicitly
   marked unavailable.
2. Member and agent default matrices fail closed and are table-tested.
3. `/api/config` flags remain false for every uninstalled roadmap capability.
4. Enabling a flag requires an injected installed provider; missing providers
   fail readiness or keep the flag false as frozen per capability.
5. Existing Issue, project, pin, auth, attachment, and realtime behavior remains
   unchanged.
6. No request can gain permission from a client-supplied actor type or workspace.

### Frozen deterministic verification after resume

```text
cd backend && go test ./internal/modules/workspace/contract ./internal/bootstrap
cd backend && go test -race ./internal/modules/workspace/contract ./internal/bootstrap
cd backend && make check
```

On Windows, loader exit `0xc0000139` is recorded as an environment limitation,
not a passing race result. A non-Windows race result or CI equivalent remains
required before technical acceptance.

### Current verification state

- The S01A authorization catalog, installed-provider seam, explicit disabled
  flags, membership isolation, and role/agent defaults are implemented inside
  the frozen paths.
- Focused contract and Bootstrap S01A tests pass.
- Policy check, changed-file formatting, generated-output cleanliness, and
  `go vet ./...` pass.
- Technical verification remains incomplete: the frozen race command exits with the
  documented Windows loader code `0xc0000139`; full `go test ./...` also
  reproduces pre-existing attachment concurrency failures outside S01A scope.
- The Human Customer explicitly accepted S01A and waived the three outstanding
  gates before directing S01B activation. The technical evidence remains
  unchanged and must not be represented as a PASS.

## PCR-001-S01B-R4

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-S01B-R4`
- Task-Revision: `r004`
- Work-Item: `PCR-S01B`
- Title: `Revision, audit, idempotency, and outbox contract and migration design`
- Status: `complete-by-customer-approval`
- Assignee: `Codex primary agent`
- Independent reviewer: `required before product-code task acceptance`
- Base commit: `cc61297be42ca5acf1fc47d9ba9d70939f406588`
- Authority source: Human Customer confirmation accepting S01A gate waivers and
  activating S01B on `2026-08-16`.
- Product-code authority: none.

### Exact allowed paths

- `backend/docs/plans/product-capability-roadmap/plan.md`
- `backend/docs/plans/product-capability-roadmap/plan_v2.md` (new, proposed)
- `backend/docs/plans/product-capability-roadmap/s01b-foundation-design.md` (new)
- `backend/docs/plans/product-capability-roadmap/story-map.md`
- `backend/docs/plans/product-capability-roadmap/task-register.md`
- `backend/docs/plans/product-capability-roadmap/journal.md`

No Go, SQL migration, generated, frontend, Control Plane, or `server/**` path is
authorized by r004. Product work requires an approved plan version and a new
task revision with exact implementation paths.

### Frozen design acceptance

1. Workspace owns its governance records; Control Plane is evidence only and is
   neither imported nor dual-written.
2. Revision conflict, idempotency replay/body-conflict, audit redaction, outbox
   state, retry, lease, and post-commit contracts are exact and testable.
3. The design resolves the root concurrent-index rule against the current
   transaction-wrapped SQLite migration runner without weakening either policy
   by implication.
4. Transaction ownership proves rollback removes domain, audit, idempotency,
   and outbox effects together.
5. A proposed `plan_v2.md` freezes exact product paths, migrations, tests,
   diagnostics, rollback, and independent-review gates for Customer approval.

### Deterministic verification

```text
git diff --check -- backend/docs/plans/product-capability-roadmap
git diff --name-only -- server
```

Revalidate the policy-bundle hashes and base commit before proposing v2.

### Design outputs

- `s01b-foundation-design.md`
- `plan_v2.md` with status `proposed; awaiting Human Customer approval`
- Design SHA256:
  `dfff4f470f29234880939b1ca2826db8f5a6c4e979df7f4c50473f906f033d3f`
- Proposed plan SHA256:
  `af0486c7625067b2b25de163271ba6f4be153c74331488eb57d9c42ab69ffd30`
- Product-code authority remains none until v2 is explicitly approved and a
  new implementation task is frozen.

## PCR-001-S01B1-R5

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-S01B1-R5`
- Task-Revision: `r005`
- Work-Item: `PCR-S01B-1`
- Title: `Workspace governance contract and SQLite migration`
- Status: `customer-accepted-with-explicit-gate-waiver`
- Assignee: `Codex primary agent`
- Independent reviewer: `required before S01B final acceptance`
- Product-code base commit: `312feda1aeaafb5d1aecffd61a7fbcdcbd7ee3c6`
- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Plan-Version: `v2`
- Plan-Step: `PCR-S01B-1`
- Policy bundle: the hashes above, including approved `plan_v2.md`.

### Exact allowed paths

- `backend/internal/modules/workspace/contract/governance.go` (new)
- `backend/internal/modules/workspace/contract/governance_test.go` (new)
- `backend/internal/modules/workspace/contract/errors.go`
- `backend/internal/modules/workspace/internal/infrastructure/sqlite/migrations/000009_workspace_governance.up.sql` (new)
- `backend/internal/modules/workspace/internal/infrastructure/sqlite/migrations/000009_workspace_governance.down.sql` (new)
- `backend/internal/modules/workspace/sqlite_workspace_services_test.go`
- `backend/internal/modules/workspace/sqlite_persistence_test.go`
- `backend/docs/plans/product-capability-roadmap/task-register.md`
- `backend/docs/plans/product-capability-roadmap/journal.md`

No runtime composition, repository, application service, HTTP, frontend,
Control Plane, generated, or `server/**` path is authorized by r005.

### Frozen acceptance

1. Governance contract values reject empty identity/action/resource data,
   invalid revisions and states, and oversized replay/audit/outbox payloads.
2. Stable errors cover revision conflict, idempotency conflict, oversized replay
   response, unavailable provider, and stale outbox claim.
3. Migration 9 creates the four Workspace-owned tables with no FK, cascade,
   trigger, explicit index, or secondary unique-index DDL.
4. Fresh install, retained version-8 upgrade, second migration run, failure
   rollback, restart persistence, and workspace isolation are proven.
5. No runtime route, worker, event behavior, or feature flag changes.

### Deterministic verification

```text
cd backend && go test ./internal/modules/workspace/contract
cd backend && go test ./internal/modules/workspace -run 'Test.*(Governance|Migration|Migrate)'
cd backend && go test -race ./internal/modules/workspace/contract ./internal/modules/workspace
cd backend && make check
git diff --check
git diff --name-only -- server
```

The Windows loader code `0xc0000139` remains an environment limitation, not a
race PASS. Existing attachment concurrency failures may not be hidden or
weakened by this task.

### Current evidence

- Product candidate commit:
  `3876791` (`feat(workspace): add governance contracts and schema`).
- Contract and focused Workspace migration tests pass.
- Changed-file formatting, generated-output cleanliness, policy check, and
  `go vet ./...` pass.
- Technical closure is blocked because the frozen Windows race command exits
  `0xc0000139`, the Makefile `fmt-check` wrapper fails before equivalent checks,
  and full `go test ./...` reproduces out-of-scope attachment/auth SQLite
  concurrency failures.
- At the time the r005 candidate evidence was recorded, PCR-S01B-2 remained
  inactive.

The Human Customer subsequently confirmed that starting S01B-2 constitutes
S01B-1 acceptance and explicit waiver of the Windows race loader failure, the
Makefile Windows wrapper failure, and the three recorded out-of-scope full-suite
concurrency failures. The underlying evidence remains unchanged and is not a
technical PASS.

## PCR-001-S01B2-R6

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-S01B2-R6`
- Task-Revision: `r006`
- Work-Item: `PCR-S01B-2`
- Title: `SQLite mutation governance`
- Status: `customer-accepted-with-explicit-gate-waiver`
- Assignee: `Codex primary agent`
- Independent reviewer: `required before S01B final acceptance`
- Product-code base commit: `3876791`
- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Plan-Version: `v2`
- Plan-Step: `PCR-S01B-2`
- Policy bundle: the frozen v2 hashes above.

### Exact allowed paths

- `backend/internal/modules/workspace/internal/application/governance_service.go` (new)
- `backend/internal/modules/workspace/internal/application/governance_service_test.go` (new)
- `backend/internal/modules/workspace/internal/infrastructure/sqlite/governance_repository.go` (new)
- `backend/internal/modules/workspace/internal/infrastructure/sqlite/governance_repository_test.go` (new)
- `backend/internal/modules/workspace/governance.go` (new)
- `backend/internal/modules/workspace/governance_test.go` (new)
- `backend/docs/plans/product-capability-roadmap/task-register.md`
- `backend/docs/plans/product-capability-roadmap/journal.md`

No contract, migration, existing capability repository, runtime composition,
HTTP, frontend, Control Plane, generated, or `server/**` path is authorized by
r006. Discovery of a required path outside this list stops implementation and
requires a new task revision.

### Frozen acceptance

1. Concurrent mutations with one expected revision produce one commit and one
   revision conflict containing the current revision.
2. Same workspace/action/key/hash replays the original response without a
   second domain mutation, audit row, outbox row, or revision advance.
3. Same key with a different request hash returns idempotency conflict.
4. Test aborts after domain, revision, audit, outbox, and replay phases leave no
   partial committed state.
5. Workspace/action isolation and audit allowlisting/redaction pass.
6. No existing repository or runtime route is retrofitted.

### Deterministic verification

```text
cd backend && go test ./internal/modules/workspace/internal/application -run Governance
cd backend && go test ./internal/modules/workspace/internal/infrastructure/sqlite -run Governance
cd backend && go test ./internal/modules/workspace -run Governance
cd backend && go test -race ./internal/modules/workspace/internal/application ./internal/modules/workspace/internal/infrastructure/sqlite ./internal/modules/workspace
cd backend && make check
git diff --check
git diff --name-only -- server
```

Windows `0xc0000139` is not a race PASS. Previously waived full-suite failures
remain indexed and may not be hidden or weakened.

### Current evidence

- Product candidate commit: `a9f04b0`.
- Focused Governance tests and the complete application, SQLite infrastructure,
  and Workspace module test packages pass.
- Concurrent expected-revision mutation, deterministic replay/hash conflict,
  five-phase rollback, workspace/action isolation, audit allowlisting, and
  cross-workspace envelope rejection pass.
- Changed-file formatting, generated-output cleanliness, policy boundary, and
  `go vet ./...` pass through direct Windows-compatible commands.
- Technical closure remains blocked:
  - the frozen race command exits with Windows loader code `0xc0000139` for all
    three packages and is not a PASS;
  - `make check` stops in the known Windows `fmt-check` wrapper with
    `exit was unexpected at this time`;
  - full `go test ./...` reproduces the out-of-scope existing failures
    `TestSQLiteRuntimeConcurrentAttachmentUploadsLoseNoReferencesOrFiles`
    (attachment 500) and
    `TestAttachmentRepositoryRetriesBusyWriteAcquisition`
    (context deadline exceeded).
- No existing capability repository or runtime composition was changed.
- Independent review and Human Customer acceptance remain outstanding;
  `PCR-S01B-3` is inactive.

The Human Customer subsequently confirmed that starting S01B-3 constitutes
S01B-2 acceptance and explicit waiver of the Windows race loader failure, the
Makefile Windows wrapper failure, and the two recorded out-of-scope attachment
concurrency failures. Independent review is deferred to S01B-4. The underlying
technical evidence remains unchanged and is not represented as a PASS.

## PCR-001-S01B3-R7

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-S01B3-R7`
- Task-Revision: `r007`
- Work-Item: `PCR-S01B-3`
- Title: `Outbox delivery and governance diagnostics`
- Status: `candidate-committed-verification-blocked`
- Assignee: `Codex primary agent`
- Independent reviewer: `deferred to PCR-S01B-4 by Human Customer confirmation`
- Product-code base commit: `40d65ad`
- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Plan-Version: `v2`
- Plan-Step: `PCR-S01B-3`
- Policy bundle: the frozen v2 hashes above.

### Exact allowed paths

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
- `backend/docs/plans/product-capability-roadmap/task-register.md`
- `backend/docs/plans/product-capability-roadmap/journal.md`

No migration, contract, existing capability repository, feature API, frontend,
Control Plane, generated, or `server/**` path is authorized by r007. Discovery
of a required path outside this list stops implementation and requires a new
task revision.

### Frozen acceptance

1. Claim batches enforce default 100 and hard cap 500 with 60-second leases.
2. Stale claim tokens cannot ack, retry, dead-letter, or replay an event.
3. Initial delivery plus three retries, deterministic jitter, dead-letter,
   operator-authorized replay, restart, and lease reclaim pass.
4. Publish/ack crash windows may redeliver only the same stable event ID and
   aggregate revision.
5. `GET /api/operations/governance` is Workspace-scoped, owner/admin-only, and
   returns counts/timestamps without payload or audit content.
6. Database/provider unavailability keeps readiness honest; backlog older than
   15 minutes is reported degraded without restarting the process.
7. Explicit Canonical composition owns worker start/stop and no product feature
   flag becomes true.

### Deterministic verification

```text
cd backend && go test ./internal/modules/workspace/internal/application -run Outbox
cd backend && go test ./internal/modules/workspace/internal/infrastructure/sqlite -run 'Governance|Outbox'
cd backend && go test ./internal/modules/workspace/internal/interfaces/http -run Governance
cd backend && go test ./internal/modules/workspace ./internal/bootstrap -run Governance
cd backend && go test -race ./internal/modules/workspace/internal/application ./internal/modules/workspace/internal/infrastructure/sqlite ./internal/modules/workspace/internal/interfaces/http ./internal/modules/workspace ./internal/bootstrap
cd backend && make check
git diff --check
git diff --name-only -- server
```

Windows `0xc0000139` is not a race PASS. The Customer waiver closing r006 does
not silently waive r007 verification. Existing attachment failures remain
indexed and may not be hidden or weakened.

### Current evidence

- Product candidate commit: `f0d86d9`.
- Focused Outbox/Governance tests pass for application, SQLite infrastructure,
  HTTP, Workspace composition, and Bootstrap runtime.
- Claim exclusivity, 60-second lease reclaim, stale-token rejection, default
  batch 100/hard cap 500, four total attempts, deterministic jitter,
  dead-letter, operator-authorized replay, restart persistence, and stable
  crash-window redelivery pass.
- Owner/admin-only diagnostics, workspace isolation, payload/audit redaction,
  15-minute degradation reporting, explicit worker start/stop, readiness, and
  stable realtime delivery identity pass.
- Empty outbox polling is read-only and does not acquire a write lock.
- Changed-file formatting, generated-output cleanliness, policy boundary, and
  `go vet ./...` pass through direct Windows-compatible commands.
- Technical closure remains blocked:
  - the frozen race command exits with Windows loader code `0xc0000139` for all
    five packages and is not a PASS;
  - `make check` stops in the known Windows `fmt-check` wrapper;
  - the complete Bootstrap package and full `go test ./...` reproduce
    `TestSQLiteRuntimeConcurrentAttachmentUploadsLoseNoReferencesOrFiles`
    (attachment 500);
  - full `go test ./...` also reproduces
    `TestAttachmentRepositoryRetriesBusyWriteAcquisition`
    (context deadline exceeded).
- All roadmap product feature flags remain false. No migration, contract,
  feature API, frontend, Control Plane, generated, or `server/**` path changed.
- Human Customer acceptance remains outstanding; independent review remains
  deferred to S01B-4 and `PCR-S01B-4` is inactive.

## Queue rules

- No task after PCR-S01B-3 may be enqueued until r007 closes and the next task
  revision freezes its exact base, paths, checks, and active-step pointer.
- A task status in this file is not a substitute for the active-step pointer in
  `plan.md`; both must agree.
- A base, plan hash, policy hash, or allowed-path mismatch stops execution.
- Completion by the assignee is not independent acceptance.
- Customer Acceptance is not delegated to this register.
