# Server function migration journal

## 2026-08-03 — P1-S1 started

- Task-Revision: `1`
- Plan-Version: `1`
- Plan-Step: `P1-S1`
- Authorization: user requested module-by-module absorption of `server`
  behavior and test examples, with Workspace first.
- Frozen source slice:
  `server/internal/modules/workspace` Workspace identity reader, SQLite adapter,
  provider migration, and persistence tests.
- Current path: caller -> concrete/legacy Workspace storage.
- Target path: local contract -> application query -> domain projection ->
  repository port <- SQLite dependency adapter.
- Next action: implement only the frozen Workspace identity tracer slice inside
  `backend/` and run deterministic verification.

## 2026-08-03 — P1-S1 implementation and verification

- Added the Workspace-owned `WorkspaceIdentityReader` public contract and stable
  `ErrWorkspaceNotFound` error.
- Added the Workspace aggregate projection and application query with an
  inward-facing repository port. Repository not-found is mapped to the public
  contract error; unrelated repository errors are propagated unchanged.
- Added the provider-owned SQLite adapter and embedded migration copied from the
  established `server` slice with module paths adapted to
  `github.com/hvritual/workspace`.
- Added explicit `NewWithSqlitePersistence`, `MigrateSqlite`, and
  `IdentityLocal` composition seams. The database-neutral `New()` path and all
  existing HTTP/gRPC registrations are unchanged.
- Ported and strengthened the source persistence tests with two Workspace rows,
  requested-ID isolation, missing-row behavior, nil-database validation, and
  persistent-module Ping compatibility.
- Added application table tests for successful projection, not-found mapping,
  and non-domain repository error propagation.
- Added `modernc.org/sqlite v1.55.0` to the standalone backend Go module; no
  database outside the test-created in-memory instance was opened or modified.
- Deterministic verification passed:
  - `go test -race ./internal/modules/workspace/... -count=1`
  - `go test ./... -count=1`
  - `go vet ./...`
  - `go mod verify` (`all modules verified`)
  - fixed temporary Buf tool
    `github.com/bufbuild/buf/cmd/buf@v1.72.0 format --diff --exit-code`
  - fixed temporary Buf tool
    `github.com/bufbuild/buf/cmd/buf@v1.72.0 lint`
- The ambient shell still has no installed `buf` executable; the fixed
  temporary tool was used without adding it to the project module.
- No Proto, generated binding, installed `server`, existing HTTP API, runtime
  route, PostgreSQL schema, or production data changed.
- P1-S1 deterministic acceptance is satisfied. Independent final acceptance
  remains pending under repository policy.

## 2026-08-03 — P2-S1 started

- Task-Revision: `2`
- Plan-Version: `2`
- Plan-Step: `P2-S1`
- Authorization: user instructed Codex to continue the Workspace-first
  migration after P1-S1.
- Frozen source behavior: installed Project status/default/name validation and
  Workspace-scoped reads, combined with the target ownership rules already
  declared in ProjectService and RelationshipService.
- Security decision: the new explicit constructor must require injected access
  authorization and Actor Workspace membership dependencies. The default
  runtime stays on generated not-implemented services until Auth adapters exist.
- Generated ownership decision: add user-owned use cases and composition files;
  do not edit dddgen-marked extension, adapter, contract, or pb files.
- Next action: implement domain/application tests and behavior, then SQLite and
  gRPC integration in dependency order.

## 2026-08-03 — P2-S1 implementation and verification

- Added public consumer-owned `WorkspaceAccessAuthorizer` and
  `WorkspaceActorReader` seams. Project/Relationship use cases cannot be
  composed without them.
- Added Project domain behavior for trimmed required names, `planned` default,
  the five installed-server lowercase statuses, stable non-empty Asset IDs, and
  UTC timestamps. Ported the server status compatibility example into a
  table-driven domain test.
- Added ProjectActorRelation domain behavior for Member `lead/member` and Agent
  `agent` role combinations plus independently validatable delete references.
- Added Project Create/Get and Relationship Put/List/Delete application use
  cases with authorization-before-storage, Workspace-scoped Project reads,
  same-Workspace Actor verification, deterministic relation ordering, and
  stale-Actor-safe idempotent delete.
- Added SQLite Project and Relationship repositories. SQL and JSON mapping stay
  in provider infrastructure; every query/write includes Workspace ID and no
  foreign key or cross-module table access was introduced.
- Added additive migrations `000002_project_relationship.up.sql` and `.down.sql`.
  The accepted `000001` files were not edited. `MigrateSqlite` now applies
  lexical up migrations in one native transaction and records versions in
  `workspace_schema_migrations`; repeat application is verified.
- Added `NewWithSqliteWorkspaceServices`, which requires authorization and Actor
  dependencies and replaces only the generated Project/Relationship extension
  instances. No dddgen-owned file was edited. `workspace.New()` remains on the
  explicit generated not-implemented implementations.
- Added application authorization/isolation tests, SQLite local-contract tests,
  migration tests, and bufconn gRPC Project/Relationship happy-path tests.
- Deterministic verification passed:
  - domain and application package tests
  - `go test -race ./internal/modules/workspace/... -count=1`
  - `go test ./tests/contract/... -count=1`
  - `go test ./... -count=1`
  - `go vet ./...`
  - `go mod verify` (`all modules verified`)
  - fixed temporary Buf v1.72.0 format and lint checks
