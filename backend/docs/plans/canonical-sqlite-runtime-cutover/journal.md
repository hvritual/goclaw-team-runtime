# Canonical SQLite runtime cutover journal

Append-only execution evidence for Plan-ID
`canonical-sqlite-runtime-cutover`.

## 2026-08-13 — Milestone established

- User approved creation of the repository execution Milestone.
- Frozen plan version: `plan_v1.md`.
- Base commit: `e4b4b1c7e3d46b19fb4774f8757cad4fb4c4f1cc`.
- Branch observed: `codex/issue-metadata-v9`.
- Integration target: `codex/multica-six-domain-baseline`.
- Active step: `M1-S0` only.
- Product implementation status: not started by this milestone-creation task.
- Issue metadata v9 remains an uncommitted candidate at this checkpoint and is
  recorded as a hard prerequisite before `M1-S1`.
- Existing dirty worktree changes were observed and preserved. This task's
  write boundary is only
  `backend/docs/plans/canonical-sqlite-runtime-cutover/**`.
- `server/**` remains read-only and is not an allowed diff.
- Readiness statement: the Canonical backend is not yet a replacement for the
  legacy server; only compatibility/runtime contract discovery is active.

### Initial policy evidence

- `AGENTS.md` SHA-256:
  `637c5ff1222ba462b3b3ff96c74e4ad0b62f52bfa086d76c396d99badb9848e0`
- `CLAUDE.md` SHA-256:
  `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646`
- `backend/AGENTS.md` SHA-256:
  `fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209`

### Next action

Complete `M1-S0`: freeze the real journey's API/event inventory, SQLite runtime
ownership, process topology, characterization tests, exact implementation write
paths, and rollback selector. Do not start product code before its exit gate and
Human approval are recorded.

## 2026-08-13 — M1-S0 discovery completed; plan v2 proposed

- Issue metadata v9 was independently reviewed, committed as `e20114c`, and
  fast-forwarded into the local Canonical baseline. Its 27 paths contain no
  `server/**` and exclude unrelated worktree changes.
- Rebased S0 evidence on `e20114cc7f401b503c6506d1b99cf0eddf894780`.
- Inventoried actual auth, Workspace, Issue list/detail, metadata and realtime
  calls. Because installed detail mounts deferred endpoint families, plan v2
  uses explicit capability gates rather than fake empty responses or hidden
  legacy routing.
- Selected one Canonical product SQLite owner; distinct Canonical/legacy DBs;
  Web/HTTP/gRPC ports 3000/8000/9000; migration-only empty bootstrap;
  dependency readiness; separate control plane; and non-destructive rollback.
- Every critical parity row now has a target decision and executable method.
  These decisions are not runtime proof.
- Added `contract-inventory-v2.md` and proposed `plan_v2.md` with exact story
  paths. No product code changed during S0.
- S0 awaits Human Customer approval. Until approval, plan v1 and M1-S0 remain
  active.

## 2026-08-13 — Plan v2 approved; M1-S1 activated

- Human Customer approved `plan_v2.md` and the frozen S0 contract inventory.
- Approved base: `e20114cc7f401b503c6506d1b99cf0eddf894780` on
  `codex/multica-six-domain-baseline`.
- M1-S0 exit gate is accepted: no critical parity row remains `Unknown`, and
  S1 paths/tests/runtime ownership/rollback are frozen.
- Active story is now `M1-S1`. Product writes remain limited to the S1 paths in
  plan v2; `server/**` remains forbidden.

## 2026-08-13 — M1-S1 RED/GREEN implementation evidence

- RED: focused bootstrap/cmd tests failed to compile because `SQLitePath`, real
  Workspace dependencies, database ownership, readiness, and close APIs did not
  exist.
- GREEN: one Canonical runtime now opens one product SQLite DB, runs Workspace
  and Auth migrations, selects their real opt-in provider graphs, registers all
  four modules, and closes the DB idempotently. Boundary services required by
  later stories are explicitly fail-closed rather than permissive.
- Empty DB, migration catalogs, retained restart/readback, dependency-aware
  readiness, liveness after dependency failure, missing-provider failure and
  idempotent close have focused tests.
- Fresh passing evidence: focused bootstrap/cmd/Auth/Workspace tests; full
  backend `go test ./...`; `go vet ./...`; `go mod verify`; `git diff --check`;
  and an empty `server/**` diff.
- A separate live background-process probe was blocked by the local execution
  policy before process launch; no live-process evidence is claimed for S1.
- Independent review is pending. M1-S2 is not active until S1 review and Human
  Customer acceptance are recorded.

## 2026-08-13 — M1-S1 review corrections

- Independent review found that direct `Runtime.Stop()` did not close the
  product DB and that `:memory:` SQLite could split across multiple pooled
  connections. RED tests reproduced both risks.
- `Stop()` now closes the DB after stopping the app, `Close()` remains
  idempotent, and the in-memory profile is restricted to one connection.
  Focused bootstrap/cmd/Auth/Workspace tests pass after correction.
- Review also confirmed that unrelated UI/local artifacts are outside S1. They
  remain unstaged and will be excluded from the scoped commit.
- The fail-closed Workspace boundary is intentional S1 behavior: real SQLite
  persistence providers are required and selected, while trusted identity and
  authorization become active only in S2/S3. Missing dependency objects still
  fail construction; fail-closed dependencies never grant access.

## 2026-08-13 — M1-S1 independently reviewed

- Independent review result: `PASS`; no remaining P0/P1 finding.
- The review verified shared DB ownership, ordered migrations, real opt-in
  provider selection, readiness, stop/close, retained restart, fail-closed
  boundary behavior, staged scope, and an empty `server/**` diff.
- Final post-review gates passed: full backend tests, vet, module verification,
  staged diff check, and server-path audit.
- M1-S1 is `Integrated` but not yet Customer Accepted. M1-S2 remains inactive
  until Human Customer acceptance is recorded.

## 2026-08-13 — M1-S1 Customer Accepted; M1-S2 activated

