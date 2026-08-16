# Product Capability Roadmap Plan v1

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Version: `v1`
- Status: `approved-for-execution`
- Base commit: `45213820fade7f61294d2287e063bf19fbd015ee`
- Branch baseline: `codex/multica-six-domain-baseline`
- Active step at approval: `PCR-S00 — contract freeze and clean task creation`
- Approval basis: user explicitly approved implementation on `2026-08-16`
- Product implementation authority: one active step at a time
- Created: `2026-08-16`

## 1. Goal

Deliver a single, workspace-isolated product surface for:

1. skill CRUD, import, versioning, and file management;
2. task CRUD;
3. full Knowledge query and candidate review;
4. project Resources, Requirements, and Retrospectives;
5. pin reorder and Issue/project search;
6. Issue similarity warnings that reduce duplicate work;
7. idempotent daily overdue-Issue reminders;
8. project phases, book-like outlines, Issue linkage, and progress boards.

The plan ports or completes required behavior in the Canonical backend instead
of extending the legacy server. It treats existing Web UI, API client methods,
SQLite use cases, and Control Plane flows as evidence or reusable contracts,
not as proof that a feature is installed and usable.

## 2. Confirmed product decisions

The following decisions are frozen for v1.

### 2.1 Domain authority

- Workspace is the product data authority for tasks, Knowledge, Requirements,
  project content, outlines, and Issue reminders.
- The append-only Control Plane may consume identifiers or events, but there is
  no dual write, shared-table access, or silent model unification.
- A task is a lightweight personal or team to-do. An Issue is a project
  delivery unit. They may be linked; promoting a task to an Issue is an
  explicit command and never creates ongoing bidirectional synchronization.

### 2.2 Project phase and outline

- `phase` is independent from the existing project lifecycle `status`.
- Initial phases are `initiation`, `planning`, `review`, and `implementation`,
  displayed as 立项、规划、评审、实施.
- v1 exposes the fixed ordered set but stores phase as an extensible contract so
  a later approved version may support workspace-defined phases.
- Phase transitions may move forward or backward. Protected transitions require
  permission, deterministic checks, and a reason when moving backward.
- An outline is a project-owned ordered tree. Database identity is stable and
  independent from display numbers such as `1`, `1.1`, and `1.1.1`.
- An Issue has at most one primary outline node and may have additional
  reference links.
- Progress defaults to completed-Issue count divided by linked-Issue count.
  Outline nodes retain an optional weight so later views may use effort-based
  progress without replacing identifiers.
- Automatic Issue planning from an outline is not implemented in this plan.
  A later proposal must provide preview, human editing, approval, duplicate
  checking, idempotent batch creation, and batch rollback.

### 2.3 Similarity detection

- Detection runs when an Issue is created or when its title or description is
  materially changed. Administrators may request a workspace rescan.
- The search corpus is workspace-scoped. Same-project matches receive a ranking
  boost; cross-workspace retrieval is forbidden.
- v1 uses a hybrid of normalized exact matching, identifier matching, lexical
  full-text search, and replaceable semantic scoring. An external model is not
  a mandatory dependency for the first release.
- Similarity produces warnings and ranked candidates. It does not block Issue
  creation. A user who proceeds records a short override reason.
- v1 may mark an Issue as a duplicate of another Issue after human confirmation;
  it never automatically merges Issues.

### 2.4 Overdue reminders

- The first delivery channel is an in-product notification center. Email is a
  later adapter and is not required for v1 acceptance.
- Default recipients are the current assignee and subscribers. If an Issue has
  neither, the project lead is the fallback recipient when one exists.
- Calendar evaluation uses the workspace timezone. Presentation may convert to
  a member timezone, but that does not change the daily deduplication key.
- Each `(workspace, issue, recipient, local_date, reminder_policy_version)` is
  emitted at most once.
- Done, cancelled, deleted, undated, or not-yet-overdue Issues are excluded.
- A paused project suppresses reminders by default. Workspace policy may opt in.
- Members may configure frequency, quiet hours, and opt-out within limits. The
  default is once per day.
- Durable schedule state and an outbox are required. A process-local timer is
  insufficient evidence of delivery.

