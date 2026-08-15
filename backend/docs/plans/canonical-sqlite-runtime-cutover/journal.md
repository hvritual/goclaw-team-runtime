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

## 2026-08-15 — M1-S7-C6 collaboration technical and manual acceptance

- RED proved that the mounted timeline, comment, reaction and subscriber
  interactions had no Canonical route family. GREEN adds Workspace-owned
  comments, comment and Issue reactions, subscribers, activities and local
  knowledge-proposal evidence through additive migration `000006`, trusted
  application services and exact HTTP adapters. No fabricated empty route is
  used to suppress a missing request.
- Every JSON mutation authenticates before Workspace resolution, enforces the
  accepted Cookie-CSRF/Bearer split, reads the target before decoding where
  hidden-target ordering requires it, bounds bodies at 1 MiB, and rejects
  malformed, unknown and trailing input. SQLite writes use per-connection
  `BEGIN IMMEDIATE`, Workspace predicates, readback before commit and explicit
  rollback. Comment-thread deletion and Issue single/batch deletion clean all
  collaboration dependents transactionally.
- Runtime acceptance covers root/reply ordering, cross-Issue parent rejection,
  owner/admin moderation, one resolution per thread, knowledge-proposal
  idempotency, comment and Issue reactions, subscribers, rollback/no-event,
  cascade deletion, restart readback and repeated concurrent writers. The
  realtime hub now accepts the collaboration event union through the existing
  bounded client queues; Core validates known payload fields before cache
  invalidation and explicitly refreshes detail keys on reconnect/socket swap.
- The committed candidate is `2081cdc`. Its 35 paths exclude `server/**`, the
  unrelated Input and controlled-input test, local artifact roots, and the two
  stat-dirty View paths whose bytes equal HEAD. Backend full tests/vet/module
  verification, Core full tests/typecheck/lint, focused IssueDetail tests,
  Web typecheck and selector/verifier gates pass. Final focused evidence is Go
  Workspace/Bootstrap/Realtime PASS, Core 31/31, IssueDetail 41/41 and selector
  22/22. A diagnostic full Views run exposed four pre-existing Windows-path and
  locale-contract test failures outside the active write boundary; temporary
  proof fixes were removed rather than mixed into this candidate.
- Activity rows are a durable post-commit projection of successful Issue
  mutations. An activity insert failure cannot turn a committed Issue into a
  false API failure and cannot publish a false `activity:created` event. The
  lack of a shared Issue/activity transaction remains an explicit final-C9
  atomic-audit item rather than an unreported guarantee.
- The Human Customer replied `已完成` after the requested visible C6 check. This
  records C6 manual acceptance paired with deterministic HTTP/SQLite/event/
  restart evidence; it is not final milestone acceptance. The sole active
  story advances to `M1-S7-C7-RED`.

## 2026-08-15 — M1-S7-C7 labels/properties/acceptance RED

- A real Canonical Runtime test creates an authenticated owner, Workspace and
  Issue, then exercises the exact mounted detail dependencies. `GET
  /api/labels?resource_type=issue`, `GET /api/issues/:id/labels`, `GET
  /api/properties` and `GET /api/issues/:id/acceptance-conclusions` each return
  the router's `404 page not found`. `/api/config` simultaneously advertises
  `issue_labels=false`, `issue_properties=false` and `issue_acceptance=false`.
- A focused Core boundary test supplies structurally malformed 200 responses.
  `listLabels()` silently resolves to `{labels:[], total:0}` instead of
  rejecting the malformed success, proving the existing compatibility fallback
  cannot serve as the accepted Canonical contract.
- Commands: `go test ./internal/bootstrap -run
  TestSQLiteRuntimeServesIssueCatalogRoutes -count=1 -v` and `pnpm --filter
  @multica/core exec vitest run api/issue-catalog-schema.test.ts`. Both fail at
  the intended missing-contract assertions, not because of compilation or the
  environment. The active step advances to `M1-S7-C7-GREEN`.

## 2026-08-15 — M1-S7-C7 labels/properties/acceptance GREEN

- Additive Workspace migration `000007` owns Issue label definitions and
  assignments, property definitions and typed Issue bags, acceptance
  conclusions and acceptance-knowledge captures. Catalog mutations use a
  per-connection `BEGIN IMMEDIATE`, Workspace predicates, readback before
  commit and rollback on every failed write. Issue deletion and label deletion
  clean dependent rows transactionally.
- The real Runtime now serves label catalog CRUD, attach/detach complete bags,
  property catalog list/create/read/update, typed set/unset complete bags, and
  acceptance list/create plus atomic completion/capture. Authentication runs
  before Workspace resolution; Cookie mutations require the accepted CSRF
  proof, Bearer mutations do not; malformed, unknown, trailing and oversized
  JSON is rejected before persistence.
- Runtime acceptance covers exact empty/success/error envelopes, missing and
  expired identity, slug/ID mismatch, same-user cross-Workspace hiding, member
  property administration denial, every supported property type, option-use
  conflicts, archive semantics, the 20-active limit, case-insensitive
  uniqueness, ID-constraint classification, malformed historical bag repair,
  rollback, dependent cleanup, restart readback and repeated concurrent
  writers. A live WebSocket proves catalog and complete-bag events publish only
  after commit; a trigger-forced failure retains the old value and emits no
  event.
- Core now rejects malformed or default-synthesized Canonical catalog success
  bodies, validates known realtime label/property fields before consumers, and
  refreshes acceptance knowledge with the authoritative reconnect invalidation
  set. The full enabled-capability IssueDetail test renders label, property and
  acceptance evidence while disabled attachment and PR consumers remain
  unmounted.
- Deterministic evidence is green: Go full tests, `go vet ./...`, `go mod
  verify`; Core 90 files/580 tests plus typecheck/lint; IssueDetail 42/42;
  Views and Web typecheck; selector/verifier 22/22; C7 concurrent writers pass
  20 repeated runs. One initial all-at-once Windows gate invocation caused two
  fixture-login 500s under host resource contention; the exact pair then
  passed 20 repeated runs and the required sequential full backend run passed.
  Browser evidence and Human Customer acceptance remain pending, so the active
  step advances only to `M1-S7-C7-INTEGRATE`.

## 2026-08-15 — M1-S7-C7 clean-candidate browser technical acceptance

