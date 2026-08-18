# Product Capability Roadmap Journal

Append-only evidence for Plan-ID `PRODUCT-CAPABILITY-ROADMAP-001`.

## 2026-08-16 — J001 — Discovery and v1 plan authored

- Mode: documentation and read-only discovery only.
- Branch: `codex/multica-six-domain-baseline`.
- Base commit observed before writing:
  `45213820fade7f61294d2287e063bf19fbd015ee`.
- User decision: use all recommended decisions D01-D36.
- Confirmed project model: phase is independent from lifecycle status; initial
  phases are initiation, planning, review, and implementation.
- Confirmed authority: Workspace is product authority; Control Plane remains a
  separate bounded runtime with no dual writes.
- Confirmed outline boundary: CRUD, Issue links, progress, and phase board are
  in scope; automatic Issue planning is deferred to a later proposal.
- Confirmed reminder boundary: durable in-product notification center first;
  email is deferred.
- Confirmed similarity boundary: workspace-scoped hybrid warning, human override
  and duplicate relation; no automatic block, merge, or delete.
- Existing worktree paths observed and excluded:
  `packages/ui/components/ui/input.tsx`,
  `packages/views/issues/components/table-view.tsx`,
  `packages/views/modals/create-issue.tsx`, `.local-runtime/`,
  `docs/code-to-product/`, `packages/views/auth/input-controlled.test.tsx`, and
  `ui/`.
- Backend discovery found partial Todo CRUD, candidate-only Knowledge,
  draft-only Requirements, opt-in project search, in-memory Issue filtering,
  append-only pin positions, and missing Skill administration, Resources,
  Retrospectives, similarity, notifications, project phases, and outlines.
- No product code, migration, runtime configuration, legacy server file, or
  existing plan directory was modified.
- No repository, backend, browser, race, or runtime test was executed during
  discovery.
- Plan state after this entry: v1 documentation complete; no active product
  implementation step; explicit implementation approval still required.

## Evidence entry template

Future entries append without rewriting J001 and include:

```text
Date / Entry-ID / Plan-Step / Task-Revision
Base commit / Candidate commit / Branch / Worktree
Authority and exact allowed paths
Changed files and migration versions
Acceptance claims
Verification commands, exit codes, durations, evidence locations
Runtime, identity, workspace, project, and sanitized request context
Independent reviewer and findings
Rollback evidence
Known limitations, blockers, and next action
```

## 2026-08-16 — J002 — v1 implementation approval and S00 activation

- User authorization: `批准实施`.
- Frozen Plan-ID and version: `PRODUCT-CAPABILITY-ROADMAP-001 / v1`.
- Activated step: `PCR-S00 — contract freeze and clean task creation`.
- Base commit revalidated:
  `45213820fade7f61294d2287e063bf19fbd015ee`.
- Branch revalidated: `codex/multica-six-domain-baseline`.
- Approval was interpreted under the one-active-step invariant. It does not
  activate PCR-S01 or any later product-code step concurrently.
- `plan_v1.md` was frozen with approval metadata before S00 execution began.
- Frozen plan SHA256:
  `4351025652e88083b8ceb25a72488e9d568a445a5f0e3a8793650393086ad0b5`.
- `CLAUDE.md` SHA256:
  `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646`.
- `backend/AGENTS.md` SHA256:
  `fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209`.
- Product-code writes performed: none.

## 2026-08-16 — J003 — S00 contract freeze and S01 pre-enqueue gate

- Added immutable contract candidate `contract-freeze_v1.md` covering data
  authority, route families, API conventions, permissions, state machines,
  similarity, reminders, Skill import limits, outline limits, numeric budgets,
  capability flags, migration order, project backfill, and deferrals.
- Added `task-register.md` revision r001.
- Frozen active documentation task: `PCR-001-S00-R1`.
- Frozen first product task candidate: `PCR-001-S01A-R1`.
- PCR-S01A remains `blocked-before-enqueue` because the Canonical cutover plan
  still reports C9 blocked, no approved repair successor, and no stable
  post-repair integration base.
- S01A overlap risk includes `backend/internal/bootstrap/sqlite.go`,
  `backend/internal/bootstrap/runtime.go`, and installed-runtime tests. Those
  paths cannot be edited until the Canonical repair plan and base are known.
- Existing dirty and untracked user paths were not edited.
- `server/**` was not edited.
- Product code, migrations, configuration, and runtime state were not edited.
- Remaining PCR-S00 gates:
  1. independent review of the frozen contract and task scope;
  2. a stable Canonical successor/base and overlap revalidation;
  3. update PCR-S01A task revision and activate it only after both gates pass.
- Current step remains PCR-S00. It is not marked complete merely because its
  documentation outputs exist.

## 2026-08-16 — J004 — Roadmap paused for Canonical C9 v11 repair

- Human Customer authorized the prerequisite C9 repair.
- Roadmap active step changed to none; PCR-S00 is paused, not completed.
- Canonical plan v11 is the sole active implementation plan.
- PCR-S01A remains blocked before enqueue until v11 produces a stable accepted
  technical base and overlap is revalidated.
- This entry grants no Canonical repair authority beyond its own v11 plan.

## 2026-08-16 — J005 — C9 dependency cleared and PCR-S01A resumed

- Human Customer statement: `C9已经通过，提交所有变动。然后回到PCR-S01A`.
- C9 is recorded as Customer Accepted in its own plan. Full Canonical milestone
  acceptance is not inferred.
- The v11 local offline-install/browser evidence gap remains documented and is
  not represented as a technical PASS.
- Canonical acceptance record commit:
  `387eb64264b42a6e23959f95f4c2c4eb15b45c91`.
- PCR-S00 is closed by explicit Customer direction. Its missing independent
  reviewer is retained as program acceptance debt rather than silently erased.
- PCR-S01A task was rebased and re-frozen as `PCR-001-S01A-R2`, revision r002,
  on `387eb64264b42a6e23959f95f4c2c4eb15b45c91`.
- Policy and approved plan hashes revalidated unchanged.
- Overlap audit found no product-path overlap: C9 v11 changed Issue HTTP paths;
  PCR-S01A is limited to new Workspace authorization contract files and named
  Bootstrap/runtime files.
- PCR-S01A is the sole active product implementation step.

## 2026-08-16 — J006 — PCR-S01A activation base finalized

- Roadmap documentation commit:
  `144997ab5fcd04544f8ffa40a1a75fc79fdb5904`.
- Because that commit created the approved plan directory in Git, the active
  product task was re-frozen as `PCR-001-S01A-R3`, revision r003, with
  `144997ab5fcd04544f8ffa40a1a75fc79fdb5904` as its product-code base.
- Allowed paths and acceptance criteria are unchanged from r002.
- No product code changed in this re-freeze.

## 2026-08-16 — J007 — PCR-S01A implementation complete; verification blocked

- Implemented the frozen capability authorization foundation:
  - 28 named roadmap permission actions with explicit owner/admin/member
    defaults and no implicit agent grants;
  - unknown actions, actor types, roles, and absent providers fail closed;
  - runtime installation requires an injected capability provider;
  - membership and workspace isolation remain server-side;
  - 17 roadmap feature flags are explicit and false without an installed
    provider.
- TDD evidence:
  - `/api/config` RED exposed 14 missing roadmap flags;
  - authorization RED returned `workspace actor is required` instead of the
    required explicit unavailable denial;
  - provider-injection RED failed to compile until the provider seam was added;
  - the focused GREEN commands passed:
    `go test ./internal/modules/workspace/contract -count=1` and
    `go test ./internal/bootstrap -run
    'Test(RuntimeReports|RoadmapFeatureFlags|SQLiteAuthorization|RuntimeRegistersHealthAndModuleRoutes)'
    -count=1`.
- Deterministic evidence:
  - changed Go files are `gofmt` clean;
  - generated output diff is clean;
  - `ci/check-policy.sh` passed using Git for Windows Bash;
  - `go vet ./...` passed with `GOTMPDIR=F:\codex-tmp\pcr-s01a`;
  - no `server/**` diff exists.
- Environment and acceptance blockers:
  - `make check` stopped in its Windows `fmt-check` wrapper with
    `exit was unexpected at this time`;
  - the equivalent direct checks were run, but full `go test ./...` reproduced
    `TestSQLiteRuntimeConcurrentAttachmentUploadsLoseNoReferencesOrFiles`
    returning HTTP 500 and
    `TestAttachmentRepositoryRetriesBusyWriteAcquisition` timing out;
  - a single serial diagnostic rerun reproduced the Bootstrap attachment 500;
  - the failures are outside the S01A changed paths and are not repaired under
    this task;
  - `go test -race` for the frozen packages exited `0xc0000139`, an environment
    limitation and not a PASS;
  - independent review remains unassigned.
- Task `PCR-001-S01A-R3` is therefore implementation-complete but not
  technically accepted. No dependent roadmap step is activated.

## 2026-08-16 — J008 — S01A waiver accepted and S01B design activated

- Human Customer directed `激活S01B` and then answered `确认` to the explicit
  question whether that direction constitutes S01A acceptance plus waiver of:
  1. the attachment concurrency failures;
  2. the Windows race-loader limitation;
  3. the missing independent review.
- S01A is recorded as Customer Accepted with those explicit waivers. The
  underlying technical evidence is preserved and is not upgraded to PASS.
- Read-only dependency discovery for S01B completed in two rounds with
  `new_dependencies = 0`, no remaining retrievable blockers, and status
  `stable` for activation planning.
- Verified ownership boundary: Canonical Workspace must own revision, audit,
  idempotency, and outbox state. The separate Control Plane implementation is
  migration evidence only; direct reads, imports, and dual writes are forbidden.
