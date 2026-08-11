# Team Control Connection Plan v2

- Plan-ID: `TC-W01-TEAM-CONTROL-001`
- Version: `v2`
- Status: `approved-for-execution`
- Supersedes: `plan_v1.md`
- Base commit: `c43f4300eb29cf6778e67594e54cc79f8fb5057e`
- Branch: `agent/tc-w01-team-control-001`
- Task-Revision: `r002`

This immutable amendment inherits every goal, invariant, allowed/forbidden path, ordered step, verification gate, dependency, risk control, and rollback instruction from `plan_v1.md` except for the explicit changes below.

## Amendment A — one authority for human membership

Human workspace roles and membership remain authoritative in the existing authenticated upstream. The control plane exposes a typed read-only workspace/member projection and reconciles that complete trusted snapshot before each business request. TC-W01 does not add local HTTP mutations for human roles or membership because those writes would create two competing sources of truth.

Control-plane Agent membership remains governed by the backend repository and is not removed by human identity reconciliation. No browser-provided actor or role is trusted.

## Amendment B — S01 acceptance wording

Replace “Expose typed read/manage endpoints for workspace and members” with:

> Expose typed, schema-versioned read endpoints for the trusted workspace/member projection. Human membership management continues through the existing authoritative product workflow; the Team Control surface identifies this boundary without duplicating writes.

All other S01 acceptance conditions remain unchanged. This reduces privilege surface and preserves v1 invariants 2, 3, 5, and 6.

## Amendment C — verification evidence

Because the active execution environment has no Go or Docker binaries, local verification is limited to static inspection and available Node checks. Go formatting, unit/race tests, policy checks, and the Docker build remain blocking GitHub CI evidence before the Draft PR can leave Draft status.