- The exact reviewed candidate is commit
  `845b5f22ea13afc915fa4cf44aeb327dac977261`. It was checked out detached at
  `F:\code\ai\goclaw-team-runtime-c7-clean-845b5f2`; the candidate contains
  the committed Base UI Input and excludes the user's unrelated Input change,
  stat-dirty byte-identical View paths and local artifact roots. `server/**`
  remains unchanged.
- The stopped main-runtime database and the detached candidate copy both had
  pre-start SHA-256
  `68D6ACE819575C3EEA7468DB1A91AC3585D44D9A4B1C277F090DB667F2858886`.
  The candidate selector owns Web `3000`, Canonical HTTP `8000` and gRPC
  `9000`; legacy `8080` is closed. `/healthz`, `/readyz`, the Web root and the
  direct/proxied `/api/config` checks pass with identical capability data.
- The Human Customer reopened retained Issue
  `f532fa94-761c-456b-bcbd-d7c0ceaca2e4` at the clean-candidate URL. Browser
  evidence shows the full detail rather than the prior title-only placeholder:
  title and description, durable activity/comment surface, attached label
  `测试`, the property-field dialog and every local status including
  `已完成` render and respond. The page has meaningful DOM, no framework error
  overlay and a retained screenshot in the acceptance conversation.
- Browser console filtering reports no warning/error for label, property,
  acceptance or any `404 /api/issues` request. The retained
  `/api/invitations` 404 is explicitly outside plan v7 (`Invitations` remains
  capability-off) and is neither reclassified as C7 success nor hidden.
- C7 has deterministic HTTP/SQLite/realtime/restart evidence and clean-candidate
  browser evidence, so it is technically accepted. The active step is
  `M1-S7-C7-ACCEPT`; Human Customer acceptance is still required before any C8
  attachment product write begins.

## 2026-08-15 — M1-S7-C7 Customer Accepted; C8 activated

- The Human Customer explicitly replied `验收通过` after the C7 deterministic
  and clean-candidate browser evidence was presented.
- `M1-S7-C7` is Customer Accepted. The sole active step advances to
  `M1-S7-C8-RED` under approved plan v7.
- This acceptance does not mark the milestone complete: C8 attachment gates
  and C9 final clean-candidate/rollback acceptance remain required before
  `Milestone Accepted`.

## 2026-08-15 — M1-S7-C8 attachment RED proven

- A real file was created on disk and sent as multipart `file` plus
  `issue_id` through the assembled Canonical Runtime. The focused test
  `TestSQLiteRuntimeServesCanonicalIssueAttachments` failed with the observed
  response `404 page not found` from `POST /api/upload-file`; this proves the
  installed Runtime has no attachment upload boundary rather than merely
  lacking a unit-level provider.
- Core RED tests demonstrated two independent false-success paths: a Workspace
  upload response with string `size_bytes` was accepted, and malformed list /
  standalone metadata responses with numeric `content_type` were returned to
  callers. The focused Vitest run failed exactly 2 of 5 tests for these
  malformed success bodies.
- Canonical Space remains a generated Asset stub and registers no HTTP routes;
  `/api/config` still advertises `issue_attachments:false`. These are current
  state facts, not accepted empty behavior.
- C8-RED is complete. The sole active step advances to `M1-S7-C8-GREEN` for
  the approved Space SQLite/file providers, trusted transport boundary,
  Workspace Issue/Comment binding and exact Core schemas. No `server/**`
  change is authorized.

## 2026-08-15 — M1-S7-C8 attachment GREEN proven

- Canonical Space now owns additive SQLite asset/version metadata and a
  server-owned `<database>.files` object root. Uploads use bounded multipart
  reads, opaque object paths and atomic rename; preview/download responses
  enforce content disposition and MIME safety, and attachment metadata/list/
  delete routes reuse trusted Auth, Workspace membership and Cookie-CSRF or
  Bearer mutation semantics.
- Issue and Comment create/update paths bind complete attachment-ID bags only
  after Workspace-scoped target validation. Capability-off runtimes reject an
  explicitly supplied Issue or Comment attachment field instead of silently
  persisting an unsupported relationship. Issue, batch-Issue and Comment
  deletion prepare file quarantine with their SQLite dependent cleanup, restore
  files on rollback, preserve shared references, and delete the object only
  after the last retained reference is gone.
- Runtime tests cover real-file upload, list, metadata, preview, download and
  delete; exact auth/isolation/CSRF and capability-route coupling; size, MIME,
  path and strict multipart/JSON boundaries; binding rollback, shared refs,
  orphan reconciliation, restart/hash readback and post-commit realtime. The
  Comment deletion RED initially left Space metadata readable after its final
  reference; GREEN now proves trigger-forced rollback restoration, sibling
  Comment retention and final-reference cleanup.
- Core rejects malformed Canonical attachment successes without misclassifying
  them as legacy direct responses, validates `issue_attachments:changed`, and
  invalidates the authoritative per-Issue attachment cache. The enabled View
  renders upload/dropzone/comment attachment controls; the disabled View does
  not mount them.
- Sequential deterministic gates pass: backend `go test ./... -count=1`,
  `go vet ./...`, `go mod verify`; Core 90 files / 587 tests plus typecheck and
  lint; focused Views 3 files / 51 tests plus Views lint/typecheck; Web
  typecheck; selector/verifier 22/22; changed-file gofmt, generated diff,
  diff-check, legacy-import scan and the permanent `server/**` boundary. Views
  lint reports three pre-existing warnings outside the C8 paths and no errors.
  One initial all-at-once host-contention run returned a transient upload 500;
  the exact attachment binding test then passed 20 repeated runs and the
  required sequential full-backend run passed.
- C8 product GREEN is proven. The sole active step advances to
  `M1-S7-C8-INTEGRATE` for scoped commit, clean-candidate real-browser upload/
  preview/download/delete, restart/hash evidence and independent review. No
  Human Customer acceptance or C9 authorization is implied.

## 2026-08-15 — M1-S7-C8 clean-candidate browser and retained-file evidence

- The exact product candidate is commit
  `0e0ab303b608006e15e35727fb4bd46c9ed42ed4`, checked out detached at
  `F:\code\ai\goclaw-team-runtime-c8-clean-0e0ab30`. The detached checkout is
  clean after evidence extraction and excludes the user's unrelated Input and
  stat-dirty View files. Both tracked and untracked `server/**` remain empty.
