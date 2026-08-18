# Product capability roadmap implementation plan v22

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Version: `22`
- Status: `approved-for-execution`
- Active step: `PCR-S06A Query Knowledge`
- Task-ID: `PCR-001-S06A-R27`
- Task-Revision: `r027`
- Work-Item: `PCR-S06A`
- Base commit: `78c5b5eb8b04d27d1fdc3c524955dd02fd756348`
- Supersedes: `plan_v21.md` for active execution only
- Approved: `2026-08-18`

## Outcome and authority

The Human Customer approved continuous execution through Release 2. Exact base
`78c5b5eb` closes PCR-S05B with final independent SPEC and code/security/quality
PASS and leaves no active task. Three bounded read-only discovery rounds reached
dependency closure with `new_dependencies=0` and no unresolved human decision:

1. the frozen story requires status, kind, source, applicability, and revision
   filtering, stable pagination/ranking, provenance projection, superseded
   reads, Workspace isolation, and explicit quarantine visibility;
2. Canonical Workspace has only an uninstalled create/get scaffold whose row
   shape cannot represent governed revisions or sources, while shared Core and
   Views contain permissive future-facing Knowledge shapes and prematurely
   visible S06B controls;
3. read-only `server/**` evidence supplies useful vocabulary but only a
   published-only offset search, so it is migration evidence rather than an
   implementation or acceptance authority.

This version activates only PCR-S06A. It does not authorize proposal, review,
publication mutations, realtime Knowledge events, S06B, push, merge, deployment,
or Release 2 completion. `server/**` remains permanently read-only.

## Frozen S06A behavior

1. Workspace owns governed Knowledge entries, immutable numbered revisions, and
   normalized source references. The existing `workspace_knowledge` create/get
   scaffold remains compatibility evidence and is not silently promoted into
   governed Knowledge because it lacks provenance. New storage is additive and
   down migration is empty-only guarded.
2. Install authenticated `GET /api/knowledge` and `GET /api/knowledge/{id}`.
   Workspace identity comes only from the trusted request context. Both routes
   authorize `workspace.knowledge.query` before repository access and fail
   closed for missing/foreign identities, malformed filters, and malformed
   cursors.
3. List accepts trimmed `query`, repeated or comma-separated `status` and
   `kind`, exact `source_type`, `source_id`, and optional `source_revision`,
   `applicability=workspace|project`, exact `project_id`, exact positive
   `revision`, `limit` default 20/max 100, and an opaque `cursor`. A source ID
   requires a source type; project applicability requires a project ID and
   workspace applicability rejects one.
4. Ordinary members may query `published` and `superseded` only. Owner/admin may
   additionally request `quarantined`; quarantine is never part of the default
   result set. Candidate, in-review, rejected, and invalidated records remain
   outside S06A query results until S06B installs their governed lifecycle.
5. No explicit status means `published`. Status filters are canonicalized and
   deduplicated. Kind values are exactly goal, decision, constraint, requirement,
   procedure, lesson, and reference. Applicability is Workspace-wide when
   `project_id` is absent and exact-project when present.
6. Text normalization is Unicode NFC plus case folding and collapsed whitespace.
   Deterministic rank buckets are exact title, title prefix, title contains,
   content contains, and source citation contains. Ties order by updated time
   descending then entry ID ascending. Empty query orders by updated time then
   ID. Ranking is provider-portable and does not depend on SQLite FTS scores.
7. Cursor pagination is keyset-based. The signed opaque cursor binds the complete
   canonical filter fingerprint and last rank/time/ID tuple; malformed,
   tampered, expired, or cross-filter reuse returns 400. Results never duplicate
   or skip rows under a stable snapshot, and cancellation is propagated.
8. Each list result projects only the selected immutable revision, status, kind,
   applicability, ordered source references, citation, rank reason, timestamps,
   and current revision number. `revision=N` selects that visible historical
   revision; without it the current revision is projected. Source filtering is
   evaluated against the selected revision, never another revision.
9. Detail returns the visible entry plus all immutable revisions and their
   ordered sources. Published and superseded history remains readable; a member
   cannot infer a quarantined entry through detail, counts, cursor, or errors.
   Unknown and non-visible IDs both return 404.
10. Source references carry type, source ID, immutable source revision, citation,
    and optional Space asset/version identity. Query is a projection only: S06A
    does not fetch bodies from Space and does not read Control Plane tables.
11. Core uses strict non-coercing schemas and typed filter/query keys containing
    every canonical filter and cursor. Malformed success responses throw; no
    empty fallback is synthesized. Shared Web/Desktop renders loaded, denied,
    error, empty, result, provenance, superseded, quarantined-admin, pagination,
    and detail states.