- No Proto/generated input changed and no `rpc/pb`, dddgen-marked, `.dddgen`, or
  generated contract file was patched; the accepted P6 generated inventory and
  checksum therefore remain the generation baseline.
- No installed HTTP API, default runtime selection, `server` path, PostgreSQL
  schema, existing SQLite-local data, or production process was changed.
- P2-S1 deterministic acceptance is satisfied. Project update/delete/search,
  relation cleanup on Project delete, and real Auth adapters remain explicitly
  deferred. Independent final acceptance remains pending.

## 2026-08-03 — P3-S1 started

- Task-Revision: `3`
- Plan-Version: `3`
- Plan-Step: `P3-S1`
- Authorization: user explicitly requested the coordinated
  Todo -> Issue -> Knowledge -> Requirement -> Setting chain.
- Frozen compatibility rules were derived from installed task/issue state
  validation, Knowledge candidate-first governance, versioned requirements, and
  the existing target Proto ownership contracts.
- Cross-module decision: add only consumer-owned Asset and Skill reference
  readers; Auth Actor validation reuses the v2 public contract.
- Runtime decision: expose one explicit full-chain SQLite constructor requiring
  all dependencies; leave the default runtime and v2 constructor unchanged.
- Next action: implement the five sub-slices in declared dependency order and
  verify each narrow package before full gates.

## 2026-08-03 — P3-S1 implementation and verification

- Added Todo domain/application behavior for required trimmed titles, the
  installed ordinary-task status vocabulary, paired Member/Agent assignees,
  Workspace-scoped Project/Issue validation, and status-only updates.
- Added Issue status mutation with ID-or-identifier lookup scoped by Workspace.
  The seven installed lowercase statuses are table-tested and invalid values do
  not reach persistence.
- Added candidate-first Knowledge Create/Get. Workspace stores stable Asset IDs
  only and validates them through the new consumer-owned
  `WorkspaceAssetReader`; foreign Assets and cross-Workspace reads are rejected.
- Added Requirement aggregate/current-version behavior with required title and
  content, ordered Issue deduplication, draft state, covered/uncovered
  projection, same-Workspace/same-Project Issue checks, and atomic immutable
  SQLite version writes. Integration coverage verifies versions 1 and 2 remain
  stored while the aggregate points at version 2.
- Added Workspace JSON setting upserts and tenant-level Skill bindings. Skill
  and optional version references are checked through the new consumer-owned
  `SkillReferenceReader`; Agent references reuse the Auth-owned Actor seam and
  are deduplicated before persistence.
- Added the additive `000003_workspace_service_chain` provider migration for
  Workspace-owned Todo, Issue, Knowledge, Requirement/version, Setting, and
  Skill-binding tables. It adds no foreign keys and reads no foreign module
  tables. Migration ordering, repeatability, catalog count, and table inventory
  are verified.
- Added `NewWithSqliteWorkspaceChain`, which composes Project/Relationship plus
  the five-service chain only when authorization, Auth Actor, Space Asset, and
  System Skill capabilities are supplied. It replaces extension instances in a
  user-owned composition file; generated extensions remain untouched.
- Added domain compatibility examples, application authorization-before-storage
  tests, SQLite local-contract isolation/reference/versioning tests, persisted
  JSON checks, and bufconn gRPC happy paths for all five services.
- Deterministic verification passed:
  - `go test -race ./internal/modules/workspace/... -count=1`
  - `go test ./tests/contract/... -count=1`
  - `go test ./... -count=1`
  - `go vet ./...`
  - `go mod verify` (`all modules verified`)
  - fixed temporary Buf v1.72.0 format and STANDARD lint checks
  - forbidden inward/cross-module import searches returned no matches
  - all Go sources are gofmt-clean
- The generated `rpc/pb` inventory remains 37 files with latest modification
  time 2026-08-02 23:12:19 +0800; no Proto, dddgen input/output, generated
  contract, HTTP API, Chi/sqlc path, PostgreSQL schema, or installed `server`
  source was modified.
- The existing backend process remains PID 32680 on HTTP `127.0.0.1:8000` and
  gRPC `127.0.0.1:9000`; health/readiness probes pass. It intentionally remains
  on the default generated stubs because real Auth/Space/System adapters do not
  yet exist.
- P3-S1 deterministic acceptance is satisfied. Richer CRUD/search/governance,
  compatibility HTTP handlers, real cross-module adapters, and runtime cutover
  remain deferred. Independent final acceptance remains pending.

## 2026-08-03 — P3-S1 verification addendum

- Strengthened the new provider schema with Todo/Issue/Knowledge status checks,
  Todo assignee-pair checks, Requirement draft/coverage checks, and boolean
  Skill-enablement checks. Direct invalid Todo and Issue inserts are rejected by
  migration tests as well as by domain/application validation.
- Repeated the Workspace package tests, full Go tests, vet, formatting and
  generated-inventory checks after the constraint change; all passed. No file
  below `rpc/pb` has a modification time on or after 2026-08-03.

## 2026-08-03 — P4-S1 started