- The retained database copied from the main runtime collided with part of the
  fixed browser fixture. The fixture correctly refused to overwrite it; that
  database is preserved as `data\multica-canonical.pre-c8-conflict.db` with
  SHA-256 `E6AF9D7BBEA7DD1C934933EAEDEB342B96E955609AAE7AC9362E7CBD0B146D83`.
  A fresh Canonical database was migrated by the real Runtime and then seeded
  through the explicit non-overwriting fixture. After real upload and before
  restart its quiescent hash was
  `B2B05A2E0C780C176109B6D4123E17049DFAD9AC4BD6C6F5B825CA1BD08E2369`.
- The in-app browser proved the real login and full Issue detail exposed both
  upload controls. Because that browser interface has no supported file chooser
  operation, the file interaction continued with repository Playwright and the
  installed Chrome executable. Chrome uploaded the real local text file
  `c8-clean-candidate.txt`, rendered its exact preview, completed a browser
  download with the canonical filename, and received
  `issue_attachments:changed`. An initial harness assertion incorrectly treated
  the frozen top-level attachment array as an envelope after a successful
  upload; the harness was corrected and the successful second run retained both
  synthetic IDs for explicit cleanup rather than hiding the first committed
  side effect.
- Before restart, the two opaque object files had SHA-256
  `D96301D4579A4C1BA233893D541BD757179CC1109E70B11D2827C11971C30E6B`
  and `7E2C965E2AFA0BA82F8D628DAC6A585FBC9AEC064D43C6E80EA38AB8CD3466F8`.
  After stopping and restarting the same Runtime/database/object root, Chrome
  received metadata, content, download and Issue-list responses with status
  200; downloaded bytes were exact and both object hashes were unchanged. One
  first readback attempt exceeded a five-second UI wait during cold Next
  compilation; the bounded wait was corrected to thirty seconds and the same
  retained runtime then passed without a product retry or data rewrite.
- After the Human Customer explicitly replied `确认删除`, Chrome deleted only
  synthetic attachment IDs `8293a954-6d69-4524-8c13-14741b340fb3` and
  `e51d6b6d-23bc-4fe5-b0c8-7d3de0ed3914`; both DELETE responses were 204. The
  selected metadata read then returned the expected hidden 404, the Issue
  attachment list returned 200 without either ID, and the object root contained
  zero regular files. The stopped final database hash is
  `0FCB8AD2B94A6BDADCC32A9B3D034D75EA27782A42E61AFE3F64FC478C40FF30`.
- The sanitized upload, restart-readback and delete traces are retained under
  `F:\code\ai\goclaw-team-runtime-c8-evidence-0e0ab30\c8-evidence` with
  SHA-256 values `67F9FD48EF751F20F5199D46AC9BFBF57E7CD2DBCE1C920E224B3A936FF72A1C`,
  `D4DE84CEF07EAEB82CC542B1E9FDD1D1FB0254598009E386354D1869C645A6B6`,
  and `AB8FB7D9D2D05C6573964CF5AE9CC97CA5409F4789C1FEA0D3944FEBC05DDD54`.
  All browser requests use origin `127.0.0.1:3000`; no HTTP or WebSocket URL
  uses legacy port 8080. The trace still records the previously disclosed
  out-of-scope `/api/invitations` 404 and the intentional post-delete attachment
  404; neither is reclassified as attachment success or concealed.
- The selector stopped Canonical Web/HTTP/gRPC cleanly and ports 3000, 8000,
  8080 and 9000 are all closed. The executable C8 phases now live in the
  approved `e2e/canonical-runtime.spec.ts`, and Playwright discovers all six
  Canonical tests. C8 deterministic and clean-candidate integration evidence is
  complete, but the required independent review is still pending. Therefore
  the active step remains `M1-S7-C8-INTEGRATE`; C8 is not Customer Accepted and
  C9 remains unauthorized.

## 2026-08-15 — C8 independent review failed; plan v8 approved

- The Human Customer authorized independent C8 review and conditionally directed
  C8 acceptance/C9 startup only after that review passed. Two independent
  read-only Navigators instead returned `CODE FAIL` and `SPEC FAIL`; the
  condition did not become true, so no C8 acceptance or C9 activation occurred.
- Code P1 findings: Issue attachment validation and the repository write occur
  in different transactions, allowing a concurrent delete/upload or ordinary
  stale Issue update to create a dangling or lost reference; and IssueDetail
  submits only the current session's pending IDs while the backend treats the
  field as a complete replacement bag.
- Specification/evidence P1 findings: stored attachment endpoints ignore the
  request's trusted Workspace context; and the retained deletion evidence used
  Playwright APIRequestContext rather than a visible UI/Core delete control.
- Supporting P2 findings: capability-off Issue create accepts an explicit empty
  attachment array; traces do not retain installed-Chrome executable/user-agent
  identity; and the required realtime event allowlist change touched
  `backend/internal/realtime/hub.go`, which plan v7 had not listed explicitly.
- The Human Customer then explicitly approved plan v8. The immutable repair
  contract is `plan_v8.md`; it authorizes only the exact C8 backend/Core/View/
  E2E repairs and the single realtime allowlist file. The sole active step is
  `M1-S7-C8-REPAIR-RED`. C9 remains inactive until RED, GREEN, integration,
  clean-candidate browser evidence and both independent re-reviews pass.

## 2026-08-15 — M1-S7-C8 repair RED and GREEN proven

- Focused RED tests reproduced every approved plan-v8 defect before product
  changes: a stored attachment read without Workspace input returned 200;
  capability-off Issue create accepted an explicit empty `attachment_ids`;
  an ordinary stale Issue update overwrote a concurrently committed attachment
  bag; and an explicit replacement succeeded after the authoritative bag had
  changed. The frontend RED showed autosave submitted only pending IDs and had
  no visible persisted-attachment delete mutation.
- Stored attachment metadata, preview, download and delete now authenticate
  first, require trusted Workspace selection and hide selected/stored Workspace
  mismatches. A user who belongs to both Workspaces cannot use Workspace A
  context to read or delete a Workspace B attachment, and a wrong-context
  delete leaves the retained asset untouched.
- Issue create and explicit attachment replacement now validate exact Space
  references through the caller-owned `BEGIN IMMEDIATE` transaction. Ordinary
  Issue updates omit `asset_ids` from the SQL write and therefore preserve the
  authoritative bag; explicit replacement compares the caller's expected bag,
  rejects drift with exact 409 `issue attachments changed`, and rolls back
  Workspace and Issue writes if Space validation fails. Capability-off create
  distinguishes an omitted field from an explicitly supplied empty array.
