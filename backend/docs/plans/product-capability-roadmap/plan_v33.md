# Product capability roadmap v33 — S07C race-gate remediation

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Plan-Version: `33`
- Task-Revision: `r038`
- Work-Item: `PCR-S07C`
- Exact base: `2f4e801552c3bdcc8a20e4e9fe6981a83c79ee1a`
- Predecessor plan: `plan_v32.md`
- Predecessor plan hash: `4c43962f9cb2fc319efa873d63c242a7234e12d557bb0b2d7f9df12fef28a823`
- Predecessor coverage plan: `plan_v31.md`
- Predecessor coverage plan hash: `422f6ea3ab573f77c7c718105b04c06f132e1b4fd9c422bc06e8ee4433e2a081`
- Predecessor closure candidate: `cd94396093ea73f3f9434fed7410036ae61170ab`
- Status: `approved-active`
- Authority: the Human Customer's confirmed continuous Release 3 direction,
  confirmed prerequisite minimal outline authority, and confirmed execution

## Predecessor and stop boundary

Immutable v32/r037 preserves the complete v31 PCR-S07C Requirement-coverage
contract and authorizes one Auth test-harness stabilization. Commit `2f4e8015`
changes only two in-process `kratoshttp.NewServer()` sites in the affected
restart-persistence test to a finite ten-second timeout. The exact Auth test and
complete Auth package pass, then a fresh complete backend `make check` passes in
383.6 seconds under normal repository parallelism. All prior r036 aggregate
deadline failures remain NON-PASS evidence.

The subsequent official seven-package Windows race command uses the
repository-owned wrapper and Scoop MinGW GCC 15.2.0. Workspace contract,
application, Requirement domain, SQLite infrastructure, HTTP, and root packages
pass. Bootstrap fails after the race detector reports an unsynchronized read and
write of the test-local mutable `now` variable in
`TestSQLiteRuntimeCompletesNewUserOnboarding`: the Governance outbox goroutine
reads through `dependencies.Now` while the test goroutine advances `now` at the
retry boundary. The complete race command remains NON-PASS after 439 seconds.

The exact Bootstrap race test passes once because scheduling does not overlap,
then a ten-execution focused race run reproduces the same read/write conflict
and fails after 39.1 seconds. This is a real test-harness race with a complete
detector stack, not an old-GCC loader limitation or product waiver. The required
path `backend/internal/bootstrap/onboarding_runtime_test.go` is outside v32's
exact list, so v32's stop condition is satisfied. r037 cannot close and v32
remains immutable.

## Frozen remediation

Inside `TestSQLiteRuntimeCompletesNewUserOnboarding` only, add one local
`sync.RWMutex`. Both injected clock functions must acquire the read lock before
returning `now`; the single test-time advance must acquire the write lock around
`now = now.Add(time.Hour)`. The existing `sync` import is retained and no new
dependency or helper surface is needed.

The fix must preserve the exact deterministic timestamps, onboarding retry,
restart persistence, outbox behavior, all existing assertions, and production
concurrency. Stopping or serializing the outbox, replacing the production clock,
adding sleep/retry, skipping the test, suppressing the race detector, removing
the time advance, or declaring the first six packages a race PASS is forbidden.

The complete v31 coverage semantics and v32 Auth test remediation remain frozen
without amendment: coverage is a bounded consistent derived read over the fixed
traceable Requirement sections, revision-relative content/link intervals,
current Issue status, and latest acceptance conclusion; stages are exactly
`unlinked`, `linked`, `implemented`, and `accepted`, with fail-closed multi-Issue
aggregation, deleted-Issue exclusion, and retired visibility. No client fallback
or stored cache is allowed.

## Exact writable boundary

Race-harness remediation:

- `backend/internal/bootstrap/onboarding_runtime_test.go`.

Retained v32 Auth test path, writable only if verification exposes a defect in
the frozen remediation:

- `backend/internal/modules/auth/local_auth_http_test.go`.

Retained v31 backend product/test paths, writable only if verification exposes
an S07C defect within the frozen contract:

- `backend/internal/modules/workspace/contract/project_requirement.go`;
- `backend/internal/modules/workspace/internal/application/project_requirement.go`;
- `backend/internal/modules/workspace/internal/application/project_requirement_test.go`;
- `backend/internal/modules/workspace/internal/infrastructure/sqlite/project_requirement_repository.go`;
- `backend/internal/modules/workspace/internal/infrastructure/sqlite/project_requirement_repository_test.go`;
- `backend/internal/modules/workspace/internal/interfaces/http/project_requirement.go`;
- `backend/internal/modules/workspace/internal/interfaces/http/project_requirement_test.go`;
- `backend/internal/modules/workspace/project_requirement_composition_test.go`.

Retained v31 strict Core/realtime paths under the same condition:

