# Context Map

## Contexts

- [Workspace](./docs/contexts/workspace/CONTEXT.md) — owns workspace-scoped collaboration and delivery concepts.
- [Auth](./docs/contexts/auth/CONTEXT.md) — owns team-member and Agent identity, membership, and authorization concepts.
- [Space](./docs/contexts/space/CONTEXT.md) — owns workspace-isolated assets and their storage lifecycle.
- [System](./docs/contexts/system/CONTEXT.md) — owns versioned Agent releases and the global Skill catalog.

## Relationships

- **Workspace → Auth**: Workspace validates Member and Agent references through Auth-owned identities; it does not copy identity truth.
- **Workspace → Space**: Workspace services reference stable Asset IDs while retaining the business meaning of each attachment or resource link.
- **Workspace → System**: Workspace references published Skill versions and records tenant-level enablement, configuration, and Agent bindings.
- **System → Auth**: System releases target Auth-owned Agent identities without owning team membership or Agent authorization.
- **System → Space**: System references Space-owned Asset IDs for release artifacts and versioned Skill content without owning storage lifecycle.
- **Workspace internal collaboration**: Project, Todo, Issue, Knowledge, Requirement, Setting, and Relationship are services inside one Workspace context and collaborate through application contracts rather than transport loopback.

## Open Decision

- Agent execution lifecycle ownership is intentionally unresolved. It is not Todo, Agent identity, or Agent release state, and no module may claim it until the product owner confirms the boundary.
