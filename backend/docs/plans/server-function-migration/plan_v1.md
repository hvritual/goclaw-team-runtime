# Incremental server function migration — Workspace first

- Plan-ID: `server-function-migration`
- Version: `1`
- Status: `approved`
- Approval source: user instruction dated 2026-08-03 to absorb `server`
  backend behavior and test examples module by module, prioritizing Workspace
- Base commit: `0c4f848a8458e256dd4fe2ed51498af969aa3c59`
- Repository: `/Users/fworld/Hvritual/goclaw`
- Branch: `codex/multica-six-domain-baseline`
- Task type: `change`

## Goal

Begin the incremental migration from the installed `server` into the standalone
`backend` with the smallest verified Workspace tracer slice: the Workspace-owned
identity projection and its SQLite-local persistence adapter. Establish a
repeatable module-by-module migration path without cutting traffic over from the
installed runtime.

## Workspace context brief

- Purpose: own Workspace identity and the tenant/isolation boundary used by all
  business modules.
- Ubiquitous language: Workspace, Workspace identity, tenant boundary.
- Owned state in this slice: Workspace ID and name projection.
- Invariant: identity lookup is scoped to the requested Workspace ID and missing
  identities use the stable Workspace not-found error.
- Query: find Workspace identity by ID.
- External dependency: provider-owned SQLite database connection.
- Downstream context: Auth consumes this projection before membership and
  invitation decisions in a later slice.
- Consistency boundary: one Workspace row.

## Scope

### P1-S1 — active Workspace identity tracer slice

- Port the existing Workspace identity contract, domain projection, application
  query service, SQLite repository adapter, provider migration, and module
  composition seam from `server` into `backend`.
- Adapt module paths to `github.com/hvritual/workspace` without copying stale
  generated files.
- Port and strengthen the existing SQLite persistence tests, including tenant
  isolation and missing-database validation.
- Preserve the existing default `workspace.New()` path and transport behavior.
- Update Workspace architecture documentation and append verification evidence.

### Subsequent ordered migration slices

1. Inventory and freeze Workspace Project behavior and compatibility tests.
2. Migrate Project and Relationship as one consistency-aware sequence.
3. Migrate Todo, Issue, Knowledge, Requirement, and Setting through separate
   verified tracer slices.
4. Begin Auth only after Workspace identity is available, followed by Space and
   System according to explicit plan revisions.

Only P1-S1 is authorized for product-code changes in this plan version.

## Non-goals

- No changes outside `backend/` and no edits to the installed `server/`.
- No installed HTTP API, Chi/sqlc runtime, PostgreSQL schema, SQLite-local
  runtime, database data, or routing cutover.
- No Workspace CRUD, Project, Relationship, Todo, Issue, Knowledge,
  Requirement, Setting, Auth, Space, or System method implementation yet.
- No Proto or generated binding changes in this slice.
- No dual write, compatibility shim, cross-module table access, or production
  database connection.

## Invariants

- Workspace remains the tenant and authorization boundary.
- All lookups are explicit by Workspace ID; no personal/global fallback exists.
- Domain and application packages do not depend on SQL, transport, generated
  Protobuf, Kratos, or another module's implementation.
- SQLite details remain in Workspace infrastructure and module composition.
- Existing Ping and HTTP/gRPC registration behavior remains unchanged.
- Existing user-owned work under `backend/` is preserved.

## Dependencies

- Completed four-module Proto foundation and runnable transport from plan v6.
- `database/sql` plus a pure-Go SQLite driver isolated to the backend Go module.
- The source behavior and tests in `server/internal/modules/workspace`.

## Acceptance criteria

1. Workspace exposes a stable local `WorkspaceIdentityReader` contract.
2. The application service maps repository not-found to
   `contract.ErrWorkspaceNotFound` and propagates other errors.
3. The SQLite adapter rejects a nil database and returns only the Workspace row
   matching the requested ID.
4. Workspace provider migration can initialize an isolated SQLite database.
5. Default non-persistent module construction and all existing transports/tests
   remain unchanged.
6. Narrow Workspace tests, `go test ./...`, `go vet ./...`, Buf lint, and scope
   checks pass.
7. No path outside `backend/` changes.

## Deterministic verification

- Run `gofmt` on changed Go files and `go mod tidy`.
- Run `go test ./internal/modules/workspace/... -count=1`.
- Run `go test ./... -count=1` and `go vet ./...`.
- Run `buf format --diff --exit-code` and `buf lint`.
- Inspect package imports for inward dependency violations.
- Verify Git status contains no newly changed path outside `backend/` and run
  `git diff --check`.

## Risks

- The source SQLite schema is only an isolated migration seam, not a production
  database migration; using it as production storage would be an unauthorized
  cutover.
- Identity is intentionally a narrow projection. Treating it as full Workspace
  CRUD would overstate the migrated behavior.
- Auth integration is deferred; the reader may be unused by the default runtime
  until the dependent Auth slice is explicitly planned.

## Rollback

Remove only the P1-S1 identity contract/domain/application/SQLite files and the
corresponding fields/methods on the Workspace module, then remove the SQLite
driver if no longer referenced. The default module and transport path remain the
safe baseline throughout.

