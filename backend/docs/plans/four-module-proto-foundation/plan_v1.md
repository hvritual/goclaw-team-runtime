# Four-module Proto foundation

- Plan-ID: `four-module-proto-foundation`
- Version: `1`
- Status: `approved`
- Approval source: user-provided standard execution prompt dated 2026-08-02
- Base commit: `0c4f848a8458e256dd4fe2ed51498af969aa3c59`
- Repository: `/Users/fworld/Hvritual/goclaw`
- Branch: `codex/multica-six-domain-baseline`
- Task type: `change`

## Goal

Establish a four-module Proto contract skeleton under `backend/` that makes
service ownership and cross-module references explicit without changing the
installed server or any runtime behavior.

## Scope

- `backend/api/workspace/v1/`: Project, Todo, Issue, Knowledge, Requirement,
  Setting, and Relationship contracts.
- `backend/api/auth/v1/`: Member and Agent identity contracts.
- `backend/api/space/v1/`: Asset lifecycle contract.
- `backend/api/system/v1/`: Agent release and Skill catalog contracts.
- Standard Buf configuration and a Go module named
  `github.com/hvritual/workspace` under `backend/`.

## Non-goals

- No generated Go, gRPC, OpenAPI, access-manifest, or reconciliation output.
- No `dddgen` execution.
- No changes to `server/`, handlers, services, sqlc, routes, databases, or the
  SQLite-local implementation.
- No HTTP annotations or runtime registration.

## Invariants

- Workspace remains the tenant and authorization boundary.
- Proto packages use `<module>.v1`; Go packages use
  `github.com/hvritual/workspace/internal/<module>/v1`.
- IDs and compatibility-sensitive states remain strings.
- Cross-module references are stable IDs, never imported generated types.
- Auth owns Member and Agent identities.
- Space owns Asset content lifecycle; consumers own business associations.
- System owns Agent releases and the versioned Skill catalog.
- Workspace owns tenant-level Skill enablement, configuration, and binding.
- RelationshipService owns Project-to-Member/Agent roles and same-Workspace
  validation; Project deletion cleans relations within the Workspace module.

## Dependencies

- Protobuf and gRPC Go runtime modules are declared for future generation.
- Buf validates source contracts only in this step.

## Ordered steps

### P1-S1 — Create and validate the contract-only skeleton

Create the scoped Proto and configuration files. Validate formatting, Buf lint,
Go module integrity, the exact service/package inventory, absence of generated
artifacts, and that every changed path remains below `backend/`.

## Acceptance criteria

1. The requested 12 Proto files exist at the exact paths in the task.
2. Each file declares one owned service, core request/response messages, and
   ownership comments.
3. Ownership and cross-module ID references match the stated invariants.
4. Buf formatting and STANDARD lint pass.
5. The Go module path is `github.com/hvritual/workspace`.
6. No generated or runtime files are created.
7. No repository path outside `backend/` is changed by this task.

## Deterministic verification

From `backend/`:

```text
buf format --diff --exit-code
buf lint
go mod verify
```

From the repository root, inspect the changed-path allowlist, enumerate Proto
services/packages/Go package options, and reject `*.pb.go`, `*_grpc.pb.go`,
OpenAPI, access-manifest, and `.dddgen` artifacts below `backend/`.

## Risks

- Contract names may accidentally imply ownership outside the agreed context.
- Direct generated-type imports could couple modules.
- Generation could create forbidden artifacts during this foundation step.

The acceptance checks address these risks before completion.

## Rollback

Remove only files added by `P1-S1` under `backend/api/`, the new `backend`
Buf/Go configuration files, and this plan directory. The existing `server/`
runtime remains untouched throughout.

