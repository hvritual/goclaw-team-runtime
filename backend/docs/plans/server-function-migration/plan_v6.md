# Incremental server function migration — Workspace completion and runtime cutover

- Plan-ID: `server-function-migration`
- Version: `6`
- Status: `approved`
- Approval source: user instruction dated 2026-08-03 to sequentially complete
  Todo CRUD, Issue mainline and owned objects, Knowledge governance,
  Requirement state machine, real adapters, PostgreSQL/HTTP/realtime, and
  per-slice runtime cutover
- Base commit: `0c4f848a8458e256dd4fe2ed51498af969aa3c59`
- Repository: `/Users/fworld/Hvritual/goclaw`
- Branch: `codex/multica-six-domain-baseline`
- Task type: `change`

## Goal and invariants

Migrate the remaining installed `server` Workspace behavior into the four-module
backend in independently verifiable slices, then replace stub/SQLite-only
composition with real Auth/Space/System adapters, PostgreSQL persistence,
compatible HTTP endpoints, realtime delivery, and explicit runtime switches.

Every slice preserves existing HTTP JSON compatibility, lowercase status and
priority strings, Workspace membership authorization, Workspace-scoped storage,
transactional side effects, realtime ordering, and SQLite-local behavior where
it does not conflict with the installed PostgreSQL implementation. No foreign
keys, cascading actions, cross-module table reads, or generated-file hand edits
are allowed.

## Ordered execution

### P6-S1 — Todo full CRUD (active)

Expand Todo from Create/UpdateStatus to the installed ordinary task lifecycle:

- Create, Get, filtered List, full Update, compatibility UpdateStatus, Delete.
- Preserve fields: Project/Issue references, title, description, status,
  priority, assignee identity, creator identity, position, start date, due date,
  completed time, created time, and updated time.
- Statuses remain `todo`, `in_progress`, `done`, `cancelled`; priorities remain
  `urgent`, `high`, `medium`, `low`, `none`.
- Empty optional Project/Issue/assignee/date values clear the association.
- List filters by Project, Issue, and status, ordered by position ascending then
  creation time descending with ID as a deterministic final tie.
- All reads and writes authorize before persistence and filter by Workspace.
- Project, Issue, and Auth actor validation uses consumer-owned ports only.
- Completion evidence and realtime publication are deferred to P6-S5/P6-S8;
  CRUD must expose sufficient actor and transition data without inventing a
  no-op integration that could be mistaken for production behavior.
- Implement first behind opt-in SQLite/local/gRPC composition. Default runtime
  remains unchanged.

### P6-S2 — Issue mainline and Issue-owned objects

Complete Issue create/get/list/search/update/move/delete and then migrate
Issue-owned labels, properties, comments, reactions, subscribers, pins,
attachments, parent/child progress, timeline, metadata, table/group/facet
queries, and acceptance conclusions. Preserve identifier allocation,
polymorphic assignees, ordering, status/priority vocabulary, dependent cleanup,
and authorization-first isolation.

### P6-S3 — Knowledge governance

Migrate proposal/candidate/review/publish/quarantine flows, entries, revisions,
sources, evidence, search, health, MCP-facing queries, and durable evidence
outbox dispatch. Space owns Asset lifecycle; Knowledge owns business evidence
and Asset references.

### P6-S4 — Requirement complete state machine

Complete Requirement create/update/version/list/get, approval transitions,
coverage transitions, Project/Issue links, conclusions/evidence, and explicit
transition guards. Requirement versions are immutable; aggregate state changes
and link updates commit atomically without database cascades.

### P6-S5 — real cross-module adapters

Bind Workspace consumer-owned ports to Auth Member/Agent, Space Asset, and
System Skill public contracts. Prefer in-process local adapters; retain gRPC
adapters for extraction tests. No module may read another module's tables.

### P6-S6 — PostgreSQL providers

Add provider-native PostgreSQL repositories, migrations, mappings, and unit of
work implementations per migrated slice. Every query is Workspace scoped.
Indexes use one `CREATE [UNIQUE] INDEX CONCURRENTLY` statement per migration;
schemas contain no foreign keys or cascading actions.

### P6-S7 — compatible HTTP boundary

Expose the installed `/api` request/response and error behavior through thin
HTTP adapters. Preserve `X-Workspace-ID`, authentication context, response
envelopes, status codes, null/omitted semantics, and UUID validation. Do not
route an endpoint to the new implementation until its compatibility tests pass.

### P6-S8 — realtime and durable side effects

Connect task/issue/knowledge/requirement events and Knowledge evidence through
explicit application ports and a durable outbox where loss is unacceptable.
Preserve event names, actor metadata, payload shape, commit-before-publish
ordering, idempotency, and self-event behavior.

### P6-S9 — per-slice runtime cutover

Introduce explicit bootstrap configuration for each accepted slice. For every
switch: run legacy-vs-new compatibility tests, PostgreSQL integration tests,
SQLite-local tests, HTTP/gRPC tests, realtime tests, race/architecture/full
gates, live health probes, and rollback verification. Remove the superseded
runtime path only after independent acceptance.

## P6-S1 current and target paths

- Current installed path: Chi task handler -> sqlc/PostgreSQL transaction ->
  task table -> Knowledge evidence outbox -> realtime bus.
- Current target-backend path: generated Todo adapter -> partial Create/Status
  use case -> SQLite repository.
- P6-S1 target path: generated local/gRPC adapter -> Todo application use case
  -> Todo domain -> Workspace-scoped repository port <- SQLite adapter.
- Later production path: compatible HTTP adapter -> same use case -> PostgreSQL
  unit of work, with Auth validation, Knowledge outbox, and realtime adapters
  bound in bootstrap.

## P6-S1 deterministic acceptance

- Proto evolution is additive and generated output is content-idempotent.
- Domain tests cover status/priority, optional update/clear semantics, UTC
  timestamps, completed-time behavior, and defensive pointer handling.
- Application tests cover authorization-first access, all CRUD operations,
  validation ports, Workspace-scoped misses, deterministic filters, and the
  UpdateStatus compatibility command.
- SQLite and bufconn tests cover all RPCs, filtering/order, update/clear,
  tenant isolation, delete/not-found behavior, and schema checks.
- Buf lint/format, generated contract tests, race tests, `go test ./...`,
  `go vet ./...`, `go mod verify`, architecture/static gates, policy hashes,
  scope audit, and live health probes pass.

## Safe stop and rollback

P6-S1 changes only additive contracts and opt-in composition. The running
backend continues to use generated stubs. Rollback is removing the additive
Todo declarations and the user-owned Todo implementation before any runtime
switch; installed `server`, HTTP routes, databases, and realtime remain
untouched.

## Deferred acceptance

This plan authorizes implementation in the declared order, but deterministic
completion of an earlier slice does not imply acceptance of later slices.
Production/runtime switching requires its own P6-S9 evidence and an explicit,
reversible bootstrap selection; no broad cutover is inferred from completing
domain or provider code.
