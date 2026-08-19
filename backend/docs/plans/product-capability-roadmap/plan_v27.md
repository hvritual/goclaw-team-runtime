# Product capability roadmap v27 — S07A independent-review remediation

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Plan-Version: `27`
- Task-Revision: `r032`
- Work-Item: `PCR-S07A`
- Exact base: `71afb3c33a4d82431a8016cb195a97e5a36d8646`
- Blocked input tree: `6559133236c2c6d02b8453ea72ad52c054a6b10c`
- Blocked input patch hash: `575e289823966eb246351e9ae2db7b71bcbd0f54`
- Status: `approved-active`
- Authority: the Human Customer's confirmed direction on 2026-08-19 to
  continue approved execution until Release 3 is complete

## Why this successor exists

The v26/r031 implementation candidate reached exact tree
`6559133236c2c6d02b8453ea72ad52c054a6b10c` with 42 product paths, zero
unstaged paths, and zero `server/**` paths. Focused S07A checks, the production
Web build, and an installed-Chrome journey passed; the root aggregate retained
an unrelated pre-existing analytics timeout and the full Windows race aggregate
retained pre-existing onboarding/outbox failures, so neither aggregate was
represented as PASS.

Fresh independent review returned both `SPEC BLOCK` and
`CODE/SECURITY/QUALITY BLOCK`. It found incomplete down-migration reference
guards, stale-state ordering and refresh time-of-check/time-of-use gaps,
mutation authorization checked outside the write transaction, one missing
Project-create HTTP permission mapping, raw malformed response values exposed
to schema logging, and explicit SQLite indexes incompatible with the repository
migration rule. v26 remains immutable and r031 is review-blocked. This plan
authorizes only the material repairs needed to re-enter the same S07A gates.

## Preserved outcome and exclusions

The complete v26 product, security, route, authorization, client, UI, and
installed-acceptance contracts remain binding. S07A still delivers only
Workspace-owned GitHub repository and generic public-URL Resources. S07B-D,
Release 3 completion, generated protobufs, unrelated Input/Issue UI, push,
merge, deployment, and every `server/**` change remain inactive or excluded.

The no-network default adapter, credential-free canonical URLs, authoritative
active counts, atomic Project create/delete behavior, strict Zod response
parsing, and false-until-installed capability rule are unchanged. This
successor does not authorize generic server-side fetching, an SQLite migration
exception, new Resource types, new routes, or unrelated verification repairs.

## Frozen remediation contract

1. The `000018` down migration aborts before any schema change when any
   `project_resources` row exists, when any `project_resource_sets` row exists
   including revision zero, or when any audit/idempotency authority row refers
   to Project Resources by `resource_kind` or the
   `workspace.project.resource.*` action namespace. Each guarded reference is
   tested independently; an abort preserves all schema and data.
2. The SQLite migration contains no explicit index DDL. Duplicate active or
   archived Resource fingerprints are rejected by a transaction-local lookup
   inside the same `BEGIN IMMEDIATE` write transaction used by create/update
   and Project-create-with-Resources. Concurrent duplicates prove one winner
   without relying on an unapproved index.
3. Every Resource write transaction revalidates the current actor against
   `auth_members`, the current Project Workspace/status, and the Project's
   current member lead after the write lock is acquired. Owner/admin or a
   matching current member lead may manage; missing membership, lead demotion,
   non-member lead, archived/completed/cancelled Project, agent default, or
   foreign Workspace denies before mutation, audit, idempotency persistence, or
   adapter work.
4. Revision precedence is transactional. After current authorization succeeds,
   mutation/restore/refresh compares the Resource-set revision before checking
   mutable Resource state. A stale authorized request therefore returns `409
   revision_conflict` even if the Resource is archived or otherwise invalid,
   with no write, audit, or adapter call.
5. Refresh is one repository-owned `BEGIN IMMEDIATE` operation. The repository
   acquires the write boundary, performs current authorization, revision, and
   active-Resource checks, then invokes only the injected typed connection
   resolver and persists the safe projection before commit. A deterministic
   barrier race proves the losing stale request invokes the resolver zero
   times. Resolver failure remains a safe retained degraded/unavailable
   projection under the unchanged v26 contract.
6. Project creation rechecks current actor membership/role and requested
   member-lead authority inside its existing write transaction before inserting
   the Project or initial Resources. Its HTTP surface maps Resource permission
   denial to the established non-secret `403` contract rather than `500` and
   proves full rollback.
