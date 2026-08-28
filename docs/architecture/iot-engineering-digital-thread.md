# IoT Engineering Digital Thread Control Plane

## Purpose

Multica evolves into an IoT engineering project, execution, and knowledge-compounding control plane by connecting durable engineering reality to existing work intent and controlled execution.

The central design object is not a Wiki folder or a Project container. It is the **Engineering Digital Thread**:

`Intent -> Work -> Engineering Entity -> Run -> Change -> Artifact -> Evidence -> Fact -> Knowledge -> Policy/Skill -> next Context`.

## Five planes

1. **Intent / Work Plane** — Workspace, Member, Project, Requirement, Issue, Task, Skill.
2. **Engineering Thread Plane** — EngineeringEntity, SourceBinding, ThreadEdge, Change, ContextPack.
3. **Execution Plane** — existing Run/Attempt/Lease/Heartbeat/Retry/Evidence runtime.
4. **Evidence & Knowledge Plane** — immutable evidence, rebuildable facts, governed knowledge and policy/skill promotion.
5. **Context Plane** — scope resolution, authority filtering, graph traversal, knowledge/change/incident retrieval, ranking, frozen ContextPack.

## Hard distinctions

- Project is a temporary change container; EngineeringEntity is durable engineering reality.
- Existing `System` context is Agent Release/Skill publication, not an IoT system catalog.
- Task is desired work; Run is an execution attempt; Change is an accepted engineering mutation.
- Evidence is source material; Fact is a provenance-backed claim; Knowledge is governed interpretation.
- A successful model/Runner/Run cannot self-accept a Task, Change, or Knowledge entry.

## Federated truth

Engineering Thread does not duplicate every external fact. It maintains canonical identity, source bindings, provenance-bearing projections, and typed edges. Git/GitHub, CI, deployment, observability, Workspace, Runtime, Auth, Space, System, and governed Knowledge remain authorities for their own facts.

## Relation policy

Edges are directed and typed. Every edge carries provenance and authority (`authoritative`, `observed`, `inferred`, `proposed`). Inferred/proposed relations are useful for exploration but may not silently override authoritative graph state.

## Context compilation

A ContextPack is a frozen manifest for a work-item revision/Run. It identifies target engineering entities and exact revisions/checksums of relevant requirements, architecture, ADRs, standards, runbooks, recent changes, incidents, and other evidence. Runtime pins the ContextPack rather than relying on transient chat history.

## Knowledge compounding

The desired loop is:

`Run -> Evidence -> Fact -> KnowledgeSuggestion -> Review -> Knowledge -> Standard/Skill -> next ContextPack`.

The highest-value knowledge may become executable checks; training references production knowledge instead of copying it into separate stale material.
