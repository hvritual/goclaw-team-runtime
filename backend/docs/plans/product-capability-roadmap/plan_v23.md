# Product capability roadmap v23 — S06B governed Knowledge review

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Plan-Version: `23`
- Task-Revision: `r028`
- Work-Item: `PCR-S06B`
- Exact base: `84ed0e4afa8f76de2cb854047f2fc4d26641810f`
- Authority: Human Customer standing direction on 2026-08-18 to complete
  Release 2 sequentially through S06B

## Outcome

Install the Workspace-owned Knowledge proposal, independent review, publication,
supersession, and invalidation lifecycle. The installed shared Web/Desktop
surface must let members propose evidence-backed Knowledge and let authorized
independent reviewers govern it without weakening the completed S06A query.

S06B completes Release 2 only after deterministic checks, installed-browser
acceptance, exact-scope checks, and a fresh independent SPEC and
code/security/quality PASS. Push, merge, and deployment remain excluded.

## Frozen product contract

1. Install the existing frozen route family:
   - `POST /api/knowledge/proposals`
   - `GET /api/knowledge/candidates`
   - `POST /api/knowledge/candidates/{id}/review`
2. Proposal uses trusted Workspace and actor identity, requires
   `Idempotency-Key`, and accepts optional `project_id` or target
   `knowledge_id`, exact kind, nonempty title/content/reason, and ordered source
   references. Reuse with another canonical body returns `409`; exact replay
   returns the original `201` response without a duplicate candidate or audit.
3. A target proposal captures the exact current published Knowledge revision.
   The client cannot provide or override `workspace_id`, proposer, candidate
   revision, target revision, status, audit identity, or timestamps.
4. Candidate list is authorized review work, uses an opaque signed cursor,
   defaults to nonterminal `candidate,in_review,quarantined`, sorts by
   `updated_at DESC,id ASC`, and never crosses Workspace. Owner/admin may list;
   ordinary members and agents may not infer the queue.
5. Review request is strict JSON with `action`, positive `expected_revision`,
   nonempty `rationale`, and optional `emergency=false`. Unknown or additional
   fields fail. Every successful action increments the candidate revision once.
6. Frozen action transitions are:
   - `approve`: `candidate -> in_review`
   - `reject`: `in_review -> rejected`
   - `quarantine`: `in_review -> quarantined`
   - `return`: `quarantined -> in_review`
   - `publish`: `in_review -> published`, only for a new candidate, creating
     one governed published entry
   - `supersede`: `in_review -> published`, only for a target proposal,
     creating one new published replacement entry and marking the exact target
     entry `superseded`
   - `invalidate`: `in_review -> published`, only for a target proposal,
     marking the exact target entry `invalidated` and creating no replacement
7. Terminal candidates are immutable. Terminal target actions recheck the
   captured target revision and published status in the same transaction.
   Candidate or target staleness returns canonical `409 code=revision_conflict`
   with `resource=candidate|knowledge`, the current revision, and zero mutation.
8. Proposal requires `workspace.knowledge.propose`: owner/admin/member allow,
   agents deny by default. Candidate list/review requires
   `workspace.knowledge.review`: owner/admin allow; members remain denied because
   no explicit reviewer-grant management route is installed in this release.
9. The proposer cannot perform any review action. Only an owner authorized for
   `workspace.knowledge.self_review_override` may override that rule with
   `emergency=true` and a distinct nonempty rationale of at least 12 trimmed
   characters. Admin/member/agent self-review remains denied even with a reason.
10. A terminal publication action requires at least one normalized source
    reference with nonempty type, ID, revision, and citation. Asset ID and asset
    version ID are an all-or-none pair; when present the Space-owned asset is
    validated read-only as belonging to the trusted Workspace. Space bodies and
    Control Plane stores are never read or written.
11. New candidate, review-event, and replacement relations are Workspace-owned.
    Existing governed revisions/source refs remain the published authority.
    Additive migration may extend governed status with `invalidated`; S06A never
    returns invalidated entries. No foreign key or cascading action is added.