- Human Customer accepted M1-S1 and authorized continuation to M1-S2.
- Accepted S1 commit: `8352c84` on `codex/multica-six-domain-baseline`.
- Active story is now M1-S2. Writes are limited to the trusted-authentication
  paths frozen in plan v2; S3 and later stories remain inactive.

## 2026-08-13 — M1-S2 RED/GREEN implementation evidence

- RED: focused Auth tests failed to compile because the local-auth composition
  and configuration did not exist; the Core boundary test proved that a
  verify-code response missing `token` was incorrectly accepted. A runtime
  composition RED also proved the trusted-auth routes were not registered.
- GREEN: Canonical SQLite now owns hashed, expiring and revocable sessions.
  The runtime implements the frozen send-code, verify-code, current-user and
  logout routes with Bearer and HttpOnly-cookie authentication, a readable
  HMAC-bound CSRF cookie, CSRF enforcement for cookie logout, exact empty
  success bodies, fail-closed missing/expired identity, and explicit six-digit
  development-only verification-code configuration.
- Core now validates the verify-code `{token,user}` envelope with Zod and
  rejects a missing or empty token. User IDs and raw session tokens are
  generated independently; only token hashes are persisted.
- Fresh passing evidence: focused Auth/bootstrap/cmd tests; full backend
  `go test ./...`; `go vet ./...`; `go mod verify`; focused Core Auth and
  metadata tests; Core typecheck; `git diff --check`; and an empty
  `server/**` diff.
- The Windows race binary exits `0xc0000139`; no race result is claimed. An
  independent review is pending, so M1-S2 is not yet Integrated and M1-S3 is
  inactive.

## 2026-08-13 — M1-S2 review correction

- Independent review identified a first-login race: concurrent verification
  for the same new email could both observe no user and one request could fail
  the unique-email insert with HTTP 500.
- A 12-way concurrent RED reproduced the failure. User creation now uses
  `INSERT ... ON CONFLICT(email) DO NOTHING` followed by an email lookup; the
  RED passes repeatedly with one user and one distinct session per request.
- Full backend tests, vet, module verification and the focused Auth runtime
  tests pass after correction. Final independent re-review is pending.

## 2026-08-13 — M1-S2 independently reviewed

- Independent re-review result: `PASS`; no remaining P0/P1 finding.
- Review verified the frozen HTTP contract, hashed expiration/revocation,
  Bearer and Cookie identity, HMAC CSRF enforcement, concurrent first login,
  runtime registration, Core response validation, scoped paths, and an empty
  `server/**` diff.
- M1-S2 is Integrated but not yet Customer Accepted. The current Web login
  continues into Workspace listing, which is intentionally M1-S3; no browser
  login journey is claimed at S2. M1-S3 remains inactive until Customer
  acceptance.

## 2026-08-13 — M1-S2 Customer Accepted; M1-S3 activated

- Human Customer accepted M1-S2 and authorized continuous execution through
  M1-S7, subject to every story's frozen scope, RED/GREEN evidence, independent
  specification and code-quality review, and deterministic gates.
- Accepted S2 commit: `02898eb` on `codex/multica-six-domain-baseline`.
- Active story is now M1-S3. Writes are limited to authorized Workspace
  selection paths frozen in plan v2; S4 and later stories remain inactive.

## 2026-08-13 — M1-S3 RED/GREEN and review evidence

- RED: backend tests failed because no trusted session-to-membership Workspace
  selection seam existed; Core accepted malformed Workspace list responses.
- GREEN: `GET /api/workspaces` now resolves the S2 session, lists only the
  user's owner/admin/member memberships, returns the exact legacy Workspace
  array ordered by creation time, and returns `[]` for an outsider. Trusted
  slug/ID resolution produces the canonical Workspace ID and member actor ID;
  missing, expired, foreign, missing-slug and ID/slug mismatch cases fail
  closed. No by-ID Workspace endpoint was added.
- Spec review initially found missing evidence for authorized ID/slug mismatch,
  missing slug, direct foreign ID, route absence and exact raw fields. Focused
  tests added all five cases; specification re-review returned `PASS`.
- Independent code/security review returned `PASS` with no P0/P1/P2. It
  verified session reuse, membership isolation, member actor identity,
  parameterized SQL, stable order, NULL/JSON mapping, HTTP/Core boundaries,
  construction and scope.
- Fresh gates pass: full backend tests, vet and module verification; Core 86
  files/550 tests, typecheck and lint; diff check; no `server/**` diff. M1-S3
  is Integrated under the user's continuous S3-S7 authorization.

## 2026-08-13 — M1-S3 accepted under continuous authorization; M1-S4 activated

- M1-S3 commit `c6c3649` is independently specification- and code-reviewed and
  accepted under the Human Customer's continuous M1-S3 through M1-S7
  authorization.
- Active story is now M1-S4. Writes are limited to the Issue list/base-detail,
  Core contract and explicit capability-gating paths frozen in plan v2; S5 and
  later stories remain inactive.

## 2026-08-13 — M1-S4 RED/GREEN and review evidence

- RED began with 404s for all frozen Issue read routes and `/api/config`, and a
  rendered detail test proved deferred consumers still issued requests.
- GREEN provides authenticated snake_case UUID/identifier detail, GET/POST
  list twins, server-authoritative table facets/groups/rows, exact supported
  controller filters/scopes/sorts, stable opaque bound cursors, hierarchy and
  exact paging envelopes, plus additive Canonical capability flags.
- Canonical config routes the detail to a pure base component; deferred
  timeline/comment/reaction/subscriber/attachment/member/label/property/pin/
  child/project/progress/acceptance/pull-request consumers do not mount. Legacy
  loaded config with absent flags retains enabled behavior.
- Specification review found and drove corrections for deferred consumers,
  hierarchy, unsupported inputs, paging, exact raw contracts and capability
  evidence. Final specification review returned `PASS`.
