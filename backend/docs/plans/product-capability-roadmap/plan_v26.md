# Product capability roadmap v26 — S07A governed project Resources

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Plan-Version: `26`
- Task-Revision: `r031`
- Work-Item: `PCR-S07A`
- Exact base: `80d92b14b1f5a1525fbce0c60ce992e28c7f0e8b`
- Status: `approved-active`
- Authority: Human Customer confirmation on 2026-08-19 to execute the
  reviewed Release 3 plan continuously, beginning with S07A

## Outcome

Deliver one installed Workspace-owned Project Resource vertical for GitHub
repositories and generic public URLs. Authorized members can list Resources;
Workspace owners/admins and the Project's current member lead can add, update,
reorder, archive, restore, and refresh connection state. The vertical preserves
external targets and credentials, survives restart, projects exact counts, and
remains disabled until all frozen gates pass.

## Frozen product contract

1. Resource types are `github_repo` and `url`. Both persist a canonical public
   `https` URL, an optional display label, a contiguous server-owned position,
   an item revision, active/archived state, actor/timestamp metadata, and a
   connection projection. GitHub Resources may additionally retain a branch or
   tag name. Unknown response types remain safely displayable by clients.
2. URLs are normalized locally before persistence. GitHub SSH/SCP/HTTP inputs
   normalize to credential-free `https://github.com/{owner}/{repository}`.
   Generic URLs require `https` and persist scheme, host, port, and path only;
   userinfo, query, fragment, control characters, non-public host forms, and
   embedded credential material are rejected. Duplicate type plus canonical
   URL plus ref is rejected within one active or archived Project.
3. External reachability never decides whether a locally valid Resource can be
   saved. Refresh uses an injected typed connection adapter, never a generic
   server-side URL fetch. The installed no-connection adapter reports
   `unavailable` with a stable diagnostic. Adapter errors retain the Resource
   and project `degraded` or `unavailable` with checked time and a safe code;
   OAuth tokens, headers, raw provider bodies, and secret-bearing URLs are never
   persisted, logged, audited, or returned.
4. Routes remain exactly:
   - `GET/POST /api/projects/{id}/resources`;
   - `PUT/DELETE /api/projects/{id}/resources/{resource_id}`.
   GET accepts `include_archived` and returns `resources`, `total`, and the
   Project Resource-set `revision`. POST requires `Idempotency-Key`. PUT carries
   one action (`update`, `reorder`, `restore`, or `refresh`) plus
   `expected_revision`. Reorder identifies the moved Resource and its
   `before_resource_id` or end position and atomically rewrites contiguous
   positions. DELETE archives the relation with `expected_revision`; it never
   calls or deletes the external target.
5. Every successful mutation increments the Resource-set revision exactly once,
   writes immutable audit and idempotency evidence where required, and returns
   the authoritative projection. A stale revision returns `409
   revision_conflict` with no mutation. Replayed POST returns the original
   response; key reuse with a different canonical request conflicts.
6. Resource read requires Workspace membership. Resource management requires
   owner/admin or the Project's current `lead_type=member` and matching actor ID.
   Missing actor, missing provider, foreign Workspace/Project, archived Project,
   non-member lead, and agent default all deny before repository or adapter
   work. Clients never supply trusted actor identity.
7. `CreateProjectRequest.resources` is accepted by the Canonical backend and
   creates the Project plus all locally validated Resources atomically. Any
   invalid or duplicate Resource rolls the full create back. Project deletion
   removes only local Resource relations in the existing application
   transaction and never invokes the connection adapter.
8. Active `resource_count` is projected from the authoritative Resource
   repository for Project list/detail/search. Archived Resources do not count.
   The `project_resources` flag remains false until the complete installed
   vertical passes; unsupported routes return explicit unavailable errors.
9. TypeScript network responses pass Zod schemas. Mutations have no success
   fallback; malformed-response tests cover list and every mutation. Shared UI
   renders loading, empty, denied, error, archived, connection, reorder, restore,
   unknown-type, and long-text states without direct platform imports.

## Data and migration contract

