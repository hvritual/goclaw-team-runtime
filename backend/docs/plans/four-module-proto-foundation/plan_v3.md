# Four-module native dddgen generation

- Plan-ID: `four-module-proto-foundation`
- Version: `3`
- Status: `approved`
- Approval source: user instruction to run dddgen dated 2026-08-02, refined by
  pinned-generator preflight evidence
- Base commit: `0c4f848a8458e256dd4fe2ed51498af969aa3c59`
- Repository: `/Users/fworld/Hvritual/goclaw`
- Branch: `codex/multica-six-domain-baseline`
- Task type: `change`

## Goal

Generate a compiling, contract-first four-module DDD skeleton under `backend/`
using the exact pinned dddgen binary and its native package conventions.

## Scope

- All v2 generation scope remains included.
- Adopt dddgen's canonical generated Proto package path
  `github.com/hvritual/workspace/gen/go/<module>/v1` for all module services.
- Add the generator-required local annotation source, Google API Buf dependency,
  primary module Proto services, and shared extension registry.
- Add module metadata to every service and access metadata to each RPC while
  leaving all 12 extension RPCs without HTTP route annotations.
- Generate Go, gRPC, and Kratos HTTP bindings only.
- Reconcile the 12 fully qualified extension services through dddgen.
- Normalize generated-code dependencies in the `backend` Go module.

## Non-goals

- No OpenAPI generation.
- No access-manifest generation or runtime access synchronization.
- No persistence provider, SQL, migration, business implementation, or runtime
  cutover.
- No changes outside `backend/`.
- No modification of the installed `server/` HTTP or gRPC surface.

## Invariants

- Existing HTTP API and runtime behavior remain unchanged because generated
  extension RPCs have no HTTP bindings and the standalone backend is not wired
  into `server/`.
- Workspace remains the tenant and authorization boundary.
- Cross-module business references remain string IDs.
- Proto is the source of truth; generated files are never hand-edited.
- Module communication remains contract-only with no dependency edges added in
  this skeleton step.

## Dependencies

- Pinned dddgen:
  `github.com/fworld/go-ddd-scaffold@v0.0.0-20260802042746-1c5b2054726a`.
- Protobuf, gRPC, Kratos v3, Google API annotations, and the installed Go/gRPC/
  Kratos HTTP protoc plugins.
- `internal/platform/module` is the only shared runtime primitive copied from
  the same pinned scaffold baseline already committed under `server/`.

## Ordered steps

### P3-S1 — Complete native generation and deterministic verification

Add the proven prerequisites, generate the remaining three primary modules,
reconcile all 12 fully qualified services, generate transport bindings, tidy the
module, inspect generated ownership and import boundaries, and run the available
Buf and Go quality gates from `backend/`.

## Acceptance criteria

1. Four primary module skeletons and all 12 extension service seams are present.
2. Protobuf, gRPC, and Kratos HTTP generated bindings compile.
3. Every extension RPC retains no HTTP annotation, so installed HTTP APIs are
   unaffected.
4. No OpenAPI, access manifest, persistence, or business implementation exists.
5. Buf format/lint, Go formatting, `go test ./...`, and `go vet ./...` pass.
6. Every changed path remains under `backend/`.

## Deterministic verification

- Verify dddgen binary module metadata against the repository pin.
- Run Buf format and STANDARD lint from `backend/`.
- Run configured Go/gRPC/Kratos HTTP binding generation; reject OpenAPI/access
  outputs.
- Run `gofmt`, `go mod tidy`, `go test ./...`, and `go vet ./...`.
- Enumerate 4 module roots, 12 extension service state files, and 12 generated
  service seams.
- Reject cross-module implementation imports and changed paths outside
  `backend/`.

## Risks

- Newly generated method stubs intentionally return not-implemented errors and
  must not be mistaken for migrated business behavior.
- Primary Ping services are generator bootstrap artifacts, not installed API
  cutover endpoints.
- Access metadata exists as source declarations but is not generated into or
  synchronized with a runtime catalog in this step.

## Rollback

Remove P2/P3 generated module, binding, annotation, registry, lock, and
reconciliation-state artifacts below `backend/`; restore the v1 Proto
`go_package` options and Buf/Go configuration. Never touch `server/`.