12. Knowledge UI and requests mount only after config has loaded with
    `knowledge_query=true`. Proposal, revision proposal, review queue, and all
    mutations remain hidden and unmounted while `knowledge_review=false`.
    Source deep links become server-side exact filters rather than client-only
    filtering.
13. `/api/config` retains every earlier installed flag and sets only
    `knowledge_query=true` for this story. `knowledge_review=false` until S06B.

## Writable scope

- `backend/internal/modules/workspace/**` for additive governed Knowledge domain,
  application, SQLite storage/migration, authorization, cursor, and HTTP query;
- `backend/internal/bootstrap/**` and `backend/cmd/server/**` for installed
  composition, cursor configuration, flag declaration, and runtime tests;
- `packages/core/knowledge/**`, `packages/core/types/knowledge.ts`, and narrow
  config/API exports for strict query contracts and filter-aware caches;
- `packages/views/knowledge/**`, Knowledge locale files, and narrow Web/Desktop
  route/navigation integration for the installed query-only surface;
- `e2e/**` for fresh-runtime installed acceptance;
- current roadmap `plan.md`, this immutable `plan_v22.md`, `story-map.md`,
  `task-register.md`, and append-only `journal.md`.

Generated protobuf, S06B mutations/realtime, Control Plane tables, Space body
reads, unrelated UI, dependency manifests/lockfiles, every recorded dirty path,
and all `server/**` paths are excluded. Protected
`packages/ui/components/ui/input.tsx` must remain blob
`a830fd2f0f82770563908d512558fe6ba48f50dd`.

## Ordered execution

### PCR-S06A-R27.1 — Activate exact authority

- Freeze v22 from exact base `78c5b5eb` and establish r027 as the sole active
  task with policy hashes, dirty exclusions, protected blobs, and empty
  `server/**` diffs recorded.

### PCR-S06A-R27.2 — RED query contracts

- Add failing domain/repository/HTTP/runtime tests for immutable revisions,
  source projection, every filter validation, member/admin visibility,
  Workspace isolation, stable rank, keyset cursor binding/tamper, cancellation,
  and restart persistence.
- Add failing Core/Views tests for strict parsing, complete query keys, loaded
  feature gating, query-only UI, source deep links, pagination, provenance,
  superseded/quarantine labels, and hidden S06B controls.

### PCR-S06A-R27.3 — GREEN installed vertical

- Implement additive Workspace storage and the hand-owned query/detail contract,
  deterministic ranking/cursor, trusted HTTP adapter, installed capability, and
  strict shared client/query-only UI required by the RED contracts.

### PCR-S06A-R27.4 — Verify and close

- Run focused adversarial/reopen/cancellation tests, root typecheck/test, backend
  check, official race, and production Web build.
- Run fresh-database installed-Chrome acceptance proving every filter, stable
  next page, provenance/detail, superseded visibility, admin-only quarantine,
  member non-disclosure, Workspace isolation, malformed cursor rejection,
  restart persistence, and hidden S06B controls.
- Verify scope, hashes, dirty exclusions, process cleanup, trailers, and empty
  `server/**`; obtain fresh independent SPEC and code-quality review. Only a
  complete PASS may close r027 and PCR-S06A.

## Acceptance criteria

1. An installed member can strictly query and inspect published/superseded
   Knowledge by every frozen filter with deterministic ranking and pagination.
2. Provenance and revision projections are exact, immutable, Workspace-scoped,
   and explain trust without exposing Space bodies or non-visible records.
3. Quarantine is excluded by default and invisible to members; explicit
   owner/admin query is isolated and labeled. Malformed/tampered/cross-filter
   cursors fail closed.
4. Only `knowledge_query` becomes true. All S06B APIs, controls, requests,
   realtime behavior, and `knowledge_review` remain absent/false.
5. Deterministic, race, production-build, installed-browser, scope/process,
   traceability, and fresh independent-review gates pass without waiver.

## Risks and rollback

Primary risks are cross-Workspace or quarantine disclosure, unstable pages,
source/revision mismatch, permissive client fallbacks, and accidentally enabling
S06B. Trusted identity, role-bounded statuses, selected-revision source joins,
filter-bound signed keyset cursors, strict schemas, and independent feature gates
address them. Rollback disables the two query routes and `knowledge_query` while
retaining governed revisions and sources. Down migration may run only when the
new tables are proven empty. Rollback never mutates the legacy scaffold, S05
history, user dirty files, or `server/**`.
