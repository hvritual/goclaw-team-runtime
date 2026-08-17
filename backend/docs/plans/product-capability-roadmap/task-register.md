# Product Capability Roadmap Task Register

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Plan-Version: `v9`
- Registry status: `Release 1 active; PCR-001-S02A-R14 is the sole active task`
- Registry revision: `r014`
- Updated: `2026-08-18`

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
- Status: `active`
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
