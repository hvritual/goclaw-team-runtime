# Canonical SQLite runtime cutover — execution plan v10

- Plan-ID: `canonical-sqlite-runtime-cutover`
- Version: `10`
- Status: `approved`
- Approval source: Human Customer confirmation `批准进行` dated `2026-08-16`
- Supersedes: `plan_v9.md`
- Base commit: `06a2273c270019e2e6ba3a449e5d8f59a9df69bd`
- Branch and integration target: `codex/multica-six-domain-baseline`
- Active step: `M1-S7-C9-INTEGRATE-PREFLIGHT`
- Policy bundle: `backend-v1`
- XP mode: strict; maximum active stories: one

## Purpose

Plan v9 closed the only C9 full-gate blocker: bounded SQLite acquisition retry
for concurrent attachment uploads.  Its repair candidate completed fresh
Backend gates and independent review, but it deliberately did not re-run the
browser-visible C9 journey.  This plan resumes only C9 integration and final
acceptance against a new clean candidate that includes the repair.

The live local runtime currently belongs to the older `0888efc` candidate and
does not contain the v9 repair.  It is invalid evidence for this plan and must
be quiesced by its verified selector supervisor before a fresh candidate owns
the C9 ports.

## Frozen integration contract

- The current C9 capability contract from `plan_v7.md` remains unchanged:
  the complete local Issue detail supports edit, move, hierarchy,
  collaboration, labels, properties, acceptance, attachments, realtime,
  reload and retained-data restart.  External VCS/GitHub integration alone is
  explicitly disabled.
- The v9 attachment retry contract remains frozen.  The C9 browser journey is
  re-run because the repaired transaction path is part of the accepted
  attachment interaction, not because this plan changes a product API.
- All final evidence originates from a new detached clean candidate containing
  the v9 repair plus this plan's evidence-only commits.  The dirty primary
  worktree and the old `0888efc` runtime cannot supply acceptance evidence.
- The supported local runtime owns only Web `127.0.0.1:3000`, Canonical HTTP
  `127.0.0.1:8000` and Canonical gRPC `127.0.0.1:9000`.  There is no accepted
  localhost legacy listener or browser request to port `8080`.
- Retained SQLite database, WAL/SHM/journal files, asset objects, logs, prior
  failure evidence and prior C9 evidence are preserved.  Rollback remains
  stop, quiescent snapshot, selector restoration and hash/readback proof; it
  never deletes retained artifacts.

## Authorized write boundary

- `e2e/canonical-runtime.spec.ts` only for a C9 clean-candidate acceptance
  scenario and sanitized trace assertions.
- `scripts/runtime-selector*` and `scripts/canonical-runtime-verifier*` only
  for a reproduced selector/verifier evidence defect discovered during this
  plan.
- `backend/docs/plans/canonical-sqlite-runtime-cutover/**` for this immutable
  plan, progress journal, evidence index, story/milestone state and review
  record.

No backend, frontend, Core, view, database schema, capability, route or
product-behavior change is authorized.  If deterministic or browser evidence
exposes any such defect, stop C9 integration, retain the failure, and propose
`plan_v11.md` before editing product code.

Explicitly excluded: `server/**`; all legacy root backend trees; external
VCS/GitHub integrations; `packages/ui/components/ui/input.tsx`,
`packages/views/auth/input-controlled.test.tsx`, the dirty Issue table and
create-modal files; and `.local-runtime/`, `docs/code-to-product/` and `ui/`
local artifact roots.

## Ordered XP sequence

1. `M1-S7-C9-INTEGRATE-PREFLIGHT`: freeze the exact base, v9 repair hash,
   plan hash, live process/listener ownership, dirty-path exclusions and the
   new detached clean-candidate path.  Confirm the old `0888efc` selector
   supervisor and children own `3000/8000/9000`; stop only that verified tree
   and prove `8080` has no local listener.
2. `M1-S7-C9-INTEGRATE-RED`: add or verify a bounded clean-candidate C9
   acceptance scenario which inventories the full enabled detail surface and
   fails on any local 404, missing control, forbidden capability, legacy
   request, persistence/restart regression or realtime omission.  This step
   may change only the authorized test/evidence paths.
