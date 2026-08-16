# Product Contract Freeze v1

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Plan-Version: `v1`
- Plan-Step: `PCR-S00`
- Contract version: `PCR-CONTRACT-v1`
- Status: `frozen for implementation planning`
- Frozen against commit: `45213820fade7f61294d2287e063bf19fbd015ee`
- Frozen: `2026-08-16`

This document freezes product semantics and boundary contracts. Individual
implementation tasks may add field-level detail that does not change these
semantics. A change to authority, state machines, security boundaries, delivery
channels, automatic actions, or the public route families requires a new plan
version.

## 1. Authority and ownership

| Data or behavior | Write authority | Allowed dependency | Forbidden coupling |
| --- | --- | --- | --- |
| Task | Workspace | Auth actor/member lookup; Issue reference contract | Control Plane dual write |
| Issue search and similarity | Workspace | Workspace Issue repository; replaceable ranker port | Cross-workspace corpus |
| Pins | Workspace | Project/Issue existence contracts | Client-owned canonical position |
| Skill catalog and versions | System | Auth actor; Space asset contract; Workspace visibility binding | Workspace or Space table reads |
| Skill binary objects | Space | System logical manifest reference | System direct object-store access |
| Knowledge | Workspace | Space evidence reader; Auth actor | Control Plane record/table reads |
| Project Resource | Workspace | Optional external connection adapter | Persisting OAuth tokens in content |
| Requirement | Workspace | Project, Issue, outline reference contracts | Control Plane dual write |
| Retrospective | Workspace | Task/Issue creation contracts | Direct mutation of Task/Issue tables |
| Notification and reminder | Workspace | Auth recipient lookup; Issue/project read contracts | Process-local delivery authority |
| Project phase and outline | Workspace | Project and Issue repositories | Reusing lifecycle status as phase |
| Delivery governance records | Control Plane | Immutable Workspace identifiers/events | Becoming product CRUD authority |

All cross-module calls use public contracts and explicit adapters composed in
bootstrap. No module reads another module's database tables.

## 2. Public route families

Existing client-compatible routes are retained where already declared. New
responses use `snake_case` JSON and TypeScript boundary schemas.

| Capability | Frozen HTTP route family | Core mutation rule |
| --- | --- | --- |
| Task | `GET/POST /api/tasks`, `GET/PATCH/DELETE /api/tasks/{id}`, `POST /api/tasks/{id}/restore`, `POST /api/tasks/{id}/promote` | Promotion requires idempotency key |
| Issue search | `GET /api/issues/search` | `q`, `limit`, `offset`, `include_closed`; stable rank and ID tie-break |
| Project search | `GET /api/projects/search` | Same search pagination contract |
| Pin reorder | `PUT /api/pins/reorder` | Complete item order plus `expected_revision` |
| Skill | `GET/POST /api/skills`, `GET/PUT/DELETE /api/skills/{id}`, `POST /api/skills/{id}/restore` | PUT creates a version; DELETE archives when retained |
| Skill import | `POST /api/skills/import`, `POST /api/skills/import/preview` | Idempotency key and declared conflict mode |
| Skill files | `GET/POST /api/skills/{id}/files`, `GET/PUT/DELETE /api/skills/{id}/files/{path}` | Path is canonical and version-scoped |
| Knowledge | `GET /api/knowledge`, `GET /api/knowledge/{id}`, `POST /api/knowledge/proposals`, `GET /api/knowledge/candidates` | Cursor pagination for revisioned lists |
| Knowledge review | `POST /api/knowledge/candidates/{id}/review` | Action, rationale, expected revision |
| Resources | `GET/POST /api/projects/{id}/resources`, `PUT/DELETE /api/projects/{id}/resources/{resource_id}` | DELETE archives relation, never external target |
| Requirements | `/api/projects/{id}/requirement-baseline/**` | Existing client route family retained; expected revision required |
| Retrospectives | `GET/POST /api/projects/{id}/retrospectives`, `GET/PUT/DELETE /api/projects/{id}/retrospectives/{retrospective_id}` | Published edits create a revision |
| Similarity | `POST /api/issues/similarity/check`, `POST /api/issues/{id}/similarity/check`, `POST /api/issues/{id}/duplicate` | Ranking version returned; human decision required |
| Notifications | `GET /api/notifications`, `PATCH /api/notifications/{id}`, `POST /api/notifications/read`, `GET/PUT /api/notification-preferences` | Recipient identity is server-derived |
| Project phase | `POST /api/projects/{id}/phase-transitions`, `GET /api/projects/{id}/phase-history` | Expected project revision and transition reason |
| Outline | `GET/POST /api/projects/{id}/outline`, `PATCH/DELETE /api/projects/{id}/outline/{node_id}`, `PUT /api/projects/{id}/outline/reorder` | Stable node ID, expected outline revision |
| Outline links | `PUT/DELETE /api/projects/{id}/outline/{node_id}/issues/{issue_id}` | `link_type` is `primary` or `reference` |
| Phase board | `GET /api/projects/phase-board` | Derived projection, not separate authority |

