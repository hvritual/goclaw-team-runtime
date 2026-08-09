# Four-module dddgen generation

- Plan-ID: `four-module-proto-foundation`
- Version: `2`
- Status: `approved`
- Approval source: user instruction to run dddgen dated 2026-08-02
- Base commit: `0c4f848a8458e256dd4fe2ed51498af969aa3c59`
- Repository: `/Users/fworld/Hvritual/goclaw`
- Branch: `codex/multica-six-domain-baseline`
- Task type: `change`

## Goal

Use the repository-pinned dddgen implementation to generate the four bounded
context module skeletons and application/interface code corresponding to the 12
approved Proto services under `backend/`.

## Scope

- All writes remain below `backend/`.
- Create the dddgen-required module roots for `workspace`, `auth`, `space`, and
  `system`.
- Reconcile these fully qualified Proto services:
  - `workspace.v1.ProjectService`
  - `workspace.v1.TodoService`
  - `workspace.v1.IssueService`
  - `workspace.v1.KnowledgeService`
  - `workspace.v1.RequirementService`
  - `workspace.v1.SettingService`
  - `workspace.v1.RelationshipService`
  - `auth.v1.MemberService`
  - `auth.v1.AgentService`
  - `space.v1.AssetService`
  - `system.v1.AgentReleaseService`
  - `system.v1.SkillService`
- Generate required Protobuf and gRPC Go bindings when needed for the generated
  code to compile.
- Update only `backend/go.mod` and `backend/go.sum` for generated-code
  dependencies.
- Permit dddgen reconciliation state and conventional primary module artifacts
  only when the pinned generator requires them.

## Non-goals

- No persistence provider, SQL, migration, database, cache, or transaction
  implementation.
- No business method implementation beyond generated stubs.
- No registration in the installed `server/` runtime and no changes to its
  HTTP API, Chi/sqlc code, routes, database behavior, or SQLite-local parity.
- No cross-module implementation imports or network loopback.
- No OpenAPI or access-manifest generation unless dddgen makes it an unavoidable
  precondition; any such expansion must stop for a new plan revision.

## Invariants

- All v1 ownership and workspace-isolation invariants remain in force.
- Proto remains the source of truth.
- Generated files are never hand-edited.
- Domain and application dependency direction follows the native DDD scaffold.
- Other modules may depend only on public contracts; this step adds no such
  dependency edge.
- IDs and compatibility-sensitive states remain strings.

## Dependencies

- Installed dddgen module version:
  `v0.0.0-20260802042746-1c5b2054726a`.
- This exactly matches the repository `DDD_SCAFFOLD_VERSION`.
- Go, Protobuf, gRPC, and any framework dependencies required by generated code
  remain isolated to the `backend` Go module.

## Ordered steps

### P2-S1 — Generate and verify the four-module code skeleton

Inspect generator preconditions, create the four module skeletons, reconcile all
12 fully qualified services, generate required Proto bindings, normalize Go
dependencies, inspect generated ownership boundaries, and run deterministic
format, lint, build, test, vet, scope, and generated-cleanliness checks that are
available in the standalone `backend` root.

## Acceptance criteria

1. The pinned dddgen binary is the generator used.
2. Four module roots exist under `backend/internal/modules/`.
3. All 12 approved Proto services have generated application/interface seams.
4. Generated Protobuf/gRPC bindings compile when required by those seams.
5. No persistence is selected and no business behavior is claimed complete.
6. Buf lint, Go formatting, compilation/tests, and vet pass for generated code.
7. No changed path escapes `backend/`; the existing `server/` runtime remains
   untouched.

## Deterministic verification

From `backend/`, use the fixed dddgen binary metadata, Buf format/lint, `gofmt`,
`go mod tidy`, `go test ./...`, and `go vet ./...`. Enumerate each generated
service seam and inspect dependency imports. From the repository root, reject
changed paths outside `backend/` and confirm the pre-existing user-owned
`backend/AGENTS.md` content hash is unchanged.

## Risks

- The standalone `backend` root may lack a generator-required primary module
  Proto or annotation dependency.
- Module creation may add conventional primary services that were outside v1's
  exact 12-file inventory.
- Generated code may reveal missing runtime or code-generation dependencies.

Use dddgen's preflight behavior and inspect every generated path before
continuing. Stop for a new plan revision if HTTP, access, persistence, or
installed-server changes become necessary.

## Rollback

Remove only artifacts created by P2-S1 under `backend/internal/`,
`backend/gen/`, `backend/.dddgen/`, and any generator-required new Proto files;
restore the v1 Go/Buf configuration from the worktree diff. Do not alter the v1
Proto contracts or any path outside `backend/`.

