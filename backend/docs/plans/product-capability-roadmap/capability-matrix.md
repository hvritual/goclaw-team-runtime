# Product Capability Baseline

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Baseline commit: `45213820fade7f61294d2287e063bf19fbd015ee`
- Evidence method: static repository inspection only
- Runtime verification performed for this baseline: none

## Status legend

- **Installed**: assembled into the Canonical runtime with an accessible product
  contract and repository evidence.
- **Partial**: some UI, API, domain, persistence, or test layers exist, but the
  end-to-end Canonical capability is incomplete.
- **Missing**: no adequate Canonical product implementation was found.
- **Separate**: behavior exists in another bounded runtime and is not the
  Workspace product authority.
- **Deferred**: an approved or historical plan explicitly excludes the feature.
- **Unknown**: static evidence cannot resolve the behavior.

## 1. Capability summary

| Capability | Web/Core evidence | Canonical backend evidence | Current state | v1 target |
| --- | --- | --- | --- | --- |
| Skill list/detail/create/edit/delete/import/files | Shared Skill views and API client methods exist | System Skill service only declares publish/get and returns not implemented; no HTTP administration | Partial UI, missing backend, deferred | Versioned Workspace-visible Skill catalog with safe import and file manifests |
| Task CRUD | Shared task page, queries, mutations, API client methods exist | Todo proto/use case/SQLite CRUD exist; default service and HTTP installation are absent; runtime authorization lacks actions | Partial, not installed, deferred | Canonical task CRUD and explicit task-to-Issue promotion |
| Knowledge query and review | Knowledge page, schemas, candidate review client methods exist | Workspace supports candidate create/get only; no list/query/review transitions | Partial, deferred | Query, review, publication, provenance, revision, audit |
| Project Resources | Loaded shared Project Resources surface with strict Core schemas, create/reorder/update/archive/restore mutations, safe connection projection, and localized states | Installed Workspace migration/repository/HTTP/runtime authority with current membership checks, revision conflicts, credential-free normalization, ordered archive/restore, audit/idempotency, cleanup, and restart evidence | Installed and independently reviewed under PCR-S07A | GitHub repository and generic URL resources |
| Requirements | Loaded shared Requirement baseline, revision/history, review/freeze/material-change, minimal root-outline link, and linked/implemented/accepted coverage surface | Installed Workspace lifecycle/outline/Issue-link/coverage migration, repository, HTTP, runtime permission and feature-flag authority with current transaction authorization, ownership guards, history, cleanup, and restart evidence | Installed and independently reviewed under PCR-S07B/C; outline limited to the confirmed root prerequisite | Workspace-owned lifecycle, traceability, coverage, impact review |
| Retrospectives | Loaded four-locale shared draft/participant/history/publication/archive and Task/Issue action-target surface with strict Core schemas and stable retry keys | Installed Workspace immutable revisions, current facilitator/lead authorization, resumable target claims through injected Task/Issue contracts, provenance links, audit/outbox, project cleanup, and restart persistence | Installed and independently reviewed under PCR-S07D | Versioned retrospective and action-item creation |
| Pin reorder | Sidebar mutation and client method exist | Pins list/create/delete; position appends with `MAX+1`; reorder deferred | Partial, deferred | Revisioned atomic reorder |
| Issue search | Search UI and client method exist | Installed HTTP list/query filters in memory by title substring, identifier, or number; no repository search RPC | Partial | Repository-backed ranked search |
| Project search | Search UI and client method exist | Opt-in gRPC/repository search exists; Product Surface HTTP route and installed default service are absent | Partial, not installed | Installed stable HTTP/gRPC search |
| Issue similarity | No authoritative shared detector found | No Issue duplicate/similarity service found | Missing | Warning-only hybrid detector and human duplicate relation |
| Daily overdue reminders | Due-date display exists | Due dates exist; no scheduler, notification service, durable reminder state, or operational entrypoint | Missing | Durable in-product notifications and daily scheduler |
| Project phase | Existing UI/backend use lifecycle status only | Project domain has planned/in-progress/paused/completed/cancelled; no separate phase | Missing | Separate phase plus guarded history |
| Project outline | Issue hierarchy UI can represent child Issues | Issue parent/child exists; no project outline/TOC model | Missing | Ordered tree, stable numbering, Issue links |
| Project progress board | Project issue counts and views exist | Project counts exist; no phase/outline board | Partial projection | Phase board plus outline progress rollups |
| Auto-plan Issues from outline | No approved UI contract | No model or service found | Deferred by v1 decision | Later proposal only |