- Task-Revision: `4`
- Plan-Version: `4`
- Plan-Step: `P4-S1`
- Authorization: user instructed Codex to begin the next stage after the
  Workspace parity audit.
- Policy hashes revalidated:
  - root `AGENTS.md`:
    `3893c1ff20196c5ee3ee0ea4a0e5ba5d7430cdca218c56278816fc0aed83e933`
  - `backend/AGENTS.md`:
    `7d81baac25cc791371509f5a2531ec3d15e2e2ea0d103f405e5d5b2a5f094a87`
  - root `CLAUDE.md`:
    `20105929c7aa15afaf6664112b5e360fc72aa23eba3ad13dc68bf805738ef5db`
- Scope decision: Project lifecycle operations are the active independently
  verifiable slice. Workspace tenant-lifecycle code is gated because creating a
  tenant must coordinate an Auth-owned owner member and Auth does not yet expose
  that durable contract.
- Generator preflight: installed dddgen reports scaffold revision
  `7266939f0e295648593739c694ab4e614b141546` and is marked dirty. P4 freezes that
  exact revision and requires inspected, repeated, content-idempotent output.
- Current path: generated Project extension -> not-implemented default service;
  opt-in SQLite composition -> Create/Get use case -> Project repository.
- Target path: generated Project extension -> explicit lifecycle use case ->
  Project domain/port <- SQLite adapter with atomic dependent cleanup.
- Next action: extend only ProjectService Proto, run the established generation
  pipeline, inspect generated changes, then implement user-owned behavior.

## 2026-08-03 — P4-S1 implementation and verification

- Extended `workspace.v1.ProjectService` additively with List, Search, Update,
  and Delete. Existing RPCs and message field numbers remain unchanged; Go
  bindings remain rooted at `github.com/hvritual/workspace/rpc/pb`.
- Reconciled ProjectService with the frozen dddgen revision, applied the exact
  import postprocessor, and regenerated Buf bindings. A repeated full generated
  tree digest remained
  `c95d82069f1757a781d100cc8e174c14687d6759a2529336e806a28e9528db65`,
  proving content-idempotence. The complete `rpc/pb` inventory remains 37 Go
  files with digest
  `af10228d10352f9322ef84a890aa8731566d4e6934d126aa5aae2d016a4380d9`.
- Added Project optional update behavior without allowing Asset association
  mutation. Blank supplied names and unsupported status strings are rejected;
  omitted values are preserved and timestamps remain UTC.
- Added authorization-first application orchestration and Workspace-scoped
  repository ports for deterministic status-filtered List, ranked Search,
  Update, and Delete.
- SQLite Search applies case-insensitive multi-word matching, frozen ranking,
  closed-state filtering, stable updated-time/ID ties, total-before-pagination,
  and bounded Unicode-safe snippets.
- Project deletion now owns one native SQLite transaction that removes
  ProjectActorRelations, clears Todo/Issue Project references, removes Project
  Requirement versions and aggregates, and deletes the Project. An injected
  SQLite abort trigger verifies that every preceding cleanup rolls back on a
  final-delete failure.
- Added domain/application tests, SQLite local-contract lifecycle/search and
  tenant-isolation tests, and bufconn gRPC round trips for all six Project RPCs.
  Authorization-denied operations are verified not to access persistence.
- Workspace lifecycle gate evidence: both installed PostgreSQL and SQLite-local
  CreateWorkspace implementations create the Workspace row and Auth-owned
  initial `owner` member in one transaction. Current `auth.v1.MemberService`
  exposes only ListMembers and UpdateMemberRole, so no durable initial-owner
  command or cross-module rollback contract exists. P4-S2 product code remains
  blocked by design; no incomplete ownerless Workspace path was introduced.
- Deterministic verification passed:
  - Buf format and STANDARD lint
  - repeated dddgen/postprocess/Buf generation with identical content digest
  - `go test -race ./internal/modules/workspace/... -count=1`
  - `go test ./tests/contract/... -count=1`
  - `go test ./... -count=1`
  - `go vet ./...`
  - `go mod verify` (`all modules verified`)
  - forbidden inward/cross-module imports, gofmt, policy hashes, and
    `git diff --check`
- The running backend remains PID 32680 on HTTP `127.0.0.1:8000` and gRPC
  `127.0.0.1:9000`; `/healthz`, `/readyz`, gRPC health, and Workspace Ping pass.
  Default bootstrap behavior remains the generated stub composition, so this
  opt-in SQLite lifecycle work does not alter runtime behavior.
- P4-S1 deterministic acceptance is satisfied. P4-S2 is a contract gate rather
  than an authorized implementation step; the next safe migration slice is the
  Auth-owned initial Workspace owner/member capability.

## 2026-08-03 — P5-S1 started

- Task-Revision: `5`
- Plan-Version: `5`
- Plan-Step: `P5-S1`
- Authorization: user instructed Codex to begin implementing the Auth module.
- Scope decision: the active tracer slice is initial Workspace owner
  provisioning plus the already-declared Member List/UpdateRole operations.
  Login, invitation, removal/leave, Agent identity, HTTP compatibility, and
  Workspace lifecycle orchestration remain deferred.
