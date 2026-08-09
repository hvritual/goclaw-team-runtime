# P6-S2 Issue migration — sidebar execution and control-thread review

- Plan-ID: `server-function-migration`
- Version: `7`
- Status: `approved`
- Approval source: user instruction dated 2026-08-03 to move concrete Issue
  migration tasks into sidebar threads while this thread owns planning and
  result review
- Parent plan: `plan_v6.md`
- Active implementation: `P6-S2A`
- Repository scope: `backend/` only

## Operating model

The control thread owns capability decomposition, dependency order, acceptance
criteria, review, and plan/journal updates. It does not implement Issue product
code while a sidebar task owns that slice. Sidebar implementation tasks use an
isolated worktree created from the current working-tree baseline. Because the
backend baseline is not committed and generated Workspace files are shared,
only one sidebar task may write Issue code at a time.

Read-only audit tasks may run concurrently. Their output is evidence for later
implementation briefs, not acceptance of migrated behavior.

## Current sidebar tasks

| Step | Sidebar task | Mode | Scope | Dependency |
| --- | --- | --- | --- | --- |
| P6-S2A | Issue mainline migration | write | Create/Get/List/Update/UpdateStatus, identifier allocation, main aggregate SQLite/local/gRPC | P6-S1 |
| P6-S2B-A | Metadata and catalogs audit | read-only | metadata, labels, property catalog and assignments | none |
| P6-S2C-A | Collaboration audit | read-only | comments, reactions, subscribers, timeline, pins, acceptance conclusions | none |
| P6-S2D-A | Query and hierarchy audit | read-only | search/query/group/table, hierarchy/progress, move, batch | none |
| P6-S2E-A | Assets and delete audit | read-only | attachment/Space boundary and complete dependent cleanup | none |

Queued client task identifiers:

- P6-S2A: `client-new-thread:51e376de-5241-408f-86cb-c1e8e20c715e`
- P6-S2B-A: `client-new-thread:1026651c-3b2f-44de-b45e-abf69a0f8e09`
- P6-S2C-A: `client-new-thread:45f8b8ee-f3b2-49f1-968a-2ef42bd83c26`
- P6-S2D-A: `client-new-thread:a1645966-9b55-442b-bea6-d91393bae315`
- P6-S2E-A: `client-new-thread:0109f980-63e5-4a15-8f4e-0822ee8b9017`

Resolved sidebar task identifiers:

- P6-S2A: `019fc806-c994-7031-a290-5ef7ae351575`
- P6-S2B-A: `019fc806-c98a-7b21-9be5-8cc33a1ddeb1`
- P6-S2C-A: `019fc806-c99c-7921-b12d-5ff20f36172e`
- P6-S2D-A: `019fc806-c98b-7593-9967-cec63c3901b1`
- P6-S2E-A: `019fc807-2c47-70a2-9128-f853c537b994`

## Review gate for P6-S2A

The control thread accepts P6-S2A only when all hold:

1. Scope contains only Issue mainline; no attached-object, HTTP, PostgreSQL,
   realtime, or runtime-cutover work is hidden in the change.
2. Proto evolution is additive, field numbers remain compatible, generation is
   content-idempotent, and generated files were not manually edited.
3. Identifier allocation and Issue creation are one SQLite-native transaction,
   safe under concurrent calls and rollback failures.
4. Authorization occurs before persistence; every repository predicate includes
   Workspace ID; Project, parent, Actor, and Asset checks use public ports.
5. Domain rules preserve installed status/priority and optional-clear behavior;
   mainline does not claim ownership of metadata/property/label mutations.
6. Domain/application/provider/local/gRPC tests cover the declared RPCs,
   concurrency, filters/order, tenant hiding, reference validation, and rollback.
7. Buf, race, contract, full test, vet, module verification, architecture,
   no-FK, formatting, scope, generation, and live-runtime gates pass.
8. Default bootstrap still selects stubs and PID/health behavior is unchanged.

Failed review returns to the same sidebar task with concrete findings. Accepted
work becomes the baseline for the next write task.

## Later write-task order

Audit results will be converted into new implementation tasks in this order:

1. P6-S2B1 — per-Issue metadata KV.
2. P6-S2B2 — Workspace label/property catalogs and Issue assignments.
3. P6-S2D1 — hierarchy, children/progress, and move ordering.
4. P6-S2D2 — search/list/query/group/table/facet projections.
5. P6-S2C1 — Issue reactions and subscribers.
6. P6-S2C2 — comments, comment reactions, timeline/activity, and pins.
7. P6-S2C3 — acceptance conclusions and Knowledge evidence handoff.
8. P6-S2E1 — Space-backed attachments and business associations.
9. P6-S2E2 — atomic Issue deletion and complete dependent cleanup.
10. P6-S2D3 — batch update/delete after single-item semantics are accepted.

Each write task gets a fresh sidebar thread from the latest accepted baseline.
No later step may edit Issue code concurrently with an earlier write step.

## Result review format

For every sidebar result, the control thread reports:

- declared versus actual scope;
- preserved invariants and ownership;
- findings by severity with file/line evidence;
- tests independently rerun;
- generated/runtime evidence;
- acceptance, correction request, or explicit block;
- next sidebar task activated.
