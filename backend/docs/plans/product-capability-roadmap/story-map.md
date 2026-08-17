# Product Capability Roadmap Story Map

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Plan-Version: `v10`
- Status: `Release 1 active; PCR-S02A verification repair is the sole active story`

## Delivery policy

- Execute one story at a time in dependency order.
- Each story starts with an observable failing acceptance test or contract test.
- Each story must deliver a usable vertical behavior through the installed
  Canonical runtime; internal scaffolding alone is not Done.
- A story may be split only through a new frozen task whose acceptance remains
  independently user-visible or operationally verifiable.
- Deterministic checks precede model review and independent acceptance.

## Release 0 — Authority and safety foundation

### PCR-S00 — Freeze the program contract

As a product owner, I need one evidence-backed authority and state model so that
implementation does not create competing Task, Knowledge, or Requirement data.

Acceptance scenarios:

1. Given current Workspace, Control Plane, Web, and legacy evidence, when the
   contract inventory is reviewed, then every capability has exactly one write
   authority and all adapters are directional.
2. Given active or blocked plan work, when the first implementation task is
   frozen, then path overlap and base drift are either resolved or block work.
3. Given current dirty files, when the task scope is frozen, then those paths are
   explicitly excluded and remain byte-for-byte preserved.

### PCR-S01A — Authorize actions without broad roles

As a workspace owner, I need capability-specific permissions so that a member
or agent receives only the operations explicitly granted.

Acceptance scenarios:

1. A member with Task read permission cannot publish Knowledge or import a Skill.
2. An agent without management grants cannot administer Skills, approve
   Knowledge, or advance a protected project phase.
3. A missing authorization provider denies every new action.

### PCR-S01B — Revision, audit, idempotency, and outbox

As an operator, I need retried and concurrent mutations to be safe and
explainable.

Acceptance scenarios:

1. Two writes with one expected revision produce one winner and one conflict.
2. Replaying an idempotent command returns the original result without a second
   domain mutation.
3. A rolled-back domain write produces no committed audit or outbox event.
4. A committed mutation publishes only after commit and remains replayable.

## Release 1 — Daily work completion

### PCR-S02A — Manage tasks

Status: `active verification repair under PRODUCT-CAPABILITY-ROADMAP-001 v10 / r015`

As a member, I can create, view, filter, reorder, edit, complete, cancel, archive,
and restore tasks in my workspace.

Acceptance scenarios cover member/agent assignees, due dates, stable ordering,
empty and denied states, concurrent edit conflicts, and restart persistence.

### PCR-S02B — Promote a task to an Issue

Status: `inactive; depends on accepted PCR-S02A`

As a member, I can explicitly promote a task into an Issue without creating
duplicates on retry.

Acceptance scenarios:

1. Promotion creates one Issue and retains an immutable source link.
2. Retrying the same command returns the same Issue.
3. Later Task or Issue edits do not silently synchronize content.

### PCR-S03A — Search Issues

As a member, I can find an Issue by title, description, human identifier, or
number with stable ranking and closed-state filters.

Acceptance scenarios include Chinese and English queries, pagination,
cancellation, empty query rules, closed inclusion, and workspace isolation.

### PCR-S03B — Search projects

As a member, I can find a project using the same stable search contract through
the installed HTTP and shared client surfaces.

Acceptance scenarios include closed-state filters, deterministic tie-breaks,
membership denial, malformed responses, and restart consistency.

### PCR-S04 — Reorder pins

As a member, I can reorder my project and Issue pins, and concurrent changes do
not silently overwrite one another.

Acceptance scenarios include complete ordered payload, duplicate item, missing
item, foreign workspace item, stale revision, optimistic rollback, and restart.

## Release 2 — Reusable Skills and governed Knowledge

### PCR-S05A — Create and version a Skill

As an authorized workspace administrator, I can create, edit as a new version,
publish, deprecate, archive, restore, and inspect Skill provenance.

Acceptance scenarios include immutable published versions, binding retention,
admin/member/agent permissions, audit history, and referenced-version reads.

### PCR-S05B — Import and manage Skill files

As an authorized administrator, I can preview and import a Skill archive and
manage its logical file tree safely.

Acceptance scenarios include new-version conflict default, explicit replacement,
path traversal, link escape, duplicate canonical path, size/count/depth limit,
forbidden type, partial write cleanup, checksum, download, and archive.

### PCR-S06A — Query Knowledge

As a member, I can search and filter Knowledge by status, kind, source,
applicability, and revision, and understand why a result is trustworthy.

Acceptance scenarios include pagination, source projection, superseded results,
workspace isolation, quarantine visibility, and stable ranking.

### PCR-S06B — Review and publish a candidate

As an independent reviewer, I can approve, reject, quarantine, return, publish,
supersede, or invalidate a Knowledge candidate using an expected revision.

Acceptance scenarios include self-review denial, stale review conflict, missing
source evidence, admin emergency reason, immutable audit, and realtime update.

