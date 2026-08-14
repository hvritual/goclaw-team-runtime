# Canonical SQLite runtime cutover — execution plan v6

- Plan-ID: `canonical-sqlite-runtime-cutover`
- Version: `6`
- Status: `approved`
- Approval source: Human Customer confirmation dated `2026-08-14`
- Supersedes: `plan_v5.md`
- Base commit: `f172418`
- Branch and integration target: `codex/multica-six-domain-baseline`
- Proposed active step: `M1-S7-C3`
- Policy bundle: `backend-v1`

## Confirmed defect

The installed Project detail route
`/drcoffee/projects/0631ebaf-d1e0-42bd-a03b-1da3f8110dd5` renders, but the
normal embedded Issue surface and its visible create action issue unsupported
requests:

- first render: `GET /api/properties` -> 404;
- first render: `GET /api/issues/child-progress` -> 404;
- opening Issue create: `GET /api/labels?resource_type=issue` -> 404;
- submitting Issue create: `POST /api/issues` -> 404.

The Web log records the exact route, request paths and 404 toast at `09:32` on
`2026-08-14`. Project list/get/member/pin requests from plan v5 remain 200.

## Proposed vertical slice

This correction permits only:

- a trusted, Workspace-scoped `POST /api/issues` HTTP adapter over the existing
  Canonical SQLite Issue use case and realtime publisher;
- exact snake_case request/response compatibility for the title-first Project
  Issue create journey;
- Bearer and Cookie-CSRF behavior, strict bounded JSON, tenant isolation,
  project/actor validation, commit-before-event, restart readback and failure
  no-event evidence;
- capability-aware query/render gates so false `issue_labels`,
  `issue_properties`, `issue_child_progress`, and `issue_attachments` never
  mount requests or controls in shared Issue surfaces/create UI;
- an explicit `issue_create` capability that moves together with HTTP route
  registration;
- Core/View/runtime tests, the existing Canonical E2E, verifier capability
  matrix, and this plan directory/journal.

`server/**` remains permanently read-only. No Label, Property, child-progress,
attachment, Issue update/move/batch, comment, timeline, or full detail API is
added. False capabilities must be hidden/disabled rather than answered with
fabricated empty success.

## Invariants

- Authenticate before resolving Workspace identity; missing/expired identity
  is 401, missing Workspace input is 400, and foreign/missing Project targets
  are hidden 404.
- Cookie mutation requires the accepted S2 HMAC CSRF token; Bearer mutation is
  CSRF-exempt. Caller-supplied identity headers are never trusted.
- Create returns the exact public Issue object expected by
  `CreateIssueResponseSchema`; the persisted Issue carries the selected
  canonical Project ID and trusted creator actor.
- Unsupported non-empty label/attachment/property input is rejected before
  persistence. Capability-gated UI does not send it.
- The Issue realtime event is emitted only after successful SQLite commit;
  validation/persistence failure emits no event.
- Existing Project, Issue metadata, selector, rollback, database and log
  evidence remains intact. No database migration is required.

## Ordered execution

1. `M1-S7-C3-RED`: add runtime/View/E2E tests proving the four observed 404s
   and the visible create failure.
2. `M1-S7-C3-GREEN`: add the minimal Issue-create HTTP adapter and complete
   capability gating for the disabled auxiliary surfaces.
3. `M1-S7-C3-INTEGRATE`: run focused/full gates, independent review, exact
   clean-candidate installed-Chrome Project-detail/create/reload journey, and
   preserve unrelated dirty paths.
4. `M1-S7-C3-ACCEPT`: record Human Customer acceptance separately from
   technical PASS.

Only one substep may be active at a time. Product writes start only after Human
approval changes this plan to approved and `plan.md` points to version 6.

## Expected write boundary after approval

- `backend/internal/modules/workspace/internal/interfaces/http/**`
- `backend/internal/modules/workspace/issue_read_extension.go`
- `backend/internal/modules/workspace/sqlite_workspace_chain.go`
- `backend/internal/bootstrap/runtime.go`
- focused tests under the same Canonical packages
- `packages/views/issues/**`, `packages/views/modals/create-issue.tsx`
- `packages/core/api/**` only if exact create parsing needs correction
- `scripts/canonical-runtime-verifier*`, `e2e/canonical-runtime.spec.ts`
- `backend/docs/plans/canonical-sqlite-runtime-cutover/**`

Explicit exclusions remain `server/**`, the user's unrelated
`packages/ui/components/ui/input.tsx`,
`packages/views/auth/input-controlled.test.tsx`, and local artifact roots.

## Acceptance

1. Project detail first render makes no request to disabled Property,
   child-progress, Label, or attachment routes and shows no 404 toast.
2. The visible create control submits a title-first Issue assigned to the
   current Project, returns 201 with an exact Issue, appears in the surface,
   survives reload/restart, and emits one `issue:created` event.
3. Cookie-CSRF, Bearer, missing/expired identity, missing Workspace, foreign
   Project, unknown/trailing/oversized body, unsupported auxiliary fields,
   persistence rollback and no-event failure cases are executable.
4. Capability and route registration cannot disagree; disabling Issue create
   removes the route and all create controls.
5. Focused/full checks, zero tracked/untracked `server/**`, independent review,
   clean-candidate Chrome verification and explicit Customer acceptance pass.

## Deterministic verification

From `backend/`:

```text
go test ./internal/modules/workspace ./internal/bootstrap -count=1
go test ./... -count=1
go vet ./...
go mod verify
```

From repository root:

```text
pnpm --filter @multica/core test
pnpm --filter @multica/core typecheck
pnpm --filter @multica/core lint
pnpm --filter @multica/views test -- <focused Issue surface/create tests>
pnpm --filter @multica/views typecheck
pnpm --filter @multica/web typecheck
node --test scripts/runtime-selector.test.mjs scripts/canonical-runtime-verifier.test.mjs
git diff --check
git status --porcelain -- server
```

## Rollback

Stop Canonical and revert only the v6 correction. No schema downgrade or data
deletion is required. Preserve the canonical database, WAL/SHM files, logs and
the v5 Project/Pin implementation. Reselect/restart the last accepted runtime
only after readiness succeeds.
