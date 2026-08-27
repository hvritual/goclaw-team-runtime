# Product Capability Roadmap Task Register

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Plan-Version: `v43`
- Registry status: `Release 3 complete-independent-reviewed; Release 4 S08A governed selective integration active under r048`
- Registry revision: `r048`
- Updated: `2026-08-27`

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
- Status: `review-blocked`
- Assignee: `Codex primary agent`
- Independent reviewer: `Codex independent subagent release3_s07a_discovery`
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

### R34.2-R34.4 implementation evidence

- Commits `29fc0ed0`, `1c437313`, and `cfffe2ae` install the exact lifecycle,
  migration, canonical repository/application/HTTP authority, legacy mutation
  disablement, minimal root outline authority, grants/links, and transactional
  Project/Issue deletion cleanup. No generated protobuf or `server/**` path is
  changed.
- The complete Workspace package passes with task-specific F-drive Go
  temp/cache after the retained C-drive disk-full failure and one bounded
  timeout. The full SQLite infrastructure package passes separately. These are
  backend implementation checks, not installed acceptance.
- Commit `8167897f` installs strict throw-on-malformed Core schemas/client,
  complete route serialization, retry-stable create idempotency, all six
  lifecycle states, server-projected access, Issue/outline traceability,
  material-change impact, immutable history, and only the confirmed minimal
  root-node create/read picker. It removes the uninstalled Requirement-driven
  Issue creation control and does not consume S07C coverage semantics.
- Captured Core RED fails all ten new contract/idempotency expectations before
  implementation; GREEN passes 10/10. Captured Views RED fails eight of nine
  new acceptance expectations against the old skeleton; GREEN passes 9/9.
  Core and Views typechecks pass, Core lint and every r034 changed TypeScript
  file lint pass, and Views locale parity plus the focused Requirement view
  pass 133/133.
- Broad Views lint remains NON-PASS on 16 pre-existing literal/unused-expression
  errors in Knowledge, Skills, and file-viewer paths plus unrelated existing
  hook warnings. The r034 Requirement warning found during that run was fixed
  and changed-file lint is clean. No broad failure is renamed PASS.
- Commit `9fe63f11` preserves the immutable legacy Requirement aggregate and
  constructor tests after the Canonical HTTP composition. Commit `48599860`
  installs only the two S07B Requirement permissions and
  `project_requirements`, while keeping `project_outline=false`; it also
  distinguishes idempotency-key/body conflict from other typed resource
  conflict responses.

### R34.5 deterministic and installed evidence

- Final backend `make check` passes in 235.3 seconds with task-specific F-drive
  Go temp/cache, including formatting, canonical policy/server-boundary,
  generated-output, `go vet ./...`, and `go test ./...`.
- A direct `go test -race` diagnostic selects legacy PATH GCC 8.1 and exits
  `0xc0000139` before any assertion; it is retained as NON-PASS environment
  evidence. The repository-owned official seven-package
  `make test-race RACE_PACKAGES=...` passes in 303.4 seconds and explicitly
  selects Scoop GCC 15.2.0 process-locally.
- Core full tests pass 625/625; Views full isolated tests pass 1683/1683; root
  typecheck passes. Root tests remain NON-PASS on three pre-existing
  five-second aggregate-load timeouts, while the two implicated files pass
  44/44 in isolation. Broad Views lint remains NON-PASS only on the previously
  recorded unrelated paths. No broad failure is renamed PASS.
- Production Web build and standalone production build both pass with all
  17/17 static pages; each retains two non-blocking CSS `::highlight` warnings.
- Fresh SQLite installed-Chrome acceptance uses a production standalone Web
  server, the real Canonical HTTP backend, and independent lead/owner browser
  identities. The lead creates a baseline and stable project-owned root outline
  node, links same-project Issue `REL-1` and the outline root, and submits v4.
  The owner's retained stale v3 submit receives HTTP 409, then the owner
  independently approves v5 and freezes effective v6.
- Frozen ordinary inputs are disabled. A lead-owned material change creates v7
  while effective content remains v6 and `REL-1` is marked review-required;
  the Issue title, todo state, and project remain unchanged before/after. A
  second independent review produces v8-v10 and moves the effective baseline
  to v10. All v1-v10 history survives reload.
- A real Canonical HTTP attempt to link foreign-project `REL-2` returns
  `404/not_found`; the baseline remains revision 7 with one Issue link. Owner
  retirement produces terminal read-only v11; after reload there are zero
  submit/approve/freeze/retire/material-save/link controls and v1/v11 history
  remains visible. Runtime config reports `project_requirements=true` and
  `project_outline=false`.
- The local invitation route is historically unavailable (`404`), so the
  second authenticated user's Workspace membership/onboarding fixture was
  inserted only for acceptance setup; project-lead selection and every S07B
  Requirement/outline/Issue action used the production UI or Canonical HTTP
  authority. No setup shortcut is counted as product acceptance.
- Exact candidate `fa1153164882adb4880d57f7349597900196402f` has tree
  `70de766121f06487eeb7c259906b1bbd0118594f`, binary patch hash
  `e975739c2daf10088af9b24ae7cf034fd3aa8994`, 55 authorized paths, zero
  `server/**` or generated paths, nine valid r034 trailer blocks, a clean
  isolated worktree, and zero overlap with the original dirty worktree.
- Fresh independent review returns `SPEC BLOCK` and
  `CODE/SECURITY/QUALITY BLOCK`. The imported canonical Issue-link down guard
  omits `workspace_id`/`project_id` ownership equality, and repository evidence
  does not explicitly cover live membership removal, lead reassignment,
  editor-grant removal, or terminal project-state mutation denial.
- r034 is review-blocked and immutable v29 remains unchanged. S07C-D, Release 3
  completion, push, merge, deployment, generated protobufs, and `server/**`
  remain inactive or excluded.