- Code/security review drove corrections for query-bound cursors, actor-pair
  and no-value filters, strict bounded JSON, controller shape coverage,
  canonical ordering, trusted `my` scope, stable Zustand selectors and sort
  validation. Final code review returned `PASS` with no P0-P2.
- Fresh gates pass: full backend tests/vet/module verification; Core 87
  files/552 tests; focused Views 39/39 and Core capability 2/2; Core/Views
  typecheck; diff check; no `server/**` diff. The broad Views test command hit
  the 120-second local command timeout, so it is not claimed; focused S4 Views
  evidence is green. Real browser/PID/network acceptance remains an S7 gate.

## 2026-08-13 — M1-S4 accepted under continuous authorization; M1-S5 activated

- M1-S4 commit `523b28a` passed independent specification and code/security
  review and is accepted under the continuous M1-S3 through M1-S7 authority.
- Active story is M1-S5. Writes are limited to accepted metadata v9 paths,
  Canonical composition, Core metadata tests and capability-required read-only
  projection paths. S6 and S7 remain inactive.

## 2026-08-13 — M1-S5 RED/GREEN and review evidence

- RED proved the real Canonical composition rejected authenticated metadata
  requests, Cookie mutations skipped CSRF, the capability remained disabled,
  and Workspace validation ran before authentication. Subsequent REDs proved
  the missing rollback selector and unbounded PUT request body.
- GREEN wires the trusted S2 session and S3 membership identity into metadata
  GET/PUT/DELETE, preserves exact v9 envelopes and errors, hides foreign
  Workspace/Issue resources, enforces Cookie HMAC-CSRF while exempting Bearer
  mutations, and supports UUID plus Workspace identifier lookup.
- Metadata persists across a file-backed runtime restart; session expiry,
  transaction rollback, distinct-key concurrency, same-key last-committer and
  overlap with mainline Issue updates are covered. PUT bodies are capped at 64
  KiB before the 8 KiB metadata rule is applied.
- `-issue-metadata=false` omits the whole metadata extension and advertises
  `issue_metadata:false`; focused tests prove GET/PUT/DELETE all return 404.
  The default accepted Canonical profile enables it and leaves realtime false.
- An explicit, non-overwriting persistent browser fixture is available. Its
  fixture creation passed, but starting the real backend/Web processes was
  blocked by the local execution policy before either process launched.
  Therefore browser readback, network trace and PID/port evidence are not
  claimed and remain a hard M1-S7/milestone gate.
- Independent specification review returned conditional pass: all executable
  S5 contracts and rollback selection pass, with browser runtime evidence still
  outstanding. Independent code/security review returned PASS after the body
  cap fix, with no remaining P0-P2.
- Fresh focused backend tests and vet, Core metadata/capability 6/6, Core and
  Views typecheck, diff check and the no-`server/**` boundary all pass.

## 2026-08-13 — M1-S5 accepted conditionally; M1-S6 activated

- M1-S5 is accepted for code integration under continuous authorization, with
  the explicitly recorded browser evidence debt above. Milestone acceptance is
  prohibited until that evidence runs successfully in M1-S7.
- Active story is M1-S6. Writes are limited to the frozen Canonical realtime
  boundary/publisher, Issue/metadata post-commit integration, Core WebSocket
  contract/sync paths and focused accepted-Issue cache tests. M1-S7 is inactive.

## 2026-08-13 — M1-S6 RED/GREEN and review evidence

- RED began with no Canonical `/ws` route or publisher. Review-driven REDs then
  proved non-exact auth ACK/errors, slow-client blocking, unbounded frames,
  over-broad event schemas, request-presence change flags, missing committed
  delete production and incomplete dependent cleanup.
- GREEN provides cookie-upgrade and token-first-frame authentication, exact
  token ACK/error frames, hidden Workspace membership failures, a strict local
  Web-origin allowlist, 64 KiB read limit, per-client bounded ordered queues,
  write deadlines and nonblocking slow-client eviction.
- The four frozen events are Workspace-isolated and schema-checked before Core
  cache work. Issue create/update/delete and metadata put/delete publish only
  after successful SQLite return; failures publish nothing. Metadata carries
  the complete bag and delete publishes the canonical UUID.
- Canonical Issue deletion resolves UUID/identifier, clears Todo, child Issue
  and Requirement references in one `BEGIN IMMEDIATE` transaction, evolves
  Requirement coverage/version/audit state, and rolls everything back on
  failure. Concurrent double delete stress runs produce one 204, one hidden
  404, one event and one audit version.
- Core accepts required `member|agent` actors and exact status/priority/date/
  datetime fields while preserving additive legacy fields. Malformed known
  events are dropped; duplicate delivery coalesces and reconnect performs an
  authoritative current-Workspace Issue graph refresh. No cursor/replay was
  introduced.
- Independent specification and code/security reviews both returned PASS with
  no remaining P0-P2 after multiple correction rounds.
- Fresh gates pass: full backend tests/vet/module verification; concurrent
  runtime stress x10; Core 87 files/557 tests, typecheck and lint; focused
  realtime 24 tests; diff check; no `server/**` diff. Windows race binaries
  remain unable to start with environment code `0xc0000139` and are not claimed.

## 2026-08-13 — M1-S6 accepted; M1-S7 activated

- M1-S6 is independently accepted under the continuous M1-S3 through M1-S7
  authorization.
- Active story is M1-S7. Writes are limited to repository selectors/commands
  outside `server/**`, local setup documentation, `e2e/**`, and this plan
  directory. Browser, process/port, no-legacy, restart/readback and
  non-destructive rollback evidence are mandatory before milestone acceptance.

## 2026-08-13 — M1-S7 hard gate discovered; plan v3 proposed

- Runtime discovery proved Canonical local auth always projects
  `onboarded_at:null`, while the installed Web Workspace layout redirects such
  users to onboarding. The frozen Issue browser journey cannot execute.
- This is not solved by the explicit fixture because the Canonical Auth schema
  lacks the field and the response projection is hardcoded nil. A fixture-only
  bypass would be fabricated evidence.
