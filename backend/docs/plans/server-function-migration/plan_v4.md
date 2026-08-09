# Incremental server function migration — Project lifecycle operations

- Plan-ID: `server-function-migration`
- Version: `4`
- Status: `approved`
- Approval source: user instruction dated 2026-08-03 to begin the next stage
  after the Workspace parity audit
- Base commit: `0c4f848a8458e256dd4fe2ed51498af969aa3c59`
- Repository: `/Users/fworld/Hvritual/goclaw`
- Branch: `codex/multica-six-domain-baseline`
- Task type: `change`

## Goal

Extend the verified Project tracer slice from Create/Get to an
operation-complete lifecycle over the current v1 Project model: List, Search,
Update, and Delete with Workspace authorization, deterministic query behavior,
and atomic dependent cleanup. Prepare, but do not implement, the subsequent
Workspace tenant-lifecycle slice until its Auth-owned creator/owner consistency
contract is explicit.

## Completed prerequisites

- P1-S1: Workspace identity and explicit SQLite provider seam.
- P2-S1: Project Create/Get and Relationship Put/List/Delete.
- P3-S1: Todo, Issue-status, Knowledge, Requirement-version, and Setting chain.
- 2026-08-03 parity audit: confirmed Project lifecycle operations and runtime
  integration remain incomplete.

## Ordered steps

### P4-S1 — active Project lifecycle tracer slice

1. Extend `workspace.v1.ProjectService` with List, Search, Update, and Delete
   contracts while retaining all existing field numbers and RPCs.
2. Reconcile ProjectService through dddgen, run the existing exact-prefix
   postprocessor, regenerate Buf bindings under `rpc/pb`, and inspect generated
   ownership changes before editing user-owned implementations.
3. Add Project domain update behavior and application orchestration.
4. Extend the SQLite Project repository with deterministic list/search,
   optimistic Workspace-scoped update, and one native delete transaction.
5. Verify local and bufconn gRPC behavior, tenant isolation, authorization
   ordering, search ranking/pagination, and dependent cleanup.

### P4-S2 — pending Workspace tenant-lifecycle contract gate

Freeze Create/List/Get/Update/Delete/Leave behavior only after Auth exposes a
durable contract for creator identity, owner-member creation, membership
listing, and owner transfer/last-owner constraints. P4-S2 permits discovery and
planning only until that dependency is approved; it permits no product code.

## Frozen Project behavior

- `ListProjects` requires Workspace authorization, accepts an optional valid
  lowercase status filter, and orders newest `created_at` first with ID as the
  stable tie-breaker.
- `SearchProjects` requires a non-empty trimmed query. It searches Project name
  and description case-insensitively, supports multi-word AND matching, excludes
  `completed` and `cancelled` unless `include_closed` is true, defaults limit to
  20, caps it at 50, accepts non-negative offset, and reports the total before
  pagination.
- Search ranking follows installed-server intent: exact name, name prefix, name
  phrase, all query words in name, then description; equal ranks order by
  `updated_at` descending and ID ascending. Each hit reports `name` or
  `description` as its match source and may include a bounded description
  snippet.
- `UpdateProject` accepts optional name, description, and status fields. A
  supplied name is trimmed and required, description may be cleared, status
  must use the accepted lowercase matrix, and omitted fields remain unchanged.
  Current Asset references remain unchanged in this slice.
- `DeleteProject` is Workspace scoped and idempotency is not implied: a missing
  or foreign Project returns the stable Project-not-found error.
- Project deletion commits these Workspace-owned changes atomically: delete all
  ProjectActorRelations, clear Todo and Issue `project_id` references, delete
  Requirement versions and Requirement aggregates owned by the Project, then
  delete the Project. This extends the declared Relationship cleanup invariant
  and mirrors the installed server's explicit dependent cleanup policy without
  foreign keys or cascades.
- Authorization precedes repository reads or writes that could reveal tenant
  state. Permission codes are
  `workspace.project.list`, `workspace.project.search`,
  `workspace.project.update`, and `workspace.project.delete`.

## Scope

- `backend/api/workspace/v1/project.proto` and generator-derived Project
  boundaries/state/tests.
- User-owned Project domain/application/provider/composition files under
  `backend/internal/modules/workspace`.