- Policy hashes revalidated:
  - root `AGENTS.md`:
    `3893c1ff20196c5ee3ee0ea4a0e5ba5d7430cdca218c56278816fc0aed83e933`
  - `backend/AGENTS.md`:
    `7d81baac25cc791371509f5a2531ec3d15e2e2ea0d103f405e5d5b2a5f094a87`
  - root `CLAUDE.md`:
    `20105929c7aa15afaf6664112b5e360fc72aa23eba3ad13dc68bf805738ef5db`
- Current path: generated Member extension -> not-implemented service.
- Target path: generated local/gRPC adapter -> Auth member use case -> member
  domain/transaction ports <- Auth-owned SQLite adapter.
- Ownership decision: Auth owns users, members, the last-owner policy, and the
  one-time membership root. It does not inspect Workspace storage; later
  Workspace composition must coordinate the public Auth capability.
- Next action: extend only MemberService Proto, reconcile generated boundaries,
  inspect output, then implement user-owned layers in dependency order.

## 2026-08-03 — P5-S1 implementation and verification

- Extended `auth.v1.MemberService` additively with
  `ProvisionWorkspaceOwner`. Existing List/Update RPCs and message field
  numbers remain unchanged; Go bindings remain rooted at
  `github.com/hvritual/workspace/rpc/pb`.
- Reconciled MemberService with the frozen dddgen revision, applied the exact
  import postprocessor, and regenerated Buf bindings. Repeated generation
  produced the stable generated-tree digest
  `83fea01bbbb29511701da97ab01ec1ff552323b191f79c4b75579448874a8a23`.
  The complete `rpc/pb` inventory remains 37 Go files with digest
  `18bd5bbf0c73da60fdb90133357b296e2d8f495825bc74151fe2bf96ba4ac419`.
- Added an Auth-owned Member domain and application use case for one-time
  Workspace owner provisioning, Workspace-scoped member listing, and role
  updates. Role policy preserves lowercase JSON-compatible values and prevents
  demotion of the last owner.
- Added explicit actor context and fail-closed authorization. Non-members cannot
  observe a Workspace membership list, administrators cannot grant or revoke
  owner, and foreign-Workspace targets resolve as not found.
- Added Auth-owned SQLite tables for users, members, and the one-time Workspace
  membership root. There are no foreign keys or cascading actions. Native
  `BEGIN IMMEDIATE` transactions serialize first-owner creation and last-owner
  checks.
- Owner provisioning is idempotent for the same Workspace/User pair, including
  replay when ID generation is unavailable. A different user cannot replace an
  initialized root, and missing Auth users roll back without partial state.
- Added domain/application tests, SQLite migration and tenant-isolation tests,
  concurrent provisioning coverage, and bufconn gRPC round trips for Provision,
  List, and UpdateRole.
- Deterministic verification passed:
  - Buf format and STANDARD lint
  - repeated dddgen/postprocess/Buf generation with identical content digest
  - `go test -race ./internal/modules/auth/... -count=1`
  - `go test ./tests/contract/... -count=1`
  - `go test ./... -count=1`
  - `go vet ./...`
  - `go mod verify` (`all modules verified`)
  - forbidden inward/cross-module imports, no-FK SQL, gofmt, policy hashes, and
    `git diff --check`
- Default bootstrap remains the generated stub composition. This slice does not
  change HTTP APIs, current runtime behavior, database selection, or Workspace
  isolation semantics.
- The backend was restarted after verification as PID 42438 on HTTP
  `127.0.0.1:8000` and gRPC `127.0.0.1:9000`; `/healthz`, `/readyz`, gRPC health,
  and Workspace Ping pass.
- Login/session, invitations, removal/leave, Agent identity, and Workspace
  lifecycle orchestration remain deferred. P5-S1 deterministic acceptance is
  satisfied; independent acceptance remains pending.

## 2026-08-03 — P6-S1 started

- Task-Revision: `6`
- Plan-Version: `6`
- Plan-Step: `P6-S1`
- Authorization: user instructed Codex to proceed sequentially through Todo
  CRUD, Issue mainline and owned objects, Knowledge governance, Requirement
  state machine, real cross-module adapters, PostgreSQL/HTTP/realtime, and
  per-slice runtime cutover.
- Active scope is Todo full CRUD only. Later authorized slices remain ordered
  and dormant until P6-S1 deterministic verification completes.
- Installed Todo baseline owns Create/Get/List/Update/Delete, status and
  priority validation, Project/Issue/member validation, terminal evidence, and
  realtime events. Current target backend owns only Create and UpdateStatus.
- P6-S1 will migrate the complete data lifecycle behind opt-in SQLite/local/gRPC
  composition. Cross-module evidence, HTTP compatibility, PostgreSQL, realtime,
  and runtime switching stay deferred to their declared steps.
- Current path: generated Todo adapter -> partial application use case ->
  SQLite repository.
- Target path: generated Todo adapter -> full Todo use case -> Todo domain ->
  Workspace-scoped repository port <- SQLite adapter.
- Next action: extend Todo Proto additively, reconcile generated boundaries,
  then drive domain/application/persistence behavior from focused tests.

## 2026-08-03 — P6-S1 implementation and verification

- Extended `workspace.v1.TodoService` additively with Get, filtered List, full
  Update, and Delete while retaining Create and compatibility UpdateStatus.
  Existing message field numbers remain unchanged; Go bindings remain under
  `github.com/hvritual/workspace/rpc/pb`.