- Issue detail waits for the authoritative attachment query before enabling
  upload, merges retained IDs with referenced pending uploads for description
  autosave, and exposes an accessible persisted-attachment delete control. The
  Core mutation uses the strict attachment client and invalidates both the
  attachment list and Issue detail. The clean-candidate E2E now requires an
  explicit installed-Chrome path, records executable and user agent, uploads a
  second retained attachment after restart, and deletes through the visible
  control rather than APIRequestContext.
- Deterministic GREEN gates pass on the repaired working tree: backend
  `go test ./... -count=1`, `go vet ./...`, and `go mod verify`; Core 91 files /
  588 tests plus lint/typecheck; focused Views 3 files / 50 tests plus
  lint/typecheck; Web typecheck; and Playwright discovery of all six Canonical
  tests. Views lint reports only three pre-existing warnings outside C8 paths.
  `git diff --check` has no error and both tracked and untracked `server/**`
  remain empty.
- Repair GREEN is proven. The sole active step advances to
  `M1-S7-C8-REPAIR-INTEGRATE` for an explicit scoped product commit, detached
  clean-candidate real-browser evidence and both independent re-reviews. C8 is
  not yet accepted and C9 remains inactive.

## 2026-08-15 — M1-S7-C8 repair clean-candidate integration proven

- The repaired product commit is
  `be9aa3de49d3feae3f4478eba997f722e8739cbb`. Its first detached installed-
  Chrome run exposed one additional real integration defect rather than being
  retried as success: the native download hook supplied trusted
  `workspace_slug` in the query because an anchor cannot set the Core request
  header, while the repaired stored endpoint required only a header. Chrome
  therefore downloaded the 400 JSON error as `download.json`. The sole
  synthetic asset from that failed run was explicitly removed and its stopped
  database/logs were retained under
  `F:\code\ai\goclaw-team-runtime-c8-repair-evidence-be9aa3d`.
- Commit `4d26bf9b2ac536ba31f3d0bc8599fec74b7c9de4` adds the focused RED/GREEN
  contract for Cookie native download with `?workspace_slug=`. Only the
  download handler promotes that untrusted slug into the existing header seam
  when no Workspace header is already present; authentication still occurs
  before trusted slug/member resolution, missing Workspace remains 400 and
  cross-Workspace ownership remains hidden 404. The full backend test, vet and
  module-verification gates pass after this correction.
- A subsequent evidence-only run stopped before upload because the harness
  assumed two hidden file inputs even though both visible `Attach file`
  controls were mounted with one current input. Commit
  `c297216eaeeadfaa3dbc2231c4fb8e18f007acef` replaces that DOM-count heuristic
  with the first visible Issue-description `Attach file` control and the real
  browser file chooser. This exact commit is the final detached product
  candidate; its `@multica/core` and `@multica/views` links resolve inside the
  detached worktree rather than the user's dirty main checkout.
- Installed Headless Chrome 151 from
  `C:\Program Files\Google\Chrome\Application\chrome.exe` then passed all
  three repair phases. Upload used the visible control, previewed exact text,
  completed a native download with the canonical filename and received
  `issue_attachments:changed`. After a real Runtime stop/restart, metadata,
  content and download for A were exact; the same visible control uploaded B;
  and the authoritative list was exactly A+B, proving retained A was not
  overwritten. The final phase clicked each persisted row's visible
  `Remove attachment` control; both DELETE responses were 204, metadata became
  the expected hidden 404 and the Issue attachment list excluded both IDs.
- Synthetic IDs were `1c9920bf-88a4-4b42-9d14-8971d550a9c5` and
  `42cb3dd4-0464-4204-9b77-e2ec5b3a5dff`. Their exact byte SHA-256 values were
  `854CFF6B4E9CA0678D657E24D79DB088E03293B9DDE6D839CEEEBA721FEAB063`
  and `16E6B0CF96DA584477CEF8CDEB1DBB0704919EEEC2EEC3F465D7CB43B369DC23`.
  After cleanup the object root contains zero regular files and the stopped
  database SHA-256 is
  `9B8652EA751D4EA772F2F4CDEB452DC739AC9FB4ADADAB07850B69DBD8D85B7D`.
- Sanitized evidence is retained under
  `F:\code\ai\goclaw-team-runtime-c8-repair-evidence-c297216\c8-evidence`.
  Upload, restart-readback and visible-delete traces contain 113, 114 and 114
  HTTP responses; every recorded origin is `http://127.0.0.1:3000` and none
  uses port 8080. Their SHA-256 values are respectively
  `CE4C3CF40006C57E5DC417D33ED95958BABB762C9E8122ABB89685ACD12506AC`,
  `9F1F40EE2D7ABD57A96C4D99B1B66EF070DCB2DF030043EAC19B31A60CB01A86`
  and `89A41499F906BE30ECCF96A824ABE70657EB77463FAF66C87B825F34EE9CA041`.
  The selector is stopped, reports Canonical selected with previous legacy,
  and ports 3000, 8000, 8080 and 9000 are all closed.
- Clean-candidate repair integration is proven, but both independent read-only
  re-reviews are still required. The active step remains
  `M1-S7-C8-REPAIR-INTEGRATE`; C8 is not yet accepted and C9 remains inactive.

## 2026-08-15 — M1-S7-C8 independent findings repaired and final evidence retained

- The first plan-v8 independent re-reviews did not pass. Code review found that
  explicit `attachment_ids:null` was still indistinguishable from omission
  while the capability was disabled, and that a successful same-page visible
  delete left the pending-upload ref able to re-submit the deleted ID on the
  next description autosave. Specification review found that the real shared
  attachment card handled pointer `onMouseDown` only, so Enter/Space did not
  activate its otherwise labelled delete button. C8 therefore remained
  unaccepted and C9 remained inactive.
- Focused RED/GREEN repairs now preserve JSON field presence for Issue and
  Comment create/update, so arrays, empty arrays and explicit null are all
  rejected when attachments are disabled. The Core delete mutation removes a
  successful delete from the authoritative attachment cache before
  invalidation, while IssueDetail removes it from the pending ref/state only on
  success. The upload-bind-delete-same-page regression proves the next edit
  submits an empty complete bag rather than resurrecting the deleted ID.
