# Canonical SQLite runtime cutover — execution plan v7

- Plan-ID: `canonical-sqlite-runtime-cutover`
- Version: `7`
- Status: `approved`
- Approval source: Human Customer confirmation `批准本地全量 plan v7` dated
  `2026-08-14`
- Supersedes: `plan_v6.md`
- Base commit: `9ab58a5204d083a757b205d512fcaa2c98c26331`
- Branch and integration target: `codex/multica-six-domain-baseline`
- Active step: `M1-S7-C4-RED`
- Policy bundle: `backend-v1`
- XP mode: strict; maximum active stories: one

## Approved change intent

The current Canonical Issue list and detail routes are only partially usable:

- the Web log proves `POST /api/issues/:id/move` returns 404 during a normal
  board/table move;
- `IssueDetail` replaces the installed detail page with a title/description-only
  projection whenever `issue_timeline=false`, producing the reported blank
  experience;
- the current frontend already owns exact request and response contracts for
  core update/move, hierarchy and batch operations, comments/timeline,
  reactions, subscribers, labels, custom properties, acceptance conclusions,
  attachments and the supporting member/project/pin projections;
- the existing Canonical Issue domain, SQLite repository and realtime publisher
  implement part of the write path, but most user-owned HTTP and persistence
  boundaries are absent;
- public member actor references in the current Web contract are Auth `user_id`
  values, while parts of the Canonical implementation currently expose or
  validate the private membership-row ID. This must be corrected without
  weakening authorization.

The Human Customer selected the complete local Issue-detail scope. This plan
therefore replaces the earlier base-detail capability gates with real Canonical
local behavior. It does not fabricate empty responses and does not route any
accepted request to the legacy server.

## Evidence and target authority

1. The installed Multica frontend at the base commit is the public interface
   authority: endpoint methods, snake_case bodies, Zod schemas, query keys,
   visible controls and realtime payload consumers define the target contract.
2. `server/**` is permanently read-only compatibility evidence. Its handlers,
   migrations and SQLite-local behavior may be characterized and independently
   ported into `backend/**`; no source, test or generated file there may change.
3. `origin/agent/tc-w01-team-control-001` is an ancestor of the current branch;
   its Canonical Issue core is already present. The separate root `teamcontrol`
   Issue model is project-governance evidence only. Its transition,
   authorization, assignment, Artifact and correlation behavior may inform
   target rules, but its distinct fields/statuses/storage are not a Multica API
   or database contract and will not be copied.
4. Existing accepted v1-v6 behavior remains a regression boundary: runtime
   selection, Auth, Workspace selection/onboarding, Project/Pin, Issue list and
   create, metadata, deletion, realtime, restart and rollback must remain green.

## Approved local capability boundary

### Issue core and hierarchy

- `PUT /api/issues/:id` exact update response;
- `POST /api/issues/:id/move` using workspace-scoped relative anchors and a
  server-owned canonical position;
- `GET /api/issues/:id/children`, `GET /api/issues/children`, and
  `GET /api/issues/child-progress`;
- `POST /api/issues/batch-update` and `POST /api/issues/batch-delete`;
- title, description, status, priority, member/agent assignee, Project,
  parent, stage, start date, due date and attachment-reference updates;
- existing create/get/list/table/delete/metadata behavior remains exact.

### Collaboration and timeline

- Issue timeline plus comment list/create/update/delete;
- replies, resolve/unresolve and comment knowledge proposals;
- comment reactions and Issue reactions;
- Issue subscriber list/subscribe/unsubscribe;
- activity records for committed Issue and relationship changes;
- current Core websocket event shapes and query-cache behavior.

### Labels, properties and acceptance

- Workspace Issue-label catalog list/create/read/update/delete;
- Issue attach/detach/list with complete post-mutation label bag;
- Workspace custom-property catalog list/create/read/update;
- atomic set/unset of Issue property values with complete post-mutation bag;
- acceptance-conclusion list/create and completion/capture behavior;
- exact current frontend schemas, nullable/default behavior and validation.

