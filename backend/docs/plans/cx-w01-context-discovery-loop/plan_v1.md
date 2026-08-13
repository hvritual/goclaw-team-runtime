# Context Discovery Loop Plan v1

- Plan-ID: `CX-W01-CONTEXT-DISCOVERY-001`
- Version: `v1`
- Status: `approved-for-execution`
- Base commit: `3fab1050fb58a7dfea638b6c94f3b2e73745e9b4`
- Branch: `agent/cx-w01-context-discovery-loop-001`
- Project-ID: `goclaw-team-runtime`
- Task-ID: `CX-W01-CONTEXT-DISCOVERY-001`
- Task-Revision: `r001`
- Policy bundle: `backend/AGENTS.md@ffe45b83ef3884d9b8bf66e6f7994b3a43d4f86e`

## Goal

Implement a deterministic, replayable Context Discovery Loop in the canonical
`backend/**` control plane so requirement intent cannot be finalized until the
minimum required context has been resolved or an explicit human-decision state
has been reached and answered.

The loop turns context discovery into durable control-plane state rather than a
transient prompt-building activity.

## Non-goals

- Do not modify `server/**`.
- Do not add an LLM provider, retrieval engine, vector database, web crawler, or
  connector implementation in this work item.
- Do not implement frontend UI in v1.
- Do not change membership, authentication, run leasing, or evidence-attestation
  trust rules.
- Do not make probabilistic model confidence authoritative for readiness.

## Invariants

1. `server/**` remains permanently read-only.
2. A Context Pack belongs to exactly one requirement and project.
3. Context readiness is computed from deterministic fields:
   - every required need is resolved;
   - no blocking gap remains open;
   - no required human question remains unanswered.
4. `requirement.intent` is rejected unless the requirement has a related Context
   Pack in `ready` state.
5. A material intent change reopens context discovery and invalidates prior
   readiness until the Context Pack is iterated again.
6. Context iterations are append-only through kernel events and each mutation
   advances the Context Pack revision exactly once.
7. The server enforces a bounded iteration count to prevent unbounded autonomous
   loops.
8. Human-required state is explicit and cannot be bypassed by setting a model
   confidence score.

## Data contract

### Context Pack

A `context_pack` work node contains:

- `requirement_id`
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

`source_refs` are sanitized, non-secret provenance references such as repository
paths, artifact URIs, decision IDs, or external source identifiers. v1 stores
references only; source acquisition is intentionally outside this plan.

## State machine

`requirement.start`
→ requirement `clarifying`
→ `context.start`
→ context pack `discovering`
→ zero or more `context.iterate`
→ one of:

- `discovering`: unresolved machine-resolvable context remains;
- `human_required`: required human question is open;
- `ready`: deterministic sufficiency predicate passes;
- `exhausted`: iteration bound reached while sufficiency still fails.

Only `ready` permits `requirement.intent`.

`requirement.change` moves the requirement back to `clarifying` and reopens the
linked Context Pack as `discovering` with readiness invalidated.

## Ordered steps

### CX-W01-S01 — Freeze plan and contracts

Allowed paths:

- `backend/docs/plans/cx-w01-context-discovery-loop/**`

Acceptance:

- versioned plan exists with deterministic state rules, scope, verification,
  risks, and rollback.

### CX-W01-S02 — Implement Context Discovery domain flow

Allowed paths:

- `backend/internal/controlplane/p2_flows.go`
- `backend/internal/controlplane/p2_flows_test.go`

Acceptance:

- Context Pack lifecycle and deterministic readiness are implemented;
- iteration bound is enforced;
- material intent change invalidates readiness;
- intent finalization is blocked before readiness;
- table/flow tests cover ready, human-required, exhausted, and invalidation paths.

### CX-W01-S03 — Expose typed command API

Allowed paths:

- `backend/internal/controlplane/http.go`
- `backend/internal/controlplane/http_test.go`

Acceptance:

- typed `context.start` and `context.iterate` commands are supported;
- unknown fields remain rejected;
- HTTP test proves `requirement.intent` fails before Context Ready and succeeds
  after Context Ready.

### CX-W01-S04 — Deterministic verification and review evidence

Allowed paths:

- `backend/docs/plans/cx-w01-context-discovery-loop/journal.md`

Verification:

```bash
cd backend
gofmt -w internal/controlplane/p2_flows.go internal/controlplane/p2_flows_test.go internal/controlplane/http.go internal/controlplane/http_test.go
go test ./internal/controlplane
make check
make test-race
```

Acceptance:

- no `server/**` diff;
- focused tests pass;
- backend deterministic checks pass or any environment-specific blocker is
  recorded with exact command/output;
- final diff is independently reviewed before merge.

## Risks

- Existing clients currently call `requirement.intent` immediately after
  `requirement.start`; enforcing the new gate is intentionally breaking for that
  incomplete flow. v1 chooses correctness over a compatibility bypass.
- Context source references are provenance identifiers, not trusted evidence
  attestations. They must never be interpreted as credentials or execution
  authorization.
- A low iteration bound can require human intervention earlier than desired; a
  high bound can waste tokens. v1 defaults to 8 and allows a smaller caller
  bound, never an unbounded value.

## Rollback

Revert the CX-W01 commits from the feature branch. No schema migration is added;
all new state is represented by existing event-sourced work-node payloads. The
legacy `server/**` tree is unaffected.
