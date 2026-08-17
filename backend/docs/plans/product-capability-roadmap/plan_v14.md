# Product capability roadmap implementation plan v14

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Version: `14`
- Status: `approved-for-execution`
- Active step: `PCR-S02B promote a Task to an Issue`
- Task-ID: `PCR-001-S02B-R19`
- Task-Revision: `r019`
- Work-Item: `PCR-S02B`
- Base commit: `3262232c5000ed449b89d98901535cd58b42a48d`
- Supersedes: `plan_v13.md` for active execution only
- Approved: `2026-08-18`

## Outcome and authority

The Human Customer directed continued approved execution through Release 1.
PCR-S02A is complete and independently reviewed at exact closure `3262232`.
This version activates only PCR-S02B: a workspace member can explicitly promote
an eligible Task into exactly one Issue, retain an immutable source link, and
receive the same Issue when the same command is replayed.

No S03A search, S03B search, S04 pin reorder, deployment, push, merge, or
release completion is authorized. `server/**` remains permanently read-only.

## Frozen product contract

### HTTP command

- Route: `POST /api/tasks/{id}/promote`.
- Required header: `Idempotency-Key`.
- Body: `expected_revision` (positive integer) and `complete_task` (boolean,
  default `false`). Unknown fields are rejected.
- Response: HTTP `201` with a schema-validated object containing the updated
  `task`, created `issue`, and `source_task_id`. An exact replay returns the
  original status and response without another domain mutation.
- Same key with a different canonical body returns `409
  code=idempotency_conflict`. A stale Task revision returns `409
  code=revision_conflict` with `current_revision` and no writes.
- A Task already promoted or already linked to an Issue by another path returns
  a stable conflict and never creates a second Issue.

### Snapshot and lifecycle rules

- Promotion copies the Task title, description, priority, project, assignee,
  start date, and due date into a new `todo` Issue. The trusted request actor is
  the Issue creator; no actor authority is accepted from the body.
- Promotion never silently synchronizes later Task or Issue edits.
- `complete_task=false` retains Task status and advances its revision once when
  the immutable Issue link is installed.
- `complete_task=true` is valid only for an `in_progress` Task and atomically
  moves it to `done` while installing the link. Other lifecycle states return a
  validation conflict without writes.
- The existing mutable `workspace_todos.issue_id` reference is not itself the
  immutable provenance record. A new additive promotion-link table is the
  authority. Once a promotion link exists, ordinary Task updates cannot replace
  or clear its Issue link.

### Authorization and transaction

- The caller must pass Task read plus creator-owned update or workspace Task
  management authorization, and the existing Issue-create authorization.
- Member and agent identity are resolved only from trusted runtime context.
- One SQLite `BEGIN IMMEDIATE` governed transaction performs: Task/revision
  validation, workspace Issue number allocation, Issue insert, immutable link
  insert, Task link/status update, resource revision, audit, outbox, and bounded
  idempotency replay. Any failure rolls back every row.
- The governance action is `workspace.task.promote`, bound to the source Task.
  Its safe envelopes contain identifiers, revision, resulting status, and the
  `complete_task` decision only. No titles, descriptions, credentials, raw
  authorization material, or caller hashes enter governance records.
- A committed `task:updated` outbox event is sufficient to refresh both Task and
  Issue React Query domains through the installed realtime invalidator. Replay
  creates no second audit or event.

## Migration

- Add paired migration `000012_task_issue_promotion.up.sql` and `.down.sql`.
- The link table stores `workspace_id`, `task_id`, `issue_id`, `created_at` with
  primary/unique constraints that enforce one Task to one promoted Issue and
  prevent an Issue from claiming multiple source Tasks.
- Do not add database foreign keys, cascades, triggers, or indexes. Relationship
  validation and cleanup remain application-transaction responsibilities.
- Down migration succeeds only when the promotion-link table is empty, deletes
  its migration catalog row, and otherwise aborts without data loss.
- Migration tests cover first install, second-run idempotence, retained data,
  empty-only down, rollback, restart, and the updated exact migration count.

## Scope

Writable product scope is limited to:

- `backend/internal/modules/workspace/contract/**` user-owned supplemental
  promotion contract and authorization tests; generated files marked
  `DO NOT EDIT` remain unchanged;
- `backend/internal/modules/workspace/internal/application/**` Task promotion,
  governance policy, and tests;
- `backend/internal/modules/workspace/internal/infrastructure/sqlite/**` Task
  promotion repository, paired 000012 migration, and tests;
- `backend/internal/modules/workspace/internal/interfaces/http/task.go` and its
  tests;
- `backend/internal/modules/workspace/sqlite_workspace_chain.go`, Task HTTP
  extension wiring, and focused module tests;
- `backend/internal/bootstrap/**` installed runtime and restart tests;
- `packages/core/types/task.ts`, `packages/core/tasks/**`, the Task API client
  boundary/schema tests, and Issue-query invalidation required by promotion;