- The keyboard repair was deliberately relocated out of the shared editor and
  into the plan-authorized Issue attachment wrapper. Pointer activation remains
  owned by the editor, while keyboard/screen-reader generated clicks are
  captured once by the Issue wrapper. A real component test proves both Enter
  and Space. The cumulative repair diff therefore contains no
  `packages/views/editor/**`, `server/**`, or unrelated dirty Input/table/create
  paths.
- A stopped detached run at commit `7cb557f` timed out waiting for the upload
  realtime frame before the harness explicitly awaited a Workspace socket; its
  sole synthetic attachment
  `2055e94b-e162-42bf-8445-ed62b3cca597` was explicitly deleted and the failed
  DB/log evidence was retained under
  `F:\code\ai\goclaw-team-runtime-c8-repair-evidence-7cb557f`.
  A later exact `52f00d7` run uploaded/previewed/downloaded successfully but
  again missed the realtime frame; synthetic attachment
  `ec4ca862-9afa-49e5-b378-d6b6e98b0cf4` was explicitly deleted and the stopped
  failed-run DB was retained under
  `F:\code\ai\goclaw-team-runtime-c8-repair-evidence-52f00d7`.
- Commit `8488898` added a bounded, self-cleaning cookie-WebSocket readiness
  probe because the frozen cookie handshake intentionally has no `auth_ack`.
  The probe event was received, yet the measured upload-bind still emitted no
  `issue_attachments:changed`; this proved a product event gap rather than a
  listener race. Synthetic attachment
  `d62cf8ec-7d80-4ce4-bf0a-cdc10799ed0b` was explicitly deleted and the stopped
  failed-run DB was retained under
  `F:\code\ai\goclaw-team-runtime-c8-repair-evidence-8488898`.
- A new real Runtime RED then reproduced the exact editor sequence: unbound
  upload followed by an atomic Issue `attachment_ids` bind produced only
  `issue:updated`. Commit `55a18e7f775890eac1a0cabc99530f02ca4feb7f`
  wires the existing attachment projection into the Issue publisher and, only
  after a successful bind commit and changed bag, publishes the complete
  `issue_attachments:changed` snapshot. The RED is GREEN with deterministic
  event order `issue:updated` then the complete attachment bag.
- One deliberately over-parallelized aggregate gate run caused SQLite 5-second
  lock timeouts and is retained as a failed orchestration experiment, not a
  product pass. The exact acceptance commands were then run sequentially:
  backend `go test ./... -count=1`, `go vet ./...`, and `go mod verify` pass;
  Core 91 files / 589 tests plus typecheck/lint pass; focused Views 2 files /
  50 tests plus typecheck/lint pass (three pre-existing warnings, zero errors);
  Web typecheck and all-six Playwright discovery pass; the attachment
  concurrency test passes ten consecutive counts. `git diff --check` has no
  error and tracked/untracked `server/**` remain empty.
- Exact detached candidate `55a18e7f775890eac1a0cabc99530f02ca4feb7f`
  passed all three installed Headless Chrome 151 phases against a fresh DB.
  Upload A used the visible Issue description control, previewed exact text,
  completed a native download, and received the committed complete attachment
  bag after a one-attempt readiness probe. A real stop/restart retained A's
  exact bytes; uploading B retained authoritative A+B. Persisted A was deleted
  with Enter and B with Space through the real labelled controls; both DELETEs
  were 204, post-delete metadata was hidden 404 and the object root contained
  zero regular files.
- Synthetic IDs are `2ca44ca4-4e15-4dc0-92ae-db22c957792c` and
  `a59c219f-5c26-4e4d-b6f9-ae7cb6f3a493`. Their byte SHA-256 values are
  `D2C16F2B698D3938AEA6C32F3461061BB07C5E30354B8ED5BDD56B01F35B94E3`
  and `8C6101456CFBEB47B1A061BF6B422112D99DDFA55F828A99A469B40F72A44F12`.
  Upload, restart-readback and keyboard-delete traces contain 168, 132 and 111
  HTTP responses, only origin `http://127.0.0.1:3000`, and zero port-8080 URLs.
  Their SHA-256 values are respectively
  `21E878A074F8CF5AB0AC83582E80BC50CA30DC759A05C9C2D3C9A70F4829005B`,
  `3CBCE221CFE8ABCFF19CEA6DBF637C609DA551D32ACE1EB3F31DF708C6BCB336`
  and `62225203A9F04CAF201C6E7A633FDD12710C37CBFFF32BDBF1B9321CCA994F1A`.
  The stopped database SHA-256 is
  `A0EA1A0BAC3CE37BC88D85DDEE9EB9A269F75B7067FD7A0265605796767FCBB8`.
- Sanitized traces and final stopped DB/object/log artifacts are retained under
  `F:\code\ai\goclaw-team-runtime-c8-repair-evidence-55a18e7`. The detached
  candidate is clean; the selector reports Canonical selected with previous
  legacy; ports 3000, 8000, 8080 and 9000 are all closed. The main worktree's
  pre-existing unrelated dirty files remain excluded. C8 is still not accepted
  and C9 remains inactive until both final independent re-reviews return no
  P0-P2.

## 2026-08-15 — M1-S7-C8 Customer Accepted; C9 RED activated

- Independent CODE/SECURITY review returned `CODE PASS` for exact product
  candidate `55a18e7f775890eac1a0cabc99530f02ca4feb7f`, with P0=0, P1=0 and
  P2=0. Its independent gates included full backend tests, focused attachment
  concurrency/event tests at ten counts, vet, module verification, Core and
  Views focused tests/typechecks, six-test Playwright discovery, scope checks
  and retained Chrome trace verification.
- Independent SPEC/EVIDENCE review returned `SPEC PASS`, with P0=0, P1=0 and
  P2=0. It independently verified the clean detached candidate, plan-v8 path
  boundary, null/omission semantics, successful pending/cache cleanup,
  keyboard Enter/Space deletion, committed complete attachment events, final
  hashes, database integrity, zero retained objects, closed fixed ports and
  separation of all failed runs from the final evidence set.
- The Human Customer had explicitly authorized independent C8 review and
  conditionally directed C8 acceptance and C9 startup after the review passed.
  Both required PASS conditions are now true, so `M1-S7-C8` is Customer
  Accepted and the sole active story advances to `M1-S7-C9-RED`.
- This story transition is not final milestone Customer acceptance. C9 must
  still prove the complete local capability surface, clean-candidate journey,
  retained restart behavior, no-legacy process/network evidence and reversible
  rollback before a final milestone acceptance request may be made.

