# Product capability roadmap v29 — S07B governed Requirements

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Plan-Version: `29`
- Task-Revision: `r034`
- Work-Item: `PCR-S07B`
- Exact base: `07aef1a577db78598c92c70312a33989e6177d64`
- Status: `approved-active`
- Authority: the Human Customer's confirmed direction on 2026-08-19 to
  continue approved execution until Release 3 is complete, plus the explicit
  confirmation of the prerequisite minimal outline authority

## Predecessor and activation boundary

Exact base `07aef1a577db78598c92c70312a33989e6177d64` records the fresh
independent closure of PCR-S07A under v28/r033. The isolated Release 3
worktree is clean at activation and the original dirty worktree remains
untouched. PCR-S07B is the only active product task.

The Customer confirmed that S07B may install the smallest real outline
authority required for Requirement traceability: persistent project-owned root
nodes, stable node IDs, create/read operations, project ownership validation,
and positive Requirement-to-outline links. That authority does not activate or
pre-deliver the S10 outline product. Nested hierarchy, move/reorder, derived
numbering, node edit/archive/restore, Issue-to-outline relations, progress
rollups, phase views, realtime outline projection, and a standalone outline UI
remain inactive.

## Goal and singular authority

Install the complete PCR-S07B Requirement lifecycle through the Canonical
Workspace SQLite/HTTP/Core/shared-view vertical. Workspace remains the sole
mutable Requirement and minimal-outline authority. The old generated gRPC
Requirement surface is compatibility-only: its mutation must fail explicitly
after activation and may not remain a second write path. No generated protobuf
change is required or authorized.

The existing `workspace_requirements` and `workspace_requirement_versions`
tables are treated as immutable legacy input after migration. Canonical code
uses new baseline/revision/link tables only. Migration may copy a valid,
singular legacy project record into the new authority, but runtime code may not
dual-write or continue mutating the legacy tables. The compatibility read may
project the Canonical current baseline; the compatibility save returns a safe,
typed disabled/failed-precondition result before any write.

## Frozen aggregate and content contract

1. A project has zero or one Requirement baseline. The baseline has one
   monotonic `current_revision` shared by content, lifecycle, and Requirement
   link mutations. Initial draft creation requires `expected_revision=0` and
   an idempotency key. Every committed mutation advances exactly once and
   appends one immutable revision snapshot; stale requests return `409`
   `revision_conflict` with `current_revision` and no domain, audit,
   idempotency, or outbox effect.
2. Structured content retains `problem_statement`, `goals`, `in_scope`,
   `out_of_scope`, `constraints`, `acceptance_criteria`, and `dependencies`.
   Every item has a non-empty stable key and text. Keys are unique across the
   baseline. Only goals, in-scope items, constraints, and acceptance criteria
   are traceable targets. Server-side length/count bounds and canonical JSON
   request hashing are deterministic.
3. Lifecycle states are exactly `draft`, `in_review`, `approved`, `frozen`,
   `changed`, and `retired`. Valid transitions are:

   ```text
   create/save:  none -> draft; draft -> draft; changed -> changed
   submit:       draft|changed -> in_review
   withdraw:     in_review -> its recorded draft|changed origin
   approve:      in_review -> approved
   freeze:       approved -> frozen
   material edit:frozen -> changed
   retire:       draft|in_review|approved|frozen|changed -> retired
   ```

   Any other transition fails without revision advance. Ordinary content edits
   in `in_review`, `approved`, `frozen`, or `retired` fail. Editing a frozen
   baseline requires `material_change=true`, an actual canonical content
   difference, and a non-empty change summary.
4. `effective_revision` is null until the first freeze. It points to the most
   recent frozen snapshot and remains unchanged during a later changed,
   in-review, or approved cycle. A new freeze atomically advances the effective
   revision. Retirement is terminal but preserves current/effective content
   and complete history for authorized reads.
5. Each immutable revision records content, lifecycle state, action, change
   summary, actor, submission/review/freeze metadata, and creation time. The
   response exposes current content, effective content, and ordered history;
   no response derives approval identity from client input.

## Authorization and independent approval

- Every handler starts from trusted Workspace identity. Every write opens one
  fixed SQLite connection with `BEGIN IMMEDIATE`, then re-reads active
  membership, current project state/lead, project grants, baseline/outline
  revision, lifecycle state, and referenced rows on that same connection
  before any mutation.