### 2.5 Governance and content lifecycle

- Member, owner, admin, and agent permissions are explicit. Agents have no
  management, review, publication, or phase-approval permissions by default.
- Published or referenced Skills, Knowledge, Requirements, and outline nodes
  are archived instead of being hard-deleted. Unreferenced drafts may be
  permanently deleted when explicitly authorized.
- Review, publication, phase transition, import, deletion, and similarity
  override actions produce immutable audit records.
- A creator cannot approve their own Knowledge candidate. An emergency admin
  override requires a reason and remains auditable.
- Skill import conflicts create a new version by default. Replacement requires
  an explicit choice. Workspace Skills are the v1 priority; personal and system
  visibility remain modeled but are not required UI surfaces.
- Knowledge publication requires source evidence or an explicit reviewed human
  provenance declaration. Published content changes through a new revision.
- Requirement lifecycle is draft, in review, approved, frozen, changed, and
  retired. A material change marks linked Issues as requiring review; it never
  silently rewrites them.
- A retrospective is versioned project content. Its action items may create a
  task or an Issue; task is the default.

## 3. Invariants

1. Every read and write is scoped by `workspace_id` and active membership.
2. Project, Issue, task, resource, Requirement, retrospective, outline, reminder,
   Knowledge, and Skill references are authorized server-side.
3. `server/**` is read-only migration evidence. No step may change it.
4. Backend product implementation lives only under `backend/**`.
5. Modules communicate through public contracts, never another module's tables.
6. No new database foreign key or cascading action is introduced. Validation
   and dependent cleanup occur in application transactions.
7. PostgreSQL indexes use their own single-statement concurrent migrations.
   SQLite-local migrations remain additive and reversible within the Canonical
   migration mechanism.
8. Mutation idempotency, revision conflicts, and post-commit event ordering are
   part of the API contract, not UI-only behavior.
9. API responses consumed by TypeScript pass through Zod schemas with malformed
   response tests.
10. React Query owns server state. Realtime events update or invalidate its
    workspace-scoped caches; Zustand does not mirror server entities.
11. A compatibility response never fabricates success for an unavailable
    capability. Disabled capabilities fail closed and remain visible in
    `/api/config` or an equivalent capability projection.
12. Static code, a generated service, or a UI affordance is not release proof.
    Acceptance requires the installed Canonical runtime and a clean candidate.
13. Audit and outbox writes that describe a successful domain mutation commit
    atomically with that mutation; publication occurs after commit.
14. Credentials, imported Skill secrets, and external-resource tokens are never
    persisted in content files, audit payloads, or logs.

## 4. Exact program scope

Implementation steps may declare a narrow subset of these roots:

- `backend/api/**`
- `backend/internal/modules/{workspace,auth,space,system}/**`
- `backend/internal/bootstrap/**`
- `backend/internal/realtime/**`
- `backend/cmd/server/**`
- `backend/docs/plans/product-capability-roadmap/**`
- `packages/core/**`
- `packages/views/**`
- `packages/ui/**` only for reusable business-neutral primitives
- `apps/web/**` for Next.js platform wiring only
- `apps/desktop/**` for Electron platform wiring only
- `e2e/**`

Every implementation task must narrow this list further and identify its exact
files, contracts, migration files, Plan-Step, task revision, and verification.
No step receives blanket authority over all listed roots.

## 5. Non-goals

- Any modification below `server/**` or migration by merging a legacy tree.
- Mobile delivery or mobile release parity.
- Production deployment, merge, release tagging, or customer acceptance.
- External email, Slack, or webhook notification delivery in v1.
- Automatic or model-autonomous Issue creation from a project outline.
- Automatic Issue merge or deletion based on a similarity score.
- Workspace-configurable project phases in the first release.
- A general document-management system inside Project Resources.
- Sharing database tables or adding dual writes between Workspace and Control
  Plane.
- Replacing the active Canonical cutover plan or claiming its blocked C9 gate.
- Production PostgreSQL acceptance unless a later approved plan explicitly adds
  that environment and its evidence.

## 6. Dependencies and ordering

