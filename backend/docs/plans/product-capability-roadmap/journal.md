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
