# Product capability roadmap v32 — S07C aggregate-gate remediation

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Plan-Version: `32`
- Task-Revision: `r037`
- Work-Item: `PCR-S07C`
- Exact base: `5d769058d289300e6b8851a5885cca183d5f9be9`
- Predecessor plan: `plan_v31.md`
- Predecessor plan hash: `422f6ea3ab573f77c7c718105b04c06f132e1b4fd9c422bc06e8ee4433e2a081`
- Predecessor closure candidate: `cd94396093ea73f3f9434fed7410036ae61170ab`
- Status: `approved-active`
- Authority: the Human Customer's confirmed continuous Release 3 direction,
  confirmed prerequisite minimal outline authority, and confirmed execution

## Predecessor and stop boundary

Immutable v31/r036 freezes the complete PCR-S07C Requirement-coverage product
contract. Commits `90acfecd` and `5d769058` install its bounded derived backend
read, strict HTTP/Core contract, realtime invalidation, shared current/effective
view, four locales, and assertion-first tests without a migration, stored cache,
new permission, new flag, Issue mutation, generated artifact, or `server/**`
change.

Focused backend coverage tests, the complete Workspace tree, Core 628/628, Views
1683/1683, locale parity, changed-file lint, and Core/Views typechecks pass. The
first fresh backend `make check` remains NON-PASS after 351.9 seconds because
`TestLocalAuthProjectsPersistedOnboardedAtAcrossRestart` returns HTTP 500 while
creating its final local user. The exact test then passes alone, the complete
Auth package passes, and the exact test passes 10/10 consecutively.

After clearing only Go test-result cache, the unchanged second complete
`make check` reproduces the same test, step, response, and failure after 270.9
seconds. A detached diagnostic worktree preserving the candidate byte-for-byte
adds only temporary safe stderr visibility and reproduces the aggregate failure
again: the hidden service error is `context deadline exceeded`. Kratos v3's
`NewServer()` default is a one-second per-request timeout. Under full package
parallel load, the persistence/restart test consumes that test-server deadline;
no SQLite lock, disk-full state, production Auth error, or S07C behavior is
implicated. The diagnostic worktree is removed and no diagnostic source enters
this plan.

Because `backend/internal/modules/auth/local_auth_http_test.go` is outside the
v31 path list, v31's explicit stop condition is satisfied. r036 cannot close and
v31 remains immutable. This successor authorizes only the smallest test-harness
stabilization plus completion of the unchanged S07C gates.

## Frozen remediation

The two `kratoshttp.NewServer()` constructor sites inside
`TestLocalAuthProjectsPersistedOnboardedAtAcrossRestart` must use an explicit,
finite ten-second server timeout. The loop site creates two restart instances;
the final site creates the new-user instance. No production handler, service,
repository, timeout, database, migration, request, response, or authorization
behavior changes.

Ten seconds is bounded and test-local. It removes dependency on Kratos's
incidental one-second default while retaining hang detection. Disabling the
timeout, sleeping, retrying the HTTP request, serializing the repository's full
test command, increasing SQLite busy timeout, weakening assertions, skipping
the test, or classifying either complete `make check` failure as PASS is not
allowed.

The complete v31 coverage semantics remain frozen without amendment:

- fixed traceable item order is `goals`, `in_scope`, `constraints`, then
  `acceptance_criteria`;
- exact stages are `unlinked`, `linked`, `implemented`, and `accepted`;
- all linked current Issues must satisfy the next stage;
- snapshot content and links are revision-relative, while Issue status and the
  latest acceptance conclusion are current canonical projections;
- a later conditional or rejected conclusion revokes accepted coverage;
- deleted Issues do not contribute, retired Requirements remain readable, and
  outline links never count as coverage;
- coverage is derived in a bounded consistent read and never stored or inferred
  by the client.

## Exact writable boundary

Test-harness remediation:

- `backend/internal/modules/auth/local_auth_http_test.go`.

Retained v31 backend product/test paths, writable only if verification exposes a
S07C defect within the frozen contract:

- `backend/internal/modules/workspace/contract/project_requirement.go`;
- `backend/internal/modules/workspace/internal/application/project_requirement.go`;
- `backend/internal/modules/workspace/internal/application/project_requirement_test.go`;
- `backend/internal/modules/workspace/internal/infrastructure/sqlite/project_requirement_repository.go`;
- `backend/internal/modules/workspace/internal/infrastructure/sqlite/project_requirement_repository_test.go`;
- `backend/internal/modules/workspace/internal/interfaces/http/project_requirement.go`;
- `backend/internal/modules/workspace/internal/interfaces/http/project_requirement_test.go`;
- `backend/internal/modules/workspace/project_requirement_composition_test.go`.

Retained v31 strict Core/realtime paths under the same condition:

