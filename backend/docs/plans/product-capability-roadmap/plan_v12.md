# Product capability roadmap implementation plan v12

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Version: `12`
- Status: `approved-for-execution`
- Active step: `PCR-S02A root parallel-test repair`
- Task-ID: `PCR-001-S02A-R17`
- Task-Revision: `r017`
- Work-Item: `PCR-S02A`
- Base commit: `3c030b4a6f8e9d47e1029d3c3245733768644415`
- Supersedes: `plan_v11.md` for active execution only
- Frozen contract: `PCR-CONTRACT-v1`
- Approved: `2026-08-18`

## 1. Authority and objective

The Human Customer approved continued follow-up actions through Release 1.
v11 removed the Desktop package module's redundant shebang and restored the
locked Electron runtime from the existing local cache. Focused Desktop tests
then passed 30/30 and 4/4. Root `pnpm test` reached all Desktop tests but three
real-Git integration cases exceeded Vitest's default five-second per-test limit
under full-workspace parallel load; the same file passes 30/30 in isolation.

Authorize only `apps/desktop/scripts/package.test.mjs` to give those three
process-heavy real-Git tests an explicit 15-second timeout. Assertions, Git
commands, production code, and all other tests remain unchanged.

## 2. Invariants and scope

- Included tracked paths are only
  `apps/desktop/scripts/package.test.mjs` and roadmap authority records
  `{plan.md,plan_v12.md,story-map.md,task-register.md,journal.md}`.
- Only the three tests under `deriveVersion (real git describe)` receive
  `15_000` millisecond timeouts.
- No test is skipped, retried, weakened, mocked, or made conditional.
- `server/**`, `plan_v1.md` through `plan_v11.md`, product code, manifests,
  lockfiles, and all dirty exclusions remain unchanged.
- S02B and later stories remain inactive.

## 3. Ordered execution and acceptance

1. Commit this immutable plan/r017 from exact base `3c030b4`; mark r016
   verification-blocked and r017 sole active.
2. Add only the three explicit timeouts and verify the isolated Desktop file.
3. Run root `pnpm typecheck` and `pnpm test`, backend `make check` and real
   `make test-race`, then exact clean-candidate Task Playwright acceptance.
4. Verify scope, immutable plans, user dirty blob, active-task uniqueness,
   hashes/trailers, owned-process cleanup, and obtain fresh independent review.
5. Only full PASS closes r017/S02A; S02B remains inactive pending a new plan.

## 4. Stop and rollback

Any additional tracked path, deterministic failure, independent `BLOCK`, or
`server/**` path stops closure. Revert r017 commits in reverse order without
resetting user work. No deployment or database action is authorized.
