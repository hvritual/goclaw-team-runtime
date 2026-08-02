---
title: Four-Module Integration and Cutover - Execution
type: refactor
date: 2026-08-02
topic: four-module-cutover
artifact_contract: multica-ddd-execution/v1
execution_status: planned
depends_on:
  - workspace-module-slices
  - auth-module-slices
  - space-module-slices
  - system-module-slices
---

# Four-Module Integration and Cutover - Execution

## Outcome

Complete composition, architecture enforcement, compatibility verification and removal of superseded legacy paths after independently verified module slices have landed.

## Preconditions

- Each migrated use case has behavior tests and reports its old/new path.
- Auth, Space and System expose stable local contracts required by Workspace.
- PostgreSQL and SQLite-local behavior is characterized for every affected route.
- No unresolved overlapping worktree changes exist in the cutover paths.

## Steps

### C1. Composition roots

- Construct concrete adapters only in the existing server composition roots.
- Bind in-process contracts locally; use gRPC only when a deployment boundary genuinely exists.
- Preserve Chi middleware order, workspace identity and existing server lifecycle.

### C2. Revalidate and expand architecture gates

- Require the depguard gate introduced with the first Workspace module slice to remain green, then expand/review it as Auth, Space and System modules appear.
- Require domain/application inward dependency rules and named cross-module contracts.
- Prove the gate with temporary forbidden-import probes, then remove the probes.

### C3. Compatibility matrix

- Compare HTTP route, request, response, error, authorization and event behavior.
- Run PostgreSQL adapter/integration tests and affected SQLite-local suites.
- Verify Knowledge fail-open behavior, Member last-owner safety and workspace query filtering.

### C4. Legacy path removal

- Remove a legacy handler/service/repository path only after all callers use the module path.
- Do not keep permanent forwarding packages or duplicate business models.
- Remove temporary compatibility only when its documented boundary no longer exists.

### C5. Generated contract adoption

- Bootstrap Buf/dddgen/access generation only as a separate reviewed change.
- Mark generated ownership and keep application method bodies user-owned.
- Inspect all generated diffs and require generated-clean verification.

**Foundation completed 2026-08-02:** the separate native-scaffold change is
recorded in `2026-08-02-008-dddgen-native-scaffold-execution.md`. Runtime
cutover remains governed by C1–C4 and is not implied by generated registration.

## Verification Gates

```sh
make sqlc                 # only when queries changed
make lint-ddd
make test
make check
```

Add focused module, contract, interface, race and integration tests in proportion to each slice. Never report an unrun gate as passing, and never run real-Agent smoke tests without explicit authorization.

## Rollback and Safe Stop

- Cut over one use case at a time so routing/composition can return to the prior implementation without data reversal.
- Stop before irreversible schema removal until read/write parity and backup/recovery evidence exist.
- Preserve role grants and inactive access resources when generated declarations change.
- Report every remaining legacy path and unresolved decision at each checkpoint.

## Definition of Done

- Four module boundaries have executable dependency gates, not only directories.
- Every business rule has one owner and every cross-module call uses an explicit contract.
- Public behavior and workspace isolation remain verified.
- Superseded legacy paths are removed, with no dual writes or hidden shared-table integration.
