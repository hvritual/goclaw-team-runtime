# Canonical SQLite runtime cutover — repair plan v11

- Plan-ID: `canonical-sqlite-runtime-cutover`
- Version: `11`
- Status: `approved`
- Approval source: Human Customer confirmation `批准` dated `2026-08-16`
- Supersedes: `plan_v10.md`
- Base commit: `45213820fade7f61294d2287e063bf19fbd015ee`
- Branch and integration target: `codex/multica-six-domain-baseline`
- Active step: `M1-S7-C9-LIST-RED`
- Policy bundle: `backend-v1`
- XP mode: strict TDD; maximum active stories: one

## Purpose

Repair the authenticated C9 clean-candidate failure in which two concurrent
status buckets returned `GET /api/issues -> 500`, then repeat the exact
clean-candidate deterministic, Chrome, restart, rollback and independent-review
gates. C9 and the milestone remain unaccepted until those gates and explicit
Customer Acceptance complete.

The retained v10 failure is sufficient reproduction evidence to authorize this
repair plan:

- environment: detached clean candidate
  `f63a9791a5f7522d7fcbc7ecea059bda58889c30`;
- role/workspace: authenticated fixture member in `canonical-fixture`;
- action: enter the complete C9 Issue detail while shared Issue queries fetch
  status buckets concurrently;
- expected: every enabled same-origin Canonical request succeeds;
- actual: `status=blocked` and `status=cancelled` each returned 500 with
  `issue operation failed`, after about 1.38 seconds;
- retained context: v10 journal and clean-candidate Playwright error context;
- control: a later direct authenticated GET returned 200, establishing a
  transient concurrency/load symptom rather than a permanently invalid row.

## Repair hypothesis

The Web client intentionally fetches seven lifecycle status buckets in
parallel. The installed HTTP handler currently discards each singular `status`
filter when it calls the existing `ListIssues` use case, causing every request
to query and decode the full workspace and filter only afterward. Under the C9
detail workload this creates avoidable connection and scan amplification; the
last buckets were the observed failures.

The first deterministic RED must prove that a singular status/priority/project/
actor filter is not propagated to the existing service contract. GREEN may only
push down a filter whose semantics are identical to the current in-memory
filter. Multi-value, OR, no-value, metadata, and presentation filters remain in
the handler. The exact C9 browser rerun is the required integration proof of the
hypothesis.

If the deterministic RED does not fail for the expected missing pushdown, or if
GREEN does not eliminate the browser 500, stop. Do not add a retry, raise a
timeout, reduce client concurrency, or change SQLite pool/global journal mode
without a new reproduced cause and `plan_v12.md`.

## Frozen product contract

- No public API, response shape, sort order, filter semantics, capability flag,
  authentication behavior, workspace isolation, or UI behavior changes.
- The same `GET /api/issues` request returns the same ordered rows and `total`;
  only safe filtering moves closer to the authoritative repository.
- Invalid status/priority values retain their existing error behavior.
- `status`, `priority`, `project_id`, `parent_issue_id`, and complete actor-pair
  filters may be pushed down only when their current request semantics are a
  single exact value.
- `statuses`, `priorities`, multi-actor filters, `include_no_*`, text, date,
  metadata, hierarchy, and mixed OR semantics remain post-query filters unless
  separately proven equivalent.
- Context cancellation and real repository errors remain errors; the repair
  does not fabricate empty success.
- Invitations remain an explicit capability-off non-goal under the frozen C9
  parity contract and are not changed by v11.

## Authorized write boundary

- `backend/internal/modules/workspace/internal/interfaces/http/issue_read.go`
- `backend/internal/modules/workspace/internal/interfaces/http/issue_read_test.go`
  for focused handler-contract RED/GREEN coverage if added.
- `backend/internal/bootstrap/issue_read_runtime_test.go` for installed-runtime
  concurrency and response-parity coverage.
- `e2e/canonical-runtime.spec.ts` only if a reproduced evidence-harness defect,
  not product behavior, requires a test-first correction.
- `scripts/runtime-selector*` and `scripts/canonical-runtime-verifier*` only for
  a newly reproduced evidence defect.
- `backend/docs/plans/canonical-sqlite-runtime-cutover/**` for plan pointers,
  journal, evidence, review and acceptance state.
- `backend/docs/plans/product-capability-roadmap/{plan.md,task-register.md,journal.md}`
  only to pause/resume the dependent roadmap gate.

No repository, use-case, SQLite pool, migration, schema, frontend/Core/view,
notification, roadmap feature, or legacy-server behavior change is authorized.
`server/**` remains permanently read-only.

## Ordered TDD sequence