- Verified migration-policy conflict requiring design resolution before code:
  root policy requires every migration-created index to use a standalone
  `CREATE [UNIQUE] INDEX CONCURRENTLY`, while the current Workspace SQLite
  runner executes ordered migrations inside one transaction and SQLite does not
  implement that PostgreSQL syntax.
- Activated `PCR-001-S01B-R4` as a documentation-only contract and migration
  design task on base `cc61297be42ca5acf1fc47d9ba9d70939f406588`.
- r004 authorizes only the exact roadmap-document paths in the task register.
  It grants no product-code, migration, API, frontend, Control Plane, or
  `server/**` write authority.

## 2026-08-16 — J009 — S01B design and proposed v2 completed

- Completed `s01b-foundation-design.md` under documentation-only task
  `PCR-001-S01B-R4`.
- Ownership decision: Canonical Workspace owns the four governance tables and
  mutation transaction. Control Plane remains evidence only and is not read,
  imported, invoked, or dual-written.
- Transaction decision: each future capability repository owns one dedicated
  SQLite connection and `BEGIN IMMEDIATE`; domain, revision, audit, outbox, and
  replay writes commit or roll back together. Nested transactions are forbidden.
- Migration decision proposed for approval: version 9 adds table definitions
  and composite primary keys only, with no explicit secondary/unique index DDL.
  A measured need for another SQLite index stops execution and requires a new
  plan version.
- Delivery decision: durable outbox uses bounded claims, 60-second leases,
  initial plus three attempts, stable event IDs, deterministic retry tests, and
  dead-letter state. At-least-once crash-window duplicates are explicit.
- Diagnostic decision: owner/admin-only
  `GET /api/operations/governance` returns counts/timestamps and never payload
  content.
- Created `plan_v2.md` with four ordered S01B steps, exact product paths,
  acceptance, deterministic checks, risks, and rollback.
- v2 status is `proposed; awaiting Human Customer approval`. No product code is
  authorized and no implementation step is active.

## 2026-08-16 — J010 — v2 approved and PCR-S01B-1 activated

- Human Customer statement: `批准 PRODUCT-CAPABILITY-ROADMAP-001 v2`.
- `plan_v2.md` SHA256 revalidated as
  `af0486c7625067b2b25de163271ba6f4be153c74331488eb57d9c42ab69ffd30`.
- `plan.md` now names v2 as the approved execution snapshot and activates only
  `PCR-S01B-1 — Contract and migration`.
- Documentation task `PCR-001-S01B-R4` is complete by Customer approval.
- Product task `PCR-001-S01B1-R5` is frozen on product-code base
  `312feda1aeaafb5d1aecffd61a7fbcdcbd7ee3c6` with the exact paths and checks in
  the task register.
- Existing unrelated dirty UI and local-runtime paths remain excluded.
- The next authorized action is a TDD RED in the contract or migration tests.

## 2026-08-16 — J011 — PCR-S01B-1 candidate committed; verification blocked

- Product candidate commit: `3876791`.
- Added Workspace governance contract values and stable errors for trusted
  mutation identity, command revision/idempotency, replay result, audit,
  outbox state/event, diagnostics, sink, and diagnostics reader.
- Added additive migration `000009_workspace_governance` with four
  Workspace-owned `WITHOUT ROWID` tables:
  - `workspace_resource_revisions`;
  - `workspace_mutation_idempotency`;
  - `workspace_audit_entries`;
  - `workspace_outbox_events`.
- Migration contains no foreign key, cascade, trigger, explicit index, or
  secondary unique-index DDL. Strict `CREATE TABLE` prevents an incompatible
  pre-existing object from being silently accepted as an applied migration.
- TDD RED evidence:
  - migration count was 8 instead of 9;
  - governance identity/command/result/audit/outbox/diagnostics contracts were
    absent;
  - empty schema identities and ready events with claims persisted;
  - replay envelopes accepted invalid JSON;
  - migration tables lacked `WITHOUT ROWID`;
  - a conflicting retained object allowed v9 to be recorded without an audit
    table until strict table creation was used.
- GREEN evidence:
  - `go test ./internal/modules/workspace/contract` passed;
  - `go test ./internal/modules/workspace -run
    'Test.*(Governance|Migration|Migrate)'` passed;
  - complete `go test ./internal/modules/workspace/contract
    ./internal/modules/workspace` passed during iteration;
  - retained version-8 upgrade, second run, partial-failure rollback, restart
    persistence, schema constraints, and workspace idempotency isolation pass.
- Equivalent Makefile checks:
  - changed-file `gofmt` clean;
  - generated output diff clean;
  - policy check passed;
  - `go vet ./...` passed.
- Outstanding verification:
  - `go test -race ./internal/modules/workspace/contract
    ./internal/modules/workspace` exited Windows loader code `0xc0000139` and is
    not a PASS;
  - `make check` stopped in the Windows `fmt-check` wrapper with
    `exit was unexpected at this time`;
  - direct full `go test ./...` failed in out-of-scope existing tests:
    `TestSQLiteRuntimeServesCanonicalIssueAttachments` (local-user create 500),
    `TestSQLiteRuntimeConcurrentAttachmentUploadsLoseNoReferencesOrFiles`
    (attachment 500), and
    `TestAttachmentRepositoryRetriesBusyWriteAcquisition` (deadline exceeded).
- No failure was weakened or repaired outside r005 scope. S01B-1 remains active
  and verification-blocked; S01B-2 is not activated.

## 2026-08-16 — J012 — S01B-1 waiver accepted and S01B-2 activated

- Human Customer directed `开始S01B-2` and answered `确认` to the explicit
  question whether that direction accepts S01B-1 and waives:
  1. Windows race exit `0xc0000139`;
  2. the Makefile Windows `fmt-check` wrapper failure;
  3. the three recorded out-of-scope attachment/auth SQLite concurrency
     failures from full `go test ./...`.
- S01B-1 is recorded as Customer Accepted with explicit gate waiver. The
  technical evidence remains unchanged and is not represented as PASS.
- Activated `PCR-001-S01B2-R6` on product-code base `3876791` with the exact
  paths, acceptance, and verification in the task register.
- S01B-2 is limited to the new application service, SQLite repository/helper,
  Workspace constructor, their tests, and roadmap evidence. It cannot retrofit
  existing repositories or compose runtime behavior.
- The next authorized action is a TDD RED for atomic governed mutation behavior.

## 2026-08-16 — J013 — PCR-S01B-2 candidate committed; verification blocked

- Product candidate commit: `a9f04b0`.
- Added an application-layer governance preparer that validates mutation input,
  derives the next resource revision, allowlists audit metadata, and prepares
  ready outbox events without importing SQLite.
- Added an explicitly opt-in SQLite governance repository. It owns one
  connection and `BEGIN IMMEDIATE`, invokes the supplied domain mutation on the
  same connection, and atomically persists resource revision, audit, outbox,
  and idempotency replay state.
- Replay is scoped by workspace, action, and idempotency key. Matching hashes
  return the original response without executing the domain callback; a
  different hash returns the stable idempotency conflict.
- Defensive envelope validation rejects audit or outbox identity, resource, or
  revision drift before the domain callback, including cross-workspace outbox
  injection.
- TDD RED evidence recorded missing application types, missing repository,
  missing five-phase failure controls, missing opt-in constructor, and an
  initially accepted cross-workspace prepared envelope.
- GREEN evidence:
  - `go test ./internal/modules/workspace/internal/application -run Governance`
    passed;
  - `go test ./internal/modules/workspace/internal/infrastructure/sqlite -run
    Governance` passed;
  - `go test ./internal/modules/workspace -run Governance` passed;
  - complete tests for those three packages passed;
  - concurrency proves one same-revision winner and one conflict reporting
    current revision 1;
  - injected aborts after domain, revision, audit, outbox, and replay each leave
    all domain and governance tables unchanged.
- Direct Windows-compatible formatting, generated-clean, policy, and
  `go vet ./...` checks passed.
- Outstanding verification:
  - the frozen three-package race command exited `0xc0000139` for each package
    and is not a PASS;
  - `make check` stopped in the known Windows `fmt-check` wrapper;
  - full `go test ./...` reproduced only the indexed out-of-scope attachment
    concurrency failures in Bootstrap and Space SQLite.
- No existing capability repository, runtime route/composition, HTTP,
  frontend, Control Plane, generated output, migration, contract, or
  `server/**` path was modified by the product candidate.
- PCR-S01B-2 remains verification-blocked pending independent review and Human
  Customer disposition of the outstanding gates. PCR-S01B-3 is not active.

## 2026-08-17 — J014 — S01B-2 waiver accepted and S01B-3 activated

- Human Customer directed `开始S01B-3` and answered `确认` to the explicit
  question whether that direction accepts S01B-2 and waives:
  1. Windows race exit `0xc0000139`;
  2. the Makefile Windows `fmt-check` wrapper failure;
  3. the two recorded out-of-scope attachment concurrency failures from full
     `go test ./...`.
- The Customer also confirmed that independent review is deferred to S01B-4.
- S01B-2 is recorded as Customer Accepted with explicit gate waiver. Its
  technical evidence remains unchanged and is not represented as PASS.
- Activated `PCR-001-S01B3-R7` on product-code base `40d65ad` with the exact
  paths, acceptance, and verification in the task register.
- S01B-3 is limited to outbox claim/delivery/retry/dead-letter/replay,
  owner/admin diagnostics, explicit worker lifecycle/readiness composition,
  their tests, and roadmap evidence. It cannot add a product capability or
  enable a roadmap feature flag.