### 6.1 Repository and plan dependencies

- The Canonical runtime cutover plan is currently blocked. Documentation may
  advance, but a product implementation step must rebase onto a stable approved
  baseline and may not overlap a repair step's paths or runtime port.
- `server-function-migration` and `p0-p2-backend-platform` remain separate plans.
  Their accepted semantics may be ported only through a new contract in this
  plan; their plan status does not satisfy this plan's acceptance.
- Existing dirty files and untracked directories present at planning time are
  outside this plan and must be preserved.

### 6.2 Capability dependency graph

```text
PCR-S00 contract freeze
  -> PCR-S01 Canonical authority, audit, revision, capability flags
      -> PCR-S02 task CRUD
      -> PCR-S03 Issue/project search
          -> PCR-S08 Issue similarity
      -> PCR-S04 pin reorder
      -> PCR-S05 Skill lifecycle and files
      -> PCR-S06 Knowledge query and review
      -> PCR-S07 project Resources, Requirements, Retrospectives
      -> PCR-S09 notification center and overdue scheduler
      -> PCR-S10 project phase, outline, links, progress board
  -> PCR-S11 cross-surface acceptance and operational handoff
```

`PCR-S11` depends on every selected release slice. A smaller release may stop at
an earlier step only if a new approved plan version updates the milestone and
acceptance scope.

## 7. Ordered implementation steps

Only one step may be active at a time.

### PCR-S00 — Contract freeze and clean task creation

Freeze API shapes, state machines, permissions, data ownership, migration
sequence, feature flags, performance budgets, and a clean implementation base.
Resolve overlap with active plans before product code changes.

Acceptance:

- the capability matrix has no unclassified implementation surface;
- Workspace/Control Plane ownership and no-dual-write rules appear in contracts;
- exact allowed paths and task metadata are frozen for PCR-S01;
- dirty user files are indexed and excluded;
- no product file changes occur in this step.

### PCR-S01 — Canonical foundation

Install reusable Workspace authorization actions, revision/conflict responses,
audit records, outbox delivery, capability flags, and health/diagnostic
projections. Define stable pagination, sorting, error, idempotency, and realtime
contracts. Do not enable an incomplete feature in the default runtime.

Acceptance:

- role and agent authorization matrices are table-tested;
- missing providers fail closed;
- audit/outbox writes roll back with failed domain writes;
- capability flags report installed behavior accurately;
- HTTP and gRPC compatibility behavior is deterministic.

### PCR-S02 — Task CRUD vertical slice

Complete and install task create, get, list, update, status transition, delete or
archive, ordering, assignee, dates, Issue linkage, and explicit promotion to an
Issue. Reuse the existing Todo domain only after naming and compatibility are
frozen; do not expose both Todo and Task as competing product entities.

Acceptance:

- all operations are workspace-scoped and permission-gated;
- promotion is idempotent and retains the source task link;
- delete cleans dependent projections transactionally;
- Web and Desktop share queries, mutations, and views;
- restart and concurrent-update tests pass in the installed runtime.

### PCR-S03 — Issue and project search

Expose stable Issue and project search APIs backed by a repository-level search
contract rather than list-then-filter handlers. Cover normalized titles,
descriptions, identifiers, closed-state inclusion, pagination, permissions, and
deterministic ranking. Preserve a replaceable semantic-ranking port.

Acceptance:

- no result crosses workspace or membership boundaries;
- Chinese and English fixtures cover normalization and ranking;
- index writes and entity writes cannot produce a permanently inconsistent
  visible state;
- malformed response and cancellation tests cover clients;
- p95 targets and maximum result limits are recorded before implementation.

### PCR-S04 — Pin reorder

Add an atomic reorder command with complete membership validation, unique-item
validation, and optimistic concurrency. The server assigns canonical contiguous
positions; clients do not persist fractional positions as authority.

Acceptance:

- duplicate, missing, foreign-workspace, and stale-revision requests fail;
- concurrent reorder has one winner and returns the current revision to losers;
- list order remains stable after restart;
- optimistic UI rolls back on conflict.

### PCR-S05 — Skill CRUD, import, and files

