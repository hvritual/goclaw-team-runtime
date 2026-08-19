# Product capability roadmap v31 — S07C Requirement coverage

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Plan-Version: `31`
- Task-Revision: `r036`
- Work-Item: `PCR-S07C`
- Exact base: `f5695de83d55e277c8eeb9db7461b81137dc93ad`
- Predecessor closure candidate: `cd94396093ea73f3f9434fed7410036ae61170ab`
- Predecessor closure tree: `b42fa34eb625c427f3ca06d001386da635881e9d`
- Status: `approved-active`
- Authority: the Human Customer's confirmed continuous Release 3 direction,
  confirmed prerequisite minimal outline authority, and confirmed execution

## Predecessor and activation boundary

Immutable v30/r035 closes both independent-review findings against S07B. Its
exact product candidate `cd94396093ea73f3f9434fed7410036ae61170ab`
passes the focused, complete Workspace, backend check, official race, scope,
trailer, dirty-tree, and fresh independent dual-review gates. Closure commit
`f5695de83d55e277c8eeb9db7461b81137dc93ad` records PCR-S07B as
`complete-independent-reviewed` with no active task.

The Customer's standing completion authority now activates only PCR-S07C from
that exact closure. PCR-S07D and Release 3 completion remain inactive. S10
outline hierarchy and progress, `project_outline`, generated protobufs, push,
merge, deployment, external service calls, the original dirty worktree, legacy
backend trees, and every `server/**` path remain excluded.

## Goal and source-of-truth boundary

Install one reviewer-facing, explainable Requirement coverage projection that
distinguishes no link, linked work, implemented work, and accepted work. The
projection is derived on read from immutable Requirement revisions, revisioned
Requirement-to-Issue link intervals, current canonical Issue status, and the
latest canonical Issue acceptance conclusion. It is never a stored coverage
cache and never treats an outline relation as Issue coverage.

The coverage universe contains only the traceable Requirement sections in this
fixed order: `goals`, `in_scope`, `constraints`, and
`acceptance_criteria`. Within each section, item order is the immutable content
order. `out_of_scope` and `dependencies` are context, not coverage units.

## Frozen coverage semantics

Each Requirement item has exactly one stage:

- `unlinked`: no current canonical Issue exists through a link interval active
  at the snapshot revision;
- `linked`: at least one Issue is linked, but at least one linked Issue has a
  current status other than `done`;
- `implemented`: at least one Issue is linked, every linked Issue currently has
  status `done`, and at least one linked Issue lacks a latest acceptance result
  of `accepted`;
- `accepted`: at least one Issue is linked, every linked Issue currently has
  status `done`, and every linked Issue's latest acceptance conclusion is
  `accepted`.

Multiple linked Issues aggregate fail closed: all must meet the next stage.
`cancelled`, `blocked`, reopened, or any other non-`done` Issue is not
implemented. A `done` Issue with no conclusion, a latest `conditional`, or a
latest `rejected` conclusion is implemented but not accepted. Conclusions are
ordered by `created_at DESC, id DESC`; a later non-accepted conclusion revokes
the accepted projection. A deleted/nonexistent current Issue cannot contribute
coverage even when an older retained link interval remains audit evidence.

Snapshot links are active when `linked_revision <= snapshot_revision` and
`unlinked_revision IS NULL OR unlinked_revision > snapshot_revision`.
`current` uses the baseline's current revision/content. `effective` uses the
baseline's effective revision/content when present. Both snapshots deliberately
use current Issue status and current latest acceptance conclusions. Requirement
link removal changes `current` immediately and changes `effective` only through
the already-frozen S07B effective-revision rules. Retired Requirements remain
readable: the current snapshot and top-level status show `retired`, while any
retained effective snapshot stays visible.

Snapshot counters are cumulative and must satisfy
`0 <= accepted <= implemented <= linked <= total` and
`unlinked = total - linked`. `linked` counts every item above `unlinked`;
`implemented` counts `implemented` plus `accepted`; `accepted` counts only the
terminal stage. Item stages remain exact, not cumulative.

## Frozen public contract

Add authenticated Workspace-member read route:

`GET /api/projects/{id}/requirement-baseline/coverage`

It returns HTTP 200 with this exact snake-case shape:

- `baseline_status`: one of the six S07B Requirement states, or `null` when no
  baseline exists;
- `current` and `effective`: a snapshot or `null`;
- snapshot: `revision`, `state`, `total`, `linked`, `implemented`, `accepted`,
  `unlinked`, and ordered `items`;
- item: `requirement_key`, `section`, `text`, exact `stage`, and ordered
  `issues`;
- issue: `id`, `identifier`, `title`, current `status`, and latest
  `acceptance_result` (`accepted`, `conditional`, `rejected`, or `null`).

Issues are ordered by identifier then stable ID. The no-baseline response is
`{"baseline_status":null,"current":null,"effective":null}`. Malformed or
inconsistent persisted Requirement content, missing effective revision, invalid
counter/stage construction, or query failure returns a safe typed internal
problem and never a partial/fallback projection.

Read authority exactly reuses S07B's current Workspace membership and project
existence boundary. Active ordinary members and members viewing completed or
cancelled projects may read; removed/nonmembers and cross-Workspace/project
requests preserve the existing fail-closed not-found mapping. The repository
uses one SQLite connection and a bounded consistent read transaction for
authority, baseline/revision, link, Issue, and latest-conclusion reads. The
query graph is constant with respect to item count: one authority/baseline
sequence and at most one linked-Issue projection query for each of current and
effective snapshots; no per-item or per-Issue query is allowed.

Core must parse this contract strictly with `parseOrThrow`; the inactive empty
fallback is removed. The shared view displays current/effective cumulative
counts and every traceable item with its exact stage and linked Issue evidence.
No client-inferred acceptance is allowed. Requirement mutations invalidate
baseline and coverage queries. Workspace Issue realtime events, including
status and acceptance-conclusion events, invalidate Workspace coverage queries;
the subsequent HTTP read remains authoritative.

`project_requirements` remains the only installation flag and stays true.
`project_outline` remains false. No new permission, route family, database
table/index, legacy write, or generated contract is introduced.

## Writable scope

Backend contract/application/repository/HTTP and tests:

- `backend/internal/modules/workspace/contract/project_requirement.go`;
- `backend/internal/modules/workspace/internal/application/project_requirement.go`;
- `backend/internal/modules/workspace/internal/application/project_requirement_test.go`;
- `backend/internal/modules/workspace/internal/infrastructure/sqlite/project_requirement_repository.go`;
- `backend/internal/modules/workspace/internal/infrastructure/sqlite/project_requirement_repository_test.go`;
- `backend/internal/modules/workspace/internal/interfaces/http/project_requirement.go`;
- `backend/internal/modules/workspace/internal/interfaces/http/project_requirement_test.go`;
- `backend/internal/modules/workspace/project_requirement_composition_test.go` only
  if the service-interface addition requires composition proof.

Strict Core/realtime contract and tests:

- `packages/core/types/project-requirement.ts`;
- `packages/core/types/index.ts`;
- `packages/core/project-requirements/schema.ts`;
- `packages/core/project-requirements/queries.ts`;
- `packages/core/project-requirements/queries.test.tsx`;
- `packages/core/api/client.ts`;
- `packages/core/api/project-requirement-schema.test.ts`;
- `packages/core/realtime/use-realtime-sync.ts`;
- `packages/core/realtime/use-realtime-sync-ws-instance.test.tsx`.

Shared view and locale acceptance:

- `packages/views/projects/components/project-requirement-baseline.tsx`;
- `packages/views/projects/components/project-requirement-baseline.test.tsx`;
- `packages/views/locales/en/projects.json`;
- `packages/views/locales/ja/projects.json`;
- `packages/views/locales/ko/projects.json`;
- `packages/views/locales/zh-Hans/projects.json`.

Governance:

- `backend/docs/plans/product-capability-roadmap/plan.md`;
- `backend/docs/plans/product-capability-roadmap/plan_v31.md`;
- `backend/docs/plans/product-capability-roadmap/story-map.md`;
- `backend/docs/plans/product-capability-roadmap/task-register.md`;
- `backend/docs/plans/product-capability-roadmap/journal.md`.