- Existing unrelated UI and local-runtime changes remain excluded.
- The next authorized product action is a TDD RED for SQLite outbox claiming.

## 2026-08-17 — J015 — PCR-S01B-3 candidate committed; verification blocked

- Product candidate commit: `f0d86d9`.
- Added SQLite outbox transitions for bounded claim, 60-second lease reclaim,
  token-guarded delivered/retry/dead-letter changes, owner-authorized replay,
  and Workspace-scoped diagnostics.
- Empty queues perform a read-only eligibility check and avoid
  `BEGIN IMMEDIATE`, preventing the one-second worker poll from adding idle
  writer contention to existing attachment paths.
- Added an application dispatcher with default batch 100, hard cap 500,
  initial delivery plus three retries, exponential minute backoff, injected
  deterministic jitter, stable failure codes, and crash-window redelivery.
- Added `GET /api/operations/governance` with owner/admin membership checks.
  Its projection contains counts, durations, timestamps, schema version,
  running/degraded state, and no event payload or audit content.
- Added explicit Workspace worker lifecycle and Canonical composition. Runtime
  starts and closes the worker, readiness checks database and dispatcher, and
  the realtime sink carries stable event ID, aggregate identity, and revision.
- TDD RED evidence included missing claim/ack methods, write-locking an empty
  queue, missing retry/dead-letter/replay/diagnostics methods, missing
  dispatcher types, missing HTTP handler, and missing Runtime lifecycle/sink.
- GREEN evidence:
  - all four frozen focused commands pass;
  - the complete application, SQLite infrastructure, HTTP, and Workspace
    packages pass;
  - the complete Bootstrap package runs all S01B-3 tests successfully and only
    reproduces the indexed attachment concurrency failure.
- Direct Windows-compatible formatting, generated-clean, policy boundary, and
  `go vet ./...` checks pass.
- Outstanding verification:
  - the frozen race command exits `0xc0000139` for all five packages and is not
    a PASS;
  - `make check` stops in the known Windows `fmt-check` wrapper;
  - full `go test ./...` reproduces the existing Bootstrap concurrent
    attachment upload 500 and Space SQLite busy-write deadline failure.
- All roadmap feature flags remain false. No migration, public contract,
  existing capability repository, feature API, frontend, Control Plane,
  generated output, or `server/**` path was changed.
- PCR-S01B-3 remains verification-blocked pending Human Customer disposition
  of the r007 gates. Independent review remains deferred to S01B-4, which is
  not active.

## 2026-08-17 — J016 — v3 approved and PCR-S01B-3R activated

- Human Customer explicitly approved `PRODUCT-CAPABILITY-ROADMAP-001 v3`.
- Activated `PCR-001-S01B3R-R8` on base
  `ab2b49088b108f771045a090b473a8e235dfa09e`.
- The repair scope is limited to cross-platform Backend checks, process-local
  Windows race compiler selection, Space attachment write contention, their
  tests, and roadmap evidence.
- Read-only discovery confirmed the current PATH selects MinGW-w64 GCC 8.1
  from `E:\mingw64\bin`; a race probe compiled with that toolchain exits with
  Windows loader code `0xc0000139`. A process-local PATH selecting the already
  installed Scoop GCC 15.2 starts the race binary; no race test PASS is yet
  claimed.
- The current Backend Makefile uses Unix `find`, `test`, environment assignment,
  and `/dev/null` syntax that is incompatible with the Windows `cmd.exe` shell.
- The Space attachment repository currently combines a five-second SQLite busy
  wait and retries inside an eight-second acquisition budget while concurrent
  writers consume multiple connections. The two retained failing tests remain
  the required RED evidence; the exact repair will be test-driven.
- No product, host, frontend, generated, unrelated dirty, or `server/**` path
  has changed. The next action is the frozen one-shot attachment RED commands.

## 2026-08-17 — J017 — PCR-S01B-3R technical gates passed

- Each indexed attachment regression was run once before production changes.
  Both passed on that single run, confirming their intermittent nature and
  making them unsuitable as deterministic TDD RED evidence by themselves.
- Added deterministic RED tests proving a queued attachment writer consumed a
  second SQLite connection while the first owned the transaction. The
  cancellation case also remained inside SQLite busy handling for about five
  seconds before returning; both reported maximum in-use connections 2 instead
  of 1.
- Added the minimal context-aware repository write slot before attachment
  connection acquisition. Queued creates/deletes no longer consume competing
  connections, cancellation remains authoritative, and the existing finite
  SQLite retry budget is reserved for writers outside the attachment repository.
- Added a Go Backend-check command for platform-neutral formatting, policy, and
  generated-output gates. Its TDD coverage includes CRLF/LF equivalence, real
  formatting drift, generated RPC exclusion, server path rejection, and Go AST
  import detection without string-literal false positives.
- Updated `make fmt-check`, `make policy-check`, and `make generated-clean` to
  use the same Go checker on Windows and Unix. `make check` now executes and
  passes on Windows rather than stopping in the wrapper.
- Added a Windows race runner with a tested resolve-only path. It selects the
  highest-version installed PATH compiler, prepends only that directory in its
  child process, and leaves permanent user/system PATH unchanged.
- Verification passed:
  - attachment repository busy/cancellation/serialization tests, count 10;
  - same-Issue twelve-upload Bootstrap regression, count 10;
  - full `go test ./... -count=1`;
  - `make check` including format, policy, generated, vet, and full tests;
  - `go mod verify`;
  - the frozen five-package `make test-race`, all five packages green using
    Scoop GCC 15.2 with no loader failure;
  - `git diff --check` and empty tracked/untracked `server/**` scope.
- Policy hashes remain unchanged. Existing UI/local-runtime dirty paths were
  preserved and excluded. No commit, push, merge, Customer Acceptance, or
  independent review is claimed; S01B-4 remains inactive.

## 2026-08-17 — J018 — PCR-S01B-3R Customer Accepted

- Human Customer stated `审核通过` after reviewing the completed v3 repair and
  its technical evidence.
- `PCR-S01B-3R` and task revision `r008` are closed as Customer Accepted.
- The acceptance covers the Windows race-loader repair, cross-platform
  `make check`, and the two retained attachment concurrency failures under the
  evidence recorded in J017.
- No new verification result, independent review, commit, push, merge, or S01B-4
  activation is inferred. S01B-4 remains inactive pending a separately frozen
  task, base/policy revalidation, and active-step update.

## 2026-08-17 — J019 — v4 approved and PCR-S01B-4 activated

- Human Customer stated `批准后续动作，按目标持续推进完成 Release 0 — Authority
  and safety foundation` after receiving the exact v4/r009 scope, exclusions,
  stop conditions, and independent-review request.
- Approved immutable `plan_v4.md` and froze `PCR-001-S01B4-R9` on base
  `ab2b49088b108f771045a090b473a8e235dfa09e` with unchanged policy hashes.
- Activated `PCR-S01B-4` as the sole roadmap step. Release 1 remains inactive.
- Authorized actions are limited to committing the accepted r008 candidate,
  running integrated deterministic evidence on the same committed candidate,
  obtaining independent read-only review, and recording Release 0 closure if
  every gate passes.
- Existing UI, local-runtime, code-to-product, and other unrelated dirty paths
  remain excluded. `server/**` remains read-only. Push, merge, deployment, and
  repair of a newly discovered defect are not authorized.
- No candidate commit, new verification result, independent verdict, or Release
  0 completion is claimed by this activation record.

## 2026-08-17 — J020 — candidate verified; independent review blocked closure

- Explicitly staged and audited the twelve v4/r009 paths, with no `server/**`
  or unrelated dirty path, then created candidate commit `b24661f`.
- The first commit message contained every required trace field but separated
  them into individual paragraphs, so Git's trailer parser returned none. The
  commit was immediately amended without changing its tree; final candidate
  `5062e84a65a3ce3114a0d2d54013d37f746836c6` exposes all required trailers.
- Deterministic evidence on the candidate passed:
  - application, Workspace SQLite, HTTP, Workspace composition, and Bootstrap
    packages, each count 1;
  - attachment busy/cancellation/serialization regressions, count 10;
  - twelve-upload Bootstrap concurrency regression, count 10;
  - Backend-check tests and full `go test ./... -count=1`;
  - `go vet ./...`, `go mod verify`, `make fmt-check`, and `make check`;
  - frozen five-package race with process-local Scoop GCC 15.2 and no loader
    waiver;
  - supplemental S01A contract/Bootstrap focused tests and race, replacing the
    historical loader limitation with current green evidence;
  - candidate allowlist, machine-readable trailers, roadmap relative links,
    diff checks, preserved unrelated dirty paths, and empty `server/**` scope.
- After deterministic checks, an independent read-only reviewer examined
  `3876791^..5062e84` against `s01b-foundation-design.md`, plans v2/v4, code, and
  tests. Overall verdict: `SPEC BLOCK`.
- The reviewer and primary-agent verification found five frozen-contract gaps:
  1. mutation preparation has no fail-closed action/resource authority map;
  2. request replay trusts a supplied digest instead of computing versioned
     canonical JSON plus action SHA-256;
  3. replay/audit/outbox envelopes permit secret-bearing values or unrestricted
     JSON despite the frozen secret-free allowlist contract;
  4. outbox transitions use the claim-time clock and only workspace/event/token,
     allowing an expired lease or duplicate event identity to mutate state;
  5. the governance down migration unconditionally deletes non-empty evidence.
- Existing tests do not cover those negative scenarios. Transaction rollback,
  diagnostics authorization/redaction, bounded retry/dead-letter/restart,
  readiness, migration up policy, exact scope, and `server/**` passed review.
