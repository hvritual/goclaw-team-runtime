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

### A3. Invitation lifecycle

- Move create, accept, revoke and expiry behavior behind Auth application contracts.
- Preserve public token boundaries and do not leak invitation storage types.

### A4. Agent identity

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

## Stop Conditions

- Agent identity semantics are still coupled to runtime/release configuration.
- Project roles would become a second Membership truth.
- A membership mutation can leave a Workspace without an Owner.
- Cache invalidation or self-event behavior cannot be preserved.