- `packages/views/tasks/**` and the existing `en`, `ja`, `ko`, and `zh-Hans`
  Task locale files;
- `e2e/tasks.spec.ts` for installed promotion acceptance;
- current roadmap `plan.md`, `plan_v14.md`, `story-map.md`,
  `task-register.md`, and append-only `journal.md`.

Excluded dirty paths remain untouched and unstaged:

- `packages/ui/components/ui/input.tsx` (required blob
  `a830fd2f0f82770563908d512558fe6ba48f50dd`);
- `packages/views/issues/components/table-view.tsx`;
- `packages/views/modals/create-issue.tsx`;
- generated protobuf line-ending/status drift;
- `.local-runtime/**`, `docs/code-to-product/**`,
  `packages/views/auth/input-controlled.test.tsx`, and `ui/**`.

Every `server/**` path, prior immutable `plan_v1.md` through `plan_v13.md`,
Issue create UI, manifests, lockfiles, and unrelated test-baseline repair are
out of scope.

## Ordered execution

### PCR-S02B-R19.1 — Activate authority

- Freeze v14 at exact base `3262232` and establish r019 as the sole active task.
- Revalidate the policy hashes, excluded blob, empty `server/**` diff, zero
  pre-activation active tasks, and repository status.

### PCR-S02B-R19.2 — RED contracts

- Add failing migration, application, repository, HTTP/runtime, Core boundary,
  View, and browser acceptance tests before implementation.
- RED evidence must distinguish absent promotion behavior from environment or
  fixture failures.

### PCR-S02B-R19.3 — GREEN backend

- Implement the user-owned promotion contract, authorization, snapshot mapping,
  governed transaction, immutable link, 000012 migration, HTTP route, runtime
  wiring, and safe error mapping.
- Cover same-key replay, different-body conflict, concurrent single winner,
  stale revision, permission denial, rollback at every governance phase,
  optional completion, immutable link, no later synchronization, and restart.

### PCR-S02B-R19.4 — GREEN shared client and view

- Add strict Zod promotion response parsing and malformed-response failure.
- Add a shared mutation that invalidates Task detail/lists and Issue lists.
- Add an explicit promotion affordance and result link in the shared Tasks page,
  with loading, conflict, denied, and error states and glossary-compliant copy.
- Web and Desktop inherit the same shared implementation; no app-specific
  business duplicate is allowed.

### PCR-S02B-R19.5 — Verify and close

- Run focused RED/GREEN tests, `pnpm typecheck`, `pnpm test`, backend
  `make check`, and real `make test-race`.
- Build an exact detached clean candidate, migrate a new database, seed the
  canonical fixture, and run installed Chrome Playwright promotion acceptance.
- Verify scope, migration reversibility, hashes/trailers, excluded dirty blob,
  zero `server/**` diff, process cleanup, and fresh independent read-only review.
- Only a complete deterministic and independent PASS closes r019/S02B. S03A
  remains inactive pending a new versioned plan.

## Acceptance criteria

1. One authorized command creates one Issue and one immutable promotion link in
   the caller's workspace.
2. Exact replay returns the original Issue and creates no second Issue, link,
   audit entry, outbox event, or Task revision.
3. Same key with a different body, stale revision, foreign workspace, denied
   actor, already-linked Task, and invalid lifecycle all fail without writes.
4. Concurrent attempts produce one durable promotion and deterministic loser
   outcomes; Issue numbering remains unique and monotonic.
5. Optional completion is atomic; rollback leaves Task, Issue numbering, Issue,
   link, audit, outbox, and replay unchanged.
6. Later Task edits do not modify the Issue, later Issue edits do not modify the
   Task, and the promotion link cannot be replaced or cleared.
7. Response parsing fails closed on malformed mutation data; successful client
   mutation refreshes both Task and Issue server state.
8. The shared Web/Desktop page visibly promotes a Task, shows the resulting
   Issue identifier/link, and renders pending and failure states.
9. Promotion and its source link survive process restart and a new API read.
10. Full deterministic gates, exact clean-candidate browser acceptance, scope
    checks, and independent review pass with no waiver.

## Risks and rollback

- Nested Issue transactions would split atomicity; the promotion repository
  must allocate and insert the Issue on the governance connection.
- Reusing mutable `issue_id` as provenance would allow source-link loss; the
  additive immutable link is authoritative and guarded during Task updates.
- Governance outbox aggregate identity is the source Task; event schemas and
  publication validation must agree before rows can be claimed.
- Shared dirty UI files can contaminate acceptance; exact-browser verification
  uses a detached clean candidate.

Rollback reverts only r019 commits. The 000012 down migration may run only when
the promotion-link table is empty. No rollback resets user work, deletes a
populated link table, modifies `server/**`, or rewrites an older plan.
