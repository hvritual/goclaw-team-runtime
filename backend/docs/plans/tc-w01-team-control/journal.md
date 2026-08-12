# TC-W01 Team Control Journal

## 2026-08-11 — TC-W01-S00 activated

- User explicitly authorized merging PR #8 and implementing TC-W01.
- PR #8 Head `18466522bafabb43d2ac2371875c9f343e5a771c` was still mergeable with no unresolved review threads.
- PR #8 was marked ready and merged with merge commit `c43f4300eb29cf6778e67594e54cc79f8fb5057e`, preserving the 49-commit P0-P2 audit chain.
- Created isolated branch `agent/tc-w01-team-control-001` from the exact merge commit.
- Confirmed `server/**` remains permanently read-only. TC-W01 may port read-only identity behavior but cannot import or modify the legacy backend.
- Confirmed Web and Desktop share `packages/core`, `packages/ui`, and `packages/views`; the Team Control view will be implemented once and wired into both platforms.
- Local environment has Node 24 and pnpm 11 but no Go, Docker, or GitHub CLI. Go, race, Docker, and remote branch checks must be obtained from CI and the connected GitHub repository.
- Next action: publish the immutable plan, then activate `TC-W01-S01`.

## 2026-08-11 — TC-W01-S01 implemented; plan v2 activated

- Added a production upstream identity resolver for the existing Web cookie session and Desktop bearer credential. Redirects are disabled, the upstream origin is validated, responses are size/time bounded, and only the authentication credential is forwarded to the three allowlisted read endpoints.
- Added local HMAC CSRF verification for cookie-authenticated unsafe requests. Production defaults to denied without `CONTROLPLANE_IDENTITY_UPSTREAM_URL`; actor headers require the explicit development switch.
- Added fail-closed trusted workspace/member reconciliation. Owners are synchronized first, stale human members are marked removed, Agents are left untouched, and every CAS conflict aborts the request instead of accepting stale authority.
- Added schema-versioned workspace/member read endpoints and identity/reconciliation tests.
- Activated `plan_v2.md`: human membership remains a read-only projection of one upstream authority; no duplicate role mutation API is introduced.
- Activated `TC-W01-S02`: OpenAPI and project-scoped resumable SSE are being implemented.

## 2026-08-11 — TC-W01-S02 verified; Draft PR opened

- Published OpenAPI v1 for the workspace, members, projection, command, Problem, append-result, and SSE contracts. All 23 supported command names are enumerated.
- Added project-scoped SSE with authorization, `after`/`Last-Event-ID` resume, one-second bounded polling, 15-second heartbeat, no intermediate state buffer, and request-context cleanup.
- Added `schema_version: 1` to replayed projections and schema-versioned workspace/member response envelopes. Problem responses use `application/problem+json` and safe public details.
- Opened Draft PR #9 at candidate `2172508f355436a658b575cb5d99f99d6ed96cf0`; the PR is mergeable and remains Draft for later frontend and independent acceptance work.
- GitHub Backend run #39 (`31512040101`) passed `make check` and `make test-race` on Go 1.26.1. This covers gofmt, path/import policy, generated-clean, vet, unit tests, and race tests.
- No `server/**` path or legacy backend import appears in the PR diff.
- Activated `TC-W01-S03`: the next slice is the shared TypeScript client and React Query model.

## 2026-08-12 — TC-W01 resumed under plan v3

- User explicitly resumed TC-W01 and authorized completion of S03 through S06,
  including independent product, code, security, and documentation review and
  merge of Draft PR #9 after blocking findings are cleared.
- User explicitly deferred CI work. Plan v3 forbids workflow, required-check,
  branch-protection, and Ruleset changes while preserving local deterministic,
  Docker, Playwright, responsive, and accessibility verification.
- Revalidated PR #9 at Head
  `8ca49704e720a545bd8a39436e74a1fc4608d9f6`: open, Draft, mergeable, with no
  submitted reviews or unresolved review threads.
- Preserved `server/**` as permanently read-only and left GOV-W01 PR #10 out of
  this execution.
- The available checkout containing legacy untracked work was rejected as a
  task worktree. A clean verification tree was reconstructed from the public
  Multica source plus the exact PR #9 files; remote commits will use the GitHub
  commit tree API with expected-Head checks because `gh` and private clone
  credentials are unavailable locally.
- Activated `TC-W01-S03` under Task Revision `r003`.