No migration, Issue production mutation path, capability flag, generated
protobuf, original dirty path, legacy backend tree, or `server/**` path is
writable. A necessary path outside this list stops r036 and requires an
immutable successor plan.

## Ordered execution

1. R36.1 — Freeze this plan from exact base `f5695de8`, move the isolated
   branch to `codex/release3-s07c-r036`, and commit only the five governance
   activation paths with one continuous nine-field trailer block.
2. R36.2 — RED the repository/service/HTTP contract for no baseline, all four
   exact stages, multi-Issue fail-closed aggregation, latest-conclusion
   precedence, current/effective revision intervals, unlink, retirement,
   deleted Issue exclusion, authorization, deterministic order, and bounded
   query count; then install only the smallest GREEN derived read path.
3. R36.3 — RED strict Core parsing, query invalidation/realtime refresh, and
   shared-view current/effective/item-stage behavior; then GREEN the exact
   schema, client, query keys, view, and four-locale contract without fallback
   or client-inferred acceptance.
4. R36.4 — Run focused backend/Core/Views/locale/realtime checks, complete
   Workspace tests, backend `make check`, the official changed-package
   `make test-race`, strict frontend type/lint/test gates, production Web build,
   and fresh installed acceptance against Canonical HTTP plus production Web.
5. R36.5 — Freeze one exact candidate; verify plan/policy hashes, all nine
   trailers on every r036 commit, exact path scope, zero `server/**` and
   generated paths, clean isolated worktree, original dirty-tree preservation,
   process cleanup, and obtain fresh independent `SPEC PASS` plus
   `CODE/SECURITY/QUALITY PASS`.

## Deterministic and installed acceptance

- Backend assertions begin RED against the absent route/service/repository and
  become GREEN for no links, open/mixed Issues, all-done Issues, accepted
  Issues, accepted-then-conditional precedence, link removal, current/effective
  divergence, retired Requirement visibility, deleted Issue exclusion,
  no-baseline response, ordering, authorization, and query-count bounds.
- Strict Core tests reject missing, unknown, mistyped, inconsistent, or partial
  fields and prove no empty fallback. Query tests prove Requirement mutations
  invalidate detail plus coverage; realtime tests prove an Issue event and a
  socket refresh window invalidate Workspace coverage.
- Shared-view tests prove counts and exact stages come only from server coverage,
  current/effective differences are labelled, all four traceable sections are
  displayed, terminal Requirement state stays visible, and locale parity holds.
- Focused packages, complete Workspace tests, backend `make check`, official
  race, changed frontend checks, production Web build, and repository-owned
  scripts pass. Any environment or unrelated aggregate failure remains NON-PASS
  and is disclosed rather than waived.
- Fresh SQLite installed acceptance uses authenticated reviewer authority, real
  Canonical HTTP, and the production Web app to prove the stage progression
  `unlinked -> linked -> implemented -> accepted`, a later conditional result
  revoking accepted coverage, multi-Issue fail-closed behavior, current versus
  effective projection, unlink, retirement visibility, realtime or reload
  refresh, and persistence after restart. Unavoidable membership setup fixtures
  are disclosed and are not counted as product behavior.
- Only a fresh independent review returning both required PASS decisions may
  close r036/PCR-S07C and authorize creation of the PCR-S07D successor plan.

## Explicit exclusions and stop conditions

PCR-S07D Retrospectives, Release 3 completion, S10 hierarchy/move/reorder/
numbering/archive/restore/Issue-outline/progress, project phases, Requirement-
driven Issue creation, stored coverage caches, historical Issue-status or
acceptance snapshots, new permissions/flags, migrations/indexes, generated
protobufs, unrelated Issue/Input behavior, push, merge, deployment, external
service calls, original dirty paths, legacy backend writes, and all `server/**`
changes are excluded.

Stop before closure on any client-inferred stage, any-link-equals-coverage
behavior, acceptance from a non-latest conclusion, non-`done` implemented
projection, per-item/per-Issue queries, partial/fallback response, authorization
leak, retired history loss, effective/current interval error, hidden test
failure, missing/duplicate trailer, scope drift, original dirty-path overlap,
unclosed process, `server/**` or generated change, or either independent-review
BLOCK decision. Any material repair outside this exact boundary requires a new
immutable plan; v31 is never amended after activation.
