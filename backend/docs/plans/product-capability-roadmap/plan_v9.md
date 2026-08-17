# Product capability roadmap implementation plan v9

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Version: `9`
- Status: `approved-for-execution`
- Active step: `PCR-S02A`
- Task-ID: `PCR-001-S02A-R14`
- Task-Revision: `r014`
- Work-Item: `PCR-S02A`
- Base commit: `628996378af6fbe12c27a916a624a5f5374a884f`
- Frozen contract: `PCR-CONTRACT-v1`
- Approved: `2026-08-18`

## 1. Authority and objective

The Human Customer directed Codex to continue all approved follow-up work until
Release 1 is complete on 2026-08-18. This version narrows that authority to the
first dependency-ordered Release 1 story only: `PCR-S02A — Manage tasks`.

Deliver one installed Canonical Task vertical slice through Workspace domain,
SQLite, HTTP, shared TypeScript client, and the shared Web/Desktop view. A
member can create, view, filter, reorder, edit, complete, cancel, archive, and
restore tasks with member or agent assignment, dates, deterministic ordering,
workspace isolation, optimistic concurrency, and restart persistence.

This version does not authorize `PCR-S02B` promotion or any later Release 1
story. Product naming is `Task`; the existing backend `Todo` aggregate remains
an internal compatibility name and must not become a second public entity.

## 2. Current evidence and dependencies

- Release 0 is complete at base `6289963`; Release 1 has no earlier active task.
- `backend/internal/modules/workspace` already composes a local/gRPC Todo use
  case and SQLite repository, but `TodoExtension.RegisterHTTP` installs no
  `/api/tasks` routes.
- The current Task model has no revision, archive/restore state, or strict
  transition enforcement. Repository updates are last-write-wins.
- `packages/core/api/client.ts` declares `/api/tasks` calls without boundary
  schemas. The shared page supports only create, status change, and hard delete.
- Web and Desktop already route to the same `TasksPage`; no parallel platform
  implementation is needed.
- `PCR-CONTRACT-v1` Sections 1 through 5 govern Workspace authority, route
  families, trusted workspace resolution, response parsing, permissions, and
  the Task state machine.
- `PCR-S02B` is a dependent story and remains inactive until S02A acceptance.

## 3. Invariants

1. `backend/**` is the only writable backend root. `server/**` is permanently
   read-only and excluded from every step, check, generator, and rollback.
2. Workspace identity and actor identity come from trusted runtime context.
   Request bodies cannot select an owner, creator, or workspace.
3. Task write authority remains Workspace. No Control Plane dual write or
   cross-module table read is introduced.
4. Missing authorization denies access. Members can read/create/update their
   own tasks; owner/admin workspace management remains separately gated; agents
   require explicit grants.
5. Public JSON uses `snake_case`. Every TypeScript-consumed response passes a
   Zod boundary schema and malformed mutation responses fail closed.
6. Revisioned mutations require `expected_revision`. Stale requests return
   `409`, `code=revision_conflict`, and the current revision without mutation,
   audit, or outbox publication.
7. Task lifecycle is exactly:

   ```text
   todo -> in_progress -> done
     \         \-> cancelled
      \--------------> cancelled
   done/cancelled -> archived -> restored to previous terminal state
   ```

   Invalid transitions do not change state or revision. Archive/restore is not
   permanent deletion.
8. List order is deterministic and ends with immutable Task ID. Reorder is
   atomic, workspace-scoped, revision-checked, and cannot partly apply.
9. Task mutations use the Release 0 governance transaction so the domain write,
   revision, idempotency result where applicable, audit, and outbox state commit
   or roll back together. No secret or unrestricted raw body enters governance
   envelopes.
10. The capability flag becomes installed only after the default HTTP runtime
    exposes the complete S02A behavior and its runtime acceptance passes.
11. Web and Desktop consume the same Core client and Views implementation.
12. Existing unrelated dirty and untracked work remains untouched.

## 4. Included scope

### 4.1 Authority and evidence documents

- `backend/docs/plans/product-capability-roadmap/plan.md`
- `backend/docs/plans/product-capability-roadmap/plan_v9.md`
- `backend/docs/plans/product-capability-roadmap/story-map.md`
- `backend/docs/plans/product-capability-roadmap/task-register.md`
- `backend/docs/plans/product-capability-roadmap/journal.md`
- `backend/docs/plans/product-capability-roadmap/capability-matrix.md`

### 4.2 Canonical Task backend

- `backend/api/workspace/v1/todo.proto`
- generated Todo outputs under `backend/rpc/pb/workspace/v1/`
- Task/Todo public contracts under
  `backend/internal/modules/workspace/contract/`
- Task/Todo domain, application, SQLite repository, migration, local/gRPC/HTTP
  adapters, and focused tests under `backend/internal/modules/workspace/`
