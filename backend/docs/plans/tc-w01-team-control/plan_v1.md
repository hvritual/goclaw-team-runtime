# Team Control Connection Plan v1

- Plan-ID: `TC-W01-TEAM-CONTROL-001`
- Version: `v1`
- Status: `approved-for-execution`
- Base commit: `c43f4300eb29cf6778e67594e54cc79f8fb5057e`
- Branch: `agent/tc-w01-team-control-001`
- Project-ID: `goclaw-team-runtime`
- Task-ID: `TC-W01-TEAM-CONTROL-001`
- Task-Revision: `r001`
- Policy bundle: `backend/AGENTS.md@ffe45b83ef3884d9b8bf66e6f7994b3a43d4f86e`

## Goal

Connect the existing Multica Web and Desktop Team Control experience to the canonical `backend/` Delivery Kernel so authenticated workspace members can read replayed project state, execute authorized typed commands, observe incremental projection updates, and manage the control-plane workspace/member projection without creating a browser-side source of truth.

## Hard invariants

1. `backend/` remains the only writable backend root; `server/**` is permanently read-only.
2. The Go Delivery Kernel remains the sole authority for project head, governed work state, Evidence, Check, acceptance, and Runner lifecycle.
3. Browser/desktop identity comes from the existing signed session or bearer credential and an authoritative upstream workspace-membership check; client-supplied actor/role headers are never trusted in production.
4. Cookie-authenticated unsafe requests require the existing HMAC-bound CSRF token. Bearer credentials are never persisted by the Team Control feature.
5. Workspace/member synchronization is a projection of an authenticated upstream membership response and cannot create an Agent Owner or remove the last active human Owner.
6. React Query owns Team Control server state. Zustand is not used for projections, members, commands, or events.
7. Commands use server-returned project Head CAS and idempotency keys; a conflict refreshes the projection instead of hiding or overwriting concurrent work.
8. SSE only invalidates/refetches the scoped React Query projection; it does not mutate a parallel client state graph.
9. Loading, empty, denied, conflict, offline, and internal-error states are explicit in the shared Web/Desktop view.
10. No credential, cookie, secret value, or unsanitized token is logged, stored in events, or committed.

## Allowed paths

- `backend/docs/plans/tc-w01-team-control/**`
- `backend/cmd/controlplane/**`
- `backend/internal/controlplane/**`
- `backend/openapi/**`
- `backend/ci/check-policy.sh`
- `backend/README.md`, `backend/Makefile`, `backend/go.mod`, `backend/go.sum`
- `packages/core/api/**`
- `packages/core/team-control/**`
- `packages/core/package.json`
- `packages/views/team-control/**`
- `packages/views/projects/components/project-detail.tsx`
- `packages/views/package.json`
- `apps/web/app/[workspaceSlug]/(dashboard)/projects/[id]/control/**`
- `apps/desktop/src/renderer/src/routes.tsx`
- `e2e/team-control.spec.ts`
- `.github/workflows/backend.yml`
- `.github/workflows/team-control.yml`

## Forbidden paths

- `server/**` without exception.
- Legacy root `teamcontrol/**`, `gateway/**`, `workstation/**`, and `ui/**`.
- Mobile, docs site, billing, unrelated issue/task/knowledge implementation, database schemas outside `backend/`, deployment, and release paths.

## Non-goals

- No production deployment, DNS, reverse-proxy mutation, release, or automatic merge.
- No replacement of existing Multica Issue, Task, Requirement, or Knowledge pages in this Wave.
- No automatic acceptance by Runner or model output.
- No direct import from `server/**` and no dual-write from the browser to old and new state stores.
- No WebSocket protocol; SSE is the single incremental transport for this Wave.

## Ordered steps

### TC-W01-S00 — Freeze plan and scope

Scope: this plan, pointer, and append-only journal only.

Acceptance: the merged base, allowed paths, identity boundary, state ownership, verification, risks, and rollback are explicit.

### TC-W01-S01 — Production identity and workspace projection

Implement an upstream-backed identity resolver that forwards only the incoming Authorization/cookie credential to authenticated read-only identity/workspace/member endpoints, validates CSRF locally for cookie mutations, maps roles fail-closed, and reconciles trusted workspace/member snapshots into the control-plane repository. Expose typed read/manage endpoints for workspace and members.

Acceptance:

- default configuration denies when the identity upstream is absent;
- arbitrary actor/role headers remain ineffective outside the explicit development switch;
- revoked/non-member/foreign-workspace sessions fail closed;
- cookie mutation without a valid CSRF token is denied;
- upstream owner/member changes reconcile without violating the human-Owner invariant;
- tests cover malformed, slow, 401/403/404/5xx, oversized, and cross-workspace upstream responses.

