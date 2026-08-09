# Workspace module

- Public seam: internal/modules/workspace/contract
- Implementation: internal/modules/workspace/internal
- Proto contract: api/workspace/v1/workspace.proto
- Tenant boundary: every business object and identity lookup remains explicitly
  scoped by Workspace ID.

## Migrated capability

The first server migration slice exposes `WorkspaceIdentityReader` as a narrow
local contract. Its application query depends on a Workspace-owned repository
port; the SQLite implementation and migration remain under Workspace
infrastructure. Missing rows are translated to the stable
`contract.ErrWorkspaceNotFound` error.

The default `workspace.New()` constructor remains database-neutral. A caller
must explicitly choose `NewWithSqlitePersistence`, apply `MigrateSqlite`, and
own the native database lifecycle. No production runtime selects that provider
yet.

Auth may consume only the public identity contract in a future planned slice.
It must not import Workspace domain, infrastructure, interfaces, or database
tables.

## Project and Relationship slice

`ProjectUseCase` owns Create/Get/List/Search/Update/Delete orchestration over the
accepted v1 Project fields. The Project domain owns required identity/name, the
`planned` default, the five legacy lowercase statuses, Asset-reference
integrity, optional update behavior, and UTC timestamps. Every repository read
or mutation uses both Workspace ID and Project ID. List ordering and
case-insensitive Search ranking/pagination are deterministic; closed Projects
are excluded from Search unless explicitly requested.

`RelationshipUseCase` is authoritative for
`ProjectActorRelation(workspace_id, project_id, actor_type, actor_id, role)`.
Member actors may be `lead` or `member`; Agent actors use `agent`. Put verifies
the Project through the Workspace-owned Project repository and Actor membership
through the injected `WorkspaceActorReader`. Delete intentionally does not
require current Actor membership so stale relations remain cleanable.

The explicit `NewWithSqliteWorkspaceServices` composition requires both a
`WorkspaceAccessAuthorizer` and `WorkspaceActorReader`. It replaces only the
generated Project and Relationship extension instances with migrated use cases;
all dddgen-owned files remain untouched. The default `workspace.New()` path
continues to expose the generated not-implemented stubs until Auth can supply
real adapters.

SQLite owns `workspace_projects` and
`workspace_project_actor_relations`. It stores Asset IDs as Project-owned JSON
references and has no foreign keys or cross-module table reads. Ordered
provider migrations run transactionally and are recorded in
`workspace_schema_migrations`.

Project deletion is the Workspace module's internal consistency boundary. One
SQLite transaction deletes ProjectActorRelations, clears Project references on
Todo and Issue, deletes Requirement versions and aggregates for the Project,
and finally deletes the Project. A failed step rolls the entire cleanup back;
no database cascade is used.

Full Workspace tenant lifecycle remains gated. Auth now exposes durable,
idempotent initial-owner provisioning, but no accepted atomic or compensating
composition yet coordinates it with Workspace creation. The backend must not
implement Workspace Create with a fake or missing owner.

## Todo through Setting service chain

`NewWithSqliteWorkspaceChain` is the explicit composition seam for the next
Workspace tracer slice. It retains the Project and Relationship use cases and
replaces the generated Todo, Issue, Knowledge, Requirement, and Setting
extensions without editing generated files. Composition fails closed unless a
Workspace authorizer, Auth Actor reader, Space Asset reader, and System Skill
reference reader are supplied. The default `workspace.New()` constructor and
the narrower `NewWithSqliteWorkspaceServices` constructor remain compatible and
unchanged in behavior.

- Todo owns ordinary task records and the `todo`, `in_progress`, `done`, and
  `cancelled` state lifecycle. It now implements Create/Get/filtered List/full
  Update/compatibility UpdateStatus/Delete with the installed priority,
  position, creator, schedule, and completion fields. Optional Project, Issue,
  Member, and Agent values are reference IDs validated within the request
  Workspace; Agent execution lifecycle remains outside Todo.
- Issue currently migrates Workspace-scoped status mutation. Reads accept the
  stable Issue ID or Workspace identifier, and the repository update predicate
  includes both Workspace and Issue ID.
- Knowledge creates candidate records before any future review/publication
  operation. It stores only Space Asset IDs and verifies each Asset through the
  public Asset reader; Workspace does not read Space storage.
- Requirement owns the requirement aggregate and append-only version rows.
  Aggregate-current-version and new immutable version writes share one SQLite
  transaction. Linked Issues must resolve in the same Workspace and Project;
  the aggregate remains draft and derives covered/uncovered from those links.
- Setting owns Workspace JSON settings and tenant-level Skill enablement,
  configuration, and Agent bindings. Skill definitions and versions remain in
  System, Agent identity remains in Auth, and repeated Agent IDs are normalized
  before persistence.

The additive `000003_workspace_service_chain` and `000004_todo_crud` migrations
create only Workspace-owned provider state, without foreign keys or
cross-module table reads. Todo List ordering is position ascending, creation
time descending, then ID ascending. Supplied empty optional references or dates
clear those values. Only `done` records `completed_at`, matching the installed
PostgreSQL query; any supplied non-done status clears it. Status, priority, and
non-empty RFC3339 inputs use exact validation rather than whitespace coercion.

Local-contract and bufconn gRPC tests exercise all Todo RPCs and the existing
service chain. Completion evidence, installed HTTP compatibility, PostgreSQL,
realtime publication, and runtime selection remain later explicit slices; the
default bootstrap still uses generated stubs.