Create a Canonical System Skill catalog contract with workspace visibility,
immutable versions, provenance, checksums, and archive semantics. Space owns
binary assets; System owns Skill metadata and logical file manifests. Import
rejects path traversal, symbolic-link escapes, forbidden file types, excessive
counts, and size-limit violations. A conflict creates a new version by default.

Acceptance:

- create, read, list, update-as-version, archive, restore, import, file add,
  replace, remove, and download are authorized and audited;
- imported paths are normalized and cannot escape the Skill root;
- a failed import leaves no asset, manifest, binding, or temp-file leak;
- referenced versions remain resolvable after a newer version is published;
- agents cannot administer Skills without an explicit grant.

### PCR-S06 — Knowledge query and candidate review

Extend Knowledge from create/get into query, filtering, source and evidence
projection, revisions, candidate review, approve, reject, quarantine, publish,
supersede, invalidate, and restore. Deduplication is advisory before human
review. The submitter cannot approve the same candidate.

Acceptance:

- every transition has an authorization and state-machine test;
- review uses expected revision and rejects stale decisions;
- published Knowledge has source evidence or reviewed provenance;
- query results expose status, revision, source, and applicability;
- publication and invalidation emit post-commit realtime events;
- Control Plane data is neither read directly nor dual-written.

### PCR-S07 — Project Resources, Requirements, and Retrospectives

Deliver project-owned resource links, the Requirement lifecycle, traceability to
Issues and outline nodes, coverage projections, versioned retrospectives, and
action-item creation. Initial resource types are GitHub repository and generic
URL. Removing a relationship never deletes an external resource.

Acceptance:

- resource validation records connection state without persisting credentials;
- Requirement transitions and material-change rules are deterministic;
- a material Requirement change marks linked Issues `review_required` without
  modifying their content;
- coverage distinguishes linked, completed, and accepted Issues;
- retrospective publication is versioned and action-item creation is idempotent;
- action items may create a task or Issue, with task as the default.

### PCR-S08 — Issue similarity warnings

Use the PCR-S03 search contract to rank possible duplicates on create and
material update. Record algorithm version, component scores, shown candidates,
user decision, and override reason without storing sensitive model prompts.

Acceptance:

- workspace isolation is proven with negative fixtures;
- exact duplicates, near-title matches, unrelated items, closed items, and
  cross-project matches are represented in a versioned evaluation set;
- thresholds meet frozen precision, recall, and latency budgets;
- service failure does not fabricate a clean result and follows the approved
  fail-open warning policy with visible diagnostic state;
- duplicate marking is human-confirmed and reversible;
- no automatic merge or deletion exists.

### PCR-S09 — Notification center and daily overdue reminders

Add durable notifications, unread/read/archive state, member preferences,
workspace-timezone evaluation, reminder policies, schedule leases, outbox
delivery, retry limits, dead-letter diagnostics, and an operator replay command.

Acceptance:

- each daily deduplication key produces at most one notification;
- parallel schedulers cannot double-send;
- restarts, clock boundaries, daylight-saving changes, assignee changes, quiet
  hours, paused projects, opt-out, and failed delivery are tested;
- unauthorized users cannot read or mutate another member's notifications;
- health and backlog age are exposed without leaking message content;
- replay is bounded, authorized, audited, and idempotent.

### PCR-S10 — Project phase, outline, Issue links, and progress board

Add project phase and transition history; ordered outline nodes with stable IDs,
parent relations, display ordering, and derived numbering; primary and reference
Issue links; roll-up progress; a project-list phase board; and a project outline
progress view.

Acceptance:

- existing projects receive a documented default phase without changing status;
- illegal transitions, unauthorized approval, stale reorder, cycles, excessive
  depth, and cross-project links fail deterministically;
- moving or deleting a node updates numbering and links transactionally;
- a referenced node archives instead of hard-deleting;
- progress is derived from current linked Issues and is not dual-owned;
- realtime updates refresh project list, outline, and progress views;
- no automatic Issue creation is present.

### PCR-S11 — Integrated acceptance and operational handoff

