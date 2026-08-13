# Canonical SQLite runtime cutover — execution plan v2

- Plan-ID: `canonical-sqlite-runtime-cutover`
- Version: `2`
- Status: `approved`
- Approval source: Human Customer confirmation dated 2026-08-13
- Supersedes: `plan_v1.md`
- Base commit: `e20114cc7f401b503c6506d1b99cf0eddf894780`
- Branch and integration target: `codex/multica-six-domain-baseline`
- Repository: `F:\code\ai\goclaw-team-runtime`
- Project-ID: `goclaw-team-runtime`
- Active step after approval: `M1-S1`
- XP mode: `strict`; maximum active stories: `1`

## Reason for revision

Issue metadata v9 is independently reviewed, committed as `e20114c`, and
fast-forwarded into the local baseline. S0 proved that the installed Issue
detail mounts many deferred sub-surfaces. This version freezes the supported
journey, honest capability gates, runtime owner, ports, DB, selector, tests and
write paths. See [contract-inventory-v2.md](contract-inventory-v2.md).

## Goal, scope and non-goals

One supported workflow starts Web plus Canonical backend on real SQLite;
login/current user, Workspace selection, Issue list/base detail, metadata
Get/Put/Delete and committed refresh work without a legacy process.

This is not full legacy parity. Deferred detail sub-surfaces are disabled by
explicit capabilities, never fabricated. Production PostgreSQL/deployment,
Team Control consolidation, desktop packaging, mobile, and any `server/**`
write are out of scope.

## Invariants

1. `server/**` has no diff and is read-only evidence.
2. Existing request/response bodies do not change; additive capability fields
   are schema-tested.
3. One Canonical process owns the product DB; control-plane storage is separate.
4. Session identity and membership authorize before repository access.
5. Every Workspace access uses canonical Workspace ID.
6. SQLite multi-state writes are atomic; events publish after commit.
7. React Query owns server state; reconnect refreshes accepted scoped keys.
8. Required providers fail startup/readiness rather than selecting stubs.
9. Unrelated worktree changes are preserved and excluded.
10. Only one active story writes its declared paths.

## Dependencies

- Issue metadata v9 commit `e20114c` is present in the local baseline.
- S0 inventory and parity decisions receive Human Customer approval.
- Before every story, revalidate base, policy, dirty exclusions, active step,
  exact paths, and preceding evidence.

## Ordered stories

### M1-S1 — Canonical SQLite application composition

RED first: empty-DB startup currently has no real providers or dependency-aware
readiness.

Allowed product paths:

- `backend/cmd/server/main.go`, `backend/cmd/server/*_test.go`
- `backend/internal/bootstrap/application.go`, `runtime.go`, focused new SQLite
  composition/test files there
- public Auth/Workspace composition and focused tests below
  `backend/internal/modules/auth/**` and `backend/internal/modules/workspace/**`
- backend-local configuration/docs required by this runtime

Acceptance: shared DB open/close, ordered repeatable Auth+Workspace migrations,
real provider graph, empty and retained restart, dependency-aware readiness,
health, graceful stop, and missing-provider failure. No auth HTTP journey is
claimed yet.

Focused command: `go test ./internal/bootstrap ./cmd/server
./internal/modules/auth ./internal/modules/workspace -count=1`, followed by full
backend tests, vet, module verification and applicable policy gates.

### M1-S2 — Trusted local authentication

Allowed paths: Auth domain/application/SQLite/user-owned HTTP/session extensions
under `backend/internal/modules/auth/**`, bootstrap wiring, Core API schemas and
focused auth tests, and shared auth tests only when RED requires a view change.

Implement frozen send-code, verify-code, current-user and logout. Sessions are
server-issued, expiring and revocable; cookie mutations enforce CSRF. Local code
delivery is development-only. Missing/expired identity fails closed.

### M1-S3 — Authorized Workspace selection

Allowed paths: membership adapters/use cases under Auth/Workspace, bootstrap
HTTP registration, and Core Workspace schema/client/query tests. Implement
authenticated list and slug-to-ID resolution. Prove empty/member/non-member,
role, missing identity and foreign matrices.

### M1-S4 — Issue list and honest base detail

Allowed paths: Issue application/SQLite/HTTP below Workspace, bootstrap
registration, `packages/core/api/**`, `packages/core/issues/**`, focused types
and tests, and capability-aware `packages/views/issues/**` paths/tests.

Implement table facets/rows/groups used by the default list, list/query required
by base detail, and UUID/identifier detail. Add `/api/config` capabilities and
prevent deferred consumers from mounting. Browser proof covers loading, empty,
denied, error and success with no unexpected legacy/deferred call.

### M1-S5 — Metadata through the real runtime

Allowed paths are accepted v9 metadata paths, Canonical composition, Core
metadata tests, and the read-only projection only when RED requires it. Prove
exact bodies/errors, UUID/identifier, auth ordering, isolation, rollback,
concurrency, restart persistence and browser readback.

### M1-S6 — Minimum realtime refresh

Allowed paths: new Canonical realtime boundary/publisher under `backend/**`,
Issue/metadata post-commit integration, `packages/core/api/ws-client.ts`,
`types/events.ts`, `realtime/**`, and accepted Issue cache tests.

Implement cookie/token handshake, Workspace authorization, four frozen events,
event schemas, commit-before-publish, duplicate tolerance and reconnect refetch.
M1 has no durable cursor. Failed transactions and foreign clients receive no
event.

### M1-S7 — Canonical-only startup and acceptance

Allowed paths: repository selectors/commands outside `server/**`, local setup
docs, `e2e/**`, and this plan directory. Freeze one workflow using Web 3000,
Canonical HTTP 8000, gRPC 9000 and `data/multica-canonical.db`. Prove clean and
retained startup, explicit fixture, browser journey, no legacy request/process,
restart/readback and non-destructive selector rollback.

## Deterministic gates

Each story runs focused RED/GREEN tests, then applicable repository scripts.
Final minimum gate covers Go format/policy/generated cleanliness, full backend
tests/vet/module verification, focused and broad Core/Views/Web checks, empty
and retained runtime, readiness failure, HTTP/realtime/browser/PID/port evidence,
rollback/readback, `git diff --check`, no `server/**`, and no unrelated path.

Windows wrapper or race-binary launch failures are indexed as limitations and
never claimed passing. Release acceptance needs an environment where mandatory
runtime/browser gates actually run.

## Promotion and milestone acceptance

After approval, `plan.md` advances one story at a time. Promotion requires
deterministic journal evidence, independent specification/code review and Human
Customer acceptance. Material contract/path/risk/rollback changes require
`plan_v3.md`; this file becomes immutable once execution starts.

Milestone acceptance requires the exact browser journey, rollback, no legacy
process/network call, no `server/**` diff, independent review and Human Customer
acceptance. It means only the frozen local Issue journey is Canonical-ready.

## Rollback and stop conditions

Keep legacy/Canonical selectors, DBs and logs until S7. Roll back only the
active story. Stop for body drift, required `server/**` write, identity/
isolation/atomicity/order loss, fabricated deferred success, production scope,
unapproved path expansion, material base/policy drift, or an out-of-scope hard
gate correction.
