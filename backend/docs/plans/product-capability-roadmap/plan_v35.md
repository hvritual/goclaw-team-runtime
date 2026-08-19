# Product capability roadmap v35 — S07D governed Retrospectives

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Plan-Version: `35`
- Task-Revision: `r040`
- Work-Item: `PCR-S07D`
- Exact base: `1d515efcca0919eed1e8a811c53d015efa89dfa3`
- Predecessor reviewed candidate: `47ee4189cb5571ec38ae39480c758d4decad22bd`
- Predecessor reviewed tree: `d0b7d56b65964e1559e3bbe33aa734f70e2f8eca`
- Predecessor plan: `plan_v34.md`
- Predecessor plan hash: `81a865472383e853052bf8a904f66591e473f1f4845924296792f4ea0676641f`
- Status: `approved-active`
- Authority: the Human Customer's confirmed continuous Release 3 direction,
  confirmed prerequisite minimal outline authority, and repeated confirmed
  execution after the v34/r039 independent dual-PASS closure

## Predecessor and activation boundary

Exact r039 candidate `47ee4189cb5571ec38ae39480c758d4decad22bd`
passes the frozen deterministic, complete backend, official race, strict
frontend, production-build, fresh installed, scope, trailer, dirty-tree,
process, and independent-review gates. Fresh review returns `SPEC PASS` and
`CODE/SECURITY/QUALITY PASS`; governance closure
`1d515efcca0919eed1e8a811c53d015efa89dfa3` marks PCR-S07C
`complete-independent-reviewed`.

r040 starts only from that exact closure and activates only PCR-S07D. Release 3
completion remains inactive until S07D itself closes and a later aggregate
DoneGate is frozen. S10, every later story, push, merge, deployment, generated
protobufs, original dirty paths, legacy backend writes, and `server/**` remain
excluded.

## Frozen product contract

### Canonical authority and routes

- Canonical Workspace owns Retrospective identities, immutable revisions,
  participant-role snapshots, action-item target claims/links, governance,
  audit, and outbox evidence. It never reads or writes `server/**` authority.
- Retain and complete the frozen route family:
  - `GET/POST /api/projects/{project_id}/retrospectives`;
  - `GET/PUT/DELETE /api/projects/{project_id}/retrospectives/{retrospective_id}`;
  - `POST /api/projects/{project_id}/retrospectives/{retrospective_id}/action-items/{action_item_id}/target`.
- Trusted identity and Workspace resolution supply every ownership field.
  Client Workspace, Project, actor, target ID, revision, role, and link claims
  are never accepted as authority.
- Reads use stable `updated_at DESC, id DESC` ordering. List limit defaults to
  50 and is capped at 100; continuation is an opaque signed cursor. Disabled or
  uninstalled behavior returns an explicit unavailable problem, never a
  successful empty envelope.

### Content, participants, and revision semantics

- A Retrospective content snapshot contains a nonempty summary, normalized
  successes, problems, and at least one lesson, plus at most 100 structured
  action items. Each action item has a stable server-owned ID, nonempty title,
  optional description, optional active Workspace-member assignee, optional due
  date, and no target ID supplied by the client.
- A participant snapshot contains at most 100 unique active Workspace members.
  Exact roles are `participant` and `facilitator`. The creator is projected as
  a participant when omitted. Removed members are never treated as currently
  authorized even when retained in historical revisions.
- Initial `POST` creates revision 1 in `draft`. `PUT` requires
  `expected_revision` and one exact action:
  - `save_draft` appends a new draft snapshot and is allowed only while the
    Retrospective is draft;
  - `publish` appends the first published snapshot from the current draft;
  - `publish_revision` appends a complete new published snapshot while the
    Retrospective is published.
- Every successful content or lifecycle mutation advances the current revision
  by exactly one and appends a complete immutable snapshot. Published content,
  participants, action-item IDs, authorship, and timestamps are never updated
  or deleted in place. An older published revision is projected as
  `superseded` only after a later published revision exists.