## 2026-08-15 — M1-S7-C9 capability RED and GREEN proven

- C9 mounted the full current Issue-detail consumer matrix in a focused View
  test and froze the Runtime capability contract in a real `/api/config` test.
  The first RED failed only because `issue_pins` and `issue_project` were false;
  every other local Issue-detail capability was already true and the external
  `issue_detail_pull_requests` integration was correctly false.
- A second RED updated the Canonical runtime verifier from its older partial
  capability snapshot. It rejected the current accepted attachment/catalog
  matrix because it still expected those flags to be false.
- Commit `2fc9525` enables only the already implemented and accepted local Pin
  and Project consumers, keeps external pull requests disabled, and aligns the
  verifier with all C4-C8 accepted local Issue capabilities. The full-detail
  test observes timeline, reactions, subscribers, attachments, Pins, Project,
  hierarchy/progress, labels, properties and acceptance consumers while proving
  the external pull-request query remains unmounted.
- GREEN evidence: the exact Runtime capability tests pass; focused Workspace
  and bootstrap Go tests pass; the IssueDetail suite passes 47/47; Views
  typecheck passes; and the selector/verifier suite passes 22/22. Scope and
  diff checks contain no `server/**` path and exclude all pre-existing unrelated
  dirty files.
- The sole active step advances to `M1-S7-C9-INTEGRATE`. C9 is not technically
  or Customer Accepted: full deterministic gates, a detached clean-candidate
  installed-Chrome journey, retained restart and realtime evidence, quiescent
  rollback hashes, independent final review and explicit Human Customer
  milestone acceptance remain required.

## 2026-08-15 — M1-S7-C9 retained attachment-concurrency gate failure

- Exact clean candidate `0888efc` completed the installed-Chrome C9 journey,
  restart/readback and rollback proof, but the required Backend full gate
  `go test ./... -count=1` failed in
  `TestSQLiteRuntimeConcurrentAttachmentUploadsLoseNoReferencesOrFiles`.
  Twelve authenticated uploads to one Issue expected twelve successful
  attachment responses, durable references and files; one response instead was
  `500 {"error":"attachment operation failed"}`.  The failure is retained as a
  C9 integration blocker and is not reclassified by later passing repetitions.
- A diagnosis-only focused run of the same test passed twenty consecutive
  counts.  This confirms intermittency but is not a replacement for the failed
  full-gate evidence.  No product code changed during diagnosis.
- Under plan-v9 RED, the new repository-level deterministic contention test
  held twelve `BEGIN IMMEDIATE` transactions for 550ms each against a
  16-connection file SQLite database.  Before the repair it failed after
  5.46s with the classified output
  `begin attachment creation: database is locked (5) (SQLITE_BUSY)`.  The test
  also requires its contention window to exceed the original five-second
  SQLite wait, so it cannot silently become a no-contention pass.
- The Human Customer confirmed execution.  Approved plan v9 records the narrow
  `M1-S7-C9-ATTACHMENT-CONCURRENCY-RED` repair authority: classify the failure,
  add deterministic RED evidence, make only a bounded Canonical SQLite
  transaction repair if justified, and re-run independent review before C9
  integration can resume.

## 2026-08-15 — M1-S7-C9 attachment-concurrency GREEN pending review

- `AttachmentRepository.writeConnection` now gives `BEGIN IMMEDIATE`
  acquisition a shared eight-second, caller-context-aware budget.  It retries
  only classified SQLite `BUSY`/`LOCKED` acquisition failures, at most twice
  with 20ms/40ms backoff.  It closes failed acquisition connections and never
  retries insert, bind, storage, constraint, rollback or commit failures.
- The deterministic repository RED is GREEN three consecutive times (each
  crosses the original five-second lock wait); the real 12-request Runtime
  upload/list/file contract passes ten consecutive counts.  Focused Canonical
  Space/Workspace/bootstrap/server tests, the complete Backend test suite,
  vet and module verification pass after the final test assertion.
- The repair remains uncommitted and C9 remains blocked.  Independent
  code/security and specification/evidence reviews must return no P0-P2 before
  a later approved plan entry can reactivate C9 integration.

## 2026-08-15 — M1-S7-C9 attachment-concurrency clean-candidate integration

- The preceding uncommitted status was accurate when that GREEN checkpoint was
  recorded.  The exact repair candidate is now committed as
  `606ce6524f0836f45b95783406a0d0ad244fedc9`
  (`fix(runtime): bound attachment SQLite contention`).  It contains only the
  v9 plan/journal, the Space attachment repository and its focused repository
  test; no `server/**` path or pre-existing user dirty path is included.
- A new detached clean candidate at
  `F:\code\ai\goclaw-team-runtime-c9-lock-clean-606ce65` was created directly
  from that hash.  Its HEAD was `606ce6524f0836f45b95783406a0d0ad244fedc9`,
  its status and tracked/untracked `server/**` scope were empty, and
  `git diff --check` passed.
- In that clean candidate, `go test ./... -count=1`, `go vet ./...`, and
  `go mod verify` all exited successfully.  This is the v9
  `ATTACHMENT-CONCURRENCY-INTEGRATE` evidence; it neither replaces nor claims
  the later C9 browser/integration acceptance.
- `plan.md` now advances to the sole v9
  `M1-S7-C9-ATTACHMENT-CONCURRENCY-REVIEW` gate.  The product repair remains
  blocked from reactivating C9 integration until independent CODE/SECURITY and
  SPEC/EVIDENCE reviews both report no P0-P2 and a later approved plan entry
  selects the next C9 step.

## 2026-08-15 — M1-S7-C9 attachment-concurrency independent review complete

- Independent CODE/SECURITY review of the v9 candidate returned PASS with
  P0=0, P1=0 and P2=0.  It confirmed that the retry is `BEGIN IMMEDIATE`
  acquisition-only, classifies extended SQLite BUSY/LOCKED codes, respects the
  shared eight-second caller-context budget, closes failed connections, and
  never retries insert/bind/storage/constraint/commit work.  It independently
  reran the deterministic contention test three times and the Runtime
  same-Issue upload contract three times.
