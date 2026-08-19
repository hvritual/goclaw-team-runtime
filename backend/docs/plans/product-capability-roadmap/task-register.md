# Product Capability Roadmap Task Register

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Plan-Version: `v29`
- Registry status: `Release 3 active; PCR-S07B is the sole active task`
- Registry revision: `r034`
- Updated: `2026-08-19`

## Frozen policy bundle

| Policy source | SHA256 |
| --- | --- |
| `CLAUDE.md` | `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646` |
| `backend/AGENTS.md` | `fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209` |
| `plan_v1.md` | `4351025652e88083b8ceb25a72488e9d568a445a5f0e3a8793650393086ad0b5` |
| `plan_v2.md` | `af0486c7625067b2b25de163271ba6f4be153c74331488eb57d9c42ab69ffd30` |
| `plan_v3.md` | `81cd93e56ff9d2d4c34e1e23133235395b7fcf1fb99e82b49aaf5bf993e2afe8` |
| `plan_v4.md` | `9f663571e5850f0e03d6ca8fc3551a23e99dc4a7d84e6dfaccd2cc257c9b9191` |
| `plan_v5.md` | `ab5b81056f26f842a1b4fa08d626928b6cfb802dfae9a2bf4e3c0f305c019e69` |
| `plan_v6.md` | `1cd1c9a68626fe6c2c70037059b0327a4fa10f37a9853cc9ced4fa7ec32a1849` |
| `plan_v7.md` | `c20d41dc6d7b61830aacba2c378fd80e96ad958ed5604d239fda0687c2062152` |
| `plan_v8.md` | `8c23420fd4fc5c6bc0ad12946a2c0ec6d3c71ec0ca95dbebc68cea262849c78b` |
| `plan_v9.md` | `50f46e32ea3658ef87e903c12d840011b86be43edd9305e7244a687a3e53a035` |
| `plan_v10.md` | `444861e85c41b30beeb0ba36f0d75ac54c95015db503ce1d2576088b7c9a2171` |
| `plan_v11.md` | `6177497d7d7e78b4d0b74b282245e01a50a8f737c9838a22b2ede1a17c4cb9b5` |
| `plan_v12.md` | `138219e9cc6710919b96b10332b5c56fbf287f1ee8430f2d1615df991efcbf32` |
| `plan_v13.md` | `830ad1b5c189d356b074e1b84cde9ca220b587bb24e322186d38a9dad9806013` |

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