- Once an action item has a target link, later published revisions must retain
  that stable action-item ID and its target-defining title, description,
  assignee, and due date. Removing or materially rewriting a linked action item
  fails before any revision advances.
- `DELETE` requires `expected_revision`, appends an archived lifecycle snapshot,
  and sets the current Retrospective status to `archived`. It never hard-deletes
  revisions, participants, action links, Tasks, or Issues. Archived records are
  excluded by default and are readable only through explicit archived/detail
  reads.

### Authorization

- Every read and write requires current active Workspace membership and exact
  Project ownership. Agents receive no implicit Retrospective permission.
- Any active member may create a draft. Draft content may be changed only by its
  creator, an active current facilitator, the current Project lead, or an
  owner/admin.
- Only an owner/admin, the current Project lead, or an active member explicitly
  designated facilitator in the current snapshot may publish, publish a new
  revision, archive, or materialize an action item.
- Only an owner/admin or current Project lead may assign or remove the
  facilitator role. An ordinary member cannot self-appoint or appoint another
  facilitator. Participant-only changes still validate every current member.
- Every mutation re-reads membership, Workspace role, Project ownership and
  lead, participant authority, Retrospective state, and expected revision on
  the same connection after `BEGIN IMMEDIATE` and before effects. Membership
  removal, lead reassignment, facilitator removal, foreign Project identity,
  and stale revisions fail with no partial domain, revision, claim, link,
  audit, outbox, or idempotency success row.
- HTTP controls are projections of server-returned access booleans; shared UI
  role guesses never grant authority.

### Action-item target creation and idempotency

- Omitting `target_kind` on the target command means `task`. `issue` is the only
  explicit alternative. Unknown, blank-explicit, or client-supplied target IDs
  fail validation.
- Create-draft and action-item target commands require an `Idempotency-Key`
  between 1 and 200 characters. Canonical versioned JSON is SHA-256 hashed.
  Same Workspace/action/key/hash replays the exact committed response; same key
  with a different hash returns `idempotency_conflict` without effects.
- The target command first acquires one Retrospective-owned, content-free claim
  for the current published action item. Concurrent or retried execution may
  resume the same normalized target kind, but cannot create a second target or
  switch target kind after a claim/link exists.
- Target creation uses only injected public Task/Issue creation contracts. The
  Retrospective repository must never insert, update, delete, or query Task or
  Issue tables as target authority. Task creation reuses the existing governed
  idempotent service. A non-generated private Issue-creation contract may add
  idempotent creation for this command without changing the public Issue HTTP,
  proto, generated files, or ordinary Issue-create behavior.
- A deterministic child idempotency identity derived from Workspace,
  Retrospective, action item, source revision, and normalized target kind makes
  a retry after the target commit return the same Task/Issue. Completion then
  atomically changes the claim to one immutable source link and appends
  Retrospective audit/outbox/idempotency evidence.
- A crash or target-service error never returns a linked success. A pending
  content-free claim is resumable; retry reauthorizes the actor and target
  inputs, reuses the same child target, and completes at most one link. Tests
  must prove interruption after claim, after target creation, and before link
  completion, plus two concurrent callers.
- The immutable link exposes Retrospective ID, action-item ID, published source
  revision, target kind, target ID, creator, and timestamp. It is the Canonical
  provenance link. Link creation does not mutate the published snapshot or
  delete a target when the Retrospective is archived.
- Task/Issue titles, descriptions, Project, assignee, and due date are derived
  only from the published action item. Existing Task/Issue services revalidate
  Project and assignee membership. A target-service denial or invalid target
  leaves no linked success or success replay.

### Persistence, audit, outbox, and deletion

- Add Workspace migration `000020_project_retrospectives` for Retrospective
  heads, immutable revisions, revisioned participants, and action-item
  claim/links. Use composite primary/unique keys only; add no foreign key,
  cascade, trigger, or explicit index.
