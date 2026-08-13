# Canonical SQLite runtime cutover — execution plan v1

- Plan-ID: `canonical-sqlite-runtime-cutover`
- Version: `1`
- Status: `approved`
- Approval source: user confirmation dated 2026-08-13 to establish this
  repository execution milestone
- Active step: `M1-S0`
- Base commit: `e4b4b1c7e3d46b19fb4774f8757cad4fb4c4f1cc`
- Branch at approval: `codex/issue-metadata-v9`
- Integration target: `codex/multica-six-domain-baseline`
- Repository: `F:\code\ai\goclaw-team-runtime`
- Project-ID: `goclaw-team-runtime`
- Delivery mode: `change-spec`
- Reimplementation mode: `compatible-replacement`
- XP mode: `strict`
- Maximum active stories: `1`
- Policy bundle:
  - `AGENTS.md`: `637c5ff1222ba462b3b3ff96c74e4ad0b62f52bfa086d76c396d99badb9848e0`
  - `CLAUDE.md`: `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646`
  - `backend/AGENTS.md`: `fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209`

## Goal

Deliver the first independently usable local runtime in which the existing Web
frontend talks only to the Canonical `backend/**` implementation and a local
SQLite database. The accepted journey is:

```text
start Canonical backend + Web frontend
  -> authenticate
  -> select a Workspace
  -> list Issues
  -> open an Issue
  -> read, put, and delete Issue metadata
  -> receive the corresponding realtime refresh
```

The journey must not require a running legacy `server/**` process. Existing
request and response JSON bodies remain unchanged. A URL change is permitted
only at the centralized frontend API boundary and must have an explicit parity
decision and compatibility test.

## Current-state boundary

At this plan's base commit:

- the default Canonical server composition registers four modules but selects
  generated/default services rather than the opt-in SQLite Workspace chain;
- Canonical SQLite Workspace services exist only through explicit composition;
- the separate control-plane executable owns its own SQLite runtime and is not
  the frontend-compatible application server;
- Canonical HTTP coverage is incomplete outside health/Ping, control-plane,
  and the opt-in Issue metadata slice;
- the frontend still depends on legacy-compatible auth, Workspace, Issue, and
  realtime behavior;
- the Issue metadata v9 candidate is uncommitted work on
  `codex/issue-metadata-v9` and is therefore a prerequisite, not baseline fact.

These statements are planning evidence, not a runtime-readiness claim.

## Scope

In scope for this milestone:

- one Canonical local SQLite application composition and lifecycle;
- the minimum authentication/session/current-user behavior required by the
  existing frontend journey;
- Workspace discovery, selection, membership, and authorization for that
  journey;
- Issue create/get/list/update behavior required by list and detail views;
- Issue metadata Get/Put/Delete through the accepted v9 contract;
- centralized `/api` compatibility adapters without body-shape drift;
- the minimum Issue and metadata realtime behavior required for cache refresh;
- local startup, readiness, seed/bootstrap, data transition, rollback, and
  browser acceptance evidence;
- Web integration through shared `packages/core` APIs. Desktop receives shared
  client compatibility evidence but desktop packaging is not a release gate.

## Non-goals

- modifying, generating into, deleting, or otherwise writing `server/**`;
- production PostgreSQL readiness or a production deployment;
- full parity for comments, reactions, subscribers, attachments, labels,
  custom properties, inbox, agents, skills, knowledge, requirements, tasks,
  billing, integrations, or administration;
- a new Issue metadata editing interface;
- changing existing request or response JSON bodies;
- merging the control-plane and product API into one domain model unless an
  approved architecture revision explicitly requires it;
- deleting the legacy startup path before rollback and data checks pass;
- including unrelated dirty worktree files in a milestone commit.

Deferred features must fail honestly or remain routed to the legacy runtime;
they must not be represented as Canonical-ready.

## Invariants

1. `backend/**` is the only writable backend implementation root.
2. `server/**` is permanently read-only evidence and must have an empty diff.
3. Request and response JSON bodies stay compatible with the installed client.
4. URL knowledge remains centralized in `packages/core/api/client.ts`.
5. Authentication and Workspace authorization complete before repository
   access; caller-supplied actor headers are not trusted as identity.
6. Every Workspace query and mutation is scoped by canonical Workspace ID.
7. SQLite mutations that span dependent state are atomic and rollback-safe.
8. Realtime publication occurs after commit and carries sufficient identity to
   invalidate the correct Workspace-scoped React Query cache.
