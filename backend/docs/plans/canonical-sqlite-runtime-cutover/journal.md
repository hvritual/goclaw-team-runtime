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
