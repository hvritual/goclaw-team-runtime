# P0-P2 Backend Platform Plan v4

- Plan-ID: `P0P2-BACKEND-PLATFORM-001`
- Version: `v4`
- Status: `approved-for-execution`
- Supersedes: `plan_v3.md`
- Approval basis: user authorized execution through P2 and backend CI
- Base commit: `1c0054ada767bf12d2efb3b0f7869dcfb8c91a4b`
- Branch: `agent/p0-p2-backend-platform-001`
- Task-Revision: `r004`

## Goal and invariants

Complete the typed P2 domain layer and Team Control backend API on the durable Delivery Kernel. All v3 authority, persistence, path, human-acceptance, secret, replay, CI, and rollback invariants remain binding without relaxation.

## Exact scope

- `backend/internal/controlplane/kernel_types.go`
- `backend/internal/controlplane/p2_flows.go`
- `backend/internal/controlplane/p2_flows_test.go`
- `backend/internal/controlplane/http.go`
- `backend/internal/controlplane/http_test.go`
- `backend/cmd/controlplane/main.go`
- `backend/docs/plans/p0-p2-backend-platform/**`
- `.github/workflows/backend.yml` remains the only non-backend exception and is not changed by this revision.

## Consolidated steps

### P0P2-S04X — P2 typed domain flows

Implement Requirement/Intent/Solution/four-review/Freeze/ChangeIntent/Task traceability; Defect/Risk/verification/close gates; ReviewFinding; KnowledgeCandidate/dedup/publication/invalidation; and Run queue/claim/lease/heartbeat/cancel/retry/evidence return. These capabilities share one typed orchestration service and the same kernel authority boundary, so they are verified as one atomic P2 domain slice.

Acceptance: material requirement changes invalidate stale revision checks; unverified quality items cannot close; model findings and knowledge cannot self-resolve/publish; raw secrets are rejected; leases are exclusive and bounded; Runner cannot accept; all state survives replay.

### P0P2-S08 — Team Control backend API

Expose typed JSON commands and projections with server-derived Actor identity, stable Problem responses, workspace isolation, and handler coverage for success, invalid, denied, not found, conflict, and invariant failures.

### P0P2-S09 — Verification and handoff

Run `make check` and `make test-race`, index Actions evidence, keep PR #8 Draft, and leave Docker/frontend/independent acceptance explicit.

## Non-goals and rollback

No `server/**`, frontend, deployment, merge, release, new workflow, secrets, or LLM authority. Revert S08 then S04X; schema remains additive and unknown state fails closed.