- `owner` and `admin` may edit Requirement drafts. A member may edit only when
  currently the project lead or holding a current project editor grant. Agents
  and actors outside the Workspace deny.
- Only an owner or a member/admin with an explicit current Requirement approver
  grant may approve or freeze. Approval is independent: the approver differs
  from both the latest content author and the current submitter. There is no
  emergency self-approval path in S07B.
- Owner-managed project access uses a revisioned full-set API under the
  Requirement route family. Grants are stored against the resolved active
  `auth_members.id`, not a client-asserted role. The only grant kinds are
  `project_editor` and `requirement_approver`. Its `expected_revision` is the
  independent access-set revision, never the Requirement or outline revision.
  A removed member or stale grant set cannot authorize a mutation.
- Minimal outline creation and Requirement-to-outline linking allow
  owner/admin and members with `project_editor`; being project lead alone does
  not imply outline authority. All project terminal states and missing action
  mappings fail closed.

## Requirement links and material-change impact

1. An Issue link targets one current traceable Requirement item key and one
   existing Issue in the same Workspace and project. An outline link targets
   one current traceable item key and one active minimal-outline node in the
   same Workspace and project. Validation, current authorization, expected
   Requirement revision, relation mutation, immutable snapshot, audit, and
   outbox insert are owned by one transaction.
2. Link rows retain linked/unlinked revision intervals so later S07C can derive
   current and effective projections without rewriting history. Relinking
   creates a new interval; it does not overwrite an earlier interval. Link and
   unlink are allowed in every non-retired lifecycle state. They preserve
   content and lifecycle; when the baseline is frozen they also advance
   `effective_revision` to the new identical-content snapshot so effective
   traceability never points at an older relation set. During a changed,
   in-review, or approved re-review cycle, the prior effective revision remains
   unchanged until the next freeze.
3. A frozen material change marks every currently linked Issue in a separate
   Requirement-owned `review_required` projection at the new revision. It does
   not update Issue title, description, status, acceptance, metadata, revision,
   or any other Issue-owned content. The response and shared UI expose the
   projection explicitly.
4. User/agent Issue deletion removes active Requirement links and projections
   in the same deletion transaction and appends one system-attributed baseline
   revision per affected baseline. Both existing Project deletion paths remove
   all canonical Requirement, link, grant, outline, audit/idempotency namespace,
   and retained legacy rows for that project atomically before deleting the
   project.
5. Creating an Issue from a Requirement item is not part of S07B. The existing
   uninstalled Core declaration is hidden/removed from the installed UI and its
   backend route stays unavailable; explicit action-item Issue creation remains
   governed by S07D.

## Confirmed minimal outline authority

- `GET /api/projects/{id}/outline` returns the outline-set revision and all
  project-owned root nodes ordered by `created_at,id`.
- `POST /api/projects/{id}/outline` requires an idempotency key,
  `expected_revision`, and a bounded non-empty title. It creates exactly one
  immutable root node with a server-generated stable ID and advances the
  outline-set revision once. Replay returns the original status/body; key/body
  conflict returns idempotency conflict.
- Nodes have no parent, sibling position, display number, weight, Issue links,
  move/reorder command, patch/delete command, or archive/restore action in
  S07B. The S10-frozen PATCH/DELETE/reorder and Issue-link routes return an
  explicit feature-unavailable response until S10 activates them.
- Requirement-to-outline linking uses the Requirement route family and advances
  the Requirement revision, not the outline-set revision. The outline set is
  revalidated transactionally so a foreign-project or fabricated node never
  links.
- `project_outline` remains false and no standalone outline surface is rendered.
  The Requirement component may show only the minimal root-node create/read
  picker required to establish traceability.

## HTTP, Core, and shared-view contract

The frozen public family `/api/projects/{id}/requirement-baseline/**` is
retained and receives exact field-level routes:

- `GET/PUT /api/projects/{id}/requirement-baseline`;
- `POST .../submit-review`, `.../withdraw`, `.../approve`, `.../freeze`, and
  `.../retire`;
- `POST .../links` and
  `DELETE .../links/{requirement_key}/{issue_id}` for Issue links;
- `POST .../outline-links` and
  `DELETE .../outline-links/{requirement_key}/{node_id}`;
