# P6-S2B1-V9 Issue metadata vertical slice

- Plan-ID: `server-function-migration`
- Version: `9`
- Status: `approved`
- Approval source: user confirmations dated 2026-08-13 to execute the staged
  Canonical backend migration, allow API URL changes when necessary, and keep
  request and response bodies unchanged
- Parent plan: `plan_v8.md`
- Active step: `P6-S2B1-V9`
- Base commit: `e4b4b1c7e3d46b19fb4774f8757cad4fb4c4f1cc`
- Branch: `codex/issue-metadata-v9`
- Repository: `F:\code\ai\goclaw-team-runtime`
- Project: `goclaw-team-runtime`
- Project-ID: `goclaw-team-runtime`
- Assignee: current control thread; implementation and acceptance reviews are
  independent tasks
- Policy bundle: `401b48d143b41e0715a8d4655abddfe0d8c15ff877db8620d2bbc93c6939a598`
  - `AGENTS.md`: `637c5ff1222ba462b3b3ff96c74e4ad0b62f52bfa086d76c396d99badb9848e0`
  - `CLAUDE.md`: `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646`
  - `backend/AGENTS.md`: `fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209`

## Goal

Port the legacy Workspace-scoped per-Issue metadata KV behavior from the
read-only `server/**` evidence tree into the Canonical `backend/**` Workspace
module, expose a body-compatible HTTP boundary, and connect the existing shared
frontend API and read-only Issue detail projection. This version supersedes the
backend-only and no-HTTP limits of plan v8 for this one vertical slice.

## Frozen compatibility contract

The indexed source/target matrix is
[issue-metadata-parity-v9.md](issue-metadata-parity-v9.md).

- The installed URLs are preferred; a URL change is allowed only when the
  canonical transport requires it and must remain centralized in
  `packages/core/api/client.ts`.
- PUT request JSON remains exactly `{"value": <string|number|boolean>}`.
- GET, successful PUT, and successful DELETE responses remain exactly
  `{"metadata": {<key>: <string|number|boolean>}}`.
- Error JSON remains `{"error":"<message>"}`. Status and message compatibility
  are acceptance-tested for the declared matrix.
- `workspace_id` is transport identity, not a request-body field. The canonical
  adapter obtains it from the established workspace request context/header.
- Issue ID accepts UUID or Workspace-scoped identifier. Missing and foreign
  Workspace Issues are indistinguishable publicly.
- Metadata is a flat object. Keys match
  `^[a-zA-Z_][a-zA-Z0-9_.-]{0,63}$`; values are JSON string, number, or bool;
  `null`, objects, arrays, missing values, and invalid JSON are rejected.
- Maximums are 50 keys and 8 KiB of compact UTF-8 JSON. Replacing an existing
  key at the count limit succeeds.
- Mutations are single-key atomic. Delete of an absent key succeeds, refreshes
  `updated_at`, and returns the complete bag. Distinct concurrent writes do not
  lose each other; same-key writes are last-committer-wins.
- Malformed historical metadata reads as `{}` and the next valid mutation
  repairs it. Returned maps are defensive copies.
- A successful mutation emits the complete `issue_metadata:changed` snapshot
  with `issue_id` and `metadata`; ordering and self-event behavior follow the
  existing realtime contract.
- `UpdateIssue` cannot write metadata or properties from a stale projection.

## Scope and write boundary

Allowed product paths for this step:

- `backend/api/workspace/v1/issue.proto` and its generated canonical outputs;
- Workspace metadata domain, application, SQLite, local, gRPC, and HTTP
  compatibility adapters under `backend/internal/modules/workspace/**`;
- opt-in Workspace SQLite composition and focused backend tests;
- `packages/core/types/**`, `packages/core/api/**`, and focused Issue hooks/tests;
- the existing shared `packages/views/issues/**` read-only metadata projection
  only when a failing compatibility test requires a change;
- this plan, parity matrix, and append-only journal.

`server/**` is read-only evidence and is never an allowed diff. Existing user
changes in `packages/ui/components/ui/input.tsx`,
`packages/views/auth/input-controlled.test.tsx`, `.local-runtime/`,
`docs/code-to-product/`, and `ui/` are outside this task and must be preserved.

## Non-goals

- Labels, custom properties, comments, subscribers, reactions, attachments,
  search/filter/group/facet migration, hierarchy, batch, move, or deletion.
- A new metadata editing UI; the existing Issue detail raw JSON view remains
  the only verified UI consumer.
- PostgreSQL production readiness, default runtime cutover, identity-provider
  migration, unrelated HTTP compatibility routes, or a broad realtime rewrite.
- Any schema migration, foreign key, cascade, or write below `server/**`.

## Invariants and dependencies

