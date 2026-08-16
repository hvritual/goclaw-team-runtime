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
