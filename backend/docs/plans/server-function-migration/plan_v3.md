# Incremental server function migration — remaining Workspace service chain

- Plan-ID: `server-function-migration`
- Version: `3`
- Status: `approved`
- Approval source: user instruction dated 2026-08-03 to continue the explicit
  `Todo -> Issue -> Knowledge -> Requirement -> Setting` chain
- Base commit: `0c4f848a8458e256dd4fe2ed51498af969aa3c59`
- Repository: `/Users/fworld/Hvritual/goclaw`
- Branch: `codex/multica-six-domain-baseline`
- Task type: `change`

## Goal

Complete the currently declared Workspace service chain behind the existing
Proto contracts, explicit authorization/cross-module ports, domain behavior,
SQLite persistence, local adapters, and gRPC adapters. Preserve the safe
database-neutral default runtime and do not cut over the installed server.

## Completed prerequisites

- P1-S1: Workspace identity and SQLite provider seam.
- P2-S1: Project and Relationship application/persistence/local/gRPC slices.

## Scope and ordered sub-slices

### P3-S1 — active coordinated Workspace chain

Execute and verify these sub-slices in order:

1. Todo Create/UpdateStatus with Project, Issue, and Member/Agent references.
2. Issue UpdateIssueStatus with UUID or Workspace identifier lookup.
3. Knowledge Create/Get with candidate-first governance and Asset references.
4. Requirement SaveVersion/Get with immutable revisions and Project/Issue
   traceability.
5. Setting PutWorkspaceSetting/PutWorkspaceSkillBinding with System Skill and
   Auth Agent references.

Add domain values, application use cases/ports, SQLite repositories, additive
`000003` migrations, explicit full-chain composition, application/integration/
gRPC tests, architecture documentation, and journal evidence. Do not edit any
generated file.

## Frozen behavior

- Todo title is trimmed/required; status defaults to `todo`; valid states are
  `todo`, `in_progress`, `done`, and `cancelled`. Project/Issue references and
  paired Member/Agent assignees must belong to the same Workspace.
- Issue valid statuses are `backlog`, `todo`, `in_progress`, `in_review`,
  `done`, `blocked`, and `cancelled`. Lookup accepts ID or Workspace-scoped
  identifier and updates only status plus `updated_at`.
- Knowledge title is trimmed/required and newly created records use
  `candidate`, preserving review-before-publication. Every Asset reference must
  resolve to the same Workspace through a Space-owned contract.
- Requirement title/content are required. New requirements start at version 1,
  existing requirements append exactly one immutable version, remain `draft`,
  and report `uncovered` or `covered` from Issue links. Linked Issues must
  belong to the same Workspace and Project.
- Workspace settings are keyed Workspace-owned JSON values. Skill bindings are
  keyed by Workspace+Skill, retain lowercase/reference JSON semantics, verify
  Skill/version through System, verify every Agent through Auth, and deduplicate
  Agent IDs while preserving order.
- Authorization precedes storage reads/writes that could reveal tenant data.

## Non-goals

- No changes outside `backend/`, no edits to `server/`, no runtime cutover, and
  no production data access.
- No Proto/generated/dddgen/OpenAPI/access-manifest changes.
- No Todo list/get/delete/full update; no Issue create/get/list/full update;
  no Knowledge review/publication/search/evidence; no Requirement approval or
  coverage calculation beyond declared issue references; no Setting get/list.
- No PostgreSQL provider, data backfill, realtime/event/outbox migration, or
  compatibility HTTP adapter.
- No Auth, Space, or System implementation. Only consumer-owned validation
  ports and test fakes are added.

## Invariants

- Workspace remains the authorization and tenant boundary for every operation.
- All repositories scope reads/writes by Workspace ID; foreign resources are
  returned as stable not-found/invalid-reference errors.
- Cross-module validation uses public contracts only, never tables or concrete
  adapters.
- Domain imports only the standard library. Application imports domain and
  stable contracts, never SQL/transport/generated packages.
- SQLite owns SQL, JSON mapping, nullable mapping, and native transactions.
- Requirement aggregate/version writes commit atomically.
- Existing generated files and the default generated stub behavior remain
  unchanged.

## Dependencies

- Existing v2 authorizer, Actor reader, Project repository, SQLite migration
  runner, and extension replacement pattern.
- New injected Space Asset and System Skill reference readers.
- Existing generated service/local/gRPC/Proto adapters for all five services.

## Acceptance criteria

1. Every declared RPC in the five-service chain succeeds through explicit
   SQLite composition on local and gRPC happy paths.
2. Todo and Issue state compatibility matrices are table-tested; invalid states
   never persist.
3. Todo rejects foreign Project/Issue/Actor references and Issue updates cannot
   cross Workspace boundaries.
4. Knowledge rejects foreign Asset references and creates candidate records.
5. Requirement versions increase monotonically, current version is returned,
   and foreign/cross-Project Issue links are rejected atomically.
6. Setting upserts are Workspace scoped; Skill/version and Agent references are
   validated before writes; configuration/value maps survive persistence.
7. Missing Asset/Skill/security dependencies fail full-chain composition while
   the v2 constructor and default module remain compatible.
8. Migrations apply transactionally/repeatably and record three versions.
9. Narrow race tests, contract/full tests, vet, Buf, import/scope checks, and
   generated-file immutability checks pass.

## Deterministic verification

- Run `gofmt` and `go mod tidy`.
- Run narrow domain/application tests after each sub-slice.
- Run `go test -race ./internal/modules/workspace/... -count=1`.
- Run contract tests, full `go test ./... -count=1`, and `go vet ./...`.
- Run `go mod verify` and fixed Buf v1.72.0 format/lint.
- Inspect forbidden inward/cross-module imports, generated-file mtimes/count,
  policy hashes, `git diff --check`, and repository status.

## Risks

- These RPC contracts are narrower than the installed Chi APIs. Completion of
  this plan does not imply HTTP cutover parity.
- Candidate Knowledge and simplified Requirement coverage are explicit interim
  contract semantics; richer review/evidence/coverage operations require Proto
  revisions.
- Real Auth/Space/System adapters are unavailable, so default runtime selection
  remains prohibited.
- SQLite schema is an isolated provider seam, not a production data migration.

## Rollback

Remove P3 domain/application/provider/composition/test files and additive
`000003` migrations. Restore the migration expectation to two versions. P1 and
P2 remain the safe verified baseline.