- `GET/PUT .../access` for the owner-managed revisioned full grant set.

All mutation bodies use `expected_revision` exactly; the old ambiguous
`revision` alias is not accepted. Create returns `201`, reads and response-body
mutations return `200`, and bodyless unlink succeeds with `204`. Exact typed
problems distinguish malformed input, permission denial, missing/foreign
project or target, revision conflict, invalid transition, self-approval,
idempotency conflict, and disabled legacy/S10 operations without echoing raw
payloads.

Core converts snake_case only at the API boundary, uses Zod for every consumed
response, throws on malformed mutation or baseline data, and records only
endpoint plus safe issue metadata in schema diagnostics. Mutation hooks retain
one idempotency key across retry of the same initial-create or root-node-create
intent. The shared Requirement view exposes the complete state machine,
current/effective distinction, immutable history, Issue and outline links,
review-required impact, stale errors, and the minimal root-node picker. It does
not infer approval authority; server-projected access controls are authoritative.

`project_requirements` remains false until the backend, strict Core boundary,
shared view, and installed acceptance are complete, then becomes true.
`project_outline`, S07C coverage acceptance, and every later feature flag remain
false. Existing `/coverage` may return a truthful S07B traceability-only
projection but may not claim linked/implemented/accepted S07C semantics.

## Migration, transaction, audit, and rollback contract

- Add only `000019_project_requirements.up.sql` and matching down migration.
  No `FOREIGN KEY`, `REFERENCES`, cascade, explicit `CREATE INDEX`, or generated
  schema output is allowed. Needed uniqueness uses table-level primary/unique
  constraints compatible with the transaction-wrapped SQLite runner.
- Canonical tables cover baseline, immutable revisions, Issue-link intervals,
  outline-link intervals, review-required projections, revisioned project
  grants, outline sets, and outline nodes. Generic governance tables retain
  idempotency, audit, and outbox authority.
- Up migration preserves legacy tables unchanged, rejects more than one legacy
  Requirement per project by transaction rollback, validates each legacy Issue
  reference against the same project, and imports valid legacy content into a
  deterministic draft snapshot with one synthetic stable traceability key.
  The imported baseline retains its legacy source ID; runtime never writes the
  old tables.
- Down migration succeeds only when every canonical row is still an exact
  untouched import and there are no new grants, nodes, projections, link
  intervals, Requirement audit/idempotency/outbox rows, or post-import
  mutations. Otherwise it aborts before the first destructive statement and
  preserves all data byte-for-byte. Tests cover every independent guard and
  exact retained-row snapshots after expected failure.
- Every transition/link/grant/node command writes its domain state, immutable
  audit, deterministic replay record where required, and one durable outbox
  event in the same transaction. Publication occurs only after commit. Failure
  injection after each write phase proves zero partial state.

## Explicit exclusions

PCR-S07C coverage semantics, PCR-S07D Retrospectives, Release 3 completion,
S10 hierarchy/move/reorder/numbering/archive/restore/Issue-outline/progress,
project phases, phase board, full outline UI, outline realtime, automatic Issue
planning, generated protobufs, unrelated Issue/Input UI, Desktop-only behavior,
external service calls, push, merge, deployment, and every `server/**` change
are inactive or excluded. The original dirty worktree remains untouched.

## Writable scope

- `backend/internal/modules/workspace/contract/**` only for exact Requirement,
  minimal-outline, grant, error, and installed-capability contracts/tests;
- `backend/internal/modules/workspace/internal/{domain,application,infrastructure,interfaces}/**`
  only for S07B behavior, `000019`, repositories, cleanup, HTTP, and tests;
- exact Requirement/minimal-outline composition, legacy compatibility,
  persistence, Project/Issue deletion, and tests under
  `backend/internal/modules/workspace/**`;
- exact capability/feature composition and tests under
  `backend/internal/bootstrap/**`;
- `packages/core/{api,project-requirements,types}/**` and exact tests;
- `packages/views/projects/components/project-requirement-baseline*`, the exact
  `project-detail` integration/tests, and Requirement keys in the four existing
  `packages/views/locales/{en,ja,ko,zh-Hans}/projects.json` files;
- `e2e/project-requirements.spec.ts` and only an existing shared e2e helper if a
  captured RED proves the new exact journey cannot use it unchanged;