- Expanded the ordinary Todo projection with installed task priority, creator,
  position, start/due date, and completed-time fields. Agent execution
  lifecycle remains outside Todo.
- Added a trusted `WorkspaceActor` context seam. Create rejects missing callers
  and verifies creator and assignee identity through the Auth-facing consumer
  port; callers cannot supply or forge creator fields in the command body.
- Added full domain behavior for lowercase status/priority validation, optional
  update versus clear, UTC timestamps, defensive pointer copying, and installed
  PostgreSQL completion semantics: only `done` records completion; any supplied
  non-done status clears it.
- Added authorization-first application orchestration for all six RPCs,
  canonical Issue-ID resolution, Project/Issue/Actor validation, stable
  Workspace-scoped not-found mapping, deterministic filters, and a dedicated
  UpdateStatus permission path.
- Added the `000004_todo_crud` SQLite migration and provider CRUD mapping. List
  orders by position ascending, creation time descending, and ID ascending.
  No foreign key, cascade, or cross-module table read was introduced.
- Added domain/application tests and expanded SQLite local and bufconn gRPC
  coverage across Create/Get/List/Update/UpdateStatus/Delete, filtering/order,
  reference clearing, completion, tenant isolation, and repeated delete.
- A focused standards/spec review tightened status, priority, and non-empty
  RFC3339 parsing to exact installed HTTP semantics; whitespace-padded values
  are rejected rather than silently normalized. The automated two-agent review
  workflow could not use a commit diff because the backend migration baseline
  is still untracked, so the review was performed directly against the P6-S1
  file set and `plan_v6.md`.
- Repeated dddgen/postprocess/Buf generation plus gofmt produced the stable
  generated-tree digest
  `9cfa57e03cac5375affb1ca94ca5241ca4728a6734dbf19b61008fdd2072e8e5`.
  The `rpc/pb` inventory remains 37 Go files with digest
  `5b7ec0e4e7933cea06edb8cdb3ae2b71fc7d12f86358e1fa46a84034e24b4dce`.
- Deterministic verification passed after the review correction:
  - Buf format and STANDARD lint
  - repeated content-idempotent generation
  - `go test -race ./internal/modules/workspace/... -count=1`
  - `go test ./... -count=1`
  - `go vet ./...`
  - `go mod verify` (`all modules verified`)
  - generated contract, architecture/import, no-FK, gofmt, policy hash, scope,
    old-import, and `git diff --check` gates
- The backend remains PID 42438 on HTTP `127.0.0.1:8000` and gRPC
  `127.0.0.1:9000`; health, readiness, gRPC health, and Workspace Ping pass.
  Default bootstrap still selects generated stubs, so installed HTTP,
  PostgreSQL, realtime, and runtime behavior remain unchanged.
- P6-S1 deterministic acceptance is satisfied. Completion evidence remains
  assigned to Knowledge/cross-module/realtime slices rather than a fake no-op
  adapter.

## 2026-08-03 — P6-S2 started

- Plan-Step: `P6-S2`
- Active scope moves to Issue mainline first, followed by Issue-owned attached
  objects in dependency order. P6-S1 remains opt-in and unchanged.
- Next action: freeze the installed Issue capability inventory and split the
  main aggregate, metadata/catalog, collaboration, attachment, query, and
  dependent-cleanup paths into independently verifiable Issue sub-slices.

## 2026-08-03 — P6-S2 sidebar orchestration started

- Plan-Version: `7`
- The user assigned concrete Issue migration work to sidebar tasks and retained
  this thread as the plan owner and result reviewer.
- One write task was queued for Issue mainline. Four concurrent read-only tasks
  were queued for metadata/catalogs, collaboration, query/hierarchy, and
  assets/delete behavior.
- The single-writer rule is mandatory because Issue Proto reconciliation and
  generated Workspace files are shared, while the current backend baseline is
  not committed. Later implementation tasks will be created only from the
  latest accepted baseline.
- `plan_v7.md` records task identifiers, dependency order, the P6-S2A review
  gate, and the later write-task sequence.

## 2026-08-03 — P6-S2A control-thread acceptance

- Reviewed the completed Issue mainline sidebar result against immutable
  `plan_v7.md`. The sidebar task used an isolated worktree for execution, but
  its absolute `backend/` write paths resolved to the main checkout; no copy or
  merge was repeated during acceptance.
- Accepted the additive Create/Get/List/Update/UpdateStatus Issue contract and
  its domain, application, SQLite, local, gRPC, and contract-test paths. The
  actual scope contains no Delete, batch, move, search/query/table/group/facet,
  metadata/property/label mutation, collaboration object, attachment object,
  HTTP compatibility, PostgreSQL, realtime, or runtime-cutover implementation.
- Confirmed authorization precedes persistence; repository access is scoped by
  Workspace; Project, parent Issue, Actor, and Asset validation uses explicit
  inward-facing ports. Todo and Requirement were changed only to depend on the
  narrower `IssueReferenceRepository` read contract.
- Confirmed SQLite Issue number allocation, identifier assignment, and insert
  use one `BEGIN IMMEDIATE` transaction. Independent race coverage creates 24
  concurrent Issues with continuous unique numbers, and an abort trigger proves
  that a failed insert rolls back both the Issue row and Workspace counter.