- Per plan v4, `SPEC BLOCK` stops closure and forbids in-plan product repair.
  `PCR-S01B-4` is independent-review-blocked, Release 0 is incomplete, and
  Release 1 remains inactive pending a separately approved plan/task.

## 2026-08-17 — J021 — v5/r010 approved and PCR-S01B-5 activated

- Human Customer explicitly approved `PRODUCT-CAPABILITY-ROADMAP-001 v5 /
  r010` after receiving the proposed authority, canonical hash, safe envelope,
  complete claim tuple, legacy-row, and empty-only down defaults.
- Approved immutable `plan_v5.md` and froze `PCR-001-S01B5-R10` on base
  `0218ecbe5457f1afb716780ad44306e5b1b3b075` with unchanged policy hashes.
- Activated `PCR-S01B-5` as the sole roadmap step. Release 0 remains incomplete
  and Release 1 remains inactive.
- The repair may touch only Section 6 paths and must use failing tests before
  production changes. It adds no table, column, index, up migration, public API,
  frontend, Desktop, Control Plane, generated output, or capability behavior.
- Retained legacy governance rows may only fail closed; no cleanup, rewrite,
  migration, publication, or deletion is authorized.
- Existing UI/local-runtime/code-to-product dirty paths remain excluded and
  `server/**` remains read-only. No repair or verification result is claimed by
  this activation record.

## 2026-08-17 — J022 — v5 candidate verified; independent re-review blocked closure

- Implemented the five authorized v5 repair areas with RED/GREEN tests and
  committed exact candidate `ee02403cbed00366a5c25bfd4da8d2ee123cb675`.
  Its 17 paths are all in the v5 Section 6 allowlist; no `server/**`, frontend,
  Control Plane, up migration, generated, or unrelated dirty path is included.
- Candidate behavior includes server-resolved action/resource policy and actor
  identity, application-derived canonical v1 SHA-256 request hashes, strict
  versioned scalar envelopes, opaque prepared mutations with defensive JSON
  copies, post-publish clocks, complete outbox tuple/lease predicates, exact
  dead-letter tuple replay, and transactional empty-only 000009 down behavior.
- Deterministic evidence on the fixed candidate passed:
  - five focused governance/Workspace/Bootstrap packages, count 1;
  - attachment SQLite busy/serialization/cancellation cases, count 10;
  - same-Issue concurrent attachment uploads, count 10;
  - `make check`, `go mod verify`, and full `make test-race` using process-local
    Scoop GCC 15.2;
  - policy hashes, parsed traceability trailers, exact diff, diff check, and an
    empty `server/**` candidate scope.
- Before the candidate was fixed, one full test run observed a single C9
  attachment HTTP 500; bounded diagnostic count 5, candidate count 10, candidate
  `make check`, and candidate race all passed. The initial failure is retained
  here and is not erased by later success.
- Independent read-only review of `6d94ff4..ee02403` returned `SPEC BLOCK`:
  1. `GovernanceRequest.ResponseBody`, `AuditMetadata`, and
     `OutboxDraft.Payload` remain caller-visible but are silently ignored;
  2. the universal forbidden-material guard omits Basic authorization fixtures;
  3. an unknown-policy row is changed to `inflight` before exact event policy
     validation, and authorized replay can change unversioned/unknown-policy
     dead letters to `ready` without envelope/policy validation.
- Authority, canonical hash, opaque preparation, full tuple/current lease,
  empty-only down, scope, and `server/**` boundaries passed independent review.
  Missing negative tests match the three blocks above.
- Per v5, independent `BLOCK` stops closure and cannot be repaired inside r010.
  `PCR-S01B-5` is review-blocked, Release 0 is incomplete, Release 1 is inactive,
  and a new approved plan/task is required before further product edits.

## 2026-08-17 — J023 — v6/r011 approved and PCR-S01B-6 activated

- Human Customer explicitly stated `PRODUCT-CAPABILITY-ROADMAP-001 v6 / r011`
  after receiving the exact raw-input, Basic guard, validate-before-claim/replay,
  immutable-policy, evidence, exclusion, and stop boundaries.
- Approved immutable `plan_v6.md`, froze `PCR-001-S01B6-R11` on base
  `f93eca764bb464245ef096429701aa0a856f0c56`, and retained unchanged CLAUDE and
  Backend policy hashes.
- Activated `PCR-S01B-6` as the sole roadmap step. Release 0 remains incomplete
  and Release 1 remains inactive.
- The repair is limited to the exact Section 6 paths and RED/GREEN order. It may
  not add schema, mutate retained invalid rows, activate a capability, touch
  `server/**`, or include existing UI/local-runtime/code-to-product dirty paths.
- No implementation, verification, independent PASS, push, merge, deployment,
  or Release 0 completion is claimed by this activation record.

## 2026-08-17 — J024 — v7/r012 approved and authority closure activated

- Candidate `4d60e50d9c03a68b2723b427506ea7db64d90d90` passed all v6 product,
  deterministic, race, attachment-concurrency, policy, scope, and `server/**`
  gates. Fresh independent review passed implementation SPEC and code quality.
- That review returned one authority/traceability BLOCK: immutable
  `plan_v6.md` declares nonexistent base
  `f93eca77c3450109b7328441812d63710f179521`, while Git, the v6 task record,
  plan pointer, and J023 use actual base
  `f93eca764bb464245ef096429701aa0a856f0c56`.
- The mismatch is retained as historical evidence; `plan_v6.md` is not amended
  or reinterpreted. Release 0 was not closed under v6.
- The Human Customer explicitly approved `PRODUCT-CAPABILITY-ROADMAP-001 v7 /
  r012` on 2026-08-17. Immutable `plan_v7.md` starts from exact verified
  candidate `4d60e50d9c03a68b2723b427506ea7db64d90d90` and activates
  `PCR-S01B-7` as the sole roadmap step.
- v7/r012 is documentation-only: five exact roadmap paths, read-only evidence,
  fresh independent review, and closure records after PASS. It does not
  authorize product/test/schema/runtime/frontend/Control Plane/`server/**`
  changes, push, merge, deployment, external data, or prior-plan amendment.
- Release 0 remains incomplete at activation and Release 1 remains inactive.

## 2026-08-18 — J025 — v8/r013 approved and single authority activated

- v7 activation `14908b9e53a73330c0cde3fe8a3f602635906858`
  passed exact base/parent/ancestry, immutable-history, five-path scope, policy
  hash, trailer, link, focused-test, dirty-exclusion, and `server/**` checks.
- Independent v7 review passed implementation, quality, scope, and Release 1
  boundaries but returned one authority BLOCK: task entries r011 and r012 were
  both marked active even though the registry header and journal claimed one
  active authority. Release 0 was not closed under v7.
- The Human Customer explicitly approved `PRODUCT-CAPABILITY-ROADMAP-001 v8 /
  r013` on 2026-08-18. Immutable `plan_v8.md` starts from exact v7 activation
  `14908b9e53a73330c0cde3fe8a3f602635906858`.
- r011 and r012 are now `independent-review-blocked`; `PCR-S01B-8`/r013 is the
  sole active roadmap authority. Prior immutable plans remain untouched.
- v8/r013 is documentation-only and limited to five exact roadmap paths,
  read-only verification, fresh independent review, and closure records after
  PASS. It does not authorize product/test/schema/runtime/frontend/Control
  Plane/`server/**` changes, push, merge, deployment, external data, or
  prior-plan amendment.
- Release 0 remains incomplete at activation. Release 1 remains inactive and no
  Release 1 task is active.

## 2026-08-18 — J026 — v8 independent PASS and Release 0 closed

- Fixed activation candidate `872b5ebe35e1d12a488e389a713d377a4c1663cc`
  has exact parent `14908b9e53a73330c0cde3fe8a3f602635906858`.
  The range contains one commit and exactly the five paths authorized by
  `plan_v8.md` Section 6.
- Deterministic v8 evidence passed:
  - exact base resolution, direct parent, ancestry, and r013 trailers;
  - SHA256 for `CLAUDE.md`, `backend/AGENTS.md`, and immutable `plan_v8.md`;
  - zero diff in `plan_v1.md` through `plan_v7.md`;
  - mechanical task enumeration with exactly one pre-closure active entry,
    r013, while r011 and r012 were review-blocked;
  - roadmap relative links, diff checks, exact five-document scope, preserved
    unrelated dirty paths, and empty tracked/staged/untracked `server/**` scope;
  - Workspace governance contract, application, SQLite repository, Workspace
    composition, and Bootstrap focused packages, count 1.
- Fresh independent read-only review returned `SPEC PASS`. It confirmed the
  verified product candidate `4d60e50d9c03a68b2723b427506ea7db64d90d90`
  remains unchanged and retained all implementation/code-quality PASS evidence.
  It also independently passed v8 authority uniqueness, immutable history,
  exact scope, traceability, Release 0 readiness, and Release 1 inactivity.
- Non-blocking record debt remains: the historical r011/r012 `Independent
  reviewer` summary fields still use their original required-review wording,
  while their status and appended outcome sections accurately record the
  review blocks. This does not grant active authority and is not changed during
  closure.
- `PCR-001-S01B8-R13` and Release 0 — Authority and safety foundation are
  complete. Mechanical post-closure active-task count is zero. Release 1
  remains inactive; no Release 1 task, product change, push, merge, or
  deployment is authorized by this closure.

## 2026-08-18 — J027 — v9/r014 approved and PCR-S02A activated

- The current repository proves Release 0 complete at exact base
  `628996378af6fbe12c27a916a624a5f5374a884f`; pre-activation active-task count
  is zero and no Release 1 implementation commit exists.