- only the Task installation and capability wiring required under
  `backend/internal/bootstrap/` and `backend/cmd/server/`
- `backend/tests/contract/workspace_todo_test.go`

The expected additive migration is `000010_task_lifecycle`. It may add Task
revision, archive/restore, and ordering support. It must not add foreign keys,
cascades, cross-module tables, or a destructive down migration. Any index uses
its repository-approved migration form.

### 4.3 Shared client and product surface

- Task schemas, types, client methods, React Query ownership, and tests under
  `packages/core/api/`, `packages/core/tasks/`, and
  `packages/core/types/task.ts`
- shared Task view and tests under `packages/views/tasks/`
- Task locale resources under `packages/views/locales/*/tasks.json`
- existing Task route wiring tests in `apps/web/` and `apps/desktop/` only when
  required to prove the shared surface remains installed
- a focused installed-runtime journey under `e2e/`

## 5. Excluded scope

- every path under `server/**`;
- `PCR-S02B` Task promotion, Issue creation, or Task/Issue synchronization;
- S03 search, S04 pin reorder, Skills, Knowledge, Resources, Requirements,
  retrospectives, reminders, project phase, and outline work;
- mobile behavior, deployment, release tags, push, merge, or external systems;
- changing frozen authority, route-family, permission, or lifecycle semantics;
- broad UI redesign or unrelated refactoring;
- permanent Task deletion through the S02A user surface;
- the following pre-existing user/workspace paths:
  - `packages/ui/components/ui/input.tsx` (working-tree blob
    `a830fd2f0f82770563908d512558fe6ba48f50dd`);
  - `packages/views/issues/components/table-view.tsx`;
  - `packages/views/modals/create-issue.tsx`;
  - `packages/views/auth/input-controlled.test.tsx`;
  - `.local-runtime/**`, `docs/code-to-product/**`, and `ui/**`.

If implementation requires a path or semantic change outside Sections 3 and 4,
stop and create a newly approved plan version. Do not silently widen v9.

## 6. Ordered execution

### PCR-S02A.1 — Activation and RED contract

- Commit this immutable plan and the r014 authority records from exact base.
- Verify one active roadmap task, frozen policy hashes, unchanged prior plans,
  dirty exclusions, and empty tracked/staged/untracked `server/**` scope.
- Add failing tests for the installed HTTP Task journey, permission denial,
  lifecycle/archive/restore, revision conflict, deterministic filters/order,
  atomic reorder, assignment, dates, restart, malformed client responses, and
  shared view empty/denied/error behavior.
- Record the observed RED failures before implementation.

### PCR-S02A.2 — Domain, persistence, and governance GREEN

- Add revision and archive/restore semantics without creating a competing Task
  aggregate.
- Enforce the frozen lifecycle, actor references, due dates, deterministic order,
  workspace isolation, and atomic reorder in Workspace contracts and SQLite.
- Route mutations through governance transactions and emit sanitized stable
  Task event/audit envelopes.
- Keep down migration safe: it may refuse rollback when lifecycle data cannot be
  represented without loss.

### PCR-S02A.3 — Installed HTTP and capability GREEN

- Install the frozen `/api/tasks` family with trusted context, CSRF/bearer rules,
  structured errors, `expected_revision`, archive via `DELETE`, and restore via
  `POST /api/tasks/{id}/restore`.
- Return `201/200/204` as frozen and `409` with current revision on conflicts.
- Report the tasks capability installed only when the default runtime has the
  complete handler/service chain and readiness remains healthy.

### PCR-S02A.4 — Shared client and view GREEN

- Parse every Task response at the Core boundary; mutation parse failures are
  errors, not fallback success.
- Keep React Query as server-state authority. Implement filtered, deterministic
  Task management with loading, empty, denied, conflict, and error states.
- Provide edit, assignee, due-date, reorder, complete, cancel, archive, and
  restore affordances in the shared Views surface used by Web and Desktop.

### PCR-S02A.5 — Integrated verification and independent review

- Run Section 8 checks from the fixed activation candidate.
- Execute an installed Canonical runtime journey proving create/read/filter/edit/
  reorder/complete/cancel/archive/restore, denial, conflict, and restart.
- Obtain fresh independent read-only review of specification, code quality,
  scope, security, migration safety, evidence, and dirty exclusions.
- Any deterministic failure or independent `BLOCK` stops closure and requires a
  new task/plan revision. Do not waive it into PASS.

### PCR-S02A.6 — Closure

After all criteria and independent review pass, append indexed evidence, mark
r014 and S02A complete, and leave exactly one next state: S02B inactive pending
its own frozen plan/task. Do not infer Release 1 completion from S02A.

## 7. Acceptance criteria

1. Default Canonical HTTP exposes the frozen Task route family and reports the
   tasks capability installed; unsupported empty success is impossible.
2. Member and agent assignees are validated within the workspace. Read/create/
   update-own and manage-workspace permissions have table-driven allow/deny
   coverage, including missing-provider denial.
