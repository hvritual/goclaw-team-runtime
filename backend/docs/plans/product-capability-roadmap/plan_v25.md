# Product capability roadmap v25 — S06B authorized realtime projection

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Plan-Version: `25`
- Task-Revision: `r030`
- Work-Item: `PCR-S06B`
- Exact base: `33ec500fcfe9d3bb751d898dfbbdd9bce664aa48`
- Authority: Human Customer standing direction on 2026-08-18 to complete
  Release 2 and remediate all independently reviewed blockers

## Why v24 is replaced before implementation

The independent review of v23 candidate
`ca8b2a41f7ca4871375584a0a4ee1628c8c0a075` confirmed the observed Hub drop
and identified a second security invariant: admitting the event unchanged would
broadcast private candidate/proposer/reason data to ordinary members who cannot
list or review the queue. v24 authorized only a raw allowlist/type repair and
cannot safely add role-aware composition or sanitize public projections. Its
uncommitted trial was discarded; r029 is design-blocked with no product commit.

## Outcome

Deliver exactly one post-commit Knowledge event through role-aware Workspace
realtime. Authorized reviewers receive the candidate projection needed to
refetch the private queue. Ordinary members receive no nonterminal candidate
event and, on publication only, receive a public governed-entry projection that
invalidates S06A query state. Complete the missing transition/cancellation tests
and exact traceability gate without changing v23 business semantics.

## Frozen authorized projection contract

1. Application event name remains `knowledge:candidate_updated`. Failed and
   replayed mutations emit no event; each successful mutation invokes publish
   exactly once after repository commit.
2. The Hub stores the trusted connected actor identity. For this event it asks
   one injected permission resolver whether that actor has
   `workspace.knowledge.review` in the trusted Workspace.
3. Authorized reviewers receive the full existing candidate projection and an
   entry when publication creates one. Ordinary members receive no frame for
   proposal/approve/reject/quarantine/return, and receive only `{entry}` for a
   successful publish/supersede. Invalidation has no public entry and therefore
   sends ordinary members no frame. No proposer, reason, candidate status, or
   private source projection is exposed to non-reviewers.
4. Existing event allowlist remains closed. Only the exact Knowledge type is
   added. Other Workspace event behavior and ordering stay unchanged.
5. Resolver errors deny the private projection. The Hub never calls the
   resolver for another Workspace and never sends either projection across
   Workspace boundaries.
6. The shared event union adds the exact Knowledge type. Realtime remains an
   invalidation hint; clients refetch authoritative candidate/query state.
7. Table-driven backend tests cover all seven transitions, invalid transitions,
   terminal immutability, cancellation zero-mutation, reviewer/private versus
   member/public projection, resolver denial, and foreign Workspace exclusion.
8. Fresh SQLite, production Web, and installed Chrome use separate member and
   owner identities. Without reload or Knowledge mocks, member proposal appears
   to owner; owner approves/publishes; published readback appears to member.
9. Final candidate commits use one continuous parseable trailer block containing
   `Task-ID`, `Project-ID`, `Task-Revision`, `Work-Item`, `Plan-ID`,
   `Plan-Version`, `Plan-Step`, and `Policy-Bundle`.

## Writable scope

- `backend/internal/realtime/hub.go` and exact tests;
- realtime construction only in `backend/internal/bootstrap/sqlite.go` and exact
  composition tests;
- S06B test-only additions under
  `backend/internal/modules/workspace/internal/{application,infrastructure}/**`;
- `packages/core/types/events.ts` and exact WebSocket/realtime tests;
- `e2e/knowledge-review.spec.ts`;
- current roadmap pointer, story map, task register, journal, and this immutable
  plan.

Knowledge routes, production mutation/storage/application code, migrations,
unrelated UI, `server/**`, generated protobufs, push, merge, deployment, and
Release 3 are excluded.

## Ordered execution and gates

1. RED: retain the v23 dual-identity browser failure; add Hub projection and
   table-driven transition/cancellation failures.
2. GREEN: implement only identity retention, injected permission resolution,
   exact allowlist, per-client projection, and shared event type.
3. Run focused Hub, Workspace, Core, realtime, and Views tests; backend `make
   check` and official `make test-race`; root typecheck/test; production Web
   build; and the fresh installed-Chrome dual-identity journey with direct
   build-time WebSocket wiring and retries disabled.
4. Verify v23/v24/v25 immutable hashes, exact `server/**` emptiness, protected
   Input blob, dirty exclusions, process cleanup, and parseable trailers.
5. Obtain fresh independent SPEC and CODE/SECURITY/QUALITY PASS on the exact
   final candidate before r030, PCR-S06B, or Release 2 closure.

## Rollback and stop conditions

Rollback removes the exact Hub resolver/allowlist/type changes; database state
is unaffected. Stop on any ordinary-member candidate projection, cross-Workspace
frame, duplicate/replayed event, pre-commit event, missing action/cancellation
coverage, mocked Knowledge browser route, `server/**` change, or independent
review block.