## PCR-001-S07B-R35

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-S07B-R35`
- Task-Revision: `r035`
- Work-Item: `PCR-S07B`
- Status: `complete-independent-reviewed`
- Assignee: `Codex primary agent`
- Independent reviewer: `Codex independent subagent release3_s07a_discovery`
- Base commit: `fa1153164882adb4880d57f7349597900196402f`
- Blocked input tree: `70de766121f06487eeb7c259906b1bbd0118594f`
- Blocked input patch hash: `e975739c2daf10088af9b24ae7cf034fd3aa8994`
- Approved plan: `PRODUCT-CAPABILITY-ROADMAP-001 v30`
- Plan hash: `374c52c1fe307e90046f4f3f170b647379a0ce8fffff795c0a38def54c4b07db`
- Policy bundle: `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646/fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209/374c52c1fe307e90046f4f3f170b647379a0ce8fffff795c0a38def54c4b07db`

### Objective and acceptance

- Preserve the complete r034 S07B product candidate and public contract.
- RED/GREEN the imported canonical Issue-link `workspace_id` and `project_id`
  down-migration ownership guards with exact retained-row/catalog proof.
- Prove same-transaction current authorization after live membership removal,
  lead reassignment, editor-grant removal, and terminal project-state changes,
  with typed denial and zero persisted mutation effects.
- Pass focused and full Workspace/backend deterministic and official race gates,
  then freeze one exact scoped candidate and obtain fresh independent
  `SPEC PASS` plus `CODE/SECURITY/QUALITY PASS`.
- Keep S07C-D, Release 3 completion, frontend behavior, generated protobufs,
  original dirty paths, push, merge, deployment, and `server/**` inactive or
  excluded.

### Activation evidence

- The exact r034 candidate and its previously passed deterministic, installed,
  scope, trailer, and cleanup evidence are retained without reclassification.
- Fresh independent review identifies exactly the two findings frozen by v30;
  no additional product, public-contract, outline-boundary, or frontend blocker
  is carried into r035.
- The Human Customer's confirmed continuous direction authorizes this minimal
  successor. Activation claims no remediation or review PASS.

### R35.2-R35.4 remediation and verification evidence

- The first focused migration RED attempt timed out during compilation at 120
  seconds and remains NON-PASS. The unchanged rerun captures both required
  behavioral failures: workspace-ownership and project-ownership drift each let
  the old down migration succeed destructively.
- The smallest GREEN adds canonical baseline and retained legacy ownership
  equality to the imported Issue-link guard. Both drift cases then pass with
  exact before/after retained row snapshots and catalog survival; the complete
  existing untouched-import and down-migration set also passes.
- Live-authority proof first exposed two test-authoring issues: a missing
  delimiter and an assertion that expected permission denial instead of the
  existing typed `ErrActorOutsideWorkspace` sentinel after membership removal.
  After correcting only the test, all five cases pass. Lead reassignment,
  editor-grant removal, completed project, and cancelled project return typed
  permission denial; membership removal returns the typed outside-Workspace
  denial. Exact persisted-effect snapshots are unchanged in every case.
- Focused migration and repository suites pass. The complete Workspace tree
  passes in 89.7 seconds, including SQLite infrastructure in 77.2 seconds.
- Backend `make check` passes in 349.6 seconds, including format, policy and
  `server/**` boundary, generated-output, vet, and `go test ./...` checks. The
  official seven-package race passes in 409.4 seconds using repository-selected
  MinGW GCC 15.2.0.
- r035 changes no frontend, HTTP/Core, runtime composition, capability flag, or
  authorization production code. The r034 production build and installed
  two-identity acceptance remain applicable without reclassification. Exact
  candidate/scope/trailer/dirty-tree audit and fresh independent dual PASS are
  still required.

### R35.5 independent closure

- Exact candidate `cd94396093ea73f3f9434fed7410036ae61170ab` has tree
  `7e6f045ec5a48c4465e7f2fd5261e0d2a3b4b42d`, binary patch hash
  `0a9f24812076a842bff24a68c27edb3709974193`, eight authorized paths,
  zero `server/**` or generated paths, three r035 commits with complete ordered
  nine-field trailer blocks, and a clean isolated worktree.
- The original worktree currently has 25 dirty entries and exact candidate
  overlap is zero. Policy and immutable v30 hashes remain exact; `git diff
  --check` passes.
- Fresh independent review returns `SPEC PASS` and
  `CODE/SECURITY/QUALITY PASS`. It independently confirms both r034 blockers are
  closed, current authorization remains same-connection after `BEGIN
  IMMEDIATE`, no public/runtime/frontend behavior expanded, and all retained
  NON-PASS/fixture limitations remain honestly disclosed.
- PCR-S07B is `complete-independent-reviewed`. PCR-S07C-D and Release 3
  completion remain inactive pending their own governed plans and DoneGates;
  push, merge, deployment, generated protobufs, and `server/**` remain excluded.

## PCR-001-S07C-R36

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-S07C-R36`
- Task-Revision: `r036`
- Work-Item: `PCR-S07C`
- Title: `Requirement coverage from current Issue and acceptance authority`
- Status: `verification-blocked-successor-active`
- Assignee: `Codex primary agent`
- Independent reviewer: `required before closure`
- Exact base: `f5695de83d55e277c8eeb9db7461b81137dc93ad`
- Predecessor candidate: `cd94396093ea73f3f9434fed7410036ae61170ab`
- Plan hash: `422f6ea3ab573f77c7c718105b04c06f132e1b4fd9c422bc06e8ee4433e2a081`
- Policy bundle: `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646/fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209/422f6ea3ab573f77c7c718105b04c06f132e1b4fd9c422bc06e8ee4433e2a081`
- Write boundary: exact paths in immutable `plan_v31.md`; no migration,
  generated protobuf, original dirty path, legacy backend write, or `server/**`
  path
- Acceptance source: `plan_v31.md` R36.2-R36.5 and story-map PCR-S07C

### Ordered outputs

- Install a strict authenticated coverage GET response for no baseline plus
  current/effective snapshots, cumulative counters, exact item stages, and
  linked current Issue evidence.
- Derive coverage from immutable Requirement content, revision-active Issue
  links, current Issue status, and the latest acceptance conclusion without a
  stored coverage cache or per-item/per-Issue query.
- Enforce fail-closed multi-Issue aggregation, latest-conclusion precedence,
  retirement visibility, link-removal/effective semantics, deterministic
  ordering, and existing member/project authorization.
- Remove the inactive Core fallback, install coverage-specific query
  invalidation and Issue-event refresh, and display server-projected stages in
  the shared Requirement view with four-locale parity.
- Pass assertion-first backend/Core/Views/realtime tests, full Workspace and
  backend gates, official race, strict frontend/build gates, production
  installed acceptance, exact scope/trailer/dirty/process audit, and fresh
  independent `SPEC PASS` plus `CODE/SECURITY/QUALITY PASS`.
- Keep PCR-S07D, Release 3 completion, S10, `project_outline`, generated
  protobufs, original dirty paths, push, merge, deployment, and `server/**`
  inactive or excluded.

### Activation evidence

- S07B exact candidate `cd943960` and closure `f5695de8` retain the fresh
  independent dual PASS, clean isolated worktree, complete continuous trailers,
  and zero original-dirty overlap recorded by v30/r035.
- A bounded five-round context discovery loop stabilizes with
  `new_dependencies_count=0`: no backend coverage route or derived repository
  exists; the Core compatibility endpoint is uninstalled and fallback-based;
  current/effective revisions, link intervals, Issue status, acceptance history,
  and Issue realtime events already provide the required authority.
- The Customer confirmed the prerequisite minimal outline authority and
  confirmed continued execution. Immutable v31 freezes only PCR-S07C from the
  clean exact base. Activation claims no implementation or PASS.

### R36.2-R36.4 implementation and aggregate stop evidence

- Commit `90acfecd` installs the bounded consistent Workspace repository,
  service, and HTTP coverage read after assertion-first RED evidence. Commit
  `5d769058` installs the strict Core schema/query/invalidation contract and
  shared four-locale current/effective view after frontend RED evidence.
- Focused backend/Core/Views/realtime/locale checks, complete Workspace, Core
  628/628, Views 1683/1683, changed-file lint, and Core/Views typechecks pass.
- The first fresh backend `make check` is NON-PASS after 351.9 seconds on
  `TestLocalAuthProjectsPersistedOnboardedAtAcrossRestart`; the exact test, full
  Auth package, and exact test 10/10 pass alone. After `go clean -testcache`, an
  unchanged fresh complete check repeats the same failure after 270.9 seconds.
- A detached diagnostic reproduction exposes the hidden aggregate-only service
  error as `context deadline exceeded`, caused by Kratos v3's one-second default
  test-server timeout under full package parallel load. The diagnostic source
  is removed and never enters the candidate. No SQLite lock, disk-full state,
  production Auth defect, or S07C behavior is implicated.
- The required Auth test path is outside v31. Its explicit stop condition
  therefore blocks r036 closure and requires immutable v32/r037. Neither
  complete check failure is renamed PASS.

## PCR-001-S07C-R37

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-S07C-R37`
- Task-Revision: `r037`
- Work-Item: `PCR-S07C`
- Title: `Stabilize aggregate Auth test and close Requirement coverage`
- Status: `verification-blocked-successor-active`
- Assignee: `Codex primary agent`
- Independent reviewer: `required before closure`
- Exact base: `5d769058d289300e6b8851a5885cca183d5f9be9`
- Retained predecessor candidate: `cd94396093ea73f3f9434fed7410036ae61170ab`
- Plan hash: `4c43962f9cb2fc319efa873d63c242a7234e12d557bb0b2d7f9df12fef28a823`
- Policy bundle: `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646/fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209/4c43962f9cb2fc319efa873d63c242a7234e12d557bb0b2d7f9df12fef28a823`
- Write boundary: exact paths in immutable `plan_v32.md`; every other Auth
  path, migration, generated protobuf, original dirty path, legacy backend
  write, and `server/**` path is read-only
- Acceptance source: `plan_v32.md` R37.2-R37.4 and story-map PCR-S07C

### Ordered outputs

- Change only the two `kratoshttp.NewServer()` sites in the affected restart
  test to a finite ten-second timeout; preserve every assertion and all Auth
  production behavior without sleep, retry, skip, or global test serialization.
- Re-run the exact/full Auth checks and a fresh complete backend `make check`
  under normal package parallelism while retaining both prior failures as
  NON-PASS evidence.
- Re-run the complete frozen S07C backend/Core/Views/realtime/locale/race/root/
  build gates and fresh real-HTTP production-Web acceptance.
- Freeze one exact scoped candidate with continuous nine-field trailers, clean
  isolated/original dirty/process evidence, and fresh independent `SPEC PASS`
  plus `CODE/SECURITY/QUALITY PASS` before closing PCR-S07C.
- Keep PCR-S07D, Release 3 completion, S10, `project_outline`, generated
  protobufs, Auth production code, original dirty paths, push, merge,
  deployment, and `server/**` inactive or excluded.

### Activation evidence

- The aggregate failure is reproduced twice in the unchanged candidate and a
  third time with diagnostic visibility; exact/full Auth and exact 10/10 pass
  outside full parallel load. Root cause is a test-only one-second request
  deadline, not an inferred product defect.
- Immutable v32 preserves the v31 product semantics and authorizes exactly one
  additional test path. The Human Customer's confirmed continuous direction
  provides successor authority; activation claims no remediation or PASS.

### R37.2-R37.3 Auth GREEN and official-race stop evidence

- Commit `2f4e8015` changes only the two in-process server constructors in the
  affected Auth restart-persistence test to a finite ten-second timeout. The
  exact test and full Auth package pass without retry; no production path or
  assertion changes.
- After clearing only test-result cache, a fresh complete backend `make check`
  passes in 383.6 seconds under normal package parallelism. The Auth package
  passes in 11.3 seconds and the full format/policy/generated/vet/test chain
  exits zero. Both earlier r036 aggregate failures remain NON-PASS evidence.
- The official seven-package race uses the repository wrapper and Scoop GCC
  15.2.0. Six Workspace packages pass, but Bootstrap detects a concurrent read
  and write of the onboarding test's mutable `now` through the running outbox
  clock and test-time advance. The command exits NON-PASS after 439 seconds.
- A focused race passes once due to scheduling; the unchanged focused x10 race
  reproduces the same detector stack and fails after 39.1 seconds. The required
  Bootstrap test path is outside v32, so r037 stops and immutable v33/r038 is
  required. No six-of-seven result is renamed race PASS.

## PCR-001-S07C-R38

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-S07C-R38`
- Task-Revision: `r038`
- Work-Item: `PCR-S07C`
- Title: `Remove onboarding test-clock race and close Requirement coverage`
- Status: `review-blocked-successor-active`
- Assignee: `Codex primary agent`
- Independent reviewer: `Galileo — SPEC BLOCK and CODE/SECURITY/QUALITY BLOCK`
- Exact base: `2f4e801552c3bdcc8a20e4e9fe6981a83c79ee1a`
- Retained predecessor candidate: `cd94396093ea73f3f9434fed7410036ae61170ab`
- Plan hash: `5f596081b0aa97ffdfc4aad7aaebce997681406d474d8bcbfa1efcd2019cb8ca`
- Policy bundle: `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646/fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209/5f596081b0aa97ffdfc4aad7aaebce997681406d474d8bcbfa1efcd2019cb8ca`
- Write boundary: exact paths in immutable `plan_v33.md`; every other
  Bootstrap/Auth path, migration, generated protobuf, original dirty path,
  legacy backend write, and `server/**` path is read-only
- Acceptance source: `plan_v33.md` R38.2-R38.4 and story-map PCR-S07C

### Ordered outputs

- Protect both injected reads and the sole write of the onboarding test's
  mutable clock with one local `sync.RWMutex`, without stopping/serializing the
  outbox or changing production behavior and assertions.
- Pass the exact Bootstrap race at least 10 consecutive times and then the
  fresh official seven-package race through the repository GCC wrapper.
- Re-run the complete frozen S07C/Auth/backend/Core/Views/realtime/locale/root/
  build gates and fresh real-HTTP production-Web acceptance.
- Freeze one exact scoped candidate with continuous nine-field trailers, clean
  isolated/original dirty/process evidence, and fresh independent `SPEC PASS`
  plus `CODE/SECURITY/QUALITY PASS` before closing PCR-S07C.
- Keep PCR-S07D, Release 3 completion, S10, `project_outline`, generated
  protobufs, production concurrency, original dirty paths, push, merge,
  deployment, and `server/**` inactive or excluded.

### Activation evidence

- The official race and focused x10 race independently expose the exact same
  test-local read/write conflict. The detector, source lines, and outbox reader
  identify a lockable test clock; no product repair or human decision remains.
- Immutable v33 preserves v31/v32 behavior and adds exactly one Bootstrap test
  path. The Human Customer's confirmed continuous direction provides successor
  authority; activation claims no remediation or PASS.

### Current evidence

- Commit `48c95172` protects both injected clock reads and the single advance
  with one test-local `RWMutex`; production concurrency and the outbox remain
  unchanged. Focused Bootstrap race x10 passes. After one retained transient
  wrapper-only NON-PASS, the unchanged official seven-package race passes all
  packages in 299.5 seconds with Scoop GCC 15.2.0.
- Focused S07C, complete Workspace, backend `make check`, full Core 628/628,
  full Views 1683/1683, strict type/lint, root forced typecheck, and production
  Web build pass. Root forced tests retain two pre-existing Team Control
  aggregate-load timeouts while the exact file passes 10/10; broad Views lint
  retains only the recorded unrelated Knowledge/Skills findings.
- Fresh SQLite, real Canonical HTTP, production Web, authenticated owner, and
  distinct reviewer prove every installed v33 scenario through retired v11:
  all four stages, multi-Issue fail-closed behavior, latest-conditional
  revocation, current/effective v10/v7 divergence, unlink, deleted-Issue
  exclusion, restart, reload, and retained effective visibility.
- Membership setup used a disclosed direct fresh-database fixture because the
  Canonical invitation endpoint is uninstalled. The in-app browser could not
  reach the host loopback, so the repository Playwright fallback used installed
  Chrome. Production `next start` logged WebSocket-upgrade 403 and invitation
  404 diagnostics; reload supplied the plan-authorized refresh proof. No Next
  overlay appeared. Dedicated ports and runtime artifacts are removed.
- Exact candidate, hashes, trailers, clean/dirty isolation, and fresh
  independent dual review complete R38.4 with both decisions BLOCK. The exact
  findings are persisted-content validation, active-link baseline ownership,
  shared-view query-error projection, and missing executable query-count-bound
  evidence. r038 is review-blocked; PCR-S07C is not closed.

## PCR-001-S07C-R39

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-S07C-R39`
- Task-Revision: `r039`
- Work-Item: `PCR-S07C`
- Title: `Close Requirement coverage independent-review findings`
- Status: `complete-independent-reviewed`
- Assignee: `Codex primary agent`
- Independent reviewer: `fresh SPEC PASS and CODE/SECURITY/QUALITY PASS on exact candidate`
- Exact base: `8116b79f075eb7fe972087abbeebbc76e30ef6b9`
- Blocked input tree: `d328692cc8a2724d2524c279c190d55c01286524`
- Blocked input binary patch hash: `943514fc385a78df5e987145b3340d897ff71a6f`
- Plan hash: `81a865472383e853052bf8a904f66591e473f1f4845924296792f4ea0676641f`
- Exact candidate: `47ee4189cb5571ec38ae39480c758d4decad22bd`
- Candidate tree: `d0b7d56b65964e1559e3bbe33aa734f70e2f8eca`
- Candidate binary patch hash: `6d58ab21a2b39a9328c457e00d0a8ea6ffef7e1493e11af8cdc29e5b178ada34`
- Policy bundle: `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646/fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209/81a865472383e853052bf8a904f66591e473f1f4845924296792f4ea0676641f`
- Write boundary: exact paths in immutable `plan_v34.md`; every migration,
  HTTP production handler, Core contract, Bootstrap/Auth path, generated
  protobuf, original dirty path, legacy backend write, and `server/**` path is
  read-only
- Acceptance source: `plan_v34.md` R39.2-R39.5 and story-map PCR-S07C

### Ordered outputs

- RED current/effective valid-JSON invalid domain content and foreign
  Workspace/Project active-link ownership drift against exact r038.
- GREEN only domain-owned persisted-content validation and ownership checking
  inside the existing single per-snapshot Issue-projection query, retaining
  deleted-Issue exclusion and all four frozen stages.
- Prove the real SQLite repository query count is constant for one versus one
  hundred traceable items and within the frozen current-plus-effective bound.
- RED then GREEN the shared-view coverage query error so it renders only a safe
  localized error state, never the successful no-baseline empty state.
- Run focused, complete backend, official race, strict frontend, root,
  production-build, targeted installed, exact scope/trailer/dirty/process, and
  fresh independent dual-review gates before closing PCR-S07C.
- Keep PCR-S07D, Release 3 completion, S10, migrations, Core/public schema,
  generated protobufs, original dirty paths, push, merge, deployment, and
  `server/**` inactive or excluded.

### Activation evidence

- Exact r038 base, tree, binary patch hash, 32-path boundary, plan/policy
  hashes, eight continuous trailer blocks, clean candidate, zero `server/**`,
  zero generated paths, and original dirty-tree preservation are retained.
- Fresh independent review returns both required BLOCK decisions with the four
  concrete findings frozen by v34. The earlier deterministic and installed
  results remain recorded without being reclassified as closure evidence.
- The Human Customer's confirmed continuous Release 3 direction, confirmed
  prerequisite outline authority, and confirmed execution authorize this
  minimal successor. Activation claims no remediation or PASS.

### R39.2-R39.4 current evidence

- Assertion-first persisted-read cases cover current/effective `{}`, invalid
  and duplicate keys, oversized content, and real Canonical HTTP typed failure.
  One domain normalization/validation function now owns every persisted
  Requirement content read; no partial coverage response is returned.
- A real foreign Workspace/Project/Issue proves the coverage snapshot query
  rejects link ownership drift before Issue projection. The real modernc SQLite
  driver counter observes exactly eight repository queries for both one and one
  hundred current-plus-effective items.
- The shared view's query-error RED rendered the successful empty state. It now
  renders only the safe localized alert in en/ja/ko/zh-Hans; true no-baseline
  and valid coverage behavior remain unchanged.
- Visual inspection of the first installed candidate caught a second projection
  of the same ownership drift through baseline Issue links. Repository RED
  reproduced the full foreign identifier/title. The existing baseline-link
  query now validates stored Workspace/Project ownership before returning any
  link; real baseline and coverage HTTP reads both return typed 500 without
  partial or foreign detail.
- Final complete Workspace passes in 15.2 seconds; backend `make check` passes
  in 31.3 seconds; official `make test-race` passes all packages in 108.5
  seconds with Scoop GCC 15.2.0. Earlier focused, complete Core 628/628, complete
  Views 1684/1684, strict type/changed-file lint, root forced typecheck 6/6, and
  production Web 17-page build results remain exact for the unchanged frontend
  tree.
- Broad Views lint remains NON-PASS on 16 errors and two warnings in unrelated
  Knowledge/Skills/editor/search files. Root forced tests remain NON-PASS on
  one pre-existing Team Control five-second aggregate timeout; its exact file
  passes 10/10. Neither result is renamed PASS.
- Final fresh-database installed acceptance passes 38 checks using real
  Canonical HTTP and production Web: valid linked rendering, corrupt-content
  typed failure and safe alert, foreign baseline/coverage typed failure without
  foreign detail, and restored valid rendering. In-app Browser host-loopback
  isolation, production WebSocket 403 reconnects, and uninstalled invitations
  404 remain disclosed. Ports 3018/38139/39139 are closed.

### R39.5 closure evidence

- Exact candidate `47ee4189cb5571ec38ae39480c758d4decad22bd` has tree
  `d0b7d56b65964e1559e3bbe33aa734f70e2f8eca` and binary patch SHA-256
  `6d58ab21a2b39a9328c457e00d0a8ea6ffef7e1493e11af8cdc29e5b178ada34`.
  Its 15 paths contain zero `server/**` and zero generated paths; the isolated
  worktree is clean and overlaps none of the original worktree's 25 dirty
  entries.
- Commits `35db90f9`, `63e666df`, `eef133c2`, and `47ee4189` each expose exactly
  one complete, continuous, ordered nine-field trailer block. v31-v34 and both
  policy hashes match; `diff --check` passes.
- All task-owned processes are stopped and ports 3018/38139/39139 have no
  listener. Two evidence directories remain outside the repository because the
  host policy denied exact file removal; they are not candidate paths and do
  not leave a running process or listener.
- Fresh independent read-only review of that exact candidate returns
  `SPEC PASS` and `CODE/SECURITY/QUALITY PASS`. It independently confirms the
  four r038 blockers and the installed sibling-baseline leak are closed, the
  eight-query bound is executable, the scope and traceability are exact, and
  every retained broad NON-PASS/fixture/browser diagnostic is honestly
  disclosed.
- r039 and PCR-S07C are `complete-independent-reviewed`. PCR-S07D and Release 3
  completion remain inactive pending their own governed plan and DoneGates;
  push, merge, deployment, generated protobufs, and `server/**` remain
  excluded.

## PCR-001-S07D-R40

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-S07D-R40`
- Task-Revision: `r040`
- Work-Item: `PCR-S07D`
- Title: `Publish governed Retrospectives and materialize action items`
- Status: `superseded-before-product-commit`
- Assignee: `Codex primary agent`
- Independent reviewer: `fresh dual decision required before closure`
- Exact base: `1d515efcca0919eed1e8a811c53d015efa89dfa3`
- Predecessor reviewed candidate: `47ee4189cb5571ec38ae39480c758d4decad22bd`
- Predecessor reviewed tree: `d0b7d56b65964e1559e3bbe33aa734f70e2f8eca`
- Plan hash: `c2c112bb1b903caad6f87e1de14c8ebdb4d5f4ff89665b47bbc734a79c10237a`
- Policy bundle: `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646/fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209/c2c112bb1b903caad6f87e1de14c8ebdb4d5f4ff89665b47bbc734a79c10237a`
- Write boundary: exact paths in immutable `plan_v35.md`; every generated
  protobuf, public Issue HTTP/proto, unrelated Task/Issue/Knowledge behavior,
  original dirty path, legacy backend write, and `server/**` path is read-only
- Acceptance source: `plan_v35.md` R40.2-R40.6 and story-map PCR-S07D

### Ordered outputs

- Assertion-first Retrospective content/participant domain, additive migration,
  immutable revision/history, draft/publication/supersede/archive lifecycle,
  current transaction-owned authorization, governance/audit/outbox, persisted
  failure closure, project cleanup, and safe down guards.
- Assertion-first content-free action target claims, omitted-kind Task default,
  explicit Issue, stable idempotent replay/conflict, interrupted/concurrent
  recovery, and immutable source links only through injected Task/Issue creation
  contracts.
- Strict authenticated HTTP and Core contracts with opaque cursors, typed
  problems, server access projection, Workspace-scoped queries, stable retry
  keys, and malformed-response denial.
- Loaded four-locale shared Project UI for drafts, participants/facilitators,
  publication/revision history, action targets/links, errors, and archive;
  remove the unsupported automatic-Knowledge-review claim.
- Complete backend/race/frontend/root/build gates, fresh real-HTTP production-Web
  two-identity installed acceptance, exact scope/trailer/dirty/process audit,
  and fresh independent dual PASS before closing PCR-S07D.
- Keep Release 3 completion, S08+, S10, generated protobufs, automatic Knowledge
  integration, realtime Retrospectives, permanent delete, re-target/unlink,
  original dirty paths, push, merge, deployment, and `server/**` inactive or
  excluded.

### Activation evidence

- Exact r039 candidate/tree/binary hash, 15-path scope, four continuous
  nine-trailer commits, clean candidate, original dirty-tree isolation, closed
  ports, and fresh independent `SPEC PASS` plus
  `CODE/SECURITY/QUALITY PASS` close PCR-S07C at governance commit `1d515efc`.
- Read-only fixed-point discovery verifies the frozen S07D story/contract,
  partial legacy Core/Views surface, missing Canonical backend, existing Task
  idempotency, and the explicit Issue-idempotency gap. It identifies no
  remaining human-owned scope decision; v35 resolves the facilitator and
  cross-contract recovery rules conservatively.
- The Human Customer's confirmed continuous Release 3 direction, confirmed
  prerequisite outline authority, and repeated confirmed execution authorize
  this exact successor. Activation claims no product implementation or PASS.

## PCR-001-S07D-R41

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-S07D-R41`
- Task-Revision: `r041`
- Work-Item: `PCR-S07D`
- Title: `Complete governed Retrospectives across both Project deletion entries`
- Status: `superseded-before-product-commit`
- Assignee: `Codex primary agent`
- Independent reviewer: `fresh dual decision required before closure`
- Exact base: `fbfc1c05b622e08540564b235661b53cc2894aad`
- Predecessor task: `PCR-001-S07D-R40` (`superseded-before-product-commit`)
- Plan hash: `e652d38ebbc1d697d822554fe689aa842183ff6721f6e0e5335c50d05abb3c5d`
- Policy bundle: `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646/fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209/e652d38ebbc1d697d822554fe689aa842183ff6721f6e0e5335c50d05abb3c5d`
- Write boundary: every exact v35 path plus only
  `backend/internal/modules/workspace/internal/infrastructure/sqlite/project_repository.go`;
  generated paths, original dirty paths, legacy backend writes, and every
  `server/**` path remain read-only
- Acceptance source: `plan_v36.md` R41.2-R41.6, inherited complete v35
  acceptance, and story-map PCR-S07D

### Ordered outputs

- Complete v35 R40.2-R40.6 unchanged as v36 R41.2-R41.6.
- Add one assertion-first shared Retrospective Project-deletion cleanup used
  inside both installed Project deletion transactions, with target/evidence
  retention, ownership isolation, and rollback preservation.
- Freeze one exact candidate and require every inherited deterministic,
  installed, scope, process, trailer, and fresh independent dual-review gate.
- Keep Release 3 completion, S08+, S10, generated protobufs, automatic
  Knowledge integration, realtime Retrospectives, permanent delete,
  re-target/unlink, original dirty paths, push, merge, deployment, and
  `server/**` inactive or excluded.

### Activation evidence

- Read-only dependency tracing found exactly two installed Project deletion
  repositories and no third canonical deletion entry. Both own the same
  `workspace_projects` identity; v35 authorized only one repository file.
- r040 committed only governance activation `fbfc1c05`; all assertion-first
  S07D product work remains uncommitted while v36 corrects the boundary.
- The Human Customer's confirmed continuous Release 3 authority permits this
  minimal successor without broadening the frozen product goal. Activation
  claims no implementation or PASS.

## PCR-001-S07D-R42

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-S07D-R42`
- Task-Revision: `r042`
- Work-Item: `PCR-S07D`
- Title: `Complete governed Retrospectives with installed migration-count evidence`
- Status: `scope-blocked-before-independent-review`
- Assignee: `Codex primary agent`
- Independent reviewer: `fresh dual decision required before closure`
- Exact base: `24d3043b322ae27add8023507330f3b9d55b1d95`
- Predecessor task: `PCR-001-S07D-R41` (`superseded-before-product-commit`)
- Plan hash: `9ef4d4aa9dca5bf44142b30dc4f6edfff31694b16cf83656a5e294df1b92050c`
- Policy bundle: `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646/fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209/9ef4d4aa9dca5bf44142b30dc4f6edfff31694b16cf83656a5e294df1b92050c`
- Write boundary: every exact v36 path plus only
  `backend/internal/modules/workspace/sqlite_workspace_services_test.go`;
  generated paths, original dirty paths, legacy backend writes, and every
  `server/**` path remain read-only
- Acceptance source: `plan_v37.md` R42.2-R42.6, inherited complete v35-v36
  acceptance, and story-map PCR-S07D

### Ordered outputs

- Complete inherited R41.2 and update only the two exact migration-count
  assertions from 19 to 20, then pass the unchanged complete Workspace graph.
- Execute inherited target/HTTP/Runtime, Core/Views, installed-verification,
  exact-candidate, and fresh independent-review outputs without product-scope
  change.
- Keep Release 3 completion, S08+, S10, generated protobufs, automatic
  Knowledge integration, realtime Retrospectives, permanent delete,
  re-target/unlink, original dirty paths, push, merge, deployment, and
  `server/**` inactive or excluded.

### Activation evidence

- Complete Workspace execution found exactly two stale count-19 assertions;
  all subpackages other than the root assertion package passed, including the
  159.9-second SQLite integration package.
- Exact repository-wide search found the already-authorized
  `sqlite_persistence_test.go` and the single omitted
  `sqlite_workspace_services_test.go`, with no third required path.
- r041 committed only governance activation `24d3043b`; all S07D product work
  remains uncommitted. Activation claims no implementation or PASS.
- R42.2-R42.4 produced candidate `f3a77a6c418332e1f0ea2c65073c4be87c41140d`
  with tree `92d136706f61e4404012ae28aa4d0928c57a1bda`, 45 product paths,
  zero `server/**` paths, and zero generated paths. Focused, complete, race,
  frontend, build, restart, and installed lifecycle evidence was collected.
- R42.6 found one exact allowed-path mismatch: immutable v35 lists the
  nonexistent `packages/core/implementation-knowledge/mutations.test.ts`,
  while the existing repository test and candidate path end in `.tsx`.
  Therefore the r042 candidate is blocked before review; its passing evidence
  is not a closure PASS and the branch remains provenance only.

## PCR-001-S07D-R43

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-S07D-R43`
- Task-Revision: `r043`
- Work-Item: `PCR-S07D`
- Title: `Complete governed Retrospectives with exact Core test-path authority`
- Status: `complete-independent-reviewed`
- Assignee: `Codex primary agent`
- Independent reviewer: `fresh SPEC PASS and CODE/SECURITY/QUALITY PASS on exact candidate`
- Exact base: `28e7a56c99855d21ed39494d2726318be5d64cd0`
- Predecessor task: `PCR-001-S07D-R42` (`scope-blocked-before-independent-review`)
- Scope-blocked predecessor candidate: `f3a77a6c418332e1f0ea2c65073c4be87c41140d`
- Scope-blocked predecessor tree: `92d136706f61e4404012ae28aa4d0928c57a1bda`
- Plan hash: `3d82ae02163a19f2ba2912db433402111b95699dbb1c204e6016abfadbe3c54d`
- Exact candidate: `64091302b703a4590bdbe88d154f65fec9d6b37c`
- Candidate tree: `e696d67ad72aad52bc53e4a6bfe3211aac2f89d7`
- Complete binary patch hash: `b0ac56383905ace9581f5c575e2e7321116253cc42b03a18dcc1661d1add53fe`
- Product binary patch hash: `a6ebeda199614944fb254d8a5c184cb2d34dc7790c511bae6cd657f6772bcfef`
- Policy bundle: `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646/fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209/3d82ae02163a19f2ba2912db433402111b95699dbb1c204e6016abfadbe3c54d`
- Write boundary: every exact v37 path with the nonexistent
  `packages/core/implementation-knowledge/mutations.test.ts` replaced
  one-for-one by the existing
  `packages/core/implementation-knowledge/mutations.test.tsx`; generated paths,
  original dirty paths, legacy backend writes, and every `server/**` path remain
  read-only
- Acceptance source: `plan_v38.md` R43.2-R43.6, inherited complete v35-v37
  acceptance, and story-map PCR-S07D

### Ordered outputs

- Preserve the r042 branch unchanged and replay the same 45-path product patch
  only after v38 activation, split into authority, target/HTTP/Runtime, bounded
  snapshot follow-up, and strict Core/Views commits with r043 trailers.
- Prove the r043 product diff is byte-identical to the blocked r042 product
  patch and that only governance records differ between the two candidates.
- Rerun every inherited deterministic, complete, official race, frontend,
  production-build, installed lifecycle/restart/archive, exact candidate,
  process, trailer, and dirty-tree gate without converting any NON-PASS.
- Obtain fresh independent `SPEC PASS` and `CODE/SECURITY/QUALITY PASS` before
  closing PCR-S07D; keep Release 3 completion for a separate aggregate DoneGate.
- Keep S08+, S10, generated protobufs, automatic Knowledge integration,
  realtime Retrospectives, permanent delete, re-target/unlink, original dirty
  paths, push, merge, deployment, external services, and `server/**` inactive
  or excluded.

### Activation evidence

- Exact audit of the 45-path predecessor product diff found 44 literal path
  matches and the sole `.ts`/`.tsx` spelling mismatch. The existing `.tsx`
  file is the intended mutation test; no `.ts` file exists at the valid base or
  blocked candidate.
- r043 starts from governance-only activation `28e7a56c`, before all r042
  product commits. Candidate `f3a77a6c` and branch
  `codex/release3-s07d-r042-scope-blocked` remain untouched provenance.
- v38 changes no product contract or behavior and authorizes only the exact
  existing `.tsx` path. Activation claims no implementation, verification, or
  independent-review PASS.

### R43.6 closure evidence

- Exact candidate `64091302b703a4590bdbe88d154f65fec9d6b37c` has tree
  `e696d67ad72aad52bc53e4a6bfe3211aac2f89d7`. Its 45-path product patch is
  byte-identical to preserved blocked r042 at SHA-256
  `a6ebeda199614944fb254d8a5c184cb2d34dc7790c511bae6cd657f6772bcfef`.
  The complete exact-base diff has five governance plus 45 product paths and
  binary SHA-256
  `b0ac56383905ace9581f5c575e2e7321116253cc42b03a18dcc1661d1add53fe`.
- All five r043 commits contain exactly one complete, continuous, ordered
  nine-field trailer block. v38 and both policy hashes match; `diff --check`
  passes; candidate `server/**` and generated path counts are zero. The
  isolated candidate is clean and overlaps none of the original worktree's 25
  dirty entries.
- Fresh focused Retrospective/domain/application/SQLite/HTTP/bootstrap and
  complete Workspace checks, backend `make check`, Core 635/635, Views
  1688/1688, Core/Views and forced-root typechecks, exact changed-file lint,
  and the production Web build pass. The first official race run failed under
  C-drive temporary-disk pressure; the unchanged full gate passed after
  redirecting only Go/OS temporary caches to F. Broad Views lint retains 16
  unrelated errors and two warnings. Forced root tests retain three aggregate
  five-second load timeouts (1685/1688), while the exact affected files pass
  44/44. These are retained NON-PASS evidence, not waivers.
- Fresh production-Web two-identity installed acceptance created Project
  `e4c67e8b-90dc-42f9-bdc0-d8b35e2b879c`, denied unauthorized facilitator
  creation, published/superseded complete Retrospective revisions, created
  default Task `b65287bf-51ee-443f-a45f-110238715b81` and explicit Issue
  `5343f691-a13d-451a-8d9e-e20140002d5f` from immutable source revision 3,
  proved byte-identical same-key replay, restarted the real backend, and
  archived Retrospective `8e193f3b-8c1c-4a6c-9413-18ded0f4e5a5` at revision
  5 without losing either target or link. The loaded page had one main, no
  Next overlay, and zero console errors; expected WebSocket reconnect and
  uninstalled-invitations warnings remain disclosed.
- Browser tab, binaries, database, logs, temporary caches, and the external
  acceptance directory are removed; owned processes are stopped and ports
  3019/3020/38140/38141/39140/39141 have no listener.
- Fresh independent read-only review returns `SPEC PASS` and
  `CODE/SECURITY/QUALITY PASS`. It independently verifies the exact candidate,
  immutable revision/authorization/cleanup contract, resumable Task/Issue
  target claims, strict Core/Views/runtime projection, installed evidence,
  scope, policy hashes, trailers, and honest NON-PASS disclosures.
- r043 and PCR-S07D are `complete-independent-reviewed`. PCR-S07A-D are all
  closed; Release 3 completion remains pending a separately frozen aggregate
  DoneGate. S08+, S10, generated protobufs, push, merge, deployment, original
  dirty paths, and `server/**` remain inactive or excluded.

## PCR-001-R3G-R44

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-R3G-R44`
- Task-Revision: `r044`
- Work-Item: `PCR-RELEASE-3-DONEGATE`
- Title: `Prove the aggregate Release 3 candidate and close the release`
- Status: `audit-blocked-before-capability-matrix-write`
- Assignee: `Codex primary agent`
- Independent reviewer: `not reached because R44.2 audit blocked`
- Exact base: `8150a0e53defe1562c5ea5b41de34bbdba3a178e`
- Release 3 base: `80d92b14b1f5a1525fbce0c60ce992e28c7f0e8b`
- Predecessor task: `PCR-001-S07D-R43` (`complete-independent-reviewed`)
- Plan hash: `ba58180a641ba2bbcbead3673c9f19c987a85e001a581137e3850eb36158f909`
- Policy bundle: `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646/fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209/ba58180a641ba2bbcbead3673c9f19c987a85e001a581137e3850eb36158f909`
- Write boundary: immutable `plan_v39.md`, the four mutable roadmap governance
  records, and evidence-backed Release 3 reconciliation only in
  `capability-matrix.md`; every product, test, migration, locale, config,
  dependency, generated, original dirty, legacy backend, and `server/**` path
  is read-only
- Acceptance source: `plan_v39.md` R44.2-R44.6 and story-map Release 3
  aggregate DoneGate

### Ordered outputs

- Verify the exact S07A-D candidate/tree/closure tuples, plan v26-v39 hashes,
  complete Release 3 trailer lineage, path scope, product-byte stability,
  original dirty-tree isolation, and zero `server/**`/generated paths.
- Reconcile only the Release 3 capability-matrix evidence, then freshly execute
  complete Workspace, backend check/race, Core/Views test/type/lint, forced-root
  aggregate, and production-build gates without converting retained NON-PASS.
- Prove Resources, Requirements/coverage/minimal outline, and Retrospectives
  coexist in one fresh real Canonical database and production Web runtime with
  two authenticated identities, restart persistence, visible history/links,
  strict isolation, and complete process/artifact cleanup.
- Freeze one exact documentation-only candidate and obtain fresh independent
  `SPEC PASS` plus `CODE/SECURITY/QUALITY PASS` before marking Release 3
  complete with zero active tasks.
- Keep every product path, Release 4/S08+, S10 expansion, generated protobufs,
  external services, original dirty paths, push, merge, deployment, legacy
  backend writes, and `server/**` inactive or excluded.

### Activation evidence

- Exact base `8150a0e5` closes PCR-S07D after fresh dual PASS. Story candidates
  `b3828be7`, `cd943960`, `47ee4189`, and `64091302` plus closures `07aef1a5`,
  `f5695de8`, `1d515efc`, and `8150a0e5` resolve in dependency order.
- Read-only matrix inspection confirms its Project Resources, Requirements, and
  Retrospectives rows still describe the historical static baseline rather
  than the now-reviewed Release 3 runtime; v39 limits correction to those rows
  and explicit aggregate evidence text.
- The Human Customer's confirmed continuous Release 3 completion direction,
  repeated confirmed execution, and confirmed prerequisite minimal outline
  authority authorize this exact governance-only successor. Activation claims
  no aggregate verification, review PASS, or Release 3 completion.

### R44.2 audit block

- All four frozen candidate/tree/closure tuples resolve exactly. Every v26-v39
  plan SHA-256 matches a registered value. The Release 3 range has 131 paths:
  114 product plus 17 roadmap, zero `server/**`, zero generated, and r044 has
  zero product-byte/path drift from S07D closure.
- The first audit script failed on a PowerShell dynamic revision suffix; the
  second used an invalid universal predecessor-hash assumption; the per-commit
  form then timed out. These are retained tooling NON-PASS diagnostics. The
  optimized single-read audit reached a valid governance result.
- Commit `71afb3c33a4d82431a8016cb195a97e5a36d8646` has one continuous
  ordered eight-field block and no `Issue`. That exactly matches its v26/r031
  frozen authority and the policy rule that `Issue` is included only when
  present. The other 41 commits through r044 activation have continuous ordered
  nine-field blocks.
- v39 nevertheless requires nine fields on every historical commit. The
  mismatch is a v39 stop condition and cannot be waived or repaired by history
  rewriting. r044 is `audit-blocked-before-capability-matrix-write`; it records
  no deterministic/installed aggregate PASS and changes no capability matrix or
  product byte.

## PCR-001-R3G-R45

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-R3G-R45`
- Task-Revision: `r045`
- Work-Item: `PCR-RELEASE-3-DONEGATE`
- Title: `Complete the Release 3 aggregate gate with policy-correct trailer audit`
- Status: `review-blocked-before-closure`
- Assignee: `Codex primary agent`
- Independent reviewer: `Codex independent subagent release3_s07a_discovery`
- Exact base: `a1f6a934370ddbc2d645035767c80a24edf2ad4c`
- Release 3 base: `80d92b14b1f5a1525fbce0c60ce992e28c7f0e8b`
- Predecessor task: `PCR-001-R3G-R44` (`audit-blocked-before-capability-matrix-write`)
- Plan hash: `1ee924cf9170866ba312f9b71b20e081008541dbe935f0fb56351fef2f71f3a7`
- Policy bundle: `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646/fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209/1ee924cf9170866ba312f9b71b20e081008541dbe935f0fb56351fef2f71f3a7`
- Write boundary: immutable `plan_v40.md`, the four mutable roadmap governance
  records, and evidence-backed Release 3 reconciliation only in
  `capability-matrix.md`; all product and `server/**` paths are read-only
- Acceptance source: `plan_v40.md` R45.2-R45.6 and inherited complete v39 gate

### Ordered outputs

- Rerun all v39 aggregate lineage/scope/matrix, deterministic, installed,
  exact-candidate, cleanup, and independent-review outputs unchanged.
- For lineage only, require the exact eight-field v26/r031 activation block and
  exact nine-field blocks on every subsequent Release 3/r044/r045 commit.
- Preserve all r044 tooling diagnostics and the valid audit BLOCK; do not amend
  v26-v39 or rewrite any historical commit.
- Keep every product path, Release 4/S08+, S10 expansion, generated protobufs,
  external services, original dirty paths, push, merge, deployment, legacy
  backend writes, and `server/**` inactive or excluded.

### Activation evidence

- r044 changes only its five activation documents and zero product bytes; the
  capability matrix remains unchanged at successor activation.
- Current direct original-worktree status still exposes the 25 pre-existing
  dirty entries excluded from all Release 3/r045 writes.
- v40 changes only the false uniform-nine-field audit predicate and retains
  every product, runtime, acceptance, scope, cleanup, and review obligation.
  Activation claims no aggregate verification, review PASS, or Release 3
  completion.

### R45.2 lineage and capability-matrix evidence

- The corrected single-read audit passes 43 commits: exact frozen eight-field
  v26 activation followed by 42 exact nine-field commits. Project/plan IDs and
  every per-version policy bundle match the computed v26-v40 plan hashes and
  the two governing policy hashes.
- All four reviewed candidate/tree/closure tuples resolve. Exact Release 3
  closure tree is `2caa262bc714b4fb08755d470e877b4ea78850c9`; the range from
  `80d92b14` contains 131 paths: 114 product plus 17 roadmap, zero `server/**`,
  zero generated, and zero overlap with the original worktree's live 25 dirty
  entries. r045 changes five activation documents and zero product paths before
  this matrix evidence commit.
- Using UTF-8 joined Git binary-diff lines with one terminal LF, the complete
  Release 3 patch SHA-256 is
  `34d5b356e382f037fdddac2371bb467945c50fee74ab9ea5f5fd8d9d08fc56a2`;
  excluding the roadmap directory, the 114-path product patch SHA-256 is
  `0a9e59e2aab7b23d3c6dad4b69c54df3f0c1ea702b7d362838b3c7dcc7668aa0`.
- Only the Project Resources, Requirements, and Retrospectives matrix rows and
  an explicit scoped-reconciliation note are updated from their historical v1
  static language. Other rows remain historically dated. No aggregate
  deterministic, installed, independent-review, or Release 3 completion PASS is
  claimed at R45.2.

### R45.3 deterministic evidence

- With `GOTMPDIR`, `GOCACHE`, `TMP`, and `TEMP` rooted in the task-owned F-drive
  directory, the separate complete Workspace graph passes in 123.4 seconds.
  Backend `make check` passes in 557.8 seconds including format, policy,
  generated, vet, and `go test ./...`; the policy checker confirms both
  `server/**` and canonical backend dependency boundaries.
- The unchanged full official `make test-race` passes `./...` in 546 seconds
  using repository-selected MinGW GCC 15.2.0. No package override, reduced
  scope, loader waiver, or post-failure retry is used.
- Core passes 103 files/635 tests in 78.3 seconds; Views passes 169 files/1688
  tests in 224.1 seconds. Core and Views typechecks pass. Core broad lint passes;
  exact lint of all 23 Core and 11 Views TypeScript paths changed in Release 3
  passes.
- Broad Views lint remains NON-PASS with 16 errors and two warnings in unrelated
  editor, Knowledge, search, and Skills files. Existing jsdom canvas/navigation,
  React `act`, and missing-test-i18n warnings remain disclosed.
- Forced root typecheck passes 6/6 tasks with zero cached. Forced root test is
  NON-PASS at 4/5 tasks because Login and Team Control each hit a five-second
  timeout, leaving Views 1686/1688; those exact two files pass 2/2 files and
  44/44 tests, while the separate complete Views run passes 1688/1688. The root
  result is not renamed PASS.
- Production Web build passes with Next 16.2.6 and 17/17 static pages. Its
  generated `apps/web/next-env.d.ts` one-line rewrite is restored to exact
  pre-build SHA-256
  `83a6738771334a63124c8acf38250eccd39fd0aba62846bb0815d952a7936205`;
  worktree and product drift are zero afterward.
- The task-created F-drive gate cache contains 747,659,709 bytes. Two exact,
  pre-validated PowerShell removal requests were rejected by host policy before
  execution. The directory remains outside the repository with no owned
  process/listener; no alternate deletion tool is used. This cleanup limitation
  remains for independent review.
- R45.4 combined installed acceptance is complete. Final exact-candidate audit,
  fresh dual review, and Release 3 closure remain pending.

### R45.4 combined installed evidence

- One fresh SQLite database, real Canonical HTTP backend, production Next
  16.2.6 Web, two independently authenticated identities, and exactly one
  Project pass the combined Release 3 journey. Loaded flags are
  `project_resources=true`, `project_requirements=true`,
  `project_retrospectives=true`, and full `project_outline=false`; the confirmed
  minimal root prerequisite remains available through Requirements.
- Project `d48a22fa-ce08-4638-8fce-4be579106181` creates, reorders, archives,
  and restores Resources `80bd9b8b-f8e6-488f-b588-d1c4a34fc490` and
  `2e25dd29-7e4d-42f7-b6fe-2b7f792fb33b`, ending with two active rows at set
  revision 5 and the reordered repository first.
- Requirement baseline `f0ee857a-1bf6-479c-b8b5-fa50634aa614` records 13
  immutable revisions through draft, four Issue links, one root-outline link,
  independent review/approval/freeze, material change, re-review/re-approval,
  and refreeze. Current coverage proves 4 linked, then 4 implemented, then 4
  accepted; final current and effective snapshots are both 4/4 accepted.
- Ordinary-member facilitator self-appointment returns 403. After current
  Project-lead assignment, Retrospective
  `b2769a21-e682-4b83-aa9c-c052bdf0e308` publishes, supersedes, and archives at
  revision 4 with four immutable history entries. Default Task
  `5334d009-43d3-4ae3-8142-76df99a08b08` and explicit Issue
  `7048a2af-986e-4417-8203-91f1515d4442` both retain immutable source revision
  3; repeated same-key Task targeting is byte-identical.
- A real backend stop/start preserves both Resources, all 13 Requirement
  revisions, accepted current/effective coverage, all four Retrospective
  revisions, both links, and both targets. Unknown Project probes return 404
  for Resources, Requirements, and Retrospectives; a foreign Workspace header
  returns 404 without Project-ID disclosure.
- Production Web visibly renders the Resources, archived Retrospective and
  Task/Issue links, four-entry Retrospective history, frozen Requirement v13,
  root outline, 4/4 accepted current/effective coverage, and all 13 Requirement
  history entries. A clean authenticated in-app Browser tab has zero Next
  overlay and zero console error. Known WebSocket reconnect and uninstalled
  invitations 404 warnings remain disclosed.
- Harness diagnostics remain NON-PASS evidence: an invalid denial fixture
  omitted mandatory lessons; an approval assertion confused user and member
  IDs; one PowerShell URL interpolated `?` into a variable name; one minified
  restart probe miscalled its login function and reused reserved `$PID`; and
  two direct production-Web launch requests were policy-rejected before a
  task-owned wrapper started the unchanged server. Corrected executions pass;
  none rewrites these diagnostics as product failures or earlier PASS results.
- Both Browser tabs, all owned processes, and ports 3021/38142/39142 are
  closed. The validated acceptance directory has 2,845 files, 265 directories,
  and 353,616,921 bytes with zero owned process/listener. Its exact recursive
  removal is rejected by host policy before execution, as were the prior
  747,659,709-byte cache removals; no alternate deletion mechanism is used.
- R45.5 exact candidate freeze/audit, fresh aggregate dual review, and R45.6
  Release 3 closure remain pending.

### R45.5 exact candidate and review block

- Exact candidate `a63bf58a0098e5b39bd04f0d96efcbc97e9f6a9f` has tree
  `aba1d1c166c9a5667d24f837c0bf0236b3df5b5e` and r045 binary patch
  SHA-256
  `f9322809dba511553c127c490705c6438943ad6466c631ff504ae2bc23b9260b`.
  It changes exactly the six allowed roadmap paths and zero product bytes; the
  isolated candidate is clean, original dirty count remains 25 with zero
  overlap, `server/**` and generated counts are zero, and ports
  3021/38142/39142 plus owned processes are closed.
- The primary audit reports 46 Release 3 commits as one eight-field plus 45
  continuous nine-field blocks. Fresh independent raw-message and Git-trailer
  parsing disproves that result: `9fb86ea056d253c82ea61b13031550db82eeb526`
  has all nine correct v27/r032 fields but a blank line between every pair, so
  standard Git parsing recognizes only `Policy-Bundle`. Historical commit
  `71afb3c3` remains the separate valid eight-field shape.
- Fresh independent review returns `SPEC BLOCK` and
  `CODE/SECURITY/QUALITY PASS`. It also confirms v39's required external
  evidence-directory removal remains unfulfilled: the exact requests were
  rejected by host policy before execution. Broad lint/root-test NON-PASS and
  installed harness diagnostics are independently accepted as isolated and do
  not add a code/security/quality blocker.
- r045 is `review-blocked-before-closure`; R45.6 is not executed and no Release
  3 completion is claimed.

## PCR-001-R3G-R46

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-R3G-R46`
- Task-Revision: `r046`
- Work-Item: `PCR-RELEASE-3-DONEGATE`
- Title: `Close the Release 3 aggregate gate after exact historical-shape and cleanup disposition review`
- Status: `complete-independent-reviewed`
- Assignee: `Codex primary agent`
- Independent reviewer: `Codex independent subagent release3_s07a_discovery`
- Exact base: `a63bf58a0098e5b39bd04f0d96efcbc97e9f6a9f`
- Release 3 base: `80d92b14b1f5a1525fbce0c60ce992e28c7f0e8b`
- Predecessor task: `PCR-001-R3G-R45` (`review-blocked-before-closure`)
- Plan hash: `f423cd768b007f98e89a5eec51e63baf89eb36534ba139c4f23f0ce765d8b57b`
- Policy bundle: `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646/fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209/f423cd768b007f98e89a5eec51e63baf89eb36534ba139c4f23f0ce765d8b57b`
- Write boundary: immutable `plan_v41.md` plus the four mutable roadmap
  governance records; `capability-matrix.md`, all product bytes, generated
  paths, original dirty paths, and `server/**` are read-only
- Acceptance source: `plan_v41.md` R46.2-R46.4 and its exact inheritance of
  the unchanged R45.3 deterministic and R45.4 installed evidence

### Ordered outputs

- Audit raw commit-message shape and standard Git parsing across the complete
  Release 3 lineage: exact eight-field `71afb3c3`, exact blank-separated
  nine-field `9fb86ea0`, and exact continuous nine-field blocks everywhere
  else.
- Reverify story tuples, registered plan/policy hashes, exact candidate/path/
  product-byte identity, original dirty isolation, closed processes/ports, and
  the exact retained external-artifact inventory.
- Preserve the r045 SPEC BLOCK, the fact that external deletion did not occur,
  all broad NON-PASS diagnostics, and every v39/v40 exclusion.
- Obtain fresh independent SPEC and CODE/SECURITY/QUALITY PASS before closure;
  leave Release 4 inactive and do not alter product or capability-matrix bytes.

### Activation evidence

- Exact r045 candidate/tree/patch identity and its six-document, zero-product-
  drift scope independently match, so it is the immutable r046 base.
- The raw `9fb86ea0` message proves the single blank-separated historical shape;
  v27 and backend policy required the nine registered fields but did not freeze
  continuity. Neither that commit nor any earlier plan is rewritten.
- Task-owned `F:\codex-tmp\release3-r45-gates-5fad0615` currently contains
  11,836 files, 527 directories, and 1,101,286,336 bytes; nested `acceptance`
  contains 2,846 files, 265 directories, and 353,626,627 bytes. The exact path
  is outside both repositories and owns no process/listener. v41 permits only
  this disclosed host-policy-retained disposition after fresh review; it does
  not claim deletion.
- The standing confirmed completion authority activates only r046. Release 3
  completion, Release 4/S08+, product changes, generated protobufs, external
  services, original dirty paths, push, merge, deployment, legacy backend
  writes, and `server/**` remain inactive or excluded.

### R46.2 corrected audit evidence

- Exact activation candidate `b9719f73f161b056b9a00b670d609ad78137f877`
  has tree `d182077505dc50294cea1460cbef3c62a400a3d6` and r046 binary patch
  SHA-256
  `1e3044468c1c2ee34ec80078c2e9da52e575e9beb597ae04f5585c4a2cd8a337`.
  It changes exactly the five v41 paths and no capability-matrix or product
  byte; plan/policy hashes, story tuples, `diff --check`, clean worktree, and
  original dirty isolation pass.
- Raw message plus standard Git parsing validates 47 commits as exactly one
  continuous eight-field `71afb3c3`, one blank-separated nine-field
  `9fb86ea0`, and 45 continuous nine-field messages. Every Task-ID resolves in
  the task register; Task-Revision/Work-Item and plan/revision mapping agree;
  every Policy-Bundle resolves to the actual v26-v41 plan SHA-256 and both
  policy hashes.
- The complete range has 135 paths: 114 product plus 21 roadmap, zero
  `server/**`, zero generated, and zero product drift. Its product-only binary
  patch remains exact at SHA-256
  `0a9e59e2aab7b23d3c6dad4b69c54df3f0c1ea702b7d362838b3c7dcc7668aa0`.
  The original worktree retains 25 dirty entries with zero overlap.
- Ports 3021/38142/39142 and all artifact-root processes are closed. Retained
  `F:\codex-tmp\release3-r45-gates-5fad0615` contains 11,837 files, 527
  directories, and 1,101,304,235 bytes; nested `acceptance` contains 2,847
  files, 265 directories, and 353,644,526 bytes. It is outside both repositories
  and explicitly remains `host-policy-retained-not-deleted`.
- Two initial strict-audit invocations remain tooling NON-PASS: a repeated
  two-process PowerShell/Git pipeline returned an empty parsed stream at valid
  continuous commit `79ceefcf`, and the first diagnostic message then accessed
  a missing Name property. Standalone raw/standard parsing was 9/9. Replacing
  only that nondeterministic pipeline with Git's single-process
  `%(trailers:only)` atom makes the complete unchanged audit pass; no historical
  exception or product result is inferred from the tooling failures.
- R46.2 is complete. R46.3 exact candidate and fresh aggregate dual review
  remain mandatory before R46.4 or Release 3 closure.

### R46.3 exact candidate and independent dual PASS

- Exact r046 candidate `daab0777b110ec6b21645ffe68771263d4619ec5`
  has tree `b012a2c1aa7cc6dae4e016206585e51102e2a69b` and r046 binary patch
  SHA-256
  `528aa873dea5d477be4ddbdef956b643d393204f6811bd19da08c02f5689d482`.
  Its exact five-document scope, clean worktree, immutable v41 hash, zero
  capability-matrix/product drift, and complete continuous r046 trailers pass.
- Final strict audit passes 48 commits as one continuous eight-field
  `71afb3c3`, one blank-separated nine-field `9fb86ea0`, and 46 continuous
  nine-field messages. The range remains 135 paths: 114 product plus 21
  roadmap, with product patch SHA-256
  `0a9e59e2aab7b23d3c6dad4b69c54df3f0c1ea702b7d362838b3c7dcc7668aa0`
  and zero `server/**`, generated, product-drift, or original-dirty overlap.
- Fresh independent review returns `SPEC PASS` and
  `CODE/SECURITY/QUALITY PASS`. It independently verifies candidate/base/tree/
  patch and plan/policy identities, all three lineage shapes, v27 authority,
  unchanged R45.3/R45.4 product evidence, retained broad NON-PASS diagnostics,
  and the bounded external-artifact disposition.
- Independent inventory matches 11,837 files, 527 directories, and
  1,101,307,455 bytes at the retained root; nested `acceptance` is 2,847 files,
  265 directories, and 353,647,746 bytes. Both paths are outside the
  repositories, ports 3021/38142/39142 and owned processes are closed, deletion
  remains explicitly unexecuted, and v41's result is not a general waiver.

### R46.4 closure

- r046 and `PCR-RELEASE-3-DONEGATE` are
  `complete-independent-reviewed`. PCR-S07A-D and the aggregate DoneGate are all
  closed, so Release 3 is `complete-independent-reviewed` with zero active
  tasks.
- Release 4/S08+, S10 expansion, product changes, generated protobufs,
  external services, original dirty paths, push, merge, deployment, legacy
  backend writes, and `server/**` remain inactive or excluded.

## PCR-001-S08A-R047

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-S08A-R047`
- Task-Revision: `r047`
- Work-Item: `PCR-S08A`
- Title: `Selectively integrate the Issue-similarity warning candidate`
- Status: `review-blocked-before-product-code — immutable v42 omitted required dependencies, risks, rollback, and frozen content-hash revalidation`
- Assignee: `Codex primary agent`
- Independent reviewer: `pending assignment after immutable plan write`
- Exact base: `bbc1a452c84945cde3fe633a43610a8f1db3ae77`
- Source evidence: `02359a01`, `496013ea`, `76265a27` from
  `codex/release4-s08a-r070`; these are content sources only, not a merge base
- Write boundary: `plan_v42.md`, the four mutable roadmap records, and the
  exact product/test path list in `plan_v42.md`; no other source paths
- Product-code authority: R47.2 activates only after independent R47.1 SPEC
  PASS; TDD transplant exception is limited to the existing tests in the frozen
  path list
- Acceptance source: `plan_v42.md` R47.1-R47.6

Expected outputs:

- a warning-only, current-workspace Issue similarity endpoint and client flow;
- deterministic backend, Core, and shared-Views coverage for ranking,
  isolation, cache partitioning, unavailable handling, and localization;
- clean exact-scope, static, regression, and dual independent-review evidence;
- fast-forward integration into the canonical branch only after every required
  gate passes; no push, deployment, legacy backend write, or S08B/S09 work.

## PCR-001-S08A-R048

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-S08A-R048`
- Task-Revision: `r048`
- Work-Item: `PCR-S08A`
- Title: `Governed successor for selective Issue-similarity warning integration`
- Status: `active — R48.1 independent SPEC PASS; R48.2 selective transplant active`
- Assignee: `Codex primary agent`
- Independent reviewer: `Codex independent subagent s08a_r48_plan_spec_review — SPEC PASS`
- Exact base: `bbc1a452c84945cde3fe633a43610a8f1db3ae77`
- Frozen plan SHA-256: `9441040d081df945dc84641dc18af057c10d1736c4cb26c832ecaf177f56a583`
- Frozen source product/test path-set SHA-256: `3492158580b5d22780c8c7faab8fb7e18df3162155651dd3dc8e3d05e356ccd3` (34 sorted paths, UTF-8 LF-terminated)
- Frozen policy bundle: `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646/fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209`
- Source evidence: `02359a01`, `496013ea`, `76265a27` from
  `codex/release4-s08a-r070`; these are content sources only, not a merge base
- Revalidation points: R48.1 review, R48.2 start, R48.5 candidate freeze, and
  R48.7 canonical fast-forward acceptance must each match the plan/base/policy/
  source/path-set identities above
- Write boundary: `plan_v43.md`, the four mutable roadmap records, and the
  exact 34 product/test paths in `plan_v43.md`; no other source path
- Product-code authority: R48.1 SPEC PASS is recorded; R48.2 may apply only
  the frozen manifest. TDD transplant exception is limited to those existing
  tests
- Acceptance source: `plan_v43.md` R48.1-R48.7