- P6-S2A Issue mainline and the existing opt-in SQLite Workspace chain remain
  prerequisites.
- Authorization completes before repository access. Every predicate includes
  `workspace_id`; mutations require a trusted same-Workspace actor.
- Existing protobuf field numbers and service contracts are not renamed or
  renumbered. Generated code is produced only by repository scripts.
- React Query continues to own Issue server state. URL knowledge stays in the
  core API client; views and app shells do not branch on backend paths.
- The default generated-stub runtime remains unchanged unless a separately
  approved cutover plan replaces it.

## Ordered execution

1. `P6-S2B1-V9-1` — freeze current base, policy hashes, dirty-tree exclusions,
   source evidence, and this immutable plan/matrix.
2. `P6-S2B1-V9-2` — add backend and core compatibility/characterization tests;
   record the expected RED result before product implementation.
3. `P6-S2B1-V9-3` — implement domain/application/SQLite atomic behavior and
   remove stale metadata/property writes from Issue mainline.
4. `P6-S2B1-V9-4` — add generated local/gRPC boundaries, the body-compatible
   HTTP adapter, and opt-in SQLite composition.
5. `P6-S2B1-V9-5` — connect the core API client and existing read-only frontend
   consumer without adding a new editing surface.
6. `P6-S2B1-V9-6` — run deterministic gates, independent spec review, then
   independent code-quality review; correct and repeat gates before acceptance.

Only one product writer may be active. Steps 3-5 may be implemented as one
strict RED/GREEN vertical increment after step 2 establishes the failure.

## Acceptance criteria

1. The candidate diff contains no `server/**` path and does not include the
   preserved unrelated dirty files.
2. The three HTTP operations preserve the frozen request/response JSON bodies,
   primitive types, auth/workspace isolation, status codes, and public errors.
3. Get/Put/Delete work through domain, application, SQLite, local, gRPC, and
   opt-in HTTP boundaries with UUID and identifier Issue references.
4. Key/value/count/size rules, malformed legacy repair, defensive copies,
   delete no-op, rollback, and timestamp behavior are tested.
5. Concurrent distinct-key writes and overlapping `UpdateIssue` writes lose no
   metadata or properties.
6. The core client owns paths, encodes path keys, validates response bodies,
   preserves string/number/bool types, and exposes no whole-bag write.
7. The existing shared Issue detail view works for Web and Desktop and keeps
   the current empty/non-empty/read-only behavior; no duplicated app wiring.
8. Realtime payload compatibility is tested at the publishing/cache boundary,
   or is explicitly retained as a documented deferred blocker if no canonical
   durable publisher exists in this opt-in slice.
9. Default bootstrap selection and health/Ping behavior remain unchanged.
10. Deterministic verification and both independent reviews pass with indexed
    evidence in `journal.md`.

## Deterministic verification

From `backend/`, use repository scripts as the source of truth:

```sh
make fmt-check
make policy-check
make generated-clean
make test
make test-race
make vet
```

Also run focused Workspace tests with `-count=1`, generation twice and require
content idempotence, `go mod verify`, `git diff --check`, and static checks for
old generated imports, forbidden dependencies, FK/cascade SQL, default runtime
selection, and any `server/**` diff.

From the repository root:

```sh
pnpm --filter @multica/core test
pnpm --filter @multica/core typecheck
pnpm --filter @multica/views test
pnpm typecheck
pnpm test
```

Use the narrowest supported filters when package scripts differ, then run the
repository-level commands. Transport verification includes legacy-shape HTTP
tests, bufconn gRPC tests, and unchanged health/readiness/Ping probes.

## Risks and mitigations

- **Proto versus JSON mismatch:** keep the raw primitive representation inside
  canonical service boundaries and project legacy JSON only in the HTTP adapter.
- **Lost updates:** use a SQLite immediate transaction and stop mainline updates
  from writing metadata/properties.
- **Runtime illusion:** label opt-in HTTP/gRPC proof honestly; do not claim
  production cutover while default bootstrap still selects stubs.
- **Numeric fidelity:** test integer and decimal JSON values end to end; do not
  quote or coerce them in the core client or adapter.
- **Frontend scope creep:** reuse types, schemas, React Query, and the existing
  read-only view; do not invent UI without separate approval.
- **Generated drift:** use only the repository generation pipeline twice and
  inspect unrelated generated changes before acceptance.

## Rollback and stop conditions

Rollback removes the additive metadata service/adapters/core methods and
restores the pre-step Issue update assignments; no schema or production data
rollback is expected.

Stop before proceeding if a request or response body change is required, data
mapping loses type or nullability, permission/event ordering cannot be kept,
a schema migration or default runtime cutover becomes necessary, any
`server/**` change appears, the base/policy bundle drifts, or a deterministic
gate fails without an in-scope correction.