- Read-only discovery converged after two rounds with no remaining retrievable
  blocker. Release 1 order is S02A, S02B, S03A, S03B, then S04; S02A is the
  only dependency-valid next story.
- Current evidence shows a composed Todo local/gRPC use case and SQLite
  repository, but no installed `/api/tasks` HTTP handler, no Task response
  schemas, no revision/archive/restore contract, and only a minimal shared view.
  Internal presence is not treated as installed product acceptance.
- The Human Customer directed continued approved actions through completion of
  Release 1 on 2026-08-18. Immutable `plan_v9.md` narrows that authority to
  `PCR-S02A` only and freezes r014 from exact base `6289963`.
- v9 uses RED/GREEN TDD and requires the complete Task lifecycle, revision
  conflicts, governance atomicity, HTTP/runtime installation, schema-parsed
  shared client, shared Web/Desktop view, restart/browser evidence, full checks,
  and fresh independent PASS.
- Existing dirty/untracked UI, local-runtime, code-to-product, and generated UI
  artifacts are excluded. `server/**` remains permanently read-only.
- No S02A implementation, verification PASS, independent review, push, merge,
  deployment, S02B activation, or Release 1 completion is claimed here.

## 2026-08-18 — J028 — v10/r015 verification repair activated

- Exact candidate `9065802` contains the complete S02A product slice and passed
  backend `make check`, real `make test-race`, Core 593/593, Task Views 8/8,
  Core/Views/Web/Desktop typechecks, clean installed Playwright 1/1, and fresh
  independent review.
- S02A closure is not claimed because frozen root `pnpm typecheck` and
  `pnpm test` reference nonexistent `@multica/mobile`; direct Views full tests
  additionally expose three dead locale plural keys and one Windows path
  separator mismatch in an already-documented exception.
- The Human Customer approved follow-up actions on 2026-08-18. Immutable
  `plan_v10.md` freezes exact base `9065802` and authorizes only those
  verification-baseline repairs.
- r014 becomes verification-blocked; r015 is the sole active task. S02B and all
  later Release 1 stories remain inactive.
- No repair implementation, deterministic PASS, S02A closure, push, merge,
  deployment, or Release 1 completion is claimed here.

## 2026-08-18 — J029 — v11/r016 Desktop verification repair activated

- v10 candidate `2aac82e` passes its focused Views gate 131/131 and root
  typecheck across the current workspace graph.
- Root test now reaches Desktop and exposes two new deterministic blockers:
  Vitest cannot evaluate the retained package-script shebang after Vite
  transformation, and the ignored Electron package lacks `path.txt/dist`.
- All package script call sites explicitly use `node scripts/package.mjs`.
  Electron 39.8.7's exact Windows x64 archive already exists in the local
  cache; no network fallback is authorized.
- The Human Customer approved continued follow-up actions through Release 1.
  Immutable v11 freezes exact base `2aac82e`, marks r015
  verification-blocked, and establishes r016 as the sole active task.
- S02B and later stories remain inactive. No repair PASS, S02A closure, push,
  merge, deployment, or Release 1 completion is claimed here.

## 2026-08-18 — J030 — v12/r017 root parallel-test repair activated

- v11 package-module and cache-only Electron repairs make the focused Desktop
  tests pass 30/30 and 4/4.
- Root test executes Desktop but three `deriveVersion` cases that create real
  temporary Git repositories exceed Vitest's default five seconds under
  full-workspace parallel load. The same file passes 30/30 in isolation.
- Continued Customer authority activates immutable v12 from exact `3c030b4`.
  Only explicit 15-second timeouts on those three tests are allowed; assertions,
  production behavior, retry count, and all other tests remain unchanged.
- r016 is verification-blocked; r017 is the sole active task. S02B and later
  stories remain inactive, and no r017 PASS or S02A closure is claimed.

## 2026-08-18 — J031 — v13/r018 real-Git timeout correction activated

- v12 root execution passes 422/423 Desktop tests. The remaining multi-step
  real-Git tag/commit/describe case exceeds 15 seconds only under the full
  workspace parallel graph; isolated package tests remain 30/30.
- Continued Customer authority activates immutable v13 from exact `8f94d69`.
  Only the same three timeout literals may move from 15 to 60 seconds; no
  assertion, retry, skip, mock, production code, or concurrency change is
  permitted.
- r017 is verification-blocked and r018 is the sole active task. S02B and later
  stories remain inactive; no r018 PASS or S02A closure is claimed.

## 2026-08-18 — J032 — v13/r018 and PCR-S02A closed

- Candidate `bc1eed7f1ccb92a0bd31a769403c8f7e5139cf21` changes only the three
  authorized real-Git timeout literals from 15 to 60 seconds after r018
  activation. Assertions, commands, concurrency, retries, production behavior,
  manifests, and lockfiles are unchanged.
- Exact-candidate root `pnpm typecheck` passes 6/6 Turbo tasks. Exact detached
  candidate root `pnpm test` passes 5/5 tasks in 3m35s; clean Views passes 165
  files and 1671 tests. The isolated Desktop package file passes 30/30.
- Backend `make check` and real `make test-race` pass; race uses MinGW GCC
  15.2.0 without an environment waiver.
- A detached `bc1eed7` runtime with a new migrated SQLite database and canonical
  fixture passes installed Chrome Playwright `e2e/tasks.spec.ts` 1/1, covering
  the Task lifecycle, filtering, reorder, archive, restore, and readback.
- Scope verification passes: tracked/staged `server/**` diff is empty; excluded
  shared Input remains blob `a830fd2f0f82770563908d512558fe6ba48f50dd`;
  candidate-only Next configuration drift is not staged or copied back.
- Candidate Web and backend process ancestry was resolved before cleanup; the
  exact candidate processes were stopped and ports 3000, 8080, and 9080 are
  closed.
- Fresh independent read-only review returned PASS. It confirmed v10-v13
  authority consistency, exact repair scope, safe explicit-Node shebang
  removal, unchanged real-Git assertions/retries/concurrency, green evidence,
  and permission to close S02A.
- r018 and PCR-S02A are `complete-independent-reviewed`. Release 1 remains
  active with no active task. S02B's dependency is satisfied but it remains
  inactive until a new versioned plan is activated.

## 2026-08-18 — J033 — v14/r019 approved and PCR-S02B activated

- Exact base `3262232c5000ed449b89d98901535cd58b42a48d` records S02A as
  complete-independent-reviewed with zero active tasks before activation.
- Three bounded read-only discovery rounds reached dependency closure with no
  retrievable blocker or human-owned decision. The frozen route is
  `POST /api/tasks/{id}/promote` with revision and idempotency contracts.
- Existing evidence has a mutable Task `issue_id`, independent Issue creation
  transaction, Issue number allocator, and reusable governance transaction, but
  no promotion service, immutable promotion link, HTTP route, client mutation,
  shared UI affordance, or installed acceptance.
- The Human Customer's standing direction to complete Release 1 activates
  immutable v14/r019 for PCR-S02B only. S03A and later stories remain inactive.
- v14 freezes an additive 000012 migration, one governed `BEGIN IMMEDIATE`
  promotion transaction, trusted authorization, snapshot copy, optional atomic
  completion, strict client schema, shared Web/Desktop UI, restart/browser
  evidence, and fresh independent review.
- Existing dirty UI, generated protobuf, local-runtime, code-to-product, and UI
  artifacts are excluded. `server/**` remains permanently read-only.
- No S02B implementation, verification PASS, independent review, push, merge,
  deployment, S03A activation, or Release 1 completion is claimed here.

## 2026-08-18 — J034 — v15/r020 S02B review remediation activated

- v14 candidate `36b18b4afac2ca2ae65ced7ee329c726c0bed73b` passed exact-candidate
  typecheck, full tests, backend check/race, and installed Chrome promotion
  acceptance. A transient unrelated navigation-test timeout was reproduced as
  resource contention: focused 2/2 passed and a zero-cache full rerun with
  concurrency 2 passed 5/5.
- Fresh independent review returned BLOCK. Exact replay reloads mutable Task
  and Issue rows after commit, Core callers cannot supply/reuse one
  idempotency key, and the nested promotion Issue schema inherits loose unknown
  field handling.
- Continued Customer authority activates immutable v15/r020 from exact blocked
  candidate `36b18b4`. Only response-snapshot persistence, same-command key
  reuse, strict nested Issue parsing, their tests, and repeated verification
  are authorized.
- r019 is `independent-review-blocked`; r020 is the sole active task. S02B is
  not complete, S03A and later stories remain inactive, and no push, merge,
  deployment, or Release 1 completion is claimed.

## 2026-08-18 — J035 — v16/r021 deterministic-gate remediation activated

- v15/r020 candidate `e97c92b23423a89d6d731ab8ba40ca9a80858b63`
  passes focused backend/Core/Views checks, forced typecheck 6/6 with zero
  cache, and forced root tests 5/5 with zero cache; Views passes 165 files and
  1672 tests after verification TEMP is placed on F because C had insufficient
  transient capacity.
- Backend `make check` stops on the existing 12-writer attachment acceptance.
  The focused test also fails on pre-r020 `36b18b4`. Temporary candidate-only
  diagnostics expose `context deadline exceeded` while waiting for the
  attachment write slot or committing; the diagnostic edit was removed and
  the detached candidate tracked diff returned empty.
- Local Kratos v3 source confirms `NewServer` defaults to a one-second request
  timeout. That transport deadline is shorter than the attachment repository's
  frozen eight-second acquisition budget and cancels otherwise serialized
  atomic writes.
