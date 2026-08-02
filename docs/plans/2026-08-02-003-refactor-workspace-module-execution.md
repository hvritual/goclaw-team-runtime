---
title: Workspace Module Incremental Migration - Execution
type: refactor
date: 2026-08-02
topic: workspace-module-migration
artifact_contract: multica-ddd-execution/v1
execution_status: planned
depends_on:
  - four-module-proto-foundation
---

# Workspace Module Incremental Migration - Execution

## Outcome

Migrate Workspace collaboration behavior into one module containing Project, Todo, Issue, Knowledge, Requirement, Setting and Relationship application services. Each use case moves as an independently verifiable tracer slice while preserving current APIs and storage behavior.

## Target Boundary

```text
server/modules/workspace/
  domain/
  application/
    contracts/       # only for demonstrated stable provider-owned contracts
  dependency/
    postgres/
    sqlite/
  interfaces/
    http/
```

Only create directories required by the active slice. Domain and application code must not import Chi, sqlc, pgx, SQLite, generated Proto types or concrete infrastructure.
Prefer consumer-owned application ports. Add `application/contracts` only for a demonstrated stable provider contract; do not create a generic root contract layer.

## Invariants

- Workspace remains the tenancy and authorization boundary.
- Every scoped query and mutation carries and validates `workspace_id`.
- Public routes, JSON fields, error statuses and WebSocket events remain stable.
- Todo is the ordinary workspace task and never represents an Agent execution.
- Knowledge remains independently disableable and its store failure does not disable core Workspace services.
- Project–Actor relation truth exists only in RelationshipService.
- Relationship writes validate Auth-owned Member/Agent identities in the same Workspace.
- No database foreign keys or cascade actions are added.

## Tracer-Slice Order

### W1. Issue status update

- Generate and review depguard rules with this first new module, then prove the gate with temporary forbidden-import probes before adding business implementation.
- **Current path:** Chi `PUT /api/issues/{id}` → mixed handler → sqlc transaction → knowledge evidence → `issue:updated`.
- **Target path:** HTTP adapter → `UpdateIssueStatus` application use case → Issue domain rule → repository/evidence/event ports.
- **Proof:** status validation, UUID/identifier resolution, `done` evidence, response shape, `status_changed`, `prev_status`, PostgreSQL and SQLite-local parity.
- **Exit:** the status branch has one application owner; unrelated Issue fields remain on the legacy path until their own slices.

### W2. Todo lifecycle

- Characterize the current `task` API and migration 235 schema.
- Expose Todo vocabulary at the application/Proto boundary while retaining storage/API compatibility where needed.
- Prove Project/Issue references, status transitions, workspace isolation and SQLite parity.

### W3. Project lifecycle

- Move one Project create/update use case first.
- Keep Project state separate from ProjectActorRelation.
- Preserve project resource references until Space adapters are available.

### W4. Project actor relationships

- Introduce `ProjectActorRelation` as the sole source of truth for lead/member/agent roles.
- Validate actors through Auth contracts.
- Migrate existing lead/member representations only after read/write parity tests exist.
- Delete duplicate relationship fields or paths only in an explicitly authorized schema slice.

### W5. Requirement lifecycle

- Reuse the existing project-requirements domain/application and repository-contract pattern.
- Preserve revision conflict and approval authorization behavior.
- Keep Requirement–Issue links in the Requirement application boundary.

### W6. Knowledge lifecycle

- Preserve evidence outbox, at-least-once delivery, idempotency and fail-open isolation.
- Keep Knowledge storage/search behind application ports.
- Consume Project, Issue, Todo and Asset references without reading their tables as contracts.

### W7. Workspace settings

- Separate tenant settings from user preferences and System upgrade policy.
- Preserve slug, issue prefix, workspace lifecycle and role-gated writes.

### W8. Workspace Skill activation

- Characterize current workspace-scoped Skill rows, Agent bindings, authorization, bundle resolution, and cache behavior.
- Consume System-published Skill versions through a stable contract.
- Keep workspace-level enablement, configuration, and Auth-owned Agent binding references in Workspace.
- Define a tested migration from current workspace Skill content only after System Skill publication is available; do not dual-write indefinitely.
- Prove cross-workspace isolation, missing/inactive version behavior, binding cleanup, and bundle-resolution compatibility.

## Verification per Slice

1. Focused domain/application tests.
2. Adapter contract tests for PostgreSQL and affected SQLite behavior.
3. HTTP behavior tests for route, JSON and errors.
4. Event/evidence tests where the slice publishes side effects.
5. `make sqlc` only after query changes.
6. `make lint-ddd`, focused `go test`, then `make test`/`make check` in proportion to risk.

## Stop Conditions

- Ownership of a rule or transaction cannot be established.
- Workspace isolation, last-owner safety, event ordering or PostgreSQL/SQLite parity would change.
- A slice requires permanent forwarding code, dual writes or cross-module table reads.
- Overlapping user changes exist in the target path.

## Completion Evidence

- Each migrated use case reports old path, new path, ports, adapters, composition changes and tests actually run.
- Superseded paths are removed only after all callers move.
- Remaining legacy use cases stay explicitly listed; directory shape alone is never reported as DDD completion.