## PCR-001-S01B3R-R8

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-S01B3R-R8`
- Task-Revision: `r008`
- Work-Item: `PCR-S01B-3R`
- Status: `customer-accepted`
- Assignee: `Codex primary agent`
- Independent reviewer: `deferred to PCR-S01B-4`
- Base commit: `ab2b49088b108f771045a090b473a8e235dfa09e`
- Approved plan: `PRODUCT-CAPABILITY-ROADMAP-001 v3`
- Plan hash: `81cd93e56ff9d2d4c34e1e23133235395b7fcf1fb99e82b49aaf5bf993e2afe8`
- Policy bundle: frozen hashes above

### Objective

Repair the Windows race loader, Windows `make check`, and the two indexed
attachment concurrency failures without changing S01B product behavior.

### Exact allowed paths

- `backend/Makefile`
- `backend/cmd/backend-check/main.go` (new)
- `backend/cmd/backend-check/main_test.go` (new)
- `backend/ci/test-race.ps1` (new)
- `backend/internal/modules/space/internal/infrastructure/sqlite/attachment_repository.go`
- `backend/internal/modules/space/internal/infrastructure/sqlite/attachment_repository_test.go`
- `backend/internal/bootstrap/issue_attachment_runtime_test.go`
- `backend/docs/plans/product-capability-roadmap/plan.md`
- `backend/docs/plans/product-capability-roadmap/plan_v3.md` (new)
- `backend/docs/plans/product-capability-roadmap/task-register.md`
- `backend/docs/plans/product-capability-roadmap/journal.md`

No migration, public contract, feature API, frontend, Desktop, Control Plane,
generated, permanent host configuration, unrelated dirty, or `server/**` path
is authorized. A required path outside this list stops implementation.

### Frozen acceptance and verification

The acceptance criteria, ordered steps, commands, risks, and rollback in
`plan_v3.md` are frozen without exception. Race and attachment failures may not
be retried, hidden, skipped, quarantined, or weakened into passing evidence.

### Current evidence

- The two pre-existing intermittent attachment tests passed their single RED
  probes, so they were not treated as deterministic TDD RED evidence.
- New deterministic tests failed before implementation because a queued writer
  consumed a second SQLite connection; cancellation also remained blocked in
  SQLite busy handling for about five seconds.
- The minimal repair adds one context-aware attachment write slot per repository
  before connection acquisition. Create and delete retain their existing
  transaction, rollback, binding, cleanup, and event boundaries.
- Repository busy/cancellation/serialization tests pass ten consecutive counts;
  the twelve-upload Bootstrap regression passes ten consecutive counts.
- `go test ./... -count=1`, `make check`, `go mod verify`, Backend-check tests,
  formatting, policy, generated-output, vet, diff, and empty `server/**` checks
  pass.
- The frozen five-package race command executes and passes with the already
  installed Scoop GCC 15.2 selected process-locally. No `0xc0000139` waiver is
  used and no permanent PATH change is made.
- No migration, public contract, product flag, frontend, Desktop, Control Plane,
  generated output, unrelated dirty, or `server/**` path changed.
- Human Customer accepted the repair on 2026-08-17. No commit, push, merge, or
  independent review is claimed, and S01B-4 remains inactive.

## Queue rules

- No task after PCR-S01B-3R may execute until the next task revision freezes its
  exact base, paths, checks, and active-step pointer. Customer Acceptance closes
  r008 but does not activate PCR-S01B-4.
- A task status in this file is not a substitute for the active-step pointer in
  `plan.md`; both must agree.
- A base, plan hash, policy hash, or allowed-path mismatch stops execution.
- Completion by the assignee is not independent acceptance.
- Customer Acceptance is not delegated to this register.

## PCR-001-S01B4-R9

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-S01B4-R9`
- Task-Revision: `r009`
- Work-Item: `PCR-S01B-4`
- Status: `independent-review-blocked`
- Assignee: `Codex primary agent`
- Independent reviewer: `Codex independent read-only reviewer; SPEC BLOCK`
- Base commit: `ab2b49088b108f771045a090b473a8e235dfa09e`
- Approved plan: `PRODUCT-CAPABILITY-ROADMAP-001 v4`
- Plan hash: `9f663571e5850f0e03d6ca8fc3551a23e99dc4a7d84e6dfaccd2cc257c9b9191`
- Policy bundle: frozen hashes above

### Objective

Commit the accepted r008 repair, verify the complete Release 0 candidate,
obtain independent read-only review, and close Release 0 without activating
Release 1.

### Exact allowed paths

- `backend/Makefile`
- `backend/cmd/backend-check/main.go`
- `backend/cmd/backend-check/main_test.go`
- `backend/ci/test-race.ps1`
- `backend/internal/modules/space/internal/infrastructure/sqlite/attachment_repository.go`
- `backend/internal/modules/space/internal/infrastructure/sqlite/attachment_repository_test.go`
- `backend/docs/plans/product-capability-roadmap/plan.md`
- `backend/docs/plans/product-capability-roadmap/plan_v3.md`
- `backend/docs/plans/product-capability-roadmap/plan_v4.md`
- `backend/docs/plans/product-capability-roadmap/story-map.md`
- `backend/docs/plans/product-capability-roadmap/task-register.md`
- `backend/docs/plans/product-capability-roadmap/journal.md`

No other product, frontend, Desktop, Control Plane, generated, unrelated dirty,
or `server/**` path is authorized.

### Frozen acceptance and verification

Sections 5 through 9 of `plan_v4.md` are frozen without exception. The first
commit explicitly stages only the accepted repair and activation records. All
deterministic gates run on that committed candidate before independent review.
Any independent `BLOCK`, scope drift, policy drift, or newly required code path
stops closure and requires a new approved plan version.

### Authority

The Human Customer stated `批准后续动作，按目标持续推进完成 Release 0 — Authority
and safety foundation` on 2026-08-17. This authorizes v4/r009, the scoped Git
commits, and an independent read-only reviewer. It does not authorize push,
merge, deployment, Release 1 activation, or out-of-plan defect repair.

### Current evidence and disposition

- Candidate commit `5062e84a65a3ce3114a0d2d54013d37f746836c6`
  contains exactly the twelve v4/r009 paths and machine-readable traceability
  trailers. Its tree is identical to the pre-audit candidate `b24661f`.
- All frozen deterministic gates passed on that candidate: focused S01A/S01B
  tests, five S01B packages, ten-count attachment contention and concurrent
  upload regressions, full Backend tests, vet, module verification, Make checks,
  five-package race, supplemental S01A race, diff, links, and `server/**` scope.
- Independent contract review examined `3876791^..5062e84` after deterministic
  checks and returned `SPEC BLOCK` in five areas:
  1. unknown action/resource mappings are not rejected by a server-side
     authority registry;
  2. idempotency trusts a caller-provided hash instead of computing the frozen
     versioned canonical JSON SHA-256 projection;
  3. replay, audit-value, and outbox payload validation does not enforce the
     frozen version/allowlist/forbidden-value secret boundary;
  4. outbox acknowledgement/failure accepts claim-time timestamps and partial
     identity instead of a current lease check plus the complete PK tuple;
  5. migration 000009 down SQL drops non-empty governance evidence tables.
- The primary agent reproduced each finding against the frozen design and
  candidate source. Existing tests omit all five negative cases.
- Transaction atomicity/rollback, HTTP diagnostics authorization/redaction,
  retry/dead-letter/restart/readiness behavior, candidate scope, and the
  `server/**` boundary passed their reviewed subdimensions.
- Per v4, any `SPEC BLOCK` prevents closure. Release 0 remains incomplete and
  product repair requires a new approved plan version and frozen task revision.

## PCR-001-S01B5-R10

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-S01B5-R10`
- Task-Revision: `r010`
- Work-Item: `PCR-S01B-5`
- Status: `independent-review-blocked`
- Assignee: `Codex primary agent`
- Independent reviewer: `SPEC BLOCK on candidate ee02403`
- Base commit: `0218ecbe5457f1afb716780ad44306e5b1b3b075`
- Approved plan: `PRODUCT-CAPABILITY-ROADMAP-001 v5`
- Plan hash: `ab5b81056f26f842a1b4fa08d626928b6cfb802dfae9a2bf4e3c0f305c019e69`
- Policy bundle: frozen hashes above

### Objective

Repair all five S01B-4 independent-review blocks under the frozen v5 design,
then produce a new Release 0 candidate for deterministic and independent
acceptance.

### Exact allowed paths

The exact writable path list is Section 6 of `plan_v5.md`. It is incorporated
into r010 without expansion. No up migration, public API, capability behavior,
frontend, Desktop, Control Plane, generated, unrelated dirty, or `server/**`
path is authorized.

### Frozen execution and acceptance

- Execute Sections 5.1 through 5.6 in order using RED/GREEN TDD.
- Sections 3, 7, and 8 are frozen design, acceptance, and verification.
- Retain and fail closed on legacy unversioned rows; do not clean, backfill,
  publish, or delete them.
- Any required path outside Section 6, non-empty data action, up-schema change,
  or independent `BLOCK` stops execution and requires a new approved plan.

### Authority

The Human Customer explicitly approved `PRODUCT-CAPABILITY-ROADMAP-001 v5 /
r010` on 2026-08-17. This activates `PCR-S01B-5` only and does not authorize
push, merge, deployment, or Release 1.

### Independent review outcome

- Candidate `ee02403cbed00366a5c25bfd4da8d2ee123cb675` passed the frozen
  deterministic gates and exact-scope audit.
- Independent review passed authority, canonical hashing, opaque preparation,
  complete outbox tuple/current lease, empty-only down, and scope boundaries.
- Review blocked closure because deprecated raw response/audit/outbox fields
  are silently ignored, Basic authorization material is not universally
  rejected, and retained unknown-policy/unversioned outbox rows may be changed
  by claim or replay before exact policy validation.
- Per v5, r010 cannot repair a post-gate independent block. A new approved plan
  version and task revision are required. Release 0 remains incomplete.

## PCR-001-S01B6-R11

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-S01B6-R11`
- Task-Revision: `r011`
- Work-Item: `PCR-S01B-6`
- Status: `independent-review-blocked`
- Assignee: `Codex primary agent`
- Independent reviewer: `required after deterministic verification`
- Base commit: `f93eca764bb464245ef096429701aa0a856f0c56`
- Approved plan: `PRODUCT-CAPABILITY-ROADMAP-001 v6`
- Plan hash: `1cd1c9a68626fe6c2c70037059b0327a4fa10f37a9853cc9ced4fa7ec32a1849`
- Policy bundle: `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646/fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209/1cd1c9a68626fe6c2c70037059b0327a4fa10f37a9853cc9ced4fa7ec32a1849`

### Objective

Close the three v5 independent-review blocks and policy snapshot debt without
changing schema or activating Release 1, then produce a new Release 0 candidate.

### Frozen execution and acceptance

- Execute Sections 5.1 through 5.6 of `plan_v6.md` using RED/GREEN TDD.
- Remove unrestricted raw inputs; never silently ignore or filter them.
- Validate exact event policy before the first claim/replay write; invalid rows
  remain unchanged and an invalid claim batch fails atomically.
- No schema, external data, frontend, Control Plane, generated, unrelated dirty,
  or `server/**` modification is authorized.
- Any required path outside Section 6 or any independent `BLOCK` stops closure
  and requires a new approved plan/task.

### Authority

The Human Customer explicitly approved `PRODUCT-CAPABILITY-ROADMAP-001 v6 /
r011` on 2026-08-17. This activates only `PCR-S01B-6`; it does not authorize
push, merge, deployment, Release 1, schema changes, or external data handling.

### Independent review outcome

- Candidate `4d60e50d9c03a68b2723b427506ea7db64d90d90` passed every v6
  implementation, deterministic, race, attachment-concurrency, policy, scope,
  and `server/**` gate.
- Independent review passed implementation SPEC and code quality but blocked
  authority/traceability because immutable `plan_v6.md` names nonexistent base
  `f93eca77c3450109b7328441812d63710f179521` instead of the actual registered
  base `f93eca764bb464245ef096429701aa0a856f0c56`.
- Per v6, Release 0 remains incomplete and repair requires a new approved plan.

## PCR-001-S01B7-R12

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-S01B7-R12`
- Task-Revision: `r012`
- Work-Item: `PCR-S01B-7`
- Status: `independent-review-blocked`
- Assignee: `Codex primary agent`
- Independent reviewer: `required after exact v7 evidence verification`
- Base commit: `4d60e50d9c03a68b2723b427506ea7db64d90d90`
- Approved plan: `PRODUCT-CAPABILITY-ROADMAP-001 v7`
- Plan hash: `c20d41dc6d7b61830aacba2c378fd80e96ad958ed5604d239fda0687c2062152`
- Policy bundle: `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646/fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209/c20d41dc6d7b61830aacba2c378fd80e96ad958ed5604d239fda0687c2062152`

### Objective

Repair only the immutable v6 authority-record mismatch through a new internally
consistent approved snapshot, obtain fresh independent review, and close
Release 0 without changing the verified product candidate.

### Frozen execution and acceptance

- Execute Sections 5.1 through 5.4 of `plan_v7.md` in order.
- Modify only the five documentation paths in Section 6; prior immutable plans
  remain byte-for-byte unchanged.
- Candidate `4d60e50` is the exact base and no product/test/schema/runtime change
  is authorized.
- Any independent BLOCK stops closure and requires a new approved plan/task.
- Release 1 remains inactive through activation and closure.

### Authority

The Human Customer explicitly approved `PRODUCT-CAPABILITY-ROADMAP-001 v7 /
r012` on 2026-08-17. This authorizes only documentation authority repair,
read-only verification, independent review, and Release 0 closure records after
PASS; it does not authorize product changes, push, merge, deployment, or
Release 1.

### Independent review outcome

- v7 base/object/parent/ancestry, immutable v6 preservation, unchanged product
  candidate, exact five-path scope, trailers, hashes, links, focused tests,
  dirty exclusions, `server/**`, and Release 1 inactivity all passed.
- Review blocked authority uniqueness because r011 and r012 were simultaneously
  marked active. This could let a recovery process resume obsolete v6 authority.
- Per v7, Release 0 remains incomplete and repair requires a new approved plan.

## PCR-001-S01B8-R13

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-S01B8-R13`
- Task-Revision: `r013`
- Work-Item: `PCR-S01B-8`
- Status: `complete-independent-reviewed`
- Assignee: `Codex primary agent`
- Independent reviewer: `SPEC PASS on activation 872b5ebe35e1d12a488e389a713d377a4c1663cc`
- Base commit: `14908b9e53a73330c0cde3fe8a3f602635906858`
- Approved plan: `PRODUCT-CAPABILITY-ROADMAP-001 v8`
- Plan hash: `8c23420fd4fc5c6bc0ad12946a2c0ec6d3c71ec0ca95dbebc68cea262849c78b`
- Policy bundle: `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646/fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209/8c23420fd4fc5c6bc0ad12946a2c0ec6d3c71ec0ca95dbebc68cea262849c78b`

### Objective

Restore exactly one active task authority, obtain fresh independent review, and
close Release 0 without changing the verified product candidate.

### Frozen execution and acceptance

- Execute Sections 5.1 through 5.4 of `plan_v8.md` in order.
- r011 and r012 remain review-blocked; r013 is the only active task before
  closure and no task remains active after closure.
- Modify only the five documentation paths in Section 6; prior plans and product
  code remain unchanged.
- Any independent BLOCK stops closure and requires a new approved plan/task.
- Release 1 remains inactive through activation and closure.

### Authority

The Human Customer explicitly approved `PRODUCT-CAPABILITY-ROADMAP-001 v8 /
r013` on 2026-08-18. This authorizes only single-authority documentation repair,
read-only verification, independent review, and Release 0 closure records after
PASS; it does not authorize product changes, push, merge, deployment, or
Release 1.

### Independent review and closure outcome

- Activation `872b5ebe35e1d12a488e389a713d377a4c1663cc` has direct parent
  `14908b9e53a73330c0cde3fe8a3f602635906858`; its range is one commit and
  exactly the five Section 6 documentation paths.
- Mechanical enumeration found exactly one active task before closure: r013.
  r011 and r012 were review-blocked; all prior immutable plans were unchanged.
- Plan pointer, story map, task register, and journal agreed on authority and
  Release state. Hashes, trailers, links, diff checks, focused tests, unrelated
  dirty exclusions, and `server/**` all passed.
- Independent review returned `SPEC PASS` across implementation, quality,
  authority/traceability, scope, Release 0 readiness, and Release 1 inactivity.
- r013 and Release 0 are complete. Release 1 remains inactive and no task is
  active after closure.

## PCR-001-S02A-R14

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-S02A-R14`
- Task-Revision: `r014`
- Work-Item: `PCR-S02A`
- Status: `verification-blocked`
- Assignee: `Codex primary agent`
- Independent reviewer: `required after deterministic verification`
- Base commit: `628996378af6fbe12c27a916a624a5f5374a884f`
- Approved plan: `PRODUCT-CAPABILITY-ROADMAP-001 v9`
- Plan hash: `50f46e32ea3658ef87e903c12d840011b86be43edd9305e7244a687a3e53a035`
- Policy bundle: `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646/fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209/50f46e32ea3658ef87e903c12d840011b86be43edd9305e7244a687a3e53a035`

### Objective

Install the complete `PCR-S02A — Manage tasks` vertical slice through the
Canonical Workspace runtime and shared Web/Desktop product surface. S02B and
all later Release 1 stories remain inactive.

### Frozen execution and acceptance

- Execute `plan_v9.md` Sections 6.1 through 6.6 in order using RED/GREEN TDD.
- Preserve the frozen Workspace authority, Task route family, permissions,
  lifecycle, revision conflict, schema parsing, and governance invariants.
- Modify only Section 4 paths and preserve every Section 5 exclusion.
- Any required out-of-scope path, deterministic failure, independent `BLOCK`,
  or `server/**` path stops closure and requires a new approved plan/task.
- S02A completion does not activate S02B or complete Release 1.

### Authority

The Human Customer directed Codex to continue approved actions until Release 1
is complete on 2026-08-18. This activates only v9/r014/PCR-S02A from exact base
`6289963`; it does not authorize S02B, later stories, push, merge, deployment,
release tags, external systems, mobile, or `server/**` changes.

### Activation evidence

- Release 0 closure at `6289963` is the direct frozen base.
- Prior `plan_v1.md` through `plan_v8.md` remain immutable.
- The pre-activation task count is zero; r014 becomes the sole active task.
- Existing user/workspace dirty and untracked paths are recorded in v9 Section
  5 and excluded from implementation.
- Product verification and independent review are not yet claimed.

### Verification outcome

- S02A product implementation reached exact candidate `9065802` and passed
  backend full checks, real race, Core 593/593, Task Views 8/8, four frontend
  typechecks, clean-candidate installed Playwright 1/1, and fresh independent
  review.
- Frozen root verification remained deterministically blocked by the stale
  nonexistent `@multica/mobile` filter; direct Views execution exposed four
  verification-baseline failures. v9 requires a new task/plan rather than a
  waiver, so r014 is verification-blocked and superseded by r015.

## PCR-001-S02A-R15

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-S02A-R15`
- Task-Revision: `r015`
- Work-Item: `PCR-S02A`
- Status: `verification-blocked`
- Assignee: `Codex primary agent`
- Independent reviewer: `required after deterministic verification`
- Base commit: `906580292e6fbcd9a0d866cae796d0b67cfd975d`
- Approved plan: `PRODUCT-CAPABILITY-ROADMAP-001 v10`
- Plan hash: `444861e85c41b30beeb0ba36f0d75ac54c95015db503ce1d2576088b7c9a2171`
- Policy bundle: `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646/fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209/444861e85c41b30beeb0ba36f0d75ac54c95015db503ce1d2576088b7c9a2171`

### Objective

Repair only the deterministic repository verification baseline needed to close
S02A without changing Task product behavior or activating S02B.

### Frozen execution and acceptance

- Execute `plan_v10.md` Sections 6 through 8 in order using the captured root
  and Views failures as RED evidence.
- Modify only v10 Section 4 paths and preserve all v10 Section 5 exclusions.
- Root `pnpm typecheck`, root `pnpm test`, backend full/race checks, exact Task
  browser acceptance, and fresh independent review must all pass.
- S02A closure does not activate S02B or complete Release 1.

### Activation evidence

- Exact base `9065802` preserves the independently reviewed S02A product PASS.
- r014 is verification-blocked; r015 is the sole active task.
- The Human Customer approved the follow-up scope on 2026-08-18.
- No verification repair or closure PASS is claimed at activation.

### Verification outcome

- v10 focused Views tests pass 131/131 and root typecheck passes across the
  current workspace graph at candidate `2aac82e`.
- Root test reaches Desktop but deterministically fails on a transformed
  shebang and incomplete ignored Electron binary state. v10 excludes both, so
  r015 is verification-blocked rather than waived and is superseded by r016.

## PCR-001-S02A-R16

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-S02A-R16`
- Task-Revision: `r016`
- Work-Item: `PCR-S02A`
- Status: `verification-blocked`
- Assignee: `Codex primary agent`
- Independent reviewer: `required after deterministic verification`
- Base commit: `2aac82e742932ddcda2ab6fa1387a9960a530974`
- Approved plan: `PRODUCT-CAPABILITY-ROADMAP-001 v11`
- Plan hash: `6177497d7d7e78b4d0b74b282245e01a50a8f737c9838a22b2ede1a17c4cb9b5`
- Policy bundle: `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646/fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209/6177497d7d7e78b4d0b74b282245e01a50a8f737c9838a22b2ede1a17c4cb9b5`

### Objective and acceptance

- Remove only the redundant Desktop package-script shebang and reconstruct the
  locked Electron runtime from the already-present local cache.
- Focused Desktop tests, root typecheck/test, backend full/race, exact Task
  browser acceptance, scope checks, and fresh independent review must pass.
- Any network fallback or additional tracked path stops execution.
- S02A closure does not activate S02B or complete Release 1.

### Activation evidence

- Exact base is `2aac82e`; r015 is verification-blocked and r016 is sole active.
- Locked Electron 39.8.7 archive exists in the local cache.
- No repair, verification PASS, or closure is claimed at activation.

### Verification outcome

- Focused package and endpoint tests pass 30/30 and 4/4 after v11 repair.
- Root test reaches Desktop but three real-Git cases exceed the default
  five-second limit only under full-workspace parallel load. r016 is therefore
  verification-blocked and superseded by the narrowly scoped r017.

## PCR-001-S02A-R17

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-S02A-R17`
- Task-Revision: `r017`
- Work-Item: `PCR-S02A`
- Status: `verification-blocked`
- Assignee: `Codex primary agent`
- Independent reviewer: `required after deterministic verification`
- Base commit: `3c030b4a6f8e9d47e1029d3c3245733768644415`
- Approved plan: `PRODUCT-CAPABILITY-ROADMAP-001 v12`
- Plan hash: `138219e9cc6710919b96b10332b5c56fbf287f1ee8430f2d1615df991efcbf32`
- Policy bundle: `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646/fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209/138219e9cc6710919b96b10332b5c56fbf287f1ee8430f2d1615df991efcbf32`

### Objective and acceptance

- Give only the three process-heavy real-Git package tests an explicit
  15-second timeout; preserve every assertion and command.
- Root typecheck/test, backend full/race, exact Task browser acceptance, scope
  checks, and fresh independent review must pass before S02A closure.
- S02B and later Release 1 stories remain inactive.

### Activation evidence

- Exact base is `3c030b4`; r016 is verification-blocked and r017 is sole active.
- Isolated package tests pass 30/30; root-parallel timeout is the captured RED.
- No r017 implementation or closure PASS is claimed at activation.

### Verification outcome

- Root execution passes 422/423 Desktop tests after the 15-second budget.
- One multi-step real-Git scenario still exceeds that ceiling under full graph
  load. r017 is verification-blocked and superseded by r018 without waiver.

## PCR-001-S02A-R18

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-S02A-R18`
- Task-Revision: `r018`
- Work-Item: `PCR-S02A`
- Status: `complete-independent-reviewed`
- Assignee: `Codex primary agent`
- Independent reviewer: `PASS on bc1eed7; closure allowed`
- Base commit: `8f94d69db72e18719b7ed4315e7d74901b1f5945`
- Approved plan: `PRODUCT-CAPABILITY-ROADMAP-001 v13`
- Plan hash: `830ad1b5c189d356b074e1b84cde9ca220b587bb24e322186d38a9dad9806013`
- Policy bundle: `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646/fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209/830ad1b5c189d356b074e1b84cde9ca220b587bb24e322186d38a9dad9806013`

### Objective and acceptance

- Change only the three real-Git timeout literals from 15 to 60 seconds.
- Root typecheck/test, backend full/race, exact browser, scope, and independent
  review must pass before S02A closure.
- S02B and all later stories remain inactive.

### Verification and review outcome

- Implementation candidate: `bc1eed7f1ccb92a0bd31a769403c8f7e5139cf21`.
- Isolated Desktop package tests pass 30/30. Exact candidate root `pnpm test`
  passes 5/5 Turbo tasks; clean-candidate Views passes 165 files and 1671 tests.
- Exact candidate root `pnpm typecheck` passes 6/6 tasks. Backend `make check`
  and real `make test-race` pass with MinGW GCC 15.2.0.
- Detached clean-candidate installed Task lifecycle Playwright passes 1/1 with
  Chrome, covering create, filter, reorder, archive, restore, and readback.
- Scope checks pass: `server/**` is empty, the excluded Input blob remains
  `a830fd2f0f82770563908d512558fe6ba48f50dd`, and the only candidate dirt is
  Next development-server configuration drift in the detached worktree.
- Fresh independent review returned PASS for v10-v13 authority, exact repair
  scope, unchanged assertions/retries/concurrency, deterministic evidence, and
  S02A closure readiness.
- r018 and PCR-S02A are complete. Release 1 remains active with no active task;
  S02B awaits a new versioned plan.

## PCR-001-S02B-R19

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-S02B-R19`
- Task-Revision: `r019`
- Work-Item: `PCR-S02B`
- Status: `independent-review-blocked`
- Assignee: `Codex primary agent`
- Independent reviewer: `required after deterministic verification`
- Base commit: `3262232c5000ed449b89d98901535cd58b42a48d`
- Approved plan: `PRODUCT-CAPABILITY-ROADMAP-001 v14`
- Plan hash: `a2517baf25a3e4312c12f5abee9ec7917ae972a16802915ced9e9cd0b0f6fb27`
- Policy bundle: `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646/fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209/a2517baf25a3e4312c12f5abee9ec7917ae972a16802915ced9e9cd0b0f6fb27`

### Objective and acceptance

- Install explicit, idempotent Task-to-Issue promotion with an immutable source
  link and optional atomic completion.
- Prove exact replay, body conflict, revision/authorization denial, concurrent
  single winner, rollback, restart, no later synchronization, shared UI, and
  clean installed-browser acceptance.
- S03A and all later stories remain inactive.

### Independent review outcome

- Candidate `36b18b4afac2ca2ae65ced7ee329c726c0bed73b` passed deterministic
  and installed-browser checks but is not closable.
- Review blocked exact replay after live-row edits, reusable client
  idempotency keys, and strict nested Issue parsing.
- r019 is superseded for active execution only by v15/r020; its implementation
  and evidence remain immutable history.

## PCR-001-S02B-R20

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-S02B-R20`
- Task-Revision: `r020`
- Work-Item: `PCR-S02B`
- Status: `complete-via-r021-independent-review`
- Assignee: `Codex primary agent`
- Independent reviewer: `required after deterministic verification`
- Base commit: `36b18b4afac2ca2ae65ced7ee329c726c0bed73b`
- Approved plan: `PRODUCT-CAPABILITY-ROADMAP-001 v15`
- Plan hash: `f13b640a0c33aacfd47fe61bd2ae981acf4fc8781fcfe839f8814482e5881e02`
- Policy bundle: `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646/fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209/f13b640a0c33aacfd47fe61bd2ae981acf4fc8781fcfe839f8814482e5881e02`

### Objective and acceptance

- Persist the originally committed promotion response outside safe governance
  envelopes and replay it after later live-row edits or database reopen.
- Preserve one explicit client idempotency key across same-command retry and
  reject unknown or malformed nested Issue response fields.
- Re-run complete S02B verification and fresh independent review before
  closure. S03A and later stories remain inactive.

### Verification outcome

- Candidate `e97c92b23423a89d6d731ab8ba40ca9a80858b63` passes focused
  backend/Core/Views checks, forced root typecheck 6/6 with zero cache, and
  forced root tests 5/5 with zero cache; Views passes 165/165 files and
  1672/1672 tests after TEMP is placed on F due C-volume capacity.
- Backend `make check` reproducibly fails only
  `TestSQLiteRuntimeConcurrentAttachmentUploadsLoseNoReferencesOrFiles`.
  The same focused failure occurs on pre-r020 `36b18b4`; sanitized diagnostics
  identify the implicit one-second Kratos HTTP request deadline cancelling
  write-slot wait or commit contexts.
- v15 requires full gates with no waiver. r020 is verification-blocked and
  superseded for active execution only by v16/r021.
- Fresh v16/r021 independent review passed the unchanged r020 product
  remediation at exact candidate `5ea1a47`; r020 is closed through that
  verified successor without rewriting its v15 gate history.

## PCR-001-S02B-R21

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-S02B-R21`
- Task-Revision: `r021`
- Work-Item: `PCR-S02B`
- Status: `complete-independent-reviewed`
- Assignee: `Codex primary agent`
- Independent reviewer: `PASS on 5ea1a47; S02B closure allowed`
- Base commit: `e97c92b23423a89d6d731ab8ba40ca9a80858b63`
- Approved plan: `PRODUCT-CAPABILITY-ROADMAP-001 v16`
- Plan hash: `3575657bbe68972b25691d2b9f077038b802a39e5ad730e07708cd215af63e5b`
- Policy bundle: `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646/fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209/3575657bbe68972b25691d2b9f077038b802a39e5ad730e07708cd215af63e5b`

### Objective and acceptance

- Replace only the Canonical Runtime's implicit one-second Kratos HTTP request
  budget with an explicit 30-second budget while preserving cancellation and
  every attachment transaction invariant.
- Prove the installed transport deadline contract and the unchanged 12-writer
  attachment acceptance, then repeat all r020 deterministic, browser, scope,
  cleanup, and independent-review gates without waiver.
- S03A and all later stories remain inactive.

### Verification and review outcome

- Exact candidate: `5ea1a47f7f3395c4778398ee1a6e79f6880bf0a0`.
- RED captured the installed Kratos request deadline as one second and the
  unchanged 12-writer attachment acceptance failed with sanitized
  `context deadline exceeded`. GREEN proves an explicit 30-second deadline and
  the same 12 writers pass without reduced concurrency, retry, or attachment
  production changes.
- r020 backend replay/migration/runtime focused tests pass; Core passes 2 files
  and 7 tests; Task Views passes 1 file and 9 tests.
- Forced root typecheck passes 6/6 Turbo tasks with zero cache. Forced root
  tests pass 5/5 with zero cache; Views passes 165 files and 1672 tests.
- Backend `make check` passes. After `go clean -testcache`, real
  `make test-race` passes with MinGW GCC 15.2.0.
- A new r021-migrated SQLite database and canonical fixture pass installed
  Chrome `e2e/tasks.spec.ts` 1/1 in 16.2 seconds. The first browser attempt hit
  the known cold-route compile window before the Tasks heading appeared; after
  verifying the route returned 200 and contained Tasks, the zero-retry
  Playwright configuration passed on one explicit rerun.
- Scope passes: `server/**` range and worktree diffs are empty; excluded Input
  remains blob `a830fd2f0f82770563908d512558fe6ba48f50dd`; policy hashes match
  v16. Candidate-only `.local-runtime/**` and
  `node_modules.main-link-r019/**` are recorded untracked runtime/dependency
  artifacts, not candidate source.
- Candidate backend and Web process ancestry was resolved and stopped; ports
  3000, 8080, and 9080 are closed.
- Fresh independent read-only review returned PASS across authority, scope,
  exact replay, stable header-only idempotency, strict nested Issue parsing,
  transport cancellation, and supplied deterministic/browser evidence.
- r021, r020, and PCR-S02B are closed. Release 1 remains active with no active
  task; S03A requires a new approved plan.

## PCR-001-S03A-R22

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-S03A-R22`
- Task-Revision: `r022`
- Work-Item: `PCR-S03A`
- Status: `complete-independent-reviewed`
- Assignee: `Codex primary agent`
- Independent reviewer: `PASS on 7479e07; S03A closure allowed after blocker remediation`
- Base commit: `2439e9c2edd3c557e849fb695210b2eab95bbb9d`
- Approved plan: `PRODUCT-CAPABILITY-ROADMAP-001 v17`
- Plan hash: `f08ace3cbecf964df88b0f22e551fe60b2f133699fb22322244e17874d61fab1`
- Policy bundle: `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646/fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209/f08ace3cbecf964df88b0f22e551fe60b2f133699fb22322244e17874d61fab1`

### Objective and acceptance

- Install repository-backed ranked Issue search at `GET /api/issues/search`
  with deterministic Unicode normalization, identifier/number lookup,
  pagination, closed filtering, cancellation, and Workspace isolation.
- Prove projection synchronization and restart, a strict cancellable Core
  boundary, shared Web/Desktop consumption, the 10,000-Issue latency budget,
  and installed Chrome acceptance.
- Enable only `issue_search`; `project_search`, S03B, and S04 remain inactive.

### Verification and review outcome

- Exact candidate: `7479e07adcc1c703a52cb348af7919df2cc68553`.
- RED/GREEN covers Unicode normalization, ranking, pagination, closed state,
  Workspace isolation, cancellation, HTTP/auth boundaries, projection
  synchronization/restart, strict Core parsing, feature flags, the fixture
  binary UDF registration, browser cookie authentication, locale stability,
  and the independent-review Project-search loading-window blocker.
- The 10,000-Issue non-race acceptance measured p50/p95 at
  20.9999/23.1776ms and 25.3918/39.5394ms, within 100/250ms. Race execution
  retains the 10,000-row functional check without treating instrumented
  latency as production performance.
- Forced root typecheck passes 6/6 with zero cache. Forced root tests pass 5/5
  with zero cache and controlled concurrency; final Views passes 166 files and
  1675 tests. Exact-candidate backend `make check` and official
  `go clean -testcache; make test-race` pass with MinGW GCC 15.2.0.
- Exact-candidate production Web build passes. A fresh migrated SQLite database
  and fixture pass installed Chrome `e2e/issue-search.spec.ts` 1/1 with
  Playwright retries disabled, covering English/Chinese, identifier/number,
  closed filtering, pagination, isolation, and the shared UI.
- Fresh independent review first blocked Project search while config was still
  loading. Candidate `7479e07` makes Project search require loaded explicit
  installation evidence, adds the failing/passing regression, repeats all
  affected gates, and receives fresh independent PASS.
- `server/**` candidate scope is empty; policy hashes and the protected Input
  blob match v17. Candidate tracked state is clean apart from recorded
  `.local-runtime/**` and `node_modules.main-link-r019/**` environment
  artifacts; exact backend/Web processes were stopped and ports
  3000/8080/9080 are closed.
- r022 and PCR-S03A are closed. Release 1 remains active with no active task;
  S03B and S04 remain inactive pending a new approved plan.

## PCR-001-S03B-R23

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-S03B-R23`
- Task-Revision: `r023`
- Work-Item: `PCR-S03B`
- Status: `complete-independent-reviewed`
- Assignee: `Codex primary agent`
- Independent reviewer: `SPEC PASS and CODE QUALITY PASS on c9e905bc after blocker remediation`
- Base commit: `781471015ec8d759cd1209fd051e59fa91507eef`
- Approved plan: `PRODUCT-CAPABILITY-ROADMAP-001 v18`
- Plan hash: `95ff6b597430cd3d58b8937eb9dae32f1e72ec6e00bb10f99d08fd6a9a0e7789`
- Policy bundle: `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646/fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209/95ff6b597430cd3d58b8937eb9dae32f1e72ec6e00bb10f99d08fd6a9a0e7789`

### Objective and acceptance

- Install repository-backed Project search at `GET /api/projects/search`
  using the public Project surface shape, shared deterministic Unicode
  normalization, strict `title|description` sources, pagination, closed-state
  filtering, cancellation, and Workspace isolation.
- Prove projection synchronization/restart across both Project write stacks, a
  strict cancellable Core boundary, shared Web/Desktop consumption, the frozen
  2,000-Project latency budget, and installed Chrome acceptance.
- Preserve `issue_search=true`; enable only `project_search`; keep S04 and
  `pin_reorder` inactive until a later plan.

### Verification and review outcome

- Exact candidate: `c9e905bc7675d253991c0a816bcd19985a49c10b`.
- RED/GREEN coverage proves Unicode normalization, ranking, pagination,
  closed-state filtering, Workspace isolation, cancellation, authentication,
  trusted identity, permission denial, strict Core payloads, loaded explicit
  feature gating, projection synchronization/restart, and installed
  Project-surface deletion followed by search disappearance.
- The 2,000-Project non-race gate measured p50/p95 at 3.0314/4.1305ms, within
  the frozen 100/250ms limits. Exact-candidate backend `make check` and the
  official clean-cache `make test-race` pass with MinGW GCC 15.2.0.
- Root typecheck passes 6/6. Root tests pass 5/5; Views passes 166 files and
  1676 tests. The final review-only test commit changes only backend HTTP and
  Playwright test files outside the root package test inputs.
- Exact-candidate production Web build passes. A fresh migrated SQLite
  database and canonical fixture pass installed Chrome
  `e2e/project-search.spec.ts` 1/1 with retries disabled, including
  English/Chinese, ranking, closed state, pagination, isolation, shared UI,
  and DELETE-trigger projection disappearance.
- Independent review first returned CODE QUALITY BLOCK for incomplete commit
  trailers and missing direct auth/identity and installed-delete coverage.
  The four unpublished commit messages were rewritten with identical final
  tree `b33a24c8fc5818cc1ddec40f9879ea62af33fac7`; all five r023 commits now
  carry the required governance trailers. Added tests pass, and fresh
  independent re-review returns SPEC PASS and CODE QUALITY PASS.
- `server/**` candidate scope is empty; policy hashes and protected Input blob
  match v18. Candidate tracked state is clean apart from recorded
  `.local-runtime/**` and `node_modules.main-link-r019/**` environment
  artifacts; exact backend/Web processes were stopped and ports
  3000/8080/9080 are closed.
- r023 and PCR-S03B are closed. Release 1 remains active with no active task;
  S04 remains inactive pending a new approved plan. No push, merge, or
  deployment is claimed.

## PCR-001-S04-R24

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-S04-R24`
- Task-Revision: `r024`
- Work-Item: `PCR-S04`
- Status: `complete-independent-reviewed`
- Assignee: `Codex primary agent`
- Independent reviewer: `SPEC PASS and CODE QUALITY PASS on b4fa3a3c after tree-identical trailer repair`
- Base commit: `b86447909e1a7614c539769514008e010b478140`
- Approved plan: `PRODUCT-CAPABILITY-ROADMAP-001 v19`
- Plan hash: `c39bce662586248b0b652e93a940f3feb7897c2e6d1761b55635594172ddd800`
- Policy bundle: `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646/fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209/c39bce662586248b0b652e93a940f3feb7897c2e6d1761b55635594172ddd800`

### Objective and acceptance

- Install atomic revisioned `PUT /api/pins/reorder` for the authenticated
  user's complete Pin set in the trusted Workspace.
- Reject duplicate, missing, additional, foreign, other-user, stale, denied,
  and cancelled requests without partial mutation; preserve contiguous order
  and monotonic revision through create/delete/reorder and restart.
- Prove strict Core parsing/request shape, Workspace/user-scoped optimistic
  rollback/refetch, explicit loaded sidebar gating, and installed Chrome
  persistence/concurrency acceptance.
- Preserve both installed search features and keep all Release 2 stories
  inactive.

### Verification and review outcome

- Exact candidate: `b4fa3a3c60d24953c9f37c82cf0d70cda0fd7b11`; implementation
  commit: `738338ff3e19d14fc18100d385aaa0c832368472`.
- Application, HTTP, repository, migration, runtime, Core, and Views tests prove
  strict complete-set validation, authentication/trusted identity/CSRF,
  authorization, canonical conflict shape, cancellation, rollback, revision
  monotonicity, restart persistence, scoped optimistic recovery, and loaded
  explicit drag gating.
- Exact-tree backend `make check` and clean-cache official `make test-race`
  pass with MinGW GCC 15.2.0. Root typecheck passes 6/6 and root tests pass 5/5;
  production Web build passes. Root `make check` exited zero but reported that
  Docker Desktop was unavailable, so it is recorded only as an environment-
  warned script result and is not used as the complete acceptance gate.
- A fresh migrated SQLite database and canonical fixture pass installed Chrome
  `e2e/pin-reorder.spec.ts` 1/1 with retries disabled. The journey proves stale
  409 rollback/refetch, successful 204 complete reorder, contiguous persisted
  positions, one shared revision, and reload persistence through the sidebar.
- The first E2E attempt exposed an ambiguous link selector; the second exposed
  native `dragTo` incompatibility with dnd-kit. Stable sidebar selectors and a
  real pointer sequence closed both test-driver gaps before the fresh-db PASS.
- Two unpublished commit trailers initially carried incorrect policy hashes.
  They were rewritten to the activation bundle without changing final tree
  `41525376c8b8563eccaff7ad05931ec78e6cc85d`; all r024 commits now carry the
  correct CLAUDE/backend AGENTS/v19 hashes.
- `server/**` candidate and worktree diffs are empty; protected Input remains
  blob `a830fd2f0f82770563908d512558fe6ba48f50dd`. Candidate tracked state is clean
  apart from recorded `.local-runtime/**` and dependency artifacts. Exact
  backend/Web processes were stopped and ports 3000/8080/9080 are closed.
- Fresh independent read-only review returns SPEC PASS and CODE QUALITY PASS.
  r024, PCR-S04, and Release 1 are closed. Release 2 remains inactive; no push,
  merge, or deployment is claimed.

## PCR-001-S05A-R25

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-S05A-R25`
- Task-Revision: `r025`
- Work-Item: `PCR-S05A`
- Status: `complete-independent-reviewed`
- Assignee: `Codex primary agent`
- Independent reviewer: `Codex independent subagent release2_baseline — SPEC PASS / CODE QUALITY PASS`
- Base commit: `0aed36871271184325b6147841348847b004a7a4`
- Approved plan: `PRODUCT-CAPABILITY-ROADMAP-001 v20`
- Plan hash: `a55f0f0f5e593d39f779fb3d67a556dfa5eb0157b49faa4562badbdb5af1ad85`
- Policy bundle: `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646/fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209/a55f0f0f5e593d39f779fb3d67a556dfa5eb0157b49faa4562badbdb5af1ad85`

### Objective and acceptance

- Install the System-owned Skill definition and immutable numbered-version
  lifecycle through the frozen `/api/skills` family with Workspace visibility
  bindings, exact revision conflicts, provenance, audit, and retained reads.
- Enforce owner/admin mutations, member published reads, explicit Agent binding,
  strict Core parsing, shared Web/Desktop lifecycle behavior, and fresh-database
  installed-browser acceptance.
- Enable only `skill_administration`. Keep S05B file/import behavior, Space
  Asset implementation, S06A/S06B, push, merge, and deployment inactive.

### Activation evidence

- Exact base `0aed36871271184325b6147841348847b004a7a4` closes Release 1 and
  has no active roadmap task before this activation.
- Bounded discovery completed three rounds with zero final new dependencies and
  no unresolved retrievable or human-owned blocker. S05A is first in the frozen
  Release 2 dependency order; S05B owns file/import behavior explicitly.
- Protected `packages/ui/components/ui/input.tsx` blob is
  `a830fd2f0f82770563908d512558fe6ba48f50dd`. Existing generated protobuf
  worktree blobs equal the index; all other recorded dirty paths remain excluded.
- `server/**` range and worktree diffs are empty at activation.

### Closure evidence

- Exact implementation candidate `17f7eb1f2db22bd64e426614b84b945325b1f90f`
  (tree `57361bc6344e2801c4393414cfcc4bdb6d6c9897`) installs the complete
  S05A catalog, immutable version lifecycle, Workspace visibility, provenance,
  audit, authorization, strict clients, shared UI, and installed acceptance.
- The final remediation protects down migration from retained Workspace
  bindings and proves `context.Canceled` and `context.DeadlineExceeded` leave
  Skill, version, audit, and binding tables without partial rows.
- Exact candidate backend `make check` and official `make test-race` pass with
  MinGW GCC 15.2.0. Frontend typecheck, serialized tests, production Web build,
  and installed-Chrome lifecycle acceptance passed before the backend-only
  final remediation; the remediation changed no frontend blob.
- Installed Chrome acceptance proves create, version, publish, archive,
  restore, provenance/audit display, and hidden import controls on a fresh
  migrated database. The first production-proxy attempt exposed missing
  `REMOTE_API_URL`; the corrected explicit 8081 backend wiring passed 1/1.
- Fresh independent review of `28c0b71e..17f7eb1f` returns SPEC PASS and CODE
  QUALITY PASS with no blocker. Candidate and worktree `server/**` diffs are
  empty, the eight governance trailers parse, and recorded dirty paths remain
  excluded. PCR-S05A is closed; S05B remains inactive pending a new frozen plan.

## PCR-001-S05B-R26

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-S05B-R26`
- Task-Revision: `r026`
- Work-Item: `PCR-S05B`
- Status: `complete-independent-reviewed`
- Assignee: `Codex primary agent`
- Independent reviewer: `Codex independent subagent release2_baseline — SPEC PASS / CODE/SECURITY/QUALITY PASS`
- Base commit: `27def068f07cb56ef7e58471fbfacb397b11b639`
- Approved plan: `PRODUCT-CAPABILITY-ROADMAP-001 v21`
- Plan hash: `e7db2df5d92345dfe0e988af3f1c1f353ddd519bda8bf07bfbfcf6a472f9544e`
- Policy bundle: `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646/fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209/e7db2df5d92345dfe0e988af3f1c1f353ddd519bda8bf07bfbfcf6a472f9544e`

### Objective and acceptance

- Install safe archive/recognized-URL preview and import, immutable
  version-scoped logical manifests, Space-owned quarantined objects, exact
  checksums, file add/replace/delete/download, and retained historical reads.
- Enforce every frozen path/type/count/size/depth/source limit, idempotency,
  exact revisions, owner/admin mutation, member published reads, Agent exact
  bindings, strict clients, and installed-browser acceptance with zero leaks.
- Enable only `skill_import` in addition to completed flags. Keep S06A/S06B,
  push, merge, deployment, and Release 2 completion inactive.

### Activation evidence

- Exact base `27def068f07cb56ef7e58471fbfacb397b11b639` closes S05A with
  fresh independent SPEC PASS and CODE QUALITY PASS and has no active task.
- Three bounded discovery rounds end with `new_dependencies=0`, no retrievable
  blocker, contradiction, blocking unknown, or unresolved human decision.
  Existing URL UI/API shape, archive contract, Space ownership, and the
  unimplemented generated Asset scaffold are explicitly classified in v21.
- Policy bundle freezes current CLAUDE, backend AGENTS, and immutable v21 hashes.
  Protected Input worktree blob remains
  `a830fd2f0f82770563908d512558fe6ba48f50dd`; generated protobuf and all other
  recorded dirty paths remain excluded.
- `server/**` activation range and worktree diffs are empty. No implementation,
  PASS, independent review, closure, push, merge, or deployment is claimed.

### Closure evidence

- Exact candidate `abce08d652c1af263a4e678a31732814d324330d`
  (tree `639d1bdfaf843c08c3a3e3df15c5d66140facc47`) installs governed
  archive/recognized-URL preview and import, immutable logical manifests,
  Space-owned quarantine/promotion/reconciliation, exact checksums and retained
  downloads, strict Core parsing, and shared Web/Desktop file management.
- Adversarial coverage rejects traversal, invalid UTF-8, duplicate NFC paths,
  links, nested or disguised container/executable bodies including ZIP, gzip,
  PDF, tar, and ar, exact count/size/depth overruns, token/source mismatch, SSRF
  boundaries, cancellation, and partial transaction/object leaks.
- Installed S05B refuses ordinary file or metadata versions before `SKILL.md`;
  subsequent metadata versions copy the complete manifest in the same SQLite
  transaction. Exact historical and Agent-bound archived/deprecated reads stay
  available while cross-Workspace and unbound reads remain denied.
- Backend `make check` and official `make test-race` pass with MinGW GCC 15.2.0.
  Root typecheck passes; Core/Views focused tests pass. The root parallel test
  run has two unrelated Team Control 5-second load timeouts, and the exact file
  passes 10/10 immediately in isolation. Production Web build passes with only
  the two pre-existing CSS `::highlight` optimizer warnings.
- Fresh SQLite, explicit production proxy wiring, installed Chrome, and retries
  disabled pass the lifecycle/import/download journey 1/1. The first run without
  build-time `REMOTE_API_URL` correctly exposed a 404 proxy misconfiguration;
  rebuilding with explicit backend 8081 wiring passed. All isolated processes
  were stopped.
- Fresh final independent review returns SPEC PASS and CODE/SECURITY/QUALITY
  PASS with no blocker. Eight governance trailers parse, immutable v21 and its
  policy hashes remain exact, protected Input remains
  `a830fd2f0f82770563908d512558fe6ba48f50dd`, recorded dirty paths stay excluded,
  and candidate/worktree `server/**` diffs are empty. PCR-S05B is closed; S06A
  remains inactive pending its own frozen plan. No push, merge, or deployment is
  claimed.

## PCR-001-S06A-R27

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-S06A-R27`
- Task-Revision: `r027`
- Work-Item: `PCR-S06A`
- Status: `complete-independent-reviewed`
- Assignee: `Codex primary agent`
- Independent reviewer: `Codex independent subagent release2_baseline — SPEC PASS / CODE/SECURITY/QUALITY PASS on 3d465f11`
- Base commit: `78c5b5eb8b04d27d1fdc3c524955dd02fd756348`
- Approved plan: `PRODUCT-CAPABILITY-ROADMAP-001 v22`
- Plan hash: `11474d4c1bdf84f0fc574ef40f913425be35b6dad8810ce382ab88df959a6edc`
- Policy bundle: `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646/fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209/11474d4c1bdf84f0fc574ef40f913425be35b6dad8810ce382ab88df959a6edc`

### Objective and acceptance

- Install strict authenticated Knowledge list/detail with status, kind, source,
  applicability, and revision filters, deterministic ranking/keyset pagination,
  exact immutable provenance, superseded reads, and role-bounded quarantine.
- Prove Workspace isolation, cursor tamper/filter binding, cancellation, restart,
  strict Core schemas, loaded shared UI, and fresh installed-browser behavior.
- Enable only `knowledge_query`; keep S06B, `knowledge_review`, push, merge,
  deployment, and Release 2 completion inactive.

### Activation evidence

- Exact base `78c5b5eb8b04d27d1fdc3c524955dd02fd756348` closes S05B with
  final independent SPEC and code/security/quality PASS and no active task.
- Three bounded discovery rounds end with `new_dependencies=0`, no retrievable
  blocker, contradiction, blocking unknown, or unresolved human decision.
  Canonical scaffold limits, legacy evidence limits, storage ownership, query
  semantics, role visibility, cursor, and feature boundaries are frozen in v22.
- Policy bundle freezes current CLAUDE, backend AGENTS, and immutable v22 hashes.
  Protected Input remains `a830fd2f0f82770563908d512558fe6ba48f50dd`;
  generated protobuf and every recorded dirty path remain excluded.
- `server/**` activation range and worktree diffs are empty. No implementation,
  verification PASS, independent review, closure, push, merge, or deployment is
  claimed.

### Closure evidence

- Exact candidate `3d465f1110fed1ce9bef8d076f77d4b553fda421`
  (tree `c54f33d48f660f6b12fbfb70dbeea9f8d4826e5e`) installs the
  authenticated governed list/detail surface, immutable provenance, strict
  Core parsing, and query-only shared Web/Desktop UI.
- Independent review first blocked explicit-empty filters and injected-provider
  reopening of `knowledge_review`. Final remediation rejects empty/repeated
  numeric and empty status/kind filters, binds cursors to Workspace, and forces
  both review permissions and feature false until S06B.
- Exact candidate backend `make check` and official `make test-race` pass with
  MinGW GCC 15.2.0. Root typecheck and production Web build pass. Root parallel
  tests retain one unrelated Team Control timeout whose exact file plus
  Knowledge passes 13/13 in focused execution.
- Fresh migrated backend runtime tests prove filtering, ranking, cursor,
  visibility, isolation, and restart. Installed Chrome passes the strict shared
  UI/provenance journey 1/1 with retries disabled; that browser spec mocks only
  Knowledge payloads and is recorded as UI projection evidence, not backend
  filter evidence.
- Fresh independent review returns SPEC PASS and CODE/SECURITY/QUALITY PASS.
  The three-table down guard, protected Input blob, trailers, recorded dirty
  exclusions, and empty `server/**` candidate/worktree scope all pass. PCR-S06A
  is closed; S06B remains inactive pending a new frozen plan. No push, merge,
  or deployment is claimed.

## PCR-001-S06B-R28

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-S06B-R28`
- Task-Revision: `r028`
- Work-Item: `PCR-S06B`
- Status: `verification-blocked`
- Assignee: `Codex primary agent`
- Independent reviewer: `Codex independent subagent release2_baseline`
- Base commit: `84ed0e4afa8f76de2cb854047f2fc4d26641810f`
- Approved plan: `PRODUCT-CAPABILITY-ROADMAP-001 v23`
- Plan hash: `29eaeb52aaedc1764b0e30ddffd173d6129ea10e1b0170786ce4bce9277190d4`
- Policy bundle: `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646/fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209/29eaeb52aaedc1764b0e30ddffd173d6129ea10e1b0170786ce4bce9277190d4`

### Objective and acceptance

- Install idempotent member proposals, owner/admin independent review, exact
  expected-revision transitions, evidence-required publication, replacement
  supersession, invalidation, immutable audit, and post-commit realtime.
- Prove self-review denial and owner-only reasoned emergency override, stale
  candidate/target conflicts, source validation, cancellation/rollback,
  restart, Workspace isolation, strict Core, loaded shared UI, and fresh
  installed-browser behavior.
- Enable only `knowledge_review` in addition to completed flags. Keep Release 3,
  push, merge, deployment, and Release 2 completion inactive until closure.

### Activation evidence

- Exact base `84ed0e4afa8f76de2cb854047f2fc4d26641810f` closes S06A with
  fresh independent SPEC and CODE/SECURITY/QUALITY PASS and no active task.
- Read-only discovery reached closure across the frozen lifecycle, dormant Core
  expectations, legacy evidence, audit/outbox/realtime seams, authorization,
  and Space's narrow asset-ownership reader. No unresolved retrievable blocker,
  contradiction, blocking unknown, or human-owned decision remains.
- Immutable v23 freezes action/state mapping, proposal idempotency, target
  capture, self-review rules, evidence publication gate, atomic audit, exact
  realtime boundary, strict clients, and feature gating.
- Protected Input remains `a830fd2f0f82770563908d512558fe6ba48f50dd`;
  generated protobuf and every recorded dirty path remain excluded.
  `server/**` range/worktree diffs are empty. No implementation, verification
  PASS, closure, push, merge, or deployment is claimed.

### Verification block

- Exact implementation candidate `ca8b2a41f7ca4871375584a0a4ee1628c8c0a075`
  passes backend check/race, root typecheck, and production build. Exact RED
  base `ffcdd1c7a87db21a3a5f5a20afb66c6c952ee8ac` upgrades browser
  acceptance to a real member and independent owner with no Knowledge mocks.
- The member proposal commits and the application invokes
  `knowledge:candidate_updated`, but `backend/internal/realtime/hub.go`
  silently drops the unrecognized event. Independent review cannot see the
  candidate without reload, so the frozen realtime gate fails.
- The required Hub and shared event-type files are outside v23 scope. r028 is
  retained as verification-blocked; no scope is silently expanded and no
  closure is claimed.

## PCR-001-S06B-R29

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-S06B-R29`
- Task-Revision: `r029`
- Work-Item: `PCR-S06B`
- Status: `design-blocked`
- Assignee: `Codex primary agent`
- Independent reviewer: `Codex independent subagent release2_baseline`
- Base commit: `ffcdd1c7a87db21a3a5f5a20afb66c6c952ee8ac`
- Approved plan: `PRODUCT-CAPABILITY-ROADMAP-001 v24`
- Plan hash: `826a1818a4a5f362512912d56f0eecc34ebcb6089656531e1adc7e1cbdcc9966`
- Policy bundle: `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646/fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209/826a1818a4a5f362512912d56f0eecc34ebcb6089656531e1adc7e1cbdcc9966`

### Objective and acceptance

- Admit only `knowledge:candidate_updated` through the Workspace-scoped Hub
  and shared WebSocket event type without changing v23 mutation semantics.
- Prove target Workspace delivery, foreign Workspace exclusion, retained
  allowlist denial, and fresh installed-Chrome member-to-owner realtime
  proposal/review/publication/readback with no reload or Knowledge mocks.
- Keep Release 3, push, merge, deployment, and Release 2 completion inactive
  until all exact candidate gates and fresh independent review pass.

### Activation evidence

- Exact base `ffcdd1c7a87db21a3a5f5a20afb66c6c952ee8ac` retains the v23
  implementation and failing dual-identity browser RED.
- The missing Hub allowlist and event-union dependencies are the only newly
  retrieved blockers. Immutable v24 freezes their exact scope; no Knowledge
  business, storage, permission, migration, route, or UI semantics may change.
- Protected Input remains `a830fd2f0f82770563908d512558fe6ba48f50dd`;
  generated protobuf and every recorded dirty path remain excluded.
  `server/**` range/worktree diffs are empty. No remediation PASS, closure,
  push, merge, or deployment is claimed.

### Design block

- Independent review confirms a raw allowlist repair would broadcast the full
  candidate/proposer/reason projection to every Workspace WebSocket client,
  including ordinary members denied access to the private queue.
- v24 does not authorize the required role-aware composition or per-client
  public/private projection. Its uncommitted trial was discarded; r029 is
  design-blocked and has no product candidate.

## PCR-001-S06B-R30

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-S06B-R30`
- Task-Revision: `r030`
- Work-Item: `PCR-S06B`
- Status: `complete-independent-reviewed`
- Assignee: `Codex primary agent`
- Independent reviewer: `Codex independent subagent release2_baseline`
- Base commit: `33ec500fcfe9d3bb751d898dfbbdd9bce664aa48`
- Approved plan: `PRODUCT-CAPABILITY-ROADMAP-001 v25`
- Plan hash: `ca46357cd46bd57a7c31663224b450e840784f26bc7f1495f6fc30e7048a1301`
- Policy bundle: `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646/fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209/ca46357cd46bd57a7c31663224b450e840784f26bc7f1495f6fc30e7048a1301`

### Objective and acceptance

- Deliver private candidate projections only to actors currently authorized for
  Knowledge review; deliver only public entry projection to ordinary members
  after publish/supersede; deny all other Knowledge frames.
- Add all-seven-action, invalid-transition, terminal, cancellation, permission,
  Workspace-isolation, and dual-identity no-reload browser evidence.
- Keep Release 3, push, merge, deployment, and Release 2 completion inactive
  until exact final checks and fresh independent review pass.

### Activation evidence

- Exact base `33ec500fcfe9d3bb751d898dfbbdd9bce664aa48` records v24 activation,
  the v23 candidate, and the failing real member-to-owner realtime acceptance.
- Independent review returns SPEC BLOCK and CODE/SECURITY/QUALITY BLOCK for the
  dropped event, unsafe raw projection, non-independent browser path, incomplete
  action/cancellation coverage, scope, and unparseable/incomplete trailers.
  v25 freezes the narrow authority required to address every item.
- Protected Input remains `a830fd2f0f82770563908d512558fe6ba48f50dd`;
  generated protobuf and recorded dirty paths remain excluded; `server/**`
  range/worktree diffs are empty. No implementation or PASS is claimed.

### Candidate evidence

- Exact candidate `32f273e47e6b9863b4ef462f28eed3e91da654d0`
  (tree `fd860e2b89cfc7a74caef39adf31b0f667cfc385`) installs the v25
  role-aware Hub resolver and per-client Knowledge projection. Authorized
  reviewers receive the private candidate; ordinary members receive only the
  public entry after publication or supersession; denied and foreign-Workspace
  clients receive no frame.
- Focused Hub tests pass five consecutive executions. Repository tests cover
  all seven actions, invalid and terminal transitions, proposal cancellation,
  persistence, audit, and idempotency. Backend `make check` and official
  `make test-race` pass with MinGW GCC 15.2.0; Core realtime tests pass 28/28;
  root and Core typechecks pass.
- The exact candidate production Web build passes with the two pre-existing
  CSS `::highlight` optimizer warnings. Root parallel tests retain one
  unrelated Team Control five-second timeout after 1,674/1,675 Views
  assertions; the exact file passes 10/10 in focused execution, so the root
  aggregate remains recorded as non-PASS evidence.
- Fresh SQLite, the Canonical HTTP runtime, production Web on `[::1]:3000`,
  and installed Chrome exercise a true member proposal, independent owner
  review, approval/publication, reviewer realtime queue update, and sanitized
  member publication update without reload. The journey passes 1/1 with
  retries disabled and no mocked Knowledge route.
- The candidate contains the exact continuous eight-trailer block required by
  v25. Protected Input, generated protobuf, recorded dirty paths, and
  `server/**` remain excluded. Fresh independent SPEC and
  CODE/SECURITY/QUALITY review remains pending; r030 and Release 2 are not yet
  closed.

### Closure evidence

- Fresh independent review of exact candidate
  `32f273e47e6b9863b4ef462f28eed3e91da654d0` returns `SPEC PASS` and
  `CODE/SECURITY/QUALITY PASS` with no blocking findings. It verifies trusted
  identity retention, role-aware composition, reviewer/private and
  member/public projection, permission-denial and Workspace isolation, exact
  post-commit publication semantics, seven-action/cancellation coverage, the
  dual-identity browser contract, scope, and all eight trailers.
- r030 and PCR-S06B are `complete-independent-reviewed`. All four Release 2
  stories are closed, so Release 2 is complete with zero active tasks. Release
  3, push, merge, and deployment remain inactive.

## PCR-001-S07A-R31

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-S07A-R31`
- Task-Revision: `r031`
- Work-Item: `PCR-S07A`
- Status: `review-blocked`
- Assignee: `Codex primary agent`
- Independent reviewer: `Codex independent subagent release3_s07a_discovery`
- Base commit: `80d92b14b1f5a1525fbce0c60ce992e28c7f0e8b`
- Approved plan: `PRODUCT-CAPABILITY-ROADMAP-001 v26`
- Plan hash: `9666afbc4408eacd684d799d17acc7e3dfb39ff3fd8448a4f010ed526644d3e2`
- Policy bundle: `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646/fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209/9666afbc4408eacd684d799d17acc7e3dfb39ff3fd8448a4f010ed526644d3e2`

### Objective and acceptance

- Install governed GitHub repository and generic URL Resources through the real
  Canonical SQLite/HTTP/Core/shared UI vertical.
- Prove local credential-free normalization, duplicate denial, revisioned
  reorder/archive/restore, safe unavailable refresh, Project-lead authorization,
  atomic Project creation/deletion semantics, count projection, restart, and
  malformed-client denial.
- Keep S07B-D, Release 3 completion, push, merge, deployment, generated
  protobufs, original dirty files, and `server/**` inactive or excluded.

### Activation evidence

- Exact base `80d92b14b1f5a1525fbce0c60ce992e28c7f0e8b` closes Release 2
  with zero active tasks before this activation.
- Bounded read-only discovery found partial GitHub-only UI/Core declarations but
  no Canonical Resource table, route, repository, installed capability, generic
  URL, strict schema, or acceptance journey.
- Human Customer confirmed the reviewed execution plan and connection-state
  default on 2026-08-19. Immutable v26 freezes r031 only. No implementation,
  verification PASS, closure, push, merge, or deployment is claimed here.

### Candidate and independent-review outcome

- The exact staged candidate contained 42 implementation paths, zero unstaged
  paths, and zero `server/**` paths at tree
  `6559133236c2c6d02b8453ea72ad52c054a6b10c`; its exact binary patch hash was
  `575e289823966eb246351e9ae2db7b71bcbd0f54`.
- Focused backend/Core/Views checks, production Web build, official race checks
  for exact S07A packages, and the fresh installed-Chrome journey passed. The
  root test aggregate retained one unchanged analytics timeout and the full
  Windows race aggregate retained existing onboarding/outbox failures; neither
  aggregate is represented as PASS.
- Fresh review returned `SPEC BLOCK` and `CODE/SECURITY/QUALITY BLOCK` for
  incomplete down-migration authority guards, revision/state ordering and
  refresh TOCTOU, authorization outside mutation transactions, missing initial
  Resource permission mapping, raw schema diagnostic values, and explicit
  SQLite indexes. r031 is review-blocked; product changes were unstaged and v26
  remains immutable.

## PCR-001-S07A-R32

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-S07A-R32`
- Task-Revision: `r032`
- Work-Item: `PCR-S07A`
- Status: `review-blocked`
- Assignee: `Codex primary agent`
- Independent reviewer: `Codex independent subagent release3_s07a_discovery`
- Base commit: `71afb3c33a4d82431a8016cb195a97e5a36d8646`
- Blocked input tree: `6559133236c2c6d02b8453ea72ad52c054a6b10c`
- Approved plan: `PRODUCT-CAPABILITY-ROADMAP-001 v27`
- Plan hash: `ab4ebd052e93b8c601442fba499e950047bdf01e02cb21da04ce2103e14b8f54`
- Policy bundle: `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646/fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209/ab4ebd052e93b8c601442fba499e950047bdf01e02cb21da04ce2103e14b8f54`

### Objective and acceptance

- Preserve the complete v26 S07A product contract while repairing only the
  seven independent-review blocker classes frozen by v27.
- Prove rollback guards for every authority reference, transaction-local
  duplicate detection without explicit SQLite indexes, transaction-owned
  current authorization and revision precedence, resolver suppression for
  failed refreshes, Project-create 403 rollback, and secret-free Core schema
  diagnostics.
- Re-enter every focused, broad, installed, scope, traceability, and fresh
  independent-review gate. Keep S07B-D, Release 3 completion, push, merge,
  deployment, generated protobufs, original dirty files, and `server/**`
  inactive or excluded.

### Activation evidence

- Exact governance base `71afb3c33a4d82431a8016cb195a97e5a36d8646`
  activates from immutable v26/r031 history after the blocked product candidate
  was unstaged without changing its working-tree content.
- Immutable v27 rejects an SQLite index exception and freezes transaction-local
  duplicate checks plus one write-transaction authority/revision/refresh
  boundary. No product repair, verification PASS, closure, push, merge, or
  deployment is claimed by activation.

### Candidate and independent-review outcome

- Exact candidate `9fb86ea056d253c82ea61b13031550db82eeb526`
  (tree `c6a2f08f41ae2424807b39fd9b5cef5990324f84`) contains 46 paths,
  zero unstaged paths, and zero `server/**` paths. Its exact binary patch hash
  from the v27 activation commit is
  `9b03028ec2218ea75948cd47179984d056221408`.
- Focused backend/Core/Views checks, backend `make check`, exact changed-package
  race checks, root typecheck, production Web build, and the fresh
  retries-disabled installed-Chrome journey pass. The root test and broad
  Windows race aggregates retain their documented unrelated failures and are
  not represented as PASS.
- Fresh independent review returns `SPEC BLOCK` and
  `CODE/SECURITY/QUALITY BLOCK`: blank lines separate the required commit
  fields so Git recognizes only `Policy-Bundle` as a trailer, and the blocked
  down-migration test proves schema/catalog survival without comparing each
  retained row value before and after failure. The reviewer found no additional
  product, security, or quality blocker. r032 is review-blocked and immutable
  v27 remains unchanged.

## PCR-001-S07A-R33

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-S07A-R33`
- Task-Revision: `r033`
- Work-Item: `PCR-S07A`
- Status: `complete-independent-reviewed`
- Assignee: `Codex primary agent`
- Independent reviewer: `Codex independent subagent release3_s07a_discovery`
- Base commit: `9fb86ea056d253c82ea61b13031550db82eeb526`
- Blocked input tree: `c6a2f08f41ae2424807b39fd9b5cef5990324f84`
- Approved plan: `PRODUCT-CAPABILITY-ROADMAP-001 v28`
- Plan hash: `fc285bd972f09073d01f5a931b9ea4a57949af428349d68d806d03877d23b90b`
- Policy bundle: `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646/fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209/fc285bd972f09073d01f5a931b9ea4a57949af428349d68d806d03877d23b90b`

### Objective and acceptance

- Preserve the complete v27 S07A product tree while repairing only the two
  independent-review evidence/traceability blockers frozen by v28.
- Prove byte-for-byte retained Resource, Resource-set, audit, and idempotency
  row values after every independently blocked down-migration case.
- Freeze a new exact candidate whose final commit exposes one continuous,
  parseable nine-field trailer block. Re-enter focused/backend/race, scope,
  traceability, and fresh independent-review gates.
- Keep S07B-D, Release 3 completion, push, merge, deployment, generated
  protobufs, original dirty files, and `server/**` inactive or excluded.

### Activation evidence

- Exact base `9fb86ea056d253c82ea61b13031550db82eeb526` is the immutable
  r032 review-blocked candidate. Its product/security review found no blocker
  beyond the two proof defects named above.
- Immutable v28 authorizes one assertion-first migration test and only a
  captured-RED minimum atomicity correction if the existing behavior actually
  mutates retained data. Otherwise product code remains unchanged.
- The v28 policy bundle is frozen. No evidence repair, PASS, closure, push,
  merge, or deployment is claimed by activation.

### Candidate and closure evidence

- Exact candidate `b3828be7b9b272732c5630975e73e35b629ed9f9`
  has tree `7c4a45fff414a555688358bd938111f8105c774f` and binary patch
  hash `eb5f5041fad3944860b914a581aee42902f191dc` from exact base
  `9fb86ea056d253c82ea61b13031550db82eeb526`. The range contains only
  six authorized governance/test paths, zero `server/**` paths, passes
  `git diff --check`, and is clean.
- Both range commits expose exactly one continuous nine-field Git trailer
  block. The six guarded down-migration cases compare every retained row field
  before and after the expected failure; focused and complete Workspace tests
  pass. Backend `make check` passes with task-specific F-drive Go temp/cache,
  and the official Windows Workspace race passes with MinGW GCC 15.2.0. The
  earlier C-drive temporary-space failure and unrelated broad aggregate
  failures remain explicitly NON-PASS evidence.
- Fresh independent review of that exact candidate returns `SPEC PASS` and
  `CODE/SECURITY/QUALITY PASS` with no blocking finding. It verifies both r032
  blockers are closed, exact v28 scope/hash/traceability, unchanged product
  behavior, empty `server/**`, and the deterministic evidence.
- r033 and PCR-S07A are `complete-independent-reviewed`. PCR-S07B remains
  inactive pending a successor plan and the resolved outline authority scope.
  Release 3 is not complete; no push, merge, or deployment is claimed.

## PCR-001-S07B-R34

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-S07B-R34`
- Task-Revision: `r034`
- Work-Item: `PCR-S07B`
- Status: `approved-active`
- Assignee: `Codex primary agent`
- Independent reviewer: `pending; reuse Codex independent subagent release3_s07a_discovery`
- Base commit: `07aef1a577db78598c92c70312a33989e6177d64`
- Approved plan: `PRODUCT-CAPABILITY-ROADMAP-001 v29`
- Plan hash: `646a6452bea95674444fc173a0a9abbe81c29b981b191e091a488efed065cb23`
- Policy bundle: `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646/fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209/646a6452bea95674444fc173a0a9abbe81c29b981b191e091a488efed065cb23`

### Objective and acceptance

- Install the singular Canonical Requirement baseline/revision authority,
  complete frozen lifecycle, independent approval, revisioned project grants,
  Issue/outline traceability, and material-change review-required projection.
- Install only the Customer-confirmed prerequisite outline root-node
  persistence/create/read/stable-ID/project-validation boundary. Keep the full
  S10 outline product and `project_outline` false.
- Prove stale/idempotency behavior, transaction-owned current authorization and
  reference validation, immutable history/audit, legacy mutation disablement,
  import/rollback/deletion safety, strict Core schemas, shared UI, and a real
  two-identity installed-Chrome journey.
- Keep S07C-D, Release 3 completion, generated protobufs, original dirty paths,
  push, merge, deployment, and `server/**` inactive or excluded.

### Activation evidence

- Exact base `07aef1a577db78598c92c70312a33989e6177d64` closes S07A
  after fresh independent `SPEC PASS` and `CODE/SECURITY/QUALITY PASS`; the
  isolated worktree is clean before activation.
- Read-only dependency closure confirms there is no existing outline domain,
  table, repository, HTTP handler, or installed acceptance; the existing
  Requirement skeleton is gRPC-only, draft-only, and not authorized by the
  real runtime. Core/shared UI declarations are partial and uninstalled.
- The Human Customer confirmed the minimal outline authority on 2026-08-19.
  Immutable v29 freezes that exact boundary and the full S07B state/security
  contract. No product implementation, verification PASS, closure, push,
  merge, or deployment is claimed by activation.
