# Context Map

## Top-level bounded contexts

- [Workspace](./docs/contexts/workspace/CONTEXT.md) — owns
  workspace-scoped collaboration and delivery intent: Project, Todo, Issue,
  Knowledge, Requirement, Setting, and Relationship.
- [Auth](./docs/contexts/auth/CONTEXT.md) — owns human and Agent identity,
  workspace membership, roles, and authorization facts.
- [Space](./docs/contexts/space/CONTEXT.md) — owns workspace-isolated assets and
  their storage lifecycle.
- [System](./docs/contexts/system/CONTEXT.md) — owns immutable Agent releases,
  upgrade policy, and the global Skill catalog. **System does not mean an IoT
  product/system catalog.**
- [Execution/Runtime](./docs/contexts/execution/CONTEXT.md) — owns execution
  attempts: Run, queue state, lease, heartbeat, attempt, retry, cancellation,
  runtime logs, and execution Evidence references.
- **Engineering Thread** — owns canonical engineering identities and the
  provenance-bearing digital thread: EngineeringEntity, SourceBinding,
  ThreadEdge, accepted Change identity, and immutable ContextPack manifests.
  Engineering Thread does not own Git/GitHub, CI, deployment, observability, or
  Knowledge truth; it stores canonical mappings/projections with source and
  authority metadata.

## Context relationships

- **Workspace → Auth**: Workspace validates Member and Agent references through
  Auth-owned identities and membership; it does not copy or mutate identity
  truth.
- **Workspace → Space**: Workspace services reference stable Asset IDs while
  retaining the business meaning of each attachment or resource link.
- **Workspace → System**: Workspace references published Agent Release and Skill
  versions and owns tenant-level enablement, configuration, and bindings.
- **Workspace → Execution/Runtime**: Workspace submits an immutable execution
  request for a frozen work-item revision. Runtime reports attempt state and
  Evidence references; it cannot change Todo/Issue/Requirement truth or advance
  DoneGate acceptance.
- **Workspace → Engineering Thread**: Workspace-owned Project, Requirement,
  Issue, and Todo/Task may reference EngineeringEntity and Change IDs through
  application contracts. Workspace remains the authority for work intent;
  Engineering Thread remains the authority for canonical engineering-thread
  identity/provenance.
- **Engineering Thread → Auth**: optional human ownership references resolve to
  Auth-owned canonical Member identity. Engineering Thread does not create
  membership or authorization facts.
- **Engineering Thread → Execution/Runtime**: a work-item revision may be paired
  with an immutable ContextPack. Runtime pins the ContextPack reference for a
  Run and may produce Evidence/Change proposals, but cannot rewrite the frozen
  context or self-accept a Change.
- **Engineering Thread → Knowledge**: engineering entities and Changes may be
  described or constrained by Architecture, ADR, Standard, Runbook,
  Troubleshooting, Incident Review, and Lesson knowledge. Automated discoveries
  are proposals/evidence until Knowledge governance publishes them.
- **Engineering Thread → external sources**: SourceBinding and ThreadEdge
  provenance may reference Git/GitHub, API schemas, CI, release, deployment,
  and observability sources. Those external systems remain authorities for
  their own source facts.
- **Execution/Runtime → Auth**: Runtime validates the claiming Agent/Runner
  identity, workspace authorization, project authorization, and capability
  grants. A lease never creates membership or a new identity.
- **Execution/Runtime → System**: a Run pins an immutable Agent Release and Skill
  versions. Runtime may resolve them but cannot publish, upgrade, or mutate
  release state.
- **Execution/Runtime → Space**: Runtime stores logs and artifacts through
  Space-owned Asset IDs; Runtime owns their execution meaning, while Space owns
  content lifecycle and access.
- **System → Auth**: System releases target Auth-owned Agent identities without
  owning membership or authorization.
- **System → Space**: System references Space-owned Asset IDs for release
  artifacts and Skill content without owning storage lifecycle.
- **Workspace internal collaboration**: Project, Todo, Issue, Knowledge,
  Requirement, Setting, and Relationship collaborate through application
  contracts rather than transport loopback.

## Engineering digital-thread distinctions

| Concern | Owner | Meaning |
| --- | --- | --- |
| Project / Requirement | Workspace | Why and what the team intends to change |
| Todo / Task | Workspace | Desired work or team commitment |
| EngineeringEntity | Engineering Thread | Durable engineering identity such as service/repository/API/thing model |
| Run | Execution/Runtime | One bounded attempt to execute a frozen work-item revision |
| Change | Engineering Thread | Accepted engineering mutation connecting work to affected entities/artifacts |
| Execution Evidence | Execution/Runtime | Immutable references produced by an execution attempt |
| Knowledge | Workspace Knowledge governance | Reviewed organizational interpretation and reusable understanding |
| ContextPack | Engineering Thread | Immutable revisioned manifest of authoritative context selected for a work item/Run |

A Project changes EngineeringEntity state but never owns the long-lived
engineering identity. An IoT engineering system is represented as an
EngineeringEntity with type `engineering_system`; it must not be added to the
existing Agent-release System context.

## Todo and Run are different concepts

| Concern | Todo | Run |
| --- | --- | --- |
| Owner | Workspace | Execution/Runtime |
| Meaning | Desired work or team commitment | One bounded attempt to execute a frozen revision |
| Lifecycle | Product workflow and prioritization | Queue, claim, lease, heartbeat, attempt, retry, cancel, validation handoff |
| Multiplicity | One Todo may exist without automation | Zero or many Runs may execute one Todo revision |
| Assignee | Human or Agent reference | One authorized execution claimant for one attempt |
| Completion | Governed by Workspace checks and DoneGate | Ends with execution outcome and Evidence; never self-accepts |

A retry creates or advances a runtime attempt; it does not rewrite the Todo.
Canceling a Run does not automatically cancel its Todo. Completing a Run only
hands Evidence to independent validation.

## Agent ownership boundaries

| Concept | Owner | Stable identity | What it does not own |
| --- | --- | --- | --- |
| Agent Identity | Auth | Agent ID | Release bytes, runtime lease, task acceptance |
| Agent Release | System | Release ID + immutable version | Membership, active Run, execution result |
| Agent Execution | Execution/Runtime | Run ID + attempt | Identity truth, release publication, Todo intent, DoneGate |

An Agent Execution binds an Auth-owned Agent Identity, a System-owned immutable
Agent Release, and a frozen Workspace work-item revision. Runtime owns only the
attempt and its operational state.

## Acceptance boundary

Runner, Agent, model output, lease ownership, heartbeat, successful process
exit, uploaded Evidence, inferred ThreadEdges, and proposed Changes are never
authoritative acceptance. Only the Workspace/Delivery Kernel's independent
deterministic checks and authorized human DoneGate can advance governed
completion. Governed Knowledge publication remains a separate review decision.