- Independent SPEC/EVIDENCE review returned PASS with P0=0, P1=0 and P2=0.
  It verified the exact product candidate `606ce6524f0836f45b95783406a0d0ad244fedc9`,
  the docs-only evidence/state commit
  `e3cd960e15956091d6fe5e96273bacfa351e622a`, clean candidate scope/status,
  retained failure/RED separation, the journal-indexed full Backend gates and
  the Windows CRLF-only `gofmt -d` checkout noise against byte-equivalent
  formatted Go blobs.
- `M1-S7-C9-ATTACHMENT-CONCURRENCY-REVIEW` is complete.  There is no active
  product step: v9 explicitly requires a later approved plan entry before C9
  integration may resume.  This repair completion is not C9 technical or
  Customer acceptance, and it does not authorize a milestone acceptance claim.

## 2026-08-16 — C9 integration reactivated under plan v10

- The Human Customer approved continuation (`批准进行`).  Immutable
  `plan_v10.md` supersedes v9 and selects only
  `M1-S7-C9-INTEGRATE-PREFLIGHT`; no product implementation authority is
  re-opened.
- The v10 base is `06a2273c270019e2e6ba3a449e5d8f59a9df69bd`; it includes the
  v9 repair `606ce6524f0836f45b95783406a0d0ad244fedc9`.  A fresh detached
  candidate will be created from the v10 evidence commit before any final C9
  browser evidence is collected.
- Preflight inventory identified a stale, repair-free `0888efc` selector tree
  on local ports `3000/8000/9000`.  Its verified supervisor and children must
  be stopped before the fresh candidate is started.  Port `8080` has no local
  listener; unrelated outbound host traffic is not legacy-runtime evidence and
  is out of scope.
- The primary worktree retains unrelated dirty UI and local-artifact paths.
  They are excluded from both the v10 candidate and its browser evidence.
- Story-map and milestone wording that still called C9 generically “active” is
  superseded by `plan.md`: only the v10 preflight step is active.  C9 technical
  and Customer acceptance remain pending.

## 2026-08-16 — M1-S7-C9-INTEGRATE-PREFLIGHT complete

- Exact v10 planning commit is
  `f6a1f67ca64c8064418fc67fe2c82293901ed38a`; its `plan_v10.md` SHA-256 is
  `E1E0E37E337864688029EFAEB947A219B88451C60513C34D69B325B36B953AB8`.
  Detached clean candidate
  `F:\code\ai\goclaw-team-runtime-c9-v10-clean-f6a1f67` was created directly
  from that commit with empty status, empty tracked/untracked `server/**`
  scope and a passing `git diff --check`.
- Before any new candidate startup, the older manifest-owned launcher tree in
  `F:\code\ai\goclaw-team-runtime-c9-clean-0888efc` was verified as the owner
  of local `3000/8000/9000` through recorded parent/child processes
  (`go.exe`/`server.exe` and Node/Next).  It was stopped through that exact
  worktree's `node scripts/runtime-selector.mjs stop`, which reported
  `{"stopped":true,"mode":"canonical"}`.
- Post-stop inspection found no local listener on `3000`, `8000`, `8080` or
  `9000`, and none of the recorded launcher/child PIDs remained.  The absence
  of a local `8080` listener is scoped to the Canonical/legacy runtime; it does
  not make a host-wide claim about unrelated outbound software.
- The old candidate's attachment repository has no v9 retry helper, so its
  historical Chrome/rollback artifacts remain retained evidence only.  They
  cannot be reused as the final repaired C9 candidate proof.
- `M1-S7-C9-INTEGRATE-RED` is now the sole active step.  It may add a tracked
  C9 acceptance scenario and repair selector/verifier evidence handling only
  within v10's authorized boundary; any product behavior failure stops the
  plan and requires a new version.

## 2026-08-16 — M1-S7-C9-INTEGRATE-RED complete

- Exact evidence-gate commit is
  `bb9e44d` (`test(runtime): add C9 clean-candidate evidence gate`).  It changes
  only v10-authorized `e2e/canonical-runtime.spec.ts` and
  `scripts/canonical-runtime-verifier*`; committed `server/**` scope is empty.
- The new verifier test first failed because
  `captureArtifactHashes` omitted
  `multica-canonical.db.files/objects/attachment.txt`.  The smallest GREEN
  recursively hashes regular files below a `*.db.files` object root using
  normalized relative names; the focused selector/verifier suite then passed
  23/23.
- The tracked C9 pre-restart scenario was run while the verified old runtime
  was quiescent.  It failed as intended at
  `page.goto(http://127.0.0.1:3000/login)` with
  `net::ERR_CONNECTION_REFUSED`; this is a harness/startup RED, not a product
  failure.  `pnpm exec playwright test e2e/canonical-runtime.spec.ts --list`
  now discovers eight tracked scenarios, including C9 pre/post restart.
- The C9 scenario records API-assisted state setup separately from visible
  login/detail/attachment interaction, captures HTTP and WebSocket origin/path
  plus received event types, includes Project and Pin readback, and rejects all
  local failures except the explicitly deferred `/api/invitations` 404.
- `M1-S7-C9-INTEGRATE-GREEN` is now the sole active step.  It may start only a
  fresh clean candidate from the exact evidence-gate commit and must stop on a
  missing local control/route, capability, persistence or realtime failure.

## 2026-08-16 — C9 evidence-scenario development validation

- Development-only candidate `2039a6e` exposed three test-harness assumptions,
  not product failures: the full detail surface no longer renders the retired
  `issue-base-detail` test ID; unauthenticated bootstrap probes precede the
  cookie login; and the C8 restart test incorrectly required the complete
  Issue attachment list to contain only its two synthetic IDs while C9 validly
  retained another attachment.
- `88ca4ea` (`test(runtime): stabilize C9 evidence trace`) removes only those
  assumptions.  Its C9 gate starts failures after authenticated detail entry,
  asserts current visible controls/text, and requires the C8 synthetic IDs to
  be present without treating independent retained attachments as a failure.
  It changes no product behavior or backend/API route.
- On the development candidate, installed Chrome passed C9 pre-restart and
  post-restart scenarios, C8 upload/preview/download, C8 retained readback and
  C8 visible keyboard deletion.  Canonical verifier mutation and retained
  metadata readback also passed.  This validation is intentionally not final
  evidence because it predates the `88ca4ea` clean candidate and its retained
  database/traces will not be reused.
- The development runtime was stopped through its own selector and the fixed
  local ports were again quiescent.  The next action remains
  `M1-S7-C9-INTEGRATE-GREEN`: create an exact fresh clean candidate containing
  `88ca4ea`, then repeat deterministic and live evidence without reusing the
  development artifacts.

