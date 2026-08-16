# Product Capability Roadmap Task Register

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Plan-Version: `v2`
- Registry status: `PCR-S01B-1 active`
- Registry revision: `r005`
- Updated: `2026-08-16`

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
- Status: `implementation-complete; verification-blocked`
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
- PCR-S01B-2 remains inactive.

## Queue rules

- No task after PCR-S01B-1 may be enqueued until r005 closes and the next task
  revision freezes its exact base, paths, checks, and active-step pointer.
- A task status in this file is not a substitute for the active-step pointer in
  `plan.md`; both must agree.
- A base, plan hash, policy hash, or allowed-path mismatch stops execution.
- Completion by the assignee is not independent acceptance.
- Customer Acceptance is not delegated to this register.
