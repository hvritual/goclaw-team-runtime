# Product Capability Roadmap Story Map

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Plan-Version: `v40`
- Status: `Release 3 aggregate DoneGate active under r045; r044 audit-blocked; PCR-S07A-D complete-independent-reviewed`

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

Status: `complete-independent-reviewed under PRODUCT-CAPABILITY-ROADMAP-001 v13 / r018`

As a member, I can create, view, filter, reorder, edit, complete, cancel, archive,
and restore tasks in my workspace.

Acceptance scenarios cover member/agent assignees, due dates, stable ordering,
empty and denied states, concurrent edit conflicts, and restart persistence.

### PCR-S02B — Promote a task to an Issue

Status: `complete-independent-reviewed under PRODUCT-CAPABILITY-ROADMAP-001 v16 / r021; r020 closed via r021; v14/r019 remains independent-review-blocked history`

As a member, I can explicitly promote a task into an Issue without creating
duplicates on retry.

Acceptance scenarios:

1. Promotion creates one Issue and retains an immutable source link.
2. Retrying the same command returns the same Issue.
3. Later Task or Issue edits do not silently synchronize content.

### PCR-S03A — Search Issues

Status: `complete-independent-reviewed under PRODUCT-CAPABILITY-ROADMAP-001 v17 / r022`

As a member, I can find an Issue by title, description, human identifier, or
number with stable ranking and closed-state filters.

Acceptance scenarios include Chinese and English queries, pagination,
cancellation, empty query rules, closed inclusion, and workspace isolation.

### PCR-S03B — Search projects

Status: `complete-independent-reviewed under PRODUCT-CAPABILITY-ROADMAP-001 v18 / r023`

As a member, I can find a project using the same stable search contract through
the installed HTTP and shared client surfaces.

Acceptance scenarios include closed-state filters, deterministic tie-breaks,
membership denial, malformed responses, and restart consistency.

### PCR-S04 — Reorder pins

Status: `complete-independent-reviewed under PRODUCT-CAPABILITY-ROADMAP-001 v19 / r024`

As a member, I can reorder my project and Issue pins, and concurrent changes do
not silently overwrite one another.

Acceptance scenarios include complete ordered payload, duplicate item, missing
item, foreign workspace item, stale revision, optimistic rollback, and restart.

## Release 2 — Reusable Skills and governed Knowledge

### PCR-S05A — Create and version a Skill

Status: `complete-independent-reviewed under PRODUCT-CAPABILITY-ROADMAP-001 v20 / r025`

As an authorized workspace administrator, I can create, edit as a new version,
publish, deprecate, archive, restore, and inspect Skill provenance.

Acceptance scenarios include immutable published versions, binding retention,
admin/member/agent permissions, audit history, and referenced-version reads.

### PCR-S05B — Import and manage Skill files

Status: `complete-independent-reviewed under PRODUCT-CAPABILITY-ROADMAP-001 v21 / r026`

As an authorized administrator, I can preview and import a Skill archive and
manage its logical file tree safely.

Acceptance scenarios include new-version conflict default, explicit replacement,
path traversal, link escape, duplicate canonical path, size/count/depth limit,
forbidden type, partial write cleanup, checksum, download, and archive.

### PCR-S06A — Query Knowledge

Status: `complete-independent-reviewed under PRODUCT-CAPABILITY-ROADMAP-001 v22 / r027`

As a member, I can search and filter Knowledge by status, kind, source,
applicability, and revision, and understand why a result is trustworthy.

Acceptance scenarios include pagination, source projection, superseded results,
workspace isolation, quarantine visibility, and stable ranking.

### PCR-S06B — Review and publish a candidate

Status: `complete-independent-reviewed under PRODUCT-CAPABILITY-ROADMAP-001 v25 / r030; v24 / r029 design-blocked; v23 / r028 verification-blocked`

As an independent reviewer, I can approve, reject, quarantine, return, publish,
supersede, or invalidate a Knowledge candidate using an expected revision.

Acceptance scenarios include self-review denial, stale review conflict, missing
source evidence, admin emergency reason, immutable audit, and realtime update.

## Release 3 — Complete project context

### PCR-S07A — Manage project Resources

Status: `complete-independent-reviewed under PRODUCT-CAPABILITY-ROADMAP-001 v28 / r033; v27 / r032 and v26 / r031 are review-blocked history`

As a project member, I can add, validate, reorder, archive, and restore GitHub
repository and generic URL resources without storing external credentials in
project content.

Acceptance scenarios include invalid URL, duplicate resource, unavailable
external service, connection-state refresh, permission denial, and project
deletion without deleting the external resource.

### PCR-S07B — Govern Requirements

