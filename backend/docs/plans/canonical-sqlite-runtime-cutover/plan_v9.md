# Canonical SQLite runtime cutover — execution plan v9

- Plan-ID: `canonical-sqlite-runtime-cutover`
- Version: `9`
- Status: `approved`
- Approval source: Human Customer confirmation `确认执行` dated `2026-08-15`
- Supersedes: `plan_v8.md`
- Base commit: `0888efc`
- Branch and integration target: `codex/multica-six-domain-baseline`
- Active step: `M1-S7-C9-ATTACHMENT-CONCURRENCY-RED`
- Policy bundle: `backend-v1`
- XP mode: strict; maximum active stories: one

## Purpose

The clean C9 candidate successfully completed the installed-Chrome full-detail
journey, but its required full Backend gate failed.  In
`TestSQLiteRuntimeConcurrentAttachmentUploadsLoseNoReferencesOrFiles`, one of
twelve authenticated, same-Issue uploads returned
`500 {"error":"attachment operation failed"}`.  The focused test later passed
twenty consecutive counts, so it is an intermittent operational failure rather
than a completed acceptance result.  C9 integration is blocked until this
failure has a deterministic RED, a bounded Canonical fix, and fresh evidence.

This revision authorizes only that repair.  It does not reopen C8, enlarge the
C9 detail surface, or change an attachment API contract.

## Frozen repair contract

- Concurrent authenticated uploads to one authorized Issue complete without a
  generic 500 and retain every committed attachment ID, metadata row and object
  file exactly once.
- SQLite writer contention is handled only through an explicitly bounded,
  context-aware Canonical transaction policy.  It may retry a classified
  transient SQLite busy/locked acquisition; it must not retry validation,
  authorization, binding, storage, constraint, commit-ambiguity or arbitrary
  database failures.
- An unsuccessful upload leaves no metadata reference or orphan object.  A
  successful upload remains one atomic asset/version/binding transaction and
  still emits its post-commit attachment bag only after commit.
- The repair may instead reduce the local Canonical SQLite connection topology
  only if the same runtime concurrency contract, readiness, restart and
  rollback guarantees remain proven.  It may not conceal failures by returning
  fabricated success or dropping uploads.
- Existing attachment tenant, Workspace, CSRF, MIME/size/path, object cleanup,
  exact HTTP and realtime contracts remain frozen.

## Authorized write boundary

- `backend/internal/modules/space/**`
- `backend/internal/modules/workspace/**` only if a narrow shared SQLite
  transaction seam is required by the repair
- `backend/internal/bootstrap/**` only for Canonical SQLite configuration and
  focused runtime reproduction/evidence
- `backend/docs/plans/canonical-sqlite-runtime-cutover/**`

Explicitly excluded: `server/**`; all frontend/product-detail changes; C9
capability flags and routes unrelated to attachment upload; legacy runtime
trees; root selector changes; migrations; external integrations; and the
user's unrelated dirty Input/table/create-modal files and local artifact roots.

## Strict XP sequence

1. `M1-S7-C9-ATTACHMENT-CONCURRENCY-RED`: freeze the failed clean-candidate
   command, add a deterministic focused reproduction that exposes the actual
   classified failure, and prove the current implementation fails it.
2. `M1-S7-C9-ATTACHMENT-CONCURRENCY-GREEN`: implement the smallest bounded
   transaction/concurrency repair; prove all twelve writes retain complete
   references/files and preserve no-event/rollback behavior.
3. `M1-S7-C9-ATTACHMENT-CONCURRENCY-INTEGRATE`: run focused and full Backend
   deterministic gates from a clean candidate.  Do not repeat the Chrome C9
   journey unless the repair changes browser-visible behavior; retain the prior
   C9 journey evidence separately from this failure/repair evidence.
4. `M1-S7-C9-ATTACHMENT-CONCURRENCY-REVIEW`: obtain independent code/security
   and specification/evidence reviews with no P0-P2.  Only then reactivate
   `M1-S7-C9-INTEGRATE` under a later approved plan entry.

## Acceptance criteria

1. The prior full-gate failure is retained with environment, command, action,
   expected/actual result and sanitized output; a passing repeat never erases
   it.
2. A focused test proves the selected bounded policy with concurrent
   same-Issue uploads and asserts 12 successful responses, 12 unique IDs, 12
   durable list entries and 12 regular object files.
3. A forced non-transient storage/binding/commit failure remains a failure,
   leaves no object/reference leak and produces no realtime event.
4. Existing attachment runtime concurrency, transaction, event and restart
   tests pass at least ten consecutive focused counts.
5. `go test ./... -count=1`, `go vet ./...`, `go mod verify`, `gofmt -d` on
   changed Go files, `git diff --check`, and tracked/untracked `server/**`
   checks pass.  Windows race exit `0xc0000139` remains a limitation, never a
   pass.
6. The repair receives independent code/security and specification/evidence
   reviews before C9 integration resumes.

## Risks and rollback

- Do not raise global timeouts or add unbounded retries to hide a SQLite lock.
  The RED must identify the error class and the GREEN must bound attempts/time.
- Do not change the Canonical DB schema or delete retained DB, WAL/SHM/journal,
  attachment objects, logs or C9 evidence.
- If the focused failure cannot be classified without a material observability
  or architecture change, stop and propose v10 rather than broadening this
  repair.
- Rollback reverts only the repair commit, keeps the retained failure/evidence,
  and leaves all runtime artifacts intact.

## Deterministic verification

From `backend/`:

```powershell
go test ./internal/modules/space/... ./internal/modules/workspace/... ./internal/bootstrap ./cmd/server -count=1
go test ./internal/bootstrap -run '^TestSQLiteRuntimeConcurrentAttachmentUploadsLoseNoReferencesOrFiles$' -count=10
go test ./... -count=1
go vet ./...
go mod verify
```

Then run `gofmt -d` on changed Go files, `git diff --check`, and both tracked
and untracked `server/**` scope checks.  The independent reviews and prior C9
browser evidence are separate gates; this plan does not claim milestone
acceptance.
