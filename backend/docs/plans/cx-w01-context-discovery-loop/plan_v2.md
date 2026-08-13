# Context Discovery Loop Plan v2

- Plan-ID: `CX-W01-CONTEXT-DISCOVERY-001`
- Version: `v2`
- Status: `approved-for-execution`
- Supersedes: `plan_v1.md`
- Base commit: `3fab1050fb58a7dfea638b6c94f3b2e73745e9b4`
- Branch: `agent/cx-w01-context-discovery-loop-001`
- Project-ID: `goclaw-team-runtime`
- Task-ID: `CX-W01-CONTEXT-DISCOVERY-001`
- Task-Revision: `r002`
- Policy bundle: `backend/AGENTS.md@ffe45b83ef3884d9b8bf66e6f7994b3a43d4f86e`

This immutable amendment inherits the goal, non-goals, deterministic sufficiency
rules, iteration bound, ordered verification, risks, and rollback from
`plan_v1.md`, except for the aggregate contract corrected below.

## Amendment A — Context Pack is part of the Requirement aggregate

`plan_v1.md` proposed a separate `context_pack` work node. That shape makes a
material `requirement.change` and Context Pack invalidation two independent
kernel commands, creating an avoidable partial-update window. Context Discovery
is requirement clarification state, so v2 stores the Context Pack inside
`RequirementData` and advances it through the same requirement work-node event.

This preserves one atomic revision stream for:

- raw request;
- Context Discovery state;
- finalized intent;
- proposed solution;
- material intent change.

No new database schema or event type is required.

## Effective data contract

`RequirementData` gains optional `context` with:

- `state`: `discovering | human_required | ready | exhausted`
- `objective`
- `iteration`
- `max_iterations`
- `needs[]`
- `gaps[]`
- `questions[]`
- `source_refs[]`
- `summary`

### Need

- `id`
- `description`
- `required`
- `status`: `open | resolved | blocked`
- `resolution`
- `source_refs[]`

### Gap

- `id`
- `description`
- `blocking`
- `status`: `open | resolved`
- `resolution`

### Human question

- `id`
- `question`
- `required`
- `status`: `open | answered`
- `answer`

## Effective state rules

1. `context.start` initializes the embedded Context Pack as `discovering` with
   iteration `0` and a bounded max iteration count.
2. `context.iterate` replaces the current discovery snapshot, increments the
   iteration exactly once, validates the snapshot, and deterministically derives
   the next Context state.
3. Required unanswered human questions take precedence and yield
   `human_required` even when the autonomous iteration budget is consumed.
4. If no required human question is open but required needs/blocking gaps remain
   when the iteration bound is reached, state becomes `exhausted`.
5. State becomes `ready` only when all required needs are resolved, all blocking
   gaps are resolved, and all required human questions are answered.
6. `requirement.intent` requires embedded Context state `ready`.
7. `requirement.change` atomically clears intent/solution and resets the embedded
   Context Pack to `discovering`, iteration `0`, with previous resolutions and
   answers reopened so they must be revalidated.

## Active step

- `CX-W01-S01`: complete.
- `CX-W01-S02`: active.
- `CX-W01-S03` and `CX-W01-S04`: blocked on S02.

## Allowed paths for active step

- `backend/internal/controlplane/p2_flows.go`
- `backend/internal/controlplane/p2_flows_test.go`

All other v1 path restrictions remain in force, especially the permanent
`server/**` prohibition.