### Attachments

- Canonical local Space asset metadata persistence and file ownership;
- multipart upload, Issue/Comment binding, list/get, safe text preview,
  download and delete;
- bounded size/MIME handling, opaque server-owned paths, path-traversal
  resistance and cleanup in the owning application transaction;
- restart and rollback preserve both SQLite metadata and retained files.

### Existing projections enabled with the detail page

- Auth-backed member list with public `user_id` actor references;
- Project list/get and Pin list/create/delete from the accepted v5 slice;
- full Issue detail shell, metadata, hierarchy and collaboration controls;
- capability and route registration move together.

## Explicit non-goals

- GitHub installation, repository synchronization, Pull Request ingestion and
  other external VCS integrations remain capability-off. They require separate
  external credentials and were not part of the approved local-full request.
- Inbox, invitations, Skills administration, Tasks, Knowledge browsing,
  production PostgreSQL, desktop/mobile packaging and legacy retirement are
  not added by this plan.
- Comment knowledge proposals may create the accepted local evidence/candidate
  boundary, but this plan does not add a new external AI/agent execution path.
- No API is answered with a fabricated empty success merely to suppress a 404.

## Ownership and architecture invariants

### Identity and authorization

- Authenticate before Workspace resolution. Missing/expired identity is 401;
  missing Workspace input is 400; missing/foreign Issue or dependent resource
  is hidden 404.
- Cookie mutations require the accepted S2 HMAC CSRF token; Bearer mutations
  are CSRF-exempt. Request actor/user headers are never trusted.
- Auth owns users, sessions, membership IDs and membership roles. Public
  `member` actors in Issue bodies/events use Auth `user_id`; private member-row
  IDs remain authorization implementation details.
- Every repository predicate includes the canonical Workspace ID. Resource
  authorization is derived from stored ownership, never a caller-supplied
  foreign Workspace or Project ID.

### Data ownership

- Workspace owns Issues, hierarchy, activity, comments, reactions,
  subscribers, labels, property definitions/values, acceptance conclusions
  and Issue/Comment-to-asset references.
- Space owns asset metadata and file bytes. Workspace receives a narrow asset
  reader/binder contract and never writes Space tables or file paths directly.
- Auth/Workspace/Space share the one Canonical product `*sql.DB` lifecycle but
  retain separate migration catalogs and public module boundaries.
- No foreign key or cascading action is introduced. All dependent cleanup is
  explicit, workspace-scoped and transactional.
- Under the current repository migration policy, SQLite migrations do not add
  explicit `CREATE INDEX` statements. Primary/unique table constraints own
  required identity; query-plan evidence is recorded for accepted local scale.

### Transaction and event ordering

- Move reads anchors, computes position and writes the Issue inside one
  `BEGIN IMMEDIATE` transaction. Invalid/stale/cross-workspace anchors fail
  without changing any row.
- Multi-row Issue, hierarchy, label, property, acceptance, comment and asset
  operations commit completely or roll back completely.
- Realtime publication occurs only after a successful commit. A failed or
  rolled-back operation emits no event.
- Per-workspace event payloads use the existing Core event union and are
  schema-validated before cache consumers run. Duplicate delivery is tolerated;
  reconnect performs authoritative refetch.
- Deletion cleans every owned dependent row/reference in the same application
  transaction while preserving foreign-Workspace data.

### HTTP and file boundaries

- JSON mutation bodies are bounded before allocation, reject unknown fields
  and trailing values, and preserve frozen status/error semantics.
- Multipart/file reads are bounded before buffering; filenames never determine
  storage paths; downloads use canonical metadata and membership checks.
- List ordering, pagination/full-list semantics and exact response envelopes
  match current Core schemas. `[]` is never encoded as `null`.
- Capability flags are true only when the complete mounted route family is
  registered and its focused acceptance tests pass.

## Ordered stories