- JSON is size-bounded to 128 KiB and fully domain-normalized on both write and
  persisted read. Invalid stored JSON or ownership drift fails closed without
  partial content, participant, action, or target projection.
- Audit metadata is allowlisted to IDs, revisions, state transitions, role and
  item counts, and target kind/ID. It never contains Retrospective content,
  titles, descriptions, credentials, request bodies, cookies, or tokens.
- Exact outbox events are `retrospective:drafted`,
  `retrospective:draft_saved`, `retrospective:published`,
  `retrospective:archived`, and `retrospective:action_item_linked`. They commit
  with the owning Retrospective mutation and carry only bounded safe projection.
- Project deletion performs explicit same-transaction dependent cleanup of
  current Retrospective authority, revisions, participant snapshots, and
  pending/linked action rows; it never deletes linked Tasks or Issues. Immutable
  audit/outbox evidence remains under its existing retention authority.
- The down migration succeeds only when every new domain/claim/link table and
  every r040 Retrospective governance dependency is empty. Each retained case
  blocks with exact before/after row and schema-catalog preservation.

### Core and shared UI

- Replace the disabled legacy five-field Retrospective boundary with strict
  schemas for heads, content, participants, revisions, access, action items,
  source links, cursors, and typed mutations. Malformed list/read responses
  fail safely; mutations never treat a fallback as success.
- Core query keys remain Workspace-scoped. Successful draft, publish,
  publish-revision, archive, and target commands invalidate only the relevant
  Retrospective, Project, Task/Issue, and list queries. Retryable mutations keep
  one stable key until success or an explicit changed command.
- The shared Project detail surface provides a loaded Retrospective section,
  draft editor, participant roles, immutable revision history, publish/new
  revision/archive controls from server access, structured action items,
  default Task and explicit Issue target controls, and target links.
- Update en/ja/ko/zh-Hans together. Loading, true empty, denied, malformed/error,
  draft, published, superseded, archived, pending target, and linked target
  states are distinct and accessible. Error states never render as an empty
  success.
- The earlier disabled UI claim that every Retrospective automatically enters
  Knowledge review is removed. r040 adds no Knowledge proposal, dual write,
  source-ref mutation, or Knowledge feature behavior because the frozen
  Retrospective dependency is only Task/Issue creation contracts.
- Enable both Retrospective permissions and `project_retrospectives=true` only
  after the complete installed vertical is composed. No app-specific routing,
  `next/*`, generated client, or new frontend state owner is added.

## Exact writable boundary

Governance:

- `backend/docs/plans/product-capability-roadmap/plan.md`;
- `backend/docs/plans/product-capability-roadmap/plan_v35.md`;
- `backend/docs/plans/product-capability-roadmap/story-map.md`;
- `backend/docs/plans/product-capability-roadmap/task-register.md`;
- `backend/docs/plans/product-capability-roadmap/journal.md`.

Canonical Workspace contract, domain, application, persistence, HTTP, and
composition:

