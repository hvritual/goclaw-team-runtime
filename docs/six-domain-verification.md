# Multica six-domain verification

## Status

- Baseline contract: `passed`
- Deterministic verification: `passed`
- Independent review: `pending`
- Baseline commit: `2e67a09a803d4d041b6761f8c02d7c2f2c32d254`

Independent review remains a separate acceptance step. This record reports deterministic checks only.

## Environment

| Item | Value |
| --- | --- |
| OS | Darwin 21.6.0 x86_64 |
| Node.js | 22.12.0 |
| pnpm | 10.28.2 |
| Go | 1.26.2 darwin/amd64 |

Dependencies were installed with `pnpm install --frozen-lockfile`. The lockfile did not change.

## Results

| Scope | Command | Result |
| --- | --- | --- |
| Six-domain boundary | `pnpm verify:six-domains` | PASS: all six domains, all eight layers per domain, and all route/event markers |
| CLI contracts | guarded `go test ./cmd/multica` for workspace, project, issue, skill, and compatibility tests | PASS |
| Workspace authorization | guarded `go test ./internal/middleware -count=1` | PASS |
| Task service and skill bundles | guarded `go test ./internal/service -run 'Test(Task\|BuildAgentSkillBundles)' -count=1` | PASS |
| Daemon skill resolution | guarded `go test ./internal/daemon -run 'Test(SkillBundleCache\|EnsureTaskSkillBundles\|LoadRuntimeLocalSkillBundle)' -count=1` | PASS |
| Shared core | six focused Vitest files for workspace, project, issue, and skill | PASS: 6 files, 71 tests |
| Shared views | seven focused Vitest files for workspace, member settings, project, issue, task transcript, and skill | PASS: 7 files, 66 tests |
| Type safety | `pnpm typecheck` | PASS: 6 Turbo tasks |
| Patch integrity | `git diff --check` | PASS |

The guarded Go commands did not resolve or execute any user-installed agent CLI.

## Reproduced test timeout

The first two grouped view runs reproduced one timeout in
`RuntimeLocalSkillImportPanel > imports a single skill when selected via checkbox`:

- role: local developer test runner;
- project: `@multica/views`;
- action: run the seven six-domain Vitest files in one process;
- expected: 66 tests pass;
- actual: 65 passed and the local-skill import test exceeded Vitest's default 5-second total timeout after about 6 seconds;
- isolated result: the same file passed 14/14;
- sanitized log: `Error: Test timed out in 5000ms`.

The test already permits individual asynchronous waits of up to 5 seconds. Its local total timeout is now 10 seconds so grouped transformation load cannot expire the test before its existing assertions finish. No assertion or product behavior changed. The same grouped command then passed 7/7 files and 66/66 tests.

## Deferred broad checks

`pnpm test`, `make test`, database-backed integration tests, Playwright, live environments, deployments, and real-agent smoke tests were not run. The six-domain boundary check, focused Go/TypeScript suites, and full TypeScript typecheck are the deterministic acceptance set for this baseline.
