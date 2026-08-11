# P0-P2 Backend Platform Plan v1

- Plan-ID: `P0P2-BACKEND-PLATFORM-001`
- Version: `v1`
- Status: `approved-for-execution`
- Base commit: `1c0054ada767bf12d2efb3b0f7869dcfb8c91a4b`
- Branch: `agent/p0-p2-backend-platform-001`
- Project-ID: `goclaw-team-runtime`
- Task-ID: `P0P2-BACKEND-PLATFORM-001`
- Task-Revision: `r001`
- Policy bundle: `backend/AGENTS.md@ffe45b83ef3884d9b8bf66e6f7994b3a43d4f86e`

## Goal

Turn the `backend/` DDD/Proto baseline into an executable backend platform through the previously identified P0, P1, and P2 scope: deterministic backend checks; workspace identity, membership, authorization, audit, persistence, errors, and validation; an append-only delivery kernel; requirement, quality, review, knowledge, execution, and Team Control HTTP capabilities.

## Hard invariants

1. Every changed path is under `backend/**`; `server/**` remains untouched.
2. Workspace data is isolated by `workspace_id`; authorization is evaluated server-side.
3. Runner executes commands but cannot advance acceptance state.
4. Only deterministic Checker/DoneGate decisions can advance governed state.
5. Event history is append-only, project-sequenced, hash chained, idempotent, and replayable.
6. Creator or assignee cannot perform independent final acceptance.
7. No credential or unsanitized secret is persisted in events, evidence, audit, or logs.
8. A path-blocked or environment-blocked gate remains explicitly incomplete.

## Scope

- `backend/docs/plans/p0-p2-backend-platform/**`
- `backend/Makefile`, `backend/Dockerfile`, `backend/README.md`
- `backend/ci/**`
- `backend/cmd/controlplane/**`
- `backend/internal/controlplane/**`

## Non-goals

- No modifications to `server/**`.
- No root GitHub workflow edits while the backend-only path boundary is active.
- No edits to `apps/**` or `packages/**`; frontend wiring is represented by a versioned HTTP contract only.
- No automatic merge, release, or production deployment.
- No LLM deciding authoritative state transitions.

## Steps

### P0P2-S00 — Freeze plan

Scope: this plan, pointer, and append-only journal only.

Acceptance: identifiers, base, scope, invariants, gates, risks, and rollback are explicit.

### P0P2-S01 — P0 backend gates

Create a backend-local Makefile, container build, deterministic CI script, generated-code drift hook, architecture/path checks, and operator README. The root workflow and frontend remain path-blocked.

Acceptance:

- one command runs format, test, race, vet, and policy checks;
- checks fail when a candidate diff includes `server/**`;
- container starts the new control-plane binary;
- six-domain terminology is explained as six delivery capability domains over the existing four platform contexts.

### P0P2-S02 — P1 platform foundation

Implement workspace lifecycle, memberships and roles, actor identity, server-side authorization, audit, consistent errors, validation, pagination limits, repository contracts, SQLite persistence, injectable production SQL persistence, migrations, transactions, idempotency, and optimistic concurrency.

Acceptance:

- cross-workspace access is denied;
- owner invariants and role transitions are table-tested;
- data survives SQLite reopen;
- conflicting versions fail closed;
- audit records are immutable and workspace-scoped.

### P0P2-S03 — P1 delivery kernel

Implement project-sequenced `SessionEvent`, SHA-256 hash chain, command idempotency, CAS, reducer/projection, deterministic replay, evidence index, unified Work Graph, and DoneGate.

Acceptance:

- replay is deterministic;
- broken chains are detected;
- duplicate commands return the original result without a second event;
- acceptance requires deterministic evidence and an independent acceptor.

### P0P2-S04 — P2 requirement flow

Implement Request, Intent clarification/decision, Solution/ADR, four reviews, Freeze, ChangeIntent, Task, and traceability/coverage projection.

Acceptance: only a fully reviewed revision can freeze and produce tasks; material changes require a new revision.

### P0P2-S05 — P2 quality flow

Implement Defect and Risk models, reproduction evidence, severity/probability/impact, response ownership, creator/approver separation, review due date, work links, verification, and close gate.

Acceptance: unverified defects and unreviewed risks cannot close.

### P0P2-S06 — P2 review and knowledge flow

Implement structured ReviewFinding, deterministic and model-review separation, KnowledgeCandidate source/evidence links, deduplication key, review, version publication, invalidation, rollback, and evaluation metadata.

Acceptance: model output cannot publish or resolve findings without deterministic/human gates.

### P0P2-S07 — P2 execution flow

Implement Run queue, claim, lease, heartbeat, cancel, timeout, retry, isolated workspace reference, sanitized secret references, evidence return, and human termination.

Acceptance: Runner cannot self-accept; expired leases can be safely reclaimed; retry is bounded and idempotent.

### P0P2-S08 — Team Control backend API

Expose typed JSON HTTP endpoints for workspace, requirement, quality, review, knowledge, execution, evidence, replay, and DoneGate projections. Include explicit success and problem response shapes.

Acceptance: handler tests cover valid, invalid, denied, not-found, conflict, and workspace-isolation responses.

### P0P2-S09 — Verification and handoff

Run deterministic checks available in the environment, index evidence, record unavailable gates, and open a Draft PR targeting `codex/multica-six-domain-baseline`.

Acceptance: changed paths are restricted to `backend/**`; no unsupported gate is claimed; independent review and missing root CI/frontend wiring remain explicit before DoneGate.

## Deterministic verification

```bash
cd backend
make check
make test-race
make policy-check BASE_REF=codex/multica-six-domain-baseline
docker build -t goclaw-controlplane:p0p2 .
```

Environment note: the current execution environment has neither `go` nor `gh`; Go checks must be executed by a toolchain-enabled CI/reviewer before DoneGate.

## Risks and controls

- Broad scope: preserve step boundaries and atomic commits.
- Existing generated contracts: add the new platform as a self-contained backend slice; do not hand-edit generated code.
- Persistence drift: keep migrations explicit, transactional where supported, and covered by reopen tests.
- Authority confusion: encode Checker/Runner separation in types and state transitions.
- Secret leakage: persist secret references only and reject raw secret fields at HTTP boundaries.
- Frontend/root-CI path conflict: publish stable API and check scripts, but keep those two gates blocked until separately authorized.

## Rollback

The implementation is isolated on one branch. Revert commits in reverse step order. Database changes are additive and versioned; the control plane refuses unknown schema versions. No production migration or release is performed by this plan.