12. Proposal, candidate transition, governed entry/revision/source writes,
    target status changes, idempotency replay, and immutable audit evidence are
    atomic on one SQLite transaction. Cancellation, conflict, audit failure, or
    source validation failure leaves zero partial rows.
13. Every successful proposal/review writes one immutable
    `workspace_audit_entries` record with secret-free metadata. After commit,
    emit exactly one Workspace realtime event containing only authorized
    candidate/entry projections. Failed/replayed mutations emit no duplicate
    event.
14. Core proposal/candidate/review success schemas are strict and fail closed;
    mutations never fall back to success. Query keys include Workspace and full
    cursor/status inputs. Successful mutations invalidate candidate and governed
    Knowledge queries.
15. Shared Knowledge UI mounts proposal/review behavior only after loaded
    `knowledge_review=true`. Members receive proposal controls; owner/admin also
    receive the review queue and valid state-specific actions. The server remains
    authoritative for permissions, self-review, source evidence, and conflicts.
16. Completed feature flags remain true and only `knowledge_review` becomes
    true. Release 3 features remain false. Existing S06A query behavior,
    quarantine visibility, source deep links, and strict schemas remain intact.

## Scope

Writable scope is limited to:

- `backend/internal/modules/workspace/**` and composition under
  `backend/internal/bootstrap/**`;
- additive Workspace SQLite migrations and their guarded down migration;
- `packages/core/knowledge/**`, `packages/core/types/knowledge.ts`, and exact API
  client/schema exports;
- `packages/views/knowledge/**`, Knowledge locale resources, and narrow shared
  navigation/config wiring;
- an exact `e2e/knowledge-review.spec.ts` acceptance path;
- this roadmap plan, pointer, story map, task register, and journal.

`server/**`, generated protobufs, Control Plane implementation, Space bodies,
unrelated UI files, push, merge, and deployment are excluded.

## TDD execution

1. RED: proposal authorization, trusted identity, strict validation,
   idempotency replay/conflict, source normalization, target capture, and atomic
   cancellation tests.
2. GREEN: candidate domain/application and additive persistence transaction.
3. RED: every state transition, self-review/owner override, stale candidate and
   target, missing/foreign source, terminal immutability, audit rollback, and
   realtime-after-commit tests.
4. GREEN: strict HTTP routes, Core schemas/queries/mutations, loaded feature
   gating, and role/state-specific shared UI.
5. Prove restart persistence, S06A invalidated exclusion, Workspace isolation,
   down-migration guards, feature signals, and installed browser behavior.

## Deterministic gates

- Focused Workspace application/domain/repository/HTTP/runtime tests cover all
  transitions and negative paths with no retry.
- Core and Views focused tests cover malformed success payloads, complete query
  keys, mutation invalidation, loaded gating, proposal, review, self-review
  display, conflict recovery, and hidden controls.
- `cd backend && make check && make test-race` must pass using the official
  repository scripts; environment failures are reported, never waived.
- Root `pnpm typecheck` and `pnpm test` run. Any unrelated aggregate failure is
  recorded honestly and followed by the exact affected focused test; it is not
  promoted to an aggregate PASS.
- Production Web build must pass. Fresh SQLite plus installed Chrome, retries
  disabled, must exercise a real member proposal and independent owner/admin
  review through publication plus visible S06A readback and realtime refresh.
- Exact candidate range and worktree `server/**` diffs must be empty; protected
  Input blob and all pre-existing dirty exclusions must remain unchanged.
- Fresh independent review must return both SPEC PASS and
  CODE/SECURITY/QUALITY PASS before PCR-S06B or Release 2 closes.

## Rollback and stop conditions

Rollback disables the three S06B routes and forces `knowledge_review=false`.
The additive down migration succeeds only when every S06B candidate/review row,
S06B-created governed relation, and retained audit/idempotency dependency is
empty; otherwise it fails without dropping data.

Stop on any cross-Workspace disclosure, self-review bypass, stale publish,
source trust bypass, partial transaction, duplicate replay/event, mutation
fallback, feature pre-enable, `server/**` change, or independent-review block.
