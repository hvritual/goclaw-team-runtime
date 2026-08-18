# Product capability roadmap implementation plan v18

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Version: `18`
- Status: `approved-for-execution`
- Active step: `PCR-S03B Project search`
- Task-ID: `PCR-001-S03B-R23`
- Task-Revision: `r023`
- Work-Item: `PCR-S03B`
- Base commit: `781471015ec8d759cd1209fd051e59fa91507eef`
- Supersedes: `plan_v17.md` for active execution only
- Approved: `2026-08-18`

## Outcome and authority

The Human Customer directed continued approved execution through Release 1.
Exact base `7814710` closes S03A with independent review and leaves no active
task. Bounded read-only discovery reached dependency closure for PCR-S03B with
no unresolved human-owned choice.

This version activates only the installed Project-search vertical. S04 Pin
reorder remains inactive. It does not authorize push, merge, deployment, or
Release 1 completion. `server/**` remains permanently read-only, and all
earlier plan versions remain immutable evidence.

## Frozen S03B behavior

1. Install `GET /api/projects/search` before the parameterized Project route.
   It accepts `q`, `limit`, `offset`, and `include_closed`, returns strict
   snake-case `{projects,total}`, and requires trusted Workspace identity plus
   `workspace.search.readable` authorization before repository access.
2. A trimmed empty query is HTTP 400. `limit` defaults to 20, must be positive,
   and is capped at 50. `offset` defaults to zero and must be non-negative.
   `include_closed` defaults false; Project statuses `completed` and
   `cancelled` are closed.
3. Normalize query and indexed text with the S03A Unicode NFKC normalization,
   Unicode case folding, and punctuation/whitespace collapse contract. English
   and Chinese text require no network service.
4. Rank exact normalized title first, all normalized query terms in title
   second, and all terms in description third. Equal ranks order by
   `updated_at DESC, id ASC`. `total` is the matched count before pagination.
   Results identify `match_source` strictly as `title` or `description`;
   description hits may include a bounded plain-text `matched_snippet`.
5. Migration `000014` adds an application-owned Project search projection,
   Workspace/closed/rank support indexes, initial backfill, and insert/update/
   delete synchronization. The projection is rebuilt with the same
   deterministic normalizer at Canonical startup so retained Unicode data
   receives the complete contract.
6. The installed Project surface repository performs filtering, ranking,
   counting, and pagination and returns the existing public Project shape with
   counts. The HTTP handler must not list and filter Projects in memory.
   Database work uses the request context so cancellation terminates the query.
7. The strict Core client forwards `AbortSignal` and rejects malformed top-
   level, Project, count, and `match_source` data. The shared Web/Desktop search
   surface calls Project search only after loaded explicit installation
   evidence; existing context mentions remain compatible with the installed
   route.
8. `issue_search` remains true and `project_search` becomes true only after
   this complete installed slice. `pin_reorder` remains false. Authorization
   installation and per-feature installation evidence remain separate.

## Writable scope

- `backend/internal/modules/workspace/contract/**` for hand-owned Project
  surface search contracts only; generated Project service contracts remain
  unchanged;
- `backend/internal/modules/workspace/internal/application/**` Project surface
  search use case and tests;
- `backend/internal/modules/workspace/internal/infrastructure/sqlite/**`
  Project search repository/projection, migration `000014`, deterministic
  normalization reuse, and tests;
- `backend/internal/modules/workspace/internal/interfaces/http/**`,
  `backend/internal/modules/workspace/project_surface.go`, and
  `backend/internal/modules/workspace/sqlite_workspace_chain.go` for the route
  and composition;
- `backend/internal/bootstrap/**` for exact installed authorization/feature
  declarations and runtime acceptance;
- `packages/core/api/**` and `packages/core/types/**` for strict Project search
  response and cancellation contracts;
- `packages/views/search/**` and
  `packages/views/editor/extensions/mention-suggestion*` only for installed
  feature gating, Project-result rendering, and direct tests;
- `e2e/**` for installed Web acceptance;
- current roadmap `plan.md`, `plan_v18.md`, `story-map.md`, `task-register.md`,
  and append-only `journal.md`.

Generated protobuf files, unrelated package files, lockfiles unless a direct
dependency classification changes, all pre-existing dirty paths, and every
`server/**` path are out of scope.

## Ordered execution

### PCR-S03B-R23.1 — Activate exact authority

- Freeze this plan at exact base `7814710` and establish r023 as the sole
  active task.
- Record policy hashes, excluded dirty paths, and empty `server/**` range and
  worktree diffs.

### PCR-S03B-R23.2 — RED contracts

- Add failing normalization/ranking/pagination/closed/workspace/cancellation
  repository and application tests, including Chinese and English fixtures.
- Add failing HTTP tests for validation, identity, authorization-before-read,
  exact snake-case shape, and route precedence.
- Add failing runtime tests proving both search flags true while Pin reorder
  remains false, strict Core response parsing, AbortSignal forwarding, and
  shared Project-result rendering.

### PCR-S03B-R23.3 — GREEN installed vertical

- Add migration `000014`, deterministic Unicode projection rebuild and trigger
  synchronization, repository ranking/count/pagination, and shared-readable
  authorization to the Project surface.
- Install the route before `/{id}`, compose the exact feature signal, and
  preserve Project list/CRUD, Proto lifecycle, Issue search, and pin behavior.
- Tighten the Core boundary and preserve explicit loaded feature gating across
  shared Web/Desktop search consumption.

### PCR-S03B-R23.4 — Verify and close

- Run focused backend/Core/Views tests, migration restart/idempotency checks,
  a 2,000-Project local performance acceptance against the frozen p50 100ms
  and p95 250ms budgets, root typecheck/test, backend check and real race.
- Run a new-database installed-Chrome journey covering English, Chinese,
  default closed exclusion, explicit closed inclusion, stable pagination, and
  Workspace isolation. Resolve and stop exact process ancestry afterward.
- Verify exact scope, policy hashes, excluded dirty blobs, and empty
  `server/**`; then obtain fresh independent read-only review. Only a complete
  PASS may close r023 and S03B. S04 requires a new plan version.

## Acceptance criteria

1. All frozen ranking, normalization, pagination, closed filtering, Workspace
   isolation, authorization, cancellation, and exact response-shape scenarios
   pass.
2. Search is repository-backed, survives database reopen, remains synchronized
   after both Project lifecycle and installed Project-surface create/update/
   delete paths, and returns at most 50 rows with stable ID tie-breaking.
3. A 2,000-Project acceptance satisfies p50 no greater than 100ms and p95 no
   greater than 250ms on the recorded local environment.
4. Core rejects malformed Project search payloads, forwards cancellation, and
   installed shared Web/Desktop consumers use only the canonical route.
5. `/api/config` reports `issue_search=true`, `project_search=true`, and
   `pin_reorder=false`; no later story is implied complete.
6. Full deterministic gates, installed browser acceptance, clean scope/process
   checks, and fresh independent review pass without waiver.

## Risks and rollback

The projection adds write-time work and depends on the process-owned Unicode
normalizer. Startup rebuild and trigger tests cover both existing Project write
stacks. Returning the public Project surface rather than the older Proto-shaped
`name` model prevents a hidden client contract fork. Rollback disables only the
Project feature signal and route, removes S03B code, and applies the empty-only
000014 down migration after confirming the projection is derived data. It must
not alter source Projects, S03A, pins, user work, `server/**`, or any older plan.
