# Engineering Scope Resolver v1

P2-S05 resolves the bounded engineering scope for one Workspace Project, Requirement, or Task from the authoritative Engineering Thread.

## Seed rule

A scope starts only from an explicit authoritative Workspace work-link:

- Project -> `changes` -> EngineeringEntity
- Requirement -> `affects` -> EngineeringEntity
- Task -> `affects` -> EngineeringEntity

The edge must carry `source_type=workspace`. Proposed/observed/inferred work links do not establish executable scope.

## Traversal

Traversal is deterministic breadth-first search over EngineeringEntity-to-EngineeringEntity edges only.

Outbound relations followed in v1:

- `part_of`
- `depends_on`
- `implements`
- `provides`
- `uses`
- `constrains`
- `governs`
- `operates`

Selected inbound traversal is allowed for `part_of`, `implements`, and `provides`. This allows a system/API seed to discover contained services or concrete implementers without recursively pulling every reverse dependency into scope.

Dynamic work/release/runtime relations (`changes`, `affects`, `introduced_by`, `included_in`, `deployed_to`) are not traversal relations after the seed step.

Only `authoritative` edges expand scope. Inferred/observed/proposed edges remain visible to other analysis surfaces but cannot silently enlarge an execution context.

## Bounds

Defaults:

- max depth: 2
- max EngineeringEntities: 64

Hard limits:

- max depth: 4
- max EngineeringEntities: 256

When the entity limit is reached, the result is explicitly `truncated` and carries an `entity_limit` warning rather than silently dropping context.

## Source projection

For every scoped entity the resolver returns SourceBindings including source type, locator, revision, authority, and observed time. A policy may define `SourceStaleAfter` to flag old source observations.

Authoritative or observed sources without a revision are flagged `unpinned_source`; this is important because the Context Compiler must prefer revision-addressable evidence.

## Warnings

V1 emits deterministic warnings for:

- dangling Workspace work-link targets;
- dangling Engineering graph edges;
- archived entities retained in historical scope;
- stale SourceBindings;
- unpinned authoritative/observed sources;
- entity-limit truncation.

A dangling graph edge is skipped; it does not cause a ghost EngineeringEntity to be created.

## Non-goals

P2-S05 does not use vector search, embeddings, semantic similarity, or an LLM. It does not fetch document bodies and does not create a ContextPack. P2-S06 consumes this structured scope and performs governed context selection/materialization.