3. A user can create, view, filter, reorder, edit, complete, cancel, archive, and
   restore Tasks through the shared Web/Desktop surface.
4. Task title, description, priority, project/Issue reference, assignee, start
   date, due date, position, status, archive state, revision, creator, and
   timestamps survive runtime restart.
5. Invalid lifecycle transitions and cross-workspace references fail without a
   domain, revision, audit, or outbox mutation.
6. Two writes with one expected revision yield one winner and one `409` loser
   with current revision. Atomic reorder rejects stale, duplicate, missing, or
   foreign items without partial changes.
7. Lists are bounded, filterable, workspace-scoped, and stably ordered with an
   immutable ID tie-break.
8. Task domain write, revision, audit, and outbox effects are atomic and replay/
   retry behavior does not duplicate a mutation.
9. Every TypeScript response is schema-parsed; malformed reads use only honest
   safe fallbacks and malformed mutations fail visibly.
10. Shared UI renders explicit loading, empty, denied, conflict, and error states
    and does not duplicate server state in a view-owned store.
11. Focused and full deterministic checks pass. Race evidence is a real PASS or
    is reported as an environment limitation; it is never silently waived.
12. The candidate contains only authorized paths, no `server/**` change, and no
    byte change to the frozen dirty/untracked exclusions.
13. Fresh independent review returns PASS before closure.

## 8. Deterministic verification

Run from repository root unless a command changes directory:

```powershell
git rev-parse HEAD
git diff --name-status 628996378af6fbe12c27a916a624a5f5374a884f...HEAD
git diff --quiet 628996378af6fbe12c27a916a624a5f5374a884f -- server
git status --porcelain -- server
git diff --quiet 628996378af6fbe12c27a916a624a5f5374a884f -- backend/docs/plans/product-capability-roadmap/plan_v1.md backend/docs/plans/product-capability-roadmap/plan_v2.md backend/docs/plans/product-capability-roadmap/plan_v3.md backend/docs/plans/product-capability-roadmap/plan_v4.md backend/docs/plans/product-capability-roadmap/plan_v5.md backend/docs/plans/product-capability-roadmap/plan_v6.md backend/docs/plans/product-capability-roadmap/plan_v7.md backend/docs/plans/product-capability-roadmap/plan_v8.md
cd backend
go test ./internal/modules/workspace/internal/domain/todo ./internal/modules/workspace/internal/application ./internal/modules/workspace/internal/infrastructure/sqlite ./internal/modules/workspace/internal/interfaces/http ./internal/bootstrap -count=1
go test ./... -count=1
make check
make test-race
cd ..
pnpm --filter @multica/core test -- tasks api
pnpm --filter @multica/views test -- tasks
pnpm --filter @multica/core typecheck
pnpm --filter @multica/views typecheck
pnpm --filter @multica/web typecheck
pnpm --filter @multica/desktop typecheck
pnpm typecheck
pnpm test
pnpm exec playwright test e2e/tasks.spec.ts --project=chromium --reporter=line
```

Before closure, compare SHA256 or Git blob identity for every Section 5 dirty
exclusion, mechanically enumerate active task markers, validate roadmap links,
confirm the v9 activation commit has exact r014 trailers and direct base parent,
and preserve raw sanitized runtime/browser evidence under an authorized evidence
path. A focused PASS is not a full-suite or release PASS.

## 9. Risks and stop conditions

| Risk | Required response |
| --- | --- |
| Existing Todo leaks as a second public product | keep public Task naming and add compatibility contract tests |
| Last-write-wins survives in one adapter | require expected revision at every mutation boundary and concurrency tests |
| Archive is implemented as destructive delete | stop; preserve terminal state and restore semantics |
| Governance envelopes expose body/secrets | stop before commit and narrow the allowlisted policy |
| Capability flag leads runtime installation | keep flag false until installed acceptance passes |
| Shared dirty UI path becomes necessary | stop and create a new approved plan version |
| Full suite reproduces unrelated failure | record exact evidence; do not call Release/S02A PASS until disposition is authorized |
| Windows race loader returns `0xc0000139` | record environment limitation; never call it a race PASS |
| Any `server/**` path appears | invalidate the candidate before further checks |

## 10. Rollback

- Before activation commit: remove only uncommitted v9 authority documents.
- After activation but before product commit: revert the v9 activation commit;
  prior immutable plans remain unchanged and Release 1 becomes inactive.
- After product commit: revert only r014 commits in reverse order. Do not reset or
  overwrite user dirty paths.
- The down migration may proceed only when no Task row uses revision/archive
  state that the old schema cannot represent. Otherwise it must fail without
  modifying data and recovery uses the pre-migration database backup.
- Stop owned local runtime processes and verify owned ports are quiescent after
  acceptance. No deployment rollback is authorized because v9 does not deploy.

