# IoT Engineering Digital Thread — Implementation Plan v1

- Plan-ID: `IOT-ENGINEERING-DIGITAL-THREAD-001`
- Version: `1`
- Status: **Approved / executing**
- Approval source: explicit user instruction on 2026-08-28
- Base commit: `d3bbafb071dc493bd17d5c0387297bbf38da9ecb`
- Canonical branch: `codex/multica-six-domain-baseline`
- Work item: `IOT-EDT-001`

## 1. Objective

Evolve Multica from a six-domain AI-native task manager into an IoT engineering project, execution, and knowledge-compounding control plane without replacing its existing Workspace/Auth/Space/System/Execution boundaries.

The product must provide one traceable engineering digital thread:

`Goal -> Project -> Requirement -> Task -> Run -> Change -> Code/PR -> Build -> Release -> Deployment -> Observation/Incident -> Fact -> Knowledge -> Standard/Skill -> next ContextPack`.

The control plane owns identity, governance, relationships, revisions, provenance, and context compilation. Existing source systems continue to own their facts: Git owns code, GitHub owns commit/PR identity, CI owns build/test evidence, deployment systems own deployment evidence, monitoring owns runtime observations, Workspace owns work intent, Runtime owns execution attempts, and governed Knowledge owns organizational conclusions.

## 2. First-principles model

The system distinguishes seven concepts that must never be collapsed:

1. **Intent** — why a change is desired: Objective/Project/Requirement.
2. **Work** — committed team work: Issue/Task/Todo.
3. **Engineering Entity** — durable engineering reality: product, engineering-system, service, component, repository, API, thing-model, environment.
4. **Change** — an accepted engineering mutation connecting work to changed engineering entities and external artifacts.
5. **Evidence / Fact** — immutable observations and rebuildable claims with provenance.
6. **Knowledge** — governed human/organizational interpretation: Architecture, ADR, Standard, Runbook, Troubleshooting, Lesson.
7. **Context** — a frozen, reproducible set of authoritative references compiled for one work-item revision or Run.

`Project != Engineering Entity`. A project is a temporary change container; an engineering entity remains after projects close.

`Task != Run != Change`. Task is desired work, Run is an execution attempt, Change is the accepted engineering mutation.

`Evidence != Fact != Knowledge`. Evidence is immutable source material, Facts are rebuildable claims, Knowledge is governed interpretation.

## 3. Existing boundaries that must remain invariant

- `server/**` is permanently read-only migration evidence.
- Six-domain user experience (`Workspace`, `Member`, `Project`, `Issue`, `Task`, `Skill`) remains intact.
- Workspace continues to own Project/Requirement/Issue/Todo/Knowledge intent and DoneGate semantics.
- Execution/Runtime continues to own Run, Attempt, Claim, Lease, Heartbeat, Retry, Cancellation, runtime logs, artifacts, and execution Evidence; a successful Run never self-accepts.
- Existing `System` context continues to own Agent Release / Agent Version / Skill publication. IoT software systems must be represented as `EngineeringEntity{type=engineering_system}`, never by expanding the existing System context.
- Auth remains the authority for human and Agent identity/membership.
- Space remains the authority for asset bytes and lifecycle.
- Knowledge remains governed: automated execution may propose knowledge; it may not silently publish human/organizational conclusions.
- No database foreign keys or cascade actions are introduced.

## 4. Target architecture

### 4.1 Intent / Work Plane

Existing Multica domains remain the human work surface:

- Workspace
- Member
- Project
- Requirement
- Issue
- Task/Todo
- Skill

They answer: why are we doing this, who owns it, what is expected, and what is its delivery state?

### 4.2 Engineering Thread Plane

A new backend `engineering` bounded context owns the canonical digital-thread identities and provenance-bearing projections:

- `EngineeringEntity`
- `SourceBinding`
- `ThreadEdge`
- `Change`
- `ContextPack`

It does **not** become a CMDB that manually duplicates GitHub/CI/runtime truth. Source bindings and edge provenance always identify where a fact came from.

### 4.3 Execution Plane

Existing Runtime consumes a frozen work-item revision plus a frozen ContextPack. Runtime produces Evidence and operational state only.

### 4.4 Evidence & Knowledge Plane

Long-term flow:

`Evidence -> Fact -> KnowledgeCandidate -> Review -> Published Knowledge -> Standard/Skill`.

Knowledge may be promoted into an executable policy/check when deterministic enforcement is possible.

### 4.5 Context Plane