- Rebuilt a temporary deterministic toolchain with Buf v1.72.0,
  protoc-gen-go v1.36.11, protoc-gen-go-grpc v1.5.1,
  protoc-gen-go-http v3.0.0, and dddgen source revision
  `7266939f0e295648593739c694ab4e614b141546`. The complete
  dddgen -> import postprocess -> Buf -> gofmt pipeline is content-idempotent.
  The 37-file `rpc/pb` digest remains
  `dd3330527ff292400a6adbd35bdbc0bd6208652d4214656b189dba3a07ae6002`.
- Independent deterministic verification passed after regeneration:
  - Buf format and STANDARD lint;
  - `go test -race ./internal/modules/workspace/... -count=1`;
  - `go test ./tests/contract/... -count=1`;
  - `go test ./... -count=1`;
  - `go vet ./...`;
  - `go mod verify` (`all modules verified`);
  - gofmt, generated import, inward dependency, cross-module internal import,
    no-FK/cascade, Issue-scope, and old generated-package static checks.
- Default bootstrap remains `workspace.New()` with generated stubs. P6-S2A is
  imported into the opt-in Workspace SQLite composition but does not perform a
  runtime cutover before real cross-module adapters and the later authorized
  tracer slice.
- The four read-only audits for metadata/catalogs, collaboration,
  query/hierarchy, and assets/delete also completed. Their evidence will be
  frozen into a new plan version before P6-S2B1 starts; no later Issue write
  task is active yet.
- Restarted the unchanged default backend after acceptance. The compiled server
  is PID `28904`, listening on HTTP `127.0.0.1:8000` and gRPC
  `127.0.0.1:9000`; `/healthz`, `/readyz`, live gRPC health, and Workspace Ping
  all pass.

## 2026-08-04 — P6-S2B1 plan approved

- Plan-Version: `8`
- Plan-Step: `P6-S2B1`
- Authorization: the user instructed the control thread to create the next
  plan version and delegate the metadata slice to a sidebar implementation
  task.
- Converted the completed metadata/catalog audit into an Issue metadata-only
  slice. Labels and custom properties remain P6-S2B2; query projections remain
  P6-S2D2.
- Contract decision: add `IssueMetadataService` with Get, single-key Put, and
  single-key Delete. Put accepts raw `value_json` to retain primitive JSON
  typing through persistence.
- Concurrency decision: SQLite mutations use `BEGIN IMMEDIATE`; the existing
  mainline Issue update must stop rewriting metadata/properties from stale
  projections. P6-S2B1 must prove concurrent metadata writes and Issue updates
  do not lose data.
- Frozen product baseline digest:
  `beb4a9b37bc07e78fbe9b89a3b09d82dc6499e50c30cc75776ed547cb2ab20bb`.
- Frozen policy bundle:
  `753606cfa0b1e99ecdeb7585ed8a541b19b1caed2f239f2f8a43f7b5ce7ab8cb`.
- Next action: create exactly one write-capable sidebar task from the current
  working-tree baseline. This control thread remains the reviewer and no other
  Issue writer may run concurrently.

## 2026-08-04 — P6-S2B1 sidebar implementation started

- The Codex app thread-creation and thread-list APIs remained unresponsive and
  produced no registered Git worktree. Git inspection confirmed no duplicate
  task checkout before fallback.
- Created an equivalent isolated task worktree at
  `/Users/fworld/.codex/worktrees/p6s2b1.fnL4aQ/goclaw`, detached at the frozen
  Git base and populated from the approved working-tree backend baseline.
- Revalidated in the task worktree:
  - HEAD `0c4f848a8458e256dd4fe2ed51498af969aa3c59`;
  - product digest
    `beb4a9b37bc07e78fbe9b89a3b09d82dc6499e50c30cc75776ed547cb2ab20bb`;
  - all three policy component hashes from `plan_v8.md`.
- Delegated the only active Issue writer to collaboration task
  `/root/p6_s2b1_metadata`. Its ownership is limited to product files below the
  isolated worktree's `backend/`; plan and journal files remain control-owned.
- Control-thread next action: wait for the sidebar result, inspect its actual
  diff against the frozen baseline, and run the P6-S2B1 acceptance gate before
  any import into the main backend.

## 2026-08-13 — P6-S2B1-V9 re-frozen and approved

- The user approved a staged Canonical backend migration with Issue metadata as
  the only active vertical slice. API URLs may change when necessary; request
  and response body content may not change.
- Re-froze the task at commit
  `e4b4b1c7e3d46b19fb4774f8757cad4fb4c4f1cc` on branch
  `codex/issue-metadata-v9`; policy bundle is
  `401b48d143b41e0715a8d4655abddfe0d8c15ff877db8620d2bbc93c6939a598`.
- Superseded v8 with approved `plan_v9.md` because v8 excluded HTTP and
  frontend compatibility while the confirmed task requires the full vertical
  connection.
- Indexed the installed GET/PUT/DELETE behavior, exact JSON shapes, validation,
  tenant hiding, concurrency intent, events, core schema, and existing shared
  read-only Issue detail consumer in `issue-metadata-parity-v9.md`.