- Continued Customer authority activates immutable v16/r021 from exact
  `e97c92b`. Only an explicit 30-second Canonical Runtime HTTP timeout, its
  transport contract test, and repeated S02B verification are authorized.
- r020 is `verification-blocked`; r021 is the sole active task. No backend
  gate PASS, S02B closure, S03A activation, push, merge, deployment, or Release
  1 completion is claimed.

## 2026-08-18 — J036 — v16/r021 and PCR-S02B closed

- Exact candidate `5ea1a47f7f3395c4778398ee1a6e79f6880bf0a0`
  changes only the approved r020 product remediation, v15/v16 governance, and
  the r021 Canonical Runtime timeout plus its contract test. `server/**` range
  and worktree diffs remain empty.
- RED records an actual one-second installed HTTP request deadline and the
  unchanged 12-writer attachment failure on both `36b18b4` and `e97c92b`.
  GREEN records an explicit 30-second deadline and the same 12-writer test
  passing without retries, skipped assertions, reduced concurrency, or
  attachment production changes.
- r020 focused backend replay/migration/runtime tests pass. Core passes 2 files
  and 7 tests; Task Views passes 1 file and 9 tests. Forced root typecheck
  passes 6/6 with zero cache; forced root tests pass 5/5 with zero cache and
  Views passes 165 files/1672 tests.
- Backend `make check` passes. Real `make test-race` passes after
  `go clean -testcache`, using MinGW GCC 15.2.0.
- A new r021 SQLite database and canonical fixture pass installed Chrome Task
  promotion acceptance 1/1 in 16.2 seconds. The initial browser run reached
  the known cold-route compile window before the five-second heading check;
  the route was then verified 200 with Tasks content and the explicit rerun
  passed under Playwright retries=0.
- Policy hashes are CLAUDE
  `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646`,
  backend AGENTS
  `fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209`,
  and v16
  `3575657bbe68972b25691d2b9f077038b802a39e5ad730e07708cd215af63e5b`.
  Excluded Input remains blob
  `a830fd2f0f82770563908d512558fe6ba48f50dd`.
- Candidate-only `.local-runtime/**` and `node_modules.main-link-r019/**` are
  recorded untracked runtime/dependency artifacts. Candidate backend/Web
  processes were resolved and stopped; ports 3000/8080/9080 are closed.
- Fresh independent read-only review returned PASS for authority, scope, all
  three original r019 blockers, v16 cancellation/timeout behavior, and supplied
  deterministic/browser evidence. It recommends closing r021/r020/S02B.
- r021 is `complete-independent-reviewed`; r020 is closed through r021; S02B is
  complete. Release 1 remains active with no active task. S03A and later stories
  remain inactive pending a new versioned plan; no push, merge, deployment, or
  Release 1 completion is claimed.

## 2026-08-18 — J037 — v17/r022 Issue search activated

- Exact base `2439e9c2edd3c557e849fb695210b2eab95bbb9d` records S02B as
  complete-independent-reviewed and leaves zero active tasks.
- Three bounded context-discovery rounds reached dependency closure. The shared
  Search UI and cancellable Core client already target `GET /api/issues/search`,
  but Canonical HTTP currently filters `ListIssues` in memory and has no search
  service, ranked repository query, projection, or installed feature evidence.
- Existing Issue title/description writes occur through Create, Update,
  hierarchy batch update, and Task promotion. The additive 000013 projection
  will backfill retained data, synchronize all writes, and rebuild with one
  Unicode NFKC/case-fold/punctuation normalizer at startup. The current
  modernc SQLite build exposes FTS5 and deterministic scalar functions; no
  external network dependency is required.
- `issue_search` and `project_search` currently share one permission-derived
  flag. v17 separates per-feature installation evidence from the shared read
  authorization so S03A can enable only Issue search and cannot claim S03B.
- The Human Customer's standing direction to complete Release 1 activates
  immutable v17/r022 for PCR-S03A only. S03B and S04 remain inactive.
- Existing dirty generated protobuf, shared Input/Issue/modal files,
  `.local-runtime/**`, `docs/code-to-product/**`, the untracked auth test, and
  `ui/**` remain excluded. `server/**` remains permanently read-only.
- No S03A implementation, verification PASS, independent review, closure,
  push, merge, deployment, S03B activation, or Release 1 completion is claimed.

## 2026-08-18 — J038 — v17/r022 and PCR-S03A closed

- Exact candidate `7479e07adcc1c703a52cb348af7919df2cc68553` installs ranked
  repository-backed `GET /api/issues/search`, deterministic Unicode NFKC/
  case-fold/punctuation normalization, identifier/number matching, stable
  pagination, closed filtering, cancellation, authorization, Workspace
  isolation, projection migration `000013`, synchronization triggers, and
  startup rebuild. Runtime reports only `issue_search=true`;
  `project_search=false`.
- RED/GREEN evidence includes application/HTTP/repository/runtime/Core/Views
  contracts. Browser discovery exposed and fixed a missing fixture-process UDF
  registration, obsolete localStorage-only authentication, locale-dependent
  selectors, and the actual `Search...` accessible name. Each failure was
  retained as evidence and rerun after a bounded fix.
- The 10,000-Issue non-race gate measured p50/p95 at
  20.9999/23.1776ms and 25.3918/39.5394ms, below the frozen 100/250ms limits.
  The first official race attempt incorrectly asserted production latency under
  instrumentation; the final split keeps 10,000-row functional coverage under
  race and measures latency only without race. Direct Windows `go test -race`
  later hit known `0xc0000139` and was not counted as PASS; the official MinGW
  wrapper passed.
- The initial full root run had two unrelated five-second Views timeouts; both
  focused files passed, and the controlled-concurrency full run passed. On the
  final candidate, forced root typecheck passes 6/6 with zero cache and forced
  root tests pass 5/5 with zero cache; Views passes 166 files/1675 tests.
  Exact-candidate backend `make check` and official clean-cache
  `make test-race` pass with MinGW GCC 15.2.0.
- Production Web build passes. A fresh migrated database and fixture pass
  installed Chrome `e2e/issue-search.spec.ts` 1/1 in 4.1 seconds with retries
  disabled, covering English, Chinese, identifier/number, default and explicit
  closed state, pagination, cross-Workspace isolation, and shared UI results.
- Fresh independent review first returned BLOCK because config-unloaded UI
  could call the inactive Project-search vertical. RED reproduced one Project
  call; `7479e07` requires loaded explicit `project_search` evidence and adds a
  passing regression while retaining Issue search. Repeated focused/full/
  build/browser gates pass, and independent re-review returns PASS.
- Scope and cleanup pass: candidate `server/**` count is zero; CLAUDE,
  backend AGENTS, and v17 SHA-256 values match the frozen policy bundle; the
  protected Input blob remains `a830fd2f0f82770563908d512558fe6ba48f50dd`;
  candidate tracked state is clean except recorded runtime/dependency artifacts;
  exact backend/Web processes were stopped and ports 3000/8080/9080 are closed.
- The interrupted-install `r022-59d314a` Git worktree registration was removed;
  Windows left an unregistered residual directory and execution policy refused
  recursive deletion. It contains no candidate source authority or running
  process and is not treated as a product PASS.
- r022 and PCR-S03A are `complete-independent-reviewed`. Release 1 remains
  active with no active task. S03B and S04 remain inactive until a new approved
  plan; no push, merge, deployment, later-story activation, or Release 1
  completion is claimed.

## 2026-08-18 — J039 — v18/r023 Project search activated

- Exact base `781471015ec8d759cd1209fd051e59fa91507eef` closes S03A with
  independent review and leaves zero active tasks before this activation.
- Bounded context discovery reached dependency closure with zero new
  dependencies. Canonical Project HTTP CRUD exposes the public snake-case
  `title` shape, while older local/Proto search exposes an internal `name`
  model and performs full in-memory filtering. The installed client already
  targets `GET /api/projects/search`; no Canonical HTTP route exists.
- v18 freezes the installed Project surface contract as strict
  `match_source=title|description`, reuses S03A Unicode NFKC/case-fold/
  punctuation normalization, and adds derived migration `000014` with startup
  rebuild and trigger synchronization for every write path.
- The frozen target is 2,000 Projects with p50 no greater than 100ms and p95 no
  greater than 250ms. Search requires `workspace.search.readable` and explicit
  per-feature installation evidence, not the older Proto-only permission.
- The Human Customer's standing direction to complete Release 1 activates
  immutable v18/r023 for PCR-S03B only. S04 remains inactive.
- Existing dirty generated protobuf, shared Input/Issue/modal files,
  `.local-runtime/**`, `docs/code-to-product/**`, the untracked auth test, and
  `ui/**` remain excluded. Protected Input remains blob
  `a830fd2f0f82770563908d512558fe6ba48f50dd`; `server/**` remains permanently
  read-only and currently has empty range/worktree diffs.
- No S03B implementation, verification PASS, independent review, closure,
  push, merge, deployment, S04 activation, or Release 1 completion is claimed.

## 2026-08-18 — J040 — v18/r023 and PCR-S03B closed

- Exact candidate `c9e905bc7675d253991c0a816bcd19985a49c10b` installs ranked,
  repository-backed `GET /api/projects/search` using the public snake-case
  Project surface, strict `title|description` match sources, shared Unicode
  normalization, stable pagination, closed filtering, cancellation,
  authorization, Workspace isolation, projection migration `000014`, trigger
  synchronization, and startup rebuild. Runtime retains `issue_search=true`,
  enables `project_search=true`, and keeps `pin_reorder=false`.