- `backend/docs/plans/product-capability-roadmap/{plan.md,plan_v29.md,story-map.md,task-register.md,journal.md}`.

No other package, migration, generated artifact, original dirty path, legacy
backend tree, or `server/**` path is writable. A necessary path outside this
list stops r034 and requires an immutable successor plan.

## Ordered execution

1. R34.1 — Freeze v29 from exact base `07aef1a5`, record the confirmed outline
   boundary, move the isolated branch to `codex/release3-s07b-r034`, and commit
   only the five governance activation paths with one continuous nine-field
   trailer block.
2. R34.2 — RED the domain/state, migration/backfill/down guards, transaction
   authorization, grant, stale/idempotency, link/impact, deletion, restart, and
   legacy-write-disable contracts. Execute each focused test and preserve the
   expected behavioral failure before production implementation.
3. R34.3 — GREEN the smallest singular Canonical SQLite/application/HTTP
   vertical. Keep both user-visible feature flags false until all backend,
   migration, security, and cleanup tests pass.
4. R34.4 — RED/GREEN strict Core schemas/client/hooks and the shared Requirement
   UI, including all lifecycle states, independent approval projection,
   material-change impact, history, Issue links, and the minimal outline picker.
5. R34.5 — Run focused and full deterministic backend/frontend gates, exact
   changed-package race, production Web build, and retries-disabled installed
   Chrome against fresh SQLite with two independent identities. Enable only
   `project_requirements` after the exact journey passes.
6. R34.6 — Freeze one exact candidate; verify plan/policy hashes, all nine
   trailers on every r034 commit, exact path scope, empty `server/**`, clean
   isolated tree, original dirty-tree preservation, process cleanup, and obtain
   fresh independent `SPEC PASS` plus `CODE/SECURITY/QUALITY PASS`.

## Acceptance and deterministic verification

- Table-driven state-machine tests cover every valid and invalid transition,
  draft/changed withdrawal origin, terminal retirement, effective revision,
  frozen plain-edit denial, material-difference requirement, and independent
  approval denial/allow.
- Transaction tests cover live membership/project lead/grant changes, terminal
  project state, revision precedence, cross-Workspace/project Issue/node
  denial, one concurrent winner, replay/body conflict, failure-phase rollback,
  exact audit/outbox cardinality, and secret-free errors.
- Fresh SQLite tests cover legacy import/duplicate/foreign-reference rollback,
  restart persistence, both Project delete paths, user/agent Issue cleanup, and
  exact down-migration guards/data preservation.
- Focused Requirement/minimal-outline backend packages and the complete
  Workspace package pass, followed by backend `make check` and the official
  changed-package `make test-race`. Task-specific F-drive Go temp/cache may be
  used if C-drive pressure recurs; the first failure remains recorded.
- Focused Core and Views tests, root/Core typechecks, root test, and production
  Web build pass. A broad failure is never renamed PASS; an unrelated failure
  requires exact focused evidence and honest classification.
- Installed Chrome, retries disabled, uses the real Canonical HTTP backend and
  fresh SQLite. A project lead creates/saves/submits, links a same-project Issue,
  an authorized editor creates and links a stable root node, an independent
  owner approves/freezes, ordinary frozen edit is denied, material change keeps
  the old effective revision and marks the linked Issue review-required without
  changing Issue content, re-review/refreeze succeeds, history survives reload,
  and retirement is terminal. Cross-project link and stale revision fail.
- Only a fresh independent review returning both required PASS decisions may
  close r034/PCR-S07B. PCR-S07C remains inactive until its own successor plan.

## Rollback and stop conditions

Rollback first disables `project_requirements`, removes only S07B composition
and UI exposure, and executes the guarded down migration only when its exact
safe predicate holds. Stop before closure on dual-write legacy behavior,
mutable revision history, self-approval, permission or revision validation
outside the owning write transaction, Issue content mutation from material
change, foreign links, partial deletion, unsafe down migration, malformed
response fallback-as-success, `project_outline=true`, S10 behavior, hidden
aggregate failure, missing/duplicate trailer, scope drift, original dirty-path
overlap, `server/**` change, or independent review BLOCK. Any material repair
requires an immutable successor plan; v29 is never amended after activation.
