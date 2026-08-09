# Incremental server function migration — Project and Relationship

- Plan-ID: `server-function-migration`
- Version: `2`
- Status: `approved`
- Approval source: user continuation instruction dated 2026-08-03 after the
  verified Workspace identity slice
- Base commit: `0c4f848a8458e256dd4fe2ed51498af969aa3c59`
- Repository: `/Users/fworld/Hvritual/goclaw`
- Branch: `codex/multica-six-domain-baseline`
- Task type: `change`

## Goal

Continue the Workspace-first migration with a verified Project and
ProjectActorRelation tracer slice. Implement the existing Proto operations
behind explicit authorization, tenant-isolation, domain, persistence, local,
and gRPC boundaries without enabling an unauthenticated default runtime.

## Completed prerequisite

P1-S1 delivered the Workspace identity contract and SQLite-local provider seam.
Its evidence is recorded in the append-only migration journal.

## Context brief

- Purpose: Project owns workspace-scoped project identity, lifecycle state, and
  Asset references. Relationship owns Project-to-Member/Agent roles.
- Ubiquitous language: Project, Project status, Project actor relation, lead,
  member, agent.
- Owned state: Project fields declared by `ProjectService` and relation tuples
  declared by `RelationshipService`.
- Commands: CreateProject, PutProjectActorRelation,
  DeleteProjectActorRelation.
- Queries: GetProject, ListProjectActorRelations.
- Upstream dependency: injected Workspace access authorizer and Auth-owned Actor
  workspace reader.
- Consistency: one Project aggregate per create; one relation tuple per put or
  delete. Project deletion cleanup remains a later slice because DeleteProject
  is not in the current Proto contract.

## Frozen behavior

- Project name is trimmed and required.
- Empty Project status defaults to `planned`.
- Valid statuses match the installed server contract: `planned`,
  `in_progress`, `paused`, `completed`, and `cancelled`.
- Project reads use `(workspace_id, project_id)` and must not reveal a Project
  from another Workspace.
- Authorization runs before repository reads or writes that could reveal tenant
  data.
- Relation actor types are `member` or `agent`; member roles are `lead` or
  `member`, and the agent role is `agent`.
- Relation put requires both the Project and Actor to belong to the request
  Workspace. Relation delete remains possible after an Actor has left so stale
  relations can be cleaned safely.
- Asset IDs are stored as stable string references owned by Project; no Space
  table or content is accessed.

## Scope

### P2-S1 — active Project and Relationship tracer slice

- Add Project and ProjectActorRelation domain values and invariant tests.
- Add consumer-owned authorization and Actor-workspace dependency contracts.
- Add Project and Relationship application use cases and repository ports.
- Add SQLite repositories plus an additive `000002` provider migration. Do not
  edit the accepted `000001` migration.
- Upgrade the SQLite migration runner to apply ordered up migrations atomically.
- Add an explicit composition constructor that requires authorization and Actor
  dependencies and replaces only the generated Project/Relationship extension
  implementations. Do not edit dddgen-owned files.
- Add local/application, SQLite integration, and bufconn gRPC tests.
- Update architecture documentation and append verification evidence.

Only P2-S1 is authorized for product-code changes in this plan version.

## Non-goals

- No changes outside `backend/` and no edits to `server/`.
- No Proto, generated binding, dddgen state, HTTP annotation, OpenAPI, access
  manifest, or existing installed HTTP API changes.
- No default runtime persistence selection or authorization bypass.
- No Project list/update/delete/search/resources/statistics/evidence/realtime
  migration in this slice.
- No Auth implementation, cross-module table read, dual write, data backfill,
  PostgreSQL provider, or production database access.
- No Todo, Issue, Knowledge, Requirement, Setting, Space, or System behavior.

## Invariants

- Workspace remains the tenant and authorization boundary.
- Project and relation repository operations always include Workspace ID.
- Relationship never reads Auth storage directly; it uses the injected public
  Actor workspace contract.
- Domain is independent of transport, SQL, generated code, Kratos, and other
  modules. Application depends inward plus stable contracts only.
- SQLite owns SQL, row mapping, JSON Asset-reference storage, and native
  transactions.
- Generated files remain untouched and generation stays idempotent.
- The database-neutral default module continues returning the explicit
  generated not-implemented behavior for these services.

## Dependencies

- Completed P1-S1 Workspace identity slice.
- Existing ProjectService and RelationshipService Proto/generated contracts.
- Existing pure-Go SQLite dependency.
- Injected test fakes for authorization and Actor workspace membership until
  Auth supplies real adapters in a later plan revision.

## Acceptance criteria

1. Project create/get succeeds through application, SQLite, local, and gRPC
   paths when authorized.
2. Empty status defaults to `planned`; all five legacy statuses are accepted;
   unknown status and blank name are rejected before persistence.
3. Cross-Workspace Project get returns the stable Project not-found error on the
   local path and never returns foreign data.
4. Relation put rejects invalid actor/role combinations and Actors outside the
   Project Workspace without writing.
5. Relation list is deterministic and Workspace scoped; delete is idempotent
   and can clean a relation after Actor removal.
6. Missing dependencies fail composition; default `workspace.New()` remains
   database-neutral and unchanged.
7. SQLite migrations run in lexical order inside one transaction and can be
   applied repeatedly.
8. Narrow race tests, full Go tests/vet, Buf checks, architecture import checks,
   generation checksum, and repository scope checks pass.

## Deterministic verification

- Run `gofmt` and `go mod tidy`.
- Run domain/application unit tests first.
- Run `go test -race ./internal/modules/workspace/... -count=1`.
- Run `go test ./tests/contract/... -count=1`, `go test ./... -count=1`, and
  `go vet ./...`.
- Run fixed Buf v1.72.0 format/lint checks without modifying the project module.
- Re-run dddgen/Buf generation only if generated inputs changed; otherwise
  verify the generated-tree checksum remains the accepted P6 checksum.
- Inspect inward imports, `git diff --check`, policy hashes, and Git status for
  paths outside `backend/`.

## Risks

- The current Proto Project shape is intentionally narrower than the installed
  HTTP Project payload. This slice implements only declared RPC behavior and
  does not claim HTTP compatibility or cutover readiness.
- Auth is not migrated. A real runtime must not select the new constructor until
  it can supply authorization and Actor membership adapters.
- Project delete cleanup is deferred until DeleteProject is added through an
  explicitly approved Proto revision.
- The provider schema is isolated and additive; it is not a migration of
  existing PostgreSQL or SQLite-local production data.

## Rollback

Remove the P2 domain/application/provider files, additive `000002` migrations,
and explicit Project/Relationship SQLite composition constructor. Restore the
migration runner to the P1 single-file implementation. P1 Workspace identity
and the generated service stubs remain the safe baseline.