- `packages/core/types/project-requirement.ts`;
- `packages/core/types/index.ts`;
- `packages/core/project-requirements/schema.ts`;
- `packages/core/project-requirements/queries.ts`;
- `packages/core/project-requirements/queries.test.tsx`;
- `packages/core/api/client.ts`;
- `packages/core/api/project-requirement-schema.test.ts`;
- `packages/core/realtime/use-realtime-sync.ts`;
- `packages/core/realtime/use-realtime-sync-ws-instance.test.tsx`.

Retained v31 shared-view/locale paths under the same condition:

- `packages/views/projects/components/project-requirement-baseline.tsx`;
- `packages/views/projects/components/project-requirement-baseline.test.tsx`;
- `packages/views/locales/en/projects.json`;
- `packages/views/locales/ja/projects.json`;
- `packages/views/locales/ko/projects.json`;
- `packages/views/locales/zh-Hans/projects.json`.

Governance:

- `backend/docs/plans/product-capability-roadmap/plan.md`;
- `backend/docs/plans/product-capability-roadmap/plan_v32.md`;
- `backend/docs/plans/product-capability-roadmap/story-map.md`;
- `backend/docs/plans/product-capability-roadmap/task-register.md`;
- `backend/docs/plans/product-capability-roadmap/journal.md`.

Immutable v31, every other Auth path, migration, Issue production mutation,
capability flag, generated protobuf, original dirty path, legacy backend tree,
and every `server/**` path are read-only. A necessary path outside this exact
list stops r037 and requires another immutable successor plan.

## Ordered execution

1. R37.1 — Freeze this plan from exact base `5d769058`, rename the isolated
   branch to `codex/release3-s07c-r037`, and commit only the five governance
   activation paths with one continuous nine-field trailer block.
2. R37.2 — Retain both complete-check failures plus the isolated and 10/10
   evidence as RED/diagnosis, set only the two test-server constructor sites to
   the finite ten-second timeout, then prove the exact Auth test, complete Auth
   package, and aggregate backend check GREEN without production changes.
3. R37.3 — Re-run focused S07C suites, complete Workspace, backend `make check`,
   official seven-package `make test-race`, strict frontend checks, full Core
   and Views tests, root typecheck/test, production Web build, and fresh
   installed acceptance against real Canonical HTTP plus production Web.
4. R37.4 — Freeze one exact candidate; verify v31/v32 and policy hashes, all
   nine trailers on every r037 commit, exact path scope, zero `server/**` and
   generated paths, clean isolated worktree, original dirty-tree preservation,
   process cleanup, and obtain fresh independent `SPEC PASS` plus
   `CODE/SECURITY/QUALITY PASS`.

## Deterministic and installed acceptance

- The stabilized Auth restart test passes without retry and preserves every
  existing persistence, response, and identity assertion. The complete Auth
  package and a fresh complete backend `make check` pass under normal repository
  parallelism; the two prior failures remain disclosed as NON-PASS evidence.
- All v31 focused coverage, strict Core, invalidation/realtime, shared view,
  locale, complete Workspace, full Core/Views, type/lint, official race, root,
  and production build gates retain their exact assertions and pass freshly.
  Any broad failure remains NON-PASS and receives exact focused attribution; it
  is never renamed PASS.
- Fresh SQLite installed acceptance uses authenticated reviewer authority, real
  Canonical HTTP, and production Web to prove `unlinked -> linked -> implemented
  -> accepted`, later conditional revocation, multi-Issue fail-closed behavior,
  current/effective divergence, unlink, retirement visibility, refresh/reload,
  and restart persistence. Unavoidable membership setup fixtures are disclosed
  and are not counted as product behavior.
- Only fresh independent review returning both required PASS decisions may
  close r037/PCR-S07C and authorize the PCR-S07D successor plan.

## Explicit exclusions and stop conditions

PCR-S07D Retrospectives, Release 3 completion, S10 hierarchy/move/reorder/
numbering/archive/restore/Issue-outline/progress, project phases, Requirement-
driven Issue creation, stored coverage caches, historical Issue-status or
acceptance snapshots, new permissions/flags, migrations/indexes, generated
protobufs, Auth production behavior, unrelated Issue/Input behavior, push,
merge, deployment, external service calls, original dirty paths, legacy backend
writes, and all `server/**` changes are excluded.

Stop before closure on any production Auth change, unbounded/sleep/retry/skip
test workaround, client-inferred coverage, fail-open Issue aggregation,
non-latest acceptance, per-item/per-Issue queries, partial response, authority
leak, hidden test failure, missing/duplicate trailer, scope drift, original
dirty-path overlap, unclosed process, `server/**` or generated change, or either
independent-review BLOCK. Any material repair outside this exact boundary
requires a new immutable plan; v32 is never amended after activation.