9. Default services never silently fall back to not-implemented stubs in the
   selected Canonical local profile.
10. React Query owns server state; the frontend does not duplicate server data
    into Zustand.
11. Only one story is active. A later step cannot start until its dependency
    gate and acceptance evidence are indexed in `journal.md`.
12. Existing unrelated changes are preserved and excluded from task commits.

## Dependencies

Hard prerequisites before `M1-S1`:

- Issue metadata v9 is committed on its scoped branch, independently reviewed,
  merged into `codex/multica-six-domain-baseline`, and the local branch is
  synchronized to that integration base;
- the candidate merge contains no `server/**` path and no unrelated dirty file;
- `M1-S0` freezes the exact endpoint inventory used by the accepted journey,
  characterization tests, identity contract, SQLite ownership, startup ports,
  and rollback switch;
- any base or policy drift is recorded and, when material, issued as
  `plan_v2.md` for Human approval.

## Ordered execution

### M1-S0 — Freeze compatibility and runtime contracts (active)

Allowed writes:

- `backend/docs/plans/canonical-sqlite-runtime-cutover/**`
- focused characterization tests only after their exact paths are added to an
  approved plan revision

Work:

- inventory the exact frontend calls made by login, Workspace selection, Issue
  list/detail, metadata mutation, and realtime reconnect;
- record legacy-observed and Canonical-target routes, methods, headers, bodies,
  status/error semantics, auth ordering, and event shapes;
- decide one SQLite owner, one Canonical process topology, and whether the
  control-plane process remains separate;
- freeze local ports, database path, seed/bootstrap behavior, and rollback
  switch;
- promote all critical rows in `parity-matrix.md` from `Unknown` to an approved
  target decision with a verification method.

Exit gate: no critical journey row is `Unknown`; tests and exact write paths for
`M1-S1` are frozen; Human Customer approves the resulting plan revision.

### M1-S1 — Canonical SQLite application composition

Target value: one command starts a Canonical backend that owns the local SQLite
database, runs migrations, wires real in-process providers, and exposes health
and readiness without selecting generated stubs for milestone capabilities.

Expected paths after S0 approval:

- `backend/cmd/**`
- `backend/internal/bootstrap/**`
- public module composition under `backend/internal/modules/**`
- focused bootstrap/composition tests and local configuration

Acceptance: startup from an empty database, restart with retained data,
migration idempotence, dependency validation, graceful close, health/readiness,
and explicit failure when a required real provider is absent.

### M1-S2 — Auth and trusted identity vertical slice

Target value: the existing login/session/current-user flow establishes trusted
identity for Canonical handlers and Workspace authorization.

Acceptance: success, invalid credentials, missing/expired session, logout,
cookie/bearer behavior selected in S0, CSRF behavior where applicable, and
fail-closed identity tests all preserve frozen bodies and public errors.

### M1-S3 — Workspace selection and authorization vertical slice

Target value: an authenticated user can list accessible Workspaces, select one,
load its membership context, and be denied access to foreign Workspaces.

Acceptance: empty/list/detail/member-role cases, slug-to-canonical-ID
resolution, missing Workspace identity, foreign actor, and role matrix tests
pass against SQLite and HTTP.

### M1-S4 — Issue mainline vertical slice

Target value: the existing Issue list and detail pages load from Canonical
SQLite, and their required create/update operations remain compatible.

Acceptance: identifier and UUID lookup, list filters/order/pagination actually
used by the UI, create/get/list/update, workspace isolation, polymorphic
assignee handling needed by the view, malformed response tests in
`packages/core`, and browser-visible empty/error/success states pass.

Any Issue-owned feature not required by the frozen journey remains deferred and
must not be hidden behind fabricated empty success responses.

### M1-S5 — Issue metadata frontend completion

Target value: the accepted Issue metadata v9 service operates through the real
runtime identity, Workspace, HTTP, SQLite, and existing shared frontend client.

Acceptance: Get/Put/Delete preserve the exact body contract, primitive types,
key encoding, limits, auth ordering, workspace isolation, rollback, concurrent
write behavior, and the existing Web/Desktop read-only projection.

### M1-S6 — Minimum realtime vertical slice

Target value: committed Issue and Issue metadata mutations trigger the frozen
realtime events and refresh only the correct Workspace-scoped frontend cache.