1. `M1-S7-C9-LIST-RED`: add the smallest focused test showing the handler calls
   the existing service without the safe singular filters. Watch it fail for
   the missing filter, not a setup error. Add an installed-runtime parallel
   status-bucket parity test if needed to bind the public route.
2. `M1-S7-C9-LIST-GREEN`: minimally build the existing
   `contract.ListIssuesRequest` from safe singular filters before calling the
   service. Keep post-filtering so response semantics remain unchanged. Watch
   focused RED become GREEN and run the existing Issue read suite.
3. `M1-S7-C9-LIST-VERIFY`: run focused Workspace/Bootstrap tests, full Backend
   deterministic gates, TypeScript gates, selector/verifier tests, and scope
   checks on the exact repair candidate.
4. `M1-S7-C9-INTEGRATE`: create a fresh detached clean candidate, start only its
   verified Canonical/Web tree, seed a new fixture, and run the existing raw
   network C9 pre/post-restart journey. Any actionable local failure stops the
   plan without retrying around it.
5. `M1-S7-C9-ROLLBACK`: quiesce the owned runtime, prove retained database,
   sidecar, object and log hashes/readback, restore the selector safely, and
   leave local `3000/8000/8080/9000` quiescent.
6. `M1-S7-C9-REVIEW`: obtain independent CODE/SECURITY and SPEC/EVIDENCE review
   of the exact candidate with no unresolved P0-P2.
7. `M1-S7-C9-ACCEPT`: request explicit Human Customer milestone acceptance.
   Technical success never records this decision automatically.

## Acceptance criteria

1. The focused RED fails because safe singular filters are absent from the
   service request and passes after the minimal handler change.
2. Response-parity tests prove filtering, total, ordering, invalid input,
   workspace isolation, and multi-value semantics are unchanged.
3. The exact clean candidate contains only v11-authorized changes, no
   `server/**`, and none of the primary worktree's unrelated dirty paths.
4. All deterministic checks pass. Windows race exit `0xc0000139` is an
   environment limitation, never a pass.
5. Installed Chrome completes C9 before and after retained-database restart.
   The raw trace records every request/response/requestfailed event before
   filtering, contains no HTTP or WebSocket `:8080` traffic, and contains no
   actionable enabled-capability failure.
6. No `GET /api/issues` response is 500 in the C9 trace. Every lifecycle bucket
   returns the same semantic result as direct authenticated readback.
7. Rollback/readback preserves retained artifacts exactly, independent reviews
   return no P0-P2, and the Customer explicitly accepts before milestone status
   changes.

## Deterministic verification

From `backend/`:

```powershell
gofmt -d internal/modules/workspace/internal/interfaces/http/issue_read.go internal/modules/workspace/internal/interfaces/http/issue_read_test.go internal/bootstrap/issue_read_runtime_test.go
go test ./internal/modules/workspace/internal/interfaces/http ./internal/bootstrap -run 'IssueRead|IssueList' -count=1
go test ./internal/modules/workspace ./internal/bootstrap ./cmd/server -count=1
go test ./... -count=1
go vet ./...
go mod verify
```

From repository root:

```powershell
pnpm --filter @multica/core test
pnpm --filter @multica/core typecheck
pnpm --filter @multica/core lint
pnpm --filter @multica/views test -- <focused Issue tests>
pnpm --filter @multica/views typecheck
pnpm --filter @multica/web typecheck
node --test scripts/runtime-selector.test.mjs scripts/canonical-runtime-verifier.test.mjs
git diff --check
git status --porcelain -- server
```

Run the existing tracked C9 trace regressions and clean installed-Chrome C9
pre/post-restart journey on the exact detached candidate.

## Risks and stop conditions

- Filter pushdown could change mixed-filter semantics. Retain post-filtering and
  prove parity before integration.
- A concurrency symptom may have another cause. Browser failure after GREEN
  invalidates this hypothesis and requires a new version, not a broader patch.
- Do not add global retries, increase `busy_timeout`, change `SetMaxOpenConns`,
  enable WAL, or reduce the client's seven status requests under v11.
- Stop on any unauthorized path, policy/base drift, unowned listener, failed
  deterministic gate, new local route failure, retained-data mismatch, or
  unresolved independent P0-P2.
- Preserve every failure artifact. Do not rerun a consumed acceptance gate to
  erase a failure.

## Rollback

Revert only the handler pushdown and its tests, leaving schemas and retained data
unchanged. For live evidence, stop only the manifest-owned candidate tree,
quiesce ports, hash retained artifacts, restore the prior selector choice, and
verify readback. Never delete retained evidence or modify `server/**`.
