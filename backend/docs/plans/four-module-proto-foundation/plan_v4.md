# Relocate generated Protobuf packages

- Plan-ID: `four-module-proto-foundation`
- Version: `4`
- Status: `approved`
- Approval source: user instruction dated 2026-08-02 to relocate pb packages,
  remove the previous generated code, and regenerate
- Base commit: `0c4f848a8458e256dd4fe2ed51498af969aa3c59`
- Repository: `/Users/fworld/Hvritual/goclaw`
- Branch: `codex/multica-six-domain-baseline`
- Task type: `change`

## Goal

Relocate every generated Protobuf Go package from
`github.com/hvritual/workspace/gen/go` to
`github.com/hvritual/workspace/rpc/pb`, remove the P3 generated code, and
regenerate the complete four-module skeleton from source contracts.

## Scope

- Change all Proto `go_package` options to
  `github.com/hvritual/workspace/rpc/pb/<package>`.
- Change Buf binding output from `gen/go` to `rpc/pb`.
- Remove only these P3 generated artifacts before regeneration:
  - `backend/gen/`
  - `backend/internal/modules/`
  - `backend/tests/contract/`
  - `backend/docs/architecture/modules/`
  - the four dddgen primary Proto files `workspace.proto`, `auth.proto`,
    `space.proto`, and `system.proto`
- Preserve the 12 business Proto contracts, annotation source, shared platform
  registry, plans, Buf configuration, and Go module.
- Recreate four primary module skeletons with the pinned dddgen binary,
  normalize the fixed primary-template pb imports in unmarked base module files,
  reconcile all 12 extension services, and regenerate Go/gRPC/Kratos HTTP
  bindings into `rpc/pb`.

## Non-goals

- No business implementation, persistence, database, OpenAPI, access manifest,
  or installed-server cutover.
- No edits to files marked `Code generated` or to the final `rpc/pb` tree.
- No changes outside `backend/`.

## Invariants

- Proto remains the source of truth.
- The final tree contains no generated `gen/go` package or import.
- Workspace isolation and all four-module ownership rules remain unchanged.
- Extension services retain no HTTP route annotations.
- Pinned dddgen and pinned Kratos v3 generation tools remain unchanged.

## Dependencies

- dddgen `v0.0.0-20260802042746-1c5b2054726a`.
- Repository-staged Buf and Go/gRPC/Kratos v3 Proto plugins.
- The dddgen primary module template has a fixed `gen/go` path; preflight proved
  it refuses to overwrite an existing primary Proto. Therefore primary module
  creation must precede path normalization.

## Ordered steps

### P4-S1 — Clean regenerate under rpc/pb

Update source package options and Buf output, validate exact deletion targets,
remove the prior generated artifacts, recreate the module skeletons, normalize
the primary-template paths, batch reconcile extension services, generate
bindings, tidy dependencies, and run deterministic quality and scope gates.

## Acceptance criteria

1. Every Proto `go_package` begins with
   `github.com/hvritual/workspace/rpc/pb`.
2. All generated binding files are below `backend/rpc/pb`; `backend/gen` does
   not exist.
3. Four module roots and 12 dddgen extension states are freshly regenerated.
4. No source or generated Go import references the old `gen/go` prefix.
5. Buf format/lint, `go test ./...`, and `go vet ./...` pass.
6. No OpenAPI, access manifest, persistence, or changed path outside `backend/`
   exists.

## Deterministic verification

- Enumerate and validate exact deletion targets before removal.
- Verify all Proto package options and Buf output paths.
- Run pinned `dddgen module` four times, then pinned `dddgen proto-services`.
- Run pinned Buf generation and verify regenerated content idempotence.
- Run `gofmt`, `go mod tidy`, Buf format/lint, `go mod verify`,
  `go test ./...`, and `go vet ./...`.
- Reject old path references, forbidden artifacts, persistence output,
  cross-module internal imports, and repository changes outside `backend/`.

## Risks

- The fixed dddgen primary template initially emits old package paths; failure
  to normalize every unmarked base file would break compilation.
- Stale reconciliation state could hide an incomplete regeneration; removing
  all module roots before generation avoids this.
- A broad deletion could remove user-owned prerequisites; targets are explicit
  and validated before removal.

## Rollback

Delete `backend/rpc/pb` and the P4 regenerated module/test/architecture trees,
restore Proto and Buf paths to `gen/go`, then rerun the P3 generation sequence.
Do not touch shared platform code or any path outside `backend/`.