Permanent deletion, where allowed, is a separately authorized operation and is
not implied by `DELETE` in the table above.

## 3. Common API contract

- Workspace resolution uses trusted identity plus `X-Workspace-Slug` or
  `X-Workspace-ID` under the existing resolver rules.
- Cookie-authenticated mutations require the existing CSRF contract. Bearer
  requests follow the existing exemption.
- All list limits default to `50` and are capped at `100` unless a route below
  has a stricter limit.
- Search routes retain `limit/offset` compatibility. New revisioned content
  lists use opaque cursors. Clients must not decode or manufacture cursors.
- Stable sort always ends with immutable ID ascending or descending as declared
  by the route.
- Revisioned mutations carry `expected_revision`. A stale request returns
  `409` with `code=revision_conflict`, `current_revision`, and no mutation.
- Retriable create, import, promotion, action-item, replay, and batch commands
  carry an idempotency key. Reuse with a different body returns conflict.
- Unsupported or disabled features return an explicit unavailable problem and
  never an empty success envelope.
- Create returns `201`, successful reads and updates return `200`, and commands
  with no response body return `204`.
- Validation problems identify stable field names without echoing secrets or
  imported file bodies.
- Every TypeScript-consumed response has a Zod schema, a safe fallback where a
  fallback is semantically honest, and a malformed-response test. Mutations
  must not treat a fallback as success.

## 4. Permission actions

`owner` and `admin` do not bypass workspace membership. Agent grants are
explicit and absent by default.

| Action family | owner | admin | member | agent default |
| --- | --- | --- | --- | --- |
| Task read/create/update own | allow | allow | allow | deny |
| Task manage workspace | allow | allow | deny | deny |
| Search readable content | allow | allow | allow | allow only with explicit read grant |
| Pin manage own sidebar | allow | allow | allow | deny |
| Skill read published | allow | allow | allow | explicit binding only |
| Skill create/import/version/archive | allow | allow | deny | deny |
| Knowledge propose | allow | allow | allow | explicit propose grant |
| Knowledge review | allow with self-review rule | allow with self-review rule | explicit reviewer grant | deny |
| Knowledge emergency self-review override | allow with rationale | deny | deny | deny |
| Resource read | allow | allow | allow | explicit project read grant |
| Resource manage | allow | allow | project lead | deny |
| Requirement edit draft | allow | allow | project lead/editor grant | deny |
| Requirement approve/freeze | allow | explicit approver grant | explicit approver grant | deny |
| Retrospective draft | allow | allow | project member | deny |
| Retrospective publish | allow | allow | facilitator/project lead | deny |
| Similarity check | allow | allow | Issue-create permission | explicit Issue-create grant |
| Duplicate mark or override | allow | allow | Issue-update permission | explicit Issue-update grant |
| Notification read/update | own only | own only | own only | deny |
| Reminder replay/repair | allow | explicit operator grant | deny | deny |
| Project phase normal transition | allow | allow | project lead | deny |
| Project protected transition | allow | explicit approver grant | explicit approver grant | deny |
| Outline edit/reorder/link | allow | allow | project editor grant | deny |