- `packages/core/types/project-requirement.ts`;
- `packages/core/types/index.ts`;
- `packages/core/project-requirements/schema.ts`;
- `packages/core/project-requirements/queries.ts`;
- `packages/core/project-requirements/queries.test.tsx`;
- `packages/core/api/client.ts`;
- `packages/core/api/project-requirement-schema.test.ts`;
- `packages/core/realtime/use-realtime-sync.ts`;
- `packages/core/realtime/use-realtime-sync-ws-instance.test.tsx`.

Retained v31 shared-view/locale paths under the same condition:

- `packages/views/projects/components/project-requirement-baseline.tsx`;
- `packages/views/projects/components/project-requirement-baseline.test.tsx`;
- `packages/views/locales/en/projects.json`;
- `packages/views/locales/ja/projects.json`;
- `packages/views/locales/ko/projects.json`;
- `packages/views/locales/zh-Hans/projects.json`.

Governance:

- `backend/docs/plans/product-capability-roadmap/plan.md`;
- `backend/docs/plans/product-capability-roadmap/plan_v33.md`;
- `backend/docs/plans/product-capability-roadmap/story-map.md`;
- `backend/docs/plans/product-capability-roadmap/task-register.md`;
- `backend/docs/plans/product-capability-roadmap/journal.md`.

Immutable v31/v32, every other Bootstrap/Auth path, migration, Issue production
mutation, capability flag, generated protobuf, original dirty path, legacy
backend tree, and every `server/**` path are read-only. A necessary path outside
this exact list stops r038 and requires another immutable successor plan.

## Ordered execution

1. R38.1 — Freeze this plan from exact base `2f4e8015`, rename the isolated
   branch to `codex/release3-s07c-r038`, and commit only the five governance
   activation paths with one continuous nine-field trailer block.
2. R38.2 — Retain the official race and focused x10 failures as RED, add only
   the test-local locked clock, then pass the exact Bootstrap race repeatedly
   and the fresh official seven-package race with GCC 15.2.0.
3. R38.3 — Re-run focused S07C/Auth/Bootstrap suites, complete Workspace,
   backend `make check`, strict frontend checks, full Core/Views, root
   typecheck/test, production Web build, and fresh installed acceptance against
   real Canonical HTTP plus production Web.
4. R38.4 — Freeze one exact candidate; verify v31/v32/v33 and policy hashes,
   all nine trailers on every r038 commit, exact path scope, zero `server/**`
   and generated paths, clean isolated worktree, original dirty-tree
   preservation, process cleanup, and obtain fresh independent `SPEC PASS` plus
   `CODE/SECURITY/QUALITY PASS`.

## Deterministic and installed acceptance

- The exact Bootstrap onboarding race test passes at least ten consecutive
  executions with the official wrapper and no detector warning. A fresh
  official seven-package race passes all seven packages using GCC 15.2.0; the
  earlier official and focused failures remain disclosed.
- The exact/full Auth tests and a fresh complete backend `make check` pass under
  normal repository parallelism. All v31 coverage, strict Core, invalidation,
  realtime, view, locale, complete Workspace, full Core/Views, type/lint, root,
  and production build gates pass freshly. Any broad failure remains NON-PASS
  with exact attribution and is never renamed PASS.
- Fresh SQLite installed acceptance uses authenticated reviewer authority, real
  Canonical HTTP, and production Web to prove `unlinked -> linked -> implemented
  -> accepted`, later conditional revocation, multi-Issue fail-closed behavior,
  current/effective divergence, unlink, retirement visibility, refresh/reload,
  and restart persistence. Membership fixtures, if unavoidable, are disclosed
  and not counted as product behavior.
- Only fresh independent review returning both required PASS decisions may
  close r038/PCR-S07C and authorize the PCR-S07D successor plan.

## Explicit exclusions and stop conditions

PCR-S07D Retrospectives, Release 3 completion, S10 hierarchy/move/reorder/
numbering/archive/restore/Issue-outline/progress, project phases, Requirement-
driven Issue creation, stored coverage caches, historical Issue-status or
acceptance snapshots, new permissions/flags, migrations/indexes, generated
protobufs, Bootstrap/Auth production behavior, unrelated Issue/Input behavior,
push, merge, deployment, external service calls, original dirty paths, legacy
backend writes, and all `server/**` changes are excluded.

Stop before closure on any production concurrency change, outbox suppression,
sleep/retry/skip/race-detector workaround, client-inferred coverage, fail-open
Issue aggregation, partial response, authority leak, hidden test failure,
missing/duplicate trailer, scope drift, original dirty-path overlap, unclosed
process, `server/**` or generated change, or either independent-review BLOCK.
Any repair outside this exact boundary requires a new immutable plan; v33 is
never amended after activation.
