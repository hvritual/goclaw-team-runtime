# P0-P2 Backend Platform Plan v2

- Plan-ID: `P0P2-BACKEND-PLATFORM-001`
- Version: `v2`
- Status: `approved-for-execution`
- Supersedes: `plan_v1.md`
- Approval: user authorized `backend CI` on 2026-08-11
- Base commit: `1c0054ada767bf12d2efb3b0f7869dcfb8c91a4b`
- Branch: `agent/p0-p2-backend-platform-001`
- Project-ID: `goclaw-team-runtime`
- Task-ID: `P0P2-BACKEND-PLATFORM-001`
- Task-Revision: `r002`
- Policy bundle: `backend/AGENTS.md@ffe45b83ef3884d9b8bf66e6f7994b3a43d4f86e`

## Goal

Turn the `backend/` DDD/Proto baseline into an executable backend platform through P0, P1, and P2: deterministic backend checks; workspace identity, membership, authorization, audit, persistence, errors, and validation; an append-only delivery kernel; requirement, quality, review, knowledge, execution, and Team Control HTTP capabilities.

This revision authorizes one repository-governance file, `.github/workflows/backend.yml`, solely to run the deterministic backend gate for PR #8. It grants no broader root-path authority.

## Hard invariants

1. Backend product and test changes remain under `backend/**`; the sole non-backend path allowed by this revision is `.github/workflows/backend.yml`.
2. `server/**` remains permanently read-only and cannot be granted an exception by this plan.
3. Workspace data is isolated by `workspace_id`; authorization is evaluated server-side.
4. Runner executes commands but cannot advance acceptance state.
5. Only deterministic Checker/DoneGate decisions can advance governed state.
6. Event history is append-only, project-sequenced, hash chained, idempotent, and replayable.
7. Creator or assignee cannot perform independent final acceptance.
8. No credential or unsanitized secret is persisted in events, evidence, audit, or logs.
9. A failed, unavailable, or path-blocked gate remains explicitly incomplete.
10. P2 implementation cannot start until `P0P2-S02G` succeeds.

## Scope

- `backend/docs/plans/p0-p2-backend-platform/**`
- `backend/Makefile`, `backend/Dockerfile`, `backend/README.md`
- `backend/ci/**`
- `backend/cmd/controlplane/**`
- `backend/internal/controlplane/**`
- `.github/workflows/backend.yml` only for backend PR validation

## Non-goals

- No modifications to `server/**`.
- No other root GitHub workflow, repository setting, or branch protection change.
- No edits to `apps/**` or `packages/**`; frontend wiring is represented by a versioned HTTP contract only.
- No automatic merge, release, or production deployment.
- No LLM deciding authoritative state transitions.
- No workflow secret, write permission, artifact publication, or external action beyond checkout and Go setup.

## Steps

### P0P2-S00 — Freeze plan

Scope: plan snapshots, pointer, and append-only journal only.

Acceptance: identifiers, base, scope, invariants, gates, risks, and rollback are explicit.

Status: implemented under v1.

### P0P2-S01 — P0 backend gates

Create a backend-local Makefile, container build, deterministic CI script, generated-code drift hook, architecture/path checks, and operator README.

Acceptance:

- one command runs format, test, race, vet, and policy checks;
- checks fail when a candidate diff includes `server/**` or paths outside this revision's exact allowlist;
- container starts the new control-plane binary;
- six-domain terminology is explained as six delivery capability domains over the existing four platform contexts.

Status: implemented; Go and container execution still require deterministic evidence.

### P0P2-S02 — P1 platform foundation

Implement workspace lifecycle, memberships and roles, actor identity, server-side authorization, audit, consistent errors, validation, pagination limits, repository contracts, SQLite persistence, injectable production SQL persistence, migrations, transactions, idempotency, and optimistic concurrency.

Acceptance:

- cross-workspace access is denied;
- owner invariants and role transitions are table-tested;
- data survives SQLite reopen;
- conflicting versions fail closed;
- audit records are immutable and workspace-scoped.

Status: implemented at `420c84ff3eacf8bfecd299f58bc55dde8f498f90`; pending CI evidence.

### P0P2-S02G — P1 deterministic CI gate

Create `.github/workflows/backend.yml` and run exactly the backend-local deterministic commands on PRs targeting `codex/multica-six-domain-baseline` when backend or workflow paths change.

Acceptance:

- workflow has read-only contents permission and no secrets;
- checkout retains enough history to compare against the PR base SHA;
- Go version is derived from `backend/go.mod`;
- workflow runs `cd backend && make check && make test-race`;
- any failure keeps `P0P2-S03` blocked;
- a success is recorded as evidence but does not self-approve or merge PR #8.

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

Run deterministic checks, index evidence, record unavailable gates, and keep Draft PR #8 targeted at `codex/multica-six-domain-baseline` until independent acceptance.

Acceptance: changed paths are restricted to the exact v2 allowlist; no unsupported gate is claimed; independent review and frontend wiring remain explicit before DoneGate.

## Deterministic verification

```bash
cd backend
make check
make test-race
make policy-check BASE_REF=codex/multica-six-domain-baseline
docker build -t goclaw-controlplane:p0p2 .
```

The GitHub workflow gate runs the first two commands with `BASE_REF` set to the PR base SHA. Docker build remains a separate handoff gate.

## Risks and controls

- Broad scope: preserve step boundaries and atomic commits.
- CI authority creep: allow exactly one root workflow path and read-only permissions.
- Base-ref ambiguity: provide the immutable PR base SHA to the policy checker.
- Existing generated contracts: do not hand-edit generated code.
- Persistence drift: keep migrations explicit, transactional where supported, and covered by reopen tests.
- Authority confusion: encode Checker/Runner separation in types and state transitions.
- Secret leakage: persist secret references only and reject raw secret fields at HTTP boundaries.
- Frontend path conflict: publish a stable API but keep frontend wiring blocked.

## Rollback

Revert commits in reverse step order. Removing `.github/workflows/backend.yml` disables the newly authorized gate without modifying runtime code. Database changes are additive and versioned; no production migration, merge, release, or deployment is performed by this plan.