Each concrete action is a named constant and receives table-driven positive and
negative tests. A missing action mapping denies access.

## 5. Frozen state machines

### Task

```text
todo -> in_progress -> done
  \         \-> cancelled
   \--------------> cancelled
done/cancelled -> archived -> restored to previous terminal state
```

Promotion does not change Task status automatically. The caller may request an
atomic promotion-and-complete command explicitly.

### Knowledge

```text
candidate -> in_review -> published -> superseded
                    \-> rejected
                    \-> quarantined -> in_review
published -> invalidated
```

### Requirement

```text
draft -> in_review -> approved -> frozen
  ^         |                     |
  |---------withdraw--------------|
                                 material change
                                      |
                                   changed -> in_review
draft/in_review/approved/frozen/changed -> retired
```

### Retrospective and Skill

```text
retrospective: draft -> published -> superseded -> archived
skill version: draft -> published -> deprecated -> archived
```

### Notification

Delivery and member-view state are independent:

```text
delivery: pending -> delivered
       \-> retry_wait -> pending
       \-> dead_letter -> replay_pending

view: unread -> read -> archived
```

### Project phase

```text
initiation <-> planning <-> review <-> implementation
```

Skipping a phase is a protected transition. Moving backward always requires a
reason. Lifecycle `status` remains unchanged by phase transitions.

## 6. Similarity and search contract

- Search token normalization applies Unicode normalization, case folding, and
  whitespace/punctuation normalization. Chinese text is indexed without
  requiring an external network service.
- Issue search ranks exact identifier, exact normalized title, title terms,
  description terms, and recency in that order before semantic enhancement.
- Similarity returns at most `5` candidates from a maximum internal candidate
  pool of `50`.
- Same-project matches receive a versioned boost; closed or cancelled matches
  are included only when requested and visibly labeled.
- A response contains `ranking_version`, component scores, truncation state,
  and detector availability.
- If optional semantic scoring is unavailable, the route returns lexical
  results plus a degraded diagnostic. It never states that no duplicate exists
  solely because the provider failed.
- Thresholds are configuration version data and may change only with a frozen
  evaluation report. User-facing enforcement remains warning-only in v1.

## 7. Reminder contract

- A workspace calendar date, not server UTC date, determines overdue status and
  the daily idempotency key.
- The scheduler acquires a renewable lease before scanning. A row can be
  claimed by one active lease at a time.
- Default recipients are assignee and subscribers, deduplicated by member ID.
  Project lead is used only when both sets are empty.
- A delivery attempt reauthorizes the recipient's target access. Lost access
  produces no content-bearing notification.
- Quiet hours delay delivery but do not create another daily logical reminder.
- Catch-up emits only the latest eligible daily reminder after downtime; it
  does not flood a recipient with one message for every missed day.
- Replay is operator-authorized, bounded by workspace and time window, audited,
  and uses the original idempotency key.

## 8. Skill import limits

- Maximum compressed request: `10 MiB`.
- Maximum decompressed content: `50 MiB`.
- Maximum files: `500`.
- Maximum individual file: `5 MiB`.
- Maximum canonical path depth: `16` segments.
- Maximum path length: `512` UTF-8 bytes.
- Symbolic links, device files, hard-link aliases, absolute paths, drive paths,
  parent traversal, duplicate canonical paths, and nested archives are rejected.
- Preview and commit run the same validator version. Commit fails if the preview
  token or content checksum no longer matches.