- [plan_v3.md](plan_v3.md) proposes the smallest additive Auth migration/store
  correction and focused tests. No Auth product path may be written until the
  Human Customer explicitly approves v3.
- Read-only port evidence also found pre-existing legacy `:8080` and Web
  `:3000` listeners not owned by this run. They were not stopped. Canonical-only
  acceptance must fail closed while either remains.

## 2026-08-14 — M1-S7 v2 acceptance tooling code-ready; live gate not run

- The portable selector now owns exact Web `3000`, Canonical HTTP `8000`, gRPC
  `9000` and `data/multica-canonical.db` identities. It rejects PID reuse,
  foreign checkout evidence, unowned or orphan `cmd/server` processes, legacy
  listeners, malformed manifests and partial startup. Stop and rollback wait
  for owned descendants and preserve database, WAL/SHM/journal and log hashes.
- The explicit fixture requires an existing migrated database, is
  transactionally non-overwriting and rejects partial, conflicting or
  relationship-inconsistent footprints. The browser specification uses the
  real login, Workspace list, Issue list/detail and metadata GET/PUT/DELETE
  paths, asserts CSRF and realtime UI readback, rejects legacy traffic and
  emits a sanitized HTTP/WebSocket trace when executed.
- Focused selector/verifier tests pass `22/22`; the fixture Go test passes;
  Playwright discovers one Chromium acceptance scenario; script syntax, diff
  check and the no-`server/**` boundary pass. Independent specification and
  code/security reviews returned PASS with no remaining P0-P2 in the v2 S7
  tooling scope.
- These results are code-readiness evidence only. No Canonical runtime,
  browser journey, restart or rollback acceptance was executed because the
  unapproved `onboarded_at` Auth correction and pre-existing unowned `:3000` /
  `:8080` processes remain hard gates. M1-S7 and the milestone are not accepted.

## 2026-08-14 — plan v3 approved; controlled local shutdown authorized

- The Human Customer explicitly approved `plan_v3.md`, authorizing only the
  additive Canonical Auth `onboarded_at` migration/store projection, focused
  tests and consistent explicit fixture state required by M1-S7.
- The Human Customer also authorized stopping the previously identified local
  Web `:3000` PID `25172` and legacy `:8080` PID `23412` process trees for the
  Canonical-only acceptance run. No other process or data deletion is
  authorized.
- Active plan is v3 and active step remains M1-S7. Auth implementation begins
  with RED tests; live shutdown and acceptance follow only after deterministic
  Auth gates pass.

## 2026-08-14 — M1-S7 live Canonical acceptance evidence

- RED reproduced the browser gate: Canonical Auth had no `onboarded_at` column
  or projection. GREEN adds nullable migration `000003`, reads it for login and
  `/api/me`, keeps new users null, and gives only the explicit fixture a
  consistent non-null timestamp alongside its owner membership/root. Retained
  v1/v2 database upgrade preserves all prior User fields. Independent Auth
  specification and code/security reviews both returned PASS with no P0-P2.
- The authorized pre-existing Web PID `25172` (and esbuild child `24612`) and
  legacy SQLite PID `23412` were identity-checked, stopped, and read back
  absent. Ports `3000/8000/8080/9000` were quiescent before Canonical startup;
  no other process was stopped and no database or log was deleted.
- The first live selector run correctly failed closed because Web was launched
  from the repository root and Next could not find `app/pages`. RED fixed the
  Web cwd to `apps/web`; selector tests returned `22/22`. Subsequent clean
  startup migrated the previously absent `data/multica-canonical.db`, exposed
  Web `3000`, HTTP `8000`, gRPC `9000`, exact health/readiness and no `8080`.
- The runtime was stopped, the explicit non-overwriting fixture reported
  `created`, and the same database restarted. The in-app Browser plugin was
  attempted twice but its runtime failed before navigation with `failed to
  write kernel assets: os error 3`; the required browser-client assets existed.
  The frozen repository Playwright scenario therefore ran as the documented
  fallback against installed Chrome. A missing bundled Chromium was handled by
  explicit `PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH`; no browser download occurred.
- Real Chromium acceptance passed in `13.3s`: UI email/code login, Workspace
  list/selection, Issue list/open and base detail; metadata GET/PUT/GET/DELETE;
  cookie CSRF; two `issue_metadata:changed` frames; and no HTTP/WebSocket URL on
  port `8080`. The accepted Canonical base-detail capability intentionally has
  no full-detail Metadata dialog, so metadata readback is asserted through the
  same-origin API while the base detail remains visible. A sanitized trace is
  retained at `.local-runtime/m1-s7-20260814/canonical-network-trace.json` and
  in the Playwright result attachment.
- Completion audit then found that this first successful browser run compiled
  a preserved, unrelated dirty `packages/ui` Input change that is excluded
  from the M1-S7 candidate. Independent specification review therefore did not
  accept that trace as clean-candidate evidence. The scoped v3 candidate must
  be committed and the same journey rerun from an isolated worktree before
  M1-S7 can be accepted; the user-owned Input change remains untouched.
- Verifier mutation wrote `retained-1786643490292`; stop/restart readback
  matched. Quiescent snapshot and rollback preserved exact hashes for the
  Canonical DB, retained legacy DBs, WAL/SHM files and logs, selected legacy
  without starting it, then reselected Canonical, restarted the same DB and
  read the retained value again.
- Final gates pass: full backend tests, vet and module verification; changed-Go
  `gofmt -d`; focused Auth/fixture x3 and retained/concurrency/restart x20;
  selector/verifier `22/22`; Core `87` files / `557` tests, Core typecheck/lint,
  Views and Web typecheck; diff check and no tracked/untracked `server/**`.
  `make fmt-check` itself is not claimed: the Windows wrapper exits before the
  formatter with `exit was unexpected at this time`; the underlying changed-Go
  format check is clean. Windows race binaries remain the recorded
  `0xc0000139` environment limitation.