Status: `complete-independent-reviewed under PRODUCT-CAPABILITY-ROADMAP-001 v30 / r035; v29 / r034 is review-blocked history`

As a project lead, I can create revisions and move a Requirement through draft,
review, approval, freeze, material change, re-review, and retirement.

Acceptance scenarios include independent approval, stale revision, Issue and
outline links, frozen-edit denial, material-change impact, review-required
projection, and history.

S07B includes only the confirmed prerequisite outline authority: persistent
project-owned root nodes, stable IDs, create/read, ownership validation, and
Requirement-to-outline links. Nested hierarchy, move/reorder, numbering, node
management, Issue-outline links, progress rollups/board, realtime outline
projection, and the standalone outline UI remain owned by S10 and inactive.

### PCR-S07C — See Requirement coverage

Status: `complete-independent-reviewed under PRODUCT-CAPABILITY-ROADMAP-001 v34 / r039; v33 / r038 is review-blocked; v32 / r037 stopped at the official race gate; v31 / r036 stopped at the aggregate backend gate`

As a reviewer, I can distinguish Requirements that are linked, implemented, and
accepted rather than treating any relation as coverage.

Acceptance scenarios cover no links, open Issues, completed Issues, accepted
Issues, retired Requirements, and link removal.

v31/r036 freezes the installed coverage stages as `unlinked`, `linked`,
`implemented`, and `accepted`. Multiple linked Issues advance only when every
current Issue satisfies the next stage; latest acceptance conclusions supersede
older conclusions. Current/effective Requirement content and link intervals are
revision-relative, while Issue status and latest acceptance remain current
canonical projections. v32/r037 changes no coverage or Auth production behavior
and stabilizes one aggregate-load test server. v33/r038 changes no production
behavior and protects only a test-local injected clock from its real outbox
reader. Independent review of exact r038 blocks on persisted-content
revalidation, active-link baseline ownership, UI query-error projection, and
executable query-count evidence. v34/r039 changes only those boundaries while
preserving every coverage stage and public contract. The first targeted
installed run additionally exposed the same stored ownership drift through the
sibling baseline link projection; the v34-authorized repository boundary now
rejects both baseline and coverage projections before returning foreign Issue
detail. Repaired full backend/race and fresh 38-check installed gates pass.
Exact r039 scope/trailer/dirty/process audit and fresh independent `SPEC PASS`
plus `CODE/SECURITY/QUALITY PASS` close S07C. PCR-S07D stays inactive until its
own immutable successor plan is activated.

### PCR-S07D — Publish a retrospective and create action items

Status: `complete-independent-reviewed under PRODUCT-CAPABILITY-ROADMAP-001 v38 / r043`

As a project team, we can draft and publish a retrospective, preserve revisions,
and turn an action item into a task or Issue.

Acceptance scenarios include participant roles, task default, explicit Issue
choice, idempotent retry, source link, published-revision immutability, and
archive.

v35 freezes active-member participant/facilitator authority, immutable complete
snapshots, an opaque-cursor HTTP/Core boundary, resumable at-most-one target
claims through injected Task/Issue creation contracts, safe server access
projection, and a loaded four-locale shared UI. Release 3 completion stays
inactive until S07D independently closes and a later aggregate DoneGate passes.

v36 preserves that complete contract and corrects one discovered write-boundary
omission: both already-installed Project deletion repositories must invoke the
same scoped Retrospective cleanup transaction. Target Tasks/Issues and immutable
audit/outbox evidence survive Project deletion. No other behavior or path scope
changes.

v37 preserves the same product contract and adds only the omitted Workspace
integration-test path needed to advance the exact installed migration count
from 19 to 20. It adds no behavior or product scope.

v38 preserves the same complete product contract and corrects one immutable
path spelling before replaying the candidate: the real Core mutation test is
`mutations.test.tsx`, not the nonexistent `.ts` path written in v35. The r042
candidate remains scope-blocked provenance; r043 must replay the same 45 product
paths after authorization and pass fresh candidate, installed, and independent
review gates.

Exact r043 product bytes match blocked r042 after the corrected authority.
Fresh complete backend/race/frontend/build and two-identity installed
lifecycle/restart/archive evidence passes with retained broad NON-PASS
disclosures. Exact scope, trailer, dirty-tree, and process audits pass, and
fresh independent `SPEC PASS` plus `CODE/SECURITY/QUALITY PASS` close S07D.
All Release 3 stories are closed; Release 3 completion remains pending the
separately frozen aggregate DoneGate.

### Release 3 aggregate DoneGate

Status: `complete-independent-reviewed under PRODUCT-CAPABILITY-ROADMAP-001 v41 / r046; v40 / r045 review-blocked before closure; v39 / r044 audit-blocked before capability-matrix write`

