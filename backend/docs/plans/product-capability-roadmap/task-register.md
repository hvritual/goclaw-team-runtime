# Product Capability Roadmap Task Register

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Plan-Version: `v1`
- Registry status: `PCR-S00 complete; PCR-S01A active`
- Registry revision: `r002`
- Updated: `2026-08-16`

## Frozen policy bundle

| Policy source | SHA256 |
| --- | --- |
| `CLAUDE.md` | `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646` |
| `backend/AGENTS.md` | `fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209` |
| `plan_v1.md` | `4351025652e88083b8ceb25a72488e9d568a445a5f0e3a8793650393086ad0b5` |

Changing any policy hash invalidates a queued product task until it is reviewed
and re-frozen. `server/**` remains excluded regardless of hash changes.

## PCR-001-S00-R1

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-S00-R1`
- Task-Revision: `r001`
- Work-Item: `PCR-S00`
- Status: `complete-by-customer-direction`
- Assignee: `Codex primary agent`
- Independent reviewer: `not assigned; retained as program acceptance debt`
- Base commit: `45213820fade7f61294d2287e063bf19fbd015ee`
- Write boundary: `backend/docs/plans/product-capability-roadmap/**` only
- Product-code authority: none
- Acceptance source: `plan_v1.md` PCR-S00
- Verification:
  - Markdown relative links resolve.
  - `git diff --check -- backend/docs/plans/product-capability-roadmap` passes.
  - HEAD and policy hashes match the frozen values.
  - Worktree exclusions remain present and unmodified.

Expected outputs:

- approved plan pointer and immutable v1 snapshot;
- `contract-freeze_v1.md`;
- this task register;
- append-only approval, drift, and gate evidence in `journal.md`.

## PCR-001-S01A-R2

- Project-ID: `PRODUCT-CAPABILITY-ROADMAP`
- Task-ID: `PCR-001-S01A-R2`
- Task-Revision: `r002`
- Work-Item: `PCR-S01A`
- Title: `Canonical capability authorization and accurate feature flags`
- Status: `active`
- Assignee: `Codex primary agent`
- Independent reviewer: `must be assigned before acceptance`
- Base commit: `387eb64264b42a6e23959f95f4c2c4eb15b45c91`
- Prior blocking gate: cleared by the Human Customer statement that C9 passed
  and the explicit direction to return to PCR-S01A.
- Evidence qualification: the local v11 dependency/browser rerun remained
  incomplete and is not represented as a technical PASS.
- Overlap audit: v11 changed Issue HTTP filtering and Canonical plan evidence;
  PCR-S01A r002 does not authorize Issue HTTP paths and has no product overlap.

### Exact allowed product paths after resume

Only the following files are authorized for task revision r002. If architecture
discovery proves another path is necessary, stop and create r003 before editing.

- `backend/internal/modules/workspace/contract/capability_authorization.go`
- `backend/internal/modules/workspace/contract/capability_authorization_test.go`
- `backend/internal/bootstrap/sqlite.go`
- `backend/internal/bootstrap/sqlite_runtime_test.go`
- `backend/internal/bootstrap/runtime.go`
- `backend/internal/bootstrap/runtime_test.go`
- `backend/docs/plans/product-capability-roadmap/journal.md`
- `backend/docs/plans/product-capability-roadmap/task-register.md`

The task may add the two new contract files above. It may make narrow edits to
the named bootstrap/runtime files. No migration, API, frontend, Control Plane,
legacy server, or unrelated refactor is authorized by S01A r002.

### Frozen acceptance

1. Every roadmap action has a named authorization constant or is explicitly
   marked unavailable.
2. Member and agent default matrices fail closed and are table-tested.
3. `/api/config` flags remain false for every uninstalled roadmap capability.
4. Enabling a flag requires an injected installed provider; missing providers
   fail readiness or keep the flag false as frozen per capability.
5. Existing Issue, project, pin, auth, attachment, and realtime behavior remains
   unchanged.
6. No request can gain permission from a client-supplied actor type or workspace.

### Frozen deterministic verification after resume

```text
cd backend && go test ./internal/modules/workspace/contract ./internal/bootstrap
cd backend && go test -race ./internal/modules/workspace/contract ./internal/bootstrap
cd backend && make check
```

On Windows, loader exit `0xc0000139` is recorded as an environment limitation,
not a passing race result. A non-Windows race result or CI equivalent remains
required before technical acceptance.

## Queue rules

- No task after PCR-S01A may be enqueued until S01A is accepted and the next
  task revision freezes exact paths and checks.
- A task status in this file is not a substitute for the active-step pointer in
  `plan.md`; both must agree.
- A base, plan hash, policy hash, or allowed-path mismatch stops execution.
- Completion by the assignee is not independent acceptance.
- Customer Acceptance is not delegated to this register.