### TC-W01-S02 — Contract and incremental projection

Publish the Team Control OpenAPI contract, complete workspace/member/projection/command endpoints, and add project-scoped SSE with Last-Event-ID/after-sequence resume, heartbeat, authorization, bounded polling, and disconnect cleanup.

Acceptance:

- contract defines stable Problem responses and all supported command payloads;
- projection, command result, workspace, and member responses are schema-versioned;
- SSE emits only authorized project-scoped updates and resumes without duplicate state authority;
- slow/disconnected clients release resources and never block command commits.

### TC-W01-S03 — Shared TypeScript client and React Query model

Add defensive Zod schemas, typed API methods sharing the existing auth/CSRF session, scoped query keys, command mutations, CAS conflict refresh, and one SSE invalidation hook under `packages/core/team-control`.

Acceptance:

- malformed response tests degrade safely instead of crashing;
- query keys include workspace and project IDs;
- independent requests are parallel and duplicate requests are deduplicated by React Query;
- command mutation never optimistically advances authoritative governed state;
- SSE only invalidates the matching projection key.

### TC-W01-S04 — Shared Web/Desktop Team Control surface

Implement one shared project-scoped Team Control view under `packages/views/team-control` and wire it into both Web and Desktop project routes and the existing project detail entry point. Reuse installed shadcn primitives and semantic tokens.

Acceptance:

- overview, requirements/tasks, defects/risks, review/knowledge, Runner, Evidence/Checks, and members are visible from one project surface;
- supported human actions dispatch typed commands with current Head and visible confirmation/error feedback;
- loading, empty, denied, conflict, reconnecting, and error states render explicitly;
- Web and Desktop use the same view and headless core logic;
- keyboard/focus labels, headings, contrast, responsive layout, and modal titles meet accessibility checks.

### TC-W01-S05 — Deterministic and rendered verification

Add root Team Control CI, backend Docker-build evidence, focused unit/contract tests, Web/Desktop typecheck/build, and rendered browser E2E for project entry, projection render, command submit/conflict refresh, SSE refresh, denied state, and a mobile-sized viewport.

Acceptance:

- backend `make check` and `make test-race` pass;
- control-plane Docker image builds;
- focused core/views/web/desktop tests and typechecks pass;
- Browser validation has page identity, non-blank, no framework overlay, healthy console, screenshot evidence, and an exercised command/state transition;
- the final diff contains no `server/**` path and no credential material.

### TC-W01-S06 — Independent acceptance handoff

Freeze the exact candidate SHA, index deterministic and rendered evidence, open a Draft PR, and request independent product/code/security/docs review. The implementation author cannot mark final DoneGate acceptance.

## Deterministic verification

```bash
cd backend && make check && make test-race
docker build -t goclaw-controlplane:tc-w01 backend
pnpm --filter @multica/core typecheck
pnpm --filter @multica/core test
pnpm --filter @multica/views typecheck
pnpm --filter @multica/views test
pnpm --filter @multica/web typecheck
pnpm --filter @multica/web build
pnpm --filter @multica/desktop typecheck
pnpm --filter @multica/desktop build
pnpm exec playwright test e2e/team-control.spec.ts
```

Repository policy checks additionally reject every `server/**` diff and imports from the legacy backend.

## Dependencies

- Merged P0-P2 base `c43f4300eb29cf6778e67594e54cc79f8fb5057e`.
- Existing Multica human session/JWT/PAT validation endpoints and workspace/member read endpoints remain available as a read-only identity upstream.
- Production reverse proxy maps one same-origin Team Control prefix to the control-plane process; deployment changes are outside this Wave.
- Node/pnpm for local TypeScript checks; Go/Docker and rendered browser checks may execute in CI when unavailable locally.

## Risks and mitigations

- Identity upstream outage: return retryable 503, never downgrade to Header identity.
- Membership drift: reconcile only authenticated complete snapshots and fail closed on malformed/incomplete Owner data.
- Credential forwarding: allowlist the upstream origin and exact endpoint paths; do not follow redirects; cap response size and timeouts; redact all auth values.
- SSE fan-out: bounded heartbeat/poll interval, per-request cancellation, no unbounded buffers.
- CAS conflicts: expose 409 Problem, invalidate/refetch scoped projection, require explicit user retry.
- Frontend/backend contract drift: Zod boundary parsing, OpenAPI contract tests, unknown-enum defaults.
- Shared UI regression: implement once in `packages/views`, with both Web and Desktop route tests and rendered validation.

## Rollback

Revert the TC-W01 merge commit or disable the Team Control route/reverse-proxy mapping. The append-only Delivery Kernel data remains intact; no `server/**` schema or API must be rolled back. Removing the new frontend route and control-plane process restores the previous product without rewriting events or legacy application state.

