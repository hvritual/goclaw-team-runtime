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
  upgrade policy, and the global Skill catalog.
- [Execution/Runtime](./docs/contexts/execution/CONTEXT.md) — owns execution
  attempts: Run, queue state, lease, heartbeat, attempt, retry, cancellation,
  runtime logs, and execution Evidence references.

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
- **Execution/Runtime → Auth**: Runtime validates the claiming Agent/Runner
  identity, workspace authorization, project authorization, and capability
  grants. A lease never creates membership or a new identity.
- **Execution/Runtime → System**: A Run pins an immutable Agent Release and Skill
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
exit, and uploaded Evidence are never authoritative acceptance. Only the
Workspace/Delivery Kernel's independent deterministic checks and authorized
human DoneGate can advance governed completion.