Only one story and one substep may be active. Every story follows
`Story Ready -> Test Ready -> RED Proven -> GREEN Proven -> Refactor Safe ->
Integrated -> Navigator Reviewed -> Customer Accepted` before promotion.

### M1-S7-C4 — Core update, move and actor identity

1. `C4-RED`: reproduce move 404, detail mutations 404 and public member actor
   mismatch through real Runtime/Core/View tests.
2. `C4-GREEN`: add trusted update/move HTTP adapters, atomic relative movement,
   public user-ID actor normalization and exact post-commit events.
3. `C4-INTEGRATE`: focused/full gates, restart/concurrency/rollback evidence,
   independent review and a browser move/edit proof.
4. `C4-ACCEPT`: record Human Customer acceptance before C5.

### M1-S7-C5 — Hierarchy and batch operations

1. `C5-RED`: children, child-progress and visible batch controls fail for the
   intended missing routes.
2. `C5-GREEN`: add exact hierarchy reads, cycle-safe reparenting, batch
   update/delete and transactional dependent cleanup.
3. `C5-INTEGRATE`: hierarchy/isolation/concurrency/restart/browser evidence and
   independent review.
4. `C5-ACCEPT`: record Human Customer acceptance before C6.

### M1-S7-C6 — Timeline, comments, reactions and subscribers

1. `C6-RED`: prove every mounted collaboration route/event is missing without
   accepting test-only empty data.
2. `C6-GREEN`: add Workspace collaboration migrations, use cases, HTTP routes,
   events, strict Core payload schemas and visible timeline behavior.
3. `C6-INTEGRATE`: ordering, reply/cycle, moderation, idempotency, concurrency,
   reconnect, restart and browser evidence plus independent review.
4. `C6-ACCEPT`: record Human Customer acceptance before C7.

### M1-S7-C7 — Labels, properties and acceptance conclusions

1. `C7-RED`: prove catalog, attach/detach, set/unset and conclusion routes are
   missing from the real detail interactions.
2. `C7-GREEN`: add definitions/relationships/value/conclusion persistence,
   strict validation, atomic bags and post-commit events.
3. `C7-INTEGRATE`: type/value matrices, uniqueness, delete cleanup,
   isolation/concurrency/restart/browser evidence and independent review.
4. `C7-ACCEPT`: record Human Customer acceptance before C8.

### M1-S7-C8 — Canonical Space attachments

1. `C8-RED`: prove upload/list/preview/download/delete and Issue/Comment binding
   are absent through bounded real-file tests.
2. `C8-GREEN`: implement Space SQLite/file providers, trusted HTTP boundaries,
   binding/cleanup transactions and exact Core schemas.
3. `C8-INTEGRATE`: size/MIME/path/security, orphan/rollback, restart, hash and
   browser upload/preview/download/delete evidence plus independent review.
4. `C8-ACCEPT`: record Human Customer acceptance before C9.

### M1-S7-C9 — Full local Issue-detail cutover

1. `C9-RED`: mount the full current detail page and inventory every emitted
   request/control/event; any local capability 404 or hidden required control
   is a failure.
2. `C9-GREEN`: enable the complete local capability matrix and refactor the
   page so external PR integration alone remains explicitly disabled.
3. `C9-INTEGRATE`: run all deterministic gates and a clean-candidate installed-
   Chrome journey covering edit, move, hierarchy, collaboration, labels,
   properties, acceptance, attachments, realtime, reload and restart with no
   port-8080 traffic.
4. `C9-ACCEPT`: independent final review, rollback preservation proof and
   explicit Human Customer milestone acceptance.

## Expected write boundary

- `backend/internal/modules/workspace/**`
- `backend/internal/modules/space/**`
- `backend/internal/modules/auth/**` only for narrow public-user/member actor
  projection needed by the approved Issue contract