## 2026-08-16 — M1-S7-C9-INTEGRATE-GREEN and VERIFY complete

- The final browser/evidence candidate is detached clean commit
  `fdb3a3e76b940bb57247e7564d1ac4e4b26f7aef` at
  `F:\code\ai\goclaw-team-runtime-c9-v10-final-fdb3a3e`.  Its status was empty
  before startup; its committed range from the v10 base has no `server/**`
  path, its tracked/untracked `server/**` scopes are empty, and `git diff
  --check` passes.  The primary worktree's dirty Input/table/create-modal and
  local-artifact paths were not present in the candidate.
- Three evidence-only commits after the earlier development checkpoint fixed
  stale detail selectors and delayed C9 route-trace capture until the
  authenticated page reached network idle:
  `1586eadf7240b27884cbb4938d88f7bd450c1305`,
  `5d8a48d68b78f2892388339843931bbfa9ff23d2`, and the final
  `fdb3a3e76b940bb57247e7564d1ac4e4b26f7aef`.  They change only the v10
  authorized Playwright evidence path; no product route, capability or backend
  behavior changed.
- Final deterministic gates on that exact candidate passed: focused Auth,
  Workspace, Bootstrap and server Go tests; `go test ./... -count=1`; `go vet
  ./...`; `go mod verify`; Core tests `91 files / 589 tests`, Core typecheck
  and lint; Views focused tests `2 files / 49 tests` and typecheck; Web
  typecheck; selector/verifier tests `23/23`; and Playwright discovery `8`
  scenarios.  No Go path changed in the v10 range, so `gofmt -d` had no
  candidate Go input.  Windows race exit `0xc0000139` was not run or claimed.
- The final candidate selector first started/stopped the empty migrated
  database, then the quiescent fixture command created
  `canonical-fixture@multica.local`, Workspace `canonical-fixture` and Issue
  `CAN-1`.  It subsequently owned only local `3000/8000/9000`; `8080` had no
  local listener.  Health, readiness and Web root returned `ok`, `ready` and
  `200` at each required startup.
- Installed system Chrome (`C:\Program Files\Google\Chrome\Application\chrome.exe`,
  Playwright user agent `HeadlessChrome/151.0.0.0`) passed all final journeys:
  the three baseline login/Workspace/Issue/Projects scenarios; C9 complete
  pre-restart detail; C8 upload/preview/download; a same-database restart plus
  verifier retained metadata readback; C9 post-restart detail; C8 retained
  attachment readback; and C8 visible synthetic-attachment deletion.
- Sanitized final C9 traces are
  `.local-runtime/c9-v10-evidence/c9-pre-restart.json`
  (`9C7659E2272928AEE47F92C2A794575BA6C19E2A804855CFEE97273864EF9CF8`)
  and `.local-runtime/c9-v10-evidence/c9-post-restart.json`
  (`9A07412EAD97E020678DD414E89B51CB2BE3921CE37AC3CFD7D064699D3902B3`).
  Both name the exact final candidate, use only same-origin
  `127.0.0.1:3000` HTTP/`ws://127.0.0.1:3000/ws`, contain no `:8080` request,
  and prove the frozen edit, move, hierarchy, collaboration, labels,
  properties, acceptance, attachment and realtime state.  The only C9 trace
  failures are the explicitly deferred `/api/invitations` 404 probes.
- Final C8 trace hashes are upload
  `2EEEE49C3377DD6EBFCB6DAE48CE06D2419CC06F5C04C5D319ACF2C8128F07EE`,
  restart readback
  `083D995D64A5D88D345BFD08BAA8C36C3A9DA105212E5D931FA8FABB0B8BF976`, and
  visible deletion
  `20CD43D49DBD0DA8C13C468D5A0C4B859F6F59A95DCDFAFD3DB305233EEEC517`.
  Its pre-login `/api/me` and `/api/workspaces` 401s, deferred invitations
  404s, and the expected post-delete attachment 404 are classified evidence,
  not missing Canonical routes.
- Verifier mutation value `retained-1786820191090` read back after the first
  restart and again after selector rollback/reselect/restart.  While quiescent,
  snapshot and preserved checks reported identical SHA-256 values for
  `multica-canonical.db`, canonical backend/Web logs, and the retained asset
  object
  `multica-canonical.db.files/01990000-0000-7000-8000-000000000003/025d2518-00ef-44e8-98fb-98a83909ced7/706ab8d0-23dd-44bb-b785-0ae32dce015b.blob`.
  The selector was restored without starting legacy and the final candidate
  runtime was stopped; `3000/8000/8080/9000` are all quiescent.
- `M1-S7-C9-INTEGRATE-GREEN` and `M1-S7-C9-INTEGRATE-VERIFY` are complete.
  The sole active step is now `M1-S7-C9-REVIEW`; no Customer acceptance is
  inferred or recorded by this entry.

## 2026-08-16 — M1-S7-C9 v10 final review blocked

- The preceding v10 entry retains the exact clean-candidate execution outputs,
  but does not constitute acceptance.  Independent CODE/SECURITY and
  SPEC/EVIDENCE reviews both returned FAIL with P0=0, P1=2 and no independent
  P2.  The final candidate runtime was already stopped and local
  `3000/8000/8080/9000` listeners remain quiescent.
- P1: final C9 traces contain four pre-restart and one post-restart
  `GET /api/invitations -> 404` responses, and the Web log/screenshot records
  the resulting API error.  `c9UnexpectedFailures` incorrectly allowlisted
  that local route.  The immutable v10 acceptance contract says a local 404 or
  missing endpoint stops the plan; the journal cannot create a deferred-route
  exception after approval.
- P1: the C9 HTTP trace listener filtered responses to the Canonical Web origin
  before asserting no `:8080`, did not observe `requestfailed`, and cleared an
  authenticated post-restart load window.  It therefore cannot prove zero HTTP
  legacy traffic.  The existing WebSocket capture does not close that HTTP
  evidence gap.
- No product endpoint, capability or frontend behavior may be changed under
  v10.  The next authorized action is an approved `plan_v11.md` that narrowly
  decides how Invitations is capability-gated or implemented and how the C9
  network gate records all requests/responses/failures before filtering.  It
  must then reproduce a fresh clean-candidate Chrome/restart/rollback run and
  independent reviews.  C9 and the milestone remain unaccepted.