The aggregate gate reopens no story and authorizes no product change. It freezes
the four reviewed candidate/tree/closure tuples, reconciles only their stale
capability-matrix rows, reruns the complete deterministic candidate gates, and
uses one fresh Canonical database plus production Web runtime to prove Project
Resources, Requirements/coverage/minimal outline, and Retrospectives coexist
with two identities and restart persistence. Exact documentation scope,
product-byte stability, trailer lineage, dirty-tree isolation, process cleanup,
and fresh independent dual PASS are mandatory before Release 3 may close.

R44.2 proves the story tuples, registered plan hashes, and zero forbidden/product
drift, but v39 incorrectly requires the historical v26 activation to contain an
`Issue` trailer that was not present in its frozen authority. Policy requires
that field only when present. v40 preserves every aggregate product/runtime
gate and corrects only the lineage predicate: exact eight fields for
`71afb3c3`, exact nine fields for every later commit. r044 remains blocked
history and Release 3 remains incomplete while r045 executes.

R45.2 passes the corrected lineage/scope/matrix audit and R45.3 completes the
full deterministic backend, race, frontend, root, lint, and production-build
evidence without relabeling retained broad NON-PASS results. R45.4 then passes
the one-database, one-Project, two-identity installed journey: Resource
create/reorder/archive/restore; Requirement revision 13 with independent
review/freeze/material re-review, one root outline, and 4/4 linked,
implemented, and accepted coverage; and Retrospective publish/supersede/archive
with byte-stable default Task replay and explicit Issue provenance from source
revision 3. All state survives backend restart and is visible in production
Web with zero Next overlay and zero clean-tab console error. Owned processes,
browser tabs, and ports close; task-owned F-drive artifacts remain only because
host policy blocks their exact removal.

R45.5 freezes candidate `a63bf58a` (tree `aba1d1c1`, r045 patch SHA-256
`f9322809dba511553c127c490705c6438943ad6466c631ff504ae2bc23b9260b`).
Fresh review independently confirms its exact six-document scope and returns
`CODE/SECURITY/QUALITY PASS`, but returns `SPEC BLOCK`: `9fb86ea0` has nine
correct historical fields separated by blank lines rather than one continuous
block, and the v39-required external-directory removal did not execute. r045
stops before closure.

v41/r046 preserves every product and story byte, freezes the exact
blank-separated `9fb86ea0` shape beside the valid eight-field `71afb3c3` and
continuous-nine-field remainder, and permits only an explicitly inventoried,
zero-live-resource host-policy retention disposition. Corrected audit and fresh
independent dual PASS still gate Release 3; Release 4 remains inactive.

R46.2 passes the raw-message-aware audit at activation candidate `b9719f73`:
47 commits resolve as one continuous eight-field, one exact blank-separated
nine-field, and 45 continuous nine-field messages; all registered plan/policy
hashes and story tuples match. The 114-path product patch remains byte-exact,
with zero forbidden/generated/dirty overlap, while the five-path r046 boundary
and host-policy-retained artifact inventory pass. Final exact candidate and
fresh independent dual PASS still gate closure.

R46.3 exact candidate `daab0777` (tree `b012a2c1`, r046 patch SHA-256
`528aa873dea5d477be4ddbdef956b643d393204f6811bd19da08c02f5689d482`)
passes the complete corrected audit. Fresh independent review returns
`SPEC PASS` and `CODE/SECURITY/QUALITY PASS`, independently confirming the
three lineage shapes, unchanged story/product evidence, exact scope, retained
NON-PASS disclosures, and bounded host-policy-retained artifact disposition.
R46.4 closes the aggregate gate and Release 3 with zero active tasks.

## Release 4 — Duplicate prevention and dependable reminders

Status: `S08A r049 review-blocked; PCR-001-S08A-R050 bounded-input correction active under independent plan PASS; S08B and S09 remain inactive`

### PCR-S08A — Warn about similar Issues

As an Issue creator, I see ranked possible duplicates before committing or after
a material edit, without exposing another workspace's data.

Acceptance scenarios include exact title, normalized punctuation, near title,
description overlap, identifier, same-project boost, closed candidate, unrelated
candidate, detector unavailable, and latency budget.

R49 selectively integrated the bounded warning candidate into an isolated
review candidate, but independent code/security/quality review blocks it because
normalized title and description input can create unbounded SQLite predicate
terms. Draft R50 is limited to assertion-first normalized input bounds before
authorization/repository use. It does not merge the Release 4 source branch or
activate a duplicate-decision record, relationship persistence, reminders,
generated contracts, or a legacy backend change. Fresh independent SPEC and
code/security/quality PASS remain required before the story may be marked
complete.

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
