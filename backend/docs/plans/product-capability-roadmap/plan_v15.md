# Product capability roadmap implementation plan v15

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Version: `15`
- Status: `approved-for-execution`
- Active step: `PCR-S02B independent-review remediation`
- Task-ID: `PCR-001-S02B-R20`
- Task-Revision: `r020`
- Work-Item: `PCR-S02B`
- Base commit: `36b18b4afac2ca2ae65ced7ee329c726c0bed73b`
- Supersedes: `plan_v14.md` for active execution only
- Approved: `2026-08-18`

## Outcome and authority

The Human Customer directed continued approved execution through Release 1.
The v14/r019 implementation passed deterministic and installed-browser gates,
but fresh independent review blocked S02B closure on exact replay persistence,
client idempotency-key reuse, and strict nested Issue parsing. This version
activates only the bounded remediation needed to resolve those findings.

No S03A or later story, push, merge, deployment, or Release 1 completion is
authorized. `server/**` remains permanently read-only. `plan_v14.md` and all
older plans remain immutable evidence.

## Frozen remediation contract

1. The immutable Task-promotion domain row also persists a versioned response
   snapshot sufficient to reconstruct the originally committed Task and Issue.
   Exact replay returns that snapshot even after either live row is edited. The
   snapshot is domain data; titles and descriptions remain prohibited from
   governance audit, outbox, and idempotency envelopes.
2. Amend the unshipped paired 000012 migration and its tests in place. The
   response snapshot is non-null for every promotion row; empty-only down,
   uniqueness, restart, rollback, and migration-count behavior remain intact.
3. The Core promotion request accepts an optional client-only idempotency key,
   excludes it from the JSON body, and sends it as `Idempotency-Key`. The shared
   mutation/view retain one key for the same Task revision and completion
   decision across a failed/retried command, then discard it after success or
   when the decision changes.
4. The promotion response uses a strict nested Issue schema. Unknown Issue
   fields and malformed typed fields reject the mutation response; general
   Issue read compatibility remains unchanged.
5. Add RED/GREEN tests for replay after independent Task and Issue edits,
   replay after database reopen, reusable header/body separation, same-command
   mutation retry, and strict nested Issue rejection.

## Writable scope

- `backend/internal/modules/workspace/internal/infrastructure/sqlite/migrations/000012_task_issue_promotion.*.sql`;
- `backend/internal/modules/workspace/internal/infrastructure/sqlite/task_promotion_repository.go` and focused tests;
- `packages/core/types/task.ts`, `packages/core/tasks/**`, Task API boundary tests,
  and `packages/core/api/client.ts`;
- `packages/views/tasks/**` only if required for stable command-key ownership;
- current roadmap `plan.md`, `plan_v15.md`, `story-map.md`, `task-register.md`,
  and append-only `journal.md`;
- `e2e/tasks.spec.ts` only if an installed retry assertion is required.

All v14 dirty exclusions remain unchanged. Generated protobuf files, lockfiles,
manifests, unrelated tests, and every `server/**` path are out of scope.

## Ordered execution

### PCR-S02B-R20.1 — Activate review remediation

- Freeze v15 at exact blocked candidate `36b18b4`.
- Mark r019 `independent-review-blocked`; establish r020 as the sole active task.
- Revalidate policy hashes, excluded blob, empty `server/**` diff, and scope.

### PCR-S02B-R20.2 — RED/GREEN exact replay

- First prove replay changes after live-row edits under v14.
- Persist and load the immutable response snapshot inside the same governed
  transaction; prove edit isolation, reopen replay, rollback, and no duplicate
  governance rows.

### PCR-S02B-R20.3 — RED/GREEN client strictness and key reuse

- First prove unknown nested Issue fields are accepted and caller keys cannot
  be reused under v14.
- Tighten only the promotion Issue schema and preserve a stable command key
  across retries without serializing it in the body.

### PCR-S02B-R20.4 — Verify and close

- Run focused RED/GREEN tests, root typecheck/test, backend check/race, exact
  detached installed-Chrome acceptance, scope/hashes/process cleanup, and a
  fresh independent read-only review.
- Only a complete independent PASS may close r020 and S02B. S03A remains
  inactive pending a new versioned plan.

## Acceptance criteria

1. Immediate replay and replay after later Task/Issue edits or database reopen
   return the originally committed promotion result without another mutation.
2. The immutable snapshot is committed and rolled back atomically with Issue,
   link, Task, revision, audit, outbox, and idempotency rows.
3. A caller can reuse one explicit key; it is sent only in the header. The
   shared command path retains the key for same-command retry and rotates it
   after success or a changed command decision.
4. Unknown or malformed nested Issue fields fail closed at the promotion write
   boundary without changing general Issue-read compatibility.
5. Full deterministic gates, exact browser acceptance, scope checks, and fresh
   independent review pass with no waiver.

## Risks and rollback

The response snapshot duplicates mutable domain content by design so exact
replay remains stable; it must never be copied into safe governance envelopes.
Rollback reverts only r020 commits. The 000012 down migration remains
empty-only, and no rollback may delete populated promotion data, reset user
work, modify `server/**`, or rewrite an older plan.