- Add Workspace-owned Resource-set and Resource authority tables through the
  next additive SQLite migration. Do not add foreign keys or cascades.
- Project and Workspace membership are validated in application code. Project
  deletion performs dependent cleanup explicitly in one transaction.
- Down migration succeeds only when every new authority, audit/idempotency
  reference, and Resource row is empty; otherwise it leaves schema and data
  unchanged.
- No generated protobuf change is required or authorized.

## Writable scope

- `backend/internal/modules/workspace/contract/**` for exact Resource contracts;
- `backend/internal/modules/workspace/internal/{domain,application,infrastructure,interfaces}/**`
  for Resource behavior, SQLite migration/repository, and HTTP handlers/tests;
- exact Workspace composition, runtime capability, Project projection/create/
  delete, and tests under `backend/internal/modules/workspace/**` and
  `backend/internal/bootstrap/**`;
- `packages/core/{api,projects,types}/**` and exact tests;
- `packages/views/projects/**`, `packages/views/modals/create-project*`,
  Resource locale keys, and exact tests;
- `e2e/project-resources.spec.ts`;
- current roadmap pointer, story map, task register, journal, and this immutable
  plan.

Existing generated protobufs, unrelated Input/Issue UI, Desktop/Mobile,
`server/**`, push, merge, deployment, S07B-D, and Release 3 completion are
excluded. The original worktree's tracked and untracked changes remain
byte-for-byte outside the task worktree.

## Ordered execution

1. R31.1 — Activate exact authority and record immutable hashes, base, branch,
   exclusions, and zero pre-existing task-worktree diff.
2. R31.2 — RED domain/repository/HTTP contracts for normalization, duplicate,
   revision, reorder, archive/restore, connection failure, authorization,
   rollback, cleanup, count, and restart.
3. R31.3 — GREEN the minimal installed Canonical backend and keep the capability
   false until the complete slice is composed.
4. R31.4 — RED/GREEN strict Core schemas and shared Web/Desktop-compatible UI,
   including generic URL and connection/archived states.
5. R31.5 — Run focused and broad deterministic checks plus fresh installed
   Chrome acceptance against a fresh Canonical SQLite runtime.
6. R31.6 — Verify hashes, trailers, exact scope, empty `server/**` diff, original
   dirty-path preservation, process cleanup, and obtain fresh independent SPEC
   plus CODE/SECURITY/QUALITY review on the exact candidate.

## Acceptance and deterministic verification

- Table-driven normalization rejects secrets, invalid schemes/hosts, malformed
  GitHub identifiers, duplicates, and foreign Workspace/Project operations.
- Concurrency proves one reorder/archive/update winner and one stale conflict;
  failed/replayed operations produce no duplicate mutation, audit, or adapter
  call.
- Fresh SQLite proves migration, restart persistence, count projection,
  Project-create atomic rollback, explicit Project-delete cleanup, and guarded
  down migration.
- Focused backend/Core/Views tests pass, followed by `backend make check`,
  official `make test-race`, root typecheck/test, and production Web build.
- Installed Chrome with retries disabled uses the real Canonical HTTP backend:
  owner creates GitHub and generic URL Resources, member reads, Project lead
  manages, stale reorder conflicts, unavailable refresh remains saved,
  archive/restore survives reload, and Project deletion causes no external
  adapter delete.
- Only a fresh independent `SPEC PASS` and `CODE/SECURITY/QUALITY PASS` may
  close r031/S07A. A broad aggregate failure remains non-PASS unless the frozen
  evidence classifies it as unrelated and the exact focused check passes.

## Rollback and stop conditions

Rollback disables `project_resources`, removes exact Resource composition and
UI exposure, and runs the guarded down migration only when authority tables are
empty. Stop before closure on credential persistence or echo, SSRF-capable
generic fetch, external-target deletion, missing Project-lead server check,
non-atomic Project create/delete behavior, stale-write success, incorrect active
count, hidden malformed response, `server/**` change, dirty-path overlap, scope
drift, or independent review BLOCK. Any material repair requires a new immutable
plan version; do not amend v26 or silently retry a consumed gate.