- `backend/internal/bootstrap/**`, `backend/cmd/server/**`
- focused tests under the same Canonical packages
- `packages/core/api/**`, `packages/core/types/**`, `packages/core/issues/**`,
  `packages/core/labels/**`, `packages/core/properties/**`
- `packages/views/issues/**`, `packages/views/labels/**`,
  `packages/views/properties/**`, and the Issue-create modal only where the
  exact shared detail contract requires it
- `scripts/canonical-runtime-verifier*`, `e2e/canonical-runtime.spec.ts`
- `backend/docs/plans/canonical-sqlite-runtime-cutover/**`

Explicit exclusions remain `server/**`, root legacy backend trees, external
VCS/GitHub integration code, the user's unrelated
`packages/ui/components/ui/input.tsx`,
`packages/views/auth/input-controlled.test.tsx`, and local artifact roots
`.local-runtime/`, `docs/code-to-product/`, and `ui/`.

## Story acceptance matrix

Every in-scope endpoint must prove:

1. exact method/path/request/response/schema, ordering and empty shape;
2. Bearer and Cookie-CSRF success plus missing/expired identity;
3. missing Workspace input, slug/ID mismatch and cross-Workspace hidden result;
4. bounded malformed/unknown/trailing/oversized input;
5. validation, duplicate/idempotent, persistence failure and rollback behavior;
6. transaction/concurrency behavior with no lost unrelated metadata,
   properties, references or files;
7. post-commit event, failed-write no-event, duplicate tolerance and reconnect;
8. retained database/file restart readback and dependent cleanup;
9. Core schema rejection of malformed success and View loading/empty/denied/
   error states;
10. browser interaction and sanitized network/WS evidence with no missing local
    route and no legacy runtime request.

## Deterministic verification

From `backend/` after each relevant story:

```text
gofmt -d <changed Go files>
go test ./internal/modules/auth ./internal/modules/workspace ./internal/modules/space ./internal/bootstrap ./cmd/server -count=1
go test ./... -count=1
go vet ./...
go mod verify
```

The Windows race binary may be attempted only as supplemental evidence. Exit
`0xc0000139` is an environment limitation and is never recorded as a pass.

From repository root:

```text
pnpm --filter @multica/core test
pnpm --filter @multica/core typecheck
pnpm --filter @multica/core lint
pnpm --filter @multica/views test -- <focused Issue tests>
pnpm --filter @multica/views typecheck
pnpm --filter @multica/web typecheck
node --test scripts/runtime-selector.test.mjs scripts/canonical-runtime-verifier.test.mjs
git diff --check
git status --porcelain -- server
```

Final C9 additionally requires the repository selector/verifier, retained-data
restart, quiescent rollback artifact hashes and clean-candidate installed-Chrome
journey. Static/component evidence is not browser or release evidence.

## Rollback

- Each story is committed independently and enables its capability flags only
  after its complete route family is accepted.
- A failed story reverts only that story's product commit and keeps its new
  capability false. Never delete or rewrite the retained Canonical database,
  WAL/SHM/journal files, asset directory or logs.
- Additive tables remain inert if a story is disabled. Destructive down
  migration is not part of runtime rollback.
- C8 file writes use server-owned opaque paths and atomic rename so an aborted
  upload can be safely identified without deleting retained referenced files.
- Runtime rollback stops Canonical, snapshots database/file/log artifacts,
  selects the prior accepted runtime and verifies hashes/readback exactly as in
  the accepted selector contract.

## Risks and stop conditions

- Stop if the current frontend contract and retained evidence disagree on a
  security, tenant, body or lifecycle decision that this plan did not freeze.
- Stop if a required implementation needs a `server/**` write, external
  GitHub/VCS credential, destructive migration, new foreign key/cascade, or
  unrelated dirty path.
- Stop promotion when any capability flag can expose a control without a real
  route, when a route returns fabricated success, or when a committed mutation
  lacks post-commit/no-event evidence.
- Stop and propose v8 for a material scope, ownership, migration, external
  integration, contract or rollback change.