Acceptance: connect/reconnect, authorization, event ordering, commit-before-
publish, resume cursor if required by S0, self-event behavior, duplicate
delivery tolerance, and cache update/invalidation tests pass.

### M1-S7 — Local cutover, migration, and acceptance

Target value: a documented repository command starts Canonical SQLite backend
and Web frontend, with no legacy server process, and the accepted browser
journey succeeds.

Acceptance:

- clean and retained SQLite startup paths pass;
- any legacy-to-Canonical local data transition is reconciled by counts and
  stable IDs, or the milestone explicitly requires a fresh local database;
- process inspection proves no `server/**` executable is serving requests;
- browser acceptance covers login, Workspace selection, Issue list/detail, and
  metadata read/write/delete plus refresh;
- rollback restores the prior startup selector without deleting either local
  database;
- independent Navigator review and Human Customer acceptance are recorded.

## Deterministic verification

Exact commands are frozen in M1-S0 using repository scripts as the source of
truth. The minimum gate set is:

```text
backend: formatting, policy, generated-clean, focused tests, all tests, race,
         vet, module verification, architecture/static checks
frontend: focused core schema/client tests, typecheck, shared view tests,
          repository tests required by changed packages
runtime:  empty/retained SQLite start, health/readiness, HTTP contract tests,
          realtime tests, browser journey, process/port ownership audit
scope:    diff check, server-path diff must be empty, unrelated files excluded
```

Windows-incompatible wrapper failures do not count as passing. The underlying
deterministic commands must be run directly and their exit codes indexed.
Claims about race, browser, live runtime, or migration require actual evidence;
static inspection is not a substitute.

## Milestone acceptance

The milestone is accepted only when all of the following are true:

1. A single documented local workflow starts Canonical backend and Web frontend
   against local SQLite.
2. Login, current user, Workspace list/selection, Issue list/detail, and Issue
   metadata Get/Put/Delete complete through Canonical handlers and repositories.
3. Existing request and response bodies are unchanged for every in-scope call;
   any URL change is centralized and tested.
4. Missing identity fails closed; foreign Workspace resources are not exposed;
   authorization occurs before repository access.
5. Required Issue/metadata realtime behavior is proven after commit and through
   frontend cache behavior.
6. No legacy `server/**` process is required for the accepted journey and the
   candidate diff contains no `server/**` path.
7. SQLite start/restart, rollback, and any data transition are proven without
   destructive cleanup.
8. Deterministic gates, independent review, and Human Customer acceptance are
   indexed in `journal.md`.

Passing this milestone does not mean full legacy-server replacement. It means
the frozen local Issue journey is Canonical-only and independently usable.

## Risks and mitigations

- **Hidden frontend dependency:** freeze network calls from the real journey in
  S0 and fail on unexpected legacy calls during browser acceptance.
- **Two SQLite authorities:** select one product database owner in S0; adapters
  cross module contracts rather than reading another module's tables.
- **Identity spoofing:** resolve actor identity through the trusted auth/session
  boundary; never accept actor identity from arbitrary workspace headers.
- **Compatibility drift:** use characterization and malformed-response tests;
  keep body shapes frozen and paths centralized.
- **Generated stub illusion:** composition tests assert concrete real providers
  for each milestone capability.
- **Realtime loss or reordering:** publish after commit and test reconnect,
  duplication, ordering, and cache effects.
- **Dirty worktree contamination:** stage explicit paths and audit every commit.
- **Oversized cutover:** keep one active vertical story and require a working
  demonstration before promoting the next.

## Rollback

- Each story must be additive or selected by an explicit local runtime switch
  until S7 acceptance.
- Preserve the previous startup command and database; do not overwrite or
  delete local legacy data during validation.
- On a failed cutover probe, stop the Canonical processes, restore the previous
  selector, and retain both logs and databases for diagnosis.
- Roll back the active story only. Do not rewrite accepted plan versions or
  revert unrelated user work.

## Stop conditions

Stop and request a plan revision when:

- any request or response JSON body must change;
- `server/**` would need a write;
- a critical parity row remains `Unknown` at the S0 exit gate;
- trusted identity, Workspace isolation, transactionality, or realtime ordering
  cannot be preserved;
- a new production database/deployment requirement enters scope;
- the current base, policy bundle, or Issue metadata prerequisite differs
  materially from this snapshot;
- a deterministic hard gate fails and the correction lies outside the active
  step's write boundary.