- `backend/internal/modules/workspace/contract/project_retrospective.go`;
- `backend/internal/modules/workspace/internal/domain/retrospective/retrospective.go`;
- `backend/internal/modules/workspace/internal/domain/retrospective/retrospective_test.go`;
- `backend/internal/modules/workspace/internal/application/project_retrospective.go`;
- `backend/internal/modules/workspace/internal/application/project_retrospective_test.go`;
- `backend/internal/modules/workspace/internal/application/issue_usecase.go`;
- `backend/internal/modules/workspace/internal/application/issue_usecase_test.go`;
- `backend/internal/modules/workspace/internal/infrastructure/sqlite/migrations/000020_project_retrospectives.up.sql`;
- `backend/internal/modules/workspace/internal/infrastructure/sqlite/migrations/000020_project_retrospectives.down.sql`;
- `backend/internal/modules/workspace/internal/infrastructure/sqlite/project_retrospective_repository.go`;
- `backend/internal/modules/workspace/internal/infrastructure/sqlite/project_retrospective_repository_test.go`;
- `backend/internal/modules/workspace/internal/infrastructure/sqlite/project_retrospective_cleanup.go`;
- `backend/internal/modules/workspace/internal/infrastructure/sqlite/project_retrospective_project_integration_test.go`;
- `backend/internal/modules/workspace/internal/infrastructure/sqlite/issue_repository.go`;
- `backend/internal/modules/workspace/internal/infrastructure/sqlite/project_surface_repository.go`;
- `backend/internal/modules/workspace/internal/interfaces/http/project_retrospective.go`;
- `backend/internal/modules/workspace/internal/interfaces/http/project_retrospective_test.go`;
- `backend/internal/modules/workspace/project_retrospective.go`;
- `backend/internal/modules/workspace/project_retrospective_composition_test.go`;
- `backend/internal/modules/workspace/project_retrospective_migration_test.go`;
- `backend/internal/modules/workspace/issue_realtime_user.go`;
- `backend/internal/modules/workspace/sqlite_workspace_chain.go`;
- `backend/internal/modules/workspace/sqlite_workspace_services.go`;
- `backend/internal/modules/workspace/sqlite_persistence_test.go`.

Installed Runtime:

- `backend/internal/bootstrap/runtime.go`;
- `backend/internal/bootstrap/runtime_test.go`;
- `backend/internal/bootstrap/project_retrospective_runtime_test.go`.

Strict Core boundary:

- `packages/core/types/implementation-knowledge.ts`;
- `packages/core/implementation-knowledge/schema.ts`;
- `packages/core/implementation-knowledge/schema.test.ts`;
- `packages/core/implementation-knowledge/queries.ts`;
- `packages/core/implementation-knowledge/mutations.ts`;
- `packages/core/implementation-knowledge/mutations.test.ts`;
- `packages/core/api/client.ts`;
- `packages/core/api/issue-update-schema.test.ts`.

Shared Views and locale behavior:

- `packages/views/implementation-knowledge/implementation-knowledge-dialogs.tsx`;
- `packages/views/implementation-knowledge/implementation-knowledge-dialogs.test.tsx`;
- `packages/views/implementation-knowledge/implementation-knowledge-history.tsx`;
- `packages/views/implementation-knowledge/implementation-knowledge-history.test.tsx`;
- `packages/views/projects/components/project-detail.tsx`;
- `packages/views/locales/en/projects.json`;
- `packages/views/locales/ja/projects.json`;
- `packages/views/locales/ko/projects.json`;
- `packages/views/locales/zh-Hans/projects.json`.

Every v1-v34 plan, protobuf/generated path, unrelated Auth/Bootstrap/Issue/Task/
Knowledge behavior, app-specific route, original dirty path, legacy backend
tree, and every `server/**` path is read-only. A necessary path outside this
exact list stops r040 and requires a new immutable successor plan.

## Ordered execution

1. R40.1 — Freeze v35 from exact S07C closure `1d515efc`, rename the isolated
   branch to `codex/release3-s07d-r040`, register r040, and commit only the five
   governance activation paths with one continuous nine-field trailer block.
2. R40.2 — Add assertion-first domain, migration/up/down, repository, current
   transaction-authorization, persisted-read validation, revision/history,
   participant, publish/supersede/archive, project-deletion, governance, audit,
   and outbox tests; GREEN only the Canonical Retrospective authority.
3. R40.3 — Add assertion-first target-claim, Task-default, explicit-Issue,
   idempotent replay/conflict/interruption/concurrency, source-link, HTTP,
   composition, Runtime, permission, and feature-flag tests; GREEN only through
   injected Task/Issue creation contracts.
4. R40.4 — Add strict Core schemas/client/query/mutation tests, then the loaded
   shared draft/history/participant/publish/archive/action-target UI and all
   four locales. Keep server access authoritative and retain exact error states.