- Preserved unrelated dirty paths and confirmed no `server/**` diff.
- Next action: add focused backend and core compatibility tests and record the
  expected RED result before product implementation.

## 2026-08-13 — P6-S2B1-V9 accepted

- Completed the Issue metadata vertical slice through domain, application,
  SQLite `BEGIN IMMEDIATE`, local, bufconn gRPC, opt-in HTTP compatibility, and
  the shared core API client. The existing Web/Desktop Issue detail projection
  remains the verified read-only UI; no new editing UI was introduced.
- Preserved installed JSON contracts: PUT accepts only `{"value": primitive}`;
  GET/PUT/DELETE return only `{"metadata": {...}}`; validation, authentication,
  Workspace hiding, count, size, and error bodies have compatibility tests.
- Core sends the active Workspace slug and its resolved ID while leaving bodies
  unchanged. HTTP identity is an injected resolver: absent configuration fails
  closed, token parsing is not duplicated, and caller-supplied actor headers are
  not trusted.
- Added deterministic evidence for malformed-data repair, defensive copies,
  delete-no-op timestamps, tenant isolation, Actor checks, transaction rollback,
  distinct-key concurrency, same-key last-committer-wins, and a real overlap
  between stale mainline `UpdateIssue` projection and a committed metadata Put.
- Both independent gates passed after corrections:
  - specification review: `PASS`;
  - code-quality review: `APPROVED` with no remaining P0-P2 findings.
- Verification passed:
  - `go test ./... -count=1`, `go vet ./...`, and `go mod verify` in `backend/`;
  - focused Workspace concurrency and compatibility tests repeatedly (10-100
    runs depending on the test);
  - all `@multica/core` tests (84 files, 545 tests) and typecheck;
  - Issue detail tests (37 tests) and `@multica/views` typecheck;
  - Web/Desktop/Core/Views typecheck through explicit Turbo filters;
  - backend policy check, `git diff --check`, generated-scope audit, and empty
    `server/**` diff.
- Environment/tool limitations retained as non-passing gates:
  - Go race binaries fail to start on this Windows host with `0xc0000139` across
    unrelated packages; no race result is claimed;
  - root `pnpm typecheck`/`pnpm test` scripts still reference the intentionally
    absent `@multica/mobile`, so explicit non-mobile package filters were used;
  - the combined frontend test run reached green Core/Web/Views results and 393
    passing Desktop tests, but Desktop `scripts/package.test.mjs` failed to parse
    with `SyntaxError: Invalid or unexpected token`; this is outside the changed
    paths and is not claimed passing;
  - `buf format --diff` reports repository-wide CRLF/LF differences on unchanged
    Proto files; target Proto structure was generated with frozen Buf and only
    the two Issue binding files remain in the generated diff;
  - the frozen private dddgen binary was unavailable; no dddgen-owned file or
    reconciliation state was hand-edited.
- Realtime publication and default-runtime cutover remain explicitly deferred.
  The default Workspace module still exposes generated stubs; the completed
  implementation is selected only by the opt-in SQLite Workspace composition.
- Candidate scope contains no `server/**` change and preserves all unrelated
  user dirty files. No commit or push was performed.

## 2026-08-13 — P6-S2B1-V9 RED/GREEN implementation evidence

- Revalidated branch `codex/issue-metadata-v9`, base
  `e4b4b1c7e3d46b19fb4774f8757cad4fb4c4f1cc`, all three frozen policy
  component hashes, dirty-tree exclusions, and the absence of a `server/**`
  diff before product changes.
- RED backend evidence: the focused domain test failed because
  `ValidateMetadataKey`, `ParseMetadataValueJSON`, and `NewMetadataBag` did not
  exist. The focused Workspace test then failed because the metadata contract,
  local composition, and request messages did not exist.
- RED core evidence: all three focused tests failed because `getIssue` returned
  an unparsed body, metadata defaulted to `undefined`, and dedicated metadata
  client methods did not exist.
- The frozen dddgen source revision was unavailable as a local binary or cache;
  installation from its private GitHub repository was rejected without
  authentication. Per the control-thread decision, no dddgen-owned file or
  `.dddgen` state was edited. Metadata contract, application, local composition,
  and extension files are explicitly user-owned.
- Buf v1.72.0, protoc-gen-go v1.36.11, and protoc-gen-go-grpc v1.5.1 were built
  or selected in a temporary tool directory. Buf format/lint and protobuf/gRPC
  generation succeeded. Ambient HTTP generator version drift rewrote unrelated
  generated HTTP files, so every unrelated generated path was restored; the
  candidate generated diff is limited to `workspace/v1/issue.pb.go` and
  `workspace/v1/issue_grpc.pb.go`. Subsequent idempotence generation must use a
  temporary output tree and compare only target Issue artifacts.
- GREEN evidence so far: focused Issue metadata domain tests, core API tests,
  core typecheck, and focused Workspace SQLite/local tests pass. Default
  Workspace composition remains metadata-free; only the explicit SQLite chain
  appends the user-owned metadata extension.
- No canonical durable realtime publisher was found in this opt-in slice.
  `issue_metadata:changed` publication remains a declared deferred risk and is
  not presented as runtime-ready behavior.

## 2026-08-13 — P6-S2B1-V9 specification review corrections

