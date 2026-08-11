# P0-P2 Backend Platform Plan v3

- Plan-ID: `P0P2-BACKEND-PLATFORM-001`
- Version: `v3`
- Status: `approved-for-execution`
- Supersedes: `plan_v2.md`
- Approval basis: user authorized execution through P2 and backend CI
- Base commit: `1c0054ada767bf12d2efb3b0f7869dcfb8c91a4b`
- Branch: `agent/p0-p2-backend-platform-001`
- Project-ID: `goclaw-team-runtime`
- Task-ID: `P0P2-BACKEND-PLATFORM-001`
- Task-Revision: `r003`
- Policy bundle: `backend/AGENTS.md@ffe45b83ef3884d9b8bf66e6f7994b3a43d4f86e`

## Goal

Complete P0, P1, and P2 on the canonical `backend/` while preserving server-side authority, durable event history, independent acceptance, and exact path governance.

## Hard invariants

1. Backend product and test changes remain under `backend/**`; `.github/workflows/backend.yml` is the sole non-backend path exception.
2. `server/**` is permanently read-only.
3. A human is required to bootstrap a workspace Owner; Agent identities can never own, administer, review, or accept.
4. Permission evaluation is fail-closed and unknown permissions are denied.
5. Every workspace retains at least one active human Owner under concurrent writes.
6. Runner, model output, Audit, and generic Records are not authoritative acceptance evidence.
7. Kernel event, command, head, and idempotency state is durable and transactionally consistent.
8. Only typed kernel commands may append governed events; raw append is not exposed to HTTP or Runner.
9. Replay validates chain integrity and fails closed on unknown event versions or types.
10. DoneGate requires deterministic Checker results, immutable evidence digests, no graph blockers, and an independent human acceptor.
11. Failed or unavailable gates remain incomplete; no implementation self-accepts or merges.

## Exact scope

- `backend/docs/plans/p0-p2-backend-platform/**`
- `backend/Makefile`, `backend/Dockerfile`, `backend/README.md`
- `backend/ci/**`
- `backend/cmd/controlplane/**`
- `backend/internal/controlplane/**`
- `.github/workflows/backend.yml`

## Non-goals

- No `server/**`, `apps/**`, or `packages/**` changes.
- No additional workflow, repository setting, branch protection, secret, deployment, automatic merge, or release.
- No model or Runner authority to resolve findings, publish knowledge, pass checks, or accept work.

## Ordered steps

### P0P2-S00/S01/S02/S02G — Completed foundation

Plan freeze, backend-local gates, P1 workspace foundation, and Backend Actions Run `31486953485` are complete. Independent final acceptance remains pending.

### P0P2-S02R — P1 invariant repair

Reproduce and repair Agent-owner escalation, unknown-permission fail-open, infrastructure-error masking, and concurrent last-Owner removal. Add deterministic regression tests.

Acceptance: Agent workspace bootstrap is denied; unknown permissions are denied for every role; infrastructure errors are preserved; owner mutations are serialized and the database rejects a zero-owner result.

### P0P2-S03 — Durable Delivery Kernel

Add dedicated durable command, project-head, session-event, evidence/check, and projection storage. Implement command request hashing, original-result replay, head CAS, project sequencing, domain-separated SHA-256 chain, typed events, deterministic replay, Work Graph, Evidence index, Checker results, and DoneGate.

Acceptance: SQLite reopen preserves chain and idempotency; same command/same request replays the original result; same command/different request conflicts; concurrent same-head commands produce one success; tampering and unknown events fail closed; graph dependency cycles and blockers are detected; Runner evidence alone cannot pass; only an independent human with current deterministic checks and immutable evidence can accept.

### P0P2-S04 — Requirement flow

Implement typed Request, Intent clarification/decision, Solution/ADR, four independent reviews, Freeze, ChangeIntent, Task, traceability, and coverage projections on kernel commands.

Acceptance: only a fully reviewed current revision freezes and creates tasks; material change creates a new revision and invalidates stale reviews.

### P0P2-S05 — Quality flow

Implement Defect and Risk models, reproduction evidence, severity/probability/impact, response ownership, creator/approver separation, review due dates, work links, verification, and close gates.

Acceptance: unverified defects and unreviewed or overdue risks cannot close.

### P0P2-S06 — Review and knowledge flow

Implement structured ReviewFinding, deterministic/model-review separation, KnowledgeCandidate source/evidence links, deduplication, human review, version publication, invalidation, rollback, and evaluation metadata.

Acceptance: model output cannot resolve findings or publish knowledge without deterministic and human gates.

### P0P2-S07 — Execution flow

Implement Run queue, claim, lease, heartbeat, cancel, timeout, bounded retry, isolated workspace reference, sanitized secret references, evidence return, and human termination.

Acceptance: Runner cannot accept; expired leases are safely reclaimed; retries and returns are idempotent; raw secrets are rejected.

### P0P2-S08 — Team Control backend API

Expose typed JSON endpoints for workspace, requirement, quality, review, knowledge, execution, evidence, replay, Work Graph, and DoneGate projections.

Acceptance: handler tests cover valid, invalid, denied, not-found, conflict, workspace isolation, and unavailable-state responses.

### P0P2-S09 — Verification and handoff

Run Backend CI after every atomic step, index immutable run evidence, keep PR #8 Draft, and record Docker/frontend/independent-review residual gates.

## Deterministic verification

```bash
cd backend
make check
make test-race
make policy-check BASE_REF=codex/multica-six-domain-baseline
docker build -t goclaw-controlplane:p0p2 .
```

GitHub Actions runs the first two commands. Docker, frontend wiring, and independent acceptance remain separate gates.

## Risks and controls

- Authority escalation: typed command services and explicit human-only acceptance.
- Split event/projection state: one SQL transaction for command, head, event, and authoritative result; projections are rebuildable.
- Hash ambiguity: domain-separated, length-prefixed fixed-order hash material.
- Secret leakage: immutable artifact references and SHA-256 only; reject credential-shaped fields and signed URLs.
- Database concurrency: CAS plus unique constraints and transaction-scoped workspace locking.
- Scope creep: exact path allowlist and one active step.

## Rollback

Revert atomic step commits in reverse order. Schema changes are additive; unknown versions fail closed. Removing the workflow disables CI only. No production migration, merge, release, or deployment occurs in this plan.