- Workspace Project application, domain, SQLite local-contract, and bufconn
  gRPC tests.
- Workspace architecture documentation and append-only migration journal.

## Non-goals

- No changes outside `backend/` and no writes to `server/`.
- No default runtime cutover, compatibility HTTP routes, OpenAPI, access
  manifest, PostgreSQL/sqlc provider, data migration, or production access.
- No Project priority, icon, calendar dates, issue/resource statistics,
  realtime events, Knowledge evidence, or Asset-association mutation. Those
  require later contract/side-effect slices; P4-S1 is operation-complete only
  for the already accepted v1 Project fields.
- No Workspace tenant lifecycle implementation in P4-S1 and no permissive fake
  Auth behavior in production composition.
- No changes to Todo, Issue, Knowledge, Requirement, Setting, Relationship
  public operations except the internal cleanup performed atomically by Project
  deletion.

## Invariants

- Workspace remains the authorization and tenant boundary.
- Every Project read/write/delete predicate includes Workspace ID.
- Project deletion performs explicit application-owned cleanup in one SQLite
  transaction; no foreign key or cascade is introduced.
- Domain imports only the standard library. Application depends only on domain
  and public contracts/ports. SQLite owns SQL, mapping, and the native
  transaction.
- Proto is the public source of truth. Generated files are changed only through
  the established dddgen/postprocess/Buf pipeline.
- Existing Create/Get behavior and default generated-stub runtime remain
  compatible.

## Dependencies

- Accepted Project aggregate/use case/repository and explicit SQLite
  composition.
- Accepted Relationship, Todo, Issue, and Requirement provider tables for
  Project-delete cleanup.
- Installed dddgen module
  `github.com/fworld/go-ddd-scaffold` revision
  `7266939f0e295648593739c694ab4e614b141546`, existing import postprocessor,
  Buf configuration, and local protoc plugins.
- Existing `WorkspaceAccessAuthorizer`; role interpretation remains outside the
  Project use case and is enforced by permission code.

## Acceptance criteria

1. ProjectService exposes Create/Get/List/Search/Update/Delete through matching
   local and gRPC contracts without changing existing field numbers.
2. List filtering/order and Search validation/ranking/pagination/closed filtering
   are deterministic and Workspace scoped.
3. Update preserves omitted values and rejects blank names or invalid statuses
   before persistence.
4. Delete rejects foreign/missing Projects and atomically performs every frozen
   dependent cleanup before removing the Project.
5. Authorization-denied calls do not access Project persistence.
6. Existing Todo/Issue/Knowledge/Requirement/Setting and Project/Relationship
   integration tests remain green.
7. The default bootstrap still selects generated stubs and the running process
   behavior remains unchanged.
8. Generated output is idempotent, `rpc/pb` remains under the approved package
   prefix, and all deterministic gates pass.

## Deterministic verification

- Buf format/lint before and after generation.
- dddgen ProjectService reconciliation, exact-prefix postprocessing, Buf
  generation, gofmt, and `go mod tidy`; repeat generation and compare content.
- Domain/application narrow tests and SQLite local/gRPC integration tests.
- `go test -race ./internal/modules/workspace/... -count=1`.
- `go test ./tests/contract/... -count=1`, `go test ./... -count=1`,
  `go vet ./...`, and `go mod verify`.
- Forbidden inward/cross-module import searches, generated inventory/checksum,
  policy hashes, `git diff --check`, repository-scope inspection, and live
  health/readiness checks.

## Risks

- ProjectService remains intentionally narrower in fields and side effects than
  the installed Chi Project API; operation completeness is not full HTTP parity.
- Delete touches multiple existing provider tables. Missing one cleanup target
  would leave orphaned Workspace references; integration tests must seed and
  assert every target.
- Reconciliation uses a locally installed generator build marked dirty. Its
  module revision is frozen above, output must be inspected, and a second run
  must be content-idempotent before acceptance.
- P4-S2 cannot safely begin until Auth ownership and cross-module consistency
  are explicit.

## Rollback

Restore `project.proto`, rerun the same generation pipeline, remove only P4
Project domain/application/provider/test additions, and restore the v3 plan
pointer. No migration rollback or production data action is required because
P4-S1 adds no tables and performs no runtime cutover.
