# Product capability roadmap v30 — S07B independent-review remediation

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Plan-Version: `30`
- Task-Revision: `r035`
- Work-Item: `PCR-S07B`
- Exact base: `fa1153164882adb4880d57f7349597900196402f`
- Blocked input tree: `70de766121f06487eeb7c259906b1bbd0118594f`
- Blocked input binary patch hash: `e975739c2daf10088af9b24ae7cf034fd3aa8994`
- Status: `approved-active`
- Authority: the Human Customer's confirmed continuous Release 3 direction,
  confirmed prerequisite minimal outline authority, and confirmed execution
  after the v29/r034 independent-review result

## Predecessor and activation boundary

Immutable v29/r034 installs the complete PCR-S07B product candidate and passes
its focused, full backend, official race, strict Core/shared-view, production
build, and installed two-identity acceptance evidence. Its exact candidate is
`fa1153164882adb4880d57f7349597900196402f`. Fresh independent review returns
both `SPEC BLOCK` and `CODE/SECURITY/QUALITY BLOCK` for exactly two unresolved
proof/safety findings:

1. `000019_project_requirements.down.sql` does not compare an imported
   canonical Issue-link row's `workspace_id` and `project_id` to its retained
   legacy/baseline ownership, so ownership drift can escape the destructive
   rollback guard.
2. Repository tests do not yet prove mutation denial after live membership
   removal, project-lead reassignment, project-editor grant removal, or a
   terminal project-state change, despite the frozen same-transaction current
   authorization contract.

r034 is review-blocked and remains immutable. r035 starts only from the exact
blocked candidate and may close only these two findings. PCR-S07C, PCR-S07D,
Release 3 completion, and every unrelated behavior remain inactive.

## Goal and frozen behavior

Preserve the complete v29/r034 S07B candidate while making the down migration
fail closed on imported Issue-link ownership drift and adding executable proof
that every Requirement mutation re-reads current authority inside its owning
SQLite write transaction.

The product contract, public HTTP/Core/shared-view contract, lifecycle,
minimal-outline authority, flags, and installed behavior do not change.
`project_requirements` remains true, `project_outline` remains false, and the
legacy tables remain immutable runtime input.

## R35.2 down-migration repair contract

- Add assertion-first migration cases that independently mutate the imported
  canonical Issue-link `workspace_id` and `project_id` after a valid legacy
  import.
- Each case executes the real `000019` down migration, requires failure before
  the first destructive statement, and compares exact before/after retained
  row snapshots plus the canonical table/catalog presence.
- The smallest GREEN repair extends the existing imported Issue-link guard to
  require canonical link ownership to equal the retained legacy Requirement
  and canonical baseline ownership. It must not relax any existing guard.
- A valid exact untouched import must still roll back successfully. No up
  migration, runtime repository behavior, schema shape, or generated artifact
  changes.

## R35.3 live-authorization proof contract

Assertion-first repository tests must establish all of the following against a
previously authorized actor:

- deleting the actor's active Workspace membership denies the next mutation;
- reassigning the current project lead denies a lead-only editor mutation;
- removing the actor's current `project_editor` grant denies the next mutation;
- changing the project to each terminal state used by the Canonical project
  model denies the next mutation.

Each denial must use a real repository command after the authority row changes,
return the typed fail-closed error, and prove zero baseline, immutable revision,
link/grant/outline, idempotency, audit, or outbox effects. The tests must inspect
the persisted state after the denied command. Existing `BEGIN IMMEDIATE` and
same-connection authorization code remains unchanged if these RED tests are
already GREEN when expressed correctly. Production repository code may change
only when a captured RED proves an actual defect, and only by the smallest
repair that preserves revision precedence and transaction ownership.

## Writable scope

- `backend/internal/modules/workspace/internal/infrastructure/sqlite/migrations/000019_project_requirements.down.sql`;
- `backend/internal/modules/workspace/project_requirement_migration_test.go`;
- `backend/internal/modules/workspace/internal/infrastructure/sqlite/project_requirement_repository_test.go`;
- `backend/internal/modules/workspace/internal/infrastructure/sqlite/project_requirement_repository.go`
  only if an R35.3 RED proves the existing implementation violates this plan;
- `backend/docs/plans/product-capability-roadmap/plan.md`;
- `backend/docs/plans/product-capability-roadmap/plan_v30.md`;
- `backend/docs/plans/product-capability-roadmap/story-map.md`;
- `backend/docs/plans/product-capability-roadmap/task-register.md`;
- `backend/docs/plans/product-capability-roadmap/journal.md`.

No frontend, capability flag, other migration, generated protobuf, original
dirty path, legacy backend tree, or `server/**` path is writable. A necessary
path outside this list stops r035 and requires an immutable successor plan.

## Ordered execution

1. R35.1 — Freeze this successor from exact base `fa115316`, move the isolated
   branch to `codex/release3-s07b-r035`, mark r034 review-blocked, and commit
   only the five governance activation paths with one continuous nine-field
   trailer block.
2. R35.2 — RED the two independent Issue-link ownership-drift cases, preserve
   exact retained rows/catalog after failure, then GREEN only the down guard
   and prove exact untouched rollback still succeeds.
3. R35.3 — Add live membership, lead, editor-grant, and terminal-project denial
   proofs. Preserve the current implementation when the new tests pass; repair
   only a captured behavioral failure.
4. R35.4 — Run focused migration/repository/Workspace tests, backend
   `make check`, and the official changed-package `make test-race`. Reuse the
   unchanged r034 frontend/build/installed evidence only while no frontend or
   runtime contract changed; otherwise rerun the affected gate.
5. R35.5 — Freeze one exact candidate; verify plan/policy hashes, all nine
   trailers on every r035 commit, exact path scope, zero `server/**` and
   generated paths, clean isolated worktree, original dirty-tree preservation,
   and obtain fresh independent `SPEC PASS` plus
   `CODE/SECURITY/QUALITY PASS`.

## Deterministic acceptance

- The two new ownership-drift tests fail against the exact r034 down migration
  for the missing guard and pass after the one-clause repair.
- All existing `000019` import, rollback, and independent guard tests pass,
  including successful rollback of an exact untouched import.
- Live-authority tests cover membership removal, lead reassignment, current
  editor-grant removal, and every terminal project state, with typed denial and
  exact zero-effect assertions.
- Focused packages, complete Workspace tests, backend `make check`, and the
  repository-owned official race command pass. Any environment or unrelated
  aggregate failure remains NON-PASS and is disclosed rather than waived.
- Exact candidate audit proves only the authorized paths changed from
  `fa115316`, all r035 commits expose the required continuous trailer block,
  and the original dirty worktree has zero candidate overlap.
- Only a fresh independent review returning both required PASS decisions may
  close r035/PCR-S07B and authorize creation of the PCR-S07C successor plan.

## Explicit exclusions and stop conditions

PCR-S07C coverage semantics, PCR-S07D Retrospectives, Release 3 completion,
S10 hierarchy/move/reorder/numbering/archive/restore/Issue-outline/progress,
project phases, full outline UI/realtime, generated protobufs, unrelated
Issue/Input behavior, push, merge, deployment, external service calls, and all
`server/**` changes are excluded.

Stop before closure on a successful down migration after either ownership
mutation, any retained-row drift after an expected failure, authorization read
outside the owning write transaction, a denied mutation with any persisted
effect, altered public behavior without a captured RED, hidden test failure,
missing/duplicate trailer, scope drift, original dirty-path overlap,
`server/**` change, or either independent-review BLOCK decision. Any material
repair outside this exact boundary requires a new immutable plan; v30 is never
amended after activation.
