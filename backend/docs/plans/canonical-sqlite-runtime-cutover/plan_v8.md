# Canonical SQLite runtime cutover — execution plan v8

- Plan-ID: `canonical-sqlite-runtime-cutover`
- Version: `8`
- Status: `approved`
- Approval source: Human Customer confirmation `批准 plan v8` dated
  `2026-08-15`
- Supersedes: `plan_v7.md`
- Base commit: `f1b4555e464ea955a47b6e322ffb06e679a787b9`
- Branch and integration target: `codex/multica-six-domain-baseline`
- Active step: `M1-S7-C8-REPAIR-RED`
- Policy bundle: `backend-v1`
- XP mode: strict; maximum active stories: one

## Purpose

Independent C8 Navigator review rejected technical acceptance with no P0, two
code P1, two specification/evidence P1 and supporting P2 findings. C9 remains
inactive. This revision authorizes only the smallest C8 repair needed to close
those findings; every other contract and non-goal from approved plan v7 remains
unchanged.

## Frozen repair contract

### Trusted Workspace context

- Every stored attachment GET, preview, download and DELETE authenticates before
  Workspace resolution.
- Missing Workspace input returns 400. The trusted slug/ID resolver defines the
  canonical Workspace; slug/ID mismatch, foreign Workspace and attachment
  ownership mismatch return the same hidden 404.
- Stored attachment ownership never silently overrides the caller's selected
  Workspace, even when the same user belongs to both Workspaces.

### Transactional reference integrity

- Ordinary Issue field updates that omit `attachment_ids` never rewrite the
  current attachment bag.
- An explicit Issue attachment replacement is validated against Space metadata
  and written inside the same `BEGIN IMMEDIATE` transaction as the Issue row.
- The repository detects a changed attachment bag between the authorized read
  and write rather than overwriting a concurrent upload/delete. A conflict does
  not mutate the Issue or Space records.
- Attachment DELETE/unbind and upload/bind remain serialized with Issue writes;
  committed state contains neither dangling nor silently lost references.
- Capability-off Issue create rejects `attachment_ids` whenever the field is
  present, including an explicit empty array.

### Complete frontend bag and real deletion control

- Description autosave submits the complete authoritative Issue attachment bag
  plus currently referenced pending uploads, deduplicated by ID. Refreshing and
  adding B cannot silently unbind retained attachment A.
- The enabled Issue detail exposes an accessible persisted-attachment delete
  control that calls the existing strict Core delete client, invalidates the
  authoritative Issue/detail/attachment caches and surfaces failure.
- The disabled capability mounts neither the query nor the delete control.

### Browser and durable evidence

- A clean detached candidate uses the installed Chrome executable and records
  executable path plus user agent in its sanitized trace.
- Browser evidence uploads A, reloads/restarts, uploads B while preserving A,
  previews/downloads retained bytes, and clicks a visible deletion control.
  It does not use `page.request.delete` as the deletion proof.
- Cleanup targets only the run's synthetic attachments after action-time Human
  confirmation. Database, object, log and trace artifacts are retained.

## Authorized write boundary

- `backend/internal/modules/space/**`
- `backend/internal/modules/workspace/**`
- `backend/internal/bootstrap/**` and `backend/cmd/server/**` only for focused
  runtime/config evidence
- `backend/internal/realtime/hub.go` only to register the already frozen
  `issue_attachments:changed` event type
- `packages/core/api/**`, `packages/core/issues/**`, `packages/core/types/**`
  and focused tests
- `packages/views/issues/**` and focused tests
- `e2e/canonical-runtime.spec.ts`
- `backend/docs/plans/canonical-sqlite-runtime-cutover/**`

Explicitly excluded: every `server/**` path; root legacy `teamcontrol/**`;
unrelated dirty Input/table/create-modal files; Invitations, Inbox, external
VCS/PR integration and every C9 product change.

## Strict XP sequence

1. `M1-S7-C8-REPAIR-RED`: add focused failing tests for missing/mismatched
   Workspace context, concurrent reference integrity, explicit empty disabled
   create, complete frontend bag and real UI deletion.
2. `M1-S7-C8-REPAIR-GREEN`: implement only enough backend/Core/View behavior to
   pass those tests.
3. `M1-S7-C8-REPAIR-INTEGRATE`: run focused/full deterministic gates, produce a
   new clean-candidate installed-Chrome trace, and obtain independent code and
   specification reviews with no P0-P2.
4. `M1-S7-C8-ACCEPT`: the Human Customer's conditional instruction
   `审查通过后验收 C8 并启动 C9` becomes effective only after both independent
   reviews pass. Record acceptance, then and only then activate `M1-S7-C9-RED`.

## Acceptance commands

From `backend/`:

```powershell
go test ./internal/modules/space/... ./internal/modules/workspace/... ./internal/bootstrap ./cmd/server -count=1
go test ./... -count=1
go vet ./...
go mod verify
```

From the repository root:

```powershell
pnpm --filter @multica/core test
pnpm --filter @multica/core typecheck
pnpm --filter @multica/core lint
pnpm --filter @multica/views test -- issue-detail.test.tsx
pnpm --filter @multica/views typecheck
pnpm --filter @multica/views lint
pnpm --filter @multica/web typecheck
pnpm exec playwright test e2e/canonical-runtime.spec.ts --list
git diff --check
git status --porcelain -- server
```

Windows race binaries that exit `0xc0000139` remain an environment limitation,
not passing race evidence. Repository wrapper scripts that rely on unavailable
Unix shell syntax must be reported honestly with their direct deterministic
equivalents; they are never relabelled as green.
