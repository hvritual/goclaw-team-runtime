# Product capability roadmap implementation plan v13

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Version: `13`
- Status: `approved-for-execution`
- Active step: `PCR-S02A real-Git timeout correction`
- Task-ID: `PCR-001-S02A-R18`
- Task-Revision: `r018`
- Work-Item: `PCR-S02A`
- Base commit: `8f94d69db72e18719b7ed4315e7d74901b1f5945`
- Supersedes: `plan_v12.md` for active execution only
- Approved: `2026-08-18`

## Authority, scope, and evidence

The Human Customer approved continued follow-up work through Release 1. v12
gave the three real-Git Desktop tests a 15-second timeout. Under root Turbo
parallel load, two passed and the most complex tag/commit/describe scenario
still exceeded 15 seconds; Desktop otherwise passed 422/423. The same file
passes 30/30 in isolation.

Authorize only changing the three `15_000` timeout literals in
`apps/desktop/scripts/package.test.mjs` to `60_000`. The 60-second ceiling is a
maximum for external-process integration tests, not a delay, retry, skip, mock,
or assertion change. Included authority paths are
`{plan.md,plan_v13.md,story-map.md,task-register.md,journal.md}` under the
roadmap directory. Every other tracked path, `server/**`, prior immutable plan,
product behavior, manifest, lockfile, and dirty exclusion remains unchanged.

## Execution and acceptance

1. Activate r018 from exact base `8f94d69`; mark r017 verification-blocked.
2. Change only the three timeout literals and run the isolated Desktop file.
3. Run root `pnpm typecheck` and `pnpm test`, backend `make check` and real
   `make test-race`, then exact clean-candidate Task Playwright acceptance.
4. Verify scope, plans, dirty blob, active-task uniqueness, hashes/trailers,
   process cleanup, and fresh independent review.
5. Only full PASS closes r018/S02A. S02B remains inactive pending a new plan.

Any additional path, deterministic failure, independent `BLOCK`, or
`server/**` path stops closure. Rollback reverts r018 commits only and never
resets user work.