## Release 3 — Complete project context

### PCR-S07A — Manage project Resources

As a project member, I can add, validate, reorder, archive, and restore GitHub
repository and generic URL resources without storing external credentials in
project content.

Acceptance scenarios include invalid URL, duplicate resource, unavailable
external service, connection-state refresh, permission denial, and project
deletion without deleting the external resource.

### PCR-S07B — Govern Requirements

As a project lead, I can create revisions and move a Requirement through draft,
review, approval, freeze, material change, re-review, and retirement.

Acceptance scenarios include independent approval, stale revision, Issue and
outline links, frozen-edit denial, material-change impact, review-required
projection, and history.

### PCR-S07C — See Requirement coverage

As a reviewer, I can distinguish Requirements that are linked, implemented, and
accepted rather than treating any relation as coverage.

Acceptance scenarios cover no links, open Issues, completed Issues, accepted
Issues, retired Requirements, and link removal.

### PCR-S07D — Publish a retrospective and create action items

As a project team, we can draft and publish a retrospective, preserve revisions,
and turn an action item into a task or Issue.

Acceptance scenarios include participant roles, task default, explicit Issue
choice, idempotent retry, source link, published-revision immutability, and
archive.

## Release 4 — Duplicate prevention and dependable reminders

### PCR-S08A — Warn about similar Issues

As an Issue creator, I see ranked possible duplicates before committing or after
a material edit, without exposing another workspace's data.

Acceptance scenarios include exact title, normalized punctuation, near title,
description overlap, identifier, same-project boost, closed candidate, unrelated
candidate, detector unavailable, and latency budget.

### PCR-S08B — Record the human duplicate decision

As a member, I can open a candidate, mark the new Issue as a duplicate, or
continue with an explanation.

Acceptance scenarios include reversible duplicate relation, no automatic merge,
override reason audit, algorithm version, shown-candidate record, and retry.

### PCR-S09A — Use the notification center

As a member, I can view unread notifications, mark them read, archive them, and
configure allowed reminder preferences.

Acceptance scenarios include recipient isolation, stale target tombstone,
pagination, unread count, quiet hours, opt-out, and realtime delivery.

### PCR-S09B — Receive one daily overdue reminder

As an assignee or subscriber, I receive at most one reminder per overdue Issue
per workspace calendar day.

Acceptance scenarios include workspace timezone, midnight and daylight-saving
boundaries, completed/cancelled/undated Issue, paused project default, assignee
change, no-recipient project-lead fallback, duplicate scheduler, retry, restart,
dead letter, and bounded replay.

## Release 5 — Project planning structure and visibility

### PCR-S10A — Move a project through phases

As a project lead, I can move a project through initiation, planning, review,
and implementation while lifecycle status remains independent.

Acceptance scenarios include initial backfill, forward transition, backward
reason, protected transition permission, failed checks, phase history, paused
status, and concurrent conflict.

### PCR-S10B — Build a book-like outline

As a project planner, I can create nested outline nodes, reorder or move them,
and see derived numbering such as `1.2.3` without changing stable identity.

Acceptance scenarios include root and child insert, sibling reorder, cross-parent
move, cycle, maximum depth, stale revision, archive, restore, and restart.

### PCR-S10C — Link Issues and review outline progress

As a project team, we can assign one primary outline node and optional references
to an Issue and see progress roll up from children to the entire project.

Acceptance scenarios include primary uniqueness, multiple references,
cross-project denial, unlinked node, zero-denominator progress, Issue status
change, archived node, and realtime refresh.

### PCR-S10D — See the project phase board

As a portfolio owner, I can view project cards grouped by phase with lifecycle
status, progress, overdue indicators, and stable ordering.

Acceptance scenarios include empty phases, permissions, filters, paused and
cancelled status, project phase move, optimistic conflict rollback, and Web/
Desktop shared behavior.

## Release 6 — Candidate acceptance

### PCR-S11A — Upgrade and recover

As an operator, I can upgrade an existing Canonical database, restart every new
worker, observe backlog and index lag, replay bounded failures, and disable each
feature independently.

### PCR-S11B — Complete user journeys

As an independent reviewer, I can complete the frozen Web and Desktop journeys
against the installed Canonical runtime with no legacy `server/**` traffic.

### PCR-S11C — Decide Customer Acceptance

As the customer, I receive indexed evidence, known limitations, rollback state,
and deferred work, and make an explicit acceptance or rejection decision. No
technical check or model review makes this decision automatically.

## Deferred story — Outline-assisted Issue planning

This story is intentionally outside v1 implementation authority.

As a project planner, I can request an Issue plan from selected outline nodes,
preview rationale and duplicate warnings, edit the proposal, approve an exact
batch, create it idempotently, and roll back only the created batch.

Before approval, a new plan version must decide model/provider authority, data
sent externally, prompt/version evidence, deterministic constraints, maximum
batch size, human approval role, partial-failure semantics, and evaluation.