## 2026-08-14 — clean-candidate browser rerun; M1-S7 technical acceptance

- The scoped candidate was committed as
  `7700a193525626771d75ae50a425d0bf50542638`, explicitly excluding the
  preserved native-Input change, its local test and all other unrelated local
  artifacts. An isolated detached worktree at that exact commit confirmed the
  candidate still used the retained Base UI Input implementation.
- From that clean candidate, a fresh Canonical database was migrated, the
  explicit fixture was prepared, and the frozen installed-Chrome journey
  passed in `33.7s`. The retained sanitized trace is
  `.local-runtime/m1-s7-20260814/clean-candidate-network-trace.json`: `106`
  HTTP requests from the Web origin on `127.0.0.1:3000`, UI login, Workspace
  selection, Issue list/detail, metadata GET/PUT/GET/DELETE, `2`
  `issue_metadata:changed` events, and zero HTTP/WebSocket traffic on port
  `8080`.
- The previously successful dirty-worktree trace remains rejected and is not
  acceptance evidence. The isolated rerun is the authoritative browser proof.
  The temporary worktree registration was removed, the runtime was stopped,
  ports `3000/8000/8080/9000` were verified quiescent, and tracked/untracked
  `server/**` remained empty. The main worktree retains only the user's
  pre-existing unrelated dirty paths.
- Independent post-live specification review returned `SPEC ACCEPT/PASS` and
  independent code/security review returned `CODE PASS/APPROVED`, with no
  remaining P0-P2. Together with the retained restart/readback and exact
  rollback hashes, M1-S7 technical acceptance is complete.
- Human Customer milestone acceptance has not been inferred from approval to
  continue testing. `Milestone Accepted` remains pending an explicit final
  acceptance statement.

## 2026-08-14 — onboarding 404 reproduced; plan v4 approved

- Human browser evidence showed `/onboarding` displaying `API error: 404 Not
  Found`. The live Web log tied the blocking action to `/api/workspaces`.
  Static route inventory confirmed Canonical registered only
  `GET /api/workspaces`; the installed onboarding flow uses the existing Core
  `POST /api/workspaces` and then
  `POST /api/me/onboarding/complete`, neither of which existed.
- This is a confirmed post-v3 compatibility gap for a real new user, distinct
  from the accepted pre-created browser fixture. It blocks final Customer
  acceptance even though the earlier frozen fixture journey remains valid.
- The Human Customer approved `plan_v4.md`. Active strict-XP step is
  `M1-S7-C1`: atomically create the first Workspace/owner membership, complete
  onboarding idempotently, keep exact API/Core compatibility, and rerun the
  installed browser journey. No general onboarding, invitation or Workspace
  CRUD expansion is authorized.

## 2026-08-14 — M1-S7-C1 RED/GREEN and live browser evidence

- RED first proved the installed boundaries independently: Canonical returned
  404 for `POST /api/workspaces`; after that boundary was added, it returned
  404 for `POST /api/me/onboarding/complete`. Core RED also proved malformed
  success bodies were previously accepted.
- GREEN creates the Workspace row and the Auth-owned owner member/root in one
  `BEGIN IMMEDIATE` transaction through a public Auth participant contract.
  Forced Auth persistence failure rolls back all three rows. Duplicate slugs
  return the frozen 409 while unrelated unique-ID failures remain 500.
  Concurrent same-slug creation is deterministic (one 201, one 409).
- Completion resolves trusted identity before request data, reuses the S2
  Cookie-CSRF boundary, validates Workspace membership, sets `onboarded_at`
  once with `COALESCE`, and returns the exact User projection. Retry before and
  after close/reopen preserves the first timestamp. Missing, expired and
  foreign identities, strict unknown/trailing JSON, generator failure and
  partial-write cases have executable coverage.
- Compatibility review corrected generated `issue_prefix` to the retained
  ASCII-letters-only rule and narrowed duplicate-slug classification to the
  actual `workspaces.slug` constraint. The existing onboarding preview still
  displays a four-character slug-derived hint while persistence uses the
  retained three-letter name-derived rule; this pre-existing display-only
  discrepancy is recorded as non-blocking UI debt and was not expanded into
  the approved API correction.
- Installed Chrome then passed the real new-user journey in `26.5s`: UI
  email/code login, `/onboarding`, first-Workspace creation, onboarding
  completion, Workspace Issue route and reload without redirecting back to
  onboarding. This directly closes the reported `API error: 404 Not Found`.
- Full gates pass: backend `go test ./...`, `go vet ./...`, `go mod verify`;
  Core `87` files / `561` tests, Core typecheck/lint, Views and Web typecheck;
  selector/verifier `22/22`; diff check and no tracked/untracked `server/**`.
  The preserved unrelated `packages/ui` Input change and local artifacts are
  excluded from the correction candidate. Clean-candidate browser rerun and
  final Human Customer acceptance remain the final promotion gates.

## 2026-08-14 — M1-S7-C1 clean-candidate technical acceptance

- The correction was committed as
  `4edc940a3797456bc696b72e6e6c4756bd0e15a4`. Its 25-path scope contains only
  plan/evidence, Canonical Auth/Workspace/bootstrap code and tests, Core API
  parsing/tests, and the existing Canonical E2E. It contains no `server/**`,
  `packages/ui/**` or unrelated local artifact.
- The two preserved user Input paths were hashed, temporarily isolated from
  the working tree, and the runtime was restarted from exact HEAD. Installed
  Chrome passed the new-user flow again in `21.6s`: UI login, onboarding,
  201 Workspace creation, 200 completion, Workspace Issue route and reload.
  The user paths were then restored and remain outside the candidate.
- Independent read-only review returned conditional SPEC/CODE PASS with no
  P0/P1. The only retained P2 is the pre-existing onboarding prefix preview:
  UI displays a four-character slug-derived hint while persisted compatibility
  remains the legacy three-letter name-derived value. It does not affect the
  created Workspace, routing or onboarding completion and is outside v4's
  narrow API correction.
