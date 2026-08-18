# Product capability roadmap implementation plan v19

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Version: `19`
- Status: `approved-for-execution`
- Active step: `PCR-S04 Pin reorder`
- Task-ID: `PCR-001-S04-R24`
- Task-Revision: `r024`
- Work-Item: `PCR-S04`
- Base commit: `b86447909e1a7614c539769514008e010b478140`
- Supersedes: `plan_v18.md` for active execution only
- Approved: `2026-08-18`

## Outcome and authority

The Human Customer directed continued approved execution through Release 1.
Exact base `b864479` closes S03B with independent review and leaves no active
task. Bounded read-only discovery reached dependency closure for PCR-S04 with
no unresolved human-owned choice.

This version activates only the installed Pin-reorder vertical. It does not
authorize push, merge, deployment, any Release 2 story, or Release 1
completion before S04 closes. `server/**` remains permanently read-only, and
all earlier plan versions remain immutable evidence.

## Frozen S04 behavior

1. Install `PUT /api/pins/reorder` before the parameterized Pin delete route.
   The strict snake-case request is
   `{items:[{id}],expected_revision}`. `items` is the complete ordered set of
   the authenticated user's current Pin IDs for the trusted Workspace;
   positions are server-derived as contiguous integers beginning at one.
2. `items` must be non-empty, every ID must be non-empty and unique, and
   `expected_revision` must be a positive integer. Duplicate, missing,
   additional, other-user, or foreign-Workspace Pin IDs are HTTP 400 and cause
   no mutation. Authentication, trusted identity, CSRF, and
   `workspace.pin.reorder` authorization precede repository access.
3. Pin order is revisioned per `(workspace_id,user_id)`. Existing non-empty Pin
   sets start at revision one. Pin create, effective delete, and reorder each
   advance the same monotonic revision in their existing transaction. Every
   listed or created Pin includes the current positive `order_revision`; the
   existing array response for `GET /api/pins` remains compatible.
4. A stale `expected_revision` returns HTTP 409 with
   `{code:"revision_conflict",current_revision,error:"revision conflict"}` and
   no position or revision change. Successful reorder updates the whole set
   atomically, advances the revision once, and returns HTTP 204.
5. Migration `000015` adds only the application-owned Pin-order revision table
   and backfills revision one for existing Pin sets. It adds no foreign key or
   cascade. The down migration refuses after an advanced revision would be
   discarded; source Pins and their positions are never dropped or rewritten.
6. The repository uses one `BEGIN IMMEDIATE` transaction to read current
   revision and complete Pin ownership, validate the exact set, update every
   position, advance the revision, and commit. Cancellation and every failure
   roll back the complete operation. Reopen preserves order and revision.
7. The strict Core boundary requires `order_revision` on installed Pin
   responses and sends only the ordered IDs plus the current expected
   revision. The mutation optimistically reorders the Workspace/user-scoped
   cache, restores the exact previous list on every error, and refetches after
   success or conflict.
8. Shared Web/Desktop sidebar drag affordances activate only after loaded
   explicit `pin_reorder=true` evidence. Pins remain visible and can still be
   opened or unpinned while reorder is unavailable. Runtime retains
   `issue_search=true` and `project_search=true`; `pin_reorder` becomes true
   only after this complete installed slice.

## Writable scope

- `backend/internal/modules/workspace/contract/**` for hand-owned Pin reorder
  request/revision contracts only; generated contracts remain unchanged;
- `backend/internal/modules/workspace/internal/application/**` for Pin order
  validation, authorization, and tests;
- `backend/internal/modules/workspace/internal/infrastructure/sqlite/**` for
  atomic Pin create/delete/reorder revision maintenance, migration `000015`,
  restart/rollback tests, and no other Project behavior;
- `backend/internal/modules/workspace/internal/interfaces/http/**` and
  `backend/internal/bootstrap/**` for route installation, exact error mapping,
  explicit capability reporting, and runtime acceptance;
