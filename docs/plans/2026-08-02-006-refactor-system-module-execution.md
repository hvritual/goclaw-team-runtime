---
title: System Module Incremental Migration - Execution
type: refactor
date: 2026-08-02
topic: system-module-migration
artifact_contract: multica-ddd-execution/v1
execution_status: planned
depends_on:
  - four-module-proto-foundation
  - auth-agent-contract
  - space-asset-contract
---

# System Module Incremental Migration - Execution

## Outcome

Establish System as the owner of versioned Agent releases/upgrades and the global versioned Skill catalog, while Auth retains Agent identity and Workspace retains collaboration objects and tenant-level Skill enablement/configuration.

## Ownership Split

| Concept | Owner |
| --- | --- |
| Agent identity, team membership, authorization | Auth |
| Agent Release, Agent Version, artifacts, upgrade policy | System |
| Skill Definition, Skill Version, publication | System |
| Workspace Skill enablement, configuration, Agent binding reference | Workspace |
| Skill files represented as reusable storage objects | Space when externalized; System owns Skill-version composition |

## Tracer-Slice Order

### Y1. Publish a Skill version

- Characterize current workspace-scoped Skill content, files, import and bundle resolution.
- Define Skill Definition and immutable Skill Version behavior.
- Preserve content validation and bundle safety.
- Do not rewrite existing workspace rows until an activation/reference migration is separately approved.

### Y2. Workspace Skill activation contract

- System exposes published-version lookup.
- Workspace records enablement, configuration and Agent binding references.
- Resolve a runnable bundle through contracts; do not join System tables from Workspace repositories.

### Y3. Publish an Agent release

- Define immutable version, platform artifacts, checksums, compatibility and publication state.
- Keep binary/object lifecycle behind Space or release-storage ports.
- No real release publishing is executed during code migration tests.

### Y4. Plan and report Agent upgrades

- Resolve Auth-owned Agent identity through a contract.
- Apply Upgrade Policy to compute eligible target versions.
- Keep rollout state and audit records in System.
- Separate planning from the external operation that performs an upgrade.

## Data Migration Rules

- Existing workspace Skill data remains authoritative until a tested migration creates System definitions/versions and Workspace activations.
- No dual writes without an explicit, time-bounded rollout plan and reconciliation proof.
- No foreign keys across System, Workspace, Auth or Space.
- Every new index follows the repository's single-statement concurrent-index migration rule.

## Verification

- Skill-version immutability, publication and bundle-resolution tests.
- Workspace activation authorization and cross-workspace isolation tests.
- Agent-release checksum, compatibility and upgrade-policy tests.
- Fake release/upgrade adapters only; no authenticated Agent CLI, account or quota use.
- Architecture lint allowing only named System → Auth and System → Space contracts, with no implementation imports.

## Stop Conditions

- Skill Definition and Workspace activation would become dual sources of truth.
- Agent identity and Agent release state cannot be separated.
- A migration would overwrite Skill content or lose Agent bindings.
- Verification would require a real release, deployment or Agent upgrade.

## Open Decision Checkpoint

Agent execution lifecycle ownership is unresolved. This execution document must not add execution contracts, persistence, or runtime behavior until the product owner confirms an owning module and its dependencies.
