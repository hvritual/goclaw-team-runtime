# Product capability roadmap v34 — S07C independent-review remediation

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Plan-Version: `34`
- Task-Revision: `r039`
- Work-Item: `PCR-S07C`
- Exact base: `8116b79f075eb7fe972087abbeebbc76e30ef6b9`
- Blocked input tree: `d328692cc8a2724d2524c279c190d55c01286524`
- Blocked input binary patch hash: `943514fc385a78df5e987145b3340d897ff71a6f`
- Predecessor plan: `plan_v33.md`
- Predecessor plan hash: `5f596081b0aa97ffdfc4aad7aaebce997681406d474d8bcbfa1efcd2019cb8ca`
- Status: `approved-active`
- Authority: the Human Customer's confirmed continuous Release 3 direction,
  confirmed prerequisite minimal outline authority, and confirmed execution
  after the v33/r038 independent-review result

## Predecessor and activation boundary

Immutable v33/r038 preserves the complete v31 Requirement-coverage contract,
the v32 aggregate Auth test stabilization, and the v33 test-clock race repair.
Its exact candidate `8116b79f075eb7fe972087abbeebbc76e30ef6b9`
passes the recorded focused, complete backend, official race, strict frontend,
production-build, and fresh installed-acceptance gates. Fresh independent
review nevertheless returns both `SPEC BLOCK` and
`CODE/SECURITY/QUALITY BLOCK` for exactly these findings:

1. persisted Requirement content is JSON-decoded but is not revalidated against
   the full domain content invariant, so `{}` can be projected as an HTTP 200
   zero-item coverage snapshot instead of failing closed;
2. active Issue links are selected by `baseline_id` but their persisted
   `workspace_id` and `project_id` are not compared with the authorized
   baseline, allowing ownership drift to expose a foreign Issue projection;
3. the shared view ignores the coverage-query error and renders the no-coverage
   empty state after an HTTP or strict-schema failure;
4. the constant coverage query graph has no executable query-count-bound
   assertion even though its static SQL graph is bounded.

r038 is review-blocked and v33 remains immutable. r039 starts only from that
exact reviewed candidate and may close only these findings. PCR-S07D, Release 3
completion, and every unrelated behavior remain inactive.

## Frozen remediation contract

- Add assertion-first corrupted-content cases covering current and effective
  persisted revisions. A valid JSON document that violates the complete domain
  invariant must return an error from the repository and a safe typed
  `internal_error` from the real Canonical HTTP composition; no count, item,
  Issue, or content fallback may be returned.
- Expose and reuse one domain-owned full content normalization/validation
  function for persisted reads. Both the baseline current-revision decoder and
  the shared revision scanner must use it, preserving the already-normalized
  valid representation.
- Add an assertion-first ownership-drift case with a real foreign Workspace and
  Project Issue. The single per-snapshot Issue-projection query must return
  active link ownership, compare it with the authorized baseline before
  projecting any Issue, and fail closed on mismatch. Deleted Issues remain
  excluded without weakening ownership validation.
- Add an executable SQL-driver query counter proving that the maximum
  current-plus-effective coverage read query count is bounded and unchanged
  when the traceable item set grows. It must observe the real repository and
  real SQLite driver; production query instrumentation and per-item queries are
  forbidden.
- Add an assertion-first shared-view query-failure case. Coverage errors must
  render a localized safe error state and must not render the no-coverage empty
  state or stale/partial coverage. Existing valid and true no-baseline states
  remain unchanged.

No route, response schema, coverage stage, lifecycle state, database schema,
permission, feature flag, stored cache, Issue mutation, generated contract, or
installed capability is added.

## Exact writable boundary

Domain validation and backend coverage behavior/tests:

- `backend/internal/modules/workspace/internal/domain/requirement/baseline.go`;
- `backend/internal/modules/workspace/internal/infrastructure/sqlite/project_requirement_repository.go`;
- `backend/internal/modules/workspace/internal/infrastructure/sqlite/project_requirement_repository_test.go`;
- `backend/internal/modules/workspace/project_requirement_composition_test.go`.

Shared view and locale behavior/tests:

- `packages/views/projects/components/project-requirement-baseline.tsx`;
- `packages/views/projects/components/project-requirement-baseline.test.tsx`;
- `packages/views/locales/en/projects.json`;
- `packages/views/locales/ja/projects.json`;
- `packages/views/locales/ko/projects.json`;
- `packages/views/locales/zh-Hans/projects.json`.

Governance:

- `backend/docs/plans/product-capability-roadmap/plan.md`;
- `backend/docs/plans/product-capability-roadmap/plan_v34.md`;
- `backend/docs/plans/product-capability-roadmap/story-map.md`;
- `backend/docs/plans/product-capability-roadmap/task-register.md`;
- `backend/docs/plans/product-capability-roadmap/journal.md`.

Immutable v31-v33, every migration, HTTP production handler, Core contract,
Bootstrap/Auth path, generated protobuf, original dirty path, legacy backend
tree, and every `server/**` path are read-only. A necessary path outside this
exact list stops r039 and requires another immutable successor plan.

## Ordered execution

1. R39.1 — Freeze this successor from exact base `8116b79f`, rename the isolated
   branch to `codex/release3-s07c-r039`, mark r038 review-blocked, and commit
   only the five governance activation paths with one continuous nine-field
   trailer block.
2. R39.2 — Capture real RED failures for valid-JSON invalid domain content and
   cross-project ownership drift, add the real-driver constant-query proof,
   then GREEN only the domain read validation and single-query ownership guard.
3. R39.3 — Capture the shared-view query-error RED, then GREEN only the safe
   localized error state and pass the changed locale contract.
4. R39.4 — Run focused domain/repository/composition/Views/locale checks,
   complete Workspace and backend `make check`, the official changed-package
   race command, full Core/Views and strict type/lint gates, root forced gates,
   production Web build, and fresh targeted installed acceptance against real
   Canonical HTTP plus production Web. Every unrelated aggregate failure stays
   disclosed as NON-PASS.
5. R39.5 — Freeze one exact candidate; verify v31-v34 and policy hashes, all
   nine trailers on every r039 commit, exact path scope, zero `server/**` and
   generated paths, clean isolated worktree, original dirty-tree preservation,
   process cleanup, and obtain fresh independent `SPEC PASS` plus
   `CODE/SECURITY/QUALITY PASS`.

## Deterministic and installed acceptance

- Current and effective revision corruption, including `{}`, invalid keys,
  duplicate keys, oversized content, and otherwise valid JSON that violates the
  domain invariant, fails closed without partial coverage.
- A foreign Workspace/Project active link fails before any foreign Issue detail
  is returned. Valid links and deleted-Issue exclusion retain their exact
  current/effective semantics.
- The real-driver counter proves the maximum current-plus-effective repository
  read uses a constant query count independent of one versus one hundred
  traceable items and stays within the frozen authority/baseline plus two
  projection-query graph.
- A rejected HTTP/strict-schema coverage query displays only the localized safe
  error state; a successful no-baseline response alone displays the empty state.
- Fresh targeted installed acceptance proves a valid projection still renders,
  corrupted content returns typed internal failure, ownership drift returns no
  foreign detail, and the production shared view exposes the safe error state.
- Only fresh independent review returning both required PASS decisions may
  close r039/PCR-S07C and authorize the PCR-S07D successor plan.

## Explicit exclusions and stop conditions

PCR-S07D Retrospectives, Release 3 completion, S10, migrations/indexes,
stored coverage caches, historical Issue snapshots, new permissions/flags,
Core/public-schema changes, generated protobufs, Bootstrap/Auth behavior,
unrelated Issue/Input behavior, push, merge, deployment, external service
calls, original dirty paths, legacy backend writes, and all `server/**` changes
are excluded.

Stop before closure on any invalid persisted content returning coverage,
foreign link ownership returning Issue detail, coverage query error rendering
the empty state, query count growing with items/Issues, a second per-item query,
partial response, authority leak, hidden test failure, missing/duplicate
trailer, scope drift, original dirty-path overlap, unclosed process,
`server/**` or generated change, or either independent-review BLOCK. Any repair
outside this exact boundary requires a new immutable plan; v34 is never amended
after activation.