- `M1-S7-C1` is technically integrated. Per milestone policy, approval to
  execute plan v4 is not treated as final Customer acceptance; the
  `Milestone Accepted` label remains pending an explicit Human statement.

## 2026-08-14 — Projects page 404 reproduced; plan v5 approved

- The installed `/drcoffee/projects` page returned 200, but Web runtime logs
  proved three normal requests returned 404: `GET /api/projects?`,
  `GET /api/workspaces/{id}/members`, and `GET /api/pins`. Repeated React Query
  retries produced the reported `API error: 404 Not Found` toast.
- The Human Customer approved plan v5. Active strict-XP story is `M1-S7-C2`:
  implement the visible Project CRUD, exact member list and real per-user
  Project/Issue pins rather than hiding missing behavior behind empty shims.
- Browser plugin initialization failed before tab acquisition with
  `Cannot redefine property: process`; the already-established repository
  Playwright/runtime-log fallback is used for RED/GREEN browser evidence.

## 2026-08-14 — M1-S7-C2 RED/GREEN technical candidate

- RED runtime evidence tied the visible toast to three absent Canonical routes:
  `GET /api/projects`, `GET /api/workspaces/{id}/members`, and
  `GET /api/pins`. Focused runtime tests first failed with 404 before any
  Project-surface implementation existed.
- GREEN adds authenticated, Workspace-scoped Project list/get/create/update/
  delete, the exact member projection, and persistent per-user Project/Issue
  pin list/create/delete. SQLite writes use explicit transactions, strict
  tenant predicates, dependent-pin cleanup, bounded strict JSON decoding,
  Cookie-CSRF enforcement, and restart-safe migrations. Project delete keeps
  the retained owner/admin rule; missing or foreign targets remain hidden 404.
- Compatibility review corrected the UI lead identity to `user_id`, preserves
  fractional pin positions, maps nullable description/icon/lead/date fields to
  the retained response shape, narrows duplicate conflicts, and proves failed
  owner/deletion operations roll back. Concurrent duplicate pin creation is
  deterministic: one 201, one 409, and one retained row.
- Unsupported resource, retrospective, requirement and control sub-surfaces
  are disabled by explicit Canonical capability flags. Pin reorder remains the
  plan-v5 deferred operation and is not exercised by the accepted Projects-page
  journey.
- Installed Chrome passed the real Projects journey in the active development
  tree: authenticated Projects/member/pin reads returned 200, visible Project
  creation and detail navigation succeeded, pin creation returned 201, cleanup
  delete returned 204, and no 404 toast appeared. This run is GREEN evidence;
  the isolated exact-commit rerun remains the promotion proof.
- Backend `go test ./...`, `go vet ./...`, and `go mod verify` pass. Core passes
  88 files / 564 tests plus typecheck/lint; Views and Web typecheck pass;
  Project-focused View tests pass 4/4; selector/verifier tests pass 22/22.
  Views full test has four unrelated baseline failures in locale plural-key and
  RichContent import-boundary guards; none of those files are in this candidate.
- Candidate scope excludes `server/**`, the user's existing
  `packages/ui/components/ui/input.tsx` and
  `packages/views/auth/input-controlled.test.tsx` changes, and local artifact
  directories. Independent final review and an exact clean-candidate Chrome
  rerun remain required before technical promotion. Human Customer acceptance
  is not inferred from approval to execute plan v5.

## 2026-08-14 — M1-S7-C2 clean-candidate technical acceptance

- The implementation candidate was committed as
  `6467b8cfd7cc04bdc5a62fe1dcdd47bd82ef1468`. Its 34-path scope contains the
  approved plan/evidence, Canonical Auth/Workspace/bootstrap implementation and
  tests, Core/View capability and schema corrections, the verifier matrix, and
  the Canonical E2E. It contains no `server/**`, `packages/ui/**`, Auth Input
  test, or unrelated local artifact.
- The user's two unrelated Input files were SHA-256 hashed, isolated through a
  path-scoped Git stash, and the Canonical runtime was restarted from exact
  commit `6467b8c`. Installed Chrome passed the Projects journey in `17.8s`:
  the three formerly missing reads returned 200, visible Project create/detail
  and pin actions succeeded, cleanup returned 204, and no Project/member/pin
  404 was observed. The stash was restored and both hashes matched exactly.
- Independent final read-only review returned CODE/SPEC PASS with no P0-P2
  functional blocker. It confirmed strict identity/tenant/role/CSRF behavior,
  transactional Project and Pin persistence, restart durability, nullable
  projection compatibility, rollback/dependent cleanup and zero `server/**`.
- Explicit non-blocking scope notes remain: Pin reorder is deferred by plan v5;
  resources, retrospectives, requirements and control are capability-disabled;
  the full Views suite has four unrelated baseline failures while all focused
  Project tests and typechecks pass.
- `M1-S7-C2` is technically integrated and the runtime was restarted for Human
  verification at `http://127.0.0.1:3000/drcoffee/projects`. Approval to execute
  is not treated as final acceptance. The milestone remains pending an explicit
  Human Customer acceptance statement.

## 2026-08-14 — Project-detail Issue surface 404 reproduced; plan v6 proposed

- Human browser evidence identified a new toast on
  `/drcoffee/projects/0631ebaf-d1e0-42bd-a03b-1da3f8110dd5`. Current Web logs
  prove the detail route itself is 200 while its embedded Issue surface sends
  `GET /api/properties` and `GET /api/issues/child-progress`, both 404.
- Opening the visible Issue-create UI then sends
  `GET /api/labels?resource_type=issue` (404), and submitting it sends
  `POST /api/issues` (404). The latter is the user-visible failed action; the
  existing Canonical Issue use case/repository/realtime publisher is present,
  but no trusted HTTP create route is registered.
- Runtime config already advertises labels, properties, child progress and
  attachments as false. Static inspection found unconditional queries in the
  shared Issue surface and create modal, so this is both a missing HTTP create
  boundary and incomplete capability enforcement—not a Project CRUD
  regression and not a case for fabricated empty responses.
