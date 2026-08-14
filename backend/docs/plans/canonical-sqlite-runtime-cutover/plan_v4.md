# Canonical SQLite runtime cutover — execution plan v4

- Plan-ID: `canonical-sqlite-runtime-cutover`
- Version: `4`
- Status: `approved`
- Approval source: Human Customer confirmation dated `2026-08-14`
- Supersedes: `plan_v3.md`
- Base commit: `0c50ed4`
- Branch and integration target: `codex/multica-six-domain-baseline`
- Active step: `M1-S7-C1`
- Policy bundle: `backend-v1`

## Confirmed defect and reference behavior

After v3 technical acceptance, a real newly authenticated user with
`onboarded_at:null` reached the installed `/onboarding` flow. The flow called
the existing Core `POST /api/workspaces` boundary and received `404 Not
Found`; Canonical also lacked the subsequent
`POST /api/me/onboarding/complete` boundary. The installed page therefore
could not create its first Workspace or leave onboarding.

Read-only legacy SQLite evidence freezes the compatible behavior:

- authenticated `POST /api/workspaces` accepts `name`, `slug`, optional
  `description` and `context`, creates the Workspace and initial owner
  membership atomically, and returns the exact Workspace projection with 201;
- a duplicate slug returns 409 `workspace slug already exists`;
- authenticated `POST /api/me/onboarding/complete` accepts optional
  `completion_path` and `workspace_id`, rejects an unavailable membership,
  sets `onboarded_at` once, updates `updated_at`, and returns the exact User;
- retry is idempotent and persisted through restart.

## Scope and architecture

This approved correction is one strict-XP vertical story. It permits only:

- this plan directory and evidence journal;
- Canonical Auth public contract/application/SQLite/HTTP composition needed
  for trusted onboarding completion;
- Canonical Workspace public contract/application/SQLite/HTTP composition
  needed for first-Workspace creation;
- a narrow shared SQLite transaction executor/participant contract when
  necessary to keep Workspace and Auth-owned writes in one `BEGIN IMMEDIATE`
  transaction without another module writing the owner's tables directly;
- bootstrap wiring and focused runtime tests;
- Core response parsing/tests for the two existing API methods;
- the existing Canonical Playwright journey extended to cover a real new-user
  onboarding path.

The transaction coordinator owns begin/commit/rollback. Workspace-owned code
writes only `workspaces`; Auth-owned code writes only `auth_members` and
`auth_workspace_membership_roots`. Cross-module calls use a public contract;
no cross-module table joins, foreign keys, cascades or direct infrastructure
imports are allowed.

## Non-goals and invariants

- No onboarding questionnaire, invitation, general Workspace CRUD, member UI,
  production-profile or unrelated capability is added.
- `server/**` remains permanently read-only evidence.
- Bearer authentication has priority; Cookie mutations require the existing
  token-bound CSRF check. Identity is resolved before request/resource data.
- Request bodies are bounded, strict and reject trailing/unknown JSON.
- Slug/name validation, exact success shapes/statuses/errors, tenant isolation,
  generated IDs, timestamps and stable ordering remain compatible.
- Existing S0-S7 Issue, metadata, realtime, selector, restart and rollback
  evidence must remain green. Existing unrelated dirty paths remain excluded.

## Story acceptance

Given a new Canonical user with no membership, when the installed onboarding
page creates a Workspace and completes onboarding, then:

1. RED first proves both boundaries return 404 in the current runtime.
2. Workspace creation returns 201 with the exact 11-key Workspace body and an
   owner membership/root for the authenticated user in one transaction.
3. Duplicate/invalid input, missing or expired identity, Cookie-CSRF failure,
   ID-generation failure and forced persistence failure return the frozen
   status/error and leave no partial Workspace/member/root.
4. Completion rejects missing/foreign Workspace membership, returns the exact
   User body, preserves the first `onboarded_at` value on retry, and survives
   close/reopen.
5. Core rejects malformed success bodies without changing the installed UI
   contract.
6. A real browser performs email/code login, creates the first Workspace,
   completes onboarding, reaches its Workspace route, and reloads without
   returning to `/onboarding`.
7. Focused and full deterministic gates, no tracked/untracked `server/**`,
   independent specification/code review and Human Customer acceptance pass.

## Deterministic verification

From `backend/`:

```text
go test ./internal/modules/auth ./internal/modules/workspace ./internal/bootstrap -count=1
go test ./... -count=1
go vet ./...
go mod verify
```

From the repository root:

```text
pnpm --filter @multica/core test
pnpm --filter @multica/core typecheck
pnpm --filter @multica/core lint
pnpm --filter @multica/views typecheck
pnpm --filter @multica/web typecheck
node --test scripts/runtime-selector.test.mjs scripts/canonical-runtime-verifier.test.mjs
git diff --check
git status --porcelain -- server
```

The existing selector starts the same Canonical DB/Web topology for the final
installed-Chrome journey. Windows race binaries remain an environment-limited
gate when they exit `0xc0000139`; that result is never reported as passing.

## Risks and rollback

- Cross-module partial creation: prevent with one SQLite transaction and
  forced-failure rollback tests.
- Duplicate/concurrent slug creation: rely on the Workspace unique constraint,
  map conflict deterministically, and stress the focused path.
- Session/CSRF drift: reuse the S2 Auth resolver/authorizer; do not duplicate
  cookie or token logic.
- Rollback: stop Canonical, select the previous runtime, and revert this slice.
  Existing DB/logs remain retained; no destructive down migration or data
  deletion is required.