Context compilation resolves task scope through authoritative typed edges, applies source-authority and freshness policies, collects governed knowledge and recent change/incident context, ranks by token budget, and freezes the result as a ContextPack with revision/checksum metadata.

## 5. Core ontology

### 5.1 EngineeringEntity types

Initial canonical types:

- `product`
- `engineering_system`
- `application`
- `service`
- `component`
- `repository`
- `api`
- `thing_model`
- `environment`

Each entity has stable ID, workspace scope, name, type, lifecycle status, optional owner reference, timestamps, and metadata. Source-specific identifiers are not used as the canonical primary identity.

### 5.2 SourceBinding

Links a canonical entity to an external source without transferring ownership of the external truth.

Examples:

- service -> GitHub repository URL
- repository -> GitHub repo node/full name
- API -> OpenAPI path/revision
- deployment environment -> deployment platform identity

A binding records source type, source locator, source revision when available, authority class, and observed time.

### 5.3 ThreadEdge

A directed typed relationship with provenance. Initial relation vocabulary:

- `part_of`
- `depends_on`
- `implements`
- `provides`
- `uses`
- `changes`
- `affects`
- `contributes_to`
- `constrains`
- `governs`
- `operates`
- `owns`
- `introduced_by`
- `included_in`
- `deployed_to`

Authority classes:

- `authoritative`
- `observed`
- `inferred`
- `proposed`

Every edge must identify source/provenance; inferred/proposed edges are never silently treated as authoritative.

### 5.4 Change

A Change is the accepted engineering mutation that closes the semantic gap between Task/Run and code/release artifacts.

Minimum fields: stable ID, workspace/project references, optional requirement/task/run refs, change status, summary, affected engineering entity IDs, external artifact refs, created/accepted timestamps, and provenance.

A Run may produce zero or more proposed Changes. Acceptance of a Change is independent from process exit and Runtime completion.

### 5.5 ContextPack

An immutable manifest for one work-item revision or Run. It records:

- work-item identity and revision
- target engineering entities
- requirements
- architecture/ADR/standards/runbooks
- recent changes/incidents/facts
- source revisions/checksums
- selection policy version
- creation timestamp
- content checksum

The first implementation freezes references and metadata; later phases add token-aware materialization.

## 6. Federated source-of-truth matrix

| Concern | Authority |
| --- | --- |
| Project/Requirement/Issue/Task | Workspace |
| Member/Agent identity and authorization | Auth |
| Run/Attempt/Lease/Execution Evidence | Execution/Runtime |
| Agent Release and Skill publication | existing System context |
| Asset bytes | Space |
| Repository/Commit/PR | Git/GitHub |
| API schema | OpenAPI/Proto repository source |
| Build/Test | CI |
| Release artifact | artifact/release system |
| Deployment | deployment platform |
| Runtime metric/log | observability source |
| Architecture/ADR/Standard/Runbook/Lesson | governed Knowledge |
| Digital-thread identity/edge/source mapping | Engineering Thread |

Engineering Thread stores canonical IDs, sourced projections, and provenance. It must not claim stronger authority than its source.

## 7. Phase roadmap

### Phase 1 — Engineering Digital Thread MVP

Goal: establish the ontology and a usable backend foundation without changing the six-domain product semantics.

#### P1-S01 — Architecture contract and plan

Allowed paths:

- `CONTEXT-MAP.md`
- `docs/architecture/**`
- `backend/docs/plans/iot-engineering-digital-thread/**`

Deliverables:

- this versioned plan
- engineering digital-thread architecture note
- ontology/boundary update in Context Map

Acceptance:

- no `server/**` diff
- existing System and Runtime ownership remains explicit
- Project/System and Task/Run/Change distinctions are documented

#### P1-S02 — Engineering domain foundation

Allowed paths:

- `backend/internal/modules/engineering/**`

Deliverables:

- domain types and deterministic validation for EngineeringEntity, SourceBinding, ThreadEdge, Change, ContextPack
- authority/lifecycle/relation enums
- repository port interfaces
- domain tests for invalid identities, self-edges, missing provenance, invalid authority transitions, frozen ContextPack checksum/revision rules

Non-goal: persistence, HTTP/gRPC UI, GitHub synchronization.

Acceptance commands:

- `cd backend && go test ./internal/modules/engineering/... -count=1`
- `cd backend && go vet ./internal/modules/engineering/...`
- `git diff --name-only <base>..HEAD -- server/` must be empty

#### P1-S03 — Durable local adapter

Allowed paths:

- `backend/internal/modules/engineering/internal/infrastructure/sqlite/**`
- `backend/internal/modules/engineering/**` only as required by repository ports

Deliverables:

- SQLite persistence for entities, bindings, edges, changes, ContextPacks
- adapter-local migrations
- no foreign keys/cascades
- transactional application-level cleanup where needed
- contract tests shared with in-memory implementation if present

Acceptance:

- deterministic CRUD/query tests
- concurrency/race tests for relevant writers
- schema rollback/reopen tests

#### P1-S04 — Application service and read API

Allowed paths:

- `backend/internal/modules/engineering/**`
- `backend/api/engineering/**`
- generated `backend/rpc/**` only if generation ownership is established in backend; otherwise HTTP adapter only in the first slice
- `backend/internal/bootstrap/**` for composition

Deliverables:

- create/get/list/update EngineeringEntity
- create/get/list SourceBinding
- create/list typed ThreadEdge
- create/get/list Change
- get frozen ContextPack manifest
- canonical workspace authorization at application boundary

Acceptance:

- authorization matrix tests
- workspace isolation tests
- malformed request tests
- backend package tests/vet

#### P1-S05 — Work-plane linking

Allowed paths:

- `backend/internal/modules/workspace/**`
- `backend/internal/modules/engineering/**`
- tests/docs required by the integration

Deliverables:

- Project/Requirement/Task may reference EngineeringEntity IDs through explicit contracts
- `Project changes EngineeringEntity`, `Requirement/Task affects EngineeringEntity` typed links
- no cross-context table access
- no ownership transfer from Workspace to Engineering

Acceptance:

- workspace/engineering boundary tests
- deletion/archival behavior defined without cascades
- no change to Todo/Run acceptance semantics

Phase 1 exit criteria:

- a workspace can register one service/repository engineering entity, bind it to a source, link a Project/Requirement/Task to it, record an accepted Change, and persist/retrieve a frozen ContextPack manifest
- all relationships have provenance
- no manual IoT `System` object is introduced into the existing Agent System context
- Six-Domain verification and backend deterministic checks remain green

### Phase 2 — GitHub-backed Engineering Thread and Context Compiler

Goal: replace manual catalog maintenance with source-backed projections and compile reproducible task context.

Major steps:

- repository-owned `engineering.yaml` schema and validation
- GitHub repository/commit/PR source adapter
- reconciliation into SourceBinding + authoritative/observed ThreadEdges
- Change creation from accepted PR/work evidence
- Scope Resolver and authoritative graph traversal
- Context policy/filter/ranking
- immutable ContextPack generation and Runtime binding
- MCP tools: engineering entity lookup, thread traversal, context-pack get/compile (read/propose boundaries only)

Phase 2 exit: Task revision -> target services/repos -> governed knowledge -> frozen ContextPack -> Run is reproducible by ID/revision/checksum.

### Phase 3 — Evidence, Fact, Release, Runtime feedback

Goal: establish requirement-to-runtime traceability and knowledge suggestions.

Major steps:

- normalized Evidence envelope beyond current Project/Issue/Task lifecycle events
- Fact projection with provenance and rebuildability
- build/test/release/deployment source adapters
- `Change -> PR -> Build -> Release -> Deployment` thread
- observation/incident linking
- stale architecture/runbook detection from accepted Changes
- KnowledgeSuggestion generation; no autonomous publication of governed conclusions

Phase 3 exit: Requirement -> Task -> Run -> Change -> PR -> Release -> Deployment can be traced forward, and Incident/Observation can be traced backward to relevant changes and work intent.

### Phase 4 — Governance, learning, and organizational capability

Goal: turn organizational learning into standards, onboarding, and reusable skills.

Major steps:

- Standard entities with MUST/SHOULD/MAY and executable-check binding
- knowledge freshness/review-cycle enforcement
- Incident -> PostMortem -> Lesson -> Standard/Runbook update workflow
- Role/Skill capability matrix using canonical Member and Skill identities
- onboarding gates: Environment Ready, Domain Aware, System Aware, Code Ready, Delivery Ready, Independent
- learning paths reference production knowledge instead of copying training content
- ownership/bus-factor/stale-owner reports

### Phase 5 — Engineering Observer and autonomous assistance

Goal: automatically discover gaps while preserving source authority and human acceptance.

Major steps:

- Engineering Observer scans connected sources
- observations become proposed facts/edges, never silent authoritative writes
- missing owner/docs/runbook/standard/incident-lesson checks
- impact analysis across engineering thread
- knowledge-to-policy and knowledge-to-skill promotion suggestions
- agent task planning consumes ContextPack and reports Evidence back through governed flows