- The 2,000-Project non-race gate measured p50/p95 at 3.0314/4.1305ms, below
  the frozen 100/250ms limits. Exact-candidate backend `make check`, official
  clean-cache `make test-race` with MinGW GCC 15.2.0, root typecheck 6/6,
  source-equivalent root tests 5/5, and production Web build pass.
- A fresh migrated database and canonical fixture pass installed Chrome
  `e2e/project-search.spec.ts` 1/1 with retries disabled. The journey covers
  English/Chinese queries, ranking, closed state, pagination, Workspace
  isolation, shared UI, and installed DELETE-trigger projection disappearance.
- Independent review first returned CODE QUALITY BLOCK for incomplete commit
  trailers and missing direct authentication/trusted-identity and installed
  deletion coverage. Four unpublished commit messages were rewritten without
  changing final tree `b33a24c8fc5818cc1ddec40f9879ea62af33fac7`;
  all five r023 commits now carry the required governance trailers. Added HTTP
  and E2E tests pass. Fresh re-review returns SPEC PASS and CODE QUALITY PASS.
- Scope and cleanup pass: `server/**` candidate diff is empty; CLAUDE,
  backend AGENTS, and v18 hashes match the frozen policy bundle; protected
  Input remains blob `a830fd2f0f82770563908d512558fe6ba48f50dd`;
  candidate tracked state is clean apart from recorded environment artifacts;
  exact backend/Web processes were stopped and ports 3000/8080/9080 are closed.
- r023 and PCR-S03B are `complete-independent-reviewed`. Release 1 remains
  active with no active task. S04 remains inactive until a new approved plan;
  no push, merge, deployment, S04 activation, or Release 1 completion is
  claimed.

## 2026-08-18 — J041 — v19/r024 Pin reorder activated

- Exact base `b86447909e1a7614c539769514008e010b478140` closes S03B with
  independent review and leaves zero active tasks before this activation.
- Bounded context discovery reached dependency closure with zero new
  dependencies or unresolved human choices. Shared Core/Views already declare
  and invoke `PUT /api/pins/reorder`; the Canonical backend has list/create/
  delete Pins but no reorder route. Read-only legacy evidence performs
  unchecked position updates and is not safe authority to copy.
- v19 freezes a strict complete ID order plus `expected_revision`, a per-user/
  Workspace monotonic `order_revision`, contiguous server-derived positions,
  one `BEGIN IMMEDIATE` transaction, exact-set ownership validation, canonical
  409 conflict response, scoped optimistic rollback/refetch, and loaded
  explicit sidebar drag gating.
- Migration `000015` may add only Pin-order revision metadata with existing-set
  backfill and guarded rollback. It may not add a foreign key, cascade, or
  rewrite/drop source Pins or positions.
- The Human Customer's standing direction to complete Release 1 activates
  immutable v19/r024 for PCR-S04 only. Release 2 remains inactive.
- Existing dirty generated protobuf, shared Input/Issue/modal files,
  `.local-runtime/**`, `docs/code-to-product/**`, the untracked auth test, and
  `ui/**` remain excluded. Protected Input remains blob
  `a830fd2f0f82770563908d512558fe6ba48f50dd`; `server/**` remains permanently
  read-only and has empty range/worktree diffs.
- No S04 implementation, verification PASS, independent review, closure,
  push, merge, deployment, or Release 1 completion is claimed.

## 2026-08-18 — J042 — v19/r024, PCR-S04, and Release 1 closed

- Exact candidate `b4fa3a3c60d24953c9f37c82cf0d70cda0fd7b11` installs strict,
  revisioned `PUT /api/pins/reorder` for the authenticated user's complete Pin
  set in the trusted Workspace. One `BEGIN IMMEDIATE` transaction validates
  revision, exact membership, ownership, and Workspace before writing all
  contiguous positions and advancing the collection revision once.
- Migration `000015` backfills collection revisions and advances them for Pin
  create/effective delete; its guarded down migration cannot discard an
  advanced concurrency token. No foreign key, cascade, source Pin rewrite, or
  `server/**` change is present.
- RED/GREEN coverage spans application, strict HTTP/auth/identity/CSRF/error
  mapping, repository atomicity/cancellation/forced-failure rollback,
  migration/restart, installed runtime config, strict Core schema/request and
  Workspace/user-scoped optimistic recovery, and loaded explicit UI gating.
- Exact-tree backend `make check` and clean-cache official `make test-race`
  pass with MinGW GCC 15.2.0. Root typecheck passes 6/6, root tests pass 5/5,
  and production Web build passes. Root `make check` also exited zero but
  warned that Docker Desktop was unavailable; it is not treated as the
  complete acceptance gate.
- A fresh migrated SQLite database and canonical fixture pass installed Chrome
  `e2e/pin-reorder.spec.ts` 1/1 with retries disabled. It proves stale 409
  rollback/refetch, successful 204 reorder, exact persisted IDs/positions/
  revision, and reload persistence. Two earlier runs correctly failed on test-
  driver selector/native-drag gaps and were fixed before the fresh-db PASS.
- Incorrect policy trailers on two unpublished commits were rewritten to the
  activation bundle. Old and new candidates share exact tree
  `41525376c8b8563eccaff7ad05931ec78e6cc85d`; final r024 commits carry correct
  CLAUDE, backend AGENTS, and v19 hashes.
- Scope and cleanup pass: candidate/worktree `server/**` diffs are empty;
  protected Input remains blob
  `a830fd2f0f82770563908d512558fe6ba48f50dd`; recorded runtime/dependency
  artifacts remain untracked; exact backend/Web processes were stopped and
  ports 3000/8080/9080 are closed.
- Fresh independent review returns SPEC PASS and CODE QUALITY PASS with no
  concrete blocker. r024 and PCR-S04 are `complete-independent-reviewed`;
  Release 1 is complete with no active task. Release 2 remains inactive, and
  no push, merge, or deployment is claimed.

## 2026-08-18 — J043 — v20/r025 activates PCR-S05A

- The Human Customer approved continued execution through Release 2. Current
  exact base `0aed36871271184325b6147841348847b004a7a4` closes Release 1
  with independent review and has no active roadmap task.
- Three bounded read-only discovery rounds reached dependency closure: final
  `new_dependencies_count=0`, remaining retrievable blockers `0`, no blocking
  unknown/contradiction, and no unresolved human-owned decision. The frozen
  story order selects S05A before S05B/S06A/S06B.
- v20/r025 activates only the System-owned Skill catalog and immutable version
  lifecycle. S05B remains the sole owner of Skill file bodies, archive import,
  Space assets, manifests, preview, and download. Both Knowledge stories remain
  inactive.
- Policy bundle is
  `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646/fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209/a55f0f0f5e593d39f779fb3d67a556dfa5eb0157b49faa4562badbdb5af1ad85`.
  Protected Input remains blob
  `a830fd2f0f82770563908d512558fe6ba48f50dd`; generated protobuf worktree
  blobs equal the index; recorded UI/runtime/dependency paths remain excluded.
- Activation `server/**` range and worktree diffs are empty. No push, merge, or
  deployment is authorized or claimed.

## 2026-08-18 — J044 — v20/r025 S05A implementation candidate

- Installed the System-owned Skill definition/version catalog, guarded SQLite
  migration, immutable audit evidence, trusted HTTP lifecycle, Workspace-owned
  visibility references, and explicit exact-version Agent binding boundary.
  S05B file bodies/import remain explicitly unavailable.
- RED/GREEN coverage now proves draft creation, immutable numbered versions,
  publish/deprecate/archive/restore, exact revision conflicts, deterministic
  concurrent mutation serialization, audit rollback, binding compensation,
  restart persistence, cross-Workspace isolation, authentication, Cookie CSRF,
  owner/admin mutations, member published reads, and explicit Agent binding.
- Core uses strict Skill schemas, Workspace/Skill/version query keys, exact
  lifecycle routes, and fail-closed malformed responses. Shared Skills UI
  requires loaded `skill_administration=true`; file controls remain gated by
  the still-false `skill_import` flag, and creator-based mutation permission is
  removed.
- Focused backend/Core/Views tests, Core/Views typecheck and lint, and backend
  `make check` pass. The parallel root test run passed Core/Web/Desktop/Docs but
  timed out three unrelated Views tests; all three pass together on immediate
  focused rerun, so the root aggregate is recorded as partial rather than PASS.
- Protected Input remains blob
  `a830fd2f0f82770563908d512558fe6ba48f50dd`; `server/**` remains unchanged.
  Browser, production build, official race, independent review, and S05A
  closure are still pending.

## 2026-08-18 — J045 — v20/r025 and PCR-S05A closed

- Exact implementation candidate `17f7eb1f2db22bd64e426614b84b945325b1f90f`
  (tree `57361bc6344e2801c4393414cfcc4bdb6d6c9897`) completes the
  System-owned Skill catalog and immutable numbered-version lifecycle with
  Workspace visibility, exact referenced reads, provenance, audit, strict Core
  parsing, shared Web/Desktop UI, and only `skill_administration=true`.
- The first independent review blocked closure on non-atomic binding, archived
  member-list leakage, missing audit/provenance read surface, visible import UI,
  and silently ignored file fields. Remediation closed all five. A second review
  found retained Workspace bindings absent from the down guard and missing
  cancellation evidence; the final candidate closes both with binding-aware
  rollback protection and explicit cancel/deadline zero-partial-row tests.
- Exact final-candidate backend `make check` and official `make test-race` pass
  with MinGW GCC 15.2.0. One preceding full check exposed a test-only SQLite
  `:memory:` multi-connection fixture; replacing it with a `t.TempDir()` file
  database made the same assertions deterministic before the PASS.
