# Product capability roadmap implementation plan v16

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Version: `16`
- Status: `approved-for-execution`
- Active step: `PCR-S02B deterministic-gate remediation`
- Task-ID: `PCR-001-S02B-R21`
- Task-Revision: `r021`
- Work-Item: `PCR-S02B`
- Base commit: `e97c92b23423a89d6d731ab8ba40ca9a80858b63`
- Supersedes: `plan_v15.md` for active execution only
- Approved: `2026-08-18`

## Outcome and authority

The Human Customer directed continued approved execution through Release 1.
The v15/r020 remediation implementation passed focused backend/Core/Views
checks, exact-candidate forced typecheck, and exact-candidate forced frontend
tests. Backend `make check` then deterministically exposed a pre-existing
transport deadline defect in the concurrent attachment acceptance test, so
r020 cannot satisfy its no-waiver full-gate contract.

This version activates only the bounded runtime transport remediation required
to unblock r020 verification. It does not authorize S03A, any later story,
push, merge, deployment, or Release 1 completion. `server/**` remains
permanently read-only, and all earlier plan versions remain immutable evidence.

## Reproduction evidence and frozen contract

1. Environment: exact detached `e97c92b` candidate and also pre-r020 baseline
   `36b18b4`, Windows, Canonical SQLite runtime, 12 concurrent authenticated
   `POST /api/upload-file` requests bound to one Issue.
2. Expected: all 12 requests commit within the attachment repository's
   existing eight-second write-acquisition budget; no reference or file is
   lost.
3. Actual: the Kratos HTTP server's implicit one-second default cancels request
   contexts while requests wait for the serialized write slot or commit.
   Sanitized diagnostics report `context deadline exceeded`; the public route
   returns 500. The exact focused test fails on both `36b18b4` and `e97c92b`.
4. Configure the Canonical Runtime HTTP server with an explicit 30-second
   request timeout. Preserve request cancellation, all attachment transaction
   logic, the eight-second repository lock budget, endpoint contracts, and
   gRPC behavior.
5. Add a runtime transport contract test that proves an installed HTTP request
   receives the explicit budget rather than the upstream library default. The
   existing concurrent attachment acceptance is the behavioral RED/GREEN gate.

## Writable scope

- `backend/internal/bootstrap/runtime.go`;
- `backend/internal/bootstrap/runtime_test.go`;
- `backend/internal/bootstrap/issue_attachment_runtime_test.go` only if a
  focused assertion needs clarification; production attachment code is out of
  scope;
- current roadmap `plan.md`, `plan_v16.md`, `story-map.md`, `task-register.md`,
  and append-only `journal.md`.

All v15 product commits remain unchanged. Generated protobuf files, migrations,
lockfiles, manifests, frontend product code, unrelated tests, and every
`server/**` path are out of scope. Existing dirty exclusions remain unchanged.

## Ordered execution

### PCR-S02B-R21.1 — Activate deterministic-gate remediation

- Freeze v16 at exact candidate `e97c92b`.
- Mark r020 `verification-blocked`; establish r021 as the sole active task.
- Revalidate policy hashes, excluded dirty paths, and empty `server/**` diff.

### PCR-S02B-R21.2 — RED/GREEN explicit HTTP budget

- Preserve the captured concurrent attachment failure as RED and add the
  transport deadline contract test before implementation.
- Set only the Canonical Runtime HTTP request timeout to 30 seconds.
- Prove the deadline contract and 12-writer attachment acceptance pass without
  retries, skipped assertions, reduced concurrency, or repository changes.

### PCR-S02B-R21.3 — Verify and close

- Re-run r020 focused tests, forced root typecheck/test, backend check and real
  race, exact detached installed-Chrome promotion acceptance, scope/hashes and
  process cleanup, then obtain a fresh independent read-only review.
- Only a complete independent PASS may close r021, r020, and S02B. S03A remains
  inactive pending a new versioned plan.

## Acceptance criteria

1. The Canonical Runtime supplies an explicit 30-second HTTP request context
   deadline; it does not inherit Kratos's one-second default.
2. Twelve concurrent Issue attachment uploads pass with all 12 database
   references and files retained, without changing concurrency or retrying
   inside the test.
3. r020 exact replay, stable idempotency key, and strict nested Issue behavior
   remain green and unchanged.
4. Full deterministic gates, exact browser acceptance, scope checks, and fresh
   independent review pass with no waiver.

## Risks and rollback

A longer request budget can retain slow handlers longer, but 30 seconds is
bounded and still propagates client cancellation. It is deliberately longer
than the repository's eight-second internal acquisition budget. Rollback
reverts only the explicit runtime timeout and its focused contract evidence;
it must not alter attachment data, r020 product commits, user work,
`server/**`, or any older plan.
