# IoT Engineering Digital Thread — Implementation Plan v3

- Plan-ID: `IOT-ENGINEERING-DIGITAL-THREAD-001`
- Version: `3`
- Status: **Approved / executing**
- Parent plans: `plan_v1.md`, `plan_v2.md`
- Canonical branch: `codex/multica-six-domain-baseline`
- Activation: explicit user instruction to start P1 wrap-up and Phase 2

## 1. Purpose

Version 3 closes Phase 1 with an end-to-end certification and then executes Phase 2 as a source-backed engineering knowledge and context-compilation program.

No Phase 1 invariant is weakened: `server/**` remains permanently read-only; Workspace owns work intent; Engineering owns digital-thread identity/provenance; Runtime owns execution attempts; governed Knowledge owns organizational conclusions; existing System continues to mean Agent Release / Skill publication.

## 2. P1-EXIT — Phase 1 end-to-end certification

### Objective

Prove one canonical scenario can traverse the complete Phase 1 chain rather than merely validating the components independently:

`EngineeringEntity -> SourceBinding -> Project/Requirement/Task link -> Change -> accepted Change -> frozen ContextPack -> readback`.

### Allowed paths

- `backend/internal/modules/engineering/**`
- `backend/internal/bootstrap/**` for canonical integration tests/composition only
- `backend/docs/plans/iot-engineering-digital-thread/**`

`server/**`, Runtime semantics, Workspace Todo DoneGate and existing System ownership remain unchanged.

### Required product gaps to close

- expose a governed application operation to accept a proposed Change;
- expose a governed application operation to freeze/persist an immutable ContextPack manifest;
- keep both operations owner/admin-write, workspace-scoped, and transport-independent;
- ContextPack freeze validates all target EngineeringEntity identities and all revision/checksum-bearing context references before persistence.

### E2E acceptance scenario

A canonical SQLite runtime test must:

1. create an EngineeringEntity;
2. create a source-backed SourceBinding;
3. link Project, Requirement and Task to the entity using P1-S05 routes;
4. create a Change referencing the task/entity/artifact;
5. accept the Change explicitly;
6. freeze a ContextPack for the exact Task revision and target entity with a Change reference carrying revision/checksum metadata;
7. read the ContextPack through the public Engineering API and prove deterministic checksum/immutable readback;
8. prove Task status/revision and Runtime state are not changed by Change acceptance or ContextPack freeze.

### Exit gate

- Go 1.26.1 `make check` succeeds;
- canonical `make test-race` succeeds;
- frontend aggregate and final `required` succeed;
- no `server/**` diff;
- Phase 1 is recorded as closed only after the code head is green.

## 3. Phase 2 objective

Move Engineering Thread from manually maintained records to source-backed projections and reproducible task context:

`Repository source -> Manifest/GitHub adapter -> Reconciler -> SourceBinding/ThreadEdge -> Scope Resolver -> Context Compiler -> frozen ContextPack -> MCP/Runtime`.

Federated truth remains mandatory: GitHub owns repository/commit/PR facts; Engineering stores canonical identity, sourced projections, reconciliation state, provenance and context manifests.

## 4. Phase 2 execution sequence

### P2-S01 — Repository manifest contract

Goal: let a repository self-describe its canonical engineering identity without creating a second CMDB.

Deliverables:

- versioned `engineering.yaml` schema/model;
- strict Go parser/validator;
- canonical entity type/status/relation validation reusing Engineering ontology;
- owner/domain/source/interface/dependency/knowledge reference fields;
- deterministic normalized representation/checksum;
- fixtures and negative tests for unknown fields, duplicate IDs, unsupported relation/entity types and malformed source locators.

Allowed paths:

- `backend/internal/modules/engineering/**`
- `docs/architecture/**` and plan journal only when required.

Non-goals: GitHub network calls, reconciliation writes, UI.

### P2-S02 — GitHub source adapter

Goal: read authoritative repository/commit/PR identity through a provider contract.

Deliverables:

- provider-neutral source interfaces;
- GitHub adapter for repository metadata, file-at-revision (`engineering.yaml`), commit and PR metadata required by the digital thread;
- no credentials in Engineering entities, logs, ContextPacks or knowledge;
- fixture/fake-provider contract tests and GitHub-specific mapping tests.

### P2-S03 — Reconciliation engine

Goal: turn manifest/source observations into idempotent sourced projections.

Deliverables:

- observation/reconcile command;
- deterministic SourceBinding identities;
- authoritative/observed ThreadEdges with source revision and observed time;
- stale projection detection without deleting history;
- proposed/inferred observations never overwrite authoritative source facts.

### P2-S04 — Change projection from source evidence

Goal: connect accepted work to actual PR/commit artifacts without equating Run success with Change acceptance.

Deliverables:

- work/PR/commit matching contracts;
- proposed Change generation from source evidence;
- explicit acceptance remains independent;
- artifact refs pin source revision/locator.

### P2-S05 — Scope Resolver

Goal: deterministically resolve a work-item revision to target EngineeringEntities and relevant graph neighborhood.

Deliverables:

- work-link entry-point resolution;
- bounded authoritative graph traversal;
- dependency/implements/part-of expansion policy;
- provenance and conflict/stale warnings;
- deterministic ordered scope result.

### P2-S06 — Context Compiler

Goal: create reproducible ContextPacks from resolved scope.

Deliverables:

- policy versioning;
- source authority/freshness ranking;
- governed knowledge/reference resolution;
- recent Change/Incident hooks;
- token-budget-aware selection boundary;
- deterministic immutable ContextPack checksum.

### P2-S07 — Runtime binding

Goal: bind a frozen work-item revision and frozen ContextPack to one Run without moving DoneGate ownership.

Deliverables:

- explicit Runtime consumer contract;
- ContextPack ID/checksum pinned at Run creation/preparation;
- no mutable context after Run starts;
- replay/audit can identify exact work revision, context checksum, Agent release and Skill version.

### P2-S08 — Engineering MCP

Goal: expose read/propose Engineering capabilities to LLM/Agent clients.

Initial tools:

- `engineering_entity_get/list/search`
- `engineering_thread_traverse`
- `engineering_change_get/list`
- `context_pack_get`
- `context_pack_compile` only after P2-S06 is accepted.

MCP cannot accept Change, publish governed Knowledge, change roles/permissions, or satisfy human DoneGate.

## 5. Phase 2 exit criteria

A single task revision must be traceable and reproducible as:

`Task revision -> affected service/repository -> source revision -> governed knowledge/recent changes -> ContextPack checksum -> Run`.

For identical work revision, source revisions and context policy, compilation must produce the same canonical ContextPack content checksum.

## 6. Ordering and gates

Execution order is strict:

`P1-EXIT -> P2-S01 -> P2-S02 -> P2-S03 -> P2-S04 -> P2-S05 -> P2-S06 -> P2-S07/P2-S08`.

P2-S07 and P2-S08 may proceed in either order after P2-S06, but Runtime binding must not precede deterministic Context Compiler acceptance.

Each step must append evidence to `journal.md`; no step may claim acceptance before canonical CI is green.