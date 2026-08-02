---
title: Four-Module Proto Contract Foundation - Execution
type: refactor
date: 2026-08-02
topic: four-module-proto-foundation
artifact_contract: multica-ddd-execution/v1
execution_status: completed
depends_on: []
---

# Four-Module Proto Contract Foundation - Execution

## Outcome

Create the source-only Proto layout for Workspace, Auth, Space, and System. This execution establishes names and ownership; it does not introduce a Proto runtime or migrate existing backend behavior.

## Authorized Changes

- Add only `.proto` files under `server/api/`.
- Relocate the existing Issue status contract from `issue.v1` to `workspace.v1` without losing fields or comments that describe current compatibility behavior.
- Add empty service declarations where no RPC contract has been characterized yet.
- Add core messages only when a confirmed ownership decision requires them, such as `ProjectActorRelation`.

## Non-Goals

- No `dddgen`, Buf, `protoc-gen-go`, `protoc-gen-go-grpc`, access generator or OpenAPI generator.
- No generated files under `gen/`.
- No Go dependencies, router wiring, handlers, application services or database changes.
- No invented HTTP routes for services whose current API has not been characterized.

## Steps

1. Create the target module/package directories.
2. Move the Issue source contract to `workspace/v1/issue.proto` and change only its package/go-package ownership.
3. Add Workspace service skeletons for Project, Todo, Knowledge, Requirement, Setting and Relationship.
4. Define the confirmed Project–Actor relationship vocabulary in `relationship.proto`.
5. Add Auth Member and Agent service skeletons.
6. Add Space Asset service skeleton and stable Asset identity/version messages.
7. Add System AgentRelease and Skill service skeletons with versioned publication vocabulary.
8. Compile every Proto source in one `protoc` invocation.
9. Verify that the repository contains no generated `*.pb.go` files from this execution.

## Acceptance Evidence

```sh
find server/api -name '*.proto' -print | sort
find server/api -name '*.proto' -print0 | sort -z | \
  xargs -0 protoc -I server -I /usr/local/include \
  --descriptor_set_out=/dev/null
find server/api -name '*.pb.go' -o -name '*_grpc.pb.go'
git diff -- server/api
```

- Exactly four Proto packages exist: `workspace.v1`, `auth.v1`, `space.v1`, and `system.v1`.
- Service names are unique within each package.
- Existing Issue status strings and response fields remain represented.
- ProjectActorRelation contains Workspace, Project, actor type, actor ID and role.
- No generated output exists.

## Completion Record

- Completed on 2026-08-02 and amended after two-axis review.
- Added 12 Proto source files across `workspace.v1`, `auth.v1`, `space.v1`, and `system.v1`.
- Relocated the Issue status source contract from `issue.v1` to `workspace.v1` after validating the replacement.
- Joint `protoc` descriptor validation passed with all 12 source files in one invocation.
- No `*.pb.go`, `*_grpc.pb.go`, Buf configuration, Go dependency, route, handler, migration, or runtime change was produced.

## Stop and Recovery

- Stop if any target Proto file overlaps unrelated user work.
- Stop rather than import unavailable scaffold annotations.
- If joint syntax validation fails, keep the old Issue source until the new package validates.
- Recovery removes only the newly added Proto sources and restores the prior Issue source path.
