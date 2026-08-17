# Product capability roadmap implementation plan v11

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Version: `11`
- Status: `approved-for-execution`
- Active step: `PCR-S02A Desktop verification repair`
- Task-ID: `PCR-001-S02A-R16`
- Task-Revision: `r016`
- Work-Item: `PCR-S02A`
- Base commit: `2aac82e742932ddcda2ab6fa1387a9960a530974`
- Supersedes: `plan_v10.md` for active execution only
- Frozen contract: `PCR-CONTRACT-v1`
- Approved: `2026-08-18`

## 1. Authority and objective

The Human Customer approved continued follow-up actions through Release 1.
v10 repaired its captured root and Views failures, after which root
`pnpm test` reached Desktop and exposed two new deterministic blockers.

Repair only those Desktop verification blockers:

1. remove the unnecessary shebang from `apps/desktop/scripts/package.mjs`,
   which is always invoked explicitly through `node` and whose shebang becomes
   an invalid token when Vitest evaluates Vite-transformed module code; and
2. restore the untracked Electron runtime from the already-present local
   Electron 39.8.7 cache so Desktop import tests run against the locked
   dependency, without changing manifests or lockfiles.

## 2. Frozen evidence

- Exact base `2aac82e` contains the v10 verification repairs.
- v10 focused Views tests pass 131/131 and root `pnpm typecheck` passes across
  the current workspace graph.
- Root `pnpm test` now reaches Desktop and fails only because
  `scripts/package.test.mjs` cannot collect with `Invalid or unexpected token`
  and Electron lacks `path.txt/dist`.
- `node --check` passes for both package files; Vite debug proves both files
  transform before evaluation, localizing the syntax failure to the retained
  CLI shebang in transformed module execution.
- Every repository call site invokes `node scripts/package.mjs`; executable
  shebang invocation is not used.
- The exact cached archive
  `electron-v39.8.7-win32-x64.zip` is present under the local Electron cache.

## 3. Invariants and scope

- `server/**` and `plan_v1.md` through `plan_v10.md` remain immutable.
- Included tracked paths are only
  `apps/desktop/scripts/package.mjs` and the roadmap authority records
  `{plan.md,plan_v11.md,story-map.md,task-register.md,journal.md}`.
- The package change removes only line 1; packaging behavior and exports remain
  unchanged.
- Electron recovery modifies only ignored/untracked `node_modules` using the
  locked 39.8.7 postinstall and the existing local cache. No manifest,
  lockfile, version, or application source changes are authorized.
- No network fallback is accepted: a cache miss or download attempt stops the
  operation for explicit disposition.
- All v10 dirty exclusions remain untouched. S02B and later stories remain
  inactive.

## 4. Ordered execution

1. Commit this immutable plan and r016 authority records from exact base
   `2aac82e`; mark r015 verification-blocked and r016 sole active.
2. Preserve the two Desktop failures as RED evidence, remove only the shebang,
   and run the focused package test.
3. Run Electron's locked postinstall against the existing cache, verify
   `path.txt/dist`, and run the focused endpoint loader test.
4. Run root `pnpm typecheck`, root `pnpm test`, backend full/race checks, and a
   clean exact-candidate Task browser journey; obtain fresh independent review.
5. Only after every gate passes, close r016/S02A and leave S02B inactive pending
   its own frozen plan.

## 5. Deterministic verification

```powershell
node --check apps/desktop/scripts/package.mjs
pnpm --filter @multica/desktop exec vitest run scripts/package.test.mjs
pnpm --filter @multica/desktop exec vitest run src/main/endpoint-config-loader.test.ts
pnpm typecheck
pnpm test
cd backend
make check
make test-race
cd ..
pnpm exec playwright test e2e/tasks.spec.ts --project=chromium --reporter=line
git diff --quiet 2aac82e742932ddcda2ab6fa1387a9960a530974 -- server
git status --porcelain -- server
```

Closure also requires exact scope, immutable-plan, dirty-blob, active-task,
trailer, plan-hash, owned-process cleanup, and independent-review checks.

## 6. Stop and rollback

- Any cache miss, network fallback, additional tracked path, deterministic
  failure, independent `BLOCK`, or `server/**` path stops closure.
- Revert r016 commits in reverse order without resetting user work.
- The local Electron runtime is reproducible ignored dependency state; tracked
  rollback does not delete shared caches or user files.
