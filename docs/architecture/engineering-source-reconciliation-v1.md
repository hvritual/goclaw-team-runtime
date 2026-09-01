# Source-backed Engineering Reconciliation v1

P2-S03 converts one GitHub repository snapshot at one immutable commit into a canonical Engineering projection.

## Reconciliation transaction

The unit of reconciliation is:

`workspace + github repository + immutable commit SHA`.

A successful apply atomically writes the primary EngineeringEntity, its stable GitHub SourceBinding, all currently resolvable manifest-derived ThreadEdges, and removal of stale edges previously derived from that same repository manifest. Partial source revisions are never intentionally exposed.

## Canonical identity ownership

The SourceBinding ID is deterministic from the canonical repository locator. One repository therefore cannot silently change its primary EngineeringEntity ID.

If an entity already exists without that SourceBinding, reconciliation may claim it only when type/name/status/owner exactly match the manifest. A differing manually managed canonical entity is preserved and reconciliation returns a conflict.

After the stable SourceBinding exists, the repository manifest owns mutable name/status/owner projection for that entity. Entity type remains immutable and type drift is a conflict.

## Relationship projection

- `dependencies` -> authoritative `depends_on` edges.
- generic manifest relations -> their allowed durable relation types.
- `interfaces.direction=provides|uses` -> authoritative `provides|uses` edges.

Every manifest-derived edge has stable identity and provenance:

- `source_type = github_manifest`
- `source_locator = github://owner/repository/engineering.yaml`
- `source_revision = immutable commit SHA`

Removed manifest declarations delete only edges carrying that same source locator. Workspace/manual/other-source edges are not cascaded or overwritten.

## Unresolved references

Manifest references do not auto-create target entities. Missing, archived, or interface-type-mismatched targets are returned as explicit unresolved references. Other resolvable projections still apply. A later reconciliation can fill the edge when the target becomes valid.

This permits repositories to be onboarded in arbitrary order without manufacturing ghost Services/APIs/ThingModels.

## Manifest metadata not yet projected

V1 domain tags and knowledge file references remain pinned source facts inside `engineering.yaml`; they are not turned into canonical graph entities by P2-S03. Later Scope Resolver/Context Compiler slices may consume them from the pinned manifest revision. This avoids inventing a second metadata store before context selection semantics are stable.

## Non-goals

P2-S03 does not:

- accept Changes;
- infer PR/work relationships;
- publish Knowledge;
- mutate Workspace work objects;
- bind ContextPack to Runtime;
- create unresolved target EngineeringEntities automatically.
