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
