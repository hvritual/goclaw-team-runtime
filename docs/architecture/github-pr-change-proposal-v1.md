# GitHub PR to Proposed Change v1

P2-S04 converts verified GitHub merge evidence into a proposed Engineering `Change`. It does not accept that Change.

## Preconditions

All three identities must already agree:

1. the GitHub PR is merged and exposes an immutable merge commit SHA;
2. the repository has one unambiguous authoritative `github` SourceBinding to a canonical EngineeringEntity;
3. the supplied Workspace Project/Requirement/Task has an authoritative Workspace work-link to that same EngineeringEntity.

If any precondition is missing, no Change is created. The system does not guess work ownership from commit messages, PR titles, branches, or AI inference in v1.

## Change identity

The proposed Change ID is deterministic over:

`workspace + canonical repository locator + PR number + work kind + work ID`.

Replaying the same source evidence is idempotent. If the Change was subsequently accepted, replay returns that accepted Change and never resets it to proposed. A same-ID semantic mismatch is a conflict.

## Evidence mapping

The generated Change includes:

- one affected EngineeringEntity: the repository SourceBinding target;
- a `pull_request` artifact pinned to the merge commit SHA;
- a `commit` artifact pinned to the same immutable merge SHA;
- provenance `source_type=github_pr`, source locator `github://owner/repository/pull/N`, revision = merge SHA.

Broader impact expansion is intentionally deferred to P2-S05 Scope Resolver rather than overclaiming that every Task-linked entity was directly modified by one PR.

## Governance boundary

P2-S04 never calls `Change.Accept`, never mutates Workspace Task/Todo state, never marks a Run successful, and never publishes Knowledge. Acceptance remains an explicit independent governance action.
