# Workspace module

- Public seam: internal/modules/workspace/contract
- Implementation: internal/modules/workspace/internal
- Proto contract: api/workspace/v1/workspace.proto
- SQLite provider: internal/modules/workspace/internal/infrastructure/sqlite
- Cross-module projection: `WorkspaceIdentityReader` exposes only Workspace ID
  and display name; consumers never read the Workspace table directly.

After generation, run make generate, register the module in internal/bootstrap, and declare any cross-module contract edges in the architecture documentation.