- `plan_v6.md` proposes narrow story `M1-S7-C3`: implement exact Issue create
  over the existing transactional service, gate every disabled auxiliary
  request/control, and rerun the Project-detail/create journey. Product code
  remains unchanged pending explicit Human approval. `server/**` remains zero.

## 2026-08-14 — plan v6 approved; M1-S7-C3 RED active

- The Human Customer explicitly approved `plan_v6.md`. The approved active
  story is `M1-S7-C3`, beginning at `M1-S7-C3-RED`.
- Product writes are now authorized only inside the plan-v6 boundary: the
  trusted Issue-create HTTP adapter, disabled-capability query/control gates,
  focused runtime/Core/View/E2E tests, verifier capability matrix and this
  plan directory. `server/**` and the preserved unrelated Input/local paths
  remain excluded.

## 2026-08-14 — M1-S7-C3 RED/GREEN technical candidate

- RED was executable at both boundaries: the focused runtime test received
  `POST /api/issues` 404, and installed Chrome opened the visible Project Issue
  composer but never observed a 201 response. The pre-fix Web trace also
  contained 404s for Property, child-progress and Label reads.
- GREEN adds a trusted, Workspace-scoped Issue-create HTTP route over the
  existing SQLite use case and realtime publisher. Bearer and Cookie-CSRF,
  missing/expired identity, missing Workspace, hidden Project lookup, strict
  unknown/trailing/oversized JSON, unsupported Label/attachment input, exact
  public Issue response and disabled route/capability behavior are covered by
  focused runtime tests.
- Shared Issue surfaces now honor false Label, Property, child-progress and
  attachment capabilities before mounting queries or controls. Independent
  review caught and closed a misplaced Status/Label gate plus the persisted
  Table Label column/picker path. A focused View test proves Labels are absent
  while Status remains available.
- Installed Chrome passed the active-tree Project detail -> visible Issue
  create -> realtime `issue:created` -> reload journey with no monitored 404.
  The exact clean-candidate rerun remains the promotion proof. The existing
  false `issue_children` parent/sub-issue workflow remains outside plan v6 and
  is not part of the accepted title-first create journey.
- Backend full tests, vet and module verification pass; Core 88 files / 564
  tests and typecheck pass; focused Views 46/46 and typecheck pass; selector
  and verifier 22/22 pass. Views full lint still reports three pre-existing
  i18n literal errors in `issue-detail.tsx`; no v6 changed-file lint error
  remains. Candidate scope excludes `server/**` and the preserved unrelated
  Input/local artifact paths.

## 2026-08-14 — M1-S7-C3 clean-candidate technical acceptance

- The reviewed implementation candidate is exact commit
  `b9a96e63896c8c7885968bb22b4439b418a17597`. A final RED/GREEN closed the
  remaining Display-menu capability leak: when Canonical Label and child
  progress flags are false, their card-property controls are not mounted.
  Focused Views tests pass 47/47 and Views typecheck passes.
- The user's unrelated Input implementation and its untracked regression test
  were SHA-256 hashed, removed through a path-scoped temporary stash, and
  restored after the browser run with both hashes matching exactly. The
  compiled tracked candidate therefore matched `b9a96e6`; `server/**` remained
  unchanged and port 8080 remained closed.
- Repository Playwright's bundled Chromium was unavailable, so no dependency
  was installed. The same repository test was rerun with the installed Google
  Chrome selected through `PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH`. All three
  Canonical journeys passed in 1.6 minutes: new-user onboarding, retained
  Workspace/metadata/realtime, and Project detail -> visible Issue create ->
  `issue:created` -> table/reload persistence with no monitored missing route.
- Backend `go test ./... -count=1`, `go vet ./...`, and `go mod verify` pass;
  Core passes 88 files / 564 tests plus typecheck; selector/verifier passes
  22/22. One intentionally concurrent multi-tool invocation produced two
  bootstrap fixture 500s; both tests passed immediately when isolated and the
  complete backend suite passed when run alone, so it is retained as parallel
  resource-interference evidence rather than hidden or counted as GREEN.
- `M1-S7-C3` is technically accepted with no remaining P0-P2 blocker in the
  approved v6 scope. The broader milestone and this correction still require
  an explicit Human Customer acceptance statement; approval to execute plan
  v6 is not interpreted as that final acceptance.

## 2026-08-14 — local full Issue detail selected; plan v7 approved

- New browser evidence on `/drcoffee/issues` records
  `POST /api/issues/f532fa94-761c-456b-bcbd-d7c0ceaca2e4/move` returning 404
  during a visible Issue move. Opening that Issue renders only identifier,
  title and `No description` because the current `IssueDetail` replaces the
  full component whenever Canonical advertises `issue_timeline=false`.
- Read-only branch analysis proves `origin/agent/tc-w01-team-control-001` is an
  ancestor of the current HEAD, so its Canonical Issue core is already present.
  The separate root `teamcontrol` model has different project-governance
  fields/statuses and is not the Multica API. It remains behavior evidence for
  authorization, transitions, assignments, Artifacts and correlations.
- Current frontend inventory identifies the approved local detail families:
  core update/move, hierarchy/batch, timeline/comments, comment and Issue
  reactions, subscribers, labels, custom properties, acceptance conclusions,
  attachments, member/project/pin projections and their realtime cache paths.
  Canonical currently lacks the majority of those HTTP/storage boundaries;
  Space Asset is still a generated stub.
- The Human Customer first selected `全量详情`, then explicitly approved
  `批准本地全量 plan v7`. External GitHub/VCS ingestion remains capability-off;
  all approved local families must become real rather than fabricated empty
  success.
- `plan_v7.md` is the immutable approved snapshot at base
  `9ab58a5204d083a757b205d512fcaa2c98c26331`. Strict XP keeps one active story;
  `M1-S7-C4-RED` is active. Product writes remain blocked until its failing
  acceptance tests prove the update/move and public member-actor defects.