### 1.1 Release 3 governed reconciliation — 2026-08-20

The baseline commit and static-inspection metadata above remain the historical
v1 discovery record. Only the Project Resources, Requirements, and
Retrospectives rows are reconciled here to later governed Release 3 evidence;
all other rows retain their prior evidence date and must not be inferred current
from this scoped update.

- PCR-S07A exact reviewed candidate/tree/closure is
  `b3828be7b9b272732c5630975e73e35b629ed9f9` /
  `7c4a45fff414a555688358bd938111f8105c774f` /
  `07aef1a577db78598c92c70312a33989e6177d64`.
- PCR-S07B exact reviewed candidate/tree/closure is
  `cd94396093ea73f3f9434fed7410036ae61170ab` /
  `7e6f045ec5a48c4465e7f2fd5261e0d2a3b4b42d` /
  `f5695de83d55e277c8eeb9db7461b81137dc93ad`; PCR-S07C is
  `47ee4189cb5571ec38ae39480c758d4decad22bd` /
  `d0b7d56b65964e1559e3bbe33aa734f70e2f8eca` /
  `1d515efcca0919eed1e8a811c53d015efa89dfa3`.
- PCR-S07D exact reviewed candidate/tree/closure is
  `64091302b703a4590bdbe88d154f65fec9d6b37c` /
  `e696d67ad72aad52bc53e4a6bfe3211aac2f89d7` /
  `8150a0e53defe1562c5ea5b41de34bbdba3a178e`.
- v40/r045 R45.2 verifies all four tuples, registered v26-v40 hashes, 131
  Release 3 paths (114 product plus 17 roadmap), zero `server/**`, zero
  generated paths, zero original-dirty overlap, zero r045 product drift, and
  policy-correct trailers across 43 commits. Deterministic, aggregate installed,
  exact-candidate, and independent-review gates remain pending; this
  reconciliation is not a Release 3 completion claim.

## 2. Installed-runtime and authorization gap

The Canonical server composes the SQLite Workspace chain in
`backend/internal/bootstrap/sqlite.go`, but composition is not sufficient proof
of usability. The bootstrap membership adapter currently recognizes Issue,
project, and pin actions, not the Todo, Knowledge, Requirement, Setting, or Skill
actions required by this roadmap. Generated default services for several domains
still return explicit not-implemented errors. PCR-S01 must close this gap without
granting broad permissions or changing a capability flag before its vertical
slice is ready.

## 3. Source evidence index

### Runtime and plan boundaries

- `backend/cmd/server/main.go`: Canonical server entrypoint.
- `backend/internal/bootstrap/runtime.go`: health, readiness, configuration, and
  capability projection.
- `backend/internal/bootstrap/sqlite.go`: installed module composition and
  membership authorization adapter.
- `backend/docs/plans/canonical-sqlite-runtime-cutover/plan.md`: current cutover
  pointer; v10 is blocked with no active step.
- `backend/docs/plans/canonical-sqlite-runtime-cutover/parity-matrix.md`: explicit
  deferrals including Skills administration, tasks, Requirements, and broader
  administration.
- `backend/docs/plans/canonical-sqlite-runtime-cutover/plan_v5.md`: pin reorder,
  Resources, Retrospectives, and Requirement baseline were not included.
- `backend/docs/plans/canonical-sqlite-runtime-cutover/plan_v7.md`: Inbox, Skills
  administration, tasks, and Knowledge browsing remained non-goals.

### Tasks

- `backend/api/workspace/v1/todo.proto`: create/get/list/update/status/delete
  contract.
- `backend/internal/modules/workspace/internal/application/todo_usecase.go`:
  implemented use cases.
