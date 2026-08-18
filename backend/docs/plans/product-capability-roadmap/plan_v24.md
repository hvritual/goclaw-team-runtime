# Product capability roadmap v24 — S06B realtime delivery remediation

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Plan-Version: `24`
- Task-Revision: `r029`
- Work-Item: `PCR-S06B`
- Exact base: `ffcdd1c7a87db21a3a5f5a20afb66c6c952ee8ac`
- Authority: Human Customer standing direction on 2026-08-18 to complete
  Release 2 and remediate every frozen S06B gate without push, merge, or deploy

## Why a new version is required

v23 candidate `ca8b2a41f7ca4871375584a0a4ee1628c8c0a075` passed its backend,
race, type, build, and single-identity browser checks. The stronger frozen
member-to-independent-reviewer browser RED at exact base
`ffcdd1c7a87db21a3a5f5a20afb66c6c952ee8ac` then proved
that the application emits `knowledge:candidate_updated` after commit while the
shared realtime Hub silently rejects that event type. The v23 writable scope
does not include `backend/internal/realtime/**` or the shared WebSocket event
type contract, so r028 is verification-blocked rather than expanded in place.

## Outcome

Deliver the already-frozen S06B candidate projection through the authorized
Workspace realtime channel so an independently signed-in owner/admin sees a
member proposal without reload and the member sees publication through S06A
readback. No Knowledge mutation, storage, permission, or route semantics change.

## Frozen remediation contract

1. The existing post-commit event name remains exactly
   `knowledge:candidate_updated`; proposal replay and failed mutations still
   emit nothing.
2. The realtime Hub admits that one event type and continues to scope delivery
   by trusted Workspace membership. No wildcard or unrelated event is admitted.
3. The shared WebSocket event union names that exact event. Existing payload
   boundary behavior remains unchanged; consumers treat realtime as cache
   invalidation and refetch authoritative server state.
4. A Hub regression test must prove delivery to the target Workspace and no
   delivery to another Workspace. Existing allowlist denial remains tested.
5. Fresh SQLite plus production Web and installed Chrome, retries disabled,
   must use separate member and owner sessions. The member proposes; the owner
   receives the candidate without reload, approves and publishes; the member
   receives visible published readback without reload. Knowledge routes may not
   be mocked.
6. Exact v23 behavior, immutable migrations/audit/idempotency, strict Core,
   feature gating, and all prior deterministic evidence remain unchanged.

## Writable scope

- `backend/internal/realtime/hub.go` and its exact tests;
- `packages/core/types/events.ts` and exact WebSocket/realtime tests if needed;
- `e2e/knowledge-review.spec.ts`;
- this roadmap pointer, story map, task register, journal, and immutable plan.

`server/**`, generated protobufs, Knowledge mutation/storage/application code,
unrelated UI, push, merge, deployment, and Release 3 remain excluded.

## TDD and deterministic gates

1. Retain the failing dual-identity installed-browser acceptance at exact base
   `ffcdd1c7a87db21a3a5f5a20afb66c6c952ee8ac` as RED evidence.
2. Add a focused Hub RED for the exact event and Workspace isolation, then make
   only the narrow allowlist/type change required for GREEN.
3. Run focused Hub/Core/realtime tests, `cd backend && make check && make
   test-race`, root typecheck/test, and production Web build. An unrelated root
   aggregate failure remains non-PASS and gets an exact focused classification.
4. Re-run the fresh dual-identity installed-Chrome journey with the backend
   WebSocket URL explicitly wired at build time.
5. Verify immutable v23 hash, exact candidate/worktree `server/**` emptiness,
   protected Input blob, dirty exclusions, trailers, and process cleanup.
6. Fresh independent SPEC and CODE/SECURITY/QUALITY PASS is required on the
   final exact candidate before r029, PCR-S06B, or Release 2 closes.

## Rollback and stop conditions

Rollback removes only the new realtime allowlist/type entry and restores the
v23 verification-blocked behavior; no database rollback is involved. Stop on
cross-Workspace delivery, pre-commit delivery, duplicate replay event, a mocked
Knowledge browser route, `server/**` change, or independent-review block.
