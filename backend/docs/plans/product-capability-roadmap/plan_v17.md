# Product capability roadmap implementation plan v17

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Version: `17`
- Status: `approved-for-execution`
- Active step: `PCR-S03A Issue search`
- Task-ID: `PCR-001-S03A-R22`
- Task-Revision: `r022`
- Work-Item: `PCR-S03A`
- Base commit: `2439e9c2edd3c557e849fb695210b2eab95bbb9d`
- Supersedes: `plan_v16.md` for active execution only
- Approved: `2026-08-18`

## Outcome and authority

The Human Customer directed continued approved execution through Release 1.
Exact base `2439e9c` closes S02B with independent review and leaves no active
task. Three bounded read-only discovery rounds reached dependency closure for
PCR-S03A with no unresolved human-owned choice.

This version activates only the installed Issue-search vertical. S03B Project
search and S04 Pin reorder remain inactive. It does not authorize push, merge,
deployment, or Release 1 completion. `server/**` remains permanently read-only,
and all earlier plan versions remain immutable evidence.

## Frozen S03A behavior

1. Install `GET /api/issues/search` before the parameterized Issue route. It
   accepts `q`, `limit`, `offset`, and `include_closed`, returns snake-case
   `{issues,total}`, and requires trusted Workspace identity plus
   `workspace.search.readable` authorization before repository access.
2. A trimmed empty query is HTTP 400. `limit` defaults to 20, must be positive,
   and is capped at 50. `offset` defaults to zero and must be non-negative.
   `include_closed` defaults false; Issue statuses `done` and `cancelled` are
   closed.
3. Normalize query and indexed text with Unicode NFKC normalization, Unicode
   case folding, and punctuation/whitespace collapse. English and Chinese text
   require no network service. Match human identifier case-insensitively and a
   positive decimal query against Issue number.
4. Rank exact normalized identifier or exact number first, exact normalized
   title second, all normalized query terms in title third, and all terms in
   description fourth. Equal ranks order by `updated_at DESC, id ASC`.
   `total` is the matched count before pagination. Results identify
   `match_source` as `identifier`, `title`, or `description`; description hits
   may include a bounded plain-text snippet.
5. Migration `000013` adds an application-owned Issue search projection,
   Workspace/closed/rank support indexes, initial backfill, and insert/update/
   delete synchronization. The projection is rebuilt with the same
   deterministic normalizer at Canonical startup so pre-existing Unicode data
   receives the full contract rather than SQLite ASCII-only folding.
6. The repository performs filtering, ranking, counting, and pagination; the
   HTTP handler must not call `ListIssues` and filter the in-memory aggregate.
   Database work uses the request context so cancellation terminates the query.
7. `issue_search` becomes true only after this complete installed slice.
   `project_search` remains false. Authorization installation and per-feature
   installation evidence are separate so S03A cannot claim S03B.

## Writable scope

- `backend/internal/modules/workspace/contract/**` for hand-owned Issue-search
  and feature-installation contracts only; generated Issue service contracts
  remain unchanged;
- `backend/internal/modules/workspace/internal/application/**` Issue-search
  use case and tests;
- `backend/internal/modules/workspace/internal/infrastructure/sqlite/**` Issue
  search repository/projection, migration `000013`, synchronization points,
  and tests;
- `backend/internal/modules/workspace/internal/interfaces/http/**`,
  `backend/internal/modules/workspace/issue_read_extension.go`, and
  `backend/internal/modules/workspace/sqlite_workspace_chain.go` for the route
  and composition;
- `backend/internal/bootstrap/**` for exact installed authorization/feature
  declarations and runtime acceptance;
- `packages/core/api/**`, `packages/core/types/**`, and
  `packages/core/issues/**` for strict search boundary and cancellation tests;
- `packages/views/search/**` only if needed to gate or render the installed
  Issue result contract without changing unrelated shared UI;
- `e2e/**` for installed Web acceptance;
- current roadmap `plan.md`, `plan_v17.md`, `story-map.md`, `task-register.md`,
  and append-only `journal.md`.

Generated protobuf files, unrelated package files, lockfiles unless a direct
dependency classification changes, all pre-existing dirty paths, and every
`server/**` path are out of scope.

## Ordered execution

### PCR-S03A-R22.1 — Activate exact authority

- Freeze this plan at exact base `2439e9c` and establish r022 as the sole
  active task.
- Record policy hashes, excluded dirty paths, and empty `server/**` range and
  worktree diffs.

### PCR-S03A-R22.2 — RED contracts

- Add failing normalization/ranking/pagination/closed/workspace/cancellation
  repository and application tests, including Chinese and English fixtures.
- Add failing HTTP tests for validation, identity, authorization-before-read,
  snake-case shape, and route precedence.
- Add failing runtime tests proving `issue_search=true` while
  `project_search=false`, strict Core response parsing, AbortSignal forwarding,
  and shared search rendering.

### PCR-S03A-R22.3 — GREEN installed vertical

- Add migration `000013`, deterministic Unicode normalizer, projection rebuild
  and atomic trigger synchronization, repository ranking/count/pagination, and
  application authorization.
- Compose the dedicated search service into the existing Issue HTTP extension,
  install the exact feature signal, and preserve all list/query behavior.
- Tighten the existing Core search response boundary and make the shared Web/
  Desktop search surface honor the installed Issue capability without enabling
  Project search.

### PCR-S03A-R22.4 — Verify and close

- Run focused backend/Core/Views tests, migration restart/idempotency checks,
  a 10,000-Issue local performance acceptance against the frozen p50 100ms and
  p95 250ms budgets, root typecheck/test, backend check and real race.
- Run a new-database installed-Chrome journey covering English, Chinese,
  identifier/number, default closed exclusion, explicit closed inclusion,
  pagination evidence, and Workspace isolation. Resolve and stop exact process
  ancestry afterward.
- Verify exact scope, policy hashes, excluded dirty blobs, and empty
  `server/**`; then obtain fresh independent read-only review. Only a complete
  PASS may close r022 and S03A. S03B requires a new plan version.

## Acceptance criteria

1. All frozen ranking, normalization, pagination, closed filtering, Workspace
   isolation, authorization, cancellation, and response-shape scenarios pass.
2. Search is repository-backed, survives database reopen, remains synchronized
   after create/update/batch-update/promotion/delete, and returns at most 50
   rows with stable ID tie-breaking.
3. A 10,000-Issue acceptance satisfies p50 no greater than 100ms and p95 no
   greater than 250ms on the recorded local environment.
4. Core rejects malformed Issue search payloads, forwards cancellation, and the
   installed shared Web/Desktop surface consumes only the canonical route.
5. `/api/config` reports `issue_search=true` and `project_search=false`; no
   later story is implied complete.
6. Full deterministic gates, installed browser acceptance, clean scope/process
   checks, and fresh independent review pass without waiver.

## Risks and rollback

The projection adds write-time work and its normalization function is process
owned. Startup rebuild makes retained rows deterministic and tests prove every
known Issue title/description write path. The bounded 10,000-row performance
gate catches regressions before closure. Rollback disables the Issue feature
signal and route, removes only S03A code, and applies the empty-only 000013 down
migration after confirming the projection is derived data. It must not alter
source Issues, user work, `server/**`, or any older plan.