- Added failing compatibility tests before corrections. RED showed that the
  HTTP adapter accepted arbitrary actor headers, could not consume Core's
  active Workspace identity, exposed internal validation text, and returned
  prefixed limit errors instead of the frozen legacy messages.
- Added an injected `WorkspaceHTTPIdentityResolver` capability to the opt-in
  composition. A nil resolver fails closed; the adapter never parses bearer
  tokens or treats a slug as a Workspace ID. Tests inject a resolver that
  validates the Bearer and active slug, then returns canonical Workspace UUID
  and Actor identity. Core continues sending `X-Workspace-Slug` and metadata
  methods additionally send the mirrored `X-Workspace-ID` without changing
  request bodies.
- HTTP validation now preserves the frozen key/body/value order and messages,
  accepts unknown body fields, maps count/size rule failures to exact 400 JSON,
  and keeps DELETE keys untrimmed.
- Added same-key last-committer coverage, failed-mutation row/timestamp rollback
  coverage, and exact HTTP limit response coverage. Existing distinct-key
  concurrency and mainline overlap protection remain green.

## 2026-08-13 — P6-S2B1-V9 second specification review corrections

- RED proved that the Delete use case trimmed a spaced key and treated it as a
  valid key. Delete now validates and passes the exact path key; direct local
  and HTTP tests require the frozen regex error for spaced keys.
- Added a nil-resolver HTTP test with forged Workspace Actor headers. The
  adapter returns exact 401 JSON and ignores the forged identity.
- Replaced the domain-prevalidation rollback assertion with a legal mutation
  that reaches SQLite. A temporary abort trigger rejects the metadata UPDATE;
  the test verifies both metadata and `updated_at` remain byte-for-byte
  unchanged, then removes the trigger.
- Strengthened same-key concurrency proof with an external `BEGIN IMMEDIATE`
  lock that queues two already-started real goroutines. After releasing the
  lock, completion order is recorded and the persisted value must equal the
  last completed commit. The focused concurrency test passes repeatedly.

## 2026-08-13 — P6-S2B1-V9 quality review corrections

- HTTP handlers now require Workspace identity presence, then resolve and
  authenticate identity, and only then validate key or body. Unauthenticated
  malformed requests return exact 401; Authorization without Workspace
  identity returns exact 400.
- PUT performs a second JSON decode and requires EOF. Concatenated JSON values
  are rejected as `invalid request body`; unknown fields remain accepted.
- Same-key concurrency now uses explicit test-only gates. Both goroutines start
  before release; A commits and returns first, then B commits and returns, and
  the persisted value is exactly B. No production hook was introduced.

## 2026-08-13 — P6-S2B1-V9 final same-key concurrency proof

- Replaced the top-level serial release test with a test-only gated repository
  wrapping the real SQLite metadata repository. Both goroutines enter
  `PutMetadata` before either delegate call is released. A performs and records
  its real SQLite commit first; only then B performs and records its real
  commit. A waits for B before returning, so the two request lifetimes overlap.
  The asserted commit sequence is exactly `A,B` and persisted value is B.

## 2026-08-13 — P6-S2B1-V9 pre-commit revalidation

- Revalidated the candidate on branch `codex/issue-metadata-v9` at base
  `e4b4b1c7e3d46b19fb4774f8757cad4fb4c4f1cc` before the scoped commit and
  baseline merge requested by the user.
- Fresh passing evidence: focused Workspace tests, full backend `go test
  ./...`, `go vet ./...`, `go mod verify`, all 545 Core tests, Core typecheck,
  the 37 Issue detail tests, Views typecheck, `git diff --check`, and an empty
  `server/**` diff.
- `go test -race ./...` again failed to launch test binaries across unrelated
  packages on this Windows host with exit status `0xc0000139`; no race result
  is claimed.
- A combined Views test/typecheck invocation exceeded its 120-second command
  budget without a result. The focused Issue detail suite and Views typecheck
  were rerun separately and passed.
- The scoped commit excludes the preserved unrelated paths named in
  `plan_v9.md`. Independent specification and code-quality review results are
  recorded separately before commit.

## 2026-08-13 — P6-S2B1-V9 pre-commit review corrections

- Independent review found that the domain value parser and SQLite historical
  bag parser accepted a valid first JSON value with trailing malformed or
  concatenated data. RED tests reproduced both defects; both parsers now require
  EOF after the single JSON value, and focused plus full Workspace tests pass.
- PUT now preserves the frozen distinction between an empty key and a
  whitespace key. A whitespace key receives the legacy regex validation error.
- The HTTP adapter now hides an actor outside the Workspace as `404 issue not
  found`, rejects unsupported actor types before repository access, preserves
  operation-specific legacy PUT/DELETE persistence errors, and has executable
  coverage for each case.
- The dedicated Core metadata response schema now requires the `metadata`
  envelope instead of defaulting a missing field to an empty bag. The new RED
  response test is green; the Issue projection keeps its separate compatibility
  default for older Issue bodies.
- Fresh post-correction evidence: all focused Workspace packages, full backend
  tests, vet, module verification, all 546 Core tests, Core typecheck,
  `git diff --check`, and an empty `server/**` diff pass. The Windows race
  binary limitation remains unchanged and is not claimed passing.