Validate the installed Canonical runtime, Web and Desktop shared surfaces,
feature flags, migrations, restart behavior, concurrency, realtime, security,
performance, and rollback in a clean candidate. Produce operator runbooks and an
independent-review evidence index.

Acceptance:

- deterministic repository checks pass on the frozen candidate;
- browser journeys cover the new user-visible capabilities without legacy
  runtime traffic;
- migrations pass upgrade, restart, rollback-safe, and partial-failure tests;
- scheduler, outbox, search, and import diagnostics have operator procedures;
- an independent reviewer verifies scope and evidence;
- Customer Acceptance remains a separate explicit decision.

## 8. API and state-machine contracts to freeze in PCR-S00

### 8.1 Shared API rules

- Workspace identity comes from the trusted server resolver, never a request
  body owner field.
- Lists use one pagination style per API generation, stable tie-break sorting,
  bounded limits, and structured filters.
- Mutations that may be retried accept an idempotency key.
- Revisioned writes return `409 Conflict` with current revision metadata.
- Deletes distinguish archive, restore, and permanent deletion.
- Search and similarity responses identify ranking version and truncation.

### 8.2 State machines

```text
Knowledge candidate:
candidate -> in_review -> published -> superseded
                    \-> rejected
                    \-> quarantined -> in_review

Requirement:
draft -> in_review -> approved -> frozen -> changed -> in_review
   \-----------------------------------------------> retired

Retrospective:
draft -> published -> superseded

Skill version:
draft -> published -> deprecated -> archived

Notification:
pending -> delivered -> read -> archived
       \-> retry_wait -> dead_letter -> replayed
```

Invalid transitions fail without changing revisions or publishing events.

## 9. Data and migration plan

PCR-S00 must freeze additive schemas for:

- authorization actions, audit entries, mutation idempotency, and outbox;
- task-Issue promotion links;
- search documents or index metadata;
- Skill catalog, versions, manifests, files, visibility, and provenance;
- Knowledge revisions, sources, reviews, and publication history;
- project resources and connection state;
- Requirement revisions, reviews, Issue links, and review-required projections;
- retrospectives, revisions, participants, and action-item links;
- similarity evaluations, decisions, and duplicate relations;
- notifications, preferences, reminder policies, schedules, attempts, and
  dead-letter state;
- project phase history, outline nodes, ordering revisions, and Issue links.

No migration may infer ownership from an untrusted client field. Backfills are
restartable and record a schema/data version. Existing projects receive the
`planning` phase only if PCR-S00 confirms that this mapping matches current
`planned` and `in_progress` semantics; otherwise the backfill uses `initiation`
and the decision is versioned before execution. Existing Issue hierarchy is not
rewritten into outline nodes automatically.

## 10. Security and privacy

- Skill archives are treated as untrusted input. Validate decompressed size,
  entry count, canonical paths, link types, file extensions, and MIME evidence.
- Generic resources store URLs and non-secret metadata only. OAuth tokens and
  repository credentials belong to a dedicated secret provider outside content
  rows and audit payloads.
- Search and similarity providers receive only workspace-authorized fields. An
  external semantic provider requires a later explicit data-boundary decision,
  redaction contract, retention contract, and feature flag.
- Notifications expose only content the recipient can still access. Rendering
  reauthorizes target access; stale targets degrade to a neutral tombstone.
- Audit payloads use allowlisted fields and must not contain attachment bodies,
  access tokens, raw Skill archives, or model prompts.

## 11. Performance and operability budgets

PCR-S00 must assign numeric budgets based on representative data before code is
written. At minimum, freeze:

- maximum page size and search candidate count;
- search p50/p95 latency at small and target workspace sizes;
- similarity evaluation latency and maximum synchronous budget;
- Skill archive compressed/decompressed size, file count, and nesting limits;
- maximum outline depth, node count, and reorder batch size;
- reminder scan batch size, lease duration, retry count, backlog-age alert, and
  daily catch-up window;
- notification retention and audit retention;
- realtime queue behavior under slow consumers.

Operational surfaces must include capability state, migration version, search
index lag, outbox backlog, reminder last-success time, dead-letter count, and
bounded repair/replay commands. Health output contains counts and timestamps,
not user content.

