# Canonical SQLite runtime cutover — execution plan v5

- Plan-ID: `canonical-sqlite-runtime-cutover`
- Version: `5`
- Status: `approved`
- Approval source: Human Customer confirmation dated `2026-08-14`
- Supersedes: `plan_v4.md`
- Base commit: `ec29bd7`
- Branch and integration target: `codex/multica-six-domain-baseline`
- Active step: `M1-S7-C2`
- Policy bundle: `backend-v1`

## Confirmed defect

The installed `/drcoffee/projects` page renders but its normal first load sends
three requests that Canonical does not register:

- `GET /api/projects?`;
- `GET /api/workspaces/{workspace_id}/members`;
- `GET /api/pins`.

All return 404 and surface the reported `API error: 404 Not Found` toast. The
same visible page exposes create/update/delete Project and pin/unpin actions,
so a list-only empty-response shim would leave the journey dishonest.

## Approved vertical slice

This correction permits:

- Project list/get/create/update/delete HTTP compatibility backed by the
  existing Canonical `workspace_projects` ownership and real SQLite state;
- the exact authenticated Workspace member-list projection backed by Auth's
  existing member service;
- per-user, per-Workspace Pin list/create/delete backed by a new additive
  Workspace SQLite table; reorder remains deferred because the Projects page
  does not issue it in the accepted journey;
- strict Core response schemas where the current client trusts raw Project or
  Pin success bodies;
- Project-page capability gating only if a control remains intentionally
  unsupported;
- bootstrap composition, exact runtime/Core/View tests, and installed-browser
  verification of the reported route;
- this plan directory and evidence journal.

`server/**` remains permanently read-only. No retrospectives, resources,
requirement baseline, invitations, general member administration or other
product surfaces are authorized.

## Frozen compatibility and safety

- Every request resolves the trusted session/Bearer identity before Workspace
  or body data. `X-Workspace-Slug` selects only an authorized Workspace.
- Cookie mutations require the existing S2 token-bound CSRF validation;
  Bearer has priority and remains CSRF-exempt.
- Project success JSON uses the Core snake_case shape and exact status codes:
  list 200 `{projects,total}`, get/update 200, create 201, delete 204.
- Member list is a top-level ordered array with the exact Core member fields.
- Pins are a top-level ordered array; create returns 201, duplicate returns
  409 `item already pinned`, delete is idempotent 204. Only `issue` and
  `project` targets in the selected Workspace are accepted.
- Missing/expired identity is 401; missing Workspace input is 400; foreign or
  missing Project/Pin targets are hidden with 404.
- Bodies are bounded, strict and reject unknown/trailing JSON. Project and Pin
  writes are retained across restart. Project deletion cleans dependent pins
  in the same application transaction and preserves existing dependent-cleanup
  rules.
- No foreign keys or cascades are introduced. SQLite uniqueness is expressed
  by table constraints without a new explicit index.

## Story acceptance

1. RED proves all three first-load requests return 404 in the current runtime.
2. The Projects page loads without any 404 request/toast and renders an empty
   or persisted Project list with real member and Pin data.
3. Create, update and delete Project work through the visible page/API with
   exact bodies, isolation, CSRF and restart evidence.
4. Pin/unpin persists per user and Workspace, rejects duplicates/foreign
   targets, and is cleaned when its Project is deleted.
5. Missing, expired and foreign identities; malformed bodies; rollback and
   concurrent duplicate Pin cases are executable and deterministic.
6. Focused/full checks, no tracked/untracked `server/**`, independent review,
   installed-Chrome verification and Human Customer acceptance pass.

## Deterministic verification

From `backend/`:

```text
go test ./internal/modules/auth ./internal/modules/workspace ./internal/bootstrap -count=1
go test ./... -count=1
go vet ./...
go mod verify
```

From repository root:

```text
pnpm --filter @multica/core test
pnpm --filter @multica/core typecheck
pnpm --filter @multica/core lint
pnpm --filter @multica/views test -- projects-page.test.tsx
pnpm --filter @multica/views typecheck
pnpm --filter @multica/web typecheck
git diff --check
git status --porcelain -- server
```

## Rollback

Stop Canonical and revert only this slice. The additive Pin table may remain
unused; no destructive down migration or retained Project/User data deletion
is required. Existing DB and logs remain preserved.