3. `M1-S7-C9-INTEGRATE-GREEN`: select and start Canonical from the fresh clean
   candidate, prepare a non-destructive local fixture, and run the installed
   Chrome C9 journey.  Capture sanitized HTTP/WebSocket trace evidence for
   edit, move, hierarchy, collaboration, labels, properties, acceptance,
   attachments, realtime, reload and restart.  A failed required action stops
   the plan before product code changes.
4. `M1-S7-C9-INTEGRATE-VERIFY`: run the complete deterministic Backend and
   package gates, selector/verifier, retained-data restart/readback, quiescent
   artifact snapshot, rollback and hash-preservation checks.  The browser
   trace must show no localhost `:8080` request.
5. `M1-S7-C9-REVIEW`: obtain independent code/security and
   specification/evidence reviews of the exact clean candidate with no
   unresolved P0-P2.
6. `M1-S7-C9-ACCEPT`: record the explicit Human Customer milestone acceptance.
   This plan may request that decision after all technical gates pass, but it
   cannot infer or record it on the Customer's behalf.

## Acceptance criteria

1. The fresh candidate is detached, clean, includes
   `606ce6524f0836f45b95783406a0d0ad244fedc9`, contains no tracked or
   untracked `server/**` path, and does not inherit the primary worktree's
   unrelated dirty files.
2. The old selector tree is stopped only after its command line, worktree and
   listener ownership are recorded; the new selector owns `3000/8000/9000` and
   no local legacy listener owns `8080`.
3. The installed-Chrome C9 scenario executes every frozen local detail
   interaction and records successful UI state, canonical same-origin network
   requests and WebSocket event evidence.  It must fail on a missing local
   endpoint/control or any `:8080` request.
4. A retained-database restart preserves the fixture mutations and a
   reconnect/reload observes the committed state.  The subsequent quiescent
   rollback preserves database, SQLite sidecar, asset and log hashes exactly.
5. All deterministic checks below pass on the exact candidate.  Windows race
   exit `0xc0000139` remains an environment limitation, never a pass.
6. Independent reviews return no P0-P2 and the Customer explicitly accepts
   the final milestone before any `Milestone Accepted` claim is made.

## Deterministic verification

From `backend/`:

```powershell
gofmt -d <changed-go-files>
go test ./internal/modules/auth ./internal/modules/workspace ./internal/modules/space ./internal/bootstrap ./cmd/server -count=1
go test ./... -count=1
go vet ./...
go mod verify
```

From repository root:

```powershell
pnpm --filter @multica/core test
pnpm --filter @multica/core typecheck
pnpm --filter @multica/core lint
pnpm --filter @multica/views test -- <focused Issue tests>
pnpm --filter @multica/views typecheck
pnpm --filter @multica/web typecheck
node --test scripts/runtime-selector.test.mjs scripts/canonical-runtime-verifier.test.mjs
git diff --check
git status --porcelain -- server
```

Run the repository selector/verifier from the new candidate, then execute the
clean installed-Chrome C9 scenario, retained restart/readback, quiescent
snapshot, rollback and preservation checks.  Record exact commands, hashes,
ports and sanitized trace locations in `journal.md`.

## Risks and stop conditions

- Stop if any candidate diff includes `server/**`, an excluded dirty path or a
  product implementation change not already present at the base.
- Stop if a process does not prove ownership by the old/new selector tree; do
  not kill an unrelated process merely because it uses Node, Go or an adjacent
  port.
- Stop if the browser evidence is rendered from the primary dirty worktree,
  an old candidate or an unknown remote runtime.
- Stop if a test or browser action exposes a missing local route/control,
  tenant/security/body/lifecycle drift, broken restart/readback, unowned
  listener or rollback hash mismatch.  Preserve evidence and request a new
  repair plan instead of retrying around the failure.

## Rollback

The selector stops only verified Canonical/Web processes, waits for their
ports to close, snapshots retained artifacts while quiescent, restores the
previous selector choice and verifies hashes/readback.  No retained database,
sidecar, object, trace or log is deleted.  Reverting this plan removes only
its evidence/test/docs commits and never reverts the v9 transaction repair.