7. Core schema fallback logging never emits raw received values. It records
   only the endpoint/operation, validation issues, and safe structural type or
   shape metadata. A malformed Resource payload containing a secret-bearing URL
   proves the logger, thrown errors, and returned fallback contain no secret.
8. Replay and key-conflict behavior from v26 remains exact. Any implementation
   rearrangement required by transactional authorization must authorize before
   returning a replay and must not create new audit/idempotency/adapter effects.
   Generating identifiers later or exposing a caller-supplied retry key may be
   considered only if it is required for a frozen failing test; it is not an
   independent scope expansion.

## Writable scope

- Every product/test path already authorized by immutable v26;
- exact S07A application/repository interfaces and SQLite migration/tests under
  `backend/internal/modules/workspace/**` needed for the seven review blockers;
- exact Project-create transaction and HTTP mapping/tests under
  `backend/internal/modules/workspace/**`;
- `packages/core/api/schemas.ts` and its exact schema tests for safe logging;
- current roadmap pointer, story map, task register, journal, and this immutable
  plan.

No other package, migration, generated artifact, legacy backend tree, or
`server/**` path is writable. The original worktree's tracked and untracked
changes remain byte-for-byte outside this isolated task worktree.

## Ordered execution

1. R32.1 — Freeze this successor from exact activation base `71afb3c3`, record
   r031's exact blocked candidate/review evidence, and commit only governance
   activation paths.
2. R32.2 — Add RED tests for revision-zero and non-create reference rollback
   guards, no-index migration shape, and concurrent duplicate safety; GREEN the
   minimal migration/repository corrections.
3. R32.3 — Add deterministic RED barrier tests for lead demotion, Project close,
   revision-before-state, refresh resolver suppression, and Project-create
   permission rollback; GREEN the transaction-owned authorization/revision/
   refresh boundary.
4. R32.4 — Add the RED HTTP permission and Core secret-logging tests; GREEN the
   narrow error mapping and safe schema diagnostics.
5. R32.5 — Re-run focused and broad deterministic checks, fresh official race
   checks for exact S07A packages, root typecheck/test, production Web build,
   and an expanded retries-disabled installed-Chrome journey against a fresh
   Canonical SQLite runtime.
6. R32.6 — Freeze one exact candidate; verify hashes, trailers, scope, empty
   `server/**` diff, original dirty-path preservation, and process cleanup; then
   obtain fresh independent `SPEC PASS` and `CODE/SECURITY/QUALITY PASS` on that
   exact candidate.

## Acceptance and deterministic verification

- Migration tests exercise every guarded authority source independently,
  confirm revision-zero protection, assert no explicit index statements, and
  prove schema/data remain unchanged after a blocked down migration.
- Concurrent create/update/Project-create tests prove transaction-local
  duplicate detection yields exactly one winner and preserves contiguous
  positions, revision, audit, and idempotency invariants.
- Barrier-controlled tests prove lead demotion and Project closure before commit
  deny the pending write, authorized stale restore/refresh returns 409 before
  state errors, and no failed operation invokes the resolver or writes effects.
- Focused backend/Core/Views tests pass, followed by backend `make check`, fresh
  official race execution for every changed S07A package, root typecheck/test,
  and production Web build. A broad aggregate failure remains non-PASS unless
  exact evidence classifies it as unrelated and the affected S07A checks pass.
- Fresh installed Chrome with retries disabled proves the unchanged v26 owner,
  member, current-lead, stale reorder, refresh, archive/restore, reload, and
  Project-delete journey plus the repaired permission/secret boundaries where
  they are browser-observable.
- Only a fresh independent review of the frozen exact candidate returning both
  `SPEC PASS` and `CODE/SECURITY/QUALITY PASS` may close r032 and S07A. Closure
  of S07A alone does not activate S07B or claim Release 3 completion.

## Rollback and stop conditions

Rollback remains the v26 capability-disable and guarded empty-authority path;
the down migration must never destroy referenced authority. Stop before closure
on any resolver call before transaction-owned authorization/revision/state
checks, stale request accepted or misclassified, role/status TOCTOU, duplicate
acceptance, secret-bearing diagnostics, Project-create partial commit, explicit
SQLite index DDL, `server/**` change, original dirty-worktree overlap, scope
drift, consumed-gate retry represented as PASS, or independent review BLOCK.
Any further material repair requires another immutable successor plan.