- Objects remain quarantined until metadata, manifest, version, and bindings
  commit. Reconciliation removes only proven-unreferenced quarantine objects.

## 9. Outline and reorder limits

- Maximum outline depth: `8`.
- Maximum active nodes per project: `2,000`.
- Maximum reorder/move batch: `500` node or pin IDs.
- Sibling positions are server-owned contiguous integers and are rewritten
  atomically after validation.
- Display numbering is derived at read/projection time and is never an identity
  or external reference key.
- The outline repository rejects cycles, foreign-project parents, duplicate
  primary links, and stale outline revisions.
- Progress uses current Issue status and acceptance projections. Stored caches
  are rebuildable and never accepted as the source of truth.

## 10. Numeric service budgets

These are acceptance ceilings for a representative local workspace with up to
`10,000` Issues, `2,000` projects, and `2,000` outline nodes per project. PCR-S00
does not claim the implementation currently meets them.

| Surface | Frozen budget |
| --- | --- |
| Issue/project lexical search | p50 <= 100 ms, p95 <= 250 ms, max 50 returned |
| Synchronous similarity check | p95 <= 400 ms lexical path; optional semantic work may continue asynchronously |
| Normal list API | p95 <= 250 ms at page size 50 |
| Outline read with progress | p95 <= 400 ms at 2,000 nodes |
| Pin/outline reorder transaction | p95 <= 500 ms at batch limit |
| Reminder scan batch | 500 Issues per claim |
| Scheduler lease | 60 seconds with renewal before 30 seconds |
| Delivery attempts | initial plus 3 bounded retries with jittered backoff |
| Backlog alert | oldest ready outbox item older than 15 minutes |
| Catch-up window | latest eligible reminder within 48 hours |
| Notification retention | 365 days after archive unless governance policy is stricter |
| Audit retention | 730 days minimum for roadmap-controlled actions |
| Realtime client queue | preserve current bounded queue and slow-consumer eviction contract |

Any budget change after product code begins requires an append-only decision and,
if it changes user-visible acceptance or capacity, a new plan version.

## 11. Capability flags and operations

Flags remain false until their entire installed vertical slice passes its step
acceptance:

```text
tasks
issue_search
project_search
pin_reorder
skill_administration
skill_import
knowledge_query
knowledge_review
project_resources
project_requirements
project_retrospectives
issue_similarity
notifications
overdue_reminders
project_phases
project_outline
project_phase_board
```

Operational projections expose flag state, schema version, search projection
lag, outbox ready count and oldest age, reminder last successful scan, active
lease age, dead-letter count, and Skill quarantine count. They expose no user
content, URLs, Skill paths, notification text, or credentials.

## 12. Migration order

Migrations are additive and execute in this dependency order:

1. governance actions, revisions, idempotency records, audit, and outbox;
2. capability-specific authority tables and their application cleanup paths;
3. search projection metadata;
4. notifications and reminder scheduling;
5. project phase, outline, and derived projection metadata.

Each capability step owns its tables rather than placing every table in S01.
Every up migration has a down migration where repository policy requires it,
but rollback defaults to code/flag rollback with retained data. Data-destructive
down migrations require separate authority and are not part of normal rollback.

## 13. Project backfill decision

Existing projects are backfilled as follows without changing lifecycle status:

- `planned` -> `planning`;
- `in_progress`, `paused`, `completed`, or `cancelled` -> `implementation`.

The mapping records `source=backfill_v1`. A project owner may later move the
phase with the normal transition history. Existing Issue parent-child hierarchy
is not converted into outline nodes automatically.

## 14. Deferred contracts

The following require a later approved plan version:

- email, Slack, webhook, or external push notification adapters;
- external semantic provider and its data processing terms;
- workspace-configurable project phase definitions;
- mobile UI;
- automatic outline-to-Issue planning or autonomous writes;
- automatic duplicate merge;
- general project file/document management beyond resource links and existing
  attachment contracts.
