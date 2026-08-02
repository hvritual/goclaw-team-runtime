---
title: Auth Module Incremental Migration - Execution
type: refactor
date: 2026-08-02
topic: auth-module-migration
artifact_contract: multica-ddd-execution/v1
execution_status: in-progress
depends_on:
  - four-module-proto-foundation
---

# Auth Module Incremental Migration - Execution

## Outcome

Establish Auth as the owner of User identity, Workspace Membership, Workspace Role and future Agent identity. Auth exposes stable contracts for other modules without taking ownership of Project relationships or Agent release data.

## Native Target

```text
server/internal/modules/auth/
  contract/
  internal/domain/
  internal/application/
  internal/infrastructure/sqlite/
  internal/interfaces/
  module.go
```

Proto under `server/api/auth/v1` is the transport and access-metadata source of truth. New persistence work starts with SQLite; PostgreSQL adapters follow only after the same public behavior is proven.

## Owned and Referenced Data

- **Owned:** User, Member, Membership, invitation, Workspace Role, last-owner invariant, Agent identity and Agent authorization.
- **Referenced:** Workspace ID as the tenancy boundary.
- **Not owned:** ProjectActorRelation, Agent Release, Agent Version, Skill Version, Agent execution, Asset.

## Tracer-Slice Order

### A1. Change member role

- Characterize current middleware, handler, membership cache invalidation and error behavior.
- Move the last-owner rule into Auth domain behavior.
- Define repository and transaction ports owned by the application use case.
- Preserve owner/admin/member authorization and workspace-scoped lookups.

**SQLite tracer completed 2026-08-02:** `MemberService.UpdateMemberRole` now owns role validation, Owner-only Owner-role changes, and the last-Owner invariant. The native SQLite provider executes requester lookup, target lookup, owner count, update, and response projection in one transaction. SQLite-local `PATCH /api/workspaces/{id}/members/{memberID}` delegates to this service and retains its response/error contract. The PostgreSQL handler remains legacy and must be migrated before A1 is fully complete.

### A2. Remove or leave membership

- Reuse the last-owner policy.
- Preserve self-event guards, navigation races and dependent cleanup behavior.
- Keep cleanup application-owned; add no cascade action.

**SQLite tracer completed 2026-08-02:** `DeleteMember` and `LeaveWorkspace` now share the Auth membership transaction port and last-Owner domain policy. The SQLite-local DELETE/leave handlers delegate to Auth, preserve 204 success and existing 403/404/last-Owner errors, and no longer contain a second membership-removal business branch. PostgreSQL and the final Kratos runtime cutover remain pending.

### A3. List workspace members

- Move member-list authorization, workspace scoping, ordering and user projection behind the Auth contract.
- Preserve the top-level JSON array and nullable avatar representation used by existing clients.

### A4. Invitation lifecycle

- Move create, accept, revoke and expiry behavior behind Auth application contracts.
- Preserve public token boundaries and do not leak invitation storage types.

### A5. Agent identity

- Introduce Agent as a team participant identity, separate from release/version/runtime state.
- Define visibility and authorization rules before adding persistence.
- Expose lookup/validation contracts used by Workspace Relationship and System rollout.

## Cross-Module Contracts

- Workspace calls Auth to validate Member/Agent existence and Workspace membership.
- Relationship stores only Auth identity IDs and actor type.
- System targets Agent IDs but owns release/version state.
- Auth must not call Workspace or System adapters directly; collaboration binds at the composition root.

## Verification

- Domain tests for last-owner and role-transition invariants.
- Application tests for authorization, missing membership and transaction rollback.
- PostgreSQL and SQLite-local parity where the current behavior exists.
- Membership cache and realtime tests.
- Architecture lint proving Auth domain/application do not import Workspace/System/Space implementations.

## A1 SQLite Evidence

- Domain and application tests cover valid roles, admin restrictions, and last-Owner protection.
- Native provider tests use the embedded Auth migration with a real in-memory SQLite database and verify commit, rollback, workspace scoping, and response projection.
- SQLite-local HTTP tests cover successful promotion, rejection of sole-Owner demotion, and the existing hidden-workspace-before-body-decoding behavior.
- The Kratos transport boundary maps Auth failures to stable 400/401/403/404/500 status codes without leaking transport concerns into domain/application code.
- dddgen regeneration preserves the user-owned application method and regenerates the Kratos HTTP/OpenAPI/access boundaries.
- Frontend `updateMember` validates the runtime response schema before returning it to React Query consumers.
- Full frontend typecheck and the serial test gate pass across Core, Views, Desktop, Web, and Docs.

## A2 SQLite Evidence

- Domain tests cover member/admin removal permissions, Owner-only Owner removal, and last-Owner departure protection.
- Application tests cover successful removal/leave and policy rejection before deletion.
- Real SQLite tests cover committed deletion/leave and deletion rollback inside the module transaction.
- SQLite-local HTTP tests cover member removal, voluntary leave, Owner-removal authorization, and sole-Owner leave rejection.
- Bufconn contract tests prove Delete/Leave request fields cross the generated gRPC client/server boundary.
- Proto-declared `http_success_status = 204` is applied deterministically to generated Kratos handlers, HTTP clients, and OpenAPI; raw HTTP and generated-client round trips both prove the no-body contract.

## A3 SQLite Evidence

- `ListMembers` is now owned by the Auth application contract and SQLite provider; the SQLite-local route no longer performs its own membership query.
- Application and real SQLite tests prove actor membership authorization, workspace isolation, stable creation-time ordering, and full user projection.
- Proto `response_body: "members"` preserves the existing top-level JSON array, while the generation postprocessor keeps the generated HTTP client and OpenAPI schema aligned with that public shape.
- Raw Kratos HTTP, generated HTTP client, generated gRPC, SQLite-local lifecycle, and frontend runtime-schema tests cover the same member-list contract.

## A4 SQLite Evidence

- `RevokeInvitation` now owns Owner/Admin authorization and pending-only, workspace-scoped revocation inside an Auth invitation transaction.
- The SQLite-local DELETE route delegates to the Auth contract and retains 204 success plus existing 403/404/500 behavior.
- Application tests prove authorization occurs before invitation lookup; real SQLite tests prove workspace isolation and that an already non-pending invitation cannot be revoked again.
- Raw/generated Kratos HTTP tests preserve the Proto-declared 204 response, while gRPC/local adapters carry the same workspace and invitation IDs.

## Stop Conditions

- Agent identity semantics are still coupled to runtime/release configuration.
- Project roles would become a second Membership truth.
- A membership mutation can leave a Workspace without an Owner.
- Cache invalidation or self-event behavior cannot be preserved.