## 12. Deterministic verification

Each implementation step runs narrow tests first and freezes its exact commands
in the task. Candidate verification must include, as applicable:

```text
cd backend && make check
cd backend && make test-race
pnpm typecheck
pnpm test
pnpm exec playwright test <frozen capability specs>
```

Additional required suites:

- domain state-transition tables and permission matrices;
- repository workspace-isolation and concurrent-write tests;
- HTTP/gRPC success, invalid, denied, not-found, conflict, and replay tests;
- migration upgrade, restart, backfill idempotency, and failure cleanup;
- malformed TypeScript API response tests;
- clean-candidate Web and Desktop journeys;
- search relevance and similarity evaluation fixtures;
- scheduler lease, calendar boundary, deduplication, retry, and replay tests;
- Skill archive adversarial fixtures;
- realtime post-commit ordering and cache-update tests.

The Windows Go race-loader failure `0xc0000139`, if reproduced, is recorded as
an environment limitation and never reported as a passing race check.

## 13. Acceptance evidence

Each step appends to [journal.md](journal.md):

- frozen task and plan identifiers;
- base commit and candidate commit;
- exact changed paths and migration versions;
- deterministic command, exit code, duration, and sanitized output location;
- API examples and runtime identity;
- negative workspace and authorization evidence;
- browser journey and network-origin evidence where user-visible;
- independent reviewer and unresolved findings;
- rollback rehearsal or reason it is not applicable.

Implementation complete, technical acceptance, independent review, and Customer
Acceptance remain distinct states. No earlier state implies a later one.

## 14. Risks and mitigations

| Risk | Mitigation |
| --- | --- |
| Workspace and Control Plane models diverge | One authority, explicit adapter contracts, no dual write |
| Existing UI implies unavailable backend success | Capability flags and fail-closed installed-runtime tests |
| Similarity false positives frustrate users | Warning-only policy, evaluation set, override feedback, versioned thresholds |
| External semantic provider leaks content | No mandatory provider; explicit later privacy approval |
| Reminder duplication or missed days | Durable lease, daily idempotency key, catch-up window, dead-letter operations |
| Skill import writes unsafe files | Canonical paths, limits, quarantine, atomic manifest/assets, adversarial tests |
| Requirement changes silently invalidate work | Review-required projection and explicit impact workflow |
| Outline reorder corrupts numbering | Stable IDs, revisioned atomic reorder, derived display numbering |
| Progress becomes conflicting stored state | Derive from linked Issues; cache only as a rebuildable projection |
| Active plan overlap changes runtime beneath work | Revalidate plan pointers and base before every step; stop on drift |
| Broad program becomes an unreviewable release | One vertical step at a time, independent acceptance per slice |

## 15. Rollback

- Feature flags disable each new surface independently without fabricating
  success.
- Code rollback proceeds in reverse step order, but additive schemas remain
  readable until a separately approved data-removal plan proves no retained
  data is needed.
- Search projections are rebuildable from authoritative entities.
- Outbox and notification rollback stops new scheduling before disabling
  delivery; pending records remain inspectable and idempotent.
- Skill import rollback quarantines unreferenced objects and reconciles manifests
  before physical cleanup.
- Project phase rollback hides the new view but preserves phase history. It does
  not map phases back into lifecycle status.
- Outline rollback preserves nodes and links as inaccessible retained data until
  export or a separately approved removal step.
- No rollback action modifies `server/**`, rewrites an approved plan snapshot,
  or deletes user data without a frozen destructive-operation task.

## 16. Approval gate

This v1 document was approved for execution on 2026-08-16. Before each product
step starts, its frozen task must name:

- Plan-ID `PRODUCT-CAPABILITY-ROADMAP-001`;
- Plan-Version `v1` or an approved successor;
- the first active step, expected to be `PCR-S00`;
- a rebased implementation commit;
- exact task scope, acceptance, verification, and independent reviewer.

Any material change to product decisions, authority, state machines, delivery
channel, automatic Issue generation, external semantic providers, or rollout
scope requires `plan_v2.md` rather than editing this snapshot after execution
begins.