- `packages/core/api/**`, `packages/core/types/**`, and `packages/core/pins/**`
  for strict response/request schemas, scoped optimistic rollback, refetch,
  and tests;
- `packages/views/layout/app-sidebar.tsx` and direct sidebar tests only for
  explicit loaded feature gating and drag behavior;
- `e2e/**` for installed Web acceptance;
- current roadmap `plan.md`, `plan_v19.md`, `story-map.md`, `task-register.md`,
  and append-only `journal.md`.

Generated protobuf files, unrelated package files, lockfiles unless a direct
dependency classification changes, all pre-existing dirty paths, and every
`server/**` path are out of scope.

## Ordered execution

### PCR-S04-R24.1 — Activate exact authority

- Freeze this plan at exact base `b864479` and establish r024 as the sole
  active task.
- Record policy hashes, excluded dirty paths, protected Input blob, and empty
  `server/**` range and worktree diffs.

### PCR-S04-R24.2 — RED contracts

- Add failing application/repository/HTTP tests for exact-set ordering,
  duplicate/missing/additional/foreign IDs, trusted identity, authorization,
  stale revision, rollback, cancellation, contiguous positions, and reopen.
- Add failing migration tests for existing-set backfill, monotonic create/
  delete maintenance, and guarded rollback.
- Add failing strict Core and Views tests for revision parsing, exact request
  shape, Workspace/user cache isolation, optimistic rollback/refetch, and
  loaded explicit drag gating.

### PCR-S04-R24.3 — GREEN installed vertical

- Add migration `000015`, collection revision maintenance, atomic repository
  reorder, application validation, HTTP route/error mapping, and explicit
  installed authorization/feature declarations.
- Tighten the Core Pin boundary and mutation lifecycle, and gate only the drag
  affordance while preserving list/open/unpin behavior.
- Preserve Task reorder, Project CRUD/search, Issue search, and every later
  roadmap feature unchanged.

### PCR-S04-R24.4 — Verify and close

- Run focused backend/Core/Views tests, migration/reopen/rollback checks, root
  typecheck/test, backend check, and the official real race suite.
- Run a new-database installed-Chrome journey that creates Issue and Project
  Pins, reorders the complete set through the sidebar, reloads to prove
  persistence, and submits a stale concurrent API reorder to prove 409/no
  mutation and UI rollback. Resolve and stop exact process ancestry afterward.
- Verify exact scope, policy hashes, excluded dirty blobs, and empty
  `server/**`; then obtain fresh independent read-only review. Only a complete
  PASS may close r024, S04, and Release 1.

## Acceptance criteria

1. Complete-order, duplicate, missing, additional, foreign Workspace/user,
   authentication, identity, authorization, stale revision, cancellation, and
   exact error-shape scenarios pass without partial mutation.
2. Create, effective delete, and reorder maintain one monotonic Pin-order
   revision; successful reorder writes contiguous positions and survives
   database reopen.
3. Core rejects malformed installed Pin/revision data, sends the frozen exact
   request, scopes optimistic state by Workspace/user, rolls back every error,
   and refetches authoritative order.
4. Shared Web/Desktop exposes drag reorder only with loaded explicit
   installation evidence while preserving non-reorder Pin behavior.
5. `/api/config` reports `issue_search=true`, `project_search=true`, and
   `pin_reorder=true`; no Release 2 capability is implied complete.
6. Full deterministic gates, installed browser acceptance, clean scope/process
   checks, and fresh independent review pass without waiver.

## Risks and rollback

The primary risks are partial multi-row writes, stale clients overwriting a
newer order, collection-revision drift during create/delete, and optimistic UI
state leaking across Workspaces. One immediate transaction, exact-set checks,
monotonic revision maintenance, scoped cache keys, and explicit rollback tests
address those risks. Rollback disables only the Pin-reorder feature and route;
the guarded `000015` down migration may run only when it cannot discard an
advanced concurrency token. It must never alter Pin rows or positions, S03A,
S03B, user work, `server/**`, or any older plan.