## 8. API and UI direction

Do not create a new top-level product domain for every engineering entity type. Keep Six-Domain work UX and add projections:

- Project Delivery Cockpit: objective, requirements, tasks, affected engineering entities, changes, releases, risk/knowledge debt
- Engineering Entity page: owner, source bindings, dependencies, active projects, recent changes, incidents, architecture/ADR/runbook, required skills
- Knowledge view: governed content plus affected entities and originating evidence/change
- Task view: why, affected entities, ContextPack, Runs, Changes, PRs/evidence, knowledge suggestions

Frontend work is deferred until backend identity and relation contracts stabilize.

## 9. MCP boundary

Existing knowledge MCP remains governed. Engineering MCP will initially expose read-oriented tools only:

- `engineering_entity_get/list/search`
- `engineering_thread_traverse`
- `engineering_change_get/list`
- `context_pack_get`
- later `context_pack_compile`

MCP must never approve Change acceptance, Knowledge publication, roles, permissions, or human DoneGate.

## 10. Security and governance

- Workspace authorization is revalidated server-side for every canonical object reference.
- External source locators are non-secret; credentials/tokens remain secret references and never enter entity metadata, evidence, context packs, logs, or knowledge.
- Source authority is explicit; inferred/proposed facts cannot overwrite authoritative facts.
- ContextPack contents are auditable and immutable after freeze.
- Automated agents cannot self-accept their Run, Change, or governed Knowledge.

## 11. Migration strategy

No big-bang rewrite.

1. Add Engineering Thread beside existing contexts.
2. Link existing Workspace objects by canonical IDs/contracts.
3. Introduce source-backed projections.
4. Bind ContextPack to Runtime only after deterministic compiler contracts exist.
5. Expand evidence producers without changing existing source transaction semantics.
6. Add UI projections after backend contracts stabilize.

Existing legacy `server/**` behavior may be inspected only as migration evidence; any required behavior is ported into `backend/**`.

## 12. Risks and mitigations

### Catalog staleness

Mitigation: source-backed bindings/reconciliation; manually entered facts are explicitly lower authority unless designated otherwise.

### Graph spaghetti

Mitigation: finite relation vocabulary, direction, provenance, authority, revision, and validation; no generic `related_to` as an authoritative relation.

### Boundary drift

Mitigation: Context Map update, package boundary tests, contracts between Workspace/Engineering/Runtime, no cross-context database access.

### AI contaminating truth

Mitigation: inferred/proposed authority class, KnowledgeCandidate review, independent DoneGate, immutable Evidence/ContextPack.

### Overbuilding a Jira/CMDB/LMS replacement

Mitigation: retain Six-Domain UX; federate GitHub/CI/observability; capability/onboarding is a projection over real engineering entities and knowledge.

### Context bloat

Mitigation: ContextPack manifest first; token-aware ranking/materialization only after authority and scope resolution are deterministic.

## 13. Rollback

Each phase is additive and independently removable.

- Phase 1 engineering module has no authority over existing six-domain state; disabling/removing it leaves Workspace/Auth/Space/System/Runtime behavior intact.
- Source adapters are projections and may be rebuilt from sources.
- ContextPack binding to Runtime is introduced only in Phase 2 and must be feature-gated until stable.
- Knowledge suggestions cannot corrupt published knowledge because publication remains separately governed.

## 14. Deterministic verification

At minimum after every product-code phase:

```bash
cd backend && go test ./... -count=1
cd backend && go vet ./...
cd backend && make check
cd backend && make test-race
pnpm typecheck
pnpm verify:six-domains
pnpm verify:no-runtime-agent-domains
git diff --name-only <base>..HEAD -- server/
```

Environment-blocked checks must be recorded explicitly; they must not be claimed as passed.

## 15. One-shot feasibility

The entire roadmap is **not** a safe one-shot change. Phase 1 is intentionally bounded and additive. Each dependent phase starts only after the preceding source/identity/authority contracts are verified.

## 16. Self-check

- Scope: bounded; first implementation stays inside a new backend engineering module.
- Dependencies: existing Workspace/Auth/Runtime contracts are referenced, not rewritten.
- Quality: domain invariants and provenance are test-first in Phase 1.
- Risks: staleness, graph ambiguity, AI authority, boundary drift, and context bloat are explicitly mitigated.
- Rollback: additive module and projections are removable without mutating six-domain truth.