5. R40.5 — Run focused and complete Workspace/backend checks, official race,
   full Core/Views tests, strict type/lint, forced root gates, production Web
   build, and fresh two-identity installed acceptance against real Canonical
   HTTP plus production Web. Record every unrelated aggregate NON-PASS.
6. R40.6 — Freeze one exact candidate; verify v35/policy hashes, nine trailers
   on every r040 commit, exact path scope, zero `server/**` and generated paths,
   clean isolated worktree, original dirty-tree preservation, process cleanup,
   and fresh independent `SPEC PASS` plus `CODE/SECURITY/QUALITY PASS` before
   closing PCR-S07D.

## Deterministic and installed acceptance

- Domain/repository tests cover normalization/limits, duplicate participants
  and action IDs, creator projection, facilitator self-appointment denial,
  membership removal, lead reassignment, facilitator removal, foreign Project,
  stale revision, invalid transitions, published-revision immutability,
  superseded history, linked-item edit denial, archive, restart, and persisted
  corruption/ownership failure closure.
- Migration tests cover first install, idempotent second run, version-19 upgrade,
  no forbidden DDL, empty down success, and independently blocked retained
  head/revision/participant/claim/link/governance/audit/outbox cases with exact
  before/after preservation.
- Target tests cover omitted-kind Task default, explicit Issue, unknown kind,
  source revision/link, stable replay, same-key different-body conflict,
  concurrent keys, interruption after claim and target, target-service denial,
  retry reauthorization, and no direct Task/Issue table access by the
  Retrospective repository.
- HTTP/composition tests cover trusted identity, CSRF/bearer behavior, status
  and problem mappings, cursor bounds, strict ownership, server access, and
  complete create/save/publish/publish-revision/target/archive flows.
- Core/Views tests cover strict malformed responses, stable retry key,
  Workspace-scoped invalidation, participant roles, server-owned controls,
  action target default/explicit selection, immutable history, archive, true
  empty versus error, four-locale parity, and accessible names/alerts.
- Fresh installed acceptance uses independent owner/lead and ordinary-member
  identities through visible login. It proves member draft, self-facilitator
  denial, owner/lead facilitator assignment, facilitator publication, default
  Task and explicit Issue creation with same-key replay and source links,
  published revision preservation/supersede, restart/reload persistence,
  archive without target deletion, runtime
  `project_retrospectives=true`, meaningful DOM/title, and no framework overlay.
- In-app Browser is attempted first. An installed repository-approved browser
  fallback is allowed only when host-loopback isolation is reproduced and
  disclosed. All owned processes/listeners are closed before exact freeze.
- Only fresh independent dual PASS may close r040/PCR-S07D. Release 3 remains
  incomplete until a later aggregate DoneGate verifies S07A-D closure together.

## Explicit exclusions and stop conditions

Release 3 completion, S08+, S10, Knowledge proposal/review integration,
realtime Retrospective projection, Task/Issue deletion semantics, external
service calls, new participant/grant administration surfaces, permanent
Retrospective deletion, target unlink/re-target, migrations beyond 000020,
explicit indexes, foreign keys/cascades/triggers, public Issue HTTP/proto
changes, generated protobufs, app-specific routes, original dirty paths, push,
merge, deployment, legacy backend writes, and all `server/**` changes are
excluded.

Stop before closure on any published snapshot mutation, facilitator authority
leak, inactive-member authorization, cross-Workspace/Project projection,
duplicate target, non-idempotent retry, missing source link, direct
Retrospective-to-Task/Issue table mutation, partial success reported as linked,
hidden test failure, missing/duplicate trailer, scope drift, original dirty-path
overlap, unclosed process, `server/**` or generated change, or either
independent-review BLOCK. Any repair outside this exact boundary requires a new
immutable plan; v35 is never amended after activation.
