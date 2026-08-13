# Context Discovery Loop Plan v3

- Plan-ID: `CX-W01-CONTEXT-DISCOVERY-001`
- Version: `v3`
- Status: `approved-for-execution`
- Supersedes: `plan_v2.md`
- Base commit: `3fab1050fb58a7dfea638b6c94f3b2e73745e9b4`
- Branch: `agent/cx-w01-context-discovery-loop-001`
- Project-ID: `goclaw-team-runtime`
- Task-ID: `CX-W01-CONTEXT-DISCOVERY-001`
- Task-Revision: `r003`
- Policy bundle: `backend/AGENTS.md@ffe45b83ef3884d9b8bf66e6f7994b3a43d4f86e`

This immutable amendment inherits the aggregate, state, readiness, security,
scope, verification, risk, and rollback contracts from `plan_v2.md`, with the
iteration semantics below clarified before final verification.

## Amendment B — Iteration budget bounds autonomous discovery, not human answers

`iteration` counts autonomous Context Discovery iterations. `max_iterations`
therefore limits autonomous discovery work and token consumption; it must not
make an already-issued required human question impossible to answer.

Effective transition rule:

1. A normal `context.iterate` in `discovering` increments `iteration` once.
2. If that snapshot opens a required human question, state becomes
   `human_required`, including when `iteration == max_iterations`.
3. While state is `human_required`, one or more answer-bearing snapshots may be
   submitted without increasing the autonomous iteration counter when the
   counter is already at the maximum.
4. After the required human question is answered:
   - if all required needs and blocking gaps are resolved, state becomes `ready`;
   - if machine-resolvable required context remains unresolved and the
     autonomous budget is already consumed, state becomes `exhausted`.
5. `ready` and `exhausted` remain terminal for Context Discovery until a material
   `requirement.change` reopens the aggregate.

This prevents the system from deadlocking at exactly the point where it asks a
human for required information, while preserving the autonomous loop bound.

## Active steps

- `CX-W01-S01`: complete.
- `CX-W01-S02`: implementation complete; Amendment B is the final semantic
  correction before verification.
- `CX-W01-S03`: command API implementation complete; focused HTTP validation is
  pending CI.
- `CX-W01-S04`: active.

## Verification evidence so far

PR `#11` governance-policy passed and confirmed there is no `server/**` diff.
The first canonical-backend attempt stopped at `fmt-check`; GitHub Actions
reported only `internal/controlplane/p2_flows_test.go` as unformatted. No Go
logic/test result has been claimed from that failed attempt.
