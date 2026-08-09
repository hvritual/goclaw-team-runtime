---
schema: goclaw.wave-evidence/v1
wave_id: STG1-W01
wave_revision: 1
issue_id: STG1-ISSUE-001
task_id: STG1-DELIVERY-001
date: 2026-08-09
status: partial
---

# STG1-W01 verification evidence r001

## Implemented scope

- Append-only `DeliveryEvent` journal with a global sequence/hash chain,
  per-stream version, project revision CAS, project-scoped command idempotency,
  command-hash-to-receipt binding, deterministic reducer, startup replay, and
  project-scoped integrity RPC.
- Request → IntentContract → SolutionSpec → Scenario/Capacity/Risk/Cost review
  → FrozenPlan → traceable WorkItem, plus immutable-post-freeze ChangeIntent.
- Separate Defect and Risk aggregates, state machines, response decisions,
  WorkItem/evidence links, active-member ownership checks, independent risk
  response decisions, and evidence gates.
- Frozen WorkItem bundles reject invalid component/dependency references,
  self-dependencies, dependency cycles, invalid verification commands, and
  missing evidence requirements before any event is appended.
- Project-scoped Gateway RPCs: `delivery.command`, `delivery.projection`,
  `delivery.events`, and `delivery.integrity`.
- Team Control UI pages use those real RPCs and preserve explicit loading,
  empty, conflict/denied, and server-error states. No mock/fallback fact store was
  introduced.

## Deterministic checks

| Check | Result | Evidence |
|---|---|---|
| UI contract tests | PASS | `cd ui && npm test`: 8 tests passed, 0 failed |
| UI production build | PASS | `cd ui && npm run build`: TypeScript and Vite build completed |
| Go unit tests | BLOCKED | No `go` executable is available in the execution environment |
| Go race detector | BLOCKED | Requires the unavailable Go toolchain |
| Go syntax/format pass | PASS (limited) | All changed Go files parsed and formatted by `@wasm-fmt/gofmt` 0.7.3; this does not replace compiler, vet, race, or unit checks |
| Go vet | BLOCKED | Requires the unavailable Go toolchain |
| Desktop/mobile browser QA | BLOCKED | Cloud browser rejected the local preview URL with `net::ERR_BLOCKED_BY_CLIENT`; no local Playwright binary is installed |
| Independent code/security/docs review | PENDING | Must be performed by an independent reviewer after deterministic Go checks pass |

## Covered Go test contracts (not executed in this environment)

- Full Request/Intent/Solution/four-review/Freeze/ChangeIntent replay flow.
- Full Defect reproduction/RCA/fix/verification/release/close flow.
- Full Risk assessment/response/evidence/review/close flow.
- Idempotent command replay, stale CAS rejection, viewer write denial, invalid
  owner denial, project-scoped integrity reporting, independent risk response,
  and corrupted hash rejection on reopen.
- Gateway authenticated-actor binding, projection/events/integrity results, and
  cross-project read denial.

## Gate decision

The implementation candidate is ready for CI and independent review, but this
evidence is intentionally `partial`. `STG1-W01` remains `active`; it must not be
marked complete until native Go test/race/vet/gofmt, rendered desktop and mobile
QA, and independent code/security/docs review all produce indexed
passing evidence with P0=0/P1=0.
