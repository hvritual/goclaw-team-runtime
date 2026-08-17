# Product capability roadmap implementation plan v10

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Version: `10`
- Status: `approved-for-execution`
- Active step: `PCR-S02A verification repair`
- Task-ID: `PCR-001-S02A-R15`
- Task-Revision: `r015`
- Work-Item: `PCR-S02A`
- Base commit: `906580292e6fbcd9a0d866cae796d0b67cfd975d`
- Supersedes: `plan_v9.md` for active execution only
- Frozen contract: `PCR-CONTRACT-v1`
- Approved: `2026-08-18`

## 1. Authority and objective

The Human Customer approved continued actions through Release 1 and explicitly
approved the follow-up scope after S02A implementation reached independent
review PASS but the frozen root verification commands remained deterministic
failures.

Repair only the repository verification baseline needed to close S02A:

1. remove the stale nonexistent `@multica/mobile` exclusion from the root
   `typecheck` and `test` scripts;
2. remove dead `_one` plural keys from locales whose runtime plural category is
   `other` only; and
3. make the rich-content boundary test's documented Tiptap exception
   path-separator independent on Windows.

This plan does not change Task product behavior, activate S02B, waive a failing
check, or claim Release 1 completion.

## 2. Frozen evidence

- Exact base `9065802` contains the complete S02A implementation.
- Backend `make check` and real `make test-race` pass.
- Core passes 593/593 tests; Task Views passes 8/8; Core, Views, Web, and Desktop
  typechecks pass.
- Exact-candidate Playwright with the Canonical fixture passes the installed
  create/edit/reorder/status/filter/archive/restore journey 1/1.
- Fresh independent S02A review returns PASS.
- Root `pnpm typecheck` and `pnpm test` stop before execution because Turbo is
  asked to exclude nonexistent package `@multica/mobile`.
- Direct Views full test execution reaches the suite and has exactly four
  deterministic failures: three dead `projects:team_control.node.assignees_one`
  locale keys and one Windows path-separator mismatch in the documented Tiptap
  NodeView exception.

## 3. Invariants

1. `server/**` remains permanently read-only.
2. `plan_v1.md` through `plan_v9.md` remain immutable.
3. S02A Task contracts, behavior, migrations, HTTP, Core queries, and Views
   product code are unchanged by this verification repair.
4. No failing test is skipped, weakened, deleted, or converted to a waiver.
5. Locale removal is limited to keys proven dead by `Intl.PluralRules` for the
   affected locale; English singular/plural keys remain intact.
6. The rich-content exception remains exactly the already-documented Tiptap
   NodeView; only path normalization changes.
7. Existing dirty and untracked user paths remain byte-for-byte untouched.

## 4. Included scope

- authority records under
  `backend/docs/plans/product-capability-roadmap/{plan.md,plan_v10.md,story-map.md,task-register.md,journal.md}`;
- root `package.json`, limited to `scripts.typecheck` and `scripts.test`;
- `packages/views/locales/{zh-Hans,ko,ja}/projects.json`, limited to deleting
  the dead `assignees_one` key;
- `packages/views/rich-content/rich-content-boundary.test.ts`, limited to
  platform-independent relative-path comparison.

## 5. Excluded scope

- every path under `server/**`;
- Task product behavior or contracts and all S02B/later capability work;
- build/lint/dev/mobile script repair, mobile restoration, dependency changes,
  package-lock changes, UI redesign, deployment, push, merge, or release tags;
- all v9 dirty exclusions, including
  `packages/ui/components/ui/input.tsx`,
  `packages/views/issues/components/table-view.tsx`,
  `packages/views/modals/create-issue.tsx`,
  `packages/views/auth/input-controlled.test.tsx`, `.local-runtime/**`,
  `docs/code-to-product/**`, and `ui/**`;
- generated protobuf status-only drift with no content diff.

Any additional path or semantic change requires a later approved plan version.

## 6. Ordered execution

### PCR-S02A-R15.1 — Activation

- Commit this immutable plan and r015 authority records from exact base
  `9065802`.
- Mark r014 verification-blocked rather than complete and establish r015 as the
  sole active task.

### PCR-S02A-R15.2 — RED and repair

- Preserve the exact root Turbo and Views full-suite failures as RED evidence.
- Apply only Section 4 changes and run focused tests after each repair.

### PCR-S02A-R15.3 — Integrated verification and review

- Run all Section 8 commands without retry-based waivers.
- Re-run the exact installed Task browser journey from a clean detached
  candidate and obtain fresh independent read-only review.

### PCR-S02A-R15.4 — Closure

- Only after every command and review passes, append indexed evidence and mark
  r015/S02A complete.
- Leave S02B inactive pending its own frozen plan/task. S02A closure alone does
  not complete Release 1.

## 7. Acceptance criteria

1. Root `pnpm typecheck` and `pnpm test` execute the current workspace graph and
   pass without referencing a nonexistent package.
2. Views full tests pass without skipping or weakening plural or rich-content
   boundary guards.
3. Focused Task tests, all four frontend typechecks, backend full checks, and
   real race checks remain green.
4. The clean exact-candidate installed Task browser journey remains green.
5. Candidate scope contains only Section 4 paths, no `server/**` change, and no
   change to v9 dirty exclusions.
6. Fresh independent review returns PASS.

## 8. Deterministic verification

```powershell
git diff --name-status 906580292e6fbcd9a0d866cae796d0b67cfd975d...HEAD
git diff --quiet 906580292e6fbcd9a0d866cae796d0b67cfd975d -- server
git status --porcelain -- server
git diff --quiet 906580292e6fbcd9a0d866cae796d0b67cfd975d -- backend/docs/plans/product-capability-roadmap/plan_v1.md backend/docs/plans/product-capability-roadmap/plan_v2.md backend/docs/plans/product-capability-roadmap/plan_v3.md backend/docs/plans/product-capability-roadmap/plan_v4.md backend/docs/plans/product-capability-roadmap/plan_v5.md backend/docs/plans/product-capability-roadmap/plan_v6.md backend/docs/plans/product-capability-roadmap/plan_v7.md backend/docs/plans/product-capability-roadmap/plan_v8.md backend/docs/plans/product-capability-roadmap/plan_v9.md
pnpm --filter @multica/views test -- locales/parity.test.ts rich-content/rich-content-boundary.test.ts
pnpm --filter @multica/core test -- tasks api
pnpm --filter @multica/views test -- tasks
pnpm --filter @multica/core typecheck
pnpm --filter @multica/views typecheck
pnpm --filter @multica/web typecheck
pnpm --filter @multica/desktop typecheck
pnpm typecheck
pnpm test
cd backend
make check
make test-race
cd ..
pnpm exec playwright test e2e/tasks.spec.ts --project=chromium --reporter=line
```

Before closure, verify the frozen dirty-file blob, enumerate active task markers,
validate plan hashes/trailers/links, and stop owned acceptance processes.

## 9. Stop and rollback

- Any new deterministic failure, independent `BLOCK`, scope escape, or
  `server/**` path stops closure.
- Revert r015 commits in reverse order without resetting or overwriting user
  dirty paths.
- No database, deployment, external system, or production rollback is involved.