- Preserved unrelated paths remain `packages/ui/components/ui/input.tsx`,
  `packages/views/auth/input-controlled.test.tsx`, `.local-runtime/`,
  `docs/code-to-product/`, and `ui/`. `server/**` remains permanently read-only.

## 2026-08-14 — M1-S7-C4 RED/GREEN deterministic candidate

- Runtime RED first proved a newly created Issue exposed the private Auth
  membership-row ID instead of the public `user_id`. The focused request then
  proved `PUT /api/issues/:id` and `POST /api/issues/:id/move` both returned
  404. Core RED separately proved `moveIssue` accepted a malformed response.
- GREEN extends the Auth membership projection with its public user ID while
  preserving private membership IDs for authorization compatibility. New HTTP
  Issue creators persist public member actor IDs, and a startup-owned
  `BEGIN IMMEDIATE` compatibility transaction normalizes retained member
  creator/assignee references only when they match an Auth membership in the
  same Workspace.
- The trusted Issue HTTP adapter now owns strict bounded update and move
  bodies, authentication-before-Workspace ordering, Cookie-CSRF enforcement,
  target-first hidden 404 behavior, nullable field clears and complete
  snake_case Issue responses. Core move responses are validated with the same
  exact Issue schema as update responses.
- Relative move never accepts a caller-authored canonical position. SQLite
  re-reads the moved Issue and Workspace-scoped anchors, computes the relative
  position and applies the allowed patch inside one `BEGIN IMMEDIATE`
  transaction. Empty, missing, self, duplicate, stale and foreign/missing
  anchors fail without a write. A trigger-forced failure rolls back status and
  position and emits no event; a successful commit emits one complete
  `issue:updated` event.
- Runtime tests cover public actor projection, retained upgrade, update/readback,
  strict request/auth/CSRF cases, move edge cases, post-commit event ordering,
  rollback, ten repeated concurrent same-Issue moves and restart persistence.
  Backend `go test ./... -count=1`, `go vet ./...` and `go mod verify` pass.
  Core passes 88 files / 565 tests plus typecheck and lint.
- Candidate paths contain no `server/**` change and exclude the preserved
  unrelated Input/local artifact paths. `M1-S7-C4-INTEGRATE` now requires the
  exact committed clean-candidate browser edit/move proof before story
  promotion.

## 2026-08-14 — M1-S7-C4 clean-candidate browser technical acceptance

- The exact committed candidate was `e27ca1639ad359a1ee29cc17f1efe25b6d9e3578`.
  Before its runtime was started, the unrelated tracked Input change and its
  untracked controlled-input test were recorded by SHA-256 and removed through
  a path-scoped temporary stash. Two stat-dirty View paths were independently
  confirmed byte-for-byte equal to HEAD, so the compiled product tree matched
  the committed candidate.
- The runtime selector owned Web `3000`, Canonical HTTP `8000` and gRPC `9000`;
  legacy `8080` remained closed. Browser automation could claim the user's
  existing in-app tab but the browser URL policy rejected all localhost page
  inspection and navigation. The policy was not bypassed. The Human Customer
  completed the requested visible refresh, edit and move in that clean runtime.
- Post-action evidence shows the retained Issue
  `f532fa94-761c-456b-bcbd-d7c0ceaca2e4` persisted a candidate-runtime
  `updated_at` of `2026-08-14T13:53:05.6470281Z` with status `backlog`. The Web
  log contains no new `/api/issues/:id/move` 404 in the action window; the only
  move 404 entries predate the candidate restart. This evidence is paired with
  the deterministic C4 HTTP/SQLite/event tests rather than presented as a
  standalone network trace.
- The selector stopped cleanly and all four fixed ports were closed. The exact
  path-scoped stash was restored and both user-file hashes matched their
  pre-run values: Input `6D5609B20D2D518DC9DDEF3956184BAA869CEB755468D912C5C153C4E2578B39`
  and its test `A74337C720C978583C38AF10BD44D7AB3A3CEF48BF7EB273FC3AE498EE9F21B9`.
  `server/**` remained untouched.
- `M1-S7-C4` is technically accepted. This is story promotion evidence, not
  final milestone Customer acceptance. The one active story advances to
  `M1-S7-C5-RED` for hierarchy and batch operations.

## 2026-08-14 — M1-S7-C5 hierarchy and batch technical acceptance

- RED first proved the installed hierarchy, child-progress and batch routes
  returned 404 and that malformed Core hierarchy responses were accepted by a
  fallback. GREEN adds exact per-parent and batched child reads, completed
  progress, bounded strict batch bodies, cycle-safe reparenting and atomic
  batch update/delete under one `BEGIN IMMEDIATE` transaction.
- Batch deletion resolves UUIDs and identifiers before mutation, cleans Todo,
  child-parent, Requirement lifecycle/version/audit and Pin references in the
  same transaction, and publishes complete update/delete events only after a
  successful commit. Rollback triggers prove all product and audit state is
  retained and no event escapes; repeated concurrency tests prove serial whole
  commits rather than mixed rows.
- The committed candidate is
  `5725587cc9ab405070128997ca87259d54e58d19`. Its 20 paths exclude every
  preserved Input/local-artifact path and contain no `server/**` change. The
  stat-dirty Issue table and create-modal paths were byte-for-byte equal to
  HEAD and were not staged.
- Backend `go test ./... -count=1`, `go vet ./...`, `go mod verify`, Core 88
  files / 567 tests plus typecheck/lint, Views focused 41 tests plus typecheck,
  Web typecheck and selector/verifier 22/22 all pass. The changed-file Views
  lint still reports three pre-existing C4 literal-string findings outside the
  C5 diff; both newly changed Surface files pass lint.
- The Human Customer replied `已完成` after the requested visible hierarchy and
  batch check. This records C5 manual acceptance paired with the deterministic
  HTTP/SQLite/event/restart evidence; it is not final milestone acceptance.
  The only active story advances to `M1-S7-C6-RED`.
