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