- `backend/internal/modules/workspace/internal/infrastructure/sqlite/todo_repository.go`:
  local persistence.
- `backend/internal/modules/workspace/internal/application/todo_service.go`:
  generated default service remains not implemented.
- `packages/views/tasks/tasks-page.tsx` and `packages/core/tasks/`: current shared
  frontend surface.

### Skills

- `backend/api/system/v1/skill.proto`: publish/get-only System contract.
- `backend/internal/modules/system/internal/application/skill_service.go`:
  explicit not-implemented implementation.
- `packages/core/types/skill.ts`: file-manifest client model.
- `packages/views/skills/`: current list, detail, create, import, and file views.
- `packages/core/api/client.ts`: current CRUD/import client expectations.

### Knowledge and Requirements

- `backend/api/workspace/v1/knowledge.proto`: create/get-only Workspace contract.
- `backend/internal/modules/workspace/internal/application/knowledge_usecase.go`:
  candidate create/get behavior.
- `backend/api/workspace/v1/requirement.proto`: save/get version contract.
- `backend/internal/modules/workspace/internal/application/requirement_usecase.go`:
  draft version behavior and reference validation.
- `backend/internal/controlplane/p2_flows.go`: richer but separate append-only
  Requirement and Knowledge flows; evidence only, not Workspace authority.
- `packages/core/knowledge/`, `packages/core/types/knowledge.ts`, and
  `packages/views/knowledge/`: current frontend review expectations.

### Projects, pins, and search

- `backend/internal/modules/workspace/internal/interfaces/http/project_surface.go`:
  project CRUD and pin routes only.
- `backend/internal/modules/workspace/internal/infrastructure/sqlite/project_surface_repository.go`:
  resource-count omission and append-only pin positioning.
- `backend/internal/modules/workspace/internal/application/project_usecase.go` and
  `backend/internal/modules/workspace/internal/infrastructure/sqlite/project_repository.go`:
  opt-in project search behavior.
- `backend/internal/modules/workspace/internal/interfaces/http/issue_read.go`:
  installed Issue query and in-memory text filtering.
- `packages/views/search/search-command.tsx`,
  `packages/views/modals/issue-picker-modal.tsx`, and
  `packages/core/api/client.ts`: frontend search contracts.
- `packages/views/layout/app-sidebar.tsx` and `packages/core/pins/mutations.ts`:
  frontend pin-reorder expectation.

### Hierarchy, dates, and missing operational services

- `backend/internal/modules/workspace/internal/interfaces/http/issue_hierarchy.go`:
  Issue child progress and movement; not a project outline.
- `backend/api/workspace/v1/issue.proto`: Issue dates, parent relation, and stage
  field; no reminder or project-outline ownership.
- `backend/internal/modules/workspace/internal/domain/project/project.go`: five
  lifecycle statuses and no project phase.
- No Canonical business implementation for reminder, notification, scheduler,
  project outline, or Issue similarity was found in the scoped backend search.

## 4. Data baseline

Existing Workspace SQLite migrations provide workspaces, projects, todos,
Issues, Knowledge candidates, Requirement drafts and versions, settings, Skill
bindings, pins, collaboration, catalog, and attachment references. They do not
provide authoritative tables for:

- Skill catalog versions or logical file manifests;
- Knowledge review history or sources;
- project resources or retrospectives;
- Requirement review/approval/freeze/change links;
- similarity decisions or duplicate relations;
- notifications, preferences, reminder schedules, attempts, or dead letters;
- project phase history, outline nodes, outline ordering, or outline-Issue links.

The implementation plan must use module contracts and additive migrations. It
must not add foreign keys, cascades, or cross-module table reads.

## 5. Evidence limitations

- No server, browser, migration, race, or release test was executed to create
  this baseline.
- Existing frontend API methods do not prove a reachable backend route.
- Existing internal use cases do not prove runtime authorization or transport
  installation.
- Existing Control Plane flows do not prove Workspace product compatibility.
- Current untracked `docs/code-to-product/` material is preserved and is not the
  authority for this plan.
- Status must be refreshed at PCR-S00 because the repository may advance before
  implementation approval.