- Root typecheck 6/6, serialized package tests (Core 613, Docs 17, Views 1677,
  Web 149, Desktop 423), and production Web build pass on the preceding tree;
  final remediation is backend-test/migration-only and changes no frontend blob.
  The build retains two pre-existing CSS `::highlight` optimizer warnings.
- Fresh migrated SQLite plus canonical fixture passes installed Chrome S05A
  lifecycle acceptance 1/1 with retries disabled. The initial production-proxy
  run exposed omitted `REMOTE_API_URL`; explicit backend 8081 wiring then passed.
  Visual inspection confirms origin Workspace, audit activity, v2 published,
  archive/restore, and no file/import controls. Isolated processes were stopped.
- Fresh final independent review returns SPEC PASS and CODE QUALITY PASS with no
  blocker. `server/**` candidate/worktree diffs are empty, protected Input stays
  blob `a830fd2f0f82770563908d512558fe6ba48f50dd`, eight trailers parse, and all
  recorded dirty paths remain excluded. PCR-S05A is
  `complete-independent-reviewed`; Release 2 remains active with no active task.
  S05B needs a new frozen plan. No push, merge, or deployment is claimed.

## 2026-08-18 — J046 — v21/r026 activates PCR-S05B

- Exact base `27def068f07cb56ef7e58471fbfacb397b11b639` closes PCR-S05A
  with final independent SPEC PASS and CODE QUALITY PASS and leaves no active
  task before this activation.
- Three bounded read-only discovery rounds reached dependency closure. Frozen
  contracts require archive preview/import, exact adversarial limits, immutable
  version-scoped manifests, Space-owned bytes, checksum/download, and safe
  cleanup. Shared Core/Views retain a URL import and file editor shape while the
  Canonical backend deliberately rejects S05B. Read-only legacy evidence offers
  URL/archive behavior but is not safe authority. The generated Space
  AssetService remains an unimplemented scaffold; existing attachment object
  mechanics are reusable only through a new narrow hand-owned contract.
- Immutable v21 freezes recognized-provider HTTPS URL adapters plus multipart
  archive input, one preview/commit validator, opaque tokens, idempotency,
  default new-version conflict, non-mutating explicit replacement, canonical
  paths, exact limits, Space quarantine/promotion/reconciliation, System logical
  manifests, strict authorization, and version-aware shared UI behavior.
- Policy bundle is
  `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646/fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209/e7db2df5d92345dfe0e988af3f1c1f353ddd519bda8bf07bfbfcf6a472f9544e`.
  Protected Input worktree blob stays
  `a830fd2f0f82770563908d512558fe6ba48f50dd`; generated protobuf and recorded
  dirty paths remain excluded. `server/**` diffs are empty.
- Only PCR-S05B is active. Knowledge, push, merge, deployment, and Release 2
  completion remain inactive; no implementation or verification PASS is claimed.

## 2026-08-18 — J047 — v21/r026 and PCR-S05B closed

- Exact candidate `abce08d652c1af263a4e678a31732814d324330d`
  (tree `639d1bdfaf843c08c3a3e3df15c5d66140facc47`) installs safe
  archive/recognized-URL preview/import, immutable version-scoped manifests,
  Space-owned object quarantine/promotion/reconciliation, checksummed download,
  strict clients, and shared Web/Desktop file management.
- Independent review first blocked governance trailers, missing `SKILL.md`
  enforcement, invalid UTF-8 paths, disguised binary/container acceptance, and
  incorrect Agent historical-file filtering. Remediation closed all five. A
  second review found metadata-only versions and disguised ar bodies still open;
  the final candidate requires and transactionally copies `SKILL.md` manifests
  and rejects ar magic. Final review returns SPEC PASS and
  CODE/SECURITY/QUALITY PASS with no blocker.
- Backend `make check` and official `make test-race` pass with MinGW GCC 15.2.0.
  Root typecheck and focused Core/Views tests pass. Root parallel tests retain
  only two unrelated Team Control 5-second load timeouts; the exact file passes
  10/10 in isolation. Production Web build passes with two existing CSS
  optimizer warnings.
- Fresh SQLite plus installed Chrome passes lifecycle, archive preview/import,
  file read, checksum-backed download, archive/restore, and mandatory-manifest
  ordering 1/1 with retries disabled after explicit build-time backend proxy
  wiring. The initial missing-proxy 404 was retained as configuration evidence;
  all isolated processes were stopped after PASS.
- Immutable v21 and policy hashes remain exact. Candidate/worktree `server/**`
  diffs are empty, protected Input remains
  `a830fd2f0f82770563908d512558fe6ba48f50dd`, trailers parse, and recorded dirty
  paths remain excluded. PCR-S05B is `complete-independent-reviewed`; Release 2
  remains active with no active task. S06A needs its own frozen plan. No push,
  merge, or deployment is claimed.

## 2026-08-18 — J048 — v22/r027 activates PCR-S06A

- Exact base `78c5b5eb8b04d27d1fdc3c524955dd02fd756348` closes PCR-S05B
  with final independent SPEC and code/security/quality PASS and leaves no
  active task before this activation.
- Three bounded read-only discovery rounds reached dependency closure with
  `new_dependencies=0`. The frozen story requires status/kind/source/
  applicability/revision filters, stable pages, source projection, superseded
  reads, Workspace isolation, and quarantine visibility. Canonical currently
  has only an uninstalled provenance-free scaffold; existing shared clients are
  permissive and expose future S06B controls; legacy `server/**` search is
  published-only offset evidence and remains read-only.
- Immutable v22 freezes additive governed revision/source storage, trusted
  Workspace identity, member versus owner/admin visibility, selected-revision
  source filtering, deterministic portable rank, filter-bound signed keyset
  cursors, strict Core contracts, and loaded query-only shared UI.
- Policy bundle is
  `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646/fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209/11474d4c1bdf84f0fc574ef40f913425be35b6dad8810ce382ab88df959a6edc`.
  Protected Input stays `a830fd2f0f82770563908d512558fe6ba48f50dd`;
  generated protobuf and every recorded dirty path remain excluded.
- Only PCR-S06A is active. S06B mutations/realtime, `knowledge_review`, push,
  merge, deployment, and Release 2 completion remain inactive. `server/**`
  range/worktree diffs are empty; no implementation or PASS is claimed.

## 2026-08-18 — J049 — v22/r027 S06A implementation candidate

- Added additive guarded Workspace migrations for governed Knowledge entries,
  immutable numbered revisions, and ordered source references without promoting
  the provenance-free legacy scaffold or adding foreign keys.
- Installed authenticated list/detail routes with capability authorization,
  role-bounded quarantine, strict status/kind/source/applicability/revision
  validation, portable Unicode-normalized rank buckets, filter-bound signed
  expiring keyset cursors, selected-revision source projection, cancellation,
  and non-disclosing detail behavior.
- Core now strictly rejects malformed list/detail success responses, carries
  every filter in typed request/query keys, and removes empty fallbacks. Shared
  Web/Desktop mounts only after loaded `knowledge_query=true`, performs source
  deep links server-side, shows provenance/revision/status/pagination states,
  and no longer mounts any S06B proposal or review request/control.
- Focused application, persistence, Runtime HTTP, schema, and Views tests pass.
  Backend `make check` passes. Root typecheck passes 6/6. Root parallel tests
  pass Core 615, Web 149, Desktop 423, and Docs 17; Views has one unrelated
  Team Control five-second load timeout, while that exact file plus Knowledge
  passes 13/13 immediately in isolation. The aggregate is not recorded as PASS.
- Official race, production build, installed-browser acceptance, exact scope,
  candidate commit, and fresh independent review remain pending. Protected
  Input and recorded dirty paths remain excluded; `server/**` remains unchanged.

## 2026-08-18 — J050 — v22/r027 and PCR-S06A closed

- Exact candidate `3d465f1110fed1ce9bef8d076f77d4b553fda421`
  (tree `c54f33d48f660f6b12fbfb70dbeea9f8d4826e5e`) installs strict
  authenticated Knowledge list/detail, deterministic source-explained ranking,
  filter/Workspace-bound signed cursors, immutable revision provenance,
  superseded reads, and owner/admin-only explicit quarantine reads.
- Independent review initially blocked explicit empty filters and an injected
  feature provider that could reopen `knowledge_review`. Remediation rejects
  explicit empty/repeated numeric filters and empty status/kind filters, binds
  cursors to Workspace, and forces review permissions/feature false under v22.
  It also strengthens down migration to reject retained rows in any of the
  three new tables. Fresh review returns SPEC PASS and
  CODE/SECURITY/QUALITY PASS with no blocker.
- Exact candidate backend `make check` and official `make test-race` pass with
  MinGW GCC 15.2.0. Root typecheck and production Web build pass. The root
  parallel suite retains one unrelated Team Control five-second timeout; the
  exact file plus Knowledge passes 13/13 immediately in focused execution.
- Fresh backend runtime tests prove real filtering, ranking, cursor integrity,
  role visibility, Workspace isolation, and restart. Installed Chrome passes
  the query-only shared UI/provenance journey 1/1 with retries disabled. The
  browser spec mocks Knowledge responses and is therefore recorded only as UI
  projection evidence; it does not replace backend runtime acceptance.
- Candidate/worktree `server/**` diffs are empty, protected Input stays
  `a830fd2f0f82770563908d512558fe6ba48f50dd`, trailers parse, and all recorded
  dirty paths remain excluded. PCR-S06A is `complete-independent-reviewed`;
  Release 2 remains active with no active task. S06B needs a new frozen plan.
  No push, merge, or deployment is claimed.
