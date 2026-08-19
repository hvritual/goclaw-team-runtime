# Product capability roadmap v28 — S07A review-evidence remediation

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Plan-Version: `28`
- Task-Revision: `r033`
- Work-Item: `PCR-S07A`
- Exact base: `9fb86ea056d253c82ea61b13031550db82eeb526`
- Blocked input tree: `c6a2f08f41ae2424807b39fd9b5cef5990324f84`
- Blocked input patch hash: `9b03028ec2218ea75948cd47179984d056221408`
- Status: `approved-active`
- Authority: the Human Customer's confirmed direction on 2026-08-19 to
  continue approved execution until Release 3 is complete

## Why this successor exists

The v27/r032 exact candidate is commit
`9fb86ea056d253c82ea61b13031550db82eeb526`, tree
`c6a2f08f41ae2424807b39fd9b5cef5990324f84`, with 46 changed paths, zero
unstaged paths, and zero `server/**` paths. Its exact binary patch hash from the
v27 activation commit is `9b03028ec2218ea75948cd47179984d056221408`.
Focused backend/Core/Views checks, backend `make check`, exact changed-package
race checks, root typecheck, production Web build, and the retries-disabled
installed-Chrome journey passed. The root test aggregate retained two unchanged
Team Control five-second timeouts and the broad Windows race bootstrap retained
the known onboarding/outbox race; neither aggregate was represented as PASS.

Fresh independent review returned `SPEC BLOCK` and
`CODE/SECURITY/QUALITY BLOCK` for two evidence/traceability defects. First,
blank lines separated the candidate's required commit fields, so Git recognized
only `Policy-Bundle` as a trailer. Second, the blocked down-migration tests
proved that the schema and catalog survived but did not compare the retained
Resource, Resource-set, audit, and idempotency row values before and after the
failed migration. The reviewer found no additional product, security, or
quality blocker and observed that the migration guard ordering appears safe.
v27 and candidate `9fb86ea` remain immutable; this successor repairs only the
two missing proofs.

## Preserved outcome and exclusions

The complete v26/v27 S07A product, security, route, authorization, client, UI,
and installed-acceptance contracts remain binding. This successor does not
authorize a new Resource behavior, route, schema, migration, adapter, UI, or
runtime policy. If the new data snapshot assertion is already green, no product
implementation may change. If it exposes actual mutation before the guard
aborts, only the minimum `000018` down-migration execution correction and its
test may change.

S07B-D, Release 3 completion, generated protobufs, unrelated Input/Issue UI,
push, merge, deployment, and every `server/**` change remain inactive or
excluded. The original dirty worktree remains untouched.

## Frozen remediation contract

1. Each independently guarded retained dependency captures an exact,
   deterministic row snapshot after fixture insertion and before executing the
   `000018` down migration. After the expected failure, the test reads the same
   row and requires byte-for-byte equality, in addition to the existing table
   and migration-catalog assertions.
2. The snapshots cover all persisted fields of the Resource, revision-zero
   Resource set, audit rows guarded by resource kind or action namespace, and
   idempotency rows guarded by resource kind or action namespace. A missing row,
   changed field, extra normalization, or partial schema change fails the gate.
3. The new exact candidate commit contains one uninterrupted Git trailer block
   with `Task-ID`, `Project-ID`, `Task-Revision`, `Work-Item`, `Plan-ID`,
   `Plan-Version`, `Plan-Step`, `Issue`, and `Policy-Bundle`. The output of
   `git show -s --format=%(trailers:only)` must recognize every required field
   exactly once.
4. Product source behavior remains identical to candidate `9fb86ea` unless the
   new assertion first demonstrates real data drift. Documentation, this exact
   test, branch/task metadata, and commit metadata are not product behavior.

## Writable scope

- `backend/internal/modules/workspace/sqlite_persistence_test.go`;
- only if the new test first fails due to data mutation,
  `backend/internal/modules/workspace/internal/infrastructure/sqlite/migrations/000018_project_resources.down.sql`
  and the exact migration executor needed to preserve transaction atomicity;
- `backend/docs/plans/product-capability-roadmap/plan.md`;
- `backend/docs/plans/product-capability-roadmap/plan_v28.md`;
- `backend/docs/plans/product-capability-roadmap/story-map.md`;
- `backend/docs/plans/product-capability-roadmap/task-register.md`;
- `backend/docs/plans/product-capability-roadmap/journal.md`.

No other package, migration, generated artifact, legacy backend tree, original
dirty path, or `server/**` path is writable.

## Ordered execution

1. R33.1 — Freeze this successor from exact base `9fb86ea`, record r032's exact
   blocked candidate/review evidence, move the isolated branch to r033, and
   commit only the five governance activation paths with a parseable continuous
   trailer block.
2. R33.2 — Add the assertion-first exact before/after data snapshot coverage to
   every guarded dependency case. Run the focused test immediately. If it is
   already green, record that the blocker was missing evidence and make no
   product change; if it is red, preserve the failure and apply only the
   minimum authorized atomicity correction before proving green.
3. R33.3 — Run the focused migration test, the complete Workspace package,
   backend `make check`, and the official changed-package race command. Verify
   that all v27 product files remain unchanged and retain the already recorded
   Core/Views/typecheck/build/browser evidence for that identical product tree.
4. R33.4 — Freeze one exact candidate; verify path scope, diff check, all nine
   parsed trailers, empty `server/**` diff, clean isolated worktree, original
   dirty-worktree preservation, and exact hashes; then obtain fresh independent
   `SPEC PASS` and `CODE/SECURITY/QUALITY PASS` on that candidate.

## Acceptance and deterministic verification

- `TestProjectResourceDownMigrationRejectsEveryRetainedDependency` proves the
  expected failure and exact retained row value for all six dependency cases,
  plus both tables and migration catalog.
- The complete Workspace package and backend checks pass. The official race
  invocation for the changed Workspace package passes or is recorded honestly
  as an environment limitation; no failed aggregate is called PASS.
- The range from exact base `9fb86ea` contains only the frozen governance and
  test paths unless a captured RED required the explicitly authorized minimum
  migration fix. `server/**` remains empty.
- The final commit's parsed trailer block contains all nine required fields
  exactly once and uses v28/r033/R33.4 values and the exact v28 policy bundle.
- Only a fresh independent review returning both `SPEC PASS` and
  `CODE/SECURITY/QUALITY PASS` may close r033 and PCR-S07A. S07B remains
  inactive pending its own resolved scope and successor plan.

## Rollback and stop conditions

The normal rollback is removal of the assertion-only candidate and restoration
of the v27 capability-disable path; no production data or schema is changed by
this plan unless a captured RED proves the existing down migration mutates
data. Stop before closure on any retained-row drift, missing/duplicate parsed
trailer, product-source change without a preceding RED, scope drift,
`server/**` change, original dirty-worktree overlap, or independent review
BLOCK. Any further material repair requires another immutable successor plan.
