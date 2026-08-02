# Auth module

- Public seam: internal/modules/auth/contract
- Implementation: internal/modules/auth/internal
- Proto contract: api/auth/v1/auth.proto
- Intentional dependency: Workspace public `WorkspaceIdentityReader`, bound in
  bootstrap for invitation response projection; no Workspace adapter or table
  is imported by Auth.

After generation, run make generate, register the module in internal/bootstrap, and declare any cross-module contract edges in the architecture documentation.
